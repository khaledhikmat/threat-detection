package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/joho/godotenv"
	"github.com/khaledhikmat/threat-detection-shared/models"
	"github.com/khaledhikmat/threat-detection-shared/utils"

	"github.com/mitchellh/mapstructure"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"

	"github.com/khaledhikmat/threat-detection-shared/service/config"
	"github.com/khaledhikmat/threat-detection-shared/service/persistence"
	"github.com/khaledhikmat/threat-detection-shared/service/pubsub"
)

var metadataTopicSubscription = &common.Subscription{
	PubsubName: models.ThreatDetectionPubSub,
	Topic:      models.MetadataTopic,
	Route:      fmt.Sprintf("/%s", models.MetadataTopic),
}

var configSvc config.IService
var pubsubSvc pubsub.IService
var persistenceSvc persistence.IService

var metadataTopic = models.MetadataTopic

var modeProcs = map[string]func(ctx context.Context) error{
	"dapr": daprModeProc,
	"aws":  awsModeProc,
}

var indexProcs = map[string]func(ctx context.Context, clip models.RecordingClip) error{
	"database": database,
	"elastic":  elastic,
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

	persistenceSvc = persistence.New(configSvc)

	fn, ok := modeProcs[configSvc.GetRuntimeMode()]
	if !ok {
		fmt.Printf("Mode processor %s not supported\n", configSvc.GetRuntimeMode())
		return
	}

	err := fn(canxCtx)
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

	// Create a DAPR service using the app port
	s := daprd.NewService(":" + os.Getenv("APP_PORT"))
	fmt.Printf("Media Indexer - DAPR Service for %s created!\n", configSvc.GetSupportedMediaIndexType())

	// Register pub/sub metadata topic handler
	if err := s.AddTopicEventHandler(metadataTopicSubscription, daprIndexerHandler); err != nil {
		panic(err)
	}
	fmt.Printf("Media Indexer - metadata topic handler registered for %s!\n", configSvc.GetSupportedMediaIndexType())

	// Start DAPR service
	// TODO: Provide cancellation context
	if err := s.Start(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}

	return nil
}

func daprIndexerHandler(ctx context.Context, e *common.TopicEvent) (bool, error) {
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

	// Create a topic for my metadata if it does not exist
	// There could be some competition here, but we will ignore it for now
	// In higher env, queues and topics would be pre-created
	metadataTopic, err := pubsubSvc.CreateTopic(ctx, models.MetadataTopic)
	if err != nil {
		return err
	}

	// Create a queue for my topic if it does not exist
	// In higher env, queues and topics would be pre-created
	queueURL, queueARN, err := pubsubSvc.CreateQueue(ctx, fmt.Sprintf("model-indexer-queue-%s", strings.ToLower(configSvc.GetSupportedMediaIndexType())), metadataTopic)
	if err != nil {
		return err
	}

	// Register a subscription to the metadata topic
	stream, err := pubsubSvc.Subscribe(ctx, metadataTopic, queueURL, queueARN)
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
	// Determine if my media indexer is required for this clip
	fmt.Printf("Processing clip %s with indexer types %v and Indexer type %s \n", evt.ID, evt.MediaIndexerTypes, configSvc.GetSupportedMediaIndexType())
	if evt.MediaIndexerTypes == nil ||
		!utils.Contains(evt.MediaIndexerTypes, configSvc.GetSupportedMediaIndexType()) {
		fmt.Printf("Ignoring the clip because our supported index type [%s] is not needed\n", configSvc.GetSupportedMediaIndexType())
		return nil
	}

	fmt.Printf("Processing the clip because our supported index type [%s] is needed\n", configSvc.GetSupportedMediaIndexType())

	fn, ok := indexProcs[configSvc.GetSupportedMediaIndexType()]
	if !ok {
		fmt.Printf("Index processor %s not supported\n", configSvc.GetSupportedMediaIndexType())
		return fmt.Errorf("Index processor %s not supported", configSvc.GetSupportedMediaIndexType())
	}

	err := fn(ctx, evt)
	if err != nil {
		fmt.Printf("Index processor returned an error %s\n", err.Error())
		return err
	}

	return nil
}
