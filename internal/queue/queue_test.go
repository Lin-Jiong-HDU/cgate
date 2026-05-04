package queue_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/Lin-Jiong-HDU/go-project-template/domain"
	"github.com/Lin-Jiong-HDU/go-project-template/internal/queue"
)

func TestQueue_EnqueueDequeue(t *testing.T) {
	t.Parallel()
	q := queue.New()
	defer q.Close()

	task := domain.Task{ID: "test-1"}
	q.Enqueue(task)

	got := <-q.Dequeue()
	if got.ID != "test-1" {
		t.Errorf("expected ID test-1, got %s", got.ID)
	}
}

func TestQueue_DequeueBlocksWhenEmpty(t *testing.T) {
	t.Parallel()
	q := queue.New()
	defer q.Close()

	done := make(chan struct{})
	var received atomic.Int32
	go func() {
		got := <-q.Dequeue()
		_ = got
		received.Store(1)
		close(done)
	}()

	if received.Load() == 1 {
		t.Error("Dequeue should block when queue is empty")
	}

	q.Enqueue(domain.Task{ID: "unblock"})
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("expected Dequeue to unblock after Enqueue")
	}
	if received.Load() != 1 {
		t.Error("expected Dequeue to unblock after Enqueue")
	}
}

func TestQueue_Len(t *testing.T) {
	t.Parallel()
	q := queue.New()
	defer q.Close()

	if q.Len() != 0 {
		t.Errorf("expected Len 0, got %d", q.Len())
	}

	q.Enqueue(domain.Task{ID: "1"})
	q.Enqueue(domain.Task{ID: "2"})

	if q.Len() != 2 {
		t.Errorf("expected Len 2, got %d", q.Len())
	}
}

func TestQueue_FIFOOrder(t *testing.T) {
	t.Parallel()
	q := queue.New()
	defer q.Close()

	q.Enqueue(domain.Task{ID: "first"})
	q.Enqueue(domain.Task{ID: "second"})

	got1 := <-q.Dequeue()
	got2 := <-q.Dequeue()

	if got1.ID != "first" {
		t.Errorf("expected first, got %s", got1.ID)
	}
	if got2.ID != "second" {
		t.Errorf("expected second, got %s", got2.ID)
	}
}
