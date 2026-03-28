package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

func (a *app) repoPath(name string) (string, error) {
	repo, ok := a.cfg.Repos[name]
	if !ok {
		return "", fmt.Errorf("The repo %q is not configured. Add it with `gbt config repo add %s <git> <path>` or choose a different repo with `--repo`.", name, name)
	}
	if strings.TrimSpace(repo.Path) == "" {
		return "", fmt.Errorf("The repo %q does not have a local path configured. Set `repos.%s.path` first.", name, name)
	}
	if st, err := os.Stat(repo.Path); err != nil || !st.IsDir() {
		return "", fmt.Errorf("Could not find the configured repo at %s. Update `repos.%s.path` or clone the repo there first.", repo.Path, name)
	}
	return repo.Path, nil
}

func (a *app) sconsPath() string {
	if runtime.GOOS == "windows" {
		p := filepath.Join(a.cfg.Paths.BuildRoot, ".venv", "Scripts", "scons.exe")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "scons"
}

func (a *app) runCommand(dir string, name string, args ...string) error {
	fmt.Fprintf(a.ui.Stdout(), "%s %s\n", a.ui.Styled("cmd", ">>>"), a.ui.Styled("cmd", name+" "+strings.Join(args, " ")))
	if dir != "" {
		fmt.Fprintf(a.ui.Stdout(), "    %s %s\n\n", a.ui.Styled("muted", "(in"), a.ui.Styled("path", dir+")"))
	}
	if a.dryRun {
		a.ui.Warning("Dry-run: command not executed")
		return nil
	}
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = a.ui.Stdout()
	cmd.Stderr = a.ui.Stderr()
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		if errorsIs(err, exec.ErrNotFound) {
			return fmt.Errorf("Could not run %q because it was not found on PATH. Start with `gbt doctor` and install the missing toolchain or command.", name)
		}
		return err
	}
	return nil
}

func (a *app) capture(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func (a *app) gitInfo(repo string) gitInfo {
	branch, _ := a.capture(repo, "git", "rev-parse", "--abbrev-ref", "HEAD")
	commit, _ := a.capture(repo, "git", "rev-parse", "--short", "HEAD")
	commitFull, _ := a.capture(repo, "git", "rev-parse", "HEAD")
	dirty, _ := a.capture(repo, "git", "status", "--porcelain")
	if branch == "" {
		branch = "?"
	}
	if commit == "" {
		commit = "?"
	}
	if commitFull == "" {
		commitFull = "?"
	}
	return gitInfo{Branch: branch, Commit: commit, CommitFull: commitFull, Dirty: strings.TrimSpace(dirty) != ""}
}

func (a *app) writeDeployMeta(channel, repoName, presetName string, info gitInfo, files []string) error {
	meta := deployMeta{
		Repo:          repoName,
		Branch:        info.Branch,
		Commit:        info.Commit,
		CommitFull:    info.CommitFull,
		Dirty:         info.Dirty,
		Preset:        presetName,
		Channel:       channel,
		DeployedFiles: files,
		DeployedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return a.writeFile(a.deployMetaPath(channel), b, 0o644)
}

func (a *app) deployMetaPath(channel string) string {
	if channel == "stable" {
		return filepath.Join(a.cfg.Paths.DeployDir, "godot.json")
	}
	return filepath.Join(a.cfg.Paths.DeployDir, "godot-dev.json")
}

func (a *app) deployedNames(channel string, mono bool) (string, string) {
	if channel == "stable" {
		if mono {
			return "godot-cs.exe", "godot-cs-console.exe"
		}
		return "godot.exe", "godot-console.exe"
	}
	if mono {
		return "godot-dev-cs.exe", "godot-dev-cs-console.exe"
	}
	return "godot-dev.exe", "godot-dev-console.exe"
}

func (a *app) launchCommand(channel string, mono bool) string {
	if channel == "stable" {
		if mono {
			return "godot-cs"
		}
		return "godot"
	}
	if mono {
		return "godot-dev-cs"
	}
	return "godot-dev"
}

func (a *app) resolveBinDir(override string) string {
	if strings.TrimSpace(override) != "" {
		return filepath.Clean(strings.TrimSpace(override))
	}
	return a.cfg.Paths.BinDir
}

func (a *app) cliShimPath(name string) string {
	return a.cliShimPathFor(a.cfg.Paths.BinDir, name)
}

func (a *app) cliShimPathFor(binDir, name string) string {
	return filepath.Join(binDir, name+".cmd")
}

func (a *app) cliTargets(preferMono bool) map[string]string {
	stablePrimary, _ := a.deployedNames("stable", preferMono)
	devPrimary, _ := a.deployedNames("dev", preferMono)
	targets := map[string]string{
		"godot":     filepath.Join(a.cfg.Paths.DeployDir, stablePrimary),
		"godot-dev": filepath.Join(a.cfg.Paths.DeployDir, devPrimary),
	}
	if _, err := os.Stat(filepath.Join(a.cfg.Paths.DeployDir, "godot-cs.exe")); err == nil || preferMono {
		targets["godot-cs"] = filepath.Join(a.cfg.Paths.DeployDir, "godot-cs.exe")
	}
	if _, err := os.Stat(filepath.Join(a.cfg.Paths.DeployDir, "godot-dev-cs.exe")); err == nil || preferMono {
		targets["godot-dev-cs"] = filepath.Join(a.cfg.Paths.DeployDir, "godot-dev-cs.exe")
	}
	return targets
}

func shimContent(target string) string {
	return fmt.Sprintf("@echo off\r\n\"%s\" %%*\r\n", target)
}

func (a *app) writeShim(name, target string) error {
	return a.writeShimAt(a.cfg.Paths.BinDir, name, target)
}

func (a *app) writeShimAt(binDir, name, target string) error {
	return a.writeFile(a.cliShimPathFor(binDir, name), []byte(shimContent(target)), 0o644)
}

func (a *app) refreshShims() error {
	targets := a.cliTargets(false)
	for name, target := range targets {
		if _, err := os.Stat(a.cliShimPath(name)); err == nil {
			if _, err := os.Stat(target); err == nil || a.dryRun {
				if err := a.writeShim(name, target); err != nil {
					return fmt.Errorf("refresh shim %s: %w", name, err)
				}
			}
		}
	}
	return nil
}

func parseRegistryPathQuery(out string) string {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(strings.ToLower(line), "path") {
			continue
		}
		for _, valueType := range []string{"REG_EXPAND_SZ", "REG_SZ"} {
			if idx := strings.Index(line, valueType); idx >= 0 {
				return strings.TrimSpace(line[idx+len(valueType):])
			}
		}
	}
	return ""
}

func normalizePathEntry(path string) string {
	path = strings.TrimSpace(strings.Trim(path, `"`))
	path = strings.TrimRight(path, `\/`)
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}

func (a *app) readUserPath() string {
	if runtime.GOOS != "windows" {
		return os.Getenv("PATH")
	}
	out, err := a.capture("", "reg", "query", `HKCU\Environment`, "/v", "Path")
	if err != nil {
		return ""
	}
	return parseRegistryPathQuery(out)
}

func (a *app) writeUserPath(value string) error {
	if runtime.GOOS != "windows" {
		return nil
	}
	return a.runCommand("", "reg", "add", `HKCU\Environment`, "/v", "Path", "/t", "REG_EXPAND_SZ", "/d", value, "/f")
}

func pathContains(pathValue, wanted string) bool {
	wanted = normalizePathEntry(wanted)
	for _, part := range strings.Split(pathValue, ";") {
		part = normalizePathEntry(part)
		if part != "" && strings.EqualFold(part, wanted) {
			return true
		}
	}
	return false
}

func (a *app) ensureCLIPathDir(binDir string) (bool, error) {
	current := a.readUserPath()
	if pathContains(current, binDir) {
		return false, nil
	}
	parts := []string{}
	for _, p := range strings.Split(current, ";") {
		if strings.TrimSpace(p) != "" {
			parts = append(parts, p)
		}
	}
	parts = append(parts, binDir)
	return true, a.writeUserPath(strings.Join(parts, ";"))
}

func (a *app) ensureCLIPath() (bool, error) {
	return a.ensureCLIPathDir(a.cfg.Paths.BinDir)
}

func (a *app) removeCLIPathDir(binDir string) (bool, error) {
	current := a.readUserPath()
	if current == "" {
		return false, nil
	}
	kept := []string{}
	removed := false
	binDir = normalizePathEntry(binDir)
	for _, p := range strings.Split(current, ";") {
		normalized := normalizePathEntry(p)
		if normalized == "" {
			continue
		}
		if strings.EqualFold(normalized, binDir) {
			removed = true
			continue
		}
		kept = append(kept, p)
	}
	if !removed {
		return false, nil
	}
	return true, a.writeUserPath(strings.Join(kept, ";"))
}

func (a *app) removeCLIPath() (bool, error) {
	return a.removeCLIPathDir(a.cfg.Paths.BinDir)
}

func (a *app) copyFile(src, dst string) error {
	if a.dryRun {
		a.ui.Line(a.ui.Styled("muted", "Would copy ") + a.ui.Styled("path", src) + a.ui.Styled("muted", " -> ") + a.ui.Styled("path", dst))
		return nil
	}
	return copyFile(src, dst)
}

func (a *app) mkdirAll(path string, perm os.FileMode) error {
	if a.dryRun {
		a.ui.Line(a.ui.Styled("muted", "Would create directory ") + a.ui.Styled("path", path))
		return nil
	}
	return os.MkdirAll(path, perm)
}

func (a *app) writeFile(path string, data []byte, perm os.FileMode) error {
	if a.dryRun {
		a.ui.Line(a.ui.Styled("muted", "Would write ") + a.ui.Styled("path", path))
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, perm)
}

func (a *app) removeFile(path string) error {
	if a.dryRun {
		a.ui.Line(a.ui.Styled("muted", "Would remove ") + a.ui.Styled("path", path))
		return nil
	}
	return os.Remove(path)
}

func copyFile(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return err
	}
	return out.Close()
}

func samePath(aPath, bPath string) bool {
	aPath = filepath.Clean(aPath)
	bPath = filepath.Clean(bPath)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(aPath, bPath)
	}
	return aPath == bPath
}

func filesEqual(aPath, bPath string) (bool, error) {
	aData, err := os.ReadFile(aPath)
	if err != nil {
		return false, err
	}
	bData, err := os.ReadFile(bPath)
	if err != nil {
		return false, err
	}
	return bytes.Equal(aData, bData), nil
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirWritable(path string) bool {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return false
	}
	tmp := filepath.Join(path, ".godot-build-write-test")
	if err := os.WriteFile(tmp, []byte("ok"), 0o644); err != nil {
		return false
	}
	_ = os.Remove(tmp)
	return true
}

func readTextFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func sortedMapKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func errorsIs(err, target error) bool {
	return errors.Is(err, target)
}
