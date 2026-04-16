package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDoOpenAIChatCompletionDetectsLengthTruncation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [
				{
					"message": { "content": "{\"assistantMessage\":\"partial\"" },
					"finish_reason": "length"
				}
			]
		}`))
	}))
	defer ts.Close()

	_, err := doOpenAIChatCompletion(ts.URL, "test-key", openAIChatRequest{
		Model: "test-model",
		Messages: []openAIChatMessage{
			newTextMessage("user", "ping"),
		},
	})
	if err == nil {
		t.Fatal("expected truncation error, got nil")
	}
	if err.Error() != "AI 输出被截断，请减少一次生成内容，或提高模型输出上限后重试" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseAIModelResponseFallbackToPlainText(t *testing.T) {
	raw := "我是通用模型，不按 JSON 输出。"
	resp, err := parseAIModelResponse(raw)
	if err != nil {
		t.Fatalf("expected fallback success, got error: %v", err)
	}
	if resp.AssistantMessage != raw {
		t.Fatalf("unexpected assistant message: %q", resp.AssistantMessage)
	}
	if len(resp.DraftFiles) != 0 {
		t.Fatalf("expected no draft files, got %d", len(resp.DraftFiles))
	}
	if len(resp.PlannedActions) != 0 {
		t.Fatalf("expected no planned actions, got %d", len(resp.PlannedActions))
	}
	if len(resp.Warnings) == 0 || !strings.Contains(resp.Warnings[0], "JSON") {
		t.Fatalf("expected JSON fallback warning, got: %#v", resp.Warnings)
	}
}
