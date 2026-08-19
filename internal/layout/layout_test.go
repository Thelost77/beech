package layout

import (
	"testing"

	"github.com/Thelost77/beech/internal/document"
	"github.com/rivo/uniseg"
)

func testDocument(t *testing.T) (*document.Document, document.NodeID, document.NodeID, document.NodeID) {
	t.Helper()
	doc := document.New("Root")
	root := doc.Roots()[0]
	a, _ := doc.AddChild(root, "First branch")
	b, _ := doc.AddSiblingAfter(a, "Second branch")
	_, _ = doc.AddChild(a, "A very long child title that needs to wrap cleanly")
	if err := doc.Validate(); err != nil {
		t.Fatal(err)
	}
	return doc, root, a, b
}

func TestComputePlacesChildrenRightOfParentWithoutOverlap(t *testing.T) {
	doc, root, a, b := testDocument(t)
	result := Compute(doc, nil, Options{MaxNodeWidth: 16})
	if len(result.Nodes) != doc.Len() {
		t.Fatalf("visible nodes = %d, want %d", len(result.Nodes), doc.Len())
	}
	if result.Nodes[a].X <= result.Nodes[root].X+result.Nodes[root].Width {
		t.Fatal("child does not sit right of parent")
	}
	if result.Nodes[a].CenterY() == result.Nodes[b].CenterY() {
		t.Fatal("siblings overlap")
	}
	if len(result.Connectors) == 0 {
		t.Fatal("no connectors")
	}
	for _, connector := range result.Connectors {
		if connector.Owner == document.NoNode {
			t.Fatal("connector has no owning node")
		}
	}
	assertNoOverlaps(t, result)
}

func TestCollapsedNodeExcludesDescendantsAndShowsCount(t *testing.T) {
	doc, _, a, _ := testDocument(t)
	result := Compute(doc, map[document.NodeID]bool{a: true}, DefaultOptions())
	if len(result.Nodes) != doc.Len()-1 {
		t.Fatalf("visible nodes = %d, want %d", len(result.Nodes), doc.Len()-1)
	}
	node := result.Nodes[a]
	if node.CollapseCount != 1 || node.CollapseLine < 0 || node.CollapseLine >= len(node.Lines) {
		t.Fatalf("collapse metadata = count %d line %d", node.CollapseCount, node.CollapseLine)
	}
}

func TestConnectorEndpointsUseTurns(t *testing.T) {
	doc, root, _, _ := testDocument(t)
	result := Compute(doc, nil, DefaultOptions())
	children := doc.Children(root)
	top := result.Nodes[children[0]]
	bottom := result.Nodes[children[1]]
	jointX := top.X - 3
	joins := make(map[[2]int]Direction)
	for _, connector := range result.Connectors {
		joins[[2]int{connector.X, connector.Y}] = connector.Join
	}
	if got := joins[[2]int{jointX, top.CenterY()}]; got&Down == 0 || got&Right == 0 || got&Up != 0 {
		t.Fatalf("top join = %04b", got)
	}
	if got := joins[[2]int{jointX, bottom.CenterY()}]; got&Up == 0 || got&Right == 0 || got&Down != 0 {
		t.Fatalf("bottom join = %04b", got)
	}
}

func TestIdeasTreeUsesCompactParentRelativeColumns(t *testing.T) {
	doc := document.New("Let's test that")
	root := doc.Roots()[0]
	dupa, _ := doc.AddChild(root, "Dupa jaś")
	fajny, _ := doc.AddChild(root, "Fajny kwas")
	_, _ = doc.AddChild(dupa, "Dupny kwas")
	_, _ = doc.AddChild(fajny, "Jebaka kwaka w dupakas")
	genialne, _ := doc.AddChild(fajny, "Genialne")
	halo, _ := doc.AddChild(genialne, "Halo")
	_, _ = doc.AddChild(fajny, "Wyśmienite")
	_, _ = doc.AddChild(fajny, "Do notatek stworzone")

	options := DefaultOptions()
	result := Compute(doc, nil, options)
	for id, node := range result.Nodes {
		for _, childID := range doc.Children(id) {
			child, visible := result.Nodes[childID]
			if !visible {
				continue
			}
			wantX := node.X + node.Width + options.LevelGap
			if child.X != wantX {
				t.Fatalf("child %q x=%d, want parent-relative x=%d", doc.Text(childID), child.X, wantX)
			}
		}
	}
	if result.Nodes[halo].X != result.Nodes[genialne].X+result.Nodes[genialne].Width+options.LevelGap {
		t.Fatal("Halo is not positioned relative to Genialne")
	}
}

func TestLongNodeDoesNotShiftUnrelatedBranchDescendants(t *testing.T) {
	doc := document.New("Root")
	root := doc.Roots()[0]
	branchA, _ := doc.AddChild(root, "Branch A")
	branchB, _ := doc.AddChild(root, "Branch B")
	unrelated, _ := doc.AddChild(branchA, "Short")
	parent, _ := doc.AddChild(branchB, "Genialne")
	halo, _ := doc.AddChild(parent, "Halo")

	before := Compute(doc, nil, DefaultOptions())
	beforeX := before.Nodes[halo].X
	if !doc.Rename(unrelated, "A very long unrelated title that should not control another branch") {
		t.Fatal("rename unrelated node")
	}
	after := Compute(doc, nil, DefaultOptions())
	if after.Nodes[halo].X != beforeX {
		t.Fatalf("unrelated long node shifted Halo from x=%d to x=%d", beforeX, after.Nodes[halo].X)
	}
}

func TestUnicodeWidthAndWrapping(t *testing.T) {
	doc := document.New("Korzeń 🌳")
	root := doc.Roots()[0]
	child, _ := doc.AddChild(root, "你好 świecie 🏳️‍🌈 bardzo-długi-tekst")
	result := Compute(doc, nil, Options{MaxNodeWidth: 12})
	for _, line := range result.Nodes[child].Lines {
		if width := uniseg.StringWidth(line); width > 12 {
			t.Fatalf("line width = %d: %q", width, line)
		}
	}
}

func assertNoOverlaps(t *testing.T, result Result) {
	t.Helper()
	for i, aID := range result.Order {
		a := result.Nodes[aID]
		for _, bID := range result.Order[i+1:] {
			b := result.Nodes[bID]
			overlapX := a.X < b.X+b.Width && b.X < a.X+a.Width
			overlapY := a.Y < b.Y+b.Height && b.Y < a.Y+a.Height
			if overlapX && overlapY {
				t.Fatalf("nodes %d and %d overlap: %#v %#v", aID, bID, a, b)
			}
		}
	}
}
