package config

import (
	"fmt"
	"log"
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

	config := &Config{
		MongoURI: getEnv("MONGO_URI", "mongodb://mongo:27017"),
		MongoDB:  getEnv("MONGO_DB", "vetflow"),
		Port:     getEnv("PORT", "8080"),
	}
	
	log.Printf("Config loaded - MONGO_URI: %s (len=%d), PORT: %s", 
		maskPassword(config.MongoURI), len(config.MongoURI), config.Port)
	
	return config
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		log.Printf("Environment variable %s is set (len=%d)", key, len(value))
		return value
	}
	log.Printf("Environment variable %s not found, using default", key)
	return defaultValue
}

// maskPassword masks the password in the connection string for logging
func maskPassword(uri string) string {
	if len(uri) < 50 {
		return uri
	}
	return uri[:50] + "..."
}

// String returns a string representation of the config
func (c *Config) String() string {
	return fmt.Sprintf("Config{MongoURI: %s, MongoDB: %s, Port: %s}", 
		maskPassword(c.MongoURI), c.MongoDB, c.Port)
}
