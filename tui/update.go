package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/hsbacot/ctx7/cache"
)

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.checkCache(),
	)
}

// Update handles messages and state transitions
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// Handle library selector input when in that state
		if m.state == stateSelectingLibrary {
			var cmd tea.Cmd
			m.librarySelector, cmd = m.librarySelector.Update(msg)

			if m.librarySelector.done {
				if m.librarySelector.choice == nil {
					// User cancelled
					m.err = fmt.Errorf("cancelled")
					m.state = stateError
					return m, tea.Quit
				}
				// User selected a library
				m.selectedLib = m.librarySelector.choice
				return m, m.checkLibraryCache()
			}
			return m, cmd
		}

		// Handle version selector input when in that state
		if m.state == stateSelectingVersion {
			var cmd tea.Cmd
			m.versionSelector, cmd = m.versionSelector.Update(msg)

			if m.versionSelector.done {
				if m.versionSelector.choice == "" {
					// User cancelled
					m.err = fmt.Errorf("cancelled")
					m.state = stateError
					return m, tea.Quit
				}
				// User selected a version
				m.selectedVer = m.versionSelector.choice
				// Check version-specific cache
				if m.cache != nil && !m.noCache {
					entry, err := m.cache.GetWithVersion(m.selectedLib.ID, m.selectedVer, 24*time.Hour)
					if err == nil {
						// Version cached!
						m.content = entry.Content
						m.wasFromCache = true
						m.state = stateSuccess
						return m, tea.Quit
					}
				}
				// Not cached, fetch it
				m.state = stateFetching
				versionID := fmt.Sprintf("%s/%s", m.selectedLib.ID, m.selectedVer)
				return m, m.fetchContent(versionID)
			}
			return m, cmd
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case cacheCheckCompleteMsg:
		if msg.found && !m.noCache && !m.showVersions {
			// Only use cache immediately if NOT showing versions
			m.cacheEntry = msg.entry
			m.content = msg.entry.Content
			m.wasFromCache = true
			m.state = stateSuccess
			return m, tea.Quit
		}

		// Cache miss OR showVersions is true
		if m.selectedLib != nil {
			// We already searched, now need to fetch or select version
			if m.showVersions {
				m.state = stateSelectingVersion
				// Initialize the version selector
				versions := m.selectedLib.Versions
				if len(versions) == 0 {
					versions = []string{"default"}
				}
				m.versionSelector = newVersionSelector(versions)
				return m, nil
			}
			// Use cached content if we have it and not showing versions
			if msg.found && msg.entry != nil {
				m.content = msg.entry.Content
				m.wasFromCache = true
				m.state = stateSuccess
				return m, tea.Quit
			}
			m.state = stateFetching
			return m, m.fetchContent(m.selectedLib.ID)
		}
		// No library selected yet, proceed with search
		m.state = stateSearching
		return m, m.searchLibraries()

	case searchCompleteMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateError
			return m, tea.Quit
		}

		m.searchResults = msg.results

		if len(msg.results) == 0 {
			m.err = fmt.Errorf("no libraries found")
			m.state = stateError
			return m, tea.Quit
		}

		if len(msg.results) == 1 {
			// Single result, check cache first
			m.selectedLib = &msg.results[0]
			return m, m.checkLibraryCache()
		}

		// Multiple results
		if m.interactive {
			m.state = stateSelectingLibrary
			m.librarySelector = newLibrarySelector(msg.results)
			return m, nil
		}

		// Use first result, check cache
		m.selectedLib = &msg.results[0]
		return m, m.checkLibraryCache()


	case fetchCompleteMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateError
			return m, tea.Quit
		}

		m.content = msg.content
		m.state = stateSuccess

		// Cache the result
		if m.cache != nil && !m.noCache && m.selectedLib != nil {
			metadata := cache.Metadata{
				LibraryID:      m.selectedLib.ID,
				Title:          m.selectedLib.Title,
				Version:        m.selectedVer,
				FetchedAt:      time.Now(),
				LastUpdateDate: m.selectedLib.LastUpdateDate,
				TotalTokens:    m.selectedLib.TotalTokens,
				TotalSnippets:  m.selectedLib.TotalSnippets,
				Stars:          m.selectedLib.Stars,
				TrustScore:     m.selectedLib.TrustScore,
				Versions:       m.selectedLib.Versions,
			}
			_ = m.cache.SetWithVersion(m.selectedLib.ID, m.selectedVer, msg.content, metadata)
		}

		return m, tea.Quit

	case errorMsg:
		m.err = msg.err
		m.state = stateError
		return m, tea.Quit
	}

	return m, nil
}

// Command functions (run async)

func (m Model) checkCache() tea.Cmd {
	return func() tea.Msg {
		// Initial cache check is skipped - go straight to search
		return cacheCheckCompleteMsg{found: false}
	}
}

func (m Model) checkLibraryCache() tea.Cmd {
	return func() tea.Msg {
		if m.cache == nil || m.noCache {
			return cacheCheckCompleteMsg{found: false}
		}

		entry, err := m.cache.Get(m.selectedLib.ID, 24*time.Hour)
		if err != nil {
			return cacheCheckCompleteMsg{found: false}
		}

		return cacheCheckCompleteMsg{
			entry: entry,
			found: true,
		}
	}
}

func (m Model) searchLibraries() tea.Cmd {
	return func() tea.Msg {
		results, err := m.client.SearchLibraries(m.query)
		return searchCompleteMsg{
			results: results,
			err:     err,
		}
	}
}

func (m Model) fetchContent(libraryID string) tea.Cmd {
	return func() tea.Msg {
		content, err := m.client.FetchLLMsTxt(libraryID)
		return fetchCompleteMsg{
			content: content,
			err:     err,
		}
	}
}

