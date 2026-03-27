package app

import (
	"errors"
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/playlogic/godot-build/internal/ui"
)

const progName = "gbt"

func (a *app) run(args []string) int {
	global, rest, printConfig, code := a.parseGlobal(args)
	if code != 0 {
		return code
	}
	a.dryRun = global.dryRun
	if printConfig {
		a.ui.Markdown(a.cfg.DebugMarkdown())
		return 0
	}
	if len(rest) == 0 {
		a.printMainHelp()
		return 0
	}

	cmd := rest[0]
	rest = rest[1:]
	var err error
	switch cmd {
	case "pull":
		err = a.cmdPull(global, rest)
	case "checkout":
		err = a.cmdCheckout(global, rest)
	case "status":
		err = a.cmdStatus(global, rest)
	case "branches":
		err = a.cmdBranches(global, rest)
	case "doctor":
		err = a.cmdDoctor(global, rest)
	case "version":
		err = a.cmdVersion(global, rest)
	case "which":
		err = a.cmdWhich(global, rest)
	case "install-self":
		err = a.cmdInstallSelf(global, rest)
	case "install-cli":
		err = a.cmdInstallCLI(global, rest)
	case "uninstall-cli":
		err = a.cmdUninstallCLI(global, rest)
	case "build":
		err = a.cmdBuild(global, rest)
	case "build-deploy":
		err = a.cmdBuildDeploy(global, rest)
	case "update":
		err = a.cmdUpdate(global, rest)
	case "custom":
		err = a.cmdCustom(global, rest)
	case "presets":
		err = a.cmdPresets(global, rest)
	case "clean":
		err = a.cmdClean(global, rest)
	case "list":
		err = a.cmdList(global, rest)
	case "deploy":
		err = a.cmdDeploy(global, rest)
	case "deploy-templates":
		err = a.cmdDeployTemplates(global, rest)
	case "info":
		err = a.cmdInfo(global, rest)
	case "config":
		err = a.cmdConfig(global, rest)
	case "onboard":
		err = a.cmdOnboard(global, rest)
	case "help", "--help", "-h":
		a.printMainHelp()
		return 0
	default:
		a.ui.Error(fmt.Sprintf("Unknown command %q. Run `gbt --help` to see the supported commands.", cmd))
		a.printMainHelp()
		return 2
	}
	if err != nil {
		a.ui.Error(err.Error())
		return 1
	}
	return 0
}

func (a *app) parseGlobal(args []string) (globalOptions, []string, bool, int) {
	global := globalOptions{repo: a.cfg.Defaults.Repo}
	printConfig := false
	for len(args) > 0 {
		switch args[0] {
		case "--repo", "-r":
			if len(args) < 2 {
				a.ui.Error("Missing value for --repo. Use `gbt --repo <name> <command>`.")
				return global, nil, false, 2
			}
			global.repo = args[1]
			args = args[2:]
		case "--dry-run":
			global.dryRun = true
			args = args[1:]
		case "--print-config":
			printConfig = true
			args = args[1:]
		case "--help", "-h":
			a.printMainHelp()
			return global, nil, false, 1
		default:
			return global, args, printConfig, 0
		}
	}
	return global, args, printConfig, 0
}

func (a *app) printMainHelp() {
	subtitle := "Current repo: " + a.ui.Styled("key", a.cfg.Defaults.Repo)
	if a.dryRun {
		subtitle += "\n" + a.ui.Styled("warning", "Dry-run mode is active")
	}
	a.ui.Panel("GBT - Godot Build Tools", subtitle)
	a.ui.Line(fmt.Sprintf("Usage: %s %s",
		a.ui.Styled("cmd", progName),
		a.ui.Styled("muted", "<command> [options]")))
	a.ui.Line("")

	a.ui.Table("Global Options", nil, [][]ui.Cell{
		{{Text: "--repo, -r", Style: "key"}, {Text: fmt.Sprintf("Repository to use (default: %s)", a.cfg.Defaults.Repo), Style: "val"}},
		{{Text: "--dry-run", Style: "key"}, {Text: "Show what would run or be copied without changing anything", Style: "val"}},
		{{Text: "-h, --help", Style: "key"}, {Text: "Show this help message", Style: "val"}},
		{{Text: "--print-config", Style: "key"}, {Text: "Print the resolved config and exit", Style: "val"}},
	})

	groups := []struct {
		name string
		rows [][]ui.Cell
	}{
		{name: "Git", rows: [][]ui.Cell{
			{{Text: "pull", Style: "git-cmd"}, {Text: "Pull latest from origin (fast-forward only)", Style: "val"}},
			{{Text: "checkout", Style: "git-cmd"}, {Text: "Checkout a branch", Style: "val"}},
			{{Text: "status", Style: "git-cmd"}, {Text: "Show git status and recent commits", Style: "val"}},
			{{Text: "branches", Style: "git-cmd"}, {Text: "List all branches (local + remote)", Style: "val"}},
		}},
		{name: "Build", rows: [][]ui.Cell{
			{{Text: "doctor", Style: "build-cmd"}, {Text: "Check environment, config, and toolchain prerequisites", Style: "val"}},
			{{Text: "version", Style: "build-cmd"}, {Text: "Show GBT version and build information", Style: "val"}},
			{{Text: "which", Style: "build-cmd"}, {Text: "Show the resolved repo, deploy paths, and shim targets", Style: "val"}},
			{{Text: "install-self", Style: "build-cmd"}, {Text: "Install the gbt binary into your personal bin", Style: "val"}},
			{{Text: "install-cli", Style: "build-cmd"}, {Text: "Install godot/godot-dev launch shims", Style: "val"}},
			{{Text: "uninstall-cli", Style: "build-cmd"}, {Text: "Remove installed launch shims", Style: "val"}},
			{{Text: "build", Style: "build-cmd"}, {Text: "Run a build preset", Style: "val"}},
			{{Text: "build-deploy", Style: "build-cmd"}, {Text: "Build preset, deploy it, and install templates", Style: "val"}},
			{{Text: "update", Style: "build-cmd"}, {Text: "Pull + build + deploy + install templates", Style: "val"}},
			{{Text: "custom", Style: "build-cmd"}, {Text: "Run scons with custom arguments", Style: "val"}},
			{{Text: "clean", Style: "build-cmd"}, {Text: "Clean build artifacts", Style: "val"}},
		}},
		{name: "Deploy", rows: [][]ui.Cell{
			{{Text: "deploy", Style: "deploy-cmd"}, {Text: "Deploy built editor as godot-dev or godot", Style: "val"}},
			{{Text: "deploy-templates", Style: "deploy-cmd"}, {Text: "Deploy export templates to Godot appdata", Style: "val"}},
		}},
		{name: "Info", rows: [][]ui.Cell{
			{{Text: "onboard", Style: "info-cmd"}, {Text: "Create a starter config and show next steps", Style: "val"}},
			{{Text: "config", Style: "info-cmd"}, {Text: "Show or modify saved user config", Style: "val"}},
			{{Text: "presets", Style: "info-cmd"}, {Text: "List available build presets", Style: "val"}},
			{{Text: "list", Style: "info-cmd"}, {Text: "Show built binaries in bin/", Style: "val"}},
			{{Text: "info", Style: "info-cmd"}, {Text: "Show deployed build details", Style: "val"}},
		}},
	}
	for _, group := range groups {
		a.ui.Table(group.name, nil, group.rows)
	}

	a.ui.Table("Examples", nil, [][]ui.Cell{
		{{Text: progName + " version", Style: "cmd"}, {Text: "Show the current GBT build information", Style: "muted"}},
		{{Text: progName + " which", Style: "cmd"}, {Text: "Inspect the resolved repo and install paths", Style: "muted"}},
		{{Text: progName + " --dry-run update --stable", Style: "cmd"}, {Text: "Preview a stable update without changing anything", Style: "muted"}},
		{{Text: progName + " onboard", Style: "cmd"}, {Text: "Create the starter config", Style: "muted"}},
		{{Text: progName + " install-self", Style: "cmd"}, {Text: "Install gbt.exe into your personal bin", Style: "muted"}},
		{{Text: progName + " doctor", Style: "cmd"}, {Text: "Check local prerequisites", Style: "muted"}},
		{{Text: progName + " update", Style: "cmd"}, {Text: "Build and deploy the dev channel", Style: "muted"}},
		{{Text: progName + " update --stable", Style: "cmd"}, {Text: "Build and deploy the stable channel", Style: "muted"}},
		{{Text: progName + " update --stable --mono", Style: "cmd"}, {Text: "Build and deploy stable C# editor and templates", Style: "muted"}},
		{{Text: progName + " --repo godot build editor", Style: "cmd"}, {Text: "Build from the configured official repo", Style: "muted"}},
	})
}

func parseCommandFlags(name string, args []string, configure func(*flag.FlagSet)) (*flag.FlagSet, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	configure(fs)
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	return fs, nil
}

func (a *app) resolvePreset(name string, stable bool) (string, preset, error) {
	if name == "stable" {
		name = "editor-production"
	}
	if name == "dev" {
		name = "editor"
	}
	p, ok := presets[name]
	if !ok {
		return "", preset{}, fmt.Errorf("Unknown preset %q. Run `gbt presets` to see the supported preset names.", name)
	}
	if stable && !strings.Contains(name, "template") && name == "editor" {
		name = "editor-production"
		p = presets[name]
	}
	return name, p, nil
}

func sortedPresetNames() []string {
	names := make([]string, 0, len(presets))
	for name := range presets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func usageError(msg string) error {
	return errors.New(msg)
}
