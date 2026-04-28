package domain

import "errors"

var (
	ErrNotFound        = errors.New("task not found")
	ErrActiveTaskExists = errors.New("issue already has an active task")
)
