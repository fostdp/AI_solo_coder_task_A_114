package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ancient-bridge-system/internal/alarm_mqtt"
	"ancient-bridge-system/internal/config"
	"ancient-bridge-system/internal/craft_identifier"
	"ancient-bridge-system/internal/database"
	"ancient-bridge-system/internal/dtu_receiver"
	"ancient-bridge-system/internal/handlers"
	"ancient-bridge-system/internal/messaging"
	"ancient-bridge-system/internal/structural_simulator"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type App struct {
	bus               *messaging.MessageBus
	dtuReceiver       *dtu_receiver.DTUReceiver
	structuralSim     *structural_simulator.StructuralSimulator
	craftIdentifier   *craft_identifier.CraftIdentifier
	alarmMQTT         *alarm_mqtt.AlarmMQTTService
	bridgeHandler     *handlers.BridgeHandler
	analysisHandler   *handlers.AnalysisHandler
	craftHandler      *handlers.CraftHandler
	sensorDataHandler *handlers.SensorDataHandler
	server            *http.Server
}

func NewApp() *App {
	config.LoadConfig()
	database.InitDB()

	bus := messaging.NewMessageBus()

	dtuReceiver := dtu_receiver.NewDTUReceiver(bus)
	structuralSim := structural_simulator.NewStructuralSimulator(bus)
	craftIdentifier := craft_identifier.NewCraftIdentifier(bus)
	alarmMQTT := alarm_mqtt.NewAlarmMQTTService(bus)

	bridgeHandler := handlers.NewBridgeHandler()
	analysisHandler := handlers.NewAnalysisHandler(bus, alarmMQTT)
	craftHandler := handlers.NewCraftHandler(bus, dtuReceiver)
	sensorDataHandler := handlers.NewSensorDataHandler(dtuReceiver)

	return &App{
		bus:               bus,
		dtuReceiver:       dtuReceiver,
		structuralSim:     structuralSim,
		craftIdentifier:   craftIdentifier,
		alarmMQTT:         alarmMQTT,
		bridgeHandler:     bridgeHandler,
		analysisHandler:   analysisHandler,
		craftHandler:      craftHandler,
		sensorDataHandler: sensorDataHandler,
	}
}

func (app *App) setupRoutes(r *gin.Engine) {
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	r.Use(cors.New(corsConfig))

	api := r.Group("/api/v1")

	bridges := api.Group("/bridges")
	{
		bridges.GET("", app.bridgeHandler.GetAllBridges)
		bridges.POST("", app.bridgeHandler.CreateBridge)
		bridges.GET("/:id", app.bridgeHandler.GetBridge)
		bridges.GET("/:id/members", app.bridgeHandler.GetBridgeMembers)
		bridges.GET("/:id/nodes", app.bridgeHandler.GetBridgeNodes)
		bridges.GET("/:id/sensors", app.bridgeHandler.GetBridgeSensors)
		bridges.GET("/:id/stress-overview", app.bridgeHandler.GetStressOverview)
		bridges.GET("/:id/latest-analysis", app.bridgeHandler.GetLatestAnalysis)
		bridges.GET("/:id/alerts", app.bridgeHandler.GetAlerts)
		bridges.POST("/:id/alerts/:alertId/acknowledge", app.bridgeHandler.AcknowledgeAlert)
		bridges.GET("/vehicle-loads", app.bridgeHandler.GetVehicleLoads)
		bridges.GET("/yingzao-specs", app.bridgeHandler.GetYingzaoSpecs)
		bridges.GET("/sensors/:sensorId/data", app.bridgeHandler.GetSensorData)
	}

	analysis := api.Group("/analysis")
	{
		analysis.POST("/static-load", app.analysisHandler.StaticLoadAnalysis)
		analysis.POST("/moving-load", app.analysisHandler.MovingLoadAnalysis)
		analysis.GET("/structure/:id", app.analysisHandler.GetStructure)
		analysis.GET("/history/:id", app.analysisHandler.GetAnalysisHistory)
		analysis.POST("/alerts/:alertId/acknowledge", app.analysisHandler.AcknowledgeAlert)
	}

	craft := api.Group("/craft")
	{
		craft.POST("/analyze", app.craftHandler.AnalyzeCraft)
		craft.GET("/history/:id", app.craftHandler.GetCraftHistory)
		craft.GET("/wood-species", app.craftHandler.GetWoodSpeciesList)
		craft.GET("/joinery-types", app.craftHandler.GetJoineryTypes)
	}

	sensors := api.Group("/sensors")
	{
		sensors.POST("/dtu-ingest", app.sensorDataHandler.IngestDTUData)
		sensors.GET("/:sensorId/data", app.sensorDataHandler.GetSensorData)
		sensors.GET("/:sensorId/latest", app.sensorDataHandler.GetLatestSensorData)
		sensors.GET("/environmental/:id", app.sensorDataHandler.GetEnvironmentalData)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"version": "1.0.0",
			"modules": gin.H{
				"dtu_receiver":        "running",
				"structural_simulator": "running",
				"craft_identifier":    "running",
				"alarm_mqtt":          "running",
			},
		})
	})
}

func (app *App) Shutdown() {
	log.Println("Shutting down services...")

	app.bus.Close()
	app.alarmMQTT.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if app.server != nil {
		if err := app.server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}

	log.Println("All services stopped")
}

func main() {
	app := NewApp()
	defer app.Shutdown()

	r := gin.Default()
	app.setupRoutes(r)

	port := config.AppConfig.ServerPort
	if port == "" {
		port = "8080"
	}

	app.server = &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Server starting on port %s", port)
		log.Printf("API endpoint: http://localhost:%s/api/v1", port)
		log.Println("Modules loaded: dtu_receiver, structural_simulator, craft_identifier, alarm_mqtt")
		if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-quit
	log.Println("Received shutdown signal")
}
