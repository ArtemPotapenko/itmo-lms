from __future__ import annotations

import json
import os
import urllib.error
import urllib.request
from typing import Any, Dict, Optional


class QwenClient:
    def __init__(self) -> None:
        self.base_url = os.environ.get("QWEN_BASE_URL", "").rstrip("/")
        self.api_key = os.environ.get("QWEN_API_KEY", "")
        self.model = os.environ.get("QWEN_MODEL", "qwen2.5:14b-instruct")

    def enabled(self) -> bool:
        return bool(self.base_url)

    def evaluate(self, system_prompt: str, user_prompt: str) -> Optional[Dict[str, Any]]:
        if not self.enabled():
            return None

        payload = {
            "model": self.model,
            "temperature": 0.1,
            "messages": [
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt},
            ],
            "response_format": {"type": "json_object"},
        }
        request = urllib.request.Request(
            url=f"{self.base_url}/chat/completions",
            data=json.dumps(payload).encode("utf-8"),
            headers=self._headers(),
            method="POST",
        )
        try:
            with urllib.request.urlopen(request, timeout=30) as response:
                body = json.loads(response.read().decode("utf-8"))
        except (urllib.error.URLError, TimeoutError, json.JSONDecodeError):
            return None

        try:
            content = body["choices"][0]["message"]["content"]
            return json.loads(content)
        except (KeyError, IndexError, TypeError, json.JSONDecodeError):
            return None

    def _headers(self) -> Dict[str, str]:
        headers = {"Content-Type": "application/json"}
        if self.api_key:
            headers["Authorization"] = f"Bearer {self.api_key}"
        return headers
