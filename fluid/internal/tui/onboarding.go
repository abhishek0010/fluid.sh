package tui

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aspectrr/fluid.sh/fluid/internal/config"
)

// OnboardingStep represents the current step in onboarding
type OnboardingStep int

const (
	StepWelcome OnboardingStep = iota
	StepInfraChoice
	StepAddHosts // New step for adding remote hosts
	StepConnectionTest
	StepShowResources
	StepAPIKey
	StepSSHCA
	StepHowItWorks
	StepOfferDemo
	StepRunningDemo
	StepWrapUp
	StepComplete
)

// InfraChoice represents the infrastructure choice
type InfraChoice int

const (
	InfraLocal InfraChoice = iota
	InfraRemote
	InfraBoth
)

// OnboardingModel is the Bubble Tea model for onboarding
type OnboardingModel struct {
	step        OnboardingStep
	infraChoice InfraChoice

	// UI components
	spinner   spinner.Model
	textInput textinput.Model
	viewport  viewport.Model
	styles    Styles

	// Dimensions
	width  int
	height int
	ready  bool

	// Configuration
	cfg        *config.Config
	configPath string

	// State
	selectedOption int
	testResults    []ConnectionTestResult
	availableVMs   []VMInfo
	loadingVMs     bool // Whether we're currently loading VMs
	apiKeyInput    string
	apiKeyValid    bool
	sshCAExists    bool
	sshCAGenerated bool
	errorMsg       string

	// Host configuration state
	hostInputs     []textinput.Model // Multiple inputs for host config
	hostInputFocus int               // Which input is focused

	// Demo state
	demoSteps        []DemoStep
	demoIndex        int
	demoConversation []ConversationEntry
	demoThinking     bool
	demoThinkingDots int
	demoCurrentTool  string
	demoCurrentArgs  map[string]interface{}

	// For async operations
	testing bool
}

// ConnectionTestResult represents the result of a connection test
type ConnectionTestResult struct {
	Name    string
	Address string
	Success bool
	Error   string
}

// VMInfo represents available VM information
type VMInfo struct {
	Name  string
	Host  string
	State string
}

// NewOnboardingModel creates a new onboarding model
func NewOnboardingModel(cfg *config.Config, configPath string) OnboardingModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6"))

	ti := textinput.New()
	ti.Placeholder = "sk-or-..."
	ti.CharLimit = 100
	ti.Width = 50

	return OnboardingModel{
		step:       StepWelcome,
		spinner:    s,
		textInput:  ti,
		styles:     DefaultStyles(),
		cfg:        cfg,
		configPath: configPath,
		demoSteps:  GetDemoScript(),
	}
}

// Init implements tea.Model
func (m OnboardingModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tea.EnterAltScreen,
	)
}

// Messages for async operations
type connectionTestDoneMsg struct {
	results []ConnectionTestResult
}

type vmListDoneMsg struct {
	vms []VMInfo
}

type apiKeyTestDoneMsg struct {
	valid bool
	err   string
}

type sshCACheckDoneMsg struct {
	exists bool
}

type sshCAGeneratedMsg struct {
	success bool
	err     string
}

type demoTickMsg struct{}

func demoTickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return demoTickMsg{}
	})
}

// Update implements tea.Model
func (m OnboardingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// For host input step, only handle special keys - let character input pass through
		if m.step == StepAddHosts && len(m.hostInputs) > 0 {
			switch msg.String() {
			case "ctrl+c", "enter", "tab", "shift+tab", "ctrl+n", "esc":
				return m.handleKeyPress(msg)
			default:
				// Let character input (including arrow keys for cursor movement) fall through
			}
		} else {
			return m.handleKeyPress(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.viewport = viewport.New(m.width, m.height-4)
			m.ready = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = m.height - 4
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case connectionTestDoneMsg:
		m.testing = false
		m.testResults = msg.results
		m.step = StepShowResources
		m.loadingVMs = true
		cmds = append(cmds, m.listVMs())

	case vmListDoneMsg:
		m.loadingVMs = false
		m.availableVMs = msg.vms

	case apiKeyTestDoneMsg:
		m.testing = false
		m.apiKeyValid = msg.valid
		if msg.valid {
			m.cfg.AIAgent.APIKey = m.apiKeyInput
			m.step = StepSSHCA
			cmds = append(cmds, m.checkSSHCA())
		} else {
			m.errorMsg = msg.err
		}

	case sshCACheckDoneMsg:
		m.sshCAExists = msg.exists

	case sshCAGeneratedMsg:
		m.testing = false
		if msg.success {
			m.sshCAGenerated = true
			m.sshCAExists = true
		} else {
			m.errorMsg = msg.err
		}

	case demoTickMsg:
		if m.step == StepRunningDemo {
			return m.advanceDemo()
		}

	case ThinkingTickMsg:
		if m.demoThinking {
			m.demoThinkingDots = (m.demoThinkingDots + 1) % 4
			cmds = append(cmds, ThinkingCmd())
		}
	}

	// Update text input if on API key step
	if m.step == StepAPIKey && !m.testing {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update host inputs if on add hosts step
	if m.step == StepAddHosts && len(m.hostInputs) > 0 {
		for i := range m.hostInputs {
			if i == m.hostInputFocus {
				var cmd tea.Cmd
				m.hostInputs[i], cmd = m.hostInputs[i].Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m OnboardingModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "enter":
		return m.handleEnter()

	case "tab", "down", "j":
		if m.step == StepAddHosts && len(m.hostInputs) > 0 {
			// Move to next input field
			m.hostInputs[m.hostInputFocus].Blur()
			m.hostInputFocus = (m.hostInputFocus + 1) % len(m.hostInputs)
			m.hostInputs[m.hostInputFocus].Focus()
			return m, nil
		}
		switch m.step {
		case StepInfraChoice:
			if m.selectedOption < 2 {
				m.selectedOption++
			}
		case StepOfferDemo:
			if m.selectedOption < 1 {
				m.selectedOption++
			}
		}

	case "shift+tab", "up", "k":
		if m.step == StepAddHosts && len(m.hostInputs) > 0 {
			// Move to previous input field
			m.hostInputs[m.hostInputFocus].Blur()
			m.hostInputFocus--
			if m.hostInputFocus < 0 {
				m.hostInputFocus = len(m.hostInputs) - 1
			}
			m.hostInputs[m.hostInputFocus].Focus()
			return m, nil
		}
		if m.step == StepInfraChoice || m.step == StepOfferDemo {
			if m.selectedOption > 0 {
				m.selectedOption--
			}
		}

	case "ctrl+n":
		// Add another host in host configuration step
		if m.step == StepAddHosts {
			m = m.addHostInputGroup()
			return m, nil
		}

	case "esc":
		if m.step == StepRunningDemo {
			// Skip demo
			m.step = StepWrapUp
			m.demoThinking = false
		}
	}

	return m, nil
}

func (m OnboardingModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.step {
	case StepWelcome:
		m.step = StepInfraChoice
		return m, nil

	case StepInfraChoice:
		m.infraChoice = InfraChoice(m.selectedOption)
		// If remote or both, go to add hosts step
		if m.infraChoice == InfraRemote || m.infraChoice == InfraBoth {
			// Check if hosts are already configured
			if len(m.cfg.Hosts) == 0 {
				m.step = StepAddHosts
				m = m.initHostInputs()
				return m, textinput.Blink
			}
		}
		// Otherwise go directly to connection test
		m.step = StepConnectionTest
		m.testing = true
		return m, tea.Batch(m.spinner.Tick, m.testConnections())

	case StepAddHosts:
		// Save the configured hosts
		m.saveHostInputs()
		m.step = StepConnectionTest
		m.testing = true
		return m, tea.Batch(m.spinner.Tick, m.testConnections())

	case StepShowResources:
		m.step = StepAPIKey
		m.textInput.Focus()
		return m, textinput.Blink

	case StepAPIKey:
		if m.textInput.Value() == "" {
			// Skip API key
			m.step = StepSSHCA
			return m, m.checkSSHCA()
		}
		m.apiKeyInput = m.textInput.Value()
		m.testing = true
		m.errorMsg = ""
		return m, m.testAPIKey()

	case StepSSHCA:
		if !m.sshCAExists && !m.sshCAGenerated {
			m.testing = true
			return m, m.generateSSHCA()
		}
		m.step = StepHowItWorks
		return m, nil

	case StepHowItWorks:
		m.step = StepOfferDemo
		m.selectedOption = 0
		return m, nil

	case StepOfferDemo:
		if m.selectedOption == 0 {
			// Run demo
			m.step = StepRunningDemo
			m.demoIndex = 0
			m.demoConversation = []ConversationEntry{}
			m.demoThinking = true
			return m, tea.Batch(demoTickCmd(), ThinkingCmd())
		}
		// Skip demo
		m.step = StepWrapUp
		return m, nil

	case StepRunningDemo:
		// During demo, enter advances if paused
		return m, nil

	case StepWrapUp:
		m.step = StepComplete
		return m, nil

	case StepComplete:
		return m, tea.Quit
	}

	return m, nil
}

// View implements tea.Model
func (m OnboardingModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	var content string

	switch m.step {
	case StepWelcome:
		content = m.viewWelcome()
	case StepInfraChoice:
		content = m.viewInfraChoice()
	case StepAddHosts:
		content = m.viewAddHosts()
	case StepConnectionTest:
		content = m.viewConnectionTest()
	case StepShowResources:
		content = m.viewShowResources()
	case StepAPIKey:
		content = m.viewAPIKey()
	case StepSSHCA:
		content = m.viewSSHCA()
	case StepHowItWorks:
		content = m.viewHowItWorks()
	case StepOfferDemo:
		content = m.viewOfferDemo()
	case StepRunningDemo:
		content = m.viewRunningDemo()
	case StepWrapUp:
		content = m.viewWrapUp()
	case StepComplete:
		return ""
	}

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

func (m OnboardingModel) viewWelcome() string {
	var b strings.Builder

	b.WriteString(FluidLogo())
	b.WriteString("\n\n")

	tagline := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Italic(true).
		Render("Make Infrastructure Safe for AI")

	b.WriteString(tagline)
	b.WriteString("\n\n\n")

	prompt := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Render("Press Enter to begin setup...")

	b.WriteString(prompt)

	return b.String()
}

func (m OnboardingModel) viewInfraChoice() string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#3B82F6")).
		Render("How will you run VMs?")

	b.WriteString(title)
	b.WriteString("\n\n")

	options := []string{
		"Local libvirt (qemu:///system)",
		"Remote hosts (SSH to KVM servers)",
		"Both local and remote",
	}

	for i, opt := range options {
		cursor := "  "
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
		if i == m.selectedOption {
			cursor = "> "
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6")).Bold(true)
		}
		b.WriteString(cursor)
		b.WriteString(style.Render(opt))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("Use arrow keys to select, Enter to confirm")
	b.WriteString(help)

	return b.String()
}

func (m OnboardingModel) viewConnectionTest() string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#3B82F6")).
		Render("Testing Connections...")

	b.WriteString(title)
	b.WriteString("\n\n")

	if m.testing {
		b.WriteString(m.spinner.View())
		b.WriteString(" Checking connectivity...")
	} else {
		for _, result := range m.testResults {
			var icon, style string
			if result.Success {
				icon = "v"
				style = "#10B981"
			} else {
				icon = "x"
				style = "#EF4444"
			}
			line := lipgloss.NewStyle().Foreground(lipgloss.Color(style)).Render(
				fmt.Sprintf("  %s %s (%s)", icon, result.Name, result.Address),
			)
			b.WriteString(line)
			b.WriteString("\n")
			if !result.Success && result.Error != "" {
				errLine := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Render(
					fmt.Sprintf("      Error: %s", result.Error),
				)
				b.WriteString(errLine)
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}

func (m OnboardingModel) viewShowResources() string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#3B82F6")).
		Render("Available VMs")

	b.WriteString(title)
	b.WriteString("\n\n")

	if m.loadingVMs {
		b.WriteString(m.spinner.View())
		b.WriteString(" Loading VMs from hosts...")
	} else if len(m.availableVMs) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Render("  No VMs found. You'll need to create base VMs to clone from."))
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("Press Enter to continue..."))
	} else {
		for _, vm := range m.availableVMs {
			hostInfo := ""
			if vm.Host != "" {
				hostInfo = fmt.Sprintf(" on %s", vm.Host)
			}
			line := fmt.Sprintf("  - %s (%s)%s", vm.Name, vm.State, hostInfo)
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Render(line))
			b.WriteString("\n")
		}
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("Press Enter to continue..."))
	}

	return b.String()
}

func (m OnboardingModel) viewAPIKey() string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#3B82F6")).
		Render("LLM Configuration")

	b.WriteString(title)
	b.WriteString("\n\n")

	desc := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Render(
		"Fluid uses an LLM under the hood to understand your requests and execute tasks.\nEnter your OpenRouter API key (or press Enter to skip):",
	)
	b.WriteString(desc)
	b.WriteString("\n\n")

	if m.testing {
		b.WriteString(m.spinner.View())
		b.WriteString(" Testing API connection...")
	} else {
		b.WriteString(m.textInput.View())

		if m.errorMsg != "" {
			b.WriteString("\n\n")
			errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
			b.WriteString(errStyle.Render(m.errorMsg))
		}

		if m.apiKeyValid {
			b.WriteString("\n\n")
			successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
			b.WriteString(successStyle.Render("v Connected successfully!"))
		}
	}

	return b.String()
}

func (m OnboardingModel) viewSSHCA() string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#3B82F6")).
		Render("SSH Certificate Authority")

	b.WriteString(title)
	b.WriteString("\n\n")

	desc := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Render(
		"SSH CA allows secure, ephemeral access to sandboxes without\nmanaging individual SSH keys.",
	)
	b.WriteString(desc)
	b.WriteString("\n\n")

	if m.testing {
		b.WriteString(m.spinner.View())
		b.WriteString(" Generating SSH CA...")
	} else if m.sshCAExists {
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
		if m.sshCAGenerated {
			b.WriteString(successStyle.Render("v SSH CA generated successfully!"))
		} else {
			b.WriteString(successStyle.Render("v SSH CA already configured"))
		}
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("Press Enter to continue..."))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24")).Render("! SSH CA not found"))
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("Press Enter to generate..."))
	}

	if m.errorMsg != "" {
		b.WriteString("\n\n")
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
		b.WriteString(errStyle.Render(m.errorMsg))
	}

	return b.String()
}

func (m OnboardingModel) viewHowItWorks() string {
	content := `1. Clone    Create a copy of a base VM
2. Isolate  Sandbox runs independently
3. Execute  AI runs commands via SSH
4. Capture  Changes recorded as Ansible playbook
5. Destroy  Sandbox deleted, playbook remains

Safety: Sandboxes are isolated VMs. Changes don't
affect your base images or production systems.

Learn more: https://fluid.sh/blog/how-sandboxes-work`

	box := BoxedText("How Fluid Sandboxes Work", content, 55)

	var b strings.Builder
	b.WriteString(box)
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("Press Enter to continue..."))

	return b.String()
}

func (m OnboardingModel) viewOfferDemo() string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#3B82F6")).
		Render("See Fluid in Action")

	b.WriteString(title)
	b.WriteString("\n\n")

	desc := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Render(
		"Would you like to watch a demo of Fluid setting up an\nApache web server and creating an Ansible playbook?",
	)
	b.WriteString(desc)
	b.WriteString("\n\n")

	options := []string{
		"Yes, run the demo",
		"No, skip to main TUI",
	}

	for i, opt := range options {
		cursor := "  "
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
		if i == m.selectedOption {
			cursor = "> "
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6")).Bold(true)
		}
		b.WriteString(cursor)
		b.WriteString(style.Render(opt))
		b.WriteString("\n")
	}

	return b.String()
}

func (m OnboardingModel) viewRunningDemo() string {
	var b strings.Builder

	// Header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#3B82F6")).
		Render("Demo Mode")
	b.WriteString(header)
	b.WriteString(" ")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("(Press Esc to skip)"))
	b.WriteString("\n\n")

	// Conversation
	for _, entry := range m.demoConversation {
		switch entry.Role {
		case "user":
			b.WriteString(m.styles.UserMessage.Render("$ " + entry.Content))
			b.WriteString("\n\n")
		case "assistant":
			b.WriteString(m.styles.AssistantMessage.Render(entry.Content))
			b.WriteString("\n\n")
		case "tool":
			if entry.Tool != nil {
				b.WriteString(m.renderToolResult(*entry.Tool))
				b.WriteString("\n")
			}
		}
	}

	// Thinking indicator
	if m.demoThinking {
		dots := strings.Repeat(".", m.demoThinkingDots)
		var statusText string
		if m.demoCurrentTool != "" {
			// Show context for run_command
			if m.demoCurrentTool == "run_command" && m.demoCurrentArgs != nil {
				if cmd, ok := m.demoCurrentArgs["command"].(string); ok {
					if len(cmd) > 50 {
						cmd = cmd[:47] + "..."
					}
					statusText = fmt.Sprintf(" Running: %s", cmd)
				} else {
					statusText = fmt.Sprintf(" Working: %s", m.demoCurrentTool)
				}
			} else {
				statusText = fmt.Sprintf(" Working: %s", m.demoCurrentTool)
			}
		} else {
			statusText = " Thinking"
		}
		b.WriteString(m.styles.Thinking.Render(m.spinner.View() + statusText + dots))
		b.WriteString("\n")
	}

	return b.String()
}

func (m OnboardingModel) viewWrapUp() string {
	content := `Things to try:
  - "Create a sandbox and install nginx"
  - "Set up a PostgreSQL database"
  - "Configure a firewall with ufw"
  - "Deploy a Python Flask app"

Need help?
  - GitHub: https://github.com/aspectrr/fluid.sh/issues
  - Discord: https://discord.gg/4WGGXJWm8J`

	box := BoxedText("You're all set!", content, 50)

	var b strings.Builder
	b.WriteString(box)
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("Press Enter to start Fluid..."))

	return b.String()
}

// viewAddHosts renders the host configuration view
func (m OnboardingModel) viewAddHosts() string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#3B82F6")).
		Render("Configure Remote Hosts")

	b.WriteString(title)
	b.WriteString("\n\n")

	desc := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Render(
		"Enter your KVM host details. SSH key authentication should be configured.",
	)
	b.WriteString(desc)
	b.WriteString("\n\n")

	// Render each host's input fields
	numHosts := len(m.hostInputs) / 3 // 3 inputs per host (name, address, user)
	for i := 0; i < numHosts; i++ {
		hostNum := i + 1
		baseIdx := i * 3

		hostHeader := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#60A5FA")).
			Render(fmt.Sprintf("Host %d", hostNum))
		b.WriteString(hostHeader)
		b.WriteString("\n")

		// Name input
		labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Width(12)
		b.WriteString(labelStyle.Render("  Name:"))
		b.WriteString(" ")
		b.WriteString(m.hostInputs[baseIdx].View())
		b.WriteString("\n")

		// Address input
		b.WriteString(labelStyle.Render("  Address:"))
		b.WriteString(" ")
		b.WriteString(m.hostInputs[baseIdx+1].View())
		b.WriteString("\n")

		// SSH User input
		b.WriteString(labelStyle.Render("  SSH User:"))
		b.WriteString(" ")
		b.WriteString(m.hostInputs[baseIdx+2].View())
		b.WriteString("\n\n")
	}

	// Help text
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	b.WriteString(help.Render("Tab/Arrow keys to navigate | Ctrl+N to add another host | Enter to continue"))

	return b.String()
}

// initHostInputs initializes the text inputs for host configuration
func (m OnboardingModel) initHostInputs() OnboardingModel {
	m.hostInputs = make([]textinput.Model, 0)
	m.hostInputFocus = 0
	m = m.addHostInputGroup()
	// Focus the first input
	if len(m.hostInputs) > 0 {
		m.hostInputs[0].Focus()
	}
	return m
}

// addHostInputGroup adds a set of inputs for configuring another host
func (m OnboardingModel) addHostInputGroup() OnboardingModel {
	// Name input
	nameInput := textinput.New()
	nameInput.Placeholder = "kvm-host-1"
	nameInput.CharLimit = 50
	nameInput.Width = 30

	// Address input
	addrInput := textinput.New()
	addrInput.Placeholder = "192.168.1.100 or hostname"
	addrInput.CharLimit = 100
	addrInput.Width = 30

	// SSH User input
	userInput := textinput.New()
	userInput.Placeholder = "root"
	userInput.CharLimit = 50
	userInput.Width = 30

	m.hostInputs = append(m.hostInputs, nameInput, addrInput, userInput)
	return m
}

// saveHostInputs saves the configured hosts to the config
func (m *OnboardingModel) saveHostInputs() {
	numHosts := len(m.hostInputs) / 3
	m.cfg.Hosts = make([]config.HostConfig, 0)

	for i := 0; i < numHosts; i++ {
		baseIdx := i * 3
		name := strings.TrimSpace(m.hostInputs[baseIdx].Value())
		addr := strings.TrimSpace(m.hostInputs[baseIdx+1].Value())
		user := strings.TrimSpace(m.hostInputs[baseIdx+2].Value())

		// Skip empty hosts
		if addr == "" {
			continue
		}

		// Use defaults
		if name == "" {
			name = fmt.Sprintf("host-%d", i+1)
		}
		if user == "" {
			user = "root"
		}

		m.cfg.Hosts = append(m.cfg.Hosts, config.HostConfig{
			Name:    name,
			Address: addr,
			SSHUser: user,
			SSHPort: 22,
		})
	}
}

// Async operation commands

func (m OnboardingModel) testConnections() tea.Cmd {
	return func() tea.Msg {
		var results []ConnectionTestResult

		switch m.infraChoice {
		case InfraLocal, InfraBoth:
			// Test local libvirt
			result := ConnectionTestResult{
				Name:    "Local libvirt",
				Address: "qemu:///system",
			}
			cmd := exec.Command("virsh", "-c", "qemu:///system", "list")
			if err := cmd.Run(); err != nil {
				result.Success = false
				result.Error = "Cannot connect to libvirt. Is libvirtd running?"
			} else {
				result.Success = true
			}
			results = append(results, result)
		}

		if m.infraChoice == InfraRemote || m.infraChoice == InfraBoth {
			// Test remote hosts from config
			for _, host := range m.cfg.Hosts {
				result := ConnectionTestResult{
					Name:    host.Name,
					Address: host.Address,
				}
				uri := fmt.Sprintf("qemu+ssh://%s@%s/system", host.SSHUser, host.Address)
				if host.SSHUser == "" {
					uri = fmt.Sprintf("qemu+ssh://root@%s/system", host.Address)
				}
				cmd := exec.Command("virsh", "-c", uri, "list")
				if err := cmd.Run(); err != nil {
					result.Success = false
					result.Error = "SSH connection failed"
				} else {
					result.Success = true
				}
				results = append(results, result)
			}

			// If no hosts configured, show a message
			if len(m.cfg.Hosts) == 0 {
				results = append(results, ConnectionTestResult{
					Name:    "Remote hosts",
					Address: "none configured",
					Success: false,
					Error:   "Add hosts to ~/.fluid/config.yaml",
				})
			}
		}

		return connectionTestDoneMsg{results: results}
	}
}

func (m OnboardingModel) listVMs() tea.Cmd {
	return func() tea.Msg {
		var vms []VMInfo

		// Try local
		if m.infraChoice == InfraLocal || m.infraChoice == InfraBoth {
			cmd := exec.Command("virsh", "-c", "qemu:///system", "list", "--all", "--name")
			output, err := cmd.Output()
			if err == nil {
				for _, name := range strings.Split(string(output), "\n") {
					name = strings.TrimSpace(name)
					if name != "" {
						vms = append(vms, VMInfo{
							Name:  name,
							Host:  "local",
							State: "available",
						})
					}
				}
			}
		}

		// Try remote hosts
		if m.infraChoice == InfraRemote || m.infraChoice == InfraBoth {
			for _, host := range m.cfg.Hosts {
				uri := fmt.Sprintf("qemu+ssh://%s@%s/system", host.SSHUser, host.Address)
				if host.SSHUser == "" {
					uri = fmt.Sprintf("qemu+ssh://root@%s/system", host.Address)
				}
				cmd := exec.Command("virsh", "-c", uri, "list", "--all", "--name")
				output, err := cmd.Output()
				if err == nil {
					for _, name := range strings.Split(string(output), "\n") {
						name = strings.TrimSpace(name)
						if name != "" {
							vms = append(vms, VMInfo{
								Name:  name,
								Host:  host.Name,
								State: "available",
							})
						}
					}
				}
			}
		}

		return vmListDoneMsg{vms: vms}
	}
}

func (m OnboardingModel) testAPIKey() tea.Cmd {
	return func() tea.Msg {
		// For now, just validate the format
		key := m.apiKeyInput
		if !strings.HasPrefix(key, "sk-or-") && !strings.HasPrefix(key, "sk-") {
			return apiKeyTestDoneMsg{valid: false, err: "Invalid API key format"}
		}
		// TODO: Actually test the API connection
		return apiKeyTestDoneMsg{valid: true}
	}
}

func (m OnboardingModel) checkSSHCA() tea.Cmd {
	return func() tea.Msg {
		// Check if SSH CA files exist
		keyPath := m.cfg.SSH.CAKeyPath
		pubPath := m.cfg.SSH.CAPubPath

		if keyPath == "" || pubPath == "" {
			return sshCACheckDoneMsg{exists: false}
		}

		// Check both files exist
		cmd := exec.Command("test", "-f", keyPath)
		keyExists := cmd.Run() == nil

		cmd = exec.Command("test", "-f", pubPath)
		pubExists := cmd.Run() == nil

		return sshCACheckDoneMsg{exists: keyExists && pubExists}
	}
}

func (m OnboardingModel) generateSSHCA() tea.Cmd {
	return func() tea.Msg {
		// Use ssh-keygen to generate CA
		keyPath := m.cfg.SSH.CAKeyPath
		if keyPath == "" {
			return sshCAGeneratedMsg{success: false, err: "CA key path not configured"}
		}

		// Create directory if needed
		cmd := exec.Command("mkdir", "-p", strings.TrimSuffix(keyPath, "/ssh-ca"))
		_ = cmd.Run()

		// Generate CA key
		cmd = exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-N", "", "-C", "fluid-ssh-ca")
		if err := cmd.Run(); err != nil {
			return sshCAGeneratedMsg{success: false, err: fmt.Sprintf("Failed to generate SSH CA: %v", err)}
		}

		return sshCAGeneratedMsg{success: true}
	}
}

func (m OnboardingModel) advanceDemo() (tea.Model, tea.Cmd) {
	if m.demoIndex >= len(m.demoSteps) {
		// Demo complete
		m.step = StepWrapUp
		m.demoThinking = false
		return m, nil
	}

	step := m.demoSteps[m.demoIndex]

	// Check if we should advance based on delay
	// For simplicity, we'll advance one step per tick for now
	// In a real implementation, you'd track elapsed time

	switch step.Type {
	case DemoStepUserMessage:
		m.demoConversation = append(m.demoConversation, ConversationEntry{
			Role:    "user",
			Content: step.Content,
		})
		m.demoIndex++

	case DemoStepAssistantMessage:
		m.demoThinking = false
		m.demoCurrentTool = ""
		m.demoCurrentArgs = nil
		m.demoConversation = append(m.demoConversation, ConversationEntry{
			Role:    "assistant",
			Content: step.Content,
		})
		m.demoIndex++

	case DemoStepToolStart:
		m.demoThinking = true
		m.demoCurrentTool = step.ToolName
		m.demoCurrentArgs = step.ToolArgs
		m.demoIndex++

	case DemoStepToolComplete:
		m.demoThinking = false
		m.demoCurrentTool = ""
		m.demoCurrentArgs = nil
		tr := ToolResult{
			Name:   step.ToolName,
			Args:   step.ToolArgs,
			Result: step.ToolResult,
			Error:  step.ToolError,
		}
		m.demoConversation = append(m.demoConversation, ConversationEntry{
			Role: "tool",
			Tool: &tr,
		})
		m.demoIndex++
	}

	// Schedule next tick with appropriate delay
	delay := step.Delay
	if delay == 0 {
		delay = 500 * time.Millisecond
	}

	return m, tea.Tick(delay, func(t time.Time) tea.Msg {
		return demoTickMsg{}
	})
}

// IsComplete returns true if onboarding is finished
func (m OnboardingModel) IsComplete() bool {
	return m.step == StepComplete
}

// renderToolResult renders a tool result for the demo (simplified version)
func (m OnboardingModel) renderToolResult(tr ToolResult) string {
	var b strings.Builder

	if tr.Error {
		icon := "x"
		b.WriteString(m.styles.ToolError.Render(fmt.Sprintf("  %s %s", icon, tr.Name)))
		b.WriteString("\n")
		if tr.ErrorMsg != "" {
			b.WriteString(m.styles.ToolDetailsError.Render(fmt.Sprintf("      Error: %s", tr.ErrorMsg)))
		}
	} else {
		icon := "v"
		b.WriteString(m.styles.ToolSuccess.Render(fmt.Sprintf("  %s %s", icon, tr.Name)))
		b.WriteString("\n")

		// Format based on tool type
		if tr.Result != nil {
			switch tr.Name {
			case "run_command":
				if tr.Args != nil {
					if cmd, ok := tr.Args["command"].(string); ok {
						if len(cmd) > 70 {
							cmd = cmd[:67] + "..."
						}
						b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      $ %s", cmd)))
						b.WriteString("\n")
					}
				}
				if exitCode, ok := tr.Result["exit_code"]; ok {
					b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      exit: %v", exitCode)))
					b.WriteString("\n")
				}
				if stdout, ok := tr.Result["stdout"].(string); ok && stdout != "" {
					lines := strings.Split(strings.TrimSpace(stdout), "\n")
					if len(lines) > 3 {
						lines = append(lines[:3], fmt.Sprintf("... (%d more lines)", len(lines)-3))
					}
					for _, line := range lines {
						if len(line) > 80 {
							line = line[:77] + "..."
						}
						b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      %s", line)))
						b.WriteString("\n")
					}
				}

			case "create_sandbox":
				if id, ok := tr.Result["sandbox_id"]; ok {
					b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      ID: %v", id)))
					b.WriteString("\n")
				}
				if ip, ok := tr.Result["ip_address"]; ok && ip != nil {
					b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      IP: %v", ip)))
					b.WriteString("\n")
				}
				if state, ok := tr.Result["state"]; ok {
					b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      State: %v", state)))
					b.WriteString("\n")
				}

			case "create_playbook", "add_playbook_task":
				if name, ok := tr.Result["name"]; ok {
					b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      Playbook: %v", name)))
					b.WriteString("\n")
				}
				if taskID, ok := tr.Result["task_id"]; ok {
					b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      Task: %v", taskID)))
					b.WriteString("\n")
				}

			default:
				content := fmt.Sprintf("%v", tr.Result)
				if len(content) > 100 {
					content = content[:97] + "..."
				}
				b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      -> %s", content)))
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}

// GetConfig returns the potentially modified config
func (m OnboardingModel) GetConfig() *config.Config {
	return m.cfg
}

// RunOnboarding runs the onboarding wizard and returns the updated config
func RunOnboarding(cfg *config.Config, configPath string) (*config.Config, error) {
	m := NewOnboardingModel(cfg, configPath)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return cfg, err
	}

	if om, ok := finalModel.(OnboardingModel); ok {
		return om.GetConfig(), nil
	}

	return cfg, nil
}
