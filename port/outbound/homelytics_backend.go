package outbound

import (
	"context"

	"github.com/AndreeJait/homelytics-agent/domain/entity"
)

// HomelyticsBackend is the control-plane client.
type HomelyticsBackend interface {
	Login(ctx context.Context, req entity.LoginRequest) (*entity.AuthSession, error)
	RefreshToken(ctx context.Context, req entity.RefreshTokenRequest) (*entity.AuthSession, error)
	GetTSNetAuthKey(ctx context.Context, token string) (*entity.TSNetAuthKey, error)
	Heartbeat(ctx context.Context, token string, heartbeat entity.AgentHeartbeat) (*entity.HeartbeatResponse, error)
}
