package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildSDKChatCompletionParamsSupportsVisionUserMessage(t *testing.T) {
	params, err := buildSDKChatCompletionParams(openAIChatRequest{
		Model: "qwen-vl-compatible",
		Messages: []openAIChatMessage{
			newTextMessage("system", "Return JSON only."),
			{
				Role: "user",
				Content: []openAIMessagePart{
					{Type: "text", Text: "请识别图片中的脚本内容"},
					{
						Type: "image_url",
						ImageURL: &openAIImageURLHolder{
							URL: "data:image/png;base64,ZmFrZQ==",
						},
					},
				},
			},
		},
		ResponseFormat: map[string]string{"type": "json_object"},
		MaxTokens:      512,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if string(params.Model) != "qwen-vl-compatible" {
		t.Fatalf("unexpected model: %s", params.Model)
	}
	if len(params.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(params.Messages))
	}
	if params.ResponseFormat.OfJSONObject == nil {
		t.Fatalf("expected json_object response format, got %#v", params.ResponseFormat)
	}

	user := params.Messages[1].OfUser
	if user == nil {
		t.Fatal("expected second message to be user message")
	}
	if len(user.Content.OfArrayOfContentParts) != 2 {
		t.Fatalf("expected 2 content parts, got %d", len(user.Content.OfArrayOfContentParts))
	}
	if user.Content.OfArrayOfContentParts[1].GetImageURL() == nil {
		t.Fatal("expected image part to be present")
	}
}

func TestBuildSDKChatCompletionParamsRejectsImageInSystemMessage(t *testing.T) {
	_, err := buildSDKChatCompletionParams(openAIChatRequest{
		Model: "test-model",
		Messages: []openAIChatMessage{
			{
				Role: "system",
				Content: []openAIMessagePart{
					{
						Type: "image_url",
						ImageURL: &openAIImageURLHolder{
							URL: "data:image/png;base64,ZmFrZQ==",
						},
					},
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBuildSDKChatCompletionParamsRejectsUnsupportedResponseFormat(t *testing.T) {
	_, err := buildSDKChatCompletionParams(openAIChatRequest{
		Model: "test-model",
		Messages: []openAIChatMessage{
			newTextMessage("user", "ping"),
		},
		ResponseFormat: map[string]string{"type": "xml_object"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListCompatibleAIModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"id": "qwen3.5-plus", "ownedBy": "dashscope"},
				{"id": "MiniMax-M2.5", "ownedBy": "dashscope"},
				{"id": "qwen3.5-plus", "ownedBy": "dashscope"},
				{"id": " "}
			]
		}`))
	}))
	defer server.Close()

	models, err := listCompatibleAIModels(server.URL, "test-key")
	if err != nil {
		t.Fatalf("listCompatibleAIModels returned error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("unexpected model count: %d", len(models))
	}
	if models[0].ID != "MiniMax-M2.5" || models[1].ID != "qwen3.5-plus" {
		t.Fatalf("unexpected models: %#v", models)
	}
}
