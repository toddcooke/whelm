package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestTabNavigation tests that it's possible to use Tab to navigate between all input fields
func TestTabNavigation(t *testing.T) {
	// Initialize the model in edit request state
	m := initialModel()
	m.state = stateEditRequest

	// Initially, URL input should be focused
	if !m.urlInput.Focused() {
		t.Errorf("Expected URL input to be focused initially, but it wasn't")
	}

	// Create a Tab key message
	tabMsg := tea.KeyMsg{
		Type: tea.KeyTab,
		Alt:  false,
	}

	// Test Tab from URL to HTTP Method
	updatedModel, _ := m.Update(tabMsg)
	m = updatedModel.(model)

	// After first Tab, URL should be blurred and method list should be "focused"
	if m.urlInput.Focused() || m.headerInput.Focused() || m.bodyInput.Focused() {
		t.Errorf("Expected all inputs to be blurred after Tab from URL, but at least one was focused")
		t.Logf("URL focused: %v, Headers focused: %v, Body focused: %v",
			m.urlInput.Focused(), m.headerInput.Focused(), m.bodyInput.Focused())
	}

	// Test Tab from HTTP Method to Headers
	updatedModel, _ = m.Update(tabMsg)
	m = updatedModel.(model)

	// After second Tab, headers input should be focused
	if !m.headerInput.Focused() {
		t.Errorf("Expected Headers input to be focused after Tab from method list, but it wasn't")
		t.Logf("URL focused: %v, Headers focused: %v, Body focused: %v",
			m.urlInput.Focused(), m.headerInput.Focused(), m.bodyInput.Focused())
	}

	// Test Tab from Headers to Body
	updatedModel, _ = m.Update(tabMsg)
	m = updatedModel.(model)

	// After third Tab, body input should be focused
	if !m.bodyInput.Focused() {
		t.Errorf("Expected Body input to be focused after Tab from headers, but it wasn't")
		t.Logf("URL focused: %v, Headers focused: %v, Body focused: %v",
			m.urlInput.Focused(), m.headerInput.Focused(), m.bodyInput.Focused())
	}

	// Test Tab from Body back to URL
	updatedModel, _ = m.Update(tabMsg)
	m = updatedModel.(model)

	// After fourth Tab, URL input should be focused again, completing the cycle
	if !m.urlInput.Focused() {
		t.Errorf("Expected URL input to be focused after Tab from body, but it wasn't")
		t.Logf("URL focused: %v, Headers focused: %v, Body focused: %v",
			m.urlInput.Focused(), m.headerInput.Focused(), m.bodyInput.Focused())
	}

	t.Log("Successfully navigated through all fields using Tab: URL → HTTP Method → Headers → Body → back to URL")
}

// TestCtrlNNavigation tests that it's possible to use Ctrl+N to navigate between all input fields
func TestCtrlNNavigation(t *testing.T) {
	// Initialize the model in edit request state
	m := initialModel()
	m.state = stateEditRequest

	// Initially, URL input should be focused
	if !m.urlInput.Focused() {
		t.Errorf("Expected URL input to be focused initially, but it wasn't")
	}

	// Create a Ctrl+N key message
	ctrlNMsg := tea.KeyMsg{
		Type:  tea.KeyCtrlN,
		Runes: []rune{'n'},
		Alt:   false,
	}

	// Test Ctrl+N from URL to HTTP Method
	updatedModel, _ := m.Update(ctrlNMsg)
	m = updatedModel.(model)

	// After first Ctrl+N, URL should be blurred and method list should be "focused"
	if m.urlInput.Focused() || m.headerInput.Focused() || m.bodyInput.Focused() {
		t.Errorf("Expected all inputs to be blurred after Ctrl+N from URL, but at least one was focused")
		t.Logf("URL focused: %v, Headers focused: %v, Body focused: %v",
			m.urlInput.Focused(), m.headerInput.Focused(), m.bodyInput.Focused())
	}

	// Test Ctrl+N from HTTP Method to Headers
	updatedModel, _ = m.Update(ctrlNMsg)
	m = updatedModel.(model)

	// After second Ctrl+N, headers input should be focused
	if !m.headerInput.Focused() {
		t.Errorf("Expected Headers input to be focused after Ctrl+N from method list, but it wasn't")
		t.Logf("URL focused: %v, Headers focused: %v, Body focused: %v",
			m.urlInput.Focused(), m.headerInput.Focused(), m.bodyInput.Focused())
	}

	// Test Ctrl+N from Headers to Body
	updatedModel, _ = m.Update(ctrlNMsg)
	m = updatedModel.(model)

	// After third Ctrl+N, body input should be focused
	if !m.bodyInput.Focused() {
		t.Errorf("Expected Body input to be focused after Ctrl+N from headers, but it wasn't")
		t.Logf("URL focused: %v, Headers focused: %v, Body focused: %v",
			m.urlInput.Focused(), m.headerInput.Focused(), m.bodyInput.Focused())
	}

	// Test Ctrl+N from Body back to URL
	updatedModel, _ = m.Update(ctrlNMsg)
	m = updatedModel.(model)

	// After fourth Ctrl+N, URL input should be focused again, completing the cycle
	if !m.urlInput.Focused() {
		t.Errorf("Expected URL input to be focused after Ctrl+N from body, but it wasn't")
		t.Logf("URL focused: %v, Headers focused: %v, Body focused: %v",
			m.urlInput.Focused(), m.headerInput.Focused(), m.bodyInput.Focused())
	}

	t.Log("Successfully navigated through all fields using Ctrl+N: URL → HTTP Method → Headers → Body → back to URL")
}

// TestFullCircleNavigation tests that navigation properly cycles through all fields and returns to the beginning
func TestFullCircleNavigation(t *testing.T) {
	// Initialize the model in edit request state
	m := initialModel()
	m.state = stateEditRequest

	// Create a Tab key message
	tabMsg := tea.KeyMsg{
		Type: tea.KeyTab,
	}

	// Navigate through a full cycle and check if we end up at the same field
	initiallyFocused := m.urlInput.Focused()

	// URL → Method → Headers → Body → URL
	for i := 0; i < 4; i++ {
		updatedModel, _ := m.Update(tabMsg)
		m = updatedModel.(model)
	}

	// After a full cycle, the URL input should be focused again
	if initiallyFocused != m.urlInput.Focused() {
		t.Errorf("Full navigation cycle failed: started with URL focused=%v, ended with URL focused=%v",
			initiallyFocused, m.urlInput.Focused())
		t.Logf("URL focused: %v, Headers focused: %v, Body focused: %v",
			m.urlInput.Focused(), m.headerInput.Focused(), m.bodyInput.Focused())
	}

	t.Log("Successfully completed a full navigation cycle")
}

// TestTextareaDoesNotConsumeTab tests that textareas (body, headers) don't consume tab key events
func TestTextareaDoesNotConsumeTab(t *testing.T) {
	// Initialize the model in edit request state with body input focused
	m := initialModel()
	m.state = stateEditRequest
	m.urlInput.Blur()
	m.bodyInput.Focus()

	// Create a Tab key message
	tabMsg := tea.KeyMsg{
		Type: tea.KeyTab,
	}

	// When tab is pressed with body focused, it should move to URL
	updatedModel, _ := m.Update(tabMsg)
	m = updatedModel.(model)

	// Body should no longer be focused, URL should be focused
	if m.bodyInput.Focused() || !m.urlInput.Focused() {
		t.Errorf("Expected focus to move from body to URL with Tab, but it didn't")
		t.Logf("URL focused: %v, Headers focused: %v, Body focused: %v",
			m.urlInput.Focused(), m.headerInput.Focused(), m.bodyInput.Focused())
	}

	// Now focus headers and try tab again
	m.urlInput.Blur()
	m.headerInput.Focus()

	updatedModel, _ = m.Update(tabMsg)
	m = updatedModel.(model)

	// Headers should no longer be focused, body should be focused
	if m.headerInput.Focused() || !m.bodyInput.Focused() {
		t.Errorf("Expected focus to move from headers to body with Tab, but it didn't")
		t.Logf("URL focused: %v, Headers focused: %v, Body focused: %v",
			m.urlInput.Focused(), m.headerInput.Focused(), m.bodyInput.Focused())
	}

	t.Log("Textareas correctly don't consume tab key events")
}

// TestSendRequestWithEnter tests that pressing Enter sends the request
func TestSendRequestWithEnter(t *testing.T) {
	// Initialize the model in edit request state
	m := initialModel()
	m.state = stateMain
	m.currentRequest.URL = "https://example.com"

	// Create an Enter key message
	enterMsg := tea.KeyMsg{
		Type: tea.KeyEnter,
	}

	// When enter is pressed in main state, it should trigger loading
	updatedModel, _ := m.Update(enterMsg)
	m = updatedModel.(model)

	// Should be in loading state
	if !m.loading {
		t.Errorf("Expected request to be sent (loading=true) after pressing Enter, but loading=%v", m.loading)
	}

	t.Log("Successfully triggered request sending with Enter key")
}

// TestKeepingValuesWhenNavigating tests that values are preserved when navigating between fields
func TestKeepingValuesWhenNavigating(t *testing.T) {
	// Initialize the model in edit request state
	m := initialModel()
	m.state = stateEditRequest

	// Set values
	testURL := "https://test.example.com"
	testMethod := "POST"
	testHeaders := "Content-Type: application/json\nAccept: application/json"
	testBody := `{"test": "value"}`

	// Set URL
	m.urlInput.SetValue(testURL)
	m.currentRequest.URL = testURL

	// Navigate to method
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ := m.Update(tabMsg)
	m = updatedModel.(model)

	// Set method
	m.methodList.Select(indexOf(testMethod, httpMethods))
	m.currentRequest.Method = testMethod

	// Navigate to headers
	updatedModel, _ = m.Update(tabMsg)
	m = updatedModel.(model)

	// Set headers
	m.headerInput.SetValue(testHeaders)
	m.currentRequest.Headers = parseHeaders(testHeaders)

	// Navigate to body
	updatedModel, _ = m.Update(tabMsg)
	m = updatedModel.(model)

	// Set body
	m.bodyInput.SetValue(testBody)
	m.currentRequest.Body = testBody

	// Navigate back to URL
	updatedModel, _ = m.Update(tabMsg)
	m = updatedModel.(model)

	// Check all values are preserved
	if m.currentRequest.URL != testURL {
		t.Errorf("URL was not preserved: expected %s, got %s", testURL, m.currentRequest.URL)
	}
	if m.currentRequest.Method != testMethod {
		t.Errorf("Method was not preserved: expected %s, got %s", testMethod, m.currentRequest.Method)
	}
	if m.headerInput.Value() != testHeaders {
		t.Errorf("Headers were not preserved: expected %s, got %s", testHeaders, m.headerInput.Value())
	}
	if m.bodyInput.Value() != testBody {
		t.Errorf("Body was not preserved: expected %s, got %s", testBody, m.bodyInput.Value())
	}

	t.Log("All field values were preserved during navigation")
}
