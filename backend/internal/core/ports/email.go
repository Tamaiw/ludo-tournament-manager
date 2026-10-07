package ports

import "context"

// EmailSender is implemented by the SMTP adapter (Resend in prod, Mailpit in dev).
type EmailSender interface {
	SendInvite(ctx context.Context, toEmail, inviteURL, inviterName string) error
	SendPasswordReset(ctx context.Context, toEmail, resetURL, requesterName string) error
}