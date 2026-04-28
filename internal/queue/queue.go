package queue

import (
	"sync"

	"github.com/Lin-Jiong-HDU/go-project-template/domain"
)

type queue struct {
	ch   chan domain.Task
	once sync.Once
	done chan struct{}
}

func New() domain.TaskQueue {
	return &queue{
		ch:   make(chan domain.Task, 256),
		done: make(chan struct{}),
	}
}

func (q *queue) Enqueue(task domain.Task) {
	select {
	case q.ch <- task:
	case <-q.done:
	}
}

func (q *queue) Dequeue() <-chan domain.Task {
	return q.ch
}

func (q *queue) Len() int {
	return len(q.ch)
}

func (q *queue) Close() {
	q.once.Do(func() {
		close(q.done)
		close(q.ch)
	})
}
