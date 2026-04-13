package server

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"config-generator/config"
)

func (s *Server) getDescriptionPath(serviceName string) string {
	fileName := url.PathEscape(serviceName) + ".json"
	if serviceName == "global" {
		fileName = "global.json"
	}
	return filepath.Join(s.descsDir, fileName)
}

func (s *Server) loadDescriptionFile(serviceName string) (map[string]string, error) {
	descPath := s.getDescriptionPath(serviceName)
	if _, err := os.Stat(descPath); os.IsNotExist(err) {
		return map[string]string{}, nil
	}

	data, err := os.ReadFile(descPath)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return map[string]string{}, nil
	}

	descriptions := map[string]string{}
	if err := json.Unmarshal(data, &descriptions); err != nil {
		return nil, err
	}

	return descriptions, nil
}

func (s *Server) writeDescriptionFile(serviceName string, descriptions map[string]string) error {
	if err := os.MkdirAll(s.descsDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(descriptions, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.getDescriptionPath(serviceName), data, 0644)
}

func (s *Server) filterDescriptions(serviceName string, descriptions map[string]string, cfg *config.Config) map[string]string {
	filtered := make(map[string]string)

	if cfg == nil {
		return filtered
	}

	if serviceName == "global" {
		for key, value := range descriptions {
			if _, ok := cfg.Global[key]; ok && strings.TrimSpace(value) != "" {
				filtered[key] = value
			}
		}
		return filtered
	}

	service, ok := cfg.ServerConfig[serviceName]
	if !ok {
		return filtered
	}

	for key, value := range descriptions {
		if strings.TrimSpace(value) == "" {
			continue
		}

		if key == "description" {
			filtered[key] = value
			continue
		}

		if _, exists := service.Vars[key]; exists {
			filtered[key] = value
		}
	}

	return filtered
}

func (s *Server) getFilteredDescriptions(serviceName string, cfg *config.Config) (map[string]string, error) {
	descriptions, err := s.loadDescriptionFile(serviceName)
	if err != nil {
		return nil, err
	}

	filtered := s.filterDescriptions(serviceName, descriptions, cfg)
	if !reflect.DeepEqual(descriptions, filtered) {
		if err := s.writeDescriptionFile(serviceName, filtered); err != nil {
			return nil, err
		}
	}

	return filtered, nil
}

func (s *Server) saveDescriptionUpdates(serviceName string, updates map[string]string, cfg *config.Config) (map[string]string, error) {
	current, err := s.loadDescriptionFile(serviceName)
	if err != nil {
		return nil, err
	}

	for key, value := range updates {
		if strings.TrimSpace(value) == "" {
			delete(current, key)
			continue
		}
		current[key] = value
	}

	filtered := s.filterDescriptions(serviceName, current, cfg)
	if err := s.writeDescriptionFile(serviceName, filtered); err != nil {
		return nil, err
	}

	return filtered, nil
}
