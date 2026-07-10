package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// 1. THE MODEL (Application State)
type model struct {
	notes  []string // Hardcoded list of note titles for testing
	cursor int      // Which index in the notes list is currently selected
}

func initialModel() model {
	return model{
		notes:  []string{"Buy groceries", "Finish Go API project", "Learn Bubble Tea TUI", "Read documentation"},
		cursor: 0,
	}
}

// Init is called when the app starts. It can return an initial background command.
func (m model) Init() tea.Cmd {
	return nil // No background commands needed yet
}

// 2. THE UPDATE FUNCTION (Event Handler)
// It takes an incoming message (keypress, API response, etc.) and returns a new model
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Handle keypress events
	case tea.KeyMsg:
		switch msg.String() {

		// Quit the application
		case "ctrl+c", "q":
			return m, tea.Quit

		// Move cursor up
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		// Move cursor down
		case "down", "j":
			if m.cursor < len(m.notes)-1 {
				m.cursor++
			}
		}
	}

	// Return the updated model back to the runtime loop
	return m, nil
}

// 3. THE VIEW FUNCTION (The Painter)
// Reads the current state from the model and translates it into a plain text string
func (m model) View() string {
	// Let's create a title with a bit of styling using Lipgloss
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#01FF70")). // Lime green color
		Bold(true).
		Padding(0, 1)

	s := titleStyle.Render("=== MY NOTES TUI ===") + "\n\n"
	s += "Use up/down arrows or j/k to navigate. Press 'q' to quit.\n\n"

	// Iterate through our data loop (exactly like React's map or PHP's while loop!)
	for i, note := range m.notes {
		// Calculate what the cursor looks like
		cursorStr := "  " // default blank padding
		if m.cursor == i {
			cursorStr = "> " // indicate selection
		}

		// Append the line to our final display string
		s += fmt.Sprintf("%s%s\n", cursorStr, note)
	}

	// The text string returned here is exactly what gets drawn to the terminal screen
	return s
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
