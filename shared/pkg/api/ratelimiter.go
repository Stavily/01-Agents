package api

import (
	"context"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	tokens   chan struct{}
	ticker   *time.Ticker
	rate     int
	capacity int
	mu       sync.Mutex
	closed   bool
}

// NewRateLimiter creates a new rate limiter with the specified rate (requests per second)
func NewRateLimiter(rate int) *RateLimiter {
	if rate <= 0 {
		rate = 10 // Default to 10 RPS
	}

	capacity := rate * 2 // Allow burst up to 2x the rate
	if capacity < 10 {
		capacity = 10 // Minimum capacity
	}

	rl := &RateLimiter{
		tokens:   make(chan struct{}, capacity),
		rate:     rate,
		capacity: capacity,
	}

	// Fill the bucket initially
	for i := 0; i < capacity; i++ {
		select {
		case rl.tokens <- struct{}{}:
		default:
			break
		}
	}

	// Start the token refill ticker
	rl.ticker = time.NewTicker(time.Second / time.Duration(rate))
	go rl.refillTokens()

	return rl
}

// Wait waits for a token to become available, respecting the context
func (rl *RateLimiter) Wait(ctx context.Context) error {
	rl.mu.Lock()
	if rl.closed {
		rl.mu.Unlock()
		return context.Canceled
	}
	rl.mu.Unlock()

	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TryWait attempts to acquire a token without blocking
func (rl *RateLimiter) TryWait() bool {
	rl.mu.Lock()
	if rl.closed {
		rl.mu.Unlock()
		return false
	}
	rl.mu.Unlock()

	select {
	case <-rl.tokens:
		return true
	default:
		return false
	}
}

// refillTokens periodically adds tokens to the bucket
func (rl *RateLimiter) refillTokens() {
	for range rl.ticker.C {
		rl.mu.Lock()
		if rl.closed {
			rl.mu.Unlock()
			return
		}
		rl.mu.Unlock()

		// Try to add a token
		select {
		case rl.tokens <- struct{}{}:
		default:
			// Bucket is full, skip
		}
	}
}

// GetStats returns current rate limiter statistics
func (rl *RateLimiter) GetStats() (available int, capacity int, rate int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	return len(rl.tokens), rl.capacity, rl.rate
}

// Close stops the rate limiter and cleans up resources
func (rl *RateLimiter) Close() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if !rl.closed {
		rl.closed = true
		if rl.ticker != nil {
			rl.ticker.Stop()
		}
		close(rl.tokens)
	}
}
