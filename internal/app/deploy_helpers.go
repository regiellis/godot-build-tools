package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/playlogic/godot-build/internal/ui"
)

// detectTemplateVersion returns the export_templates folder name the built
// editor looks in: Godot's GODOT_VERSION_FULL_CONFIG (core/version.h), i.e.
// MAJOR.MINOR[.PATCH when PATCH > 0].STATUS + MODULE_CONFIG. It reads the
// build's core/version_generated.gen.h first (it carries build-time overrides
// such as GODOT_VERSION_STATUS), then version.py. It does not use git tags: a
// branch past its last tag (4.7 after 4.7.2-stable reports 4.7.3.rc) would
// otherwise install templates where the editor never looks.
func (a *app) detectTemplateVersion(repo string, override string) string {
	if override != "" {
		return override
	}
	if fields, ok := readVersionFields(filepath.Join(repo, "core", "version_generated.gen.h"), "#define GODOT_VERSION_", " "); ok {
		return templateVersion(fields["MAJOR"], fields["MINOR"], fields["PATCH"], fields["STATUS"], fields["MODULE_CONFIG"])
	}
	if fields, ok := readVersionFields(filepath.Join(repo, "version.py"), "", "="); ok {
		return templateVersion(fields["major"], fields["minor"], fields["patch"], fields["status"], fields["module_config"])
	}
	return "unknown"
}

// readVersionFields collects `<prefix>KEY<sep>value` lines, unquoting values.
// It reports ok only when major, minor and status were all found.
func readVersionFields(path, prefix, sep string) (map[string]string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	fields := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		key, value, found := strings.Cut(strings.TrimPrefix(line, prefix), sep)
		if !found {
			continue
		}
		fields[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	for _, key := range []string{"major", "minor", "status"} {
		if fields[key] == "" && fields[strings.ToUpper(key)] == "" {
			return nil, false
		}
	}
	return fields, true
}

func templateVersion(major, minor, patch, status, moduleConfig string) string {
	version := major + "." + minor
	if patch != "" && patch != "0" {
		version += "." + patch
	}
	return version + "." + status + moduleConfig
}

func (a *app) deployTemplates(repo string, versionOverride string) error {
	binDir := filepath.Join(repo, "bin")
	files, err := filepath.Glob(filepath.Join(binDir, "godot.windows.template_*.exe"))
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("No export template binaries were found in %s. Build the template presets first.", binDir)
	}
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		return fmt.Errorf("APPDATA is not set, so GBT does not know where to install export templates.")
	}
	version := a.detectTemplateVersion(repo, versionOverride)
	destDir := filepath.Join(appdata, "Godot", "export_templates", version)
	if err := a.mkdirAll(destDir, 0o755); err != nil {
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
		if err := a.copyFile(src, filepath.Join(destDir, destName)); err != nil {
			return err
		}
		rows = append(rows, []ui.Cell{{Text: name, Style: "val"}, {Text: destName, Style: "val"}})
		copies++
	}
	a.ui.Panel("Deploy Templates", filepath.Base(repo)+"\nVersion: "+version)
	a.ui.Table("Templates", []ui.Cell{{Text: "Source"}, {Text: "Destination"}}, rows)
	if a.dryRun {
		a.ui.Success(fmt.Sprintf("Dry-run complete. %d file(s) would be installed into %s", copies, destDir))
	} else {
		a.ui.Success(fmt.Sprintf("Template deploy complete. %d files -> %s", copies, destDir))
	}
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
		if err := a.copyFile(filepath.Join(binDir, name), filepath.Join(a.cfg.Paths.DeployDir, dest)); err != nil {
			return err
		}
		deployed = append(deployed, dest)
		rows = append(rows, []ui.Cell{{Text: name, Style: "val"}, {Text: dest, Style: "val"}})
	}
	if found == 0 {
		return fmt.Errorf("No editor binaries were found in %s. Build an editor preset first.", binDir)
	}
	if err := a.writeDeployMeta(channel, repoName, presetName, info, deployed); err != nil {
		return err
	}
	if err := a.refreshShims(); err != nil {
		return err
	}
	a.ui.Panel("Deploy", fmt.Sprintf("%s @ %s\nChannel: %s", info.Branch, info.Commit, channel))
	a.ui.Table("Editor Binaries", []ui.Cell{{Text: "Source"}, {Text: "Destination"}}, rows)
	if a.dryRun {
		a.ui.Success(fmt.Sprintf("Dry-run complete. Run %s to launch after a real deploy.", a.launchCommand(channel, mono)))
	} else {
		a.ui.Success(fmt.Sprintf("Deploy complete. Run %s to launch", a.launchCommand(channel, mono)))
	}
	return nil
}
