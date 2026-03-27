package app

import (
	"flag"
	"fmt"
)

func (a *app) cmdConfig(global globalOptions, args []string) error {
	if len(args) == 0 {
		a.ui.Markdown(a.cfg.DebugMarkdown())
		return nil
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "show":
		a.ui.Markdown(a.cfg.DebugMarkdown())
		return nil
	case "path":
		a.ui.Line(a.cfg.ConfigPath)
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
		if written {
			a.ui.Success("Wrote config file")
		} else {
			a.ui.Warning("Config file already exists")
		}
		a.ui.KeyValue("Config path", a.cfg.ConfigPath)
		return nil
	case "set":
		if len(rest) < 2 {
			return usageError("usage: config set <key> <value>")
		}
		if err := a.cfg.Set(rest[0], rest[1]); err != nil {
			return err
		}
		if err := a.cfg.Save(); err != nil {
			return err
		}
		a.ui.Success("Updated config value")
		a.ui.KeyValue(rest[0], rest[1])
		return nil
	case "get":
		if len(rest) < 1 {
			return usageError("usage: config get <key>")
		}
		return a.printConfigKey(rest[0])
	default:
		return fmt.Errorf("unknown config subcommand: %s", sub)
	}
}

func (a *app) printConfigKey(key string) error {
	switch key {
	case "paths.bin_dir":
		a.ui.Line(a.cfg.Paths.BinDir)
	case "paths.build_root":
		a.ui.Line(a.cfg.Paths.BuildRoot)
	case "paths.deploy_dir":
		a.ui.Line(a.cfg.Paths.DeployDir)
	case "branches.dev":
		a.ui.Line(a.cfg.Branches.Dev)
	case "branches.stable":
		a.ui.Line(a.cfg.Branches.Stable)
	case "defaults.repo":
		a.ui.Line(a.cfg.Defaults.Repo)
	case "defaults.jobs":
		a.ui.Line(fmt.Sprintf("%d", a.cfg.Defaults.Jobs))
	default:
		if len(key) > 6 && key[:6] == "repos." {
			for name, repo := range a.cfg.Repos {
				if key == "repos."+name+".git" {
					a.ui.Line(repo.Git)
					return nil
				}
				if key == "repos."+name+".path" {
					a.ui.Line(repo.Path)
					return nil
				}
			}
		}
		return fmt.Errorf("unknown config key: %s", key)
	}
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
	if written {
		a.ui.Success("Created starter config")
	} else {
		a.ui.Warning("Using existing config")
	}
	a.ui.KeyValue("Config path", a.cfg.ConfigPath)
	a.ui.KeyValue("Repo", a.cfg.Defaults.Repo)
	a.ui.KeyValue("Build root", a.cfg.Paths.BuildRoot)
	a.ui.KeyValue("Deploy dir", a.cfg.Paths.DeployDir)
	a.ui.KeyValue("Bin dir", a.cfg.Paths.BinDir)
	a.ui.Markdown(`
## Next Steps

- run ` + "`godot-build config show`" + `
- adjust any local paths with ` + "`godot-build config set <key> <value>`" + `
- run ` + "`godot-build doctor`" + `
- run ` + "`godot-build install-cli`" + ` after your first deploy
`)
	return nil
}
