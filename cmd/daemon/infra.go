package main

import (
	"context"
	"fmt"

	"github.com/AndreeJait/go-utility/v2/logw"
	"github.com/AndreeJait/homelytics-agent/adapter/outbound"
	"github.com/AndreeJait/homelytics-agent/config"
	"github.com/AndreeJait/homelytics-agent/domain/entity"
	portOutbound "github.com/AndreeJait/homelytics-agent/port/outbound"
	"go.uber.org/dig"
)

// provideInfrastructure registers infrastructure providers into the dig container.
func provideInfrastructure(c *dig.Container) {
	c.Provide(newContainerRuntime)
	c.Provide(newVPN)
	c.Provide(newHomelyticsBackend)
	c.Provide(newSessionStore)
}

func newContainerRuntime(cfg *config.AppConfig, cc *CleanupCollector) (portOutbound.ContainerRuntime, error) {
	runtime, cleanup, err := outbound.NewContainerdRuntime(cfg)
	if err != nil {
		logw.CtxWarningf(context.Background(), "containerd not available: %v", err)
		// Return a no-op runtime so the daemon can still start without containerd.
		return &unavailableContainerRuntime{err: err}, nil
	}
	cc.Add(func(_ context.Context) error { return cleanup() })
	return runtime, nil
}

func newVPN(cfg *config.AppConfig, cc *CleanupCollector) (portOutbound.VPN, error) {
	vpn, cleanup, err := outbound.NewTSNetVPN(cfg)
	if err != nil {
		return nil, err
	}
	cc.Add(func(_ context.Context) error { return cleanup() })
	return vpn, nil
}

func newHomelyticsBackend(cfg *config.AppConfig) portOutbound.HomelyticsBackend {
	if cfg.Homelytics.MockMode {
		return outbound.NewMockHomelyticsBackend(cfg)
	}
	return outbound.NewHTTPHomelyticsBackend(cfg)
}

func newSessionStore() portOutbound.SessionStore {
	return outbound.NewMemorySessionStore()
}

type unavailableContainerRuntime struct {
	err error
}

func (r *unavailableContainerRuntime) Status(_ context.Context) (*entity.RuntimeStatus, error) {
	return &entity.RuntimeStatus{Connected: false, Error: r.err.Error()}, nil
}

func (r *unavailableContainerRuntime) PullImage(_ context.Context, _ string) error {
	return fmt.Errorf("containerd unavailable: %w", r.err)
}

func (r *unavailableContainerRuntime) CreateContainer(_ context.Context, _ entity.RunWorkloadRequest) (string, error) {
	return "", fmt.Errorf("containerd unavailable: %w", r.err)
}

func (r *unavailableContainerRuntime) StartContainer(_ context.Context, _ string) error {
	return fmt.Errorf("containerd unavailable: %w", r.err)
}

func (r *unavailableContainerRuntime) StopContainer(_ context.Context, _ string) error {
	return fmt.Errorf("containerd unavailable: %w", r.err)
}

func (r *unavailableContainerRuntime) DeleteContainer(_ context.Context, _ string) error {
	return fmt.Errorf("containerd unavailable: %w", r.err)
}

func (r *unavailableContainerRuntime) ListContainers(_ context.Context) ([]entity.Workload, error) {
	return nil, fmt.Errorf("containerd unavailable: %w", r.err)
}

func (r *unavailableContainerRuntime) ContainerStatus(_ context.Context, _ string) (*entity.Workload, error) {
	return nil, fmt.Errorf("containerd unavailable: %w", r.err)
}
