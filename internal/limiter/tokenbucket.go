package limiter

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// TokenBucket implements a token bucket rate limiter for bandwidth control
type TokenBucket struct {
	enabled   bool
	capacity  int64 // Maximum tokens (bytes)
	tokens    int64 // Current tokens (atomic)
	rate      int64 // Refill rate per second (bytes/second)
	lastRefill int64 // Unix nanoseconds of last refill (atomic)
	mu        sync.Mutex
}

// NewTokenBucket creates a new token bucket with the given rate limit
func NewTokenBucket(rateBPS int64, enabled bool) *TokenBucket {
	tb := &TokenBucket{
		enabled:   enabled,
		capacity:  rateBPS * 5, // Capacity equals 5x rate for better burst handling
		tokens:    rateBPS * 5, // Start with full bucket
		rate:      rateBPS,
		lastRefill: time.Now().UnixNano(),
	}
	return tb
}

// Wait blocks until n tokens are available, then consumes them.
// If the limiter is disabled, this returns immediately.
// If the context is cancelled, returns ctx.Err().
func (tb *TokenBucket) Wait(ctx context.Context, n int64) error {
	if !tb.enabled || n <= 0 {
		return nil
	}

	for {
		// Check for cancellation first
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Try to get tokens
		if tb.tryConsume(n) {
			return nil
		}

		// Not enough tokens, wait a bit and retry, but check for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
			// Retry tryConsume
		}
	}
}

// tryConsume attempts to consume n tokens, refilling first if needed
// Returns true if successful, false if not enough tokens
func (tb *TokenBucket) tryConsume(n int64) bool {
	tb.refill()

	// Try to atomically consume tokens
	for {
		current := atomic.LoadInt64(&tb.tokens)
		if current < n {
			return false // Not enough tokens
		}

		if atomic.CompareAndSwapInt64(&tb.tokens, current, current-n) {
			return true // Successfully consumed
		}
		// CAS failed, retry
	}
}

// refill adds tokens based on elapsed time since last refill
func (tb *TokenBucket) refill() {
	now := time.Now().UnixNano()
	last := atomic.LoadInt64(&tb.lastRefill)

	// Calculate elapsed time in seconds
	elapsed := float64(now-last) / float64(time.Second)

	// Calculate tokens to add
	tokensToAdd := int64(elapsed * float64(tb.rate))

	if tokensToAdd <= 0 {
		return // Nothing to add
	}

	// Update lastRefill if we're the ones doing the refill
	if atomic.CompareAndSwapInt64(&tb.lastRefill, last, now) {
		// Add tokens, but don't exceed capacity
		for {
			current := atomic.LoadInt64(&tb.tokens)
			newTokens := current + tokensToAdd
			if newTokens > tb.capacity {
				newTokens = tb.capacity
			}

			if atomic.CompareAndSwapInt64(&tb.tokens, current, newTokens) {
				return
			}
			// CAS failed, retry
		}
	}
}

// GetCapacity returns the bucket capacity
func (tb *TokenBucket) GetCapacity() int64 {
	return tb.capacity
}

// GetRate returns the refill rate
func (tb *TokenBucket) GetRate() int64 {
	return tb.rate
}

// IsEnabled returns whether the limiter is enabled
func (tb *TokenBucket) IsEnabled() bool {
	return tb.enabled
}
