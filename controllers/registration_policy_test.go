package controllers

import (
	"context"
	"testing"

	"github.com/vetflow/veterinary-api/models"
)

func TestPolicyNoopOutsideProduction(t *testing.T) {
	repo := newFakeUserRepo()
	p := NewRegistrationPolicy(false, "secret", 1, repo)
	// Fill past the "cap" to prove it is ignored outside production.
	_ = repo.Create(context.Background(), &models.User{Email: "a@x.com"})
	_ = repo.Create(context.Background(), &models.User{Email: "b@x.com"})

	if err := p.CheckInviteCode(""); err != nil {
		t.Fatalf("invite check must be no-op outside production, got %v", err)
	}
	if err := p.CheckCapacity(context.Background()); err != nil {
		t.Fatalf("capacity check must be no-op outside production, got %v", err)
	}
}

func TestPolicyInviteCodeInProduction(t *testing.T) {
	repo := newFakeUserRepo()

	// No invite code configured -> registration closed (403).
	closed := NewRegistrationPolicy(true, "", 0, repo)
	if err := closed.CheckInviteCode("anything"); err == nil || err.status != 403 {
		t.Fatalf("empty invite code should close registration (403), got %v", err)
	}

	// Invite code configured.
	p := NewRegistrationPolicy(true, "let-me-in", 0, repo)
	if err := p.CheckInviteCode("wrong"); err == nil || err.status != 401 {
		t.Fatalf("wrong invite code should be 401, got %v", err)
	}
	if err := p.CheckInviteCode(""); err == nil || err.status != 401 {
		t.Fatalf("missing invite code should be 401, got %v", err)
	}
	if err := p.CheckInviteCode("let-me-in"); err != nil {
		t.Fatalf("correct invite code should pass, got %v", err)
	}
	// Surrounding whitespace is trimmed.
	if err := p.CheckInviteCode("  let-me-in  "); err != nil {
		t.Fatalf("correct invite code with whitespace should pass, got %v", err)
	}
}

func TestPolicyCapacityInProduction(t *testing.T) {
	repo := newFakeUserRepo()
	p := NewRegistrationPolicy(true, "", 2, repo)

	// Below cap: allowed.
	if err := p.CheckCapacity(context.Background()); err != nil {
		t.Fatalf("below cap should pass, got %v", err)
	}
	_ = repo.Create(context.Background(), &models.User{Email: "a@x.com"})
	if err := p.CheckCapacity(context.Background()); err != nil {
		t.Fatalf("at count 1 of cap 2 should pass, got %v", err)
	}
	// At cap: refused with 503.
	_ = repo.Create(context.Background(), &models.User{Email: "b@x.com"})
	if err := p.CheckCapacity(context.Background()); err == nil || err.status != 503 {
		t.Fatalf("at cap should be 503, got %v", err)
	}

	// Cap <= 0 means unlimited.
	unlimited := NewRegistrationPolicy(true, "", 0, repo)
	if err := unlimited.CheckCapacity(context.Background()); err != nil {
		t.Fatalf("cap <= 0 means unlimited, got %v", err)
	}
}
