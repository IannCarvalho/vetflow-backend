package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
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

	// AdminResetToken, when set, enables the POST /admin/reset-demo endpoint,
	// which wipes and re-seeds the demo database. Callers must present this
	// value in the X-Admin-Token header. It is only honored outside
	// production (the handler and the seed layer both refuse in prod). Leave
	// it empty in production so the endpoint stays disabled.
	AdminResetToken string

	// RegisterInviteCode gates NEW-account creation in PRODUCTION only. When
	// set, a classic /auth/register must present a matching invite_code or it
	// is refused. Empty in production DISABLES classic registration entirely
	// (closed beta). Ignored outside production, so the demo/dev stay open.
	RegisterInviteCode string

	// MaxProductionUsers caps the total number of accounts in PRODUCTION as a
	// safety net against runaway signups on free-tier infra (covers classic,
	// social and OAuth account creation). <= 0 means no cap. Ignored outside
	// production.
	MaxProductionUsers int
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

		AdminResetToken: os.Getenv("ADMIN_RESET_TOKEN"),

		RegisterInviteCode: os.Getenv("REGISTER_INVITE_CODE"),
		MaxProductionUsers: parseIntEnv("MAX_PRODUCTION_USERS", 0),
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

// parseIntEnv reads an integer environment variable, falling back to
// defaultValue when unset or unparseable.
func parseIntEnv(key string, defaultValue int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
		log.Printf("Environment variable %s is not a valid integer; using default %d", key, defaultValue)
	}
	return defaultValue
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

// maskPassword hides the password in a connection string's userinfo so the URI
// can be logged safely:
//
//	mongodb+srv://app:s3cret@cluster0.example.net/vetflow
//	-> mongodb+srv://app:****@cluster0.example.net/vetflow
//
// URIs without a password are returned unchanged. It deliberately does not use
// net/url: a MongoDB URI may list several hosts ("h1:27017,h2:27017"), which
// url.Parse rejects. Reserved characters inside the credentials must be
// percent-encoded in a valid MongoDB URI, so the authority ends at the first
// '/', '?' or '#', and the userinfo ends at the last '@' inside it.
func maskPassword(uri string) string {
	schemeEnd := strings.Index(uri, "://")
	if schemeEnd < 0 {
		return uri
	}
	authorityStart := schemeEnd + len("://")
	rest := uri[authorityStart:]

	authorityEnd := strings.IndexAny(rest, "/?#")
	if authorityEnd < 0 {
		authorityEnd = len(rest)
	}
	at := strings.LastIndex(rest[:authorityEnd], "@")
	if at < 0 {
		return uri // no credentials
	}
	colon := strings.Index(rest[:at], ":")
	if colon < 0 {
		return uri // username only: nothing secret to hide
	}
	return uri[:authorityStart] + rest[:colon] + ":****" + rest[at:]
}

// String returns a string representation of the config
func (c *Config) String() string {
	return fmt.Sprintf("Config{MongoURI: %s, MongoDB: %s, Port: %s}",
		maskPassword(c.MongoURI), c.MongoDB, c.Port)
}
