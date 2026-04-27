package docker

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/Lin-Jiong-HDU/go-project-template/domain"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type runner struct {
	cli         *client.Client
	cfg         domain.DockerConfig
	apiKey      string
	githubToken string
	cgateURL    string
}

func NewRunner(cfg domain.DockerConfig, apiKey, githubToken, cgateURL string) domain.DockerRunner {
	cli, _ := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	return &runner{
		cli:         cli,
		cfg:         cfg,
		apiKey:      apiKey,
		githubToken: githubToken,
		cgateURL:    cgateURL,
	}
}

func (r *runner) StartContainer(ctx context.Context, task domain.Task) (string, error) {
	timeout := r.cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Minute
	}

	_ = timeout

	prompt := fmt.Sprintf("处理 Issue #%d: %s\n%s", task.IssueNumber, task.Title, task.Body)

	env := []string{
		fmt.Sprintf("ANTHROPIC_API_KEY=%s", r.apiKey),
		fmt.Sprintf("GITHUB_TOKEN=%s", r.githubToken),
		fmt.Sprintf("CGATE_URL=%s", r.cgateURL),
	}

	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: fmt.Sprintf("/tmp/cgate/repos/%s", task.ID),
			Target: "/workspace",
		},
	}

	if r.cfg.SettingsPath != "" {
		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   r.cfg.SettingsPath,
			Target:   "/root/.claude/settings.json",
			ReadOnly: true,
		})
	}

	resp, err := r.cli.ContainerCreate(ctx, &container.Config{
		Image: r.cfg.Image,
		Env:   env,
		Cmd:   []string{"claude", "--max-turns", "15", "--prompt", prompt},
		Tty:   false,
	}, &container.HostConfig{
		Mounts: mounts,
	}, nil, nil, fmt.Sprintf("cgate-%s", task.ID))
	if err != nil {
		return "", fmt.Errorf("create container: %w", err)
	}

	if err := r.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		_ = r.cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return "", fmt.Errorf("start container: %w", err)
	}

	return resp.ID, nil
}

func (r *runner) StopContainer(ctx context.Context, containerID string) error {
	return r.cli.ContainerStop(ctx, containerID, container.StopOptions{})
}

func (r *runner) ContainerLogs(ctx context.Context, containerID string) (<-chan string, error) {
	reader, err := r.cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		return nil, fmt.Errorf("container logs: %w", err)
	}

	ch := make(chan string, 64)
	go func() {
		defer close(ch)
		defer func() { _ = reader.Close() }()

		pr, pw := io.Pipe()
		go func() {
			_, _ = stdcopy.StdCopy(pw, io.Discard, reader)
			_ = pw.Close()
		}()

		buf := make([]byte, 4096)
		for {
			n, err := pr.Read(buf)
			if n > 0 {
				ch <- string(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	return ch, nil
}

func (r *runner) WaitContainer(ctx context.Context, containerID string) (int, error) {
	statusCh, errCh := r.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case status := <-statusCh:
		return int(status.StatusCode), nil
	case err := <-errCh:
		return -1, err
	}
}

func (r *runner) IsRunning(ctx context.Context, containerID string) (bool, error) {
	inspect, err := r.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return false, err
	}
	return inspect.State.Running, nil
}
