package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hsbacot/ctx7/client"
)

type libraryItem struct {
	lib client.Library
}

func (i libraryItem) Title() string {
	// Primary line: Title + Stars + Trust Score
	stars := formatNumber(i.lib.Stars)
	trust := fmt.Sprintf("%.1f", i.lib.TrustScore)
	vip := ""
	if i.lib.VIP {
		vip = " ✨"
	}

	return fmt.Sprintf("%s  ⭐ %s  🏆 %s%s",
		i.lib.Title, stars, trust, vip)
}

func (i libraryItem) Description() string {
	// Secondary line: Org + Updated + Tokens
	org := extractOrg(i.lib.ID)
	updated := formatDate(i.lib.LastUpdateDate)
	tokens := formatTokens(i.lib.TotalTokens)

	desc := fmt.Sprintf("@%s • %s • 🔢 %s\n", org, updated, tokens)

	// Third line: Description (wrapped)
	if i.lib.Description != "" {
		desc += wrapText(i.lib.Description, 70)
	}

	// Fourth line: Version count + Branch
	if len(i.lib.Versions) > 0 {
		desc += fmt.Sprintf("\n[%d versions] • %s branch",
			len(i.lib.Versions), i.lib.Branch)
	}

	return desc
}

func (i libraryItem) FilterValue() string {
	return i.lib.Title + " " + i.lib.Description + " " + i.lib.ID
}

type sortMode int

const (
	sortByStars sortMode = iota
	sortByTrust
	sortByUpdated
	sortByTokens
	sortByRelevance
)

type librarySelectorModel struct {
	list           list.Model
	libraries      []client.Library
	allLibraries   []client.Library // Keep original for filtering
	choice         *client.Library
	done           bool
	sortMode       sortMode
	filterActive   bool
	filterInput    string
}

func newLibrarySelector(libraries []client.Library) librarySelectorModel {
	// Sort by stars by default
	sortedLibs := make([]client.Library, len(libraries))
	copy(sortedLibs, libraries)
	sortLibraries(sortedLibs, sortByStars)

	items := make([]list.Item, len(sortedLibs))
	for i, lib := range sortedLibs {
		items[i] = libraryItem{lib: lib}
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetSpacing(1)
	delegate.SetHeight(5) // Each result takes ~5 lines

	// Show up to 5 libraries at once
	itemCount := len(items)
	if itemCount > 5 {
		itemCount = 5
	}
	listHeight := itemCount*6 + 4

	l := list.New(items, delegate, 80, listHeight)
	l.Title = fmt.Sprintf("🔍 Library Search (%d results)", len(libraries))
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false) // We'll handle filtering ourselves
	l.SetShowHelp(true)
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		MarginLeft(2)

	return librarySelectorModel{
		list:         l,
		libraries:    sortedLibs,
		allLibraries: sortedLibs,
		sortMode:     sortByStars,
	}
}

func (m librarySelectorModel) Update(msg tea.Msg) (librarySelectorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if item, ok := m.list.SelectedItem().(libraryItem); ok {
				m.choice = &item.lib
				m.done = true
				return m, nil
			}
		case "/":
			// Toggle filter mode
			m.filterActive = !m.filterActive
			if !m.filterActive {
				// Reset filter when exiting
				m.filterInput = ""
				m = m.applyFilter()
			}
			return m, nil
		case "s":
			// Cycle through sort modes
			m.sortMode = (m.sortMode + 1) % 5
			m = m.resort()
			return m, nil
		case "q", "esc":
			m.done = true
			return m, nil
		case "ctrl+c":
			m.done = true
			return m, tea.Quit
		default:
			// Handle filter input
			if m.filterActive {
				if msg.String() == "backspace" {
					if len(m.filterInput) > 0 {
						m.filterInput = m.filterInput[:len(m.filterInput)-1]
						m = m.applyFilter()
					}
				} else if len(msg.String()) == 1 {
					m.filterInput += msg.String()
					m = m.applyFilter()
				}
				return m, nil
			}
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-2)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m librarySelectorModel) View() string {
	view := m.list.View()

	// Show filter input if active
	if m.filterActive {
		filterStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)
		view += "\n" + filterStyle.Render(fmt.Sprintf("Filter: %s_", m.filterInput))
	}

	// Show current sort mode
	sortLabel := []string{"Stars", "Trust", "Updated", "Tokens", "Relevance"}[m.sortMode]
	sortStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	view += "\n" + sortStyle.Render(fmt.Sprintf("Sort: %s ▼", sortLabel))

	return "\n" + view
}

// Helper methods

func (m librarySelectorModel) resort() librarySelectorModel {
	sorted := make([]client.Library, len(m.libraries))
	copy(sorted, m.libraries)
	sortLibraries(sorted, m.sortMode)

	items := make([]list.Item, len(sorted))
	for i, lib := range sorted {
		items[i] = libraryItem{lib: lib}
	}

	m.list.SetItems(items)
	m.libraries = sorted
	return m
}

func (m librarySelectorModel) applyFilter() librarySelectorModel {
	if m.filterInput == "" {
		// Reset to all libraries
		m.libraries = m.allLibraries
		return m.resort()
	}

	filtered := []client.Library{}
	for _, lib := range m.allLibraries {
		searchText := strings.ToLower(lib.Title + " " + lib.Description + " " + lib.ID)
		if strings.Contains(searchText, strings.ToLower(m.filterInput)) {
			filtered = append(filtered, lib)
		}
	}

	m.libraries = filtered

	// Resort filtered results
	sortLibraries(filtered, m.sortMode)

	items := make([]list.Item, len(filtered))
	for i, lib := range filtered {
		items[i] = libraryItem{lib: lib}
	}

	m.list.SetItems(items)
	m.list.Title = fmt.Sprintf("🔍 Library Search (%d results)", len(filtered))
	return m
}

// Sorting functions

func sortLibraries(libs []client.Library, mode sortMode) {
	switch mode {
	case sortByStars:
		sort.Slice(libs, func(i, j int) bool {
			return libs[i].Stars > libs[j].Stars
		})
	case sortByTrust:
		sort.Slice(libs, func(i, j int) bool {
			return libs[i].TrustScore > libs[j].TrustScore
		})
	case sortByUpdated:
		sort.Slice(libs, func(i, j int) bool {
			ti, _ := time.Parse(time.RFC3339, libs[i].LastUpdateDate)
			tj, _ := time.Parse(time.RFC3339, libs[j].LastUpdateDate)
			return ti.After(tj)
		})
	case sortByTokens:
		sort.Slice(libs, func(i, j int) bool {
			return libs[i].TotalTokens > libs[j].TotalTokens
		})
	case sortByRelevance:
		sort.Slice(libs, func(i, j int) bool {
			return libs[i].Score > libs[j].Score
		})
	}
}

// Formatting helper functions

func formatNumber(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	} else if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

func formatTokens(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM tokens", float64(n)/1000000)
	} else if n >= 1000 {
		return fmt.Sprintf("%dK tokens", n/1000)
	}
	return fmt.Sprintf("%d tokens", n)
}

func formatDate(dateStr string) string {
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return dateStr
	}

	now := time.Now()
	diff := now.Sub(t)

	if diff < 24*time.Hour {
		return "today"
	} else if diff < 48*time.Hour {
		return "yesterday"
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	} else if diff < 30*24*time.Hour {
		weeks := int(diff.Hours() / 24 / 7)
		return fmt.Sprintf("%d weeks ago", weeks)
	} else if diff < 365*24*time.Hour {
		months := int(diff.Hours() / 24 / 30)
		return fmt.Sprintf("%d months ago", months)
	} else {
		years := int(diff.Hours() / 24 / 365)
		return fmt.Sprintf("%d years ago", years)
	}
}

func extractOrg(id string) string {
	parts := strings.Split(id, "/")
	if len(parts) >= 2 {
		return parts[1] // e.g. "/remix-run/react-router" -> "remix-run"
	}
	return ""
}

func wrapText(text string, width int) string {
	if len(text) <= width {
		return text
	}

	// Simple word wrap
	words := strings.Fields(text)
	var lines []string
	var currentLine string

	for _, word := range words {
		if len(currentLine)+len(word)+1 <= width {
			if currentLine != "" {
				currentLine += " "
			}
			currentLine += word
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	// Limit to 2 lines
	if len(lines) > 2 {
		lines = lines[:2]
		lines[1] += "..."
	}

	return strings.Join(lines, "\n")
}
