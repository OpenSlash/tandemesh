from __future__ import annotations

import json
import os
import sys
import urllib.request


def main() -> int:
    base = os.environ.get("ORCA_BASE_URL", "http://127.0.0.1:8080")
    with urllib.request.urlopen(base + "/api/v1/runs") as resp:
        data = json.load(resp)
    items = data.get("items") or []
    if not items:
        print("no runs")
        return 0
    for run in items:
        print(f"{run['id']:<30} {run['status']:<10} {run.get('repo', ''):<20} {run['work'].get('url', '')}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
