package main

import (
	"fmt"
	"os"

	"github.com/playlogic/godot-build/internal/app"
)

func main() {
	code := app.Run(os.Args[1:])
	if code != 0 {
		os.Exit(code)
	}

	fmt.Println()
}
