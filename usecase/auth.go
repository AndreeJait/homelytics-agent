package usecase

import (
	"context"

	"github.com/AndreeJait/homelytics-agent/domain/entity"
	portInbound "github.com/AndreeJait/homelytics-agent/port/inbound/auth"
	portOutbound "github.com/AndreeJait/homelytics-agent/port/outbound"
)

type authUseCase struct {
	backend portOutbound.HomelyticsBackend
	store   portOutbound.SessionStore
}

// NewAuthUseCase creates a login use case.
func NewAuthUseCase(backend portOutbound.HomelyticsBackend, store portOutbound.SessionStore) portInbound.UseCase {
	return &authUseCase{backend: backend, store: store}
}

// Login authenticates against the control plane and stores the returned session.
func (u *authUseCase) Login(ctx context.Context, req entity.LoginRequest) (*entity.AuthSession, error) {
	session, err := u.backend.Login(ctx, req)
	if err != nil {
		return nil, err
	}

	if err := u.store.SetSession(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}
