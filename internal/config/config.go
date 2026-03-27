package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type Config struct {
	ConfigPath string
	Paths      Paths
	Branches   Branches
	Defaults   Defaults
	Repos      map[string]Repo
}

type Paths struct {
	BinDir    string
	BuildRoot string
	DeployDir string
}

type Branches struct {
	Dev    string
	Stable string
}

type Defaults struct {
	Repo string
	Jobs int
}

type Repo struct {
	Git  string
	Path string
}

func Load() (*Config, error) {
	configPath, err := UserConfigPath()
	if err != nil {
		return nil, err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user home: %w", err)
	}

	buildRoot := defaultBuildRoot(home)
	deployDir := filepath.Join(home, "Engines", "Godot", "current")
	binDir := filepath.Join(home, "bin")
	jobs := 8
	if n := runtime.NumCPU(); n > 0 {
		jobs = n
	}

	cfg := &Config{
		ConfigPath: configPath,
		Paths: Paths{
			BinDir:    binDir,
			BuildRoot: buildRoot,
			DeployDir: deployDir,
		},
		Branches: Branches{
			Dev:    "master",
			Stable: "4.6",
		},
		Defaults: Defaults{
			Repo: "godot",
			Jobs: jobs,
		},
		Repos: map[string]Repo{},
	}

	if runtime.GOOS == "windows" {
		cfg.Paths.BuildRoot = `D:\Builds`
		cfg.Paths.DeployDir = `D:\Engines\Godot\current`
		cfg.Paths.BinDir = filepath.Join(home, "bin")
	}

	cfg.Repos["godot"] = Repo{
		Git:  "https://github.com/godotengine/godot.git",
		Path: filepath.Join(cfg.Paths.BuildRoot, "godot"),
	}
	cfg.Repos["godot-nv"] = Repo{
		Git:  "https://github.com/NVIDIA-Omniverse/godot.git",
		Path: filepath.Join(cfg.Paths.BuildRoot, "godot-nv"),
	}

	return cfg, nil
}

func defaultBuildRoot(home string) string {
	return filepath.Join(home, "Builds")
}

func UserConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(configDir, "godot-build", "config.toml"), nil
}

func (c *Config) DebugString() string {
	return strings.ReplaceAll(c.DebugMarkdown(), "`", "")
}

func (c *Config) DebugMarkdown() string {
	repoNames := make([]string, 0, len(c.Repos))
	for name := range c.Repos {
		repoNames = append(repoNames, name)
	}
	sort.Strings(repoNames)

	lines := []string{
		"## Resolved Config",
		"",
		fmt.Sprintf("- **Config file:** `%s`", c.ConfigPath),
		fmt.Sprintf("- **Bin dir:** `%s`", c.Paths.BinDir),
		fmt.Sprintf("- **Build root:** `%s`", c.Paths.BuildRoot),
		fmt.Sprintf("- **Deploy dir:** `%s`", c.Paths.DeployDir),
		fmt.Sprintf("- **Default repo:** `%s`", c.Defaults.Repo),
		fmt.Sprintf("- **Default jobs:** `%d`", c.Defaults.Jobs),
		fmt.Sprintf("- **Dev branch:** `%s`", c.Branches.Dev),
		fmt.Sprintf("- **Stable branch:** `%s`", c.Branches.Stable),
		"",
		"## Repositories",
		"",
	}

	for _, name := range repoNames {
		repo := c.Repos[name]
		lines = append(lines, fmt.Sprintf("### `%s`", name))
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("- **Git:** `%s`", repo.Git))
		lines = append(lines, fmt.Sprintf("- **Path:** `%s`", repo.Path))
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}
