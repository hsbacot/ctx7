package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hsbacot/ctx7/cache"
	"github.com/hsbacot/ctx7/cmd"
	"github.com/hsbacot/ctx7/tui"
	"github.com/hsbacot/ctx7/ui"
)

func main() {
	// Check for cache subcommand before parsing flags
	if len(os.Args) > 1 && os.Args[1] == "cache" {
		cacheManager, err := initCache()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing cache: %v\n", err)
			os.Exit(1)
		}
		cmd.RunCacheCommand(os.Args[2:], cacheManager)
		return
	}

	// Parse command-line flags
	interactive := flag.Bool("i", false, "interactive mode - show selection menu for multiple matches")
	flag.BoolVar(interactive, "interactive", false, "interactive mode - show selection menu for multiple matches")

	verbose := flag.Bool("v", false, "verbose mode - show detailed logs")
	flag.BoolVar(verbose, "verbose", false, "verbose mode - show detailed logs")

	noCache := flag.Bool("no-cache", false, "skip cache, force fresh fetch")
	clearCache := flag.Bool("clear-cache", false, "clear all cached content")

	showVersions := flag.Bool("versions", false, "show and select version")
	flag.BoolVar(showVersions, "select-version", false, "show and select version")

	flag.Parse()

	// Handle clear-cache command
	if *clearCache {
		c, err := initCache()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing cache: %v\n", err)
			os.Exit(1)
		}
		if err := c.Clear(); err != nil {
			fmt.Fprintf(os.Stderr, "Error clearing cache: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Cache cleared successfully")
		return
	}

	// Require query argument
	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	query := args[0]

	// Initialize cache
	cacheManager, err := initCache()
	if err != nil && *verbose {
		fmt.Fprintf(os.Stderr, "Warning: Cache unavailable: %v\n", err)
	}

	// Initialize logger
	logger := ui.InitLogger(*verbose)

	// Create and run Bubble Tea model
	opts := tui.Options{
		Interactive:  *interactive,
		Verbose:      *verbose,
		NoCache:      *noCache,
		ShowVersions: *showVersions,
		Logger:       logger,
		Cache:        cacheManager,
	}

	m := tui.NewModel(query, opts)

	// Create program with appropriate options
	// Output TUI to stderr so stdout only contains the final content (for piping)
	var p *tea.Program
	p = tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithOutput(os.Stderr))

	finalModel, err := p.Run()
	if err != nil {
		logger.Error("Application error", "error", err)
		os.Exit(1)
	}

	// Extract final state
	final := finalModel.(tui.Model)

	if final.Err() != nil {
		logger.Error("Fetch failed", "error", final.Err())
		os.Exit(1)
	}

	// Output content to stdout
	fmt.Print(final.Content())
}

func initCache() (*cache.Cache, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	cacheDir := filepath.Join(homeDir, ".cache", "ctx7")
	return cache.NewCache(cacheDir)
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: ctx7 [OPTIONS] <library-name>")
	fmt.Fprintln(os.Stderr, "       ctx7 cache <command> [OPTIONS]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Options:")
	fmt.Fprintln(os.Stderr, "  -i, --interactive       Show selection menu for multiple matches")
	fmt.Fprintln(os.Stderr, "  -v, --verbose           Show detailed logs")
	fmt.Fprintln(os.Stderr, "  --versions              Show version selection menu")
	fmt.Fprintln(os.Stderr, "  --no-cache              Skip cache, force fresh fetch")
	fmt.Fprintln(os.Stderr, "  --clear-cache           Clear all cached content")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Cache Commands:")
	fmt.Fprintln(os.Stderr, "  ctx7 cache stats        Show cache statistics")
	fmt.Fprintln(os.Stderr, "  ctx7 cache list         List all cached libraries")
	fmt.Fprintln(os.Stderr, "  ctx7 cache clear        Clear entire cache")
	fmt.Fprintln(os.Stderr, "  ctx7 cache remove <lib> Remove specific library")
	fmt.Fprintln(os.Stderr, "  ctx7 cache update <lib> Force refresh specific library")
	fmt.Fprintln(os.Stderr, "  ctx7 cache prune        Remove old cache entries")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Examples:")
	fmt.Fprintln(os.Stderr, "  ctx7 react-router")
	fmt.Fprintln(os.Stderr, "  ctx7 -i react")
	fmt.Fprintln(os.Stderr, "  ctx7 --versions react-router")
	fmt.Fprintln(os.Stderr, "  ctx7 cache stats")
	fmt.Fprintln(os.Stderr, "  ctx7 cache prune --days 30")
}
