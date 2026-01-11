package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
)

// View renders the UI based on the current state
func (m Model) View() string {
	switch m.state {
	case stateCheckingCache:
		return fmt.Sprintf("%s Checking cache...\n",
			spinnerStyle.Render(m.spinner.View()))

	case stateSearching:
		return fmt.Sprintf("%s Searching context7.com for '%s'...\n",
			spinnerStyle.Render(m.spinner.View()), m.query)

	case stateSelectingLibrary:
		return m.librarySelector.View()

	case stateSelectingVersion:
		return m.versionSelector.View()

	case stateFetching:
		lib := "library"
		if m.selectedLib != nil {
			lib = m.selectedLib.Title
		}
		return fmt.Sprintf("%s Fetching llms.txt for %s...\n",
			spinnerStyle.Render(m.spinner.View()), lib)

	case stateSuccess:
		source := "context7.com"
		if m.wasFromCache {
			source = "cache"
		}
		return successStyle.Render(fmt.Sprintf("✓ Fetched from %s\n", source))

	case stateError:
		return errorStyle.Render(fmt.Sprintf("✗ Error: %v\n", m.err))

	default:
		return ""
	}
}
