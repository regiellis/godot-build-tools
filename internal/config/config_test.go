package config

import "testing"

func TestUserConfigPath(t *testing.T) {
	path, err := UserConfigPath()
	if err != nil {
		t.Fatalf("UserConfigPath returned error: %v", err)
	}
	if path == "" {
		t.Fatal("UserConfigPath returned an empty path")
	}
}
