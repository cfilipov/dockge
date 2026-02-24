package compose

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGlobalEnvArgs(t *testing.T) {
	t.Run("no global.env", func(t *testing.T) {
		dir := t.TempDir()
		stackName := "mystack"
		os.MkdirAll(filepath.Join(dir, stackName), 0755)

		args := GlobalEnvArgs(dir, stackName)
		if args != nil {
			t.Errorf("expected nil, got %v", args)
		}
	})

	t.Run("global.env exists, no stack .env", func(t *testing.T) {
		dir := t.TempDir()
		stackName := "mystack"
		os.MkdirAll(filepath.Join(dir, stackName), 0755)
		os.WriteFile(filepath.Join(dir, "global.env"), []byte("FOO=bar"), 0644)

		args := GlobalEnvArgs(dir, stackName)
		expected := []string{"--env-file", "../global.env"}
		if len(args) != len(expected) {
			t.Fatalf("expected %v, got %v", expected, args)
		}
		for i, v := range expected {
			if args[i] != v {
				t.Errorf("args[%d] = %q, want %q", i, args[i], v)
			}
		}
	})

	t.Run("both global.env and stack .env exist", func(t *testing.T) {
		dir := t.TempDir()
		stackName := "mystack"
		os.MkdirAll(filepath.Join(dir, stackName), 0755)
		os.WriteFile(filepath.Join(dir, "global.env"), []byte("FOO=bar"), 0644)
		os.WriteFile(filepath.Join(dir, stackName, ".env"), []byte("BAZ=qux"), 0644)

		args := GlobalEnvArgs(dir, stackName)
		expected := []string{"--env-file", "../global.env", "--env-file", "./.env"}
		if len(args) != len(expected) {
			t.Fatalf("expected %v, got %v", expected, args)
		}
		for i, v := range expected {
			if args[i] != v {
				t.Errorf("args[%d] = %q, want %q", i, args[i], v)
			}
		}
	})
}
