from __future__ import annotations

from .common import api_base, http_json, live_http_enabled, seed_state, service_available


def run():
    if not live_http_enabled():
        return True
    if not service_available(f"{api_base().rstrip('/api')}/healthz"):
        return True
    site_id = seed_state()["siteId"]
    create_status, _ = http_json("POST", f"{api_base()}/sites/{site_id}/workspace/files", {"path": "content/smoke.css", "content": "body { color: red; }", "fileType": "css"})
    if create_status not in (200, 201):
        return f"create file failed: {create_status}"
    list_status, list_data = http_json("GET", f"{api_base()}/sites/{site_id}/workspace/files")
    if list_status != 200 or not any(item.get("path") == "content/smoke.css" for item in list_data.get("items", [])):
        return "created file not listed"
    read_status, read_data = http_json("GET", f"{api_base()}/sites/{site_id}/workspace/files/content?path=content/smoke.css")
    if read_status != 200 or "color: red" not in read_data.get("item", {}).get("content", ""):
        return "file read failed"
    save_status, _ = http_json("PUT", f"{api_base()}/sites/{site_id}/workspace/files/content", {"path": "content/smoke.css", "content": "body { color: blue; }"})
    if save_status != 200:
        return f"file save failed: {save_status}"
    rename_status, _ = http_json("PUT", f"{api_base()}/sites/{site_id}/workspace/files/rename", {"path": "content/smoke.css", "newPath": "content/smoke-renamed.css"})
    if rename_status != 200:
        return f"rename failed: {rename_status}"
    duplicate_status, _ = http_json("POST", f"{api_base()}/sites/{site_id}/workspace/files/duplicate", {"path": "content/smoke-renamed.css"})
    if duplicate_status != 200:
        return f"duplicate failed: {duplicate_status}"
    delete_status, _ = http_json("DELETE", f"{api_base()}/sites/{site_id}/workspace/files", {"path": "content/smoke-renamed-copy.css"})
    if delete_status != 200:
        return f"delete failed: {delete_status}"
    protected_status, _ = http_json("DELETE", f"{api_base()}/sites/{site_id}/workspace/files", {"path": "index.html"})
    if protected_status == 200:
        return "index.html delete should fail"
    return True
