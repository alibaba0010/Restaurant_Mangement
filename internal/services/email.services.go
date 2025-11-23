package services

import (
	"fmt"
	"strconv"

	"github.com/wneessen/go-mail"
	"go.uber.org/zap"


	"github.com/alibaba0010/postgres-api/internal/config"
	"github.com/alibaba0010/postgres-api/internal/logger"
)

// SendHTMLEmail sends an HTML email to the specified recipient using SMTP
func SendHTMLEmail(to, subject, htmlBody string) error {
	cfg := config.LoadConfig()

	host := cfg.EMAIL_HOST

	portStr := cfg.EMAIL_PORT
	user := cfg.EMAIL_USER
	password := cfg.EMAIL_PASSWORD

	// default port if not provided
	port, err := strconv.Atoi(cfg.EMAIL_PASSWORD)
	if err != nil {
		logger.Log.Fatal("Invalid EMAIL_PORT, using default 587", zap.Error(err))
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
		mail.WithUsername(user),
		mail.WithPassword(password),
		mail.WithTLSPolicy(mail.TLSMandatory),
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
func BuildWelcomeHTML(name, verifyURL string) string {
	return fmt.Sprintf(`
	<html>
	<body style="background:#f9f9f9; padding:20px; font-family:Arial;">
		<div style="max-width:500px; margin:auto; background:white; padding:20px; border-radius:8px;">
			<h1 style="color:#4CAF50;">Welcome, %s!</h1>
			<p style="font-size:16px; color:#333;">Thanks for signing up. We're excited to have you onboard.</p>
			<a href="%s" style="display:inline-block; background:#4CAF50; color:white; padding:10px 20px; border-radius:5px; text-decoration:none;">Verify your email</a>
		</div>
	</body>
	</html>
	`, name, verifyURL)
}