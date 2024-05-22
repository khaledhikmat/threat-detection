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
	"github.com/khaledhikmat/threat-detection-shared/service/persistence"
)

var metadataTopicSubscription = &common.Subscription{
	PubsubName: models.ThreatDetectionPubSub,
	Topic:      models.MetadataTopic,
	Route:      fmt.Sprintf("/%s", models.MetadataTopic),
}

var configSvc config.IService
var persistenceSvc persistence.IService

var modeProcs = map[string]func(ctx context.Context, configSvc config.IService) error{
	"dapr": daprModeProc,
	"aws":  awsModeProc,
}

var indexProcs = map[string]func(ctx context.Context, clip models.RecordingClip) error{
	"database": database,
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
	configSvc = config.New()
	persistenceSvc = persistence.New(configSvc)

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

	// Create a DAPR service using a hard-coded port (must match make start)
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

func awsModeProc(ctx context.Context, configSvc config.IService) error {
	return fmt.Errorf("aws mode not supported")
}
