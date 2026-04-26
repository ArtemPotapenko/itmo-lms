package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	contentapp "itmo-lms/content-service/internal/application"
	contentdomain "itmo-lms/content-service/internal/domain"
	contenthttp "itmo-lms/content-service/internal/transport/http"
	"itmo-lms/pkg/events"
	"itmo-lms/pkg/platform"
)

func TestWorkbookSubmissionPublishesAttemptEventsEndToEnd(t *testing.T) {
	secret := "test-secret"
	repo := newE2EContentRepo()
	publisher := &recordingPublisher{}
	service := contentapp.NewService(repo, nil, nil)

	server := httptest.NewServer(contenthttp.New(service, secret, publisher).Routes())
	defer server.Close()

	teacherToken := mustToken(t, secret, "usr_teacher", []string{"teacher"})
	studentToken := mustToken(t, secret, "usr_student", []string{"student"})

	theoryID := createJSON(t, server.URL+"/theory", teacherToken, map[string]any{
		"title":      "Теория дискриминанта",
		"latex_body": "D=b^2-4ac",
		"summary":    "Коротко",
		"topic_ids":  []string{"top_1"},
	})

	tagID := createJSON(t, server.URL+"/tags", teacherToken, map[string]any{
		"code":   "disc",
		"name":   "Дискриминант",
		"kind":   "skill",
		"status": "active",
	})

	task1ID := createJSON(t, server.URL+"/tasks", teacherToken, map[string]any{
		"title":          "Первая задача",
		"latex_body":     "x^2-5x+6=0",
		"topic_ids":      []string{"top_1"},
		"tags":           []map[string]any{{"tag_id": tagID, "weight": 0.7}},
		"difficulty":     1,
		"correct_answer": "2,3",
		"status":         "published",
	})

	task2ID := createJSON(t, server.URL+"/tasks", teacherToken, map[string]any{
		"title":          "Вторая задача",
		"latex_body":     "x^2-9=0",
		"topic_ids":      []string{"top_1"},
		"tags":           []map[string]any{{"tag_id": tagID, "weight": 0.3}},
		"difficulty":     1,
		"correct_answer": "3,-3",
		"status":         "published",
	})

	workID := createJSON(t, server.URL+"/work-templates", teacherToken, map[string]any{
		"title": "Рабочая тетрадь",
		"items": []map[string]any{
			{"order": 1, "kind": "theory", "content_id": theoryID},
			{"order": 2, "kind": "task", "content_id": task1ID},
			{"order": 3, "kind": "task", "content_id": task2ID},
		},
		"status":     "published",
		"created_by": "usr_teacher",
	})

	resp := postJSON(t, server.URL+"/work-templates/"+workID+"/check", studentToken, map[string]any{
		"user_id": "usr_student",
		"source":  "workbook",
		"answers": []map[string]any{
			{"task_id": task1ID, "answer": "2,3"},
			{"task_id": task2ID, "answer": "0"},
		},
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("check work status = %d body=%s", resp.StatusCode, body)
	}

	var result struct {
		TotalTasks   int `json:"total_tasks"`
		CorrectTasks int `json:"correct_tasks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode check response: %v", err)
	}
	if result.TotalTasks != 2 || result.CorrectTasks != 1 {
		t.Fatalf("result = %+v", result)
	}

	if len(publisher.events) != 2 {
		t.Fatalf("published events = %d, want 2", len(publisher.events))
	}
	if publisher.events[0].UserID != "usr_student" || publisher.events[0].ContentID != task1ID || !publisher.events[0].IsCorrect {
		t.Fatalf("first event = %+v", publisher.events[0])
	}
	if publisher.events[1].ContentID != task2ID || publisher.events[1].IsCorrect {
		t.Fatalf("second event = %+v", publisher.events[1])
	}
}

func mustToken(t *testing.T, secret, sub string, roles []string) string {
	t.Helper()
	token, err := platform.SignToken(secret, platform.Claims{Subject: sub, Roles: roles, Expires: time.Now().Add(time.Hour).Unix()})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

func createJSON(t *testing.T, url, token string, payload any) string {
	t.Helper()
	resp := postJSON(t, url, token, payload)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("create %s status=%d body=%s", url, resp.StatusCode, body)
	}
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	id, _ := decoded["id"].(string)
	return id
}

func postJSON(t *testing.T, url, token string, payload any) *http.Response {
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

type recordingPublisher struct {
	events []events.AttemptEvaluated
}

func (p *recordingPublisher) PublishAttempt(_ *http.Request, event events.AttemptEvaluated) error {
	p.events = append(p.events, event)
	return nil
}

type e2EContentRepo struct {
	tags     map[string]contentdomain.Tag
	tasks    map[string]contentdomain.Task
	theories map[string]contentdomain.Theory
	works    map[string]contentdomain.WorkTemplate
}

func newE2EContentRepo() *e2EContentRepo {
	return &e2EContentRepo{
		tags:     map[string]contentdomain.Tag{},
		tasks:    map[string]contentdomain.Task{},
		theories: map[string]contentdomain.Theory{},
		works:    map[string]contentdomain.WorkTemplate{},
	}
}

func (r *e2EContentRepo) CreateTopic(context.Context, contentdomain.Topic) error { return nil }
func (r *e2EContentRepo) ListTopics(context.Context) ([]contentdomain.Topic, error) {
	return nil, nil
}
func (r *e2EContentRepo) CreateTag(_ context.Context, tag contentdomain.Tag) error {
	r.tags[tag.ID] = tag
	return nil
}
func (r *e2EContentRepo) ListTags(context.Context) ([]contentdomain.Tag, error) { return nil, nil }
func (r *e2EContentRepo) GetTag(_ context.Context, id string) (contentdomain.Tag, bool, error) {
	tag, ok := r.tags[id]
	return tag, ok, nil
}
func (r *e2EContentRepo) CreateTask(_ context.Context, task contentdomain.Task) error {
	r.tasks[task.ID] = task
	return nil
}
func (r *e2EContentRepo) ListTasks(_ context.Context, _ string) ([]contentdomain.Task, error) {
	return nil, nil
}
func (r *e2EContentRepo) GetTask(_ context.Context, id string) (contentdomain.Task, bool, error) {
	task, ok := r.tasks[id]
	return task, ok, nil
}
func (r *e2EContentRepo) CreateTheory(_ context.Context, theory contentdomain.Theory) error {
	r.theories[theory.ID] = theory
	return nil
}
func (r *e2EContentRepo) ListTheory(_ context.Context, _ string) ([]contentdomain.Theory, error) {
	return nil, nil
}
func (r *e2EContentRepo) GetTheory(_ context.Context, id string) (contentdomain.Theory, bool, error) {
	theory, ok := r.theories[id]
	return theory, ok, nil
}
func (r *e2EContentRepo) CreateWork(_ context.Context, work contentdomain.WorkTemplate) error {
	r.works[work.ID] = work
	return nil
}
func (r *e2EContentRepo) ListWorks(context.Context) ([]contentdomain.WorkTemplate, error) {
	return nil, nil
}
func (r *e2EContentRepo) GetWork(_ context.Context, id string) (contentdomain.WorkTemplate, bool, error) {
	work, ok := r.works[id]
	return work, ok, nil
}
