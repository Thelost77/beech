package app

import (
	"strconv"
	"time"

	"github.com/Thelost77/beech/internal/document"
	"github.com/Thelost77/beech/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

const nodeFlashDuration = 167 * time.Millisecond

type actionOutcome struct {
	kind     ui.FeedbackKind
	message  string
	affected []document.NodeID
}

type nodeFeedback struct {
	kind       ui.FeedbackKind
	generation uint64
}

type feedbackExpiredMsg struct {
	generation uint64
	nodes      []document.NodeID
}

func (m *Model) applyOutcome(outcome actionOutcome) tea.Cmd {
	if outcome.message != "" {
		m.setStatus(outcome.message)
	}
	if outcome.kind == ui.FeedbackNone || len(outcome.affected) == 0 {
		return nil
	}
	return m.startFeedback(outcome.kind, outcome.affected)
}

func (m *Model) startFeedback(kind ui.FeedbackKind, ids []document.NodeID) tea.Cmd {
	m.feedbackGeneration++
	generation := m.feedbackGeneration
	if m.feedbackNodes == nil {
		m.feedbackNodes = make(map[document.NodeID]nodeFeedback)
	}
	unique := make(map[document.NodeID]struct{}, len(ids))
	nodes := make([]document.NodeID, 0, len(ids))
	for _, id := range ids {
		if id == document.NoNode {
			continue
		}
		if _, exists := m.doc.Node(id); !exists {
			continue
		}
		if _, exists := unique[id]; exists {
			continue
		}
		unique[id] = struct{}{}
		nodes = append(nodes, id)
		m.feedbackNodes[id] = nodeFeedback{kind: kind, generation: generation}
	}
	if len(nodes) == 0 {
		return nil
	}
	return tea.Tick(nodeFlashDuration, func(time.Time) tea.Msg {
		return feedbackExpiredMsg{generation: generation, nodes: nodes}
	})
}

func (m *Model) handleFeedbackExpired(msg feedbackExpiredMsg) {
	for _, id := range msg.nodes {
		entry, exists := m.feedbackNodes[id]
		if exists && entry.generation == msg.generation {
			delete(m.feedbackNodes, id)
		}
	}
}

func (m *Model) feedbackKind(id document.NodeID) ui.FeedbackKind {
	return m.feedbackNodes[id].kind
}

func (m *Model) subtreeNodeIDs(root document.NodeID) []document.NodeID {
	var nodes []document.NodeID
	var add func(document.NodeID)
	add = func(id document.NodeID) {
		nodes = append(nodes, id)
		for _, child := range m.doc.Children(id) {
			add(child)
		}
	}
	add(root)
	return nodes
}

func describeNodeAction(action, title string, count int) string {
	if count <= 1 {
		return action + " “" + title + "”"
	}
	return action + " “" + title + "” and " + strconv.Itoa(count-1) + " descendants"
}
