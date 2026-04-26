package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itmo-lms/pkg/platform"
	"itmo-lms/statistic-service/internal/application"
	"itmo-lms/statistic-service/internal/domain"
)

func TestCreateAttemptRequiresAuthAndCreatesAttempt(t *testing.T) {
	repo := &handlerRepo{}
	service := application.NewService(repo, handlerMetadata{}, nil, 0)
	handler := New(service, "test-secret")
	server := httptest.NewServer(handler.Routes())
	defer server.Close()

	reqBody := map[string]any{
		"user_id":    "usr_1",
		"content_id": "tsk_1",
		"answer":     "2,3",
		"is_correct": true,
	}

	resp := postStat(t, server.URL+"/attempts", "", reqBody)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	token := mustStatToken(t, "test-secret", "usr_1", []string{"student"})
	resp = postStat(t, server.URL+"/attempts", token, reqBody)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("created status = %d", resp.StatusCode)
	}
	if len(repo.attempts) != 1 {
		t.Fatalf("saved attempts = %d, want 1", len(repo.attempts))
	}
	if repo.attempts[0].ContentID != "tsk_1" || len(repo.attempts[0].TagScores) != 1 {
		t.Fatalf("saved attempt = %+v", repo.attempts[0])
	}
	if repo.attempts[0].Difficulty != 2 {
		t.Fatalf("saved attempt difficulty = %d", repo.attempts[0].Difficulty)
	}
}

func TestProfileReturnsKnowledgeProfile(t *testing.T) {
	now := time.Now().UTC()
	repo := &handlerRepo{
		profile: domain.KnowledgeProfile{
			UserID: "usr_1",
			Topics: map[string]domain.TopicStat{
				"top_1": {UserID: "usr_1", TopicID: "top_1", Attempts: 2, Correct: 1, WeightedAttempts: 5, WeightedCorrect: 4, Accuracy: 0.5, Rating: 8, UpdatedAt: now},
			},
			Tags: map[string]domain.TagStat{
				"tag_1": {UserID: "usr_1", TagID: "tag_1", WeightedAttempts: 1, WeightedCorrect: 0.7, Mastery: 0.7, UpdatedAt: now},
			},
			UpdatedAt: now,
		},
	}
	service := application.NewService(repo, nil, nil, 0)
	handler := New(service, "test-secret")
	server := httptest.NewServer(handler.Routes())
	defer server.Close()

	token := mustStatToken(t, "test-secret", "usr_1", []string{"student"})
	req, err := http.NewRequest(http.MethodGet, server.URL+"/users/usr_1/knowledge-profile", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var decoded domain.KnowledgeProfile
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode profile: %v", err)
	}
	if decoded.Topics["top_1"].Accuracy != 0.5 || decoded.Topics["top_1"].Rating != 8 || decoded.Tags["tag_1"].Mastery != 0.7 {
		t.Fatalf("decoded profile = %+v", decoded)
	}
}

func TestCourseCalibrationEndpointReturnsPayload(t *testing.T) {
	repo := &handlerRepo{
		courseAttempts: []domain.Attempt{
			{ID: "att_1", UserID: "usr_1", CourseID: "crs_1", ContentID: "tsk_1", TopicIDs: []string{"top_1"}, TagScores: []domain.TagScore{{TagID: "tag_1", Weight: 1}}, Difficulty: 3, IsCorrect: true},
			{ID: "att_2", UserID: "usr_2", CourseID: "crs_1", ContentID: "tsk_1", TopicIDs: []string{"top_1"}, TagScores: []domain.TagScore{{TagID: "tag_1", Weight: 1}}, Difficulty: 3, IsCorrect: false},
		},
	}
	service := application.NewService(repo, nil, nil, 0)
	handler := New(service, "test-secret")
	server := httptest.NewServer(handler.Routes())
	defer server.Close()

	token := mustStatToken(t, "test-secret", "usr_1", []string{"teacher"})
	req, err := http.NewRequest(http.MethodGet, server.URL+"/courses/crs_1/calibration", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var decoded domain.CourseCalibration
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode calibration: %v", err)
	}
	if decoded.CourseID != "crs_1" || decoded.TaskCalibrations["tsk_1"].AttemptCount != 2 {
		t.Fatalf("decoded calibration = %+v", decoded)
	}
}

func mustStatToken(t *testing.T, secret, sub string, roles []string) string {
	t.Helper()
	token, err := platform.SignToken(secret, platform.Claims{Subject: sub, Roles: roles, Expires: time.Now().Add(time.Hour).Unix()})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

func postStat(t *testing.T, url, token string, payload any) *http.Response {
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

type handlerMetadata struct{}

func (handlerMetadata) ResolveTask(context.Context, string) ([]string, []domain.TagScore, int, error) {
	return []string{"top_1"}, []domain.TagScore{{TagID: "tag_1", Code: "disc", Weight: 0.7}}, 2, nil
}

type handlerRepo struct {
	attempts       []domain.Attempt
	profile        domain.KnowledgeProfile
	courseAttempts []domain.Attempt
}

func (r *handlerRepo) AddAttempt(_ context.Context, attempt domain.Attempt) error {
	r.attempts = append(r.attempts, attempt)
	return nil
}

func (r *handlerRepo) ListAttempts(_ context.Context, _ string) ([]domain.Attempt, error) {
	return r.attempts, nil
}

func (r *handlerRepo) ListCourseAttempts(_ context.Context, _ string) ([]domain.Attempt, error) {
	return r.courseAttempts, nil
}

func (r *handlerRepo) Profile(_ context.Context, _ string) (domain.KnowledgeProfile, error) {
	return r.profile, nil
}
