from __future__ import annotations

import datetime as dt
import math
from collections import defaultdict
from typing import Dict, List, Protocol, Sequence, Tuple

from ..domain.models import (
    RecommendationVectorEntry,
    RecommendationRequest,
    RecommendationResponse,
    RecommendedTask,
    RecommendedTheory,
    RecommendedWorkbook,
    StoredTagVector,
    SubjectTagValue,
    TagVectorEntry,
    WorkbookItem,
)


class ContentProvider(Protocol):
    def list_topics(self) -> List[dict]: ...
    def list_tags(self) -> List[dict]: ...
    def list_tasks(self) -> List[dict]: ...
    def list_theory(self) -> List[dict]: ...


class StatisticProvider(Protocol):
    def get_profile(self, user_id: str) -> dict: ...
    def get_attempts(self, user_id: str) -> List[dict]: ...
    def get_course_calibration(self, course_id: str) -> dict: ...


class RecommendationRepository(Protocol):
    def get_subject_profile(self, subject: str) -> List[SubjectTagValue]: ...
    def upsert_subject_profile(self, subject: str, tags: List[SubjectTagValue]) -> None: ...
    def store_vector(self, vector: StoredTagVector) -> None: ...
    def list_vectors(self, user_id: str, subject: str = "", limit: int = 10) -> List[StoredTagVector]: ...


class RecommendationService:
    def __init__(
        self, content: ContentProvider, statistics: StatisticProvider, repository: RecommendationRepository
    ) -> None:
        self._content = content
        self._statistics = statistics
        self._repository = repository

    def build_workbook(self, request: RecommendationRequest) -> RecommendationResponse:
        if not request.user_id:
            raise ValueError("user_id is required")
        if request.max_tasks <= 0:
            raise ValueError("max_tasks must be positive")
        if request.max_theory_items < 0:
            raise ValueError("max_theory_items must be non-negative")

        topics = self._content.list_topics()
        tags = self._content.list_tags()
        tasks = self._content.list_tasks()
        theory = self._content.list_theory()
        profile = self._statistics.get_profile(request.user_id)
        attempts = self._statistics.get_attempts(request.user_id)
        calibration = {}
        if request.course_id:
            calibration = self._statistics.get_course_calibration(request.course_id)
        subject = request.subject.strip() or "math"
        subject_profile = self._repository.get_subject_profile(subject)
        if not subject_profile:
            subject_profile = self._bootstrap_subject_profile(subject, tags)

        topic_titles = {str(item.get("id", "")): str(item.get("title", "")).strip() for item in topics}
        tag_meta = {
            str(item.get("id", "")): {
                "name": str(item.get("name", "")).strip(),
                "code": str(item.get("code", "")).strip(),
                "kind": str(item.get("kind", "")).strip(),
            }
            for item in tags
        }

        weak_tags = self._build_weak_tag_vector(
            profile.get("tags", {}), tag_meta, subject_profile, request.max_tag_vector_size
        )
        recommendation_vector = self._build_recommendation_vector(
            profile.get("tags", {}),
            attempts,
            tag_meta,
            subject_profile,
            request.max_tag_vector_size,
        )
        topic_weakness = self._build_topic_weakness(profile.get("topics", {}))
        weak_tag_map = {item.tag_id: item.score for item in recommendation_vector}
        calibration_map = calibration.get("task_calibrations", {}) if isinstance(calibration, dict) else {}

        ranked = self._rank_tasks(tasks, weak_tag_map, topic_weakness, calibration_map)
        selected = self._select_tasks(ranked, request.max_tasks)
        selected_theory = self._select_theory(theory, selected, calibration_map, request.max_theory_items)
        self._repository.store_vector(
            StoredTagVector(
                user_id=request.user_id,
                subject=subject,
                course_id=request.course_id,
                generated_at=dt.datetime.now(dt.timezone.utc).isoformat(),
                weak_tags=weak_tags,
                recommendation_vector=recommendation_vector,
                topic_weakness=topic_weakness,
            )
        )
        workbook = self._build_workbook_payload(
            request=request,
            topic_titles=topic_titles,
            selected_tasks=selected,
            selected_theory=selected_theory,
        )
        return RecommendationResponse(
            user_id=request.user_id,
            course_id=request.course_id,
            subject=subject,
            generated_at=dt.datetime.now(dt.timezone.utc).isoformat(),
            weak_tags=weak_tags,
            recommendation_vector=recommendation_vector,
            selected_tasks=selected,
            selected_theory=selected_theory,
            workbook=workbook,
        )

    def get_subject_tags(self, subject: str) -> List[SubjectTagValue]:
        if not subject.strip():
            raise ValueError("subject is required")
        return self._repository.get_subject_profile(subject.strip())

    def update_subject_tags(self, subject: str, tags: Sequence[dict]) -> List[SubjectTagValue]:
        if not subject.strip():
            raise ValueError("subject is required")
        values: List[SubjectTagValue] = []
        for item in tags:
            tag_id = str(item.get("tag_id", "")).strip()
            if not tag_id:
                raise ValueError("tag_id is required")
            values.append(
                SubjectTagValue(
                    tag_id=tag_id,
                    code=str(item.get("code", "")).strip(),
                    name=str(item.get("name", "")).strip(),
                    kind=str(item.get("kind", "")).strip(),
                    prior_weight=max(0.0, _to_float(item.get("prior_weight", 1.0)) or 1.0),
                    aliases=[str(value).strip() for value in item.get("aliases", []) if str(value).strip()],
                    related_topics=[
                        str(value).strip() for value in item.get("related_topics", []) if str(value).strip()
                    ],
                )
            )
        self._repository.upsert_subject_profile(subject.strip(), values)
        return values

    def list_user_vectors(self, user_id: str, subject: str = "", limit: int = 10) -> List[StoredTagVector]:
        if not user_id.strip():
            raise ValueError("user_id is required")
        return self._repository.list_vectors(user_id.strip(), subject.strip(), limit=max(1, limit))

    def _build_weak_tag_vector(
        self, profile_tags: dict, tag_meta: dict, subject_profile: Sequence[SubjectTagValue], limit: int
    ) -> List[TagVectorEntry]:
        subject_meta = {item.tag_id: item for item in subject_profile}
        items: List[TagVectorEntry] = []
        for tag_id, raw in profile_tags.items():
            mastery = _to_float(raw.get("mastery", 0.0))
            weighted_attempts = _to_float(raw.get("weighted_attempts", 0.0))
            confidence = min(1.0, weighted_attempts / 3.0)
            prior = max(
                0.1,
                subject_meta.get(tag_id, SubjectTagValue(tag_id, "", "", "", 1.0, [], [])).prior_weight,
            )
            score = round(max(0.0, 1.0 - mastery) * (0.4 + 0.6 * confidence) * prior, 4)
            if score <= 0:
                continue
            meta = tag_meta.get(tag_id, {})
            subject_item = subject_meta.get(tag_id)
            items.append(
                TagVectorEntry(
                    tag_id=tag_id,
                    code=str(raw.get("code") or meta.get("code") or (subject_item.code if subject_item else "") or ""),
                    name=str(raw.get("name") or meta.get("name") or (subject_item.name if subject_item else "") or tag_id),
                    kind=str(raw.get("kind") or meta.get("kind") or (subject_item.kind if subject_item else "") or ""),
                    mastery=round(mastery, 4),
                    weighted_attempts=round(weighted_attempts, 4),
                    score=score,
                )
            )
        items.sort(key=lambda item: (-item.score, item.tag_id))
        return items[:limit]

    def _build_recommendation_vector(
        self,
        profile_tags: dict,
        attempts: Sequence[dict],
        tag_meta: dict,
        subject_profile: Sequence[SubjectTagValue],
        limit: int,
    ) -> List[RecommendationVectorEntry]:
        subject_meta = {item.tag_id: item for item in subject_profile}
        attempt_stats = self._attempt_stats_by_tag(attempts)
        now = dt.datetime.now(dt.timezone.utc)
        items: List[RecommendationVectorEntry] = []
        for tag_id, raw in profile_tags.items():
            meta = tag_meta.get(tag_id, {})
            subject_item = subject_meta.get(tag_id)
            mastery = _to_float(raw.get("mastery", 0.0))
            mastery_gap = max(0.0, 1.0 - mastery)
            prior_weight = max(0.1, (subject_item.prior_weight if subject_item else 1.0))
            tag_attempts = attempt_stats.get(tag_id, {})
            last_attempt_at = _parse_time(str(tag_attempts.get("last_attempt_at", "")))
            days_since_last = 365.0 if last_attempt_at is None else max(0.0, (now - last_attempt_at).total_seconds() / 86400.0)
            recency_factor = min(1.0, days_since_last / 30.0)
            recent_total = _to_float(tag_attempts.get("recent_total", 0.0))
            recent_wrong = _to_float(tag_attempts.get("recent_wrong", 0.0))
            recent_error_rate = 0.5 if recent_total == 0 else min(1.0, recent_wrong / recent_total)
            practice_gap = max(0.0, 1.0 - min(1.0, _to_float(tag_attempts.get("attempt_count", 0.0)) / 5.0))
            recent_success = _to_float(tag_attempts.get("recent_success_rate", mastery))
            trend_penalty = max(0.0, mastery - recent_success)
            score = round(
                prior_weight
                * (
                    0.35 * mastery_gap
                    + 0.25 * recent_error_rate
                    + 0.20 * recency_factor
                    + 0.15 * practice_gap
                    + 0.05 * trend_penalty
                ),
                4,
            )
            if score <= 0:
                continue
            items.append(
                RecommendationVectorEntry(
                    tag_id=tag_id,
                    code=str(raw.get("code") or meta.get("code") or (subject_item.code if subject_item else "") or ""),
                    name=str(raw.get("name") or meta.get("name") or (subject_item.name if subject_item else "") or tag_id),
                    kind=str(raw.get("kind") or meta.get("kind") or (subject_item.kind if subject_item else "") or ""),
                    mastery_gap=round(mastery_gap, 4),
                    recent_error_rate=round(recent_error_rate, 4),
                    recency_factor=round(recency_factor, 4),
                    practice_gap=round(practice_gap, 4),
                    trend_penalty=round(trend_penalty, 4),
                    prior_weight=round(prior_weight, 4),
                    score=score,
                )
            )
        items.sort(key=lambda item: (-item.score, item.tag_id))
        return items[:limit]

    def _attempt_stats_by_tag(self, attempts: Sequence[dict]) -> Dict[str, dict]:
        by_tag: Dict[str, dict] = {}
        cutoff = dt.datetime.now(dt.timezone.utc) - dt.timedelta(days=21)
        for attempt in attempts:
            created_at = _parse_time(str(attempt.get("created_at", "")))
            is_recent = created_at is not None and created_at >= cutoff
            is_correct = bool(attempt.get("is_correct", False))
            for tag in attempt.get("tag_scores", []) or []:
                tag_id = str(tag.get("tag_id", "")).strip()
                if not tag_id:
                    continue
                item = by_tag.setdefault(
                    tag_id,
                    {
                        "attempt_count": 0.0,
                        "last_attempt_at": "",
                        "recent_total": 0.0,
                        "recent_wrong": 0.0,
                        "recent_correct": 0.0,
                    },
                )
                item["attempt_count"] += 1.0
                if created_at is not None:
                    previous = _parse_time(str(item["last_attempt_at"]))
                    if previous is None or created_at > previous:
                        item["last_attempt_at"] = created_at.isoformat()
                if is_recent:
                    item["recent_total"] += 1.0
                    if is_correct:
                        item["recent_correct"] += 1.0
                    else:
                        item["recent_wrong"] += 1.0
        for item in by_tag.values():
            recent_total = _to_float(item.get("recent_total", 0.0))
            if recent_total > 0:
                item["recent_success_rate"] = _to_float(item.get("recent_correct", 0.0)) / recent_total
            else:
                item["recent_success_rate"] = 0.0
        return by_tag

    def _bootstrap_subject_profile(self, subject: str, tags: Sequence[dict]) -> List[SubjectTagValue]:
        profile = [
            SubjectTagValue(
                tag_id=str(item.get("id", "")).strip(),
                code=str(item.get("code", "")).strip(),
                name=str(item.get("name", "")).strip(),
                kind=str(item.get("kind", "")).strip(),
                prior_weight=1.0,
                aliases=[],
                related_topics=[],
            )
            for item in tags
            if str(item.get("id", "")).strip()
        ]
        self._repository.upsert_subject_profile(subject, profile)
        return profile

    def _build_topic_weakness(self, profile_topics: dict) -> Dict[str, float]:
        result: Dict[str, float] = {}
        for topic_id, raw in profile_topics.items():
            rating = _to_float(raw.get("rating", 0.0))
            weighted_attempts = _to_float(raw.get("weighted_attempts", 0.0))
            confidence = min(1.0, weighted_attempts / 3.0)
            result[topic_id] = round(max(0.0, (10.0 - rating) / 10.0) * (0.4 + 0.6 * confidence), 4)
        return result

    def _rank_tasks(
        self,
        tasks: Sequence[dict],
        weak_tag_map: Dict[str, float],
        topic_weakness: Dict[str, float],
        calibration_map: dict,
    ) -> List[RecommendedTask]:
        average_weakness = sum(weak_tag_map.values()) / len(weak_tag_map) if weak_tag_map else 0.45
        target_difficulty = min(8.0, max(2.0, 3.0 + average_weakness * 3.0))
        ranked: List[RecommendedTask] = []
        for raw in tasks:
            task_id = str(raw.get("id", "")).strip()
            if not task_id:
                continue
            task_topics = [str(item) for item in raw.get("topic_ids", []) if str(item).strip()]
            calibration = calibration_map.get(task_id, {})
            task_tags = self._task_tag_weights(raw, calibration)
            topic_weights = self._task_topic_weights(task_topics, calibration)
            tag_similarity = _cosine_similarity(weak_tag_map, task_tags)
            topic_alignment = _weighted_overlap(topic_weakness, topic_weights)
            difficulty = _to_float(calibration.get("suggested_difficulty", raw.get("difficulty", 1) or 1))
            difficulty_score = max(0.0, 1.0 - abs(difficulty - target_difficulty) / 9.0)
            score = round(0.65 * tag_similarity + 0.25 * topic_alignment + 0.10 * difficulty_score, 4)
            ranked.append(
                RecommendedTask(
                    id=task_id,
                    title=str(raw.get("title", "")).strip(),
                    latex_body=str(raw.get("latex_body", "")).strip(),
                    topic_ids=task_topics,
                    tags=[
                        {
                            "tag_id": str(item.get("tag_id") or item.get("id") or ""),
                            "weight": round(_to_float(item.get("weight", 0.0)), 4),
                        }
                        for item in raw.get("tags", [])
                    ],
                    difficulty=round(difficulty, 2),
                    recommendation_score=score,
                    score_breakdown={
                        "tag_similarity": round(tag_similarity, 4),
                        "topic_alignment": round(topic_alignment, 4),
                        "difficulty_score": round(difficulty_score, 4),
                    },
                )
            )
        ranked.sort(key=lambda item: (-item.recommendation_score, item.id))
        return ranked

    def _select_tasks(self, ranked: Sequence[RecommendedTask], max_tasks: int) -> List[RecommendedTask]:
        selected: List[RecommendedTask] = []
        by_primary_topic: Dict[str, int] = defaultdict(int)
        for task in ranked:
            primary_topic = task.topic_ids[0] if task.topic_ids else ""
            if primary_topic and by_primary_topic[primary_topic] >= 2 and len(selected) < max_tasks - 1:
                continue
            selected.append(task)
            if primary_topic:
                by_primary_topic[primary_topic] += 1
            if len(selected) == max_tasks:
                break
        if len(selected) < max_tasks:
            taken = {item.id for item in selected}
            for task in ranked:
                if task.id in taken:
                    continue
                selected.append(task)
                if len(selected) == max_tasks:
                    break
        return selected

    def _select_theory(
        self, theory: Sequence[dict], tasks: Sequence[RecommendedTask], calibration_map: dict, limit: int
    ) -> List[RecommendedTheory]:
        if limit == 0 or not tasks:
            return []
        topic_scores: Dict[str, float] = defaultdict(float)
        for task in tasks:
            calibration = calibration_map.get(task.id, {})
            weighted_topics = self._task_topic_weights(task.topic_ids, calibration)
            task_score = max(task.recommendation_score, 0.1)
            for topic_id, weight in weighted_topics.items():
                topic_scores[topic_id] += task_score * weight
        ranked: List[Tuple[float, dict]] = []
        for item in theory:
            topic_ids = [str(value) for value in item.get("topic_ids", []) if str(value).strip()]
            score = sum(topic_scores.get(topic_id, 0.0) for topic_id in topic_ids)
            if score <= 0:
                continue
            ranked.append((score, item))
        ranked.sort(key=lambda pair: (-pair[0], str(pair[1].get("id", ""))))
        selected: List[RecommendedTheory] = []
        covered_topics: set[str] = set()
        for score, item in ranked:
            topic_ids = [str(value) for value in item.get("topic_ids", []) if str(value).strip()]
            if covered_topics.intersection(topic_ids):
                continue
            selected.append(
                RecommendedTheory(
                    id=str(item.get("id", "")).strip(),
                    title=str(item.get("title", "")).strip(),
                    latex_body=str(item.get("latex_body", "")).strip(),
                    summary=str(item.get("summary", "")).strip(),
                    topic_ids=topic_ids,
                    recommendation_score=round(score, 4),
                )
            )
            covered_topics.update(topic_ids)
            if len(selected) == limit:
                break
        return selected

    def _build_workbook_payload(
        self,
        request: RecommendationRequest,
        topic_titles: Dict[str, str],
        selected_tasks: Sequence[RecommendedTask],
        selected_theory: Sequence[RecommendedTheory],
    ) -> RecommendedWorkbook:
        title = request.title.strip() or self._workbook_title(selected_tasks, topic_titles)
        items: List[WorkbookItem] = []
        used_tasks: set[str] = set()
        order = 1
        for theory in selected_theory:
            items.append(
                WorkbookItem(
                    order=order,
                    kind="theory",
                    content_id=theory.id,
                    title=theory.title,
                    topic_ids=theory.topic_ids,
                )
            )
            order += 1
            for task in selected_tasks:
                if task.id in used_tasks or not set(task.topic_ids).intersection(theory.topic_ids):
                    continue
                items.append(
                    WorkbookItem(
                        order=order,
                        kind="task",
                        content_id=task.id,
                        title=task.title,
                        topic_ids=task.topic_ids,
                    )
                )
                used_tasks.add(task.id)
                order += 1
        for task in selected_tasks:
            if task.id in used_tasks:
                continue
            items.append(
                WorkbookItem(
                    order=order,
                    kind="task",
                    content_id=task.id,
                    title=task.title,
                    topic_ids=task.topic_ids,
                )
            )
            order += 1
        latex = self._build_workbook_latex(title, selected_theory, selected_tasks, items)
        rationale = self._build_rationale(selected_tasks, selected_theory, topic_titles)
        return RecommendedWorkbook(title=title, items=items, latex=latex, rationale=rationale)

    def _workbook_title(self, tasks: Sequence[RecommendedTask], topic_titles: Dict[str, str]) -> str:
        topic_order: Dict[str, float] = defaultdict(float)
        for task in tasks:
            for topic_id in task.topic_ids:
                topic_order[topic_id] += task.recommendation_score
        if not topic_order:
            return "Рекомендованная рабочая тетрадь"
        top_topic = max(topic_order.items(), key=lambda item: item[1])[0]
        return f"Рекомендованная рабочая тетрадь: {topic_titles.get(top_topic, top_topic)}"

    def _build_rationale(
        self, tasks: Sequence[RecommendedTask], theory: Sequence[RecommendedTheory], topic_titles: Dict[str, str]
    ) -> str:
        topic_scores: Dict[str, float] = defaultdict(float)
        for task in tasks:
            for topic_id in task.topic_ids:
                topic_scores[topic_id] += task.recommendation_score
        dominant_topics = sorted(topic_scores.items(), key=lambda item: (-item[1], item[0]))[:3]
        topic_labels = [topic_titles.get(topic_id, topic_id) for topic_id, _ in dominant_topics]
        parts = []
        if topic_labels:
            parts.append("Фокус на темах: " + ", ".join(topic_labels) + ".")
        if theory:
            parts.append("Теория поставлена перед связанными задачами.")
        parts.append("Задачи отсортированы по близости к вектору слабых тегов пользователя.")
        return " ".join(parts)

    def _build_workbook_latex(
        self,
        title: str,
        selected_theory: Sequence[RecommendedTheory],
        selected_tasks: Sequence[RecommendedTask],
        items: Sequence[WorkbookItem],
    ) -> str:
        theory_by_id = {item.id: item for item in selected_theory}
        task_by_id = {item.id: item for item in selected_tasks}
        lines = [
            "\\documentclass[12pt,a4paper]{article}",
            "\\usepackage[utf8]{inputenc}",
            "\\usepackage[T2A]{fontenc}",
            "\\usepackage[russian]{babel}",
            "\\usepackage{amsmath,amssymb,geometry}",
            "\\geometry{margin=2cm}",
            "\\begin{document}",
            f"\\section*{{{_escape_latex(title)}}}",
        ]
        task_number = 0
        for item in sorted(items, key=lambda value: value.order):
            if item.kind == "theory":
                section = theory_by_id.get(item.content_id)
                if section is None:
                    continue
                lines.append(f"\\subsection*{{Теория. {_escape_latex(section.title)}}}")
                lines.append(section.latex_body)
                lines.append("\\vspace{8mm}")
                continue
            section = task_by_id.get(item.content_id)
            if section is None:
                continue
            task_number += 1
            lines.append(f"\\subsection*{{Задача {task_number}. {_escape_latex(section.title)}}}")
            lines.append(section.latex_body)
            lines.append("\\vspace{8mm}")
        lines.append("\\end{document}")
        return "\n".join(lines) + "\n"

    def _task_tag_weights(self, task: dict, calibration: dict) -> Dict[str, float]:
        weights = calibration.get("tag_weights", [])
        if weights:
            vector = {
                str(item.get("id", "")): _to_float(item.get("weight", 0.0))
                for item in weights
                if str(item.get("id", "")).strip()
            }
            return _normalize_positive_weights(vector)
        vector = {
            str(item.get("tag_id", "")): _to_float(item.get("weight", 0.0) or 0.0)
            for item in task.get("tags", [])
            if str(item.get("tag_id", "")).strip()
        }
        if vector and any(weight > 0 for weight in vector.values()):
            return _normalize_positive_weights(vector)
        if vector:
            uniform = 1.0 / len(vector)
            return {key: round(uniform, 4) for key in vector}
        return {}

    def _task_topic_weights(self, topic_ids: Sequence[str], calibration: dict) -> Dict[str, float]:
        weights = calibration.get("topic_weights", [])
        if weights:
            vector = {
                str(item.get("id", "")): _to_float(item.get("weight", 0.0))
                for item in weights
                if str(item.get("id", "")).strip()
            }
            return _normalize_positive_weights(vector)
        unique_topics = [topic_id for topic_id in topic_ids if topic_id]
        if not unique_topics:
            return {}
        weight = 1.0 / len(unique_topics)
        return {topic_id: round(weight, 4) for topic_id in unique_topics}


def _normalize_positive_weights(values: Dict[str, float]) -> Dict[str, float]:
    positive = {key: value for key, value in values.items() if value > 0}
    total = sum(positive.values())
    if total <= 0:
        return {}
    return {key: round(value / total, 4) for key, value in positive.items()}


def _weighted_overlap(reference: Dict[str, float], target: Dict[str, float]) -> float:
    if not reference or not target:
        return 0.0
    return min(1.0, max(0.0, sum(reference.get(key, 0.0) * weight for key, weight in target.items())))


def _cosine_similarity(left: Dict[str, float], right: Dict[str, float]) -> float:
    if not left or not right:
        return 0.0
    dot = sum(left.get(key, 0.0) * right.get(key, 0.0) for key in right)
    left_norm = math.sqrt(sum(value * value for value in left.values()))
    right_norm = math.sqrt(sum(value * value for value in right.values()))
    if left_norm == 0 or right_norm == 0:
        return 0.0
    return min(1.0, max(0.0, dot / (left_norm * right_norm)))


def _to_float(value: object) -> float:
    try:
        return float(value)
    except (TypeError, ValueError):
        return 0.0


def _parse_time(value: str) -> dt.datetime | None:
    value = value.strip()
    if not value:
        return None
    try:
        parsed = dt.datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError:
        return None
    if parsed.tzinfo is None:
        return parsed.replace(tzinfo=dt.timezone.utc)
    return parsed.astimezone(dt.timezone.utc)


def _escape_latex(value: str) -> str:
    return (
        value.replace("&", "\\&")
        .replace("%", "\\%")
        .replace("$", "\\$")
        .replace("#", "\\#")
        .replace("_", "\\_")
        .replace("{", "\\{")
        .replace("}", "\\}")
    )
