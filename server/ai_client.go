package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
)

type AIClient interface {
	ChatCompletion(baseURL, apiKey string, req openAIChatRequest) (string, error)
	StreamChatCompletion(baseURL, apiKey string, req openAIChatRequest, onDelta func(string)) (string, error)
}

type openAISDKClient struct {
	httpFallback AIClient
}

type httpCompatibleAIClient struct{}

type openAIChatRequest struct {
	Model          string              `json:"model"`
	Messages       []openAIChatMessage `json:"messages"`
	ResponseFormat map[string]string   `json:"response_format,omitempty"`
	Temperature    float64             `json:"temperature,omitempty"`
	MaxTokens      int                 `json:"max_tokens,omitempty"`
}

type openAIChatMessage struct {
	Role    string              `json:"role"`
	Content []openAIMessagePart `json:"content,omitempty"`
}

type openAIMessagePart struct {
	Type     string                `json:"type"`
	Text     string                `json:"text,omitempty"`
	ImageURL *openAIImageURLHolder `json:"image_url,omitempty"`
}

type openAIImageURLHolder struct {
	URL string `json:"url"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

type openAIModelsResponse struct {
	Data  []aiProviderModel `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// newDefaultAIClient selects SDK mode by default and keeps an HTTP fallback for compatibility.
func newDefaultAIClient() AIClient {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("CONFIG_GENERATOR_AI_CLIENT_MODE")))
	switch mode {
	case "", "sdk":
		fmt.Printf("[AI] client mode=sdk\n")
		return &openAISDKClient{
			httpFallback: &httpCompatibleAIClient{},
		}
	case "http":
		fmt.Printf("[AI] client mode=http\n")
		return &httpCompatibleAIClient{}
	default:
		fmt.Printf("[AI] unknown client mode=%s, fallback to sdk\n", mode)
		return &openAISDKClient{
			httpFallback: &httpCompatibleAIClient{},
		}
	}
}

// ChatCompletion sends a non-streaming chat/completions request via openai-go.
func (c *openAISDKClient) ChatCompletion(baseURL, apiKey string, reqBody openAIChatRequest) (string, error) {
	fmt.Printf("[AI] client=sdk request baseUrl=%s model=%s hasApiKey=%t summary=%s\n",
		redactBaseURL(baseURL), reqBody.Model, strings.TrimSpace(apiKey) != "", summarizeChatRequestForLog(reqBody))

	params, err := buildSDKChatCompletionParams(reqBody)
	if err != nil {
		if c.httpFallback != nil {
			fmt.Printf("[AI] sdk request build failed, fallback to http: %v\n", err)
			return c.httpFallback.ChatCompletion(baseURL, apiKey, reqBody)
		}
		return "", err
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(normalizeBaseURL(baseURL)),
		option.WithHTTPClient(&http.Client{Timeout: 90 * time.Second}),
	)

	resp, err := client.Chat.Completions.New(context.Background(), params)
	if err != nil {
		return "", err
	}
	fmt.Printf("[AI] client=sdk response choices=%d\n", len(resp.Choices))

	if len(resp.Choices) == 0 {
		return "", errors.New("AI 接口未返回有效结果")
	}
	if reason := strings.TrimSpace(resp.Choices[0].FinishReason); reason == "length" || reason == "max_tokens" {
		fmt.Printf("[AI] client=sdk response truncated finish_reason=%s\n", reason)
		return "", errors.New("AI 输出被截断，请减少一次生成内容，或提高模型输出上限后重试")
	}

	content := resp.Choices[0].Message.Content
	if strings.TrimSpace(content) == "" {
		if refusal := strings.TrimSpace(resp.Choices[0].Message.Refusal); refusal != "" {
			return "", errors.New(refusal)
		}
		return "", errors.New("AI 接口未返回有效内容")
	}
	fmt.Printf("[AI] client=sdk response finishReason=%s contentBytes=%d preview=%q\n",
		resp.Choices[0].FinishReason, len(content), previewLogText(content, 160))

	return content, nil
}

// StreamChatCompletion streams assistant deltas from openai-go and aggregates the final response text.
func (c *openAISDKClient) StreamChatCompletion(baseURL, apiKey string, reqBody openAIChatRequest, onDelta func(string)) (string, error) {
	fmt.Printf("[AI] client=sdk stream request baseUrl=%s model=%s hasApiKey=%t summary=%s\n",
		redactBaseURL(baseURL), reqBody.Model, strings.TrimSpace(apiKey) != "", summarizeChatRequestForLog(reqBody))

	params, err := buildSDKChatCompletionParams(reqBody)
	if err != nil {
		if c.httpFallback != nil {
			fmt.Printf("[AI] sdk stream request build failed, fallback to http: %v\n", err)
			return c.httpFallback.StreamChatCompletion(baseURL, apiKey, reqBody, onDelta)
		}
		return "", err
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(normalizeBaseURL(baseURL)),
		// Avoid hanging forever when provider side blocks without returning stream chunks.
		option.WithHTTPClient(&http.Client{Timeout: 120 * time.Second}),
	)

	stream := client.Chat.Completions.NewStreaming(context.Background(), params)
	defer stream.Close()

	var builder strings.Builder
	refusal := ""
	finishReason := ""

	for stream.Next() {
		chunk := stream.Current()
		for _, choice := range chunk.Choices {
			if delta := choice.Delta.Content; delta != "" {
				builder.WriteString(delta)
				if onDelta != nil {
					onDelta(delta)
				}
			}
			if refusal == "" && strings.TrimSpace(choice.Delta.Refusal) != "" {
				refusal = choice.Delta.Refusal
			}
			if finishReason == "" && strings.TrimSpace(choice.FinishReason) != "" {
				finishReason = strings.TrimSpace(choice.FinishReason)
			}
		}
	}

	if err := stream.Err(); err != nil {
		fmt.Printf("[AI] client=sdk stream error model=%s err=%v\n", reqBody.Model, err)
		return "", err
	}
	fmt.Printf("[AI] client=sdk stream response bytes=%d finish_reason=%s preview=%q\n",
		builder.Len(), finishReason, previewLogText(builder.String(), 160))

	if finishReason == "length" || finishReason == "max_tokens" {
		return "", errors.New("AI 输出被截断，请减少一次生成内容，或提高模型输出上限后重试")
	}

	content := builder.String()
	if strings.TrimSpace(content) == "" {
		if strings.TrimSpace(refusal) != "" {
			return "", errors.New(strings.TrimSpace(refusal))
		}
		return "", errors.New("AI 接口未返回有效内容")
	}

	return content, nil
}

// buildSDKChatCompletionParams converts the internal OpenAI-compatible request to SDK parameters.
func buildSDKChatCompletionParams(reqBody openAIChatRequest) (openai.ChatCompletionNewParams, error) {
	if len(reqBody.Messages) == 0 {
		return openai.ChatCompletionNewParams{}, errors.New("AI 璇锋眰娑堟伅涓嶈兘涓虹┖")
	}

	params := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(reqBody.Model),
		Messages: make([]openai.ChatCompletionMessageParamUnion, 0, len(reqBody.Messages)),
	}

	for _, message := range reqBody.Messages {
		converted, err := buildSDKChatMessage(message)
		if err != nil {
			return openai.ChatCompletionNewParams{}, err
		}
		params.Messages = append(params.Messages, converted)
	}

	if reqBody.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(reqBody.MaxTokens))
	}
	if reqBody.Temperature != 0 {
		params.Temperature = openai.Float(reqBody.Temperature)
	}

	switch strings.ToLower(strings.TrimSpace(reqBody.ResponseFormat["type"])) {
	case "":
	case "json_object":
		params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
		}
	case "text":
		params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
			OfText: &shared.ResponseFormatTextParam{},
		}
	default:
		return openai.ChatCompletionNewParams{}, fmt.Errorf("SDK 鏆備笉鏀寔鐨?response_format 绫诲瀷: %s", reqBody.ResponseFormat["type"])
	}

	return params, nil
}

// buildSDKChatMessage maps one message role/content pair to the SDK union structure.
func buildSDKChatMessage(message openAIChatMessage) (openai.ChatCompletionMessageParamUnion, error) {
	role := strings.ToLower(strings.TrimSpace(message.Role))
	switch role {
	case "developer":
		content, err := joinTextOnlyMessageContent(message.Content, role)
		if err != nil {
			return openai.ChatCompletionMessageParamUnion{}, err
		}
		return openai.ChatCompletionMessageParamUnion{
			OfDeveloper: &openai.ChatCompletionDeveloperMessageParam{
				Content: openai.ChatCompletionDeveloperMessageParamContentUnion{
					OfString: openai.String(content),
				},
			},
		}, nil
	case "system":
		content, err := joinTextOnlyMessageContent(message.Content, role)
		if err != nil {
			return openai.ChatCompletionMessageParamUnion{}, err
		}
		return openai.ChatCompletionMessageParamUnion{
			OfSystem: &openai.ChatCompletionSystemMessageParam{
				Content: openai.ChatCompletionSystemMessageParamContentUnion{
					OfString: openai.String(content),
				},
			},
		}, nil
	case "assistant":
		content, err := joinTextOnlyMessageContent(message.Content, role)
		if err != nil {
			return openai.ChatCompletionMessageParamUnion{}, err
		}
		return openai.ChatCompletionMessageParamUnion{
			OfAssistant: &openai.ChatCompletionAssistantMessageParam{
				Content: openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: openai.String(content),
				},
			},
		}, nil
	case "user":
		content, err := buildSDKUserMessageContent(message.Content)
		if err != nil {
			return openai.ChatCompletionMessageParamUnion{}, err
		}
		return openai.ChatCompletionMessageParamUnion{
			OfUser: &openai.ChatCompletionUserMessageParam{
				Content: content,
			},
		}, nil
	default:
		return openai.ChatCompletionMessageParamUnion{}, fmt.Errorf("SDK 鏆備笉鏀寔鐨勬秷鎭鑹? %s", message.Role)
	}
}

// buildSDKUserMessageContent converts text/image parts to SDK user content parts.
func buildSDKUserMessageContent(parts []openAIMessagePart) (openai.ChatCompletionUserMessageParamContentUnion, error) {
	if len(parts) == 0 {
		return openai.ChatCompletionUserMessageParamContentUnion{
			OfString: openai.String(""),
		}, nil
	}

	allText := true
	for _, part := range parts {
		partType := normalizeMessagePartType(part.Type)
		if partType != "text" {
			allText = false
			break
		}
	}

	if allText {
		content, err := joinTextOnlyMessageContent(parts, "user")
		if err != nil {
			return openai.ChatCompletionUserMessageParamContentUnion{}, err
		}
		return openai.ChatCompletionUserMessageParamContentUnion{
			OfString: openai.String(content),
		}, nil
	}

	converted := make([]openai.ChatCompletionContentPartUnionParam, 0, len(parts))
	for _, part := range parts {
		partType := normalizeMessagePartType(part.Type)
		switch partType {
		case "text":
			converted = append(converted, openai.ChatCompletionContentPartUnionParam{
				OfText: &openai.ChatCompletionContentPartTextParam{
					Text: part.Text,
				},
			})
		case "image_url":
			if part.ImageURL == nil || strings.TrimSpace(part.ImageURL.URL) == "" {
				return openai.ChatCompletionUserMessageParamContentUnion{}, errors.New("鍥剧墖娑堟伅缂哄皯 image_url")
			}
			converted = append(converted, openai.ChatCompletionContentPartUnionParam{
				OfImageURL: &openai.ChatCompletionContentPartImageParam{
					ImageURL: openai.ChatCompletionContentPartImageImageURLParam{
						URL:    strings.TrimSpace(part.ImageURL.URL),
						Detail: "auto",
					},
				},
			})
		default:
			return openai.ChatCompletionUserMessageParamContentUnion{}, fmt.Errorf("SDK 鏆備笉鏀寔鐨勬秷鎭唴瀹圭被鍨? %s", part.Type)
		}
	}

	return openai.ChatCompletionUserMessageParamContentUnion{
		OfArrayOfContentParts: converted,
	}, nil
}

// joinTextOnlyMessageContent joins text-only parts and rejects non-text content for strict roles.
func joinTextOnlyMessageContent(parts []openAIMessagePart, role string) (string, error) {
	if len(parts) == 0 {
		return "", nil
	}

	texts := make([]string, 0, len(parts))
	for _, part := range parts {
		partType := normalizeMessagePartType(part.Type)
		if partType != "text" {
			return "", fmt.Errorf("瑙掕壊 %s 鏆備笉鏀寔闈炴枃鏈唴瀹? %s", role, part.Type)
		}
		texts = append(texts, part.Text)
	}
	return strings.Join(texts, "\n\n"), nil
}

func normalizeMessagePartType(raw string) string {
	partType := strings.ToLower(strings.TrimSpace(raw))
	if partType == "" {
		return "text"
	}
	return partType
}

// listCompatibleAIModels retrieves model metadata from an OpenAI-compatible /models endpoint.
func listCompatibleAIModels(baseURL, apiKey string) ([]aiProviderModel, error) {
	fmt.Printf("[AI] models request baseUrl=%s hasApiKey=%t\n", redactBaseURL(baseURL), strings.TrimSpace(apiKey) != "")
	req, err := http.NewRequest(http.MethodGet, normalizeBaseURL(baseURL)+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed openAIModelsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		if resp.StatusCode >= http.StatusBadRequest {
			return nil, fmt.Errorf("AI 鎺ュ彛杩斿洖閿欒: %s", strings.TrimSpace(string(body)))
		}
		return nil, err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
			return nil, errors.New(parsed.Error.Message)
		}
		return nil, fmt.Errorf("AI 妯″瀷鍒楄〃鑾峰彇澶辫触: HTTP %d", resp.StatusCode)
	}

	models := make([]aiProviderModel, 0, len(parsed.Data))
	seen := make(map[string]struct{}, len(parsed.Data))
	for _, model := range parsed.Data {
		model.ID = strings.TrimSpace(model.ID)
		if model.ID == "" {
			continue
		}
		if _, exists := seen[model.ID]; exists {
			continue
		}
		seen[model.ID] = struct{}{}
		models = append(models, model)
	}
	sort.SliceStable(models, func(i, j int) bool {
		return models[i].ID < models[j].ID
	})
	if len(models) == 0 {
		fmt.Printf("[AI] models response count=0\n")
	} else {
		fmt.Printf("[AI] models response count=%d first=%s\n", len(models), models[0].ID)
	}
	return models, nil
}

// ChatCompletion sends a raw OpenAI-compatible HTTP request and parses the standard response envelope.
func (c *httpCompatibleAIClient) ChatCompletion(baseURL, apiKey string, reqBody openAIChatRequest) (string, error) {
	fmt.Printf("[AI] client=http request baseUrl=%s model=%s hasApiKey=%t summary=%s\n",
		redactBaseURL(baseURL), reqBody.Model, strings.TrimSpace(apiKey) != "", summarizeChatRequestForLog(reqBody))
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	fmt.Printf("[AI] client=http request payloadBytes=%d\n", len(payload))

	req, err := http.NewRequest(http.MethodPost, normalizeBaseURL(baseURL)+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	fmt.Printf("[AI] client=http response http=%d bytes=%d\n", resp.StatusCode, len(body))

	var parsed openAIChatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		if resp.StatusCode >= http.StatusBadRequest {
			return "", fmt.Errorf("AI 接口返回错误: %s", strings.TrimSpace(string(body)))
		}
		return "", err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
			return "", errors.New(parsed.Error.Message)
		}
		return "", fmt.Errorf("AI 接口调用失败: HTTP %d", resp.StatusCode)
	}

	if len(parsed.Choices) == 0 {
		return "", errors.New("AI 接口未返回有效结果")
	}
	if reason := strings.TrimSpace(parsed.Choices[0].FinishReason); reason == "length" || reason == "max_tokens" {
		fmt.Printf("[AI] client=http response truncated finish_reason=%s\n", reason)
		return "", errors.New("AI 输出被截断，请减少一次生成内容，或提高模型输出上限后重试")
	}

	content := parsed.Choices[0].Message.Content
	fmt.Printf("[AI] client=http response finishReason=%s contentBytes=%d preview=%q\n",
		parsed.Choices[0].FinishReason, len(content), previewLogText(content, 160))
	return content, nil
}

// StreamChatCompletion provides a compatibility stream path by returning one final delta from HTTP mode.
func (c *httpCompatibleAIClient) StreamChatCompletion(baseURL, apiKey string, reqBody openAIChatRequest, onDelta func(string)) (string, error) {
	content, err := c.ChatCompletion(baseURL, apiKey, reqBody)
	if err != nil {
		return "", err
	}
	if onDelta != nil && strings.TrimSpace(content) != "" {
		onDelta(content)
	}
	return content, nil
}
