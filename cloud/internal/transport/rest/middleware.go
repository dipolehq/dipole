/*
Package rest — middleware.go provides HTTP middleware for the REST API.

Implements: auth (Bearer token validation), tenant scoping, request ID generation,
rate limiting (per-tenant token bucket), CORS, and Resource Set scoping.
All middleware stores extracted values in request context for downstream handlers.
*/
package rest

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type contextKey string

const (
	ctxTenantID    contextKey = "tenant_id"
	ctxUserEmail   contextKey = "user_email"
	ctxRequestID   contextKey = "request_id"
	ctxResourceSet contextKey = "resource_set_id"
)

func TenantIDFromContext(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(ctxTenantID).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

func UserEmailFromContext(ctx context.Context) string {
	if email, ok := ctx.Value(ctxUserEmail).(string); ok {
		return email
	}
	return ""
}

func ResourceSetFromContext(ctx context.Context) *uuid.UUID {
	if id, ok := ctx.Value(ctxResourceSet).(*uuid.UUID); ok {
		return id
	}
	return nil
}

type APIAuthenticator interface {
	ValidateToken(ctx context.Context, token string) (tenantID uuid.UUID, email string, err error)
}

func AuthMiddleware(auth APIAuthenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractAPIToken(r)
			if token == "" {
				http.Error(w, `{"error":"missing or invalid Authorization header"}`, http.StatusUnauthorized)
				return
			}

			tenantID, email, err := auth.ValidateToken(r.Context(), token)
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ctxTenantID, tenantID)
			ctx = context.WithValue(ctx, ctxUserEmail, email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := uuid.New().String()
		w.Header().Set("X-Request-Id", reqID)
		ctx := context.WithValue(r.Context(), ctxRequestID, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ResourceSetMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rsHeader := r.Header.Get("X-Dipole-ResourceSet"); rsHeader != "" {
			if parsed, err := uuid.Parse(rsHeader); err == nil {
				ctx := context.WithValue(r.Context(), ctxResourceSet, &parsed)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Dipole-ResourceSet, Idempotency-Key")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type RateLimiter struct {
	mu        sync.Mutex
	buckets   map[uuid.UUID]*tokenBucket
	readRate  int
	writeRate int
}

type tokenBucket struct {
	tokens    float64
	lastCheck time.Time
	rate      float64
}

func NewRateLimiter(readRate, writeRate int) *RateLimiter {
	return &RateLimiter{
		buckets:   make(map[uuid.UUID]*tokenBucket),
		readRate:  readRate,
		writeRate: writeRate,
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := TenantIDFromContext(r.Context())
		if tenantID == uuid.Nil {
			next.ServeHTTP(w, r)
			return
		}

		rate := rl.readRate
		if r.Method != http.MethodGet {
			rate = rl.writeRate
		}

		if !rl.allow(tenantID, rate) {
			w.Header().Set("Retry-After", "1")
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) allow(tenantID uuid.UUID, rate int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, ok := rl.buckets[tenantID]
	if !ok {
		bucket = &tokenBucket{tokens: float64(rate), lastCheck: time.Now(), rate: float64(rate)}
		rl.buckets[tenantID] = bucket
	}

	now := time.Now()
	elapsed := now.Sub(bucket.lastCheck).Seconds()
	bucket.tokens += elapsed * bucket.rate
	if bucket.tokens > float64(rate) {
		bucket.tokens = float64(rate)
	}
	bucket.lastCheck = now

	if bucket.tokens < 1 {
		return false
	}
	bucket.tokens--
	return true
}

func extractAPIToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}
