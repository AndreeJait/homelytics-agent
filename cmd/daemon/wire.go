package main

import (
	"context"

	"github.com/AndreeJait/homelytics-agent/adapter/inbound/ipc"
	"github.com/AndreeJait/homelytics-agent/config"
	"go.uber.org/dig"
)

// CleanupCollector accumulates cleanup functions from infrastructure providers.
type CleanupCollector struct {
	cleanups []func(ctx context.Context) error
}

// Add appends a cleanup function.
func (cc *CleanupCollector) Add(fn func(ctx context.Context) error) {
	cc.cleanups = append(cc.cleanups, fn)
}

// Cleanup runs all collected cleanup functions, returning the first error.
func (cc *CleanupCollector) Cleanup(ctx context.Context) error {
	var firstErr error
	for _, fn := range cc.cleanups {
		if err := fn(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// wire builds the dependency graph using dig and returns the IPC server + cleanup.
func wire(cfg *config.AppConfig) (*ipc.Server, func(ctx context.Context) error, error) {
	c := dig.New()

	c.Provide(func() *config.AppConfig { return cfg })
	c.Provide(func() *CleanupCollector { return &CleanupCollector{} })

	provideInfrastructure(c)
	provideServices(c)
	provideIPC(c)

	var server *ipc.Server
	if err := c.Invoke(func(s *ipc.Server) { server = s }); err != nil {
		return nil, nil, err
	}

	var cc *CleanupCollector
	if err := c.Invoke(func(collector *CleanupCollector) { cc = collector }); err != nil {
		return nil, nil, err
	}

	return server, cc.Cleanup, nil
}
