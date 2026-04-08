package server

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	root := t.TempDir()
	return &Server{
		templatesDir:   filepath.Join(root, "templates"),
		aiDir:          filepath.Join(root, "data", "ai"),
		aiSessionsDir:  filepath.Join(root, "data", "ai", "sessions"),
		aiUploadsDir:   filepath.Join(root, "data", "ai", "uploads"),
		aiSettingsPath: filepath.Join(root, "data", "ai", "settings.json"),
		aiRulesPath:    filepath.Join(root, "data", "ai", "template_rules.md"),
	}
}

func TestEncryptDecryptWithKey(t *testing.T) {
	key := bytes.Repeat([]byte("a"), 32)
	ciphertext, err := encryptWithKey(key, "secret-key")
	if err != nil {
		t.Fatalf("encryptWithKey returned error: %v", err)
	}
	if ciphertext == "" || ciphertext == "secret-key" {
		t.Fatalf("unexpected ciphertext: %q", ciphertext)
	}

	plaintext, err := decryptWithKey(key, ciphertext)
	if err != nil {
		t.Fatalf("decryptWithKey returned error: %v", err)
	}
	if plaintext != "secret-key" {
		t.Fatalf("unexpected plaintext: %q", plaintext)
	}
}

func TestNormalizeTemplatePath(t *testing.T) {
	server := newTestServer(t)

	normalized, fullPath, err := server.normalizeTemplatePath("templates/elasticsearch/install.sh.tmpl")
	if err != nil {
		t.Fatalf("normalizeTemplatePath returned error: %v", err)
	}
	if normalized != "templates/elasticsearch/install.sh.tmpl" {
		t.Fatalf("unexpected normalized path: %s", normalized)
	}
	if !strings.HasSuffix(filepath.ToSlash(fullPath), "/templates/elasticsearch/install.sh.tmpl") {
		t.Fatalf("unexpected full path: %s", fullPath)
	}

	if _, _, err := server.normalizeTemplatePath("../outside.sh.tmpl"); err == nil {
		t.Fatal("expected path traversal to fail")
	}

	if _, _, err := server.normalizeTemplatePath("templates/elasticsearch/install.sh"); err == nil {
		t.Fatal("expected non-tmpl file to fail")
	}
}

func TestValidateDraftTemplate(t *testing.T) {
	server := newTestServer(t)

	if err := server.validateDraftTemplate("RUN_USER={{ .Global.user }}\n"); err != nil {
		t.Fatalf("expected valid template, got error: %v", err)
	}

	if err := server.validateDraftTemplate("{{ if .Global.user }}"); err == nil {
		t.Fatal("expected invalid template to fail")
	}
}

func TestSaveUploadedAttachmentValidation(t *testing.T) {
	server := newTestServer(t)
	session := &aiSession{ID: "session_1"}

	textData := []byte("echo hello")
	attachment, err := server.saveUploadedAttachment(session, "install.sh", "text/plain", textData)
	if err != nil {
		t.Fatalf("saveUploadedAttachment returned error: %v", err)
	}
	if attachment.Kind != "text" || !attachment.Pending {
		t.Fatalf("unexpected attachment: %+v", attachment)
	}
	if got := len(session.Attachments); got != 1 {
		t.Fatalf("expected 1 attachment, got %d", got)
	}

	imageData := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	if _, err := server.saveUploadedAttachment(session, "screen.png", "image/png", imageData); err != nil {
		t.Fatalf("expected png upload to succeed: %v", err)
	}

	if _, err := server.saveUploadedAttachment(session, "bad.exe", "application/octet-stream", []byte("123")); err == nil {
		t.Fatal("expected unsupported file type to fail")
	}
}

func TestSaveDraftFiles(t *testing.T) {
	server := newTestServer(t)

	savedFiles, createdDirs, err := server.saveDraftFiles([]aiDraftFile{
		{
			Path:    "templates/elasticsearch/install.sh.tmpl",
			Content: "#!/bin/bash\nRUN_USER={{ .Global.user }}\n",
		},
	})
	if err != nil {
		t.Fatalf("saveDraftFiles returned error: %v", err)
	}
	if len(savedFiles) != 1 || savedFiles[0] != "templates/elasticsearch/install.sh.tmpl" {
		t.Fatalf("unexpected saved files: %#v", savedFiles)
	}
	if len(createdDirs) != 1 || createdDirs[0] != "templates/elasticsearch" {
		t.Fatalf("unexpected created dirs: %#v", createdDirs)
	}

	if _, _, err := server.saveDraftFiles([]aiDraftFile{{
		Path:    "templates/elasticsearch/install.sh.tmpl",
		Content: "{{ if .Global.user }}",
	}}); err == nil {
		t.Fatal("expected invalid template content to fail")
	}
}
