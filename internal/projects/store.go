// Package projects stores the operator's project list: the folders the SPA's
// Projects page shows and the project workspace opens. A project is only a
// name for a workspace path; conversations stay where the session store keeps
// them and are found by their cwd, so deleting a project never touches a chat.
package projects

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/EvilFreelancer/coddy-agent/internal/session"
)

// FileName is the registry file under CODDY_HOME.
const FileName = "projects.json"

var (
	// ErrNotFound names an id that is not in the registry.
	ErrNotFound = errors.New("project not found")
	// ErrExists names a workspace another project already holds.
	ErrExists = errors.New("project already exists")
	// ErrInvalidPath names a path that is not an existing directory.
	ErrInvalidPath = errors.New("project path is not a directory")
)

// Project is one registered workspace.
type Project struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	Color     string `json:"color,omitempty"`
	Pinned    bool   `json:"pinned,omitempty"`
	Order     int    `json:"order"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// Patch changes the editable fields of a project. Nil leaves a field alone.
type Patch struct {
	Name   *string
	Path   *string
	Color  *string
	Pinned *bool
	Order  *int
}

type registryFile struct {
	Version  int       `json:"version"`
	Projects []Project `json:"projects"`
}

// Store is the projects.json file. The zero path is invalid; use New.
type Store struct {
	path string
	mu   sync.Mutex
}

// New returns the registry rooted at a CODDY_HOME directory.
func New(home string) *Store {
	return &Store{path: filepath.Join(strings.TrimSpace(home), FileName)}
}

// Path reports the file the store reads and writes (useful in tests and logs).
func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func (s *Store) loadLocked() (registryFile, error) {
	var f registryFile
	if s.path == "" || s.path == FileName {
		return f, errors.New("projects: empty home")
	}
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return registryFile{Version: 1, Projects: []Project{}}, nil
	}
	if err != nil {
		return f, err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return registryFile{Version: 1, Projects: []Project{}}, nil
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return f, fmt.Errorf("projects: decode %s: %w", s.path, err)
	}
	if f.Version == 0 {
		f.Version = 1
	}
	if f.Projects == nil {
		f.Projects = []Project{}
	}
	return f, nil
}

func (s *Store) saveLocked(f registryFile) error {
	if f.Version == 0 {
		f.Version = 1
	}
	if f.Projects == nil {
		f.Projects = []Project{}
	}
	sortProjects(f.Projects)
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".projects-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

// List returns the registry in display order.
func (s *Store) List() ([]Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	sortProjects(f.Projects)
	return append([]Project(nil), f.Projects...), nil
}

// Create registers a workspace. Name defaults to the folder's base name; a
// color is optional.
func (s *Store) Create(path, name, color string) (Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.loadLocked()
	if err != nil {
		return Project{}, err
	}
	canonical, err := canonicalDir(path)
	if err != nil {
		return Project{}, err
	}
	for _, p := range f.Projects {
		if session.SameWorkspacePath(p.Path, canonical) {
			return Project{}, ErrExists
		}
	}
	name = cleanName(name)
	if name == "" {
		name = projectName(canonical)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	p := Project{
		ID:        newID(f.Projects),
		Name:      name,
		Path:      canonical,
		Color:     strings.TrimSpace(color),
		Order:     nextOrder(f.Projects),
		CreatedAt: now,
		UpdatedAt: now,
	}
	f.Projects = append(f.Projects, p)
	if err := s.saveLocked(f); err != nil {
		return Project{}, err
	}
	return p, nil
}

// Update changes one project and returns the stored row.
func (s *Store) Update(id string, patch Patch) (Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.loadLocked()
	if err != nil {
		return Project{}, err
	}
	idx := -1
	for i := range f.Projects {
		if f.Projects[i].ID == strings.TrimSpace(id) {
			idx = i
			break
		}
	}
	if idx < 0 {
		return Project{}, ErrNotFound
	}
	p := &f.Projects[idx]
	if patch.Path != nil {
		canonical, err := canonicalDir(*patch.Path)
		if err != nil {
			return Project{}, err
		}
		for i, other := range f.Projects {
			if i != idx && session.SameWorkspacePath(other.Path, canonical) {
				return Project{}, ErrExists
			}
		}
		p.Path = canonical
	}
	if patch.Name != nil {
		if name := cleanName(*patch.Name); name != "" {
			p.Name = name
		}
	}
	if patch.Color != nil {
		p.Color = strings.TrimSpace(*patch.Color)
	}
	if patch.Pinned != nil {
		p.Pinned = *patch.Pinned
	}
	if patch.Order != nil {
		p.Order = *patch.Order
	}
	targetID := p.ID
	p.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.saveLocked(f); err != nil {
		return Project{}, err
	}
	// saveLocked sorts the slice in place; p may no longer point at the row
	// after a pin or an order change moved it, so read it back by id.
	for _, row := range f.Projects {
		if row.ID == targetID {
			return row, nil
		}
	}
	return Project{}, ErrNotFound
}

// Delete removes one project. The folder and its conversations are untouched.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.loadLocked()
	if err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	for i := range f.Projects {
		if f.Projects[i].ID == id {
			f.Projects = append(f.Projects[:i], f.Projects[i+1:]...)
			return s.saveLocked(f)
		}
	}
	return ErrNotFound
}

func canonicalDir(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", ErrInvalidPath
	}
	canonical := session.CanonicalWorkspacePath(path)
	info, err := os.Stat(canonical)
	if err != nil || !info.IsDir() {
		return "", ErrInvalidPath
	}
	return canonical, nil
}

func cleanName(name string) string {
	name = strings.TrimSpace(name)
	if len(name) > 120 {
		name = strings.TrimSpace(name[:120])
	}
	return name
}

func projectName(path string) string {
	base := filepath.Base(path)
	if base == "." || base == string(filepath.Separator) || base == "" {
		return path
	}
	return base
}

func nextOrder(rows []Project) int {
	next := 0
	for _, p := range rows {
		if p.Order >= next {
			next = p.Order + 1
		}
	}
	return next
}

func newID(rows []Project) string {
	used := make(map[string]struct{}, len(rows))
	for _, p := range rows {
		used[p.ID] = struct{}{}
	}
	for {
		var b [9]byte
		if _, err := rand.Read(b[:]); err != nil {
			panic("projects: crypto/rand failed: " + err.Error())
		}
		id := "prj_" + hex.EncodeToString(b[:])
		if _, exists := used[id]; !exists {
			return id
		}
	}
}

// sortProjects puts pinned projects first, then the operator's order, then
// the name. Pinning is a second, small list rather than a stripe through all
// projects, so a pinned row is never repeated.
func sortProjects(rows []Project) {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.Pinned != b.Pinned {
			return a.Pinned
		}
		if a.Order != b.Order {
			if a.Order == 0 {
				return false
			}
			if b.Order == 0 {
				return true
			}
			return a.Order < b.Order
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
}
