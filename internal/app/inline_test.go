package app

import (
	"testing"

	"github.com/Thelost77/beech/internal/layout"
)

func TestInlineEditCursorAtEndOfWrappedText(t *testing.T) {
	value := "a much longer title that wraps across the normal node width"
	lines := layout.WrapText(value, 20)
	line, x, cursor := inlineEditCursor(value, len([]rune(value)), lines)
	if line != len(lines)-1 || x != len([]rune(lines[len(lines)-1])) || cursor != " " {
		t.Fatalf("cursor = line:%d x:%d text:%q lines:%q", line, x, cursor, lines)
	}
}

func TestInlineEditCursorTracksWideGrapheme(t *testing.T) {
	value := "A你好🏳️‍🌈Z"
	lines := layout.WrapText(value, 6)
	line, x, cursor := inlineEditCursor(value, len([]rune("A你好")), lines)
	if line != 0 || x != 5 || cursor != "🏳️‍🌈" {
		t.Fatalf("cursor = line:%d x:%d text:%q lines:%q", line, x, cursor, lines)
	}
}

func TestInlineEditCursorHandlesEmptyValue(t *testing.T) {
	line, x, cursor := inlineEditCursor("", 0, []string{""})
	if line != 0 || x != 0 || cursor != " " {
		t.Fatalf("cursor = line:%d x:%d text:%q", line, x, cursor)
	}
}

func TestInlineEditCursorMovesToWrappedWordLine(t *testing.T) {
	value := "12345 next"
	lines := layout.WrapText(value, 6)
	line, x, _ := inlineEditCursor(value, len([]rune("12345 ")), lines)
	if line != 1 || x != 0 {
		t.Fatalf("cursor before wrapped word = line:%d x:%d lines:%q", line, x, lines)
	}
}
