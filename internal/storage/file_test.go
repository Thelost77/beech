package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveCreatesAndReplacesAtomically(t *testing.T) {
	path := filepath.Join(t.TempDir(), "map.hmm")
	fp, err := Save(path, []byte("Root\n"), Fingerprint{}, 0o640)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Fingerprint != fp || string(loaded.Data) != "Root\n" {
		t.Fatalf("loaded = %#v %q", loaded.Fingerprint, loaded.Data)
	}
	if loaded.Mode != 0o640 {
		t.Fatalf("mode = %o", loaded.Mode)
	}

	if _, err := Save(path, []byte("Changed\n"), fp, loaded.Mode); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "Changed\n" {
		t.Fatalf("data = %q", data)
	}
}

func TestSaveRejectsExternalChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "map.hmm")
	fp, err := Save(path, []byte("Root\n"), Fingerprint{}, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("External\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Save(path, []byte("Beech\n"), fp, 0o644); !errors.Is(err, ErrConflict) {
		t.Fatalf("error = %v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "External\n" {
		t.Fatalf("external data overwritten: %q", data)
	}
}

func TestLoadResolvesSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.hmm")
	link := filepath.Join(dir, "link.hmm")
	if err := os.WriteFile(target, []byte("Root\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	loaded, err := Load(link)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Path != target {
		t.Fatalf("resolved path = %q, want %q", loaded.Path, target)
	}
}
