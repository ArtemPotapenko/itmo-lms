from __future__ import annotations

import unittest

from internal.application.service import RecommendationService
from internal.domain.models import RecommendationRequest


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


class RecommendationServiceTest(unittest.TestCase):
    def test_build_workbook_prefers_tasks_close_to_weak_tags(self) -> None:
        service = RecommendationService(FakeContent(), FakeStatistics())

        result = service.build_workbook(RecommendationRequest(user_id="usr_1", course_id="crs_1", max_tasks=2))

        self.assertEqual("usr_1", result.user_id)
        self.assertEqual(2, len(result.selected_tasks))
        self.assertEqual("top_quad", result.selected_tasks[0].topic_ids[0])
        self.assertEqual("thr_quad", result.selected_theory[0].id)
        self.assertEqual("theory", result.workbook.items[0].kind)
        self.assertEqual("task", result.workbook.items[1].kind)
        self.assertIn("Дискриминант", result.workbook.latex)
        self.assertGreater(result.weak_tags[0].score, result.weak_tags[-1].score)

    def test_build_workbook_requires_user_id(self) -> None:
        service = RecommendationService(FakeContent(), FakeStatistics())

        with self.assertRaisesRegex(ValueError, "user_id is required"):
            service.build_workbook(RecommendationRequest(user_id=""))


if __name__ == "__main__":
    unittest.main()
