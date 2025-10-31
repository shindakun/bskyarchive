package archiver

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	requestsPerWindow int           // Maximum requests allowed per window
	windowDuration    time.Duration // Duration of the rate limit window
	burst             int           // Maximum burst size

	mu         sync.Mutex
	tokens     int       // Current available tokens
	lastRefill time.Time // Last time tokens were refilled
}

// NewRateLimiter creates a new token bucket rate limiter
func NewRateLimiter(requestsPerWindow int, windowDuration time.Duration, burst int) *RateLimiter {
	if burst <= 0 {
		burst = requestsPerWindow / 10 // Default burst to 10% of window
		if burst < 1 {
			burst = 1
		}
	}

	return &RateLimiter{
		requestsPerWindow: requestsPerWindow,
		windowDuration:    windowDuration,
		burst:             burst,
		tokens:            burst, // Start with full burst capacity
		lastRefill:        time.Now(),
	}
}

// Wait blocks until a token is available or context is cancelled
func (rl *RateLimiter) Wait(ctx context.Context) error {
	for {
		// Try to acquire a token
		if rl.tryAcquire() {
			return nil
		}

		// Calculate how long to wait before next refill
		rl.mu.Lock()
		waitTime := rl.timeUntilNextToken()
		rl.mu.Unlock()

		// Wait for either the context to be done or the wait time to elapse
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Continue loop to try again
		}
	}
}

// tryAcquire attempts to acquire a token without blocking
func (rl *RateLimiter) tryAcquire() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refillTokens()

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// refillTokens adds tokens based on time elapsed since last refill
func (rl *RateLimiter) refillTokens() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)

	// Calculate how many tokens should be added based on elapsed time
	tokensToAdd := int(elapsed.Seconds() / rl.windowDuration.Seconds() * float64(rl.requestsPerWindow))

	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		// Cap at burst size
		if rl.tokens > rl.burst {
			rl.tokens = rl.burst
		}
		rl.lastRefill = now
	}
}

// timeUntilNextToken calculates how long until the next token will be available
func (rl *RateLimiter) timeUntilNextToken() time.Duration {
	// If we have tokens, no wait needed
	if rl.tokens > 0 {
		return 0
	}

	// Calculate time per token
	timePerToken := rl.windowDuration / time.Duration(rl.requestsPerWindow)

	// Time since last refill
	elapsed := time.Since(rl.lastRefill)

	// Time until next token
	timeUntilNext := timePerToken - elapsed
	if timeUntilNext < 0 {
		timeUntilNext = 0
	}

	return timeUntilNext
}

// Stats returns current rate limiter statistics
func (rl *RateLimiter) Stats() map[string]interface{} {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refillTokens() // Update tokens before reporting

	return map[string]interface{}{
		"tokens":              rl.tokens,
		"burst":               rl.burst,
		"requests_per_window": rl.requestsPerWindow,
		"window_duration":     rl.windowDuration.String(),
		"last_refill":         rl.lastRefill.Format(time.RFC3339),
	}
}

// String returns a human-readable representation of the rate limiter
func (rl *RateLimiter) String() string {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refillTokens()

	return fmt.Sprintf("RateLimiter(%d/%d tokens, %d req/%s)",
		rl.tokens, rl.burst, rl.requestsPerWindow, rl.windowDuration)
}
