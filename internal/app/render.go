package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Thelost77/beech/internal/document"
	"github.com/Thelost77/beech/internal/highlight"
	"github.com/Thelost77/beech/internal/layout"
	"github.com/Thelost77/beech/internal/ui"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
)

// View implements tea.Model.
func (m *Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	width := m.viewWidth()
	if width == 0 {
		return strings.Repeat("\n", max(0, m.height-1))
	}

	lines := make([]string, 0, m.height)
	lines = append(lines, m.viewHeader(width))
	bodyHeight := m.contentHeight()
	switch {
	case m.width < 30 || m.height < 8:
		lines = append(lines, m.smallTerminalView(width, bodyHeight)...)
	case m.mode == modeHelp:
		lines = append(lines, m.helpView(width, bodyHeight)...)
	default:
		lines = append(lines, m.mapView(width, bodyHeight)...)
	}
	lines = append(lines, m.viewFooter(width))

	for len(lines) < m.height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	if len(lines) > m.height {
		lines = lines[:m.height]
	}
	return strings.Join(lines, "\n")
}

func (m *Model) viewHeader(width int) string {
	name := "untitled"
	if m.path != "" {
		name = filepath.Base(m.path)
	} else if m.suggestedPath != "" {
		name = filepath.Base(m.suggestedPath)
		if m.imported {
			name += " • imported"
		}
	}
	state := ""
	if m.saving {
		state = " • saving"
	} else if m.dirty {
		state = " • modified"
	}
	left := m.styles.Title.Render("beech") + m.styles.Muted.Render(" › "+name+state)
	return fixedLine(left, width)
}

func (m *Model) mapView(width, height int) []string {
	canvas := ui.NewCanvas(width, height)
	offsetY := max(0, (height-m.layout.Height)/2)
	branches := m.branchIndexes()
	activePath := m.activePath()
	for _, connector := range m.layout.Connectors {
		owner := m.layout.Nodes[connector.Owner]
		style := ui.CellStyle{
			Role:     ui.RoleConnector,
			Branch:   branches[connector.Owner],
			Active:   owner.Depth > 0 && activePath[connector.Owner],
			Feedback: m.feedbackKind(connector.Owner),
		}
		canvas.PutConnector(connector.X-m.viewportX, connector.Y+offsetY-m.viewportY, connector.Join, style)
	}
	for _, id := range m.layout.Order {
		node := m.layout.Nodes[id]
		baseStyle := m.nodeCellStyle(id)
		for row, line := range node.Lines {
			y := node.Y + row + offsetY - m.viewportY
			x := node.X - m.viewportX
			if id == m.selected {
				background := baseStyle
				background.Role = ui.RoleNone
				canvas.PutText(x, y, strings.Repeat(" ", node.Width), background)
			}
			textX := x + 1
			for _, span := range highlight.Parse(line) {
				style := baseStyle
				style.Role = highlightedRole(span.Role, baseStyle.Role)
				canvas.PutText(textX, y, span.Text, style)
				textX += uniseg.StringWidth(span.Text)
			}
			if node.CollapseCount > 0 && row == node.CollapseLine {
				marker := "▸ " + itoa(node.CollapseCount)
				if node.CollapseOffset > 0 {
					marker = "  " + marker
				}
				style := baseStyle
				style.Role = ui.RoleCollapse
				canvas.PutText(x+1+node.CollapseOffset, y, marker, style)
			}
		}
	}
	return canvas.Lines(m.styles.RenderCell)
}

func (m *Model) nodeCellStyle(id document.NodeID) ui.CellStyle {
	return m.nodeCellStyleWithLayout(id, m.layout, m.branchIndexes(), m.activePath())
}

func (m *Model) nodeCellStyleWithLayout(id document.NodeID, result layout.Result, branches map[document.NodeID]int8, activePath map[document.NodeID]bool) ui.CellStyle {
	node := result.Nodes[id]
	role := ui.RoleLeaf
	switch {
	case node.Depth == 0:
		role = ui.RoleRoot
	case node.Depth == 1:
		role = ui.RoleFirstBranch
	case len(m.doc.Children(id)) > 0:
		role = ui.RoleBranch
	}
	return ui.CellStyle{
		Role:     role,
		Branch:   branches[id],
		Selected: id == m.selected,
		Active:   activePath[id],
		Feedback: m.feedbackKind(id),
	}
}

func (m *Model) branchIndexes() map[document.NodeID]int8 {
	branches := make(map[document.NodeID]int8, m.doc.Len())
	var assign func(document.NodeID, int8)
	assign = func(id document.NodeID, branch int8) {
		branches[id] = branch
		for _, child := range m.doc.Children(id) {
			assign(child, branch)
		}
	}
	for _, root := range m.doc.Roots() {
		branches[root] = -1
		for index, child := range m.doc.Children(root) {
			assign(child, int8(index%ui.BranchColorCount))
		}
	}
	return branches
}

func (m *Model) activePath() map[document.NodeID]bool {
	path := make(map[document.NodeID]bool)
	for id := m.selected; id != document.NoNode; id = m.doc.Parent(id) {
		path[id] = true
	}
	return path
}

func highlightedRole(role highlight.Role, fallback ui.CellRole) ui.CellRole {
	switch role {
	case highlight.TaskPending:
		return ui.RoleTaskPending
	case highlight.TaskDone:
		return ui.RoleTaskDone
	case highlight.Strong:
		return ui.RoleStrong
	case highlight.Code:
		return ui.RoleCode
	case highlight.Link:
		return ui.RoleLink
	case highlight.Tag:
		return ui.RoleTag
	case highlight.Syntax:
		return ui.RoleSyntax
	default:
		return fallback
	}
}

func (m *Model) viewFooter(width int) string {
	var content string
	if m.err != nil {
		content = m.styles.Error.Render("⚠ " + sanitizeMessage(m.err.Error()))
	} else {
		switch m.mode {
		case modeEdit:
			content = m.nodeInput.View()
		case modeSaveAs:
			content = m.pathInput.View()
		case modeHelp:
			content = m.styles.Muted.Render("? / esc close help")
		default:
			if m.status != "" {
				content = m.styles.Accent.Render(m.status) + m.styles.Muted.Render("  •  ? help")
			} else {
				content = m.styles.Muted.Render("enter sibling  tab child  hjkl move  e edit  space fold  s save  ? help")
			}
		}
	}
	content = ansi.Truncate(content, width, "")
	return m.styles.Status.Width(width).Render(content)
}

func (m *Model) helpView(width, height int) []string {
	rows := []string{
		m.styles.Title.Render("Beech keybindings"),
		"",
		m.styles.Accent.Render("Edit"),
		"  enter / o     create sibling",
		"  tab / O       create child",
		"  e / i         edit node",
		"  d / y / p / P cut, copy, paste child/sibling",
		"  J / K         move sibling down/up",
		"  [ / ]         promote/demote",
		"  u / ctrl+r    undo/redo",
		"",
		m.styles.Accent.Render("Navigate"),
		"  h j k l       parent, down, up, child",
		"  arrows        navigate",
		"  space         collapse/expand",
		"  c / g         center/go to root",
		"  ctrl+arrows   pan viewport",
		"",
		m.styles.Accent.Render("File"),
		"  s             save",
		"  q / ctrl+c    save and quit",
		"  Q             discard changes and quit",
	}
	result := make([]string, height)
	for index := range result {
		if index < len(rows) {
			result[index] = fixedLine(rows[index], width)
		} else {
			result[index] = strings.Repeat(" ", width)
		}
	}
	return result
}

func (m *Model) smallTerminalView(width, height int) []string {
	result := make([]string, height)
	message := fmt.Sprintf("Terminal too small: %dx%d; need at least 30x8", m.width, m.height)
	for index := range result {
		if index == height/2 {
			result[index] = fixedLine(m.styles.Error.Render(message), width)
		} else {
			result[index] = strings.Repeat(" ", width)
		}
	}
	return result
}

func fixedLine(value string, width int) string {
	value = ansi.Truncate(value, width, "")
	missing := width - lipgloss.Width(value)
	if missing > 0 {
		value += strings.Repeat(" ", missing)
	}
	return value
}

func sanitizeMessage(value string) string {
	value = ansi.Strip(value)
	return strings.Join(strings.Fields(value), " ")
}
