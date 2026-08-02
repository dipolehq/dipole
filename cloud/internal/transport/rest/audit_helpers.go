/*
audit_helpers.go provides shared utilities for audit log emission from HTTP handlers.
*/
package rest

import (
	"net/http"
	"strings"

	"github.com/dipolehq/dipole/cloud/internal/service/events"
)

type AuditEmitter = events.AuditLogEmitter

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.SplitN(forwarded, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

func userAgent(r *http.Request) string {
	return r.Header.Get("User-Agent")
}
