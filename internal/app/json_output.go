package app

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/playlogic/godot-build/internal/config"
)

type jsonConfig struct {
	ConfigPath string                 `json:"config_path"`
	Created    bool                   `json:"created"`
	Paths      config.Paths           `json:"paths"`
	Branches   config.Branches        `json:"branches"`
	Defaults   config.Defaults        `json:"defaults"`
	Repos      map[string]config.Repo `json:"repos"`
}

type jsonValidationIssue struct {
	Level   string `json:"level"`
	Key     string `json:"key"`
	Message string `json:"message"`
}

func (a *app) writeJSON(v any) error {
	enc := json.NewEncoder(a.ui.Stdout())
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func (a *app) writeJSONError(err error) {
	_ = json.NewEncoder(a.ui.Stderr()).Encode(map[string]string{"error": err.Error()})
}

func (a *app) jsonConfig() jsonConfig {
	repos := make(map[string]config.Repo, len(a.cfg.Repos))
	for name, repo := range a.cfg.Repos {
		repos[name] = repo
	}
	return jsonConfig{
		ConfigPath: a.cfg.ConfigPath,
		Created:    a.cfg.Created,
		Paths:      a.cfg.Paths,
		Branches:   a.cfg.Branches,
		Defaults:   a.cfg.Defaults,
		Repos:      repos,
	}
}

func jsonValidationIssues(issues []config.ValidationIssue) []jsonValidationIssue {
	out := make([]jsonValidationIssue, 0, len(issues))
	for _, issue := range issues {
		out = append(out, jsonValidationIssue{
			Level:   issue.Level,
			Key:     issue.Key,
			Message: issue.Message,
		})
	}
	return out
}

func (a *app) jsonVersionPayload() map[string]any {
	return map[string]any{
		"version":    versionInfo.Version,
		"commit":     versionInfo.Commit,
		"build_date": versionInfo.BuildDate,
		"runtime":    runtime.GOOS + "/" + runtime.GOARCH,
	}
}

func (a *app) jsonUnsupported(command string) error {
	return fmt.Errorf("--json is not supported for %s yet because it streams external command output", command)
}

func metadataSlot(path string) string {
	base := filepath.Base(path)
	switch base {
	case "godot.json":
		return "stable"
	case "godot-dev.json":
		return "dev"
	default:
		return base
	}
}
