package ui

import (
	"os"

	"github.com/charmbracelet/log"
)

// InitLogger initializes and configures a Charm logger
func InitLogger(verbose bool) *log.Logger {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    verbose,
		ReportTimestamp: verbose,
	})

	if verbose {
		logger.SetLevel(log.DebugLevel)
	} else {
		logger.SetLevel(log.InfoLevel)
	}

	return logger
}
