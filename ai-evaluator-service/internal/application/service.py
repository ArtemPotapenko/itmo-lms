from __future__ import annotations

import math
import re
from typing import Dict, List

from ..domain.models import EvaluateTaskRequest, EvaluateTaskResponse, WeightedTag
from ..infrastructure.qwen_client import QwenClient


PROMPT_VERSION = "qwen-math-v1"


class EvaluationService:
    def __init__(self, qwen_client: QwenClient) -> None:
        self._qwen = qwen_client

    def evaluate_task(self, request: EvaluateTaskRequest) -> EvaluateTaskResponse:
        qwen_response = self._evaluate_with_qwen(request)
        if qwen_response is not None:
            return qwen_response
        return self._heuristic_evaluate(request)

    def _evaluate_with_qwen(self, request: EvaluateTaskRequest) -> EvaluateTaskResponse | None:
        payload = self._qwen.evaluate(self._system_prompt(), self._user_prompt(request))
        if not payload:
            return None

        try:
            difficulty = int(payload["difficulty"])
            difficulty = max(1, min(10, difficulty))
            weights = payload.get("tag_weights", [])
            normalized = self._normalize_weights(request, weights)
            rationale = str(payload.get("rationale", ""))
            confidence = float(payload.get("confidence", 0.75))
        except (KeyError, TypeError, ValueError):
            return None

        return EvaluateTaskResponse(
            difficulty=difficulty,
            tag_weights=normalized,
            provider="qwen",
            prompt_version=PROMPT_VERSION,
            confidence=max(0.0, min(1.0, confidence)),
            rationale=rationale or "Qwen math evaluation.",
        )

    def _heuristic_evaluate(self, request: EvaluateTaskRequest) -> EvaluateTaskResponse:
        text = f"{request.title}\n{request.latex_body}".lower()
        score = 2.0

        if "sqrt" in text or "\\sqrt" in text:
            score += 1.5
        if "\\frac" in text or "/" in request.correct_answer:
            score += 0.8
        if any(token in text for token in ("\\sin", "\\cos", "\\tan", "\\log", "\\ln")):
            score += 1.5
        if any(token in text for token in ("\\int", "\\sum", "\\prod", "\\lim")):
            score += 2.5
        if any(token in text for token in ("matrix", "\\begin{pmatrix}", "\\det")):
            score += 2.0
        if len(re.findall(r"[a-zA-Z]", request.latex_body)) > 30:
            score += 0.5
        if len(re.findall(r"=", request.latex_body)) > 1:
            score += 0.5
        if "докажите" in text or "prove" in text:
            score += 2.0
        if "квадрат" in text or "x^2" in text:
            score += 0.5

        difficulty = max(1, min(10, int(math.ceil(score))))
        weights = self._heuristic_tag_weights(request)
        rationale = self._heuristic_rationale(text, difficulty)
        return EvaluateTaskResponse(
            difficulty=difficulty,
            tag_weights=weights,
            provider="heuristic",
            prompt_version=PROMPT_VERSION,
            confidence=0.62,
            rationale=rationale,
        )

    def _heuristic_tag_weights(self, request: EvaluateTaskRequest) -> List[WeightedTag]:
        if not request.tags:
            return []

        text = f"{request.title}\n{request.latex_body}".lower()
        raw_scores: Dict[str, float] = {}
        for tag in request.tags:
            score = 1.0
            keyspace = " ".join(value.lower() for value in (tag.code, tag.name, tag.kind) if value)
            if "дискр" in keyspace or "disc" in keyspace:
                if "x^2" in text or "d =" in text or "b^2" in text:
                    score += 1.4
            if "кор" in keyspace or "roots" in keyspace:
                if "= 0" in text:
                    score += 1.2
            if "арифм" in keyspace or "algebra" in keyspace:
                if "\\frac" in text or "+" in text or "-" in text:
                    score += 0.6
            if "триг" in keyspace or "trig" in keyspace:
                if any(token in text for token in ("\\sin", "\\cos", "\\tan")):
                    score += 1.8
            if "интег" in keyspace or "integral" in keyspace:
                if "\\int" in text:
                    score += 2.0
            raw_scores[tag.tag_id] = score
        total = sum(raw_scores.values()) or float(len(request.tags))
        return [WeightedTag(tag_id=tag.tag_id, weight=round(raw_scores[tag.tag_id] / total, 4)) for tag in request.tags]

    def _heuristic_rationale(self, text: str, difficulty: int) -> str:
        if "\\int" in text:
            return "Integral form and symbolic manipulation increase the estimated difficulty."
        if any(token in text for token in ("\\sin", "\\cos", "\\tan")):
            return "Trigonometric expressions usually require more transformation steps."
        if "x^2" in text:
            return f"Standard school algebra pattern with moderate symbolic work; estimated difficulty {difficulty}/10."
        return f"Math task estimated from symbolic density and operation complexity; difficulty {difficulty}/10."

    def _normalize_weights(self, request: EvaluateTaskRequest, weights: List[dict]) -> List[WeightedTag]:
        if not request.tags:
            return []

        by_id = {str(item.get("tag_id", "")): float(item.get("weight", 0.0)) for item in weights}
        normalized = []
        total = 0.0
        for tag in request.tags:
            weight = by_id.get(tag.tag_id, 0.0)
            if weight <= 0:
                weight = 1.0
            normalized.append(WeightedTag(tag_id=tag.tag_id, weight=weight))
            total += weight
        for item in normalized:
            item.weight = round(item.weight / total, 4)
        return normalized

    def _system_prompt(self) -> str:
        return (
            "You are an expert evaluator for Russian mathematics LMS content. "
            "Estimate initial task difficulty on a 1..10 scale and distribute tag weights across provided tags. "
            "Focus on school and early university mathematics. "
            "Consider symbolic complexity, number of transformations, hidden prerequisite knowledge, "
            "proof vs direct computation, and the likely student effort. "
            "Return strict JSON with keys: difficulty, confidence, rationale, tag_weights. "
            "tag_weights must be an array of objects {tag_id, weight}. "
            "Weights must sum to 1.0. Do not invent tags not present in the request."
        )

    def _user_prompt(self, request: EvaluateTaskRequest) -> str:
        tags = [
            {
                "tag_id": tag.tag_id,
                "code": tag.code,
                "name": tag.name,
                "kind": tag.kind,
            }
            for tag in request.tags
        ]
        payload = {
            "title": request.title,
            "latex_body": request.latex_body,
            "topic_titles": request.topic_titles,
            "correct_answer": request.correct_answer,
            "tags": tags,
        }
        return str(payload)
