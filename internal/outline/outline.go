package outline

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Thelost77/beech/internal/document"
)

const (
	formatMarker    = "<!-- beech:outline v1 -->"
	collapsedMarker = "<!-- beech:collapsed -->"
)

var utf8BOM = []byte{0xef, 0xbb, 0xbf}

// Style records source conventions that should survive a save.
type Style struct {
	Newline      string
	FinalNewline bool
}

// Parsed is one native Beech Markdown document.
type Parsed struct {
	Document  *document.Document
	Collapsed map[document.NodeID]bool
	Style     Style
}

// DefaultStyle returns the native Beech Markdown style.
func DefaultStyle() Style {
	return Style{Newline: "\n", FinalNewline: true}
}

// Parse reads a Beech Markdown outline. Native indentation uses one tab per
// tree level and every node is an unordered Markdown list item.
func Parse(data []byte) (Parsed, error) {
	style, lines, err := prepareLines(data)
	if err != nil {
		return Parsed{}, err
	}

	doc := document.NewEmpty()
	collapsed := make(map[document.NodeID]bool)
	lastAtDepth := make([]document.NodeID, 0)
	seenMarker := false
	seenNode := false
	previousDepth := 0

	for index, raw := range lines {
		lineNumber := index + 1
		if strings.TrimSpace(raw) == "" {
			continue
		}
		if raw == formatMarker {
			if seenMarker || seenNode {
				return Parsed{}, fmt.Errorf("line %d: format marker must appear once before the outline", lineNumber)
			}
			seenMarker = true
			continue
		}

		depth := 0
		for depth < len(raw) && raw[depth] == '\t' {
			depth++
		}
		if depth < len(raw) && raw[depth] == ' ' {
			return Parsed{}, fmt.Errorf("line %d: use tabs, not spaces, for tree indentation", lineNumber)
		}
		content := raw[depth:]
		if !strings.HasPrefix(content, "- ") {
			return Parsed{}, fmt.Errorf("line %d: expected a Markdown list item starting with '- '", lineNumber)
		}
		text := strings.TrimSpace(strings.TrimPrefix(content, "- "))
		isCollapsed := false
		if strings.HasSuffix(text, " "+collapsedMarker) {
			text = strings.TrimSpace(strings.TrimSuffix(text, " "+collapsedMarker))
			isCollapsed = true
		}
		if text == "" {
			return Parsed{}, fmt.Errorf("line %d: node text cannot be empty", lineNumber)
		}
		if seenNode && depth > previousDepth+1 {
			return Parsed{}, fmt.Errorf("line %d: indentation skips a tree level", lineNumber)
		}
		if !seenNode && depth != 0 {
			return Parsed{}, fmt.Errorf("line %d: the first node must be a root", lineNumber)
		}

		var id document.NodeID
		if depth == 0 {
			id = doc.AppendRoot(text)
		} else {
			if depth-1 >= len(lastAtDepth) || lastAtDepth[depth-1] == document.NoNode {
				return Parsed{}, fmt.Errorf("line %d: node has no parent", lineNumber)
			}
			id, err = doc.AppendChild(lastAtDepth[depth-1], text)
			if err != nil {
				return Parsed{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
		}
		if depth >= len(lastAtDepth) {
			lastAtDepth = append(lastAtDepth, make([]document.NodeID, depth-len(lastAtDepth)+1)...)
		}
		lastAtDepth[depth] = id
		lastAtDepth = lastAtDepth[:depth+1]
		if isCollapsed {
			// The node may gain children later in the parse. Keep the marker and
			// validate it after the complete tree is known.
			collapsed[id] = true
		}
		seenNode = true
		previousDepth = depth
	}

	if !seenNode {
		doc = document.New("New map")
	}
	for id := range collapsed {
		if len(doc.Children(id)) == 0 {
			delete(collapsed, id)
		}
	}
	if err := doc.Validate(); err != nil {
		return Parsed{}, fmt.Errorf("invalid outline: %w", err)
	}
	return Parsed{Document: doc, Collapsed: collapsed, Style: style}, nil
}

// Serialize writes a native Beech Markdown outline using tabs for depth.
func Serialize(doc *document.Document, collapsed map[document.NodeID]bool, style Style) ([]byte, error) {
	if err := doc.Validate(); err != nil {
		return nil, err
	}
	if style.Newline == "" {
		style.Newline = "\n"
	}
	lines := []string{formatMarker, ""}
	var serializeErr error
	var appendNode func(document.NodeID, int)
	appendNode = func(id document.NodeID, depth int) {
		if serializeErr != nil {
			return
		}
		text := doc.Text(id)
		if strings.HasSuffix(strings.TrimSpace(text), collapsedMarker) {
			serializeErr = fmt.Errorf("node %d ends with reserved Beech metadata", id)
			return
		}
		line := strings.Repeat("\t", depth) + "- " + text
		if collapsed[id] && len(doc.Children(id)) > 0 {
			line += " " + collapsedMarker
		}
		lines = append(lines, line)
		for _, child := range doc.Children(id) {
			appendNode(child, depth+1)
		}
	}
	for _, root := range doc.Roots() {
		appendNode(root, 0)
	}
	if serializeErr != nil {
		return nil, serializeErr
	}
	output := strings.Join(lines, style.Newline)
	if style.FinalNewline {
		output += style.Newline
	}
	return []byte(output), nil
}

func prepareLines(data []byte) (Style, []string, error) {
	style := DefaultStyle()
	if !utf8.Valid(data) {
		return style, nil, errors.New("outline is not valid UTF-8")
	}
	data = bytes.TrimPrefix(data, utf8BOM)
	if bytes.Contains(data, []byte("\r\n")) {
		style.Newline = "\r\n"
	}
	style.FinalNewline = bytes.HasSuffix(data, []byte("\n"))
	normalized := strings.ReplaceAll(string(data), "\r\n", "\n")
	return style, strings.Split(normalized, "\n"), nil
}
