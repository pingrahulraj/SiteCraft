from __future__ import annotations

from urllib import request

from .common import api_base, auth_headers, delivery_base, http_json, http_text, live_http_enabled, login_token, seed_state, service_available


EXPECTED_FILES = [
    "index.html",
    "about.html",
    "services.html",
    "case-studies.html",
    "pricing.html",
    "shop.html",
    "subscriptions.html",
    "blogs.html",
    "contact.html",
    "styles.css",
    "script.js",
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

    state = seed_state()
    site_id = state.get("siteId")
    tenant_id = state.get("tenantId")
    if not site_id or not tenant_id:
        return "missing seeded site ids"

    token = login_token()
    headers = auth_headers(token)

    file_status, file_data = http_json("GET", f"{api_base()}/sites/{site_id}/workspace/files", headers=headers)
    if file_status != 200:
        return f"workspace list failed: {file_status}"
    file_names = {item.get("path") for item in (file_data.get("items") or [])}
    for name in EXPECTED_FILES:
        if name not in file_names:
            return f"missing workspace file {name}"

    publish_status, publish_data = http_json("POST", f"{api_base()}/sites/{site_id}/publish", {}, headers=headers)
    if publish_status != 200 or not publish_data.get("success"):
        return f"publish failed: {publish_status}"

    base_url = publish_data.get("public_url") or f"{delivery_base()}/sites/{site_id}/client-landing-page"
    compatibility_base = f"{delivery_base()}/sites/{site_id}"
    page_checks = {
        f"{base_url}/shop.html": ["Starter kit", "Messaging architecture sprint", "Automation workflow blueprint"],
        f"{base_url}/blogs.html": ["Launching the new site", "Positioning a consulting firm for the next stage"],
        f"{base_url}/subscriptions.html": ["Essentials box", "Annual operating partner"],
    }
    for url, expected in page_checks.items():
        status, body = http_text(url)
        if status != 200:
            return f"page fetch failed: {url} -> {status}"
        for value in expected:
            if value not in body:
                return f"missing rendered content {value!r} in {url}"
        if "{{" in body or "{%" in body:
            return f"raw template syntax leaked into {url}"

    compatibility_checks = {
        f"{base_url}": "Client Advisory Studio",
        f"{base_url}/index.html": "Client Advisory Studio",
        f"{compatibility_base}": "Client Advisory Studio",
        f"{compatibility_base}/index.html": "Client Advisory Studio",
    }
    for url, expected in compatibility_checks.items():
        status, body = http_text(url)
        if status != 200:
            return f"compatibility fetch failed: {url} -> {status}"
        if expected not in body:
            return f"missing rendered site name in {url}"
        if "{{ site.name }}" in body or "{%" in body:
            return f"raw template syntax leaked into {url}"

    css_status, css_type = fetch_content_type(f"{base_url}/styles.css")
    if css_status != 200 or "text/css" not in css_type:
        return f"bad css content type: {css_status} {css_type}"

    js_status, js_type = fetch_content_type(f"{base_url}/script.js")
    if js_status != 200 or "javascript" not in js_type:
        return f"bad js content type: {js_status} {js_type}"

    return True
