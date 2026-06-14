package main

import (
	"context"
	"flag"
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
	"ancient-bridge-system/internal/observability"
	"ancient-bridge-system/internal/structural_simulator"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var (
	buildStamp = "dev"
	gitHash    = "local"
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
	metricsServer     *http.Server
	pprofServer       *http.Server
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
	r.Use(observability.PrometheusMiddleware())

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
			"status":      "ok",
			"version":     "1.0.0",
			"build_stamp": buildStamp,
			"git_hash":    gitHash,
			"modules": gin.H{
				"dtu_receiver":         "running",
				"structural_simulator": "running",
				"craft_identifier":     "running",
				"alarm_mqtt":           "running",
			},
		})
	})
}

func (app *App) startMetricsServer() {
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", observability.MetricsHandler().HandlerFunc())

	app.metricsServer = &http.Server{
		Addr:         ":9090",
		Handler:      metricsMux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Prometheus metrics server starting on :9090 (/metrics)")
		if err := app.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()
}

func (app *App) startPprofServer() {
	app.pprofServer = &http.Server{
		Addr:         ":6060",
		Handler:      observability.PprofHandler(),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	go func() {
		log.Printf("pprof server starting on :6060 (/debug/pprof)")
		if err := app.pprofServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("pprof server error: %v", err)
		}
	}()
}

func (app *App) Shutdown() {
	log.Println("Shutting down services...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if app.metricsServer != nil {
		app.metricsServer.Shutdown(ctx)
	}
	if app.pprofServer != nil {
		app.pprofServer.Shutdown(ctx)
	}

	app.bus.Close()
	app.alarmMQTT.Close()

	if app.server != nil {
		if err := app.server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}

	log.Println("All services stopped")
}

func runHealthCheck() {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:8080/health")
	if err != nil {
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		os.Exit(1)
	}
	os.Exit(0)
}

func main() {
	healthcheck := flag.Bool("healthcheck", false, "Run healthcheck and exit")
	flag.Parse()

	if *healthcheck {
		runHealthCheck()
	}

	log.Printf("Starting server build=%s git=%s", buildStamp, gitHash)

	app := NewApp()
	defer app.Shutdown()

	r := gin.Default()
	app.setupRoutes(r)

	port := config.AppConfig.ServerPort
	if port == "" {
		port = "8080"
	}

	app.server = &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	app.startMetricsServer()
	app.startPprofServer()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("API server starting on port %s", port)
		log.Printf("  API:      http://localhost:%s/api/v1", port)
		log.Printf("  Health:   http://localhost:%s/health", port)
		log.Printf("  Metrics:  http://localhost:9090/metrics")
		log.Printf("  pprof:    http://localhost:6060/debug/pprof")
		log.Println("Modules: dtu_receiver | structural_simulator | craft_identifier | alarm_mqtt")
		if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-quit
	log.Println("Received shutdown signal")
}
