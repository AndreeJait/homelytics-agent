package runtime

import (
	"context"

	"github.com/AndreeJait/homelytics-agent/domain/entity"
)

// UseCase reports container runtime connectivity.
type UseCase interface {
	Status(ctx context.Context) (*entity.RuntimeStatus, error)
}
