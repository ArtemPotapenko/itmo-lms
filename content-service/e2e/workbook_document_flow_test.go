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

	contentapp "itmo-lms/content-service/internal/application"
	contentdomain "itmo-lms/content-service/internal/domain"
	contenthttp "itmo-lms/content-service/internal/transport/http"
)

func TestWorkbookDocumentBuildStoresPDFArtifact(t *testing.T) {
	secret := "test-secret"
	dataDir := t.TempDir()

	compiler := &artifactCompiler{dir: dataDir}
	service := contentapp.NewService(newE2EContentRepo(), compiler)

	server := httptest.NewServer(contenthttp.New(service, secret, nil).Routes())
	defer server.Close()

	teacherToken := mustToken(t, secret, "usr_teacher", []string{"teacher"})

	theoryID := createJSON(t, server.URL+"/theory", teacherToken, map[string]any{
		"title":      "Теория дискриминанта",
		"latex_body": "D=b^2-4ac",
		"summary":    "Коротко",
		"topic_ids":  []string{"top_1"},
	})

	taskID := createJSON(t, server.URL+"/tasks", teacherToken, map[string]any{
		"title":          "Первая задача",
		"latex_body":     "x^2-5x+6=0",
		"topic_ids":      []string{"top_1"},
		"difficulty":     1,
		"correct_answer": "2,3",
		"status":         "published",
	})

	workID := createJSON(t, server.URL+"/work-templates", teacherToken, map[string]any{
		"title": "Рабочая тетрадь",
		"items": []map[string]any{
			{"order": 1, "kind": "theory", "content_id": theoryID},
			{"order": 2, "kind": "task", "content_id": taskID},
		},
		"status":     "published",
		"created_by": "usr_teacher",
	})

	resp := postJSON(t, server.URL+"/work-templates/"+workID+"/documents", teacherToken, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("build document status=%d body=%s", resp.StatusCode, body)
	}

	var decoded map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode build response: %v", err)
	}
	jobID := decoded["job_id"]

	path := filepath.Join(dataDir, jobID+".pdf")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read pdf artifact: %v", err)
	}
	if !bytes.HasPrefix(raw, []byte("%PDF-")) {
		t.Fatalf("artifact is not pdf: %q", raw[:min(len(raw), 16)])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type artifactCompiler struct {
	dir string
}

func (c *artifactCompiler) Compile(_ context.Context, _ string, tasks []contentdomain.Task) (string, error) {
	jobID := "doc_workbook"
	var body bytes.Buffer
	body.WriteString("%PDF-1.4\n")
	for _, task := range tasks {
		body.WriteString(task.Title)
		body.WriteByte('\n')
		body.WriteString(task.LatexBody)
		body.WriteByte('\n')
	}
	if err := os.WriteFile(filepath.Join(c.dir, jobID+".pdf"), body.Bytes(), 0o644); err != nil {
		return "", err
	}
	return jobID, nil
}
