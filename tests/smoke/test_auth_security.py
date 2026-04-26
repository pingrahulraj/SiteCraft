from __future__ import annotations

from .common import api_base, auth_headers, delivery_base, http_json, http_text, live_http_enabled, login_token, seed_state, service_available


def run():
    if not live_http_enabled():
        return True
    if not service_available(f"{api_base().rstrip('/api')}/healthz"):
        return True

    unauthorized_status, unauthorized_body = http_json("GET", f"{api_base()}/projects")
    if unauthorized_status != 401 or ((unauthorized_body.get("error") or {}).get("code")) != "unauthorized":
        return f"expected 401 for unauthenticated projects list, got {unauthorized_status}"

    invalid_status, invalid_body = http_json("GET", f"{api_base()}/projects", headers={"Authorization": "Bearer invalid-token"})
    if invalid_status != 401 or ((invalid_body.get("error") or {}).get("code")) != "unauthorized":
        return f"expected 401 for invalid token, got {invalid_status}"

    token = login_token()
    headers = auth_headers(token)
    me_status, me_body = http_json("GET", f"{api_base()}/me", headers=headers)
    if me_status != 200 or not ((me_body.get("item") or {}).get("id")):
        return "authenticated /me failed"

    site = seed_state()
    site_id = site["siteId"]
    site_status, _ = http_json("GET", f"{api_base()}/projects/{site_id}", headers=headers)
    if site_status != 200:
        return f"authorized site access failed: {site_status}"

    publish_status, publish_body = http_json("POST", f"{api_base()}/sites/{site_id}/publish", {}, headers=headers)
    if publish_status != 200:
        return f"authorized publish failed: {publish_status}"
    public_url = publish_body.get("public_url") or f"{delivery_base()}/sites/{site_id}/client-landing-page"

    public_status, public_body = http_text(public_url)
    if public_status != 200 or "Client Advisory Studio" not in public_body:
        return "public delivery failed without token"

    workspace_status, _ = http_text(f"{delivery_base()}/sites/{site_id}/client-landing-page/../workspace/index.html")
    if workspace_status == 200:
        return "workspace path should not be publicly served"

    traversal_status, _ = http_json(
        "GET",
        f"{api_base()}/sites/{site_id}/workspace/files/content?path=../secret.txt",
        headers=headers,
    )
    if traversal_status == 200:
        return "path traversal should be denied"

    return True
