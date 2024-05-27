package server

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/khaledhikmat/threat-detection-shared/service/config"
	"github.com/khaledhikmat/threat-detection-shared/service/persistence"
)

const (
	DefaultPageSize = 10
)

// Injected DAPR client and other services
var ConfigService config.IService
var PersistenceService persistence.IService

type ginWithContext func(ctx context.Context) error

func Run(canxCtx context.Context, port string) error {
	//=========================
	// ROUTER
	//=========================
	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://example.com"} //TODO: Update
	config.AllowCredentials = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "OPTIONS", "DELETE"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	// Set function map if any...

	// Link up templates and static files
	r.LoadHTMLGlob("./templates/**/*")
	r.Static("/static", "./static")

	//=========================
	// Setup Home ROUTES
	//=========================
	homeRoutes(canxCtx, r)

	f := cancellableGin(canxCtx, r, port)
	return f(canxCtx)
}

func cancellableGin(_ context.Context, r *gin.Engine, port string) ginWithContext {
	return func(ctx context.Context) error {
		go func() {
			err := r.Run(":" + port)
			if err != nil {
				fmt.Println("Server start error...exiting", err)
				return
			}
		}()

		// Wait
		for {
			select {
			case <-ctx.Done():
				fmt.Println("Server context cancelled...existing!!!")
				return ctx.Err()
			case <-time.After(time.Duration(100 * time.Second)):
				fmt.Println("Timeout....do something periodic here!!!")
			}
		}
	}
}
