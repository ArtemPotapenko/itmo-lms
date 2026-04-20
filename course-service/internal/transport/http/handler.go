package httptransport

import (
	"errors"
	"net/http"

	"itmo-lms/course-service/internal/application"
	"itmo-lms/course-service/internal/domain"
	"itmo-lms/pkg/platform"
)

type Handler struct {
	service *application.Service
	secret  string
}

func New(service *application.Service, secret string) *Handler {
	return &Handler{service: service, secret: secret}
}

func (h *Handler) Routes() http.Handler {
	mux := platform.NewMux("course-service")
	mux.Handle("POST /courses", platform.RequireRoles(h.secret, []string{"teacher", "admin"}, http.HandlerFunc(h.createCourse)))
	mux.Handle("GET /courses", platform.RequireAuth(h.secret, http.HandlerFunc(h.listCourses)))
	mux.Handle("POST /courses/{id}/members", platform.RequireRoles(h.secret, []string{"teacher", "admin"}, http.HandlerFunc(h.addMember)))
	mux.Handle("GET /courses/{id}/members", platform.RequireAuth(h.secret, http.HandlerFunc(h.listMembers)))
	mux.Handle("POST /courses/{id}/assignments", platform.RequireRoles(h.secret, []string{"teacher", "admin"}, http.HandlerFunc(h.createAssignment)))
	mux.Handle("GET /courses/{id}/assignments", platform.RequireAuth(h.secret, http.HandlerFunc(h.listAssignments)))
	mux.Handle("POST /assignments/{id}/submissions", platform.RequireRoles(h.secret, []string{"student", "teacher", "admin"}, http.HandlerFunc(h.createSubmission)))
	mux.Handle("GET /assignments/{id}/submissions", platform.RequireRoles(h.secret, []string{"teacher", "admin"}, http.HandlerFunc(h.listSubmissions)))
	mux.Handle("POST /submissions/{id}/review", platform.RequireRoles(h.secret, []string{"teacher", "admin"}, http.HandlerFunc(h.reviewSubmission)))
	return mux
}

func (h *Handler) createCourse(w http.ResponseWriter, r *http.Request) {
	var req domain.Course
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	item, err := h.service.CreateCourse(r.Context(), req)
	writeResult(w, item, err)
}
func (h *Handler) listCourses(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListCourses(r.Context())
	writeList(w, items, err)
}
func (h *Handler) addMember(w http.ResponseWriter, r *http.Request) {
	var req domain.CourseMember
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	item, err := h.service.AddMember(r.Context(), r.PathValue("id"), req)
	writeResult(w, item, err)
}
func (h *Handler) listMembers(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListMembers(r.Context(), r.PathValue("id"))
	writeList(w, items, err)
}
func (h *Handler) createAssignment(w http.ResponseWriter, r *http.Request) {
	var req domain.Assignment
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	item, err := h.service.CreateAssignment(r.Context(), r.PathValue("id"), req)
	writeResult(w, item, err)
}
func (h *Handler) listAssignments(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListAssignments(r.Context(), r.PathValue("id"))
	writeList(w, items, err)
}
func (h *Handler) createSubmission(w http.ResponseWriter, r *http.Request) {
	var req domain.Submission
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	item, err := h.service.CreateSubmission(r.Context(), r.PathValue("id"), req)
	writeResult(w, item, err)
}
func (h *Handler) listSubmissions(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListSubmissions(r.Context(), r.PathValue("id"))
	writeList(w, items, err)
}
func (h *Handler) reviewSubmission(w http.ResponseWriter, r *http.Request) {
	var req domain.TeacherReview
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	item, err := h.service.ReviewSubmission(r.Context(), r.PathValue("id"), req)
	writeResult(w, item, err)
}

func writeResult[T any](w http.ResponseWriter, value T, err error) {
	if err == nil {
		platform.WriteJSON(w, http.StatusCreated, value)
		return
	}
	writeError(w, err)
}

func writeList[T any](w http.ResponseWriter, value []T, err error) {
	if err == nil {
		platform.WriteJSON(w, http.StatusOK, value)
		return
	}
	writeError(w, err)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, application.ErrNotFound) {
		status = http.StatusNotFound
	}
	platform.WriteError(w, status, err.Error())
}
