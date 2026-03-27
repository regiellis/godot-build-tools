package app

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (a *app) parseBuildFlags(fs *flag.FlagSet) (*bool, *bool, *string, *bool, *bool, *bool, *int) {
	d3d12 := fs.Bool("d3d12", false, "Enable Direct3D 12")
	vulkan := fs.Bool("vulkan", false, "Enable Vulkan")
	lto := fs.String("lto", "", "Link-time optimization level")
	llvm := fs.Bool("llvm", false, "Use LLVM/Clang")
	dev := fs.Bool("dev", false, "Dev build with assertions")
	mono := fs.Bool("mono", false, "Enable C#/Mono support")
	jobs := fs.Int("jobs", a.cfg.Defaults.Jobs, "Number of parallel SCons jobs")
	fs.IntVar(jobs, "j", a.cfg.Defaults.Jobs, "Number of parallel SCons jobs")
	return d3d12, vulkan, lto, llvm, dev, mono, jobs
}

func collectSConsFlags(d3d12, vulkan bool, lto string, llvm, dev, mono bool, jobs int) []string {
	flags := []string{fmt.Sprintf("-j%d", jobs)}
	if d3d12 {
		flags = append(flags, "d3d12=yes")
	}
	if vulkan {
		flags = append(flags, "vulkan=yes")
	}
	if lto != "" && lto != "none" {
		flags = append(flags, "lto="+lto)
	}
	if llvm {
		flags = append(flags, "use_llvm=yes")
	}
	if dev {
		flags = append(flags, "dev_build=yes")
	}
	if mono {
		flags = append(flags, "module_mono_enabled=yes")
	}
	return flags
}

func collectTemplateFlags(d3d12, vulkan bool, lto string, llvm bool, jobs int) []string {
	flags := []string{fmt.Sprintf("-j%d", jobs)}
	if d3d12 {
		flags = append(flags, "d3d12=yes")
	}
	if vulkan {
		flags = append(flags, "vulkan=yes")
	}
	if lto != "" && lto != "none" {
		flags = append(flags, "lto="+lto)
	}
	if llvm {
		flags = append(flags, "use_llvm=yes")
	}
	return flags
}

func parseKV(items []string) map[string]string {
	m := map[string]string{}
	for _, item := range items {
		if strings.HasPrefix(item, "-") || !strings.Contains(item, "=") {
			continue
		}
		parts := strings.SplitN(item, "=", 2)
		m[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return m
}

func (a *app) angleDepsRoot() string {
	if v := os.Getenv("LOCALAPPDATA"); v != "" {
		return filepath.Join(v, "Godot", "build_deps")
	}
	return filepath.Join("bin", "build_deps")
}

func (a *app) angleLibDir(allArgs []string) string {
	values := parseKV(allArgs)
	platform := values["platform"]
	if platform == "" {
		platform = "windows"
	}
	if platform != "windows" {
		return ""
	}
	angle := strings.ToLower(values["angle"])
	if angle == "no" || angle == "false" || angle == "0" {
		return ""
	}
	arch := values["arch"]
	if arch == "" {
		arch = "x86_64"
	}
	toolchain := "msvc"
	if strings.EqualFold(values["use_mingw"], "yes") {
		if strings.EqualFold(values["use_llvm"], "yes") {
			toolchain = "llvm"
		} else {
			toolchain = "gcc"
		}
	}
	return filepath.Join(a.angleDepsRoot(), fmt.Sprintf("angle-%s-%s", arch, toolchain))
}

func (a *app) ensureAngle(repo string, sconsArgs, extra []string) error {
	all := append([]string{}, sconsArgs...)
	all = append(all, extra...)
	for _, item := range all {
		if item == "-c" {
			return nil
		}
	}
	angleDir := a.angleLibDir(all)
	if angleDir == "" || dirExists(angleDir) {
		return nil
	}
	a.ui.Warning("ANGLE dependencies missing: " + angleDir)
	return a.runCommand(repo, "python", filepath.Join("misc", "scripts", "install_angle.py"))
}

func (a *app) runSCons(repo string, sconsArgs, extra []string) error {
	if err := a.ensureAngle(repo, sconsArgs, extra); err != nil {
		return err
	}
	args := append([]string{}, sconsArgs...)
	args = append(args, extra...)
	start := time.Now()
	err := a.runCommand(repo, a.sconsPath(), args...)
	elapsed := time.Since(start).Round(time.Second)
	if err != nil {
		return fmt.Errorf("build failed in %s: %w", elapsed, err)
	}
	a.ui.Success(fmt.Sprintf("Build finished in %s", elapsed))
	return nil
}

func (a *app) checkoutChannelBranch(repo string, stable bool) error {
	branch := a.cfg.Branches.Dev
	if stable {
		branch = a.cfg.Branches.Stable
	}
	a.ui.Section("Switching branch")
	a.ui.KeyValue("Branch", branch)
	if err := a.runCommand(repo, "git", "fetch", "--all", "--prune"); err != nil {
		return err
	}
	if err := a.runCommand(repo, "git", "checkout", branch); err != nil {
		return err
	}
	return a.runCommand(repo, "git", "pull", "--ff-only")
}
