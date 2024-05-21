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
	"github.com/khaledhikmat/threat-detection/media-api/internal/fsdata"
	"github.com/khaledhikmat/threat-detection/media-api/server"
)

func main() {
	rootCanx := context.Background()
	canxCtx, cancel := signal.NotifyContext(rootCanx, os.Interrupt)

	// Load env vars
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	if os.Getenv("SQLLITE_FILE_PATH") == "" {
		fmt.Printf("Failed to start - %s env var is required\n", "SQLLITE_FILE_PATH")
		return
	}

	fmt.Printf("***** 💰 SQLLITE file path: %s\n", os.Getenv("SQLLITE_FILE_PATH"))

	if os.Getenv("APP_PORT") == "" {
		fmt.Printf("Failed to start - %s env var is required\n", "APP_PORT")
		return
	}

	// Setup services
	configData := fsdata.GetEmbeddedConfigData()
	configSvc := config.New(configData)
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
