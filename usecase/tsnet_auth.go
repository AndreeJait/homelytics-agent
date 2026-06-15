package usecase

import (
	"context"
	"fmt"

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

	hostname := u.resolveHostname(ctx, session)
	if err := u.store.SetHostname(ctx, hostname); err != nil {
		return nil, err
	}

	if err := u.vpn.Start(ctx, key.AuthKey, hostname); err != nil {
		logw.CtxErrorf(ctx, "tsnet start failed: %v", err)
		return nil, err
	}

	logw.CtxInfof(ctx, "tsnet started with auth key as %s", hostname)
	return key, nil
}

func (u *tsnetAuthUseCase) resolveHostname(ctx context.Context, session *entity.AuthSession) string {
	if session.MerchantID != "" {
		return fmt.Sprintf("homelytics-agent-%s", sanitizeHostname(session.MerchantID))
	}
	if existing, err := u.store.GetHostname(ctx); err == nil && existing != "" {
		return existing
	}
	return "homelytics-agent"
}

func sanitizeHostname(id string) string {
	out := make([]byte, 0, len(id))
	for i := 0; i < len(id); i++ {
		c := id[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			out = append(out, c)
		} else if c >= 'A' && c <= 'Z' {
			out = append(out, c+('a'-'A'))
		} else {
			out = append(out, '-')
		}
	}
	return string(out)
}
