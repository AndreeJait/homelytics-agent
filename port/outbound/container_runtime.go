package outbound

import (
	"context"

	"github.com/AndreeJait/homelytics-agent/domain/entity"
)

// ContainerRuntime abstracts the containerd client.
type ContainerRuntime interface {
	Status(ctx context.Context) (*entity.RuntimeStatus, error)
}
