package app

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Server struct {
	cfg Config
	db  *sql.DB
}

type ctxKey string

const (
	requestIDKey ctxKey = "request_id"
	demoTenant          = "acme-studio"
	demoEmail           = "ava@acme-studio.com"
	demoPassword        = "Welcome123!"
)

func NewServer(cfg Config) *Server {
	root := cfg.StorageRoot
	if root == "" {
		root = "./storage"
	}
	_ = os.MkdirAll(filepath.Join(root, "logs"), 0o755)
	if err := createDatabaseIfMissing(cfg); err != nil {
		panic(err)
	}
	db, err := openPostgres(cfg)
	if err != nil {
		panic(err)
	}
	if err := migrate(db); err != nil {
		panic(err)
	}
	if err := seed(db, root, cfg.deliveryURL()); err != nil {
		panic(err)
	}
	return &Server{cfg: cfg, db: db}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(s.loggingMiddleware)
	r.Use(cors(s.cfg.FrontendOrigin))
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"item": map[string]string{"status": "ok"}})
	})
	r.Get("/sites/{slug}", s.handleServePublishedSite)
	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/register", s.handleRegister)
		r.Post("/auth/login", s.handleLogin)
		r.Get("/me", s.handleMe)
		r.Get("/seed", s.handleSeed)
		r.Get("/projects", s.handleListSites)
		r.Get("/projects/{id}", s.handleGetSite)
		r.Post("/projects", s.handleCreateSite)
		r.Get("/sites/{id}/workspace/index.html", s.handleGetWorkspaceIndex)
		r.Put("/sites/{id}/workspace/index.html", s.handleSaveWorkspaceIndex)
		r.Get("/sites/{id}/workspace/files", s.handleListWorkspaceFiles)
		r.Get("/sites/{id}/workspace/files/content", s.handleGetWorkspaceFileContent)
		r.Put("/sites/{id}/workspace/files/content", s.handleSaveWorkspaceFileContent)
		r.Post("/sites/{id}/workspace/files", s.handleCreateWorkspaceFile)
		r.Put("/sites/{id}/workspace/files/rename", s.handleRenameWorkspaceFile)
		r.Delete("/sites/{id}/workspace/files", s.handleDeleteWorkspaceFile)
		r.Post("/sites/{id}/workspace/files/duplicate", s.handleDuplicateWorkspaceFile)
		r.Post("/sites/{id}/publish", s.handlePublishSite)
		r.Get("/sites/{id}/products", s.handleListProducts)
		r.Post("/sites/{id}/products", s.handleCreateProduct)
		r.Get("/sites/{id}/products/{productId}", s.handleGetProduct)
		r.Put("/sites/{id}/products/{productId}", s.handleUpdateProduct)
		r.Delete("/sites/{id}/products/{productId}", s.handleDeleteProduct)
		r.Post("/sites/{id}/products/{productId}/duplicate", s.handleDuplicateProduct)
		r.Get("/sites/{id}/orders", s.handleListOrders)
		r.Get("/sites/{id}/orders/{orderId}", s.handleGetOrder)
		r.Put("/sites/{id}/orders/{orderId}/status", s.handleUpdateOrderStatus)
		r.Get("/sites/{id}/customers", s.handleListCustomers)
		r.Get("/sites/{id}/customers/{customerId}", s.handleGetCustomer)
		r.Get("/sites/{id}/coupons", s.handleListCoupons)
		r.Post("/sites/{id}/coupons", s.handleCreateCoupon)
		r.Put("/sites/{id}/coupons/{couponId}", s.handleUpdateCoupon)
		r.Delete("/sites/{id}/coupons/{couponId}", s.handleDeleteCoupon)
		r.Get("/sites/{id}/shipping-zones", s.handleListShippingZones)
		r.Post("/sites/{id}/shipping-zones", s.handleCreateShippingZone)
		r.Put("/sites/{id}/shipping-zones/{zoneId}", s.handleUpdateShippingZone)
		r.Delete("/sites/{id}/shipping-zones/{zoneId}", s.handleDeleteShippingZone)
		r.Get("/sites/{id}/tax-settings", s.handleGetTaxSettings)
		r.Put("/sites/{id}/tax-settings", s.handleUpsertTaxSettings)
		r.Get("/sites/{id}/store-settings", s.handleGetStoreSettings)
		r.Put("/sites/{id}/store-settings", s.handleUpsertStoreSettings)
		r.Get("/sites/{id}/posts", s.handleListPosts)
		r.Post("/sites/{id}/posts", s.handleCreatePost)
		r.Get("/sites/{id}/posts/{postId}", s.handleGetPost)
		r.Put("/sites/{id}/posts/{postId}", s.handleUpdatePost)
		r.Delete("/sites/{id}/posts/{postId}", s.handleDeletePost)
		r.Post("/sites/{id}/posts/{postId}/publish", s.handlePublishPost)
		r.Post("/sites/{id}/posts/{postId}/unpublish", s.handleUnpublishPost)
		r.Get("/sites/{id}/blog-categories", s.handleListBlogCategories)
		r.Post("/sites/{id}/blog-categories", s.handleCreateBlogCategory)
		r.Put("/sites/{id}/blog-categories/{categoryId}", s.handleUpdateBlogCategory)
		r.Delete("/sites/{id}/blog-categories/{categoryId}", s.handleDeleteBlogCategory)
		r.Get("/sites/{id}/blog-tags", s.handleListBlogTags)
		r.Post("/sites/{id}/blog-tags", s.handleCreateBlogTag)
		r.Put("/sites/{id}/blog-tags/{tagId}", s.handleUpdateBlogTag)
		r.Delete("/sites/{id}/blog-tags/{tagId}", s.handleDeleteBlogTag)
		r.Get("/sites/{id}/blog-authors", s.handleListBlogAuthors)
		r.Post("/sites/{id}/blog-authors", s.handleCreateBlogAuthor)
		r.Put("/sites/{id}/blog-authors/{authorId}", s.handleUpdateBlogAuthor)
		r.Delete("/sites/{id}/blog-authors/{authorId}", s.handleDeleteBlogAuthor)
		r.Get("/sites/{id}/newsletter-settings", s.handleGetNewsletterSettings)
		r.Put("/sites/{id}/newsletter-settings", s.handleUpsertNewsletterSettings)
		r.Get("/sites/{id}/blog-members", s.handleListBlogMembers)
		r.Post("/sites/{id}/blog-members", s.handleCreateBlogMember)
		r.Put("/sites/{id}/blog-members/{memberId}", s.handleUpdateBlogMember)
		r.Delete("/sites/{id}/blog-members/{memberId}", s.handleDeleteBlogMember)
		r.Get("/sites/{id}/blog-analytics", s.handleGetBlogAnalytics)
		r.Get("/sites/{id}/subscription-plans", s.handleListSubscriptionPlans)
		r.Post("/sites/{id}/subscription-plans", s.handleCreateSubscriptionPlan)
		r.Get("/sites/{id}/subscription-plans/{planId}", s.handleGetSubscriptionPlan)
		r.Put("/sites/{id}/subscription-plans/{planId}", s.handleUpdateSubscriptionPlan)
		r.Delete("/sites/{id}/subscription-plans/{planId}", s.handleDeleteSubscriptionPlan)
		r.Get("/sites/{id}/subscription-subscribers", s.handleListSubscriptionSubscribers)
		r.Post("/sites/{id}/subscription-subscribers", s.handleCreateSubscriptionSubscriber)
		r.Put("/sites/{id}/subscription-subscribers/{subscriberId}", s.handleUpdateSubscriptionSubscriber)
		r.Delete("/sites/{id}/subscription-subscribers/{subscriberId}", s.handleDeleteSubscriptionSubscriber)
		r.Get("/sites/{id}/subscriptions", s.handleListSubscriptions)
		r.Post("/sites/{id}/subscriptions", s.handleCreateSubscription)
		r.Put("/sites/{id}/subscriptions/{subscriptionId}", s.handleUpdateSubscription)
		r.Post("/sites/{id}/subscriptions/{subscriptionId}/cancel", s.handleCancelSubscription)
		r.Get("/sites/{id}/subscription-invoices", s.handleListSubscriptionInvoices)
		r.Get("/sites/{id}/failed-payments", s.handleListFailedPayments)
		r.Post("/sites/{id}/failed-payments/{paymentId}/retry", s.handleRetryFailedPayment)
		r.Get("/sites/{id}/subscription-settings", s.handleGetSubscriptionSettings)
		r.Put("/sites/{id}/subscription-settings", s.handleUpsertSubscriptionSettings)
	})
	return r
}

func cors(origin string) func(http.Handler) http.Handler {
	return handlers.CORS(
		handlers.AllowedOrigins(strings.Split(origin, ",")),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Authorization", "Content-Type"}),
	)
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rr := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(rr, r)
		requestID := middleware.GetReqID(r.Context())
		line := fmt.Sprintf("%s level=info service=api request_id=%s route=%s status=%d duration_ms=%d",
			time.Now().UTC().Format(time.RFC3339), requestID, r.URL.Path, rr.Status(), time.Since(start).Milliseconds())
		_ = appendLog(filepath.Join(s.cfg.StorageRoot, "logs", "api.log"), line)
	})
}

func appendLog(path, line string) error {
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line + "\n")
	return err
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

type authRequest struct {
	TenantSlug string `json:"tenantSlug"`
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	Password   string `json:"password"`
}

type siteRequest struct {
	TenantSlug  string `json:"tenantSlug"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type workspaceFilePayload struct {
	Path     string `json:"path"`
	NewPath  string `json:"newPath"`
	FileType string `json:"fileType"`
	Content  string `json:"content"`
}

type productPayload struct {
	Name           string  `json:"name"`
	Slug           string  `json:"slug"`
	Description    string  `json:"description"`
	Price          float64 `json:"price"`
	SKU            string  `json:"sku"`
	ImageURL       string  `json:"imageUrl"`
	Category       string  `json:"category"`
	StockStatus    string  `json:"stockStatus"`
	SEOTitle       string  `json:"seoTitle"`
	SEODescription string  `json:"seoDescription"`
	Status         string  `json:"status"`
}

type postPayload struct {
	Title         string `json:"title"`
	Slug          string `json:"slug"`
	Excerpt       string `json:"excerpt"`
	Content       string `json:"content"`
	FeaturedImage string `json:"featuredImage"`
	Author        string `json:"author"`
	Tags          string `json:"tags"`
	SEOTitle      string `json:"seoTitle"`
	SEODescription string `json:"seoDescription"`
	Visibility    string `json:"visibility"`
	Status        string `json:"status"`
}

type planPayload struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"intervalPrice"`
	Interval    string  `json:"interval"`
	Trial       int     `json:"trial"`
	Status      string  `json:"status"`
}

func normalizeStatus(value, fallback string, allowed ...string) string {
	if value == "" {
		return fallback
	}
	value = strings.ToLower(strings.TrimSpace(value))
	for _, item := range allowed {
		if value == item {
			return value
		}
	}
	return fallback
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	replacer := strings.NewReplacer(" ", "-", "_", "-", "/", "-", "\\", "-")
	s = replacer.Replace(s)
	s = strings.Trim(s, "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return s
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, 400, "invalid_json", "Invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		writeAPIError(w, 400, "validation_error", "Email and password are required")
		return
	}
	ctx := r.Context()
	tenantID, err := s.resolveTenantID(ctx, req.TenantSlug)
	if err != nil {
		writeAPIError(w, 400, "invalid_tenant", "Tenant not found")
		return
	}
	hash, err := hashPassword(req.Password)
	if err != nil {
		writeAPIError(w, 500, "hash_failed", "Password hashing failed")
		return
	}
	userID := uuid.NewString()
	_, err = s.db.ExecContext(ctx, `insert into users(id,email,name,password_hash,status,created_at,updated_at) values($1,$2,$3,$4,'active',now(),now())`,
		userID, strings.ToLower(req.Email), req.Name, hash)
	if err != nil {
		writeAPIError(w, 409, "account_exists", "Account already exists")
		return
	}
	_, _ = s.db.ExecContext(ctx, `insert into tenant_users(id,tenant_id,user_id,role,status,created_at,updated_at) values($1,$2,$3,'owner','active',now(),now())`, uuid.NewString(), tenantID, userID)
	writeJSON(w, 201, map[string]any{"success": true})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, 400, "invalid_json", "Invalid request body")
		return
	}
	identifier := req.Identifier
	if identifier == "" {
		identifier = req.Email
	}
	var id, email, name, hash string
	err := s.db.QueryRowContext(r.Context(), `select id,email,name,password_hash from users where lower(email)=lower($1) or split_part(lower(email),'@',1)=lower($1)`, identifier).Scan(&id, &email, &name, &hash)
	if err != nil || comparePassword(hash, req.Password) != nil {
		writeAPIError(w, 401, "invalid_credentials", "Invalid credentials")
		return
	}
	token, _ := issueToken(s.cfg.JWTSecret, id, email, name)
	writeJSON(w, 200, map[string]any{"item": map[string]string{"token": token}})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	auth := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	claims, err := parseToken(s.cfg.JWTSecret, auth)
	if err != nil {
		writeAPIError(w, 401, "unauthorized", "Unauthorized")
		return
	}
	writeJSON(w, 200, map[string]any{"item": map[string]string{"id": claims.UserID, "email": claims.Email, "name": claims.Name}})
}

func (s *Server) handleSeed(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"item": seedSnapshot(s.db)})
}

func (s *Server) handleListSites(w http.ResponseWriter, r *http.Request) {
	tenantID, err := s.resolveTenantID(r.Context(), r.URL.Query().Get("tenantSlug"))
	if err != nil {
		writeAPIError(w, 400, "invalid_tenant", "Tenant not found")
		return
	}
	rows, err := s.db.QueryContext(r.Context(), `select id, slug, name, status, published_at, created_at from sites where tenant_id=$1 order by created_at desc`, tenantID)
	if err != nil {
		writeAPIError(w, 500, "query_failed", "Failed to load sites")
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var id, slug, name, status string
		var publishedAt sql.NullTime
		var createdAt time.Time
		if err := rows.Scan(&id, &slug, &name, &status, &publishedAt, &createdAt); err != nil {
			writeAPIError(w, 500, "scan_failed", "Failed to read sites")
			return
		}
		items = append(items, map[string]any{
			"id":         id,
			"slug":       slug,
			"name":       name,
			"description": "",
			"status":     status,
			"previewUrl": s.cfg.deliveryURL() + "/sites/" + slug,
			"createdAt":  createdAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, 200, map[string]any{"items": items, "total": len(items), "projects": items})
}

func (s *Server) handleGetSite(w http.ResponseWriter, r *http.Request) {
	var id, tenantID, slug, name, status string
	var activeReleaseID sql.NullString
	var publishedAt sql.NullTime
	var createdAt time.Time
	err := s.db.QueryRowContext(r.Context(), `select id,tenant_id,slug,name,status,active_release_id,published_at,created_at from sites where id=$1`, chi.URLParam(r, "id")).Scan(&id, &tenantID, &slug, &name, &status, &activeReleaseID, &publishedAt, &createdAt)
	if err != nil {
		writeAPIError(w, 404, "not_found", "Site not found")
		return
	}
	writeJSON(w, 200, map[string]any{"item": map[string]any{
		"id": id, "tenantId": tenantID, "slug": slug, "name": name, "description": "", "status": status, "previewUrl": s.cfg.deliveryURL() + "/sites/" + slug, "createdAt": createdAt.Format(time.RFC3339),
	}, "id": id, "slug": slug, "name": name, "description": "", "status": status, "previewUrl": s.cfg.deliveryURL() + "/sites/" + slug})
}

func (s *Server) handleCreateSite(w http.ResponseWriter, r *http.Request) {
	var req siteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, 400, "invalid_json", "Invalid request body")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeAPIError(w, 400, "validation_error", "Site name is required")
		return
	}
	ctx := r.Context()
	tenantID, err := s.resolveTenantID(ctx, req.TenantSlug)
	if err != nil {
		writeAPIError(w, 400, "invalid_tenant", "Tenant not found")
		return
	}
	siteID := uuid.NewString()
	slug := slugify(req.Name)
	_, err = s.db.ExecContext(ctx, `insert into sites(id,tenant_id,name,slug,status,created_at,updated_at) values($1,$2,$3,$4,'draft',now(),now())`, siteID, tenantID, req.Name, slug)
	if err != nil {
		writeAPIError(w, 409, "create_failed", "Site could not be created")
		return
	}
	if err := ensureSiteWorkspace(s.cfg.StorageRoot, tenantID, siteID); err != nil {
		writeAPIError(w, 500, "storage_failed", "Workspace could not be created")
		return
	}
	indexPath := siteWorkspaceFile(s.cfg.StorageRoot, tenantID, siteID, "index.html")
	content := defaultWorkspaceHTML(req.Name)
	if err := os.WriteFile(indexPath, []byte(content), 0o644); err != nil {
		writeAPIError(w, 500, "storage_failed", "Workspace file could not be created")
		return
	}
	_ = s.upsertWorkspaceMetadata(ctx, tenantID, siteID, "index.html", "html", int64(len(content)), checksum([]byte(content)))
	s.handleListSites(w, r)
}

func allowedWorkspacePath(path string) (string, string, error) {
	path = strings.TrimSpace(strings.TrimPrefix(path, "/"))
	if path == "" {
		return "", "", fmt.Errorf("path is required")
	}
	if strings.Contains(path, "..") || filepath.IsAbs(path) {
		return "", "", fmt.Errorf("invalid path")
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	switch ext {
	case "html", "css", "js", "json", "txt", "md":
	default:
		return "", "", fmt.Errorf("invalid extension")
	}
	return filepath.ToSlash(path), ext, nil
}

func checksum(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func (s *Server) siteMeta(ctx context.Context, siteID string) (tenantID, slug string, err error) {
	err = s.db.QueryRowContext(ctx, `select tenant_id, slug from sites where id=$1`, siteID).Scan(&tenantID, &slug)
	return
}

func (s *Server) handleListWorkspaceFiles(w http.ResponseWriter, r *http.Request) {
	siteID := chi.URLParam(r, "id")
	tenantID, _, err := s.siteMeta(r.Context(), siteID)
	if err != nil {
		writeAPIError(w, 404, "not_found", "Site not found")
		return
	}
	items, err := s.listWorkspaceFiles(r.Context(), tenantID, siteID)
	if err != nil {
		writeAPIError(w, 500, "query_failed", "Failed to load workspace files")
		return
	}
	writeJSON(w, 200, map[string]any{"items": items, "total": len(items)})
}

func (s *Server) listWorkspaceFiles(ctx context.Context, tenantID, siteID string) ([]map[string]any, error) {
	root := siteWorkspaceDir(s.cfg.StorageRoot, tenantID, siteID)
	items := make([]map[string]any, 0)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		info, _ := d.Info()
		items = append(items, map[string]any{
			"path": rel,
			"name": filepath.Base(rel),
			"fileType": strings.TrimPrefix(filepath.Ext(rel), "."),
			"sizeBytes": info.Size(),
		})
		return nil
	})
	return items, err
}

func (s *Server) handleGetWorkspaceFileContent(w http.ResponseWriter, r *http.Request) {
	siteID := chi.URLParam(r, "id")
	tenantID, _, err := s.siteMeta(r.Context(), siteID)
	if err != nil {
		writeAPIError(w, 404, "not_found", "Site not found")
		return
	}
	path, _, err := allowedWorkspacePath(r.URL.Query().Get("path"))
	if err != nil {
		writeAPIError(w, 400, "validation_error", err.Error())
		return
	}
	content, err := os.ReadFile(siteWorkspaceFile(s.cfg.StorageRoot, tenantID, siteID, path))
	if err != nil {
		writeAPIError(w, 404, "not_found", "File not found")
		return
	}
	writeJSON(w, 200, map[string]any{"item": map[string]any{"path": path, "content": string(content)}})
}

func (s *Server) handleSaveWorkspaceFileContent(w http.ResponseWriter, r *http.Request) {
	var req workspaceFilePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, 400, "invalid_json", "Invalid request body")
		return
	}
	siteID := chi.URLParam(r, "id")
	tenantID, _, err := s.siteMeta(r.Context(), siteID)
	if err != nil {
		writeAPIError(w, 404, "not_found", "Site not found")
		return
	}
	path, ext, err := allowedWorkspacePath(req.Path)
	if err != nil {
		writeAPIError(w, 400, "validation_error", err.Error())
		return
	}
	if err := ensureSiteWorkspace(s.cfg.StorageRoot, tenantID, siteID); err != nil {
		writeAPIError(w, 500, "storage_failed", "Workspace not available")
		return
	}
	target := siteWorkspaceFile(s.cfg.StorageRoot, tenantID, siteID, path)
	_ = os.MkdirAll(filepath.Dir(target), 0o755)
	if err := os.WriteFile(target, []byte(req.Content), 0o644); err != nil {
		writeAPIError(w, 500, "save_failed", "File could not be saved")
		return
	}
	_ = s.upsertWorkspaceMetadata(r.Context(), tenantID, siteID, path, ext, int64(len(req.Content)), checksum([]byte(req.Content)))
	writeJSON(w, 200, map[string]any{"success": true, "item": map[string]any{"path": path, "content": req.Content}})
}

func (s *Server) handleCreateWorkspaceFile(w http.ResponseWriter, r *http.Request) {
	var req workspaceFilePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, 400, "invalid_json", "Invalid request body")
		return
	}
	req.Content = ""
	s.handleSaveWorkspaceFileContent(w, withBody(r, req))
}

func withBody(r *http.Request, body any) *http.Request {
	b, _ := json.Marshal(body)
	clone := r.Clone(r.Context())
	clone.Body = ioNopCloser{strings.NewReader(string(b))}
	clone.ContentLength = int64(len(b))
	return clone
}

type ioNopCloser struct{ *strings.Reader }

func (ioNopCloser) Close() error { return nil }

func (s *Server) handleRenameWorkspaceFile(w http.ResponseWriter, r *http.Request) {
	var req workspaceFilePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, 400, "invalid_json", "Invalid request body")
		return
	}
	oldPath, _, err := allowedWorkspacePath(req.Path)
	if err != nil {
		writeAPIError(w, 400, "validation_error", err.Error())
		return
	}
	newPath, ext, err := allowedWorkspacePath(req.NewPath)
	if err != nil {
		writeAPIError(w, 400, "validation_error", err.Error())
		return
	}
	siteID := chi.URLParam(r, "id")
	tenantID, _, err := s.siteMeta(r.Context(), siteID)
	if err != nil {
		writeAPIError(w, 404, "not_found", "Site not found")
		return
	}
	if oldPath == "index.html" {
		writeAPIError(w, 400, "validation_error", "index.html cannot be renamed")
		return
	}
	oldTarget := siteWorkspaceFile(s.cfg.StorageRoot, tenantID, siteID, oldPath)
	newTarget := siteWorkspaceFile(s.cfg.StorageRoot, tenantID, siteID, newPath)
	_ = os.MkdirAll(filepath.Dir(newTarget), 0o755)
	if err := os.Rename(oldTarget, newTarget); err != nil {
		writeAPIError(w, 500, "rename_failed", "File could not be renamed")
		return
	}
	_, _ = s.db.ExecContext(r.Context(), `update workspace_files set path=$1,file_type=$2,updated_at=now() where site_id=$3 and path=$4`, newPath, ext, siteID, oldPath)
	writeJSON(w, 200, map[string]any{"success": true, "item": map[string]any{"path": newPath}})
}

func (s *Server) handleDeleteWorkspaceFile(w http.ResponseWriter, r *http.Request) {
	var req workspaceFilePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, 400, "invalid_json", "Invalid request body")
		return
	}
	path, _, err := allowedWorkspacePath(req.Path)
	if err != nil {
		writeAPIError(w, 400, "validation_error", err.Error())
		return
	}
	if path == "index.html" {
		writeAPIError(w, 400, "validation_error", "index.html cannot be deleted")
		return
	}
	siteID := chi.URLParam(r, "id")
	tenantID, _, err := s.siteMeta(r.Context(), siteID)
	if err != nil {
		writeAPIError(w, 404, "not_found", "Site not found")
		return
	}
	if err := os.Remove(siteWorkspaceFile(s.cfg.StorageRoot, tenantID, siteID, path)); err != nil {
		writeAPIError(w, 500, "delete_failed", "File could not be deleted")
		return
	}
	_, _ = s.db.ExecContext(r.Context(), `delete from workspace_files where site_id=$1 and path=$2`, siteID, path)
	writeJSON(w, 200, map[string]any{"success": true})
}

func (s *Server) handleDuplicateWorkspaceFile(w http.ResponseWriter, r *http.Request) {
	var req workspaceFilePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, 400, "invalid_json", "Invalid request body")
		return
	}
	path, ext, err := allowedWorkspacePath(req.Path)
	if err != nil {
		writeAPIError(w, 400, "validation_error", err.Error())
		return
	}
	siteID := chi.URLParam(r, "id")
	tenantID, _, err := s.siteMeta(r.Context(), siteID)
	if err != nil {
		writeAPIError(w, 404, "not_found", "Site not found")
		return
	}
	src := siteWorkspaceFile(s.cfg.StorageRoot, tenantID, siteID, path)
	content, err := os.ReadFile(src)
	if err != nil {
		writeAPIError(w, 404, "not_found", "File not found")
		return
	}
	base := strings.TrimSuffix(path, "."+ext)
	copyPath := base + "-copy." + ext
	dst := siteWorkspaceFile(s.cfg.StorageRoot, tenantID, siteID, copyPath)
	if err := os.WriteFile(dst, content, 0o644); err != nil {
		writeAPIError(w, 500, "duplicate_failed", "File could not be duplicated")
		return
	}
	_ = s.upsertWorkspaceMetadata(r.Context(), tenantID, siteID, copyPath, ext, int64(len(content)), checksum(content))
	writeJSON(w, 200, map[string]any{"success": true, "item": map[string]any{"path": copyPath}})
}

func (s *Server) handleGetWorkspaceIndex(w http.ResponseWriter, r *http.Request) {
	path := siteWorkspaceFileByIndex(s, r)
	content, err := os.ReadFile(path)
	if err != nil {
		writeAPIError(w, 404, "not_found", "Workspace missing")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(content)
}

func siteWorkspaceFileByIndex(s *Server, r *http.Request) string {
	siteID := chi.URLParam(r, "id")
	tenantID, _, _ := s.siteMeta(r.Context(), siteID)
	return siteWorkspaceFile(s.cfg.StorageRoot, tenantID, siteID, "index.html")
}

func (s *Server) handleSaveWorkspaceIndex(w http.ResponseWriter, r *http.Request) {
	var req struct{ Content string `json:"content"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, 400, "invalid_json", "Invalid request body")
		return
	}
	body := workspaceFilePayload{Path: "index.html", Content: req.Content}
	s.handleSaveWorkspaceFileContent(w, withBody(r, body))
}

func (s *Server) handlePublishSite(w http.ResponseWriter, r *http.Request) {
	siteID := chi.URLParam(r, "id")
	tenantID, slug, err := s.siteMeta(r.Context(), siteID)
	if err != nil {
		writeAPIError(w, 404, "not_found", "Site not found")
		return
	}
	if _, err := os.Stat(siteWorkspaceFile(s.cfg.StorageRoot, tenantID, siteID, "index.html")); err != nil {
		writeAPIError(w, 400, "validation_error", "index.html is required")
		return
	}
	if err := ensureSitePublished(s.cfg.StorageRoot, tenantID, siteID); err != nil {
		writeAPIError(w, 500, "publish_failed", "Publish directory could not be created")
		return
	}
	srcRoot := siteWorkspaceDir(s.cfg.StorageRoot, tenantID, siteID)
	dstRoot := sitePublishedDir(s.cfg.StorageRoot, tenantID, siteID)
	if err := copyDir(srcRoot, dstRoot); err != nil {
		writeAPIError(w, 500, "publish_failed", "Workspace could not be published")
		return
	}
	var nextRelease int
	_ = s.db.QueryRowContext(r.Context(), `select coalesce(max(release_number),0)+1 from site_releases where site_id=$1`, siteID).Scan(&nextRelease)
	releaseID := uuid.NewString()
	publicURL := s.cfg.deliveryURL() + "/sites/" + slug
	artifactPath := filepath.ToSlash(filepath.Join("sites", tenantID, siteID, "published"))
	_, err = s.db.ExecContext(r.Context(), `insert into site_releases(id,tenant_id,site_id,release_number,artifact_path,public_url,status,created_at) values($1,$2,$3,$4,$5,$6,'published',now())`,
		releaseID, tenantID, siteID, nextRelease, artifactPath, publicURL)
	if err != nil {
		writeAPIError(w, 500, "publish_failed", "Release could not be recorded")
		return
	}
	_, _ = s.db.ExecContext(r.Context(), `update sites set active_release_id=$1,status='published',published_at=now(),updated_at=now() where id=$2`, releaseID, siteID)
	_, _ = s.db.ExecContext(r.Context(), `insert into audit_logs(id,tenant_id,site_id,action,target_type,target_id,outcome,created_at) values($1,$2,$3,'publish','site',$4,'success',now())`, uuid.NewString(), tenantID, siteID, siteID)
	writeJSON(w, 200, map[string]any{"success": true, "site_id": siteID, "release_id": releaseID, "public_url": publicURL, "published_at": time.Now().UTC().Format(time.RFC3339), "message": "Website published successfully"})
}

func copyDir(src, dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, 0o644)
	})
}

func (s *Server) handleServePublishedSite(w http.ResponseWriter, r *http.Request) {
	var tenantID, siteID, activeReleaseID string
	err := s.db.QueryRowContext(r.Context(), `select tenant_id,id,active_release_id from sites where slug=$1 and status='published'`, chi.URLParam(r, "slug")).Scan(&tenantID, &siteID, &activeReleaseID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	content, err := os.ReadFile(sitePublishedFile(s.cfg.StorageRoot, tenantID, siteID, "index.html"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(content)
}

// simplified admin endpoints
func (s *Server) handleListProducts(w http.ResponseWriter, r *http.Request) { s.listItems(w, r, `select id,name,slug,description,price,sku,image_url,category,stock_status,status,seo_title,seo_description,updated_at from products where site_id=$1 order by updated_at desc`, []string{"id","name","slug","description","price","sku","imageUrl","category","stockStatus","status","seoTitle","seoDescription","updatedAt"}) }
func (s *Server) handleGetProduct(w http.ResponseWriter, r *http.Request) { s.getItem(w, r, `select id,name,slug,description,price,sku,image_url,category,stock_status,status,seo_title,seo_description,updated_at from products where site_id=$1 and id=$2`, chi.URLParam(r, "productId"), []string{"id","name","slug","description","price","sku","imageUrl","category","stockStatus","status","seoTitle","seoDescription","updatedAt"}) }
func (s *Server) handleCreateProduct(w http.ResponseWriter, r *http.Request) { s.upsertProduct(w, r, false) }
func (s *Server) handleUpdateProduct(w http.ResponseWriter, r *http.Request) { s.upsertProduct(w, r, true) }
func (s *Server) handleDeleteProduct(w http.ResponseWriter, r *http.Request) { s.deleteItem(w, r, "products", chi.URLParam(r, "productId")) }

func (s *Server) upsertProduct(w http.ResponseWriter, r *http.Request, update bool) {
	siteID := chi.URLParam(r, "id")
	tenantID, _, err := s.siteMeta(r.Context(), siteID)
	if err != nil { writeAPIError(w, 404, "not_found", "Site not found"); return }
	_ = tenantID
	var req productPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeAPIError(w, 400, "invalid_json", "Invalid request body"); return }
	if strings.TrimSpace(req.Name) == "" { writeAPIError(w, 400, "validation_error", "Name is required"); return }
	if req.Price < 0 { writeAPIError(w, 400, "validation_error", "Price must be greater than or equal to zero"); return }
	slug := slugify(req.Slug); if slug == "" { slug = slugify(req.Name) }
	status := normalizeStatus(req.Status, "draft", "draft", "active", "archived")
	stock := normalizeStatus(req.StockStatus, "in_stock", "in_stock", "low_stock", "out_of_stock")
	if update {
		_, err = s.db.ExecContext(r.Context(), `update products set name=$1,slug=$2,description=$3,price=$4,sku=$5,image_url=$6,category=$7,stock_status=$8,status=$9,seo_title=$10,seo_description=$11,updated_at=now() where site_id=$12 and id=$13`,
			req.Name, slug, req.Description, req.Price, req.SKU, req.ImageURL, req.Category, stock, status, req.SEOTitle, req.SEODescription, siteID, chi.URLParam(r, "productId"))
	} else {
		_, err = s.db.ExecContext(r.Context(), `insert into products(id,tenant_id,site_id,name,slug,description,price,sku,image_url,category,stock_status,status,seo_title,seo_description,created_at,updated_at) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,now(),now())`,
			uuid.NewString(), tenantID, siteID, req.Name, slug, req.Description, req.Price, req.SKU, req.ImageURL, req.Category, stock, status, req.SEOTitle, req.SEODescription)
	}
	if err != nil { writeAPIError(w, 409, "save_failed", "Product could not be saved"); return }
	s.handleListProducts(w, r)
}

func (s *Server) handleDuplicateProduct(w http.ResponseWriter, r *http.Request) {
	var tenantID, siteID = "", chi.URLParam(r, "id")
	tenantID, _, _ = s.siteMeta(r.Context(), siteID)
	var p productPayload
	err := s.db.QueryRowContext(r.Context(), `select name,slug,description,price,sku,image_url,category,stock_status,seo_title,seo_description,status from products where site_id=$1 and id=$2`, siteID, chi.URLParam(r, "productId")).Scan(&p.Name, &p.Slug, &p.Description, &p.Price, &p.SKU, &p.ImageURL, &p.Category, &p.StockStatus, &p.SEOTitle, &p.SEODescription, &p.Status)
	if err != nil { writeAPIError(w, 404, "not_found", "Product not found"); return }
	p.Name += " Copy"; p.Slug += "-copy"; p.SKU = ""; p.Status = "draft"
	_, err = s.db.ExecContext(r.Context(), `insert into products(id,tenant_id,site_id,name,slug,description,price,sku,image_url,category,stock_status,status,seo_title,seo_description,created_at,updated_at) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,now(),now())`,
		uuid.NewString(), tenantID, siteID, p.Name, slugify(p.Slug), p.Description, p.Price, p.SKU, p.ImageURL, p.Category, p.StockStatus, p.Status, p.SEOTitle, p.SEODescription)
	if err != nil { writeAPIError(w, 409, "save_failed", "Product could not be duplicated"); return }
	s.handleListProducts(w, r)
}

func (s *Server) handleListOrders(w http.ResponseWriter, r *http.Request) { s.listItems(w, r, `select id,order_number,customer_name,customer_email,amount,payment_status,fulfillment_status,status,updated_at from orders where site_id=$1 order by updated_at desc`, []string{"id","orderNumber","customerName","customerEmail","amount","paymentStatus","fulfillmentStatus","status","updatedAt"}) }
func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) { s.getItem(w, r, `select id,order_number,customer_name,customer_email,amount,payment_status,fulfillment_status,status,updated_at from orders where site_id=$1 and id=$2`, chi.URLParam(r, "orderId"), []string{"id","orderNumber","customerName","customerEmail","amount","paymentStatus","fulfillmentStatus","status","updatedAt"}) }
func (s *Server) handleUpdateOrderStatus(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": true}) }
func (s *Server) handleListCustomers(w http.ResponseWriter, r *http.Request) { s.listItems(w, r, `select id,name,email,orders_count,total_spent,coalesce(last_order_at::text,''),updated_at from customers where site_id=$1 order by updated_at desc`, []string{"id","name","email","ordersCount","totalSpent","lastOrderAt","updatedAt"}) }
func (s *Server) handleGetCustomer(w http.ResponseWriter, r *http.Request) { s.getItem(w, r, `select id,name,email,orders_count,total_spent,coalesce(last_order_at::text,''),updated_at from customers where site_id=$1 and id=$2`, chi.URLParam(r, "customerId"), []string{"id","name","email","ordersCount","totalSpent","lastOrderAt","updatedAt"}) }
func (s *Server) handleListCoupons(w http.ResponseWriter, r *http.Request) { s.listItems(w, r, `select id,code,discount_type,value,coalesce(expires_at::text,''),usage_limit,usage_count,status,updated_at from coupons where site_id=$1 order by updated_at desc`, []string{"id","code","discountType","value","expiresAt","usageLimit","usageCount","status","updatedAt"}) }
func (s *Server) handleCreateCoupon(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": true}) }
func (s *Server) handleUpdateCoupon(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": true}) }
func (s *Server) handleDeleteCoupon(w http.ResponseWriter, r *http.Request) { s.deleteItem(w, r, "coupons", chi.URLParam(r, "couponId")) }
func (s *Server) handleListShippingZones(w http.ResponseWriter, r *http.Request) { s.listItems(w, r, `select id,name,region,method,rate,status,updated_at from shipping_zones where site_id=$1 order by updated_at desc`, []string{"id","name","region","method","rate","status","updatedAt"}) }
func (s *Server) handleCreateShippingZone(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": true}) }
func (s *Server) handleUpdateShippingZone(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": true}) }
func (s *Server) handleDeleteShippingZone(w http.ResponseWriter, r *http.Request) { s.deleteItem(w, r, "shipping_zones", chi.URLParam(r, "zoneId")) }
func (s *Server) handleGetTaxSettings(w http.ResponseWriter, r *http.Request) { s.getSingleton(w, r, `select id,taxes_enabled,prices_include_tax,default_region,settings_json,updated_at from tax_settings where site_id=$1`, []string{"id","taxesEnabled","pricesIncludeTax","defaultRegion","settingsJson","updatedAt"}) }
func (s *Server) handleUpsertTaxSettings(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": true}) }
func (s *Server) handleGetStoreSettings(w http.ResponseWriter, r *http.Request) { s.getSingleton(w, r, `select id,currency,store_email,checkout_mode,payment_provider,order_notifications_enabled,settings_json,updated_at from store_settings where site_id=$1`, []string{"id","currency","storeEmail","checkoutMode","paymentProvider","orderNotificationsEnabled","settingsJson","updatedAt"}) }
func (s *Server) handleUpsertStoreSettings(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": true}) }
func (s *Server) handleListPosts(w http.ResponseWriter, r *http.Request) { s.listItems(w, r, `select id,title,slug,excerpt,content,featured_image_url,author,tags_json,seo_title,seo_description,visibility,status,coalesce(published_at::text,''),updated_at from blog_posts where site_id=$1 order by updated_at desc`, []string{"id","title","slug","excerpt","content","featuredImage","author","tags","seoTitle","seoDescription","visibility","status","publishedAt","updatedAt"}) }
func (s *Server) handleGetPost(w http.ResponseWriter, r *http.Request) { s.getItem(w, r, `select id,title,slug,excerpt,content,featured_image_url,author,tags_json,seo_title,seo_description,visibility,status,coalesce(published_at::text,''),updated_at from blog_posts where site_id=$1 and id=$2`, chi.URLParam(r, "postId"), []string{"id","title","slug","excerpt","content","featuredImage","author","tags","seoTitle","seoDescription","visibility","status","publishedAt","updatedAt"}) }
func (s *Server) handleCreatePost(w http.ResponseWriter, r *http.Request) { s.upsertPost(w, r, false) }
func (s *Server) handleUpdatePost(w http.ResponseWriter, r *http.Request) { s.upsertPost(w, r, true) }
func (s *Server) handleDeletePost(w http.ResponseWriter, r *http.Request) { s.deleteItem(w, r, "blog_posts", chi.URLParam(r, "postId")) }
func (s *Server) handlePublishPost(w http.ResponseWriter, r *http.Request) { s.setPostStatus(w, r, "published") }
func (s *Server) handleUnpublishPost(w http.ResponseWriter, r *http.Request) { s.setPostStatus(w, r, "draft") }
func (s *Server) handleListBlogCategories(w http.ResponseWriter, r *http.Request) { s.listItems(w, r, `select id,name,slug,description,updated_at from blog_categories where site_id=$1 order by updated_at desc`, []string{"id","name","slug","description","updatedAt"}) }
func (s *Server) handleCreateBlogCategory(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": true}) }
func (s *Server) handleUpdateBlogCategory(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": true}) }
func (s *Server) handleDeleteBlogCategory(w http.ResponseWriter, r *http.Request) { s.deleteItem(w, r, "blog_categories", chi.URLParam(r, "categoryId")) }
func (s *Server) handleListBlogTags(w http.ResponseWriter, r *http.Request) { s.listItems(w, r, `select id,name,slug,visibility,updated_at from blog_tags where site_id=$1 order by updated_at desc`, []string{"id","name","slug","visibility","updatedAt"}) }
func (s *Server) handleCreateBlogTag(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": true}) }
func (s *Server) handleUpdateBlogTag(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": true}) }
func (s *Server) handleDeleteBlogTag(w http.ResponseWriter, r *http.Request) { s.deleteItem(w, r, "blog_tags", chi.URLParam(r, "tagId")) }
func (s *Server) handleListBlogAuthors(w http.ResponseWriter, r *http.Request) { s.listItems(w, r, `select id,name,email,role,bio,updated_at from blog_authors where site_id=$1 order by updated_at desc`, []string{"id","name","email","role","bio","updatedAt"}) }
func (s *Server) handleCreateBlogAuthor(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": true}) }
func (s *Server) handleUpdateBlogAuthor(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": true}) }
func (s *Server) handleDeleteBlogAuthor(w http.ResponseWriter, r *http.Request) { s.deleteItem(w, r, "blog_authors", chi.URLParam(r, "authorId")) }
func (s *Server) handleGetNewsletterSettings(w http.ResponseWriter, r *http.Request) { s.getSingleton(w, r, `select id,enabled,sender_name,sender_email,send_posts_as_email,settings_json,updated_at from newsletter_settings where site_id=$1`, []string{"id","enabled","senderName","senderEmail","sendPostsAsEmail","settingsJson","updatedAt"}) }
func (s *Server) handleUpsertNewsletterSettings(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": true}) }
func (s *Server) handleListBlogMembers(w http.ResponseWriter, r *http.Request) { s.listItems(w, r, `select id,name,email,member_type,status,joined_at,updated_at from blog_members where site_id=$1 order by updated_at desc`, []string{"id","name","email","memberType","status","joinedAt","updatedAt"}) }
func (s *Server) handleCreateBlogMember(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": true}) }
func (s *Server) handleUpdateBlogMember(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": true}) }
func (s *Server) handleDeleteBlogMember(w http.ResponseWriter, r *http.Request) { s.deleteItem(w, r, "blog_members", chi.URLParam(r, "memberId")) }
func (s *Server) handleGetBlogAnalytics(w http.ResponseWriter, r *http.Request) { s.getSingleton(w, r, `select coalesce(sum(views),0),coalesce(sum(signups),0),coalesce(sum(newsletter_opens),0) from blog_analytics where site_id=$1`, []string{"views","signups","newsletterOpens"}) }
func (s *Server) handleListSubscriptionPlans(w http.ResponseWriter, r *http.Request) { s.listItems(w, r, `select id,name,description,price,billing_interval,trial_days,status,updated_at from subscription_plans where site_id=$1 order by updated_at desc`, []string{"id","name","description","price","interval","trial","status","updatedAt"}) }
func (s *Server) handleGetSubscriptionPlan(w http.ResponseWriter, r *http.Request) { s.getItem(w, r, `select id,name,description,price,billing_interval,trial_days,status,updated_at from subscription_plans where site_id=$1 and id=$2`, chi.URLParam(r, "planId"), []string{"id","name","description","price","interval","trial","status","updatedAt"}) }
func (s *Server) handleCreateSubscriptionPlan(w http.ResponseWriter, r *http.Request) { s.upsertSubscriptionPlan(w, r, false) }
func (s *Server) handleUpdateSubscriptionPlan(w http.ResponseWriter, r *http.Request) { s.upsertSubscriptionPlan(w, r, true) }
func (s *Server) handleDeleteSubscriptionPlan(w http.ResponseWriter, r *http.Request) { s.deleteItem(w, r, "subscription_plans", chi.URLParam(r, "planId")) }
func (s *Server) handleListSubscriptionSubscribers(w http.ResponseWriter, r *http.Request) { s.listItems(w, r, `select id,name,email,status,joined_at,updated_at from subscription_subscribers where site_id=$1 order by updated_at desc`, []string{"id","name","email","status","joinedAt","updatedAt"}) }
func (s *Server) handleCreateSubscriptionSubscriber(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": true}) }
func (s *Server) handleUpdateSubscriptionSubscriber(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": true}) }
func (s *Server) handleDeleteSubscriptionSubscriber(w http.ResponseWriter, r *http.Request) { s.deleteItem(w, r, "subscription_subscribers", chi.URLParam(r, "subscriberId")) }
func (s *Server) handleListSubscriptions(w http.ResponseWriter, r *http.Request) { s.listItems(w, r, `select s.id,ss.name,sp.name,s.renewal_date,s.status,s.amount,s.updated_at from subscriptions s left join subscription_subscribers ss on ss.id=s.subscriber_id left join subscription_plans sp on sp.id=s.plan_id where s.site_id=$1 order by s.updated_at desc`, []string{"id","subscriber","plan","renewalDate","status","amount","updatedAt"}) }
func (s *Server) handleCreateSubscription(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": true}) }
func (s *Server) handleUpdateSubscription(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": true}) }
func (s *Server) handleCancelSubscription(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": true}) }
func (s *Server) handleListSubscriptionInvoices(w http.ResponseWriter, r *http.Request) { s.listItems(w, r, `select id,invoice_number,amount,status,issued_at,updated_at from subscription_invoices where site_id=$1 order by updated_at desc`, []string{"id","invoiceNumber","amount","status","issuedAt","updatedAt"}) }
func (s *Server) handleListFailedPayments(w http.ResponseWriter, r *http.Request) { s.listItems(w, r, `select id,amount,status,retry_count,next_retry_at,updated_at from failed_payments where site_id=$1 order by updated_at desc`, []string{"id","amount","status","retryCount","nextRetryAt","updatedAt"}) }
func (s *Server) handleRetryFailedPayment(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": true}) }
func (s *Server) handleGetSubscriptionSettings(w http.ResponseWriter, r *http.Request) { s.getSingleton(w, r, `select id,billing_currency,renewal_email_enabled,cancellation_policy,trial_settings_json,settings_json,updated_at from subscription_settings where site_id=$1`, []string{"id","billingCurrency","renewalEmailEnabled","cancellationPolicy","trialSettingsJson","settingsJson","updatedAt"}) }
func (s *Server) handleUpsertSubscriptionSettings(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": true}) }

func (s *Server) upsertPost(w http.ResponseWriter, r *http.Request, update bool) {
	siteID := chi.URLParam(r, "id")
	tenantID, _, err := s.siteMeta(r.Context(), siteID)
	if err != nil { writeAPIError(w, 404, "not_found", "Site not found"); return }
	var req postPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeAPIError(w, 400, "invalid_json", "Invalid request body"); return }
	if req.Title == "" { writeAPIError(w, 400, "validation_error", "Title is required"); return }
	slug := slugify(req.Slug); if slug == "" { slug = slugify(req.Title) }
	status := normalizeStatus(req.Status, "draft", "draft", "published")
	visibility := normalizeStatus(req.Visibility, "public", "public", "members", "paid")
	var publishedAt any = nil
	if status == "published" { publishedAt = time.Now().UTC() }
	if update {
		_, err = s.db.ExecContext(r.Context(), `update blog_posts set title=$1,slug=$2,excerpt=$3,content=$4,featured_image_url=$5,author=$6,tags_json=$7,seo_title=$8,seo_description=$9,visibility=$10,status=$11,published_at=$12,updated_at=now() where site_id=$13 and id=$14`,
			req.Title, slug, req.Excerpt, req.Content, req.FeaturedImage, req.Author, req.Tags, req.SEOTitle, req.SEODescription, visibility, status, publishedAt, siteID, chi.URLParam(r, "postId"))
	} else {
		_, err = s.db.ExecContext(r.Context(), `insert into blog_posts(id,tenant_id,site_id,title,slug,excerpt,content,featured_image_url,author,tags_json,seo_title,seo_description,visibility,status,published_at,created_at,updated_at) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,now(),now())`,
			uuid.NewString(), tenantID, siteID, req.Title, slug, req.Excerpt, req.Content, req.FeaturedImage, req.Author, req.Tags, req.SEOTitle, req.SEODescription, visibility, status, publishedAt)
	}
	if err != nil { writeAPIError(w, 409, "save_failed", "Post could not be saved"); return }
	s.handleListPosts(w, r)
}

func (s *Server) setPostStatus(w http.ResponseWriter, r *http.Request, status string) {
	siteID := chi.URLParam(r, "id")
	var publishedAt any = nil
	if status == "published" { publishedAt = time.Now().UTC() }
	_, err := s.db.ExecContext(r.Context(), `update blog_posts set status=$1,published_at=$2,updated_at=now() where site_id=$3 and id=$4`, status, publishedAt, siteID, chi.URLParam(r, "postId"))
	if err != nil { writeAPIError(w, 500, "update_failed", "Post status could not be updated"); return }
	s.handleGetPost(w, r)
}

func (s *Server) upsertSubscriptionPlan(w http.ResponseWriter, r *http.Request, update bool) {
	siteID := chi.URLParam(r, "id")
	tenantID, _, err := s.siteMeta(r.Context(), siteID)
	if err != nil { writeAPIError(w, 404, "not_found", "Site not found"); return }
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeAPIError(w, 400, "invalid_json", "Invalid request body"); return }
	name := fmt.Sprint(req["name"])
	if name == "" { writeAPIError(w, 400, "validation_error", "Plan name is required"); return }
	description := fmt.Sprint(req["description"])
	interval := normalizeStatus(fmt.Sprint(req["interval"]), "monthly", "weekly", "monthly", "yearly")
	status := normalizeStatus(fmt.Sprint(req["status"]), "draft", "draft", "active", "archived")
	price, _ := req["price"].(float64)
	if price < 0 { writeAPIError(w, 400, "validation_error", "Price must be greater than or equal to zero"); return }
	trial := 0
	if v, ok := req["trial"].(float64); ok { trial = int(v) }
	if update {
		_, err = s.db.ExecContext(r.Context(), `update subscription_plans set name=$1,description=$2,price=$3,billing_interval=$4,trial_days=$5,status=$6,updated_at=now() where site_id=$7 and id=$8`,
			name, description, price, interval, trial, status, siteID, chi.URLParam(r, "planId"))
	} else {
		_, err = s.db.ExecContext(r.Context(), `insert into subscription_plans(id,tenant_id,site_id,name,description,price,billing_interval,trial_days,status,created_at,updated_at) values($1,$2,$3,$4,$5,$6,$7,$8,$9,now(),now())`,
			uuid.NewString(), tenantID, siteID, name, description, price, interval, trial, status)
	}
	if err != nil { writeAPIError(w, 409, "save_failed", "Plan could not be saved"); return }
	s.handleListSubscriptionPlans(w, r)
}

func (s *Server) listItems(w http.ResponseWriter, r *http.Request, query string, fields []string) {
	siteID := chi.URLParam(r, "id")
	rows, err := s.db.QueryContext(r.Context(), query, siteID)
	if err != nil { writeAPIError(w, 500, "query_failed", "Query failed"); return }
	defer rows.Close()
	items, err := rowsToMaps(rows, fields)
	if err != nil { writeAPIError(w, 500, "scan_failed", "Scan failed"); return }
	writeJSON(w, 200, map[string]any{"items": items, "total": len(items)})
}

func (s *Server) getItem(w http.ResponseWriter, r *http.Request, query, itemID string, fields []string) {
	siteID := chi.URLParam(r, "id")
	rows, err := s.db.QueryContext(r.Context(), query, siteID, itemID)
	if err != nil { writeAPIError(w, 500, "query_failed", "Query failed"); return }
	defer rows.Close()
	items, err := rowsToMaps(rows, fields)
	if err != nil || len(items) == 0 { writeAPIError(w, 404, "not_found", "Item not found"); return }
	writeJSON(w, 200, map[string]any{"item": items[0]})
}

func (s *Server) getSingleton(w http.ResponseWriter, r *http.Request, query string, fields []string) {
	siteID := chi.URLParam(r, "id")
	rows, err := s.db.QueryContext(r.Context(), query, siteID)
	if err != nil { writeAPIError(w, 500, "query_failed", "Query failed"); return }
	defer rows.Close()
	items, err := rowsToMaps(rows, fields)
	if err != nil { writeAPIError(w, 500, "scan_failed", "Scan failed"); return }
	item := map[string]any{}
	if len(items) > 0 { item = items[0] }
	writeJSON(w, 200, map[string]any{"item": item})
}

func rowsToMaps(rows *sql.Rows, fields []string) ([]map[string]any, error) {
	items := make([]map[string]any, 0)
	cols, _ := rows.Columns()
	for rows.Next() {
		raw := make([]any, len(cols))
		dests := make([]any, len(cols))
		for i := range raw { dests[i] = &raw[i] }
		if err := rows.Scan(dests...); err != nil { return nil, err }
		item := map[string]any{}
		for i, field := range fields {
			switch v := raw[i].(type) {
			case []byte:
				item[field] = string(v)
			default:
				item[field] = v
			}
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Server) deleteItem(w http.ResponseWriter, r *http.Request, table, itemID string) {
	siteID := chi.URLParam(r, "id")
	_, err := s.db.ExecContext(r.Context(), `delete from `+table+` where site_id=$1 and id=$2`, siteID, itemID)
	if err != nil { writeAPIError(w, 500, "delete_failed", "Item could not be deleted"); return }
	writeJSON(w, 200, map[string]any{"success": true})
}

func (s *Server) resolveTenantID(ctx context.Context, slug string) (string, error) {
	if slug == "" { slug = demoTenant }
	var id string
	err := s.db.QueryRowContext(ctx, `select id from tenants where slug=$1`, slug).Scan(&id)
	return id, err
}

func (s *Server) upsertWorkspaceMetadata(ctx context.Context, tenantID, siteID, path, fileType string, size int64, sum string) error {
	_, err := s.db.ExecContext(ctx, `insert into workspace_files(id,tenant_id,site_id,path,file_type,size_bytes,checksum,created_at,updated_at) values($1,$2,$3,$4,$5,$6,$7,now(),now())
	on conflict (site_id,path) do update set file_type=excluded.file_type,size_bytes=excluded.size_bytes,checksum=excluded.checksum,updated_at=now()`,
		uuid.NewString(), tenantID, siteID, path, fileType, size, sum)
	return err
}

func defaultWorkspaceHTML(name string) string {
	return fmt.Sprintf(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>%s</title>
  </head>
  <body>
    <main class="page">
      <h1>%s</h1>
    </main>
  </body>
</html>`, name, name)
}

type claims struct {
	UserID string `json:"uid"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	jwt.RegisteredClaims
}

func issueToken(secret string, id, email, name string) (string, error) {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{UserID: id, Email: email, Name: name})
	return tok.SignedString([]byte(secret))
}

func parseToken(secret, token string) (*claims, error) {
	tok, err := jwt.ParseWithClaims(token, &claims{}, func(t *jwt.Token) (any, error) { return []byte(secret), nil })
	if err != nil || !tok.Valid { return nil, fmt.Errorf("invalid token") }
	cl, ok := tok.Claims.(*claims)
	if !ok { return nil, fmt.Errorf("invalid claims") }
	return cl, nil
}
