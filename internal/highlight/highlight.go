package highlight

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Role identifies one inline Markdown token class.
type Role uint8

const (
	Plain Role = iota
	TaskPending
	TaskDone
	Strong
	Code
	Link
	Tag
	Syntax
)

// Span is one styled fragment. Joining all span text recreates the input.
type Span struct {
	Text string
	Role Role
}

// Parse highlights Beech's supported one-line Markdown tokens. Malformed or
// unsupported syntax remains plain text.
func Parse(text string) []Span {
	var spans []Span
	position := 0
	if strings.HasPrefix(text, "[ ]") {
		spans = append(spans, Span{Text: text[:3], Role: TaskPending})
		position = 3
	} else if strings.HasPrefix(text, "[x]") || strings.HasPrefix(text, "[X]") {
		spans = append(spans, Span{Text: text[:3], Role: TaskDone})
		position = 3
	}

	plainStart := position
	flushPlain := func(end int) {
		if end > plainStart {
			spans = append(spans, Span{Text: text[plainStart:end], Role: Plain})
		}
	}
	for position < len(text) {
		if text[position] == '\\' && position+1 < len(text) {
			_, escapedSize := utf8.DecodeRuneInString(text[position+1:])
			position += 1 + escapedSize
			continue
		}
		if strings.HasPrefix(text[position:], "**") {
			if closing := strings.Index(text[position+2:], "**"); closing >= 0 {
				closing += position + 2
				flushPlain(position)
				spans = append(spans,
					Span{Text: "**", Role: Syntax},
					Span{Text: text[position+2 : closing], Role: Strong},
					Span{Text: "**", Role: Syntax},
				)
				position = closing + 2
				plainStart = position
				continue
			}
		}
		if text[position] == '`' {
			if closing := strings.IndexByte(text[position+1:], '`'); closing >= 0 {
				closing += position + 1
				flushPlain(position)
				spans = append(spans, Span{Text: text[position : closing+1], Role: Code})
				position = closing + 1
				plainStart = position
				continue
			}
		}
		if text[position] == '[' {
			if end := markdownLinkEnd(text, position); end > position {
				flushPlain(position)
				spans = append(spans, Span{Text: text[position:end], Role: Link})
				position = end
				plainStart = position
				continue
			}
		}
		if text[position] == '#' && (position == 0 || unicode.IsSpace(rune(text[position-1]))) {
			if end := tagEnd(text, position+1); end > position+1 {
				flushPlain(position)
				spans = append(spans, Span{Text: text[position:end], Role: Tag})
				position = end
				plainStart = position
				continue
			}
		}
		_, size := utf8.DecodeRuneInString(text[position:])
		if size == 0 {
			break
		}
		position += size
	}
	flushPlain(len(text))
	if len(spans) == 0 {
		return []Span{{Text: text, Role: Plain}}
	}
	return spans
}

func markdownLinkEnd(text string, start int) int {
	labelEnd := strings.IndexByte(text[start+1:], ']')
	if labelEnd < 0 {
		return -1
	}
	labelEnd += start + 1
	if labelEnd+1 >= len(text) || text[labelEnd+1] != '(' {
		return -1
	}
	urlEnd := strings.IndexByte(text[labelEnd+2:], ')')
	if urlEnd < 0 {
		return -1
	}
	return labelEnd + 2 + urlEnd + 1
}

func tagEnd(text string, start int) int {
	position := start
	for position < len(text) {
		r, size := utf8.DecodeRuneInString(text[position:])
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-') {
			break
		}
		position += size
	}
	return position
}
