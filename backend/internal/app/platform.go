package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type fixedWindowLimiter struct {
	mu      sync.Mutex
	entries map[string]limitEntry
}

type limitEntry struct {
	count     int
	windowEnd time.Time
}

func newFixedWindowLimiter() *fixedWindowLimiter {
	return &fixedWindowLimiter{entries: map[string]limitEntry{}}
}

func (s *Server) requestContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), requestIDKey, middleware.GetReqID(r.Context()))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (l *fixedWindowLimiter) Allow(key string, limit int, window time.Duration) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	entry := l.entries[key]
	if now.After(entry.windowEnd) {
		entry = limitEntry{count: 0, windowEnd: now.Add(window)}
	}
	if entry.count >= limit {
		l.entries[key] = entry
		return false
	}
	entry.count++
	l.entries[key] = entry
	return true
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logEvent("panic_recovered", map[string]any{
					"request_id": middleware.GetReqID(r.Context()),
					"route":      r.URL.Path,
					"error":      recovered,
				})
				writeAPIError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"item": map[string]string{"status": "ok"}})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.db.PingContext(ctx); err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "not_ready", "Database unavailable")
		return
	}
	info, err := os.Stat(s.cfg.StorageRoot)
	if err != nil || !info.IsDir() {
		writeAPIError(w, http.StatusServiceUnavailable, "not_ready", "Storage unavailable")
		return
	}
	if err := os.MkdirAll(filepath.Join(s.cfg.StorageRoot, "logs"), 0o755); err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "not_ready", "Log storage unavailable")
		return
	}
	if err := os.MkdirAll(filepath.Join(s.cfg.StorageRoot, "sites"), 0o755); err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "not_ready", "Site storage unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": map[string]string{"status": "ready"}})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func (s *Server) rateLimit(key string, limit int, window time.Duration, w http.ResponseWriter) bool {
	if s.limiter == nil {
		return true
	}
	if s.limiter.Allow(key, limit, window) {
		return true
	}
	writeAPIError(w, http.StatusTooManyRequests, "rate_limited", "Too many requests")
	return false
}

func (s *Server) auditLog(ctx context.Context, actorID, tenantID, siteID, action, targetType, targetID, outcome string, metadata map[string]any) {
	payload := "{}"
	if len(metadata) > 0 {
		if encoded, err := jsonMarshal(metadata); err == nil {
			payload = encoded
		}
	}
	_, _ = s.db.ExecContext(ctx, `
		insert into audit_logs(id,tenant_id,site_id,actor_id,action,target_type,target_id,outcome,metadata_json,created_at)
		values($1,nullif($2,''),nullif($3,''),nullif($4,''),$5,$6,$7,$8,$9::jsonb,now())
	`, uuid.NewString(), tenantID, siteID, actorID, action, targetType, targetID, outcome, payload)
}

func jsonMarshal(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func activeReleaseRetentionNote(root, tenantID, siteID string) error {
	notePath := filepath.Join(siteRootDir(root, tenantID, siteID), "releases", "RETENTION.md")
	if _, err := os.Stat(notePath); err == nil {
		return nil
	}
	content := []byte("Release retention: keep immutable releases, never delete the active release, and prune only with an explicit maintenance process.\n")
	return atomicWriteFile(notePath, content, 0o644)
}

func logServerLifecycle(path, message string) {
	_ = appendLog(filepath.Join(path, "logs", "api.log"), fmt.Sprintf("%s level=info service=api lifecycle=%s", time.Now().UTC().Format(time.RFC3339), message))
}
