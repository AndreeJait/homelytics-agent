package status

import (
	"context"

	"github.com/AndreeJait/homelytics-agent/domain/entity"
)

// UseCase aggregates overall agent status.
type UseCase interface {
	Get(ctx context.Context) (*entity.AgentStatus, error)
}
