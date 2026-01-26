package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"fluid/internal/ansible"
	"fluid/internal/config"
	"fluid/internal/libvirt"
	"fluid/internal/llm"
	"fluid/internal/store"
	"fluid/internal/vm"
)

// FluidAgent implements AgentRunner for the fluid CLI
type FluidAgent struct {
	cfg             *config.Config
	store           store.Store
	vmService       *vm.Service
	manager         libvirt.Manager
	llmClient       llm.Client
	playbookService *ansible.PlaybookService

	// Multi-host support
	multiHostMgr *libvirt.MultiHostDomainManager

	// Status callback for sending updates to TUI
	statusCallback func(tea.Msg)

	// Conversation history for context
	history []llm.Message
}

// NewFluidAgent creates a new fluid agent
func NewFluidAgent(cfg *config.Config, store store.Store, vmService *vm.Service, manager libvirt.Manager) *FluidAgent {
	var llmClient llm.Client
	if cfg.AIAgent.Provider == "openrouter" {
		llmClient = llm.NewOpenRouterClient(cfg.AIAgent)
	}

	agent := &FluidAgent{
		cfg:             cfg,
		store:           store,
		vmService:       vmService,
		manager:         manager,
		llmClient:       llmClient,
		playbookService: ansible.NewPlaybookService(store, cfg.Ansible.PlaybooksDir),
		history:         make([]llm.Message, 0),
	}

	// Initialize multi-host manager if hosts are configured
	if len(cfg.Hosts) > 0 {
		// Use a silent logger for multi-host manager to avoid TUI corruption
		silentLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
		agent.multiHostMgr = libvirt.NewMultiHostDomainManager(cfg.Hosts, silentLogger)
	}

	return agent
}

// SetStatusCallback sets the callback function for status updates
func (a *FluidAgent) SetStatusCallback(callback func(tea.Msg)) {
	a.statusCallback = callback
}

// sendStatus sends a status message through the callback if set
func (a *FluidAgent) sendStatus(msg tea.Msg) {
	if a.statusCallback != nil {
		a.statusCallback(msg)
	}
}

// Run executes a command and returns the result
func (a *FluidAgent) Run(input string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Handle slash commands
		if strings.HasPrefix(input, "/") {
			a.sendStatus(AgentDoneMsg{})
			switch input {
			case "/vms":
				result, err := a.listVMs(ctx)
				return AgentResponseMsg{Response: AgentResponse{
					Content: a.formatVMsResult(result, err),
				}}
			case "/hosts":
				return AgentResponseMsg{Response: AgentResponse{
					Content: a.getHostsInfo(),
				}}
			default:
				return AgentResponseMsg{Response: AgentResponse{
					Content: fmt.Sprintf("Unknown command: %s. Available: /vms, /hosts, /settings", input),
				}}
			}
		}

		// Add user message to history
		a.history = append(a.history, llm.Message{Role: llm.RoleUser, Content: input})

		// LLM client is required
		if a.llmClient == nil || a.cfg.AIAgent.APIKey == "" {
			a.sendStatus(AgentDoneMsg{})
			return AgentErrorMsg{Err: fmt.Errorf("LLM provider not configured. Please set OPENROUTER_API_KEY environment variable or configure it in config.yaml")}
		}

		// LLM-driven execution loop
		for {
			req := llm.ChatRequest{
				Messages: append([]llm.Message{{
					Role:    llm.RoleSystem,
					Content: a.cfg.AIAgent.DefaultSystem,
				}}, a.history...),
				Tools: llm.GetTools(),
			}

			resp, err := a.llmClient.Chat(ctx, req)
			if err != nil {
				a.sendStatus(AgentDoneMsg{})
				return AgentErrorMsg{Err: fmt.Errorf("llm chat: %w", err)}
			}

			if len(resp.Choices) == 0 {
				a.sendStatus(AgentDoneMsg{})
				return AgentErrorMsg{Err: fmt.Errorf("llm returned no choices")}
			}

			msg := resp.Choices[0].Message
			a.history = append(a.history, msg)

			if len(msg.ToolCalls) > 0 {
				// Handle tool calls
				for _, tc := range msg.ToolCalls {
					result, err := a.executeTool(ctx, tc)

					var toolResultContent string
					var resultMap map[string]interface{}
					success := true
					errMsg := ""

					if err != nil {
						success = false
						errMsg = err.Error()
						toolResultContent = fmt.Sprintf("Error: %v", err)
					} else {
						if m, ok := result.(map[string]interface{}); ok {
							resultMap = m
						}
						jsonResult, _ := json.Marshal(result)
						toolResultContent = string(jsonResult)
					}

					// Send tool completion status to TUI
					a.sendStatus(ToolCompleteMsg{
						ToolName: tc.Function.Name,
						Success:  success,
						Result:   resultMap,
						Error:    errMsg,
					})

					a.history = append(a.history, llm.Message{
						Role:       llm.RoleTool,
						Content:    toolResultContent,
						ToolCallID: tc.ID,
						Name:       tc.Function.Name,
					})
				}
				// Continue loop to let LLM process tool results
				continue
			}

			// No more tool calls, return final response
			// Tool results were already sent via ToolCompleteMsg
			// Send done message to unblock status listener
			a.sendStatus(AgentDoneMsg{})
			return AgentResponseMsg{Response: AgentResponse{
				Content: msg.Content,
			}}
		}
	}
}

// executeTool dispatches tool calls to internal methods
func (a *FluidAgent) executeTool(ctx context.Context, tc llm.ToolCall) (interface{}, error) {
	// Parse args for status message
	var args map[string]interface{}
	_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)

	// Send tool start status
	a.sendStatus(ToolStartMsg{
		ToolName: tc.Function.Name,
		Args:     args,
	})

	switch tc.Function.Name {
	case "list_sandboxes":
		return a.listSandboxes(ctx)
	case "create_sandbox":
		var args struct {
			SourceVM string `json:"source_vm"`
			Name     string `json:"name"`
			Host     string `json:"host"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		createArgs := []string{args.SourceVM}
		if args.Name != "" {
			createArgs = append(createArgs, "--name="+args.Name)
		}
		if args.Host != "" {
			createArgs = append(createArgs, "--host="+args.Host)
		}
		return a.createSandbox(ctx, createArgs)
	case "destroy_sandbox":
		var args struct {
			SandboxID string `json:"sandbox_id"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.destroySandbox(ctx, args.SandboxID)
	case "run_command":
		var args struct {
			SandboxID string `json:"sandbox_id"`
			Command   string `json:"command"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.runCommand(ctx, args.SandboxID, args.Command)
	case "start_sandbox":
		var args struct {
			SandboxID string `json:"sandbox_id"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.startSandbox(ctx, args.SandboxID)
	case "stop_sandbox":
		var args struct {
			SandboxID string `json:"sandbox_id"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.stopSandbox(ctx, args.SandboxID)
	case "get_sandbox":
		var args struct {
			SandboxID string `json:"sandbox_id"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.getSandbox(ctx, args.SandboxID)
	case "list_vms":
		return a.listVMs(ctx)
	case "create_snapshot":
		var args struct {
			SandboxID string `json:"sandbox_id"`
			Name      string `json:"name"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.createSnapshot(ctx, args.SandboxID, args.Name)
	case "create_playbook":
		var args ansible.CreatePlaybookRequest
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.playbookService.CreatePlaybook(ctx, args)
	case "add_playbook_task":
		var args struct {
			PlaybookID string                 `json:"playbook_id"`
			Name       string                 `json:"name"`
			Module     string                 `json:"module"`
			Params     map[string]interface{} `json:"params"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.playbookService.AddTask(ctx, args.PlaybookID, ansible.AddTaskRequest{
			Name:   args.Name,
			Module: args.Module,
			Params: args.Params,
		})
	default:
		return nil, fmt.Errorf("unknown tool: %s", tc.Function.Name)
	}
}

// Reset clears the conversation history
func (a *FluidAgent) Reset() {
	a.history = make([]llm.Message, 0)
}

// Command implementations

func (a *FluidAgent) listSandboxes(ctx context.Context) (map[string]interface{}, error) {
	sandboxes, err := a.vmService.GetSandboxes(ctx, store.SandboxFilter{}, nil)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0, len(sandboxes))
	for _, sb := range sandboxes {
		item := map[string]interface{}{
			"id":         sb.ID,
			"name":       sb.SandboxName,
			"state":      sb.State,
			"base_image": sb.BaseImage,
			"created_at": sb.CreatedAt.Format(time.RFC3339),
		}
		if sb.IPAddress != nil {
			item["ip"] = *sb.IPAddress
		}
		if sb.HostName != nil {
			item["host"] = *sb.HostName
		}
		if sb.HostAddress != nil {
			item["host_address"] = *sb.HostAddress
		}
		result = append(result, item)
	}

	return map[string]interface{}{
		"sandboxes": result,
		"count":     len(result),
	}, nil
}

func (a *FluidAgent) createSandbox(ctx context.Context, args []string) (map[string]interface{}, error) {
	sourceVM := ""
	name := ""
	hostName := "" // Optional: specify target host

	// Parse args
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--source-vm=") {
			sourceVM = strings.TrimPrefix(args[i], "--source-vm=")
		} else if strings.HasPrefix(args[i], "--name=") {
			name = strings.TrimPrefix(args[i], "--name=")
		} else if strings.HasPrefix(args[i], "--host=") {
			hostName = strings.TrimPrefix(args[i], "--host=")
		} else if i == 0 && !strings.HasPrefix(args[i], "-") {
			sourceVM = args[i]
		}
	}

	if sourceVM == "" {
		return nil, fmt.Errorf("source-vm is required (e.g., create ubuntu-base)")
	}

	// If multihost is configured, find the host that has the source VM
	if a.multiHostMgr != nil {
		host, err := a.findHostForSourceVM(ctx, sourceVM, hostName)
		if err != nil {
			return nil, fmt.Errorf("find host for source VM: %w", err)
		}
		if host != nil {
			// Create on remote host
			sb, ip, err := a.vmService.CreateSandboxOnHost(ctx, host, sourceVM, "tui-agent", name, 0, 0, nil, true, true)
			if err != nil {
				return nil, err
			}

			result := map[string]interface{}{
				"sandbox_id": sb.ID,
				"name":       sb.SandboxName,
				"state":      sb.State,
				"host":       host.Name,
			}
			if ip != "" {
				result["ip"] = ip
			}
			return result, nil
		}
	}

	// Fall back to local creation
	sb, ip, err := a.vmService.CreateSandbox(ctx, sourceVM, "tui-agent", name, 0, 0, nil, true, true)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"sandbox_id": sb.ID,
		"name":       sb.SandboxName,
		"state":      sb.State,
	}
	if ip != "" {
		result["ip"] = ip
	}

	return result, nil
}

// findHostForSourceVM finds the host that has the given source VM.
// If hostName is specified, only that host is checked.
// Returns nil if no remote host has the VM (fallback to local).
func (a *FluidAgent) findHostForSourceVM(ctx context.Context, sourceVM, hostName string) (*config.HostConfig, error) {
	if a.multiHostMgr == nil {
		return nil, nil
	}

	// If specific host requested, check only that host
	if hostName != "" {
		hosts := a.multiHostMgr.GetHosts()
		for i := range hosts {
			if hosts[i].Name == hostName {
				return &hosts[i], nil
			}
		}
		return nil, fmt.Errorf("host %q not found in configuration", hostName)
	}

	// Search all hosts for the source VM
	host, err := a.multiHostMgr.FindHostForVM(ctx, sourceVM)
	if err != nil {
		// Not found on any remote host - will try local
		return nil, nil
	}

	return host, nil
}

func (a *FluidAgent) destroySandbox(ctx context.Context, id string) (map[string]interface{}, error) {
	_, err := a.vmService.DestroySandbox(ctx, id)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"destroyed":  true,
		"sandbox_id": id,
	}, nil
}

func (a *FluidAgent) runCommand(ctx context.Context, sandboxID, command string) (map[string]interface{}, error) {
	user := a.cfg.SSH.DefaultUser
	result, err := a.vmService.RunCommand(ctx, sandboxID, user, "", command, 0, nil)
	if err != nil {
		// Return partial result if available
		if result != nil {
			return map[string]interface{}{
				"sandbox_id": sandboxID,
				"exit_code":  result.ExitCode,
				"stdout":     result.Stdout,
				"stderr":     result.Stderr,
				"error":      err.Error(),
			}, nil
		}
		return nil, err
	}

	return map[string]interface{}{
		"sandbox_id": sandboxID,
		"exit_code":  result.ExitCode,
		"stdout":     result.Stdout,
		"stderr":     result.Stderr,
	}, nil
}

func (a *FluidAgent) startSandbox(ctx context.Context, id string) (map[string]interface{}, error) {
	ip, err := a.vmService.StartSandbox(ctx, id, true)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"started":    true,
		"sandbox_id": id,
	}
	if ip != "" {
		result["ip"] = ip
	}

	return result, nil
}

func (a *FluidAgent) stopSandbox(ctx context.Context, id string) (map[string]interface{}, error) {
	err := a.vmService.StopSandbox(ctx, id, false)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"stopped":    true,
		"sandbox_id": id,
	}, nil
}

func (a *FluidAgent) getSandbox(ctx context.Context, id string) (map[string]interface{}, error) {
	sb, err := a.vmService.GetSandbox(ctx, id)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"sandbox_id": sb.ID,
		"name":       sb.SandboxName,
		"state":      sb.State,
		"base_image": sb.BaseImage,
		"network":    sb.Network,
		"agent_id":   sb.AgentID,
		"created_at": sb.CreatedAt.Format(time.RFC3339),
		"updated_at": sb.UpdatedAt.Format(time.RFC3339),
	}
	if sb.IPAddress != nil {
		result["ip"] = *sb.IPAddress
	}
	if sb.HostName != nil {
		result["host"] = *sb.HostName
	}
	if sb.HostAddress != nil {
		result["host_address"] = *sb.HostAddress
	}

	return result, nil
}

func (a *FluidAgent) listVMs(ctx context.Context) (map[string]interface{}, error) {
	// If multihost manager is configured, query remote hosts
	if a.multiHostMgr != nil {
		return a.listVMsFromHosts(ctx)
	}

	// Fall back to local virsh
	return a.listVMsLocal(ctx)
}

// listVMsFromHosts queries all configured remote hosts for VMs
func (a *FluidAgent) listVMsFromHosts(ctx context.Context) (map[string]interface{}, error) {
	listResult, err := a.multiHostMgr.ListDomains(ctx)
	if err != nil {
		return nil, fmt.Errorf("list domains from hosts: %w", err)
	}

	result := make([]map[string]interface{}, 0)
	for _, domain := range listResult.Domains {
		item := map[string]interface{}{
			"name":         domain.Name,
			"state":        domain.State.String(),
			"host":         domain.HostName,
			"host_address": domain.HostAddress,
		}
		if domain.UUID != "" {
			item["uuid"] = domain.UUID
		}
		result = append(result, item)
	}

	// Include any host errors in the response
	response := map[string]interface{}{
		"vms":   result,
		"count": len(result),
	}

	if len(listResult.HostErrors) > 0 {
		errors := make([]map[string]interface{}, 0, len(listResult.HostErrors))
		for _, he := range listResult.HostErrors {
			errors = append(errors, map[string]interface{}{
				"host":    he.HostName,
				"address": he.HostAddress,
				"error":   he.Error,
			})
		}
		response["host_errors"] = errors
	}

	return response, nil
}

// listVMsLocal queries local virsh for VMs
func (a *FluidAgent) listVMsLocal(ctx context.Context) (map[string]interface{}, error) {
	// Use virsh list --all --name to get all VMs
	cmd := exec.CommandContext(ctx, "virsh", "list", "--all", "--name")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("virsh list: %w: %s", err, stderr.String())
	}

	result := make([]map[string]interface{}, 0)
	for _, name := range strings.Split(stdout.String(), "\n") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		result = append(result, map[string]interface{}{
			"name":  name,
			"state": "unknown",
			"host":  "local",
		})
	}

	return map[string]interface{}{
		"vms":   result,
		"count": len(result),
	}, nil
}

func (a *FluidAgent) createSnapshot(ctx context.Context, sandboxID, name string) (map[string]interface{}, error) {
	if name == "" {
		name = fmt.Sprintf("snap-%d", time.Now().Unix())
	}

	snap, err := a.vmService.CreateSnapshot(ctx, sandboxID, name, false)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"snapshot_id": snap.ID,
		"sandbox_id":  sandboxID,
		"name":        snap.Name,
		"kind":        snap.Kind,
	}, nil
}

// Formatting helpers

func (a *FluidAgent) formatListResult(result map[string]interface{}) string {
	if result == nil {
		return "No sandboxes found."
	}

	sandboxes, ok := result["sandboxes"].([]map[string]interface{})
	if !ok || len(sandboxes) == 0 {
		return "No sandboxes found."
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d sandbox(es):\n\n", len(sandboxes)))
	for _, sb := range sandboxes {
		b.WriteString(fmt.Sprintf("- **%s** (%s)\n", sb["name"], sb["id"]))
		b.WriteString(fmt.Sprintf("  State: %s", sb["state"]))
		if host, ok := sb["host"].(string); ok && host != "" {
			b.WriteString(fmt.Sprintf(" | Host: %s", host))
		}
		if ip, ok := sb["ip"].(string); ok && ip != "" {
			b.WriteString(fmt.Sprintf(" | IP: %s", ip))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (a *FluidAgent) formatCreateResult(result map[string]interface{}, err error) string {
	if err != nil {
		return fmt.Sprintf("Failed to create sandbox: %v", err)
	}
	if result == nil {
		return "Sandbox created but no details available."
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Sandbox created successfully!\n\n"))
	b.WriteString(fmt.Sprintf("- **ID**: %s\n", result["sandbox_id"]))
	b.WriteString(fmt.Sprintf("- **Name**: %s\n", result["name"]))
	b.WriteString(fmt.Sprintf("- **State**: %s\n", result["state"]))
	if host, ok := result["host"].(string); ok && host != "" {
		b.WriteString(fmt.Sprintf("- **Host**: %s\n", host))
	}
	if ip, ok := result["ip"].(string); ok && ip != "" {
		b.WriteString(fmt.Sprintf("- **IP**: %s\n", ip))
	}
	return b.String()
}

func (a *FluidAgent) formatDestroyResult(result map[string]interface{}, err error) string {
	if err != nil {
		return fmt.Sprintf("Failed to destroy sandbox: %v", err)
	}
	return fmt.Sprintf("Sandbox %s destroyed successfully.", result["sandbox_id"])
}

func (a *FluidAgent) formatRunResult(result map[string]interface{}, err error) string {
	if err != nil && result == nil {
		return fmt.Sprintf("Command failed: %v", err)
	}

	var b strings.Builder
	exitCode := 0
	if ec, ok := result["exit_code"].(int); ok {
		exitCode = ec
	}

	b.WriteString(fmt.Sprintf("Command completed (exit code: %d)\n\n", exitCode))

	if stdout, ok := result["stdout"].(string); ok && stdout != "" {
		b.WriteString("**stdout:**\n```\n")
		b.WriteString(stdout)
		b.WriteString("```\n")
	}

	if stderr, ok := result["stderr"].(string); ok && stderr != "" {
		b.WriteString("**stderr:**\n```\n")
		b.WriteString(stderr)
		b.WriteString("```\n")
	}

	return b.String()
}

func (a *FluidAgent) formatStartResult(result map[string]interface{}, err error) string {
	if err != nil {
		return fmt.Sprintf("Failed to start sandbox: %v", err)
	}

	msg := fmt.Sprintf("Sandbox %s started.", result["sandbox_id"])
	if ip, ok := result["ip"].(string); ok && ip != "" {
		msg += fmt.Sprintf(" IP: %s", ip)
	}
	return msg
}

func (a *FluidAgent) formatStopResult(result map[string]interface{}, err error) string {
	if err != nil {
		return fmt.Sprintf("Failed to stop sandbox: %v", err)
	}
	return fmt.Sprintf("Sandbox %s stopped.", result["sandbox_id"])
}

func (a *FluidAgent) formatGetResult(result map[string]interface{}, err error) string {
	if err != nil {
		return fmt.Sprintf("Failed to get sandbox: %v", err)
	}
	if result == nil {
		return "Sandbox not found."
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return fmt.Sprintf("Sandbox details:\n```json\n%s\n```", string(data))
}

func (a *FluidAgent) formatVMsResult(result map[string]interface{}, err error) string {
	if err != nil {
		return fmt.Sprintf("Failed to list VMs: %v", err)
	}

	vms, ok := result["vms"].([]map[string]interface{})
	if !ok || len(vms) == 0 {
		return "No VMs found."
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d VM(s) available for cloning:\n\n", len(vms)))

	// Group VMs by host if host information is present
	hostVMs := make(map[string][]map[string]interface{})
	for _, vm := range vms {
		host := "local"
		if h, ok := vm["host"].(string); ok && h != "" {
			host = h
		}
		hostVMs[host] = append(hostVMs[host], vm)
	}

	// Display VMs grouped by host
	for host, hvms := range hostVMs {
		if len(hostVMs) > 1 || host != "local" {
			b.WriteString(fmt.Sprintf("### Host: %s\n", host))
		}
		for _, vm := range hvms {
			state := "unknown"
			if s, ok := vm["state"].(string); ok {
				state = s
			}
			b.WriteString(fmt.Sprintf("- **%s** (%s)\n", vm["name"], state))
		}
		b.WriteString("\n")
	}

	// Display any host errors
	if hostErrors, ok := result["host_errors"].([]map[string]interface{}); ok && len(hostErrors) > 0 {
		b.WriteString("### Host Errors\n")
		for _, he := range hostErrors {
			b.WriteString(fmt.Sprintf("- **%s**: %s\n", he["host"], he["error"]))
		}
	}

	return b.String()
}

func (a *FluidAgent) formatSnapshotResult(result map[string]interface{}, err error) string {
	if err != nil {
		return fmt.Sprintf("Failed to create snapshot: %v", err)
	}
	return fmt.Sprintf("Snapshot **%s** created for sandbox %s.", result["name"], result["sandbox_id"])
}

func (a *FluidAgent) getHelpText() string {
	return `# Available Commands

## Sandbox Management
- **list** (ls) - List all sandboxes
- **create** <source-vm> [--name=name] [--host=hostname] - Create a new sandbox
- **destroy** <sandbox-id> - Destroy a sandbox
- **get** <sandbox-id> - Get sandbox details
- **start** <sandbox-id> - Start a stopped sandbox
- **stop** <sandbox-id> - Stop a running sandbox

## Command Execution
- **run** <sandbox-id> <command> - Run a command in a sandbox

## Snapshots
- **snapshot** <sandbox-id> [name] - Create a snapshot

## Other
- **vms** - List available VMs from all configured hosts
- **hosts** - Show configured remote hosts
- **help** - Show this help message

## Multi-Host Support
When hosts are configured, VMs will be queried from all remote hosts.
Use --host=<hostname> to target a specific host when creating sandboxes.

## Examples
` + "```" + `
hosts
vms
create ubuntu-base
create ubuntu-base --host=kvm-01
list
run SBX-abc123 whoami
snapshot SBX-abc123 my-snapshot
destroy SBX-abc123
` + "```"
}

func (a *FluidAgent) getHostsInfo() string {
	var b strings.Builder

	if a.multiHostMgr == nil || len(a.cfg.Hosts) == 0 {
		return "No remote hosts configured. Using local virsh only.\n\nTo configure hosts, add them to your config.yaml:\n```yaml\nhosts:\n  - name: kvm-01\n    address: 10.0.0.11\n    ssh_user: root  # optional, default: root\n    ssh_port: 22    # optional, default: 22\n```"
	}

	b.WriteString(fmt.Sprintf("# Configured Hosts (%d)\n\n", len(a.cfg.Hosts)))

	for _, host := range a.cfg.Hosts {
		b.WriteString(fmt.Sprintf("### %s\n", host.Name))
		b.WriteString(fmt.Sprintf("- **Address**: %s\n", host.Address))

		sshUser := host.SSHUser
		if sshUser == "" {
			sshUser = "root (default)"
		}
		b.WriteString(fmt.Sprintf("- **SSH User**: %s\n", sshUser))

		sshPort := host.SSHPort
		if sshPort == 0 {
			sshPort = 22
		}
		b.WriteString(fmt.Sprintf("- **SSH Port**: %d\n", sshPort))
		b.WriteString("\n")
	}

	b.WriteString("Use `vms` to query VMs from all hosts.\n")

	return b.String()
}
