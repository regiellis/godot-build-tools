package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveRepoRejectsDefaultRepo(t *testing.T) {
	cfg := defaultConfig(filepath.Join(t.TempDir(), "config.toml"), t.TempDir())
	err := cfg.RemoveRepo(cfg.Defaults.Repo)
	if err == nil {
		t.Fatal("expected removing the default repo to fail")
	}
	if !strings.Contains(err.Error(), "change defaults.repo first") {
		t.Fatalf("unexpected error: %v", err)
	}
}
