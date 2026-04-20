package httptransport

import (
	"errors"
	"net/http"
	"slices"
	"strings"

	"itmo-lms/auth-service/internal/application"
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
	mux := platform.NewMux("auth-service")
	mux.HandleFunc("POST /register", h.register)
	mux.HandleFunc("POST /login", h.login)
	mux.Handle("GET /me", platform.RequireAuth(h.secret, http.HandlerFunc(h.me)))
	mux.Handle("GET /users", platform.RequireRoles(h.secret, []string{"teacher", "admin"}, http.HandlerFunc(h.listUsers)))
	return mux
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Phone     string   `json:"phone"`
		Email     string   `json:"email"`
		FirstName string   `json:"first_name"`
		LastName  string   `json:"last_name"`
		Nick      string   `json:"nick"`
		Password  string   `json:"password"`
		Roles     []string `json:"roles"`
	}
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if len(req.Roles) > 0 {
		claims, ok := platform.ClaimsFromContext(r.Context())
		if !ok {
			token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
			claimsRaw, err := platform.ParseToken(h.secret, token)
			if err == nil {
				claims = claimsRaw
				ok = true
			}
		}
		if !ok || !slices.Contains(claims.Roles, "admin") {
			platform.WriteError(w, http.StatusForbidden, "admin role required to assign roles")
			return
		}
	}
	user, err := h.service.Register(r.Context(), application.RegisterCommand{
		Phone: req.Phone, Email: req.Email, FirstName: req.FirstName, LastName: req.LastName, Nick: req.Nick, Password: req.Password, Roles: req.Roles,
	})
	if err != nil {
		switch {
		case errors.Is(err, application.ErrConflict):
			platform.WriteError(w, http.StatusConflict, err.Error())
		default:
			platform.WriteError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	platform.WriteJSON(w, http.StatusCreated, user)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	token, user, err := h.service.Login(r.Context(), req.Phone, req.Password)
	if err != nil {
		if errors.Is(err, application.ErrUnauthorized) {
			platform.WriteError(w, http.StatusUnauthorized, err.Error())
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	platform.WriteJSON(w, http.StatusOK, map[string]any{"access_token": token, "token_type": "Bearer", "user": user})
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	claims, _ := platform.ClaimsFromContext(r.Context())
	user, err := h.service.Me(r.Context(), claims.Subject)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, application.ErrNotFound) {
			status = http.StatusNotFound
		}
		platform.WriteError(w, status, err.Error())
		return
	}
	platform.WriteJSON(w, http.StatusOK, user)
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.List(r.Context())
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	platform.WriteJSON(w, http.StatusOK, users)
}
