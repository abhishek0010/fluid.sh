package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Message types for the TUI

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Name      string
	Args      map[string]interface{}
	Result    map[string]interface{}
	Error     bool
	ErrorMsg  string
	StartTime time.Time
	EndTime   time.Time
}

// AgentResponse represents a response from the agent
type AgentResponse struct {
	Content       string
	ToolResults   []ToolResult
	Done          bool
	AwaitingInput bool
}

// UserInputMsg is sent when the user submits input
type UserInputMsg struct {
	Input string
}

// AgentResponseMsg is sent when the agent responds
type AgentResponseMsg struct {
	Response AgentResponse
}

// AgentErrorMsg is sent when the agent encounters an error
type AgentErrorMsg struct {
	Err error
}

// ThinkingTickMsg is sent for the thinking animation
type ThinkingTickMsg struct{}

// ThinkingCmd returns a command for the thinking animation
func ThinkingCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg {
		return ThinkingTickMsg{}
	})
}

// AgentStatus represents what the agent is currently doing
type AgentStatus int

const (
	StatusThinking AgentStatus = iota // Waiting for LLM response
	StatusWorking                     // Executing a tool
)

// ToolStartMsg is sent when a tool starts executing
type ToolStartMsg struct {
	ToolName string
	Args     map[string]interface{}
}

// ToolCompleteMsg is sent when a tool finishes executing
type ToolCompleteMsg struct {
	ToolName string
	Success  bool
	Result   map[string]interface{}
	Error    string
}

// AgentDoneMsg is sent through the status channel when the agent finishes
// This unblocks the status listener
type AgentDoneMsg struct{}

// ClearThinkingMsg is sent to clear the thinking indicator
type ClearThinkingMsg struct{}

// StartAgentMsg is sent to start the agent processing
type StartAgentMsg struct {
	Input string
}

// WindowSizeMsg wraps the window size message
type WindowSizeMsg struct {
	Width  int
	Height int
}

// QuitMsg is sent when the user wants to quit
type QuitMsg struct{}

// ResetMsg is sent when the user wants to reset the conversation
type ResetMsg struct{}

// FocusInputMsg is sent to focus the input field
type FocusInputMsg struct{}

// ScrollMsg is sent to scroll the conversation view
type ScrollMsg struct {
	Direction int // positive = down, negative = up
}

// SettingsOpenMsg is sent to open the settings modal
type SettingsOpenMsg struct{}

// SettingsCloseMsg is sent when settings are closed
type SettingsCloseMsg struct {
	Saved bool
}

// ReviewRequestMsg is sent when the agent requests human review
type ReviewRequestMsg struct {
	Reason  string
	Summary map[string]interface{}
}

// ReviewResponseMsg is sent when the user responds to a review request
type ReviewResponseMsg struct {
	Approved bool
	Feedback string
}

// TaskCompleteMsg is sent when a task is completed
type TaskCompleteMsg struct {
	Summary string
	Stats   map[string]interface{}
}
