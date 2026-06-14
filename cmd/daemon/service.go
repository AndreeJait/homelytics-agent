package main

import (
	"github.com/AndreeJait/homelytics-agent/config"
	"github.com/AndreeJait/homelytics-agent/port/outbound"
	"github.com/AndreeJait/homelytics-agent/usecase"
	authUC "github.com/AndreeJait/homelytics-agent/port/inbound/auth"
	runtimeUC "github.com/AndreeJait/homelytics-agent/port/inbound/runtime"
	statusUC "github.com/AndreeJait/homelytics-agent/port/inbound/status"
	tsnetUC "github.com/AndreeJait/homelytics-agent/port/inbound/tsnet"
	workloadUC "github.com/AndreeJait/homelytics-agent/port/inbound/workload"
	"go.uber.org/dig"
)

// provideServices registers repository and use case providers into the dig container.
func provideServices(c *dig.Container) {
	c.Provide(newAuthUseCase)
	c.Provide(newTSNetAuthUseCase)
	c.Provide(newRuntimeStatusUseCase)
	c.Provide(newAgentStatusUseCase)
	c.Provide(newWorkloadRunUseCase)
	c.Provide(newWorkloadStopUseCase)
	c.Provide(newWorkloadDeleteUseCase)
	c.Provide(newWorkloadListUseCase)
	c.Provide(newWorkloadStatusUseCase)
}

func newAuthUseCase(backend outbound.HomelyticsBackend, store outbound.SessionStore) authUC.UseCase {
	return usecase.NewAuthUseCase(backend, store)
}

func newTSNetAuthUseCase(backend outbound.HomelyticsBackend, store outbound.SessionStore, vpn outbound.VPN) tsnetUC.UseCase {
	return usecase.NewTSNetAuthUseCase(backend, store, vpn)
}

func newRuntimeStatusUseCase(runtime outbound.ContainerRuntime) runtimeUC.UseCase {
	return usecase.NewRuntimeStatusUseCase(runtime)
}

func newAgentStatusUseCase(cfg *config.AppConfig, store outbound.SessionStore, runtime outbound.ContainerRuntime, vpn outbound.VPN) statusUC.UseCase {
	return usecase.NewAgentStatusUseCase(cfg.App.Name, store, runtime, vpn)
}

func newWorkloadRunUseCase(runtime outbound.ContainerRuntime) workloadUC.RunUseCase {
	return usecase.NewWorkloadRunUseCase(runtime)
}

func newWorkloadStopUseCase(runtime outbound.ContainerRuntime) workloadUC.StopUseCase {
	return usecase.NewWorkloadStopUseCase(runtime)
}

func newWorkloadDeleteUseCase(runtime outbound.ContainerRuntime) workloadUC.DeleteUseCase {
	return usecase.NewWorkloadDeleteUseCase(runtime)
}

func newWorkloadListUseCase(runtime outbound.ContainerRuntime) workloadUC.ListUseCase {
	return usecase.NewWorkloadListUseCase(runtime)
}

func newWorkloadStatusUseCase(runtime outbound.ContainerRuntime) workloadUC.StatusUseCase {
	return usecase.NewWorkloadStatusUseCase(runtime)
}
