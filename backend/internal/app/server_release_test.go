package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	renderpkg "sitecraft/backend/internal/app/renderer"
)

func TestAtomicWriteFileOverwritesFully(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "workspace", "index.html")

	if err := atomicWriteFile(target, []byte("first"), 0o644); err != nil {
		t.Fatalf("first write failed: %v", err)
	}
	if err := atomicWriteFile(target, []byte("second"), 0o644); err != nil {
		t.Fatalf("second write failed: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(got) != "second" {
		t.Fatalf("got %q want %q", string(got), "second")
	}
}

func TestCopyDirCopiesNestedWorkspaceTree(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "workspace")
	dst := filepath.Join(root, "releases", "r1")

	if err := atomicWriteFile(filepath.Join(src, "index.html"), []byte("<html>ok</html>"), 0o644); err != nil {
		t.Fatalf("write index failed: %v", err)
	}
	if err := atomicWriteFile(filepath.Join(src, "assets", "styles.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatalf("write css failed: %v", err)
	}

	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}

	cases := []string{
		filepath.Join(dst, "index.html"),
		filepath.Join(dst, "assets", "styles.css"),
	}
	for _, path := range cases {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}
}

func TestRenderWorkspaceDirRendersHTMLAndCopiesAssets(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "workspace")
	dst := filepath.Join(root, "releases", "r1")

	if err := atomicWriteFile(filepath.Join(src, "shop.html"), []byte(`<section>{% for product in products %}<h3>{{ product.name }}</h3>{% endfor %}</section>`), 0o644); err != nil {
		t.Fatalf("write html failed: %v", err)
	}
	if err := atomicWriteFile(filepath.Join(src, "index.html"), []byte(`<html><body>{{ site.name }}</body></html>`), 0o644); err != nil {
		t.Fatalf("write index failed: %v", err)
	}
	if err := atomicWriteFile(filepath.Join(src, "styles.css"), []byte(`body{color:#111;}`), 0o644); err != nil {
		t.Fatalf("write css failed: %v", err)
	}

	renderCtx := renderpkg.RenderContext{
		Site: renderpkg.SiteContext{Name: "Client Advisory Studio"},
		Products: []renderpkg.Product{
			{Name: "Strategy Sprint", Price: 999},
			{Name: "Operator Playbook", Price: 1299},
		},
	}

	renderErrors, err := renderWorkspaceDir(src, dst, renderCtx)
	if err != nil {
		t.Fatalf("renderWorkspaceDir failed: %v", err)
	}
	if len(renderErrors) > 0 {
		t.Fatalf("unexpected render errors: %+v", renderErrors)
	}

	htmlOut, err := os.ReadFile(filepath.Join(dst, "shop.html"))
	if err != nil {
		t.Fatalf("read rendered html failed: %v", err)
	}
	if string(htmlOut) == "" || !containsAll(string(htmlOut), "Strategy Sprint", "Operator Playbook") {
		t.Fatalf("unexpected rendered html: %q", string(htmlOut))
	}
	if containsAny(string(htmlOut), "{% for", "{{ product.name }}") {
		t.Fatalf("template syntax leaked into release: %q", string(htmlOut))
	}

	cssOut, err := os.ReadFile(filepath.Join(dst, "styles.css"))
	if err != nil {
		t.Fatalf("read copied css failed: %v", err)
	}
	if string(cssOut) != `body{color:#111;}` {
		t.Fatalf("unexpected css copy: %q", string(cssOut))
	}
}

func TestRenderWorkspaceDirReturnsValidationErrors(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "workspace")
	dst := filepath.Join(root, "releases", "r1")

	if err := atomicWriteFile(filepath.Join(src, "shop.html"), []byte(`<section>{{ product.fake }}</section>`), 0o644); err != nil {
		t.Fatalf("write html failed: %v", err)
	}
	if err := atomicWriteFile(filepath.Join(src, "index.html"), []byte(`<html><body>{{ site.name }}</body></html>`), 0o644); err != nil {
		t.Fatalf("write index failed: %v", err)
	}

	renderErrors, err := renderWorkspaceDir(src, dst, renderpkg.RenderContext{
		Site: renderpkg.SiteContext{Name: "Client Advisory Studio"},
	})
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}
	if len(renderErrors) == 0 {
		t.Fatalf("expected validation errors")
	}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}

func containsAny(value string, parts ...string) bool {
	for _, part := range parts {
		if strings.Contains(value, part) {
			return true
		}
	}
	return false
}
