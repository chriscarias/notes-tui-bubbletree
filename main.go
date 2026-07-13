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

// Define the target API base URL here to easily swap between PHP, Go, and Python
var baseURL = "http://127.0.0.1:8002"

// Define our explicit view states
type viewState int

const (
	loginView viewState = iota
	listView
	createView
	deleteView
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

	// Create Note
	noteTitleInput   textinput.Model
	noteContentInput textinput.Model
}

// Custom message types for our async flow
type notesMsg []Note
type authMsg string
type errMsg error
type noteCreatedMsg struct{}
type noteUpdatedMsg struct{}
type noteDeletedMsg struct{}

func initialModel() model {
	u := textinput.New()
	u.Placeholder = "Username"
	u.Focus()

	p := textinput.New()
	p.Placeholder = "Password"
	p.EchoMode = textinput.EchoPassword

	// Setup Note Title Input
	nt := textinput.New()
	nt.Placeholder = "Note Title"

	// Setup Note Content Input
	nc := textinput.New()
	nc.Placeholder = "Note Content"

	return model{
		currentView:      loginView,
		usernameInput:    u,
		passwordInput:    p,
		notes:            []Note{},
		noteTitleInput:   nt,
		noteContentInput: nc,
		cursor:           0,
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
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			// Only allow 'q' to quit if we are in the list view or an error screen
			if m.currentView == listView || m.err != nil {
				return m, tea.Quit
			}
			// If we are typing in loginView or createView, we do nothing here,
			// allowing the 'q' to fall through to the text input updates below

		case "enter":
			if m.currentView == loginView {
				// Clear any previous errors and fire auth command
				m.err = nil
				return m, loginUserCmd(m.usernameInput.Value(), m.passwordInput.Value())
			} else if m.currentView == createView {
				// Don't submit if the title is empty
				if m.noteTitleInput.Value() == "" {
					return m, nil
				}

				m.err = nil
				// Fire the creation command with title, content, and the session token
				return m, createNoteCmd(m.noteTitleInput.Value(), m.noteContentInput.Value(), m.token)
			}

		case " ":
			if m.currentView == listView && len(m.notes) > 0 {
				selectedNote := m.notes[m.cursor]
				m.err = nil // Clear any lingering errors
				return m, toggleNoteCmd(selectedNote.ID, selectedNote.IsCompleted, m.token)
			}

		// Trigger the delete confirmation view
		case "d":
			if m.currentView == listView && len(m.notes) > 0 {
				m.currentView = deleteView
				return m, nil
			}

		// Confirm deletion
		case "y":
			if m.currentView == deleteView {
				selectedNote := m.notes[m.cursor]
				m.err = nil
				return m, deleteNoteCmd(selectedNote.ID, m.token)
			}

		// Cancel deletion (explicit 'n' key)
		case "n":
			if m.currentView == deleteView {
				m.currentView = listView
				return m, nil
			}
			// Let it fall through if we are in createView so 'n' can be typed!
			if m.currentView == listView {
				m.currentView = createView
				m.noteTitleInput.Focus()
				return m, nil
			}

		case "up", "k", "w":
			if m.currentView == listView && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j", "s":
			if m.currentView == listView && m.cursor < len(m.notes)-1 {
				m.cursor++
			}
		// Add "esc" to act as a cancel/back button
		case "esc":
			// If there's an error, clear it and stop processing so the user can retry
			if m.err != nil {
				m.err = nil
				return m, nil
			}

			// Otherwise, handle normal view cancellation
			if m.currentView == createView || m.currentView == deleteView {
				m.currentView = listView
				m.noteTitleInput.SetValue("")
				m.noteContentInput.SetValue("")
			}

		// Modify the "tab" case to handle both forms
		case "tab":
			if m.currentView == loginView {
				if m.usernameInput.Focused() {
					m.usernameInput.Blur()
					m.passwordInput.Focus()
				} else {
					m.passwordInput.Blur()
					m.usernameInput.Focus()
				}
			} else if m.currentView == createView {
				if m.noteTitleInput.Focused() {
					m.noteTitleInput.Blur()
					m.noteContentInput.Focus()
				} else {
					m.noteContentInput.Blur()
					m.noteTitleInput.Focus()
				}
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

	// Catch successful note creation
	case noteCreatedMsg:
		m.noteTitleInput.SetValue("")
		m.noteContentInput.SetValue("")
		m.currentView = listView
		return m, fetchNotesCmd(m.token)

	case noteUpdatedMsg:
		// Fetch the updated list from the server so the checkbox updates visually
		return m, fetchNotesCmd(m.token)

	// Catch successful note deletion
	case noteDeletedMsg:
		m.currentView = listView
		// Disarm the out-of-bounds trap!
		if m.cursor > 0 && m.cursor == len(m.notes)-1 {
			m.cursor--
		}
		return m, fetchNotesCmd(m.token)

	}

	// Keep typing animations fluid in the active inputs
	if m.currentView == loginView {
		if m.usernameInput.Focused() {
			m.usernameInput, cmd = m.usernameInput.Update(msg)
		} else {
			m.passwordInput, cmd = m.passwordInput.Update(msg)
		}
	} else if m.currentView == createView {
		if m.noteTitleInput.Focused() {
			m.noteTitleInput, cmd = m.noteTitleInput.Update(msg)
		} else {
			m.noteContentInput, cmd = m.noteContentInput.Update(msg)
		}
	}

	return m, cmd
}

// --- VIEW (The Painter) ---
func (m model) View() string {
	// Global error rendering
	if m.err != nil {
		return fmt.Sprintf("\n  System Error: %v\n  Press 'q' or 'ctrl+c' to quit or 'Esc' to retry.\n", m.err)
	}

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#01FF70")).Bold(true).Padding(0, 1)
	s := titleStyle.Render("=== POLYGLOT NOTES TUI ===") + "\n\n"

	switch m.currentView {
	case loginView:
		s += "Please Authenticate to access your database:\n\n"
		s += fmt.Sprintf(" %s\n\n", m.usernameInput.View())
		s += fmt.Sprintf(" %s\n\n", m.passwordInput.View())
		s += "Press [Tab] to switch fields • [Enter] to login • [Ctrl+C] or 'q' to exit"

	case listView:
		s += "Authorized Session Token Active\n"
		s += "Use Up/Down or W/S or J/K to navigate notes. Press 'n' to create a new note. Press 'd' to delete a note. \nPress 'q' or 'ctrl+c' to quit.\n\n"

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
	case createView:
		s += "--- Create a New Note ---\n\n"
		s += fmt.Sprintf(" %s\n\n", m.noteTitleInput.View())
		s += fmt.Sprintf(" %s\n\n", m.noteContentInput.View())
		s += "Press [Tab] to switch fields • [Enter] to save • [Esc] to cancel"

	case deleteView:
		// Target the exact note we are threatening to delete
		selectedNote := m.notes[m.cursor]

		warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4136")).Bold(true)
		s += warningStyle.Render("!!! WARNING: PERMANENT DELETION !!!") + "\n\n"

		s += fmt.Sprintf("Are you sure you want to cast Obliterate on the note: '%s'?\n\n", selectedNote.Title)
		s += "Press 'y' to confirm • 'n' or [Esc] to cancel"

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
		req, err := http.NewRequest("POST", baseURL+"/login", bytes.NewBuffer(jsonPayload))
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
		req, err := http.NewRequest("GET", baseURL+"/notes", nil)
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

// createNoteCmd executes a POST request to save a new note
func createNoteCmd(title, content, token string) tea.Cmd {
	return func() tea.Msg {
		payload := map[string]interface{}{
			"title":   title,
			"content": content,
		}
		jsonPayload, _ := json.Marshal(payload)

		client := &http.Client{Timeout: 5 * time.Second}
		req, err := http.NewRequest("POST", baseURL+"/notes", bytes.NewBuffer(jsonPayload))
		if err != nil {
			return errMsg(err)
		}

		// Set headers: JSON content type and the Bearer token for authentication
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			return errMsg(err)
		}
		defer resp.Body.Close()

		// Your PHP API returns 201 Created on success
		if resp.StatusCode != http.StatusCreated {
			return errMsg(fmt.Errorf("failed to create note with status: %d", resp.StatusCode))
		}

		return noteCreatedMsg{}
	}
}

// toggleNoteCmd executes a PATCH request to flip the completion status
func toggleNoteCmd(noteID int, currentStatus bool, token string) tea.Cmd {
	return func() tea.Msg {
		payload := map[string]interface{}{
			"is_completed": !currentStatus,
		}
		jsonPayload, _ := json.Marshal(payload)

		client := &http.Client{Timeout: 5 * time.Second}
		url := fmt.Sprintf("%s/notes/%d", baseURL, noteID)
		req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonPayload))
		if err != nil {
			return errMsg(err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			return errMsg(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return errMsg(fmt.Errorf("failed to update note with status: %d", resp.StatusCode))
		}

		return noteUpdatedMsg{}
	}
}

// deleteNoteCmd executes a DELETE request to remove a note permanently
func deleteNoteCmd(noteID int, token string) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 5 * time.Second}
		url := fmt.Sprintf("%s/notes/%d", baseURL, noteID)
		req, err := http.NewRequest("DELETE", url, nil)
		if err != nil {
			return errMsg(err)
		}

		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			return errMsg(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return errMsg(fmt.Errorf("failed to delete note with status: %d", resp.StatusCode))
		}

		return noteDeletedMsg{}
	}
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
