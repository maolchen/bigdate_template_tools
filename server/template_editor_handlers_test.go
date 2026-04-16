package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func templateEditorRequest(method, target, body string, principal authPrincipal) (*httptest.ResponseRecorder, *http.Request) {
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	req = req.WithContext(context.WithValue(req.Context(), authContextKey{}, principal))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return httptest.NewRecorder(), req
}

func TestTemplateEditorItemCreateAndDelete(t *testing.T) {
	server := newTestServer(t)
	admin := authPrincipal{Username: "admin", Role: roleAdmin}

	w, req := templateEditorRequest(http.MethodPost, "/api/templates/editor/item", `{"path":"spark3","type":"dir"}`, admin)
	server.handleTemplateEditorItem(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create dir status=%d body=%s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(filepath.Join(server.templatesDir, "spark3")); err != nil {
		t.Fatalf("expected directory created: %v", err)
	}

	w, req = templateEditorRequest(http.MethodPost, "/api/templates/editor/item", `{"path":"spark3/install.sh.tmpl","type":"file","content":"echo {{ .Global.user }}\n"}`, admin)
	server.handleTemplateEditorItem(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create file status=%d body=%s", w.Code, w.Body.String())
	}
	content, err := os.ReadFile(filepath.Join(server.templatesDir, "spark3", "install.sh.tmpl"))
	if err != nil {
		t.Fatalf("expected file created: %v", err)
	}
	if got := string(content); !strings.Contains(got, "{{ .Global.user }}") {
		t.Fatalf("unexpected file content: %q", got)
	}

	w, req = templateEditorRequest(http.MethodDelete, "/api/templates/editor/item?path=spark3", "", admin)
	server.handleTemplateEditorItem(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("delete dir status=%d body=%s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(filepath.Join(server.templatesDir, "spark3")); !os.IsNotExist(err) {
		t.Fatalf("expected directory deleted, got err=%v", err)
	}
}

func TestTemplateEditorItemRejectsNonAdminAndUnsafePath(t *testing.T) {
	server := newTestServer(t)
	user := authPrincipal{Username: "alice", Role: roleUser}
	admin := authPrincipal{Username: "admin", Role: roleAdmin}

	w, req := templateEditorRequest(http.MethodPost, "/api/templates/editor/item", `{"path":"spark3","type":"dir"}`, user)
	server.handleTemplateEditorItem(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden for normal user, got status=%d body=%s", w.Code, w.Body.String())
	}

	w, req = templateEditorRequest(http.MethodPost, "/api/templates/editor/item", `{"path":"../outside.tmpl","type":"file","content":"ok"}`, admin)
	server.handleTemplateEditorItem(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request for traversal, got status=%d body=%s", w.Code, w.Body.String())
	}

	w, req = templateEditorRequest(http.MethodPost, "/api/templates/editor/item", `{"path":"spark3/readme.txt","type":"file","content":"ok"}`, admin)
	server.handleTemplateEditorItem(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request for non-tmpl file, got status=%d body=%s", w.Code, w.Body.String())
	}
}
