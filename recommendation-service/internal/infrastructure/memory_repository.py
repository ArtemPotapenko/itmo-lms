from __future__ import annotations

from collections import defaultdict
from typing import Dict, List

from ..domain.models import StoredTagVector, SubjectTagValue


class InMemoryRecommendationRepository:
    def __init__(self) -> None:
        self._profiles: Dict[str, List[SubjectTagValue]] = {}
        self._vectors: Dict[str, List[StoredTagVector]] = defaultdict(list)

    def get_subject_profile(self, subject: str) -> List[SubjectTagValue]:
        return list(self._profiles.get(subject, []))

    def upsert_subject_profile(self, subject: str, tags: List[SubjectTagValue]) -> None:
        self._profiles[subject] = list(tags)

    def store_vector(self, vector: StoredTagVector) -> None:
        self._vectors[vector.user_id].insert(0, vector)

    def list_vectors(self, user_id: str, subject: str = "", limit: int = 10) -> List[StoredTagVector]:
        items = self._vectors.get(user_id, [])
        if subject:
            items = [item for item in items if item.subject == subject]
        return list(items[:limit])
