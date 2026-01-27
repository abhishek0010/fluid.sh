package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/aspectrr/fluid.sh/fluid/internal/config"
)

// State represents the current state of the TUI
type State int

const (
	StateIdle State = iota
	StateThinking
	StateAwaitingReview
	StateSettings
)

// ConversationEntry represents a single entry in the conversation
type ConversationEntry struct {
	Role    string // "user", "assistant", "tool", "system"
	Content string
	Tool    *ToolResult
}

// Model is the main Bubble Tea model for the TUI
type Model struct {
	// UI components
	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model
	styles   Styles

	// State
	state        State
	conversation []ConversationEntry
	thinking     bool
	thinkingDots int

	// Agent activity status
	agentStatus     AgentStatus
	currentToolName string
	currentToolArgs map[string]interface{}

	// Status channel for agent updates
	statusChan chan tea.Msg

	// Dimensions
	width  int
	height int
	ready  bool

	// Configuration
	title      string
	provider   string
	model      string
	cfg        *config.Config
	configPath string

	// Settings screen
	settingsModel SettingsModel
	inSettings    bool

	// Agent
	agentRunner AgentRunner

	// Markdown renderer
	mdRenderer *glamour.TermRenderer
}

// AgentRunner is the interface for running agent commands
type AgentRunner interface {
	Run(input string) tea.Cmd
	Reset()
	// SetStatusCallback sets a callback for status updates during execution
	SetStatusCallback(func(tea.Msg))
}

// NewModel creates a new TUI model
func NewModel(title, provider, modelName string, runner AgentRunner, cfg *config.Config, configPath string) Model {
	ta := textarea.New()
	ta.Placeholder = "Type your message... (type /settings to configure)"
	ta.Focus()
	ta.Prompt = ""
	ta.CharLimit = 0
	ta.SetWidth(80)
	ta.SetHeight(1)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6"))

	// Create markdown renderer
	mdRenderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)

	// Build startup message
	startupMsg := "Welcome to fluid TUI! Type 'help' for commands."

	if len(cfg.Hosts) > 0 {
		hostNames := make([]string, len(cfg.Hosts))
		for i, h := range cfg.Hosts {
			hostNames[i] = h.Name
		}
		startupMsg = fmt.Sprintf("Connected with %d remote hosts: %s. Type '/hosts' or '/vms' to query them.",
			len(cfg.Hosts), strings.Join(hostNames, ", "))
	}

	// Create status channel for agent updates
	statusChan := make(chan tea.Msg, 10)

	m := Model{
		textarea:     ta,
		spinner:      s,
		styles:       DefaultStyles(),
		state:        StateIdle,
		conversation: make([]ConversationEntry, 0),
		title:        title,
		provider:     provider,
		model:        modelName,
		cfg:          cfg,
		configPath:   configPath,
		agentRunner:  runner,
		mdRenderer:   mdRenderer,
		statusChan:   statusChan,
	}

	// Set up status callback for the agent
	if runner != nil {
		runner.SetStatusCallback(func(msg tea.Msg) {
			select {
			case statusChan <- msg:
			default:
				// Channel full, drop message
			}
		})
	}

	// Add startup message
	m.conversation = append(m.conversation, ConversationEntry{
		Role:    "system",
		Content: startupMsg,
	})

	return m
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
	)
}

// listenForStatus returns a command that listens for status updates from the agent
func (m Model) listenForStatus() tea.Cmd {
	return func() tea.Msg {
		return <-m.statusChan
	}
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle SettingsCloseMsg first, before delegating to settings
	if closeMsg, ok := msg.(SettingsCloseMsg); ok {
		m.inSettings = false
		m.state = StateIdle
		if closeMsg.Saved {
			m.cfg = m.settingsModel.GetConfig()
			m.addSystemMessage("Settings saved. Some changes may require restart.")
		} else {
			m.addSystemMessage("Settings cancelled.")
		}
		m.updateViewportContent()
		m.textarea.Focus()
		return m, nil
	}

	// If in settings mode, delegate to settings model
	if m.inSettings {
		var cmd tea.Cmd
		settingsModel, cmd := m.settingsModel.Update(msg)
		m.settingsModel = settingsModel.(SettingsModel)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+r":
			m.conversation = make([]ConversationEntry, 0)
			m.addSystemMessage("Conversation reset.")
			if m.agentRunner != nil {
				m.agentRunner.Reset()
			}
			m.updateViewportContent()
			return m, nil
		case "enter":
			if m.state == StateIdle && m.textarea.Value() != "" {
				input := strings.TrimSpace(m.textarea.Value())
				m.textarea.Reset()

				// Handle /settings command
				if input == "/settings" || input == "settings" {
					m.inSettings = true
					m.settingsModel = NewSettingsModel(m.cfg, m.configPath)
					return m, m.settingsModel.Init()
				}

				// Add user message
				m.addUserMessage(input)

				// Start thinking
				m.state = StateThinking
				m.thinking = true
				m.thinkingDots = 0
				m.updateViewportContent()

				// Run agent
				if m.agentRunner != nil {
					return m, tea.Batch(
						m.agentRunner.Run(input),
						ThinkingCmd(),
						m.listenForStatus(),
					)
				}
			}
		case "esc":
			if m.state == StateSettings {
				m.state = StateIdle
				m.textarea.Focus()
			}
		}

	case SettingsCloseMsg:
		m.inSettings = false
		m.state = StateIdle
		if msg.Saved {
			m.cfg = m.settingsModel.GetConfig()
			m.addSystemMessage("Settings saved. Some changes may require restart.")
		} else {
			m.addSystemMessage("Settings cancelled.")
		}
		m.updateViewportContent()
		m.textarea.Focus()
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 1
		inputHeight := 3
		helpHeight := 1
		conversationHeight := m.height - headerHeight - inputHeight - helpHeight - 2

		if !m.ready {
			m.viewport = viewport.New(m.width, conversationHeight)
			m.viewport.YPosition = headerHeight + 1
			m.ready = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = conversationHeight
		}

		m.textarea.SetWidth(m.width - 4)
		m.updateViewportContent()

	case ThinkingTickMsg:
		if m.thinking {
			m.thinkingDots = (m.thinkingDots + 1) % 4
			m.updateViewportContent()
			return m, ThinkingCmd()
		}

	case AgentDoneMsg:
		// Agent finished, don't restart the status listener
		return m, nil

	case ToolStartMsg:
		m.agentStatus = StatusWorking
		m.currentToolName = msg.ToolName
		m.currentToolArgs = msg.Args
		m.updateViewportContent()
		return m, m.listenForStatus()

	case ToolCompleteMsg:
		// Add tool result to conversation
		tr := ToolResult{
			Name:   msg.ToolName,
			Args:   m.currentToolArgs, // Capture args from when tool started
			Result: msg.Result,
			Error:  !msg.Success,
		}
		if msg.Error != "" {
			tr.ErrorMsg = msg.Error
		}
		m.addToolResult(tr)
		// Switch back to thinking while waiting for next LLM response
		m.agentStatus = StatusThinking
		m.currentToolName = ""
		m.currentToolArgs = nil
		m.updateViewportContent()
		return m, m.listenForStatus()

	case AgentResponseMsg:
		m.thinking = false
		m.state = StateIdle
		m.agentStatus = StatusThinking
		m.currentToolName = ""

		// Add assistant message (tool results were already sent via ToolCompleteMsg)
		if msg.Response.Content != "" {
			m.addAssistantMessage(msg.Response.Content)
		}

		// Check for review request or completion
		if msg.Response.AwaitingInput {
			// Handle review - we'd need more context here
			m.state = StateAwaitingReview
		}

		m.updateViewportContent()
		m.textarea.Focus()
		return m, nil

	case AgentErrorMsg:
		m.thinking = false
		m.state = StateIdle
		m.addSystemMessage(fmt.Sprintf("Error: %v", msg.Err))
		m.updateViewportContent()
		m.textarea.Focus()
		return m, nil

	case ReviewResponseMsg:
		m.state = StateIdle
		if msg.Approved {
			m.addSystemMessage("Review approved.")
			if m.agentRunner != nil {
				m.state = StateThinking
				m.thinking = true
				return m, tea.Batch(
					m.agentRunner.Run("Review approved by human. You may proceed."),
					ThinkingCmd(),
				)
			}
		} else {
			m.addSystemMessage("Review rejected. Please provide feedback.")
		}
		m.textarea.Focus()
		m.updateViewportContent()
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update textarea
	var taCmd tea.Cmd
	m.textarea, taCmd = m.textarea.Update(msg)
	cmds = append(cmds, taCmd)

	// Update viewport
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m Model) View() string {
	// Show settings screen if in settings mode
	if m.inSettings {
		return m.settingsModel.View()
	}

	if !m.ready {
		return "Initializing..."
	}

	var b strings.Builder

	// Status bar
	statusBar := m.styles.StatusBar.Width(m.width).Render(
		fmt.Sprintf(" %s - %s: %s", m.title, m.provider, m.model),
	)
	b.WriteString(statusBar)
	b.WriteString("\n")

	// Conversation viewport
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Input area
	inputBox := m.styles.Border.Width(m.width - 2).Render(
		m.styles.InputPrompt.Render("$ ") + m.textarea.View(),
	)
	b.WriteString(inputBox)
	b.WriteString("\n")

	// Help line
	helpStyle := m.styles.Help
	help := helpStyle.Render(
		m.styles.HelpKey.Render("enter") + m.styles.HelpDesc.Render(" send") + "  " +
			m.styles.HelpKey.Render("/settings") + m.styles.HelpDesc.Render(" config") + "  " +
			m.styles.HelpKey.Render("ctrl+r") + m.styles.HelpDesc.Render(" reset") + "  " +
			m.styles.HelpKey.Render("ctrl+c") + m.styles.HelpDesc.Render(" quit"),
	)
	b.WriteString(help)

	return b.String()
}

// Helper methods

func (m *Model) addUserMessage(content string) {
	m.conversation = append(m.conversation, ConversationEntry{
		Role:    "user",
		Content: content,
	})
}

func (m *Model) addAssistantMessage(content string) {
	m.conversation = append(m.conversation, ConversationEntry{
		Role:    "assistant",
		Content: content,
	})
}

func (m *Model) addSystemMessage(content string) {
	m.conversation = append(m.conversation, ConversationEntry{
		Role:    "system",
		Content: content,
	})
}

func (m *Model) addToolResult(tr ToolResult) {
	m.conversation = append(m.conversation, ConversationEntry{
		Role: "tool",
		Tool: &tr,
	})
}

func (m *Model) updateViewportContent() {
	var b strings.Builder

	for _, entry := range m.conversation {
		switch entry.Role {
		case "user":
			b.WriteString(m.styles.UserMessage.Render("$ " + entry.Content))
			b.WriteString("\n")
		case "assistant":
			// Render markdown
			rendered := entry.Content
			if m.mdRenderer != nil {
				if r, err := m.mdRenderer.Render(entry.Content); err == nil {
					rendered = r
				}
			}
			b.WriteString(m.styles.AssistantMessage.Render(rendered))
			b.WriteString("\n")
		case "system":
			b.WriteString(m.styles.Thinking.Render(entry.Content))
			b.WriteString("\n")
		case "tool":
			if entry.Tool != nil {
				b.WriteString(m.renderToolResult(*entry.Tool))
				b.WriteString("\n")
			}
		}
	}

	// Add status indicator if active
	if m.thinking {
		dots := strings.Repeat(".", m.thinkingDots)
		var statusText string
		switch m.agentStatus {
		case StatusWorking:
			if m.currentToolName != "" {
				statusText = fmt.Sprintf(" Working: %s", m.currentToolName)
				// Show relevant context for specific tools
				if m.currentToolArgs != nil {
					switch m.currentToolName {
					case "run_command":
						if cmd, ok := m.currentToolArgs["command"].(string); ok {
							// Truncate long commands
							if len(cmd) > 60 {
								cmd = cmd[:57] + "..."
							}
							statusText = fmt.Sprintf(" Running: %s", cmd)
						}
					case "create_sandbox":
						if src, ok := m.currentToolArgs["source_vm_name"].(string); ok {
							statusText = fmt.Sprintf(" Creating sandbox from: %s", src)
						}
					case "destroy_sandbox":
						if id, ok := m.currentToolArgs["sandbox_id"].(string); ok {
							statusText = fmt.Sprintf(" Destroying: %s", id)
						}
					case "start_sandbox", "stop_sandbox":
						if id, ok := m.currentToolArgs["sandbox_id"].(string); ok {
							action := "Starting"
							if m.currentToolName == "stop_sandbox" {
								action = "Stopping"
							}
							statusText = fmt.Sprintf(" %s: %s", action, id)
						}
					}
				}
			} else {
				statusText = " Working"
			}
		default:
			statusText = " Thinking"
		}
		b.WriteString(m.styles.Thinking.Render(m.spinner.View() + statusText + dots))
		b.WriteString("\n")
	}

	m.viewport.SetContent(b.String())
	m.viewport.GotoBottom()
}

func (m *Model) renderToolResult(tr ToolResult) string {
	var b strings.Builder

	if tr.Error {
		icon := "x"
		b.WriteString(m.styles.ToolError.Render(fmt.Sprintf("  %s %s", icon, tr.Name)))
		b.WriteString("\n")
		if tr.ErrorMsg != "" {
			// Truncate long error messages
			errMsg := tr.ErrorMsg
			if len(errMsg) > 200 {
				errMsg = errMsg[:197] + "..."
			}
			b.WriteString(m.styles.ToolDetailsError.Render(fmt.Sprintf("      Error: %s", errMsg)))
		}
	} else {
		icon := "v"
		b.WriteString(m.styles.ToolSuccess.Render(fmt.Sprintf("  %s %s", icon, tr.Name)))
		b.WriteString("\n")

		// Format result based on tool type
		if tr.Result != nil {
			formatted := m.formatToolOutput(tr.Name, tr.Args, tr.Result)
			b.WriteString(formatted)
		}
	}

	return b.String()
}

// formatToolOutput formats tool results in a human-readable way
func (m *Model) formatToolOutput(toolName string, args, result map[string]interface{}) string {
	var b strings.Builder

	switch toolName {
	case "run_command":
		// Show the command that was run
		if args != nil {
			if cmd, ok := args["command"].(string); ok {
				// Truncate long commands
				if len(cmd) > 80 {
					cmd = cmd[:77] + "..."
				}
				b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      $ %s", cmd)))
				b.WriteString("\n")
			}
		}
		// Show exit code
		if exitCode, ok := result["exit_code"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      exit: %v", exitCode)))
			b.WriteString("\n")
		}
		// Show stdout (truncated)
		if stdout, ok := result["stdout"].(string); ok && stdout != "" {
			stdout = strings.TrimSpace(stdout)
			lines := strings.Split(stdout, "\n")
			if len(lines) > 5 {
				lines = append(lines[:5], fmt.Sprintf("... (%d more lines)", len(lines)-5))
			}
			for _, line := range lines {
				if len(line) > 100 {
					line = line[:97] + "..."
				}
				b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      %s", line)))
				b.WriteString("\n")
			}
		}
		// Show stderr if present
		if stderr, ok := result["stderr"].(string); ok && stderr != "" {
			stderr = strings.TrimSpace(stderr)
			// Skip the common SSH warning
			if !strings.HasPrefix(stderr, "Warning: Permanently added") {
				lines := strings.Split(stderr, "\n")
				if len(lines) > 3 {
					lines = append(lines[:3], "...")
				}
				for _, line := range lines {
					if len(line) > 100 {
						line = line[:97] + "..."
					}
					b.WriteString(m.styles.ToolDetailsError.Render(fmt.Sprintf("      stderr: %s", line)))
					b.WriteString("\n")
				}
			}
		}

	case "list_sandboxes":
		if sandboxes, ok := result["sandboxes"].([]interface{}); ok {
			if len(sandboxes) == 0 {
				b.WriteString(m.styles.ToolDetails.Render("      No sandboxes found"))
				b.WriteString("\n")
			} else {
				b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      Found %d sandbox(es)", len(sandboxes))))
				b.WriteString("\n")
				for i, sb := range sandboxes {
					if i >= 5 {
						b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      ... and %d more", len(sandboxes)-5)))
						b.WriteString("\n")
						break
					}
					if sbMap, ok := sb.(map[string]interface{}); ok {
						id := sbMap["id"]
						name := sbMap["name"]
						state := sbMap["state"]
						ip := sbMap["ip_address"]
						line := fmt.Sprintf("      - %v (%v) %v", name, id, state)
						if ip != nil && ip != "" {
							line += fmt.Sprintf(" @ %v", ip)
						}
						b.WriteString(m.styles.ToolDetails.Render(line))
						b.WriteString("\n")
					}
				}
			}
		}

	case "create_sandbox":
		if id, ok := result["sandbox_id"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      ID: %v", id)))
			b.WriteString("\n")
		}
		if name, ok := result["name"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      Name: %v", name)))
			b.WriteString("\n")
		}
		if ip, ok := result["ip_address"]; ok && ip != nil && ip != "" {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      IP: %v", ip)))
			b.WriteString("\n")
		}
		if state, ok := result["state"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      State: %v", state)))
			b.WriteString("\n")
		}

	case "get_sandbox":
		if id, ok := result["id"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      ID: %v", id)))
			b.WriteString("\n")
		}
		if name, ok := result["name"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      Name: %v", name)))
			b.WriteString("\n")
		}
		if state, ok := result["state"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      State: %v", state)))
			b.WriteString("\n")
		}
		if ip, ok := result["ip_address"]; ok && ip != nil && ip != "" {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      IP: %v", ip)))
			b.WriteString("\n")
		}
		if host, ok := result["host"]; ok && host != nil && host != "" {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      Host: %v", host)))
			b.WriteString("\n")
		}

	case "destroy_sandbox":
		if id, ok := result["sandbox_id"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      Destroyed: %v", id)))
			b.WriteString("\n")
		}

	case "start_sandbox", "stop_sandbox":
		if id, ok := result["sandbox_id"]; ok {
			action := "Started"
			if toolName == "stop_sandbox" {
				action = "Stopped"
			}
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      %s: %v", action, id)))
			b.WriteString("\n")
		}
		if ip, ok := result["ip_address"]; ok && ip != nil && ip != "" {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      IP: %v", ip)))
			b.WriteString("\n")
		}

	case "list_vms":
		if vms, ok := result["vms"].([]interface{}); ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      Found %d VM(s)", len(vms))))
			b.WriteString("\n")
			for i, vm := range vms {
				if i >= 10 {
					b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      ... and %d more", len(vms)-10)))
					b.WriteString("\n")
					break
				}
				if vmMap, ok := vm.(map[string]interface{}); ok {
					name := vmMap["name"]
					state := vmMap["state"]
					host := vmMap["host"]
					line := fmt.Sprintf("      - %v (%v)", name, state)
					if host != nil && host != "" {
						line += fmt.Sprintf(" on %v", host)
					}
					b.WriteString(m.styles.ToolDetails.Render(line))
					b.WriteString("\n")
				}
			}
		}

	case "create_snapshot":
		if id, ok := result["snapshot_id"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      Snapshot: %v", id)))
			b.WriteString("\n")
		}
		if name, ok := result["name"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      Name: %v", name)))
			b.WriteString("\n")
		}

	case "add_playbook_task", "create_playbook", "get_playbook":
		if name, ok := result["name"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      Playbook: %v", name)))
			b.WriteString("\n")
		}
		if id, ok := result["id"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      ID: %v", id)))
			b.WriteString("\n")
		}
		if taskID, ok := result["task_id"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      Task ID: %v", taskID)))
			b.WriteString("\n")
		}

	default:
		// Generic formatting for unknown tools
		content := fmt.Sprintf("%v", result)
		if len(content) > 150 {
			content = content[:147] + "..."
		}
		b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      -> %s", content)))
		b.WriteString("\n")
	}

	return b.String()
}

// Run starts the TUI application
func Run(m Model) error {
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
