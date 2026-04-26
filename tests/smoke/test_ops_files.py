from __future__ import annotations

from pathlib import Path

from .common import ROOT


def run():
    required = [
        ROOT / "deploy" / "env" / ".env.staging.example",
        ROOT / "deploy" / "env" / ".env.production.example",
        ROOT / "deploy" / "nginx" / "websitebuilder.conf",
        ROOT / "deploy" / "systemd" / "websitebuilder-api.service",
        ROOT / "deploy" / "systemd" / "websitebuilder-delivery.service",
        ROOT / "deploy" / "systemd" / "websitebuilder-worker.service",
        ROOT / "deploy" / "scripts" / "backup_db.sh",
        ROOT / "deploy" / "scripts" / "restore_db.sh",
        ROOT / "deploy" / "scripts" / "backup_storage.sh",
        ROOT / "deploy" / "scripts" / "restore_storage.sh",
        ROOT / "deploy" / "scripts" / "smoke_test.sh",
    ]
    for path in required:
        if not path.exists():
            return f"missing deploy asset: {path}"
    return True
