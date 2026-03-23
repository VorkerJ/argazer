package notification

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/sirupsen/logrus"
)

// EmailNotifier handles sending notifications via Email
type EmailNotifier struct {
	smtpHost     string
	smtpPort     int
	smtpUsername string
	smtpPassword string
	from         string
	to           []string
	useTLS       bool
	logger       *logrus.Entry
}

// NewEmailNotifier creates a new Email notifier
func NewEmailNotifier(smtpHost string, smtpPort int, smtpUsername, smtpPassword, from string, to []string, useTLS bool, logger *logrus.Entry) *EmailNotifier {
	return &EmailNotifier{
		smtpHost:     smtpHost,
		smtpPort:     smtpPort,
		smtpUsername: smtpUsername,
		smtpPassword: smtpPassword,
		from:         from,
		to:           to,
		useTLS:       useTLS,
		logger:       logger,
	}
}

// sanitizeHeader removes CR and LF characters from email header values
// to prevent header injection and SMTP injection attacks.
func sanitizeHeader(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

// Send sends an email notification (implements Notifier interface)
func (e *EmailNotifier) Send(ctx context.Context, subject, message string) error {
	// Sanitize header fields to prevent header injection and SMTP injection
	safeSubject := sanitizeHeader(subject)
	safeFrom := sanitizeHeader(e.from)
	safeTo := make([]string, len(e.to))
	for i, addr := range e.to {
		safeTo[i] = sanitizeHeader(addr)
	}

	// Prepare email headers and body
	body := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		safeFrom,
		strings.Join(safeTo, ", "),
		safeSubject,
		message,
	)

	addr := fmt.Sprintf("%s:%d", e.smtpHost, e.smtpPort)

	e.logger.WithFields(logrus.Fields{
		"smtp_host": e.smtpHost,
		"smtp_port": e.smtpPort,
		"from":      e.from,
		"to":        e.to,
		"subject":   subject,
	}).Debug("Sending email notification")

	// Refuse to send SMTP credentials over plaintext — set email_use_tls: true
	if !e.useTLS && e.smtpUsername != "" {
		return fmt.Errorf("SMTP authentication over a plaintext connection is not allowed; " +
			"set email_use_tls: true or remove SMTP credentials")
	}

	var auth smtp.Auth
	if e.smtpUsername != "" && e.smtpPassword != "" {
		auth = smtp.PlainAuth("", e.smtpUsername, e.smtpPassword, e.smtpHost)
	}

	// Send email with TLS if enabled
	if e.useTLS {
		return e.sendWithTLS(addr, auth, safeFrom, safeTo, []byte(body))
	}

	// Send without TLS
	err := smtp.SendMail(addr, auth, safeFrom, safeTo, []byte(body))
	if err == nil {
		e.logger.WithField("to", e.to).Info("Successfully sent email notification")
	}
	return err
}

// sendWithTLS sends email with TLS encryption.
// from and to are pre-sanitized by the caller.
func (e *EmailNotifier) sendWithTLS(addr string, auth smtp.Auth, from string, to []string, body []byte) error {
	// Connect to SMTP server
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			e.logger.WithError(err).Warn("Failed to close SMTP client")
		}
	}()

	// Start TLS
	tlsConfig := &tls.Config{
		ServerName: e.smtpHost,
		MinVersion: tls.VersionTLS12, // Require TLS 1.2 or higher for security
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	// Authenticate
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("failed to authenticate: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send email body
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			e.logger.WithError(err).Warn("Failed to close data writer")
		}
	}()

	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}

	e.logger.WithField("to", e.to).Info("Successfully sent email notification")
	return nil
}
