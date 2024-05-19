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
	"github.com/khaledhikmat/threat-detection-shared/service/publisher"
	"github.com/khaledhikmat/threat-detection-shared/service/storage"
	"github.com/khaledhikmat/threat-detection/model-invoker/internal/fsdata"
)

var recordingsTopicSubscription = &common.Subscription{
	PubsubName: equates.ThreatDetectionPubSub,
	Topic:      equates.RecordingsTopic,
	Route:      fmt.Sprintf("/%s", equates.RecordingsTopic),
}

// Global DAPR client
var canxCtx context.Context
var daprClient dapr.Client
var configSvc config.IService
var publisherSvc publisher.IService
var storageSvc storage.IService

var modelProcs = map[string]func(ctx context.Context, clip equates.RecordingClip) error{
	"weapon": weapon,
	"fire":   fire,
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

	publisherSvc = publisher.New(daprClient, configSvc)
	storageSvc = storage.New(daprClient, configSvc)

	// Create a DAPR service using a hard-coded port (must match make start)
	s = daprd.NewService(":" + os.Getenv("APP_PORT"))
	fmt.Printf("Model Invoker - DAPR Service for %s created!\n", configSvc.GetSupportedAIModel())

	// Register pub/sub recordings topic handler
	if err := s.AddTopicEventHandler(recordingsTopicSubscription, recordingsHandler); err != nil {
		panic(err)
	}
	fmt.Printf("Model Invoker - recordings topic handler registered for %s!\n", configSvc.GetSupportedAIModel())

	// Start DAPR service
	// TODO: Provide cancellation context
	if err := s.Start(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

func recordingsHandler(ctx context.Context, e *common.TopicEvent) (bool, error) {
	// Decode pledge
	evt := equates.RecordingClip{}
	err := mapstructure.Decode(e.Data, &evt)
	if err != nil {
		fmt.Println("Failed to decode event", err)
		return false, err
	}

	// Determine if my AI Model is required for this clip
	if evt.Analytics == nil ||
		!utils.Contains(evt.Analytics, configSvc.GetSupportedAIModel()) {
		fmt.Printf("Ignoring the clip because our supported model [%s] is not needed\n", configSvc.GetSupportedAIModel())
		return false, err
	}

	fmt.Printf("Processing the clip because our supported model [%s] is needed\n", configSvc.GetSupportedAIModel())

	fn, ok := modelProcs[configSvc.GetSupportedAIModel()]
	if !ok {
		fmt.Printf("AI Model %s not supported\n", configSvc.GetSupportedAIModel())
		return false, err
	}

	err = fn(ctx, evt)
	if err != nil {
		fmt.Printf("AI Model processor returned an error %s\n", err.Error())
		return false, err
	}

	return false, nil
}
