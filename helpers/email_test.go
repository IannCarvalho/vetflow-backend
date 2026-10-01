package helpers

import "testing"

func TestMailerLogOnlyWhenNotConfigured(t *testing.T) {
	m := NewMailer(EmailConfig{}) // no host/from

	if m.Configured() {
		t.Fatal("expected mailer to be unconfigured when host/from are empty")
	}

	// In log-only mode Send must succeed without attempting a real connection.
	if err := m.Send("vet@example.com", "Assunto", "Corpo"); err != nil {
		t.Fatalf("expected nil error in log-only mode, got %v", err)
	}
}

func TestMailerConfigured(t *testing.T) {
	m := NewMailer(EmailConfig{Host: "smtp.example.com", From: "no-reply@example.com"})
	if !m.Configured() {
		t.Fatal("expected mailer to report configured when host and from are set")
	}
}
