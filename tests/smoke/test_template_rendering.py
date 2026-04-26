from __future__ import annotations

from .common import api_base, auth_headers, delivery_base, http_json, http_text, live_http_enabled, login_token, seed_state, service_available


SHOP_TEMPLATE = """<!doctype html><html><body>{% for product in products %}<h3>{{ product.name }}</h3>{% endfor %}</body></html>"""
BLOGS_TEMPLATE = """<!doctype html><html><body>{% for post in posts %}<h3>{{ post.title }}</h3>{% endfor %}</body></html>"""
SUBSCRIPTIONS_TEMPLATE = """<!doctype html><html><body>{% for plan in subscription_plans %}<h3>{{ plan.name }}</h3>{% endfor %}</body></html>"""


def run():
    if not live_http_enabled():
        return True
    if not service_available(f"{api_base().rstrip('/api')}/healthz"):
        return True

    state = seed_state()
    site_id = state.get("siteId")
    if not site_id:
        return "missing seeded site id"

    token = login_token()
    headers = auth_headers(token)

    originals = {}
    for path in ("shop.html", "blogs.html", "subscriptions.html"):
        status, payload = http_json("GET", f"{api_base()}/sites/{site_id}/workspace/files/content?path={path}", headers=headers)
        if status != 200:
            return f"read failed for {path}: {status}"
        originals[path] = ((payload.get("item") or {}).get("content") or "")

    saves = {
        "shop.html": SHOP_TEMPLATE,
        "blogs.html": BLOGS_TEMPLATE,
        "subscriptions.html": SUBSCRIPTIONS_TEMPLATE,
    }
    for path, content in saves.items():
        status, _ = http_json("PUT", f"{api_base()}/sites/{site_id}/workspace/files/content", {"path": path, "content": content}, headers=headers)
        if status != 200:
            return f"save failed for {path}: {status}"

    preview_status, preview_html = http_text(f"{api_base()}/sites/{site_id}/preview?path=shop.html", headers=headers)
    if preview_status != 200:
        return f"preview failed: {preview_status}"
    if "{% for" in preview_html or "{{" in preview_html:
        return "template syntax leaked into preview"
    if "Starter kit" not in preview_html:
        return "preview missing rendered product content"

    publish_status, publish_data = http_json("POST", f"{api_base()}/sites/{site_id}/publish", {}, headers=headers)
    if publish_status != 200 or not publish_data.get("success"):
        return f"publish failed: {publish_status}"

    base_url = publish_data.get("public_url") or f"{delivery_base()}/sites/{site_id}/client-landing-page"
    checks = {
        f"{base_url}/shop.html": ["Starter kit", "Growth dashboard package"],
        f"{base_url}/blogs.html": ["Launching the new site", "How consulting teams scale delivery"],
        f"{base_url}/subscriptions.html": ["Essentials box", "Growth review"],
    }

    for url, expected in checks.items():
        status, body = http_text(url)
        if status != 200:
            return f"rendered page failed: {url} -> {status}"
        if "{% for" in body or "{{" in body:
            return f"template syntax leaked into {url}"
        for value in expected:
            if value not in body:
                return f"missing rendered content {value!r} in {url}"

    invalid_status, invalid_data = http_json(
        "PUT",
        f"{api_base()}/sites/{site_id}/workspace/files/content",
        {"path": "shop.html", "content": "<!doctype html><html><body>{{ product.fake }}</body></html>"},
        headers=headers,
    )
    if invalid_status != 200:
        return f"save invalid template failed: {invalid_status}"

    republish_status, republish_data = http_json("POST", f"{api_base()}/sites/{site_id}/publish", {}, headers=headers)
    if republish_status != 400 or republish_data.get("success") is not False:
        return f"expected publish validation failure, got {republish_status}"

    status, body = http_text(f"{base_url}/shop.html")
    if status != 200 or "Starter kit" not in body or "{{ product.fake }}" in body:
        return "live release changed after failed publish"

    for path, content in originals.items():
        restore_status, _ = http_json(
            "PUT",
            f"{api_base()}/sites/{site_id}/workspace/files/content",
            {"path": path, "content": content},
            headers=headers,
        )
        if restore_status != 200:
            return f"restore failed for {path}: {restore_status}"

    return True
