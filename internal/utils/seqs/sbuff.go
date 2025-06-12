package seqs

import (
	"sync"
)

type SyncBuffer[T any] struct {
	buffer   []T
	ch       chan []T
	cap      int
	overflow int
	mu       sync.RWMutex
	once     sync.Once
}

func NewCircularBuffer[T any](cap int) *SyncBuffer[T] {
	overflow := cap / 10
	if overflow < 10 {
		overflow = 10
	}
	b := &SyncBuffer[T]{
		cap:      cap,
		overflow: overflow,
		buffer:   make([]T, 0, cap+overflow),
		ch:       make(chan []T, 4),
	}
	go b.worker()
	return b
}

func (b *SyncBuffer[T]) worker() {
	for values := range b.ch {
		b.Write(values...)
	}
}

func (b *SyncBuffer[T]) AsyncWrite(values ...T) {
	b.ch <- values
}

func (b *SyncBuffer[T]) Write(values ...T) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.buffer = append(b.buffer, values...)

	n := len(b.buffer)
	if n > b.cap+b.overflow {
		newBuffer := make([]T, b.cap)
		copy(newBuffer, b.buffer[n-b.cap:])
		b.buffer = newBuffer
	}
}

func (b *SyncBuffer[T]) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.buffer)
}

func (b *SyncBuffer[T]) Read(limit int) []T {
	b.mu.RLock()
	defer b.mu.RUnlock()

	n := len(b.buffer)
	if limit < 0 {
		limit = n
	}
	limit = min(limit, n)
	result := make([]T, limit, limit+1)
	copy(result, b.buffer[n-limit:])
	return result
}

func (b *SyncBuffer[T]) ReadIndex(i int) T {
	b.mu.RLock()
	defer b.mu.RUnlock()

	n := len(b.buffer)
	if i < 0 {
		return b.buffer[n+i]
	}
	return b.buffer[min(i, n-1)]
}

func (b *SyncBuffer[T]) WithLock(f func([]T)) {
	b.mu.Lock()
	defer b.mu.Unlock()

	f(b.buffer)
}

func (b *SyncBuffer[T]) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.once.Do(func() {
		close(b.ch)
		b.buffer = nil
	})
}
