package cmd

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hsbacot/ctx7/cache"
)

// RunCacheCommand handles all cache subcommands
func RunCacheCommand(args []string, cacheManager *cache.Cache) {
	if len(args) == 0 {
		printCacheUsage()
		os.Exit(1)
	}

	subcommand := args[0]

	switch subcommand {
	case "stats":
		handleCacheStats(cacheManager, args[1:])
	case "list":
		handleCacheList(cacheManager, args[1:])
	case "clear":
		handleCacheClear(cacheManager, args[1:])
	case "remove":
		handleCacheRemove(cacheManager, args[1:])
	case "update":
		handleCacheUpdate(cacheManager, args[1:])
	case "prune":
		handleCachePrune(cacheManager, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown cache command: %s\n\n", subcommand)
		printCacheUsage()
		os.Exit(1)
	}
}

func printCacheUsage() {
	fmt.Println("Cache Management Commands:")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ctx7 cache stats              Show cache statistics")
	fmt.Println("  ctx7 cache list               List all cached libraries")
	fmt.Println("  ctx7 cache clear              Clear entire cache")
	fmt.Println("  ctx7 cache remove <library>   Remove specific library")
	fmt.Println("  ctx7 cache update <library>   Force refresh specific library")
	fmt.Println("  ctx7 cache prune --days N     Remove entries older than N days")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --json            Output in JSON format (stats, list)")
	fmt.Println("  --force, -f       Skip confirmation prompts")
	fmt.Println("  --dry-run         Preview changes without applying them")
	fmt.Println("  --version <ver>   Target specific version (remove, update)")
	fmt.Println("  --days <N>        Age threshold in days (prune)")
	fmt.Println("  --keep-latest     Keep latest version of each library (prune)")
}

// handleCacheStats shows cache statistics
func handleCacheStats(c *cache.Cache, args []string) {
	fs := flag.NewFlagSet("stats", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "Output in JSON format")
	fs.Parse(args)

	stats, err := c.GetDetailedStats()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting cache stats: %v\n", err)
		os.Exit(1)
	}

	if *jsonOutput {
		if err := printJSON(stats); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Print human-readable stats
	printHeader("Cache Statistics")

	fmt.Printf("Location:        %s\n", stats.CacheDir)
	fmt.Printf("Total Libraries: %d\n", len(stats.LibraryBreakdown))
	fmt.Printf("Total Versions:  %d\n", stats.TotalEntries)
	fmt.Printf("Total Size:      %s\n", formatSize(stats.TotalSize))

	if !stats.OldestEntry.IsZero() {
		fmt.Printf("Oldest Entry:    %s (%s)\n", formatDate(stats.OldestEntry), formatAge(stats.OldestEntry))
	}
	if !stats.NewestEntry.IsZero() {
		fmt.Printf("Newest Entry:    %s (%s)\n", formatDate(stats.NewestEntry), formatAge(stats.NewestEntry))
	}

	// Show top libraries by size
	if len(stats.LibraryBreakdown) > 0 {
		fmt.Println()
		fmt.Println("Top Libraries by Size:")
		count := len(stats.LibraryBreakdown)
		if count > 5 {
			count = 5
		}
		for i := 0; i < count; i++ {
			lib := stats.LibraryBreakdown[i]
			fmt.Printf("  %d. %-30s %10s  (%d versions)\n",
				i+1, lib.LibraryID, formatSize(lib.TotalSize), lib.VersionCount)
		}
	}

	// Show search cache stats
	if stats.SearchCacheEntries > 0 {
		fmt.Println()
		fmt.Printf("Search Cache:    %s (%d entries)\n",
			formatSize(stats.SearchCacheSize), stats.SearchCacheEntries)
	}
}

// handleCacheList lists all cached libraries
func handleCacheList(c *cache.Cache, args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "Output in JSON format")
	fs.Parse(args)

	libraries, err := c.ListCachedLibraries()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing cache: %v\n", err)
		os.Exit(1)
	}

	if len(libraries) == 0 {
		fmt.Println("Cache is empty")
		return
	}

	if *jsonOutput {
		data := map[string]interface{}{
			"libraries":       libraries,
			"total_libraries": len(libraries),
		}
		if err := printJSON(data); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Print human-readable list
	printHeader("Cached Libraries")

	var totalSize int64
	var totalVersions int

	for _, lib := range libraries {
		fmt.Printf("%s\n", lib.LibraryID)
		for _, v := range lib.Versions {
			totalSize += v.Size
			totalVersions++
			defaultMarker := ""
			if v.IsDefault {
				defaultMarker = " (default)"
			}
			fmt.Printf("  └─ %-12s %10s    %s%s\n",
				v.Version, formatSize(v.Size), formatDate(v.FetchedAt), defaultMarker)
		}
		fmt.Println()
	}

	fmt.Printf("Total: %d libraries, %d versions, %s\n",
		len(libraries), totalVersions, formatSize(totalSize))
}

// handleCacheClear clears the entire cache
func handleCacheClear(c *cache.Cache, args []string) {
	fs := flag.NewFlagSet("clear", flag.ExitOnError)
	force := fs.Bool("force", false, "Skip confirmation")
	fs.BoolVar(force, "f", false, "Skip confirmation (shorthand)")
	dryRun := fs.Bool("dry-run", false, "Preview without deleting")
	fs.Parse(args)

	// Get stats for confirmation
	stats, err := c.GetStats()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting cache stats: %v\n", err)
		os.Exit(1)
	}

	if stats.TotalEntries == 0 {
		fmt.Println("Cache is already empty")
		return
	}

	fmt.Printf("⚠️  Warning: This will delete ALL cached libraries (%s)\n\n",
		formatSize(stats.TotalSize))

	if *dryRun {
		fmt.Println("[DRY RUN] Would remove all cache entries")
		return
	}

	if !*force {
		if !confirmAction("Are you sure?") {
			fmt.Println("Cancelled")
			return
		}
	}

	fmt.Println("\nClearing cache...")

	if err := c.Clear(); err != nil {
		fmt.Fprintf(os.Stderr, "Error clearing cache: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Removed %d library versions\n", stats.TotalEntries)
	fmt.Printf("✓ Freed %s of disk space\n\n", formatSize(stats.TotalSize))
	fmt.Println("Cache cleared successfully")
}

// handleCacheRemove removes a specific library or version
func handleCacheRemove(c *cache.Cache, args []string) {
	fs := flag.NewFlagSet("remove", flag.ExitOnError)
	version := fs.String("version", "", "Remove only this version")
	force := fs.Bool("force", false, "Skip confirmation")
	fs.BoolVar(force, "f", false, "Skip confirmation (shorthand)")
	dryRun := fs.Bool("dry-run", false, "Preview without deleting")
	fs.Parse(args)

	if len(fs.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "Error: library ID required")
		fmt.Fprintln(os.Stderr, "Usage: ctx7 cache remove <library-id> [--version <ver>]")
		os.Exit(1)
	}

	libraryID := fs.Arg(0)

	// Get library info for confirmation
	libraries, err := c.ListCachedLibraries()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing cache: %v\n", err)
		os.Exit(1)
	}

	// Find the library
	var targetLib *cache.CachedLibrary
	for i := range libraries {
		if libraries[i].LibraryID == libraryID || libraries[i].LibraryID == "/"+libraryID {
			targetLib = &libraries[i]
			break
		}
	}

	if targetLib == nil {
		fmt.Fprintf(os.Stderr, "Error: library not found in cache: %s\n", libraryID)
		os.Exit(1)
	}

	if *version != "" {
		// Remove specific version
		var targetVersion *cache.VersionInfo
		for i := range targetLib.Versions {
			if targetLib.Versions[i].Version == *version {
				targetVersion = &targetLib.Versions[i]
				break
			}
		}

		if targetVersion == nil {
			fmt.Fprintf(os.Stderr, "Error: version not found: %s@%s\n", libraryID, *version)
			os.Exit(1)
		}

		fmt.Printf("Found cached version: %s@%s\n", targetLib.LibraryID, *version)
		fmt.Printf("  Size: %s\n\n", formatSize(targetVersion.Size))

		if *dryRun {
			fmt.Printf("[DRY RUN] Would remove %s@%s\n", targetLib.LibraryID, *version)
			return
		}

		if !*force {
			if !confirmAction(fmt.Sprintf("Remove %s@%s?", targetLib.LibraryID, *version)) {
				fmt.Println("Cancelled")
				return
			}
		}

		if err := c.RemoveLibraryVersion(targetLib.LibraryID, *version); err != nil {
			fmt.Fprintf(os.Stderr, "Error removing version: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\n✓ Removed %s@%s\n", targetLib.LibraryID, *version)
		fmt.Printf("✓ Freed %s of disk space\n\n", formatSize(targetVersion.Size))
		fmt.Println("Version removed successfully")
	} else {
		// Remove entire library
		var totalSize int64
		for _, v := range targetLib.Versions {
			totalSize += v.Size
		}

		fmt.Printf("Found cached library: %s\n", targetLib.LibraryID)
		fmt.Printf("  Versions: %d (", len(targetLib.Versions))
		for i, v := range targetLib.Versions {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(v.Version)
		}
		fmt.Printf(")\n  Total Size: %s\n\n", formatSize(totalSize))

		if *dryRun {
			fmt.Printf("[DRY RUN] Would remove %s\n", targetLib.LibraryID)
			return
		}

		if !*force {
			if !confirmAction(fmt.Sprintf("Remove this library?")) {
				fmt.Println("Cancelled")
				return
			}
		}

		if err := c.RemoveLibrary(targetLib.LibraryID); err != nil {
			fmt.Fprintf(os.Stderr, "Error removing library: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\n✓ Removed %d versions\n", len(targetLib.Versions))
		fmt.Printf("✓ Freed %s of disk space\n\n", formatSize(totalSize))
		fmt.Println("Library removed successfully")
	}
}

// handleCacheUpdate forces a cache refresh for a library
func handleCacheUpdate(c *cache.Cache, args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	version := fs.String("version", "", "Update only this version")
	fs.Parse(args)

	if len(fs.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "Error: library ID required")
		fmt.Fprintln(os.Stderr, "Usage: ctx7 cache update <library-id> [--version <ver>]")
		os.Exit(1)
	}

	libraryID := fs.Arg(0)

	fmt.Printf("Invalidating cache for: %s", libraryID)
	if *version != "" {
		fmt.Printf("@%s", *version)
	}
	fmt.Println()

	if err := c.ForceUpdate(libraryID, *version); err != nil {
		fmt.Fprintf(os.Stderr, "Error invalidating cache: %v\n", err)
		os.Exit(1)
	}

	if *version != "" {
		fmt.Printf("\n✓ Cache invalidated for %s@%s\n", libraryID, *version)
	} else {
		fmt.Printf("\n✓ Cache invalidated for all versions\n")
	}
	fmt.Println("✓ Next fetch will retrieve fresh content from context7.com")
	fmt.Printf("\nTo fetch now, run: ctx7 %s\n", libraryID)
}

// handleCachePrune removes old cache entries
func handleCachePrune(c *cache.Cache, args []string) {
	fs := flag.NewFlagSet("prune", flag.ExitOnError)
	days := fs.Int("days", 0, "Remove entries older than this many days")
	keepLatest := fs.Bool("keep-latest", false, "Keep latest version of each library")
	force := fs.Bool("force", false, "Skip confirmation")
	fs.BoolVar(force, "f", false, "Skip confirmation (shorthand)")
	dryRun := fs.Bool("dry-run", false, "Preview without deleting")
	fs.Parse(args)

	if *days <= 0 {
		fmt.Fprintln(os.Stderr, "Error: --days flag is required and must be positive")
		fmt.Fprintln(os.Stderr, "Usage: ctx7 cache prune --days N [--keep-latest] [--force]")
		os.Exit(1)
	}

	maxAge := time.Duration(*days) * 24 * time.Hour

	fmt.Printf("Analyzing cache entries older than %d days...\n\n", *days)

	result, err := c.Prune(cache.PruneOptions{
		MaxAge:     maxAge,
		DryRun:     true, // Always dry-run first to show what would be deleted
		KeepLatest: *keepLatest,
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error analyzing cache: %v\n", err)
		os.Exit(1)
	}

	if result.RemovedCount == 0 {
		fmt.Println("No stale entries found")
		return
	}

	fmt.Printf("Found %d stale entries:\n", result.RemovedCount)
	for _, item := range result.RemovedItems {
		fmt.Printf("  └─ %s\n", item)
	}
	fmt.Printf("\nTotal: %s to be freed\n\n", formatSize(result.FreedSpace))

	if *dryRun {
		fmt.Println("[DRY RUN] Preview complete")
		return
	}

	if !*force {
		if !confirmAction("Prune these entries?") {
			fmt.Println("Cancelled")
			return
		}
	}

	fmt.Println("\nPruning cache...")

	// Actually prune
	result, err = c.Prune(cache.PruneOptions{
		MaxAge:     maxAge,
		DryRun:     false,
		KeepLatest: *keepLatest,
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error pruning cache: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Removed %d entries\n", result.RemovedCount)
	fmt.Printf("✓ Freed %s of disk space\n\n", formatSize(result.FreedSpace))
	fmt.Println("Cache pruned successfully")
}
