package queue

import (
	"sync"

	"github.com/Lin-Jiong-HDU/go-project-template/domain"
)

type queue struct {
	ch     chan domain.Task
	once   sync.Once
	done   chan struct{}
	closed bool
	mu     sync.RWMutex
}

func New() domain.TaskQueue {
	return &queue{
		ch:   make(chan domain.Task, 256),
		done: make(chan struct{}),
	}
}

func (q *queue) Enqueue(task domain.Task) {
	q.mu.RLock()
	if q.closed {
		q.mu.RUnlock()
		return
	}
	defer q.mu.RUnlock()
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
		q.mu.Lock()
		q.closed = true
		q.mu.Unlock()
		close(q.done)
		close(q.ch)
	})
}
