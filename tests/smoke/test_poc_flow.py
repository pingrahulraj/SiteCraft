from __future__ import annotations

import time

from .common import api_base, http_json, http_text, live_http_enabled, service_available


def run():
    if not live_http_enabled():
        return True
    if not service_available(f"{api_base().rstrip('/api')}/healthz"):
        return True
    name = f"Smoke Flow Site {int(time.time())}"
    create_payload = {"tenantSlug": "acme-studio", "name": name, "description": "Created from smoke test"}
    status, data = http_json("POST", f"{api_base()}/projects", create_payload)
    if status not in (200, 201):
        return f"site create failed: {status}"
    items = data.get("items") or data.get("projects") or []
    site = next((item for item in items if item.get("name") == name), None)
    if not site:
        return "created site not returned"
    site_id = site["id"]
    save_status, _ = http_json("PUT", f"{api_base()}/sites/{site_id}/workspace/files/content", {"path": "index.html", "content": "<!doctype html><html><body>HELLO_PUBLISHED_SITE</body></html>"})
    if save_status != 200:
        return f"save failed: {save_status}"
    publish_status, publish_data = http_json("POST", f"{api_base()}/sites/{site_id}/publish", {})
    if publish_status != 200 or not publish_data.get("success"):
        return f"publish failed: {publish_status}"
    public_url = publish_data.get("public_url")
    if not public_url:
        return "missing public_url"
    status, body = http_text(public_url)
    if status != 200 or "HELLO_PUBLISHED_SITE" not in body:
        return "published delivery did not return saved html"
    return True
