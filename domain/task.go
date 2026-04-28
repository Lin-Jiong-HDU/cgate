package domain

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusRunning    TaskStatus = "running"
	TaskStatusSucceeded  TaskStatus = "succeeded"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

type Task struct {
	ID          string     `json:"id"`
	IssueNumber int        `json:"issue_number"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	Author      string     `json:"author"`
	Repository  string     `json:"repository"`
	HTMLURL     string     `json:"html_url"`
	Status      TaskStatus `json:"status"`
	ContainerID string     `json:"container_id"`
	Log         string     `json:"log"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
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

func NewTask(payload WebhookPayload) (Task, error) {
	id, err := generateID()
	if err != nil {
		return Task{}, err
	}
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
	}, nil
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
