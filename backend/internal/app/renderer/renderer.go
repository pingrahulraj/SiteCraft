package renderer

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	identifierPattern = regexp.MustCompile(`^[a-z_][a-z0-9_]*(\.[a-z_][a-z0-9_]*)*$`)
	forPattern        = regexp.MustCompile(`^for\s+([a-z_][a-z0-9_]*)\s+in\s+([a-z_][a-z0-9_]*)$`)
)

type RenderContext struct {
	Site              SiteContext   `json:"site"`
	Products          []Product     `json:"products"`
	Posts             []Post        `json:"posts"`
	SubscriptionPlans []Plan        `json:"subscription_plans"`
}

type SiteContext struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	PublicURL string `json:"public_url"`
}

type Product struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Price       any    `json:"price"`
	SKU         string `json:"sku"`
	ImageURL    string `json:"image_url"`
	Category    string `json:"category"`
	StockStatus string `json:"stock_status"`
	Status      string `json:"status"`
}

type Post struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Slug             string `json:"slug"`
	Excerpt          string `json:"excerpt"`
	Content          string `json:"content"`
	FeaturedImageURL string `json:"featured_image_url"`
	Author           string `json:"author"`
	Status           string `json:"status"`
	PublishedAt      string `json:"published_at"`
}

type Plan struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Price           any    `json:"price"`
	BillingInterval string `json:"billing_interval"`
	TrialDays       any    `json:"trial_days"`
	Status          string `json:"status"`
}

type RenderError struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
}

type Renderer struct{}

func New() *Renderer {
	return &Renderer{}
}

func (r *Renderer) Validate(template string, ctx RenderContext) []RenderError {
	_, errs := r.Render(template, ctx)
	return errs
}

func (r *Renderer) Render(template string, ctx RenderContext) (string, []RenderError) {
	return r.render(template, ctx, "", map[string]map[string]any{
		"site": {
			"id":         ctx.Site.ID,
			"tenant_id":  ctx.Site.TenantID,
			"name":       ctx.Site.Name,
			"slug":       ctx.Site.Slug,
			"public_url": ctx.Site.PublicURL,
		},
	})
}

func (r *Renderer) render(template string, ctx RenderContext, file string, scope map[string]map[string]any) (string, []RenderError) {
	template = stripComments(template)
	var out strings.Builder
	var errs []RenderError

	for i := 0; i < len(template); {
		switch {
		case strings.HasPrefix(template[i:], "{{"):
			end := strings.Index(template[i+2:], "}}")
			if end < 0 {
				errs = append(errs, renderError(file, template, i, "Unclosed variable tag"))
				return "", errs
			}
			end += i + 2
			expr := strings.TrimSpace(template[i+2 : end])
			value, err := resolveExpression(expr, scope)
			if err != nil {
				errs = append(errs, renderError(file, template, i, err.Error()))
			} else {
				out.WriteString(value)
			}
			i = end + 2
		case strings.HasPrefix(template[i:], "{%"):
			tagEnd := strings.Index(template[i+2:], "%}")
			if tagEnd < 0 {
				errs = append(errs, renderError(file, template, i, "Unclosed block tag"))
				return "", errs
			}
			tagEnd += i + 2
			tagBody := strings.TrimSpace(template[i+2 : tagEnd])
			if tagBody == "endfor" {
				errs = append(errs, renderError(file, template, i, "Unexpected endfor"))
				i = tagEnd + 2
				continue
			}
			match := forPattern.FindStringSubmatch(tagBody)
			if len(match) != 3 {
				errs = append(errs, renderError(file, template, i, "Malformed loop tag"))
				i = tagEnd + 2
				continue
			}
			itemName, collectionName := match[1], match[2]
			if !validLoopBinding(itemName, collectionName) {
				if collectionName != "products" && collectionName != "posts" && collectionName != "subscription_plans" {
					errs = append(errs, renderError(file, template, i, fmt.Sprintf("Unknown collection: %s", collectionName)))
				} else {
					errs = append(errs, renderError(file, template, i, "Malformed loop tag"))
				}
				i = tagEnd + 2
				continue
			}
			loopBody, nextIndex, loopErr := extractLoopBody(template, tagEnd+2, file)
			if loopErr != nil {
				errs = append(errs, *loopErr)
				return "", errs
			}
			if strings.Contains(loopBody, "{% for") {
				errs = append(errs, renderError(file, template, i, "Nested loops are not supported"))
				i = nextIndex
				continue
			}
			items, err := collectionForName(collectionName, ctx)
			if err != nil {
				errs = append(errs, renderError(file, template, i, err.Error()))
				i = nextIndex
				continue
			}
			for _, item := range items {
				childScope := cloneScope(scope)
				childScope[itemName] = item
				rendered, childErrs := r.render(loopBody, ctx, file, childScope)
				if len(childErrs) > 0 {
					errs = append(errs, childErrs...)
					continue
				}
				out.WriteString(rendered)
			}
			i = nextIndex
		default:
			out.WriteByte(template[i])
			i++
		}
	}

	if len(errs) > 0 {
		return "", errs
	}
	return out.String(), nil
}

func extractLoopBody(template string, start int, file string) (string, int, *RenderError) {
	endIndex := strings.Index(template[start:], "{% endfor %}")
	if endIndex < 0 {
		err := renderError(file, template, start, "Unclosed loop tag")
		return "", 0, &err
	}
	endIndex += start
	return template[start:endIndex], endIndex + len("{% endfor %}"), nil
}

func collectionForName(name string, ctx RenderContext) ([]map[string]any, error) {
	switch name {
	case "products":
		items := make([]map[string]any, 0, len(ctx.Products))
		for _, product := range ctx.Products {
			items = append(items, map[string]any{
				"id":           product.ID,
				"name":         product.Name,
				"slug":         product.Slug,
				"description":  product.Description,
				"price":        product.Price,
				"sku":          product.SKU,
				"image_url":    product.ImageURL,
				"category":     product.Category,
				"stock_status": product.StockStatus,
				"status":       product.Status,
			})
		}
		return items, nil
	case "posts":
		items := make([]map[string]any, 0, len(ctx.Posts))
		for _, post := range ctx.Posts {
			items = append(items, map[string]any{
				"id":                 post.ID,
				"title":              post.Title,
				"slug":               post.Slug,
				"excerpt":            post.Excerpt,
				"content":            post.Content,
				"featured_image_url": post.FeaturedImageURL,
				"author":             post.Author,
				"status":             post.Status,
				"published_at":       post.PublishedAt,
			})
		}
		return items, nil
	case "subscription_plans":
		items := make([]map[string]any, 0, len(ctx.SubscriptionPlans))
		for _, plan := range ctx.SubscriptionPlans {
			items = append(items, map[string]any{
				"id":               plan.ID,
				"name":             plan.Name,
				"description":      plan.Description,
				"price":            plan.Price,
				"billing_interval": plan.BillingInterval,
				"trial_days":       plan.TrialDays,
				"status":           plan.Status,
			})
		}
		return items, nil
	default:
		return nil, fmt.Errorf("Unknown collection: %s", name)
	}
}

func resolveExpression(expr string, scope map[string]map[string]any) (string, error) {
	parts := strings.Split(expr, "|")
	for idx := range parts {
		parts[idx] = strings.TrimSpace(parts[idx])
	}
	if len(parts) == 0 || parts[0] == "" {
		return "", fmt.Errorf("Empty variable expression")
	}
	if !identifierPattern.MatchString(parts[0]) {
		return "", fmt.Errorf("Unknown variable: %s", parts[0])
	}

	raw, err := lookupValue(parts[0], scope)
	if err != nil {
		return "", err
	}
	value := stringify(raw)
	alreadyEscaped := false

	for _, filter := range parts[1:] {
		switch filter {
		case "money":
			value = html.EscapeString(formatMoney(raw))
			alreadyEscaped = true
		case "escape":
			value = html.EscapeString(value)
			alreadyEscaped = true
		case "url":
			value = url.PathEscape(value)
			alreadyEscaped = true
		case "":
			return "", fmt.Errorf("Unknown filter: ")
		default:
			return "", fmt.Errorf("Unknown filter: %s", filter)
		}
	}

	if !alreadyEscaped {
		value = html.EscapeString(value)
	}
	return value, nil
}

func lookupValue(path string, scope map[string]map[string]any) (any, error) {
	segments := strings.Split(path, ".")
	root, ok := scope[segments[0]]
	if !ok {
		return "", fmt.Errorf("Unknown variable: %s", path)
	}
	var current any = root
	for _, segment := range segments[1:] {
		currentMap, ok := current.(map[string]any)
		if !ok {
			return "", fmt.Errorf("Unknown variable: %s", path)
		}
		next, ok := currentMap[segment]
		if !ok {
			return "", fmt.Errorf("Unknown variable: %s", path)
		}
		current = next
	}
	return current, nil
}

func stringify(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return fmt.Sprint(v)
	}
}

func formatMoney(value any) string {
	number := 0.0
	switch v := value.(type) {
	case int:
		number = float64(v)
	case int32:
		number = float64(v)
	case int64:
		number = float64(v)
	case float32:
		number = float64(v)
	case float64:
		number = v
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err == nil {
			number = parsed
		}
	default:
		parsed, err := strconv.ParseFloat(fmt.Sprint(v), 64)
		if err == nil {
			number = parsed
		}
	}
	return fmt.Sprintf("INR %.2f", number)
}

func stripComments(template string) string {
	for {
		start := strings.Index(template, "{#")
		if start < 0 {
			return template
		}
		end := strings.Index(template[start+2:], "#}")
		if end < 0 {
			return template
		}
		end += start + 2
		template = template[:start] + template[end+2:]
	}
}

func renderError(file, template string, index int, message string) RenderError {
	line, column := lineColumn(template, index)
	return RenderError{File: file, Line: line, Column: column, Message: message}
}

func lineColumn(template string, index int) (int, int) {
	line := 1
	column := 1
	if index < 0 {
		return line, column
	}
	for i, r := range template {
		if i >= index {
			break
		}
		if r == '\n' {
			line++
			column = 1
			continue
		}
		column++
	}
	return line, column
}

func cloneScope(scope map[string]map[string]any) map[string]map[string]any {
	next := make(map[string]map[string]any, len(scope))
	for key, value := range scope {
		next[key] = value
	}
	return next
}

func validLoopBinding(itemName, collectionName string) bool {
	switch {
	case itemName == "product" && collectionName == "products":
		return true
	case itemName == "post" && collectionName == "posts":
		return true
	case itemName == "plan" && collectionName == "subscription_plans":
		return true
	default:
		return false
	}
}
