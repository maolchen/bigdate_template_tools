package server

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var customSkillIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,63}$`)

func normalizeCustomSkillID(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func validateCustomSkillID(skillID string) error {
	if !customSkillIDPattern.MatchString(skillID) {
		return errors.New("skill ID 只能包含小写字母、数字、-、_，且长度为 2-64")
	}
	return nil
}

func normalizeCustomSkillScope(scope string) string {
	scope = strings.ToLower(strings.TrimSpace(scope))
	switch scope {
	case "core", "task", "service":
		return scope
	default:
		return "task"
	}
}

func normalizeCustomSkillTags(tags []string) []string {
	return uniqueSortedStrings(tags)
}

func normalizeSelectedSkillIDs(skillIDs []string) []string {
	normalized := make([]string, 0, len(skillIDs))
	seen := make(map[string]struct{}, len(skillIDs))
	for _, skillID := range skillIDs {
		skillID = strings.TrimSpace(skillID)
		if skillID == "" {
			continue
		}
		if strings.HasPrefix(skillID, "custom/") {
			skillID = strings.TrimPrefix(skillID, "custom/")
		}
		skillID = normalizeCustomSkillID(skillID)
		if skillID == "" {
			continue
		}
		if _, exists := seen[skillID]; exists {
			continue
		}
		seen[skillID] = struct{}{}
		normalized = append(normalized, skillID)
	}
	sort.Strings(normalized)
	return normalized
}

func (s *Server) customSkillsDir() string {
	return filepath.Join(s.aiSkillsDir, "custom")
}

func (s *Server) customSkillPath(skillID string) string {
	return filepath.Join(s.customSkillsDir(), skillID+".md")
}

func parseCustomSkillFile(raw string) (aiCustomSkillMeta, string, error) {
	content := strings.ReplaceAll(raw, "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return aiCustomSkillMeta{}, strings.TrimSpace(content), nil
	}

	rest := strings.TrimPrefix(content, "---\n")
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		return aiCustomSkillMeta{}, "", errors.New("自定义 skill front matter 格式无效")
	}

	var meta aiCustomSkillMeta
	if err := yaml.Unmarshal([]byte(rest[:end]), &meta); err != nil {
		return aiCustomSkillMeta{}, "", err
	}
	body := strings.TrimSpace(rest[end+5:])
	return meta, body, nil
}

func formatCustomSkillFile(meta aiCustomSkillMeta, body string) (string, error) {
	meta.Tags = normalizeCustomSkillTags(meta.Tags)
	meta.Scope = normalizeCustomSkillScope(meta.Scope)
	if strings.TrimSpace(meta.Title) == "" {
		return "", errors.New("title 不能为空")
	}

	frontMatter, err := yaml.Marshal(meta)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(fmt.Sprintf("---\n%s---\n\n%s\n", string(frontMatter), strings.TrimSpace(body))), nil
}

func (s *Server) loadCustomSkill(skillID string) (aiCustomSkill, error) {
	skillID = normalizeCustomSkillID(skillID)
	if err := validateCustomSkillID(skillID); err != nil {
		return aiCustomSkill{}, err
	}

	data, err := os.ReadFile(s.customSkillPath(skillID))
	if err != nil {
		return aiCustomSkill{}, err
	}
	meta, body, err := parseCustomSkillFile(string(data))
	if err != nil {
		return aiCustomSkill{}, err
	}

	info, err := os.Stat(s.customSkillPath(skillID))
	if err != nil {
		return aiCustomSkill{}, err
	}

	updatedAt := strings.TrimSpace(meta.UpdatedAt)
	if updatedAt == "" {
		updatedAt = info.ModTime().Format(timeRFC3339Layout())
	}

	return aiCustomSkill{
		ID:         skillID,
		Title:      strings.TrimSpace(meta.Title),
		Scope:      normalizeCustomSkillScope(meta.Scope),
		Tags:       normalizeCustomSkillTags(meta.Tags),
		AutoAttach: meta.AutoAttach,
		Path:       filepath.ToSlash(filepath.Join("custom", skillID+".md")),
		Content:    body,
		UpdatedAt:  updatedAt,
	}, nil
}

func timeRFC3339Layout() string {
	return "2006-01-02T15:04:05Z07:00"
}

func (s *Server) listCustomAISkills() ([]aiCustomSkill, error) {
	if err := s.ensureAIDirs(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(s.customSkillsDir(), 0755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(s.customSkillsDir())
	if err != nil {
		return nil, err
	}

	skills := make([]aiCustomSkill, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".md" {
			continue
		}
		skillID := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		skill, err := s.loadCustomSkill(skillID)
		if err != nil {
			return nil, err
		}
		skills = append(skills, skill)
	}

	sort.SliceStable(skills, func(i, j int) bool {
		if skills[i].Scope == skills[j].Scope {
			return skills[i].ID < skills[j].ID
		}
		return skills[i].Scope < skills[j].Scope
	})
	return skills, nil
}

func (s *Server) saveCustomSkill(req aiCustomSkillUpsertRequest) (aiCustomSkill, error) {
	if err := s.ensureAIDirs(); err != nil {
		return aiCustomSkill{}, err
	}
	if err := os.MkdirAll(s.customSkillsDir(), 0755); err != nil {
		return aiCustomSkill{}, err
	}

	skillID := normalizeCustomSkillID(req.ID)
	if err := validateCustomSkillID(skillID); err != nil {
		return aiCustomSkill{}, err
	}
	if strings.TrimSpace(req.Title) == "" {
		return aiCustomSkill{}, errors.New("title 不能为空")
	}

	meta := aiCustomSkillMeta{
		Title:      strings.TrimSpace(req.Title),
		Scope:      normalizeCustomSkillScope(req.Scope),
		Tags:       normalizeCustomSkillTags(req.Tags),
		AutoAttach: req.AutoAttach,
		UpdatedAt:  nowRFC3339(),
	}
	formatted, err := formatCustomSkillFile(meta, req.Content)
	if err != nil {
		return aiCustomSkill{}, err
	}
	if err := os.WriteFile(s.customSkillPath(skillID), []byte(formatted), 0644); err != nil {
		return aiCustomSkill{}, err
	}
	return s.loadCustomSkill(skillID)
}

func (s *Server) deleteCustomSkill(skillID string) error {
	skillID = normalizeCustomSkillID(skillID)
	if err := validateCustomSkillID(skillID); err != nil {
		return err
	}
	if err := os.Remove(s.customSkillPath(skillID)); err != nil {
		return err
	}
	return nil
}

func tokenizeKeywords(values ...string) []string {
	keywords := make([]string, 0, len(values)*2)
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			continue
		}
		normalized = strings.NewReplacer("/", " ", "\\", " ", "-", " ", "_", " ", ".", " ").Replace(normalized)
		for _, part := range strings.Fields(normalized) {
			if len([]rune(part)) < 2 {
				continue
			}
			keywords = append(keywords, part)
		}
	}
	return uniqueSortedStrings(keywords)
}

func collectPromptContextKeywords(message string, selectedDrafts []aiDraftFile) string {
	values := []string{message}
	for _, draft := range selectedDrafts {
		values = append(values, draft.Path, draft.Reason, serviceNameFromTemplatePath(draft.Path))
	}
	return strings.ToLower(strings.Join(values, " "))
}

func (s *Server) selectMatchingCustomPromptSkills(message string, selectedDrafts []aiDraftFile) ([]aiPromptSkill, error) {
	customSkills, err := s.listCustomAISkills()
	if err != nil {
		return nil, err
	}

	haystack := collectPromptContextKeywords(message, selectedDrafts)
	selected := make([]aiPromptSkill, 0)
	for _, custom := range customSkills {
		if !custom.AutoAttach {
			continue
		}

		match := custom.Scope == "core"
		if !match {
			keywords := tokenizeKeywords(append([]string{custom.ID, custom.Title}, custom.Tags...)...)
			for _, keyword := range keywords {
				if strings.Contains(haystack, keyword) {
					match = true
					break
				}
			}
		}
		if !match {
			continue
		}

		selected = append(selected, aiPromptSkill{
			Ref: aiSkillRef{
				ID:       "custom/" + custom.ID,
				Scope:    custom.Scope,
				Title:    custom.Title,
				Tags:     append([]string(nil), custom.Tags...),
				Required: false,
			},
			Path:    filepath.Join(s.customSkillsDir(), custom.ID+".md"),
			Content: custom.Content,
		})
	}

	sort.SliceStable(selected, func(i, j int) bool {
		if selected[i].Ref.Scope == selected[j].Ref.Scope {
			return selected[i].Ref.ID < selected[j].Ref.ID
		}
		return selected[i].Ref.Scope < selected[j].Ref.Scope
	})
	return selected, nil
}

func (s *Server) selectExplicitCustomPromptSkills(skillIDs []string) ([]aiPromptSkill, error) {
	skillIDs = normalizeSelectedSkillIDs(skillIDs)
	selected := make([]aiPromptSkill, 0, len(skillIDs))
	for _, skillID := range skillIDs {
		custom, err := s.loadCustomSkill(skillID)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		selected = append(selected, aiPromptSkill{
			Ref: aiSkillRef{
				ID:       "custom/" + custom.ID,
				Scope:    custom.Scope,
				Title:    custom.Title,
				Tags:     append([]string(nil), custom.Tags...),
				Required: false,
			},
			Path:    filepath.Join(s.customSkillsDir(), custom.ID+".md"),
			Content: custom.Content,
		})
	}
	sort.SliceStable(selected, func(i, j int) bool {
		if selected[i].Ref.Scope == selected[j].Ref.Scope {
			return selected[i].Ref.ID < selected[j].Ref.ID
		}
		return selected[i].Ref.Scope < selected[j].Ref.Scope
	})
	return selected, nil
}
