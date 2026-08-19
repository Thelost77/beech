package document

import (
	"reflect"
	"testing"
)

func TestStructuralOperations(t *testing.T) {
	doc := New("Root")
	root := doc.Roots()[0]
	a, ok := doc.AddChild(root, "A")
	if !ok {
		t.Fatal("add A")
	}
	b, _ := doc.AddSiblingAfter(a, "B")
	b1, _ := doc.AddChild(b, "B1")

	if !doc.MoveSibling(b, -1) {
		t.Fatal("move B before A")
	}
	if got := doc.Children(root); !reflect.DeepEqual(got, []NodeID{b, a}) {
		t.Fatalf("children after move = %v", got)
	}
	if !doc.Promote(b1) {
		t.Fatal("promote B1")
	}
	if got := doc.Children(root); !reflect.DeepEqual(got, []NodeID{b, b1, a}) {
		t.Fatalf("children after promote = %v", got)
	}
	if !doc.Demote(a) {
		t.Fatal("demote A")
	}
	if got := doc.Children(b1); !reflect.DeepEqual(got, []NodeID{a}) {
		t.Fatalf("B1 children = %v", got)
	}
	if err := doc.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestCopyPasteCreatesIndependentSubtree(t *testing.T) {
	doc := New("Root")
	root := doc.Roots()[0]
	a, _ := doc.AddChild(root, "A")
	_, _ = doc.AddChild(a, "A1")

	branch, ok := doc.CopyBranch(a)
	if !ok {
		t.Fatal("copy branch")
	}
	copyID, ok := doc.PasteSibling(a, branch)
	if !ok {
		t.Fatal("paste branch")
	}
	if copyID == a {
		t.Fatal("paste reused source ID")
	}
	if !doc.Rename(copyID, "A copy") {
		t.Fatal("rename copy")
	}
	if got := doc.Text(a); got != "A" {
		t.Fatalf("source changed to %q", got)
	}
	if got := doc.Text(doc.Children(copyID)[0]); got != "A1" {
		t.Fatalf("copied child = %q", got)
	}
	if err := doc.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestFinalRootCannotBeDeleted(t *testing.T) {
	doc := New("Root")
	if doc.Delete(doc.Roots()[0]) {
		t.Fatal("deleted final root")
	}
	if err := doc.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestTextRemovesTerminalControls(t *testing.T) {
	doc := New("Root\x1b[31m red")
	root := doc.Roots()[0]
	if got := doc.Text(root); got != "Root[31m red" {
		t.Fatalf("text = %q", got)
	}
}

func TestCloneIsIndependent(t *testing.T) {
	doc := New("Root")
	root := doc.Roots()[0]
	clone := doc.Clone()
	if !clone.Rename(root, "Changed") {
		t.Fatal("rename clone")
	}
	if got := doc.Text(root); got != "Root" {
		t.Fatalf("original changed to %q", got)
	}
}
