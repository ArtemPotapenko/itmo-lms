# ITMO LMS Backend MVP

Монорепозиторий бэкенда LMS для дипломного MVP.

Архитектурные принципы:

- внешний периметр: HTTP через `gateway`
- внутреннее синхронное взаимодействие: gRPC
- внутреннее асинхронное взаимодействие: Kafka
- хранение данных: PostgreSQL
- service discovery: Consul

Рекомендательный сервис пока не реализован. Прохождение линейное, но модель контента и статистики уже подготовлена под будущую адаптивность.

## Структура репозитория

Каждый top-level сервис имеет одинаковую структуру:

- `cmd` — точка входа
- `internal/domain` — сущности и контракты
- `internal/application` — use cases
- `internal/infrastructure` — Postgres, gRPC clients, Kafka
- `internal/transport/http` — внешний HTTP API
- `internal/transport/grpc` — внутренний gRPC API
- `grpc` — `.proto`
- `gen` — сгенерированный код

Общие пакеты:

- `pkg/platform` — HTTP/gRPC runtime, auth helpers, env
- `pkg/postgres` — подключение и миграции
- `pkg/kafka` — publisher/consumer
- `pkg/events` — форматы событий

## Сервисы

- `auth-service` — пользователи, роли, логин, JWT
- `content-service` — темы, тэги, задачи, теория, шаблоны работ
- `course-service` — курсы, участники, назначения, сдачи, проверки
- `document-service` — сборка LaTeX-документов, хранение job-ов
- `statistic-service` — попытки, агрегаты по темам и тэгам
- `gateway` — внешний HTTP reverse proxy на nginx

## Синхронные и асинхронные взаимодействия

### HTTP

Используется только для внешнего API через gateway:

- `http://localhost:8080/auth/...`
- `http://localhost:8080/content/...`
- `http://localhost:8080/courses...`
- `http://localhost:8080/documents...`
- `http://localhost:8080/statistics...`

### gRPC

Используется для внутренних запросов между сервисами:

- `content-service -> document-service`
- `statistic-service -> content-service`
- остальные сервисы уже публикуют свои gRPC контракты для расширения

### Kafka

Используется там, где действительно выгодна асинхронность:

1. `content-service -> Kafka -> statistic-service`
   После проверки задачи публикуется событие `attempt.evaluated`, а статистика считается в фоне.

Kafka topics:

- `attempt-events`

## Запуск

### Локально один сервис

```bash
go run ./content-service/cmd/content-service
```

### Полный стек

```bash
docker compose -f compose/docker-compose.yml up --build
```

Поднимаются:

- PostgreSQL
- Redpanda
- Consul
- все Go-сервисы
- nginx gateway

## Порты

Внутри docker-compose:

- `auth-service`: HTTP `8081`, gRPC `9081`
- `content-service`: HTTP `8082`, gRPC `9082`
- `course-service`: HTTP `8083`, gRPC `9083`
- `document-service`: HTTP `8084`, gRPC `9084`
- `statistic-service`: HTTP `8085`, gRPC `9085`
- `gateway`: `8080`
- `consul`: `8500`

## Внешние HTTP endpoint-ы

Ниже перечислены именно внешние endpoint-ы через `gateway`.

### Auth

#### `POST /auth/register`

Регистрация пользователя.

Request:

```json
{
  "phone": "79990000000",
  "email": "student@example.com",
  "first_name": "Ivan",
  "last_name": "Petrov",
  "nick": "ipetrov",
  "password": "secret",
  "roles": ["student"]
}
```

#### `POST /auth/login`

Request:

```json
{
  "phone": "79990000000",
  "password": "secret"
}
```

Response:

```json
{
  "access_token": "jwt",
  "token_type": "Bearer",
  "user": {}
}
```

#### `GET /auth/me`

Требует `Authorization: Bearer <token>`.

#### `GET /auth/users`

Требует роль `teacher` или `admin`.

### Content

#### `POST /content/topics`

```json
{
  "parent_id": "",
  "title": "Квадратные уравнения",
  "order": 1,
  "status": "published"
}
```

#### `GET /content/topics`

#### `POST /content/tags`

```json
{
  "code": "discriminant",
  "name": "Дискриминант",
  "description": "Навык вычисления дискриминанта",
  "kind": "skill",
  "status": "active"
}
```

#### `GET /content/tags`

#### `POST /content/tasks`

```json
{
  "title": "Решить квадратное уравнение",
  "latex_body": "Решите $x^2-5x+6=0$.",
  "topic_ids": ["top_xxx"],
  "tags": [
    {
      "tag_id": "tag_xxx",
      "weight": 0.7
    },
    {
      "tag_id": "tag_yyy",
      "weight": 0.3
    }
  ],
  "difficulty": 1,
  "correct_answer": "2,3",
  "status": "published",
  "author_id": "usr_teacher"
}
```

#### `GET /content/tasks?topic_id=<topic_id>`

#### `GET /content/tasks/{id}`

#### `POST /content/tasks/{id}/check`

Если передан `user_id`, сервис публикует событие в Kafka, и `statistic-service` обновляет статистику асинхронно.

```json
{
  "user_id": "usr_student",
  "answer": "2,3",
  "source": "practice"
}
```

Response:

```json
{
  "content_id": "tsk_xxx",
  "topic_ids": ["top_xxx"],
  "tags": [
    {
      "tag_id": "tag_xxx",
      "code": "discriminant",
      "name": "Дискриминант",
      "kind": "skill",
      "weight": 0.7
    }
  ],
  "is_correct": true
}
```

#### `POST /content/theory`

```json
{
  "title": "Теория по квадратным уравнениям",
  "latex_body": "Для квадратного уравнения $ax^2+bx+c=0$ дискриминант равен $D=b^2-4ac$.",
  "summary": "Краткое описание",
  "topic_ids": ["top_xxx"],
  "status": "published"
}
```

Теория хранится в LaTeX, так же как и задачи, чтобы ее можно было включать в рабочую тетрадь без дополнительной конвертации.

#### `GET /content/theory?topic_id=<topic_id>`

#### `POST /content/work-templates`

```json
{
  "title": "Рабочая тетрадь по квадратным уравнениям",
  "items": [
    {
      "order": 1,
      "kind": "theory",
      "content_id": "thr_a"
    },
    {
      "order": 2,
      "kind": "task",
      "content_id": "tsk_a"
    },
    {
      "order": 3,
      "kind": "task",
      "content_id": "tsk_b"
    }
  ],
  "status": "published",
  "created_by": "usr_teacher"
}
```

#### `GET /content/work-templates`

#### `GET /content/work-templates/{id}`

#### `GET /content/work-templates/{id}/latex`

Возвращает объединенный LaTeX-исходник рабочей тетради, где в одном документе идут и теоретические блоки, и задачи.

#### `POST /content/work-templates/{id}/check`

Единая отправка ответов по всей рабочей тетради.

Если передан `user_id`, по каждой задаче публикуется событие в Kafka, и `statistic-service` обновляет статистику асинхронно.

```json
{
  "user_id": "usr_student",
  "source": "workbook",
  "answers": [
    {
      "task_id": "tsk_a",
      "answer": "2,3"
    },
    {
      "task_id": "tsk_b",
      "answer": "-1,4"
    }
  ]
}
```

Response:

```json
{
  "work_id": "wrk_xxx",
  "user_id": "usr_student",
  "checked_at": "2026-04-19T18:00:00Z",
  "total_tasks": 2,
  "correct_tasks": 1,
  "results": [
    {
      "task_id": "tsk_a",
      "title": "Решить квадратное уравнение",
      "topic_ids": ["top_xxx"],
      "tags": [
        {
          "tag_id": "tag_xxx",
          "code": "discriminant",
          "name": "Дискриминант",
          "kind": "skill",
          "weight": 0.7
        }
      ],
      "answer": "2,3",
      "is_correct": true
    }
  ]
}
```

#### `POST /content/work-templates/{id}/documents`

Запускает сборку документа через `document-service`.

Response:

```json
{
  "job_id": "doc_xxx"
}
```

#### `GET /content/learning-path?topic_id=<topic_id>`

Возвращает линейный путь по задачам, отсортированный по сложности и времени создания.

### Courses

#### `POST /courses`

```json
{
  "title": "Математика 1 семестр",
  "owner_id": "usr_teacher",
  "status": "active"
}
```

#### `GET /courses`

#### `POST /courses/{id}/members`

```json
{
  "user_id": "usr_student",
  "role": "student"
}
```

#### `GET /courses/{id}/members`

#### `POST /courses/{id}/assignments`

```json
{
  "title": "Домашняя работа 1",
  "work_id": "wrk_xxx",
  "task_ids": [],
  "due_at": "2026-05-01T20:00:00Z",
  "assigned_by": "usr_teacher",
  "status": "published"
}
```

#### `GET /courses/{id}/assignments`

#### `POST /assignments/{id}/submissions`

```json
{
  "user_id": "usr_student",
  "answers": [
    {
      "content_id": "tsk_xxx",
      "answer": "2,3"
    }
  ]
}
```

#### `GET /assignments/{id}/submissions`

#### `POST /submissions/{id}/review`

```json
{
  "reviewer_id": "usr_teacher",
  "score": 10,
  "comment": "Хорошо"
}
```

### Documents

#### `POST /documents/compile`

Синхронно создает документ и возвращает готовый job.

```json
{
  "title": "Подборка задач",
  "format": "tex",
  "tasks": [
    {
      "id": "tsk_xxx",
      "title": "Задача 1",
      "latex_body": "Решите $x^2-5x+6=0$."
    }
  ]
}
```

#### `GET /documents/{id}`

Возвращает job со статусом:

- `completed`
- `failed`

#### `GET /documents/{id}/download`

Доступно только после `completed`.

### Statistics

#### `POST /statistics/attempts`

Резервный синхронный путь. Основной поток обновления статистики идет через Kafka.

```json
{
  "user_id": "usr_student",
  "content_id": "tsk_xxx",
  "answer": "2,3",
  "is_correct": true,
  "source": "practice"
}
```

#### `GET /statistics/users/{id}/attempts`

#### `GET /statistics/users/{id}/knowledge-profile`

Response содержит два агрегата:

- `topics`
- `tags`

## Как считается статистика

### Topic

`topic` нужен для структуры LMS:

- дерево курса
- навигация
- группировка контента
- прогресс по разделам

Формула:

- `attempts += 1`
- `correct += 1`, если ответ верный
- `accuracy = correct / attempts`

### Tag

`tag` нужен для аналитики и будущих рекомендаций.

Каждая задача может иметь несколько тэгов с весами:

```json
[
  { "tag_id": "tag_disc", "weight": 0.6 },
  { "tag_id": "tag_alg", "weight": 0.4 }
]
```

Формула:

- `weighted_attempts += weight`
- `weighted_correct += weight`, если ответ верный
- `mastery = weighted_correct / weighted_attempts`

Таким образом:

- `topic` отвечает за логику LMS
- `tag` отвечает за профиль навыков

## Схема БД

### auth-service

- `users`
  - `id`
  - `phone`
  - `email`
  - `first_name`
  - `last_name`
  - `nick`
  - `password_hash`
  - `roles_json`
  - `status`
  - `created_at`

### content-service

- `topics`
  - `id`
  - `parent_id`
  - `title`
  - `order_no`
  - `status`
  - `created_at`

- `tags`
  - `id`
  - `code`
  - `name`
  - `description`
  - `kind`
  - `status`
  - `created_at`

- `tasks`
  - `id`
  - `title`
  - `latex_body`
  - `topic_ids`
  - `tags`
  - `difficulty`
  - `correct_answer`
  - `status`
  - `author_id`
  - `created_at`
  - `updated_at`

- `task_tags`
  - `task_id`
  - `tag_id`
  - `weight`

- `theories`
  - `id`
  - `title`
  - `body`  (`latex_body` в API)
  - `summary`
  - `topic_ids`
  - `status`
  - `created_at`
  - `updated_at`

- `work_templates`
  - `id`
  - `title`
  - `task_ids`
  - `items_json`
  - `status`
  - `created_by`
  - `created_at`

### course-service

- `courses`
  - `id`
  - `title`
  - `owner_id`
  - `status`
  - `created_at`

- `course_members`
  - `course_id`
  - `user_id`
  - `role`

- `assignments`
  - `id`
  - `course_id`
  - `title`
  - `work_id`
  - `task_ids`
  - `due_at`
  - `assigned_by`
  - `status`
  - `created_at`

- `submissions`
  - `id`
  - `assignment_id`
  - `user_id`
  - `answers_json`
  - `status`
  - `submitted_at`
  - `review_json`

### document-service

- `document_jobs`
  - `id`
  - `format`
  - `status`
  - `files_json`
  - `error`
  - `created_at`
  - `completed_at`

### statistic-service

- `attempts`
  - `id`
  - `user_id`
  - `content_id`
  - `topic_ids`
  - `tag_scores`
  - `answer`
  - `is_correct`
  - `source`
  - `created_at`

## Внутренние gRPC контракты

- [auth-service/grpc/auth.proto](/home/artem/GolandProjects/itmo-lms/auth-service/grpc/auth.proto)
- [content-service/grpc/content.proto](/home/artem/GolandProjects/itmo-lms/content-service/grpc/content.proto)
- [course-service/grpc/course.proto](/home/artem/GolandProjects/itmo-lms/course-service/grpc/course.proto)
- [document-service/grpc/document.proto](/home/artem/GolandProjects/itmo-lms/document-service/grpc/document.proto)
- [statistic-service/grpc/statistic.proto](/home/artem/GolandProjects/itmo-lms/statistic-service/grpc/statistic.proto)
