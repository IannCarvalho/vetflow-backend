package helpers

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

// EmailConfig holds the SMTP settings used by the mailer.
type EmailConfig struct {
	Host     string
	Port     string
	User     string
	Pass     string
	From     string
	FromName string
}

// Mailer sends transactional e-mails. When the SMTP host is not configured it
// runs in "log only" mode so development and tests never fail on e-mail.
type Mailer struct {
	cfg EmailConfig
}

// NewMailer creates a Mailer from the given config.
func NewMailer(cfg EmailConfig) *Mailer {
	return &Mailer{cfg: cfg}
}

// Configured reports whether real SMTP delivery is available.
func (m *Mailer) Configured() bool {
	return m.cfg.Host != "" && m.cfg.From != ""
}

// Send delivers a plain-text e-mail. If SMTP is not configured, it logs the
// message instead of sending and returns nil (non-fatal in dev).
func (m *Mailer) Send(to, subject, body string) error {
	if !m.Configured() {
		log.Printf("[MAILER] (log-only, SMTP not configured) to=%s subject=%q", to, subject)
		return nil
	}

	fromHeader := m.cfg.From
	if m.cfg.FromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", m.cfg.FromName, m.cfg.From)
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("From: %s\r\n", fromHeader))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	addr := m.cfg.Host + ":" + m.cfg.Port

	var auth smtp.Auth
	if m.cfg.User != "" {
		auth = smtp.PlainAuth("", m.cfg.User, m.cfg.Pass, m.cfg.Host)
	}

	if err := smtp.SendMail(addr, auth, m.cfg.From, []string{to}, []byte(msg.String())); err != nil {
		return fmt.Errorf("failed to send email to %s: %w", to, err)
	}
	return nil
}
