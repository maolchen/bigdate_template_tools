package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type aiExampleIndexEntry struct {
	Path      string   `json:"path"`
	Service   string   `json:"service"`
	FileName  string   `json:"fileName"`
	Kind      string   `json:"kind"`
	Tags      []string `json:"tags"`
	IsCluster bool     `json:"isCluster"`
	Snippet   string   `json:"snippet"`
}

type aiPromptExample struct {
	Ref     aiExampleRef
	Snippet string
}

func detectExampleKind(path string) string {
	lowerPath := strings.ToLower(filepath.ToSlash(path))
	switch {
	case strings.HasSuffix(lowerPath, ".service.tmpl") || strings.Contains(lowerPath, "/systemd/"):
		return "systemd_service"
	case strings.HasSuffix(lowerPath, ".sh.tmpl"):
		return "shell_script"
	case strings.HasSuffix(lowerPath, ".md.tmpl"):
		return "readme"
	default:
		return "config_file"
	}
}

func looksClusterTemplate(content string, path string) bool {
	lowerPath := strings.ToLower(path)
	lowerContent := strings.ToLower(content)
	return containsAny(lowerPath, "cluster", "namenode", "datanode", "journalnode", "resourcemanager", "nodemanager") ||
		containsAny(lowerContent, ".allinstances", "servicenodes", "serviceendpoints", "getservicenodes")
}

func buildExampleTags(path string, content string) []string {
	lowerPath := strings.ToLower(filepath.ToSlash(path))
	tags := make([]string, 0, 8)
	switch detectExampleKind(lowerPath) {
	case "shell_script":
		tags = append(tags, "shell")
	case "config_file":
		tags = append(tags, "config")
	case "systemd_service":
		tags = append(tags, "systemd", "config")
	case "readme":
		tags = append(tags, "readme")
	}
	for _, candidate := range []string{"install", "setup", "start", "stop", "check", "init", "deploy"} {
		if strings.Contains(lowerPath, candidate) {
			tags = append(tags, candidate)
		}
	}
	if looksClusterTemplate(content, lowerPath) {
		tags = append(tags, "cluster")
	}
	return uniqueSortedStrings(tags)
}

func buildExampleSnippet(content string) string {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) > 40 {
		lines = lines[:40]
	}
	snippet := strings.Join(lines, "\n")
	runes := []rune(snippet)
	if len(runes) > 1800 {
		snippet = string(runes[:1800]) + "\n# ...truncated..."
	}
	return strings.TrimSpace(snippet)
}

func uniqueSortedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	sort.Strings(unique)
	return unique
}

func (s *Server) buildExampleIndex() ([]aiExampleIndexEntry, error) {
	entries := make([]aiExampleIndexEntry, 0, 32)
	err := filepath.WalkDir(s.templatesDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".tmpl") {
			return nil
		}

		relPath, err := filepath.Rel(filepath.Dir(s.templatesDir), path)
		if err != nil {
			return err
		}
		normalized := filepath.ToSlash(relPath)
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(contentBytes)
		entries = append(entries, aiExampleIndexEntry{
			Path:      normalized,
			Service:   serviceNameFromTemplatePath(normalized),
			FileName:  filepath.Base(normalized),
			Kind:      detectExampleKind(normalized),
			Tags:      buildExampleTags(normalized, content),
			IsCluster: looksClusterTemplate(content, normalized),
			Snippet:   buildExampleSnippet(content),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Service == entries[j].Service {
			return entries[i].Path < entries[j].Path
		}
		return entries[i].Service < entries[j].Service
	})

	if err := s.writeExampleIndex(entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *Server) writeExampleIndex(entries []aiExampleIndexEntry) error {
	if err := s.ensureAIDirs(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.aiExamplesPath, data, 0644)
}

func extractRequestedServices(message string, selectedDrafts []aiDraftFile, entries []aiExampleIndexEntry) []string {
	services := make([]string, 0)
	for _, draft := range selectedDrafts {
		serviceName := serviceNameFromTemplatePath(draft.Path)
		if serviceName != "" {
			services = append(services, serviceName)
		}
	}

	messageLower := strings.ToLower(message)
	knownServices := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.Service != "" {
			knownServices[entry.Service] = struct{}{}
		}
	}
	for serviceName := range knownServices {
		if strings.Contains(messageLower, strings.ToLower(serviceName)) {
			services = append(services, serviceName)
		}
	}
	return uniqueSortedStrings(services)
}

func inferDesiredExampleKinds(message string, selectedDrafts []aiDraftFile, skills []aiPromptSkill) []string {
	kinds := make([]string, 0, 4)
	messageLower := strings.ToLower(message)
	for _, skill := range skills {
		switch skill.Ref.ID {
		case "task/install_script":
			kinds = append(kinds, "shell_script")
		case "task/config_file":
			kinds = append(kinds, "config_file")
		case "task/systemd_service":
			kinds = append(kinds, "systemd_service")
		}
	}

	for _, draft := range selectedDrafts {
		kinds = append(kinds, detectExampleKind(draft.Path))
	}

	if containsAny(messageLower, ".service", "systemd") {
		kinds = append(kinds, "systemd_service")
	}
	if containsAny(messageLower, ".conf", ".yaml", ".yml", ".xml", ".json", "配置") {
		kinds = append(kinds, "config_file")
	}
	if containsAny(messageLower, ".sh.tmpl", "install", "setup", "start", "stop", "check", "安装", "部署") {
		kinds = append(kinds, "shell_script")
	}
	return uniqueSortedStrings(kinds)
}

func scoreExampleEntry(entry aiExampleIndexEntry, requestedServices []string, desiredKinds []string, wantsCluster bool, selectedDrafts []aiDraftFile, message string) (int, string) {
	score := 0
	reasons := make([]string, 0, 4)
	for _, serviceName := range requestedServices {
		if entry.Service == serviceName {
			score += 6
			reasons = append(reasons, "同服务")
			break
		}
	}
	for _, kind := range desiredKinds {
		if entry.Kind == kind {
			score += 5
			reasons = append(reasons, "同类型模板")
			break
		}
	}
	if wantsCluster && entry.IsCluster {
		score += 4
		reasons = append(reasons, "集群结构参考")
	}
	for _, draft := range selectedDrafts {
		if filepath.Base(draft.Path) == entry.FileName {
			score += 3
			reasons = append(reasons, "同文件命名")
			break
		}
	}

	messageLower := strings.ToLower(message)
	for _, tag := range entry.Tags {
		if strings.Contains(messageLower, tag) {
			score++
		}
	}
	return score, strings.Join(uniqueSortedStrings(reasons), "、")
}

func (s *Server) selectPromptExamples(message string, selectedDrafts []aiDraftFile, skills []aiPromptSkill) ([]aiPromptExample, error) {
	entries, err := s.buildExampleIndex()
	if err != nil {
		return nil, err
	}

	requestedServices := extractRequestedServices(message, selectedDrafts, entries)
	desiredKinds := inferDesiredExampleKinds(message, selectedDrafts, skills)
	wantsCluster := false
	for _, skill := range skills {
		if skill.Ref.ID == "task/cluster_service" {
			wantsCluster = true
			break
		}
	}

	type candidate struct {
		Entry  aiExampleIndexEntry
		Score  int
		Reason string
	}

	candidates := make([]candidate, 0, len(entries))
	for _, entry := range entries {
		score, reason := scoreExampleEntry(entry, requestedServices, desiredKinds, wantsCluster, selectedDrafts, message)
		if score <= 0 {
			continue
		}
		candidates = append(candidates, candidate{Entry: entry, Score: score, Reason: reason})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].Entry.Path < candidates[j].Entry.Path
		}
		return candidates[i].Score > candidates[j].Score
	})

	selected := make([]aiPromptExample, 0, 2)
	for _, candidate := range candidates {
		if len(selected) >= 2 {
			break
		}
		selected = append(selected, aiPromptExample{
			Ref: aiExampleRef{
				Path:    candidate.Entry.Path,
				Service: candidate.Entry.Service,
				Kind:    candidate.Entry.Kind,
				Tags:    append([]string(nil), candidate.Entry.Tags...),
				Reason:  candidate.Reason,
			},
			Snippet: candidate.Entry.Snippet,
		})
	}

	return selected, nil
}
