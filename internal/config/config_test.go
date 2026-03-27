package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestUserConfigPath(t *testing.T) {
	path, err := UserConfigPath()
	if err != nil {
		t.Fatalf("UserConfigPath returned error: %v", err)
	}
	if path == "" {
		t.Fatal("UserConfigPath returned an empty path")
	}
}

func TestSetRepoPath(t *testing.T) {
	cfg := defaultConfig(filepath.Join(t.TempDir(), "config.toml"), t.TempDir())
	if err := cfg.Set("repos.test.path", `D:\Repos\test`); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if got := cfg.Repos["test"].Path; got != `D:\Repos\test` {
		t.Fatalf("unexpected repo path: %s", got)
	}
}

func TestEncodeTOMLContainsOfficialRepo(t *testing.T) {
	cfg := defaultConfig(filepath.Join(t.TempDir(), "config.toml"), t.TempDir())
	text := cfg.EncodeTOML()
	for _, needle := range []string{"[paths]", "[branches]", "[defaults]", "[repos.godot]"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("missing %s from encoded toml", needle)
		}
	}
	if strings.Contains(text, "[repos.godot-nv]") {
		t.Fatal("encoded toml should not include the NVIDIA repo by default")
	}
}

func TestValidateDefaultRepoExists(t *testing.T) {
	cfg := defaultConfig(filepath.Join(t.TempDir(), "config.toml"), t.TempDir())
	cfg.Defaults.Repo = "missing"
	issues := cfg.Validate()
	found := false
	for _, issue := range issues {
		if issue.Key == "defaults.repo" && issue.Level == "FAIL" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected defaults.repo validation failure")
	}
}

func TestValidateSetRejectsEmptyBuildRoot(t *testing.T) {
	cfg := defaultConfig(filepath.Join(t.TempDir(), "config.toml"), t.TempDir())
	if err := cfg.ValidateSet("paths.build_root", ""); err == nil {
		t.Fatal("expected validation error for empty build root")
	}
}

func TestValidateRepoRejectsEmptyGit(t *testing.T) {
	cfg := defaultConfig(filepath.Join(t.TempDir(), "config.toml"), t.TempDir())
	if err := cfg.ValidateRepo("fork", "", filepath.Join(t.TempDir(), "fork")); err == nil {
		t.Fatal("expected validation error for empty repo git url")
	}
}
