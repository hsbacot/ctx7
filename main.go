package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hsbacot/ctx7/client"
	"github.com/hsbacot/ctx7/ui"
)

func main() {
	// Parse command-line flags
	interactive := flag.Bool("i", false, "interactive mode - show selection menu for multiple matches")
	flag.BoolVar(interactive, "interactive", false, "interactive mode - show selection menu for multiple matches")
	verbose := flag.Bool("v", false, "verbose mode - show detailed logs")
	flag.BoolVar(verbose, "verbose", false, "verbose mode - show detailed logs")
	flag.Parse()

	// Initialize logger
	logger := ui.InitLogger(*verbose)

	// Get query from arguments
	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: ctx7 [OPTIONS] <library-name>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Options:")
		fmt.Fprintln(os.Stderr, "  -i, --interactive    Show selection menu for multiple matches")
		fmt.Fprintln(os.Stderr, "  -v, --verbose        Show detailed logs")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example:")
		fmt.Fprintln(os.Stderr, "  ctx7 react-router")
		fmt.Fprintln(os.Stderr, "  ctx7 -i react")
		os.Exit(1)
	}

	query := args[0]
	logger.Debug("Starting search", "query", query, "interactive", *interactive)

	// Create API client
	apiClient := client.NewClient()

	// Search for libraries
	logger.Info("Searching context7.com", "query", query)
	results, err := apiClient.SearchLibraries(query)
	if err != nil {
		logger.Error("Search failed", "error", err)
		os.Exit(1)
	}

	logger.Debug("Search completed", "results", len(results))

	// Handle no results
	if len(results) == 0 {
		logger.Warn("No libraries found", "query", query)
		os.Exit(1)
	}

	// Select library
	var selectedLib *client.Library
	if len(results) == 1 {
		// Only one result - use it automatically
		selectedLib = &results[0]
		logger.Info("Found library", "title", selectedLib.Title, "id", selectedLib.ID)
	} else {
		// Multiple results
		if *interactive {
			// Interactive mode - show selection menu
			logger.Info("Found multiple libraries", "count", len(results))
			selected, err := ui.SelectLibrary(results)
			if err != nil {
				logger.Error("Selection failed", "error", err)
				os.Exit(1)
			}
			selectedLib = selected
			logger.Info("Selected library", "title", selectedLib.Title, "id", selectedLib.ID)
		} else {
			// Non-interactive mode - use first result
			selectedLib = &results[0]
			logger.Info("Found multiple libraries, using first match", "title", selectedLib.Title, "id", selectedLib.ID)
			logger.Info("Use -i flag to select interactively")
		}
	}

	// Fetch llms.txt content
	logger.Info("Fetching llms.txt", "library", selectedLib.Title)
	content, err := apiClient.FetchLLMsTxt(selectedLib.ID)
	if err != nil {
		logger.Error("Fetch failed", "library", selectedLib.ID, "error", err)
		os.Exit(1)
	}

	logger.Debug("Fetch completed", "bytes", len(content))
	logger.Info("Success!", "library", selectedLib.Title)

	// Output raw content to stdout
	fmt.Print(content)
}
