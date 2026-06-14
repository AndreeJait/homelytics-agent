package outbound

import (
	"context"

	"github.com/AndreeJait/homelytics-agent/domain/entity"
)

// SessionStore keeps the current merchant session and tsnet auth key in memory.
type SessionStore interface {
	SetSession(ctx context.Context, session *entity.AuthSession) error
	GetSession(ctx context.Context) (*entity.AuthSession, error)

	SetTSNetAuthKey(ctx context.Context, key *entity.TSNetAuthKey) error
	GetTSNetAuthKey(ctx context.Context) (*entity.TSNetAuthKey, error)
}
