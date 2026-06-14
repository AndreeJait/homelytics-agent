package outbound

import (
	"context"

	"github.com/AndreeJait/go-utility/v2/containerdw"
	"github.com/AndreeJait/homelytics-agent/config"
	"github.com/AndreeJait/homelytics-agent/domain/entity"
	portOutbound "github.com/AndreeJait/homelytics-agent/port/outbound"
)

type containerdRuntime struct {
	client containerdw.Containerd
}

// NewContainerdRuntime wraps containerdw into the domain container runtime port.
func NewContainerdRuntime(cfg *config.AppConfig) (portOutbound.ContainerRuntime, func() error, error) {
	client, err := containerdw.New(&containerdw.Config{
		Address:   cfg.Containerd.Address,
		Namespace: cfg.Containerd.Namespace,
		Timeout:   cfg.Containerd.Timeout,
	})
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() error { return client.Close() }
	return &containerdRuntime{client: client}, cleanup, nil
}

func (r *containerdRuntime) Status(ctx context.Context) (*entity.RuntimeStatus, error) {
	version, err := r.client.Version(ctx)
	if err != nil {
		return &entity.RuntimeStatus{Connected: false, Error: err.Error()}, nil
	}

	return &entity.RuntimeStatus{
		Connected: true,
		Version:   version.Version,
		Revision:  version.Revision,
	}, nil
}
