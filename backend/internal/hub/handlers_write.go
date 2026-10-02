package hub

import (
	"errors"
	"net/http"

	"github.com/LiStudioorg/boxli/internal/auth"
)

// createRepo POST /api/v1/repos（需登录）
func (s *Server) createRepo(w http.ResponseWriter, r *http.Request) {
	u := auth.UserFrom(r.Context())
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in RepoInput
	if !decodeJSON(w, r, &in) {
		return
	}
	// namespace 缺省用登录用户名
	if in.Namespace == "" {
		in.Namespace = u.Username
	}

	detail, err := s.store.createRepo(r.Context(), u.ID, in)
	switch {
	case errors.Is(err, ErrInvalid):
		writeErr(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrConflict):
		writeErr(w, http.StatusConflict, "repository already exists")
	case err != nil:
		writeErr(w, http.StatusInternalServerError, "create repo failed")
	default:
		writeJSON(w, http.StatusCreated, response{Code: 0, Message: "ok", Data: detail})
	}
}

// updateRepo PUT /api/v1/repos/{ns}/{repo}（需 owner）
func (s *Server) updateRepo(w http.ResponseWriter, r *http.Request, ns, repo string) {
	u := auth.UserFrom(r.Context())
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in RepoInput
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Namespace = ns
	in.Name = repo

	detail, err := s.store.updateRepo(r.Context(), u.ID, in)
	switch {
	case errors.Is(err, ErrInvalid):
		writeErr(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNotFound):
		writeErr(w, http.StatusNotFound, "repository not found")
	case errors.Is(err, ErrForbidden):
		writeErr(w, http.StatusForbidden, "not the repository owner")
	case err != nil:
		writeErr(w, http.StatusInternalServerError, "update repo failed")
	default:
		writeOK(w, detail)
	}
}

// deleteRepo DELETE /api/v1/repos/{ns}/{repo}（需 owner）
func (s *Server) deleteRepo(w http.ResponseWriter, r *http.Request, ns, repo string) {
	u := auth.UserFrom(r.Context())
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	err := s.store.deleteRepo(r.Context(), u.ID, ns, repo)
	switch {
	case errors.Is(err, ErrNotFound):
		writeErr(w, http.StatusNotFound, "repository not found")
	case errors.Is(err, ErrForbidden):
		writeErr(w, http.StatusForbidden, "not the repository owner")
	case err != nil:
		writeErr(w, http.StatusInternalServerError, "delete repo failed")
	default:
		writeOK(w, map[string]string{"deleted": ns + "/" + repo})
	}
}
