package outline

import (
	"fmt"
	"strings"

	"github.com/Thelost77/beech/internal/document"
)

type legacyLine struct {
	number int
	indent string
	text   string
}

// ImportHMM imports one legacy h-m-m indentation file into Beech's in-memory
// document. Beech never writes the legacy format.
func ImportHMM(data []byte) (Parsed, error) {
	style, rawLines, err := prepareLines(data)
	if err != nil {
		return Parsed{}, err
	}
	lines := make([]legacyLine, 0, len(rawLines))
	indentKind := byte(0)
	minimum := -1
	for index, raw := range rawLines {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		prefixLen := 0
		for prefixLen < len(raw) && (raw[prefixLen] == ' ' || raw[prefixLen] == '\t') {
			if indentKind == 0 {
				indentKind = raw[prefixLen]
			}
			if raw[prefixLen] != indentKind {
				return Parsed{}, fmt.Errorf("line %d: mixed tabs and spaces in indentation", index+1)
			}
			prefixLen++
		}
		indent := raw[:prefixLen]
		text := strings.TrimSpace(raw[prefixLen:])
		if text == "" {
			continue
		}
		if minimum < 0 || len(indent) < minimum {
			minimum = len(indent)
		}
		lines = append(lines, legacyLine{number: index + 1, indent: indent, text: text})
	}

	if len(lines) == 0 {
		return Parsed{Document: document.New("New map"), Collapsed: make(map[document.NodeID]bool), Style: style}, nil
	}

	base := minimum
	levels := []int{0}
	lastAtDepth := make([]document.NodeID, 0)
	doc := document.NewEmpty()
	previousIndent := 0
	previousDepth := 0

	for index, line := range lines {
		indent := len(line.indent) - base
		if indent < 0 {
			return Parsed{}, fmt.Errorf("line %d: indentation is shallower than the document root", line.number)
		}
		depth := previousDepth
		switch {
		case index == 0:
			if indent != 0 {
				return Parsed{}, fmt.Errorf("line %d: invalid root indentation", line.number)
			}
			depth = 0
		case indent > previousIndent:
			depth = previousDepth + 1
			levels = append(levels[:depth], indent)
		case indent == previousIndent:
			depth = previousDepth
		default:
			depth = -1
			for candidate, value := range levels {
				if value == indent {
					depth = candidate
					break
				}
			}
			if depth < 0 {
				return Parsed{}, fmt.Errorf("line %d: indentation does not match an earlier level", line.number)
			}
		}

		var id document.NodeID
		if depth == 0 {
			id = doc.AppendRoot(line.text)
		} else {
			if depth-1 >= len(lastAtDepth) || lastAtDepth[depth-1] == document.NoNode {
				return Parsed{}, fmt.Errorf("line %d: node has no parent", line.number)
			}
			id, err = doc.AppendChild(lastAtDepth[depth-1], line.text)
			if err != nil {
				return Parsed{}, fmt.Errorf("line %d: %w", line.number, err)
			}
		}
		if depth >= len(lastAtDepth) {
			lastAtDepth = append(lastAtDepth, make([]document.NodeID, depth-len(lastAtDepth)+1)...)
		}
		lastAtDepth[depth] = id
		lastAtDepth = lastAtDepth[:depth+1]
		previousIndent = indent
		previousDepth = depth
	}

	if err := doc.Validate(); err != nil {
		return Parsed{}, fmt.Errorf("invalid outline: %w", err)
	}
	return Parsed{Document: doc, Collapsed: make(map[document.NodeID]bool), Style: style}, nil
}
