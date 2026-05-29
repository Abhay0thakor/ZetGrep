package utils

import (
	"sync"
	"sync/atomic"
)

// WorkStealer is a simple work-stealing pool implementation
type WorkStealer[T any] struct {
	queues   []chan T
	size     int32
	count    int
	mu       sync.Mutex
}

func NewWorkStealer[T any](numWorkers int, queueSize int) *WorkStealer[T] {
	ws := &WorkStealer[T]{
		queues: make([]chan T, numWorkers),
		count:  numWorkers,
	}
	for i := 0; i < numWorkers; i++ {
		ws.queues[i] = make(chan T, queueSize)
	}
	return ws
}

func (ws *WorkStealer[T]) Push(item T) {
	// Round-robin distribution for initial push
	idx := atomic.AddInt32(&ws.size, 1) % int32(ws.count)
	ws.queues[idx] <- item
}

func (ws *WorkStealer[T]) GetQueue(workerID int) chan T {
	if workerID < 0 || workerID >= ws.count {
		return nil
	}
	return ws.queues[workerID]
}

func (ws *WorkStealer[T]) Close() {
	for i := 0; i < ws.count; i++ {
		close(ws.queues[i])
	}
}

// Steal allows a worker to try and find work from other queues
func (ws *WorkStealer[T]) Steal(workerID int) (T, bool) {
	var empty T
	for i := 1; i < ws.count; i++ {
		target := (workerID + i) % ws.count
		select {
		case item, ok := <-ws.queues[target]:
			if ok {
				return item, true
			}
		default:
			continue
		}
	}
	return empty, false
}
