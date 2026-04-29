package bootstrap

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/spf13/viper"

	"github.com/Lin-Jiong-HDU/go-project-template/api/route"
	"github.com/Lin-Jiong-HDU/go-project-template/domain"
	"github.com/Lin-Jiong-HDU/go-project-template/internal/docker"
	"github.com/Lin-Jiong-HDU/go-project-template/internal/queue"
	"github.com/Lin-Jiong-HDU/go-project-template/repository"
	"github.com/Lin-Jiong-HDU/go-project-template/usecase"
)

type App struct {
	Server *http.Server
	UC     domain.TaskUsecase
	DB     *sql.DB
}

func Init() (*App, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/app")

	viper.AutomaticEnv()

	if err := viper.BindEnv("github.webhook_secret", "GITHUB_WEBHOOK_SECRET"); err != nil {
		slog.Warn("bind env", "error", err)
	}
	if err := viper.BindEnv("github.pat", "GITHUB_PAT"); err != nil {
		slog.Warn("bind env", "error", err)
	}
	if err := viper.BindEnv("docker.settings_path", "DOCKER_SETTINGS_PATH"); err != nil {
		slog.Warn("bind env", "error", err)
	}
	if err := viper.BindEnv("docker.git_user_name", "GIT_USER_NAME"); err != nil {
		slog.Warn("bind env", "error", err)
	}
	if err := viper.BindEnv("docker.git_user_email", "GIT_USER_EMAIL"); err != nil {
		slog.Warn("bind env", "error", err)
	}

	if err := viper.ReadInConfig(); err != nil {
		slog.Warn("no config file found, using defaults and env vars")
	}

	cfg := domain.Config{
		Server: domain.ServerConfig{
			Port: viper.GetInt("server.port"),
		},
		Docker: domain.DockerConfig{
			Image:          viper.GetString("docker.image"),
			MaxConcurrency: viper.GetInt("docker.max_concurrency"),
			Timeout:        viper.GetDuration("docker.timeout"),
			MaxTurns:       viper.GetInt("docker.max_turns"),
			SettingsPath:   viper.GetString("docker.settings_path"),
			PermissionMode: viper.GetString("docker.permission_mode"),
			GitUserName:    viper.GetString("docker.git_user_name"),
			GitUserEmail:   viper.GetString("docker.git_user_email"),
		},
		Queue: domain.QueueConfig{
			MaxRetries: viper.GetInt("queue.max_retries"),
		},
		GitHub: domain.GitHubConfig{
			WebhookSecret: viper.GetString("github.webhook_secret"),
			PAT:           viper.GetString("github.pat"),
		},
	}

	// Apply defaults
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Docker.Image == "" {
		cfg.Docker.Image = "claude-code-runner:latest"
	}
	if cfg.Docker.MaxConcurrency == 0 {
		cfg.Docker.MaxConcurrency = 3
	}
	if cfg.Docker.Timeout == 0 {
		cfg.Docker.Timeout = 30 * time.Minute
	}
	if cfg.Docker.MaxTurns == 0 {
		cfg.Docker.MaxTurns = 15
	}
	if cfg.Docker.PermissionMode == "" {
		cfg.Docker.PermissionMode = "strict"
	}
	if cfg.Queue.MaxRetries == 0 {
		cfg.Queue.MaxRetries = 1
	}
	if cfg.Docker.GitUserName == "" {
		cfg.Docker.GitUserName = "cgate-bot"
	}
	if cfg.Docker.GitUserEmail == "" {
		cfg.Docker.GitUserEmail = "cgate-bot@users.noreply.github.com"
	}

	dbPath := viper.GetString("database.path")
	if dbPath == "" {
		dbPath = "./data/cgate.db"
	}
	if err := os.MkdirAll("./data", 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	db, err := repository.InitDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("init db: %w", err)
	}

	taskRepo := repository.NewTaskRepository(db)
	taskQueue := queue.New()
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	cgateURL := os.Getenv("CGATE_URL")

	runner, err := docker.NewRunner(cfg.Docker, apiKey, cfg.GitHub.PAT, cgateURL)
	if err != nil {
		return nil, fmt.Errorf("init docker runner: %w", err)
	}
	uc := usecase.NewTaskUsecase(taskRepo, taskQueue, runner, cfg.Docker)

	mux := route.NewMux(uc, cfg.GitHub.WebhookSecret)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return &App{
		Server: server,
		UC:     uc,
		DB:     db,
	}, nil
}
