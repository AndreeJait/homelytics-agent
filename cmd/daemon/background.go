package main

import (
	"github.com/AndreeJait/homelytics-agent/config"
	portOutbound "github.com/AndreeJait/homelytics-agent/port/outbound"
	workloadUC "github.com/AndreeJait/homelytics-agent/port/inbound/workload"
	"github.com/AndreeJait/homelytics-agent/usecase/command"
	"go.uber.org/dig"
)

// provideBackgroundTasks registers non-IPC background services.
func provideBackgroundTasks(c *dig.Container) {
	c.Provide(newCommandExecutor)
	c.Provide(newHeartbeatLoop)
	c.Provide(newTSNetCommandListener)
}

func newCommandExecutor(
	runUC workloadUC.RunUseCase,
	stopUC workloadUC.StopUseCase,
	deleteUC workloadUC.DeleteUseCase,
) *command.Executor {
	return command.NewExecutor(runUC, stopUC, deleteUC)
}

func newHeartbeatLoop(
	cfg *config.AppConfig,
	backend portOutbound.HomelyticsBackend,
	store portOutbound.SessionStore,
	runtime portOutbound.ContainerRuntime,
	vpn portOutbound.VPN,
	executor *command.Executor,
) *HeartbeatLoop {
	return NewHeartbeatLoop(cfg, backend, store, runtime, vpn, executor)
}

func newTSNetCommandListener(
	cfg *config.AppConfig,
	vpn portOutbound.VPN,
	executor *command.Executor,
) *TSNetCommandListener {
	return NewTSNetCommandListener(cfg, vpn, executor)
}
