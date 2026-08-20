package ui

import (
	"strings"

	"github.com/Thelost77/beech/internal/layout"
	"github.com/rivo/uniseg"
)

// CellRole describes the semantic content of terminal cells.
type CellRole uint8

const (
	RoleNone CellRole = iota
	RoleText
	RoleRoot
	RoleFirstBranch
	RoleBranch
	RoleLeaf
	RoleConnector
	RoleCollapse
	RoleTaskPending
	RoleTaskDone
	RoleStrong
	RoleCode
	RoleLink
	RoleTag
	RoleSyntax
)

// FeedbackKind describes a short-lived action pulse.
type FeedbackKind uint8

const (
	FeedbackNone FeedbackKind = iota
	FeedbackCopy
	FeedbackCreate
	FeedbackEdit
	FeedbackMove
	FeedbackFold
	FeedbackDelete
	FeedbackUndo
)

// CellStyle identifies one semantic style in the map canvas. Branch is -1
// when the cell does not belong to a colored first-level branch.
type CellStyle struct {
	Role     CellRole
	Branch   int8
	Selected bool
	Active   bool
	Feedback FeedbackKind
}

var (
	StyleNone     = CellStyle{Branch: -1}
	StyleText     = CellStyle{Role: RoleText, Branch: -1}
	StyleSelected = CellStyle{Role: RoleText, Branch: -1, Selected: true}
)

type cell struct {
	value        string
	style        CellStyle
	join         layout.Direction
	continuation bool
}

// Canvas is a viewport-sized, grapheme-aware terminal cell grid.
type Canvas struct {
	width  int
	height int
	rows   [][]cell
}

// NewCanvas creates an empty canvas.
func NewCanvas(width, height int) *Canvas {
	width = max(0, width)
	height = max(0, height)
	rows := make([][]cell, height)
	for y := range rows {
		rows[y] = make([]cell, width)
	}
	return &Canvas{width: width, height: height, rows: rows}
}

// PutConnector draws one connector cell.
func (c *Canvas) PutConnector(x, y int, join layout.Direction, style CellStyle) {
	if !c.inside(x, y) {
		return
	}
	if existing := c.rows[y][x]; existing.join != 0 {
		join |= existing.join
		style.Active = style.Active || existing.style.Active
		if style.Branch < 0 {
			style.Branch = existing.style.Branch
		}
		if style.Feedback == FeedbackNone {
			style.Feedback = existing.style.Feedback
		}
	} else {
		c.clearCell(x, y)
	}
	c.rows[y][x] = cell{value: connectorGlyph(join), style: style, join: join}
}

// PutText draws grapheme-safe text. A grapheme clipped by either viewport edge
// is omitted rather than split into invalid terminal cells.
func (c *Canvas) PutText(x, y int, text string, style CellStyle) {
	if y < 0 || y >= c.height {
		return
	}
	position := x
	graphemes := uniseg.NewGraphemes(text)
	for graphemes.Next() {
		value := graphemes.Str()
		width := graphemes.Width()
		if width <= 0 {
			continue
		}
		if position >= 0 && position+width <= c.width {
			for offset := 0; offset < width; offset++ {
				c.clearCell(position+offset, y)
			}
			c.rows[y][position] = cell{value: value, style: style}
			for offset := 1; offset < width; offset++ {
				c.rows[y][position+offset] = cell{style: style, continuation: true}
			}
		}
		position += width
		if position >= c.width {
			break
		}
	}
}

// Lines renders the fixed-width canvas. render applies terminal styling to
// groups of adjacent cells with the same semantic style.
func (c *Canvas) Lines(render func(CellStyle, string) string) []string {
	lines := make([]string, c.height)
	for y, row := range c.rows {
		var output strings.Builder
		var group strings.Builder
		currentStyle := StyleNone
		flush := func() {
			if group.Len() == 0 {
				return
			}
			output.WriteString(render(currentStyle, group.String()))
			group.Reset()
		}
		for _, current := range row {
			if current.continuation {
				continue
			}
			value := current.value
			if value == "" {
				value = " "
			}
			if group.Len() > 0 && current.style != currentStyle {
				flush()
			}
			currentStyle = current.style
			group.WriteString(value)
		}
		flush()
		lines[y] = output.String()
	}
	return lines
}

func (c *Canvas) clearCell(x, y int) {
	if !c.inside(x, y) {
		return
	}
	start := x
	for start > 0 && c.rows[y][start].continuation {
		start--
	}
	current := c.rows[y][start]
	if current.value == "" {
		c.rows[y][x] = cell{}
		return
	}
	width := max(1, uniseg.StringWidth(current.value))
	for offset := 0; offset < width && start+offset < c.width; offset++ {
		c.rows[y][start+offset] = cell{}
	}
}

func (c *Canvas) inside(x, y int) bool {
	return x >= 0 && x < c.width && y >= 0 && y < c.height
}

func connectorGlyph(join layout.Direction) string {
	switch join {
	case layout.Up | layout.Down:
		return "│"
	case layout.Left | layout.Right:
		return "─"
	case layout.Down | layout.Right:
		return "╭"
	case layout.Down | layout.Left:
		return "╮"
	case layout.Up | layout.Right:
		return "╰"
	case layout.Up | layout.Left:
		return "╯"
	case layout.Up | layout.Down | layout.Right:
		return "├"
	case layout.Up | layout.Down | layout.Left:
		return "┤"
	case layout.Left | layout.Right | layout.Down:
		return "┬"
	case layout.Left | layout.Right | layout.Up:
		return "┴"
	case layout.Up | layout.Down | layout.Left | layout.Right:
		return "┼"
	default:
		if join&(layout.Up|layout.Down) != 0 {
			return "│"
		}
		return "─"
	}
}
