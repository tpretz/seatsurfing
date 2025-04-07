package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// AuthRateLimitMiddleware applies rate limiting only to authentication routes
func AuthRateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("APP_ENV") == "test" {
			next.ServeHTTP(w, r)
			return
		}

		// Check if the path is an auth route that needs rate limiting
		if isAuthRoute(r.URL.Path) {
			// Get client IP address
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}

			// If X-Forwarded-For header is present, use the first IP in the chain
			// This is important for applications behind proxies or load balancers
			if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
				ips := strings.Split(forwardedFor, ",")
				if len(ips) > 0 {
					ip = strings.TrimSpace(ips[0])
				}
			}

			// Apply rate limiting
			limitInfo := GetRateLimiter().GetLimitInfo(ip)

			// Set rate limit headers for client information
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", GetRateLimiter().MaxRequests))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", limitInfo.RemainingCount))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", limitInfo.ResetTime.Unix()))

			// Log rate limit information for debugging
			log.Printf("Rate limit for IP %s: %d/%d requests, reset at %s",
				ip, limitInfo.CurrentCount, GetRateLimiter().MaxRequests,
				limitInfo.ResetTime.Format(time.RFC3339))

			if !limitInfo.Allowed {
				log.Printf("Rate limit exceeded for IP: %s on path: %s (count: %d)",
					ip, r.URL.Path, limitInfo.CurrentCount)

				// Calculate seconds until reset
				retryAfter := int(time.Until(limitInfo.ResetTime).Seconds())
				if retryAfter < 1 {
					retryAfter = 60 // Default to 60 seconds if calculation is off
				}

				w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
				http.Error(w, "Too many login attempts. Please try again later.", http.StatusTooManyRequests)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// isAuthRoute checks if a path is an authentication route that needs rate limiting
func isAuthRoute(path string) bool {
	// Add all authentication-related paths here
	authPaths := []string{
		"/auth/login",
		/*"/auth/verify/",
		"/auth/preflight",
		"/auth/refresh",
		"/auth/initpwreset",*/
	}

	for _, authPath := range authPaths {
		if strings.HasPrefix(path, authPath) {
			return true
		}
	}

	// Check for login path with any provider ID
	if strings.Contains(path, "/auth/") && strings.Contains(path, "/login/") {
		return true
	}

	return false
}
