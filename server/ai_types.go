package server

import "config-generator/config"

type aiSettingsFile struct {
	BaseURL         string `json:"baseUrl"`
	Model           string `json:"model"`
	APIKeyEncrypted string `json:"apiKeyEncrypted"`
	UpdatedAt       string `json:"updatedAt"`
}

type aiSettingsResponse struct {
	BaseURL      string `json:"baseUrl"`
	Model        string `json:"model"`
	HasAPIKey    bool   `json:"hasApiKey"`
	MaskedAPIKey string `json:"maskedApiKey"`
	UpdatedAt    string `json:"updatedAt"`
}

type aiSession struct {
	ID             string                `json:"id"`
	Messages       []aiSessionMessage    `json:"messages"`
	DraftFiles     []aiDraftFile         `json:"draftFiles"`
	PlannedActions []aiPlannedAction     `json:"plannedActions"`
	ConfigPatch    aiConfigPatch         `json:"configPatch"`
	ConfigIssues   []aiConfigIssue       `json:"configIssues"`
	Attachments    []aiSessionAttachment `json:"attachments"`
	SessionRules   string                `json:"sessionRules"`
	CreatedAt      string                `json:"createdAt"`
	UpdatedAt      string                `json:"updatedAt"`
}

type aiSessionMessage struct {
	ID                string            `json:"id"`
	Role              string            `json:"role"`
	Content           string            `json:"content"`
	CreatedAt         string            `json:"createdAt"`
	AttachmentIDs     []string          `json:"attachmentIds,omitempty"`
	Warnings          []string          `json:"warnings,omitempty"`
	FollowUpQuestions []string          `json:"followUpQuestions,omitempty"`
	DraftFiles        []aiDraftFile     `json:"draftFiles,omitempty"`
	PlannedActions    []aiPlannedAction `json:"plannedActions,omitempty"`
	ConfigPatch       aiConfigPatch     `json:"configPatch,omitempty"`
	ConfigIssues      []aiConfigIssue   `json:"configIssues,omitempty"`
}

type aiDraftFile struct {
	Path        string `json:"path"`
	Content     string `json:"content"`
	Reason      string `json:"reason"`
	Source      string `json:"source,omitempty"`
	NeedsReview bool   `json:"needsReview"`
}

type aiPlannedAction struct {
	Type   string `json:"type"`
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type aiSessionAttachment struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	MimeType    string `json:"mimeType"`
	Size        int64  `json:"size"`
	StoredPath  string `json:"-"`
	PreviewText string `json:"previewText,omitempty"`
	DownloadURL string `json:"downloadUrl"`
	Pending     bool   `json:"pending"`
	CreatedAt   string `json:"createdAt"`
}

type aiSettingsUpdateRequest struct {
	BaseURL     string `json:"baseUrl"`
	Model       string `json:"model"`
	APIKey      string `json:"apiKey"`
	ClearAPIKey bool   `json:"clearApiKey"`
}

type aiSessionMessageRequest struct {
	Message            string   `json:"message"`
	SessionRules       string   `json:"sessionRules"`
	SelectedDraftPaths []string `json:"selectedDraftPaths"`
}

type aiSessionSaveRequest struct {
	Files            []aiDraftFile `json:"files"`
	ApplyConfigPatch bool          `json:"applyConfigPatch"`
}

type aiModelResponse struct {
	AssistantMessage  string            `json:"assistantMessage"`
	DraftFiles        []aiDraftFile     `json:"draftFiles"`
	PlannedActions    []aiPlannedAction `json:"plannedActions"`
	Warnings          []string          `json:"warnings"`
	FollowUpQuestions []string          `json:"followUpQuestions"`
	ConfigPatch       aiConfigPatch     `json:"configPatch"`
}

type aiConfigPatch struct {
	ServiceTop   map[string]config.ServiceTopo   `json:"serviceTop"`
	ServerConfig map[string]config.ServiceConfig `json:"serverConfig"`
}

type aiConfigIssue struct {
	Severity string `json:"severity"`
	Service  string `json:"service"`
	Field    string `json:"field"`
	Message  string `json:"message"`
}
