package server

import "testing"

func TestExtractAssistantMessagePreview(t *testing.T) {
	raw := `{"assistantMessage":"已生成 elasticsearch 安装脚本模板，正在补充变量建议","draftFiles":[]}`
	preview := extractAssistantMessagePreview(raw)
	if preview != "已生成 elasticsearch 安装脚本模板，正在补充变量建议" {
		t.Fatalf("unexpected preview: %q", preview)
	}
}

func TestExtractAssistantMessagePreviewHandlesEscapes(t *testing.T) {
	raw := "{\"assistantMessage\":\"第一行\\n第二行 \\\"install.sh.tmpl\\\"\",\"draftFiles\":[]}"
	preview := extractAssistantMessagePreview(raw)
	expected := "第一行\n第二行 \"install.sh.tmpl\""
	if preview != expected {
		t.Fatalf("unexpected preview: %q", preview)
	}
}

func TestAssistantPreviewExtractorPush(t *testing.T) {
	extractor := &assistantPreviewExtractor{}

	first := extractor.Push(`{"assistantMessage":"已生成 `)
	second := extractor.Push(`elasticsearch 模板","draftFiles":[]}`)

	if first.AssistantDelta != "已生成 " {
		t.Fatalf("unexpected first chunk: %#v", first)
	}
	if second.AssistantDelta != "elasticsearch 模板" {
		t.Fatalf("unexpected second chunk: %#v", second)
	}
}

func TestExtractDraftFilePreviewsHandlesPartialContent(t *testing.T) {
	raw := `{"assistantMessage":"ok","draftFiles":[{"path":"templates/elasticsearch/install.sh.tmpl","content":"#!/bin/bash\necho hello","reason":"生成安装脚本"}],"plannedActions":[]}`
	previews := extractDraftFilePreviews(raw)
	if len(previews) != 1 {
		t.Fatalf("unexpected preview count: %d", len(previews))
	}
	if previews[0].Path != "templates/elasticsearch/install.sh.tmpl" {
		t.Fatalf("unexpected draft path: %q", previews[0].Path)
	}
	if previews[0].Reason != "生成安装脚本" {
		t.Fatalf("unexpected draft reason: %q", previews[0].Reason)
	}
	if previews[0].Content != "#!/bin/bash\necho hello" {
		t.Fatalf("unexpected draft content: %q", previews[0].Content)
	}
}

func TestAssistantPreviewExtractorStreamsDraftPreview(t *testing.T) {
	extractor := &assistantPreviewExtractor{}

	first := extractor.Push(`{"assistantMessage":"正在生成","draftFiles":[{"path":"templates/elasticsearch/check.sh.tmpl","content":"#!/bin/bash\n`)
	if len(first.DraftFiles) != 1 {
		t.Fatalf("unexpected first draft previews: %#v", first.DraftFiles)
	}
	if first.DraftFiles[0].Content != "#!/bin/bash\n" {
		t.Fatalf("unexpected first draft content: %q", first.DraftFiles[0].Content)
	}

	second := extractor.Push(`echo ok\n","reason":"生成检查脚本"}]}`)
	if len(second.DraftFiles) != 1 {
		t.Fatalf("unexpected second draft previews: %#v", second.DraftFiles)
	}
	if second.DraftFiles[0].Content != "#!/bin/bash\necho ok\n" {
		t.Fatalf("unexpected second draft content: %q", second.DraftFiles[0].Content)
	}
}
