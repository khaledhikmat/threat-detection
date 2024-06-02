package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/khaledhikmat/threat-detection-shared/models"
	"github.com/khaledhikmat/threat-detection-shared/utils"

	"github.com/mitchellh/mapstructure"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"

	"github.com/khaledhikmat/threat-detection-shared/service/config"
	"github.com/khaledhikmat/threat-detection-shared/service/pubsub"
	"github.com/khaledhikmat/threat-detection-shared/service/storage"
	otelprovider "github.com/khaledhikmat/threat-detection-shared/telemetry/provider"
)

type headerRoundTripper struct {
	Next http.RoundTripper
}

func (b headerRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Content-Type", "application/json")
	return b.Next.RoundTrip(r)
}

type loggingRoundTripper struct {
	Next   http.RoundTripper
	Logger io.Writer
}

func (b loggingRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	_, _ = b.Logger.Write([]byte("Request to " + r.URL.String() + "\n"))
	return b.Next.RoundTrip(r)
}

var recordingsTopicSubscription = &common.Subscription{
	PubsubName: models.ThreatDetectionPubSub,
	Topic:      models.RecordingsTopic,
	Route:      fmt.Sprintf("/%s", models.RecordingsTopic),
}

// Global DAPR client
var configSvc config.IService
var pubsubSvc pubsub.IService
var storageSvc storage.IService

var recordingsTopic = models.RecordingsTopic
var alertsTopic = models.AlertsTopic
var metadataTopic = models.MetadataTopic

var modeProcs = map[string]func(ctx context.Context) error{
	"dapr": daprModeProc,
	"aws":  awsModeProc,
}

var modelProcs = map[string]func(ctx context.Context, clip models.RecordingClip) error{
	"weapon": weapon,
	"fire":   fire,
}

func main() {
	rootCtx := context.Background()
	canxCtx, _ := signal.NotifyContext(rootCtx, os.Interrupt)

	// Setup services
	configSvc = config.New()

	// Load env vars if running in local rutime mode
	if configSvc.GetRuntimeEnv() == "local" {
		err := godotenv.Load()
		if err != nil {
			fmt.Println("Failed to start env vars", err)
			return
		}
	}

	if configSvc.GetRuntimeEnv() == "local" && os.Getenv("APP_PORT") == "" {
		fmt.Printf("Failed to start - %s env var is required\n", "APP_PORT")
		return
	}

	// Setup Otel
	optype := otelprovider.AwsOtelProvider
	if configSvc.GetOtelProvider() == "noop" || configSvc.GetOtelProvider() == "" {
		optype = otelprovider.NoOp
	}

	shutdown, err := otelprovider.New(canxCtx, "threat-detection-media-api", otelprovider.WithProviderType(optype))
	if err != nil {
		fmt.Println("Failed to start otel", err)
		return
	}

	defer func() {
		_ = shutdown(canxCtx)
	}()

	fn, ok := modeProcs[configSvc.GetRuntimeMode()]
	if !ok {
		fmt.Printf("Mode processor %s not supported\n", configSvc.GetRuntimeMode())
		return
	}

	err = fn(canxCtx)
	if err != nil {
		fmt.Printf("Failed to start mode processor %s %v\n", configSvc.GetRuntimeMode(), err)
		return
	}
}

func daprModeProc(_ context.Context) error {
	c, err := dapr.NewClient()
	if err != nil {
		fmt.Println("Failed to start dapr client", err)
		return err
	}
	defer c.Close()

	pubsubSvc = pubsub.NewDaprPubsub(c, configSvc)
	// WARNING I am using AWS storage while in DAPR runtime mode because I can store to S3
	//storageSvc = storage.NewDaprStorage(c, configSvc)
	storageSvc = storage.NewAwsStorage(configSvc)

	// Create a DAPR service using the app port
	s := daprd.NewService(":" + os.Getenv("APP_PORT"))
	fmt.Printf("Model Invoker  - DAPR Service for %s created!\n", configSvc.GetSupportedAIModel())

	// Register pub/sub metadata topic handler
	if err := s.AddTopicEventHandler(recordingsTopicSubscription, daprRecordingsHandler); err != nil {
		panic(err)
	}
	fmt.Printf("Model Invoker  - metadata topic handler registered for %s!\n", configSvc.GetSupportedAIModel())

	// Start DAPR service
	// TODO: Provide cancellation context
	if err := s.Start(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}

	return nil
}

func daprRecordingsHandler(ctx context.Context, e *common.TopicEvent) (bool, error) {
	// Decode pledge
	evt := models.RecordingClip{}
	err := mapstructure.Decode(e.Data, &evt)
	if err != nil {
		fmt.Println("Failed to decode event", err)
		return false, err
	}

	return false, processRecordingClip(ctx, evt)
}

func awsModeProc(ctx context.Context) error {
	pubsubSvc = pubsub.NewAwsPubsub(configSvc)
	storageSvc = storage.NewAwsStorage(configSvc)

	// Create a topic for my recordings if it does not exist
	// There could be some competition here, but we will ignore it for now
	// In higher env, queues and topics would be pre-created
	recordingsTopic, err := pubsubSvc.CreateTopic(ctx, models.RecordingsTopic)
	if err != nil {
		return err
	}

	// Create a topic for my alerts if it does not exist
	// There could be some competition here, but we will ignore it for now
	// In higher env, queues and topics would be pre-created
	alertsTopic, err = pubsubSvc.CreateTopic(ctx, models.AlertsTopic)
	if err != nil {
		return err
	}

	// Create a topic for my metadata if it does not exist
	// There could be some competition here, but we will ignore it for now
	// In higher env, queues and topics would be pre-created
	metadataTopic, err = pubsubSvc.CreateTopic(ctx, models.MetadataTopic)
	if err != nil {
		return err
	}

	// Create a queue for my topic if it does not exist
	// In higher env, queues and topics would be pre-created
	queueURL, queueARN, err := pubsubSvc.CreateQueue(ctx, fmt.Sprintf("model-invoker-queue-%s", strings.ToLower(configSvc.GetSupportedAIModel())), recordingsTopic)
	if err != nil {
		return err
	}

	// Register a subscription to the recordings topic
	stream, err := pubsubSvc.Subscribe(ctx, recordingsTopic, queueURL, queueARN)
	if err != nil {
		return err
	}
	fmt.Printf("Model Invoker  - Subscribed to %s\n", recordingsTopic)

	// Process messages from the subscription stream
	for msg := range stream {
		select {
		case <-ctx.Done():
			fmt.Println("awsModeProc - context cancelled")
			break
		default:
			// Decode message
			clip := models.RecordingClip{}
			err := json.Unmarshal([]byte(msg), &clip)
			if err != nil {
				fmt.Println("Record but ignore - Failed to decode event", err)
				continue
			}

			err = processRecordingClip(ctx, clip)
			if err != nil {
				fmt.Println("Record but ignore - Failed to process event", err)
			}
		}
	}

	return nil
}

func processRecordingClip(ctx context.Context, evt models.RecordingClip) error {
	// Determine if my AI Model is required for this clip
	fmt.Printf("Processing clip %s with AI models %v and AI Model %s \n", evt.ID, evt.Analytics, configSvc.GetSupportedAIModel())
	if evt.Analytics == nil ||
		!utils.Contains(evt.Analytics, configSvc.GetSupportedAIModel()) {
		fmt.Printf("Ignoring the clip because our supported model [%s] is not needed\n", configSvc.GetSupportedAIModel())
		return nil
	}

	fmt.Printf("Processing the clip because our supported model [%s] is needed\n", configSvc.GetSupportedAIModel())

	fn, ok := modelProcs[configSvc.GetSupportedAIModel()]
	if !ok {
		fmt.Printf("AI Model %s not supported\n", configSvc.GetSupportedAIModel())
		return fmt.Errorf("AI Model %s not supported", configSvc.GetSupportedAIModel())
	}

	evt.ModelInvocationBeginTime = time.Now()
	err := fn(ctx, evt)
	if err != nil {
		fmt.Printf("AI Model processor returned an error %s\n", err.Error())
		return err
	}

	return nil
}
