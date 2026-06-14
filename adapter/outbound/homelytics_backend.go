package outbound

import (
	"context"
	"time"

	"github.com/AndreeJait/go-utility/v2/statusw"
	"github.com/AndreeJait/homelytics-agent/config"
	"github.com/AndreeJait/homelytics-agent/domain/entity"
	portOutbound "github.com/AndreeJait/homelytics-agent/port/outbound"
)

const (
	mockEmail    = "merchant@example.com"
	mockPassword = "password"
	mockToken    = "mock-token-12345"
	mockAuthKey  = "tskey-auth-mock-abcde"
)

type mockHomelyticsBackend struct{}

// NewMockHomelyticsBackend returns an in-memory mock of the control plane.
func NewMockHomelyticsBackend() portOutbound.HomelyticsBackend {
	return &mockHomelyticsBackend{}
}

func (b *mockHomelyticsBackend) Login(_ context.Context, req entity.LoginRequest) (*entity.AuthSession, error) {
	if req.Email != mockEmail || req.Password != mockPassword {
		return nil, statusw.InvalidCredential.WithCustomMessage("invalid email or password")
	}
	return &entity.AuthSession{
		Token:     mockToken,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}, nil
}

func (b *mockHomelyticsBackend) GetTSNetAuthKey(_ context.Context, token string) (*entity.TSNetAuthKey, error) {
	if token != mockToken {
		return nil, statusw.InvalidCredential.WithCustomMessage("invalid session token")
	}
	return &entity.TSNetAuthKey{AuthKey: mockAuthKey}, nil
}

type httpHomelyticsBackend struct {
	cfg *config.AppConfig
}

// NewHTTPHomelyticsBackend creates a real-mode control-plane client stub.
func NewHTTPHomelyticsBackend(cfg *config.AppConfig) portOutbound.HomelyticsBackend {
	return &httpHomelyticsBackend{cfg: cfg}
}

func (b *httpHomelyticsBackend) Login(_ context.Context, _ entity.LoginRequest) (*entity.AuthSession, error) {
	return nil, statusw.InternalServerError.WithCustomMessage("real homelytics-be backend is not implemented yet")
}

func (b *httpHomelyticsBackend) GetTSNetAuthKey(_ context.Context, _ string) (*entity.TSNetAuthKey, error) {
	return nil, statusw.InternalServerError.WithCustomMessage("real homelytics-be backend is not implemented yet")
}
