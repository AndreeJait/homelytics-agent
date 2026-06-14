package usecase

import (
	"context"

	"github.com/AndreeJait/homelytics-agent/domain/entity"
	portInbound "github.com/AndreeJait/homelytics-agent/port/inbound/status"
	portOutbound "github.com/AndreeJait/homelytics-agent/port/outbound"
)

type agentStatusUseCase struct {
	serviceName    string
	store          portOutbound.SessionStore
	runtime        portOutbound.ContainerRuntime
}

// NewAgentStatusUseCase creates a use case that aggregates overall agent status.
func NewAgentStatusUseCase(serviceName string, store portOutbound.SessionStore, runtime portOutbound.ContainerRuntime) portInbound.UseCase {
	return &agentStatusUseCase{serviceName: serviceName, store: store, runtime: runtime}
}

// Get returns the current agent status.
func (u *agentStatusUseCase) Get(ctx context.Context) (*entity.AgentStatus, error) {
	status := &entity.AgentStatus{ServiceName: u.serviceName}

	if session, err := u.store.GetSession(ctx); err == nil && session != nil {
		status.LoggedIn = true
	}

	if key, err := u.store.GetTSNetAuthKey(ctx); err == nil && key != nil {
		status.TSNetAuthKeyPresent = true
	}

	if runtimeStatus, err := u.runtime.Status(ctx); err == nil && runtimeStatus != nil {
		status.RuntimeConnected = runtimeStatus.Connected
		status.RuntimeVersion = runtimeStatus.Version
	}

	return status, nil
}
