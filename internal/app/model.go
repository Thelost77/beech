package app

import (
	"bytes"
	"io/fs"
	"slices"

	"github.com/Thelost77/beech/internal/document"
	"github.com/Thelost77/beech/internal/layout"
	"github.com/Thelost77/beech/internal/outline"
	"github.com/Thelost77/beech/internal/storage"
	"github.com/Thelost77/beech/internal/ui"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const maxHistory = 100

type mode uint8

const (
	modeNormal mode = iota
	modeEdit
	modeSaveAs
	modeHelp
)

type historyEntry struct {
	doc       *document.Document
	selected  document.NodeID
	collapsed map[document.NodeID]bool
}

type editSession struct {
	Target      document.NodeID
	Before      *historyEntry
	Layout      layout.Result
	EditorWidth int
}

type savedMsg struct {
	path        string
	data        []byte
	fingerprint storage.Fingerprint
	revision    uint64
	err         error
}

// InitialDocument describes the file state used to create a model.
type InitialDocument struct {
	Document      *document.Document
	Collapsed     map[document.NodeID]bool
	Path          string
	SuggestedPath string
	Imported      bool
	Style         outline.Style
	Fingerprint   storage.Fingerprint
	FileMode      fs.FileMode
	SavedData     []byte
	Dirty         bool
}

// Model is the Beech Bubble Tea application model.
type Model struct {
	doc                *document.Document
	selected           document.NodeID
	collapsed          map[document.NodeID]bool
	clipboard          *document.Branch
	feedbackNodes      map[document.NodeID]nodeFeedback
	feedbackGeneration uint64
	layout             layout.Result

	viewportX int
	viewportY int
	width     int
	height    int

	mode      mode
	nodeInput textinput.Model
	pathInput textinput.Model
	edit      *editSession

	undo []historyEntry
	redo []historyEntry

	path          string
	suggestedPath string
	imported      bool
	style         outline.Style
	fingerprint   storage.Fingerprint
	fileMode      fs.FileMode
	lastSaved     []byte
	dirty         bool
	saving        bool
	revision      uint64

	pendingSavePath string
	quitAfterSave   bool

	status           string
	statusGeneration uint64
	err              error
	styles           ui.Styles
}

// New creates the root application model.
func New(initial InitialDocument) *Model {
	doc := initial.Document
	if doc == nil || doc.Len() == 0 {
		doc = document.New("New map")
	}
	style := initial.Style
	if style.Newline == "" {
		style = outline.DefaultStyle()
	}
	nodeInput := textinput.New()
	nodeInput.CharLimit = 1024
	nodeInput.Width = 43
	pathInput := textinput.New()
	pathInput.Prompt = "› "
	pathInput.CharLimit = 4096
	pathInput.Width = 60

	m := &Model{
		doc:           doc,
		selected:      doc.Roots()[0],
		collapsed:     cloneCollapsed(initial.Collapsed),
		path:          initial.Path,
		suggestedPath: initial.SuggestedPath,
		imported:      initial.Imported,
		style:         style,
		fingerprint:   initial.Fingerprint,
		fileMode:      initial.FileMode,
		lastSaved:     slices.Clone(initial.SavedData),
		dirty:         initial.Dirty,
		nodeInput:     nodeInput,
		pathInput:     pathInput,
		styles:        ui.DefaultStyles(),
	}
	m.configureInputStyles()
	m.rebuildLayout()
	if !initial.Dirty && initial.SavedData == nil {
		m.lastSaved, _ = outline.Serialize(m.doc, m.collapsed, m.style)
	}
	return m
}

func (m *Model) configureInputStyles() {
	configureTextInput(&m.nodeInput, m.styles)
	configureTextInput(&m.pathInput, m.styles)
}

func configureTextInput(input *textinput.Model, styles ui.Styles) {
	input.PromptStyle = styles.Accent
	input.TextStyle = styles.Text
	input.PlaceholderStyle = styles.Muted
	input.Cursor.Style = styles.Accent
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	return nil
}

// Document returns a deep copy of the current document.
func (m *Model) Document() *document.Document {
	return m.doc.Clone()
}

// Selected returns the active node.
func (m *Model) Selected() document.NodeID {
	return m.selected
}

// Dirty reports whether the in-memory document differs from the last save.
func (m *Model) Dirty() bool {
	return m.dirty
}

// Path returns the active document path.
func (m *Model) Path() string {
	return m.path
}

func (m *Model) snapshot() historyEntry {
	return historyEntry{
		doc:       m.doc.Clone(),
		selected:  m.selected,
		collapsed: cloneCollapsed(m.collapsed),
	}
}

func (m *Model) restore(entry historyEntry) {
	m.doc = entry.doc.Clone()
	m.selected = entry.selected
	m.collapsed = cloneCollapsed(entry.collapsed)
	if _, ok := m.doc.Node(m.selected); !ok {
		m.selected = m.doc.Roots()[0]
	}
	m.rebuildLayout()
	m.ensureSelectedVisible()
}

func (m *Model) pushUndo(entry historyEntry) {
	m.undo = append(m.undo, entry)
	if len(m.undo) > maxHistory {
		m.undo = append([]historyEntry(nil), m.undo[len(m.undo)-maxHistory:]...)
	}
	m.redo = nil
}

func (m *Model) changed() {
	m.revision++
	m.refreshDirty()
	m.rebuildLayout()
	m.ensureSelectedVisible()
}

func (m *Model) refreshDirty() {
	data, err := outline.Serialize(m.doc, m.collapsed, m.style)
	m.dirty = err != nil || !bytes.Equal(data, m.lastSaved)
}

func cloneCollapsed(source map[document.NodeID]bool) map[document.NodeID]bool {
	clone := make(map[document.NodeID]bool, len(source))
	for id, value := range source {
		clone[id] = value
	}
	return clone
}
