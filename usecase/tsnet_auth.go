package usecase

import (
	"context"

	"github.com/AndreeJait/go-utility/v2/statusw"
	"github.com/AndreeJait/homelytics-agent/domain/entity"
	portInbound "github.com/AndreeJait/homelytics-agent/port/inbound/tsnet"
	portOutbound "github.com/AndreeJait/homelytics-agent/port/outbound"
)

type tsnetAuthUseCase struct {
	backend portOutbound.HomelyticsBackend
	store   portOutbound.SessionStore
}

// NewTSNetAuthUseCase creates a use case for retrieving a Tailscale auth key.
func NewTSNetAuthUseCase(backend portOutbound.HomelyticsBackend, store portOutbound.SessionStore) portInbound.UseCase {
	return &tsnetAuthUseCase{backend: backend, store: store}
}

// GetAuthKey fetches a tsnet auth key from the control plane for the current session.
func (u *tsnetAuthUseCase) GetAuthKey(ctx context.Context) (*entity.TSNetAuthKey, error) {
	session, err := u.store.GetSession(ctx)
	if err != nil {
		return nil, statusw.InvalidCredential.WithCustomMessage("not logged in")
	}

	key, err := u.backend.GetTSNetAuthKey(ctx, session.Token)
	if err != nil {
		return nil, err
	}

	if err := u.store.SetTSNetAuthKey(ctx, key); err != nil {
		return nil, err
	}

	return key, nil
}
