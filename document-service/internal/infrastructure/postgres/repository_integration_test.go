package postgres

import (
	"testing"
	"time"

	"itmo-lms/document-service/internal/domain"
	basepg "itmo-lms/pkg/postgres"
	"itmo-lms/pkg/testutil"
)

func TestRepository_SaveUpdateAndGetJob(t *testing.T) {
	ctx, dsn := testutil.StartPostgres(t, "document_repo_test")
	db, err := basepg.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := basepg.RunMigrations(ctx, db, "document-service", Migrations); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	repo := NewRepository(db)
	now := time.Now().UTC()
	job := domain.DocumentJob{
		ID:          "doc_1",
		Format:      "tex",
		Status:      "completed",
		Files:       []domain.DocumentFile{{ID: "file_1", JobID: "doc_1", Kind: "source", StorageKey: "doc_1.tex", MimeType: "text/x-tex", Size: 42, Checksum: "abc", CreatedAt: now}},
		CreatedAt:   now,
		CompletedAt: now,
	}

	if err := repo.SaveJob(ctx, job); err != nil {
		t.Fatalf("save job: %v", err)
	}

	job.Status = "failed"
	job.Error = "compile error"
	if err := repo.UpdateJob(ctx, job); err != nil {
		t.Fatalf("update job: %v", err)
	}

	got, ok, err := repo.GetJob(ctx, "doc_1")
	if err != nil || !ok {
		t.Fatalf("get job: ok=%v err=%v", ok, err)
	}
	if got.Status != "failed" || got.Error != "compile error" {
		t.Fatalf("got job = %+v", got)
	}
	if len(got.Files) != 1 || got.Files[0].StorageKey != "doc_1.tex" {
		t.Fatalf("got files = %+v", got.Files)
	}
}
