package app

import (
	"errors"
	"flag"
	"fmt"
	"sort"
	"strings"
)

func (a *app) run(args []string) int {
	global, rest, printConfig, code := a.parseGlobal(args)
	if code != 0 {
		return code
	}
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
	case "help", "--help", "-h":
		a.printMainHelp()
		return 0
	default:
		a.ui.Error(fmt.Sprintf("unknown command: %s", cmd))
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
				a.ui.Error("missing value for --repo")
				return global, nil, false, 2
			}
			global.repo = args[1]
			args = args[2:]
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
	a.ui.Title("godot-build")
	a.ui.Subtitle("Go Godot build helper")
	a.ui.Markdown(`
## Commands

- ` + "`pull`" + `
- ` + "`checkout <branch>`" + `
- ` + "`status`" + `
- ` + "`branches`" + `
- ` + "`doctor`" + `
- ` + "`install-cli`" + `
- ` + "`uninstall-cli`" + `
- ` + "`build <preset>`" + `
- ` + "`build-deploy <preset>`" + `
- ` + "`update [preset]`" + `
- ` + "`custom <scons args...>`" + `
- ` + "`presets`" + `
- ` + "`clean`" + `
- ` + "`list`" + `
- ` + "`deploy`" + `
- ` + "`deploy-templates`" + `
- ` + "`info`" + `
`)
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
		return "", preset{}, fmt.Errorf("unknown preset: %s", name)
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
