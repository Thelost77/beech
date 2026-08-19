package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Thelost77/beech/internal/document"
	"github.com/Thelost77/beech/internal/layout"
	"github.com/Thelost77/beech/internal/outline"
	"github.com/Thelost77/beech/internal/storage"
	"github.com/Thelost77/beech/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

// Update implements tea.Model.
func (m *Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := message.(tea.KeyMsg); ok && keyMsg.Type == tea.KeyRunes && keyMsg.Alt {
		if len(keyMsg.Runes) == 1 && keyMsg.Runes[0] >= 0x80 {
			keyMsg.Alt = false
			message = keyMsg
		}
	}

	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.pathInput.Width = max(10, m.viewWidth()-4)
		if m.mode == modeEdit && m.edit != nil {
			m.refreshEditLayout()
		} else {
			m.ensureSelectedVisible()
		}
		return m, nil
	case savedMsg:
		return m.handleSaved(msg)
	case feedbackExpiredMsg:
		m.handleFeedbackExpired(msg)
		return m, nil
	case statusExpiredMsg:
		m.handleStatusExpired(msg)
		return m, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m.handleCtrlC()
		}
		switch m.mode {
		case modeEdit:
			return m.updateEdit(msg)
		case modeSaveAs:
			return m.updateSaveAs(msg)
		case modeHelp:
			if msg.Type == tea.KeyEsc || msg.String() == "?" || msg.String() == "q" {
				m.mode = modeNormal
			}
			return m, nil
		default:
			return m.updateNormal(msg)
		}
	}
	if m.mode == modeEdit {
		return m, m.updateNodeInput(message)
	}
	if m.mode == modeSaveAs {
		var cmd tea.Cmd
		m.pathInput, cmd = m.pathInput.Update(message)
		return m, cmd
	}
	return m, nil
}

func (m *Model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	statusGeneration := m.statusGeneration
	var actionCmd tea.Cmd
	switch msg.String() {
	case "esc":
		m.err = nil
		m.clearStatus()
		return m, nil
	case "q":
		return m.requestQuit()
	case "Q":
		return m, tea.Quit
	case "s":
		return m.requestSave(false)
	case "?":
		m.mode = modeHelp
		return m, nil
	case "h", "left":
		m.selectParent()
	case "l", "right":
		m.selectChild()
	case "j", "down":
		m.selectVertical(1)
	case "k", "up":
		m.selectVertical(-1)
	case "g":
		m.selected = m.doc.Roots()[0]
		m.ensureSelectedVisible()
	case "c":
		m.centerSelected()
	case "ctrl+left":
		m.viewportX -= 2
		m.clampViewport()
	case "ctrl+right":
		m.viewportX += 2
		m.clampViewport()
	case "ctrl+up":
		m.viewportY--
		m.clampViewport()
	case "ctrl+down":
		m.viewportY++
		m.clampViewport()
	case "e", "i":
		m.beginEdit(m.selected, m.doc.Text(m.selected), nil)
		return m, m.nodeInput.Focus()
	case "enter", "o":
		return m.addSibling()
	case "tab", "O":
		return m.addChild()
	case " ":
		actionCmd = m.applyOutcome(m.toggleSelectedFold())
	case "J":
		actionCmd = m.applyOutcome(m.moveSelected(1))
	case "K":
		actionCmd = m.applyOutcome(m.moveSelected(-1))
	case "[":
		actionCmd = m.applyOutcome(m.promoteSelected())
	case "]":
		actionCmd = m.applyOutcome(m.demoteSelected())
	case "d":
		actionCmd = m.applyOutcome(m.deleteSelected())
	case "y":
		actionCmd = m.applyOutcome(m.copySelected())
	case "p":
		actionCmd = m.applyOutcome(m.pasteChild())
	case "P":
		actionCmd = m.applyOutcome(m.pasteSibling())
	case "u":
		actionCmd = m.applyOutcome(m.undoChange())
	case "ctrl+r":
		actionCmd = m.applyOutcome(m.redoChange())
	}
	if m.statusGeneration != statusGeneration {
		if actionCmd != nil {
			return m, tea.Batch(actionCmd, m.statusExpiryCmd())
		}
		return m, m.statusExpiryCmd()
	}
	return m, actionCmd
}

func (m *Model) addSibling() (tea.Model, tea.Cmd) {
	before := m.snapshot()
	id, ok := m.doc.AddSiblingAfter(m.selected, "New node")
	if !ok {
		return m, nil
	}
	m.selected = id
	m.rebuildLayout()
	m.ensureSelectedVisible()
	m.beginEdit(id, "", &before)
	return m, m.nodeInput.Focus()
}

func (m *Model) addChild() (tea.Model, tea.Cmd) {
	before := m.snapshot()
	id, ok := m.doc.AddChild(m.selected, "New node")
	if !ok {
		return m, nil
	}
	m.collapsed[m.selected] = false
	m.selected = id
	m.rebuildLayout()
	m.ensureSelectedVisible()
	m.beginEdit(id, "", &before)
	return m, m.nodeInput.Focus()
}

func (m *Model) beginEdit(target document.NodeID, value string, before *historyEntry) {
	m.mode = modeEdit
	m.edit = &editSession{Target: target, Before: before}
	m.nodeInput.Prompt = ""
	m.nodeInput.SetValue(value)
	m.nodeInput.CursorEnd()
	m.clearStatus()
	m.refreshEditLayout()
}

func (m *Model) refreshEditLayout() {
	if m.edit == nil {
		return
	}
	options := layout.DefaultOptions()
	options.NodeText = map[document.NodeID]string{m.edit.Target: m.nodeInput.Value()}
	m.edit.Layout = layout.Compute(m.doc, m.collapsed, options)
	node := m.edit.Layout.Nodes[m.edit.Target]
	m.edit.EditorWidth = max(1, node.Width-1)
	m.nodeInput.Width = m.edit.EditorWidth
	m.nodeInput.SetCursor(m.nodeInput.Position())
	m.ensureSelectedVisibleIn(m.edit.Layout)
}

func (m *Model) updateNodeInput(msg tea.Msg) tea.Cmd {
	before := m.nodeInput.Value()
	var cmd tea.Cmd
	m.nodeInput, cmd = m.nodeInput.Update(msg)
	if m.nodeInput.Value() != before {
		m.refreshEditLayout()
	}
	return cmd
}

func (m *Model) updateEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.err != nil && msg.Type != tea.KeyEnter {
		m.err = nil
	}
	switch msg.Type {
	case tea.KeyEsc:
		outcome := m.cancelEdit()
		m.applyOutcome(outcome)
		return m, m.statusExpiryCmd()
	case tea.KeyEnter:
		if strings.TrimSpace(m.nodeInput.Value()) == "" {
			m.err = errors.New("node text cannot be empty")
			return m, nil
		}
		feedbackCmd := m.applyOutcome(m.commitEdit())
		return m, tea.Batch(feedbackCmd, m.statusExpiryCmd())
	}
	return m, m.updateNodeInput(msg)
}

func (m *Model) commitEdit() actionOutcome {
	if m.edit == nil {
		return actionOutcome{message: "No active edit"}
	}
	session := m.edit
	value := m.nodeInput.Value()
	old := m.doc.Text(session.Target)
	isNew := session.Before != nil
	if isNew {
		m.doc.Rename(session.Target, value)
		m.pushUndo(*session.Before)
		m.changed()
	} else if strings.TrimSpace(value) != old {
		before := m.snapshot()
		m.doc.Rename(session.Target, value)
		if m.doc.Text(session.Target) != old {
			m.pushUndo(before)
			m.changed()
		}
	}
	target := session.Target
	m.mode = modeNormal
	m.nodeInput.Blur()
	m.edit = nil
	m.ensureSelectedVisible()
	if isNew {
		return actionOutcome{kind: ui.FeedbackCreate, message: "Created “" + m.doc.Text(target) + "”", affected: []document.NodeID{target}}
	}
	if m.doc.Text(target) == old {
		return actionOutcome{message: "No text changes"}
	}
	return actionOutcome{kind: ui.FeedbackEdit, message: "Updated “" + m.doc.Text(target) + "”", affected: []document.NodeID{target}}
}

func (m *Model) cancelEdit() actionOutcome {
	if m.edit == nil {
		return actionOutcome{message: "No active edit"}
	}
	if m.edit.Before != nil {
		m.restore(*m.edit.Before)
	}
	m.mode = modeNormal
	m.nodeInput.Blur()
	m.edit = nil
	m.ensureSelectedVisible()
	return actionOutcome{message: "Edit cancelled"}
}

func (m *Model) toggleSelectedFold() actionOutcome {
	children := m.doc.Children(m.selected)
	if len(children) == 0 {
		return actionOutcome{message: "Leaf nodes cannot be collapsed"}
	}
	title := m.doc.Text(m.selected)
	if m.collapsed[m.selected] {
		m.collapsed[m.selected] = false
		m.changed()
		return actionOutcome{kind: ui.FeedbackFold, message: "Expanded “" + title + "”", affected: m.subtreeNodeIDs(m.selected)}
	}
	m.collapsed[m.selected] = true
	m.changed()
	return actionOutcome{kind: ui.FeedbackFold, message: "Collapsed “" + title + "”", affected: []document.NodeID{m.selected}}
}

func (m *Model) moveSelected(delta int) actionOutcome {
	direction := "down"
	boundary := "Already the last sibling"
	if delta < 0 {
		direction = "up"
		boundary = "Already the first sibling"
	}
	before := m.snapshot()
	if !m.doc.MoveSibling(m.selected, delta) {
		return actionOutcome{message: boundary}
	}
	m.pushUndo(before)
	m.changed()
	return actionOutcome{kind: ui.FeedbackMove, message: "Moved “" + m.doc.Text(m.selected) + "” " + direction, affected: m.subtreeNodeIDs(m.selected)}
}

func (m *Model) promoteSelected() actionOutcome {
	if m.doc.Parent(m.selected) == document.NoNode {
		return actionOutcome{message: "The root cannot be promoted"}
	}
	before := m.snapshot()
	if !m.doc.Promote(m.selected) {
		return actionOutcome{message: "This node cannot be promoted"}
	}
	m.pushUndo(before)
	m.changed()
	return actionOutcome{kind: ui.FeedbackMove, message: "Promoted “" + m.doc.Text(m.selected) + "”", affected: m.subtreeNodeIDs(m.selected)}
}

func (m *Model) demoteSelected() actionOutcome {
	before := m.snapshot()
	if !m.doc.Demote(m.selected) {
		return actionOutcome{message: "A node needs a previous sibling before it can be demoted"}
	}
	m.pushUndo(before)
	m.changed()
	return actionOutcome{kind: ui.FeedbackMove, message: "Demoted “" + m.doc.Text(m.selected) + "”", affected: m.subtreeNodeIDs(m.selected)}
}

func (m *Model) copySelected() actionOutcome {
	branch, ok := m.doc.CopyBranch(m.selected)
	if !ok {
		return actionOutcome{message: "Nothing to copy"}
	}
	m.clipboard = &branch
	nodes := m.subtreeNodeIDs(m.selected)
	return actionOutcome{kind: ui.FeedbackCopy, message: describeNodeAction("Copied", m.doc.Text(m.selected), len(nodes)), affected: nodes}
}

func (m *Model) deleteSelected() actionOutcome {
	node, ok := m.doc.Node(m.selected)
	if !ok {
		return actionOutcome{message: "Nothing to cut"}
	}
	nodes := m.subtreeNodeIDs(m.selected)
	next := node.Parent
	siblings := m.doc.Roots()
	if node.Parent != document.NoNode {
		siblings = m.doc.Children(node.Parent)
	}
	index := slices.Index(siblings, m.selected)
	if index > 0 {
		next = siblings[index-1]
	} else if index >= 0 && index+1 < len(siblings) {
		next = siblings[index+1]
	}
	before := m.snapshot()
	branch, _ := m.doc.CopyBranch(m.selected)
	if !m.doc.Delete(m.selected) {
		return actionOutcome{message: "The final root cannot be cut"}
	}
	m.clipboard = &branch
	m.selected = next
	m.pushUndo(before)
	m.changed()
	return actionOutcome{kind: ui.FeedbackDelete, message: describeNodeAction("Cut", node.Text, len(nodes)) + " • u undo", affected: []document.NodeID{m.selected}}
}

func (m *Model) pasteChild() actionOutcome {
	if m.clipboard == nil {
		return actionOutcome{message: "The clipboard is empty"}
	}
	before := m.snapshot()
	id, ok := m.doc.PasteChild(m.selected, *m.clipboard)
	if !ok {
		return actionOutcome{message: "Could not paste below this node"}
	}
	m.collapsed[m.selected] = false
	m.selected = id
	m.pushUndo(before)
	m.changed()
	nodes := m.subtreeNodeIDs(id)
	return actionOutcome{kind: ui.FeedbackCreate, message: describeNodeAction("Pasted", m.doc.Text(id), len(nodes)) + " as child", affected: nodes}
}

func (m *Model) pasteSibling() actionOutcome {
	if m.clipboard == nil {
		return actionOutcome{message: "The clipboard is empty"}
	}
	before := m.snapshot()
	id, ok := m.doc.PasteSibling(m.selected, *m.clipboard)
	if !ok {
		return actionOutcome{message: "Could not paste beside this node"}
	}
	m.selected = id
	m.pushUndo(before)
	m.changed()
	nodes := m.subtreeNodeIDs(id)
	return actionOutcome{kind: ui.FeedbackCreate, message: describeNodeAction("Pasted", m.doc.Text(id), len(nodes)) + " as sibling", affected: nodes}
}

func (m *Model) undoChange() actionOutcome {
	if len(m.undo) == 0 {
		return actionOutcome{message: "Nothing to undo"}
	}
	m.redo = append(m.redo, m.snapshot())
	entry := m.undo[len(m.undo)-1]
	m.undo = m.undo[:len(m.undo)-1]
	m.restore(entry)
	m.revision++
	m.refreshDirty()
	return actionOutcome{kind: ui.FeedbackUndo, message: "Undid change", affected: []document.NodeID{m.selected}}
}

func (m *Model) redoChange() actionOutcome {
	if len(m.redo) == 0 {
		return actionOutcome{message: "Nothing to redo"}
	}
	m.undo = append(m.undo, m.snapshot())
	entry := m.redo[len(m.redo)-1]
	m.redo = m.redo[:len(m.redo)-1]
	m.restore(entry)
	m.revision++
	m.refreshDirty()
	return actionOutcome{kind: ui.FeedbackUndo, message: "Redid change", affected: []document.NodeID{m.selected}}
}

func (m *Model) requestQuit() (tea.Model, tea.Cmd) {
	if !m.dirty {
		return m, tea.Quit
	}
	m.quitAfterSave = true
	return m.requestSave(true)
}

func (m *Model) requestSave(quit bool) (tea.Model, tea.Cmd) {
	if m.saving {
		m.quitAfterSave = m.quitAfterSave || quit
		return m, nil
	}
	if m.path == "" {
		m.mode = modeSaveAs
		m.pathInput.Width = max(10, m.viewWidth()-4)
		m.pathInput.Prompt = "save as › "
		if m.imported {
			m.pathInput.Prompt = "save imported as › "
		}
		suggestedPath := m.suggestedPath
		if suggestedPath == "" {
			suggestedPath = "map.md"
		}
		m.pathInput.SetValue(suggestedPath)
		m.pathInput.CursorEnd()
		m.quitAfterSave = quit
		return m, m.pathInput.Focus()
	}
	cmd := m.saveCmd(m.path)
	return m, cmd
}

func (m *Model) updateSaveAs(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.err != nil && msg.Type != tea.KeyEnter {
		m.err = nil
	}
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = modeNormal
		m.quitAfterSave = false
		m.pathInput.Blur()
		m.setStatus("Save cancelled")
		return m, m.statusExpiryCmd()
	case tea.KeyEnter:
		value := strings.TrimSpace(m.pathInput.Value())
		if value == "" {
			m.err = errors.New("save path cannot be empty")
			return m, nil
		}
		extension := strings.ToLower(filepath.Ext(value))
		if extension == "" {
			value += ".md"
		} else if extension != ".md" {
			m.err = errors.New("Beech saves Markdown files with the .md extension")
			return m, nil
		}
		path, err := storage.NewPath(value)
		if err != nil {
			m.err = err
			return m, nil
		}
		m.mode = modeNormal
		m.pathInput.Blur()
		return m, m.saveCmd(path)
	}
	var cmd tea.Cmd
	m.pathInput, cmd = m.pathInput.Update(msg)
	return m, cmd
}

func (m *Model) saveCmd(path string) tea.Cmd {
	data, err := outline.Serialize(m.doc, m.collapsed, m.style)
	if err != nil {
		m.err = err
		m.quitAfterSave = false
		return nil
	}
	m.saving = true
	m.pendingSavePath = path
	revision := m.revision
	expected := m.fingerprint
	if path != m.path {
		expected = storage.Fingerprint{}
	}
	mode := m.fileMode
	return func() tea.Msg {
		fingerprint, saveErr := storage.Save(path, data, expected, mode)
		return savedMsg{
			path:        path,
			data:        data,
			fingerprint: fingerprint,
			revision:    revision,
			err:         saveErr,
		}
	}
}

func (m *Model) handleSaved(msg savedMsg) (tea.Model, tea.Cmd) {
	m.saving = false
	m.pendingSavePath = ""
	if msg.err != nil {
		if errors.Is(msg.err, storage.ErrConflict) {
			m.err = fmt.Errorf("%s changed outside Beech; your changes were not written", filepath.Base(msg.path))
		} else {
			m.err = msg.err
		}
		m.quitAfterSave = false
		return m, nil
	}
	m.path = msg.path
	m.suggestedPath = ""
	m.imported = false
	m.fingerprint = msg.fingerprint
	m.lastSaved = slices.Clone(msg.data)
	m.refreshDirty()
	m.setStatus("✓ Saved " + filepath.Base(msg.path))
	m.err = nil
	if m.quitAfterSave {
		if m.dirty {
			return m, m.saveCmd(m.path)
		}
		m.quitAfterSave = false
		return m, tea.Quit
	}
	return m, m.statusExpiryCmd()
}

func (m *Model) handleCtrlC() (tea.Model, tea.Cmd) {
	statusGeneration := m.statusGeneration
	switch m.mode {
	case modeEdit:
		if strings.TrimSpace(m.nodeInput.Value()) != "" {
			m.commitEdit()
		} else {
			m.cancelEdit()
		}
	case modeSaveAs:
		m.mode = modeNormal
		m.pathInput.Blur()
	case modeHelp:
		m.mode = modeNormal
	}
	model, cmd := m.requestQuit()
	if m.statusGeneration != statusGeneration {
		return model, tea.Batch(cmd, m.statusExpiryCmd())
	}
	return model, cmd
}
