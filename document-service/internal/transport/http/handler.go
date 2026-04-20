package httptransport

import (
	"errors"
	"net/http"

	"itmo-lms/document-service/internal/application"
	"itmo-lms/document-service/internal/domain"
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
	mux := platform.NewMux("document-service")
	mux.Handle("POST /documents/compile", platform.RequireRoles(h.secret, []string{"teacher", "admin"}, http.HandlerFunc(h.compile)))
	mux.Handle("GET /documents/{id}", platform.RequireAuth(h.secret, http.HandlerFunc(h.getJob)))
	mux.Handle("GET /documents/{id}/download", platform.RequireAuth(h.secret, http.HandlerFunc(h.download)))
	return mux
}

func (h *Handler) compile(w http.ResponseWriter, r *http.Request) {
	var req domain.CompileRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	item, err := h.service.Compile(r.Context(), req)
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	platform.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) getJob(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	platform.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	http.ServeFile(w, r, h.service.FilePath(item))
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, application.ErrNotFound) {
		status = http.StatusNotFound
	}
	platform.WriteError(w, status, err.Error())
}
