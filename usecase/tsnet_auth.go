package usecase

import (
	"context"

	"github.com/AndreeJait/go-utility/v2/logw"
	"github.com/AndreeJait/go-utility/v2/statusw"
	"github.com/AndreeJait/homelytics-agent/domain/entity"
	portInbound "github.com/AndreeJait/homelytics-agent/port/inbound/tsnet"
	portOutbound "github.com/AndreeJait/homelytics-agent/port/outbound"
)

type tsnetAuthUseCase struct {
	backend portOutbound.HomelyticsBackend
	store   portOutbound.SessionStore
	vpn     portOutbound.VPN
}

// NewTSNetAuthUseCase creates a use case for retrieving a Tailscale auth key.
func NewTSNetAuthUseCase(backend portOutbound.HomelyticsBackend, store portOutbound.SessionStore, vpn portOutbound.VPN) portInbound.UseCase {
	return &tsnetAuthUseCase{backend: backend, store: store, vpn: vpn}
}

// GetAuthKey fetches a tsnet auth key from the control plane for the current session and joins the tailnet.
func (u *tsnetAuthUseCase) GetAuthKey(ctx context.Context) (*entity.TSNetAuthKey, error) {
	session, err := u.store.GetSession(ctx)
	if err != nil {
		return nil, statusw.InvalidCredential.WithCustomMessage("not logged in")
	}

	key, err := u.backend.GetTSNetAuthKey(ctx, session.AccessToken)
	if err != nil {
		return nil, err
	}

	if err := u.store.SetTSNetAuthKey(ctx, key); err != nil {
		return nil, err
	}

	if err := u.vpn.Start(ctx, key.AuthKey); err != nil {
		logw.CtxErrorf(ctx, "tsnet start failed: %v", err)
		return nil, err
	}

	logw.CtxInfof(ctx, "tsnet started with auth key")
	return key, nil
}
