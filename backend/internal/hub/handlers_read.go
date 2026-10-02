package hub

import (
	"errors"
	"log"
	"net/http"
	"strconv"
)

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	q := r.URL.Query().Get("q")
	limit := atoiDefault(r.URL.Query().Get("limit"), 20)

	repos, err := s.store.search(r.Context(), q, limit)
	if err != nil {
		log.Printf("search: %v", err)
		writeErr(w, http.StatusInternalServerError, "search failed")
		return
	}
	writeOK(w, map[string]interface{}{"query": q, "count": len(repos), "items": repos})
}

func (s *Server) listRepos(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	ns := q.Get("namespace")
	limit := atoiDefault(q.Get("limit"), 20)
	offset := atoiDefault(q.Get("offset"), 0)

	repos, err := s.store.listRepos(r.Context(), ns, limit, offset)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list repos failed")
		return
	}
	writeOK(w, map[string]interface{}{"count": len(repos), "items": repos})
}

func (s *Server) getRepo(w http.ResponseWriter, r *http.Request, ns, repo string) {
	detail, err := s.store.getRepo(r.Context(), ns, repo)
	if errors.Is(err, ErrNotFound) {
		writeErr(w, http.StatusNotFound, "repository not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get repo failed")
		return
	}
	writeOK(w, detail)
}

func (s *Server) getRepoTags(w http.ResponseWriter, r *http.Request, ns, repo string) {
	detail, err := s.store.getRepo(r.Context(), ns, repo)
	if errors.Is(err, ErrNotFound) {
		writeErr(w, http.StatusNotFound, "repository not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get tags failed")
		return
	}
	writeOK(w, map[string]interface{}{"count": len(detail.Tags), "items": detail.Tags})
}

func (s *Server) getRepoReadme(w http.ResponseWriter, r *http.Request, ns, repo string) {
	readme, err := s.store.readme(r.Context(), ns, repo)
	if errors.Is(err, ErrNotFound) {
		writeErr(w, http.StatusNotFound, "repository not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get readme failed")
		return
	}
	writeOK(w, map[string]string{"readme": readme})
}

func atoiDefault(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}
