package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	MQTTBroker   string
	MQTTPort     int
	MQTTClientID string
	MQTTUsername string
	MQTTPassword string
	MQTTTopic    string

	StressWarningRatio float64
	StressDangerRatio  float64
}

var AppConfig Config

func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using default values")
	}

	AppConfig = Config{
		ServerPort:         getEnv("SERVER_PORT", "8080"),
		DBHost:             getEnv("DB_HOST", "localhost"),
		DBPort:             getEnv("DB_PORT", "5432"),
		DBUser:             getEnv("DB_USER", "postgres"),
		DBPassword:         getEnv("DB_PASSWORD", "postgres"),
		DBName:             getEnv("DB_NAME", "ancient_bridges"),
		DBSSLMode:          getEnv("DB_SSLMODE", "disable"),
		MQTTBroker:         getEnv("MQTT_BROKER", "localhost"),
		MQTTPort:           getEnvInt("MQTT_PORT", 1883),
		MQTTClientID:       getEnv("MQTT_CLIENT_ID", "bridge-alert-service"),
		MQTTUsername:       getEnv("MQTT_USERNAME", ""),
		MQTTPassword:       getEnv("MQTT_PASSWORD", ""),
		MQTTTopic:          getEnv("MQTT_TOPIC", "bridges/alerts"),
		StressWarningRatio: getEnvFloat("STRESS_WARNING_RATIO", 0.8),
		StressDangerRatio:  getEnvFloat("STRESS_DANGER_RATIO", 1.0),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value, exists := os.LookupEnv(key); exists {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}
