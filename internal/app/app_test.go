package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/playlogic/godot-build/internal/config"
	"github.com/playlogic/godot-build/internal/ui"
)

func newTestApp(t *testing.T) (*app, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	root := t.TempDir()
	repoPath := filepath.Join(root, "godot")
	if err := os.MkdirAll(filepath.Join(repoPath, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "SConstruct"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "version.py"), []byte("major = 4\nminor = 6\npatch = 0\nstatus = 'stable'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cfg := &config.Config{
		ConfigPath: filepath.Join(root, "config.toml"),
		Paths: config.Paths{
			BinDir:    filepath.Join(root, "bin"),
			BuildRoot: root,
			DeployDir: filepath.Join(root, "deploy"),
		},
		Branches: config.Branches{Dev: "master", Stable: "4.6"},
		Defaults: config.Defaults{Repo: "godot", Jobs: 4},
		Repos: map[string]config.Repo{
			"godot": {Git: "https://github.com/godotengine/godot.git", Path: repoPath},
		},
	}
	return &app{cfg: cfg, ui: ui.New(out, errOut)}, out, errOut
}

func TestParseGlobalDryRun(t *testing.T) {
	a, _, _ := newTestApp(t)
	global, rest, printConfig, code := a.parseGlobal([]string{"--dry-run", "--json", "--repo", "godot", "version"})
	if code != 0 {
		t.Fatalf("unexpected code: %d", code)
	}
	if printConfig {
		t.Fatal("printConfig should be false")
	}
	if !global.dryRun {
		t.Fatal("expected dryRun to be true")
	}
	if !global.jsonOutput {
		t.Fatal("expected jsonOutput to be true")
	}
	if global.repo != "godot" {
		t.Fatalf("unexpected repo: %s", global.repo)
	}
	if len(rest) != 1 || rest[0] != "version" {
		t.Fatalf("unexpected rest args: %#v", rest)
	}
}

func TestCmdVersionShowsVersionInfo(t *testing.T) {
	a, out, _ := newTestApp(t)
	old := versionInfo
	versionInfo = toolVersion{Version: "1.2.3", Commit: "abc123", BuildDate: "2026-03-27"}
	defer func() { versionInfo = old }()

	if err := a.cmdVersion(globalOptions{repo: "godot"}, nil); err != nil {
		t.Fatalf("cmdVersion returned error: %v", err)
	}
	text := out.String()
	for _, needle := range []string{"GBT Version", "1.2.3", "abc123", "2026-03-27"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected output to contain %q\n%s", needle, text)
		}
	}
}

func TestCmdVersionJSON(t *testing.T) {
	a, out, _ := newTestApp(t)
	a.jsonOutput = true
	old := versionInfo
	versionInfo = toolVersion{Version: "1.2.3", Commit: "abc123", BuildDate: "2026-03-27"}
	defer func() { versionInfo = old }()

	if err := a.cmdVersion(globalOptions{repo: "godot"}, nil); err != nil {
		t.Fatalf("cmdVersion returned error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid JSON output: %v\n%s", err, out.String())
	}
	if payload["version"] != "1.2.3" {
		t.Fatalf("unexpected version payload: %#v", payload)
	}
	if payload["commit"] != "abc123" {
		t.Fatalf("unexpected commit payload: %#v", payload)
	}
}

func TestCmdWhichShowsResolvedTargets(t *testing.T) {
	a, out, _ := newTestApp(t)
	oldAppData := os.Getenv("APPDATA")
	if err := os.Setenv("APPDATA", filepath.Join(t.TempDir(), "appdata")); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Setenv("APPDATA", oldAppData) }()

	if err := a.cmdWhich(globalOptions{repo: "godot"}, []string{"--stable"}); err != nil {
		t.Fatalf("cmdWhich returned error: %v", err)
	}
	text := out.String()
	for _, needle := range []string{"Which", "Resolved Targets", filepath.Join(a.cfg.Paths.DeployDir, "godot.exe"), a.cfg.Paths.BinDir} {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected output to contain %q\n%s", needle, text)
		}
	}
}

func TestCmdWhichJSON(t *testing.T) {
	a, out, _ := newTestApp(t)
	a.jsonOutput = true
	oldAppData := os.Getenv("APPDATA")
	if err := os.Setenv("APPDATA", filepath.Join(t.TempDir(), "appdata")); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Setenv("APPDATA", oldAppData) }()

	if err := a.cmdWhich(globalOptions{repo: "godot"}, []string{"--stable"}); err != nil {
		t.Fatalf("cmdWhich returned error: %v", err)
	}

	var payload struct {
		ResolvedTargets map[string]string `json:"resolved_targets"`
		Repository      map[string]string `json:"repository"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid JSON output: %v\n%s", err, out.String())
	}
	if payload.Repository["repo"] != "godot" {
		t.Fatalf("unexpected repo payload: %#v", payload.Repository)
	}
	if payload.ResolvedTargets["gui_binary"] != filepath.Join(a.cfg.Paths.DeployDir, "godot.exe") {
		t.Fatalf("unexpected targets payload: %#v", payload.ResolvedTargets)
	}
}

func TestCmdConfigValidateFailsForInvalidConfig(t *testing.T) {
	a, _, _ := newTestApp(t)
	a.cfg.Defaults.Repo = "missing"
	if err := a.cmdConfigValidate(); err == nil {
		t.Fatal("expected config validation to fail")
	}
}

func TestWriteFileDryRunDoesNotWrite(t *testing.T) {
	a, _, _ := newTestApp(t)
	a.dryRun = true
	target := filepath.Join(t.TempDir(), "nested", "file.txt")
	if err := a.writeFile(target, []byte("hello"), 0o644); err != nil {
		t.Fatalf("writeFile returned error: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected file to not exist after dry-run write, got err=%v", err)
	}
}

func TestRepoPathUnknownRepoMessage(t *testing.T) {
	a, _, _ := newTestApp(t)
	_, err := a.repoPath("missing")
	if err == nil {
		t.Fatal("expected repoPath to fail")
	}
	if !strings.Contains(err.Error(), "gbt config repo add") {
		t.Fatalf("expected helpful repoPath error, got: %v", err)
	}
}
