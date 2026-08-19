package app

import (
	"strings"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

type inlineCluster struct {
	text      string
	runeStart int
	runeEnd   int
}

// inlineEditCursor maps the text input's rune cursor into the exact wrapped
// lines produced by the layout engine.
func inlineEditCursor(value string, runePosition int, lines []string) (line, x int, cursorText string) {
	runes := []rune(value)
	runePosition = min(max(0, runePosition), len(runes))
	cursorRune := runePosition
	cursorText = " "

	graphemes := uniseg.NewGraphemes(value)
	runeIndex := 0
	for graphemes.Next() {
		text := graphemes.Str()
		count := utf8.RuneCountInString(text)
		cluster := inlineCluster{text: text, runeStart: runeIndex, runeEnd: runeIndex + count}
		if runePosition >= cluster.runeStart && runePosition < cluster.runeEnd {
			cursorRune = cluster.runeStart
			cursorText = cluster.text
			break
		}
		runeIndex += count
	}
	cursorByte := len(string(runes[:cursorRune]))
	if len(lines) == 0 {
		return 0, 0, cursorText
	}

	searchFrom := 0
	for index, wrappedLine := range lines {
		if wrappedLine == "" {
			if cursorByte <= searchFrom {
				return index, 0, cursorText
			}
			continue
		}
		relative := strings.Index(value[searchFrom:], wrappedLine)
		if relative < 0 {
			continue
		}
		start := searchFrom + relative
		end := start + len(wrappedLine)
		if cursorByte < start {
			return index, 0, cursorText
		}
		if cursorByte <= end {
			return index, uniseg.StringWidth(value[start:cursorByte]), cursorText
		}
		searchFrom = end
	}
	last := len(lines) - 1
	return last, uniseg.StringWidth(lines[last]), cursorText
}
