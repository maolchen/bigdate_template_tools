package server

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var undefinedTemplateFunctionPattern = regexp.MustCompile(`function "([^"]+)" not defined`)

type aiMessageProgress struct {
	Type        string
	Message     string
	PromptTrace aiPromptTrace
}

type aiMessageRunnerResult struct {
	Session       *aiSession
	ModelResponse *aiModelResponse
}

// runAISessionMessage executes one message turn with default model executor.
func (s *Server) runAISessionMessage(session *aiSession, req aiSessionMessageRequest, progress func(aiMessageProgress)) (*aiMessageRunnerResult, int, error) {
	return s.runAISessionMessageWithExecutor(session, req, progress, nil)
}

// runAISessionMessageWithExecutor handles end-to-end AI turn processing and session persistence.
func (s *Server) runAISessionMessageWithExecutor(session *aiSession, req aiSessionMessageRequest, progress func(aiMessageProgress), execute func(baseURL, apiKey string, req openAIChatRequest) (string, error)) (*aiMessageRunnerResult, int, error) {
	if req.SelectedSkillIDs == nil {
		req.SelectedSkillIDs = append([]string(nil), session.SelectedSkillIDs...)
	}
	req.SelectedSkillIDs = normalizeSelectedSkillIDs(req.SelectedSkillIDs)

	attachmentIDs := consumePendingAttachments(session)
	fmt.Printf("[AI] message session=%s messageLen=%d pendingAttachments=%d draftPaths=%d skills=%d\n",
		session.ID, len(strings.TrimSpace(req.Message)), len(attachmentIDs), len(req.SelectedDraftPaths), len(req.SelectedSkillIDs))

	emitAIProgress(progress, "status", "正在整理本轮附件、草稿与规则上下文", aiPromptTrace{})
	chatReq, err := s.buildAIChatRequest(session, req, attachmentIDs)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, errAIMasterKeyMissing) || errors.Is(err, errAISettingsMissing) {
			status = http.StatusBadRequest
		}
		return nil, status, err
	}

	emitAIProgress(progress, "trace", "", session.PromptTrace)
	emitAIProgress(progress, "status",
		fmt.Sprintf("已装载 %d 条规则，参考 %d 个模板示例",
			len(session.PromptTrace.SkillRefs), len(session.PromptTrace.ExampleRefs)),
		session.PromptTrace,
	)

	apiKey, settings, err := s.getAIAPIKey()
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	emitAIProgress(progress, "status", "正在调用模型生成结构化模板草稿", session.PromptTrace)
	if execute == nil {
		execute = s.aiClient.ChatCompletion
	}
	rawResponse, err := execute(normalizeBaseURL(settings.BaseURL), apiKey, chatReq)
	if err != nil {
		restorePendingAttachments(session, attachmentIDs)
		if len(attachmentIDs) > 0 {
			return nil, http.StatusBadRequest, fmt.Errorf("AI 调用失败，当前模型或接口可能不支持图像输入，或请求格式被拒绝: %w", err)
		}
		return nil, http.StatusBadRequest, err
	}

	emitAIProgress(progress, "status", "正在解析模型结果并校验动作类型", session.PromptTrace)
	modelResponse, err := parseAIModelResponse(rawResponse)
	if err != nil {
		fmt.Printf("[AI] response parse failed session=%s err=%v\n", session.ID, err)
		return nil, http.StatusBadGateway, err
	}

	if modelResponse.PlannedActions, err = validatePlannedActions(modelResponse.PlannedActions); err != nil {
		fmt.Printf("[AI] planned actions invalid session=%s err=%v\n", session.ID, err)
		return nil, http.StatusBadGateway, err
	}

	for idx := range modelResponse.DraftFiles {
		normalizedPath := filepath.ToSlash(filepath.Clean(strings.TrimSpace(modelResponse.DraftFiles[idx].Path)))
		modelResponse.DraftFiles[idx].Path = normalizedPath
		if !strings.HasPrefix(normalizedPath, "templates/") {
			return nil, http.StatusBadGateway, errors.New("AI 返回了非法模板路径")
		}
		modelResponse.DraftFiles[idx].NeedsReview = true
		if modelResponse.DraftFiles[idx].Source == "" {
			modelResponse.DraftFiles[idx].Source = "ai"
		}
		for _, attachmentID := range attachmentIDs {
			attachment, findErr := s.findAttachment(session, attachmentID)
			if findErr == nil && attachment.Kind == "image" {
				modelResponse.DraftFiles[idx].Source = "image"
			}
		}
	}

	emitAIProgress(progress, "status", "正在补全配置同步建议并写入会话草稿", session.PromptTrace)
	modelResponse.Warnings = s.appendDraftValidationWarnings(modelResponse.DraftFiles, modelResponse.Warnings)
	var configIssues []aiConfigIssue
	modelResponse.ConfigPatch, configIssues = s.enrichConfigPatchForDrafts(modelResponse.DraftFiles, modelResponse.ConfigPatch)
	for _, issue := range configIssues {
		if issue.Severity == "warning" || issue.Severity == "error" {
			modelResponse.Warnings = mergeStringLists(modelResponse.Warnings, []string{issue.Message})
		}
	}
	fmt.Printf("[AI] response session=%s drafts=%d actions=%d warnings=%d followUps=%d configIssues=%d\n",
		session.ID, len(modelResponse.DraftFiles), len(modelResponse.PlannedActions), len(modelResponse.Warnings),
		len(modelResponse.FollowUpQuestions), len(configIssues))

	messageID, err := generateID("msg_")
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	session.Messages = append(session.Messages, aiSessionMessage{
		ID:            messageID,
		Role:          "user",
		Content:       strings.TrimSpace(req.Message),
		CreatedAt:     nowRFC3339(),
		AttachmentIDs: attachmentIDs,
	})
	if strings.TrimSpace(session.Title) == "" || strings.TrimSpace(session.Title) == defaultAISessionTitle() {
		if title := summarizeSessionTitle(req.Message); title != "" {
			session.Title = title
		}
	}

	assistantMessageID, err := generateID("msg_")
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	session.Messages = append(session.Messages, aiSessionMessage{
		ID:                assistantMessageID,
		Role:              "assistant",
		Content:           strings.TrimSpace(modelResponse.AssistantMessage),
		CreatedAt:         nowRFC3339(),
		Warnings:          modelResponse.Warnings,
		FollowUpQuestions: modelResponse.FollowUpQuestions,
		DraftFiles:        modelResponse.DraftFiles,
		PlannedActions:    modelResponse.PlannedActions,
		ConfigPatch:       modelResponse.ConfigPatch,
		ConfigIssues:      configIssues,
		PromptTrace:       session.PromptTrace,
	})

	session.SessionRules = req.SessionRules
	session.SelectedSkillIDs = req.SelectedSkillIDs
	session.DraftFiles = mergeDraftFiles(session.DraftFiles, modelResponse.DraftFiles)
	session.PlannedActions = modelResponse.PlannedActions
	session.ConfigPatch = mergeAIConfigPatches(session.ConfigPatch, modelResponse.ConfigPatch)
	session.ConfigIssues = mergeAIConfigIssues(session.ConfigIssues, configIssues)

	if err := s.saveAISession(session); err != nil {
		fmt.Printf("[AI] save session failed session=%s err=%v\n", session.ID, err)
		return nil, http.StatusInternalServerError, err
	}
	s.populateAttachmentURLs(session)

	return &aiMessageRunnerResult{
		Session:       session,
		ModelResponse: modelResponse,
	}, http.StatusOK, nil
}

// extractUndefinedTemplateFunctions parses template errors and returns unique undefined function names.
func extractUndefinedTemplateFunctions(err error) []string {
	if err == nil {
		return []string{}
	}
	matches := undefinedTemplateFunctionPattern.FindAllStringSubmatch(err.Error(), -1)
	if len(matches) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(matches))
	functions := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		name := strings.TrimSpace(match[1])
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		functions = append(functions, name)
	}
	sort.Strings(functions)
	return functions
}

// appendDraftValidationWarnings pre-validates drafts and appends actionable warnings before save.
func (s *Server) appendDraftValidationWarnings(drafts []aiDraftFile, currentWarnings []string) []string {
	warnings := append([]string{}, currentWarnings...)
	for _, draft := range drafts {
		if strings.TrimSpace(draft.Content) == "" {
			continue
		}
		if err := s.validateDraftTemplate(draft.Content); err != nil {
			undefinedFunctions := extractUndefinedTemplateFunctions(err)
			if len(undefinedFunctions) > 0 {
				warnings = mergeStringLists(warnings, []string{
					fmt.Sprintf("草稿 %s 预检发现未注册函数: %s。请改为项目支持函数，避免保存失败。", draft.Path, strings.Join(undefinedFunctions, ", ")),
				})
				continue
			}
			warnings = mergeStringLists(warnings, []string{
				fmt.Sprintf("草稿 %s 预检发现模板语法风险，保存前请先修正: %v", draft.Path, err),
			})
		}
	}
	return warnings
}

func emitAIProgress(progress func(aiMessageProgress), eventType, message string, promptTrace aiPromptTrace) {
	if progress == nil {
		return
	}
	progress(aiMessageProgress{
		Type:        eventType,
		Message:     message,
		PromptTrace: promptTrace,
	})
}

// restorePendingAttachments marks attachments as pending again after a failed model call.
func restorePendingAttachments(session *aiSession, attachmentIDs []string) {
	if len(attachmentIDs) == 0 {
		return
	}
	for _, attachmentID := range attachmentIDs {
		attachment, err := sessionAttachmentByID(session, attachmentID)
		if err == nil {
			attachment.Pending = true
		}
	}
}

func sessionAttachmentByID(session *aiSession, attachmentID string) (*aiSessionAttachment, error) {
	for idx := range session.Attachments {
		if session.Attachments[idx].ID == attachmentID {
			return &session.Attachments[idx], nil
		}
	}
	return nil, os.ErrNotExist
}
