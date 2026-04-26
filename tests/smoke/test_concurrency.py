from __future__ import annotations

import threading
import time

from .common import api_base, auth_headers, delivery_base, http_json, http_text, live_http_enabled, login_token, seed_state, service_available


def run():
    if not live_http_enabled():
        return True
    if not service_available(f"{api_base().rstrip('/api')}/healthz"):
        return True

    headers = auth_headers(login_token())
    state = seed_state()
    site_id = state["siteId"]
    public_root = f"{delivery_base()}/sites/{site_id}/client-landing-page"

    errors: list[str] = []

    def saver(index: int):
        status, _ = http_json(
            "PUT",
            f"{api_base()}/sites/{site_id}/workspace/files/content",
            {"path": "content/concurrency.txt", "content": f"save-{index}-{time.time_ns()}"},
            headers=headers,
        )
        if status != 200:
            errors.append(f"save failed: {status}")

    def reader():
        for _ in range(5):
            status, _ = http_text(public_root)
            if status not in (200, 304):
                errors.append(f"public read failed: {status}")

    save_threads = [threading.Thread(target=saver, args=(i,)) for i in range(4)]
    read_threads = [threading.Thread(target=reader) for _ in range(4)]

    for thread in save_threads + read_threads:
        thread.start()

    publish_statuses: list[int] = []

    def publisher():
        status, _ = http_json("POST", f"{api_base()}/sites/{site_id}/publish", {}, headers=headers)
        publish_statuses.append(status)

    publish_threads = [threading.Thread(target=publisher) for _ in range(2)]
    for thread in publish_threads:
        thread.start()

    for thread in save_threads + read_threads + publish_threads:
        thread.join()

    if errors:
        return errors[0]
    if not publish_statuses or 200 not in publish_statuses:
        return f"publish did not succeed: {publish_statuses}"
    if not any(status == 409 for status in publish_statuses) and len(publish_statuses) > 1:
        return f"expected one concurrent publish rejection: {publish_statuses}"

    status, body = http_text(public_root)
    if status != 200 or "Client Advisory Studio" not in body:
        return "final public site did not load"

    css_status, css_body = http_text(f"{public_root}/styles.css")
    if css_status != 200 or ".site-header" not in css_body:
        return "published css missing after concurrent publish"

    return True
