from __future__ import annotations

import json
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any, Dict, List

from ...application.service import EvaluationService
from ...domain.models import EvaluateTaskRequest, TagCandidate


class Handler(BaseHTTPRequestHandler):
    service: EvaluationService

    def do_GET(self) -> None:
        if self.path == "/healthz":
            self._write_json(HTTPStatus.OK, {"status": "ok"})
            return
        self._write_json(HTTPStatus.NOT_FOUND, {"error": "not found"})

    def do_POST(self) -> None:
        if self.path != "/evaluate/task":
            self._write_json(HTTPStatus.NOT_FOUND, {"error": "not found"})
            return
        payload = self._read_json()
        if payload is None:
            self._write_json(HTTPStatus.BAD_REQUEST, {"error": "invalid json"})
            return
        try:
            request = self._decode_request(payload)
            result = self.service.evaluate_task(request)
        except ValueError as err:
            self._write_json(HTTPStatus.BAD_REQUEST, {"error": str(err)})
            return
        self._write_json(
            HTTPStatus.OK,
            {
                "difficulty": result.difficulty,
                "tag_weights": [{"tag_id": item.tag_id, "weight": item.weight} for item in result.tag_weights],
                "provider": result.provider,
                "prompt_version": result.prompt_version,
                "confidence": result.confidence,
                "rationale": result.rationale,
            },
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

    def _decode_request(self, payload: Dict[str, Any]) -> EvaluateTaskRequest:
        title = str(payload.get("title", "")).strip()
        latex_body = str(payload.get("latex_body", "")).strip()
        if not title or not latex_body:
            raise ValueError("title and latex_body are required")
        tags_payload = payload.get("tags", [])
        tags: List[TagCandidate] = []
        for item in tags_payload:
            tags.append(
                TagCandidate(
                    tag_id=str(item.get("tag_id", "")).strip(),
                    code=str(item.get("code", "")).strip(),
                    name=str(item.get("name", "")).strip(),
                    kind=str(item.get("kind", "")).strip(),
                    weight=float(item.get("weight", 0.0) or 0.0),
                )
            )
        return EvaluateTaskRequest(
            title=title,
            latex_body=latex_body,
            topic_titles=[str(item) for item in payload.get("topic_titles", [])],
            tags=tags,
            correct_answer=str(payload.get("correct_answer", "")).strip(),
        )

    def _write_json(self, status: HTTPStatus, payload: Dict[str, Any]) -> None:
        raw = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)


def serve(addr: str, service: EvaluationService) -> None:
    host, port = _split_addr(addr)
    Handler.service = service
    server = ThreadingHTTPServer((host, port), Handler)
    server.serve_forever()


def _split_addr(value: str) -> tuple[str, int]:
    value = value.strip()
    if value.startswith(":"):
        return "0.0.0.0", int(value[1:])
    host, port = value.rsplit(":", 1)
    return host, int(port)
