package main

import (
	"log"

	"ancient-bridge-system/internal/alert"
	"ancient-bridge-system/internal/config"
	"ancient-bridge-system/internal/database"
	"ancient-bridge-system/internal/handlers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	config.LoadConfig()

	database.InitDB()
	defer database.CloseDB()

	alert.InitAlertService()
	defer alert.GlobalAlertService.Close()

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	setupRoutes(r)

	log.Printf("Server starting on port %s", config.AppConfig.ServerPort)
	if err := r.Run(":" + config.AppConfig.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupRoutes(r *gin.Engine) {
	bridgeHandler := handlers.NewBridgeHandler()
	analysisHandler := handlers.NewAnalysisHandler()
	craftHandler := handlers.NewCraftHandler()
	sensorHandler := handlers.NewSensorDataHandler()

	api := r.Group("/api/v1")
	{
		bridges := api.Group("/bridges")
		{
			bridges.GET("", bridgeHandler.GetAllBridges)
			bridges.POST("", bridgeHandler.CreateBridge)
			bridges.GET("/:id", bridgeHandler.GetBridge)
			bridges.GET("/:id/members", bridgeHandler.GetBridgeMembers)
			bridges.GET("/:id/nodes", bridgeHandler.GetBridgeNodes)
			bridges.GET("/:id/sensors", bridgeHandler.GetBridgeSensors)
			bridges.GET("/:id/stress-overview", bridgeHandler.GetStressOverview)
			bridges.GET("/:id/latest-analysis", bridgeHandler.GetLatestAnalysis)
			bridges.GET("/:id/alerts", bridgeHandler.GetAlerts)
			bridges.POST("/:id/alerts/:alertId/acknowledge", bridgeHandler.AcknowledgeAlert)
			bridges.GET("/:id/environmental", sensorHandler.GetEnvironmentalData)
		}

		analysis := api.Group("/analysis")
		{
			analysis.POST("/static-load", analysisHandler.StaticLoadAnalysis)
			analysis.POST("/moving-load", analysisHandler.MovingLoadAnalysis)
			analysis.GET("/history/:id", analysisHandler.GetAnalysisHistory)
			analysis.GET("/detail/:analysisId", analysisHandler.GetAnalysisDetail)
			analysis.GET("/structure/:id", analysisHandler.GetBridgeStructure)
		}

		craft := api.Group("/craft")
		{
			craft.POST("/analyze", craftHandler.AnalyzeCraft)
			craft.GET("/history/:id", craftHandler.GetCraftHistory)
			craft.GET("/wood-species", craftHandler.GetWoodSpeciesList)
			craft.GET("/joinery-types", craftHandler.GetJoineryTypes)
		}

		sensors := api.Group("/sensors")
		{
			sensors.POST("/dtu-ingest", sensorHandler.IngestDTUData)
			sensors.GET("/:sensorId/data", sensorHandler.GetSensorData)
			sensors.GET("/:sensorId/latest", sensorHandler.GetLatestSensorData)
		}

		api.GET("/vehicle-loads", bridgeHandler.GetVehicleLoads)
		api.GET("/yingzao-specs", bridgeHandler.GetYingzaoSpecs)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "ancient-bridge-system",
		})
	})
}
