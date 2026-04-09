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
