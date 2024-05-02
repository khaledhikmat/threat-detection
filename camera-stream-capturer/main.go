package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"

	"github.com/khaledhikmat/threat-detection/shared/service/capturer"
	"github.com/khaledhikmat/threat-detection/shared/service/config"
	"github.com/khaledhikmat/threat-detection/shared/service/soicat"

	"github.com/khaledhikmat/threat-detection/camera-stream-capturer/agent"
	"github.com/khaledhikmat/threat-detection/camera-stream-capturer/internal/fsdata"
)

// TODO:
// 1. Need an API to command each agent: start, stop, pause, resume and pull stats, etc.
// 2. Need to create a nested context to allow the stopping of agents.
var (
	activeAgents  = 0
	agentCommands map[string]chan string
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
						commandsStream := agentCommands[c.Name]

						go func() {
							fmt.Printf("capturer %s discovery processor agent %s - starting....\n", capturerName, c.Name)
							defer close(commandsStream)

							agentErr := agent.Run(canxCtx, commandsStream, capturerName, c)
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
			// TODO: This will be in the form of key/val store
			// where the key = capturer_name = pod name
			// and the value = timestamp + number of agents
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
