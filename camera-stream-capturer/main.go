package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"

	"github.com/khaledhikmat/threat-detection/shared/service/config"
	"github.com/khaledhikmat/threat-detection/shared/service/soicat"

	"github.com/khaledhikmat/threat-detection/camera-stream-capturer/capturer"
	"github.com/khaledhikmat/threat-detection/camera-stream-capturer/internal/fsdata"
)

//TODO:
// 1. Error Processor

func main() {
	rootCanx := context.Background()
	canxCtx, cancel := signal.NotifyContext(rootCanx, os.Interrupt)

	// Load env vars
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	configData := fsdata.GetEmbeddedConfigData()
	cfgsvc := config.New(configData)
	soicatsvc := soicat.New()

	cameras, err := soicatsvc.UncapturedCameras()
	if err != nil {
		panic(err)
	}

	if len(cameras) == 0 {
		panic(fmt.Errorf("No cameras"))
	}

	defer func() {
		cancel()
	}()

	// TODO: Launch an agent for each camera
	agentErr := make(chan error, 1)
	go func() {
		agentErr <- capturer.Run(canxCtx, cfgsvc, cameras[0])
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
