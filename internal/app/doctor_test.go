package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type doctorPayload struct {
	Repo    string            `json:"repo"`
	OK      bool              `json:"ok"`
	Summary map[string]int    `json:"summary"`
	Checks  []doctorJSONCheck `json:"checks"`
}

type doctorJSONCheck struct {
	Status string `json:"status"`
	Check  string `json:"check"`
	Detail string `json:"detail"`
}

func runDoctorJSON(t *testing.T, a *app, out *bytes.Buffer) (doctorPayload, string, error) {
	t.Helper()
	a.jsonOutput = true
	err := a.cmdDoctor(globalOptions{repo: "godot"}, nil)
	text := out.String()
	var payload doctorPayload
	if jerr := json.Unmarshal([]byte(text), &payload); jerr != nil {
		t.Fatalf("expected only JSON on stdout: %v\n%s", jerr, text)
	}
	return payload, text, err
}

func TestCmdDoctorJSON(t *testing.T) {
	a, out, _ := newTestApp(t)
	payload, text, _ := runDoctorJSON(t, a, out)
	if strings.Contains(text, "Environment Checks") {
		t.Fatalf("table output leaked into JSON:\n%s", text)
	}
	if payload.Repo != "godot" || len(payload.Checks) == 0 {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	total := payload.Summary["ok"] + payload.Summary["warnings"] + payload.Summary["failures"]
	if total != len(payload.Checks) {
		t.Fatalf("summary %v does not add up to %d checks", payload.Summary, len(payload.Checks))
	}
	for _, c := range payload.Checks {
		if c.Check == "SConstruct" && c.Status != "OK" {
			t.Fatalf("SConstruct should pass in the test repo: %#v", c)
		}
	}
}

func TestCmdDoctorJSONReportsFailures(t *testing.T) {
	a, out, _ := newTestApp(t)
	if err := os.Remove(filepath.Join(a.cfg.Repos["godot"].Path, "SConstruct")); err != nil {
		t.Fatal(err)
	}
	payload, _, err := runDoctorJSON(t, a, out)
	if err == nil {
		t.Fatal("expected an error when a required check fails")
	}
	if payload.OK || payload.Summary["failures"] == 0 {
		t.Fatalf("expected ok=false with failures: %#v", payload)
	}
	found := false
	for _, c := range payload.Checks {
		if c.Check == "SConstruct" && c.Status == "FAIL" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a FAIL SConstruct check: %#v", payload.Checks)
	}
}
