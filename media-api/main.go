package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"

	"github.com/khaledhikmat/threat-detection-shared/service/config"
	"github.com/khaledhikmat/threat-detection-shared/service/persistence"
	otelprovider "github.com/khaledhikmat/threat-detection-shared/telemetry/provider"
	"github.com/khaledhikmat/threat-detection/media-api/server"
)

func main() {
	rootCanx := context.Background()
	canxCtx, cancel := signal.NotifyContext(rootCanx, os.Interrupt)

	// Setup services
	configSvc := config.New()

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

	shutdown, err := otelprovider.New(canxCtx, "threat-detection-media-api", otelprovider.WithProviderType(optype))
	if err != nil {
		fmt.Println("Failed to start otel", err)
		return
	}

	defer func() {
		_ = shutdown(canxCtx)
	}()

	persistenceSvc := persistence.New(configSvc)

	// Inject into server
	server.ConfigService = configSvc
	server.PersistenceService = persistenceSvc

	port := os.Getenv("APP_PORT")
	args := os.Args[1:]
	if len(args) > 0 {
		port = args[0]
	}

	defer func() {
		cancel()
	}()

	// Launch the http server
	httpServerErr := make(chan error, 1)
	go func() {
		httpServerErr <- server.Run(canxCtx, port)
	}()

	// Wait until server exits or context is cancelled
	for {
		select {
		case err := <-httpServerErr:
			fmt.Println("http server error", err)
			return
		case <-canxCtx.Done():
			fmt.Println("application cancelled...")
			cancel()
			// Wait until downstream processors are done
			fmt.Println("wait for 2 seconds until downstream processors are cancelled...")
			time.Sleep(2 * time.Second)
			return
		}
	}
}
