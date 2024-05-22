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
	"github.com/khaledhikmat/threat-detection-shared/service/storage"
	"github.com/khaledhikmat/threat-detection/alert-notifier/internal/fsdata"
)

var alertTopicSubscription = &common.Subscription{
	PubsubName: models.ThreatDetectionPubSub,
	Topic:      models.AlertTopic,
	Route:      fmt.Sprintf("/%s", models.AlertTopic),
}

// Global DAPR client
var configSvc config.IService
var storageSvc storage.IService

var modeProcs = map[string]func(ctx context.Context, configSvc config.IService) error{
	"dapr": daprModeProc,
	"aws":  awsModeProc,
}

var alertProcs = map[string]func(ctx context.Context, clip models.RecordingClip) error{
	"ccure": ccure,
	"snow":  snow,
	"pers":  pers,
	"slack": slack,
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

	storageSvc = storage.New(c, configSvc)

	// Create a DAPR service using a hard-coded port (must match make start)
	s := daprd.NewService(":" + os.Getenv("APP_PORT"))
	fmt.Printf("Media Indexer - DAPR Service for %s created!\n", configSvc.GetSupportedAlertType())

	// Register pub/sub metadata topic handler
	if err := s.AddTopicEventHandler(alertTopicSubscription, daprAlertHandler); err != nil {
		panic(err)
	}
	fmt.Printf("Media Indexer - metadata topic handler registered for %s!\n", configSvc.GetSupportedAlertType())

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

	// Determine if my alert notifier is required for this clip
	if evt.AlertTypes == nil ||
		!utils.Contains(evt.AlertTypes, configSvc.GetSupportedAlertType()) {
		fmt.Printf("Ignoring the clip because our supported alert type [%s] is not needed\n", configSvc.GetSupportedAlertType())
		return false, err
	}

	fmt.Printf("Processing the clip because our supported alert type [%s] is needed\n", configSvc.GetSupportedAlertType())

	fn, ok := alertProcs[configSvc.GetSupportedAlertType()]
	if !ok {
		fmt.Printf("Alert processor %s not supported\n", configSvc.GetSupportedAlertType())
		return false, err
	}

	err = fn(ctx, evt)
	if err != nil {
		fmt.Printf("Alert processor returned an error %s\n", err.Error())
		return false, err
	}

	return false, nil
}

func awsModeProc(_ context.Context, _ config.IService) error {
	return fmt.Errorf("aws mode not supported")
}
