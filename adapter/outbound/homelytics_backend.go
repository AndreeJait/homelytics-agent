package outbound

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
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

type mockHomelyticsBackend struct {
	cfg        *config.AppConfig
	cachedKey  string
	cachedTime time.Time
}

// NewMockHomelyticsBackend returns an in-memory mock of the control plane.
func NewMockHomelyticsBackend(cfg *config.AppConfig) portOutbound.HomelyticsBackend {
	return &mockHomelyticsBackend{cfg: cfg}
}

func (b *mockHomelyticsBackend) Login(_ context.Context, req entity.LoginRequest) (*entity.AuthSession, error) {
	if req.Email != mockEmail || req.Password != mockPassword {
		return nil, statusw.InvalidCredential.WithCustomMessage("invalid email or password")
	}
	return &entity.AuthSession{
		AccessToken:  mockToken,
		RefreshToken: mockToken,
		TokenType:    "Bearer",
		ExpiresIn:    900,
		MerchantID:   mockMerchantID(mockEmail),
		ExpiresAt:    time.Now().UTC().Add(24 * time.Hour),
	}, nil
}

func (b *mockHomelyticsBackend) GetTSNetAuthKey(_ context.Context, token string) (*entity.TSNetAuthKey, error) {
	if token != mockToken {
		return nil, statusw.InvalidCredential.WithCustomMessage("invalid session token")
	}

	// Allow a hardcoded fallback key from config for offline testing.
	if b.cfg.Homelytics.MockTSNetAuthKey != "" {
		return &entity.TSNetAuthKey{AuthKey: b.cfg.Homelytics.MockTSNetAuthKey}, nil
	}

	key, err := b.fetchTailscaleAuthKey()
	if err != nil {
		return nil, err
	}

	return &entity.TSNetAuthKey{AuthKey: key}, nil
}

func (b *mockHomelyticsBackend) fetchTailscaleAuthKey() (string, error) {
	tailnet := os.Getenv("TAILSCALE_TAILNET")
	apiKey := os.Getenv("TAILSCALE_API_KEY")

	if tailnet == "" || apiKey == "" {
		return "", statusw.InvalidReqParam.WithCustomMessage("TAILSCALE_TAILNET and TAILSCALE_API_KEY must be set, or homelytics.mock_tsnet_auth_key must be configured")
	}

	// Cache a generated key for 5 minutes to avoid hitting API on every tsnet.auth call.
	if b.cachedKey != "" && time.Since(b.cachedTime) < 5*time.Minute {
		return b.cachedKey, nil
	}

	url := fmt.Sprintf("https://api.tailscale.com/api/v2/tailnet/%s/keys", tailnet)
	body, _ := json.Marshal(map[string]any{
		"capabilities": map[string]any{
			"devices": map[string]any{
				"create": map[string]any{
					"reusable":      false,
					"ephemeral":     false,
					"preauthorized": true,
					"tags":          []string{},
				},
			},
		},
	})

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", statusw.InternalServerError.WithCustomMessage(fmt.Sprintf("build tailscale request: %v", err))
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", statusw.InternalServerError.WithCustomMessage(fmt.Sprintf("tailscale api request: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", statusw.InternalServerError.WithCustomMessage(fmt.Sprintf("tailscale api returned %s", resp.Status))
	}

	var result struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", statusw.InternalServerError.WithCustomMessage(fmt.Sprintf("decode tailscale response: %v", err))
	}

	if result.Key == "" {
		return "", statusw.InternalServerError.WithCustomMessage("tailscale api returned empty key")
	}

	b.cachedKey = result.Key
	b.cachedTime = time.Now().UTC()
	return result.Key, nil
}

func (b *mockHomelyticsBackend) RefreshToken(_ context.Context, req entity.RefreshTokenRequest) (*entity.AuthSession, error) {
	if req.RefreshToken != mockToken {
		return nil, statusw.InvalidCredential.WithCustomMessage("invalid refresh token")
	}
	return &entity.AuthSession{
		AccessToken:  mockToken,
		RefreshToken: mockToken,
		TokenType:    "Bearer",
		ExpiresIn:    900,
		MerchantID:   mockMerchantID(mockEmail),
		ExpiresAt:    time.Now().UTC().Add(24 * time.Hour),
	}, nil
}

func (b *mockHomelyticsBackend) Heartbeat(_ context.Context, _ string, _ entity.AgentHeartbeat) (*entity.HeartbeatResponse, error) {
	return &entity.HeartbeatResponse{Commands: []entity.Command{}}, nil
}

func mockMerchantID(email string) string {
	if id := os.Getenv("HOMELYTICS_MOCK_MERCHANT_ID"); id != "" {
		return id
	}
	// Deterministic, safe merchant ID derived from the email local-part.
	local := email
	if at := strings.Index(email, "@"); at >= 0 {
		local = email[:at]
	}
	return "merchant-" + strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		return '-'
	}, local)
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

func (b *httpHomelyticsBackend) RefreshToken(_ context.Context, _ entity.RefreshTokenRequest) (*entity.AuthSession, error) {
	return nil, statusw.InternalServerError.WithCustomMessage("real homelytics-be backend is not implemented yet")
}

func (b *httpHomelyticsBackend) Heartbeat(_ context.Context, _ string, _ entity.AgentHeartbeat) (*entity.HeartbeatResponse, error) {
	return nil, statusw.InternalServerError.WithCustomMessage("real homelytics-be backend is not implemented yet")
}
