// Package agent implements the gRPC client that connects the sandbox host
// to the control plane. It handles registration, heartbeat, and dispatching
// of commands received from the control plane to local managers.
package agent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"

	"github.com/aspectrr/fluid.sh/sandbox-host/internal/image"
	"github.com/aspectrr/fluid.sh/sandbox-host/internal/microvm"
	"github.com/aspectrr/fluid.sh/sandbox-host/internal/network"
	"github.com/aspectrr/fluid.sh/sandbox-host/internal/sourcevm"
	"github.com/aspectrr/fluid.sh/sandbox-host/internal/state"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client connects to the control plane via gRPC bidirectional streaming.
type Client struct {
	hostID     string
	hostname   string
	version    string
	cpAddr     string
	insecure   bool
	certFile   string
	keyFile    string
	caFile     string

	vmMgr      *microvm.Manager
	netMgr     *network.NetworkManager
	imgStore   *image.Store
	srcVMMgr   *sourcevm.Manager
	localStore *state.Store
	logger     *slog.Logger

	// stream is the active bidirectional stream to the control plane.
	mu     sync.Mutex
	stream fluidv1.HostService_ConnectClient
	conn   *grpc.ClientConn
}

// Config holds configuration for the gRPC agent client.
type Config struct {
	HostID   string
	Hostname string
	Version  string
	Address  string
	Insecure bool
	CertFile string
	KeyFile  string
	CAFile   string
}

// NewClient creates a new agent client.
func NewClient(
	cfg Config,
	vmMgr *microvm.Manager,
	netMgr *network.NetworkManager,
	imgStore *image.Store,
	srcVMMgr *sourcevm.Manager,
	localStore *state.Store,
	logger *slog.Logger,
) *Client {
	hostname := cfg.Hostname
	if hostname == "" {
		hostname, _ = os.Hostname()
	}

	return &Client{
		hostID:     cfg.HostID,
		hostname:   hostname,
		version:    cfg.Version,
		cpAddr:     cfg.Address,
		insecure:   cfg.Insecure,
		certFile:   cfg.CertFile,
		keyFile:    cfg.KeyFile,
		caFile:     cfg.CAFile,
		vmMgr:      vmMgr,
		netMgr:     netMgr,
		imgStore:   imgStore,
		srcVMMgr:   srcVMMgr,
		localStore: localStore,
		logger:     logger.With("component", "agent"),
	}
}

// Run connects to the control plane and runs the message loop. It reconnects
// automatically on failure using exponential backoff. Blocks until ctx is done.
func (c *Client) Run(ctx context.Context) error {
	return RunWithReconnect(ctx, c.logger, c.connectAndServe)
}

// connectAndServe establishes a single connection, registers, and runs the
// message loop. Returns an error when the connection drops.
func (c *Client) connectAndServe(ctx context.Context) error {
	// Dial control plane
	opts := []grpc.DialOption{}
	if c.insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(c.cpAddr, opts...)
	if err != nil {
		return fmt.Errorf("dial control plane %s: %w", c.cpAddr, err)
	}
	defer conn.Close()

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	client := fluidv1.NewHostServiceClient(conn)
	stream, err := client.Connect(ctx)
	if err != nil {
		return fmt.Errorf("open stream: %w", err)
	}

	c.mu.Lock()
	c.stream = stream
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.stream = nil
		c.conn = nil
		c.mu.Unlock()
	}()

	// Send registration
	if err := c.register(stream); err != nil {
		return err
	}

	// Start heartbeat goroutine
	heartbeatCtx, heartbeatCancel := context.WithCancel(ctx)
	defer heartbeatCancel()
	go c.heartbeatLoop(heartbeatCtx, stream)

	// Main recv loop
	return c.recvLoop(ctx, stream)
}

// register sends the HostRegistration message and waits for RegistrationAck.
func (c *Client) register(stream fluidv1.HostService_ConnectClient) error {
	reg := c.buildRegistration()

	reqID := uuid.New().String()
	msg := &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_Registration{
			Registration: reg,
		},
	}

	c.logger.Info("sending registration",
		"host_id", c.hostID,
		"hostname", c.hostname,
	)

	if err := stream.Send(msg); err != nil {
		return fmt.Errorf("send registration: %w", err)
	}

	// Wait for ack
	resp, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("recv registration ack: %w", err)
	}

	ack := resp.GetRegistrationAck()
	if ack == nil {
		return fmt.Errorf("expected RegistrationAck, got different message type")
	}

	if !ack.GetAccepted() {
		return fmt.Errorf("registration rejected: %s", ack.GetReason())
	}

	if assigned := ack.GetAssignedHostId(); assigned != "" && assigned != c.hostID {
		c.logger.Info("host ID reassigned by control plane", "old", c.hostID, "new", assigned)
		c.hostID = assigned
	}

	c.logger.Info("registered with control plane", "host_id", c.hostID)
	return nil
}

// buildRegistration constructs the HostRegistration message from local state.
func (c *Client) buildRegistration() *fluidv1.HostRegistration {
	reg := &fluidv1.HostRegistration{
		HostId:   c.hostID,
		Hostname: c.hostname,
		Version:  c.version,
	}

	// System resources
	reg.TotalCpus = int32(runtime.NumCPU())
	// Memory and disk are platform-specific; use estimates that can be refined
	// with actual sysinfo if needed.
	reg.AvailableCpus = int32(runtime.NumCPU())

	// Base images
	if c.imgStore != nil {
		names, _ := c.imgStore.ListNames()
		reg.BaseImages = names
	}

	// Source VMs
	if c.srcVMMgr != nil {
		vms, err := c.srcVMMgr.ListVMs(context.Background())
		if err == nil {
			for _, vm := range vms {
				reg.SourceVms = append(reg.SourceVms, &fluidv1.SourceVMInfo{
					Name:      vm.Name,
					State:     vm.State,
					IpAddress: vm.IPAddress,
					Prepared:  vm.Prepared,
				})
			}
		}
	}

	return reg
}

// heartbeatLoop sends periodic heartbeats to the control plane.
func (c *Client) heartbeatLoop(ctx context.Context, stream fluidv1.HostService_ConnectClient) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hb := &fluidv1.Heartbeat{
				AvailableCpus: int32(runtime.NumCPU()),
			}

			// Count active sandboxes
			if c.vmMgr != nil {
				hb.ActiveSandboxes = int32(len(c.vmMgr.List()))
			}

			msg := &fluidv1.HostMessage{
				Payload: &fluidv1.HostMessage_Heartbeat{
					Heartbeat: hb,
				},
			}

			if err := stream.Send(msg); err != nil {
				c.logger.Error("send heartbeat failed", "error", err)
				return
			}
		}
	}
}

// recvLoop receives and dispatches ControlMessages from the control plane.
func (c *Client) recvLoop(ctx context.Context, stream fluidv1.HostService_ConnectClient) error {
	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				c.logger.Info("stream closed by control plane")
				return nil
			}
			return fmt.Errorf("recv: %w", err)
		}

		// Dispatch in a goroutine so we don't block the recv loop.
		go c.handleCommand(ctx, stream, msg)
	}
}

// handleCommand dispatches a ControlMessage to the appropriate handler and
// sends the response back.
func (c *Client) handleCommand(ctx context.Context, stream fluidv1.HostService_ConnectClient, msg *fluidv1.ControlMessage) {
	reqID := msg.GetRequestId()

	var resp *fluidv1.HostMessage

	switch cmd := msg.Payload.(type) {
	case *fluidv1.ControlMessage_CreateSandbox:
		resp = c.handleCreateSandbox(ctx, reqID, cmd.CreateSandbox)
	case *fluidv1.ControlMessage_DestroySandbox:
		resp = c.handleDestroySandbox(ctx, reqID, cmd.DestroySandbox)
	case *fluidv1.ControlMessage_StartSandbox:
		resp = c.handleStartSandbox(ctx, reqID, cmd.StartSandbox)
	case *fluidv1.ControlMessage_StopSandbox:
		resp = c.handleStopSandbox(ctx, reqID, cmd.StopSandbox)
	case *fluidv1.ControlMessage_RunCommand:
		resp = c.handleRunCommand(ctx, reqID, cmd.RunCommand)
	case *fluidv1.ControlMessage_CreateSnapshot:
		resp = c.handleCreateSnapshot(ctx, reqID, cmd.CreateSnapshot)
	case *fluidv1.ControlMessage_PrepareSourceVm:
		resp = c.handlePrepareSourceVM(ctx, reqID, cmd.PrepareSourceVm)
	case *fluidv1.ControlMessage_RunSourceCommand:
		resp = c.handleRunSourceCommand(ctx, reqID, cmd.RunSourceCommand)
	case *fluidv1.ControlMessage_ReadSourceFile:
		resp = c.handleReadSourceFile(ctx, reqID, cmd.ReadSourceFile)
	case *fluidv1.ControlMessage_ListSourceVms:
		resp = c.handleListSourceVMs(ctx, reqID)
	case *fluidv1.ControlMessage_ValidateSourceVm:
		resp = c.handleValidateSourceVM(ctx, reqID, cmd.ValidateSourceVm)
	default:
		c.logger.Warn("unknown command type", "request_id", reqID)
		resp = errorResponse(reqID, "", "unknown command type")
	}

	if resp != nil {
		if err := stream.Send(resp); err != nil {
			c.logger.Error("send response failed", "request_id", reqID, "error", err)
		}
	}
}

// ---------------------------------------------------------------------------
// Sandbox command handlers
// ---------------------------------------------------------------------------

func (c *Client) handleCreateSandbox(ctx context.Context, reqID string, cmd *fluidv1.CreateSandboxCommand) *fluidv1.HostMessage {
	sandboxID := cmd.GetSandboxId()
	c.logger.Info("creating sandbox", "sandbox_id", sandboxID, "base_image", cmd.GetBaseImage())

	if c.vmMgr == nil {
		return errorResponse(reqID, sandboxID, "microVM manager not available")
	}

	// Resolve bridge
	bridge, err := c.netMgr.ResolveBridge(ctx, cmd.GetSourceVm(), cmd.GetNetwork())
	if err != nil {
		return errorResponse(reqID, sandboxID, fmt.Sprintf("resolve bridge: %v", err))
	}

	// Get base image path
	imagePath, err := c.imgStore.GetImagePath(cmd.GetBaseImage())
	if err != nil {
		return errorResponse(reqID, sandboxID, fmt.Sprintf("get base image: %v", err))
	}

	// Get kernel path
	kernelPath, err := c.imgStore.GetKernelPath(cmd.GetBaseImage())
	if err != nil {
		return errorResponse(reqID, sandboxID, fmt.Sprintf("get kernel: %v", err))
	}

	// Create overlay disk
	overlayPath, err := microvm.CreateOverlay(ctx, imagePath, c.vmMgr.WorkDir(), sandboxID)
	if err != nil {
		return errorResponse(reqID, sandboxID, fmt.Sprintf("create overlay: %v", err))
	}

	// Generate MAC address and TAP device
	mac := microvm.GenerateMACAddress()
	tapName := network.TAPName(sandboxID)

	// Create TAP device
	if err := network.CreateTAP(ctx, tapName, bridge, c.logger); err != nil {
		microvm.RemoveOverlay(c.vmMgr.WorkDir(), sandboxID)
		return errorResponse(reqID, sandboxID, fmt.Sprintf("create TAP: %v", err))
	}

	// Launch microVM
	vcpus := int(cmd.GetVcpus())
	if vcpus == 0 {
		vcpus = 2
	}
	memMB := int(cmd.GetMemoryMb())
	if memMB == 0 {
		memMB = 2048
	}

	info, err := c.vmMgr.Launch(ctx, microvm.LaunchConfig{
		SandboxID:   sandboxID,
		Name:        cmd.GetName(),
		OverlayPath: overlayPath,
		KernelPath:  kernelPath,
		TAPDevice:   tapName,
		MACAddress:  mac,
		Bridge:      bridge,
		VCPUs:       vcpus,
		MemoryMB:    memMB,
	})
	if err != nil {
		_ = network.DestroyTAP(ctx, tapName)
		microvm.RemoveOverlay(c.vmMgr.WorkDir(), sandboxID)
		return errorResponse(reqID, sandboxID, fmt.Sprintf("launch microVM: %v", err))
	}

	// Discover IP
	ip, err := c.netMgr.DiscoverIP(ctx, mac, bridge, 2*time.Minute)
	if err != nil {
		c.logger.Warn("IP discovery failed", "sandbox_id", sandboxID, "error", err)
		// Continue without IP - it may be discovered later
	}

	// Persist to local state
	localSandbox := &state.Sandbox{
		ID:         sandboxID,
		Name:       cmd.GetName(),
		BaseImage:  cmd.GetBaseImage(),
		State:      "RUNNING",
		IPAddress:  ip,
		MACAddress: mac,
		TAPDevice:  tapName,
		Bridge:     bridge,
		VCPUs:      vcpus,
		MemoryMB:   memMB,
		TTLSeconds: int(cmd.GetTtlSeconds()),
		AgentID:    cmd.GetAgentId(),
	}
	if err := c.localStore.CreateSandbox(ctx, localSandbox); err != nil {
		c.logger.Error("failed to persist sandbox locally", "sandbox_id", sandboxID, "error", err)
	}

	c.logger.Info("sandbox created",
		"sandbox_id", sandboxID,
		"ip", ip,
		"pid", info.PID,
		"bridge", bridge,
	)

	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_SandboxCreated{
			SandboxCreated: &fluidv1.SandboxCreated{
				SandboxId:  sandboxID,
				Name:       cmd.GetName(),
				State:      "RUNNING",
				IpAddress:  ip,
				MacAddress: mac,
				Bridge:     bridge,
				Pid:        int32(info.PID),
			},
		},
	}
}

func (c *Client) handleDestroySandbox(ctx context.Context, reqID string, cmd *fluidv1.DestroySandboxCommand) *fluidv1.HostMessage {
	sandboxID := cmd.GetSandboxId()
	c.logger.Info("destroying sandbox", "sandbox_id", sandboxID)

	if c.vmMgr != nil {
		info, err := c.vmMgr.Get(sandboxID)
		if err == nil {
			_ = network.DestroyTAP(ctx, info.TAPDevice)
		}
		if err := c.vmMgr.Destroy(ctx, sandboxID); err != nil {
			c.logger.Error("destroy microVM failed", "sandbox_id", sandboxID, "error", err)
		}
		microvm.RemoveOverlay(c.vmMgr.WorkDir(), sandboxID)
	}

	if err := c.localStore.DeleteSandbox(ctx, sandboxID); err != nil {
		c.logger.Error("delete local sandbox state failed", "sandbox_id", sandboxID, "error", err)
	}

	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_SandboxDestroyed{
			SandboxDestroyed: &fluidv1.SandboxDestroyed{
				SandboxId: sandboxID,
			},
		},
	}
}

func (c *Client) handleStartSandbox(ctx context.Context, reqID string, cmd *fluidv1.StartSandboxCommand) *fluidv1.HostMessage {
	sandboxID := cmd.GetSandboxId()

	// For microVMs, "start" means re-launching from the overlay.
	// The overlay preserves state. For now, report current state.
	if c.vmMgr == nil {
		return errorResponse(reqID, sandboxID, "microVM manager not available")
	}

	info, err := c.vmMgr.Get(sandboxID)
	if err != nil {
		return errorResponse(reqID, sandboxID, fmt.Sprintf("get sandbox: %v", err))
	}

	ip := ""
	if c.netMgr != nil {
		ip, _ = c.netMgr.DiscoverIP(ctx, info.MACAddress, info.Bridge, 30*time.Second)
	}

	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_SandboxStarted{
			SandboxStarted: &fluidv1.SandboxStarted{
				SandboxId: sandboxID,
				State:     "RUNNING",
				IpAddress: ip,
			},
		},
	}
}

func (c *Client) handleStopSandbox(ctx context.Context, reqID string, cmd *fluidv1.StopSandboxCommand) *fluidv1.HostMessage {
	sandboxID := cmd.GetSandboxId()

	if c.vmMgr == nil {
		return errorResponse(reqID, sandboxID, "microVM manager not available")
	}

	if err := c.vmMgr.Stop(ctx, sandboxID, cmd.GetForce()); err != nil {
		return errorResponse(reqID, sandboxID, fmt.Sprintf("stop: %v", err))
	}

	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_SandboxStopped{
			SandboxStopped: &fluidv1.SandboxStopped{
				SandboxId: sandboxID,
				State:     "STOPPED",
			},
		},
	}
}

func (c *Client) handleRunCommand(ctx context.Context, reqID string, cmd *fluidv1.RunCommandCommand) *fluidv1.HostMessage {
	sandboxID := cmd.GetSandboxId()
	command := cmd.GetCommand()

	c.logger.Info("running command", "sandbox_id", sandboxID, "command", command)

	if c.vmMgr == nil {
		return errorResponse(reqID, sandboxID, "microVM manager not available")
	}

	// Get sandbox IP for SSH
	info, err := c.vmMgr.Get(sandboxID)
	if err != nil {
		return errorResponse(reqID, sandboxID, fmt.Sprintf("get sandbox: %v", err))
	}

	ip := ""
	if c.netMgr != nil {
		ip, _ = c.netMgr.DiscoverIP(ctx, info.MACAddress, info.Bridge, 30*time.Second)
	}
	if ip == "" {
		return errorResponse(reqID, sandboxID, "unable to discover sandbox IP for SSH")
	}

	timeout := time.Duration(cmd.GetTimeoutSeconds()) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	startedAt := time.Now()

	// Execute command via SSH
	stdout, stderr, exitCode, err := c.runSSHCommand(ctx, ip, command, timeout)
	if err != nil {
		return errorResponse(reqID, sandboxID, fmt.Sprintf("run command: %v", err))
	}

	durationMs := time.Since(startedAt).Milliseconds()

	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_CommandResult{
			CommandResult: &fluidv1.CommandResult{
				SandboxId:  sandboxID,
				Stdout:     stdout,
				Stderr:     stderr,
				ExitCode:   int32(exitCode),
				DurationMs: durationMs,
			},
		},
	}
}

func (c *Client) handleCreateSnapshot(ctx context.Context, reqID string, cmd *fluidv1.SnapshotCommand) *fluidv1.HostMessage {
	sandboxID := cmd.GetSandboxId()
	name := cmd.GetSnapshotName()

	if c.vmMgr == nil {
		return errorResponse(reqID, sandboxID, "microVM manager not available")
	}

	// QEMU snapshots via qemu-img snapshot
	snapshotID := "SNP-" + uuid.New().String()[:8]

	// For now, return the snapshot info. Full qemu-img snapshot implementation
	// can be added when needed.
	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_SnapshotCreated{
			SnapshotCreated: &fluidv1.SnapshotCreated{
				SandboxId:    sandboxID,
				SnapshotId:   snapshotID,
				SnapshotName: name,
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Source VM command handlers
// ---------------------------------------------------------------------------

func (c *Client) handlePrepareSourceVM(ctx context.Context, reqID string, cmd *fluidv1.PrepareSourceVMCommand) *fluidv1.HostMessage {
	vmName := cmd.GetSourceVm()

	if c.srcVMMgr == nil {
		return errorResponse(reqID, "", "source VM manager not available")
	}

	result, err := c.srcVMMgr.PrepareSourceVM(ctx, vmName, cmd.GetSshUser(), cmd.GetSshKeyPath())
	if err != nil {
		return errorResponse(reqID, "", fmt.Sprintf("prepare source VM %s: %v", vmName, err))
	}

	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_SourceVmPrepared{
			SourceVmPrepared: &fluidv1.SourceVMPrepared{
				SourceVm:          result.SourceVM,
				IpAddress:         result.IPAddress,
				Prepared:          result.Prepared,
				UserCreated:       result.UserCreated,
				ShellInstalled:    result.ShellInstalled,
				CaKeyInstalled:    result.CAKeyInstalled,
				SshdConfigured:    result.SSHDConfigured,
				PrincipalsCreated: result.PrincipalsCreated,
				SshdRestarted:     result.SSHDRestarted,
			},
		},
	}
}

func (c *Client) handleRunSourceCommand(ctx context.Context, reqID string, cmd *fluidv1.RunSourceCommandCommand) *fluidv1.HostMessage {
	vmName := cmd.GetSourceVm()
	command := cmd.GetCommand()

	if c.srcVMMgr == nil {
		return errorResponse(reqID, "", "source VM manager not available")
	}

	timeout := time.Duration(cmd.GetTimeoutSeconds()) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	stdout, stderr, exitCode, err := c.srcVMMgr.RunSourceCommand(ctx, vmName, command, timeout)
	if err != nil {
		return errorResponse(reqID, "", fmt.Sprintf("run source command: %v", err))
	}

	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_SourceCommandResult{
			SourceCommandResult: &fluidv1.SourceCommandResult{
				SourceVm: vmName,
				ExitCode: int32(exitCode),
				Stdout:   stdout,
				Stderr:   stderr,
			},
		},
	}
}

func (c *Client) handleReadSourceFile(ctx context.Context, reqID string, cmd *fluidv1.ReadSourceFileCommand) *fluidv1.HostMessage {
	vmName := cmd.GetSourceVm()

	if c.srcVMMgr == nil {
		return errorResponse(reqID, "", "source VM manager not available")
	}

	content, err := c.srcVMMgr.ReadSourceFile(ctx, vmName, cmd.GetPath())
	if err != nil {
		return errorResponse(reqID, "", fmt.Sprintf("read source file: %v", err))
	}

	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_SourceFileResult{
			SourceFileResult: &fluidv1.SourceFileResult{
				SourceVm: vmName,
				Path:     cmd.GetPath(),
				Content:  content,
			},
		},
	}
}

func (c *Client) handleListSourceVMs(ctx context.Context, reqID string) *fluidv1.HostMessage {
	if c.srcVMMgr == nil {
		return errorResponse(reqID, "", "source VM manager not available")
	}

	vms, err := c.srcVMMgr.ListVMs(ctx)
	if err != nil {
		return errorResponse(reqID, "", fmt.Sprintf("list VMs: %v", err))
	}

	entries := make([]*fluidv1.SourceVMListEntry, len(vms))
	for i, vm := range vms {
		entries[i] = &fluidv1.SourceVMListEntry{
			Name:      vm.Name,
			State:     vm.State,
			IpAddress: vm.IPAddress,
			Prepared:  vm.Prepared,
		}
	}

	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_SourceVmsList{
			SourceVmsList: &fluidv1.SourceVMsList{
				Vms: entries,
			},
		},
	}
}

func (c *Client) handleValidateSourceVM(ctx context.Context, reqID string, cmd *fluidv1.ValidateSourceVMCommand) *fluidv1.HostMessage {
	vmName := cmd.GetSourceVm()

	if c.srcVMMgr == nil {
		return errorResponse(reqID, "", "source VM manager not available")
	}

	result, err := c.srcVMMgr.ValidateSourceVM(ctx, vmName)
	if err != nil {
		return errorResponse(reqID, "", fmt.Sprintf("validate source VM: %v", err))
	}

	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_SourceVmValidation{
			SourceVmValidation: &fluidv1.SourceVMValidation{
				SourceVm:   result.VMName,
				Valid:      result.Valid,
				State:      result.State,
				MacAddress: result.MACAddress,
				IpAddress:  result.IPAddress,
				HasNetwork: result.HasNetwork,
				Warnings:   result.Warnings,
				Errors:     result.Errors,
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// runSSHCommand executes a command on a sandbox via SSH.
func (c *Client) runSSHCommand(ctx context.Context, ip, command string, timeout time.Duration) (stdout, stderr string, exitCode int, err error) {
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	sshArgs := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		fmt.Sprintf("sandbox@%s", ip),
		command,
	}

	cmd2 := exec.CommandContext(cmdCtx, "ssh", sshArgs...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd2.Stdout = &stdoutBuf
	cmd2.Stderr = &stderrBuf

	err = cmd2.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return stdoutBuf.String(), stderrBuf.String(), exitErr.ExitCode(), nil
		}
		return "", "", -1, err
	}

	return stdoutBuf.String(), stderrBuf.String(), 0, nil
}

// errorResponse builds an ErrorReport HostMessage.
func errorResponse(reqID, sandboxID, errMsg string) *fluidv1.HostMessage {
	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_ErrorReport{
			ErrorReport: &fluidv1.ErrorReport{
				Error:     errMsg,
				SandboxId: sandboxID,
			},
		},
	}
}
