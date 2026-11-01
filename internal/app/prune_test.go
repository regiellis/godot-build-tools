package app

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/playlogic/godot-build/internal/config"
	"github.com/playlogic/godot-build/internal/ui"
)

// newPruneTestApp builds an app backed by a real git repo containing one
// staged (tracked) file, one ignored file, and one untracked orphan.
func newPruneTestApp(t *testing.T) (*app, string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := t.TempDir()
	repoPath := filepath.Join(root, "godot")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoPath
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	run("init")
	writeFile(t, filepath.Join(repoPath, ".gitignore"), "ignored/\n")
	writeFile(t, filepath.Join(repoPath, "tracked.txt"), "keep me")
	run("add", ".gitignore", "tracked.txt")
	writeFile(t, filepath.Join(repoPath, "ignored", "build.o"), "artifact")
	writeFile(t, filepath.Join(repoPath, "orphan.cpp"), "stale")

	cfg := &config.Config{
		ConfigPath: filepath.Join(root, "config.toml"),
		Paths:      config.Paths{BinDir: filepath.Join(root, "bin"), BuildRoot: root, DeployDir: filepath.Join(root, "deploy")},
		Branches:   config.Branches{Dev: "master", Stable: "4.6"},
		Defaults:   config.Defaults{Repo: "godot", Jobs: 4},
		Repos:      map[string]config.Repo{"godot": {Path: repoPath}},
	}
	return &app{cfg: cfg, ui: ui.New(&strings.Builder{}, &strings.Builder{})}, repoPath
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPruneWithoutForceOnlyPreviews(t *testing.T) {
	a, repoPath := newPruneTestApp(t)
	if err := a.cmdPrune(globalOptions{repo: "godot"}, nil); err != nil {
		t.Fatalf("cmdPrune returned error: %v", err)
	}
	// The orphan must still exist when --force is not passed.
	if _, err := os.Stat(filepath.Join(repoPath, "orphan.cpp")); err != nil {
		t.Fatalf("orphan should not be deleted without --force: %v", err)
	}
}

func TestPruneForceRemovesOnlyOrphans(t *testing.T) {
	a, repoPath := newPruneTestApp(t)
	if err := a.cmdPrune(globalOptions{repo: "godot"}, []string{"--force"}); err != nil {
		t.Fatalf("cmdPrune returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoPath, "orphan.cpp")); !os.IsNotExist(err) {
		t.Fatalf("orphan should be deleted with --force, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(repoPath, "tracked.txt")); err != nil {
		t.Fatalf("tracked file must be preserved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoPath, "ignored", "build.o")); err != nil {
		t.Fatalf("ignored build output must be preserved: %v", err)
	}
}

func TestPruneDryRunDoesNotDelete(t *testing.T) {
	a, repoPath := newPruneTestApp(t)
	a.dryRun = true
	if err := a.cmdPrune(globalOptions{repo: "godot"}, []string{"--force"}); err != nil {
		t.Fatalf("cmdPrune returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoPath, "orphan.cpp")); err != nil {
		t.Fatalf("dry-run must not delete files: %v", err)
	}
}

func TestPruneJSON(t *testing.T) {
	a, repoPath := newPruneTestApp(t)
	a.jsonOutput = true
	out := &strings.Builder{}
	a.ui = ui.New(out, &strings.Builder{})

	if err := a.cmdPrune(globalOptions{repo: "godot"}, []string{"--force"}); err != nil {
		t.Fatalf("cmdPrune returned error: %v", err)
	}
	var payload struct {
		Count   int      `json:"count"`
		Files   []string `json:"files"`
		Deleted bool     `json:"deleted"`
	}
	if err := json.Unmarshal([]byte(out.String()), &payload); err != nil {
		t.Fatalf("expected valid JSON: %v\n%s", err, out.String())
	}
	if payload.Count != 1 || len(payload.Files) != 1 || payload.Files[0] != "orphan.cpp" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if !payload.Deleted {
		t.Fatal("expected deleted=true with --force in JSON mode")
	}
	if _, err := os.Stat(filepath.Join(repoPath, "orphan.cpp")); !os.IsNotExist(err) {
		t.Fatalf("JSON --force should delete the orphan, got err=%v", err)
	}
}
