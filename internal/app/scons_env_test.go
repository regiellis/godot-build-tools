package app

import (
	"slices"
	"testing"
)

func TestSconsEnvStripsMSYSTEMForWindowsMSVCBuilds(t *testing.T) {
	environ := []string{"PATH=C:\\bin", "MSYSTEM=MINGW64", "LOCALAPPDATA=C:\\Users\\x\\AppData\\Local"}
	env, stripped := sconsEnv("windows", environ, `D:\Builds\.venv\Scripts\scons.exe`, []string{"platform=windows", "production=yes"})
	if !stripped {
		t.Fatal("expected MSYSTEM to be stripped for an MSVC scons build")
	}
	want := []string{"PATH=C:\\bin", "LOCALAPPDATA=C:\\Users\\x\\AppData\\Local"}
	if !slices.Equal(env, want) {
		t.Fatalf("env = %v, want %v", env, want)
	}
}

func TestSconsEnvLeavesOtherCasesAlone(t *testing.T) {
	withMSYS := []string{"PATH=C:\\bin", "MSYSTEM=UCRT64"}
	cases := []struct {
		name    string
		goos    string
		environ []string
		command string
		args    []string
	}{
		{"mingw build keeps MSYSTEM", "windows", withMSYS, "scons", []string{"platform=windows", "use_mingw=yes"}},
		{"non-scons command", "windows", withMSYS, "git", []string{"pull"}},
		{"not windows", "linux", withMSYS, "scons", []string{"platform=linuxbsd"}},
		{"no MSYSTEM set", "windows", []string{"PATH=C:\\bin"}, "scons", []string{"platform=windows"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env, stripped := sconsEnv(tc.goos, tc.environ, tc.command, tc.args)
			if stripped || env != nil {
				t.Fatalf("expected inherited env, got stripped=%v env=%v", stripped, env)
			}
		})
	}
}
