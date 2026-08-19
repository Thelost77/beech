package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const statusDuration = 500 * time.Millisecond

type statusExpiredMsg struct {
	generation uint64
}

func (m *Model) setStatus(value string) {
	m.status = value
	m.statusGeneration++
}

func (m *Model) clearStatus() {
	m.status = ""
	m.statusGeneration++
}

func (m *Model) statusExpiryCmd() tea.Cmd {
	generation := m.statusGeneration
	return tea.Tick(statusDuration, func(time.Time) tea.Msg {
		return statusExpiredMsg{generation: generation}
	})
}

func (m *Model) handleStatusExpired(msg statusExpiredMsg) {
	if msg.generation != m.statusGeneration {
		return
	}
	m.status = ""
}
