from __future__ import annotations

import importlib
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
TESTS_ROOT = ROOT / "tests"
if str(TESTS_ROOT) not in sys.path:
    sys.path.insert(0, str(TESTS_ROOT))


TESTS = [
    "smoke.test_health",
    "smoke.test_database",
    "smoke.test_storage",
    "smoke.test_ops_files",
    "smoke.test_auth_security",
    "smoke.test_tenant_isolation",
    "smoke.test_poc_flow",
    "smoke.test_template_rendering",
    "smoke.test_demo_site",
    "smoke.test_demo_sites",
    "smoke.test_workspace_files",
    "smoke.test_concurrency",
]


def main() -> int:
    failed = []
    for module_name in TESTS:
        module = importlib.import_module(module_name)
        result = module.run()
        if result is not True:
            failed.append(f"{module_name}: {result}")
    if failed:
        for item in failed:
            print(item)
        return 1
    print("all smoke tests passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
