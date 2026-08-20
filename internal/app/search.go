package app

import (
	"strconv"
	"strings"

	"github.com/Thelost77/beech/internal/document"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"
)

// openSearch shows the bottom search input. The previous query keeps working
// for n and N, but the input starts empty.
func (m *Model) openSearch() (tea.Model, tea.Cmd) {
	m.mode = modeSearch
	m.searchInput.Prompt = "search › "
	m.searchInput.Width = max(10, m.viewWidth())
	m.searchInput.SetValue("")
	m.searchInput.CursorEnd()
	m.clearStatus()
	return m, m.searchInput.Focus()
}

func (m *Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = modeNormal
		m.searchInput.Blur()
		m.setStatus("Search cancelled")
		return m, m.statusExpiryCmd()
	case tea.KeyEnter:
		m.searchQuery = strings.TrimSpace(m.searchInput.Value())
		m.mode = modeNormal
		m.searchInput.Blur()
		statusGeneration := m.statusGeneration
		cmd := m.applyOutcome(m.searchNext(1))
		if m.statusGeneration != statusGeneration {
			if cmd != nil {
				return m, tea.Batch(cmd, m.statusExpiryCmd())
			}
			return m, m.statusExpiryCmd()
		}
		return m, cmd
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

// searchNext selects the next match in document order. With no query it
// reports that nothing has been searched for yet.
func (m *Model) searchNext(direction int) actionOutcome {
	if m.searchQuery == "" {
		return actionOutcome{message: "No search yet"}
	}
	matches := m.searchMatches()
	if len(matches) == 0 {
		return actionOutcome{message: "No matches for “" + m.searchQuery + "”"}
	}
	current := -1
	for position, id := range matches {
		if id == m.selected {
			current = position
			break
		}
	}
	index := 0
	if current >= 0 {
		index = (current + direction + len(matches)) % len(matches)
	}
	target := matches[index]
	m.selected = target
	m.ensureSelectedVisible()
	return actionOutcome{
		message: "Match " + strconv.Itoa(index+1) + "/" + strconv.Itoa(len(matches)) + " “" + m.doc.Text(target) + "”",
	}
}

// searchMatches returns node IDs whose title fuzzily matches the query. The
// matcher accepts query characters in order but not necessarily adjacent, and
// it ignores case. Matches keep document order so n and N cycle spatially.
func (m *Model) searchMatches() []document.NodeID {
	if m.searchQuery == "" {
		return nil
	}
	var ids []document.NodeID
	var texts []string
	var visit func(document.NodeID)
	visit = func(id document.NodeID) {
		ids = append(ids, id)
		texts = append(texts, m.doc.Text(id))
		for _, child := range m.doc.Children(id) {
			visit(child)
		}
	}
	for _, root := range m.doc.Roots() {
		visit(root)
	}
	matches := fuzzy.FindNoSort(m.searchQuery, texts)
	result := make([]document.NodeID, 0, len(matches))
	for _, match := range matches {
		result = append(result, ids[match.Index])
	}
	return result
}
