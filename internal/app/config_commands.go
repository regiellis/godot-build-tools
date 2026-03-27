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
	case "init":
		fs, err := parseCommandFlags("config init", rest, func(fs *flag.FlagSet) {
			fs.Bool("force", false, "Overwrite existing config")
			fs.SetOutput(a.ui.Stdout())
		})
		if err != nil {
			return err
		}
		force := fs.Lookup("force").Value.String() == "true"
		written, err := a.cfg.InitIfMissing(force)
		if err != nil {
			return err
		}
		a.ui.Panel("Config Init", a.cfg.ConfigPath)
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
		value := strings.Join(rest[1:], " ")
		if err := a.cfg.Set(rest[0], value); err != nil {
			return err
		}
		if err := a.cfg.Save(); err != nil {
			return err
		}
		a.ui.Table("Config Updated", []ui.Cell{{Text: "Key"}, {Text: "Value"}}, [][]ui.Cell{{{Text: rest[0], Style: "info-cmd"}, {Text: value, Style: "val"}}})
		a.ui.Success("Updated config value")
		return nil
	case "unset":
		if len(rest) < 1 {
			return usageError("usage: config unset <key>")
		}
		if err := a.cfg.Unset(rest[0]); err != nil {
			return err
		}
		if err := a.cfg.Save(); err != nil {
			return err
		}
		a.ui.Table("Config Cleared", []ui.Cell{{Text: "Key"}}, [][]ui.Cell{{{Text: rest[0], Style: "info-cmd"}}})
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
		return fmt.Errorf("unknown config subcommand: %s", sub)
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
		if err := a.cfg.RemoveRepo(args[1]); err != nil {
			return err
		}
		if err := a.cfg.Save(); err != nil {
			return err
		}
		a.ui.Table("Repository Removed", []ui.Cell{{Text: "Name"}}, [][]ui.Cell{{{Text: args[1], Style: "info-cmd"}}})
		a.ui.Success("Removed repository config")
		return nil
	default:
		return fmt.Errorf("unknown config repo subcommand: %s", args[0])
	}
}

func (a *app) printConfigKey(key string) error {
	var value string
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
		if len(key) > 6 && key[:6] == "repos." {
			for name, repo := range a.cfg.Repos {
				if key == "repos."+name+".git" {
					value = repo.Git
					break
				}
				if key == "repos."+name+".path" {
					value = repo.Path
					break
				}
			}
		}
		if value == "" {
			return fmt.Errorf("unknown config key: %s", key)
		}
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
	written, err := a.cfg.InitIfMissing(force)
	if err != nil {
		return err
	}
	a.ui.Panel("Onboarding", a.cfg.ConfigPath)
	if written {
		a.ui.Success("Created starter config")
	} else {
		a.ui.Warning("Using existing config")
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
- inspect editable keys with ` + "`gbt config keys`" + `
- adjust any local paths with ` + "`gbt config set <key> <value>`" + `
- run ` + "`gbt doctor`" + `
- run ` + "`gbt install-cli`" + ` after your first deploy
`)
	return nil
}
