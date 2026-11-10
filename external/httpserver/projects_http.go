//go:build http

package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/EvilFreelancer/coddy-agent/internal/projects"
)

// projectStores holds one registry per CODDY_HOME. A server reload keeps the
// same home, and two instances over one home must serialize writes rather than
// each keeping an unlocked copy of the same file.
var projectStores sync.Map // string -> *projects.Store

func projectStoreFor(home string) *projects.Store {
	home = strings.TrimSpace(home)
	if v, ok := projectStores.Load(home); ok {
		return v.(*projects.Store)
	}
	v, _ := projectStores.LoadOrStore(home, projects.New(home))
	return v.(*projects.Store)
}

func (s *Server) projectStore() *projects.Store {
	home := ""
	if cfg := s.activeCfg(); cfg != nil {
		home = cfg.Paths.Home
	}
	return projectStoreFor(home)
}

func (s *Server) registerProjectRoutes() {
	s.mux.HandleFunc("GET /coddy/projects", s.coddyProjectsList)
	s.mux.HandleFunc("POST /coddy/projects", s.coddyProjectsCreate)
	s.mux.HandleFunc("PATCH /coddy/projects/{id}", s.coddyProjectsUpdate)
	s.mux.HandleFunc("DELETE /coddy/projects/{id}", s.coddyProjectsDelete)
}

func (s *Server) coddyProjectsList(w http.ResponseWriter, r *http.Request) {
	rows, err := s.projectStore().List()
	if err != nil {
		s.log.Error("coddy projects list", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": map[string]string{"message": "could not read projects"},
		})
		return
	}
	if rows == nil {
		rows = []projects.Project{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"projects": rows})
}

func (s *Server) coddyProjectsCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path  string `json:"path"`
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": map[string]string{"message": "invalid JSON body"},
		})
		return
	}
	row, err := s.projectStore().Create(req.Path, req.Name, req.Color)
	if err != nil {
		s.writeProjectError(w, err, "could not create project")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"project": row})
}

func (s *Server) coddyProjectsUpdate(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.NotFound(w, r)
		return
	}
	var req struct {
		Name   *string `json:"name"`
		Path   *string `json:"path"`
		Color  *string `json:"color"`
		Pinned *bool   `json:"pinned"`
		Order  *int    `json:"order"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": map[string]string{"message": "invalid JSON body"},
		})
		return
	}
	row, err := s.projectStore().Update(id, projects.Patch{
		Name:   req.Name,
		Path:   req.Path,
		Color:  req.Color,
		Pinned: req.Pinned,
		Order:  req.Order,
	})
	if err != nil {
		s.writeProjectError(w, err, "could not update project")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"project": row})
}

func (s *Server) coddyProjectsDelete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.NotFound(w, r)
		return
	}
	if err := s.projectStore().Delete(id); err != nil {
		s.writeProjectError(w, err, "could not delete project")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func (s *Server) writeProjectError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, projects.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]interface{}{
			"error": map[string]string{"message": "project not found"},
		})
	case errors.Is(err, projects.ErrExists):
		writeJSON(w, http.StatusConflict, map[string]interface{}{
			"error": map[string]string{"message": "a project already uses this folder"},
		})
	case errors.Is(err, projects.ErrInvalidPath):
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": map[string]string{"message": "path must be an existing folder"},
		})
	default:
		s.log.Error("coddy projects", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": map[string]string{"message": fallback},
		})
	}
}
