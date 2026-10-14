package db

import (
	"testing"
)

// TestResetDemoDatabaseRefusesInProduction verifies the most important safety
// property of the demo-reset path: it must NEVER drop/re-seed a production
// database. The production guard runs before any database access, so we can
// assert it with a nil *mongo.Database — if the guard ever regressed, the call
// would dereference nil and panic instead of returning an error, which this
// test would catch.
func TestResetDemoDatabaseRefusesInProduction(t *testing.T) {
	for _, env := range []string{"production", "prod", "PRODUCTION", "Prod"} {
		t.Run(env, func(t *testing.T) {
			t.Setenv("APP_ENV", env)
			if err := ResetDemoDatabase(nil); err == nil {
				t.Fatalf("ResetDemoDatabase must refuse to run with APP_ENV=%q, got nil error", env)
			}
		})
	}
}
