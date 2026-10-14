package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoURI  string
	MongoDB   string
	Port      string
	AppEnv    string
	JWTSecret string

	// Email/SMTP settings for outbound notification e-mails. When SMTPHost is
	// empty, the mailer runs in "log only" mode (no real e-mail is sent).
	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
	SMTPFrom string
	FromName string

	// Daily notification scheduler.
	EmailNotificationsEnabled bool
}

// ProductionDBName is the MongoDB database that holds real customer data. A
// non-production environment (demo/development) must never point MONGO_DB at
// this database, otherwise demo seeding and visitor logins would run against
// live records. The guard in Validate enforces this.
const ProductionDBName = "vetflow"

// IsProduction reports whether the app is running in a production environment.
func (c *Config) IsProduction() bool {
	env := strings.ToLower(c.AppEnv)
	return env == "production" || env == "prod"
}

// IsDemo reports whether the app is running as a public demo. Demo mode runs
// the exact same code as production, but against a throwaway database with
// fake data (seeded on boot) and a well-known visitor login — so it must stay
// isolated from the production database.
func (c *Config) IsDemo() bool {
	return strings.ToLower(c.AppEnv) == "demo"
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	// Load .env file (optional)
	_ = godotenv.Load()

	config := &Config{
		MongoURI:  getEnv("MONGO_URI", "mongodb://mongo:27017"),
		MongoDB:   getEnv("MONGO_DB", "vetflow"),
		Port:      getEnv("PORT", "8080"),
		AppEnv:    getEnv("APP_ENV", "development"),
		JWTSecret: os.Getenv("JWT_SECRET"),

		SMTPHost: os.Getenv("SMTP_HOST"),
		SMTPPort: getEnv("SMTP_PORT", "587"),
		SMTPUser: os.Getenv("SMTP_USER"),
		SMTPPass: os.Getenv("SMTP_PASS"),
		SMTPFrom: os.Getenv("SMTP_FROM"),
		FromName: getEnv("SMTP_FROM_NAME", "VetFlow"),

		EmailNotificationsEnabled: getEnv("EMAIL_NOTIFICATIONS_ENABLED", "true") == "true",
	}

	log.Printf("Config loaded - MONGO_URI: %s (len=%d), PORT: %s, APP_ENV: %s, SMTP configured: %t, email notifications: %t",
		maskPassword(config.MongoURI), len(config.MongoURI), config.Port, config.AppEnv,
		config.SMTPHost != "", config.EmailNotificationsEnabled)

	return config
}

// Validate checks that required configuration is present. A dedicated
// JWT_SECRET is mandatory in EVERY environment: the app must refuse to start
// without it rather than silently signing tokens with a well-known default
// secret (which would let anyone forge valid tokens for any user).
func (c *Config) Validate() error {
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET must be set (APP_ENV=%s); refusing to start without a signing secret", c.AppEnv)
	}

	// Isolation guard: a non-production environment must not point at the
	// production database. Demo seeding drops/recreates collections and
	// inserts accounts with a well-known password, so letting a demo or
	// development instance target the live "vetflow" database would corrupt
	// or expose real customer data. Fail closed at startup instead.
	if !c.IsProduction() && strings.EqualFold(c.MongoDB, ProductionDBName) {
		return fmt.Errorf(
			"refusing to start: APP_ENV=%s is non-production but MONGO_DB=%q is the production database; set MONGO_DB to a separate database (e.g. %q_demo)",
			c.AppEnv, c.MongoDB, ProductionDBName,
		)
	}

	return nil
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
