package app

import (
	"fmt"

	"github.com/playlogic/godot-build/internal/ui"
)

func (a *app) probe(command string, args ...string) (bool, string) {
	out, err := a.capture("", command, args...)
	if err != nil {
		if out == "" {
			return false, err.Error()
		}
		return false, firstLine(out)
	}
	if out == "" {
		return true, "ok"
	}
	return true, firstLine(out)
}

func firstLine(text string) string {
	for _, line := range splitLines(text) {
		if line != "" {
			return line
		}
	}
	return ""
}

func splitLines(text string) []string {
	lines := []string{}
	cur := ""
	for _, r := range text {
		if r == '\n' || r == '\r' {
			if cur != "" {
				lines = append(lines, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

func (a *app) cmdDoctor(global globalOptions, args []string) error {
	repo, err := a.repoPath(global.repo)
	if err != nil {
		return err
	}
	type row struct{ status, check, detail string }
	rows := []row{}
	add := func(status, check, detail string) { rows = append(rows, row{status, check, detail}) }
	required := func(ok bool, check, detail string) {
		if ok {
			add("OK", check, detail)
		} else {
			add("FAIL", check, detail)
		}
	}
	optional := func(ok bool, check, detail string) {
		if ok {
			add("OK", check, detail)
		} else {
			add("WARN", check, detail)
		}
	}

	for _, issue := range a.cfg.Validate() {
		status := "WARN"
		if issue.Level == "FAIL" {
			status = "FAIL"
		}
		add(status, "Config: "+issue.Key, issue.Message)
	}

	required(dirExists(a.cfg.Paths.BuildRoot), "Builds Dir", a.cfg.Paths.BuildRoot)
	required(dirExists(repo), "Repo Dir", repo)
	required(pathExists(filepathJoin(repo, ".git")), "Git Repo", filepathJoin(repo, ".git"))
	required(fileExists(filepathJoin(repo, "SConstruct")), "SConstruct", filepathJoin(repo, "SConstruct"))
	required(fileExists(filepathJoin(repo, "version.py")), "version.py", filepathJoin(repo, "version.py"))
	optional(fileExists(a.sconsPath()) || a.sconsPath() == "scons", "SCons CLI", a.sconsPath())
	ok, detail := a.probe("git", "--version")
	required(ok, "Git CLI", detail)
	optional(dirExists(a.cfg.Paths.DeployDir), "Deploy Dir", a.cfg.Paths.DeployDir)
	optional(dirExists(a.cfg.Paths.BinDir), "CLI Bin Dir", a.cfg.Paths.BinDir)
	optional(dirWritable(a.cfg.Paths.BinDir), "CLI Bin Writable", a.cfg.Paths.BinDir)
	optional(pathContains(a.readUserPath(), a.cfg.Paths.BinDir), "CLI Bin On User PATH", a.cfg.Paths.BinDir)
	for _, name := range sortedMapKeys(a.cliTargets(false)) {
		target := a.cliTargets(false)[name]
		optional(fileExists(a.cliShimPath(name)), "CLI Shim "+name, a.cliShimPath(name))
		optional(fileExists(target), "CLI Target "+name, target)
	}
	optional(getenvSimple("APPDATA") != "", "APPDATA", getenvSimple("APPDATA"))
	ok, detail = a.probe("where.exe", "cl")
	optional(ok, "MSVC cl.exe", detail)
	ok, detail = a.probe("where.exe", "msbuild")
	optional(ok, "MSBuild", detail)
	ok, detail = a.probe("where.exe", "dotnet")
	optional(ok, ".NET SDK", detail)
	angleRoot := a.angleDepsRoot()
	optional(dirExists(angleRoot), "ANGLE Deps Root", angleRoot)
	for _, name := range []string{"angle-x86_64-msvc", "angle-x86_64-gcc", "angle-x86_64-llvm"} {
		p := filepathJoin(angleRoot, name)
		optional(dirExists(p), "ANGLE "+name, p)
	}
	binDir := filepathJoin(repo, "bin")
	optional(dirExists(binDir), "bin/ Dir", binDir)
	if dirExists(binDir) {
		exes, _ := glob(filepathJoin(binDir, "*.exe"))
		optional(len(exes) > 0, "Built EXEs", fmt.Sprintf("%d found in %s", len(exes), binDir))
	}
	okCount, warnCount, failCount := 0, 0, 0
	tableRows := make([][]ui.Cell, 0, len(rows))
	for _, row := range rows {
		switch row.status {
		case "OK":
			okCount++
		case "WARN":
			warnCount++
		case "FAIL":
			failCount++
		}
		tableRows = append(tableRows, []ui.Cell{
			{Text: row.status, Style: map[string]string{"OK": "success", "WARN": "warning", "FAIL": "error"}[row.status]},
			{Text: row.check, Style: "key"},
			{Text: row.detail, Style: "val"},
		})
	}

	a.ui.Panel("Doctor", global.repo+"\n"+repo+"\nChecks what GBT expects locally.")
	a.ui.Table("Environment Checks", []ui.Cell{{Text: "Status"}, {Text: "Check"}, {Text: "Detail"}}, tableRows)
	summary := fmt.Sprintf("Summary: %d ok, %d warnings, %d failures", okCount, warnCount, failCount)
	if failCount > 0 {
		a.ui.Error(summary)
		a.ui.Line("")
		a.ui.Line(a.ui.Styled("muted", "Start with `gbt config validate` for config issues, then compare the missing tools against the official Godot build docs."))
		return fmt.Errorf("Doctor found blocking issues. Start with the failing rows above and fix those first.")
	}
	if warnCount > 0 {
		a.ui.Warning(summary)
		a.ui.Line(a.ui.Styled("muted", "Warnings are not always blockers, but they are usually where to look first if a build or deploy behaves strangely."))
		return nil
	}
	a.ui.Success(summary)
	return nil
}
