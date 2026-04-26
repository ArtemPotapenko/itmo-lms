package postgres

import (
	"testing"
	"time"

	"itmo-lms/pkg/postgres"
	"itmo-lms/pkg/testutil"
	"itmo-lms/statistic-service/internal/domain"
)

func TestRepository_ProfileAggregatesTopicsAndTags(t *testing.T) {
	ctx, dsn := testutil.StartPostgres(t, "stat_repo_test")
	db, err := postgres.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := postgres.RunMigrations(ctx, db, "statistic-service", Migrations); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	repo := NewRepository(db)
	now := time.Now().UTC()

	attempts := []domain.Attempt{
		{ID: "att_1", UserID: "usr_1", CourseID: "crs_1", ContentID: "tsk_1", TopicIDs: []string{"top_1"}, TagScores: []domain.TagScore{{TagID: "tag_1", Code: "disc", Weight: 0.6}}, Difficulty: 3, Answer: "2,3", IsCorrect: true, Source: "practice", CreatedAt: now},
		{ID: "att_2", UserID: "usr_1", CourseID: "crs_1", ContentID: "tsk_2", TopicIDs: []string{"top_1"}, TagScores: []domain.TagScore{{TagID: "tag_1", Code: "disc", Weight: 0.4}}, Difficulty: 1, Answer: "7", IsCorrect: false, Source: "practice", CreatedAt: now.Add(time.Second)},
	}
	for _, attempt := range attempts {
		if err := repo.AddAttempt(ctx, attempt); err != nil {
			t.Fatalf("add attempt %s: %v", attempt.ID, err)
		}
	}

	profile, err := repo.Profile(ctx, "usr_1")
	if err != nil {
		t.Fatalf("profile: %v", err)
	}

	if got := profile.Topics["top_1"].Accuracy; got != 0.5 {
		t.Fatalf("topic accuracy = %v, want 0.5", got)
	}
	if got := profile.Topics["top_1"].WeightedAttempts; got != 4 {
		t.Fatalf("weighted attempts = %v, want 4", got)
	}
	if got := profile.Topics["top_1"].WeightedCorrect; got != 3 {
		t.Fatalf("weighted correct = %v, want 3", got)
	}
	if got := profile.Topics["top_1"].Rating; got != 7.5 {
		t.Fatalf("topic rating = %v, want 7.5", got)
	}
	if got := profile.Tags["tag_1"].WeightedAttempts; got != 1.0 {
		t.Fatalf("weighted attempts = %v, want 1.0", got)
	}
	if got := profile.Tags["tag_1"].WeightedCorrect; got != 0.6 {
		t.Fatalf("weighted correct = %v, want 0.6", got)
	}
	if got := profile.Tags["tag_1"].Mastery; got != 0.6 {
		t.Fatalf("mastery = %v, want 0.6", got)
	}

	courseAttempts, err := repo.ListCourseAttempts(ctx, "crs_1")
	if err != nil {
		t.Fatalf("list course attempts: %v", err)
	}
	if len(courseAttempts) != 2 {
		t.Fatalf("course attempts = %d, want 2", len(courseAttempts))
	}
}
