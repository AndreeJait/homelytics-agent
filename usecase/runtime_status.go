package usecase

import (
	"context"

	"github.com/AndreeJait/homelytics-agent/domain/entity"
	portInbound "github.com/AndreeJait/homelytics-agent/port/inbound/runtime"
	portOutbound "github.com/AndreeJait/homelytics-agent/port/outbound"
)

type runtimeStatusUseCase struct {
	runtime portOutbound.ContainerRuntime
}

// NewRuntimeStatusUseCase creates a use case that reports container runtime status.
func NewRuntimeStatusUseCase(runtime portOutbound.ContainerRuntime) portInbound.UseCase {
	return &runtimeStatusUseCase{runtime: runtime}
}

// Status returns the current containerd runtime status.
func (u *runtimeStatusUseCase) Status(ctx context.Context) (*entity.RuntimeStatus, error) {
	return u.runtime.Status(ctx)
}
