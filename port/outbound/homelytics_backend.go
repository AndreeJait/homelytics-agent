package outbound

import (
	"context"

	"github.com/AndreeJait/homelytics-agent/domain/entity"
)

// HomelyticsBackend is the control-plane client.
type HomelyticsBackend interface {
	Login(ctx context.Context, req entity.LoginRequest) (*entity.AuthSession, error)
	GetTSNetAuthKey(ctx context.Context, token string) (*entity.TSNetAuthKey, error)
}
