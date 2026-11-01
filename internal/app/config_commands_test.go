package app

import (
	"os"
	"strings"
	"testing"
)

func TestCmdConfigSetReportsSavedPath(t *testing.T) {
	a, out, _ := newTestApp(t)

	if err := a.cmdConfig(globalOptions{}, []string{"set", "paths.deploy_dir", `D:\Engines\Godot\current`}); err != nil {
		t.Fatalf("cmdConfig returned error: %v", err)
	}

	text := out.String()
	if !strings.Contains(text, "Saved to "+a.cfg.ConfigPath) {
		t.Fatalf("expected output to include saved config path\n%s", text)
	}

	data, err := os.ReadFile(a.cfg.ConfigPath)
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}
	if !strings.Contains(string(data), `deploy_dir = "D:\\Engines\\Godot\\current"`) {
		t.Fatalf("expected config file to contain updated deploy_dir\n%s", string(data))
	}
}
