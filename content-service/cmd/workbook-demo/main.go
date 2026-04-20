package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"itmo-lms/content-service/internal/application"
	"itmo-lms/content-service/internal/domain"
)

type fragments struct {
	Work     domain.WorkTemplate `json:"work"`
	Theories []domain.Theory     `json:"theories"`
	Tasks    []domain.Task       `json:"tasks"`
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fail(err)
	}
	inputPath := filepath.Join(root, "examples", "workbook-fragments.json")
	outputDir := filepath.Join(root, "examples", "generated")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		fail(err)
	}

	raw, err := os.ReadFile(inputPath)
	if err != nil {
		fail(err)
	}

	var demo fragments
	if err := json.Unmarshal(raw, &demo); err != nil {
		fail(err)
	}

	repo := newRepo()
	for _, theory := range demo.Theories {
		repo.theories[theory.ID] = theory
	}
	for _, task := range demo.Tasks {
		repo.tasks[task.ID] = task
	}
	repo.works[demo.Work.ID] = demo.Work

	service := application.NewService(repo, nil)
	latex, err := service.BuildWorkLatex(context.Background(), demo.Work.ID)
	if err != nil {
		fail(err)
	}

	texPath := filepath.Join(outputDir, "workbook-from-fragments.tex")
	if err := os.WriteFile(texPath, []byte(latex), 0o644); err != nil {
		fail(err)
	}

	if compiler, ok := latexCompiler(); ok {
		cmd := exec.Command(compiler, "-interaction=nonstopmode", "-halt-on-error", "-output-directory", outputDir, texPath)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			fail(fmt.Errorf("%s failed: %w: %s", compiler, err, string(out)))
		}
		fmt.Printf("generated:\n- %s\n- %s\n", texPath, filepath.Join(outputDir, "workbook-from-fragments.pdf"))
		return
	}

	fmt.Printf("generated:\n- %s\n", texPath)
	fmt.Println("pdf was not generated because pdflatex/xelatex is not installed")
}

func latexCompiler() (string, bool) {
	for _, name := range []string{"pdflatex", "xelatex"} {
		path, err := exec.LookPath(name)
		if err == nil {
			return path, true
		}
	}
	return "", false
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

type repo struct {
	tasks    map[string]domain.Task
	theories map[string]domain.Theory
	works    map[string]domain.WorkTemplate
}

func newRepo() *repo {
	return &repo{
		tasks:    map[string]domain.Task{},
		theories: map[string]domain.Theory{},
		works:    map[string]domain.WorkTemplate{},
	}
}

func (r *repo) CreateTopic(context.Context, domain.Topic) error { return nil }

func (r *repo) ListTopics(context.Context) ([]domain.Topic, error) { return nil, nil }

func (r *repo) CreateTag(context.Context, domain.Tag) error { return nil }

func (r *repo) ListTags(context.Context) ([]domain.Tag, error) { return nil, nil }

func (r *repo) GetTag(context.Context, string) (domain.Tag, bool, error) {
	return domain.Tag{}, false, nil
}

func (r *repo) CreateTask(_ context.Context, task domain.Task) error {
	r.tasks[task.ID] = task
	return nil
}

func (r *repo) ListTasks(_ context.Context, _ string) ([]domain.Task, error) { return nil, nil }

func (r *repo) GetTask(_ context.Context, id string) (domain.Task, bool, error) {
	task, ok := r.tasks[id]
	return task, ok, nil
}

func (r *repo) CreateTheory(_ context.Context, theory domain.Theory) error {
	r.theories[theory.ID] = theory
	return nil
}

func (r *repo) ListTheory(_ context.Context, _ string) ([]domain.Theory, error) { return nil, nil }

func (r *repo) GetTheory(_ context.Context, id string) (domain.Theory, bool, error) {
	theory, ok := r.theories[id]
	return theory, ok, nil
}

func (r *repo) CreateWork(_ context.Context, work domain.WorkTemplate) error {
	if work.CreatedAt.IsZero() {
		work.CreatedAt = time.Now().UTC()
	}
	r.works[work.ID] = work
	return nil
}

func (r *repo) ListWorks(_ context.Context) ([]domain.WorkTemplate, error) { return nil, nil }

func (r *repo) GetWork(_ context.Context, id string) (domain.WorkTemplate, bool, error) {
	work, ok := r.works[id]
	return work, ok, nil
}
