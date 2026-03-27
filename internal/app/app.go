package app

import (
	"fmt"
	"os"

	"github.com/playlogic/godot-build/internal/config"
	"github.com/playlogic/godot-build/internal/ui"
)

type app struct {
	cfg *config.Config
	ui  *ui.Renderer
}

func Run(args []string) int {
	r := ui.New(os.Stdout, os.Stderr)
	cfg, err := config.Load()
	if err != nil {
		r.Error(fmt.Sprintf("config error: %v", err))
		return 1
	}
	if cfg.Created {
		r.Success("Created starter config")
		r.KeyValue("Config path", cfg.ConfigPath)
		r.Markdown(`
## First Run

- review the generated config with ` + "`godot-build config show`" + `
- adjust local paths with ` + "`godot-build config set <key> <value>`" + `
- run ` + "`godot-build doctor`" + ` before your first build
`)
	}
	a := &app{cfg: cfg, ui: r}
	return a.run(args)
}
