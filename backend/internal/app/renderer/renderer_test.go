package renderer

import (
	"strings"
	"testing"
)

func testContext() RenderContext {
	return RenderContext{
		Site: SiteContext{
			ID:        "site-1",
			TenantID:  "tenant-1",
			Name:      "Client Advisory Studio",
			Slug:      "client-advisory-studio",
			PublicURL: "http://localhost:8083/sites/site-1/client-advisory-studio",
		},
		Products: []Product{
			{Name: "Strategy Sprint", Description: "Rapid advisory engagement", Price: 999.0, Slug: "strategy-sprint", Category: "Advisory", Status: "active"},
			{Name: "Revenue Systems Audit", Description: "Operational review", Price: 1499.0, Slug: "revenue-systems-audit", Category: "Operations", Status: "active"},
		},
		Posts: []Post{
			{Title: "How firms scale delivery", Excerpt: "A practical growth note.", Slug: "scale-delivery", Author: "Ava Chen", Status: "published"},
		},
		SubscriptionPlans: []Plan{
			{Name: "Advisory Retainer", Description: "Monthly advisory support", Price: 2999.0, BillingInterval: "monthly", TrialDays: 14, Status: "active"},
		},
	}
}

func TestRenderPlainHTMLUnchanged(t *testing.T) {
	r := New()
	got, errs := r.Render("<section><h1>Hello</h1></section>", testContext())
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if got != "<section><h1>Hello</h1></section>" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderCommentsRemoved(t *testing.T) {
	r := New()
	got, errs := r.Render("<section>{# hidden #}<h1>Hello</h1></section>", testContext())
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if got != "<section><h1>Hello</h1></section>" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderSiteName(t *testing.T) {
	r := New()
	got, errs := r.Render("<h1>{{ site.name }}</h1>", testContext())
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if !strings.Contains(got, "Client Advisory Studio") {
		t.Fatalf("missing rendered site name: %q", got)
	}
}

func TestRenderProductLoop(t *testing.T) {
	r := New()
	got, errs := r.Render("{% for product in products %}<h3>{{ product.name }}</h3>{% endfor %}", testContext())
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if !strings.Contains(got, "Strategy Sprint") || !strings.Contains(got, "Revenue Systems Audit") {
		t.Fatalf("missing rendered products: %q", got)
	}
}

func TestRenderPostLoop(t *testing.T) {
	r := New()
	got, errs := r.Render("{% for post in posts %}<h3>{{ post.title }}</h3>{% endfor %}", testContext())
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if !strings.Contains(got, "How firms scale delivery") {
		t.Fatalf("missing rendered post: %q", got)
	}
}

func TestRenderPlanLoop(t *testing.T) {
	r := New()
	got, errs := r.Render("{% for plan in subscription_plans %}<h3>{{ plan.name }}</h3>{% endfor %}", testContext())
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if !strings.Contains(got, "Advisory Retainer") {
		t.Fatalf("missing rendered plan: %q", got)
	}
}

func TestMoneyFilter(t *testing.T) {
	r := New()
	got, errs := r.Render("{% for product in products %}{{ product.price | money }}{% endfor %}", RenderContext{
		Site: testContext().Site,
		Products: []Product{{Name: "One", Price: 999}},
	})
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if got != "INR 999.00" {
		t.Fatalf("got %q", got)
	}
}

func TestEscapeFilter(t *testing.T) {
	r := New()
	got, errs := r.Render("{{ site.name | escape }}", RenderContext{
		Site: SiteContext{Name: "<Studio>"},
	})
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if got != "&lt;Studio&gt;" {
		t.Fatalf("got %q", got)
	}
}

func TestURLFilter(t *testing.T) {
	r := New()
	got, errs := r.Render("{% for post in posts %}{{ post.slug | url }}{% endfor %}", RenderContext{
		Posts: []Post{{Slug: "hello world"}},
	})
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if got != "hello%20world" {
		t.Fatalf("got %q", got)
	}
}

func TestUnknownVariableReturnsError(t *testing.T) {
	r := New()
	_, errs := r.Render("{{ product.fake }}", testContext())
	if len(errs) == 0 || !strings.Contains(errs[0].Message, "Unknown variable") {
		t.Fatalf("expected unknown variable error, got %+v", errs)
	}
}

func TestUnknownCollectionReturnsError(t *testing.T) {
	r := New()
	_, errs := r.Render("{% for x in unknown %}{{ x }}{% endfor %}", testContext())
	if len(errs) == 0 {
		t.Fatalf("expected collection error")
	}
}

func TestUnknownFilterReturnsError(t *testing.T) {
	r := New()
	_, errs := r.Render("{% for product in products %}{{ product.name | unknownfilter }}{% endfor %}", testContext())
	if len(errs) == 0 || !strings.Contains(errs[0].Message, "Unknown filter") {
		t.Fatalf("expected unknown filter error, got %+v", errs)
	}
}

func TestMalformedLoopReturnsError(t *testing.T) {
	r := New()
	_, errs := r.Render("{% for product in products %}<h3>{{ product.name }}</h3>", testContext())
	if len(errs) == 0 || !strings.Contains(errs[0].Message, "Unclosed loop tag") {
		t.Fatalf("expected loop error, got %+v", errs)
	}
}

func TestDangerousPayloadsFailValidation(t *testing.T) {
	r := New()
	cases := []string{
		"{{ db.query }}",
		"{{ os.Getenv }}",
		"{{ system.exec }}",
		"{{ product.fake_field }}",
		"{% for x in unknown %}{{ x }}{% endfor %}",
		"{{ product.name | unknownfilter }}",
		"{% for product in products %}{{ product.name }}",
	}
	for _, tpl := range cases {
		if errs := r.Validate(tpl, testContext()); len(errs) == 0 {
			t.Fatalf("expected validation error for %q", tpl)
		}
	}
}
