package app

import (
	"flag"
	"fmt"
	"strings"

	"github.com/playlogic/godot-build/internal/config"
	"github.com/playlogic/godot-build/internal/ui"
)

func (a *app) cmdConfig(global globalOptions, args []string) error {
	if len(args) == 0 {
		a.ui.Panel("Config", a.cfg.ConfigPath)
		a.ui.Markdown(a.cfg.DebugMarkdown())
		a.ui.Table("Config Commands", []ui.Cell{{Text: "Command"}, {Text: "Description"}}, [][]ui.Cell{
			{{Text: "config show", Style: "info-cmd"}, {Text: "Show the resolved config", Style: "val"}},
			{{Text: "config path", Style: "info-cmd"}, {Text: "Print the config file path", Style: "val"}},
			{{Text: "config keys", Style: "info-cmd"}, {Text: "List editable config keys", Style: "val"}},
			{{Text: "config validate", Style: "info-cmd"}, {Text: "Check whether the saved config is usable", Style: "val"}},
			{{Text: "config get <key>", Style: "info-cmd"}, {Text: "Read a single config value", Style: "val"}},
			{{Text: "config set <key> <value>", Style: "info-cmd"}, {Text: "Update a config value", Style: "val"}},
			{{Text: "config unset <key>", Style: "info-cmd"}, {Text: "Clear a config value", Style: "val"}},
			{{Text: "config repo add <name> <git> <path>", Style: "info-cmd"}, {Text: "Add or replace a repo entry", Style: "val"}},
			{{Text: "config repo remove <name>", Style: "info-cmd"}, {Text: "Remove a repo entry", Style: "val"}},
		})
		return nil
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "show":
		a.ui.Panel("Config", a.cfg.ConfigPath)
		a.ui.Markdown(a.cfg.DebugMarkdown())
		return nil
	case "path":
		a.ui.Panel("Config Path", a.cfg.ConfigPath)
		return nil
	case "keys":
		rows := [][]ui.Cell{}
		for _, key := range config.KnownKeys() {
			rows = append(rows, []ui.Cell{{Text: key, Style: "info-cmd"}})
		}
		a.ui.Table("Editable Keys", []ui.Cell{{Text: "Key"}}, rows)
		return nil
	case "validate":
		return a.cmdConfigValidate()
	case "init":
		fs, err := parseCommandFlags("config init", rest, func(fs *flag.FlagSet) {
			fs.Bool("force", false, "Overwrite existing config")
			fs.SetOutput(a.ui.Stdout())
		})
		if err != nil {
			return err
		}
		force := fs.Lookup("force").Value.String() == "true"
		a.ui.Panel("Config Init", a.cfg.ConfigPath)
		if a.dryRun {
			a.ui.Warning("Dry-run: config file not written")
			return nil
		}
		written, err := a.cfg.InitIfMissing(force)
		if err != nil {
			return err
		}
		if written {
			a.ui.Success("Wrote config file")
		} else {
			a.ui.Warning("Config file already exists")
		}
		return nil
	case "set":
		if len(rest) < 2 {
			return usageError("usage: config set <key> <value>")
		}
		key := rest[0]
		value := strings.Join(rest[1:], " ")
		if err := a.cfg.ValidateSet(key, value); err != nil {
			return err
		}
		if a.dryRun {
			a.ui.Table("Config Update Plan", []ui.Cell{{Text: "Key"}, {Text: "Value"}}, [][]ui.Cell{{{Text: key, Style: "info-cmd"}, {Text: value, Style: "val"}}})
			a.ui.Warning("Dry-run: config file not written")
			return nil
		}
		if err := a.cfg.Set(key, value); err != nil {
			return err
		}
		if err := a.cfg.Save(); err != nil {
			return err
		}
		a.ui.Table("Config Updated", []ui.Cell{{Text: "Key"}, {Text: "Value"}}, [][]ui.Cell{{{Text: key, Style: "info-cmd"}, {Text: value, Style: "val"}}})
		a.ui.Success("Updated config value")
		return nil
	case "unset":
		if len(rest) < 1 {
			return usageError("usage: config unset <key>")
		}
		key := rest[0]
		if err := a.cfg.ValidateUnset(key); err != nil {
			return err
		}
		if a.dryRun {
			a.ui.Table("Config Clear Plan", []ui.Cell{{Text: "Key"}}, [][]ui.Cell{{{Text: key, Style: "info-cmd"}}})
			a.ui.Warning("Dry-run: config file not written")
			return nil
		}
		if err := a.cfg.Unset(key); err != nil {
			return err
		}
		if err := a.cfg.Save(); err != nil {
			return err
		}
		a.ui.Table("Config Cleared", []ui.Cell{{Text: "Key"}}, [][]ui.Cell{{{Text: key, Style: "info-cmd"}}})
		a.ui.Success("Cleared config value")
		return nil
	case "get":
		if len(rest) < 1 {
			return usageError("usage: config get <key>")
		}
		return a.printConfigKey(rest[0])
	case "repo":
		return a.cmdConfigRepo(rest)
	default:
		return fmt.Errorf("Unknown config subcommand %q. Run `gbt config` to see the supported config commands.", sub)
	}
}

func (a *app) cmdConfigRepo(args []string) error {
	if len(args) == 0 {
		return usageError("usage: config repo <add|remove>")
	}
	switch args[0] {
	case "add":
		if len(args) < 4 {
			return usageError("usage: config repo add <name> <git> <path>")
		}
		name := args[1]
		gitURL := args[2]
		path := strings.Join(args[3:], " ")
		if err := a.cfg.ValidateRepo(name, gitURL, path); err != nil {
			return err
		}
		if a.dryRun {
			a.ui.Table("Repository Add Plan", []ui.Cell{{Text: "Name"}, {Text: "Git"}, {Text: "Path"}}, [][]ui.Cell{{{Text: name, Style: "info-cmd"}, {Text: gitURL, Style: "val"}, {Text: path, Style: "val"}}})
			a.ui.Warning("Dry-run: config file not written")
			return nil
		}
		a.cfg.SetRepo(name, gitURL, path)
		if err := a.cfg.Save(); err != nil {
			return err
		}
		a.ui.Table("Repository Added", []ui.Cell{{Text: "Name"}, {Text: "Git"}, {Text: "Path"}}, [][]ui.Cell{{{Text: name, Style: "info-cmd"}, {Text: gitURL, Style: "val"}, {Text: path, Style: "val"}}})
		a.ui.Success("Saved repository config")
		return nil
	case "remove":
		if len(args) < 2 {
			return usageError("usage: config repo remove <name>")
		}
		name := args[1]
		if a.dryRun {
			a.ui.Table("Repository Remove Plan", []ui.Cell{{Text: "Name"}}, [][]ui.Cell{{{Text: name, Style: "info-cmd"}}})
			a.ui.Warning("Dry-run: config file not written")
			return nil
		}
		if err := a.cfg.RemoveRepo(name); err != nil {
			return err
		}
		if err := a.cfg.Save(); err != nil {
			return err
		}
		a.ui.Table("Repository Removed", []ui.Cell{{Text: "Name"}}, [][]ui.Cell{{{Text: name, Style: "info-cmd"}}})
		a.ui.Success("Removed repository config")
		return nil
	default:
		return fmt.Errorf("Unknown config repo subcommand %q.", args[0])
	}
}

func (a *app) cmdConfigValidate() error {
	issues := a.cfg.Validate()
	rows := [][]ui.Cell{}
	for _, issue := range issues {
		style := "warning"
		if issue.Level == "FAIL" {
			style = "error"
		}
		rows = append(rows, []ui.Cell{{Text: issue.Level, Style: style}, {Text: issue.Key, Style: "info-cmd"}, {Text: issue.Message, Style: "val"}})
	}
	a.ui.Panel("Config Validation", a.cfg.ConfigPath)
	if len(rows) == 0 {
		a.ui.Success("Config looks usable")
		return nil
	}
	a.ui.Table("Issues", []ui.Cell{{Text: "Level"}, {Text: "Key"}, {Text: "Message"}}, rows)
	return fmt.Errorf("Config has %d issue(s). Fix the failing keys before relying on this setup.", len(rows))
}

func (a *app) printConfigKey(key string) error {
	var value string
	found := true
	switch key {
	case "paths.bin_dir":
		value = a.cfg.Paths.BinDir
	case "paths.build_root":
		value = a.cfg.Paths.BuildRoot
	case "paths.deploy_dir":
		value = a.cfg.Paths.DeployDir
	case "branches.dev":
		value = a.cfg.Branches.Dev
	case "branches.stable":
		value = a.cfg.Branches.Stable
	case "defaults.repo":
		value = a.cfg.Defaults.Repo
	case "defaults.jobs":
		value = fmt.Sprintf("%d", a.cfg.Defaults.Jobs)
	default:
		found = false
		if len(key) > 6 && key[:6] == "repos." {
			for name, repo := range a.cfg.Repos {
				if key == "repos."+name+".git" {
					value = repo.Git
					found = true
					break
				}
				if key == "repos."+name+".path" {
					value = repo.Path
					found = true
					break
				}
			}
		}
	}
	if !found {
		return fmt.Errorf("Unknown config key %q.", key)
	}
	if value == "" {
		value = "(empty)"
	}
	a.ui.Table("Config Value", []ui.Cell{{Text: "Key"}, {Text: "Value"}}, [][]ui.Cell{{{Text: key, Style: "info-cmd"}, {Text: value, Style: "val"}}})
	return nil
}

func (a *app) cmdOnboard(global globalOptions, args []string) error {
	fs, err := parseCommandFlags("onboard", args, func(fs *flag.FlagSet) {
		fs.Bool("force", false, "Overwrite existing config")
		fs.SetOutput(a.ui.Stdout())
	})
	if err != nil {
		return err
	}
	force := fs.Lookup("force").Value.String() == "true"
	a.ui.Panel("Onboarding", a.cfg.ConfigPath)
	if a.dryRun {
		a.ui.Warning("Dry-run: config file not written")
	} else {
		written, err := a.cfg.InitIfMissing(force)
		if err != nil {
			return err
		}
		if written {
			a.ui.Success("Created starter config")
		} else {
			a.ui.Warning("Using existing config")
		}
	}
	a.ui.Table("Environment Defaults", []ui.Cell{{Text: "Setting"}, {Text: "Value"}}, [][]ui.Cell{
		{{Text: "Repo", Style: "info-cmd"}, {Text: a.cfg.Defaults.Repo, Style: "val"}},
		{{Text: "Build root", Style: "info-cmd"}, {Text: a.cfg.Paths.BuildRoot, Style: "val"}},
		{{Text: "Deploy dir", Style: "info-cmd"}, {Text: a.cfg.Paths.DeployDir, Style: "val"}},
		{{Text: "Bin dir", Style: "info-cmd"}, {Text: a.cfg.Paths.BinDir, Style: "val"}},
	})
	a.ui.Markdown(`
## Next Steps

- run ` + "`gbt config show`" + `
- validate the config with ` + "`gbt config validate`" + `
- inspect editable keys with ` + "`gbt config keys`" + `
- adjust any local paths with ` + "`gbt config set <key> <value>`" + `
- run ` + "`gbt doctor`" + `
- run ` + "`gbt install-cli`" + ` after your first deploy
`)
	return nil
}
