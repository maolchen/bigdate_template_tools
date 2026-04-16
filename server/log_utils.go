package server

import (
	"fmt"
	"sort"
	"strings"
)

func compactLogText(raw string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
}

func truncateRunes(raw string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(raw)
	if len(runes) <= max {
		return raw
	}
	return string(runes[:max]) + "..."
}

func previewLogText(raw string, max int) string {
	return truncateRunes(compactLogText(raw), max)
}

func summarizePathsForLog(paths []string, limit int) string {
	if len(paths) == 0 {
		return "[]"
	}
	if limit <= 0 {
		limit = 3
	}
	items := make([]string, 0, len(paths))
	for idx, path := range paths {
		if idx >= limit {
			break
		}
		items = append(items, path)
	}
	if len(paths) > limit {
		return fmt.Sprintf("[%s ... +%d]", strings.Join(items, ", "), len(paths)-limit)
	}
	return fmt.Sprintf("[%s]", strings.Join(items, ", "))
}

func summarizeSetForLog(values map[string]struct{}, limit int) string {
	if len(values) == 0 {
		return "[]"
	}
	items := make([]string, 0, len(values))
	for key := range values {
		items = append(items, key)
	}
	sort.Strings(items)
	return summarizePathsForLog(items, limit)
}

func summarizeDraftFilesForLog(drafts []aiDraftFile, limit int) string {
	if len(drafts) == 0 {
		return "[]"
	}
	if limit <= 0 {
		limit = 3
	}
	items := make([]string, 0, len(drafts))
	for idx, draft := range drafts {
		if idx >= limit {
			break
		}
		items = append(items, fmt.Sprintf("%s(len=%d,source=%s)", draft.Path, len(draft.Content), draft.Source))
	}
	if len(drafts) > limit {
		return fmt.Sprintf("[%s ... +%d]", strings.Join(items, ", "), len(drafts)-limit)
	}
	return fmt.Sprintf("[%s]", strings.Join(items, ", "))
}

func summarizeActionsForLog(actions []aiPlannedAction, limit int) string {
	if len(actions) == 0 {
		return "[]"
	}
	if limit <= 0 {
		limit = 4
	}
	items := make([]string, 0, len(actions))
	for idx, action := range actions {
		if idx >= limit {
			break
		}
		items = append(items, fmt.Sprintf("%s:%s", action.Type, action.Path))
	}
	if len(actions) > limit {
		return fmt.Sprintf("[%s ... +%d]", strings.Join(items, ", "), len(actions)-limit)
	}
	return fmt.Sprintf("[%s]", strings.Join(items, ", "))
}

func summarizeChatRequestForLog(req openAIChatRequest) string {
	if len(req.Messages) == 0 {
		return "messages=0"
	}
	messageSummaries := make([]string, 0, len(req.Messages))
	for idx, msg := range req.Messages {
		textParts := 0
		imageParts := 0
		textBytes := 0
		previewParts := make([]string, 0, len(msg.Content))
		for _, part := range msg.Content {
			switch normalizeMessagePartType(part.Type) {
			case "image_url":
				imageParts++
			default:
				textParts++
				textBytes += len(part.Text)
				if preview := previewLogText(part.Text, 80); preview != "" {
					previewParts = append(previewParts, preview)
				}
			}
		}
		preview := "-"
		if len(previewParts) > 0 {
			preview = truncateRunes(strings.Join(previewParts, " | "), 100)
		}
		messageSummaries = append(messageSummaries, fmt.Sprintf("#%d role=%s parts=%d textParts=%d images=%d textBytes=%d preview=%q",
			idx+1, msg.Role, len(msg.Content), textParts, imageParts, textBytes, preview))
	}
	return fmt.Sprintf("responseFormat=%s temp=%.2f maxTokens=%d messages={%s}",
		req.ResponseFormat["type"], req.Temperature, req.MaxTokens, strings.Join(messageSummaries, "; "))
}
