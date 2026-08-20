package app

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Thelost77/beech/internal/document"
	"github.com/Thelost77/beech/internal/outline"
	"github.com/Thelost77/beech/internal/storage"
	"github.com/Thelost77/beech/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func newTestModel(t *testing.T) *Model {
	t.Helper()
	doc := document.New("Root")
	root := doc.Roots()[0]
	_, _ = doc.AddChild(root, "First")
	data, err := outline.Serialize(doc, nil, outline.DefaultStyle())
	if err != nil {
		t.Fatal(err)
	}
	m := New(InitialDocument{Document: doc, Style: outline.DefaultStyle(), SavedData: data})
	updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	return m
}

func updateModel(m *Model, msg tea.Msg) tea.Cmd {
	_, cmd := m.Update(msg)
	return cmd
}

func keyMsg(value string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
}

func textPosition(view, text string) (int, int) {
	for row, line := range strings.Split(view, "\n") {
		if column := strings.Index(line, text); column >= 0 {
			return row, column
		}
	}
	return -1, -1
}

func TestJourneyCreateEditUndoRedo(t *testing.T) {
	m := newTestModel(t)
	root := m.Selected()

	updateModel(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.mode != modeEdit {
		t.Fatal("expected edit mode after adding child")
	}
	updateModel(m, keyMsg("Idea"))
	updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	created := m.Selected()
	if created == root || m.doc.Text(created) != "Idea" {
		t.Fatalf("created node = %d %q", created, m.doc.Text(created))
	}
	if !m.Dirty() {
		t.Fatal("document should be dirty")
	}

	updateModel(m, keyMsg("u"))
	if m.doc.Len() != 2 {
		t.Fatalf("nodes after undo = %d", m.doc.Len())
	}
	updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlR})
	if m.doc.Len() != 3 || m.doc.Text(m.Selected()) != "Idea" {
		t.Fatal("redo did not restore node")
	}
}

func TestCopyFeedbackCoversSubtreeAndExpires(t *testing.T) {
	m := newTestModel(t)
	root := m.doc.Roots()[0]
	first := m.doc.Children(root)[0]
	grandchild, _ := m.doc.AddChild(first, "Detail")
	second, _ := m.doc.AddSiblingAfter(first, "Second")
	m.rebuildLayout()
	m.selected = first

	cmd := updateModel(m, keyMsg("y"))
	if cmd == nil {
		t.Fatal("copy did not schedule feedback expiry")
	}
	if m.feedbackKind(first) != ui.FeedbackCopy || m.feedbackKind(grandchild) != ui.FeedbackCopy {
		t.Fatal("copied subtree does not use copy feedback")
	}
	if m.feedbackKind(root) != ui.FeedbackNone || m.feedbackKind(second) != ui.FeedbackNone {
		t.Fatal("nodes outside copied subtree received feedback")
	}
	style := m.nodeCellStyle(first)
	if !style.Selected || style.Feedback != ui.FeedbackCopy {
		t.Fatalf("selected copy style = %#v", style)
	}

	generation := m.feedbackNodes[first].generation
	updateModel(m, feedbackExpiredMsg{generation: generation, nodes: []document.NodeID{first, grandchild}})
	if m.feedbackKind(first) != ui.FeedbackNone || m.feedbackKind(grandchild) != ui.FeedbackNone {
		t.Fatal("copy feedback did not expire")
	}
}

func TestPasteFeedbackUsesCreateColorOnlyForNewSubtree(t *testing.T) {
	m := newTestModel(t)
	root := m.doc.Roots()[0]
	first := m.doc.Children(root)[0]
	originalDetail, _ := m.doc.AddChild(first, "Detail")
	m.rebuildLayout()
	m.selected = first
	updateModel(m, keyMsg("y"))
	m.selected = root

	cmd := updateModel(m, keyMsg("p"))
	pasted := m.selected
	pastedChildren := m.doc.Children(pasted)
	if cmd == nil {
		t.Fatal("paste did not schedule feedback expiry")
	}
	if len(pastedChildren) != 1 || m.feedbackKind(pasted) != ui.FeedbackCreate || m.feedbackKind(pastedChildren[0]) != ui.FeedbackCreate {
		t.Fatal("new pasted subtree does not use create feedback")
	}
	if m.feedbackKind(first) != ui.FeedbackCopy || m.feedbackKind(originalDetail) != ui.FeedbackCopy {
		t.Fatal("paste replaced the source copy feedback")
	}
}

func TestStaleFeedbackExpiryDoesNotClearNewFeedback(t *testing.T) {
	m := newTestModel(t)
	updateModel(m, keyMsg("y"))
	old := m.feedbackNodes[m.selected]
	updateModel(m, keyMsg("y"))
	current := m.feedbackNodes[m.selected]
	updateModel(m, feedbackExpiredMsg{generation: old.generation, nodes: []document.NodeID{m.selected}})
	if got := m.feedbackNodes[m.selected]; got != current {
		t.Fatalf("stale expiry changed feedback: got %#v want %#v", got, current)
	}
}

func TestStructuralStylesFollowTopLevelBranchesAndActivePath(t *testing.T) {
	m := newTestModel(t)
	root := m.doc.Roots()[0]
	first := m.doc.Children(root)[0]
	grandchild, _ := m.doc.AddChild(first, "Detail")
	second, _ := m.doc.AddSiblingAfter(first, "Second")
	m.rebuildLayout()
	m.selected = grandchild

	branches := m.branchIndexes()
	if branches[root] != -1 || branches[first] != 0 || branches[grandchild] != 0 || branches[second] != 1 {
		t.Fatalf("branch indexes: root=%d first=%d grandchild=%d second=%d", branches[root], branches[first], branches[grandchild], branches[second])
	}
	path := m.activePath()
	if !path[root] || !path[first] || !path[grandchild] || path[second] {
		t.Fatalf("active path = %#v", path)
	}
	if got := m.nodeCellStyle(root).Role; got != ui.RoleRoot {
		t.Fatalf("root role = %v", got)
	}
	if got := m.nodeCellStyle(first).Role; got != ui.RoleFirstBranch {
		t.Fatalf("first-level role = %v", got)
	}
	if got := m.nodeCellStyle(grandchild).Role; got != ui.RoleLeaf {
		t.Fatalf("leaf role = %v", got)
	}
}

func TestActionsReportSuccessAndValidNoOps(t *testing.T) {
	m := newTestModel(t)
	root := m.selected

	updateModel(m, keyMsg("["))
	if !strings.Contains(m.status, "root cannot be promoted") {
		t.Fatalf("promote root status = %q", m.status)
	}
	updateModel(m, keyMsg(" "))
	if m.feedbackKind(root) != ui.FeedbackFold || !strings.Contains(m.status, "Collapsed") {
		t.Fatalf("fold feedback=%v status=%q", m.feedbackKind(root), m.status)
	}
	updateModel(m, keyMsg(" "))
	updateModel(m, keyMsg("l"))
	first := m.selected
	second, _ := m.doc.AddSiblingAfter(first, "Second")
	m.rebuildLayout()
	updateModel(m, keyMsg("J"))
	if m.feedbackKind(first) != ui.FeedbackMove || !strings.Contains(m.status, "Moved") {
		t.Fatalf("move feedback=%v status=%q", m.feedbackKind(first), m.status)
	}
	if got := m.doc.Children(root); len(got) != 2 || got[1] != first || got[0] != second {
		t.Fatalf("children after move = %v", got)
	}
	updateModel(m, keyMsg("J"))
	if !strings.Contains(m.status, "last sibling") {
		t.Fatalf("move boundary status = %q", m.status)
	}
}

func TestCreateEditCutAndUndoUseActionFeedback(t *testing.T) {
	m := newTestModel(t)
	updateModel(m, tea.KeyMsg{Type: tea.KeyTab})
	updateModel(m, keyMsg("New idea"))
	updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	created := m.selected
	if m.feedbackKind(created) != ui.FeedbackCreate || !strings.Contains(m.status, "Created") {
		t.Fatalf("create feedback=%v status=%q", m.feedbackKind(created), m.status)
	}

	updateModel(m, keyMsg("e"))
	updateModel(m, keyMsg(" updated"))
	updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.feedbackKind(created) != ui.FeedbackEdit || !strings.Contains(m.status, "Updated") {
		t.Fatalf("edit feedback=%v status=%q", m.feedbackKind(created), m.status)
	}

	updateModel(m, keyMsg("d"))
	if m.feedbackKind(m.selected) != ui.FeedbackDelete || !strings.Contains(m.status, "Cut") {
		t.Fatalf("cut feedback=%v status=%q", m.feedbackKind(m.selected), m.status)
	}
	updateModel(m, keyMsg("u"))
	if m.feedbackKind(m.selected) != ui.FeedbackUndo || m.doc.Text(m.selected) != "New idea updated" {
		t.Fatalf("undo feedback=%v selected=%q", m.feedbackKind(m.selected), m.doc.Text(m.selected))
	}
}

func TestCollapsedMarkerUsesSeparateSemanticStyle(t *testing.T) {
	m := newTestModel(t)
	updateModel(m, keyMsg(" "))
	view := m.View()
	if !strings.Contains(view, "▸ 1") || strings.Contains(view, "[+1]") {
		t.Fatalf("collapsed marker is not semantic: %q", view)
	}
}

func TestCollapseStateMakesDocumentDirtyAndRoundTrips(t *testing.T) {
	m := newTestModel(t)
	root := m.doc.Roots()[0]
	updateModel(m, keyMsg(" "))
	if !m.collapsed[root] || !m.Dirty() {
		t.Fatalf("collapsed=%v dirty=%v", m.collapsed[root], m.Dirty())
	}
	data, err := outline.Serialize(m.doc, m.collapsed, m.style)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := outline.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	parsedRoot := parsed.Document.Roots()[0]
	if !parsed.Collapsed[parsedRoot] {
		t.Fatal("collapsed state was not restored")
	}

	updateModel(m, keyMsg(" "))
	if m.Dirty() {
		t.Fatal("returning to the saved fold state should make the document clean")
	}
}

func TestImportedHMMPromptsForSuggestedMarkdownPathOnQuit(t *testing.T) {
	m := newTestModel(t)
	m.path = ""
	m.suggestedPath = filepath.Join(t.TempDir(), "ideas.md")
	m.imported = true
	m.dirty = true

	cmd := updateModel(m, keyMsg("q"))
	if cmd == nil || m.mode != modeSaveAs {
		t.Fatal("quit did not open Save As for imported document")
	}
	if got := m.pathInput.Value(); got != m.suggestedPath {
		t.Fatalf("suggested path = %q, want %q", got, m.suggestedPath)
	}
	if view := m.View(); !strings.Contains(view, "imported") || !strings.Contains(view, "save imported as") {
		t.Fatalf("imported Save As context is missing: %q", view)
	}

	target := m.suggestedPath
	saveCmd := updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	if saveCmd == nil {
		t.Fatal("accepting the suggested path did not save")
	}
	updateModel(m, saveCmd())
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "<!-- beech:outline v1 -->\n\n- Root") {
		t.Fatalf("saved import is not native Markdown: %q", data)
	}
}

func TestSaveAsRejectsNonMarkdownExtension(t *testing.T) {
	m := newTestModel(t)
	m.path = ""
	m.dirty = true
	updateModel(m, keyMsg("s"))
	m.pathInput.SetValue(filepath.Join(t.TempDir(), "map.hmm"))
	cmd := updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || m.err == nil || !strings.Contains(m.err.Error(), ".md extension") {
		t.Fatalf("command=%v error=%v", cmd, m.err)
	}
	if !strings.Contains(m.View(), ".md extension") {
		t.Fatal("Save As validation error is not visible")
	}
}

func TestEditingUsesBottomInputLineWithStaticMap(t *testing.T) {
	m := newTestModel(t)
	root := m.selected
	before := m.View()
	beforeLayout := m.layout
	beforeViewportX, beforeViewportY := m.viewportX, m.viewportY

	updateModel(m, keyMsg("e"))
	if m.nodeInput.Value() != "Root" {
		t.Fatalf("edit input = %q, want the current node text", m.nodeInput.Value())
	}
	if m.doc.Text(root) != "Root" || !reflect.DeepEqual(m.layout, beforeLayout) {
		t.Fatal("entering edit changed the committed map")
	}

	updateModel(m, keyMsg(" changed"))
	view := m.View()
	lines := strings.Split(view, "\n")
	footer := lines[len(lines)-1]
	if !strings.Contains(footer, "Root changed") {
		t.Fatalf("typed text is not shown on the bottom input line: %q", footer)
	}
	if !strings.Contains(view, "Root") {
		t.Fatal("map no longer shows the edited node")
	}
	if m.doc.Text(root) != "Root" || !reflect.DeepEqual(m.layout, beforeLayout) {
		t.Fatal("typing changed the committed map before Enter")
	}
	if m.viewportX != beforeViewportX || m.viewportY != beforeViewportY {
		t.Fatalf("typing scrolled the viewport: %d,%d", m.viewportX, m.viewportY)
	}

	for index := range lines {
		if index < len(lines)-1 && strings.Split(before, "\n")[index] != lines[index] {
			t.Fatalf("typing changed map row %d", index)
		}
	}

	updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.doc.Text(root) != "Root changed" {
		t.Fatalf("committed text = %q", m.doc.Text(root))
	}
	if !m.Dirty() {
		t.Fatal("commit did not mark the document dirty")
	}
	for index, line := range strings.Split(ansi.Strip(m.View()), "\n") {
		if index < m.height-1 && strings.Contains(line, "Root changed") {
			return
		}
	}
	t.Fatal("map does not show the committed title")
}

func TestEditingScrollsLongInputHorizontally(t *testing.T) {
	m := newTestModel(t)
	updateModel(m, keyMsg("e"))
	for range 100 {
		updateModel(m, keyMsg("x"))
	}
	footer := strings.Split(m.View(), "\n")
	input := footer[len(footer)-1]
	if width := lipgloss.Width(input); width > m.viewWidth() {
		t.Fatalf("input line width = %d, want at most %d", width, m.viewWidth())
	}
	if !strings.Contains(ansi.Strip(input), "x") {
		t.Fatal("scrolled input line lost the typed text")
	}
}

func TestCursorBlinkChangesOnlyTheInputLine(t *testing.T) {
	m := newTestModel(t)
	blinkCmd := updateModel(m, keyMsg("e"))
	if blinkCmd == nil {
		t.Fatal("edit did not start cursor blinking")
	}
	updateModel(m, keyMsg(" more text"))
	before := m.View()
	updateModel(m, blinkCmd())
	after := m.View()
	beforeLines := strings.Split(before, "\n")
	afterLines := strings.Split(after, "\n")
	for index := range beforeLines {
		if index != len(beforeLines)-1 && beforeLines[index] != afterLines[index] {
			t.Fatalf("cursor blink changed map row %d", index)
		}
	}
}

func TestCancelRestoresCommittedLayoutExactly(t *testing.T) {
	m := newTestModel(t)
	before := m.layout
	updateModel(m, keyMsg("e"))
	updateModel(m, keyMsg(" with changes"))
	updateModel(m, tea.KeyMsg{Type: tea.KeyEsc})
	if !reflect.DeepEqual(m.layout, before) {
		t.Fatal("cancel changed committed layout")
	}
}

func TestEditEscapeRestoresNewNodeTransaction(t *testing.T) {
	m := newTestModel(t)
	before := m.doc.Len()
	updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	updateModel(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.doc.Len() != before {
		t.Fatalf("nodes = %d, want %d", m.doc.Len(), before)
	}
	if m.Dirty() {
		t.Fatal("cancelled edit made document dirty")
	}
}

func TestStatusExpiresAfterHalfSecondAndIgnoresStaleTimers(t *testing.T) {
	m := newTestModel(t)
	m.setStatus("Saved map.hmm")
	savedGeneration := m.statusGeneration
	if statusDuration != 500*time.Millisecond {
		t.Fatalf("status duration = %v", statusDuration)
	}
	m.setStatus("Copied subtree")
	updateModel(m, statusExpiredMsg{generation: savedGeneration})
	if m.status != "Copied subtree" {
		t.Fatalf("stale timer cleared status: %q", m.status)
	}
	updateModel(m, statusExpiredMsg{generation: m.statusGeneration})
	if m.status != "" {
		t.Fatalf("status did not expire: %q", m.status)
	}
}

func TestUpperQQuitsAndDiscardsChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "map.md")
	original := []byte("<!-- beech:outline v1 -->\n\n- Root\n")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := storage.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := outline.Parse(loaded.Data)
	if err != nil {
		t.Fatal(err)
	}
	m := New(InitialDocument{Document: parsed.Document, Path: loaded.Path, Style: parsed.Style, Fingerprint: loaded.Fingerprint, FileMode: loaded.Mode, SavedData: loaded.Data})
	updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	updateModel(m, keyMsg("e"))
	m.nodeInput.SetValue("Changed")
	updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.Dirty() {
		t.Fatal("document should be dirty before discard")
	}

	cmd := updateModel(m, keyMsg("Q"))
	if cmd == nil {
		t.Fatal("Q returned no quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("Q did not return tea.Quit")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(original) {
		t.Fatalf("Q wrote discarded changes: %q", data)
	}
}

func TestSaveAndExternalConflict(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "map.md")
	initial := []byte("<!-- beech:outline v1 -->\n\n- Root\n")
	if err := os.WriteFile(path, initial, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := storage.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := outline.Parse(loaded.Data)
	if err != nil {
		t.Fatal(err)
	}
	m := New(InitialDocument{Document: parsed.Document, Collapsed: parsed.Collapsed, Path: loaded.Path, Style: parsed.Style, Fingerprint: loaded.Fingerprint, FileMode: loaded.Mode, SavedData: loaded.Data})
	updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	updateModel(m, keyMsg("e"))
	for range len("Root") {
		updateModel(m, tea.KeyMsg{Type: tea.KeyBackspace})
	}
	updateModel(m, keyMsg("Changed"))
	updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	cmd := updateModel(m, keyMsg("s"))
	if cmd == nil {
		t.Fatal("save returned no command")
	}
	expiryCmd := updateModel(m, cmd())
	data, _ := os.ReadFile(path)
	if string(data) != "<!-- beech:outline v1 -->\n\n- Changed\n" || m.Dirty() {
		t.Fatalf("saved data = %q dirty=%v", data, m.Dirty())
	}
	if expiryCmd == nil || !strings.Contains(m.status, "Saved") {
		t.Fatalf("save confirmation was not scheduled: status=%q", m.status)
	}
	updateModel(m, statusExpiredMsg{generation: m.statusGeneration})
	if m.status != "" {
		t.Fatalf("save confirmation did not return to normal view: %q", m.status)
	}

	if err := os.WriteFile(path, []byte("External\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	updateModel(m, keyMsg("e"))
	updateModel(m, keyMsg(" again"))
	updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	cmd = updateModel(m, keyMsg("s"))
	updateModel(m, cmd())
	if m.err == nil || !strings.Contains(m.err.Error(), "changed outside") {
		t.Fatalf("error = %v", m.err)
	}
	data, _ = os.ReadFile(path)
	if string(data) != "External\n" {
		t.Fatalf("external file overwritten: %q", data)
	}
}

func TestTreeIsVerticallyCenteredWhenItFits(t *testing.T) {
	m := newTestModel(t)
	view := m.View()
	rootLine := -1
	for index, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "Root") {
			rootLine = index
			break
		}
	}
	want := 1 + (m.contentHeight()-m.layout.Height)/2
	if rootLine != want {
		t.Fatalf("root line = %d, want vertically centered line %d", rootLine, want)
	}
}

func TestViewHasExactTerminalBounds(t *testing.T) {
	m := newTestModel(t)
	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) != 24 {
		t.Fatalf("lines = %d", len(lines))
	}
	for index, line := range lines {
		if width := lipgloss.Width(line); width > 79 {
			t.Fatalf("line %d width = %d", index, width)
		}
	}
}

func TestHelpIsModal(t *testing.T) {
	m := newTestModel(t)
	selected := m.Selected()
	updateModel(m, keyMsg("?"))
	updateModel(m, keyMsg("j"))
	if m.Selected() != selected {
		t.Fatal("help allowed navigation")
	}
	updateModel(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != modeNormal {
		t.Fatal("escape did not close help")
	}
}

func TestSaveFailureKeepsDocumentDirty(t *testing.T) {
	m := newTestModel(t)
	m.path = filepath.Join(t.TempDir(), "missing", "map.hmm")
	m.fingerprint = storage.Fingerprint{Exists: true}
	m.dirty = true
	cmd := updateModel(m, keyMsg("s"))
	if cmd == nil {
		t.Fatal("missing save command")
	}
	msg := cmd()
	saved, ok := msg.(savedMsg)
	if !ok {
		t.Fatalf("message = %T", msg)
	}
	if saved.err == nil {
		saved.err = errors.New("forced failure")
	}
	updateModel(m, saved)
	if !m.Dirty() || m.err == nil {
		t.Fatalf("dirty=%v error=%v", m.Dirty(), m.err)
	}
}
