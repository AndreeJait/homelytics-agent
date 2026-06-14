package outbound

import (
	"context"

	"github.com/AndreeJait/go-utility/v2/tailscalew/tsnetw"
	"github.com/AndreeJait/homelytics-agent/config"
	portOutbound "github.com/AndreeJait/homelytics-agent/port/outbound"
)

type tsnetVPN struct {
	vpn tsnetw.VPN
}

// NewTSNetVPN wraps tsnetw into the domain VPN port.
func NewTSNetVPN(cfg *config.AppConfig) (portOutbound.VPN, func() error, error) {
	vpn, err := tsnetw.New(&tsnetw.Config{
		Hostname:      cfg.TSNet.Hostname,
		ControlURL:    cfg.TSNet.ControlURL,
		AdvertiseTags: cfg.TSNet.AdvertiseTags,
		Dir:           cfg.TSNet.Dir,
	})
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() error { return vpn.Close() }
	return &tsnetVPN{vpn: vpn}, cleanup, nil
}

func (v *tsnetVPN) Start(ctx context.Context) error {
	_, err := v.vpn.Start(ctx)
	return err
}

func (v *tsnetVPN) Status(ctx context.Context) (bool, error) {
	status, err := v.vpn.Status(ctx)
	if err != nil {
		return false, err
	}
	return status != nil && status.BackendState == "Running", nil
}

func (v *tsnetVPN) Close() error {
	return v.vpn.Close()
}
