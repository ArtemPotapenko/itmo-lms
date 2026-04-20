package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"itmo-lms/document-service/internal/application"
	"itmo-lms/document-service/internal/domain"
	httptransport "itmo-lms/document-service/internal/transport/http"
	"itmo-lms/pkg/platform"
)

func TestCompileGetAndDownloadDocument(t *testing.T) {
	secret := "test-secret"
	dataDir := t.TempDir()
	service := application.NewService(newDocRepo(), dataDir)
	server := httptest.NewServer(httptransport.New(service, secret).Routes())
	defer server.Close()

	teacherToken := mustDocToken(t, secret, "usr_teacher", []string{"teacher"})
	studentToken := mustDocToken(t, secret, "usr_student", []string{"student"})

	compileResp := postDocJSON(t, server.URL+"/documents/compile", teacherToken, map[string]any{
		"title":  "Подборка",
		"format": "tex",
		"tasks": []map[string]any{
			{"id": "tsk_1", "title": "Задача 1", "latex_body": "x^2-5x+6=0"},
		},
	})
	defer compileResp.Body.Close()
	if compileResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(compileResp.Body)
		t.Fatalf("compile status=%d body=%s", compileResp.StatusCode, body)
	}
	var created domain.DocumentJob
	if err := json.NewDecoder(compileResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode compile response: %v", err)
	}
	if created.Status != "completed" || len(created.Files) != 1 {
		t.Fatalf("created job = %+v", created)
	}

	getResp := getDoc(t, server.URL+"/documents/"+created.ID, studentToken)
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(getResp.Body)
		t.Fatalf("get status=%d body=%s", getResp.StatusCode, body)
	}

	downloadResp := getDoc(t, server.URL+"/documents/"+created.ID+"/download", studentToken)
	defer downloadResp.Body.Close()
	if downloadResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(downloadResp.Body)
		t.Fatalf("download status=%d body=%s", downloadResp.StatusCode, body)
	}
	raw, err := io.ReadAll(downloadResp.Body)
	if err != nil {
		t.Fatalf("read download body: %v", err)
	}
	if !bytes.Contains(raw, []byte("Задача 1")) {
		t.Fatalf("downloaded content missing task title: %s", string(raw))
	}

	if _, err := os.Stat(filepath.Join(dataDir, created.Files[0].StorageKey)); err != nil {
		t.Fatalf("compiled file not found: %v", err)
	}
}

func mustDocToken(t *testing.T, secret, sub string, roles []string) string {
	t.Helper()
	token, err := platform.SignToken(secret, platform.Claims{Subject: sub, Roles: roles, Expires: time.Now().Add(time.Hour).Unix()})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

func postDocJSON(t *testing.T, url, token string, payload any) *http.Response {
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
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func getDoc(t *testing.T, url, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

type docRepo struct {
	jobs map[string]domain.DocumentJob
}

func newDocRepo() *docRepo {
	return &docRepo{jobs: map[string]domain.DocumentJob{}}
}

func (r *docRepo) SaveJob(_ context.Context, job domain.DocumentJob) error {
	r.jobs[job.ID] = job
	return nil
}

func (r *docRepo) UpdateJob(_ context.Context, job domain.DocumentJob) error {
	r.jobs[job.ID] = job
	return nil
}

func (r *docRepo) GetJob(_ context.Context, id string) (domain.DocumentJob, bool, error) {
	job, ok := r.jobs[id]
	return job, ok, nil
}
