package app

import (
	"fmt"
	"os"
	"runtime"

	"github.com/playlogic/godot-build/internal/config"
	"github.com/playlogic/godot-build/internal/ui"
)

type app struct {
	cfg    *config.Config
	ui     *ui.Renderer
	dryRun bool
}

func Run(args []string) int {
	r := ui.New(os.Stdout, os.Stderr)
	cfg, err := config.Load()
	if err != nil {
		r.Error(fmt.Sprintf("Config could not be loaded. Start by checking your user config path and TOML syntax: %v", err))
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

func (a *app) versionRows() [][]ui.Cell {
	return [][]ui.Cell{
		{{Text: "Version", Style: "info-cmd"}, {Text: versionInfo.Version, Style: "val"}},
		{{Text: "Commit", Style: "info-cmd"}, {Text: versionInfo.Commit, Style: "val"}},
		{{Text: "Build date", Style: "info-cmd"}, {Text: versionInfo.BuildDate, Style: "val"}},
		{{Text: "Runtime", Style: "info-cmd"}, {Text: runtime.GOOS + "/" + runtime.GOARCH, Style: "val"}},
	}
}
