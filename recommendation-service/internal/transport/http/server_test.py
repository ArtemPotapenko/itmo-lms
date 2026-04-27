from __future__ import annotations

import json
import socket
import threading
import unittest
import urllib.request

from internal.application.service import RecommendationService
from internal.transport.http.server import build_server
from internal.application.service_test import FakeContent, FakeStatistics


class HTTPServerTest(unittest.TestCase):
    def test_recommendation_endpoint_returns_workbook(self) -> None:
        service = RecommendationService(FakeContent(), FakeStatistics())
        port = _free_port()
        server = build_server(f"127.0.0.1:{port}", service)
        self.addCleanup(server.shutdown)
        self.addCleanup(server.server_close)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()

        payload = json.dumps({"user_id": "usr_1", "course_id": "crs_1", "max_tasks": 2}).encode("utf-8")
        request = urllib.request.Request(
            url=f"http://127.0.0.1:{port}/recommendations/workbooks",
            data=payload,
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
        with opener.open(request, timeout=5) as response:
            body = json.loads(response.read().decode("utf-8"))

        self.assertEqual("usr_1", body["user_id"])
        self.assertEqual("Рекомендованная рабочая тетрадь: Квадратные уравнения", body["workbook"]["title"])
        self.assertGreaterEqual(len(body["selected_tasks"]), 1)


def _free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


if __name__ == "__main__":
    unittest.main()
