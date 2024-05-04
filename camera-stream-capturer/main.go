package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/joho/godotenv"

	"github.com/khaledhikmat/threat-detection/shared/equates"
	"github.com/khaledhikmat/threat-detection/shared/service/capturer"
	"github.com/khaledhikmat/threat-detection/shared/service/config"
	"github.com/khaledhikmat/threat-detection/shared/service/soicat"

	"github.com/khaledhikmat/threat-detection/camera-stream-capturer/agent"
	"github.com/khaledhikmat/threat-detection/camera-stream-capturer/internal/fsdata"
)

// TODO:
// - Need to create a nested context to allow the stopping of agents.
var (
	activeAgents  = 0 //TODO: Needs an atomic counter
	agentCommands map[string]chan string
	daprClient    dapr.Client
)

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
	configsvc := config.New(configData)
	soicatsvc := soicat.New()
	capturersvc := capturer.New()

	if configsvc.IsDapr() {
		// Create a DAPR client
		// Must be a global client since it is singleton
		// Hence it would be injected in actor packages as needed
		daprClient, err = dapr.NewClient()
		if err != nil {
			fmt.Println("Failed to start dapr client", err)
			return
		}
		defer daprClient.Close()

		// Inject dapr client in agents
		agent.DaprClient = daprClient
	}

	// Run a discovery processor to grab camera agents
	go func() {
		for {
			select {
			case <-canxCtx.Done():
				fmt.Printf("capturer %s discovery processor context cancelled\n", capturerName)
				return
			case <-time.After(time.Duration(10 * time.Second)): // TODO: Need a backoff time
				fmt.Printf("capturer %s discovery processor timeout to grab camera agents....\n", capturerName)
				agents, err := agents(capturerName, soicatsvc, capturersvc)
				if err != nil {
					fmt.Printf("capturer %s discovery processor error: %v\n", capturerName, err)
					continue
				}

				for _, c := range agents {
					if activeAgents < configsvc.GetCapturer().MaxCameras {
						err = soicatsvc.UpdateCamera(c) //TODO: not enough ...we might run into a race condition
						if err != nil {
							fmt.Printf("capturer %s discovery processor error: %v\n", capturerName, err)
							continue
						}

						// Create a commands channels for agent
						agentCommands[c.Name] = make(chan string)

						go func() {
							fmt.Printf("capturer %s discovery processor agent %s - starting....\n", capturerName, c.Name)
							defer close(agentCommands[c.Name])

							agentErr := agent.Run(canxCtx, configsvc, agentCommands[c.Name], capturerName, c)
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
			if configsvc.IsDapr() {
				err := daprClient.SaveState(canxCtx,
					equates.ThreatDetectionStateStore,
					fmt.Sprintf("%s_%s", "heartbeat", capturerName),
					[]byte(fmt.Sprintf("%s_%d", time.Now().UTC().Format("2006-01-02 15:04:05"), activeAgents)), nil)
				if err != nil {
					fmt.Printf("capturer %s heartbeat processor - error: %v\n", capturerName, err)
				}
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
