package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	ConfigPath string
	Created    bool            `toml:"-"`
	Paths      Paths           `toml:"paths"`
	Branches   Branches        `toml:"branches"`
	Defaults   Defaults        `toml:"defaults"`
	Repos      map[string]Repo `toml:"repos"`
}

type Paths struct {
	BinDir    string `toml:"bin_dir"`
	BuildRoot string `toml:"build_root"`
	DeployDir string `toml:"deploy_dir"`
}

type Branches struct {
	Dev    string `toml:"dev"`
	Stable string `toml:"stable"`
}

type Defaults struct {
	Repo string `toml:"repo"`
	Jobs int    `toml:"jobs"`
}

type Repo struct {
	Git  string `toml:"git"`
	Path string `toml:"path"`
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

	if !fileExists(configPath) {
		if _, err := cfg.InitIfMissing(false); err != nil {
			return nil, err
		}
		cfg.Created = true
	}

	var userCfg Config
	if _, err := toml.DecodeFile(configPath, &userCfg); err != nil {
		return nil, fmt.Errorf("decode config file: %w", err)
	}
	mergeConfig(cfg, &userCfg)

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

func (c *Config) InitIfMissing(force bool) (bool, error) {
	if !force && fileExists(c.ConfigPath) {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(c.ConfigPath), 0o755); err != nil {
		return false, fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(c.ConfigPath, []byte(c.EncodeTOML()), 0o644); err != nil {
		return false, fmt.Errorf("write config file: %w", err)
	}
	return true, nil
}

func (c *Config) Save() error {
	if err := os.MkdirAll(filepath.Dir(c.ConfigPath), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(c.ConfigPath, []byte(c.EncodeTOML()), 0o644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	return nil
}

func (c *Config) EncodeTOML() string {
	var b strings.Builder
	b.WriteString("[paths]\n")
	b.WriteString(fmt.Sprintf("bin_dir = %q\n", c.Paths.BinDir))
	b.WriteString(fmt.Sprintf("build_root = %q\n", c.Paths.BuildRoot))
	b.WriteString(fmt.Sprintf("deploy_dir = %q\n\n", c.Paths.DeployDir))
	b.WriteString("[branches]\n")
	b.WriteString(fmt.Sprintf("dev = %q\n", c.Branches.Dev))
	b.WriteString(fmt.Sprintf("stable = %q\n\n", c.Branches.Stable))
	b.WriteString("[defaults]\n")
	b.WriteString(fmt.Sprintf("repo = %q\n", c.Defaults.Repo))
	b.WriteString(fmt.Sprintf("jobs = %d\n\n", c.Defaults.Jobs))
	for _, name := range sortedRepoNames(c.Repos) {
		repo := c.Repos[name]
		b.WriteString(fmt.Sprintf("[repos.%s]\n", name))
		b.WriteString(fmt.Sprintf("git = %q\n", repo.Git))
		b.WriteString(fmt.Sprintf("path = %q\n\n", repo.Path))
	}
	return b.String()
}

func (c *Config) Set(key, value string) error {
	switch {
	case key == "paths.bin_dir":
		c.Paths.BinDir = value
	case key == "paths.build_root":
		c.Paths.BuildRoot = value
	case key == "paths.deploy_dir":
		c.Paths.DeployDir = value
	case key == "branches.dev":
		c.Branches.Dev = value
	case key == "branches.stable":
		c.Branches.Stable = value
	case key == "defaults.repo":
		c.Defaults.Repo = value
	case key == "defaults.jobs":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("defaults.jobs must be an integer: %w", err)
		}
		c.Defaults.Jobs = n
	case strings.HasPrefix(key, "repos."):
		parts := strings.Split(key, ".")
		if len(parts) != 3 {
			return fmt.Errorf("repo key must look like repos.<name>.git or repos.<name>.path")
		}
		name, field := parts[1], parts[2]
		repo := c.Repos[name]
		switch field {
		case "git":
			repo.Git = value
		case "path":
			repo.Path = value
		default:
			return fmt.Errorf("unknown repo field: %s", field)
		}
		if c.Repos == nil {
			c.Repos = map[string]Repo{}
		}
		c.Repos[name] = repo
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
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

func mergeConfig(dst, src *Config) {
	if src.Paths.BinDir != "" {
		dst.Paths.BinDir = src.Paths.BinDir
	}
	if src.Paths.BuildRoot != "" {
		dst.Paths.BuildRoot = src.Paths.BuildRoot
	}
	if src.Paths.DeployDir != "" {
		dst.Paths.DeployDir = src.Paths.DeployDir
	}
	if src.Branches.Dev != "" {
		dst.Branches.Dev = src.Branches.Dev
	}
	if src.Branches.Stable != "" {
		dst.Branches.Stable = src.Branches.Stable
	}
	if src.Defaults.Repo != "" {
		dst.Defaults.Repo = src.Defaults.Repo
	}
	if src.Defaults.Jobs > 0 {
		dst.Defaults.Jobs = src.Defaults.Jobs
	}
	if len(src.Repos) > 0 {
		if dst.Repos == nil {
			dst.Repos = map[string]Repo{}
		}
		for name, repo := range src.Repos {
			base := dst.Repos[name]
			if repo.Git != "" {
				base.Git = repo.Git
			}
			if repo.Path != "" {
				base.Path = repo.Path
			}
			dst.Repos[name] = base
		}
	}
}

func sortedRepoNames(repos map[string]Repo) []string {
	names := make([]string, 0, len(repos))
	for name := range repos {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}
