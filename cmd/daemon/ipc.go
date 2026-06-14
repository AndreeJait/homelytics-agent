package main

import (
	"github.com/AndreeJait/homelytics-agent/adapter/inbound/ipc"
	"github.com/AndreeJait/homelytics-agent/config"
	authUC "github.com/AndreeJait/homelytics-agent/port/inbound/auth"
	runtimeUC "github.com/AndreeJait/homelytics-agent/port/inbound/runtime"
	statusUC "github.com/AndreeJait/homelytics-agent/port/inbound/status"
	tsnetUC "github.com/AndreeJait/homelytics-agent/port/inbound/tsnet"
	workloadUC "github.com/AndreeJait/homelytics-agent/port/inbound/workload"
	"go.uber.org/dig"
)

// provideIPC registers the IPC server provider into the dig container.
func provideIPC(c *dig.Container) {
	c.Provide(newIPCServer)
}

func newIPCServer(
	cfg *config.AppConfig,
	authUC authUC.UseCase,
	tsnetAuthUC tsnetUC.UseCase,
	runtimeUC runtimeUC.UseCase,
	statusUC statusUC.UseCase,
	workloadRunUC workloadUC.RunUseCase,
	workloadStopUC workloadUC.StopUseCase,
	workloadDelUC workloadUC.DeleteUseCase,
	workloadListUC workloadUC.ListUseCase,
	workloadStatUC workloadUC.StatusUseCase,
) (*ipc.Server, error) {
	return ipc.NewServer(
		cfg.IPC.SocketPath,
		authUC,
		tsnetAuthUC,
		runtimeUC,
		statusUC,
		workloadRunUC,
		workloadStopUC,
		workloadDelUC,
		workloadListUC,
		workloadStatUC,
	)
}
