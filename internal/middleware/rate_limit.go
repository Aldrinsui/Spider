package middleware

import (
	"net/http"
	"sync"
	"time"
)

type rateLimiter struct {
	mu      sync.Mutex
	records map[string][]time.Time
	limit   int
	window  time.Duration
}

var otpLimiter = &rateLimiter{
	records: make(map[string][]time.Time),
	limit:   5, // max 5 requests per minute
	window:  time.Minute,
}

func RateLimitOTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr

		otpLimiter.mu.Lock()
		now := time.Now()
		var valid []time.Time
		for _, t := range otpLimiter.records[ip] {
			if now.Sub(t) < otpLimiter.window {
				valid = append(valid, t)
			}
		}

		if len(valid) >= otpLimiter.limit {
			otpLimiter.records[ip] = valid
			otpLimiter.mu.Unlock()
			http.Error(w, `{"error":"too many OTP requests. Please wait a minute before retrying."}`, http.StatusTooManyRequests)
			return
		}

		valid = append(valid, now)
		otpLimiter.records[ip] = valid
		otpLimiter.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}
