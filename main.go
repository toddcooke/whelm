package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	// UI states
	stateMain int = iota
	stateEditRequest
	stateViewResponse
	stateSaveRequest
	stateLoadRequest
)

// HTTP methods
var httpMethods = []string{
	"GET",
	"POST",
	"PUT",
	"DELETE",
	"PATCH",
	"HEAD",
	"OPTIONS",
}

// Styles
var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2).Bold(true)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	helpStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	urlInputStyle     = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("63"))
	headerStyle       = lipgloss.NewStyle().Bold(true).Underline(true).Foreground(lipgloss.Color("99"))
	focusedInputStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("205"))
)

// HTTPRequest represents an HTTP request
type HTTPRequest struct {
	Name    string            `json:"name"`
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

// HTTPResponse represents an HTTP response
type HTTPResponse struct {
	StatusCode int               `json:"status_code"`
	Status     string            `json:"status"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Error      string            `json:"error,omitempty"`
}

// Model represents the application state
type model struct {
	state          int
	width          int
	height         int
	currentRequest HTTPRequest
	response       HTTPResponse
	methodList     list.Model
	urlInput       textinput.Model
	bodyInput      textarea.Model
	headerInput    textarea.Model
	nameInput      textinput.Model
	responseView   viewport.Model
	spinner        spinner.Model
	loading        bool
	savedRequests  []HTTPRequest
	requestList    list.Model
	err            error
}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

// Messages
type responseMsg HTTPResponse
type errMsg struct{ error }
type savedRequestsMsg []HTTPRequest

func (e errMsg) Error() string { return e.error.Error() }

func initialModel() model {
	// Initialize method list
	methodItems := []list.Item{}
	for _, method := range httpMethods {
		methodItems = append(methodItems, item{title: method})
	}
	methodList := list.New(methodItems, list.NewDefaultDelegate(), 0, 0)
	methodList.Title = "HTTP Method"
	methodList.SetShowStatusBar(false)
	methodList.SetFilteringEnabled(false)
	methodList.SetShowHelp(false)

	// Initialize URL input
	urlInput := textinput.New()
	urlInput.Placeholder = "https://example.com/api"
	urlInput.Focus()
	urlInput.Width = 40

	// Initialize body input
	bodyInput := textarea.New()
	bodyInput.Placeholder = "Request body (JSON, form data, etc.)"
	bodyInput.SetHeight(10)

	// Initialize header input
	headerInput := textarea.New()
	headerInput.Placeholder = "Headers (one per line, format: Key: Value)"
	headerInput.SetHeight(5)

	// Initialize name input for saving requests
	nameInput := textinput.New()
	nameInput.Placeholder = "Request name"
	nameInput.Focus()
	nameInput.Width = 40

	// Initialize response view
	responseView := viewport.New(80, 20)
	responseView.SetContent("")

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Initialize request list
	requestList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	requestList.Title = "Saved Requests"
	requestList.SetShowStatusBar(false)
	requestList.SetShowHelp(true)

	return model{
		state: stateMain,
		currentRequest: HTTPRequest{
			Method:  "GET",
			URL:     "",
			Headers: make(map[string]string),
			Body:    "",
		},
		methodList:    methodList,
		urlInput:      urlInput,
		bodyInput:     bodyInput,
		headerInput:   headerInput,
		nameInput:     nameInput,
		responseView:  responseView,
		spinner:       s,
		loading:       false,
		savedRequests: []HTTPRequest{},
		requestList:   requestList,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		loadSavedRequests,
		textinput.Blink,
		textarea.Blink,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle tab navigation specially to ensure it works correctly
		if m.state == stateEditRequest && (msg.String() == "ctrl+n" || msg.Type == tea.KeyCtrlN || msg.String() == "tab" || msg.Type == tea.KeyTab) {
			// If tab is pressed, handle field navigation directly instead of passing to component
			if m.urlInput.Focused() {
				// Navigate from URL to Method List
				m.urlInput.Blur()
				m.methodList.Select(indexOf(m.currentRequest.Method, httpMethods))
				return m, nil
			} else if !m.urlInput.Focused() && !m.headerInput.Focused() && !m.bodyInput.Focused() {
				// Method list is "focused" (no actual focus, but we're on this field)
				m.currentRequest.Method = httpMethods[m.methodList.Index()]
				m.headerInput.Focus()
				return m, textarea.Blink
			} else if m.headerInput.Focused() {
				// Navigate from Headers to Body
				m.headerInput.Blur()
				m.bodyInput.Focus()
				return m, textarea.Blink
			} else if m.bodyInput.Focused() {
				// Navigate from Body back to URL
				m.bodyInput.Blur()
				m.urlInput.Focus()
				return m, textinput.Blink
			}
		}

		switch m.state {
		case stateMain:
			switch msg.String() {
			case "q", "ctrl+c", "esc":
				return m, tea.Quit
			case "e", "n":
				m.state = stateEditRequest
				m.urlInput.Focus()
				return m, nil
			case "l":
				m.state = stateLoadRequest
				return m, nil
			case "enter":
				if m.currentRequest.URL != "" {
					m.loading = true
					return m, tea.Batch(
						m.spinner.Tick,
						sendRequest(m.currentRequest),
					)
				}
			}

		case stateEditRequest:
			switch msg.String() {
			case "esc":
				m.state = stateMain
				return m, nil
			case "enter":
				// Enter key when method list is active (nothing else is focused)
				if !m.urlInput.Focused() && !m.headerInput.Focused() && !m.bodyInput.Focused() {
					m.currentRequest.Method = httpMethods[m.methodList.Index()]
					m.headerInput.Focus()
					return m, nil
				}
			case "s":
				if msg.Alt {
					m.state = stateSaveRequest
					m.nameInput.Focus()
					return m, nil
				}
			case "ctrl+s":
				m.state = stateMain
				m.loading = true
				return m, tea.Batch(
					m.spinner.Tick,
					sendRequest(m.currentRequest),
				)
			}

		case stateViewResponse:
			switch msg.String() {
			case "esc", "q":
				m.state = stateMain
				return m, nil
			case "e":
				m.state = stateEditRequest
				return m, nil
			}

		case stateSaveRequest:
			switch msg.String() {
			case "esc":
				m.state = stateEditRequest
				return m, nil
			case "enter":
				if m.nameInput.Value() != "" {
					m.currentRequest.Name = m.nameInput.Value()
					m.state = stateEditRequest
					m.nameInput.Reset()
					return m, saveRequest(m.currentRequest)
				}
			}

		case stateLoadRequest:
			switch msg.String() {
			case "esc":
				m.state = stateMain
				return m, nil
			case "enter":
				if len(m.savedRequests) > 0 && m.requestList.Index() >= 0 {
					m.currentRequest = m.savedRequests[m.requestList.Index()]
					m.state = stateMain
					return m, nil
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.methodList.SetSize(30, 10)
		m.requestList.SetSize(msg.Width, msg.Height-4)

		m.responseView.Width = msg.Width
		m.responseView.Height = msg.Height - 4

		m.bodyInput.SetWidth(msg.Width - 4)
		m.headerInput.SetWidth(msg.Width - 4)

	case responseMsg:
		m.loading = false
		m.response = HTTPResponse(msg)
		m.state = stateViewResponse

		// Format response content to include request details
		content := fmt.Sprintf("Request:\n%s %s\n\n",
			lipgloss.NewStyle().Bold(true).Render(m.currentRequest.Method),
			m.currentRequest.URL)

		// Add request headers
		if len(m.currentRequest.Headers) > 0 {
			content += "Request Headers:\n"
			for k, v := range m.currentRequest.Headers {
				content += fmt.Sprintf("%s: %s\n", k, v)
			}
			content += "\n"
		}

		// Add request body if present
		if m.currentRequest.Body != "" {
			content += "Request Body:\n"
			content += m.currentRequest.Body
			content += "\n\n"
		}

		// Add response details
		content += fmt.Sprintf("Response Status: %d %s\n\n", m.response.StatusCode, m.response.Status)

		if len(m.response.Headers) > 0 {
			content += "Response Headers:\n"
			for k, v := range m.response.Headers {
				content += fmt.Sprintf("%s: %s\n", k, v)
			}
			content += "\n"
		}

		content += "Response Body:\n" + m.response.Body

		m.responseView.SetContent(content)
		return m, nil

	case savedRequestsMsg:
		m.savedRequests = []HTTPRequest(msg)

		// Update request list items
		items := []list.Item{}
		for _, req := range m.savedRequests {
			items = append(items, item{
				title: req.Name,
				desc:  fmt.Sprintf("%s %s", req.Method, req.URL),
			})
		}
		m.requestList.SetItems(items)

		return m, nil

	case errMsg:
		m.loading = false
		m.err = msg
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	// Handle component updates
	switch m.state {
	case stateEditRequest:
		// Skip component updates for navigation keys to avoid them being consumed by the components
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			// Don't pass Tab or Ctrl+N to components to prevent them from capturing these keys
			if keyMsg.String() == "ctrl+n" || keyMsg.Type == tea.KeyCtrlN || keyMsg.String() == "tab" || keyMsg.Type == tea.KeyTab {
				return m, nil
			}
		}

		// Only update the component that is currently focused
		if m.urlInput.Focused() {
			m.urlInput, cmd = m.urlInput.Update(msg)
			m.currentRequest.URL = m.urlInput.Value()
			cmds = append(cmds, cmd)
		} else if m.bodyInput.Focused() {
			// Handle component update but intercept tab key
			if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.Type == tea.KeyTab {
				// Handled above, don't pass to textarea
				return m, nil
			}
			m.bodyInput, cmd = m.bodyInput.Update(msg)
			m.currentRequest.Body = m.bodyInput.Value()
			cmds = append(cmds, cmd)
		} else if m.headerInput.Focused() {
			// Handle component update but intercept tab key
			if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.Type == tea.KeyTab {
				// Handled above, don't pass to textarea
				return m, nil
			}
			m.headerInput, cmd = m.headerInput.Update(msg)
			// Parse headers
			m.currentRequest.Headers = parseHeaders(m.headerInput.Value())
			cmds = append(cmds, cmd)
		} else {
			// Only update the method list if no other input is focused
			m.methodList, cmd = m.methodList.Update(msg)
			cmds = append(cmds, cmd)
		}

	case stateSaveRequest:
		m.nameInput, cmd = m.nameInput.Update(msg)
		cmds = append(cmds, cmd)

	case stateViewResponse:
		m.responseView, cmd = m.responseView.Update(msg)
		cmds = append(cmds, cmd)

	case stateLoadRequest:
		m.requestList, cmd = m.requestList.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	switch m.state {
	case stateMain:
		if m.loading {
			return fmt.Sprintf("\n  %s Sending request...\n\n", m.spinner.View())
		}

		s := titleStyle.Render("HTTP Client")
		s += "\n\n"

		if m.currentRequest.URL != "" {
			s += fmt.Sprintf("  Current Request: %s %s\n",
				lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Render(m.currentRequest.Method),
				m.currentRequest.URL)
		} else {
			s += "  No request configured\n"
		}

		s += "\n"
		s += helpStyle.Render("  e: Edit request • enter: Send request • l: Load saved • q: Quit\n")

		if m.err != nil {
			s += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(fmt.Sprintf("  Error: %v", m.err))
		}

		return s

	case stateEditRequest:
		s := titleStyle.Render("Edit Request")
		s += "\n\n"

		// URL input
		s += "  URL:\n"
		if m.urlInput.Focused() {
			s += focusedInputStyle.Render(m.urlInput.View()) + "\n\n"
		} else {
			s += urlInputStyle.Render(m.urlInput.View()) + "\n\n"
		}

		// Method selection
		s += "  " + m.methodList.View() + "\n\n"

		// Headers
		s += headerStyle.Render("  Headers:") + "\n"
		if m.headerInput.Focused() {
			s += focusedInputStyle.Render(m.headerInput.View()) + "\n\n"
		} else {
			s += m.headerInput.View() + "\n\n"
		}

		// Body
		s += headerStyle.Render("  Body:") + "\n"
		if m.bodyInput.Focused() {
			s += focusedInputStyle.Render(m.bodyInput.View()) + "\n\n"
		} else {
			s += m.bodyInput.View() + "\n\n"
		}

		s += helpStyle.Render("  ctrl+n/tab: Next field • ctrl+s: Send • alt+s: Save • esc: Back\n")

		return s

	case stateViewResponse:
		s := titleStyle.Render("Response")
		s += "\n\n"
		s += m.responseView.View()
		s += "\n\n"
		s += helpStyle.Render("  q: Back • e: Edit request\n")

		return s

	case stateSaveRequest:
		s := titleStyle.Render("Save Request")
		s += "\n\n"
		s += "  Name:\n"
		if m.nameInput.Focused() {
			s += focusedInputStyle.Render(m.nameInput.View()) + "\n\n"
		} else {
			s += urlInputStyle.Render(m.nameInput.View()) + "\n\n"
		}
		s += helpStyle.Render("  enter: Save • esc: Cancel\n")

		return s

	case stateLoadRequest:
		s := titleStyle.Render("Load Request")
		s += "\n\n"
		s += m.requestList.View()
		s += "\n"
		s += helpStyle.Render("  enter: Select • esc: Cancel\n")

		return s

	default:
		return "Unknown state"
	}
}

func sendRequest(req HTTPRequest) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{
			Timeout: 30 * time.Second,
		}

		var reqBody io.Reader
		if req.Body != "" {
			reqBody = strings.NewReader(req.Body)
		}

		httpReq, err := http.NewRequest(req.Method, req.URL, reqBody)
		if err != nil {
			return errMsg{err}
		}

		// Add headers
		for k, v := range req.Headers {
			httpReq.Header.Add(k, v)
		}

		// Set default content-type if not specified and body exists
		if req.Body != "" && httpReq.Header.Get("Content-Type") == "" {
			httpReq.Header.Set("Content-Type", "application/json")
		}

		resp, err := client.Do(httpReq)
		if err != nil {
			return errMsg{err}
		}
		defer resp.Body.Close()

		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return errMsg{err}
		}

		// Convert response headers
		headers := make(map[string]string)
		for k, v := range resp.Header {
			headers[k] = strings.Join(v, ", ")
		}

		return responseMsg{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Headers:    headers,
			Body:       string(body),
		}
	}
}

func saveRequest(req HTTPRequest) tea.Cmd {
	return func() tea.Msg {
		// Create requests directory if it doesn't exist
		if err := os.MkdirAll("requests", 0755); err != nil {
			return errMsg{err}
		}

		// Save request to file
		filename := filepath.Join("requests", fmt.Sprintf("%s.json", req.Name))
		file, err := os.Create(filename)
		if err != nil {
			return errMsg{err}
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(req); err != nil {
			return errMsg{err}
		}

		return loadSavedRequests()
	}
}

func loadSavedRequests() tea.Msg {
	// Check if requests directory exists
	if _, err := os.Stat("requests"); os.IsNotExist(err) {
		return savedRequestsMsg{}
	}

	// Read all files in requests directory
	files, err := os.ReadDir("requests")
	if err != nil {
		return errMsg{err}
	}

	var requests []HTTPRequest
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			// Read and parse request file
			data, err := os.ReadFile(filepath.Join("requests", file.Name()))
			if err != nil {
				continue
			}

			var req HTTPRequest
			if err := json.Unmarshal(data, &req); err != nil {
				continue
			}

			requests = append(requests, req)
		}
	}

	return savedRequestsMsg(requests)
}

func parseHeaders(input string) map[string]string {
	headers := make(map[string]string)
	lines := strings.Split(input, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			headers[key] = value
		}
	}

	return headers
}

func indexOf(val string, slice []string) int {
	for i, item := range slice {
		if item == val {
			return i
		}
	}
	return 0 // Default to first item if not found
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
