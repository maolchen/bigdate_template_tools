package server

import (
	"errors"
	"strings"
	"testing"
)

func TestExtractUndefinedTemplateFunctions(t *testing.T) {
	err := errors.New(`template: draft:5: function "joinx" not defined; template: draft:9: function "fooBar" not defined`)
	names := extractUndefinedTemplateFunctions(err)
	if len(names) != 2 {
		t.Fatalf("unexpected function count: %d", len(names))
	}
	if names[0] != "fooBar" || names[1] != "joinx" {
		t.Fatalf("unexpected function names: %#v", names)
	}
}

func TestAppendDraftValidationWarningsForUndefinedFunction(t *testing.T) {
	server := &Server{}
	warnings := server.appendDraftValidationWarnings([]aiDraftFile{
		{
			Path:    "templates/elasticsearch/README.md.tmpl",
			Content: `{{ joinx "," (serviceIPs "elasticsearch") }}`,
		},
	}, nil)

	if len(warnings) != 1 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if !strings.Contains(warnings[0], "joinx") {
		t.Fatalf("expected undefined function warning, got: %s", warnings[0])
	}
}

func TestAppendDraftValidationWarningsForValidTemplate(t *testing.T) {
	server := &Server{}
	warnings := server.appendDraftValidationWarnings([]aiDraftFile{
		{
			Path:    "templates/elasticsearch/install.sh.tmpl",
			Content: `RUN_USER={{ .Global.user }}`,
		},
	}, nil)
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got: %#v", warnings)
	}
}

func TestTemplateFunctionWhitelistSummaryIncludesJoin(t *testing.T) {
	summary := templateFunctionWhitelistSummary()
	if !strings.Contains(summary, "join") {
		t.Fatalf("expected summary to include join, got: %s", summary)
	}
	if !strings.Contains(summary, "serviceEndpoints") {
		t.Fatalf("expected summary to include serviceEndpoints, got: %s", summary)
	}
}

func TestBuildAIChatRequestUsesTextModeForGeneralQuestion(t *testing.T) {
	server := newTestServer(t)
	req, err := server.buildAIChatRequest(&aiSession{}, aiSessionMessageRequest{
		Message: "你是什么模型",
	}, nil)
	if err != nil {
		t.Fatalf("buildAIChatRequest error: %v", err)
	}
	if got := req.ResponseFormat["type"]; got != "text" {
		t.Fatalf("expected text response format, got %q", got)
	}
	if len(req.Messages) == 0 || !strings.Contains(req.Messages[0].Content[0].Text, "直接回答用户问题") {
		t.Fatalf("expected general-chat system prompt, got %#v", req.Messages)
	}
}

func TestBuildAIChatRequestUsesJSONModeForTemplateQuestion(t *testing.T) {
	server := newTestServer(t)
	req, err := server.buildAIChatRequest(&aiSession{}, aiSessionMessageRequest{
		Message: "请生成 elasticsearch 安装脚本模板",
	}, nil)
	if err != nil {
		t.Fatalf("buildAIChatRequest error: %v", err)
	}
	if got := req.ResponseFormat["type"]; got != "json_object" {
		t.Fatalf("expected json_object response format, got %q", got)
	}
	if len(req.Messages) == 0 || !strings.Contains(req.Messages[0].Content[0].Text, "Follow the selected skill modules") {
		t.Fatalf("expected structured template system prompt, got %#v", req.Messages)
	}
}
