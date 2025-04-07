package main

import (
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimiter defines a simple rate limiter with a sliding window
type RateLimiter struct {
	mu           sync.Mutex
	requestCount map[string][]time.Time
	MaxRequests  int // Made public for easier access
	windowSize   time.Duration
}

// NewRateLimiter creates a new rate limiter with the specified maximum requests per window
func NewRateLimiter(maxRequests int, windowSize time.Duration) *RateLimiter {
	log.Printf("Creating new rate limiter with max %d requests per %v", maxRequests, windowSize)
	return &RateLimiter{
		requestCount: make(map[string][]time.Time),
		MaxRequests:  maxRequests,
		windowSize:   windowSize,
	}
}

// RateLimitInfo contains information about the rate limit status
type RateLimitInfo struct {
	Allowed        bool
	CurrentCount   int
	RemainingCount int
	ResetTime      time.Time
}

// Allow checks if a request from a given IP is allowed
func (rl *RateLimiter) Allow(ip string) bool {
	info := rl.GetLimitInfo(ip)
	return info.Allowed
}

// GetLimitInfo provides detailed information about the rate limit status for an IP
func (rl *RateLimiter) GetLimitInfo(ip string) RateLimitInfo {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Clean up old requests that are outside the window
	var validTimestamps []time.Time
	var oldestTimestamp time.Time

	if timestamps, ok := rl.requestCount[ip]; ok {
		for _, t := range timestamps {
			if now.Sub(t) <= rl.windowSize {
				validTimestamps = append(validTimestamps, t)
				if oldestTimestamp.IsZero() || t.Before(oldestTimestamp) {
					oldestTimestamp = t
				}
			}
		}
		rl.requestCount[ip] = validTimestamps
	}

	currentCount := len(validTimestamps)
	// Check if the current request would be allowed (less than MaxRequests)
	allowed := currentCount < rl.MaxRequests

	var resetTime time.Time
	if !oldestTimestamp.IsZero() {
		resetTime = oldestTimestamp.Add(rl.windowSize)
	} else {
		resetTime = now.Add(rl.windowSize)
	}

	// Add current request timestamp if allowed
	if allowed {
		rl.requestCount[ip] = append(rl.requestCount[ip], now)
		currentCount++
	}

	return RateLimitInfo{
		Allowed:        allowed,
		CurrentCount:   currentCount,
		RemainingCount: rl.MaxRequests - currentCount,
		ResetTime:      resetTime,
	}
}

// Periodically clean up old entries to prevent memory leaks
func (rl *RateLimiter) StartCleanupTask(cleanupInterval time.Duration) {
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()

		for range ticker.C {
			rl.mu.Lock()
			now := time.Now()
			for ip, timestamps := range rl.requestCount {
				var validTimestamps []time.Time
				for _, t := range timestamps {
					if now.Sub(t) <= rl.windowSize {
						validTimestamps = append(validTimestamps, t)
					}
				}
				if len(validTimestamps) == 0 {
					delete(rl.requestCount, ip)
				} else {
					rl.requestCount[ip] = validTimestamps
				}
			}
			rl.mu.Unlock()
		}
	}()
}

// RateLimitMiddleware creates middleware to apply rate limiting
func RateLimitMiddleware(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}

			if !rl.Allow(ip) {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Global rate limiter instance
var _rateLimiterInstance *RateLimiter
var _rateLimiterOnce sync.Once

// GetRateLimiter returns the singleton rate limiter instance
func GetRateLimiter() *RateLimiter {
	_rateLimiterOnce.Do(func() {
		// Initialize rate limiter with 5 requests per minute
		_rateLimiterInstance = NewRateLimiter(5, time.Minute)
		// Start cleanup task every 5 minutes
		_rateLimiterInstance.StartCleanupTask(5 * time.Minute)
		log.Println("Rate limiter initialized with 5 requests per minute limit")
	})
	return _rateLimiterInstance
}
