package grpc

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"

	"github.com/aspectrr/fluid.sh/control-plane/internal/registry"
	"github.com/aspectrr/fluid.sh/control-plane/internal/store"
	"github.com/aspectrr/fluid.sh/control-plane/internal/store/postgres"
)

// StreamHandler implements fluidv1.HostServiceServer.
// It manages the bidirectional stream lifecycle for each connected host.
type StreamHandler struct {
	fluidv1.UnimplementedHostServiceServer

	registry *registry.Registry
	store    *postgres.Store
	logger   *slog.Logger

	// pendingRequests maps request_id to a channel that will receive the
	// host's response message. Used by SendAndWait to correlate
	// request/response pairs across the async stream.
	pendingRequests sync.Map // map[string]chan *fluidv1.HostMessage

	// streams maps host_id to the active server stream, allowing the
	// orchestrator to send commands to specific hosts.
	streams sync.Map // map[string]fluidv1.HostService_ConnectServer
}

// NewStreamHandler creates a stream handler wired to the given dependencies.
func NewStreamHandler(
	reg *registry.Registry,
	st *postgres.Store,
	logger *slog.Logger,
) *StreamHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &StreamHandler{
		registry: reg,
		store:    st,
		logger:   logger.With("component", "stream-handler"),
	}
}

// Connect handles a single bidirectional stream from a sandbox host.
// Protocol:
//  1. First message must be a HostRegistration.
//  2. Control plane sends a RegistrationAck.
//  3. Host is registered in the in-memory registry.
//  4. A heartbeat monitoring goroutine is spawned.
//  5. Main recv loop dispatches incoming HostMessages:
//     - Heartbeats update the registry and persistent store.
//     - All other response payloads are routed to waiting callers via
//     request_id correlation (pendingRequests).
//  6. On disconnect the host is unregistered.
func (h *StreamHandler) Connect(stream fluidv1.HostService_ConnectServer) error {
	// ---------------------------------------------------------------
	// Step 1: Receive first message - must be a HostRegistration.
	// ---------------------------------------------------------------
	firstMsg, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("recv registration: %w", err)
	}

	reg := firstMsg.GetRegistration()
	if reg == nil {
		return fmt.Errorf("first message must be HostRegistration")
	}

	hostID := reg.GetHostId()
	hostname := reg.GetHostname()

	logger := h.logger.With("host_id", hostID, "hostname", hostname)
	logger.Info("host connecting", "version", reg.GetVersion())

	// ---------------------------------------------------------------
	// Step 2: Send RegistrationAck.
	// ---------------------------------------------------------------
	ack := &fluidv1.ControlMessage{
		RequestId: firstMsg.GetRequestId(),
		Payload: &fluidv1.ControlMessage_RegistrationAck{
			RegistrationAck: &fluidv1.RegistrationAck{
				Accepted:       true,
				AssignedHostId: hostID,
			},
		},
	}
	if err := stream.Send(ack); err != nil {
		return fmt.Errorf("send registration ack: %w", err)
	}

	// ---------------------------------------------------------------
	// Step 3: Register host in registry and store the stream.
	// ---------------------------------------------------------------
	if err := h.registry.Register(hostID, hostname, stream); err != nil {
		return fmt.Errorf("register host: %w", err)
	}
	h.registry.SetRegistration(hostID, reg)
	h.streams.Store(hostID, stream)

	// Persist or update host in the database.
	h.persistHostRegistration(stream.Context(), hostID, reg)

	logger.Info("host registered",
		"total_cpus", reg.GetTotalCpus(),
		"total_memory_mb", reg.GetTotalMemoryMb(),
		"base_images", reg.GetBaseImages(),
	)

	// ---------------------------------------------------------------
	// Step 4: Spawn heartbeat monitoring goroutine.
	// ---------------------------------------------------------------
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	go h.monitorHeartbeat(ctx, hostID, logger)

	// ---------------------------------------------------------------
	// Step 5: Cleanup on disconnect.
	// ---------------------------------------------------------------
	defer func() {
		h.registry.Unregister(hostID)
		h.streams.Delete(hostID)
		logger.Info("host disconnected")
	}()

	// ---------------------------------------------------------------
	// Step 6: Main recv loop.
	// ---------------------------------------------------------------
	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				logger.Info("host stream closed by peer")
				return nil
			}
			logger.Error("stream recv error", "error", err)
			return err
		}

		h.handleHostMessage(ctx, hostID, msg, logger)
	}
}

// handleHostMessage dispatches an incoming HostMessage to the appropriate handler.
func (h *StreamHandler) handleHostMessage(ctx context.Context, hostID string, msg *fluidv1.HostMessage, logger *slog.Logger) {
	switch msg.Payload.(type) {
	case *fluidv1.HostMessage_Heartbeat:
		hb := msg.GetHeartbeat()
		h.registry.UpdateHeartbeat(hostID)
		_ = h.store.UpdateHostHeartbeat(
			ctx,
			hostID,
			hb.GetAvailableCpus(),
			hb.GetAvailableMemoryMb(),
			hb.GetAvailableDiskMb(),
		)

	case *fluidv1.HostMessage_ResourceReport:
		h.registry.UpdateHeartbeat(hostID)
		logger.Info("received resource report")

	case *fluidv1.HostMessage_ErrorReport:
		er := msg.GetErrorReport()
		logger.Error("host reported error",
			"sandbox_id", er.GetSandboxId(),
			"error", er.GetError(),
			"context", er.GetContext(),
		)

	default:
		// All other payload types are responses to pending requests.
		// Route them by request_id.
		reqID := msg.GetRequestId()
		if reqID == "" {
			logger.Warn("received message without request_id, dropping")
			return
		}
		if ch, ok := h.pendingRequests.LoadAndDelete(reqID); ok {
			respCh := ch.(chan *fluidv1.HostMessage)
			select {
			case respCh <- msg:
			default:
				logger.Warn("response channel full, dropping", "request_id", reqID)
			}
		} else {
			logger.Warn("no pending request for response", "request_id", reqID)
		}
	}
}

// SendAndWait sends a ControlMessage to a specific host and blocks until the
// host responds with a matching request_id or the timeout expires.
func (h *StreamHandler) SendAndWait(hostID string, msg *fluidv1.ControlMessage, timeout time.Duration) (*fluidv1.HostMessage, error) {
	streamVal, ok := h.streams.Load(hostID)
	if !ok {
		return nil, fmt.Errorf("host %s is not connected", hostID)
	}
	stream := streamVal.(fluidv1.HostService_ConnectServer)

	reqID := msg.GetRequestId()
	if reqID == "" {
		return nil, fmt.Errorf("control message must have a request_id")
	}

	// Create a buffered channel so the recv loop can deliver without blocking.
	respCh := make(chan *fluidv1.HostMessage, 1)
	h.pendingRequests.Store(reqID, respCh)
	defer h.pendingRequests.Delete(reqID)

	if err := stream.Send(msg); err != nil {
		return nil, fmt.Errorf("send to host %s: %w", hostID, err)
	}

	select {
	case resp := <-respCh:
		return resp, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for response from host %s (request_id=%s)", hostID, reqID)
	}
}

// GetStream returns the active stream for a host, if connected.
func (h *StreamHandler) GetStream(hostID string) (fluidv1.HostService_ConnectServer, bool) {
	v, ok := h.streams.Load(hostID)
	if !ok {
		return nil, false
	}
	return v.(fluidv1.HostService_ConnectServer), true
}

// monitorHeartbeat runs until the context is cancelled and logs when a host
// has not sent a heartbeat within the expected interval.
func (h *StreamHandler) monitorHeartbeat(ctx context.Context, hostID string, logger *slog.Logger) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			host, ok := h.registry.GetHost(hostID)
			if !ok {
				return
			}
			if time.Since(host.LastHeartbeat) > 90*time.Second {
				logger.Warn("host heartbeat overdue",
					"last_heartbeat", host.LastHeartbeat,
					"overdue_by", time.Since(host.LastHeartbeat)-90*time.Second,
				)
			}
		}
	}
}

// persistHostRegistration upserts the host record in PostgreSQL based on the
// registration data. Errors are logged but do not interrupt the stream.
func (h *StreamHandler) persistHostRegistration(ctx context.Context, hostID string, reg *fluidv1.HostRegistration) {
	existing, err := h.store.GetHost(ctx, hostID)
	if err != nil {
		// Host does not exist yet - create it.
		host := hostFromRegistration(hostID, reg)
		if createErr := h.store.CreateHost(ctx, host); createErr != nil {
			h.logger.Error("failed to create host in store", "host_id", hostID, "error", createErr)
		}
		return
	}

	// Update existing host record.
	existing.Hostname = reg.GetHostname()
	existing.Version = reg.GetVersion()
	existing.TotalCPUs = reg.GetTotalCpus()
	existing.TotalMemoryMB = reg.GetTotalMemoryMb()
	existing.TotalDiskMB = reg.GetTotalDiskMb()
	existing.AvailableCPUs = reg.GetAvailableCpus()
	existing.AvailableMemoryMB = reg.GetAvailableMemoryMb()
	existing.AvailableDiskMB = reg.GetAvailableDiskMb()
	existing.BaseImages = reg.GetBaseImages()
	existing.Status = store.HostStatusOnline
	existing.LastHeartbeat = time.Now()

	sourceVMs := make(store.SourceVMSlice, 0, len(reg.GetSourceVms()))
	for _, vm := range reg.GetSourceVms() {
		sourceVMs = append(sourceVMs, store.SourceVMJSON{
			Name:      vm.GetName(),
			State:     vm.GetState(),
			IPAddress: vm.GetIpAddress(),
			Prepared:  vm.GetPrepared(),
		})
	}
	existing.SourceVMs = sourceVMs

	bridges := make(store.BridgeSlice, 0, len(reg.GetBridges()))
	for _, b := range reg.GetBridges() {
		bridges = append(bridges, store.BridgeJSON{
			Name:   b.GetName(),
			Subnet: b.GetSubnet(),
		})
	}
	existing.Bridges = bridges

	if err := h.store.UpdateHost(ctx, existing); err != nil {
		h.logger.Error("failed to update host in store", "host_id", hostID, "error", err)
	}
}

// hostFromRegistration builds a store.Host from registration data.
func hostFromRegistration(hostID string, reg *fluidv1.HostRegistration) *store.Host {
	sourceVMs := make(store.SourceVMSlice, 0, len(reg.GetSourceVms()))
	for _, vm := range reg.GetSourceVms() {
		sourceVMs = append(sourceVMs, store.SourceVMJSON{
			Name:      vm.GetName(),
			State:     vm.GetState(),
			IPAddress: vm.GetIpAddress(),
			Prepared:  vm.GetPrepared(),
		})
	}

	bridges := make(store.BridgeSlice, 0, len(reg.GetBridges()))
	for _, b := range reg.GetBridges() {
		bridges = append(bridges, store.BridgeJSON{
			Name:   b.GetName(),
			Subnet: b.GetSubnet(),
		})
	}

	return &store.Host{
		ID:                hostID,
		Hostname:          reg.GetHostname(),
		Version:           reg.GetVersion(),
		TotalCPUs:         reg.GetTotalCpus(),
		TotalMemoryMB:     reg.GetTotalMemoryMb(),
		TotalDiskMB:       reg.GetTotalDiskMb(),
		AvailableCPUs:     reg.GetAvailableCpus(),
		AvailableMemoryMB: reg.GetAvailableMemoryMb(),
		AvailableDiskMB:   reg.GetAvailableDiskMb(),
		BaseImages:        reg.GetBaseImages(),
		SourceVMs:         sourceVMs,
		Bridges:           bridges,
		Status:            store.HostStatusOnline,
		LastHeartbeat:     time.Now(),
	}
}
