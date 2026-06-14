package outbound

import (
	"context"

	"github.com/AndreeJait/homelytics-agent/domain/entity"
)

// ContainerRuntime abstracts the containerd runtime for workload execution.
type ContainerRuntime interface {
	Status(ctx context.Context) (*entity.RuntimeStatus, error)
	PullImage(ctx context.Context, ref string) error
	CreateContainer(ctx context.Context, req entity.RunWorkloadRequest) (string, error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string) error
	DeleteContainer(ctx context.Context, id string) error
	ListContainers(ctx context.Context) ([]entity.Workload, error)
	ContainerStatus(ctx context.Context, id string) (*entity.Workload, error)
}
