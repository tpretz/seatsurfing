package main

import (
	"log"
	"sync"
	"time"
)

// IPLimitInfo stores rate limit data for a specific IP
type IPLimitInfo struct {
	Count    int       // Current count of requests
	FirstHit time.Time // Time of first request in current window
	LastHit  time.Time // Time of most recent request (for cleanup)
}

// RateLimiter defines a counter-based rate limiter with a sliding window
type RateLimiter struct {
	mu          sync.Mutex
	ipLimits    map[string]*IPLimitInfo
	MaxRequests int // Made public for easier access
	windowSize  time.Duration
}

// NewRateLimiter creates a new rate limiter with the specified maximum requests per window
func NewRateLimiter(maxRequests int, windowSize time.Duration) *RateLimiter {
	log.Printf("Creating new rate limiter with max %d requests per %v", maxRequests, windowSize)
	return &RateLimiter{
		ipLimits:    make(map[string]*IPLimitInfo),
		MaxRequests: maxRequests,
		windowSize:  windowSize,
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

	// Check if this IP has previous request data
	limitData, exists := rl.ipLimits[ip]

	// If no existing data or the window has expired, start a new counter
	if !exists || now.Sub(limitData.FirstHit) > rl.windowSize {
		// Create or reset the counter
		rl.ipLimits[ip] = &IPLimitInfo{
			Count:    1, // This counts the current request
			FirstHit: now,
			LastHit:  now,
		}

		return RateLimitInfo{
			Allowed:        true,
			CurrentCount:   1,
			RemainingCount: rl.MaxRequests - 1,
			ResetTime:      now.Add(rl.windowSize),
		}
	}

	// Update last hit time for cleanup purposes
	limitData.LastHit = now

	// Check if limit is reached
	allowed := limitData.Count < rl.MaxRequests

	// Increment counter if allowed
	if allowed {
		limitData.Count++
	}

	return RateLimitInfo{
		Allowed:        allowed,
		CurrentCount:   limitData.Count,
		RemainingCount: rl.MaxRequests - limitData.Count,
		ResetTime:      limitData.FirstHit.Add(rl.windowSize),
	}
}

// Periodically clean up old entries to prevent memory leaks
func (rl *RateLimiter) StartCleanupTask(cleanupInterval time.Duration, done <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				rl.mu.Lock()
				now := time.Now()

				// Remove IPs that haven't been seen recently
				for ip, data := range rl.ipLimits {
					// Remove if last hit was more than window size ago
					if now.Sub(data.LastHit) > rl.windowSize {
						delete(rl.ipLimits, ip)
					}
				}

				rl.mu.Unlock()
				log.Println("Rate limiter cleanup completed")

			case <-done:
				log.Println("Stopping rate limiter cleanup task")
				return
			}
		}
	}()
}

// Global rate limiter instance and shutdown channel
var _rateLimiterInstance *RateLimiter
var _rateLimiterOnce sync.Once
var _rateLimiterDone chan struct{}

// GetRateLimiter returns the singleton rate limiter instance
func GetRateLimiter() *RateLimiter {
	_rateLimiterOnce.Do(func() {
		config := GetConfig()
		maxRequests := config.RateLimitMaxRequests
		windowSeconds := config.RateLimitWindowSeconds
		cleanupIntervalMinutes := config.RateLimitCleanupIntervalMinutes

		// Initialize rate limiter with config values
		_rateLimiterInstance = NewRateLimiter(
			maxRequests,
			time.Duration(windowSeconds)*time.Second,
		)

		// Create done channel for cleanup task
		_rateLimiterDone = make(chan struct{})

		// Start cleanup task with done channel
		_rateLimiterInstance.StartCleanupTask(
			time.Duration(cleanupIntervalMinutes)*time.Minute,
			_rateLimiterDone,
		)

		log.Printf("Rate limiter initialized with %d requests per %d seconds limit",
			maxRequests, windowSeconds)
	})
	return _rateLimiterInstance
}

// ShutdownRateLimiter sends signal to stop the cleanup goroutine
func ShutdownRateLimiter() {
	if _rateLimiterDone != nil {
		close(_rateLimiterDone)
		log.Println("Rate limiter shutdown signal sent")
	}
}
