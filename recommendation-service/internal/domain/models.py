from __future__ import annotations

from dataclasses import asdict, dataclass, field
from typing import Any, Dict, List


@dataclass(slots=True)
class RecommendationRequest:
    user_id: str
    subject: str = "math"
    course_id: str = ""
    max_tasks: int = 5
    max_theory_items: int = 2
    max_tag_vector_size: int = 8
    title: str = ""


@dataclass(slots=True)
class TagVectorEntry:
    tag_id: str
    code: str
    name: str
    kind: str
    mastery: float
    weighted_attempts: float
    score: float


@dataclass(slots=True)
class SubjectTagValue:
    tag_id: str
    code: str
    name: str
    kind: str
    prior_weight: float
    aliases: List[str]
    related_topics: List[str]


@dataclass(slots=True)
class StoredTagVector:
    user_id: str
    subject: str
    course_id: str
    generated_at: str
    weak_tags: List[TagVectorEntry]
    topic_weakness: Dict[str, float]


@dataclass(slots=True)
class RecommendedTask:
    id: str
    title: str
    latex_body: str
    topic_ids: List[str]
    tags: List[Dict[str, float | str]]
    difficulty: float
    recommendation_score: float
    score_breakdown: Dict[str, float]


@dataclass(slots=True)
class RecommendedTheory:
    id: str
    title: str
    latex_body: str
    summary: str
    topic_ids: List[str]
    recommendation_score: float


@dataclass(slots=True)
class WorkbookItem:
    order: int
    kind: str
    content_id: str
    title: str
    topic_ids: List[str] = field(default_factory=list)


@dataclass(slots=True)
class RecommendedWorkbook:
    title: str
    items: List[WorkbookItem]
    latex: str
    rationale: str


@dataclass(slots=True)
class RecommendationResponse:
    user_id: str
    course_id: str
    subject: str
    generated_at: str
    weak_tags: List[TagVectorEntry]
    selected_tasks: List[RecommendedTask]
    selected_theory: List[RecommendedTheory]
    workbook: RecommendedWorkbook

    def to_dict(self) -> Dict[str, Any]:
        return asdict(self)
