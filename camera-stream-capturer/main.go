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
	"github.com/khaledhikmat/threat-detection-shared/service/publisher"
	"github.com/khaledhikmat/threat-detection-shared/service/soicat"
	"github.com/khaledhikmat/threat-detection-shared/service/storage"

	"github.com/khaledhikmat/threat-detection/camera-stream-capturer/agent"
	"github.com/khaledhikmat/threat-detection/camera-stream-capturer/internal/fsdata"
)

var (
	activeAgents  = 0 //TODO: Needs an atomic counter
	agentCommands map[string]chan string
	daprClient    dapr.Client
)

var modeProcs = map[string]func(ctx context.Context, configSvc config.IService) (dapr.Client, publisher.IService, storage.IService, error){
	"dapr": daprModeProc,
	"aws":  awsModeProc,
}

func main() {
	capturerName := "capturer1" // TODO: read from the pod
	agentCommands = map[string]chan string{}

	rootCanx := context.Background()
	canxCtx, cancel := signal.NotifyContext(rootCanx, os.Interrupt)

	// Load env vars
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	// Setup services
	configData := fsdata.GetEmbeddedConfigData()
	configSvc := config.New(configData)
	soicatSvc := soicat.New()
	capturerSvc := capturer.New()

	fn, ok := modeProcs[configSvc.GetRuntime()]
	if !ok {
		fmt.Printf("Mode processor %s not supported\n", configSvc.GetRuntime())
		return
	}

	c, publisherSvc, storageSvc, err := fn(canxCtx, configSvc)
	if err != nil {
		fmt.Println("Failed to start dapr client", err)
		return
	}

	// DAPR client must be global
	daprClient = c
	defer daprClient.Close()

	// Run a discovery processor to grab camera agents
	go func() {
		for {
			select {
			case <-canxCtx.Done():
				fmt.Printf("capturer %s discovery processor context cancelled\n", capturerName)
				return
			case <-time.After(time.Duration(10 * time.Second)): // TODO: Need a backoff time
				fmt.Printf("capturer %s discovery processor timeout to grab camera agents....\n", capturerName)
				agents, err := agents(capturerName, soicatSvc, capturerSvc)
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

							agentErr := agent.Run(canxCtx, configSvc, storageSvc, publisherSvc, agentCommands[c.Name], capturerName, c)
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

	defer func() {
		cancel()
	}()

	// Wait until context is cancelled
	for {
		select {
		case <-canxCtx.Done():
			fmt.Println("context cancelled...")
			cancel()
			// Wait until downstream processors are done
			fmt.Println("wait for 5 seconds until downstream processors are cancelled...")
			time.Sleep(5 * time.Second)
			return
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

func agents(capturerName string, soicatsvc soicat.IService, capturersvc capturer.IService) ([]soicat.Camera, error) {
	agents := []soicat.Camera{}

	cameras, err := soicatsvc.Cameras()
	if err != nil {
		return agents, err
	}

	capturers, err := capturersvc.Capturers()
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

func daprModeProc(_ context.Context, configSvc config.IService) (dapr.Client, publisher.IService, storage.IService, error) {
	c, err := dapr.NewClient()
	if err != nil {
		fmt.Println("Failed to start dapr client", err)
		return nil, nil, nil, err
	}

	publisherSvc := publisher.New(c, configSvc)
	storageSvc := storage.New(c, configSvc)

	return c, publisherSvc, storageSvc, nil
}

func awsModeProc(_ context.Context, _ config.IService) (dapr.Client, publisher.IService, storage.IService, error) {
	return nil, nil, nil, fmt.Errorf("aws mode not supported")
}
