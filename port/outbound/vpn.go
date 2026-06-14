package outbound

import "context"

// VPN abstracts the embedded tsnet node.
type VPN interface {
	Start(ctx context.Context) error
	Status(ctx context.Context) (connected bool, err error)
	Close() error
}
