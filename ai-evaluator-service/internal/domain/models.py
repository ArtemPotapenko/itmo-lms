from __future__ import annotations

from dataclasses import dataclass, field
from typing import List


@dataclass(slots=True)
class TagCandidate:
    tag_id: str
    code: str = ""
    name: str = ""
    kind: str = ""
    weight: float = 0.0


@dataclass(slots=True)
class EvaluateTaskRequest:
    title: str
    latex_body: str
    topic_titles: List[str] = field(default_factory=list)
    tags: List[TagCandidate] = field(default_factory=list)
    correct_answer: str = ""


@dataclass(slots=True)
class WeightedTag:
    tag_id: str
    weight: float


@dataclass(slots=True)
class EvaluateTaskResponse:
    difficulty: int
    tag_weights: List[WeightedTag]
    provider: str
    prompt_version: str
    confidence: float
    rationale: str
