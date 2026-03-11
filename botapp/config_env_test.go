package botapp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv(t *testing.T) {
	t.Setenv("EXISTING", "from_env")
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "# comment\n" +
		"A=1\n" +
		"export B=2\n" +
		"C=three # comment\n" +
		"D='hello world'\n" +
		"E=\"line\\nvalue\"\n" +
		"EXISTING=from_file\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := loadDotEnv(path); err != nil {
		t.Fatal(err)
	}

	assertEnv(t, "A", "1")
	assertEnv(t, "B", "2")
	assertEnv(t, "C", "three")
	assertEnv(t, "D", "hello world")
	assertEnv(t, "E", "line\nvalue")
	assertEnv(t, "EXISTING", "from_env")
}

func TestLoadDotEnvNotExist(t *testing.T) {
	if err := loadDotEnv(filepath.Join(t.TempDir(), ".env")); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestLoadDotEnvInvalidQuoted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("A=\"unterminated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := loadDotEnv(path); err == nil {
		t.Fatal("expected error")
	}
}

func assertEnv(t *testing.T, key, want string) {
	t.Helper()
	if got := os.Getenv(key); got != want {
		t.Fatalf("%s: got %q want %q", key, got, want)
	}
}
