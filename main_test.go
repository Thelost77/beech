package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenInitialExistingMarkdownFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ideas.md")
	data := []byte("<!-- beech:outline v1 -->\n\n- Ideas\n\t- One <!-- beech:collapsed -->\n\t\t- Detail\n")
	if err := os.WriteFile(path, data, 0o640); err != nil {
		t.Fatal(err)
	}
	initial, err := openInitial(path)
	if err != nil {
		t.Fatal(err)
	}
	if initial.Document.Len() != 3 || initial.Path == "" || initial.Dirty || initial.Imported {
		t.Fatalf("initial = path %q nodes %d dirty %v imported %v", initial.Path, initial.Document.Len(), initial.Dirty, initial.Imported)
	}
	child := initial.Document.Children(initial.Document.Roots()[0])[0]
	if !initial.Collapsed[child] {
		t.Fatal("collapsed state was not loaded")
	}
	if initial.FileMode != 0o640 {
		t.Fatalf("mode = %o", initial.FileMode)
	}
}

func TestOpenInitialImportsHMMAsUnsavedMarkdown(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "ideas.hmm")
	data := []byte("Ideas\n\tOne\n")
	if err := os.WriteFile(source, data, 0o640); err != nil {
		t.Fatal(err)
	}
	initial, err := openInitial(source)
	if err != nil {
		t.Fatal(err)
	}
	wantTarget := filepath.Join(dir, "ideas.md")
	if initial.Path != "" || initial.SuggestedPath != wantTarget || !initial.Imported || !initial.Dirty {
		t.Fatalf("path=%q suggested=%q imported=%v dirty=%v", initial.Path, initial.SuggestedPath, initial.Imported, initial.Dirty)
	}
	if initial.Document.Len() != 2 {
		t.Fatalf("nodes = %d", initial.Document.Len())
	}
	after, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(data) {
		t.Fatalf("legacy source changed: %q", after)
	}
}

func TestOpenInitialNewPathAddsMarkdownExtension(t *testing.T) {
	path := filepath.Join(t.TempDir(), "project-plan")
	initial, err := openInitial(path)
	if err != nil {
		t.Fatal(err)
	}
	if initial.Path != path+".md" {
		t.Fatalf("path = %q", initial.Path)
	}
	root := initial.Document.Roots()[0]
	if got := initial.Document.Text(root); got != "project-plan" {
		t.Fatalf("root = %q", got)
	}
	if !initial.Dirty {
		t.Fatal("new named document should be dirty")
	}
}

func TestOpenInitialRejectsNewHMMPath(t *testing.T) {
	_, err := openInitial(filepath.Join(t.TempDir(), "new.hmm"))
	if err == nil || !strings.Contains(err.Error(), ".md extension") {
		t.Fatalf("error = %v", err)
	}
}

func TestOpenInitialUntitledStartsClean(t *testing.T) {
	initial, err := openInitial("")
	if err != nil {
		t.Fatal(err)
	}
	if initial.Dirty || len(initial.SavedData) == 0 {
		t.Fatalf("dirty=%v saved=%q", initial.Dirty, initial.SavedData)
	}
}
