package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/joho/godotenv"
	"github.com/khaledhikmat/threat-detection-shared/models"
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
	PubsubName: models.ThreatDetectionPubSub,
	Topic:      models.RecordingsTopic,
	Route:      fmt.Sprintf("/%s", models.RecordingsTopic),
}

// Global DAPR client
var configSvc config.IService
var publisherSvc publisher.IService
var storageSvc storage.IService

var modeProcs = map[string]func(ctx context.Context, configSvc config.IService) error{
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

	fn, ok := modeProcs[configSvc.GetRuntime()]
	if !ok {
		fmt.Printf("Mode processor %s not supported\n", configSvc.GetRuntime())
		return
	}

	err = fn(canxCtx, configSvc)
	if err != nil {
		fmt.Println("Failed to start dapr client", err)
		return
	}
}

func daprModeProc(_ context.Context, configSvc config.IService) error {
	c, err := dapr.NewClient()
	if err != nil {
		fmt.Println("Failed to start dapr client", err)
		return err
	}
	defer c.Close()

	publisherSvc = publisher.New(c, configSvc)
	storageSvc = storage.New(c, configSvc)

	// Create a DAPR service using a hard-coded port (must match make start)
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

func awsModeProc(_ context.Context, configSvc config.IService) error {
	return fmt.Errorf("aws mode not supported")
}
