package server

import (
	"bytes"
	"os"
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
		aiSkillsDir:    filepath.Join(root, "data", "ai", "skills"),
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
	if err := server.validateDraftTemplate("ZK_NODES={{ join \",\" (serviceIPs \"zookeeper\") }}\n"); err != nil {
		t.Fatalf("expected template with join to be valid, got error: %v", err)
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

func TestLoadAISessionRestoresAttachmentStoredPath(t *testing.T) {
	server := newTestServer(t)
	session, err := server.createAISession()
	if err != nil {
		t.Fatalf("createAISession error: %v", err)
	}

	attachment, err := server.saveUploadedAttachment(session, "notes.txt", "text/plain", []byte("hello"))
	if err != nil {
		t.Fatalf("saveUploadedAttachment error: %v", err)
	}
	expectedPath := attachment.StoredPath
	if err := server.saveAISession(session); err != nil {
		t.Fatalf("saveAISession error: %v", err)
	}

	reloaded, err := server.loadAISession(session.ID)
	if err != nil {
		t.Fatalf("loadAISession error: %v", err)
	}
	if len(reloaded.Attachments) != 1 {
		t.Fatalf("unexpected attachment count: %d", len(reloaded.Attachments))
	}
	if reloaded.Attachments[0].StoredPath == "" {
		t.Fatal("expected restored StoredPath, got empty string")
	}
	if reloaded.Attachments[0].StoredPath != expectedPath {
		t.Fatalf("unexpected restored StoredPath: %s", reloaded.Attachments[0].StoredPath)
	}
	if _, err := os.Stat(reloaded.Attachments[0].StoredPath); err != nil {
		t.Fatalf("expected restored attachment file to exist, got err=%v", err)
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

func TestListAISessionsAndUpdateTitle(t *testing.T) {
	server := newTestServer(t)

	first, err := server.createAISession()
	if err != nil {
		t.Fatalf("createAISession first error: %v", err)
	}
	first.Messages = append(first.Messages, aiSessionMessage{
		ID:        "msg_1",
		Role:      "user",
		Content:   "帮我生成 elasticsearch 安装模板",
		CreatedAt: nowRFC3339(),
	})
	if err := server.saveAISession(first); err != nil {
		t.Fatalf("saveAISession first error: %v", err)
	}

	second, err := server.createAISession()
	if err != nil {
		t.Fatalf("createAISession second error: %v", err)
	}
	if err := server.updateAISessionTitle(second, "Spark3 模板"); err != nil {
		t.Fatalf("updateAISessionTitle error: %v", err)
	}

	summaries, err := server.listAISessions()
	if err != nil {
		t.Fatalf("listAISessions error: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("unexpected session count: %d", len(summaries))
	}

	titleByID := map[string]string{}
	for _, summary := range summaries {
		titleByID[summary.ID] = summary.Title
	}
	if titleByID[first.ID] == "" || !strings.Contains(titleByID[first.ID], "elasticsearch") {
		t.Fatalf("unexpected derived title: %q", titleByID[first.ID])
	}
	if titleByID[second.ID] != "Spark3 模板" {
		t.Fatalf("unexpected updated title: %q", titleByID[second.ID])
	}
}

func TestDeleteDraftFiles(t *testing.T) {
	server := newTestServer(t)
	session, err := server.createAISession()
	if err != nil {
		t.Fatalf("createAISession error: %v", err)
	}

	_, _, err = server.saveDraftFiles([]aiDraftFile{
		{
			Path:    "templates/elasticsearch/start.sh.tmpl",
			Content: "#!/bin/bash\nRUN_USER={{ .Global.user }}\n",
		},
		{
			Path:    "templates/elasticsearch/stop.sh.tmpl",
			Content: "#!/bin/bash\nRUN_USER={{ .Global.user }}\n",
		},
	})
	if err != nil {
		t.Fatalf("saveDraftFiles error: %v", err)
	}

	session.DraftFiles = []aiDraftFile{
		{Path: "templates/elasticsearch/start.sh.tmpl", Content: "a", NeedsReview: true},
		{Path: "templates/elasticsearch/stop.sh.tmpl", Content: "b", NeedsReview: true},
	}
	session.PlannedActions = []aiPlannedAction{
		{Type: "write_file", Path: "templates/elasticsearch/start.sh.tmpl"},
		{Type: "write_file", Path: "templates/elasticsearch/stop.sh.tmpl"},
	}
	session.Messages = []aiSessionMessage{
		{
			ID:         "msg_1",
			Role:       "assistant",
			Content:    "ok",
			CreatedAt:  nowRFC3339(),
			DraftFiles: append([]aiDraftFile(nil), session.DraftFiles...),
		},
	}
	if err := server.saveAISession(session); err != nil {
		t.Fatalf("saveAISession error: %v", err)
	}

	deletedDrafts, deletedTemplates, err := server.deleteDraftFiles(session, []string{"templates/elasticsearch/start.sh.tmpl"}, true)
	if err != nil {
		t.Fatalf("deleteDraftFiles error: %v", err)
	}
	if len(deletedDrafts) != 1 || deletedDrafts[0] != "templates/elasticsearch/start.sh.tmpl" {
		t.Fatalf("unexpected deleted drafts: %#v", deletedDrafts)
	}
	if len(deletedTemplates) != 1 || deletedTemplates[0] != "templates/elasticsearch/start.sh.tmpl" {
		t.Fatalf("unexpected deleted templates: %#v", deletedTemplates)
	}
	if len(session.DraftFiles) != 1 || session.DraftFiles[0].Path != "templates/elasticsearch/stop.sh.tmpl" {
		t.Fatalf("unexpected remaining drafts: %#v", session.DraftFiles)
	}
	if _, err := os.Stat(filepath.Join(server.templatesDir, "elasticsearch", "start.sh.tmpl")); !os.IsNotExist(err) {
		t.Fatalf("expected deleted template file to be removed, got err=%v", err)
	}
}

func TestDeleteAISession(t *testing.T) {
	server := newTestServer(t)
	session, err := server.createAISession()
	if err != nil {
		t.Fatalf("createAISession error: %v", err)
	}

	if err := os.MkdirAll(server.uploadDir(session.ID), 0755); err != nil {
		t.Fatalf("mkdir upload dir error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(server.uploadDir(session.ID), "example.txt"), []byte("ok"), 0644); err != nil {
		t.Fatalf("write upload file error: %v", err)
	}

	if err := server.deleteAISession(session.ID); err != nil {
		t.Fatalf("deleteAISession error: %v", err)
	}
	if _, err := os.Stat(server.sessionPath(session.ID)); !os.IsNotExist(err) {
		t.Fatalf("expected session file removed, got err=%v", err)
	}
	if _, err := os.Stat(server.uploadDir(session.ID)); !os.IsNotExist(err) {
		t.Fatalf("expected upload dir removed, got err=%v", err)
	}
}

func TestDeletePendingAttachment(t *testing.T) {
	server := newTestServer(t)
	session, err := server.createAISession()
	if err != nil {
		t.Fatalf("createAISession error: %v", err)
	}

	attachment, err := server.saveUploadedAttachment(session, "image.png", "image/png", []byte{0x89, 'P', 'N', 'G'})
	if err != nil {
		t.Fatalf("saveUploadedAttachment error: %v", err)
	}
	if err := server.deletePendingAttachment(session, attachment.ID); err != nil {
		t.Fatalf("deletePendingAttachment error: %v", err)
	}
	if len(session.Attachments) != 0 {
		t.Fatalf("expected attachment list to be empty, got %d", len(session.Attachments))
	}
	if _, err := os.Stat(attachment.StoredPath); !os.IsNotExist(err) {
		t.Fatalf("expected attachment file removed, got err=%v", err)
	}
}

func TestValidatePlannedActionsAliases(t *testing.T) {
	actions, err := validatePlannedActions([]aiPlannedAction{
		{Type: "create", Path: "templates/elasticsearch"},
		{Type: "create", Path: "templates/elasticsearch/install.sh.tmpl"},
		{Type: "create_dir", Path: "templates/elasticsearch/conf"},
		{Type: "write", Path: "templates/elasticsearch/elasticsearch.yml.tmpl"},
		{Type: "create_template", Path: "templates/elasticsearch/start.sh.tmpl"},
		{Type: "render_template", Path: "templates/elasticsearch/stop.sh.tmpl"},
		{Type: "create_folder", Path: "templates/elasticsearch/scripts"},
	})
	if err != nil {
		t.Fatalf("validatePlannedActions returned error: %v", err)
	}
	if len(actions) != 7 {
		t.Fatalf("unexpected action count: %d", len(actions))
	}
	if actions[0].Type != "mkdir" {
		t.Fatalf("expected first action to normalize to mkdir, got %s", actions[0].Type)
	}
	if actions[1].Type != "write_file" {
		t.Fatalf("expected second action to normalize to write_file, got %s", actions[1].Type)
	}
	if actions[2].Type != "mkdir" {
		t.Fatalf("expected third action to normalize to mkdir, got %s", actions[2].Type)
	}
	if actions[3].Type != "write_file" {
		t.Fatalf("expected fourth action to normalize to write_file, got %s", actions[3].Type)
	}
	if actions[4].Type != "write_file" {
		t.Fatalf("expected fifth action to normalize to write_file, got %s", actions[4].Type)
	}
	if actions[5].Type != "write_file" {
		t.Fatalf("expected sixth action to normalize to write_file, got %s", actions[5].Type)
	}
	if actions[6].Type != "mkdir" {
		t.Fatalf("expected seventh action to normalize to mkdir, got %s", actions[6].Type)
	}
}

func TestValidatePlannedActionsIgnoresControlActions(t *testing.T) {
	actions, err := validatePlannedActions([]aiPlannedAction{
		{Type: "confirm_draft", Path: "", Reason: "等待人工确认"},
		{Type: "review", Path: ".", Reason: "仅表示流程状态"},
		{Type: "write_file", Path: "templates/elasticsearch/install.sh.tmpl"},
	})
	if err != nil {
		t.Fatalf("validatePlannedActions returned error: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected only one actionable item after filtering control actions, got %d", len(actions))
	}
	if actions[0].Type != "write_file" {
		t.Fatalf("expected remaining action to be write_file, got %s", actions[0].Type)
	}
}

func TestParseAIModelResponseSupportsCodeFence(t *testing.T) {
	raw := "```json\n{\"assistantMessage\":\"ok\",\"draftFiles\":[],\"plannedActions\":[],\"warnings\":[],\"followUpQuestions\":[],\"configPatch\":{\"serviceTop\":{},\"serverConfig\":{}}}\n```"
	result, err := parseAIModelResponse(raw)
	if err != nil {
		t.Fatalf("parseAIModelResponse returned error: %v", err)
	}
	if result.AssistantMessage != "ok" {
		t.Fatalf("unexpected assistant message: %s", result.AssistantMessage)
	}
}

func TestParseAIModelResponseSupportsWrappedJSON(t *testing.T) {
	raw := "下面是结果，请审核后保存：\n\n{\"assistantMessage\":\"ok\",\"draftFiles\":[],\"plannedActions\":[],\"warnings\":[],\"followUpQuestions\":[],\"configPatch\":{\"serviceTop\":{},\"serverConfig\":{}}}\n\n说明结束"
	result, err := parseAIModelResponse(raw)
	if err != nil {
		t.Fatalf("parseAIModelResponse returned error: %v", err)
	}
	if result.AssistantMessage != "ok" {
		t.Fatalf("unexpected assistant message: %s", result.AssistantMessage)
	}
}

func TestParseAIModelResponseRepairsDraftContent(t *testing.T) {
	raw := "{\n" +
		"  \"assistantMessage\": \"ok\",\n" +
		"  \"draftFiles\": [\n" +
		"    {\n" +
		"      \"path\": \"templates/elasticsearch/install.sh.tmpl\",\n" +
		"      \"content\": \"#!/bin/bash\n" +
		"echo \"hello\"\n" +
		"mkdir -p /data/es\n" +
		"\",\n" +
		"      \"reason\": \"generated\"\n" +
		"    }\n" +
		"  ],\n" +
		"  \"plannedActions\": [\n" +
		"    {\"type\": \"create\", \"path\": \"templates/elasticsearch\"},\n" +
		"  ],\n" +
		"  \"warnings\": [],\n" +
		"  \"followUpQuestions\": [],\n" +
		"  \"configPatch\": {\"serviceTop\": {}, \"serverConfig\": {}},\n" +
		"}"

	result, err := parseAIModelResponse(raw)
	if err != nil {
		t.Fatalf("parseAIModelResponse returned error: %v", err)
	}
	if len(result.DraftFiles) != 1 {
		t.Fatalf("unexpected draft file count: %d", len(result.DraftFiles))
	}
	if !strings.Contains(result.DraftFiles[0].Content, "echo \"hello\"") {
		t.Fatalf("unexpected repaired draft content: %s", result.DraftFiles[0].Content)
	}
}
