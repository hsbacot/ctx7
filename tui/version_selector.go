package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

type versionItem struct {
	version string
	label   string
}

func (i versionItem) Title() string       { return i.label }
func (i versionItem) Description() string { return "" }
func (i versionItem) FilterValue() string { return i.version }

type versionSelectorModel struct {
	list   list.Model
	choice string
	done   bool
}

func newVersionSelector(versions []string) versionSelectorModel {
	items := make([]list.Item, len(versions))

	for i, ver := range versions {
		label := ver
		if len(versions) == 1 && ver == "default" {
			label = "documentation available (not versioned)"
		} else if len(versions) == 1 {
			label = fmt.Sprintf("%s (only version)", ver)
		} else if i == 0 {
			label = fmt.Sprintf("%s (latest)", ver)
		}
		items[i] = versionItem{version: ver, label: label}
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	// Reduce spacing between items
	delegate.SetSpacing(0)
	delegate.SetHeight(1)

	// Calculate height: title (2) + items (min 1 per item, max 10) + help (2) + padding (2)
	itemCount := len(items)
	if itemCount > 10 {
		itemCount = 10
	}
	listHeight := 2 + itemCount + 2 + 2

	l := list.New(items, delegate, 60, listHeight)
	l.Title = "Select a version"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(true)
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		MarginLeft(2)

	return versionSelectorModel{list: l}
}

func (m versionSelectorModel) Update(msg tea.Msg) (versionSelectorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if item, ok := m.list.SelectedItem().(versionItem); ok {
				m.choice = item.version
				m.done = true
				return m, nil
			}
		case "q", "esc":
			m.done = true
			return m, nil
		case "ctrl+c":
			m.done = true
			m.choice = ""
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-2)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m versionSelectorModel) View() string {
	return "\n" + m.list.View()
}
