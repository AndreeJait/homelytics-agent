package main

import (
	"github.com/AndreeJait/homelytics-agent/config"
	"github.com/AndreeJait/homelytics-agent/port/outbound"
	"github.com/AndreeJait/homelytics-agent/usecase"
	authUC "github.com/AndreeJait/homelytics-agent/port/inbound/auth"
	runtimeUC "github.com/AndreeJait/homelytics-agent/port/inbound/runtime"
	statusUC "github.com/AndreeJait/homelytics-agent/port/inbound/status"
	tsnetUC "github.com/AndreeJait/homelytics-agent/port/inbound/tsnet"
	"go.uber.org/dig"
)

// provideServices registers repository and use case providers into the dig container.
func provideServices(c *dig.Container) {
	c.Provide(newAuthUseCase)
	c.Provide(newTSNetAuthUseCase)
	c.Provide(newRuntimeStatusUseCase)
	c.Provide(newAgentStatusUseCase)
}

func newAuthUseCase(backend outbound.HomelyticsBackend, store outbound.SessionStore) authUC.UseCase {
	return usecase.NewAuthUseCase(backend, store)
}

func newTSNetAuthUseCase(backend outbound.HomelyticsBackend, store outbound.SessionStore) tsnetUC.UseCase {
	return usecase.NewTSNetAuthUseCase(backend, store)
}

func newRuntimeStatusUseCase(runtime outbound.ContainerRuntime) runtimeUC.UseCase {
	return usecase.NewRuntimeStatusUseCase(runtime)
}

func newAgentStatusUseCase(cfg *config.AppConfig, store outbound.SessionStore, runtime outbound.ContainerRuntime) statusUC.UseCase {
	return usecase.NewAgentStatusUseCase(cfg.App.Name, store, runtime)
}
