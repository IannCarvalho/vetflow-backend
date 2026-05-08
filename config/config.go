package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoURI   string
	MongoDB    string
	Port       string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	// Load .env file (optional)
	_ = godotenv.Load()

	return &Config{
		MongoURI: getEnv("MONGO_URI", "mongodb://mongo:27017"),
		MongoDB:  getEnv("MONGO_DB", "vetflow"),
		Port:     getEnv("PORT", "8080"),
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// String returns a string representation of the config
func (c *Config) String() string {
	return fmt.Sprintf("Config{MongoURI: %s, MongoDB: %s, Port: %s}", c.MongoURI, c.MongoDB, c.Port)
}
