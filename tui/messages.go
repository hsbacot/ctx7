package tui

import (
	"github.com/hsbacot/ctx7/cache"
	"github.com/hsbacot/ctx7/client"
)

// Message types for Bubble Tea state transitions

type searchCompleteMsg struct {
	results []client.Library
	err     error
}

type fetchCompleteMsg struct {
	content string
	err     error
}

type cacheCheckCompleteMsg struct {
	entry *cache.CacheEntry
	found bool
}

type errorMsg struct {
	err error
}
