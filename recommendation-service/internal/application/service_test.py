from __future__ import annotations

import datetime as dt
import unittest

from internal.application.service import RecommendationService
from internal.domain.models import RecommendationRequest, SubjectTagValue
from internal.infrastructure.memory_repository import InMemoryRecommendationRepository


class FakeContent:
    def list_topics(self) -> list[dict]:
        return [
            {"id": "top_quad", "title": "Квадратные уравнения"},
            {"id": "top_lin", "title": "Линейные уравнения"},
        ]

    def list_tags(self) -> list[dict]:
        return [
            {"id": "tag_disc", "name": "Дискриминант", "code": "disc", "kind": "skill"},
            {"id": "tag_roots", "name": "Корни", "code": "roots", "kind": "skill"},
            {"id": "tag_linear", "name": "Линейные", "code": "linear", "kind": "skill"},
        ]

    def list_tasks(self) -> list[dict]:
        return [
            {
                "id": "tsk_1",
                "title": "Найдите корни",
                "latex_body": "\\[x^2 - 5x + 6 = 0\\]",
                "topic_ids": ["top_quad"],
                "difficulty": 4,
                "tags": [{"tag_id": "tag_disc", "weight": 0.7}, {"tag_id": "tag_roots", "weight": 0.3}],
            },
            {
                "id": "tsk_2",
                "title": "Решите линейное уравнение",
                "latex_body": "\\[2x + 1 = 3\\]",
                "topic_ids": ["top_lin"],
                "difficulty": 2,
                "tags": [{"tag_id": "tag_linear", "weight": 1.0}],
            },
            {
                "id": "tsk_3",
                "title": "Определите число корней",
                "latex_body": "\\[x^2 + 4x + 5 = 0\\]",
                "topic_ids": ["top_quad"],
                "difficulty": 5,
                "tags": [{"tag_id": "tag_disc", "weight": 0.5}, {"tag_id": "tag_roots", "weight": 0.5}],
            },
        ]

    def list_theory(self) -> list[dict]:
        return [
            {
                "id": "thr_quad",
                "title": "Дискриминант",
                "latex_body": "Для квадратного уравнения \\[D=b^2-4ac\\].",
                "summary": "Формула дискриминанта",
                "topic_ids": ["top_quad"],
            },
            {
                "id": "thr_lin",
                "title": "Линейные уравнения",
                "latex_body": "Для линейного уравнения переносим слагаемые.",
                "summary": "Базовый алгоритм",
                "topic_ids": ["top_lin"],
            },
        ]


class FakeStatistics:
    def get_profile(self, user_id: str) -> dict:
        return {
            "user_id": user_id,
            "topics": {
                "top_quad": {"topic_id": "top_quad", "rating": 3.5, "weighted_attempts": 4},
                "top_lin": {"topic_id": "top_lin", "rating": 8.0, "weighted_attempts": 3},
            },
            "tags": {
                "tag_disc": {"tag_id": "tag_disc", "mastery": 0.2, "weighted_attempts": 3, "name": "Дискриминант"},
                "tag_roots": {"tag_id": "tag_roots", "mastery": 0.35, "weighted_attempts": 2, "name": "Корни"},
                "tag_linear": {"tag_id": "tag_linear", "mastery": 0.9, "weighted_attempts": 3, "name": "Линейные"},
            },
        }

    def get_course_calibration(self, course_id: str) -> dict:
        return {
            "course_id": course_id,
            "task_calibrations": {
                "tsk_1": {
                    "suggested_difficulty": 4.8,
                    "tag_weights": [{"id": "tag_disc", "weight": 0.8}, {"id": "tag_roots", "weight": 0.2}],
                    "topic_weights": [{"id": "top_quad", "weight": 1.0}],
                }
            },
        }

    def get_attempts(self, user_id: str) -> list[dict]:
        now = dt.datetime.now(dt.timezone.utc)
        return [
            {
                "id": "att_recent_fail",
                "user_id": user_id,
                "content_id": "tsk_3",
                "tag_scores": [{"tag_id": "tag_disc", "weight": 0.5}, {"tag_id": "tag_roots", "weight": 0.5}],
                "is_correct": False,
                "created_at": (now - dt.timedelta(days=2)).isoformat(),
            },
            {
                "id": "att_old_ok",
                "user_id": user_id,
                "content_id": "tsk_1",
                "tag_scores": [{"tag_id": "tag_disc", "weight": 0.7}],
                "is_correct": True,
                "created_at": (now - dt.timedelta(days=40)).isoformat(),
            },
            {
                "id": "att_linear_ok",
                "user_id": user_id,
                "content_id": "tsk_2",
                "tag_scores": [{"tag_id": "tag_linear", "weight": 1.0}],
                "is_correct": True,
                "created_at": (now - dt.timedelta(days=1)).isoformat(),
            },
        ]


class RecommendationServiceTest(unittest.TestCase):
    def test_build_workbook_prefers_tasks_close_to_weak_tags(self) -> None:
        repository = InMemoryRecommendationRepository()
        repository.upsert_subject_profile(
            "math",
            [
                SubjectTagValue("tag_disc", "disc", "Дискриминант", "skill", 1.6, [], ["top_quad"]),
                SubjectTagValue("tag_roots", "roots", "Корни", "skill", 1.2, [], ["top_quad"]),
                SubjectTagValue("tag_linear", "linear", "Линейные", "skill", 0.5, [], ["top_lin"]),
            ],
        )
        service = RecommendationService(FakeContent(), FakeStatistics(), repository)

        result = service.build_workbook(
            RecommendationRequest(user_id="usr_1", subject="math", course_id="crs_1", max_tasks=2)
        )

        self.assertEqual("usr_1", result.user_id)
        self.assertEqual("math", result.subject)
        self.assertEqual(2, len(result.selected_tasks))
        self.assertEqual("top_quad", result.selected_tasks[0].topic_ids[0])
        self.assertEqual("thr_quad", result.selected_theory[0].id)
        self.assertEqual("theory", result.workbook.items[0].kind)
        self.assertEqual("task", result.workbook.items[1].kind)
        self.assertIn("Дискриминант", result.workbook.latex)
        self.assertGreater(result.weak_tags[0].score, result.weak_tags[-1].score)
        self.assertEqual("tag_disc", result.recommendation_vector[0].tag_id)
        self.assertGreater(result.recommendation_vector[0].recent_error_rate, 0.0)
        self.assertEqual(1, len(repository.list_vectors("usr_1", "math", 10)))

    def test_recommendation_vector_differs_from_plain_profile_weakness(self) -> None:
        repository = InMemoryRecommendationRepository()
        repository.upsert_subject_profile(
            "math",
            [
                SubjectTagValue("tag_disc", "disc", "Дискриминант", "skill", 1.0, [], ["top_quad"]),
                SubjectTagValue("tag_roots", "roots", "Корни", "skill", 1.0, [], ["top_quad"]),
                SubjectTagValue("tag_linear", "linear", "Линейные", "skill", 1.0, [], ["top_lin"]),
            ],
        )
        service = RecommendationService(FakeContent(), FakeStatistics(), repository)

        result = service.build_workbook(RecommendationRequest(user_id="usr_1", subject="math", max_tasks=2))

        weak = {item.tag_id: item.score for item in result.weak_tags}
        rec = {item.tag_id: item.score for item in result.recommendation_vector}
        self.assertNotEqual(round(weak["tag_roots"], 4), round(rec["tag_roots"], 4))
        self.assertGreater(rec["tag_roots"], weak["tag_roots"])

    def test_build_workbook_requires_user_id(self) -> None:
        service = RecommendationService(FakeContent(), FakeStatistics(), InMemoryRecommendationRepository())

        with self.assertRaisesRegex(ValueError, "user_id is required"):
            service.build_workbook(RecommendationRequest(user_id=""))

    def test_update_subject_tags_persists_values(self) -> None:
        repository = InMemoryRecommendationRepository()
        service = RecommendationService(FakeContent(), FakeStatistics(), repository)

        items = service.update_subject_tags(
            "physics",
            [{"tag_id": "tag_vectors", "name": "Векторы", "kind": "skill", "prior_weight": 1.4}],
        )

        stored = service.get_subject_tags("physics")
        self.assertEqual(1.4, items[0].prior_weight)
        self.assertEqual("tag_vectors", stored[0].tag_id)
        self.assertEqual("tag_vectors", repository.get_subject_profile("physics")[0].tag_id)


if __name__ == "__main__":
    unittest.main()
