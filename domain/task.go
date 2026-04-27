package domain

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusSucceeded TaskStatus = "succeeded"
	TaskStatusFailed    TaskStatus = "failed"
)

type Task struct {
	ID          string
	IssueNumber int
	Title       string
	Body        string
	Author      string
	Repository  string
	HTMLURL     string
	Status      TaskStatus
	ContainerID string
	Log         string
	CreatedAt   time.Time
	StartedAt   *time.Time
	FinishedAt  *time.Time
}

type WebhookPayload struct {
	Action      string   `json:"action"`
	IssueNumber int      `json:"issue_number"`
	Title       string   `json:"title"`
	Body        string   `json:"body"`
	Author      string   `json:"author"`
	Labels      []string `json:"labels"`
	URL         string   `json:"url"`
	Repository  string   `json:"repository"`
	CreatedAt   string   `json:"created_at"`
}

func NewTask(payload WebhookPayload) Task {
	id := generateID()
	return Task{
		ID:          id,
		IssueNumber: payload.IssueNumber,
		Title:       payload.Title,
		Body:        payload.Body,
		Author:      payload.Author,
		Repository:  payload.Repository,
		HTMLURL:     payload.URL,
		Status:      TaskStatusPending,
		CreatedAt:   time.Now(),
	}
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
