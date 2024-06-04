package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/joho/godotenv"

	"github.com/khaledhikmat/threat-detection-shared/models"
	"github.com/khaledhikmat/threat-detection-shared/service/capturer"
	"github.com/khaledhikmat/threat-detection-shared/service/config"
	"github.com/khaledhikmat/threat-detection-shared/service/pubsub"
	"github.com/khaledhikmat/threat-detection-shared/service/soicat"
	"github.com/khaledhikmat/threat-detection-shared/service/storage"
	otelprovider "github.com/khaledhikmat/threat-detection-shared/telemetry/provider"

	"github.com/khaledhikmat/threat-detection/camera-stream-capturer/agent"
)

var (
	activeAgents  = 0 //TODO: Needs an atomic counter
	agentCommands map[string]chan string
)

var daprClient dapr.Client
var configSvc config.IService
var soicatSvc soicat.IService
var capturerSvc capturer.IService
var pubsubSvc pubsub.IService
var storageSvc storage.IService

var modeProcs = map[string]func(ctx context.Context, capturer string) error{
	"dapr": daprModeProc,
	"aws":  awsModeProc,
}

var recordingsTopic = models.RecordingsTopic

func main() {
	capturerName := "capturer1" // TODO: read from the pod
	agentCommands = map[string]chan string{}

	rootCanx := context.Background()
	canxCtx, _ := signal.NotifyContext(rootCanx, os.Interrupt)

	// Setup services
	configSvc = config.New()
	soicatSvc = soicat.New(configSvc)
	capturerSvc = capturer.New()

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

	otelServiceName := "threat-detection-camera-stream-capturer"
	fmt.Printf("About to start otel with provider %s on service %s\n", optype, otelServiceName)
	shutdown, err := otelprovider.New(canxCtx, otelServiceName, otelprovider.WithProviderType(optype))
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

	err = fn(canxCtx, capturerName)
	if err != nil {
		fmt.Printf("Failed to start mode processor %s %v\n", configSvc.GetRuntimeMode(), err)
		return
	}
}

func daprModeProc(ctx context.Context, capturer string) error {
	c, err := dapr.NewClient()
	if err != nil {
		fmt.Println("Failed to start dapr client", err)
		return err
	}
	defer c.Close()

	daprClient = c
	pubsubSvc = pubsub.NewDaprPubsub(daprClient, configSvc)
	// WARNING I am using AWS storage while in DAPR runtime mode because I can store to S3
	// storageSvc = storage.NewDaprStorage(daprClient, configSvc)
	storageSvc = storage.NewAwsStorage(configSvc)

	return runProc(ctx, capturer)
}

func awsModeProc(ctx context.Context, capturer string) error {
	pubsubSvc = pubsub.NewAwsPubsub(configSvc)
	storageSvc = storage.NewAwsStorage(configSvc)

	// Create a topic for my recordings if it does not exist
	// There could be some competition here, but we will ignore it for now
	// In higher env, queues and topics would be pre-created
	rt, err := pubsubSvc.CreateTopic(ctx, models.RecordingsTopic)
	if err != nil {
		return err
	}

	fmt.Printf("**** Created a topic %s\n", rt)
	recordingsTopic = rt

	return runProc(ctx, capturer)
}

func runProc(canxCtx context.Context, capturerName string) error {
	// Run a discovery processor to grab camera agents
	go func() {
		for {
			select {
			case <-canxCtx.Done():
				fmt.Printf("capturer %s discovery processor context cancelled\n", capturerName)
				return
			case <-time.After(time.Duration(10 * time.Second)): // TODO: Need a backoff time
				fmt.Printf("capturer %s discovery processor timeout to grab camera agents....\n", capturerName)
				agents, err := agents(capturerName)
				if err != nil {
					fmt.Printf("capturer %s discovery processor error: %v\n", capturerName, err)
					continue
				}

				for _, c := range agents {
					if activeAgents < configSvc.GetCapturer().MaxCameras {
						err = soicatSvc.UpdateCamera(c) //TODO: not enough ...we might run into a race condition
						if err != nil {
							fmt.Printf("capturer %s discovery processor error: %v\n", capturerName, err)
							continue
						}

						// Create a commands channels for agent
						agentCommands[c.Name] = make(chan string)

						go func() {
							fmt.Printf("capturer %s discovery processor agent %s - starting....\n", capturerName, c.Name)
							defer close(agentCommands[c.Name])

							agentErr := agent.Run(canxCtx, configSvc, storageSvc, pubsubSvc, recordingsTopic, agentCommands[c.Name], capturerName, c)
							if agentErr != nil {
								fmt.Printf("capturer %s discovery processor agent: %s - start error: %v\n", capturerName, c.Name, agentErr)
							}
						}()

						activeAgents++
					}
				}
			}
		}
	}()

	// Wait until context is cancelled
	for {
		select {
		case <-canxCtx.Done():
			fmt.Println("context cancelled...")
			// Wait until downstream processors are done
			fmt.Println("wait for 5 seconds until downstream processors are cancelled...")
			time.Sleep(5 * time.Second)
			return nil
		case <-time.After(time.Duration(20 * time.Second)):
			fmt.Printf("capturer %s heartbeat processor timeout to send heartbeat....\n", capturerName)
			// Send a heartbeat signal
			err := storageSvc.StoreKeyValue(canxCtx,
				models.ThreatDetectionStateStore,
				fmt.Sprintf("%s_%s", "heartbeat", capturerName),
				fmt.Sprintf("%s_%d", time.Now().UTC().Format("2006-01-02 15:04:05"), activeAgents))
			if err != nil {
				fmt.Printf("capturer %s heartbeat processor - error: %v\n", capturerName, err)
			}
		}
	}
}

func agents(capturerName string) ([]soicat.Camera, error) {
	agents := []soicat.Camera{}

	cameras, err := soicatSvc.Cameras()
	if err != nil {
		return agents, err
	}

	capturers, err := capturerSvc.Capturers()
	if err != nil {
		return agents, err
	}

	for _, camera := range cameras {
		// If the camera capturer is the same as this capturer, break
		if camera.Capturer == capturerName {
			continue
		}

		// Locate the camera's capturer to see if in the list of alive capturers
		for _, capturer := range capturers {
			if capturer.Name == camera.Capturer {
				continue
			}
		}

		// The camera's capturer is dead or missing, grab it
		camera.Capturer = capturerName
		agents = append(agents, camera)
	}

	return agents, err
}
