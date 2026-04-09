package server

import (
	"net/http"
	"net/http/httptest"
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
