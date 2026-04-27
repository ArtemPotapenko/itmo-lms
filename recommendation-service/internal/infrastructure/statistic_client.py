from __future__ import annotations

import json
import urllib.parse
import urllib.request
from typing import Dict


class StatisticClient:
    def __init__(self, base_url: str, timeout: float = 5.0) -> None:
        self._base_url = base_url.rstrip("/")
        self._timeout = timeout
        self._opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))

    def get_profile(self, user_id: str) -> Dict[str, object]:
        return self._get_json(f"/internal/users/{user_id}/knowledge-profile")

    def get_course_calibration(self, course_id: str) -> Dict[str, object]:
        return self._get_json(f"/internal/courses/{course_id}/calibration")

    def _get_json(self, path: str) -> Dict[str, object]:
        url = urllib.parse.urljoin(self._base_url + "/", path.lstrip("/"))
        request = urllib.request.Request(url=url, method="GET")
        with self._opener.open(request, timeout=self._timeout) as response:
            payload = json.loads(response.read().decode("utf-8"))
        if not isinstance(payload, dict):
            raise ValueError(f"unexpected statistic response for {path}")
        return payload
