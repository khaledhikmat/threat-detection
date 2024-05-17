package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/joho/godotenv"
	"github.com/khaledhikmat/threat-detection-shared/equates"

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

func main() {
	rootCtx := context.Background()
	canxCtx, _ = signal.NotifyContext(rootCtx, os.Interrupt)

	// Load env vars
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Failed to start env vars", err)
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
	s = daprd.NewService(":8081")
	fmt.Println("DAPR Service created!")

	// Register pub/sub campaigns handlers
	if err := s.AddTopicEventHandler(recordingsTopicSubscription, recordingsHandler); err != nil {
		panic(err)
	}
	fmt.Println("Recordings topic handler registered!")

	// Start DAPR service
	// TODO: Provide cancellation context
	if err := s.Start(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

func recordingsHandler(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
	// Decode pledge
	evt := equates.RecordingClip{}
	err = mapstructure.Decode(e.Data, &evt)
	if err != nil {
		fmt.Println("Failed to decode event", err)
		return
	}

	// Determine if the AI Model is available
	if evt.Analytics == nil ||
		!modelSupported(configSvc.GetSupportedAIModel(), evt.Analytics) {
		fmt.Printf("Ignoring the clip because our supported model [%s] is not needed\n", configSvc.GetSupportedAIModel())
		return
	}

	// Retrieve the recording clip from storage
	b, err := storageSvc.RetrieveRecordingClip(ctx, evt)
	if err != nil {
		fmt.Println("Failed to retrieve event's clip", err)
		return
	}

	fmt.Printf("Received a recording clip - MODEL %s - CLOUD REF %s - BYTES %d - PROVIDER %s - CAPTURER %s - AGENT %s\n",
		configSvc.GetSupportedAIModel(), evt.CloudReference, len(b), evt.StorageProvider, evt.Capturer, evt.Camera)

	// TODO: Invoke the model based on the clip
	return false, nil
}

func modelSupported(model string, models []string) bool {
	for _, v := range models {
		if v == model {
			return true
		}
	}

	return false
}
