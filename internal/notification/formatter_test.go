package notification

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMessageFormatterWithTemplate_ValidTemplate(t *testing.T) {
	tmpl := "{{.AppName}} {{.CurrentVersion}} -> {{.LatestVersion}}"
	f, err := NewMessageFormatterWithTemplate(tmpl)
	require.NoError(t, err)
	require.NotNil(t, f)
	require.NotNil(t, f.Template)
}

func TestNewMessageFormatterWithTemplate_InvalidTemplate(t *testing.T) {
	_, err := NewMessageFormatterWithTemplate("{{.AppName")
	require.Error(t, err)
}

func TestFormatSingleUpdate_CustomTemplate(t *testing.T) {
	tmpl := "{{.AppName}} {{.CurrentVersion}} -> {{.LatestVersion}}\n"
	f, err := NewMessageFormatterWithTemplate(tmpl)
	require.NoError(t, err)

	update := ApplicationUpdate{
		AppName:        "cert-manager",
		CurrentVersion: "1.0.0",
		LatestVersion:  "2.0.0",
	}

	result := f.formatSingleUpdate(update)
	assert.Equal(t, "cert-manager 1.0.0 -> 2.0.0\n", result)
}

func TestFormatSingleUpdate_DefaultWhenNoTemplate(t *testing.T) {
	f := NewMessageFormatter()
	update := ApplicationUpdate{
		AppName:        "cert-manager",
		Project:        "internal",
		ChartName:      "cert-manager",
		CurrentVersion: "1.0.0",
		LatestVersion:  "2.0.0",
		RepoURL:        "https://charts.jetstack.io",
	}
	result := f.formatSingleUpdate(update)
	assert.Contains(t, result, "cert-manager")
	assert.Contains(t, result, "1.0.0")
	assert.Contains(t, result, "2.0.0")
}

func TestFormatSingleUpdate_TemplateExecutionError_FallsBackToDefault(t *testing.T) {
	// Template that references a non-existent function
	tmpl := "{{call .NonExistentFunc}}"
	f, err := NewMessageFormatterWithTemplate(tmpl)
	require.NoError(t, err) // Parse succeeds

	update := ApplicationUpdate{AppName: "myapp", CurrentVersion: "1.0.0", LatestVersion: "2.0.0"}
	result := f.formatSingleUpdate(update)
	// Should fall back to default (contains app name)
	assert.Contains(t, result, "myapp")
}

func TestFormatMessages_CustomTemplate_AllFieldsAvailable(t *testing.T) {
	tmpl := "{{.AppName}}|{{.Project}}|{{.ChartName}}|{{.CurrentVersion}}|{{.LatestVersion}}|{{.RepoURL}}\n"
	f, err := NewMessageFormatterWithTemplate(tmpl)
	require.NoError(t, err)

	updates := []ApplicationUpdate{
		{
			AppName:        "app1",
			Project:        "proj",
			ChartName:      "chart",
			CurrentVersion: "1.0.0",
			LatestVersion:  "2.0.0",
			RepoURL:        "https://example.com",
		},
	}
	messages := f.FormatMessages(updates)
	require.Len(t, messages, 1)
	assert.Contains(t, messages[0], "app1|proj|chart|1.0.0|2.0.0|https://example.com")
}

func TestNewMessageFormatterWithTemplate_EmptyString(t *testing.T) {
	// Empty string should not be called (caller checks), but must not panic
	f, err := NewMessageFormatterWithTemplate("")
	require.NoError(t, err)
	// Empty template renders empty string — caller should use NewMessageFormatter() instead
	_ = f
}

func TestFormatSingleUpdate_ConstraintFields(t *testing.T) {
	tmpl := "{{if .HasUpdateOutsideConstraint}}OUTSIDE:{{.LatestVersionAll}}{{end}}\n"
	f, err := NewMessageFormatterWithTemplate(tmpl)
	require.NoError(t, err)

	update := ApplicationUpdate{
		AppName:                    "myapp",
		HasUpdateOutsideConstraint: true,
		LatestVersionAll:           "3.0.0",
		LatestVersion:              "1.5.0",
	}
	result := f.formatSingleUpdate(update)
	assert.Contains(t, result, "OUTSIDE:3.0.0")
}

func TestFormatSingleUpdate_MultilineTemplate(t *testing.T) {
	tmpl := "App: {{.AppName}}\nVersion: {{.CurrentVersion}} -> {{.LatestVersion}}\n"
	f, err := NewMessageFormatterWithTemplate(tmpl)
	require.NoError(t, err)

	update := ApplicationUpdate{AppName: "myapp", CurrentVersion: "1.0.0", LatestVersion: "2.0.0"}
	result := f.formatSingleUpdate(update)
	assert.True(t, strings.Contains(result, "App: myapp"))
	assert.True(t, strings.Contains(result, "Version: 1.0.0 -> 2.0.0"))
}
