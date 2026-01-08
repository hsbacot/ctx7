package ui

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/hsbacot/ctx7/client"
)

// SelectLibrary presents an interactive selection menu for choosing a library
func SelectLibrary(libraries []client.Library) (*client.Library, error) {
	if len(libraries) == 0 {
		return nil, errors.New("no libraries to select from")
	}

	var selected string
	options := make([]huh.Option[string], len(libraries))

	// Create options from libraries
	for i, lib := range libraries {
		label := lib.Title
		if lib.Description != "" {
			// Truncate long descriptions
			desc := lib.Description
			if len(desc) > 80 {
				desc = desc[:77] + "..."
			}
			label = fmt.Sprintf("%s - %s", lib.Title, desc)
		}
		if lib.Stars > 0 {
			label = fmt.Sprintf("%s ⭐ %d", label, lib.Stars)
		}

		options[i] = huh.NewOption(label, lib.ID)
	}

	// Create selection form
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Multiple libraries found - choose one:").
				Options(options...).
				Value(&selected),
		),
	)

	// Run the form
	if err := form.Run(); err != nil {
		return nil, err
	}

	// Find and return the selected library
	for i := range libraries {
		if libraries[i].ID == selected {
			return &libraries[i], nil
		}
	}

	return nil, errors.New("selection not found")
}
