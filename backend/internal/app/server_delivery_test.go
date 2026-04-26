package app

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestContentTypeForPath(t *testing.T) {
	cases := map[string]string{
		"index.html":  "text/html; charset=utf-8",
		"styles.css":  "text/css; charset=utf-8",
		"script.js":   "application/javascript; charset=utf-8",
		"data.json":   "application/json; charset=utf-8",
		"logo.png":    "image/png",
		"photo.jpg":   "image/jpeg",
		"photo.jpeg":  "image/jpeg",
		"icon.svg":    "image/svg+xml",
		"archive.bin": "application/octet-stream",
	}

	for path, want := range cases {
		if got := contentTypeForPath(path); got != want {
			t.Fatalf("%s: got %q want %q", path, got, want)
		}
	}
}

func TestResolveDeliveryLookup(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want deliveryLookup
	}{
		{
			name: "site root by id",
			url:  "/sites/site-123",
			want: deliveryLookup{SiteID: "site-123", Slug: "", RequestPath: ""},
		},
		{
			name: "site file by id compatibility",
			url:  "/sites/site-123/index.html",
			want: deliveryLookup{SiteID: "site-123", Slug: "", RequestPath: "index.html"},
		},
		{
			name: "canonical slug root",
			url:  "/sites/site-123/my-site",
			want: deliveryLookup{SiteID: "site-123", Slug: "my-site", RequestPath: ""},
		},
		{
			name: "canonical slug file",
			url:  "/sites/site-123/my-site/shop.html",
			want: deliveryLookup{SiteID: "site-123", Slug: "my-site", RequestPath: "shop.html"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.url, nil)
			rctx := chiRouteContextForURL(tc.url)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			got := resolveDeliveryLookup(req)
			if got != tc.want {
				t.Fatalf("got %#v want %#v", got, tc.want)
			}
		})
	}
}

func TestNormalizeDeliveryPath(t *testing.T) {
	cases := []struct {
		path string
		want string
		ok   bool
	}{
		{"", "index.html", true},
		{".", "index.html", true},
		{"shop.html", "shop.html", true},
		{"nested/about.html", "nested/about.html", true},
		{"../workspace/index.html", "", false},
		{"/absolute/path.html", "", false},
	}
	for _, tc := range cases {
		got, ok := normalizeDeliveryPath(tc.path)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("%q => (%q,%v), want (%q,%v)", tc.path, got, ok, tc.want, tc.ok)
		}
	}
}

func chiRouteContextForURL(url string) *chi.Context {
	trimmed := url
	if len(trimmed) >= len("/sites/") && trimmed[:7] == "/sites/" {
		trimmed = trimmed[7:]
	}
	parts := splitNonEmpty(trimmed, '/')
	rctx := chi.NewRouteContext()
	if len(parts) > 0 {
		rctx.URLParams.Add("siteID", parts[0])
	}
	if len(parts) > 1 {
		rctx.URLParams.Add("slug", parts[1])
	}
	if len(parts) > 2 {
		tail := ""
		for i, part := range parts[2:] {
			if i > 0 {
				tail += "/"
			}
			tail += part
		}
		rctx.URLParams.Add("*", tail)
	}
	return rctx
}

func splitNonEmpty(value string, sep rune) []string {
	current := ""
	var items []string
	for _, r := range value {
		if r == sep {
			if current != "" {
				items = append(items, current)
				current = ""
			}
			continue
		}
		current += string(r)
	}
	if current != "" {
		items = append(items, current)
	}
	return items
}
