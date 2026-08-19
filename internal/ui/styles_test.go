package ui

import "testing"

func TestFeedbackColorsRemainDistinct(t *testing.T) {
	styles := DefaultStyles()
	copyColor := styles.feedbackColor(FeedbackCopy)
	createColor := styles.feedbackColor(FeedbackCreate)
	moveColor := styles.feedbackColor(FeedbackMove)
	if copyColor == createColor || createColor == moveColor || copyColor == moveColor {
		t.Fatalf("feedback colors are not distinct: copy=%v create=%v move=%v", copyColor, createColor, moveColor)
	}
}

func TestStructuralColorGrammar(t *testing.T) {
	styles := DefaultStyles()
	firstLevel := styles.cellStyle(CellStyle{Role: RoleFirstBranch, Branch: 1})
	internal := styles.cellStyle(CellStyle{Role: RoleBranch, Branch: 1})
	leaf := styles.cellStyle(CellStyle{Role: RoleLeaf, Branch: 1})
	connector := styles.cellStyle(CellStyle{Role: RoleConnector, Branch: 1})

	if firstLevel.GetForeground() != styles.branchColor(1) || !firstLevel.GetBold() {
		t.Fatalf("first-level style = color %v bold %v", firstLevel.GetForeground(), firstLevel.GetBold())
	}
	if internal.GetForeground() != styles.foreground || !internal.GetBold() {
		t.Fatalf("internal style = color %v bold %v", internal.GetForeground(), internal.GetBold())
	}
	if leaf.GetForeground() != styles.foreground || leaf.GetBold() {
		t.Fatalf("leaf style = color %v bold %v", leaf.GetForeground(), leaf.GetBold())
	}
	if connector.GetForeground() != styles.branchColor(1) {
		t.Fatalf("connector color = %v, want %v", connector.GetForeground(), styles.branchColor(1))
	}
}

func TestBranchPaletteCyclesDeterministically(t *testing.T) {
	styles := DefaultStyles()
	if styles.branchColor(0) == styles.branchColor(1) {
		t.Fatal("adjacent branches use the same color")
	}
	if styles.branchColor(0) != styles.branchColor(BranchColorCount) {
		t.Fatal("branch palette does not cycle")
	}
}
