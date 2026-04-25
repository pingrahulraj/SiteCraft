from __future__ import annotations

import json
from pathlib import Path

from .common import ROOT


def run():
    state = json.loads((ROOT / "storage" / "seed_state.json").read_text(encoding="utf-8"))
    workspace = ROOT / "storage" / "sites" / state["tenantId"] / state["siteId"] / "workspace" / "index.html"
    logs = ROOT / "storage" / "logs"
    if not workspace.exists():
        return "workspace index.html missing"
    if not logs.exists():
        return "storage/logs missing"
    return True
