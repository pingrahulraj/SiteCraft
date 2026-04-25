from __future__ import annotations

import json
import os
from pathlib import Path
from urllib import error, request

import psycopg


ROOT = Path(__file__).resolve().parents[2]
ENV_FILE = ROOT / ".env"


def load_env() -> dict[str, str]:
    values = dict(os.environ)
    if ENV_FILE.exists():
        for line in ENV_FILE.read_text(encoding="utf-8").splitlines():
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, value = line.split("=", 1)
            values[key.strip()] = value.strip()
    return values


def database_url() -> str:
    env = load_env()
    return env["DATABASE_URL"]


def db_connect():
    return psycopg.connect(database_url())


def api_base() -> str:
    env = load_env()
    return f"http://localhost:{env.get('API_PORT', '8080')}/api"


def frontend_base() -> str:
    env = load_env()
    return f"http://localhost:{env.get('FRONTEND_PORT', '5137')}"


def delivery_base() -> str:
    env = load_env()
    return env.get("DELIVERY_BASE_URL") or f"http://localhost:{env.get('DELIVERY_PORT', '8083')}"


def http_json(method: str, url: str, payload: dict | None = None) -> tuple[int, dict]:
    headers = {"Content-Type": "application/json"}
    data = json.dumps(payload).encode("utf-8") if payload is not None else None
    req = request.Request(url, method=method, data=data, headers=headers)
    try:
        with request.urlopen(req, timeout=5) as response:
            body = response.read().decode("utf-8")
            return response.getcode(), json.loads(body) if body else {}
    except error.HTTPError as exc:
        body = exc.read().decode("utf-8")
        return exc.code, json.loads(body) if body else {}
    except Exception:
        return 0, {}


def http_text(url: str) -> tuple[int, str]:
    try:
        with request.urlopen(url, timeout=5) as response:
            return response.getcode(), response.read().decode("utf-8")
    except error.HTTPError as exc:
        return exc.code, exc.read().decode("utf-8")
    except Exception:
        return 0, ""


def service_available(url: str) -> bool:
    status, _ = http_text(url)
    return status == 200


def seed_state() -> dict:
    return json.loads((ROOT / "storage" / "seed_state.json").read_text(encoding="utf-8"))


def live_http_enabled() -> bool:
    return load_env().get("RUN_LIVE_HTTP_TESTS", "").lower() in {"1", "true", "yes"}
