package usecase

import (
	"context"

	"github.com/AndreeJait/homelytics-agent/domain/entity"
	portInbound "github.com/AndreeJait/homelytics-agent/port/inbound/workload"
	portOutbound "github.com/AndreeJait/homelytics-agent/port/outbound"
)

type workloadUseCase struct {
	runtime portOutbound.ContainerRuntime
}

// NewWorkloadRunUseCase creates a use case for deploying workloads.
func NewWorkloadRunUseCase(runtime portOutbound.ContainerRuntime) portInbound.RunUseCase {
	return &workloadUseCase{runtime: runtime}
}

// NewWorkloadStopUseCase creates a use case for stopping workloads.
func NewWorkloadStopUseCase(runtime portOutbound.ContainerRuntime) portInbound.StopUseCase {
	return &workloadUseCase{runtime: runtime}
}

// NewWorkloadDeleteUseCase creates a use case for deleting workloads.
func NewWorkloadDeleteUseCase(runtime portOutbound.ContainerRuntime) portInbound.DeleteUseCase {
	return &workloadUseCase{runtime: runtime}
}

// NewWorkloadListUseCase creates a use case for listing workloads.
func NewWorkloadListUseCase(runtime portOutbound.ContainerRuntime) portInbound.ListUseCase {
	return &workloadUseCase{runtime: runtime}
}

// NewWorkloadStatusUseCase creates a use case for workload status.
func NewWorkloadStatusUseCase(runtime portOutbound.ContainerRuntime) portInbound.StatusUseCase {
	return &workloadUseCase{runtime: runtime}
}

func (u *workloadUseCase) Run(ctx context.Context, req entity.RunWorkloadRequest) (*entity.Workload, error) {
	id, err := u.runtime.CreateContainer(ctx, req)
	if err != nil {
		return nil, err
	}

	if err := u.runtime.StartContainer(ctx, id); err != nil {
		_ = u.runtime.DeleteContainer(ctx, id)
		return nil, err
	}

	return u.runtime.ContainerStatus(ctx, id)
}

func (u *workloadUseCase) Stop(ctx context.Context, id string) (*entity.Workload, error) {
	w, err := u.runtime.ContainerStatus(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := u.runtime.StopContainer(ctx, id); err != nil {
		return nil, err
	}

	w.Status = "stopped"
	return w, nil
}

func (u *workloadUseCase) Delete(ctx context.Context, id string) (*entity.Workload, error) {
	w, err := u.runtime.ContainerStatus(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := u.runtime.DeleteContainer(ctx, id); err != nil {
		return nil, err
	}

	w.Status = "deleted"
	return w, nil
}

func (u *workloadUseCase) List(ctx context.Context) (*entity.WorkloadList, error) {
	workloads, err := u.runtime.ListContainers(ctx)
	if err != nil {
		return nil, err
	}
	return &entity.WorkloadList{Workloads: workloads}, nil
}

func (u *workloadUseCase) Status(ctx context.Context, id string) (*entity.Workload, error) {
	return u.runtime.ContainerStatus(ctx, id)
}
