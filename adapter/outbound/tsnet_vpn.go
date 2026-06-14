package outbound

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/AndreeJait/homelytics-agent/config"
	portOutbound "github.com/AndreeJait/homelytics-agent/port/outbound"
	"tailscale.com/tsnet"
)

type tsnetVPN struct {
	cfg      *config.AppConfig
	server   *tsnet.Server
	mu       sync.Mutex
	started  bool
	authKey  string
}

// NewTSNetVPN wraps tsnetw into the domain VPN port.
// The underlying tsnet.Server is created lazily on Start with the auth key.
func NewTSNetVPN(cfg *config.AppConfig) (portOutbound.VPN, func() error, error) {
	cleanup := func() error { return nil }
	return &tsnetVPN{cfg: cfg}, cleanup, nil
}

func (v *tsnetVPN) Start(ctx context.Context, authKey string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.started {
		return nil
	}

	v.authKey = authKey
	v.server = v.buildServer()

	if err := v.server.Start(); err != nil {
		return fmt.Errorf("tsnet start: %w", err)
	}

	if _, err := v.server.Up(ctx); err != nil {
		_ = v.server.Close()
		v.server = nil
		return fmt.Errorf("tsnet up: %w", err)
	}

	v.started = true
	return nil
}

func (v *tsnetVPN) buildServer() *tsnet.Server {
	dir := v.cfg.TSNet.Dir
	if dir == "" {
		dir = defaultTSNetStateDir(v.cfg.TSNet.Hostname)
	}

	_ = os.MkdirAll(dir, 0o700)

	return &tsnet.Server{
		Hostname:      v.cfg.TSNet.Hostname,
		Dir:           dir,
		AuthKey:       v.authKey,
		ControlURL:    v.cfg.TSNet.ControlURL,
		AdvertiseTags: v.cfg.TSNet.AdvertiseTags,
		Logf:          func(string, ...any) {},
		UserLogf:      func(string, ...any) {},
	}
}

func defaultTSNetStateDir(hostname string) string {
	return filepath.Join("/opt", "homelytics", "lib", "tsnet", hostname)
}

func (v *tsnetVPN) Status(ctx context.Context) (bool, error) {
	v.mu.Lock()
	server := v.server
	v.mu.Unlock()

	if server == nil {
		return false, nil
	}

	lc, err := server.LocalClient()
	if err != nil {
		return false, err
	}

	status, err := lc.Status(ctx)
	if err != nil {
		return false, err
	}

	return status != nil && status.BackendState == "Running", nil
}

func (v *tsnetVPN) Listen(network, addr string) (net.Listener, error) {
	v.mu.Lock()
	server := v.server
	v.mu.Unlock()

	if server == nil {
		return nil, fmt.Errorf("tsnet not started")
	}
	return server.Listen(network, addr)
}

func (v *tsnetVPN) Close() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.server != nil {
		return v.server.Close()
	}
	return nil
}
