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
	a := &app{cfg: cfg, ui: r}
	return a.run(args)
}
