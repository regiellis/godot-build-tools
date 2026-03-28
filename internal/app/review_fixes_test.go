package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseRegistryPathQueryPreservesSpaces(t *testing.T) {
	out := "HKEY_CURRENT_USER\\Environment\r\n    Path    REG_EXPAND_SZ    C:\\Program Files\\Go\\bin;C:\\Users\\me\\bin\r\n"
	got := parseRegistryPathQuery(out)
	want := `C:\Program Files\Go\bin;C:\Users\me\bin`
	if got != want {
		t.Fatalf("unexpected parsed path: %q", got)
	}
}

func TestErrorsIsUsesWrappedErrors(t *testing.T) {
	err := &exec.Error{Name: "missing", Err: exec.ErrNotFound}
	wrapped := fmt.Errorf("outer: %w", err)
	if !errorsIs(wrapped, exec.ErrNotFound) {
		t.Fatal("expected errorsIs to match exec.ErrNotFound through wrapping")
	}
}

func TestSamePathHandlesWindowsCase(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific path semantics")
	}
	if !samePath(`C:\Users\Playlogic\bin`, `c:\users\playlogic\bin`) {
		t.Fatal("expected samePath to treat Windows paths case-insensitively")
	}
}

func TestCopyFilePreservesModeAndContents(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src.bin")
	dst := filepath.Join(t.TempDir(), "nested", "dst.bin")
	if err := os.WriteFile(src, []byte("hello"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile returned error: %v", err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("unexpected copy contents: %q", string(data))
	}
	st, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && st.Mode().Perm() != 0o755 {
		t.Fatalf("expected copied mode 0755, got %o", st.Mode().Perm())
	}
}

func TestPathContainsNormalizesQuotedEntries(t *testing.T) {
	pathValue := `"C:\Users\me\bin";C:\Tools`
	if !pathContains(pathValue, `C:\Users\me\bin\`) {
		t.Fatal("expected pathContains to normalize quotes and trailing slashes")
	}
}
