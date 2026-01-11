package cache

import "time"

// Metadata stores metadata about a cached library
type Metadata struct {
	LibraryID      string    `json:"library_id"`
	Title          string    `json:"title"`
	Version        string    `json:"version,omitempty"`
	FetchedAt      time.Time `json:"fetched_at"`
	LastUpdateDate string    `json:"last_update_date"`
	TotalTokens    int       `json:"total_tokens"`
	TotalSnippets  int       `json:"total_snippets"`
	Stars          int       `json:"stars"`
	TrustScore     float64   `json:"trust_score"`
	Versions       []string  `json:"versions"`
}

// CacheEntry represents a complete cache entry with metadata and content
type CacheEntry struct {
	Metadata Metadata
	Content  string
}

// CacheStats contains statistics about the cache
type CacheStats struct {
	TotalEntries  int
	TotalSize     int64
	OldestEntry   time.Time
	NewestEntry   time.Time
	CacheDir      string
}

// CachedLibrary represents a library entry in the cache with all its versions
type CachedLibrary struct {
	LibraryID    string
	Organization string
	Name         string
	Versions     []VersionInfo
}

// VersionInfo contains information about a specific cached version
type VersionInfo struct {
	Version    string
	IsDefault  bool
	Size       int64
	FetchedAt  time.Time
	Metadata   Metadata
}

// DetailedCacheStats extends CacheStats with per-library breakdown
type DetailedCacheStats struct {
	CacheStats          // Embedded basic stats
	LibraryBreakdown    []LibraryStats
	SearchCacheSize     int64
	SearchCacheEntries  int
}

// LibraryStats contains statistics for a single library
type LibraryStats struct {
	LibraryID      string
	VersionCount   int
	TotalSize      int64
	OldestVersion  time.Time
	NewestVersion  time.Time
}

// PruneOptions configures cache pruning behavior
type PruneOptions struct {
	MaxAge       time.Duration
	DryRun       bool
	KeepLatest   bool  // Keep latest version of each library
}

// PruneResult contains information about pruned entries
type PruneResult struct {
	RemovedCount  int
	FreedSpace    int64
	RemovedItems  []string
}
