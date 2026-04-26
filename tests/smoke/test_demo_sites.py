from __future__ import annotations

from urllib import request

from .common import api_base, auth_headers, delivery_base, http_json, http_text, live_http_enabled, login_token, service_available


SITES = [
    {
        "name": "Client Advisory Studio",
        "slug": "client-advisory-studio",
        "files": {"index.html", "about.html", "services.html", "case-studies.html", "pricing.html", "contact.html", "styles.css", "script.js"},
        "pages": ["index.html", "services.html", "pricing.html"],
        "expect": "Client Advisory Studio",
    },
    {
        "name": "Maison Vela",
        "slug": "maison-vela",
        "files": {"index.html", "about.html", "shop.html", "collections.html", "contact.html", "styles.css", "script.js"},
        "pages": ["index.html", "shop.html", "collections.html"],
        "expect": "Maison Vela",
    },
    {
        "name": "PawNest Club",
        "slug": "pawnest-club",
        "files": {"index.html", "about.html", "subscriptions.html", "shop.html", "faq.html", "contact.html", "styles.css", "script.js"},
        "pages": ["index.html", "subscriptions.html", "faq.html"],
        "expect": "PawNest Club",
    },
    {
        "name": "PrintForge Studio",
        "slug": "printforge-studio",
        "files": {"index.html", "about.html", "products.html", "customize.html", "pricing.html", "contact.html", "styles.css", "script.js"},
        "pages": ["index.html", "products.html", "customize.html"],
        "expect": "PrintForge Studio",
    },
    {
        "name": "PixelShelf",
        "slug": "pixelshelf",
        "files": {"index.html", "about.html", "products.html", "downloads.html", "pricing.html", "contact.html", "styles.css", "script.js"},
        "pages": ["index.html", "products.html", "downloads.html"],
        "expect": "PixelShelf",
    },
]


def fetch_content_type(url: str) -> tuple[int, str]:
    req = request.Request(url)
    try:
        with request.urlopen(req, timeout=5) as response:
            return response.getcode(), response.headers.get("Content-Type", "")
    except Exception:
        return 0, ""


def run():
    if not live_http_enabled():
        return True
    if not service_available(f"{api_base().rstrip('/api')}/healthz"):
        return True

    token = login_token()
    headers = auth_headers(token)

    status, payload = http_json("GET", f"{api_base()}/projects", headers=headers)
    if status != 200:
        return f"project list failed: {status}"
    items = payload.get("items") or []
    by_slug = {item.get("slug"): item for item in items}

    for site in SITES:
        item = by_slug.get(site["slug"])
        if not item:
            return f"missing site {site['slug']}"
        site_id = item.get("id")
        if not site_id:
            return f"missing site id for {site['slug']}"

        file_status, file_data = http_json("GET", f"{api_base()}/sites/{site_id}/workspace/files", headers=headers)
        if file_status != 200:
            return f"workspace list failed for {site['slug']}: {file_status}"
        actual_files = {entry.get("path") for entry in (file_data.get("items") or []) if entry.get("kind") != "folder"}
        for path in site["files"]:
            if path not in actual_files:
                return f"missing workspace file {path} for {site['slug']}"

        publish_status, publish_data = http_json("POST", f"{api_base()}/sites/{site_id}/publish", {}, headers=headers)
        if publish_status != 200 or not publish_data.get("success"):
            return f"publish failed for {site['slug']}: {publish_status}"
        base_url = publish_data.get("public_url") or f"{delivery_base()}/sites/{site_id}/{site['slug']}"

        for page in site["pages"]:
            page_url = base_url if page == "index.html" else f"{base_url}/{page}"
            page_status, body = http_text(page_url)
            if page_status != 200:
                return f"page failed for {site['slug']}: {page_url} -> {page_status}"
            if site["expect"] not in body:
                return f"missing expected brand copy in {page_url}"

        css_status, css_type = fetch_content_type(f"{base_url}/styles.css")
        if css_status != 200 or "text/css" not in css_type:
            return f"bad css content type for {site['slug']}: {css_status} {css_type}"

        js_status, js_type = fetch_content_type(f"{base_url}/script.js")
        if js_status != 200 or "javascript" not in js_type:
            return f"bad js content type for {site['slug']}: {js_status} {js_type}"

    return True
