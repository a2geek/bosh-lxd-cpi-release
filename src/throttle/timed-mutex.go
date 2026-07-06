package throttle

import (
	"context"
	"time"
)

// TimedMutex is a mutex with timeout capabilities.
// From: https://oneuptime.com/blog/post/2026-01-23-go-mutex/view

type TimedMutex struct {
	ch   chan struct{}
	when time.Time
}

func NewTimedMutex() *TimedMutex {
	return &TimedMutex{
		ch: make(chan struct{}, 1),
	}
}

func (m *TimedMutex) Lock() {
	m.ch <- struct{}{}
}

func (m *TimedMutex) Unlock() {
	<-m.ch
}

func (m *TimedMutex) UnlockWithTimeout(timeout time.Duration) bool {
	if time.Now().After(m.when.Add(timeout)) {
		m.Unlock()
		return true
	}
	return false
}

func (m *TimedMutex) LockWithTimeout(timeout time.Duration) bool {
	select {
	case m.ch <- struct{}{}:
		m.when = time.Now()
		return true
	case <-time.After(timeout):
		return false
	}
}

func (m *TimedMutex) LockWithContext(ctx context.Context) error {
	select {
	case m.ch <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
