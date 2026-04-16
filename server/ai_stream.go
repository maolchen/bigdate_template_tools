package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type aiStreamEvent struct {
	Type        string        `json:"type"`
	Message     string        `json:"message,omitempty"`
	Delta       string        `json:"delta,omitempty"`
	DraftFiles  []aiDraftFile `json:"draftFiles,omitempty"`
	PromptTrace aiPromptTrace `json:"promptTrace,omitempty"`
	Session     *aiSession    `json:"session,omitempty"`
}

type assistantPreviewExtractor struct {
	rawBuilder strings.Builder
	visible    string
	draftFiles []aiDraftFile
}

type aiStreamPreview struct {
	AssistantDelta string
	DraftFiles     []aiDraftFile
}

// handleAISessionMessageStream emits SSE events for accepted/status/delta/complete message lifecycle.
func (s *Server) handleAISessionMessageStream(w http.ResponseWriter, r *http.Request, session *aiSession, req aiSessionMessageRequest) {
	fmt.Printf("[AI] stream start session=%s model=%s selectedDraftPaths=%s selectedSkills=%s messagePreview=%q\n",
		session.ID, strings.TrimSpace(req.Model), summarizePathsForLog(req.SelectedDraftPaths, 8),
		summarizePathsForLog(req.SelectedSkillIDs, 8), previewLogText(req.Message, 160))
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	if err := writeSSEEvent(w, flusher, "accepted", aiStreamEvent{
		Type:    "accepted",
		Message: "消息已发送，开始整理模板上下文",
	}); err != nil {
		return
	}

	extractor := &assistantPreviewExtractor{}
	result, _, err := s.runAISessionMessageWithExecutor(session, req, func(progress aiMessageProgress) {
		_ = writeSSEEvent(w, flusher, progress.Type, aiStreamEvent{
			Type:        progress.Type,
			Message:     progress.Message,
			PromptTrace: progress.PromptTrace,
		})
	}, func(baseURL, apiKey string, chatReq openAIChatRequest) (string, error) {
		_ = writeSSEEvent(w, flusher, "status", aiStreamEvent{
			Type:    "status",
			Message: fmt.Sprintf("模型 %s 开始流式生成", strings.TrimSpace(chatReq.Model)),
		})
		return s.aiClient.StreamChatCompletion(baseURL, apiKey, chatReq, func(rawDelta string) {
			preview := extractor.Push(rawDelta)
			if preview.AssistantDelta == "" && len(preview.DraftFiles) == 0 {
				return
			}
			_ = writeSSEEvent(w, flusher, "delta", aiStreamEvent{
				Type:       "delta",
				Delta:      preview.AssistantDelta,
				DraftFiles: preview.DraftFiles,
			})
		})
	})
	if err != nil {
		fmt.Printf("[AI] stream failed session=%s err=%v\n", session.ID, err)
		_ = writeSSEEvent(w, flusher, "error", aiStreamEvent{
			Type:    "error",
			Message: err.Error(),
		})
		return
	}
	fmt.Printf("[AI] stream complete session=%s messages=%d drafts=%d\n",
		session.ID, len(result.Session.Messages), len(result.Session.DraftFiles))

	_ = writeSSEEvent(w, flusher, "complete", aiStreamEvent{
		Type:    "complete",
		Message: "生成完成",
		Session: result.Session,
	})
}

// Push updates assistant preview text and incremental draft-file previews from partial JSON deltas.
func (e *assistantPreviewExtractor) Push(rawDelta string) aiStreamPreview {
	if rawDelta == "" {
		return aiStreamPreview{}
	}
	e.rawBuilder.WriteString(rawDelta)
	raw := e.rawBuilder.String()
	preview := extractAssistantMessagePreview(raw)
	draftFiles := extractDraftFilePreviews(raw)
	streamPreview := aiStreamPreview{}
	if preview == e.visible {
		streamPreview.DraftFiles = cloneDraftFilesIfChanged(e.draftFiles, draftFiles)
		if len(streamPreview.DraftFiles) > 0 {
			e.draftFiles = streamPreview.DraftFiles
		}
		return streamPreview
	}
	nextRunes := []rune(preview)
	prevRunes := []rune(e.visible)
	if len(nextRunes) < len(prevRunes) {
		e.visible = preview
		streamPreview.DraftFiles = cloneDraftFilesIfChanged(e.draftFiles, draftFiles)
		if len(streamPreview.DraftFiles) > 0 {
			e.draftFiles = streamPreview.DraftFiles
		}
		return streamPreview
	}
	streamPreview.AssistantDelta = string(nextRunes[len(prevRunes):])
	e.visible = preview
	streamPreview.DraftFiles = cloneDraftFilesIfChanged(e.draftFiles, draftFiles)
	if len(streamPreview.DraftFiles) > 0 {
		e.draftFiles = streamPreview.DraftFiles
	}
	return streamPreview
}

func extractAssistantMessagePreview(raw string) string {
	keyIndex := strings.Index(raw, `"assistantMessage"`)
	if keyIndex < 0 {
		return ""
	}

	afterKey := raw[keyIndex+len(`"assistantMessage"`):]
	colonIndex := strings.Index(afterKey, ":")
	if colonIndex < 0 {
		return ""
	}

	valuePart := strings.TrimLeft(afterKey[colonIndex+1:], " \r\n\t")
	if valuePart == "" || valuePart[0] != '"' {
		return ""
	}

	return decodePartialJSONString(valuePart[1:])
}

func decodePartialJSONString(raw string) string {
	var builder strings.Builder
	for idx := 0; idx < len(raw); idx++ {
		ch := raw[idx]
		if ch == '"' {
			break
		}
		if ch != '\\' {
			builder.WriteByte(ch)
			continue
		}
		if idx+1 >= len(raw) {
			break
		}

		next := raw[idx+1]
		switch next {
		case '"', '\\', '/':
			builder.WriteByte(next)
			idx++
		case 'b':
			builder.WriteByte('\b')
			idx++
		case 'f':
			builder.WriteByte('\f')
			idx++
		case 'n':
			builder.WriteByte('\n')
			idx++
		case 'r':
			builder.WriteByte('\r')
			idx++
		case 't':
			builder.WriteByte('\t')
			idx++
		case 'u':
			if idx+5 >= len(raw) {
				return builder.String()
			}
			hexText := raw[idx+2 : idx+6]
			codepoint, err := strconv.ParseInt(hexText, 16, 32)
			if err != nil {
				return builder.String()
			}
			builder.WriteRune(rune(codepoint))
			idx += 5
		default:
			builder.WriteByte(next)
			idx++
		}
	}
	return builder.String()
}

func extractDraftFilePreviews(raw string) []aiDraftFile {
	keyIndex := strings.Index(raw, `"draftFiles"`)
	if keyIndex < 0 {
		return nil
	}

	arrayStart := strings.Index(raw[keyIndex:], "[")
	if arrayStart < 0 {
		return nil
	}
	arrayStart += keyIndex

	previews := make([]aiDraftFile, 0, 2)
	for idx := arrayStart + 1; idx < len(raw); idx++ {
		switch raw[idx] {
		case ']':
			return previews
		case '{':
			snippet, nextIndex := extractJSONObjectSnippet(raw, idx)
			draft := aiDraftFile{
				Path:        extractPartialJSONStringField(snippet, "path"),
				Content:     extractPartialJSONStringField(snippet, "content"),
				Reason:      extractPartialJSONStringField(snippet, "reason"),
				NeedsReview: true,
				Source:      "ai",
			}
			if draft.Path != "" || draft.Content != "" || draft.Reason != "" {
				previews = append(previews, draft)
			}
			if nextIndex <= idx {
				return previews
			}
			idx = nextIndex - 1
		}
	}
	return previews
}

func extractJSONObjectSnippet(raw string, start int) (string, int) {
	if start < 0 || start >= len(raw) || raw[start] != '{' {
		return "", start
	}

	depth := 0
	inString := false
	escaped := false
	for idx := start; idx < len(raw); idx++ {
		ch := raw[idx]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == '{' {
			depth++
			continue
		}
		if ch == '}' {
			depth--
			if depth == 0 {
				return raw[start : idx+1], idx + 1
			}
		}
	}
	return raw[start:], len(raw)
}

func extractPartialJSONStringField(raw, field string) string {
	keyIndex := strings.Index(raw, `"`+field+`"`)
	if keyIndex < 0 {
		return ""
	}
	afterKey := raw[keyIndex+len(field)+2:]
	colonIndex := strings.Index(afterKey, ":")
	if colonIndex < 0 {
		return ""
	}
	valuePart := strings.TrimLeft(afterKey[colonIndex+1:], " \r\n\t")
	if valuePart == "" || valuePart[0] != '"' {
		return ""
	}
	return decodePartialJSONString(valuePart[1:])
}

func cloneDraftFilesIfChanged(previous []aiDraftFile, next []aiDraftFile) []aiDraftFile {
	if draftFilesEqual(previous, next) {
		return nil
	}
	cloned := make([]aiDraftFile, len(next))
	copy(cloned, next)
	return cloned
}

func draftFilesEqual(left []aiDraftFile, right []aiDraftFile) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx].Path != right[idx].Path || left[idx].Content != right[idx].Content || left[idx].Reason != right[idx].Reason || left[idx].Source != right[idx].Source || left[idx].NeedsReview != right[idx].NeedsReview {
			return false
		}
	}
	return true
}

// writeSSEEvent serializes and flushes one SSE event frame.
func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, event string, payload aiStreamEvent) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\n", event); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", body); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
