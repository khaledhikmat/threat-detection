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
	"github.com/khaledhikmat/threat-detection/model-invoker/internal/fsdata"
)

var recordingsTopicSubscription = &common.Subscription{
	PubsubName: equates.ThreatDetectionPubSub,
	Topic:      equates.RecordingsTopic,
	Route:      fmt.Sprintf("/%s", equates.RecordingsTopic),
}

// Global DAPR client
var canxCtx context.Context
var daprclient dapr.Client

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
	configsvc := config.New(configData)

	if !configsvc.IsDapr() && !configsvc.IsDiagrid() {
		fmt.Println("This Microservice requires that we run in DAPR or Diagrid mode", err)
		return
	}

	var c dapr.Client
	var s common.Service

	if configsvc.IsDapr() || configsvc.IsDiagrid() {
		// Create a DAPR client
		// Must be a global client since it is singleton
		// Hence it would be injected in actor packages as needed
		c, err = dapr.NewClient()
		if err != nil {
			fmt.Println("Failed to start dapr client", err)
			return
		}
		daprclient = c
		defer daprclient.Close()

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
}

func recordingsHandler(_ context.Context, e *common.TopicEvent) (retry bool, err error) {
	go func() {
		// Decode pledge
		evt := equates.RecordingClip{}
		err := mapstructure.Decode(e.Data, &evt)
		if err != nil {
			fmt.Println("Failed to decode event", err)
			return
		}

		fmt.Printf("Received a recording clip - LOCAL REF %s - CLOUD REF %s - PROVIDER %s - CAPTURER %s - AGENT %s\n",
			evt.LocalReference, evt.CloudReference, evt.StorageProvider, evt.Capturer, evt.Camera)
	}()

	return false, nil
}
