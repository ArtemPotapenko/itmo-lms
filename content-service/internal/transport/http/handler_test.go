package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itmo-lms/content-service/internal/application"
	"itmo-lms/content-service/internal/domain"
	"itmo-lms/pkg/events"
	"itmo-lms/pkg/platform"
)

func TestCreateTaskRequiresTeacherRole(t *testing.T) {
	repo := newHandlerRepo()
	repo.tags["tag_1"] = domain.Tag{ID: "tag_1", Code: "disc", Name: "Дискриминант", Kind: "skill"}

	handler := New(application.NewService(repo, nil, nil), "test-secret", nil)
	server := httptest.NewServer(handler.Routes())
	defer server.Close()

	token := mustContentToken(t, "test-secret", "usr_student", []string{"student"})
	resp := postContent(t, server.URL+"/tasks", token, map[string]any{
		"title":          "Задача",
		"latex_body":     "x^2-5x+6=0",
		"topic_ids":      []string{"top_1"},
		"tags":           []map[string]any{{"tag_id": "tag_1", "weight": 1}},
		"correct_answer": "2,3",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestCheckTaskPublishesAttemptEvent(t *testing.T) {
	repo := newHandlerRepo()
	repo.tasks["tsk_1"] = domain.Task{
		ID:            "tsk_1",
		Title:         "Квадратное уравнение",
		LatexBody:     "x^2-5x+6=0",
		TopicIDs:      []string{"top_1"},
		Tags:          []domain.TaskTag{{TagID: "tag_1", Code: "disc", Weight: 1}},
		CorrectAnswer: "2,3",
	}
	publisher := &recordingAttemptPublisher{}

	handler := New(application.NewService(repo, nil, nil), "test-secret", publisher)
	server := httptest.NewServer(handler.Routes())
	defer server.Close()

	token := mustContentToken(t, "test-secret", "usr_student", []string{"student"})
	resp := postContent(t, server.URL+"/tasks/tsk_1/check", token, map[string]any{
		"user_id": "usr_student",
		"answer":  "2,3",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if len(publisher.events) != 1 {
		t.Fatalf("published events = %d, want 1", len(publisher.events))
	}
	event := publisher.events[0]
	if event.UserID != "usr_student" || event.ContentID != "tsk_1" || !event.IsCorrect || event.Source != "practice" {
		t.Fatalf("event = %+v", event)
	}
}

func TestCheckWorkPublishesEventPerTask(t *testing.T) {
	repo := newHandlerRepo()
	repo.tasks["tsk_1"] = domain.Task{
		ID:            "tsk_1",
		Title:         "Первая задача",
		LatexBody:     "x^2-5x+6=0",
		TopicIDs:      []string{"top_1"},
		Tags:          []domain.TaskTag{{TagID: "tag_1", Code: "disc", Weight: 0.7}},
		CorrectAnswer: "2,3",
	}
	repo.tasks["tsk_2"] = domain.Task{
		ID:            "tsk_2",
		Title:         "Вторая задача",
		LatexBody:     "x^2-9=0",
		TopicIDs:      []string{"top_1"},
		Tags:          []domain.TaskTag{{TagID: "tag_2", Code: "roots", Weight: 0.3}},
		CorrectAnswer: "3,-3",
	}
	repo.works["wrk_1"] = domain.WorkTemplate{
		ID:    "wrk_1",
		Title: "Тетрадь",
		Items: []domain.WorkItem{
			{Order: 1, Kind: "task", ContentID: "tsk_1"},
			{Order: 2, Kind: "task", ContentID: "tsk_2"},
		},
	}
	publisher := &recordingAttemptPublisher{}

	handler := New(application.NewService(repo, nil, nil), "test-secret", publisher)
	server := httptest.NewServer(handler.Routes())
	defer server.Close()

	token := mustContentToken(t, "test-secret", "usr_student", []string{"student"})
	resp := postContent(t, server.URL+"/work-templates/wrk_1/check", token, map[string]any{
		"user_id": "usr_student",
		"answers": []map[string]any{
			{"task_id": "tsk_1", "answer": "2,3"},
			{"task_id": "tsk_2", "answer": "0"},
		},
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var decoded domain.WorkCheckResult
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if decoded.TotalTasks != 2 || decoded.CorrectTasks != 1 {
		t.Fatalf("decoded = %+v", decoded)
	}
	if len(publisher.events) != 2 {
		t.Fatalf("published events = %d, want 2", len(publisher.events))
	}
	if publisher.events[0].Source != "workbook" || publisher.events[1].Source != "workbook" {
		t.Fatalf("events = %+v", publisher.events)
	}
}

func mustContentToken(t *testing.T, secret, sub string, roles []string) string {
	t.Helper()
	token, err := platform.SignToken(secret, platform.Claims{Subject: sub, Roles: roles, Expires: time.Now().Add(time.Hour).Unix()})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

func postContent(t *testing.T, url, token string, payload any) *http.Response {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

type recordingAttemptPublisher struct {
	events []events.AttemptEvaluated
}

func (p *recordingAttemptPublisher) PublishAttempt(_ *http.Request, event events.AttemptEvaluated) error {
	p.events = append(p.events, event)
	return nil
}

type handlerRepo struct {
	tags     map[string]domain.Tag
	tasks    map[string]domain.Task
	theories map[string]domain.Theory
	works    map[string]domain.WorkTemplate
}

func newHandlerRepo() *handlerRepo {
	return &handlerRepo{
		tags:     map[string]domain.Tag{},
		tasks:    map[string]domain.Task{},
		theories: map[string]domain.Theory{},
		works:    map[string]domain.WorkTemplate{},
	}
}

func (r *handlerRepo) CreateTopic(context.Context, domain.Topic) error { return nil }

func (r *handlerRepo) ListTopics(context.Context) ([]domain.Topic, error) { return nil, nil }

func (r *handlerRepo) CreateTag(_ context.Context, tag domain.Tag) error {
	r.tags[tag.ID] = tag
	return nil
}

func (r *handlerRepo) ListTags(context.Context) ([]domain.Tag, error) { return nil, nil }

func (r *handlerRepo) GetTag(_ context.Context, id string) (domain.Tag, bool, error) {
	tag, ok := r.tags[id]
	return tag, ok, nil
}

func (r *handlerRepo) CreateTask(_ context.Context, task domain.Task) error {
	r.tasks[task.ID] = task
	return nil
}

func (r *handlerRepo) ListTasks(_ context.Context, _ string) ([]domain.Task, error) { return nil, nil }

func (r *handlerRepo) GetTask(_ context.Context, id string) (domain.Task, bool, error) {
	task, ok := r.tasks[id]
	return task, ok, nil
}

func (r *handlerRepo) CreateTheory(_ context.Context, theory domain.Theory) error {
	r.theories[theory.ID] = theory
	return nil
}

func (r *handlerRepo) ListTheory(_ context.Context, _ string) ([]domain.Theory, error) {
	return nil, nil
}

func (r *handlerRepo) GetTheory(_ context.Context, id string) (domain.Theory, bool, error) {
	theory, ok := r.theories[id]
	return theory, ok, nil
}

func (r *handlerRepo) CreateWork(_ context.Context, work domain.WorkTemplate) error {
	r.works[work.ID] = work
	return nil
}

func (r *handlerRepo) ListWorks(context.Context) ([]domain.WorkTemplate, error) { return nil, nil }

func (r *handlerRepo) GetWork(_ context.Context, id string) (domain.WorkTemplate, bool, error) {
	work, ok := r.works[id]
	return work, ok, nil
}
