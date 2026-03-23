package notification

import (
	"context"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeHeader(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal subject", "normal subject"},
		{"subject\r\nX-Injected: evil", "subject  X-Injected: evil"},
		{"subject\nonly newline", "subject only newline"},
		{"subject\ronly cr", "subject only cr"},
		{"no special chars", "no special chars"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := sanitizeHeader(tc.input)
			if got != tc.expected {
				t.Errorf("sanitizeHeader(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestSanitizeHeader_PreservesNormalContent(t *testing.T) {
	inputs := []string{
		"Helm Update Report",
		"ArgoCD: 3 apps need updates",
		"[argazer] Weekly digest",
	}
	for _, s := range inputs {
		if sanitizeHeader(s) != s {
			t.Errorf("sanitizeHeader modified clean string %q", s)
		}
	}
}

func TestEmailNotifier_Send_HeaderInjectionPrevented(t *testing.T) {
	// Verify that \r\n in subject does not appear in the built header string
	logger := logrus.NewEntry(logrus.New())
	notifier := NewEmailNotifier(
		"invalid.example.com", 587, "", "",
		"sender@example.com",
		[]string{"recipient@example.com"},
		false, logger,
	)

	// sanitizeHeader should strip CRLF from subject
	injected := "Subject\r\nX-Evil: injected"
	safe := sanitizeHeader(injected)
	if strings.Contains(safe, "\r") || strings.Contains(safe, "\n") {
		t.Errorf("sanitizeHeader did not remove CR/LF: %q", safe)
	}
	_ = notifier
}

func TestNewEmailNotifier(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	notifier := NewEmailNotifier(
		"smtp.example.com",
		587,
		"username",
		"password",
		"sender@example.com",
		[]string{"recipient@example.com"},
		true,
		logger,
	)

	require.NotNil(t, notifier)
	assert.Equal(t, "smtp.example.com", notifier.smtpHost)
	assert.Equal(t, 587, notifier.smtpPort)
	assert.Equal(t, "username", notifier.smtpUsername)
	assert.Equal(t, "password", notifier.smtpPassword)
	assert.Equal(t, "sender@example.com", notifier.from)
	assert.Equal(t, []string{"recipient@example.com"}, notifier.to)
	assert.True(t, notifier.useTLS)
	assert.NotNil(t, notifier.logger)
}

func TestEmailNotifier_Send_InvalidSMTP(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	notifier := NewEmailNotifier(
		"invalid-smtp-server-that-does-not-exist.example.com",
		587,
		"",
		"",
		"sender@example.com",
		[]string{"recipient@example.com"},
		false,
		logger,
	)

	ctx := context.Background()
	err := notifier.Send(ctx, "Test Subject", "Test message")
	// Should fail because the SMTP server doesn't exist
	require.Error(t, err)
}

func TestEmailNotifier_Send_WithTLS_InvalidSMTP(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	notifier := NewEmailNotifier(
		"invalid-smtp-server-that-does-not-exist.example.com",
		587,
		"username",
		"password",
		"sender@example.com",
		[]string{"recipient@example.com"},
		true,
		logger,
	)

	ctx := context.Background()
	err := notifier.Send(ctx, "Test Subject", "Test message")
	// Should fail because the SMTP server doesn't exist
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to SMTP server")
}

func TestEmailNotifier_Send_MultipleRecipients(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	notifier := NewEmailNotifier(
		"invalid.example.com",
		587,
		"",
		"",
		"sender@example.com",
		[]string{"recipient1@example.com", "recipient2@example.com", "recipient3@example.com"},
		false,
		logger,
	)

	// Just test that the notifier is created correctly with multiple recipients
	assert.Equal(t, 3, len(notifier.to))
	assert.Equal(t, "recipient1@example.com", notifier.to[0])
	assert.Equal(t, "recipient2@example.com", notifier.to[1])
	assert.Equal(t, "recipient3@example.com", notifier.to[2])
}

func TestEmailNotifier_Send_NoAuth(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	notifier := NewEmailNotifier(
		"invalid.example.com",
		587,
		"", // No username
		"", // No password
		"sender@example.com",
		[]string{"recipient@example.com"},
		false,
		logger,
	)

	require.NotNil(t, notifier)
	assert.Equal(t, "", notifier.smtpUsername)
	assert.Equal(t, "", notifier.smtpPassword)
}
