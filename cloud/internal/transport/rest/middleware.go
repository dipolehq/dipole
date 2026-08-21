// middleware.go provides HTTP middleware: proxy secret verification, auth, tenant scoping, request ID, rate limiting, CORS, and resource set scoping.
package rest

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/scope"
	"github.com/dipolehq/dipole/cloud/internal/service"
	"github.com/google/uuid"
)

type contextKey string

const (
	ctxTenantID    contextKey = "tenant_id"
	ctxUserEmail   contextKey = "user_email"
	ctxUserID      contextKey = "user_id"
	ctxRequestID   contextKey = "request_id"
	ctxPermissions contextKey = "permissions"
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

// UserIDFromContext returns the acting admin user's ID for audit
// attribution, as resolved by RBACPermissionsMiddleware. Returns nil when
// the caller isn't a real user row (API tokens, which authenticate via a
// synthetic email with no backing user, per resolvePermissions) or when
// RBAC permission resolution hasn't run for this request.
func UserIDFromContext(ctx context.Context) *uuid.UUID {
	if id, ok := ctx.Value(ctxUserID).(uuid.UUID); ok {
		return &id
	}
	return nil
}
func ResourceSetFromContext(ctx context.Context) *uuid.UUID {
	if s := scope.FromContext(ctx); s != nil {
		return &s.ResourceSetID
	}
	return nil
}

func PermissionsFromContext(ctx context.Context) service.Permissions {
	if p, ok := ctx.Value(ctxPermissions).(service.Permissions); ok {
		return p
	}
	return nil
}

type APIAuthenticator interface {
	ValidateToken(ctx context.Context, token string) (tenantID uuid.UUID, email string, resourceSetID *uuid.UUID, err error)
}

func AuthMiddleware(auth APIAuthenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractAPIToken(r)
			if token == "" {
				http.Error(w, `{"error":"missing or invalid Authorization header"}`, http.StatusUnauthorized)
				return
			}

			tenantID, email, tokenResourceSet, err := auth.ValidateToken(r.Context(), token)
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ctxTenantID, tenantID)
			ctx = context.WithValue(ctx, ctxUserEmail, email)

			if tokenResourceSet != nil {
				ctx = scope.WithScope(ctx, &scope.RequestScope{
					TenantID:      tenantID,
					ResourceSetID: *tokenResourceSet,
				})
			}

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

		if scope.FromContext(r.Context()) != nil {
			next.ServeHTTP(w, r)
			return
		}

		tenantID := TenantIDFromContext(r.Context())

		rsID := scope.DefaultResourceSetID
		if rsHeader := r.Header.Get("X-Dipole-ResourceSet"); rsHeader != "" {
			if parsed, err := uuid.Parse(rsHeader); err == nil {
				rsID = parsed
			}
		}

		ctx := scope.WithScope(r.Context(), &scope.RequestScope{
			TenantID:      tenantID,
			ResourceSetID: rsID,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if len(allowedOrigins) == 0 {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if allowed[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Vary", "Origin")
			} else {
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Dipole-ResourceSet, Idempotency-Key")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

const (
	rateLimiterBucketTTL     = 1 * time.Hour
	rateLimiterSweepInterval = 10 * time.Minute
)

type RateLimiter struct {
	mu        sync.Mutex
	buckets   map[uuid.UUID]*tokenBucket
	readRate  int
	writeRate int
	lastSweep time.Time
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
		lastSweep: time.Now(),
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

	now := time.Now()
	rl.evictStaleBucketsLocked(now)

	bucket, ok := rl.buckets[tenantID]
	if !ok {
		bucket = &tokenBucket{tokens: float64(rate), lastCheck: now, rate: float64(rate)}
		rl.buckets[tenantID] = bucket
	}

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

func (rl *RateLimiter) evictStaleBucketsLocked(now time.Time) {
	if now.Sub(rl.lastSweep) < rateLimiterSweepInterval {
		return
	}
	rl.lastSweep = now
	for tenantID, bucket := range rl.buckets {
		if now.Sub(bucket.lastCheck) > rateLimiterBucketTTL {
			delete(rl.buckets, tenantID)
		}
	}
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

func ApplyMiddleware(mux *http.ServeMux, authenticator APIAuthenticator, allowedOrigins []string, proxySecret string, users repository.UserRepository, roles repository.RoleRepository, logger *slog.Logger) http.Handler {
	rateLimiter := NewRateLimiter(100, 30)
	protected := rateLimiter.Middleware(mux)
	protected = RBACEnforcementMiddleware(protected)
	protected = RBACPermissionsMiddleware(users, roles, logger)(protected)
	protected = ResourceSetMiddleware(protected)
	protected = AuthMiddleware(authenticator)(protected)
	routed := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if IsPublicPath(r.URL.Path) {
			mux.ServeHTTP(w, r)
			return
		}
		protected.ServeHTTP(w, r)
	})
	handler := RequestIDMiddleware(CORSMiddleware(allowedOrigins)(routed))
	if proxySecret != "" {
		handler = ProxySecretMiddleware(proxySecret)(handler)
	}
	return handler
}

func ProxySecretMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/v1/health" {
				next.ServeHTTP(w, r)
				return
			}
			if r.Header.Get("X-Dipole-Proxy-Secret") != secret {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func IsPublicPath(path string) bool {
	if path == "/api/v1/health" || path == "/api/v1/auth/login" || path == "/api/v1/auth/signup" || path == "/api/v1/auth/forgot-password" || path == "/api/v1/auth/reset-password" || path == "/api/v1/auth/mfa-verify" || strings.HasPrefix(path, "/api/v1/auth/sso/") || strings.HasPrefix(path, "/internal/") {
		return true
	}
	if strings.HasPrefix(path, "/edge/") {
		return true
	}
	if strings.HasPrefix(path, "/.well-known/") ||
		strings.HasPrefix(path, "/register") ||
		strings.HasPrefix(path, "/connect/") ||
		strings.HasPrefix(path, "/callback/") ||
		strings.HasPrefix(path, "/mcp") {
		return true
	}
	return false
}

func ParseCORSOrigins(raw string) []string {
	if raw == "" {
		return nil
	}
	var origins []string
	for _, o := range strings.Split(raw, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins = append(origins, o)
		}
	}
	return origins
}
