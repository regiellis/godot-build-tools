package app

import "fmt"

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
	for _, row := range rows {
		switch row.status {
		case "OK":
			okCount++
		case "WARN":
			warnCount++
		case "FAIL":
			failCount++
		}
		a.ui.Line(fmt.Sprintf("%s | %s | %s", row.status, row.check, row.detail))
	}
	a.ui.Line("")
	if failCount > 0 {
		return fmt.Errorf("Summary: %d ok, %d warnings, %d failures", okCount, warnCount, failCount)
	}
	a.ui.Success(fmt.Sprintf("Summary: %d ok, %d warnings, 0 failures", okCount, warnCount))
	return nil
}
