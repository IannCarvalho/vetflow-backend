package controllers

import (
	"context"
	"crypto/subtle"
	"strings"

	"github.com/vetflow/veterinary-api/repositories"
)

// RegistrationPolicy decides whether a NEW account may be created. It exists so
// production can be a closed, controlled environment (free-tier infra must not
// be flooded by open signups) while demo/development stay fully open — with the
// rule living in ONE place instead of being duplicated across the classic,
// social and OAuth auth controllers.
//
// All gates apply in PRODUCTION ONLY. Outside production the policy is a no-op,
// so the public demo keeps its open signup / one-click visitor login.
type RegistrationPolicy struct {
	isProduction bool
	inviteCode   string
	maxUsers     int
	userRepo     repositories.UserRepository
}

// NewRegistrationPolicy builds the policy from resolved config values.
func NewRegistrationPolicy(isProduction bool, inviteCode string, maxUsers int, userRepo repositories.UserRepository) *RegistrationPolicy {
	return &RegistrationPolicy{
		isProduction: isProduction,
		inviteCode:   strings.TrimSpace(inviteCode),
		maxUsers:     maxUsers,
		userRepo:     userRepo,
	}
}

// registrationError carries an HTTP status alongside the message so callers can
// surface the right code (403 closed/forbidden, 401 bad code, 503 cap reached).
type registrationError struct {
	status  int
	message string
}

func (e *registrationError) Error() string { return e.message }

// CheckInviteCode validates the invite code for CLASSIC registration. It only
// applies in production:
//   - invite code configured + matching code supplied -> allowed
//   - invite code configured + wrong/missing code     -> 401
//   - invite code NOT configured                      -> 403 (registration closed)
//
// The comparison is constant-time to avoid a timing oracle on the code.
func (p *RegistrationPolicy) CheckInviteCode(supplied string) *registrationError {
	if !p.isProduction {
		return nil
	}
	if p.inviteCode == "" {
		return &registrationError{status: 403, message: "registration is closed"}
	}
	supplied = strings.TrimSpace(supplied)
	if subtle.ConstantTimeCompare([]byte(supplied), []byte(p.inviteCode)) != 1 {
		return &registrationError{status: 401, message: "invalid invite code"}
	}
	return nil
}

// CheckCapacity enforces the production account cap. It applies to EVERY
// new-account path (classic, social, OAuth) so a signup method cannot bypass
// it. A non-positive cap means unlimited. Outside production it is a no-op.
func (p *RegistrationPolicy) CheckCapacity(ctx context.Context) *registrationError {
	if !p.isProduction || p.maxUsers <= 0 {
		return nil
	}
	count, err := p.userRepo.Count(ctx)
	if err != nil {
		// Fail CLOSED: if we cannot verify capacity in production, refuse the
		// new account rather than risk blowing past the cap.
		return &registrationError{status: 503, message: "cannot verify capacity, try again later"}
	}
	if count >= int64(p.maxUsers) {
		return &registrationError{status: 503, message: "registration temporarily unavailable (capacity reached)"}
	}
	return nil
}
