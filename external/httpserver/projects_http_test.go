//go:build http

package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func projectRequestBody(t *testing.T, body any) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(b)
}

func decodeProject(t *testing.T, res *http.Response) map[string]any {
	t.Helper()
	defer func() { _ = res.Body.Close() }()
	var out map[string]any
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestProjectsHTTPCRUD(t *testing.T) {
	ts, _ := newInfoTestServer(t, "")
	dir := filepath.Join(t.TempDir(), "workspace")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := http.Post(
		ts.URL+"/coddy/projects",
		"application/json",
		projectRequestBody(t, map[string]string{"path": dir}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST status = %d", res.StatusCode)
	}
	created := decodeProject(t, res)
	project, _ := created["project"].(map[string]any)
	id, _ := project["id"].(string)
	if id == "" {
		t.Fatalf("created project = %+v", created)
	}

	res, err = http.Get(ts.URL + "/coddy/projects")
	if err != nil {
		t.Fatal(err)
	}
	listed := decodeProject(t, res)
	rows, _ := listed["projects"].([]any)
	if len(rows) != 1 {
		t.Fatalf("GET projects = %+v", listed)
	}

	patchReq, err := http.NewRequest(
		http.MethodPatch,
		ts.URL+"/coddy/projects/"+id,
		projectRequestBody(t, map[string]any{"name": "renamed", "pinned": true}),
	)
	if err != nil {
		t.Fatal(err)
	}
	patchReq.Header.Set("Content-Type", "application/json")
	patched, err := http.DefaultClient.Do(patchReq)
	if err != nil {
		t.Fatal(err)
	}
	if patched.StatusCode != http.StatusOK {
		t.Fatalf("PATCH status = %d", patched.StatusCode)
	}
	body := decodeProject(t, patched)
	got, _ := body["project"].(map[string]any)
	if got["name"] != "renamed" || got["pinned"] != true {
		t.Fatalf("patched project = %+v", got)
	}

	req, err := http.NewRequest(
		http.MethodDelete,
		ts.URL+"/coddy/projects/"+id,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	deleted, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if deleted.StatusCode != http.StatusOK {
		t.Fatalf("DELETE status = %d", deleted.StatusCode)
	}
	_ = deleted.Body.Close()

	res, err = http.Get(ts.URL + "/coddy/projects")
	if err != nil {
		t.Fatal(err)
	}
	listed = decodeProject(t, res)
	rows, _ = listed["projects"].([]any)
	if len(rows) != 0 {
		t.Fatalf("after delete = %+v", listed)
	}
}

func TestProjectsHTTPRefusesDuplicateAndMissing(t *testing.T) {
	ts, _ := newInfoTestServer(t, "")
	dir := t.TempDir()

	create := func() *http.Response {
		res, err := http.Post(
			ts.URL+"/coddy/projects",
			"application/json",
			projectRequestBody(t, map[string]string{"path": dir}),
		)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	first := create()
	if first.StatusCode != http.StatusCreated {
		t.Fatalf("first status = %d", first.StatusCode)
	}
	_ = first.Body.Close()
	second := create()
	if second.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate status = %d", second.StatusCode)
	}
	_ = second.Body.Close()

	req, err := http.NewRequest(
		http.MethodPatch,
		ts.URL+"/coddy/projects/missing",
		projectRequestBody(t, map[string]any{"name": "x"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("missing PATCH status = %d", res.StatusCode)
	}
	_ = res.Body.Close()
}

func TestProjectsHTTPRejectsANonFolder(t *testing.T) {
	ts, _ := newInfoTestServer(t, "")
	file := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := http.Post(
		ts.URL+"/coddy/projects",
		"application/json",
		projectRequestBody(t, map[string]string{"path": file}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("file status = %d", res.StatusCode)
	}
	_ = res.Body.Close()
}
