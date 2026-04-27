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
	SettingsPath   string
	GitUserName    string
	GitUserEmail   string
}

type QueueConfig struct {
	MaxRetries int
}

type GitHubConfig struct {
	PAT           string
	WebhookSecret string
}
