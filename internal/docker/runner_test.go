package docker_test

import (
	"context"
	"testing"

	"github.com/Lin-Jiong-HDU/go-project-template/domain"
	"github.com/Lin-Jiong-HDU/go-project-template/internal/docker"
)

func TestSanitizeEnvValue_RemovesControlChars(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"removes null bytes", "hello\x00world", "helloworld"},
		{"removes control chars", "test\x01\x02\x03value", "testvalue"},
		{"keeps newlines", "line1\nline2", "line1\nline2"},
		{"keeps tabs", "col1\tcol2", "col1\tcol2"},
		{"keeps carriage returns", "line1\r\nline2", "line1\r\nline2"},
		{"empty string", "", ""},
		{"clean string", "hello world", "hello world"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := docker.SanitizeEnvValue(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeEnvValue(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSanitizeShellValue_StripsCommandSubstitution(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"strips dollar paren", "$(rm -rf /)", "rm -rf /)"},
		{"strips backticks", "`cat /etc/passwd`", "cat /etc/passwd"},
		{"strips both", "$(whoami)`id`", "whoami)id"},
		{"clean title", "Fix bug [claude bot]", "Fix bug [claude bot]"},
		{"nested command", "$(echo $(secret))", "echo secret))"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := docker.SanitizeShellValue(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeShellValue(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNewRunner_InvalidImage(t *testing.T) {
	t.Parallel()
	cfg := domain.DockerConfig{
		Image:          "nonexistent-image-that-does-not-exist:latest",
		MaxConcurrency: 1,
		Timeout:        0,
		MaxTurns:       15,
		PermissionMode: "strict",
		SettingsPath:   "/dev/null",
	}
	r, err := docker.NewRunner(cfg, "fake-api-key", "fake-github-token", "http://localhost:8080", "", "")
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}

	_, err = r.StartContainer(context.Background(), domain.Task{
		ID:          "test-id",
		IssueNumber: 1,
		Title:       "test",
		Repository:  "owner/repo",
	})
	if err == nil {
		t.Error("expected error with nonexistent image")
	}
}

func TestNewRunner_StopNonexistent(t *testing.T) {
	t.Parallel()
	cfg := domain.DockerConfig{
		Image:          "alpine:latest",
		MaxConcurrency: 1,
		MaxTurns:       15,
		PermissionMode: "strict",
		SettingsPath:   "/dev/null",
	}
	r, err := docker.NewRunner(cfg, "", "", "", "", "")
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}

	err = r.StopContainer(context.Background(), "nonexistent-container-id")
	if err == nil {
		t.Error("expected error stopping nonexistent container")
	}
}

func TestCleanupTask_NonexistentContainer_NoError(t *testing.T) {
	t.Parallel()
	cfg := domain.DockerConfig{Image: "alpine:latest", MaxTurns: 15}
	r, err := docker.NewRunner(cfg, "", "", "", "", "")
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}

	err = r.CleanupTask(context.Background(), "nonexistent-task", "nonexistent-container")
	if err != nil {
		t.Errorf("CleanupTask should not fail for missing container: %v", err)
	}
}

func TestCleanupTask_EmptyContainerID_NoError(t *testing.T) {
	t.Parallel()
	cfg := domain.DockerConfig{Image: "alpine:latest", MaxTurns: 15}
	r, err := docker.NewRunner(cfg, "", "", "", "", "")
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}

	err = r.CleanupTask(context.Background(), "some-task-id", "")
	if err != nil {
		t.Errorf("CleanupTask should not fail with empty containerID: %v", err)
	}
}
