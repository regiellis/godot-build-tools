package app

import (
	"path/filepath"
	"testing"
)

func TestTemplateVersionMatchesGodotFullConfig(t *testing.T) {
	cases := []struct {
		major, minor, patch, status, module string
		want                                string
	}{
		{"4", "7", "3", "rc", "", "4.7.3.rc"},
		{"4", "7", "0", "stable", "", "4.7.stable"},
		{"4", "8", "", "dev", "", "4.8.dev"},
		{"4", "7", "2", "stable", ".mono", "4.7.2.stable.mono"},
	}
	for _, tc := range cases {
		if got := templateVersion(tc.major, tc.minor, tc.patch, tc.status, tc.module); got != tc.want {
			t.Errorf("templateVersion(%q,%q,%q,%q,%q) = %q, want %q", tc.major, tc.minor, tc.patch, tc.status, tc.module, got, tc.want)
		}
	}
}

const versionPy = `short_name = "godot"
name = "Godot Engine"
major = 4
minor = 7
patch = 3
status = "rc"
module_config = ""
website = "https://godotengine.org"
`

func TestDetectTemplateVersionReadsVersionPy(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "version.py"), versionPy)
	a := &app{}
	if got := a.detectTemplateVersion(repo, ""); got != "4.7.3.rc" {
		t.Fatalf("got %q, want 4.7.3.rc", got)
	}
}

func TestDetectTemplateVersionPrefersGeneratedHeader(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "version.py"), versionPy)
	// A build with GODOT_VERSION_STATUS=rc2 records the override only here.
	writeFile(t, filepath.Join(repo, "core", "version_generated.gen.h"), `/* THIS FILE IS GENERATED DO NOT EDIT */
#pragma once
#define GODOT_VERSION_SHORT_NAME "godot"
#define GODOT_VERSION_MAJOR 4
#define GODOT_VERSION_MINOR 7
#define GODOT_VERSION_PATCH 3
#define GODOT_VERSION_STATUS "rc2"
#define GODOT_VERSION_BUILD "custom_build"
#define GODOT_VERSION_MODULE_CONFIG ""
#define GODOT_VERSION_DOCS_URL "https://docs.godotengine.org/en/" GODOT_VERSION_DOCS_BRANCH
`)
	a := &app{}
	if got := a.detectTemplateVersion(repo, ""); got != "4.7.3.rc2" {
		t.Fatalf("got %q, want 4.7.3.rc2", got)
	}
}

func TestDetectTemplateVersionOverrideAndUnknown(t *testing.T) {
	a := &app{}
	if got := a.detectTemplateVersion(t.TempDir(), "4.5.stable"); got != "4.5.stable" {
		t.Fatalf("override: got %q", got)
	}
	if got := a.detectTemplateVersion(t.TempDir(), ""); got != "unknown" {
		t.Fatalf("empty repo: got %q, want unknown", got)
	}
}
