from __future__ import annotations

import json
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any, Dict

from ...application.service import RecommendationService
from ...domain.models import RecommendationRequest


class Handler(BaseHTTPRequestHandler):
    service: RecommendationService

    def do_GET(self) -> None:
        if self.path == "/healthz":
            self._write_json(HTTPStatus.OK, {"status": "ok"})
            return
        self._write_json(HTTPStatus.NOT_FOUND, {"error": "not found"})

    def do_POST(self) -> None:
        if self.path != "/recommendations/workbooks":
            self._write_json(HTTPStatus.NOT_FOUND, {"error": "not found"})
            return
        payload = self._read_json()
        if payload is None:
            self._write_json(HTTPStatus.BAD_REQUEST, {"error": "invalid json"})
            return
        try:
            request = RecommendationRequest(
                user_id=str(payload.get("user_id", "")).strip(),
                course_id=str(payload.get("course_id", "")).strip(),
                max_tasks=int(payload.get("max_tasks", 5) or 5),
                max_theory_items=int(payload.get("max_theory_items", 2) or 2),
                max_tag_vector_size=int(payload.get("max_tag_vector_size", 8) or 8),
                title=str(payload.get("title", "")).strip(),
            )
            result = self.service.build_workbook(request)
        except ValueError as err:
            self._write_json(HTTPStatus.BAD_REQUEST, {"error": str(err)})
            return
        self._write_json(HTTPStatus.OK, result.to_dict())

    def log_message(self, format: str, *args: Any) -> None:
        return

    def _read_json(self) -> Dict[str, Any] | None:
        try:
            length = int(self.headers.get("Content-Length", "0"))
            raw = self.rfile.read(length)
            return json.loads(raw.decode("utf-8"))
        except (ValueError, json.JSONDecodeError):
            return None

    def _write_json(self, status: HTTPStatus, payload: Dict[str, Any]) -> None:
        raw = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)


def serve(addr: str, service: RecommendationService) -> None:
    server = build_server(addr, service)
    server.serve_forever()


def build_server(addr: str, service: RecommendationService) -> ThreadingHTTPServer:
    host, port = _split_addr(addr)
    Handler.service = service
    return ThreadingHTTPServer((host, port), Handler)


def _split_addr(value: str) -> tuple[str, int]:
    value = value.strip()
    if value.startswith(":"):
        return "0.0.0.0", int(value[1:])
    host, port = value.rsplit(":", 1)
    return host, int(port)
