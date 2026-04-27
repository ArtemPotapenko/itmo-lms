from __future__ import annotations

import json
import urllib.parse
import urllib.request
from typing import List


class ContentClient:
    def __init__(self, base_url: str, timeout: float = 5.0) -> None:
        self._base_url = base_url.rstrip("/")
        self._timeout = timeout
        self._opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))

    def list_topics(self) -> List[dict]:
        return self._get_json("/topics")

    def list_tags(self) -> List[dict]:
        return self._get_json("/tags")

    def list_tasks(self) -> List[dict]:
        return self._get_json("/tasks")

    def list_theory(self) -> List[dict]:
        return self._get_json("/theory")

    def _get_json(self, path: str) -> List[dict]:
        url = urllib.parse.urljoin(self._base_url + "/", path.lstrip("/"))
        request = urllib.request.Request(url=url, method="GET")
        with self._opener.open(request, timeout=self._timeout) as response:
            payload = json.loads(response.read().decode("utf-8"))
        if not isinstance(payload, list):
            raise ValueError(f"unexpected content response for {path}")
        return payload
