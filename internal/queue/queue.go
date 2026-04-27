package queue

import (
	"sync"

	"github.com/Lin-Jiong-HDU/go-project-template/domain"
)

type queue struct {
	ch   chan domain.Task
	once sync.Once
}

func New() domain.TaskQueue {
	return &queue{
		ch: make(chan domain.Task, 256),
	}
}

func (q *queue) Enqueue(task domain.Task) {
	q.ch <- task
}

func (q *queue) Dequeue() <-chan domain.Task {
	return q.ch
}

func (q *queue) Len() int {
	return len(q.ch)
}

func (q *queue) Close() {
	q.once.Do(func() {
		close(q.ch)
	})
}
