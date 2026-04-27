from __future__ import annotations

from dataclasses import asdict
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
        if self.path.startswith("/recommendations/subjects/") and self.path.endswith("/tags"):
            subject = self.path[len("/recommendations/subjects/") : -len("/tags")].strip("/")
            self._handle_get_subject_tags(subject)
            return
        if self.path.startswith("/recommendations/users/") and "/vectors" in self.path:
            self._handle_get_vectors()
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

    def do_PUT(self) -> None:
        if not (self.path.startswith("/recommendations/subjects/") and self.path.endswith("/tags")):
            self._write_json(HTTPStatus.NOT_FOUND, {"error": "not found"})
            return
        subject = self.path[len("/recommendations/subjects/") : -len("/tags")].strip("/")
        payload = self._read_json()
        if payload is None:
            self._write_json(HTTPStatus.BAD_REQUEST, {"error": "invalid json"})
            return
        try:
            result = self.service.update_subject_tags(subject, payload.get("tags", []))
        except ValueError as err:
            self._write_json(HTTPStatus.BAD_REQUEST, {"error": str(err)})
            return
        self._write_json(
            HTTPStatus.OK,
            {"subject": subject, "tags": [asdict(item) for item in result]},
        )

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

    def _handle_get_subject_tags(self, subject: str) -> None:
        try:
            items = self.service.get_subject_tags(subject)
        except ValueError as err:
            self._write_json(HTTPStatus.BAD_REQUEST, {"error": str(err)})
            return
        self._write_json(HTTPStatus.OK, {"subject": subject, "tags": [asdict(item) for item in items]})

    def _handle_get_vectors(self) -> None:
        path, _, query = self.path.partition("?")
        prefix = "/recommendations/users/"
        suffix = "/vectors"
        user_id = path[len(prefix) : -len(suffix)].strip("/")
        params = {}
        if query:
            for pair in query.split("&"):
                if "=" not in pair:
                    continue
                key, value = pair.split("=", 1)
                params[key] = value
        try:
            items = self.service.list_user_vectors(
                user_id=user_id,
                subject=params.get("subject", ""),
                limit=int(params.get("limit", "10") or "10"),
            )
        except ValueError as err:
            self._write_json(HTTPStatus.BAD_REQUEST, {"error": str(err)})
            return
        self._write_json(
            HTTPStatus.OK,
            {
                "user_id": user_id,
                "vectors": [
                    {
                        "user_id": item.user_id,
                        "subject": item.subject,
                        "course_id": item.course_id,
                        "generated_at": item.generated_at,
                        "weak_tags": [asdict(entry) for entry in item.weak_tags],
                        "topic_weakness": item.topic_weakness,
                    }
                    for item in items
                ],
            },
        )


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
