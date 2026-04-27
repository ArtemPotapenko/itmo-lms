from __future__ import annotations

from typing import List

from pymongo import ASCENDING, DESCENDING, MongoClient

from ..domain.models import StoredTagVector, SubjectTagValue, TagVectorEntry


class MongoRecommendationRepository:
    def __init__(self, mongo_url: str, database: str) -> None:
        self._client = MongoClient(mongo_url)
        self._db = self._client[database]
        self._profiles = self._db["subject_tag_profiles"]
        self._vectors = self._db["tag_vectors"]
        self._profiles.create_index([("subject", ASCENDING)], unique=True)
        self._vectors.create_index([("user_id", ASCENDING), ("subject", ASCENDING), ("generated_at", DESCENDING)])

    def get_subject_profile(self, subject: str) -> List[SubjectTagValue]:
        doc = self._profiles.find_one({"subject": subject})
        if not doc:
            return []
        return [_decode_subject_tag_value(item) for item in doc.get("tags", [])]

    def upsert_subject_profile(self, subject: str, tags: List[SubjectTagValue]) -> None:
        self._profiles.update_one(
            {"subject": subject},
            {"$set": {"subject": subject, "tags": [_encode_subject_tag_value(item) for item in tags]}},
            upsert=True,
        )

    def store_vector(self, vector: StoredTagVector) -> None:
        self._vectors.insert_one(
            {
                "user_id": vector.user_id,
                "subject": vector.subject,
                "course_id": vector.course_id,
                "generated_at": vector.generated_at,
                "weak_tags": [_encode_tag_vector_entry(item) for item in vector.weak_tags],
                "topic_weakness": vector.topic_weakness,
            }
        )

    def list_vectors(self, user_id: str, subject: str = "", limit: int = 10) -> List[StoredTagVector]:
        query = {"user_id": user_id}
        if subject:
            query["subject"] = subject
        cursor = self._vectors.find(query).sort("generated_at", DESCENDING).limit(limit)
        return [_decode_stored_vector(item) for item in cursor]


def _encode_subject_tag_value(item: SubjectTagValue) -> dict:
    return {
        "tag_id": item.tag_id,
        "code": item.code,
        "name": item.name,
        "kind": item.kind,
        "prior_weight": item.prior_weight,
        "aliases": item.aliases,
        "related_topics": item.related_topics,
    }


def _decode_subject_tag_value(item: dict) -> SubjectTagValue:
    return SubjectTagValue(
        tag_id=str(item.get("tag_id", "")).strip(),
        code=str(item.get("code", "")).strip(),
        name=str(item.get("name", "")).strip(),
        kind=str(item.get("kind", "")).strip(),
        prior_weight=float(item.get("prior_weight", 1.0) or 1.0),
        aliases=[str(value).strip() for value in item.get("aliases", []) if str(value).strip()],
        related_topics=[str(value).strip() for value in item.get("related_topics", []) if str(value).strip()],
    )


def _encode_tag_vector_entry(item: TagVectorEntry) -> dict:
    return {
        "tag_id": item.tag_id,
        "code": item.code,
        "name": item.name,
        "kind": item.kind,
        "mastery": item.mastery,
        "weighted_attempts": item.weighted_attempts,
        "score": item.score,
    }


def _decode_stored_vector(item: dict) -> StoredTagVector:
    return StoredTagVector(
        user_id=str(item.get("user_id", "")).strip(),
        subject=str(item.get("subject", "")).strip(),
        course_id=str(item.get("course_id", "")).strip(),
        generated_at=str(item.get("generated_at", "")).strip(),
        weak_tags=[
            TagVectorEntry(
                tag_id=str(tag.get("tag_id", "")).strip(),
                code=str(tag.get("code", "")).strip(),
                name=str(tag.get("name", "")).strip(),
                kind=str(tag.get("kind", "")).strip(),
                mastery=float(tag.get("mastery", 0.0) or 0.0),
                weighted_attempts=float(tag.get("weighted_attempts", 0.0) or 0.0),
                score=float(tag.get("score", 0.0) or 0.0),
            )
            for tag in item.get("weak_tags", [])
        ],
        topic_weakness={str(key): float(value or 0.0) for key, value in item.get("topic_weakness", {}).items()},
    )
