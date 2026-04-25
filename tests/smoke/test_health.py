from __future__ import annotations

from .common import api_base, delivery_base, frontend_base, http_text, live_http_enabled


def run():
    if not live_http_enabled():
        return True
    for url in [frontend_base(), f"{api_base().rstrip('/api')}/healthz", f"{delivery_base()}/healthz"]:
        status, _ = http_text(url)
        if status not in (0, 200):
            return f"health check failed for {url}: {status}"
    return True
