package domain

import "time"

type Config struct {
	Server ServerConfig
	Docker DockerConfig
	Queue  QueueConfig
	GitHub GitHubConfig
}

type ServerConfig struct {
	Port int
}

type DockerConfig struct {
	Image          string
	MaxConcurrency int
	Timeout        time.Duration
	MaxTurns       int
	SettingsPath   string
	PermissionMode string // "strict" uses settings_path, "permissive" allows all operations
	GitUserName    string
	GitUserEmail   string
	BaseURL        string
	Model          string
}

type QueueConfig struct {
	MaxRetries int
}

type GitHubConfig struct {
	PAT             string
	WebhookSecret   string
	AllowedAuthors  []string
}
