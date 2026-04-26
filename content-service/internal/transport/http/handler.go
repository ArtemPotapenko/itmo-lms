package httptransport

import (
	"errors"
	"net/http"

	"itmo-lms/content-service/internal/application"
	"itmo-lms/content-service/internal/domain"
	"itmo-lms/pkg/events"
	"itmo-lms/pkg/platform"
)

type Handler struct {
	service *application.Service
	secret  string
	publish AttemptEventPublisher
}

type AttemptEventPublisher interface {
	PublishAttempt(r *http.Request, event events.AttemptEvaluated) error
}

func New(service *application.Service, secret string, publish AttemptEventPublisher) *Handler {
	return &Handler{service: service, secret: secret, publish: publish}
}

func (h *Handler) Routes() http.Handler {
	mux := platform.NewMux("content-service")
	mux.Handle("POST /topics", platform.RequireRoles(h.secret, []string{"teacher", "admin"}, http.HandlerFunc(h.createTopic)))
	mux.HandleFunc("GET /topics", h.listTopics)
	mux.Handle("POST /tags", platform.RequireRoles(h.secret, []string{"teacher", "admin"}, http.HandlerFunc(h.createTag)))
	mux.HandleFunc("GET /tags", h.listTags)
	mux.Handle("POST /tasks", platform.RequireRoles(h.secret, []string{"teacher", "admin"}, http.HandlerFunc(h.createTask)))
	mux.Handle("POST /tasks/scoped", platform.RequireRoles(h.secret, []string{"teacher", "admin"}, http.HandlerFunc(h.createTasksScoped)))
	mux.HandleFunc("GET /tasks", h.listTasks)
	mux.HandleFunc("GET /tasks/{id}", h.getTask)
	mux.Handle("POST /tasks/{id}/check", platform.RequireAuth(h.secret, http.HandlerFunc(h.checkTask)))
	mux.Handle("POST /theory", platform.RequireRoles(h.secret, []string{"teacher", "admin"}, http.HandlerFunc(h.createTheory)))
	mux.HandleFunc("GET /theory", h.listTheory)
	mux.Handle("POST /work-templates", platform.RequireRoles(h.secret, []string{"teacher", "admin"}, http.HandlerFunc(h.createWork)))
	mux.HandleFunc("GET /work-templates", h.listWorks)
	mux.HandleFunc("GET /work-templates/{id}", h.getWork)
	mux.HandleFunc("GET /work-templates/{id}/latex", h.workLatex)
	mux.Handle("POST /work-templates/{id}/check", platform.RequireAuth(h.secret, http.HandlerFunc(h.checkWork)))
	mux.Handle("POST /work-templates/{id}/documents", platform.RequireRoles(h.secret, []string{"teacher", "admin"}, http.HandlerFunc(h.workDocument)))
	mux.HandleFunc("GET /learning-path", h.learningPath)
	return mux
}

func (h *Handler) createTopic(w http.ResponseWriter, r *http.Request) {
	var req domain.Topic
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	item, err := h.service.CreateTopic(r.Context(), req)
	writeResult(w, item, err)
}

func (h *Handler) listTopics(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListTopics(r.Context())
	writeResult(w, items, err)
}

func (h *Handler) createTag(w http.ResponseWriter, r *http.Request) {
	var req domain.Tag
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	item, err := h.service.CreateTag(r.Context(), req)
	writeResult(w, item, err)
}

func (h *Handler) listTags(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListTags(r.Context())
	writeResult(w, items, err)
}

func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	var req domain.Task
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	item, err := h.service.CreateTask(r.Context(), req)
	writeResult(w, item, err)
}

func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListTasks(r.Context(), r.URL.Query().Get("topic_id"))
	writeResult(w, items, err)
}

func (h *Handler) getTask(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetTask(r.Context(), r.PathValue("id"))
	writeResult(w, item, err)
}

func (h *Handler) checkTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		CourseID string `json:"course_id"`
		Answer   string `json:"answer"`
		Source   string `json:"source"`
	}
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	item, err := h.service.CheckTask(r.Context(), r.PathValue("id"), req.Answer)
	if err == nil && h.publish != nil && req.UserID != "" {
		h.publishAttempt(r, req.UserID, req.CourseID, r.PathValue("id"), req.Answer, req.Source, item)
	}
	writeResult(w, item, err)
}

func (h *Handler) createTasksScoped(w http.ResponseWriter, r *http.Request) {
	var req domain.TaskScope
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	items, err := h.service.CreateTasksInScope(r.Context(), req)
	writeResult(w, items, err)
}

func (h *Handler) createTheory(w http.ResponseWriter, r *http.Request) {
	var req domain.Theory
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	item, err := h.service.CreateTheory(r.Context(), req)
	writeResult(w, item, err)
}

func (h *Handler) listTheory(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListTheory(r.Context(), r.URL.Query().Get("topic_id"))
	writeResult(w, items, err)
}

func (h *Handler) createWork(w http.ResponseWriter, r *http.Request) {
	var req domain.WorkTemplate
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	item, err := h.service.CreateWork(r.Context(), req)
	writeResult(w, item, err)
}

func (h *Handler) listWorks(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListWorks(r.Context())
	writeResult(w, items, err)
}

func (h *Handler) getWork(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetWork(r.Context(), r.PathValue("id"))
	writeResult(w, item, err)
}

func (h *Handler) workLatex(w http.ResponseWriter, r *http.Request) {
	body, err := h.service.BuildWorkLatex(r.Context(), r.PathValue("id"))
	if err != nil {
		writeResult(w, nil, err)
		return
	}
	w.Header().Set("Content-Type", "text/x-tex; charset=utf-8")
	_, _ = w.Write([]byte(body))
}

func (h *Handler) workDocument(w http.ResponseWriter, r *http.Request) {
	jobID, err := h.service.BuildWorkDocument(r.Context(), r.PathValue("id"))
	if err != nil {
		writeResult(w, nil, err)
		return
	}
	platform.WriteJSON(w, http.StatusCreated, map[string]string{"job_id": jobID})
}

func (h *Handler) checkWork(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string              `json:"user_id"`
		CourseID string              `json:"course_id"`
		Source   string              `json:"source"`
		Answers  []domain.WorkAnswer `json:"answers"`
	}
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	result, err := h.service.CheckWork(r.Context(), r.PathValue("id"), req.UserID, req.Source, req.Answers)
	if err != nil {
		writeResult(w, nil, err)
		return
	}
	if h.publish != nil && req.UserID != "" {
		for _, item := range result.Results {
			_ = h.publish.PublishAttempt(r, events.AttemptEvaluated{
				UserID:    req.UserID,
				CourseID:  req.CourseID,
				ContentID: item.TaskID,
				Answer:    item.Answer,
				IsCorrect: item.IsCorrect,
				Source:    sourceOr(req.Source, "workbook"),
			})
		}
	}
	platform.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) learningPath(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListTasks(r.Context(), r.URL.Query().Get("topic_id"))
	writeResult(w, items, err)
}

func writeResult(w http.ResponseWriter, value any, err error) {
	if err == nil {
		status := http.StatusOK
		switch value.(type) {
		case domain.Topic, domain.Tag, domain.Task, domain.Theory, domain.WorkTemplate:
			status = http.StatusCreated
		}
		platform.WriteJSON(w, status, value)
		return
	}
	status := http.StatusInternalServerError
	if errors.Is(err, application.ErrNotFound) {
		status = http.StatusNotFound
	} else {
		status = http.StatusBadRequest
	}
	platform.WriteError(w, status, err.Error())
}

func (h *Handler) publishAttempt(r *http.Request, userID, courseID, contentID, answer, source string, item map[string]any) {
	isCorrect, _ := item["is_correct"].(bool)
	_ = h.publish.PublishAttempt(r, events.AttemptEvaluated{
		UserID:    userID,
		CourseID:  courseID,
		ContentID: contentID,
		Answer:    answer,
		IsCorrect: isCorrect,
		Source:    sourceOr(source, "practice"),
	})
}

func sourceOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
