package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/hsbacot/ctx7/cache"
	"github.com/hsbacot/ctx7/client"
)

type state int

const (
	stateInitializing state = iota
	stateCheckingCache
	stateSearching
	stateSelectingLibrary
	stateSelectingVersion
	stateFetching
	stateSuccess
	stateError
)

// Options contains configuration for the Model
type Options struct {
	Interactive  bool
	Verbose      bool
	NoCache      bool
	ShowVersions bool
	Logger       *log.Logger
	Cache        *cache.Cache
}

// Model is the Bubble Tea model for ctx7
type Model struct {
	// Configuration
	query        string
	interactive  bool
	verbose      bool
	noCache      bool
	showVersions bool

	// State
	state state
	err   error

	// Data
	searchResults []client.Library
	selectedLib   *client.Library
	selectedVer   string
	content       string
	cacheEntry    *cache.CacheEntry

	// UI Components
	spinner         spinner.Model
	versionSelector versionSelectorModel
	librarySelector librarySelectorModel
	logger          *log.Logger

	// Services
	client *client.Client
	cache  *cache.Cache

	// Flags
	wasFromCache bool
}

// NewModel creates a new Bubble Tea model
func NewModel(query string, opts Options) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		query:        query,
		interactive:  opts.Interactive,
		verbose:      opts.Verbose,
		noCache:      opts.NoCache,
		showVersions: opts.ShowVersions,
		state:        stateInitializing,
		spinner:      s,
		logger:       opts.Logger,
		client:       client.NewClient(),
		cache:        opts.Cache,
	}
}

// Err returns the error if one occurred
func (m Model) Err() error {
	return m.err
}

// Content returns the fetched content
func (m Model) Content() string {
	return m.content
}

// WasFromCache returns true if content was loaded from cache
func (m Model) WasFromCache() bool {
	return m.wasFromCache
}
