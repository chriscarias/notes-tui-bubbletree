# Notes API Client (Bubble Tea TUI)
## Frontend for Polyglot Notes Project

A terminal-native, keyboard-driven text user interface (TUI) built in **Go** using the **Charm Bubble Tea** framework. This client communicates with the same Notes REST API backend as its Tkinter desktop counterpart, allowing you to manage your personal notes database directly from your command-line environment.

Designed around the **Elm Architecture** (Model-View-Update), this client offers highly responsive navigation, minimalist aesthetic layouts via **Lipgloss**, and custom interactive inputs via **Bubbles**.

## 🚀 Features

- **Responsive Terminal UI**: Master/Detail viewport displaying a clean notes list and interactive note viewer/editor.
- **Vim-Style Navigation**: Seamlessly traverse lists and switch focus areas using intuitive Vim keys or arrow keys.
- **Secure Authentication Step**: Safe interactive terminal credential entry for JWT-token generation and storage.
- **Dynamic List Filtering**: Instantly search or filter down cached note indexes.
- **Stateful Completion Toggle**: Visual indicators rendering completed notes cleanly in the list (e.g., custom muted styling or unicode checkmarks).
- **Asynchronous Execution**: Designed around the Bubble Tea command structure to execute HTTP calls concurrently without blocking UI updates.

---

## 🛠️ Tech Stack & Architecture

- **Language**: Go 1.21+
- **Frameworks & Libraries**:
  - `github.com/charmbracelet/bubbletea` — TUI event loop core
  - `github.com/charmbracelet/bubbles` — Common interactive components (inputs, lists, textareas)
  - `github.com/charmbracelet/lipgloss` — Styled text layout and border formatting
  - `github.com/go-resty/resty` (or standard library `net/http`) — Synchronous/Asynchronous API communication

---

## 📂 Project Structure

```text
notes-tui-bubbletea/
├── main.go             # Application Entry Point & Core Loop
├── model/              # Elm Model definition (state)
├── view/               # Terminal renderers and Lipgloss styling sheets
├── update/             # Event switchboard handling keyboard/API messages
├── api/                # Client package wrapping HTTP calls to backend
└── README.md           # Project Documentation
```

## ⚙️ Compilation & Setup
1. Clone the repository and enter the directory:
```Bash
cd notes-tui-bubbletea
```
2. Initialize modules and fetch dependencies:
```Bash
go mod tidy
```
3. Verify Host URL config:Ensure your API target environment matches your backend port (configured in your API layer or environment variables):
```Go
const BaseURL = "http://127.0.0.1:8002"
```
4. Run the TUI Client:
```Bash
go run main.go
```
5. Build the optimized binary:
```Bash
go build -o notes-tui
./notes-tui
```

## ⌨️ Keyboard Mapping Controls
Here is your data converted into a clean Markdown table:

| Key | Action |
| --- | --- |
| `j` / `↓` | Move down the list |
| `k` / `↑` | Move up the list |
| `tab` | Toggle focus between Notes List and the Editor pane |
| `n` | Create a clean new note workspace |
| `s` | Save (Create or Patch update) the selected note |
| `d` | Trigger note deletion (awaits verification key) |
| `c` | Quick toggle completion state on selected note |
| `esc` | Back out of editor mode / clear current action |
| `ctrl+c` | Exit application safely |