# ai-evaluator-service

Python service for initial AI-assisted evaluation of math tasks.

Responsibilities:
- estimate initial task difficulty on a `1..10` scale
- suggest tag weights when the author did not specify them
- use a math-specific prompt for Qwen
- fall back to deterministic heuristics when Qwen is unavailable

Run locally:

```bash
python3 ./ai-evaluator-service/cmd/ai-evaluator-service/main.py
```

Main endpoint:

```http
POST /evaluate/task
Content-Type: application/json
```

Request:

```json
{
  "title": "Найдите корни уравнения",
  "latex_body": "\\[x^2 - 5x + 6 = 0\\]",
  "topic_titles": ["Квадратные уравнения"],
  "tags": [
    {"tag_id": "tag_disc", "code": "disc", "name": "Дискриминант", "kind": "skill"},
    {"tag_id": "tag_roots", "code": "roots", "name": "Корни уравнения", "kind": "skill"}
  ],
  "correct_answer": "2,3"
}
```

Response:

```json
{
  "difficulty": 3,
  "tag_weights": [
    {"tag_id": "tag_disc", "weight": 0.55},
    {"tag_id": "tag_roots", "weight": 0.45}
  ],
  "provider": "heuristic",
  "prompt_version": "qwen-math-v1",
  "confidence": 0.63,
  "rationale": "Quadratic equation with explicit coefficients and standard root extraction."
}
```
