package application

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"itmo-lms/content-service/internal/domain"
	"itmo-lms/pkg/platform"
)

var ErrNotFound = errors.New("entity not found")

type Service struct {
	repo     domain.Repository
	compiler DocumentCompiler
}

type DocumentCompiler interface {
	Compile(context.Context, string, []domain.Task) (string, error)
}

func NewService(repo domain.Repository, compiler DocumentCompiler) *Service {
	return &Service{repo: repo, compiler: compiler}
}

func (s *Service) CreateTopic(ctx context.Context, topic domain.Topic) (domain.Topic, error) {
	if strings.TrimSpace(topic.Title) == "" {
		return domain.Topic{}, errors.New("title is required")
	}
	topic.ID = platform.NewID("top")
	topic.Status = valueOr(topic.Status, "published")
	topic.CreatedAt = time.Now().UTC()
	return topic, s.repo.CreateTopic(ctx, topic)
}

func (s *Service) ListTopics(ctx context.Context) ([]domain.Topic, error) {
	return s.repo.ListTopics(ctx)
}

func (s *Service) CreateTag(ctx context.Context, tag domain.Tag) (domain.Tag, error) {
	if strings.TrimSpace(tag.Name) == "" {
		return domain.Tag{}, errors.New("name is required")
	}
	if strings.TrimSpace(tag.Code) == "" {
		return domain.Tag{}, errors.New("code is required")
	}
	tag.ID = platform.NewID("tag")
	tag.Kind = valueOr(tag.Kind, "skill")
	tag.Status = valueOr(tag.Status, "active")
	tag.CreatedAt = time.Now().UTC()
	return tag, s.repo.CreateTag(ctx, tag)
}

func (s *Service) ListTags(ctx context.Context) ([]domain.Tag, error) {
	return s.repo.ListTags(ctx)
}

func (s *Service) CreateTask(ctx context.Context, task domain.Task) (domain.Task, error) {
	if strings.TrimSpace(task.Title) == "" || strings.TrimSpace(task.LatexBody) == "" {
		return domain.Task{}, errors.New("title and latex_body are required")
	}
	totalWeight := 0.0
	for i := range task.Tags {
		if task.Tags[i].TagID == "" {
			return domain.Task{}, errors.New("tag_id is required")
		}
		tag, ok, err := s.repo.GetTag(ctx, task.Tags[i].TagID)
		if err != nil {
			return domain.Task{}, err
		}
		if !ok {
			return domain.Task{}, fmt.Errorf("unknown tag_id: %s", task.Tags[i].TagID)
		}
		if task.Tags[i].Weight <= 0 {
			return domain.Task{}, fmt.Errorf("weight must be positive for tag_id: %s", task.Tags[i].TagID)
		}
		task.Tags[i].Code = tag.Code
		task.Tags[i].Name = tag.Name
		task.Tags[i].Kind = tag.Kind
		totalWeight += task.Tags[i].Weight
	}
	if len(task.Tags) > 0 && totalWeight <= 0 {
		return domain.Task{}, errors.New("tag weights must be positive")
	}
	now := time.Now().UTC()
	task.ID = platform.NewID("tsk")
	task.Status = valueOr(task.Status, "published")
	task.CreatedAt = now
	task.UpdatedAt = now
	return task, s.repo.CreateTask(ctx, task)
}

func (s *Service) ListTasks(ctx context.Context, topicID string) ([]domain.Task, error) {
	return s.repo.ListTasks(ctx, topicID)
}

func (s *Service) GetTask(ctx context.Context, id string) (domain.Task, error) {
	task, ok, err := s.repo.GetTask(ctx, id)
	if err != nil {
		return domain.Task{}, err
	}
	if !ok {
		return domain.Task{}, ErrNotFound
	}
	return task, nil
}

func (s *Service) CheckTask(ctx context.Context, id, answer string) (map[string]any, error) {
	task, err := s.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}
	return map[string]any{"content_id": task.ID, "topic_ids": task.TopicIDs, "tags": task.Tags, "is_correct": isCorrectAnswer(task.CorrectAnswer, answer)}, nil
}

func (s *Service) CreateTheory(ctx context.Context, theory domain.Theory) (domain.Theory, error) {
	if strings.TrimSpace(theory.Title) == "" || strings.TrimSpace(theory.LatexBody) == "" {
		return domain.Theory{}, errors.New("title and latex_body are required")
	}
	now := time.Now().UTC()
	theory.ID = platform.NewID("thr")
	theory.Status = valueOr(theory.Status, "published")
	theory.CreatedAt = now
	theory.UpdatedAt = now
	return theory, s.repo.CreateTheory(ctx, theory)
}

func (s *Service) ListTheory(ctx context.Context, topicID string) ([]domain.Theory, error) {
	return s.repo.ListTheory(ctx, topicID)
}

func (s *Service) GetTheory(ctx context.Context, id string) (domain.Theory, error) {
	theory, ok, err := s.repo.GetTheory(ctx, id)
	if err != nil {
		return domain.Theory{}, err
	}
	if !ok {
		return domain.Theory{}, ErrNotFound
	}
	return theory, nil
}

func (s *Service) CreateWork(ctx context.Context, work domain.WorkTemplate) (domain.WorkTemplate, error) {
	if strings.TrimSpace(work.Title) == "" || len(work.Items) == 0 {
		return domain.WorkTemplate{}, errors.New("title and items are required")
	}
	for i := range work.Items {
		item := &work.Items[i]
		if item.Kind == "" || item.ContentID == "" {
			return domain.WorkTemplate{}, errors.New("work item kind and content_id are required")
		}
		switch item.Kind {
		case "task":
			task, err := s.GetTask(ctx, item.ContentID)
			if err != nil {
				return domain.WorkTemplate{}, fmt.Errorf("unknown task_id: %s", item.ContentID)
			}
			item.Title = task.Title
		case "theory":
			theory, err := s.GetTheory(ctx, item.ContentID)
			if err != nil {
				return domain.WorkTemplate{}, fmt.Errorf("unknown theory_id: %s", item.ContentID)
			}
			item.Title = theory.Title
		default:
			return domain.WorkTemplate{}, fmt.Errorf("unsupported work item kind: %s", item.Kind)
		}
		if item.Order == 0 {
			item.Order = i + 1
		}
	}
	work.ID = platform.NewID("wrk")
	work.Status = valueOr(work.Status, "published")
	work.CreatedAt = time.Now().UTC()
	return work, s.repo.CreateWork(ctx, work)
}

func (s *Service) ListWorks(ctx context.Context) ([]domain.WorkTemplate, error) {
	return s.repo.ListWorks(ctx)
}

func (s *Service) GetWork(ctx context.Context, id string) (domain.WorkTemplate, error) {
	work, ok, err := s.repo.GetWork(ctx, id)
	if err != nil {
		return domain.WorkTemplate{}, err
	}
	if !ok {
		return domain.WorkTemplate{}, ErrNotFound
	}
	return work, nil
}

func (s *Service) BuildWorkLatex(ctx context.Context, id string) (string, error) {
	work, sections, err := s.workSections(ctx, id)
	if err != nil {
		return "", err
	}
	return BuildWorkbookLatex(work.Title, sections), nil
}

func (s *Service) BuildWorkDocument(ctx context.Context, id string) (string, error) {
	if s.compiler == nil {
		return "", errors.New("document compiler is not configured")
	}
	work, sections, err := s.workSections(ctx, id)
	if err != nil {
		return "", err
	}
	compileUnits := make([]domain.Task, 0, len(sections))
	taskNumber := 0
	for _, section := range sections {
		title := section.Title
		if section.Kind == "theory" {
			title = "Теория. " + title
		} else {
			taskNumber++
		}
		compileUnits = append(compileUnits, domain.Task{ID: fmt.Sprintf("%s_%d", section.Kind, section.Order), Title: title, LatexBody: section.Body})
	}
	return s.compiler.Compile(ctx, work.Title, compileUnits)
}

func (s *Service) CheckWork(ctx context.Context, id, userID, source string, answers []domain.WorkAnswer) (domain.WorkCheckResult, error) {
	work, err := s.GetWork(ctx, id)
	if err != nil {
		return domain.WorkCheckResult{}, err
	}
	answerByTask := make(map[string]string, len(answers))
	for _, answer := range answers {
		answerByTask[answer.TaskID] = answer.Answer
	}
	result := domain.WorkCheckResult{WorkID: work.ID, UserID: userID, CheckedAt: time.Now().UTC()}
	for _, item := range work.Items {
		if item.Kind != "task" {
			continue
		}
		task, err := s.GetTask(ctx, item.ContentID)
		if err != nil {
			return domain.WorkCheckResult{}, err
		}
		answer := answerByTask[task.ID]
		isCorrect := isCorrectAnswer(task.CorrectAnswer, answer)
		result.TotalTasks++
		if isCorrect {
			result.CorrectTasks++
		}
		result.Results = append(result.Results, domain.TaskCheckResult{
			TaskID:    task.ID,
			Title:     task.Title,
			TopicIDs:  task.TopicIDs,
			Tags:      task.Tags,
			Answer:    answer,
			IsCorrect: isCorrect,
		})
	}
	return result, nil
}

func (s *Service) workTasks(ctx context.Context, id string) (domain.WorkTemplate, []domain.Task, error) {
	work, err := s.GetWork(ctx, id)
	if err != nil {
		return domain.WorkTemplate{}, nil, err
	}
	var tasks []domain.Task
	for _, item := range work.Items {
		if item.Kind != "task" {
			continue
		}
		task, err := s.GetTask(ctx, item.ContentID)
		if err == nil {
			tasks = append(tasks, task)
		}
	}
	return work, tasks, nil
}

type workbookSection struct {
	Order int
	Kind  string
	Title string
	Body  string
}

func (s *Service) workSections(ctx context.Context, id string) (domain.WorkTemplate, []workbookSection, error) {
	work, err := s.GetWork(ctx, id)
	if err != nil {
		return domain.WorkTemplate{}, nil, err
	}
	sections := make([]workbookSection, 0, len(work.Items))
	for _, item := range work.Items {
		switch item.Kind {
		case "task":
			task, err := s.GetTask(ctx, item.ContentID)
			if err != nil {
				return domain.WorkTemplate{}, nil, err
			}
			sections = append(sections, workbookSection{Order: item.Order, Kind: item.Kind, Title: task.Title, Body: task.LatexBody})
		case "theory":
			theory, err := s.GetTheory(ctx, item.ContentID)
			if err != nil {
				return domain.WorkTemplate{}, nil, err
			}
			sections = append(sections, workbookSection{Order: item.Order, Kind: item.Kind, Title: theory.Title, Body: theory.LatexBody})
		}
	}
	sort.Slice(sections, func(i, j int) bool {
		if sections[i].Order == sections[j].Order {
			return sections[i].Title < sections[j].Title
		}
		return sections[i].Order < sections[j].Order
	})
	return work, sections, nil
}

func isCorrectAnswer(expected, actual string) bool {
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)
	if expected == "" || actual == "" {
		return false
	}

	expectedNumbers, expectedNumeric := parseNumericAnswer(expected)
	actualNumbers, actualNumeric := parseNumericAnswer(actual)
	if expectedNumeric && actualNumeric {
		return equalNumericAnswers(expectedNumbers, actualNumbers)
	}

	return normalizeTextAnswer(expected) == normalizeTextAnswer(actual)
}

func parseNumericAnswer(value string) ([]float64, bool) {
	parts := splitAnswerTokens(value)
	if len(parts) == 0 {
		return nil, false
	}

	values := make([]float64, 0, len(parts))
	for _, part := range parts {
		number, ok := parseNumericToken(part)
		if !ok {
			return nil, false
		}
		values = append(values, number)
	}
	sort.Float64s(values)
	return values, true
}

func splitAnswerTokens(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		switch r {
		case ',', ';':
			return true
		}
		return unicode.IsSpace(r)
	})
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			out = append(out, field)
		}
	}
	return out
}

func parseNumericToken(value string) (float64, bool) {
	value = strings.TrimSpace(strings.ReplaceAll(value, ",", "."))
	if value == "" {
		return 0, false
	}
	if strings.Count(value, "/") == 1 {
		parts := strings.Split(value, "/")
		left, errLeft := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		right, errRight := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if errLeft != nil || errRight != nil || right == 0 {
			return 0, false
		}
		return left / right, true
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return number, true
}

func equalNumericAnswers(expected, actual []float64) bool {
	if len(expected) != len(actual) {
		return false
	}
	const epsilon = 1e-6
	for i := range expected {
		if math.Abs(expected[i]-actual[i]) > epsilon {
			return false
		}
	}
	return true
}

func normalizeTextAnswer(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.Join(strings.Fields(value), " ")
	return value
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func BuildWorkbookLatex(title string, sections []workbookSection) string {
	var b strings.Builder
	b.WriteString("\\documentclass[12pt,a4paper]{article}\n")
	b.WriteString("\\usepackage[utf8]{inputenc}\n\\usepackage[T2A]{fontenc}\n\\usepackage[russian]{babel}\n")
	b.WriteString("\\usepackage{amsmath,amssymb,geometry}\n\\geometry{margin=2cm}\n")
	b.WriteString("\\begin{document}\n")
	b.WriteString(fmt.Sprintf("\\section*{%s}\n", escapeLatex(title)))
	taskNumber := 0
	for _, section := range sections {
		switch section.Kind {
		case "theory":
			b.WriteString(fmt.Sprintf("\\subsection*{Теория. %s}\n", escapeLatex(section.Title)))
		default:
			taskNumber++
			b.WriteString(fmt.Sprintf("\\subsection*{Задача %d. %s}\n", taskNumber, escapeLatex(section.Title)))
		}
		b.WriteString(section.Body)
		b.WriteString("\n\\vspace{8mm}\n")
	}
	b.WriteString("\\end{document}\n")
	return b.String()
}

func escapeLatex(value string) string {
	replacer := strings.NewReplacer("&", "\\&", "%", "\\%", "$", "\\$", "#", "\\#", "_", "\\_", "{", "\\{", "}", "\\}")
	return replacer.Replace(value)
}
