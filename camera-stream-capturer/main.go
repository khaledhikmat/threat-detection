package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"

	"github.com/kerberos-io/agent/machinery/src/models"
	"github.com/kerberos-io/agent/machinery/src/utils"

	"github.com/khaledhikmat/threat-detection/camera-stream-capturer/agent"
)

func main() {
	rootCanx := context.Background()
	canxCtx, cancel := signal.NotifyContext(rootCanx, os.Interrupt)

	// Load env vars
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	// Print Kerberos.io ASCII art
	utils.PrintASCIIArt()

	// Print the environment variables which include "AGENT_" as prefix.
	utils.PrintEnvironmentVariables()

	// Read the config on start, and pass it to the other
	// function and features. Please note that this might be changed
	// when saving or updating the configuration through the REST api or MQTT handler.
	var configuration models.Configuration
	configuration.Name = "my-camera-agent"
	configuration.Port = "80" // not used

	// Open this configuration either from Kerberos Agent or Kerberos Factory.
	var configDir = "."
	err = readConfig(configDir, &configuration)
	if err != nil {
		panic(err)
	}

	// We will override the configuration with the environment variables
	//configService.OverrideWithEnvironmentVariables(&configuration)

	// Printing final configuration
	utils.PrintConfiguration(&configuration)
	fmt.Printf("RTSP URL: %s\n", configuration.Config.Capture.IPCamera.RTSP)

	defer func() {
		cancel()
	}()

	// Launch the agent
	agentErr := make(chan error, 1)
	go func() {
		agentErr <- agent.Run(canxCtx, cancel, configDir, &configuration)
	}()

	// Wait until agent exits or context is cancelled
	for {
		select {
		case err := <-agentErr:
			fmt.Println("agent error", err)
			return
		case <-canxCtx.Done():
			fmt.Println("application cancelled...")
			cancel()
			// Wait until downstream processors are done
			fmt.Println("wait for 5 seconds until downstream processors are cancelled...")
			time.Sleep(5 * time.Second)
			return
		}
	}
}

func readConfig(configDirectory string, configuration *models.Configuration) error {
	jsonFile, err := os.Open(configDirectory + "/data/config/config.json")
	if err != nil {
		return err
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	err = json.Unmarshal(byteValue, configuration)
	if err != nil {
		return err
	}

	return nil
}
