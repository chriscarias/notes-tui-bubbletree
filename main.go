package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"encoding/json"
	"net/http"
	"time"
)

// 1. THE MODEL (Application State)
type model struct {
	notes  []string // Hardcoded list of note titles for testing
	cursor int      // Which index in the notes list is currently selected
}

type notesMsg []string

func initialModel() model {
	return model{
		notes: []string{
			"Buy groceries",
			"Finish Go API project",
			"Learn Bubble Tea TUI",
			"Read documentation",
		},
		cursor: 0,
	}
}

// Init is called when the app starts. It can return an initial background command.
func (m model) Init() tea.Cmd {
	return fetchNotesCmd
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
		case "up", "k", "w":
			if m.cursor > 0 {
				m.cursor--
			}

		// Move cursor down
		case "down", "j", "s":
			if m.cursor < len(m.notes)-1 {
				m.cursor++
			}
		}
	case notesMsg:
		m.notes = msg // Swaps the hardcoded notes out for the background data!
		return m, nil
	case errMsg:
		m.notes = []string{"Error: Could not connect to API server.", msg.Error()}
		return m, nil
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

// A custom message type to pass network errors to our Update loop safely
type errMsg error

func fetchNotesCmd() tea.Msg {
	// Create an HTTP client with a built-in timeout safeguard
	client := &http.Client{Timeout: 10 * time.Second}

	// 1. Target your running Go/Python API endpoint (adjust port if needed!)
	resp, err := client.Get("http://127.0.0.1:8002/notes")
	if err != nil {
		return errMsg(err)
	}
	defer resp.Body.Close()

	// 2. Decode the incoming live JSON array into a slice of strings
	var liveNotes []string
	if err := json.NewDecoder(resp.Body).Decode(&liveNotes); err != nil {
		return errMsg(err)
	}

	// 3. Hand the data directly back to Bubble Tea's engine
	return notesMsg(liveNotes)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
