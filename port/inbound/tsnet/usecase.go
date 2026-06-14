package tsnet

import (
	"context"

	"github.com/AndreeJait/homelytics-agent/domain/entity"
)

// UseCase retrieves a Tailscale auth key from the control plane.
type UseCase interface {
	GetAuthKey(ctx context.Context) (*entity.TSNetAuthKey, error)
}
