package workload

import (
	"context"

	"github.com/AndreeJait/homelytics-agent/domain/entity"
)

// RunUseCase deploys a new workload.
type RunUseCase interface {
	Run(ctx context.Context, req entity.RunWorkloadRequest) (*entity.Workload, error)
}

// StopUseCase stops a running workload.
type StopUseCase interface {
	Stop(ctx context.Context, id string) (*entity.Workload, error)
}

// DeleteUseCase removes a workload.
type DeleteUseCase interface {
	Delete(ctx context.Context, id string) (*entity.Workload, error)
}

// ListUseCase lists all workloads.
type ListUseCase interface {
	List(ctx context.Context) (*entity.WorkloadList, error)
}

// StatusUseCase returns the status of one workload.
type StatusUseCase interface {
	Status(ctx context.Context, id string) (*entity.Workload, error)
}
