package ui

import (
	"strings"
	"testing"

	"github.com/Thelost77/beech/internal/layout"
	"github.com/rivo/uniseg"
)

func TestCanvasPreservesUnicodeCellWidth(t *testing.T) {
	canvas := NewCanvas(12, 2)
	canvas.PutText(1, 0, "你好 🌳", StyleText)
	canvas.PutText(0, 1, "🏳️‍🌈 map", StyleSelected)
	lines := canvas.Lines(func(_ CellStyle, text string) string { return text })
	for i, line := range lines {
		if width := uniseg.StringWidth(line); width != 12 {
			t.Fatalf("line %d width = %d: %q", i, width, line)
		}
	}
	if !strings.Contains(lines[0], "你好") || !strings.Contains(lines[1], "🏳️‍🌈") {
		t.Fatalf("lines = %#v", lines)
	}
}

func TestTextOverwritesConnector(t *testing.T) {
	canvas := NewCanvas(8, 1)
	canvas.PutConnector(2, 0, layout.Left|layout.Right, CellStyle{Role: RoleConnector, Branch: -1})
	canvas.PutText(1, 0, "node", StyleSelected)
	line := canvas.Lines(func(_ CellStyle, text string) string { return text })[0]
	if strings.Contains(line, "─") {
		t.Fatalf("connector remained under text: %q", line)
	}
}

func TestOverwritingWideGraphemeClearsAllOccupiedCells(t *testing.T) {
	canvas := NewCanvas(6, 1)
	canvas.PutText(0, 0, "⸺", StyleText)
	canvas.PutText(1, 0, "x", StyleText)
	line := canvas.Lines(func(_ CellStyle, text string) string { return text })[0]
	if strings.Contains(line, "⸺") {
		t.Fatalf("wide grapheme remained after overwrite: %q", line)
	}
	if width := uniseg.StringWidth(line); width != 6 {
		t.Fatalf("line width = %d: %q", width, line)
	}
}

func TestConnectorCellsMergeDirections(t *testing.T) {
	canvas := NewCanvas(5, 1)
	style := CellStyle{Role: RoleConnector, Branch: -1}
	canvas.PutConnector(2, 0, layout.Left|layout.Right, style)
	canvas.PutConnector(2, 0, layout.Up|layout.Down, style)
	line := canvas.Lines(func(_ CellStyle, text string) string { return text })[0]
	if !strings.Contains(line, "┼") {
		t.Fatalf("connector directions did not merge: %q", line)
	}
}

func TestConnectorGlyphs(t *testing.T) {
	tests := map[layout.Direction]string{
		layout.Left | layout.Right:                           "─",
		layout.Up | layout.Down:                              "│",
		layout.Up | layout.Down | layout.Left | layout.Right: "┼",
		layout.Down | layout.Right:                           "╭",
	}
	for join, want := range tests {
		if got := connectorGlyph(join); got != want {
			t.Errorf("glyph(%d) = %q, want %q", join, got, want)
		}
	}
}
