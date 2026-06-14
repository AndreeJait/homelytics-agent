package outbound

import (
	"context"

	"github.com/AndreeJait/homelytics-agent/domain/entity"
)

// SessionStore keeps the current merchant session, tsnet auth key, and hostname in memory.
type SessionStore interface {
	SetSession(ctx context.Context, session *entity.AuthSession) error
	GetSession(ctx context.Context) (*entity.AuthSession, error)

	SetTSNetAuthKey(ctx context.Context, key *entity.TSNetAuthKey) error
	GetTSNetAuthKey(ctx context.Context) (*entity.TSNetAuthKey, error)

	SetHostname(ctx context.Context, hostname string) error
	GetHostname(ctx context.Context) (string, error)
}
