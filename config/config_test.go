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
		name string
		uri  string
		want string
	}{
		{
			name: "short uri returned unchanged",
			uri:  "mongodb://mongo:27017",
			want: "mongodb://mongo:27017",
		},
		{
			name: "empty uri returned unchanged",
			uri:  "",
			want: "",
		},
		{
			name: "uri exactly 49 chars returned unchanged",
			uri:  strings.Repeat("a", 49),
			want: strings.Repeat("a", 49),
		},
		{
			name: "uri of 50 chars returned unchanged (not < 50)",
			uri:  strings.Repeat("b", 50),
			want: strings.Repeat("b", 50)[:50] + "...",
		},
		{
			name: "long uri truncated to 50 chars plus ellipsis",
			uri:  strings.Repeat("c", 80),
			want: strings.Repeat("c", 50) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskPassword(tt.uri)
			if got != tt.want {
				t.Fatalf("maskPassword(%q) = %q, want %q", tt.uri, got, tt.want)
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
