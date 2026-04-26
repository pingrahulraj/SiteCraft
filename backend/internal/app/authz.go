package app

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type Actor struct {
	UserID string
	Email  string
	Name   string
}

type siteAccess struct {
	SiteID   string
	TenantID string
	Slug     string
	Role     string
}

const (
	actorContextKey      ctxKey = "actor"
	siteAccessContextKey ctxKey = "site_access"
)

func currentActor(ctx context.Context) (Actor, bool) {
	actor, ok := ctx.Value(actorContextKey).(Actor)
	return actor, ok
}

func currentSiteAccess(ctx context.Context) (siteAccess, bool) {
	access, ok := ctx.Value(siteAccessContextKey).(siteAccess)
	return access, ok
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
			s.logEvent("auth_denied", map[string]any{"route": r.URL.Path, "reason": "missing_bearer", "ip": clientIP(r)})
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}
		token := strings.TrimSpace(header[len("Bearer "):])
		if token == "" {
			s.logEvent("auth_denied", map[string]any{"route": r.URL.Path, "reason": "empty_token", "ip": clientIP(r)})
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}
		parsed, err := parseToken(s.cfg.JWTSecret, token)
		if err != nil || parsed.UserID == "" {
			s.logEvent("auth_denied", map[string]any{"route": r.URL.Path, "reason": "invalid_token", "ip": clientIP(r)})
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}
		actor := Actor{UserID: parsed.UserID, Email: parsed.Email, Name: parsed.Name}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorContextKey, actor)))
	})
}

func (s *Server) requireSiteAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := currentActor(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}
		siteID := chi.URLParam(r, "id")
		requiredRole := requiredSiteRole(r.Method)
		tenantID, slug, role, err := s.siteMetaForActor(r.Context(), actor.UserID, siteID, requiredRole)
		if err != nil {
			s.logEvent("site_access_denied", map[string]any{"actor_id": actor.UserID, "site_id": siteID, "route": r.URL.Path, "required_role": requiredRole})
			s.writeSiteAccessError(w, err)
			return
		}
		access := siteAccess{
			SiteID:   siteID,
			TenantID: tenantID,
			Slug:     slug,
			Role:     role,
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), siteAccessContextKey, access)))
	})
}

func requiredSiteRole(method string) string {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return "viewer"
	default:
		return "editor"
	}
}

func (s *Server) tenantForActor(ctx context.Context, actorID, preferredSlug string) (string, error) {
	var tenantID string
	if preferredSlug != "" {
		err := s.db.QueryRowContext(ctx, `
			select t.id
			from tenants t
			join tenant_users tu on tu.tenant_id = t.id
			where tu.user_id = $1 and tu.status = 'active' and t.slug = $2
			order by tu.created_at
			limit 1
		`, actorID, preferredSlug).Scan(&tenantID)
		if err == nil {
			return tenantID, nil
		}
		if errors.Is(err, sql.ErrNoRows) {
			return "", errForbidden
		}
		return "", err
	}
	err := s.db.QueryRowContext(ctx, `
		select t.id
		from tenants t
		join tenant_users tu on tu.tenant_id = t.id
		where tu.user_id = $1 and tu.status = 'active'
		order by tu.created_at
		limit 1
	`, actorID).Scan(&tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errForbidden
		}
		return "", err
	}
	return tenantID, nil
}

var (
	errForbidden    = errors.New("access denied")
	errSiteNotFound = errors.New("site not found")
)

func roleRank(role string) int {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "viewer":
		return 1
	case "editor":
		return 2
	case "admin":
		return 3
	case "owner":
		return 4
	default:
		return 0
	}
}

func hasRequiredRole(actualRole, requiredRole string) bool {
	return roleRank(actualRole) >= roleRank(requiredRole)
}

func (s *Server) siteMetaForActor(ctx context.Context, actorID, siteID, requiredRole string) (tenantID, slug, role string, err error) {
	err = s.db.QueryRowContext(ctx, `
		select s.tenant_id, s.slug, tu.role
		from sites s
		join tenant_users tu on tu.tenant_id = s.tenant_id
		where s.id = $1 and tu.user_id = $2 and tu.status = 'active'
		limit 1
	`, siteID, actorID).Scan(&tenantID, &slug, &role)
	if err == nil {
		if !hasRequiredRole(role, requiredRole) {
			return "", "", "", errForbidden
		}
		return tenantID, slug, role, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", "", "", err
	}
	var exists bool
	if lookupErr := s.db.QueryRowContext(ctx, `select exists(select 1 from sites where id = $1)`, siteID).Scan(&exists); lookupErr != nil {
		return "", "", "", lookupErr
	}
	if !exists {
		return "", "", "", errSiteNotFound
	}
	return "", "", "", errForbidden
}

func (s *Server) writeSiteAccessError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errForbidden):
		writeAPIError(w, http.StatusForbidden, "forbidden", "Access denied")
	case errors.Is(err, errSiteNotFound), errors.Is(err, sql.ErrNoRows):
		writeAPIError(w, http.StatusNotFound, "not_found", "Site not found")
	default:
		writeAPIError(w, http.StatusInternalServerError, "query_failed", "Request could not be completed")
	}
}
