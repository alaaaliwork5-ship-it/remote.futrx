package httphandlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	serviceproject "github.com/futrx-com/remote.futrx.com/internal/service/project"
	"github.com/futrx-com/remote.futrx.com/internal/stores/fileproject"
	"github.com/futrx-com/remote.futrx.com/internal/stores/fileprojectmemory"
)

func TestProjectMemoryRoutes(t *testing.T) {
	handler, project := newMemoryProjectHandler(t)
	base := "/api/projects/" + string(project.ID) + "/memory"

	// Empty memory on first read.
	getReq := httptest.NewRequest(http.MethodGet, base, nil)
	getRec := httptest.NewRecorder()
	handler.HandleResource(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET empty = %d body=%s", getRec.Code, getRec.Body.String())
	}
	var empty serviceproject.Memory
	if err := json.NewDecoder(getRec.Body).Decode(&empty); err != nil {
		t.Fatal(err)
	}
	if empty.Content != "" || empty.Enabled || empty.UpdatedAt != 0 {
		t.Fatalf("empty memory = %+v", empty)
	}

	// Save memory.
	body := `{"content":"Use Go 1.25. Never touch prod.","enabled":true}`
	putReq := httptest.NewRequest(http.MethodPut, base, strings.NewReader(body))
	putRec := httptest.NewRecorder()
	handler.HandleResource(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT = %d body=%s", putRec.Code, putRec.Body.String())
	}
	var saved serviceproject.Memory
	if err := json.NewDecoder(putRec.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if saved.Content != "Use Go 1.25. Never touch prod." || !saved.Enabled || saved.UpdatedAt == 0 {
		t.Fatalf("saved = %+v", saved)
	}

	// Read it back.
	getReq = httptest.NewRequest(http.MethodGet, base, nil)
	getRec = httptest.NewRecorder()
	handler.HandleResource(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET after save = %d body=%s", getRec.Code, getRec.Body.String())
	}
	if err := json.NewDecoder(getRec.Body).Decode(&empty); err != nil {
		t.Fatal(err)
	}
	if empty.Content != saved.Content || !empty.Enabled {
		t.Fatalf("read back = %+v", empty)
	}

	// Disable without changing content.
	body = `{"content":"Use Go 1.25. Never touch prod.","enabled":false}`
	putReq = httptest.NewRequest(http.MethodPut, base, strings.NewReader(body))
	putRec = httptest.NewRecorder()
	handler.HandleResource(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT disabled = %d body=%s", putRec.Code, putRec.Body.String())
	}
	if err := json.NewDecoder(putRec.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if saved.Enabled {
		t.Fatalf("expected disabled, got %+v", saved)
	}

	// Reject oversized memory.
	oversized := `{"content":"` + strings.Repeat("x", serviceproject.MaxMemoryBytes+1) + `","enabled":true}`
	putReq = httptest.NewRequest(http.MethodPut, base, strings.NewReader(oversized))
	putRec = httptest.NewRecorder()
	handler.HandleResource(putRec, putReq)
	if putRec.Code != http.StatusInternalServerError {
		t.Fatalf("PUT oversized = %d body=%s, want 500", putRec.Code, putRec.Body.String())
	}

	// Reject wrong method.
	delReq := httptest.NewRequest(http.MethodDelete, base, nil)
	delRec := httptest.NewRecorder()
	handler.HandleResource(delRec, delReq)
	if delRec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("DELETE = %d, want 405", delRec.Code)
	}
}

func TestProjectMemoryUnavailable(t *testing.T) {
	repo, err := fileproject.NewWithWorkspaceRoot(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	projects := serviceproject.New(repo, serviceproject.ContainerDependencies{}, nil, nil, nil)
	project, err := projects.Create(context.Background(), serviceproject.CreateInput{Name: "No Memory"}, "user@example.com")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewProjectHandler(projects, nil, nil, "remote.futrx.com")
	base := "/api/projects/" + string(project.ID) + "/memory"

	// Reads degrade to empty memory.
	getReq := httptest.NewRequest(http.MethodGet, base, nil)
	getRec := httptest.NewRecorder()
	handler.HandleResource(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET = %d body=%s", getRec.Code, getRec.Body.String())
	}

	// Writes fail closed with 503.
	putReq := httptest.NewRequest(http.MethodPut, base, strings.NewReader(`{"content":"x","enabled":true}`))
	putRec := httptest.NewRecorder()
	handler.HandleResource(putRec, putReq)
	if putRec.Code != http.StatusServiceUnavailable {
		t.Fatalf("PUT = %d body=%s, want 503", putRec.Code, putRec.Body.String())
	}
}

func newMemoryProjectHandler(t *testing.T) (*ProjectHandler, serviceproject.Meta) {
	t.Helper()
	repo, err := fileproject.NewWithWorkspaceRoot(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	memory, err := fileprojectmemory.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	projects := serviceproject.New(repo, serviceproject.ContainerDependencies{}, nil, nil, memory)
	project, err := projects.Create(context.Background(), serviceproject.CreateInput{Name: "Memory Project"}, "user@example.com")
	if err != nil {
		t.Fatal(err)
	}
	return NewProjectHandler(projects, nil, nil, "remote.futrx.com"), project
}
