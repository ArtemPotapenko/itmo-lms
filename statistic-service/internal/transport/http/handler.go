package httptransport

import (
	"net/http"

	"itmo-lms/pkg/platform"
	"itmo-lms/statistic-service/internal/application"
	"itmo-lms/statistic-service/internal/domain"
)

type Handler struct {
	service *application.Service
	secret  string
}

func New(service *application.Service, secret string) *Handler {
	return &Handler{service: service, secret: secret}
}

func (h *Handler) Routes() http.Handler {
	mux := platform.NewMux("statistic-service")
	mux.Handle("POST /attempts", platform.RequireRoles(h.secret, []string{"student", "teacher", "admin"}, http.HandlerFunc(h.createAttempt)))
	mux.Handle("GET /users/{id}/attempts", platform.RequireAuth(h.secret, http.HandlerFunc(h.listAttempts)))
	mux.Handle("GET /users/{id}/knowledge-profile", platform.RequireAuth(h.secret, http.HandlerFunc(h.profile)))
	mux.Handle("GET /courses/{id}/calibration", platform.RequireAuth(h.secret, http.HandlerFunc(h.courseCalibration)))
	mux.Handle("GET /internal/users/{id}/knowledge-profile", http.HandlerFunc(h.profile))
	mux.Handle("GET /internal/courses/{id}/calibration", http.HandlerFunc(h.courseCalibration))
	return mux
}

func (h *Handler) createAttempt(w http.ResponseWriter, r *http.Request) {
	var req domain.Attempt
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	item, err := h.service.CreateAttempt(r.Context(), req)
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	platform.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) listAttempts(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListAttempts(r.Context(), r.PathValue("id"))
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	platform.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) profile(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Profile(r.Context(), r.PathValue("id"))
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	platform.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) courseCalibration(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.CourseCalibration(r.Context(), r.PathValue("id"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	platform.WriteJSON(w, http.StatusOK, item)
}
