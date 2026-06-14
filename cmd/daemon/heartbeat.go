package main

import (
	"context"
	"fmt"
	"time"

	"github.com/AndreeJait/go-utility/v2/logw"
	"github.com/AndreeJait/homelytics-agent/config"
	"github.com/AndreeJait/homelytics-agent/domain/entity"
	portOutbound "github.com/AndreeJait/homelytics-agent/port/outbound"
	"github.com/AndreeJait/homelytics-agent/usecase/command"
)

// HeartbeatLoop periodically reports agent state to the control plane and executes any returned commands.
type HeartbeatLoop struct {
	cfg      *config.AppConfig
	backend  portOutbound.HomelyticsBackend
	store    portOutbound.SessionStore
	runtime  portOutbound.ContainerRuntime
	vpn      portOutbound.VPN
	executor *command.Executor
	version  string
}

// NewHeartbeatLoop creates a heartbeat loop.
func NewHeartbeatLoop(
	cfg *config.AppConfig,
	backend portOutbound.HomelyticsBackend,
	store portOutbound.SessionStore,
	runtime portOutbound.ContainerRuntime,
	vpn portOutbound.VPN,
	executor *command.Executor,
) *HeartbeatLoop {
	return &HeartbeatLoop{
		cfg:      cfg,
		backend:  backend,
		store:    store,
		runtime:  runtime,
		vpn:      vpn,
		executor: executor,
		version:  "v0.2.0",
	}
}

// Start begins the heartbeat goroutine. The loop exits when ctx is cancelled.
func (h *HeartbeatLoop) Start(ctx context.Context) {
	if !h.cfg.Heartbeat.Enabled {
		return
	}

	go h.loop(ctx)
}

func (h *HeartbeatLoop) loop(ctx context.Context) {
	ticker := time.NewTicker(h.cfg.Heartbeat.Interval)
	defer ticker.Stop()

	// Send one heartbeat immediately on start.
	h.beat(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.beat(ctx)
		}
	}
}

func (h *HeartbeatLoop) beat(ctx context.Context) {
	session, err := h.store.GetSession(ctx)
	if err != nil {
		logw.CtxInfof(ctx, "heartbeat: no active session, skipping")
		return
	}

	heartbeat, err := h.buildHeartbeat(ctx)
	if err != nil {
		logw.CtxWarningf(ctx, "heartbeat: build heartbeat: %v", err)
		return
	}

	resp, err := h.backend.Heartbeat(ctx, session.AccessToken, heartbeat)
	if err != nil {
		logw.CtxWarningf(ctx, "heartbeat: backend call failed: %v", err)
		return
	}

	for _, cmd := range resp.Commands {
		logw.CtxInfof(ctx, "heartbeat: executing command %s/%s", cmd.ID, cmd.Type)
		if _, err := h.executor.Execute(ctx, cmd); err != nil {
			logw.CtxWarningf(ctx, "heartbeat: execute command %s failed: %v", cmd.ID, err)
		}
	}
}

func (h *HeartbeatLoop) buildHeartbeat(ctx context.Context) (entity.AgentHeartbeat, error) {
	hb := entity.AgentHeartbeat{
		AgentID:   h.cfg.TSNet.Hostname,
		Hostname:  h.cfg.TSNet.Hostname,
		Version:   h.version,
		Timestamp: time.Now().UTC(),
	}

	if runtimeStatus, err := h.runtime.Status(ctx); err == nil && runtimeStatus != nil {
		hb.Runtime = *runtimeStatus
	}

	if connected, err := h.vpn.Status(ctx); err == nil {
		hb.TSNetIP = h.tsnetIP(ctx, connected)
	}

	workloads, err := h.runtime.ListContainers(ctx)
	if err == nil {
		hb.Workloads = make([]entity.WorkloadHeartbeat, 0, len(workloads))
		for _, w := range workloads {
			hb.Workloads = append(hb.Workloads, entity.WorkloadHeartbeat{
				ID:     w.ID,
				Status: w.Status,
				Image:  w.Image,
			})
		}
	}

	return hb, nil
}

func (h *HeartbeatLoop) tsnetIP(ctx context.Context, connected bool) string {
	if !connected {
		return ""
	}
	// The backend knows the agent by hostname; we do not need the exact Tailscale IP here.
	// Returning a placeholder keeps the schema intact.
	return fmt.Sprintf("%s.tailnet", h.cfg.TSNet.Hostname)
}
