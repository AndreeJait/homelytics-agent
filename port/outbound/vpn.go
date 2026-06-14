package outbound

import (
	"context"
	"net"
)

// VPN abstracts the embedded tsnet node.
type VPN interface {
	Start(ctx context.Context, authKey string) error
	Status(ctx context.Context) (connected bool, err error)
	Listen(network, addr string) (net.Listener, error)
	Close() error
}
