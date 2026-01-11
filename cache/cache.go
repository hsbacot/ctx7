package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Cache manages the local file cache for ctx7
type Cache struct {
	baseDir string
}

// NewCache creates a new cache manager with the specified directory
func NewCache(dir string) (*Cache, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Create subdirectories
	libsDir := filepath.Join(dir, "libraries")
	searchDir := filepath.Join(dir, "searches")

	if err := os.MkdirAll(libsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create libraries directory: %w", err)
	}

	if err := os.MkdirAll(searchDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create searches directory: %w", err)
	}

	return &Cache{baseDir: dir}, nil
}

// Get retrieves a cache entry for the given library ID
func (c *Cache) Get(libraryID string, maxAge time.Duration) (*CacheEntry, error) {
	return c.GetWithVersion(libraryID, "", maxAge)
}

// GetWithVersion retrieves a cache entry for a specific version
func (c *Cache) GetWithVersion(libraryID, version string, maxAge time.Duration) (*CacheEntry, error) {
	cacheDir := c.getCacheDir(libraryID, version)

	// Check if cache exists and is valid
	metadataPath := filepath.Join(cacheDir, "metadata.json")
	contentPath := filepath.Join(cacheDir, "content.txt")

	// Read metadata
	metadataFile, err := os.Open(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("cache miss: %w", err)
	}
	defer metadataFile.Close()

	var metadata Metadata
	if err := json.NewDecoder(metadataFile).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	// Check if cache is still valid
	if time.Since(metadata.FetchedAt) > maxAge {
		return nil, fmt.Errorf("cache expired")
	}

	// Read content
	contentBytes, err := os.ReadFile(contentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	return &CacheEntry{
		Metadata: metadata,
		Content:  string(contentBytes),
	}, nil
}

// Set saves content and metadata to the cache
func (c *Cache) Set(libraryID, content string, metadata Metadata) error {
	return c.SetWithVersion(libraryID, "", content, metadata)
}

// SetWithVersion saves content for a specific version
func (c *Cache) SetWithVersion(libraryID, version, content string, metadata Metadata) error {
	cacheDir := c.getCacheDir(libraryID, version)

	// Create cache directory
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Write content atomically (write to temp file, then rename)
	contentPath := filepath.Join(cacheDir, "content.txt")
	tmpContentPath := contentPath + ".tmp"

	if err := os.WriteFile(tmpContentPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	if err := os.Rename(tmpContentPath, contentPath); err != nil {
		os.Remove(tmpContentPath)
		return fmt.Errorf("failed to save content: %w", err)
	}

	// Write metadata atomically
	metadataPath := filepath.Join(cacheDir, "metadata.json")
	tmpMetadataPath := metadataPath + ".tmp"

	metadataFile, err := os.Create(tmpMetadataPath)
	if err != nil {
		return fmt.Errorf("failed to create metadata file: %w", err)
	}

	encoder := json.NewEncoder(metadataFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(metadata); err != nil {
		metadataFile.Close()
		os.Remove(tmpMetadataPath)
		return fmt.Errorf("failed to encode metadata: %w", err)
	}
	metadataFile.Close()

	if err := os.Rename(tmpMetadataPath, metadataPath); err != nil {
		os.Remove(tmpMetadataPath)
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	return nil
}

// Clear removes all cached content
func (c *Cache) Clear() error {
	libsDir := filepath.Join(c.baseDir, "libraries")
	searchDir := filepath.Join(c.baseDir, "searches")

	// Remove libraries directory
	if err := os.RemoveAll(libsDir); err != nil {
		return fmt.Errorf("failed to clear libraries cache: %w", err)
	}

	// Remove searches directory
	if err := os.RemoveAll(searchDir); err != nil {
		return fmt.Errorf("failed to clear searches cache: %w", err)
	}

	// Recreate directories
	if err := os.MkdirAll(libsDir, 0755); err != nil {
		return fmt.Errorf("failed to recreate libraries directory: %w", err)
	}

	if err := os.MkdirAll(searchDir, 0755); err != nil {
		return fmt.Errorf("failed to recreate searches directory: %w", err)
	}

	return nil
}

// GetStats returns statistics about the cache
func (c *Cache) GetStats() (*CacheStats, error) {
	stats := &CacheStats{
		CacheDir:    c.baseDir,
		OldestEntry: time.Now(),
		NewestEntry: time.Time{},
	}

	libsDir := filepath.Join(c.baseDir, "libraries")

	err := filepath.Walk(libsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if !info.IsDir() && filepath.Base(path) == "metadata.json" {
			stats.TotalEntries++
			stats.TotalSize += info.Size()

			// Update oldest/newest
			if info.ModTime().Before(stats.OldestEntry) {
				stats.OldestEntry = info.ModTime()
			}
			if info.ModTime().After(stats.NewestEntry) {
				stats.NewestEntry = info.ModTime()
			}

			// Also count content file size
			contentPath := filepath.Join(filepath.Dir(path), "content.txt")
			if contentInfo, err := os.Stat(contentPath); err == nil {
				stats.TotalSize += contentInfo.Size()
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk cache directory: %w", err)
	}

	return stats, nil
}

// IsValid checks if a cache entry exists and is still valid
func (c *Cache) IsValid(libraryID string, maxAge time.Duration) bool {
	_, err := c.Get(libraryID, maxAge)
	return err == nil
}

// getCacheDir returns the cache directory path for a library
func (c *Cache) getCacheDir(libraryID, version string) string {
	// libraryID format: /org/library
	// Remove leading slash and split
	parts := strings.Split(strings.TrimPrefix(libraryID, "/"), "/")

	var versionDir string
	if version != "" {
		versionDir = version
	} else {
		versionDir = "default"
	}

	// Build path: baseDir/libraries/org/library/version
	pathParts := append([]string{c.baseDir, "libraries"}, parts...)
	pathParts = append(pathParts, versionDir)

	return filepath.Join(pathParts...)
}

// CacheSearchResults caches search results with a hash of the query
func (c *Cache) CacheSearchResults(query string, results interface{}) error {
	hash := hashQuery(query)
	searchDir := filepath.Join(c.baseDir, "searches")
	searchPath := filepath.Join(searchDir, hash+".json")

	// Write with timestamp
	cacheData := map[string]interface{}{
		"query":      query,
		"timestamp":  time.Now(),
		"results":    results,
	}

	tmpPath := searchPath + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create search cache file: %w", err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(cacheData); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to encode search cache: %w", err)
	}
	file.Close()

	if err := os.Rename(tmpPath, searchPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to save search cache: %w", err)
	}

	return nil
}

// GetCachedSearchResults retrieves cached search results
func (c *Cache) GetCachedSearchResults(query string, maxAge time.Duration, results interface{}) error {
	hash := hashQuery(query)
	searchPath := filepath.Join(c.baseDir, "searches", hash+".json")

	file, err := os.Open(searchPath)
	if err != nil {
		return fmt.Errorf("search cache miss: %w", err)
	}
	defer file.Close()

	var cacheData map[string]interface{}
	if err := json.NewDecoder(file).Decode(&cacheData); err != nil {
		return fmt.Errorf("failed to decode search cache: %w", err)
	}

	// Check timestamp
	timestampStr, ok := cacheData["timestamp"].(string)
	if !ok {
		return fmt.Errorf("invalid timestamp in cache")
	}

	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return fmt.Errorf("failed to parse timestamp: %w", err)
	}

	if time.Since(timestamp) > maxAge {
		return fmt.Errorf("search cache expired")
	}

	// Extract results
	resultsData, ok := cacheData["results"]
	if !ok {
		return fmt.Errorf("no results in cache")
	}

	// Marshal and unmarshal to convert to target type
	jsonData, err := json.Marshal(resultsData)
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := json.Unmarshal(jsonData, results); err != nil {
		return fmt.Errorf("failed to unmarshal results: %w", err)
	}

	return nil
}

// hashQuery creates a hash of the query string for caching
func hashQuery(query string) string {
	h := sha256.New()
	io.WriteString(h, query)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// ListCachedLibraries returns all cached libraries with their versions
func (c *Cache) ListCachedLibraries() ([]CachedLibrary, error) {
	librariesDir := filepath.Join(c.baseDir, "libraries")

	// Check if libraries directory exists
	if _, err := os.Stat(librariesDir); os.IsNotExist(err) {
		return []CachedLibrary{}, nil
	}

	// Map to group versions by library ID
	libraryMap := make(map[string]*CachedLibrary)

	// Walk the libraries directory
	err := filepath.Walk(librariesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a metadata.json file
		if info.IsDir() || info.Name() != "metadata.json" {
			return nil
		}

		// Read and parse metadata
		data, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip corrupted files
		}

		var metadata Metadata
		if err := json.Unmarshal(data, &metadata); err != nil {
			return nil // Skip corrupted metadata
		}

		// Calculate size (metadata.json + content.txt)
		contentPath := filepath.Join(filepath.Dir(path), "content.txt")
		var size int64 = info.Size()
		if contentInfo, err := os.Stat(contentPath); err == nil {
			size += contentInfo.Size()
		}

		// Extract version from path
		// Path format: .../libraries/org/library/version/metadata.json
		relPath, _ := filepath.Rel(librariesDir, path)
		parts := strings.Split(relPath, string(filepath.Separator))

		if len(parts) < 3 {
			return nil // Invalid path structure
		}

		org := parts[0]
		name := parts[1]
		version := parts[2]
		libraryID := "/" + org + "/" + name

		// Create or get library entry
		lib, exists := libraryMap[libraryID]
		if !exists {
			lib = &CachedLibrary{
				LibraryID:    libraryID,
				Organization: org,
				Name:         name,
				Versions:     []VersionInfo{},
			}
			libraryMap[libraryID] = lib
		}

		// Add version info
		versionInfo := VersionInfo{
			Version:    version,
			IsDefault:  version == "default",
			Size:       size,
			FetchedAt:  metadata.FetchedAt,
			Metadata:   metadata,
		}
		lib.Versions = append(lib.Versions, versionInfo)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk cache directory: %w", err)
	}

	// Convert map to slice
	result := make([]CachedLibrary, 0, len(libraryMap))
	for _, lib := range libraryMap {
		// Sort versions: default first, then by version string
		sort.Slice(lib.Versions, func(i, j int) bool {
			if lib.Versions[i].IsDefault {
				return true
			}
			if lib.Versions[j].IsDefault {
				return false
			}
			return lib.Versions[i].Version < lib.Versions[j].Version
		})
		result = append(result, *lib)
	}

	// Sort libraries by ID
	sort.Slice(result, func(i, j int) bool {
		return result[i].LibraryID < result[j].LibraryID
	})

	return result, nil
}

// RemoveLibrary removes all cached versions of a specific library
func (c *Cache) RemoveLibrary(libraryID string) error {
	// Normalize library ID - remove leading slash if present
	libraryID = strings.TrimPrefix(libraryID, "/")

	// Parse org and library name
	parts := strings.Split(libraryID, "/")
	if len(parts) < 2 {
		return fmt.Errorf("invalid library ID format: %s (expected: org/library)", libraryID)
	}

	org := parts[0]
	library := parts[1]

	// Construct library directory path
	libraryDir := filepath.Join(c.baseDir, "libraries", org, library)

	// Check if library exists
	if _, err := os.Stat(libraryDir); os.IsNotExist(err) {
		return fmt.Errorf("library not found in cache: %s", libraryID)
	}

	// Remove the library directory
	if err := os.RemoveAll(libraryDir); err != nil {
		return fmt.Errorf("failed to remove library: %w", err)
	}

	// Clean up empty parent directory (org folder)
	orgDir := filepath.Join(c.baseDir, "libraries", org)
	if entries, err := os.ReadDir(orgDir); err == nil && len(entries) == 0 {
		os.Remove(orgDir)
	}

	return nil
}

// RemoveLibraryVersion removes a specific version of a library
func (c *Cache) RemoveLibraryVersion(libraryID, version string) error {
	// Normalize library ID
	libraryID = strings.TrimPrefix(libraryID, "/")

	// Get version directory path
	versionDir := c.getCacheDir(libraryID, version)

	// Check if version exists
	if _, err := os.Stat(versionDir); os.IsNotExist(err) {
		return fmt.Errorf("version not found in cache: %s@%s", libraryID, version)
	}

	// Remove the version directory
	if err := os.RemoveAll(versionDir); err != nil {
		return fmt.Errorf("failed to remove version: %w", err)
	}

	// Clean up empty parent directories
	parts := strings.Split(libraryID, "/")
	if len(parts) >= 2 {
		org := parts[0]
		library := parts[1]

		// Check if library directory is now empty
		libraryDir := filepath.Join(c.baseDir, "libraries", org, library)
		if entries, err := os.ReadDir(libraryDir); err == nil && len(entries) == 0 {
			os.Remove(libraryDir)

			// Check if org directory is now empty
			orgDir := filepath.Join(c.baseDir, "libraries", org)
			if entries, err := os.ReadDir(orgDir); err == nil && len(entries) == 0 {
				os.Remove(orgDir)
			}
		}
	}

	return nil
}

// GetDetailedStats returns comprehensive cache statistics with per-library breakdown
func (c *Cache) GetDetailedStats() (*DetailedCacheStats, error) {
	// Get basic stats
	basicStats, err := c.GetStats()
	if err != nil {
		return nil, err
	}

	// Get all cached libraries
	libraries, err := c.ListCachedLibraries()
	if err != nil {
		return nil, err
	}

	// Build library breakdown
	libraryBreakdown := make([]LibraryStats, 0, len(libraries))
	for _, lib := range libraries {
		var totalSize int64
		var oldestVersion, newestVersion time.Time

		for i, v := range lib.Versions {
			totalSize += v.Size
			if i == 0 || v.FetchedAt.Before(oldestVersion) {
				oldestVersion = v.FetchedAt
			}
			if i == 0 || v.FetchedAt.After(newestVersion) {
				newestVersion = v.FetchedAt
			}
		}

		libraryBreakdown = append(libraryBreakdown, LibraryStats{
			LibraryID:      lib.LibraryID,
			VersionCount:   len(lib.Versions),
			TotalSize:      totalSize,
			OldestVersion:  oldestVersion,
			NewestVersion:  newestVersion,
		})
	}

	// Sort by size (largest first)
	sort.Slice(libraryBreakdown, func(i, j int) bool {
		return libraryBreakdown[i].TotalSize > libraryBreakdown[j].TotalSize
	})

	// Calculate search cache stats
	searchDir := filepath.Join(c.baseDir, "searches")
	var searchCacheSize int64
	var searchCacheEntries int

	if entries, err := os.ReadDir(searchDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				if info, err := entry.Info(); err == nil {
					searchCacheSize += info.Size()
					searchCacheEntries++
				}
			}
		}
	}

	return &DetailedCacheStats{
		CacheStats:         *basicStats,
		LibraryBreakdown:   libraryBreakdown,
		SearchCacheSize:    searchCacheSize,
		SearchCacheEntries: searchCacheEntries,
	}, nil
}

// Prune removes cache entries based on age criteria
func (c *Cache) Prune(opts PruneOptions) (*PruneResult, error) {
	libraries, err := c.ListCachedLibraries()
	if err != nil {
		return nil, err
	}

	result := &PruneResult{
		RemovedItems: []string{},
	}

	now := time.Now()

	// Track latest version per library if KeepLatest is true
	latestVersions := make(map[string]string)
	if opts.KeepLatest {
		for _, lib := range libraries {
			var latestTime time.Time
			var latestVersion string
			for _, v := range lib.Versions {
				if v.FetchedAt.After(latestTime) {
					latestTime = v.FetchedAt
					latestVersion = v.Version
				}
			}
			if latestVersion != "" {
				latestVersions[lib.LibraryID] = latestVersion
			}
		}
	}

	// Iterate through all libraries and versions
	for _, lib := range libraries {
		for _, v := range lib.Versions {
			// Check if this is the latest version and should be kept
			if opts.KeepLatest && latestVersions[lib.LibraryID] == v.Version {
				continue
			}

			// Check age
			age := now.Sub(v.FetchedAt)
			if age > opts.MaxAge {
				itemName := lib.LibraryID + "@" + v.Version

				if !opts.DryRun {
					// Remove the version
					if err := c.RemoveLibraryVersion(lib.LibraryID, v.Version); err != nil {
						// Log error but continue
						continue
					}
				}

				result.RemovedCount++
				result.FreedSpace += v.Size
				result.RemovedItems = append(result.RemovedItems, itemName)
			}
		}
	}

	return result, nil
}

// ForceUpdate invalidates cache for a library, forcing fresh fetch
func (c *Cache) ForceUpdate(libraryID, version string) error {
	// If no version specified, remove all versions
	if version == "" {
		return c.RemoveLibrary(libraryID)
	}

	// Remove specific version
	return c.RemoveLibraryVersion(libraryID, version)
}
