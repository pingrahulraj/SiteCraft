package app

import (
	"context"
	"database/sql"
	"fmt"
	"html"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	renderpkg "sitecraft/backend/internal/app/renderer"
)

func (s *Server) buildRenderContext(ctx context.Context, tenantID, siteID, slug string) (renderpkg.RenderContext, error) {
	return buildRenderContextFromDB(ctx, s.db, s.cfg.deliveryURL(), tenantID, siteID, slug)
}

func buildRenderContextFromDB(ctx context.Context, db *sql.DB, deliveryBase, tenantID, siteID, slug string) (renderpkg.RenderContext, error) {
	renderCtx := renderpkg.RenderContext{
		Site: renderpkg.SiteContext{
			ID:        siteID,
			TenantID:  tenantID,
			Slug:      slug,
			PublicURL: publicSiteURL(deliveryBase, siteID, slug),
		},
	}
	if err := db.QueryRowContext(ctx, `select name from sites where id=$1 and tenant_id=$2`, siteID, tenantID).Scan(&renderCtx.Site.Name); err != nil {
		return renderCtx, err
	}

	productRows, err := db.QueryContext(ctx, `
		select id,name,slug,description,price,sku,image_url,category,stock_status,status
		from products
		where tenant_id=$1 and site_id=$2 and status='active'
		order by updated_at desc, created_at desc
	`, tenantID, siteID)
	if err != nil {
		return renderCtx, err
	}
	defer productRows.Close()
	for productRows.Next() {
		var product renderpkg.Product
		if err := productRows.Scan(&product.ID, &product.Name, &product.Slug, &product.Description, &product.Price, &product.SKU, &product.ImageURL, &product.Category, &product.StockStatus, &product.Status); err != nil {
			return renderCtx, err
		}
		renderCtx.Products = append(renderCtx.Products, product)
	}

	postRows, err := db.QueryContext(ctx, `
		select id,title,slug,excerpt,content,featured_image_url,author,status,coalesce(to_char(published_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),'')
		from blog_posts
		where tenant_id=$1 and site_id=$2 and status='published'
		order by published_at desc nulls last, updated_at desc
	`, tenantID, siteID)
	if err != nil {
		return renderCtx, err
	}
	defer postRows.Close()
	for postRows.Next() {
		var post renderpkg.Post
		if err := postRows.Scan(&post.ID, &post.Title, &post.Slug, &post.Excerpt, &post.Content, &post.FeaturedImageURL, &post.Author, &post.Status, &post.PublishedAt); err != nil {
			return renderCtx, err
		}
		renderCtx.Posts = append(renderCtx.Posts, post)
	}

	planRows, err := db.QueryContext(ctx, `
		select id,name,description,price,billing_interval,trial_days,status
		from subscription_plans
		where tenant_id=$1 and site_id=$2 and status='active'
		order by updated_at desc, created_at desc
	`, tenantID, siteID)
	if err != nil {
		return renderCtx, err
	}
	defer planRows.Close()
	for planRows.Next() {
		var plan renderpkg.Plan
		if err := planRows.Scan(&plan.ID, &plan.Name, &plan.Description, &plan.Price, &plan.BillingInterval, &plan.TrialDays, &plan.Status); err != nil {
			return renderCtx, err
		}
		renderCtx.SubscriptionPlans = append(renderCtx.SubscriptionPlans, plan)
	}

	return renderCtx, nil
}

func renderWorkspaceDir(srcRoot, dstRoot string, renderCtx renderpkg.RenderContext) ([]renderpkg.RenderError, error) {
	engine := renderpkg.New()
	var renderErrors []renderpkg.RenderError
	err := filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(srcRoot, path)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return os.MkdirAll(dstRoot, 0o755)
		}
		target := filepath.Join(dstRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.EqualFold(filepath.Ext(path), ".html") {
			rendered, errs := engine.Render(string(content), renderCtx)
			if len(errs) > 0 {
				for _, item := range errs {
					item.File = filepath.ToSlash(rel)
					renderErrors = append(renderErrors, item)
				}
				return nil
			}
			return atomicWriteFile(target, []byte(rendered), 0o644)
		}
		return atomicWriteFile(target, content, 0o644)
	})
	if err != nil {
		return nil, err
	}
	if len(renderErrors) > 0 {
		return renderErrors, nil
	}
	if _, err := os.Stat(filepath.Join(dstRoot, "index.html")); err != nil {
		return nil, fmt.Errorf("release validation failed")
	}
	return nil, nil
}

func releaseErrorsPayload(errs []renderpkg.RenderError) []map[string]any {
	items := make([]map[string]any, 0, len(errs))
	for _, item := range errs {
		items = append(items, map[string]any{
			"file":    item.File,
			"line":    item.Line,
			"column":  item.Column,
			"message": item.Message,
		})
	}
	return items
}

func currentActiveReleaseID(ctx context.Context, conn *sql.Conn, siteID string) string {
	var activeReleaseID sql.NullString
	_ = conn.QueryRowContext(ctx, `select active_release_id::text from sites where id=$1`, siteID).Scan(&activeReleaseID)
	if activeReleaseID.Valid {
		return activeReleaseID.String
	}
	return ""
}

func renderWorkspaceHTMLFile(srcRoot, relativePath string, renderCtx renderpkg.RenderContext) (string, []renderpkg.RenderError, error) {
	engine := renderpkg.New()
	targetPath := filepath.Join(srcRoot, filepath.FromSlash(relativePath))
	content, err := os.ReadFile(targetPath)
	if err != nil {
		return "", nil, err
	}
	rendered, errs := engine.Render(string(content), renderCtx)
	if len(errs) > 0 {
		for index := range errs {
			errs[index].File = filepath.ToSlash(relativePath)
		}
		return "", errs, nil
	}
	cssBlocks, err := readWorkspaceAssets(srcRoot, ".css")
	if err != nil {
		return "", nil, err
	}
	jsBlocks, err := readWorkspaceAssets(srcRoot, ".js")
	if err != nil {
		return "", nil, err
	}
	return injectWorkspaceAssets(rendered, cssBlocks, jsBlocks), nil, nil
}

func readWorkspaceAssets(srcRoot, ext string) ([]string, error) {
	var blocks []string
	err := filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(path), ext) {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		blocks = append(blocks, string(content))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return blocks, nil
}

func injectWorkspaceAssets(doc string, cssBlocks, jsBlocks []string) string {
	headClose := "</head>"
	bodyClose := "</body>"
	if len(cssBlocks) > 0 {
		css := "<style>\n" + strings.Join(cssBlocks, "\n") + "\n</style>"
		if strings.Contains(doc, headClose) {
			doc = strings.Replace(doc, headClose, css+headClose, 1)
		} else {
			doc = "<!doctype html><html><head>" + css + "</head><body>" + doc + "</body></html>"
		}
	}
	if len(jsBlocks) > 0 {
		js := "<script>\n" + strings.Join(jsBlocks, "\n") + "\n</script>"
		if strings.Contains(doc, bodyClose) {
			doc = strings.Replace(doc, bodyClose, js+bodyClose, 1)
		} else {
			doc += js
		}
	}
	return doc
}

func previewErrorHTML(errs []renderpkg.RenderError) string {
	var rows []string
	for _, item := range errs {
		rows = append(rows, fmt.Sprintf(
			"<li><strong>%s</strong> line %d, column %d: %s</li>",
			html.EscapeString(item.File),
			item.Line,
			item.Column,
			html.EscapeString(item.Message),
		))
	}
	return "<!doctype html><html><head><meta charset=\"utf-8\"><title>Preview error</title><style>body{font-family:Segoe UI,Arial,sans-serif;padding:24px;background:#f8fafc;color:#14202b}main{max-width:840px;margin:0 auto;background:#fff;border:1px solid #d8e0e8;padding:24px;border-radius:6px}h1{margin-top:0}li+li{margin-top:10px}code{background:#eef2f5;padding:2px 6px;border-radius:4px}</style></head><body><main><p>Preview error</p><h1>Template rendering failed</h1><p>The preview could not be rendered because one or more template tags are invalid.</p><ul>" + strings.Join(rows, "") + "</ul></main></body></html>"
}
