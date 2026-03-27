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
		r.Panel("First Run", "Created starter config\n"+cfg.ConfigPath)
		r.Markdown(`
## Next Steps

- review the generated config with ` + "`gbt config show`" + `
- adjust local paths with ` + "`gbt config set <key> <value>`" + `
- run ` + "`gbt doctor`" + ` before your first build
`)
	}
	a := &app{cfg: cfg, ui: r}
	return a.run(args)
}
