package app

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (a *app) cmdPull(global globalOptions, args []string) error {
	repo, err := a.repoPath(global.repo)
	if err != nil {
		return err
	}
	if err := a.runCommand(repo, "git", "fetch", "--all", "--prune"); err != nil {
		return err
	}
	return a.runCommand(repo, "git", "pull", "--ff-only")
}

func (a *app) cmdCheckout(global globalOptions, args []string) error {
	if len(args) == 0 {
		return usageError("usage: checkout <branch>")
	}
	repo, err := a.repoPath(global.repo)
	if err != nil {
		return err
	}
	if err := a.runCommand(repo, "git", "fetch", "--all", "--prune"); err != nil {
		return err
	}
	if err := a.runCommand(repo, "git", "checkout", args[0]); err != nil {
		return err
	}
	_ = a.runCommand(repo, "git", "pull", "--ff-only")
	return nil
}

func (a *app) cmdStatus(global globalOptions, args []string) error {
	repo, err := a.repoPath(global.repo)
	if err != nil {
		return err
	}
	if err := a.runCommand(repo, "git", "status", "-sb"); err != nil {
		return err
	}
	return a.runCommand(repo, "git", "log", "--oneline", "-5")
}

func (a *app) cmdBranches(global globalOptions, args []string) error {
	repo, err := a.repoPath(global.repo)
	if err != nil {
		return err
	}
	return a.runCommand(repo, "git", "branch", "-a")
}

func (a *app) cmdInstallCLI(global globalOptions, args []string) error {
	fs, err := parseCommandFlags("install-cli", args, func(fs *flag.FlagSet) {
		fs.Bool("mono", false, "Prefer Mono/C# target")
		fs.SetOutput(a.ui.Stdout())
	})
	if err != nil {
		return err
	}
	mono := fs.Lookup("mono").Value.String() == "true"
	targets := a.cliTargets(mono)
	primary := targets["godot"]
	if mono {
		if v, ok := targets["godot-cs"]; ok {
			primary = v
		}
	}
	if !fileExists(primary) {
		return fmt.Errorf("deployed binary not found: %s", primary)
	}
	if err := os.MkdirAll(a.cfg.Paths.BinDir, 0o755); err != nil {
		return err
	}
	for _, name := range sortedMapKeys(targets) {
		target := targets[name]
		if fileExists(target) {
			if err := a.writeShim(name, target); err != nil {
				return err
			}
			a.ui.KeyValue(name, target)
		}
	}
	added, err := a.ensureCLIPath()
	if err != nil {
		return err
	}
	if added {
		a.ui.Success("Added CLI bin directory to user PATH")
	} else {
		a.ui.Success("CLI bin directory already on PATH")
	}
	return nil
}

func (a *app) cmdUninstallCLI(global globalOptions, args []string) error {
	fs, err := parseCommandFlags("uninstall-cli", args, func(fs *flag.FlagSet) {
		fs.Bool("remove-path", false, "Remove bin dir from PATH")
		fs.SetOutput(a.ui.Stdout())
	})
	if err != nil {
		return err
	}
	for _, name := range []string{"godot", "godot-dev", "godot-cs", "godot-dev-cs"} {
		_ = os.Remove(a.cliShimPath(name))
	}
	if fs.Lookup("remove-path").Value.String() == "true" {
		removed, err := a.removeCLIPath()
		if err != nil {
			return err
		}
		if removed {
			a.ui.Success("Removed CLI bin directory from user PATH")
		}
	}
	return nil
}

func (a *app) cmdBuild(global globalOptions, args []string) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	d3d12, vulkan, lto, llvm, dev, mono, jobs := a.parseBuildFlags(fs)
	fs.SetOutput(a.ui.Stdout())
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) == 0 {
		return usageError("usage: build <preset> [extra ...]")
	}
	repo, err := a.repoPath(global.repo)
	if err != nil {
		return err
	}
	presetName := rest[0]
	extraArgs := rest[1:]
	p, ok := presets[presetName]
	if !ok {
		return fmt.Errorf("unknown preset: %s", presetName)
	}
	extra := append(collectSConsFlags(*d3d12, *vulkan, *lto, *llvm, *dev, *mono, *jobs), extraArgs...)
	if len(p.Batch) > 0 {
		for _, name := range p.Batch {
			if err := a.runSCons(repo, presets[name].Args, extra); err != nil {
				return err
			}
		}
		return nil
	}
	return a.runSCons(repo, p.Args, extra)
}

func (a *app) cmdCustom(global globalOptions, args []string) error {
	fs := flag.NewFlagSet("custom", flag.ContinueOnError)
	d3d12, vulkan, lto, llvm, dev, mono, jobs := a.parseBuildFlags(fs)
	fs.SetOutput(a.ui.Stdout())
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) == 0 {
		return usageError("usage: custom <scons args...>")
	}
	repo, err := a.repoPath(global.repo)
	if err != nil {
		return err
	}
	extra := collectSConsFlags(*d3d12, *vulkan, *lto, *llvm, *dev, *mono, *jobs)
	return a.runSCons(repo, rest, extra)
}

func (a *app) cmdPresets(global globalOptions, args []string) error {
	for _, name := range sortedPresetNames() {
		p := presets[name]
		if len(p.Batch) > 0 {
			a.ui.Line(fmt.Sprintf("%s | %s | batch: %s", name, p.Desc, strings.Join(p.Batch, ", ")))
		} else {
			a.ui.Line(fmt.Sprintf("%s | %s | %s", name, p.Desc, strings.Join(p.Args, " ")))
		}
	}
	return nil
}

func (a *app) cmdClean(global globalOptions, args []string) error {
	repo, err := a.repoPath(global.repo)
	if err != nil {
		return err
	}
	return a.runSCons(repo, []string{"-c", "platform=windows"}, nil)
}

func (a *app) cmdList(global globalOptions, args []string) error {
	repo, err := a.repoPath(global.repo)
	if err != nil {
		return err
	}
	binDir := filepath.Join(repo, "bin")
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".exe") {
			continue
		}
		st, err := e.Info()
		if err != nil {
			continue
		}
		a.ui.Line(fmt.Sprintf("%s | %.1f MB | %s", e.Name(), float64(st.Size())/(1024*1024), st.ModTime().Format("2006-01-02 15:04")))
	}
	return nil
}

func (a *app) cmdDeploy(global globalOptions, args []string) error {
	fs, err := parseCommandFlags("deploy", args, func(fs *flag.FlagSet) {
		fs.Bool("mono", false, "Deploy Mono build")
		fs.Bool("stable", false, "Deploy stable slot")
		fs.Bool("yes", false, "Skip confirmation")
		fs.Bool("y", false, "Skip confirmation")
		fs.SetOutput(a.ui.Stdout())
	})
	if err != nil {
		return err
	}
	repo, err := a.repoPath(global.repo)
	if err != nil {
		return err
	}
	channel := "dev"
	if fs.Lookup("stable").Value.String() == "true" {
		channel = "stable"
	}
	mono := fs.Lookup("mono").Value.String() == "true"
	return a.deployEditor(repo, global.repo, "", mono, channel)
}

func (a *app) cmdDeployTemplates(global globalOptions, args []string) error {
	fs, err := parseCommandFlags("deploy-templates", args, func(fs *flag.FlagSet) {
		fs.String("version", "", "Override version")
		fs.Bool("yes", false, "Skip confirmation")
		fs.Bool("y", false, "Skip confirmation")
		fs.SetOutput(a.ui.Stdout())
	})
	if err != nil {
		return err
	}
	repo, err := a.repoPath(global.repo)
	if err != nil {
		return err
	}
	return a.deployTemplates(repo, fs.Lookup("version").Value.String())
}

func (a *app) cmdBuildDeploy(global globalOptions, args []string) error {
	fs := flag.NewFlagSet("build-deploy", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "Skip confirmation")
	fs.BoolVar(yes, "y", false, "Skip confirmation")
	stable := fs.Bool("stable", false, "Use stable channel")
	noTemplates := fs.Bool("no-templates", false, "Skip templates")
	d3d12, vulkan, lto, llvm, dev, mono, jobs := a.parseBuildFlags(fs)
	fs.SetOutput(a.ui.Stdout())
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) == 0 {
		return usageError("usage: build-deploy <preset> [extra...]")
	}
	repo, err := a.repoPath(global.repo)
	if err != nil {
		return err
	}
	if err := a.checkoutChannelBranch(repo, *stable); err != nil {
		return err
	}
	presetName, p, err := a.resolvePreset(rest[0], *stable)
	if err != nil {
		return err
	}
	extra := append(collectSConsFlags(*d3d12, *vulkan, *lto, *llvm, *dev, *mono, *jobs), rest[1:]...)
	if len(p.Batch) > 0 {
		for _, name := range p.Batch {
			if err := a.runSCons(repo, presets[name].Args, extra); err != nil {
				return err
			}
		}
	} else if err := a.runSCons(repo, p.Args, extra); err != nil {
		return err
	}
	if strings.Contains(presetName, "template") {
		return a.deployTemplates(repo, "")
	}
	channel := "dev"
	if *stable {
		channel = "stable"
	}
	if err := a.deployEditor(repo, global.repo, presetName, *mono, channel); err != nil {
		return err
	}
	if !*noTemplates {
		return a.autoBuildTemplates(repo, *d3d12, *vulkan, *lto, *llvm, *jobs)
	}
	return nil
}

func (a *app) cmdUpdate(global globalOptions, args []string) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "Skip confirmation")
	fs.BoolVar(yes, "y", false, "Skip confirmation")
	stable := fs.Bool("stable", false, "Use stable channel")
	noTemplates := fs.Bool("no-templates", false, "Skip templates")
	d3d12, vulkan, lto, llvm, dev, mono, jobs := a.parseBuildFlags(fs)
	fs.SetOutput(a.ui.Stdout())
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	presetArg := "editor"
	if *stable {
		presetArg = "editor-production"
	}
	if len(rest) > 0 {
		presetArg = rest[0]
		rest = rest[1:]
	}
	repo, err := a.repoPath(global.repo)
	if err != nil {
		return err
	}
	if err := a.checkoutChannelBranch(repo, *stable); err != nil {
		return err
	}
	presetName, p, err := a.resolvePreset(presetArg, *stable)
	if err != nil {
		return err
	}
	extra := append(collectSConsFlags(*d3d12, *vulkan, *lto, *llvm, *dev, *mono, *jobs), rest...)
	if len(p.Batch) > 0 {
		for _, name := range p.Batch {
			if err := a.runSCons(repo, presets[name].Args, extra); err != nil {
				return err
			}
		}
	} else if err := a.runSCons(repo, p.Args, extra); err != nil {
		return err
	}
	if strings.Contains(presetName, "template") {
		return a.deployTemplates(repo, "")
	}
	channel := "dev"
	if *stable {
		channel = "stable"
	}
	if err := a.deployEditor(repo, global.repo, presetName, *mono, channel); err != nil {
		return err
	}
	if !*noTemplates {
		return a.autoBuildTemplates(repo, *d3d12, *vulkan, *lto, *llvm, *jobs)
	}
	return nil
}

func (a *app) cmdInfo(global globalOptions, args []string) error {
	for _, p := range []string{a.deployMetaPath("dev"), a.deployMetaPath("stable")} {
		if !fileExists(p) {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		var meta deployMeta
		if err := json.Unmarshal(data, &meta); err != nil {
			return err
		}
		a.ui.Section(filepath.Base(p))
		a.ui.KeyValue("Repo", meta.Repo)
		a.ui.KeyValue("Branch", meta.Branch)
		a.ui.KeyValue("Commit", meta.Commit)
		a.ui.KeyValue("Preset", meta.Preset)
		a.ui.KeyValue("Channel", meta.Channel)
		a.ui.KeyValue("Deployed", meta.DeployedAt)
		for _, name := range meta.DeployedFiles {
			a.ui.KeyValue("File", name)
		}
		a.ui.Line("")
	}
	return nil
}
