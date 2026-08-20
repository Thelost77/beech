package ui

import "github.com/charmbracelet/lipgloss"

// BranchColorCount is the number of structural colors in the default palette.
const BranchColorCount = 5

// Styles is Beech's curated Everforest interface palette.
type Styles struct {
	Title     lipgloss.Style
	Text      lipgloss.Style
	Muted     lipgloss.Style
	Accent    lipgloss.Style
	Connector lipgloss.Style
	Error     lipgloss.Style
	Status    lipgloss.Style

	foreground         lipgloss.Color
	selectedBackground lipgloss.Color
	accentColor        lipgloss.Color
	warningColor       lipgloss.Color
	infoColor          lipgloss.Color
	purpleColor        lipgloss.Color
	orangeColor        lipgloss.Color
	errorColor         lipgloss.Color
	branchColors       [BranchColorCount]lipgloss.Color
}

// DefaultStyles returns the default Everforest Dark styles.
func DefaultStyles() Styles {
	foreground := lipgloss.Color("#d3c6aa")
	background := lipgloss.Color("#2b3339")
	accent := lipgloss.Color("#a7c080")
	muted := lipgloss.Color("#859289")
	selected := lipgloss.Color("#475258")
	border := lipgloss.Color("#4f585e")
	warning := lipgloss.Color("#dbbc7f")
	info := lipgloss.Color("#7fbbb3")
	purple := lipgloss.Color("#d699b6")
	orange := lipgloss.Color("#e69875")
	errorColor := lipgloss.Color("#e67e80")
	branches := [BranchColorCount]lipgloss.Color{warning, info, purple, orange, accent}

	return Styles{
		Title:              lipgloss.NewStyle().Foreground(accent).Bold(true),
		Text:               lipgloss.NewStyle().Foreground(foreground),
		Muted:              lipgloss.NewStyle().Foreground(muted),
		Accent:             lipgloss.NewStyle().Foreground(accent),
		Connector:          lipgloss.NewStyle().Foreground(border),
		Error:              lipgloss.NewStyle().Foreground(errorColor).Bold(true),
		Status:             lipgloss.NewStyle().Foreground(muted).Background(background),
		foreground:         foreground,
		selectedBackground: selected,
		accentColor:        accent,
		warningColor:       warning,
		infoColor:          info,
		purpleColor:        purple,
		orangeColor:        orange,
		errorColor:         errorColor,
		branchColors:       branches,
	}
}

// RenderCell renders one group of canvas cells from semantic style data.
func (s Styles) RenderCell(key CellStyle, text string) string {
	if key.Role == RoleNone && !key.Selected && key.Feedback == FeedbackNone {
		return text
	}
	return s.cellStyle(key).Render(text)
}

func (s Styles) cellStyle(key CellStyle) lipgloss.Style {
	style := lipgloss.NewStyle().Foreground(s.foreground)
	switch key.Role {
	case RoleRoot:
		style = style.Foreground(s.accentColor).Bold(true)
	case RoleFirstBranch:
		style = style.Foreground(s.branchColor(key.Branch)).Bold(true)
	case RoleBranch:
		style = style.Bold(true)
	case RoleLeaf:
		// Branch membership is carried by connectors; leaves stay neutral.
	case RoleText:
	case RoleConnector:
		if key.Branch >= 0 {
			style = style.Foreground(s.branchColor(key.Branch))
		} else if key.Active {
			style = style.Foreground(s.accentColor)
		} else {
			style = s.Connector
		}
		if key.Active {
			style = style.Bold(true)
		}
	case RoleCollapse:
		style = style.Foreground(s.orangeColor).Bold(true)
	case RoleTaskPending:
		style = style.Foreground(s.warningColor).Bold(true)
	case RoleTaskDone:
		style = style.Foreground(s.accentColor).Bold(true)
	case RoleStrong:
		style = style.Bold(true)
	case RoleCode:
		style = style.Foreground(s.orangeColor)
	case RoleLink:
		style = style.Foreground(s.infoColor).Underline(true)
	case RoleTag:
		style = style.Foreground(s.purpleColor)
	case RoleSyntax:
		style = s.Muted
	}

	if key.Active && (key.Role == RoleRoot || key.Role == RoleFirstBranch || key.Role == RoleBranch) {
		style = style.Bold(true)
	}
	if key.Selected {
		style = style.Background(s.selectedBackground).Foreground(s.foreground).Bold(true)
	}
	if key.Feedback != FeedbackNone {
		style = style.Foreground(s.feedbackColor(key.Feedback)).Bold(true)
		if key.Selected {
			style = style.Background(s.selectedBackground)
		}
	}
	return style
}

func (s Styles) branchColor(index int8) lipgloss.Color {
	if index < 0 {
		return s.foreground
	}
	return s.branchColors[int(index)%len(s.branchColors)]
}

func (s Styles) feedbackColor(kind FeedbackKind) lipgloss.Color {
	switch kind {
	case FeedbackCopy:
		return s.infoColor
	case FeedbackCreate:
		return s.accentColor
	case FeedbackEdit:
		return s.warningColor
	case FeedbackMove:
		return s.purpleColor
	case FeedbackFold:
		return s.orangeColor
	case FeedbackDelete:
		return s.errorColor
	case FeedbackUndo:
		return s.infoColor
	default:
		return s.foreground
	}
}
