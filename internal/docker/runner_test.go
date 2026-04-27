package docker_test

import (
	"context"
	"testing"

	"github.com/Lin-Jiong-HDU/go-project-template/domain"
	"github.com/Lin-Jiong-HDU/go-project-template/internal/docker"
)

func TestNewRunner_InvalidImage(t *testing.T) {
	t.Parallel()
	cfg := domain.DockerConfig{
		Image:          "nonexistent-image-that-does-not-exist:latest",
		MaxConcurrency: 1,
		Timeout:        0,
		SettingsPath:   "/dev/null",
	}
	r := docker.NewRunner(cfg, "fake-api-key", "fake-github-token", "http://localhost:8080")

	_, err := r.StartContainer(context.Background(), domain.Task{
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
		SettingsPath:   "/dev/null",
	}
	r := docker.NewRunner(cfg, "", "", "")

	err := r.StopContainer(context.Background(), "nonexistent-container-id")
	if err == nil {
		t.Error("expected error stopping nonexistent container")
	}
}
