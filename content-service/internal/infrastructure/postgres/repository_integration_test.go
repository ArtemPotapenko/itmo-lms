package postgres

import (
	"testing"
	"time"

	"itmo-lms/content-service/internal/domain"
	basepg "itmo-lms/pkg/postgres"
	"itmo-lms/pkg/testutil"
)

func TestRepository_CreateAndLoadWorkWithTheoryAndTask(t *testing.T) {
	ctx, dsn := testutil.StartPostgres(t, "content_repo_test")
	db, err := basepg.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := basepg.RunMigrations(ctx, db, "content-service", Migrations); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	repo := NewRepository(db)
	now := time.Now().UTC()

	tag := domain.Tag{ID: "tag_1", Code: "disc", Name: "Discriminant", Kind: "skill", Status: "active", CreatedAt: now}
	if err := repo.CreateTag(ctx, tag); err != nil {
		t.Fatalf("create tag: %v", err)
	}

	task := domain.Task{
		ID:            "tsk_1",
		Title:         "Solve",
		LatexBody:     "x^2-5x+6=0",
		TopicIDs:      []string{"top_1"},
		Tags:          []domain.TaskTag{{TagID: "tag_1", Code: "disc", Name: "Discriminant", Kind: "skill", Weight: 0.7}},
		Difficulty:    1,
		CorrectAnswer: "2,3",
		Status:        "published",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repo.CreateTask(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	theory := domain.Theory{
		ID:        "thr_1",
		Title:     "Theory",
		LatexBody: "D=b^2-4ac",
		TopicIDs:  []string{"top_1"},
		Status:    "published",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.CreateTheory(ctx, theory); err != nil {
		t.Fatalf("create theory: %v", err)
	}

	work := domain.WorkTemplate{
		ID:        "wrk_1",
		Title:     "Workbook",
		Items:     []domain.WorkItem{{Order: 1, Kind: "theory", ContentID: "thr_1", Title: "Theory"}, {Order: 2, Kind: "task", ContentID: "tsk_1", Title: "Solve"}},
		Status:    "published",
		CreatedBy: "usr_teacher",
		CreatedAt: now,
	}
	if err := repo.CreateWork(ctx, work); err != nil {
		t.Fatalf("create work: %v", err)
	}

	gotTask, ok, err := repo.GetTask(ctx, "tsk_1")
	if err != nil || !ok {
		t.Fatalf("get task: ok=%v err=%v", ok, err)
	}
	if len(gotTask.Tags) != 1 || gotTask.Tags[0].Weight != 0.7 {
		t.Fatalf("task tags = %+v, want weighted tag", gotTask.Tags)
	}

	gotTheory, ok, err := repo.GetTheory(ctx, "thr_1")
	if err != nil || !ok {
		t.Fatalf("get theory: ok=%v err=%v", ok, err)
	}
	if gotTheory.LatexBody != "D=b^2-4ac" {
		t.Fatalf("theory latex = %q", gotTheory.LatexBody)
	}

	gotWork, ok, err := repo.GetWork(ctx, "wrk_1")
	if err != nil || !ok {
		t.Fatalf("get work: ok=%v err=%v", ok, err)
	}
	if len(gotWork.Items) != 2 || gotWork.Items[0].Kind != "theory" || gotWork.Items[1].Kind != "task" {
		t.Fatalf("work items = %+v", gotWork.Items)
	}
}
