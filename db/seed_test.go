package db

import (
	"testing"
)

// TestMaybeResetDemoOnBootNoopOutsideDemo verifies that reset-on-boot does
// nothing unless APP_ENV is exactly "demo". The mode guard runs before any
// database access, so a nil *mongo.Database must not panic for non-demo envs —
// which protects production and local development from an accidental wipe.
func TestMaybeResetDemoOnBootNoopOutsideDemo(t *testing.T) {
	for _, env := range []string{"production", "prod", "development", "", "staging"} {
		t.Run(env, func(t *testing.T) {
			t.Setenv("APP_ENV", env)
			// Must return without touching db (nil); a panic here fails the test.
			MaybeResetDemoOnBoot(nil)
		})
	}
}

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
