package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aspectrr/fluid.sh/fluid/internal/ansible"
	"github.com/aspectrr/fluid.sh/fluid/internal/config"
	"github.com/aspectrr/fluid.sh/fluid/internal/libvirt"
	"github.com/aspectrr/fluid.sh/fluid/internal/llm"
	"github.com/aspectrr/fluid.sh/fluid/internal/store"
	"github.com/aspectrr/fluid.sh/fluid/internal/telemetry"
	"github.com/aspectrr/fluid.sh/fluid/internal/vm"
)

// FluidAgent implements AgentRunner for the fluid CLI
type FluidAgent struct {
	cfg             *config.Config
	store           store.Store
	vmService       *vm.Service
	manager         libvirt.Manager
	llmClient       llm.Client
	playbookService *ansible.PlaybookService
	telemetry       telemetry.Service

	// Multi-host support
	multiHostMgr *libvirt.MultiHostDomainManager

	// Status callback for sending updates to TUI
	statusCallback func(tea.Msg)

	// Conversation history for context
	history []llm.Message

	// Track sandboxes created during this session for cleanup on exit
	createdSandboxes []string
}

// NewFluidAgent creates a new fluid agent
func NewFluidAgent(cfg *config.Config, store store.Store, vmService *vm.Service, manager libvirt.Manager, tele telemetry.Service) *FluidAgent {
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
		telemetry:       tele,
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
			case "/sandboxes":
				result, err := a.listSandboxes(ctx)
				return AgentResponseMsg{Response: AgentResponse{
					Content: a.formatSandboxesResult(result, err),
				}}
			case "/hosts":
				result, err := a.listHostsWithVMs(ctx)
				return AgentResponseMsg{Response: AgentResponse{
					Content: a.formatHostsResult(result, err),
				}}
			case "/playbooks":
				result, err := a.listPlaybooks(ctx)
				return AgentResponseMsg{Response: AgentResponse{
					Content: a.formatPlaybooksResult(result, err),
				}}
			default:
				return AgentResponseMsg{Response: AgentResponse{
					Content: fmt.Sprintf("Unknown command: %s. Available: /vms, /sandboxes, /hosts, /playbooks, /settings", input),
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

			if a.telemetry != nil {
				a.telemetry.Track("agent_prompt_sent", map[string]any{
					"message_count": len(req.Messages),
					"provider":      a.cfg.AIAgent.Provider,
					"model":         a.cfg.AIAgent.Model,
				})
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

	if a.telemetry != nil {
		a.telemetry.Track("agent_tool_call", map[string]interface{}{
			"tool_name": tc.Function.Name,
		})
	}

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

			// Track the created sandbox for cleanup on exit
			a.createdSandboxes = append(a.createdSandboxes, sb.ID)

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

	// Track the created sandbox for cleanup on exit
	a.createdSandboxes = append(a.createdSandboxes, sb.ID)

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

// listVMsFromHosts queries all configured remote hosts for VMs (excludes sandboxes)
func (a *FluidAgent) listVMsFromHosts(ctx context.Context) (map[string]interface{}, error) {
	listResult, err := a.multiHostMgr.ListDomains(ctx)
	if err != nil {
		return nil, fmt.Errorf("list domains from hosts: %w", err)
	}

	result := make([]map[string]interface{}, 0)
	for _, domain := range listResult.Domains {
		// Skip sandboxes (names starting with "sbx-")
		if strings.HasPrefix(domain.Name, "sbx-") {
			continue
		}
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

// listVMsLocal queries local virsh for VMs (excludes sandboxes)
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
		// Skip sandboxes (names starting with "sbx-")
		if strings.HasPrefix(name, "sbx-") {
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

func (a *FluidAgent) formatSandboxesResult(result map[string]interface{}, err error) string {
	if err != nil {
		return fmt.Sprintf("Failed to list sandboxes: %v", err)
	}

	sandboxes, ok := result["sandboxes"].([]map[string]interface{})
	if !ok || len(sandboxes) == 0 {
		return "No sandboxes found."
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d sandbox(es):\n\n", len(sandboxes)))

	// Group sandboxes by host if host information is present
	hostSandboxes := make(map[string][]map[string]interface{})
	for _, sb := range sandboxes {
		host := "local"
		if h, ok := sb["host"].(string); ok && h != "" {
			host = h
		}
		hostSandboxes[host] = append(hostSandboxes[host], sb)
	}

	// Display sandboxes grouped by host
	for host, sbs := range hostSandboxes {
		if len(hostSandboxes) > 1 || host != "local" {
			b.WriteString(fmt.Sprintf("### Host: %s\n", host))
		}
		for _, sb := range sbs {
			state := "unknown"
			if s, ok := sb["state"].(string); ok {
				state = s
			}
			name := sb["name"]
			id := sb["id"]
			baseImage := sb["base_image"]

			b.WriteString(fmt.Sprintf("- **%s** (%s)\n", name, id))
			b.WriteString(fmt.Sprintf("  State: %s | Base: %s", state, baseImage))
			if ip, ok := sb["ip"].(string); ok {
				b.WriteString(fmt.Sprintf(" | IP: %s", ip))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}

// listHostsWithVMs queries all hosts and returns VMs differentiated by type (host VM vs sandbox)
func (a *FluidAgent) listHostsWithVMs(ctx context.Context) (map[string]interface{}, error) {
	// Get sandboxes from database
	sandboxes, err := a.vmService.GetSandboxes(ctx, store.SandboxFilter{}, nil)
	if err != nil {
		return nil, fmt.Errorf("list sandboxes: %w", err)
	}

	// Build a set of sandbox names for quick lookup
	sandboxNames := make(map[string]bool)
	for _, sb := range sandboxes {
		sandboxNames[sb.SandboxName] = true
	}

	// Get all domains from libvirt
	var domains []map[string]interface{}
	var hostErrors []map[string]interface{}

	if a.multiHostMgr != nil {
		listResult, err := a.multiHostMgr.ListDomains(ctx)
		if err != nil {
			return nil, fmt.Errorf("list domains from hosts: %w", err)
		}
		for _, domain := range listResult.Domains {
			isSandbox := strings.HasPrefix(domain.Name, "sbx-") || sandboxNames[domain.Name]
			domains = append(domains, map[string]interface{}{
				"name":         domain.Name,
				"state":        domain.State.String(),
				"host":         domain.HostName,
				"host_address": domain.HostAddress,
				"type":         vmType(isSandbox),
			})
		}
		for _, he := range listResult.HostErrors {
			hostErrors = append(hostErrors, map[string]interface{}{
				"host":    he.HostName,
				"address": he.HostAddress,
				"error":   he.Error,
			})
		}
	} else {
		// Local virsh
		cmd := exec.CommandContext(ctx, "virsh", "list", "--all", "--name")
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("virsh list: %w: %s", err, stderr.String())
		}

		for _, name := range strings.Split(stdout.String(), "\n") {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			isSandbox := strings.HasPrefix(name, "sbx-") || sandboxNames[name]
			domains = append(domains, map[string]interface{}{
				"name":  name,
				"state": "unknown",
				"host":  "local",
				"type":  vmType(isSandbox),
			})
		}
	}

	response := map[string]interface{}{
		"domains": domains,
		"count":   len(domains),
	}
	if len(hostErrors) > 0 {
		response["host_errors"] = hostErrors
	}

	return response, nil
}

// vmType returns "sandbox" or "host_vm" based on whether the domain is a sandbox
func vmType(isSandbox bool) string {
	if isSandbox {
		return "sandbox"
	}
	return "host_vm"
}

func (a *FluidAgent) formatHostsResult(result map[string]interface{}, err error) string {
	if err != nil {
		return fmt.Sprintf("Failed to list hosts: %v", err)
	}

	domains, ok := result["domains"].([]map[string]interface{})
	if !ok || len(domains) == 0 {
		return "No domains found on any host."
	}

	var b strings.Builder

	// Group domains by host
	hostDomains := make(map[string][]map[string]interface{})
	for _, d := range domains {
		host := "local"
		if h, ok := d["host"].(string); ok && h != "" {
			host = h
		}
		hostDomains[host] = append(hostDomains[host], d)
	}

	// Count totals
	totalHostVMs := 0
	totalSandboxes := 0
	for _, ds := range hostDomains {
		for _, d := range ds {
			if d["type"] == "sandbox" {
				totalSandboxes++
			} else {
				totalHostVMs++
			}
		}
	}

	b.WriteString("# Hosts Overview\n\n")
	b.WriteString(fmt.Sprintf("Total: %d host VM(s), %d sandbox(es)\n\n", totalHostVMs, totalSandboxes))

	// Display domains grouped by host
	for host, ds := range hostDomains {
		// Count per host
		hostVMCount := 0
		sandboxCount := 0
		for _, d := range ds {
			if d["type"] == "sandbox" {
				sandboxCount++
			} else {
				hostVMCount++
			}
		}

		b.WriteString(fmt.Sprintf("## %s\n", host))
		b.WriteString(fmt.Sprintf("Host VMs: %d | Sandboxes: %d\n\n", hostVMCount, sandboxCount))

		// Display host VMs first
		if hostVMCount > 0 {
			b.WriteString("**Host VMs (available for cloning):**\n")
			for _, d := range ds {
				if d["type"] != "host_vm" {
					continue
				}
				state := "unknown"
				if s, ok := d["state"].(string); ok {
					state = s
				}
				b.WriteString(fmt.Sprintf("- %s (%s)\n", d["name"], state))
			}
			b.WriteString("\n")
		}

		// Display sandboxes
		if sandboxCount > 0 {
			b.WriteString("**Sandboxes (ephemeral VMs):**\n")
			for _, d := range ds {
				if d["type"] != "sandbox" {
					continue
				}
				state := "unknown"
				if s, ok := d["state"].(string); ok {
					state = s
				}
				b.WriteString(fmt.Sprintf("- %s (%s)\n", d["name"], state))
			}
			b.WriteString("\n")
		}
	}

	// Display any host errors
	if hostErrors, ok := result["host_errors"].([]map[string]interface{}); ok && len(hostErrors) > 0 {
		b.WriteString("## Host Errors\n")
		for _, he := range hostErrors {
			b.WriteString(fmt.Sprintf("- **%s**: %s\n", he["host"], he["error"]))
		}
	}

	return b.String()
}

func (a *FluidAgent) listPlaybooks(ctx context.Context) (map[string]interface{}, error) {
	playbooks, err := a.playbookService.ListPlaybooks(ctx, nil)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0, len(playbooks))
	for _, pb := range playbooks {
		path := ""
		if pb.FilePath != nil && *pb.FilePath != "" {
			path = *pb.FilePath
		} else {
			path = filepath.Join(a.cfg.Ansible.PlaybooksDir, pb.Name+".yml")
		}
		result = append(result, map[string]interface{}{
			"id":         pb.ID,
			"name":       pb.Name,
			"path":       path,
			"created_at": pb.CreatedAt.Format(time.RFC3339),
		})
	}

	return map[string]interface{}{
		"playbooks": result,
		"count":     len(result),
	}, nil
}

func (a *FluidAgent) formatPlaybooksResult(result map[string]interface{}, err error) string {
	if err != nil {
		return fmt.Sprintf("Failed to list playbooks: %v", err)
	}

	playbooks, ok := result["playbooks"].([]map[string]interface{})
	if !ok || len(playbooks) == 0 {
		return "No playbooks found."
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d playbook(s):\n\n", len(playbooks)))
	for _, pb := range playbooks {
		name := pb["name"].(string)
		path := pb["path"].(string)

		absPath, _ := filepath.Abs(path)
		// OSC 8 hyperlink
		link := fmt.Sprintf("\033]8;;file://%s\033\\%s\033]8;;\033\\", absPath, path)

		b.WriteString(fmt.Sprintf("- **%s**: %s\n", name, link))
	}
	return b.String()
}

// Cleanup destroys all sandboxes created during this session.
// This is called when the TUI exits to ensure no orphaned VMs are left running.
func (a *FluidAgent) Cleanup(ctx context.Context) error {
	if len(a.createdSandboxes) == 0 {
		return nil
	}

	var errs []error
	for _, id := range a.createdSandboxes {
		// Check if sandbox still exists before destroying
		if _, err := a.vmService.GetSandbox(ctx, id); err != nil {
			// Sandbox no longer exists (already destroyed by user), skip
			continue
		}

		if _, err := a.vmService.DestroySandbox(ctx, id); err != nil {
			errs = append(errs, fmt.Errorf("destroy sandbox %s: %w", id, err))
			// Continue trying to destroy others even if one fails
		}
	}

	// Clear the list
	a.createdSandboxes = nil

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %v", errs)
	}
	return nil
}

// CreatedSandboxCount returns the number of sandboxes created during this session.
func (a *FluidAgent) CreatedSandboxCount() int {
	return len(a.createdSandboxes)
}
