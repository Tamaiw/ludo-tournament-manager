// Package smtp implements ports.EmailSender using net/smtp. The same code path
// is used in dev (Mailpit on 1025) and prod (Resend via SMTP).
package smtp

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

// Config captures the SMTP credentials used by both dev (Mailpit) and prod (Resend).
type Config struct {
	Host      string
	Port      int
	Username  string
	Password  string
	From      string
	PublicURL string
}

// Sender implements ports.EmailSender.
type Sender struct{ cfg Config }

func NewSender(cfg Config) *Sender { return &Sender{cfg: cfg} }

// SendInvite emails a platform invite URL.
func (s *Sender) SendInvite(ctx context.Context, toEmail, inviteURL, inviterName string) error {
	if s.cfg.Host == "" {
		return fmt.Errorf("smtp host not configured")
	}
	subject := fmt.Sprintf("%s invited you to Ludo Tournament Manager", inviterName)
	body := fmt.Sprintf(`Hi,

%s has invited you to join the Ludo Tournament Manager.

Click the link below to set your password and accept the invitation:

%s

This link is valid for 7 days. If you weren't expecting this, please ignore.

— Ludo Tournament Manager`, inviterName, inviteURL)
	return s.send(toEmail, subject, body)
}

// SendPasswordReset emails a password reset URL.
func (s *Sender) SendPasswordReset(ctx context.Context, toEmail, resetURL, requesterName string) error {
	if s.cfg.Host == "" {
		return fmt.Errorf("smtp host not configured")
	}
	subject := fmt.Sprintf("Reset your Ludo Tournament Manager password")
	body := fmt.Sprintf(`Hi,

%s has requested a password reset on your behalf.

Click the link below to set a new password:

%s

This link is valid for 1 hour and can only be used once.

— Ludo Tournament Manager`, requesterName, resetURL)
	return s.send(toEmail, subject, body)
}

func (s *Sender) send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	msg := buildMessage(s.cfg.From, to, subject, body)
	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}
	return smtp.SendMail(addr, auth, s.cfg.From, []string{to}, []byte(msg))
}

func buildMessage(from, to, subject, body string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "From: %s\r\n", from)
	fmt.Fprintf(&sb, "To: %s\r\n", to)
	fmt.Fprintf(&sb, "Subject: %s\r\n", subject)
	fmt.Fprintf(&sb, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&sb, "Content-Type: text/plain; charset=UTF-8\r\n")
	fmt.Fprintf(&sb, "\r\n")
	fmt.Fprintf(&sb, "%s\r\n", body)
	return sb.String()
}
