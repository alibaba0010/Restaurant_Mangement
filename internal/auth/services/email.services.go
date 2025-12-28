package services

import (
	"fmt"
	"strconv"
	"time"

	"github.com/wneessen/go-mail"
	"go.uber.org/zap"

	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/config"
)

// SendHTMLEmail sends an HTML email to the specified recipient using SMTP
func SendEmail(to, subject, htmlBody string) error {
	cfg := config.LoadConfig()

	host := cfg.EMAIL_HOST

	portStr := cfg.EMAIL_PORT
	user := cfg.EMAIL_USER
	password := cfg.EMAIL_PASSWORD

	// default port if not provided
	port, err := strconv.Atoi(cfg.EMAIL_PORT)
	if err != nil {
		// don't fatally exit on misconfigured port; fall back to 587
		logger.Log.Error("Invalid EMAIL_PORT, using default 587", zap.Error(err))
		port = 587
	}
	if portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid EMAIL_PORT %q: %w", portStr, err)
		}
		port = p
	}

	client, err := mail.NewClient(host,
		mail.WithPort(port),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(user),
		mail.WithPassword(password),
		mail.WithTLSPolicy(mail.TLSMandatory),
		mail.WithTLSPolicy(mail.TLSMandatory),
		mail.WithTimeout(10*time.Second),
	)
	if err != nil {
		return fmt.Errorf("failed to create mail client: %w", err)
	}

	msg := mail.NewMsg()
	if user != "" {
		msg.From(user)
	}
	msg.To(to)
	msg.Subject(subject)
	msg.SetBodyString(mail.TypeTextHTML, htmlBody)

	if err := client.DialAndSend(msg); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	return nil
}

// BuildWelcomeHTML returns a simple welcome HTML body. You can expand this
// to include verification links, tokens, etc.
func VerifyMailHTML(name, verifyURL string) string {
		return fmt.Sprintf(`
		<!doctype html>
		<html lang="en">
		<head>
			<meta charset="utf-8">
			<meta name="viewport" content="width=device-width,initial-scale=1">
			<title>Verify your email</title>
			<style>
				body { background:#f4f6f8; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial; margin:0; padding:0; }
				.container { max-width:600px; margin:36px auto; background:#ffffff; border-radius:8px; overflow:hidden; box-shadow:0 4px 18px rgba(0,0,0,0.06); }
				.header { background:linear-gradient(90deg,#3b82f6,#06b6d4); padding:24px; color:#fff; text-align:center; }
				.logo { font-weight:700; font-size:20px; }
				.content { padding:28px; color:#333; }
				h1 { margin:0 0 8px 0; font-size:22px; }
				p { margin:8px 0 16px 0; line-height:1.5; }
				.button { display:inline-block; background:#10b981; color:#fff; padding:12px 20px; border-radius:6px; text-decoration:none; font-weight:600; }
				.muted { color:#667085; font-size:13px; }
				.footer { background:#f8fafc; padding:16px 24px; text-align:center; font-size:13px; color:#94a3b8; }
				@media (max-width:420px) { .container { margin:12px; } .content { padding:18px; } }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<div class="logo">GourmetHub Restaurant Management Platform</div>
				</div>
				<div class="content">
					<h1>Welcome, %s 👋</h1>
					<p class="muted">You're signing up to the <strong>GourmetHub Restaurant Management Platform</strong>. To finish creating your account and secure your restaurant data, please verify your email address by clicking the button below.</p>
					<p style="font-weight:700; color:#dc2626;">Please verify your email within <strong>15 minutes</strong>; the verification link will expire after that.</p>
					<p style="text-align:center; margin:24px 0;"><a class="button" href="%s">Verify your email</a></p>
					<p class="muted">If the button doesn't work, copy and paste the following link into your browser:</p>
					<p class="muted"><a href="%s">%s</a></p>
					<hr style="border:none;border-top:1px solid #eef2f7;margin:20px 0;" />
					<p class="muted">Need help? Reply to this email or contact our support team at <a href="mailto:support@example.com">support@example.com</a>.</p>
				</div>
				<div class="footer">© %d GourmetHub Restaurant Management Platform — Manage reservations, menus and staff with ease.</div>
			</div>
		</body>
		</html>
		`, name, verifyURL, verifyURL, verifyURL, time.Now().Year())
}

// ResetPasswordMailHTML returns the HTML template for password reset emails
func ResetPasswordMailHTML(name, resetURL string) string {
	return fmt.Sprintf(`
		<!doctype html>
		<html lang="en">
		<head>
			<meta charset="utf-8">
			<meta name="viewport" content="width=device-width,initial-scale=1">
			<title>Reset your password</title>
			<style>
				body { background:#f4f6f8; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial; margin:0; padding:0; }
				.container { max-width:600px; margin:36px auto; background:#ffffff; border-radius:8px; overflow:hidden; box-shadow:0 4px 18px rgba(0,0,0,0.06); }
				.header { background:linear-gradient(90deg,#ef4444,#f97316); padding:24px; color:#fff; text-align:center; }
				.logo { font-weight:700; font-size:20px; }
				.content { padding:28px; color:#333; }
				h1 { margin:0 0 8px 0; font-size:22px; }
				p { margin:8px 0 16px 0; line-height:1.5; }
				.button { display:inline-block; background:#ef4444; color:#fff; padding:12px 20px; border-radius:6px; text-decoration:none; font-weight:600; }
				.muted { color:#667085; font-size:13px; }
				.footer { background:#f8fafc; padding:16px 24px; text-align:center; font-size:13px; color:#94a3b8; }
				.warning { background:#fef3c7; border-left:4px solid #f59e0b; padding:12px; margin:16px 0; border-radius:4px; }
				@media (max-width:420px) { .container { margin:12px; } .content { padding:18px; } }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<div class="logo">GourmetHub Restaurant Management Platform</div>
				</div>
				<div class="content">
					<h1>Reset Your Password 🔐</h1>
					<p>Hi %s,</p>
					<p class="muted">We received a request to reset your password for your <strong>GourmetHub Restaurant Management Platform</strong> account. Click the button below to create a new password.</p>
					<div class="warning">
						<p style="margin:0; color:#92400e; font-size:13px;"><strong>⚠️ Important:</strong> This link will expire in <strong>15 minutes</strong> for security reasons.</p>
					</div>
					<p style="text-align:center; margin:24px 0;"><a class="button" href="%s">Reset Password</a></p>
					<p class="muted">If the button doesn't work, copy and paste the following link into your browser:</p>
					<p class="muted"><a href="%s">%s</a></p>
					<hr style="border:none;border-top:1px solid #eef2f7;margin:20px 0;" />
					<p class="muted"><strong>Didn't request a password reset?</strong> You can safely ignore this email. Your password will remain unchanged.</p>
					<p class="muted">Need help? Reply to this email or contact our support team at <a href="mailto:support@example.com">support@example.com</a>.</p>
				</div>
				<div class="footer">© %d GourmetHub Restaurant Management Platform — Manage reservations, menus and staff with ease.</div>
			</div>
		</body>
		</html>
		`, name, resetURL, resetURL, resetURL, time.Now().Year())
}