from __future__ import annotations

import os
import pathlib
import sys
import urllib.parse
import urllib.request


ROOT = pathlib.Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT))

from internal.application.service import EvaluationService
from internal.infrastructure.qwen_client import QwenClient
from internal.transport.http.server import serve


def register_consul() -> None:
    consul_url = os.environ.get("CONSUL_URL", "").strip()
    if not consul_url:
        return
    service_name = os.environ.get("SERVICE_NAME", "ai-evaluator-service")
    service_id = os.environ.get("SERVICE_ID", f"{service_name}-1")
    service_host = os.environ.get("SERVICE_HOST", "127.0.0.1")
    port = int(os.environ.get("PORT", "8090"))
    payload = (
        "{"
        f"\"ID\":\"{service_id}\","
        f"\"Name\":\"{service_name}\","
        f"\"Address\":\"{service_host}\","
        f"\"Port\":{port}"
        "}"
    ).encode("utf-8")
    request = urllib.request.Request(
        url=urllib.parse.urljoin(consul_url.rstrip("/") + "/", "v1/agent/service/register"),
        data=payload,
        headers={"Content-Type": "application/json"},
        method="PUT",
    )
    try:
        with urllib.request.urlopen(request, timeout=5):
            return
    except Exception:
        return


def main() -> None:
    register_consul()
    addr = os.environ.get("ADDR", ":8090")
    service = EvaluationService(QwenClient())
    serve(addr, service)


if __name__ == "__main__":
    main()
