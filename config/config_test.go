package config

import (
	"strings"
	"testing"
)

func TestGetEnv_ReturnsValueWhenSet(t *testing.T) {
	t.Setenv("VETFLOW_TEST_KEY", "actual-value")

	got := getEnv("VETFLOW_TEST_KEY", "fallback")
	if got != "actual-value" {
		t.Fatalf("getEnv() = %q, want %q", got, "actual-value")
	}
}

func TestGetEnv_ReturnsDefaultWhenUnset(t *testing.T) {
	// Ensure the variable is empty for this test.
	t.Setenv("VETFLOW_TEST_KEY_UNSET", "")

	got := getEnv("VETFLOW_TEST_KEY_UNSET", "fallback")
	if got != "fallback" {
		t.Fatalf("getEnv() = %q, want %q", got, "fallback")
	}
}

func TestGetEnv_ReturnsDefaultWhenEmpty(t *testing.T) {
	// An explicitly empty env var should be treated as "not set".
	t.Setenv("VETFLOW_TEST_KEY_EMPTY", "")

	got := getEnv("VETFLOW_TEST_KEY_EMPTY", "default-val")
	if got != "default-val" {
		t.Fatalf("getEnv() = %q, want %q", got, "default-val")
	}
}

func TestMaskPassword(t *testing.T) {
	tests := []struct {
		name   string
		uri    string
		want   string
		secret string // must never appear in the masked output
	}{
		{
			name:   "atlas srv uri: password masked, host/db/options kept",
			uri:    "mongodb+srv://vetflow:S3cretPassw0rd@cluster0.ab1cd.mongodb.net/vetflow?retryWrites=true&w=majority",
			want:   "mongodb+srv://vetflow:****@cluster0.ab1cd.mongodb.net/vetflow?retryWrites=true&w=majority",
			secret: "S3cretPassw0rd",
		},
		{
			name:   "short uri with password (was logged in full before)",
			uri:    "mongodb://admin:admin@localhost:27017",
			want:   "mongodb://admin:****@localhost:27017",
			secret: ":admin@",
		},
		{
			name:   "multi-host replica set",
			uri:    "mongodb://app:pw123@h1:27017,h2:27017/db?replicaSet=rs0",
			want:   "mongodb://app:****@h1:27017,h2:27017/db?replicaSet=rs0",
			secret: "pw123",
		},
		{
			name:   "percent-encoded reserved chars in the password",
			uri:    "mongodb://app:p%40ss%2Fw0rd@host/db",
			want:   "mongodb://app:****@host/db",
			secret: "p%40ss",
		},
		{
			name: "no credentials returned unchanged",
			uri:  "mongodb://mongo:27017",
			want: "mongodb://mongo:27017",
		},
		{
			name: "username only returned unchanged",
			uri:  "mongodb://reader@host:27017/db",
			want: "mongodb://reader@host:27017/db",
		},
		{
			name: "empty uri returned unchanged",
			uri:  "",
			want: "",
		},
		{
			name: "not a uri returned unchanged",
			uri:  "not-a-connection-string",
			want: "not-a-connection-string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskPassword(tt.uri)
			if got != tt.want {
				t.Fatalf("maskPassword(%q) = %q, want %q", tt.uri, got, tt.want)
			}
			if tt.secret != "" && strings.Contains(got, tt.secret) {
				t.Fatalf("maskPassword(%q) leaked %q: %q", tt.uri, tt.secret, got)
			}
		})
	}
}

func TestLoadConfig_UsesDefaults(t *testing.T) {
	t.Setenv("MONGO_URI", "")
	t.Setenv("MONGO_DB", "")
	t.Setenv("PORT", "")

	cfg := LoadConfig()

	if cfg.MongoURI != "mongodb://mongo:27017" {
		t.Errorf("MongoURI = %q, want default %q", cfg.MongoURI, "mongodb://mongo:27017")
	}
	if cfg.MongoDB != "vetflow" {
		t.Errorf("MongoDB = %q, want default %q", cfg.MongoDB, "vetflow")
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want default %q", cfg.Port, "8080")
	}
}

func TestLoadConfig_UsesEnvValues(t *testing.T) {
	t.Setenv("MONGO_URI", "mongodb://custom-host:1234")
	t.Setenv("MONGO_DB", "customdb")
	t.Setenv("PORT", "9090")

	cfg := LoadConfig()

	if cfg.MongoURI != "mongodb://custom-host:1234" {
		t.Errorf("MongoURI = %q, want %q", cfg.MongoURI, "mongodb://custom-host:1234")
	}
	if cfg.MongoDB != "customdb" {
		t.Errorf("MongoDB = %q, want %q", cfg.MongoDB, "customdb")
	}
	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9090")
	}
}

func TestConfig_String(t *testing.T) {
	cfg := &Config{
		MongoURI: "mongodb://mongo:27017",
		MongoDB:  "vetflow",
		Port:     "8080",
	}

	got := cfg.String()

	// The string representation should include each field value.
	for _, want := range []string{"mongodb://mongo:27017", "vetflow", "8080"} {
		if !strings.Contains(got, want) {
			t.Errorf("String() = %q, expected it to contain %q", got, want)
		}
	}
}

func TestConfig_Validate_ProductionRequiresJWTSecret(t *testing.T) {
	cfg := &Config{AppEnv: "production", JWTSecret: ""}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected Validate() to fail in production without JWT_SECRET")
	}
}

func TestConfig_Validate_ProductionWithJWTSecretOK(t *testing.T) {
	cfg := &Config{AppEnv: "production", JWTSecret: "a-long-random-secret"}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected Validate() to pass when JWT_SECRET is set, got %v", err)
	}
}

func TestConfig_Validate_DevelopmentRequiresJWTSecret(t *testing.T) {
	// Fail-closed in EVERY environment: even development must not fall back to
	// a well-known default signing secret.
	cfg := &Config{AppEnv: "development", JWTSecret: ""}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected Validate() to fail in development without JWT_SECRET")
	}
}

func TestConfig_Validate_DevelopmentWithJWTSecretOK(t *testing.T) {
	cfg := &Config{AppEnv: "development", JWTSecret: "a-long-random-secret"}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected Validate() to pass in development with JWT_SECRET, got %v", err)
	}
}

func TestConfig_Validate_NonProductionRejectsProductionDB(t *testing.T) {
	// A demo/development env pointed at the production database must fail to
	// start, so the demo can never write to real customer data.
	for _, env := range []string{"demo", "development", "staging"} {
		cfg := &Config{AppEnv: env, JWTSecret: "secret", MongoDB: ProductionDBName}
		if err := cfg.Validate(); err == nil {
			t.Fatalf("expected Validate() to fail for APP_ENV=%s with MONGO_DB=%q", env, ProductionDBName)
		}
	}
}

func TestConfig_Validate_NonProductionWithSeparateDBOK(t *testing.T) {
	cfg := &Config{AppEnv: "demo", JWTSecret: "secret", MongoDB: "vetflow_demo"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected Validate() to pass for demo env with a separate DB, got %v", err)
	}
}

func TestConfig_Validate_ProductionMayUseProductionDB(t *testing.T) {
	// Production is of course allowed to use the production database.
	cfg := &Config{AppEnv: "production", JWTSecret: "secret", MongoDB: ProductionDBName}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected Validate() to pass for production with production DB, got %v", err)
	}
}

func TestConfig_IsDemo(t *testing.T) {
	cases := map[string]bool{
		"demo":        true,
		"DEMO":        true,
		"production":  false,
		"development": false,
		"":            false,
	}
	for env, want := range cases {
		cfg := &Config{AppEnv: env}
		if got := cfg.IsDemo(); got != want {
			t.Errorf("IsDemo(%q) = %v, want %v", env, got, want)
		}
	}
}

func TestConfig_IsProduction(t *testing.T) {
	cases := map[string]bool{
		"production":  true,
		"prod":        true,
		"PRODUCTION":  true,
		"development": false,
		"":            false,
		"staging":     false,
	}
	for env, want := range cases {
		cfg := &Config{AppEnv: env}
		if got := cfg.IsProduction(); got != want {
			t.Errorf("IsProduction(%q) = %v, want %v", env, got, want)
		}
	}
}
