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

type aiProviderModel struct {
	ID      string `json:"id"`
	OwnedBy string `json:"ownedBy,omitempty"`
	Created int64  `json:"created,omitempty"`
}

type aiSession struct {
	ID               string                `json:"id"`
	Title            string                `json:"title"`
	Messages         []aiSessionMessage    `json:"messages"`
	DraftFiles       []aiDraftFile         `json:"draftFiles"`
	PlannedActions   []aiPlannedAction     `json:"plannedActions"`
	ConfigPatch      aiConfigPatch         `json:"configPatch"`
	ConfigIssues     []aiConfigIssue       `json:"configIssues"`
	Attachments      []aiSessionAttachment `json:"attachments"`
	PromptTrace      aiPromptTrace         `json:"promptTrace"`
	SelectedSkillIDs []string              `json:"selectedSkillIds"`
	SelectedModel    string                `json:"selectedModel"`
	SessionRules     string                `json:"sessionRules"`
	CreatedAt        string                `json:"createdAt"`
	UpdatedAt        string                `json:"updatedAt"`
}

type aiSessionMessage struct {
	ID                string            `json:"id"`
	Role              string            `json:"role"`
	Content           string            `json:"content"`
	Model             string            `json:"model,omitempty"`
	CreatedAt         string            `json:"createdAt"`
	AttachmentIDs     []string          `json:"attachmentIds,omitempty"`
	Warnings          []string          `json:"warnings,omitempty"`
	FollowUpQuestions []string          `json:"followUpQuestions,omitempty"`
	DraftFiles        []aiDraftFile     `json:"draftFiles,omitempty"`
	PlannedActions    []aiPlannedAction `json:"plannedActions,omitempty"`
	ConfigPatch       aiConfigPatch     `json:"configPatch,omitempty"`
	ConfigIssues      []aiConfigIssue   `json:"configIssues,omitempty"`
	PromptTrace       aiPromptTrace     `json:"promptTrace,omitempty"`
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

type aiSessionMetaRequest struct {
	Title string `json:"title"`
}

type aiSessionMessageRequest struct {
	Message            string   `json:"message"`
	SessionRules       string   `json:"sessionRules"`
	SelectedDraftPaths []string `json:"selectedDraftPaths"`
	SelectedSkillIDs   []string `json:"selectedSkillIds"`
	Model              string   `json:"model"`
}

type aiSessionSummary struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
	MessageCount int    `json:"messageCount"`
	DraftCount   int    `json:"draftCount"`
}

type aiSessionSaveRequest struct {
	Files            []aiDraftFile `json:"files"`
	ApplyConfigPatch bool          `json:"applyConfigPatch"`
}

type aiSessionDeleteDraftRequest struct {
	Paths          []string `json:"paths"`
	RemoveFromDisk bool     `json:"removeFromDisk"`
}

type aiSessionDeleteResponse struct {
	DeletedSessionID string `json:"deletedSessionId"`
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

type aiConfigNextAction struct {
	Severity string `json:"severity"`
	Service  string `json:"service"`
	Field    string `json:"field"`
	Action   string `json:"action"`
	Detail   string `json:"detail"`
}

type aiSkillRef struct {
	ID       string   `json:"id"`
	Scope    string   `json:"scope"`
	Title    string   `json:"title"`
	Tags     []string `json:"tags"`
	Required bool     `json:"required"`
}

type aiExampleRef struct {
	Path    string   `json:"path"`
	Service string   `json:"service"`
	Kind    string   `json:"kind"`
	Tags    []string `json:"tags"`
	Reason  string   `json:"reason"`
}

type aiPromptTrace struct {
	SkillRefs   []aiSkillRef   `json:"skillRefs"`
	ExampleRefs []aiExampleRef `json:"exampleRefs"`
	Mode        string         `json:"mode"`
	Provider    string         `json:"provider"`
}

type aiPromptPreviewAttachment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type aiPromptPreviewResponse struct {
	Summary            string                      `json:"summary"`
	PromptTrace        aiPromptTrace               `json:"promptTrace"`
	SelectedDraftPaths []string                    `json:"selectedDraftPaths"`
	SelectedSkillIDs   []string                    `json:"selectedSkillIds"`
	Attachments        []aiPromptPreviewAttachment `json:"attachments"`
	HasSessionRules    bool                        `json:"hasSessionRules"`
}

type aiSkillCatalogItem struct {
	ID       string   `json:"id"`
	Scope    string   `json:"scope"`
	Title    string   `json:"title"`
	Tags     []string `json:"tags"`
	Required bool     `json:"required"`
	Path     string   `json:"path"`
	Source   string   `json:"source"`
	Content  string   `json:"content"`
}

type aiPromptCatalogResponse struct {
	Skills   []aiSkillCatalogItem  `json:"skills"`
	Examples []aiExampleIndexEntry `json:"examples"`
}

type aiSkillFileUpdateRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type aiCustomSkill struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Scope      string   `json:"scope"`
	Tags       []string `json:"tags"`
	AutoAttach bool     `json:"autoAttach"`
	Path       string   `json:"path"`
	Content    string   `json:"content"`
	UpdatedAt  string   `json:"updatedAt"`
}

type aiCustomSkillMeta struct {
	Title      string   `yaml:"title"`
	Scope      string   `yaml:"scope"`
	Tags       []string `yaml:"tags"`
	AutoAttach bool     `yaml:"autoAttach"`
	UpdatedAt  string   `yaml:"updatedAt"`
}

type aiCustomSkillUpsertRequest struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Scope      string   `json:"scope"`
	Tags       []string `json:"tags"`
	AutoAttach bool     `json:"autoAttach"`
	Content    string   `json:"content"`
}
