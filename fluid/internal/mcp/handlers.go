package mcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/aspectrr/fluid.sh/fluid/internal/ansible"
	"github.com/aspectrr/fluid.sh/fluid/internal/config"
	"github.com/aspectrr/fluid.sh/fluid/internal/store"
)

// jsonResult marshals v to JSON and returns it as a text tool result.
func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	return mcp.NewToolResultText(string(data)), nil
}

// shellEscape safely escapes a string for use in a shell command.
func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// findHostForSourceVM finds the host that has the given source VM.
func (s *Server) findHostForSourceVM(ctx context.Context, sourceVM, hostName string) (*config.HostConfig, error) {
	if s.multiHostMgr == nil {
		return nil, nil
	}

	if hostName != "" {
		hosts := s.multiHostMgr.GetHosts()
		for i := range hosts {
			if hosts[i].Name == hostName {
				return &hosts[i], nil
			}
		}
		return nil, fmt.Errorf("host %q not found in configuration", hostName)
	}

	host, err := s.multiHostMgr.FindHostForVM(ctx, sourceVM)
	if err != nil {
		return nil, nil
	}
	return host, nil
}

// --- Handlers ---

func (s *Server) handleListSandboxes(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sandboxes, err := s.vmService.GetSandboxes(ctx, store.SandboxFilter{}, nil)
	if err != nil {
		return nil, fmt.Errorf("list sandboxes: %w", err)
	}

	result := make([]map[string]any, 0, len(sandboxes))
	for _, sb := range sandboxes {
		item := map[string]any{
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

	return jsonResult(map[string]any{
		"sandboxes": result,
		"count":     len(result),
	})
}

func (s *Server) handleCreateSandbox(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceVM := request.GetString("source_vm", "")
	if sourceVM == "" {
		return nil, fmt.Errorf("source_vm is required")
	}
	hostName := request.GetString("host", "")
	cpu := request.GetInt("cpu", 0)
	memoryMB := request.GetInt("memory_mb", 0)

	var host *config.HostConfig
	if s.multiHostMgr != nil {
		var err error
		host, err = s.findHostForSourceVM(ctx, sourceVM, hostName)
		if err != nil {
			return nil, fmt.Errorf("find host for source VM: %w", err)
		}
	}

	if host != nil {
		sb, ip, err := s.vmService.CreateSandboxOnHost(ctx, host, sourceVM, "mcp-agent", "", cpu, memoryMB, nil, true, true)
		if err != nil {
			return nil, fmt.Errorf("create sandbox on host: %w", err)
		}
		result := map[string]any{
			"sandbox_id": sb.ID,
			"name":       sb.SandboxName,
			"state":      sb.State,
			"host":       host.Name,
		}
		if ip != "" {
			result["ip"] = ip
		}
		return jsonResult(result)
	}

	sb, ip, err := s.vmService.CreateSandbox(ctx, sourceVM, "mcp-agent", "", cpu, memoryMB, nil, true, true)
	if err != nil {
		return nil, fmt.Errorf("create sandbox: %w", err)
	}
	result := map[string]any{
		"sandbox_id": sb.ID,
		"name":       sb.SandboxName,
		"state":      sb.State,
	}
	if ip != "" {
		result["ip"] = ip
	}
	return jsonResult(result)
}

func (s *Server) handleDestroySandbox(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := request.GetString("sandbox_id", "")
	if id == "" {
		return nil, fmt.Errorf("sandbox_id is required")
	}

	_, err := s.vmService.DestroySandbox(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("destroy sandbox: %w", err)
	}

	return jsonResult(map[string]any{
		"destroyed":  true,
		"sandbox_id": id,
	})
}

func (s *Server) handleRunCommand(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sandboxID := request.GetString("sandbox_id", "")
	command := request.GetString("command", "")
	if sandboxID == "" {
		return nil, fmt.Errorf("sandbox_id is required")
	}
	if command == "" {
		return nil, fmt.Errorf("command is required")
	}

	user := s.cfg.SSH.DefaultUser
	result, err := s.vmService.RunCommand(ctx, sandboxID, user, "", command, 0, nil)
	if err != nil {
		if result != nil {
			return jsonResult(map[string]any{
				"sandbox_id": sandboxID,
				"exit_code":  result.ExitCode,
				"stdout":     result.Stdout,
				"stderr":     result.Stderr,
				"error":      err.Error(),
			})
		}
		return nil, fmt.Errorf("run command: %w", err)
	}

	return jsonResult(map[string]any{
		"sandbox_id": sandboxID,
		"exit_code":  result.ExitCode,
		"stdout":     result.Stdout,
		"stderr":     result.Stderr,
	})
}

func (s *Server) handleStartSandbox(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := request.GetString("sandbox_id", "")
	if id == "" {
		return nil, fmt.Errorf("sandbox_id is required")
	}

	ip, err := s.vmService.StartSandbox(ctx, id, true)
	if err != nil {
		return nil, fmt.Errorf("start sandbox: %w", err)
	}

	result := map[string]any{
		"started":    true,
		"sandbox_id": id,
	}
	if ip != "" {
		result["ip"] = ip
	}
	return jsonResult(result)
}

func (s *Server) handleStopSandbox(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := request.GetString("sandbox_id", "")
	if id == "" {
		return nil, fmt.Errorf("sandbox_id is required")
	}

	err := s.vmService.StopSandbox(ctx, id, false)
	if err != nil {
		return nil, fmt.Errorf("stop sandbox: %w", err)
	}

	return jsonResult(map[string]any{
		"stopped":    true,
		"sandbox_id": id,
	})
}

func (s *Server) handleGetSandbox(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := request.GetString("sandbox_id", "")
	if id == "" {
		return nil, fmt.Errorf("sandbox_id is required")
	}

	sb, err := s.vmService.GetSandbox(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get sandbox: %w", err)
	}

	result := map[string]any{
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

	return jsonResult(result)
}

func (s *Server) handleListVMs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.multiHostMgr != nil {
		return s.listVMsFromHosts(ctx)
	}
	return s.listVMsLocal(ctx)
}

func (s *Server) listVMsFromHosts(ctx context.Context) (*mcp.CallToolResult, error) {
	listResult, err := s.multiHostMgr.ListDomains(ctx)
	if err != nil {
		return nil, fmt.Errorf("list domains from hosts: %w", err)
	}

	vms := make([]map[string]any, 0)
	for _, domain := range listResult.Domains {
		if strings.HasPrefix(domain.Name, "sbx-") {
			continue
		}
		item := map[string]any{
			"name":         domain.Name,
			"state":        domain.State.String(),
			"host":         domain.HostName,
			"host_address": domain.HostAddress,
		}
		if domain.UUID != "" {
			item["uuid"] = domain.UUID
		}
		vms = append(vms, item)
	}

	response := map[string]any{
		"vms":   vms,
		"count": len(vms),
	}

	if len(listResult.HostErrors) > 0 {
		errors := make([]map[string]any, 0, len(listResult.HostErrors))
		for _, he := range listResult.HostErrors {
			errors = append(errors, map[string]any{
				"host":    he.HostName,
				"address": he.HostAddress,
				"error":   he.Error,
			})
		}
		response["host_errors"] = errors
	}

	return jsonResult(response)
}

func (s *Server) listVMsLocal(ctx context.Context) (*mcp.CallToolResult, error) {
	cmd := exec.CommandContext(ctx, "virsh", "list", "--all", "--name")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("virsh list: %w: %s", err, stderr.String())
	}

	vms := make([]map[string]any, 0)
	for _, name := range strings.Split(stdout.String(), "\n") {
		name = strings.TrimSpace(name)
		if name == "" || strings.HasPrefix(name, "sbx-") {
			continue
		}
		vms = append(vms, map[string]any{
			"name":  name,
			"state": "unknown",
			"host":  "local",
		})
	}

	return jsonResult(map[string]any{
		"vms":   vms,
		"count": len(vms),
	})
}

func (s *Server) handleCreateSnapshot(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sandboxID := request.GetString("sandbox_id", "")
	if sandboxID == "" {
		return nil, fmt.Errorf("sandbox_id is required")
	}
	name := request.GetString("name", "")
	if name == "" {
		name = fmt.Sprintf("snap-%d", time.Now().Unix())
	}

	snap, err := s.vmService.CreateSnapshot(ctx, sandboxID, name, false)
	if err != nil {
		return nil, fmt.Errorf("create snapshot: %w", err)
	}

	return jsonResult(map[string]any{
		"snapshot_id": snap.ID,
		"sandbox_id":  sandboxID,
		"name":        snap.Name,
		"kind":        snap.Kind,
	})
}

func (s *Server) handleCreatePlaybook(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := request.GetString("name", "")
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	hosts := request.GetString("hosts", "")
	become := request.GetBool("become", false)

	pb, err := s.playbookService.CreatePlaybook(ctx, ansible.CreatePlaybookRequest{
		Name:   name,
		Hosts:  hosts,
		Become: become,
	})
	if err != nil {
		return nil, fmt.Errorf("create playbook: %w", err)
	}

	result := map[string]any{
		"id":         pb.ID,
		"name":       pb.Name,
		"hosts":      pb.Hosts,
		"become":     pb.Become,
		"created_at": pb.CreatedAt.Format(time.RFC3339),
	}
	if pb.FilePath != nil {
		result["file_path"] = *pb.FilePath
	}
	return jsonResult(result)
}

func (s *Server) handleAddPlaybookTask(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	playbookID := request.GetString("playbook_id", "")
	if playbookID == "" {
		return nil, fmt.Errorf("playbook_id is required")
	}
	name := request.GetString("name", "")
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	module := request.GetString("module", "")
	if module == "" {
		return nil, fmt.Errorf("module is required")
	}

	args := request.GetArguments()
	var params map[string]any
	if p, ok := args["params"]; ok {
		if m, ok := p.(map[string]any); ok {
			params = m
		}
	}

	task, err := s.playbookService.AddTask(ctx, playbookID, ansible.AddTaskRequest{
		Name:   name,
		Module: module,
		Params: params,
	})
	if err != nil {
		return nil, fmt.Errorf("add playbook task: %w", err)
	}

	return jsonResult(map[string]any{
		"id":          task.ID,
		"playbook_id": playbookID,
		"name":        task.Name,
		"module":      task.Module,
		"position":    task.Position,
	})
}

func (s *Server) handleEditFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sandboxID := request.GetString("sandbox_id", "")
	if sandboxID == "" {
		return nil, fmt.Errorf("sandbox_id is required")
	}
	path := request.GetString("path", "")
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("path must be absolute: %s", path)
	}
	oldStr := request.GetString("old_str", "")
	newStr := request.GetString("new_str", "")

	user := s.cfg.SSH.DefaultUser

	if oldStr == "" {
		// Create/overwrite file
		encoded := base64.StdEncoding.EncodeToString([]byte(newStr))
		cmd := fmt.Sprintf("echo '%s' | base64 -d > %s", encoded, shellEscape(path))
		result, err := s.vmService.RunCommand(ctx, sandboxID, user, "", cmd, 0, nil)
		if err != nil {
			return nil, fmt.Errorf("create file: %w", err)
		}
		if result.ExitCode != 0 {
			return nil, fmt.Errorf("create file: %s", result.Stderr)
		}
		return jsonResult(map[string]any{
			"sandbox_id": sandboxID,
			"path":       path,
			"action":     "created_file",
		})
	}

	// Read existing file
	readResult, err := s.vmService.RunCommand(ctx, sandboxID, user, "", fmt.Sprintf("base64 %s", shellEscape(path)), 0, nil)
	if err != nil {
		return nil, fmt.Errorf("read file for edit: %w", err)
	}
	if readResult.ExitCode != 0 {
		return nil, fmt.Errorf("read file: %s", readResult.Stderr)
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(readResult.Stdout))
	if err != nil {
		return nil, fmt.Errorf("decode file content: %w", err)
	}
	original := string(decoded)

	if !strings.Contains(original, oldStr) {
		return jsonResult(map[string]any{
			"sandbox_id": sandboxID,
			"path":       path,
			"action":     "old_str_not_found",
		})
	}

	edited := strings.Replace(original, oldStr, newStr, 1)
	encoded := base64.StdEncoding.EncodeToString([]byte(edited))
	writeCmd := fmt.Sprintf("echo '%s' | base64 -d > %s", encoded, shellEscape(path))
	writeResult, err := s.vmService.RunCommand(ctx, sandboxID, user, "", writeCmd, 0, nil)
	if err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}
	if writeResult.ExitCode != 0 {
		return nil, fmt.Errorf("write file: %s", writeResult.Stderr)
	}

	return jsonResult(map[string]any{
		"sandbox_id": sandboxID,
		"path":       path,
		"action":     "edited",
	})
}

func (s *Server) handleReadFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sandboxID := request.GetString("sandbox_id", "")
	if sandboxID == "" {
		return nil, fmt.Errorf("sandbox_id is required")
	}
	path := request.GetString("path", "")
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("path must be absolute: %s", path)
	}

	user := s.cfg.SSH.DefaultUser
	result, err := s.vmService.RunCommand(ctx, sandboxID, user, "", fmt.Sprintf("base64 %s", shellEscape(path)), 0, nil)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("read file: %s", result.Stderr)
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(result.Stdout))
	if err != nil {
		return nil, fmt.Errorf("decode file content: %w", err)
	}

	return jsonResult(map[string]any{
		"sandbox_id": sandboxID,
		"path":       path,
		"content":    string(decoded),
	})
}

func (s *Server) handleListPlaybooks(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	playbooks, err := s.playbookService.ListPlaybooks(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("list playbooks: %w", err)
	}

	result := make([]map[string]any, 0, len(playbooks))
	for _, pb := range playbooks {
		path := ""
		if pb.FilePath != nil && *pb.FilePath != "" {
			path = *pb.FilePath
		} else {
			path = filepath.Join(s.cfg.Ansible.PlaybooksDir, pb.Name+".yml")
		}
		result = append(result, map[string]any{
			"id":         pb.ID,
			"name":       pb.Name,
			"path":       path,
			"created_at": pb.CreatedAt.Format(time.RFC3339),
		})
	}

	return jsonResult(map[string]any{
		"playbooks": result,
		"count":     len(result),
	})
}

func (s *Server) handleGetPlaybook(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	playbookID := request.GetString("playbook_id", "")
	if playbookID == "" {
		return nil, fmt.Errorf("playbook_id is required")
	}

	pbWithTasks, err := s.playbookService.GetPlaybookWithTasks(ctx, playbookID)
	if err != nil {
		return nil, fmt.Errorf("get playbook: %w", err)
	}

	yamlContent, err := s.playbookService.ExportPlaybook(ctx, playbookID)
	if err != nil {
		return nil, fmt.Errorf("export playbook: %w", err)
	}

	tasks := make([]map[string]any, 0, len(pbWithTasks.Tasks))
	for _, t := range pbWithTasks.Tasks {
		tasks = append(tasks, map[string]any{
			"id":       t.ID,
			"position": t.Position,
			"name":     t.Name,
			"module":   t.Module,
			"params":   t.Params,
		})
	}

	result := map[string]any{
		"id":           pbWithTasks.Playbook.ID,
		"name":         pbWithTasks.Playbook.Name,
		"hosts":        pbWithTasks.Playbook.Hosts,
		"become":       pbWithTasks.Playbook.Become,
		"tasks":        tasks,
		"yaml_content": string(yamlContent),
		"created_at":   pbWithTasks.Playbook.CreatedAt.Format(time.RFC3339),
	}
	if pbWithTasks.Playbook.FilePath != nil {
		result["file_path"] = *pbWithTasks.Playbook.FilePath
	}

	return jsonResult(result)
}

func (s *Server) handleRunSourceCommand(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceVM := request.GetString("source_vm", "")
	if sourceVM == "" {
		return nil, fmt.Errorf("source_vm is required")
	}
	command := request.GetString("command", "")
	if command == "" {
		return nil, fmt.Errorf("command is required")
	}

	result, err := s.vmService.RunSourceVMCommand(ctx, sourceVM, command, 0)
	if err != nil {
		if result != nil {
			return jsonResult(map[string]any{
				"source_vm": sourceVM,
				"exit_code": result.ExitCode,
				"stdout":    result.Stdout,
				"stderr":    result.Stderr,
				"error":     err.Error(),
			})
		}
		return nil, fmt.Errorf("run source command: %w", err)
	}

	return jsonResult(map[string]any{
		"source_vm": sourceVM,
		"exit_code": result.ExitCode,
		"stdout":    result.Stdout,
		"stderr":    result.Stderr,
	})
}

func (s *Server) handleReadSourceFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceVM := request.GetString("source_vm", "")
	if sourceVM == "" {
		return nil, fmt.Errorf("source_vm is required")
	}
	path := request.GetString("path", "")
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("path must be absolute: %s", path)
	}

	cmd := fmt.Sprintf("base64 %s", shellEscape(path))
	result, err := s.vmService.RunSourceVMCommand(ctx, sourceVM, cmd, 0)
	if err != nil {
		return nil, fmt.Errorf("read source file: %w", err)
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("read source file: %s", result.Stderr)
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(result.Stdout))
	if err != nil {
		return nil, fmt.Errorf("decode file content: %w", err)
	}

	return jsonResult(map[string]any{
		"source_vm": sourceVM,
		"path":      path,
		"content":   string(decoded),
	})
}
