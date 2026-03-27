package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/playlogic/godot-build/internal/ui"
)

func (a *app) detectTemplateVersion(repo string, override string) string {
	if override != "" {
		return override
	}
	if out, err := a.capture(repo, "git", "describe", "--tags", "--abbrev=0"); err == nil && out != "" {
		return strings.ReplaceAll(strings.TrimPrefix(out, "v"), "-", ".")
	}
	versionPy := filepath.Join(repo, "version.py")
	data, err := os.ReadFile(versionPy)
	if err != nil {
		return "unknown"
	}
	text := string(data)
	grab := func(key string) string {
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, key+" =") {
				parts := strings.SplitN(line, "=", 2)
				return strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			}
		}
		return "?"
	}
	return fmt.Sprintf("%s.%s.%s.%s", grab("major"), grab("minor"), grab("patch"), grab("status"))
}

func (a *app) deployTemplates(repo string, versionOverride string) error {
	binDir := filepath.Join(repo, "bin")
	files, err := filepath.Glob(filepath.Join(binDir, "godot.windows.template_*.exe"))
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no export template binaries found in %s", binDir)
	}
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		return fmt.Errorf("APPDATA not set")
	}
	version := a.detectTemplateVersion(repo, versionOverride)
	destDir := filepath.Join(appdata, "Godot", "export_templates", version)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	rows := [][]ui.Cell{}
	copies := 0
	for _, src := range files {
		name := filepath.Base(src)
		isConsole := strings.Contains(name, ".console.")
		stem := strings.TrimSuffix(strings.TrimPrefix(name, "godot.windows."), ".exe")
		stem = strings.ReplaceAll(stem, ".console", "")
		stem = strings.TrimPrefix(stem, "template_")
		parts := strings.Split(stem, ".")
		if len(parts) != 2 {
			continue
		}
		targetType, arch := parts[0], parts[1]
		destName := fmt.Sprintf("windows_%s_%s.exe", targetType, arch)
		if isConsole {
			destName = fmt.Sprintf("windows_%s_%s_console.exe", targetType, arch)
		}
		if err := copyFile(src, filepath.Join(destDir, destName)); err != nil {
			return err
		}
		rows = append(rows, []ui.Cell{{Text: name, Style: "val"}, {Text: destName, Style: "val"}})
		copies++
	}
	a.ui.Panel("Deploy Templates", filepath.Base(repo)+"\nVersion: "+version)
	a.ui.Table("Templates", []ui.Cell{{Text: "Source"}, {Text: "Destination"}}, rows)
	a.ui.Success(fmt.Sprintf("Template deploy complete. %d files -> %s", copies, destDir))
	return nil
}

func (a *app) autoBuildTemplates(repo string, d3d12, vulkan bool, lto string, llvm bool, jobs int) error {
	a.ui.Section("Building Export Templates")
	extra := collectTemplateFlags(d3d12, vulkan, lto, llvm, jobs)
	for _, name := range presets["templates-all"].Batch {
		p := presets[name]
		a.ui.Line(a.ui.Styled("preset", name) + " " + a.ui.Styled("muted", p.Desc))
		if err := a.runSCons(repo, p.Args, extra); err != nil {
			return err
		}
	}
	a.ui.Section("Installing Export Templates")
	return a.deployTemplates(repo, "")
}

func (a *app) deployEditor(repoPath, repoName, presetName string, mono bool, channel string) error {
	binDir := filepath.Join(repoPath, "bin")
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return err
	}
	guiName, consoleName := a.deployedNames(channel, mono)
	info := a.gitInfo(repoPath)
	deployed := []string{}
	rows := [][]ui.Cell{}
	found := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "godot.windows.editor") || !strings.HasSuffix(name, ".exe") {
			continue
		}
		if mono && !strings.Contains(name, ".mono.") {
			continue
		}
		if !mono && strings.Contains(name, ".mono.") {
			continue
		}
		found++
		dest := guiName
		if strings.Contains(name, ".console.") {
			dest = consoleName
		}
		if err := copyFile(filepath.Join(binDir, name), filepath.Join(a.cfg.Paths.DeployDir, dest)); err != nil {
			return err
		}
		deployed = append(deployed, dest)
		rows = append(rows, []ui.Cell{{Text: name, Style: "val"}, {Text: dest, Style: "val"}})
	}
	if found == 0 {
		return fmt.Errorf("no editor binaries found in %s", binDir)
	}
	if err := a.writeDeployMeta(channel, repoName, presetName, info, deployed); err != nil {
		return err
	}
	a.refreshShims()
	a.ui.Panel("Deploy", fmt.Sprintf("%s @ %s\nChannel: %s", info.Branch, info.Commit, channel))
	a.ui.Table("Editor Binaries", []ui.Cell{{Text: "Source"}, {Text: "Destination"}}, rows)
	a.ui.Success(fmt.Sprintf("Deploy complete. Run %s to launch", a.launchCommand(channel, mono)))
	return nil
}
