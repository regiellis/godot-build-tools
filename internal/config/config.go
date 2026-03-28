package config

import (
	"errors"
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
	ConfigPath string          `json:"config_path"`
	Created    bool            `toml:"-" json:"created"`
	Paths      Paths           `toml:"paths" json:"paths"`
	Branches   Branches        `toml:"branches" json:"branches"`
	Defaults   Defaults        `toml:"defaults" json:"defaults"`
	Repos      map[string]Repo `toml:"repos" json:"repos"`
}

type Paths struct {
	BinDir    string `toml:"bin_dir" json:"bin_dir"`
	BuildRoot string `toml:"build_root" json:"build_root"`
	DeployDir string `toml:"deploy_dir" json:"deploy_dir"`
}

type Branches struct {
	Dev    string `toml:"dev" json:"dev"`
	Stable string `toml:"stable" json:"stable"`
}

type Defaults struct {
	Repo string `toml:"repo" json:"repo"`
	Jobs int    `toml:"jobs" json:"jobs"`
}

type Repo struct {
	Git  string `toml:"git" json:"git"`
	Path string `toml:"path" json:"path"`
}

type ValidationIssue struct {
	Level   string `json:"level"`
	Key     string `json:"key"`
	Message string `json:"message"`
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

	cfg := defaultConfig(configPath, home)

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

func defaultConfig(configPath, home string) *Config {
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
		Repos: map[string]Repo{
			"godot": {
				Git:  "https://github.com/godotengine/godot.git",
				Path: filepath.Join(buildRoot, "godot"),
			},
		},
	}
	return cfg
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

func (c *Config) Clone() *Config {
	clone := *c
	clone.Repos = map[string]Repo{}
	for name, repo := range c.Repos {
		clone.Repos[name] = repo
	}
	return &clone
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

func (c *Config) Unset(key string) error {
	switch {
	case key == "paths.bin_dir":
		c.Paths.BinDir = ""
	case key == "paths.build_root":
		c.Paths.BuildRoot = ""
	case key == "paths.deploy_dir":
		c.Paths.DeployDir = ""
	case key == "branches.dev":
		c.Branches.Dev = ""
	case key == "branches.stable":
		c.Branches.Stable = ""
	case key == "defaults.repo":
		c.Defaults.Repo = ""
	case key == "defaults.jobs":
		c.Defaults.Jobs = 0
	case strings.HasPrefix(key, "repos."):
		parts := strings.Split(key, ".")
		if len(parts) != 3 {
			return fmt.Errorf("repo key must look like repos.<name>.git or repos.<name>.path")
		}
		name, field := parts[1], parts[2]
		repo, ok := c.Repos[name]
		if !ok {
			return fmt.Errorf("unknown repo: %s", name)
		}
		switch field {
		case "git":
			repo.Git = ""
		case "path":
			repo.Path = ""
		default:
			return fmt.Errorf("unknown repo field: %s", field)
		}
		c.Repos[name] = repo
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

func (c *Config) SetRepo(name, gitURL, path string) {
	if c.Repos == nil {
		c.Repos = map[string]Repo{}
	}
	c.Repos[name] = Repo{Git: gitURL, Path: path}
}

func (c *Config) RemoveRepo(name string) error {
	if _, ok := c.Repos[name]; !ok {
		return fmt.Errorf("unknown repo: %s", name)
	}
	if name == c.Defaults.Repo {
		return fmt.Errorf("cannot remove %q because it is the default repo; change defaults.repo first", name)
	}
	delete(c.Repos, name)
	return nil
}

func (c *Config) Validate() []ValidationIssue {
	issues := []ValidationIssue{}
	add := func(level, key, message string) {
		issues = append(issues, ValidationIssue{Level: level, Key: key, Message: message})
	}

	if strings.TrimSpace(c.Paths.BinDir) == "" {
		add("FAIL", "paths.bin_dir", "Set a personal bin directory so GBT knows where to install gbt.exe and the command shims.")
	}
	if strings.TrimSpace(c.Paths.BuildRoot) == "" {
		add("FAIL", "paths.build_root", "Set the root folder where your Godot source checkouts live.")
	}
	if strings.TrimSpace(c.Paths.DeployDir) == "" {
		add("FAIL", "paths.deploy_dir", "Set the deploy directory where built Godot binaries should be installed.")
	}
	if strings.TrimSpace(c.Branches.Dev) == "" {
		add("FAIL", "branches.dev", "Set the branch name GBT should use for the dev channel.")
	}
	if strings.TrimSpace(c.Branches.Stable) == "" {
		add("FAIL", "branches.stable", "Set the branch name GBT should use for the stable channel.")
	}
	if strings.TrimSpace(c.Defaults.Repo) == "" {
		add("FAIL", "defaults.repo", "Choose a default repo name so commands know which checkout to use when --repo is omitted.")
	}
	if c.Defaults.Jobs <= 0 {
		add("FAIL", "defaults.jobs", "Set defaults.jobs to a number greater than 0.")
	}
	if len(c.Repos) == 0 {
		add("FAIL", "repos", "Add at least one repository entry. The official Godot repo is the default starter entry.")
	}
	if strings.TrimSpace(c.Defaults.Repo) != "" {
		if _, ok := c.Repos[c.Defaults.Repo]; !ok {
			add("FAIL", "defaults.repo", fmt.Sprintf("The default repo %q does not exist in [repos]. Add it with `gbt config repo add %s <git> <path>` or change defaults.repo.", c.Defaults.Repo, c.Defaults.Repo))
		}
	}
	for _, name := range sortedRepoNames(c.Repos) {
		repo := c.Repos[name]
		if strings.TrimSpace(repo.Git) == "" {
			add("FAIL", "repos."+name+".git", fmt.Sprintf("Repo %q is missing its git remote URL.", name))
		}
		if strings.TrimSpace(repo.Path) == "" {
			add("FAIL", "repos."+name+".path", fmt.Sprintf("Repo %q is missing its local checkout path.", name))
		}
	}
	return issues
}

func (c *Config) ValidateSet(key, value string) error {
	trial := c.Clone()
	if err := trial.Set(key, value); err != nil {
		return err
	}
	return trial.validateChangedKey(key)
}

func (c *Config) ValidateUnset(key string) error {
	trial := c.Clone()
	if err := trial.Unset(key); err != nil {
		return err
	}
	return trial.validateChangedKey(key)
}

func (c *Config) ValidateRepo(name, gitURL, path string) error {
	trial := c.Clone()
	trial.SetRepo(name, gitURL, path)
	for _, issue := range trial.Validate() {
		if issue.Level != "FAIL" {
			continue
		}
		if issue.Key == "repos."+name+".git" || issue.Key == "repos."+name+".path" {
			return errors.New(issue.Message)
		}
		if issue.Key == "defaults.repo" && trial.Defaults.Repo == name {
			return errors.New(issue.Message)
		}
	}
	return nil
}

func (c *Config) validateChangedKey(key string) error {
	for _, issue := range c.Validate() {
		if issue.Level != "FAIL" {
			continue
		}
		if issue.Key == key || strings.HasPrefix(key, "repos.") && strings.HasPrefix(issue.Key, repoKeyPrefix(key)) {
			return errors.New(issue.Message)
		}
		if key == "defaults.repo" && issue.Key == "defaults.repo" {
			return errors.New(issue.Message)
		}
	}
	return nil
}

func repoKeyPrefix(key string) string {
	parts := strings.Split(key, ".")
	if len(parts) < 2 {
		return key
	}
	return parts[0] + "." + parts[1]
}

func KnownKeys() []string {
	return []string{
		"paths.bin_dir",
		"paths.build_root",
		"paths.deploy_dir",
		"branches.dev",
		"branches.stable",
		"defaults.repo",
		"defaults.jobs",
		"repos.<name>.git",
		"repos.<name>.path",
	}
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
