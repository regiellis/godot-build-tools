package app

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/playlogic/godot-build/internal/ui"
)

func (a *app) cmdPull(global globalOptions, args []string) error {
	repo, err := a.repoPath(global.repo)
	if err != nil {
		return err
	}
	a.ui.Panel("Pulling Latest", repo)
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
	a.ui.Panel("Checkout", args[0]+"\n"+repo)
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
	a.ui.Panel("Status", repo)
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
	a.ui.Panel("Branches", repo)
	return a.runCommand(repo, "git", "branch", "-a")
}

func (a *app) cmdInstallSelf(global globalOptions, args []string) error {
	fs, err := parseCommandFlags("install-self", args, func(fs *flag.FlagSet) {
		fs.String("bin-dir", "", "Override the user bin directory for this install")
		fs.Bool("no-path", false, "Do not add the bin directory to the user PATH")
		fs.Bool("force", false, "Overwrite an existing gbt binary")
		fs.SetOutput(a.ui.Stdout())
	})
	if err != nil {
		return err
	}
	binDir := a.resolveBinDir(fs.Lookup("bin-dir").Value.String())
	force := fs.Lookup("force").Value.String() == "true"
	noPath := fs.Lookup("no-path").Value.String() == "true"
	if !dirWritable(binDir) {
		return fmt.Errorf("bin directory is not writable: %s", binDir)
	}
	src, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve current executable: %w", err)
	}
	src = filepath.Clean(src)
	targetName := progName
	if runtime.GOOS == "windows" {
		targetName += ".exe"
	}
	dest := filepath.Join(binDir, targetName)
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}
	status := "new"
	style := "success"
	if samePath(src, dest) {
		status = "current"
		style = "muted"
	} else if fileExists(dest) {
		equal, err := filesEqual(src, dest)
		if err != nil {
			return err
		}
		switch {
		case equal:
			status = "unchanged"
			style = "muted"
		case force:
			status = "overwrite"
			style = "warning"
		default:
			status = "conflict"
			style = "error"
		}
	}
	a.ui.Panel("Install Self", "Bin dir: "+binDir)
	a.ui.Table("Binary Install Plan", []ui.Cell{{Text: "Source"}, {Text: "Status"}, {Text: "Destination"}}, [][]ui.Cell{{{Text: src, Style: "val"}, {Text: status, Style: style}, {Text: dest, Style: "val"}}})
	if status == "conflict" {
		a.ui.Warning("An existing gbt binary was left unchanged. Re-run with --force to overwrite it.")
		return fmt.Errorf("existing binary conflicts with install target: %s", dest)
	}
	if status == "new" || status == "overwrite" {
		if err := copyFile(src, dest); err != nil {
			return err
		}
	}
	if !noPath {
		added, err := a.ensureCLIPathDir(binDir)
		if err != nil {
			return err
		}
		if added {
			a.ui.Success("Added CLI bin directory to user PATH")
		} else {
			a.ui.Success("CLI bin directory already on PATH")
		}
	} else {
		a.ui.Line(a.ui.Styled("muted", "Skipped PATH update (--no-path)."))
	}
	if fs.Lookup("bin-dir").Value.String() != "" {
		a.ui.Line(a.ui.Styled("muted", "Used temporary bin-dir override. Save it with `gbt config set paths.bin_dir <path>` if you want it permanent."))
	}
	if status == "current" || status == "unchanged" {
		a.ui.Warning("No binary copy was needed")
	} else {
		a.ui.Success("Installed gbt binary")
	}
	return nil
}

func (a *app) cmdInstallCLI(global globalOptions, args []string) error {
	fs, err := parseCommandFlags("install-cli", args, func(fs *flag.FlagSet) {
		fs.Bool("mono", false, "Prefer Mono/C# target")
		fs.String("bin-dir", "", "Override the user bin directory for this install")
		fs.Bool("no-path", false, "Do not add the bin directory to the user PATH")
		fs.Bool("force", false, "Overwrite conflicting existing shim files")
		fs.SetOutput(a.ui.Stdout())
	})
	if err != nil {
		return err
	}
	mono := fs.Lookup("mono").Value.String() == "true"
	binDir := a.resolveBinDir(fs.Lookup("bin-dir").Value.String())
	force := fs.Lookup("force").Value.String() == "true"
	noPath := fs.Lookup("no-path").Value.String() == "true"
	if !dirWritable(binDir) {
		return fmt.Errorf("bin directory is not writable: %s", binDir)
	}

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
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}

	rows := [][]ui.Cell{}
	conflicts := []string{}
	written := 0
	for _, name := range sortedMapKeys(targets) {
		target := targets[name]
		if !fileExists(target) {
			continue
		}
		shimPath := a.cliShimPathFor(binDir, name)
		desired := shimContent(target)
		status := "new"
		style := "success"
		if fileExists(shimPath) {
			existing := readTextFile(shimPath)
			switch {
			case existing == desired:
				status = "unchanged"
				style = "muted"
			case force:
				status = "overwrite"
				style = "warning"
			default:
				status = "conflict"
				style = "error"
				conflicts = append(conflicts, shimPath)
				rows = append(rows, []ui.Cell{{Text: name, Style: "build-cmd"}, {Text: status, Style: style}, {Text: shimPath, Style: "val"}, {Text: target, Style: "val"}})
				continue
			}
		}
		if status != "unchanged" {
			if err := a.writeShimAt(binDir, name, target); err != nil {
				return err
			}
			written++
		}
		rows = append(rows, []ui.Cell{{Text: name, Style: "build-cmd"}, {Text: status, Style: style}, {Text: shimPath, Style: "val"}, {Text: target, Style: "val"}})
	}

	a.ui.Panel("Install CLI", "Bin dir: "+binDir)
	a.ui.Table("CLI Shim Plan", []ui.Cell{{Text: "Command"}, {Text: "Status"}, {Text: "Shim"}, {Text: "Target"}}, rows)
	if len(conflicts) > 0 {
		a.ui.Warning("Conflicting shim files were left unchanged. Re-run with --force to overwrite them.")
		return fmt.Errorf("found %d conflicting shim file(s)", len(conflicts))
	}
	if !noPath {
		added, err := a.ensureCLIPathDir(binDir)
		if err != nil {
			return err
		}
		if added {
			a.ui.Success("Added CLI bin directory to user PATH")
		} else {
			a.ui.Success("CLI bin directory already on PATH")
		}
	} else {
		a.ui.Line(a.ui.Styled("muted", "Skipped PATH update (--no-path)."))
	}
	if fs.Lookup("bin-dir").Value.String() != "" {
		a.ui.Line(a.ui.Styled("muted", "Used temporary bin-dir override. Save it with `gbt config set paths.bin_dir <path>` if you want it permanent."))
	}
	if written == 0 {
		a.ui.Warning("No shim files needed changes")
	} else {
		a.ui.Success(fmt.Sprintf("Wrote %d shim file(s)", written))
	}
	return nil
}

func (a *app) cmdUninstallCLI(global globalOptions, args []string) error {
	fs, err := parseCommandFlags("uninstall-cli", args, func(fs *flag.FlagSet) {
		fs.String("bin-dir", "", "Override the user bin directory for this uninstall")
		fs.Bool("remove-path", false, "Remove bin dir from PATH")
		fs.SetOutput(a.ui.Stdout())
	})
	if err != nil {
		return err
	}
	binDir := a.resolveBinDir(fs.Lookup("bin-dir").Value.String())
	removed := []string{}
	for _, name := range []string{"godot", "godot-dev", "godot-cs", "godot-dev-cs"} {
		if err := os.Remove(a.cliShimPathFor(binDir, name)); err == nil {
			removed = append(removed, name)
		}
	}
	a.ui.Panel("Uninstall CLI", binDir)
	if len(removed) > 0 {
		rows := [][]ui.Cell{}
		for _, name := range removed {
			rows = append(rows, []ui.Cell{{Text: name, Style: "build-cmd"}})
		}
		a.ui.Table("Removed CLI Shims", []ui.Cell{{Text: "Command"}}, rows)
	} else {
		a.ui.Warning("No CLI shims were installed")
	}
	if fs.Lookup("remove-path").Value.String() == "true" {
		removed, err := a.removeCLIPathDir(binDir)
		if err != nil {
			return err
		}
		if removed {
			a.ui.Success("Removed CLI bin directory from user PATH")
		} else {
			a.ui.Line(a.ui.Styled("muted", "Shim directory was not present in your user PATH."))
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
	a.ui.Panel("Build", p.Desc)
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
	a.ui.Panel("Custom SCons Build", strings.Join(rest, " "))
	extra := collectSConsFlags(*d3d12, *vulkan, *lto, *llvm, *dev, *mono, *jobs)
	return a.runSCons(repo, rest, extra)
}

func (a *app) cmdPresets(global globalOptions, args []string) error {
	rows := [][]ui.Cell{}
	for _, name := range sortedPresetNames() {
		p := presets[name]
		scons := strings.Join(p.Args, " ")
		if len(p.Batch) > 0 {
			scons = "batch: " + strings.Join(p.Batch, ", ")
		}
		rows = append(rows, []ui.Cell{{Text: name, Style: "info-cmd"}, {Text: p.Desc, Style: "val"}, {Text: scons, Style: "muted"}})
	}
	a.ui.Table("Build Presets", []ui.Cell{{Text: "Preset"}, {Text: "Description"}, {Text: "SCons Arguments"}}, rows)
	return nil
}

func (a *app) cmdClean(global globalOptions, args []string) error {
	repo, err := a.repoPath(global.repo)
	if err != nil {
		return err
	}
	a.ui.Panel("Clean", repo)
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
	info := a.gitInfo(repo)
	title := fmt.Sprintf("%s (%s @ %s)", filepath.Base(repo), info.Branch, info.Commit)
	if info.Dirty {
		title += " (dirty)"
	}
	a.ui.Panel("Binaries", title+"\n"+binDir)
	rows := [][]ui.Cell{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".exe") {
			continue
		}
		st, err := e.Info()
		if err != nil {
			continue
		}
		tag := ""
		if strings.Contains(e.Name(), ".console.") {
			tag = "console"
		}
		rows = append(rows, []ui.Cell{
			{Text: e.Name(), Style: "info-cmd"},
			{Text: fmt.Sprintf("%.1f MB", float64(st.Size())/(1024*1024)), Style: "muted"},
			{Text: st.ModTime().Format("2006-01-02 15:04"), Style: "muted"},
			{Text: tag, Style: "muted"},
		})
	}
	a.ui.Table("", []ui.Cell{{Text: "File"}, {Text: "Size"}, {Text: "Modified"}, {Text: ""}}, rows)
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
	a.ui.Panel("Build Deploy", p.Desc)
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
	a.ui.Panel("Update", p.Desc)
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
	rows := [][]ui.Cell{}
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
		for _, file := range meta.DeployedFiles {
			rows = append(rows, []ui.Cell{
				{Text: filepath.Base(p), Style: "info-cmd"},
				{Text: meta.Repo, Style: "val"},
				{Text: meta.Branch + " @ " + meta.Commit, Style: "val"},
				{Text: file, Style: "muted"},
				{Text: meta.DeployedAt, Style: "muted"},
			})
		}
	}
	if len(rows) == 0 {
		a.ui.Warning("No deployed build metadata found")
		return nil
	}
	a.ui.Table("Deployed Builds", []ui.Cell{{Text: "Slot"}, {Text: "Repo"}, {Text: "Branch"}, {Text: "File"}, {Text: "Deployed"}}, rows)
	return nil
}
