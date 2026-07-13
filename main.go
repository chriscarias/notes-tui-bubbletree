package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Define our explicit view states
type viewState int

const (
	loginView viewState = iota
	listView
)

// --- DATA MODELS ---

// Note represents the schema of a note resource from the API
type Note struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	IsCompleted bool   `json:"is_completed"`
	UserID      int    `json:"user_id"`
}

// TokenResponse represents the payload received upon successful login
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// --- APPLICATION STATE ---
type model struct {
	currentView viewState
	token       string
	notes       []Note
	cursor      int
	err         error // Added to track and display network/parsing errors

	// Form Inputs
	usernameInput textinput.Model
	passwordInput textinput.Model
}

// Custom message types for our async flow
type notesMsg []Note
type authMsg string
type errMsg error

func initialModel() model {
	u := textinput.New()
	u.Placeholder = "Username"
	u.Focus()

	p := textinput.New()
	p.Placeholder = "Password"
	p.EchoMode = textinput.EchoPassword

	return model{
		currentView:   loginView,
		usernameInput: u,
		passwordInput: p,
		notes:         []Note{},
		cursor:        0,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// --- UPDATE (Event Handler) ---
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "tab":
			if m.usernameInput.Focused() {
				m.usernameInput.Blur()
				m.passwordInput.Focus()
			} else {
				m.passwordInput.Blur()
				m.usernameInput.Focus()
			}
			return m, nil

		case "enter":
			if m.currentView == loginView {
				// Clear any previous errors and fire auth command
				m.err = nil
				return m, loginUserCmd(m.usernameInput.Value(), m.passwordInput.Value())
			}

		case "up", "k", "w":
			if m.currentView == listView && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j", "s":
			if m.currentView == listView && m.cursor < len(m.notes)-1 {
				m.cursor++
			}
		}

	// Catch successful login token
	case authMsg:
		m.token = string(msg)
		m.currentView = listView
		return m, fetchNotesCmd(m.token)

	// Catch successful notes data load
	case notesMsg:
		m.notes = msg
		return m, nil

	// Catch network or parsing errors gracefully
	case errMsg:
		m.err = msg
		return m, nil
	}

	// Keep typing animations fluid in the inputs
	if m.currentView == loginView {
		if m.usernameInput.Focused() {
			m.usernameInput, cmd = m.usernameInput.Update(msg)
		} else {
			m.passwordInput, cmd = m.passwordInput.Update(msg)
		}
	}

	return m, cmd
}

// --- VIEW (The Painter) ---
func (m model) View() string {
	// Global error rendering
	if m.err != nil {
		return fmt.Sprintf("\n  System Error: %v\n  Press 'q' or 'ctrl+c' to quit.\n", m.err)
	}

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#01FF70")).Bold(true).Padding(0, 1)
	s := titleStyle.Render("=== SECURE NOTES ENGINE ===") + "\n\n"

	switch m.currentView {
	case loginView:
		s += "Please Authenticate to access your database:\n\n"
		s += fmt.Sprintf(" %s\n\n", m.usernameInput.View())
		s += fmt.Sprintf(" %s\n\n", m.passwordInput.View())
		s += "Press [Tab] to switch fields • [Enter] to login • [Ctrl+C] to exit"

	case listView:
		s += "Authorized Session Token Active\n"
		s += "Use W/S or J/K to navigate notes. Press 'q' to quit.\n\n"

		if len(m.notes) == 0 {
			s += "  Loading notes or no notes found...\n"
		}

		for i, note := range m.notes {
			cursorStr := "  "
			if m.cursor == i {
				cursorStr = "> "
			}

			// Format the boolean state into a visual checkbox
			status := "[ ]"
			if note.IsCompleted {
				status = "[x]"
			}

			s += fmt.Sprintf("%s %s %s (ID: %d)\n", cursorStr, status, note.Title, note.ID)
		}
	}

	return s
}

// --- COMMANDS (Async Workers) ---

// loginUserCmd executes a real HTTP POST to your PHP API
func loginUserCmd(username, password string) tea.Cmd {
	return func() tea.Msg {
		// Prepare the JSON payload mapped to your PHP requirements
		payload := map[string]string{
			"username": username,
			"password": password,
		}
		jsonPayload, _ := json.Marshal(payload)

		client := &http.Client{Timeout: 5 * time.Second}
		req, err := http.NewRequest("POST", "http://127.0.0.1:8002/login", bytes.NewBuffer(jsonPayload))
		if err != nil {
			return errMsg(err)
		}

		// PHP expects the raw input to be read, so we must set the content type
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return errMsg(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			return errMsg(fmt.Errorf("authentication failed with status: %d", resp.StatusCode))
		}

		// Decode the valid JWT response
		var tokenResp TokenResponse
		if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
			return errMsg(err)
		}

		return authMsg(tokenResp.AccessToken)
	}
}

// fetchNotesCmd executes a secured HTTP GET to fetch the user's notes
func fetchNotesCmd(token string) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 5 * time.Second}
		req, err := http.NewRequest("GET", "http://127.0.0.1:8002/notes", nil)
		if err != nil {
			return errMsg(err)
		}

		// Attach the JWT extracted from the login phase
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			return errMsg(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return errMsg(fmt.Errorf("failed to fetch notes with status: %d", resp.StatusCode))
		}

		// Decode the raw array of JSON objects directly into the slice
		var notes []Note
		if err := json.NewDecoder(resp.Body).Decode(&notes); err != nil {
			return errMsg(err)
		}

		return notesMsg(notes)
	}
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
