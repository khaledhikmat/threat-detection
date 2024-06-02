package main

import (
	"context"
	"encoding/json"
	"fmt"
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

var alertTopicSubscription = &common.Subscription{
	PubsubName: models.ThreatDetectionPubSub,
	Topic:      models.AlertsTopic,
	Route:      fmt.Sprintf("/%s", models.AlertsTopic),
}

// Global DAPR client
var configSvc config.IService
var pubsubSvc pubsub.IService
var storageSvc storage.IService

var modeProcs = map[string]func(ctx context.Context) error{
	"dapr": daprModeProc,
	"aws":  awsModeProc,
}

var alertProcs = map[string]func(ctx context.Context, clip models.RecordingClip) error{
	"ccure": ccure,
	"snow":  snow,
	"pers":  pers,
	"slack": slack,
}

var alertsTopic = models.AlertsTopic

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

	// Start the mode processor
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

	// WARNING I am using AWS storage while in DAPR runtime mode because I can store to S3
	//storageSvc = storage.NewDaprStorage(c, configSvc)
	storageSvc = storage.NewAwsStorage(configSvc)

	// Create a DAPR service using the app port
	s := daprd.NewService(":" + os.Getenv("APP_PORT"))
	fmt.Printf("Alerts Notifier - DAPR Service for %s created!\n", configSvc.GetSupportedAlertType())

	// Register pub/sub metadata topic handler
	if err := s.AddTopicEventHandler(alertTopicSubscription, daprAlertHandler); err != nil {
		panic(err)
	}
	fmt.Printf("Alerts Notifier - metadata topic handler registered for %s!\n", configSvc.GetSupportedAlertType())

	// Start DAPR service
	// TODO: Provide cancellation context
	if err := s.Start(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}

	return nil
}

func daprAlertHandler(ctx context.Context, e *common.TopicEvent) (bool, error) {
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

	// Create a topic for my alerts if it does not exist
	// There could be some competition here, but we will ignore it for now
	// In higher env, queues and topics would be pre-created
	alertsTopic, err := pubsubSvc.CreateTopic(ctx, models.AlertsTopic)
	if err != nil {
		return err
	}

	// Create a queue for my topic if it does not exist
	// In higher env, queues and topics would be pre-created
	queueURL, queueARN, err := pubsubSvc.CreateQueue(ctx, fmt.Sprintf("alert-notifier-queue-%s", strings.ToLower(configSvc.GetSupportedAlertType())), alertsTopic)
	if err != nil {
		return err
	}

	// Register a subscription to the metadata topic
	stream, err := pubsubSvc.Subscribe(ctx, alertsTopic, queueURL, queueARN)
	if err != nil {
		return err
	}

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
	// Determine if my alert notifier is required for this clip
	fmt.Printf("Processing clip %s with alert types %v and Alert type %s \n", evt.ID, evt.AlertTypes, configSvc.GetSupportedAlertType())
	if evt.AlertTypes == nil ||
		!utils.Contains(evt.AlertTypes, configSvc.GetSupportedAlertType()) {
		fmt.Printf("Ignoring the clip because our supported alert type [%s] is not needed\n", configSvc.GetSupportedAlertType())
		return nil
	}

	fmt.Printf("Processing the clip because our supported alert type [%s] is needed\n", configSvc.GetSupportedAlertType())

	fn, ok := alertProcs[configSvc.GetSupportedAlertType()]
	if !ok {
		fmt.Printf("Alert processor %s not supported\n", configSvc.GetSupportedAlertType())
		return fmt.Errorf("Alert processor %s not supported", configSvc.GetSupportedAlertType())
	}

	evt.AlertInvocationBeginTime = time.Now()
	err := fn(ctx, evt)
	if err != nil {
		fmt.Printf("Alert processor returned an error %s\n", err.Error())
		return err
	}

	return nil
}
