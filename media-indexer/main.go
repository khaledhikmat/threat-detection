package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/joho/godotenv"
	"github.com/khaledhikmat/threat-detection-shared/equates"
	"github.com/khaledhikmat/threat-detection-shared/utils"

	"github.com/mitchellh/mapstructure"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"

	"github.com/khaledhikmat/threat-detection-shared/service/config"
	"github.com/khaledhikmat/threat-detection-shared/service/persistence"
	"github.com/khaledhikmat/threat-detection/media-indexer/internal/fsdata"
)

var metadataTopicSubscription = &common.Subscription{
	PubsubName: equates.ThreatDetectionPubSub,
	Topic:      equates.MetadataTopic,
	Route:      fmt.Sprintf("/%s", equates.MetadataTopic),
}

// Global DAPR client
var canxCtx context.Context
var daprClient dapr.Client
var configSvc config.IService
var persistenceSvc persistence.IService

var indexProcs = map[string]func(ctx context.Context, clip equates.RecordingClip) error{
	"database": database,
}

func main() {
	rootCtx := context.Background()
	canxCtx, _ = signal.NotifyContext(rootCtx, os.Interrupt)

	// Load env vars
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Failed to start env vars", err)
		return
	}

	if os.Getenv("APP_PORT") == "" {
		fmt.Printf("Failed to start - %s env var is required\n", "APP_PORT")
		return
	}

	// Setup services
	configData := fsdata.GetEmbeddedConfigData()
	configSvc = config.New(configData)

	if !configSvc.IsDapr() && !configSvc.IsDiagrid() {
		fmt.Println("This Microservice requires that we run in DAPR or Diagrid mode", err)
		return
	}

	var c dapr.Client
	var s common.Service

	// Create a DAPR client
	// Must be a global client since it is singleton
	// Hence it would be injected in actor packages as needed
	c, err = dapr.NewClient()
	if err != nil {
		fmt.Println("Failed to start dapr client", err)
		return
	}
	daprClient = c
	defer daprClient.Close()

	persistenceSvc = persistence.New(configSvc)

	// Create a DAPR service using a hard-coded port (must match make start)
	s = daprd.NewService(":" + os.Getenv("APP_PORT"))
	fmt.Printf("Media Indexer - DAPR Service for %s created!\n", configSvc.GetSupportedMediaIndexType())

	// Register pub/sub metadata topic handler
	if err := s.AddTopicEventHandler(metadataTopicSubscription, indexerHandler); err != nil {
		panic(err)
	}
	fmt.Printf("Media Indexer - metadata topic handler registered for %s!\n", configSvc.GetSupportedMediaIndexType())

	// Start DAPR service
	// TODO: Provide cancellation context
	if err := s.Start(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

func indexerHandler(ctx context.Context, e *common.TopicEvent) (bool, error) {
	// Decode pledge
	evt := equates.RecordingClip{}
	err := mapstructure.Decode(e.Data, &evt)
	if err != nil {
		fmt.Println("Failed to decode event", err)
		return false, err
	}

	// Determine if my media indexer is required for this clip
	if evt.MediaIndexerTypes == nil ||
		!utils.Contains(evt.MediaIndexerTypes, configSvc.GetSupportedMediaIndexType()) {
		fmt.Printf("Ignoring the clip because our supported index type [%s] is not needed\n", configSvc.GetSupportedMediaIndexType())
		return false, err
	}

	fmt.Printf("Processing the clip because our supported index type [%s] is needed\n", configSvc.GetSupportedMediaIndexType())

	fn, ok := indexProcs[configSvc.GetSupportedMediaIndexType()]
	if !ok {
		fmt.Printf("Index processor %s not supported\n", configSvc.GetSupportedMediaIndexType())
		return false, err
	}

	err = fn(ctx, evt)
	if err != nil {
		fmt.Printf("Index processor returned an error %s\n", err.Error())
		return false, err
	}

	return false, nil
}
