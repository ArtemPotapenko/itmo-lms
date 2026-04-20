package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"itmo-lms/content-service/internal/domain"
)

func TestCreateWorkPopulatesTitlesAndOrder(t *testing.T) {
	repo := newFakeContentRepo()
	repo.tasks["tsk_1"] = domain.Task{ID: "tsk_1", Title: "Задача на дискриминант", LatexBody: "Solve"}
	repo.theories["thr_1"] = domain.Theory{ID: "thr_1", Title: "Формула дискриминанта", LatexBody: "D=b^2-4ac"}

	service := NewService(repo, nil)

	work, err := service.CreateWork(context.Background(), domain.WorkTemplate{
		Title: "Тетрадь 1",
		Items: []domain.WorkItem{
			{Kind: "theory", ContentID: "thr_1"},
			{Kind: "task", ContentID: "tsk_1"},
		},
	})
	if err != nil {
		t.Fatalf("CreateWork() error = %v", err)
	}

	if got := work.Items[0].Title; got != "Формула дискриминанта" {
		t.Fatalf("first item title = %q, want theory title", got)
	}
	if got := work.Items[1].Title; got != "Задача на дискриминант" {
		t.Fatalf("second item title = %q, want task title", got)
	}
	if work.Items[0].Order != 1 || work.Items[1].Order != 2 {
		t.Fatalf("orders = %v, want 1 and 2", []int{work.Items[0].Order, work.Items[1].Order})
	}
}

func TestBuildWorkLatexIncludesTheoryAndTasksInOrder(t *testing.T) {
	repo := newFakeContentRepo()
	repo.works["wrk_1"] = domain.WorkTemplate{
		ID:    "wrk_1",
		Title: "Тетрадь по квадратным уравнениям",
		Items: []domain.WorkItem{
			{Order: 2, Kind: "task", ContentID: "tsk_1"},
			{Order: 1, Kind: "theory", ContentID: "thr_1"},
		},
	}
	repo.theories["thr_1"] = domain.Theory{ID: "thr_1", Title: "Теория", LatexBody: "D=b^2-4ac"}
	repo.tasks["tsk_1"] = domain.Task{ID: "tsk_1", Title: "Задача", LatexBody: "x^2-5x+6=0"}

	service := NewService(repo, nil)

	latex, err := service.BuildWorkLatex(context.Background(), "wrk_1")
	if err != nil {
		t.Fatalf("BuildWorkLatex() error = %v", err)
	}

	theoryPos := strings.Index(latex, "Теория. Теория")
	taskPos := strings.Index(latex, "Задача 1. Задача")
	if theoryPos == -1 || taskPos == -1 {
		t.Fatalf("latex does not contain expected sections: %s", latex)
	}
	if theoryPos > taskPos {
		t.Fatalf("theory appears after task in latex: %s", latex)
	}
}

func TestCheckWorkAggregatesResults(t *testing.T) {
	repo := newFakeContentRepo()
	repo.works["wrk_1"] = domain.WorkTemplate{
		ID:    "wrk_1",
		Title: "Тетрадь",
		Items: []domain.WorkItem{
			{Order: 1, Kind: "theory", ContentID: "thr_1"},
			{Order: 2, Kind: "task", ContentID: "tsk_1"},
			{Order: 3, Kind: "task", ContentID: "tsk_2"},
		},
	}
	repo.theories["thr_1"] = domain.Theory{ID: "thr_1", Title: "Теория", LatexBody: "text"}
	repo.tasks["tsk_1"] = domain.Task{ID: "tsk_1", Title: "Первая", CorrectAnswer: "2,3", TopicIDs: []string{"top_1"}}
	repo.tasks["tsk_2"] = domain.Task{ID: "tsk_2", Title: "Вторая", CorrectAnswer: "5", TopicIDs: []string{"top_2"}}

	service := NewService(repo, nil)
	result, err := service.CheckWork(context.Background(), "wrk_1", "usr_1", "workbook", []domain.WorkAnswer{
		{TaskID: "tsk_1", Answer: "2,3"},
		{TaskID: "tsk_2", Answer: "7"},
	})
	if err != nil {
		t.Fatalf("CheckWork() error = %v", err)
	}

	if result.TotalTasks != 2 || result.CorrectTasks != 1 {
		t.Fatalf("totals = (%d, %d), want (2,1)", result.TotalTasks, result.CorrectTasks)
	}
	if len(result.Results) != 2 {
		t.Fatalf("results len = %d, want 2", len(result.Results))
	}
	if !result.Results[0].IsCorrect || result.Results[1].IsCorrect {
		t.Fatalf("unexpected correctness flags: %+v", result.Results)
	}
}

func TestCheckTaskAcceptsNumericAnswersIgnoringOrderAndFormat(t *testing.T) {
	repo := newFakeContentRepo()
	repo.tasks["tsk_1"] = domain.Task{ID: "tsk_1", Title: "Корни", CorrectAnswer: "2,3"}

	service := NewService(repo, nil)

	result, err := service.CheckTask(context.Background(), "tsk_1", "3; 2")
	if err != nil {
		t.Fatalf("CheckTask() error = %v", err)
	}
	if !result["is_correct"].(bool) {
		t.Fatalf("expected numeric set answer to be correct, got %+v", result)
	}
}

func TestCheckTaskAcceptsFractionsAndDecimals(t *testing.T) {
	repo := newFakeContentRepo()
	repo.tasks["tsk_1"] = domain.Task{ID: "tsk_1", Title: "Дробь", CorrectAnswer: "1/2"}

	service := NewService(repo, nil)

	result, err := service.CheckTask(context.Background(), "tsk_1", "0.5")
	if err != nil {
		t.Fatalf("CheckTask() error = %v", err)
	}
	if !result["is_correct"].(bool) {
		t.Fatalf("expected fraction/decimal answer to be correct, got %+v", result)
	}
}

func TestCheckTaskRejectsDifferentNumericAnswer(t *testing.T) {
	repo := newFakeContentRepo()
	repo.tasks["tsk_1"] = domain.Task{ID: "tsk_1", Title: "Число", CorrectAnswer: "5"}

	service := NewService(repo, nil)

	result, err := service.CheckTask(context.Background(), "tsk_1", "7")
	if err != nil {
		t.Fatalf("CheckTask() error = %v", err)
	}
	if result["is_correct"].(bool) {
		t.Fatalf("expected answer to be incorrect, got %+v", result)
	}
}

type fakeContentRepo struct {
	topics   map[string]domain.Topic
	tags     map[string]domain.Tag
	tasks    map[string]domain.Task
	theories map[string]domain.Theory
	works    map[string]domain.WorkTemplate
}

func newFakeContentRepo() *fakeContentRepo {
	return &fakeContentRepo{
		topics:   map[string]domain.Topic{},
		tags:     map[string]domain.Tag{},
		tasks:    map[string]domain.Task{},
		theories: map[string]domain.Theory{},
		works:    map[string]domain.WorkTemplate{},
	}
}

func (r *fakeContentRepo) CreateTopic(_ context.Context, topic domain.Topic) error {
	r.topics[topic.ID] = topic
	return nil
}

func (r *fakeContentRepo) ListTopics(_ context.Context) ([]domain.Topic, error) { return nil, nil }
func (r *fakeContentRepo) CreateTag(_ context.Context, tag domain.Tag) error {
	r.tags[tag.ID] = tag
	return nil
}
func (r *fakeContentRepo) ListTags(_ context.Context) ([]domain.Tag, error) { return nil, nil }
func (r *fakeContentRepo) GetTag(_ context.Context, id string) (domain.Tag, bool, error) {
	tag, ok := r.tags[id]
	return tag, ok, nil
}
func (r *fakeContentRepo) CreateTask(_ context.Context, task domain.Task) error {
	r.tasks[task.ID] = task
	return nil
}
func (r *fakeContentRepo) ListTasks(_ context.Context, _ string) ([]domain.Task, error) {
	return nil, nil
}
func (r *fakeContentRepo) GetTask(_ context.Context, id string) (domain.Task, bool, error) {
	task, ok := r.tasks[id]
	return task, ok, nil
}
func (r *fakeContentRepo) CreateTheory(_ context.Context, theory domain.Theory) error {
	r.theories[theory.ID] = theory
	return nil
}
func (r *fakeContentRepo) ListTheory(_ context.Context, _ string) ([]domain.Theory, error) {
	return nil, nil
}
func (r *fakeContentRepo) GetTheory(_ context.Context, id string) (domain.Theory, bool, error) {
	theory, ok := r.theories[id]
	return theory, ok, nil
}
func (r *fakeContentRepo) CreateWork(_ context.Context, work domain.WorkTemplate) error {
	if work.CreatedAt.IsZero() {
		work.CreatedAt = time.Now().UTC()
	}
	r.works[work.ID] = work
	return nil
}
func (r *fakeContentRepo) ListWorks(_ context.Context) ([]domain.WorkTemplate, error) {
	return nil, nil
}
func (r *fakeContentRepo) GetWork(_ context.Context, id string) (domain.WorkTemplate, bool, error) {
	work, ok := r.works[id]
	return work, ok, nil
}
