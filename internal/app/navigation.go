package app

import (
	"math"
	"slices"

	"github.com/Thelost77/beech/internal/document"
	"github.com/Thelost77/beech/internal/layout"
)

func (m *Model) rebuildLayout() {
	m.layout = layout.Compute(m.doc, m.collapsed, layout.DefaultOptions())
	if _, ok := m.layout.Nodes[m.selected]; !ok && len(m.layout.Order) > 0 {
		m.selected = m.layout.Order[0]
	}
	m.clampViewport()
}

func (m *Model) selectParent() {
	parent := m.doc.Parent(m.selected)
	if parent == document.NoNode {
		return
	}
	m.selected = parent
	m.ensureSelectedVisible()
}

func (m *Model) selectChild() {
	children := m.doc.Children(m.selected)
	if len(children) == 0 {
		return
	}
	if m.collapsed[m.selected] {
		before := m.snapshot()
		m.collapsed[m.selected] = false
		m.pushUndo(before)
		m.changed()
	}
	current := m.layout.Nodes[m.selected]
	best := document.NoNode
	bestDistance := math.MaxInt
	for _, child := range children {
		node, visible := m.layout.Nodes[child]
		if !visible {
			continue
		}
		distance := abs(node.CenterY() - current.CenterY())
		if distance < bestDistance {
			best = child
			bestDistance = distance
		}
	}
	if best != document.NoNode {
		m.selected = best
		m.ensureSelectedVisible()
	}
}

func (m *Model) selectVertical(direction int) {
	current, ok := m.layout.Nodes[m.selected]
	if !ok {
		return
	}
	siblings := m.doc.Roots()
	if parent := m.doc.Parent(m.selected); parent != document.NoNode {
		siblings = m.doc.Children(parent)
	}
	if index := slices.Index(siblings, m.selected); index >= 0 {
		candidate := index + direction
		if candidate >= 0 && candidate < len(siblings) {
			if _, visible := m.layout.Nodes[siblings[candidate]]; visible {
				m.selected = siblings[candidate]
				m.ensureSelectedVisible()
				return
			}
		}
	}

	best := document.NoNode
	bestScore := math.MaxInt
	for id, node := range m.layout.Nodes {
		deltaY := node.CenterY() - current.CenterY()
		if direction < 0 && deltaY >= 0 || direction > 0 && deltaY <= 0 {
			continue
		}
		deltaX := node.X - current.X
		score := deltaY*deltaY*16 + deltaX*deltaX
		if score < bestScore {
			best = id
			bestScore = score
		}
	}
	if best != document.NoNode {
		m.selected = best
		m.ensureSelectedVisible()
	}
}

func (m *Model) centerSelected() {
	node, ok := m.layout.Nodes[m.selected]
	if !ok {
		return
	}
	m.viewportX = max(0, node.X+node.Width/2-m.contentWidth()/2)
	m.viewportY = max(0, node.CenterY()-m.contentHeight()/2)
}

func (m *Model) clampViewport() {
	m.clampViewportTo(m.layout)
}

func (m *Model) clampViewportTo(result layout.Result) {
	m.viewportX = min(max(0, m.viewportX), max(0, result.Width-m.contentWidth()))
	m.viewportY = min(max(0, m.viewportY), max(0, result.Height-m.contentHeight()))
}

func (m *Model) mapOffsetY() int {
	return max(0, (m.contentHeight()-m.layout.Height)/2)
}

func (m *Model) ensureSelectedVisible() {
	m.ensureSelectedVisibleIn(m.layout)
}

func (m *Model) ensureSelectedVisibleIn(result layout.Result) {
	node, ok := result.Nodes[m.selected]
	if !ok || m.contentWidth() <= 0 || m.contentHeight() <= 0 {
		return
	}
	marginX := 2
	marginY := 1
	if node.X-m.viewportX < marginX {
		m.viewportX = max(0, node.X-marginX)
	}
	if node.X+node.Width-m.viewportX > m.contentWidth()-marginX {
		m.viewportX = max(0, node.X+node.Width-m.contentWidth()+marginX)
	}
	if node.Y-m.viewportY < marginY {
		m.viewportY = max(0, node.Y-marginY)
	}
	if node.Y+node.Height-m.viewportY > m.contentHeight()-marginY {
		m.viewportY = max(0, node.Y+node.Height-m.contentHeight()+marginY)
	}
	m.clampViewportTo(result)
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func (m *Model) viewWidth() int {
	return max(0, m.width-1)
}

func (m *Model) contentWidth() int {
	return m.viewWidth()
}

func (m *Model) contentHeight() int {
	return max(0, m.height-2)
}
