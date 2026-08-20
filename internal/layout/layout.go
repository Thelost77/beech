package layout

import (
	"cmp"
	"slices"
	"strings"

	"github.com/Thelost77/beech/internal/document"
	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
)

// Direction describes which neighboring connector cells a cell joins.
type Direction uint8

const (
	Up Direction = 1 << iota
	Down
	Left
	Right
)

// Connector is one cell in the map's connector layer.
type Connector struct {
	X     int
	Y     int
	Join  Direction
	Owner document.NodeID
}

// Node is the measured position of one visible document node.
type Node struct {
	ID             document.NodeID
	X              int
	Y              int
	Width          int
	Height         int
	Depth          int
	Lines          []string
	CollapseCount  int
	CollapseLine   int
	CollapseOffset int
}

// CenterY returns the row where connectors meet a node.
func (n Node) CenterY() int {
	return n.Y + n.Height/2
}

// Result is a deterministic layout of the visible document tree.
type Result struct {
	Nodes      map[document.NodeID]Node
	Order      []document.NodeID
	Connectors []Connector
	Width      int
	Height     int
}

// Options controls map geometry.
type Options struct {
	MaxNodeWidth int
	LevelGap     int
	SiblingGap   int
	RootGap      int
}

// DefaultOptions returns the curated Beech layout defaults.
func DefaultOptions() Options {
	return Options{
		MaxNodeWidth: 42,
		LevelGap:     7,
		SiblingGap:   1,
		RootGap:      2,
	}
}

type measuredNode struct {
	id             document.NodeID
	depth          int
	lines          []string
	width          int
	height         int
	subtreeHeight  int
	children       []document.NodeID
	collapseCount  int
	collapseLine   int
	collapseOffset int
}

// Compute lays out the visible part of doc from left to right.
func Compute(doc *document.Document, collapsed map[document.NodeID]bool, options Options) Result {
	options = normalizedOptions(options)
	result := Result{Nodes: make(map[document.NodeID]Node)}
	if doc == nil || doc.Len() == 0 {
		return result
	}

	measured := make(map[document.NodeID]*measuredNode, doc.Len())
	var collect func(document.NodeID, int)
	collect = func(id document.NodeID, depth int) {
		text := doc.Text(id)
		children := doc.Children(id)
		visibleChildren := children
		collapseCount := 0
		if collapsed[id] && len(children) > 0 {
			visibleChildren = nil
			collapseCount = descendantCount(doc, id)
		}
		lines := WrapText(text, options.MaxNodeWidth)
		width := 1
		for _, line := range lines {
			width = max(width, uniseg.StringWidth(line))
		}
		collapseLine := 0
		collapseOffset := 0
		if collapseCount > 0 {
			collapseLine = len(lines) - 1
			collapseOffset = uniseg.StringWidth(lines[collapseLine])
			markerWidth := uniseg.StringWidth(collapseMarker(collapseCount, true))
			if collapseOffset+markerWidth > options.MaxNodeWidth {
				lines = append(lines, "")
				collapseLine = len(lines) - 1
				collapseOffset = 0
				markerWidth = uniseg.StringWidth(collapseMarker(collapseCount, false))
			}
			width = max(width, collapseOffset+markerWidth)
		}
		width += 2 // one cell of breathing room on each side
		item := &measuredNode{
			id:             id,
			depth:          depth,
			lines:          lines,
			width:          width,
			height:         max(1, len(lines)),
			children:       visibleChildren,
			collapseCount:  collapseCount,
			collapseLine:   collapseLine,
			collapseOffset: collapseOffset,
		}
		measured[id] = item
		for _, child := range visibleChildren {
			collect(child, depth+1)
		}
	}
	for _, root := range doc.Roots() {
		collect(root, 0)
	}

	var measureHeight func(document.NodeID) int
	measureHeight = func(id document.NodeID) int {
		item := measured[id]
		if len(item.children) == 0 {
			item.subtreeHeight = item.height
			return item.subtreeHeight
		}
		childrenHeight := 0
		for index, child := range item.children {
			if index > 0 {
				childrenHeight += options.SiblingGap
			}
			childrenHeight += measureHeight(child)
		}
		item.subtreeHeight = max(item.height, childrenHeight)
		return item.subtreeHeight
	}
	for _, root := range doc.Roots() {
		measureHeight(root)
	}

	var place func(document.NodeID, int, int)
	place = func(id document.NodeID, top, x int) {
		item := measured[id]
		y := top
		if len(item.children) > 0 {
			childrenHeight := 0
			for index, child := range item.children {
				if index > 0 {
					childrenHeight += options.SiblingGap
				}
				childrenHeight += measured[child].subtreeHeight
			}
			childTop := top + max(0, (item.subtreeHeight-childrenHeight)/2)
			childX := x + item.width + options.LevelGap
			for _, child := range item.children {
				place(child, childTop, childX)
				childTop += measured[child].subtreeHeight + options.SiblingGap
			}
			first := result.Nodes[item.children[0]].CenterY()
			last := result.Nodes[item.children[len(item.children)-1]].CenterY()
			y = (first+last)/2 - item.height/2
		}
		node := Node{
			ID:             id,
			X:              x,
			Y:              y,
			Width:          item.width,
			Height:         item.height,
			Depth:          item.depth,
			Lines:          slices.Clone(item.lines),
			CollapseCount:  item.collapseCount,
			CollapseLine:   item.collapseLine,
			CollapseOffset: item.collapseOffset,
		}
		result.Nodes[id] = node
		result.Width = max(result.Width, node.X+node.Width)
		result.Height = max(result.Height, node.Y+node.Height)
	}
	top := 0
	for index, root := range doc.Roots() {
		if index > 0 {
			top += options.RootGap
		}
		place(root, top, 0)
		top += measured[root].subtreeHeight
	}

	result.Order = make([]document.NodeID, 0, len(result.Nodes))
	for id := range result.Nodes {
		result.Order = append(result.Order, id)
	}
	slices.SortFunc(result.Order, func(a, b document.NodeID) int {
		an, bn := result.Nodes[a], result.Nodes[b]
		if order := cmp.Compare(an.CenterY(), bn.CenterY()); order != 0 {
			return order
		}
		if order := cmp.Compare(an.X, bn.X); order != 0 {
			return order
		}
		return cmp.Compare(a, b)
	})

	for id, item := range measured {
		if len(item.children) == 0 {
			continue
		}
		connectorCells := make(map[[2]int]Direction)
		parent := result.Nodes[id]
		if len(item.children) == 1 {
			child := result.Nodes[item.children[0]]
			addDirectHorizontal(connectorCells, parent.X+parent.Width, child.X-1, parent.CenterY())
		} else {
			first := result.Nodes[item.children[0]]
			last := result.Nodes[item.children[len(item.children)-1]]
			jointX := first.X - 3
			addParentHorizontal(connectorCells, parent.X+parent.Width, jointX, parent.CenterY())
			addVertical(connectorCells, jointX, first.CenterY(), last.CenterY())
			for _, childID := range item.children {
				child := result.Nodes[childID]
				addChildHorizontal(connectorCells, jointX, child.X-1, child.CenterY())
			}
		}
		for position, join := range connectorCells {
			result.Connectors = append(result.Connectors, Connector{X: position[0], Y: position[1], Join: join, Owner: id})
		}
	}
	slices.SortFunc(result.Connectors, func(a, b Connector) int {
		if order := cmp.Compare(a.Y, b.Y); order != 0 {
			return order
		}
		if order := cmp.Compare(a.X, b.X); order != 0 {
			return order
		}
		return cmp.Compare(a.Owner, b.Owner)
	})
	return result
}

func normalizedOptions(options Options) Options {
	defaults := DefaultOptions()
	if options.MaxNodeWidth <= 0 {
		options.MaxNodeWidth = defaults.MaxNodeWidth
	}
	if options.LevelGap < 4 {
		options.LevelGap = defaults.LevelGap
	}
	if options.SiblingGap < 0 {
		options.SiblingGap = defaults.SiblingGap
	}
	if options.RootGap < 0 {
		options.RootGap = defaults.RootGap
	}
	return options
}

// WrapText wraps one node title with the same display-width rules used by the
// layout engine.
func WrapText(text string, width int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{""}
	}
	wordWrapped := ansi.Wordwrap(text, width, " ")
	var result []string
	for _, line := range strings.Split(wordWrapped, "\n") {
		if uniseg.StringWidth(line) <= width {
			result = append(result, strings.TrimSpace(line))
			continue
		}
		for _, hardLine := range strings.Split(ansi.Hardwrap(line, width, false), "\n") {
			result = append(result, strings.TrimSpace(hardLine))
		}
	}
	if len(result) == 0 {
		return []string{""}
	}
	return result
}

func addDirectHorizontal(cells map[[2]int]Direction, x1, x2, y int) {
	addParentHorizontal(cells, x1, x2, y)
	cells[[2]int{x2, y}] |= Right
}

func addParentHorizontal(cells map[[2]int]Direction, x1, x2, y int) {
	if x2 < x1 {
		x1, x2 = x2, x1
	}
	for x := x1; x <= x2; x++ {
		cells[[2]int{x, y}] |= Left
		if x < x2 {
			cells[[2]int{x, y}] |= Right
		}
	}
}

func addChildHorizontal(cells map[[2]int]Direction, x1, x2, y int) {
	if x2 < x1 {
		x1, x2 = x2, x1
	}
	for x := x1; x <= x2; x++ {
		cells[[2]int{x, y}] |= Right
		if x > x1 {
			cells[[2]int{x, y}] |= Left
		}
	}
}

func addVertical(cells map[[2]int]Direction, x, y1, y2 int) {
	if y2 < y1 {
		y1, y2 = y2, y1
	}
	for y := y1; y <= y2; y++ {
		if y > y1 {
			cells[[2]int{x, y}] |= Up
		}
		if y < y2 {
			cells[[2]int{x, y}] |= Down
		}
	}
}

func descendantCount(doc *document.Document, id document.NodeID) int {
	count := 0
	for _, child := range doc.Children(id) {
		count++
		count += descendantCount(doc, child)
	}
	return count
}

func collapseMarker(count int, inline bool) string {
	prefix := "▸ "
	if inline {
		prefix = "  " + prefix
	}
	return prefix + itoa(count)
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	index := len(digits)
	for value > 0 {
		index--
		digits[index] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[index:])
}
