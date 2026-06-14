package command

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/AndreeJait/homelytics-agent/domain/entity"
	workloadUC "github.com/AndreeJait/homelytics-agent/port/inbound/workload"
)

// Executor turns control-plane commands into use-case invocations.
type Executor struct {
	runUC   workloadUC.RunUseCase
	stopUC  workloadUC.StopUseCase
	deleteUC workloadUC.DeleteUseCase
}

// NewExecutor creates a command executor.
func NewExecutor(runUC workloadUC.RunUseCase, stopUC workloadUC.StopUseCase, deleteUC workloadUC.DeleteUseCase) *Executor {
	return &Executor{runUC: runUC, stopUC: stopUC, deleteUC: deleteUC}
}

// Execute runs a single command and returns the result payload (or nil).
func (e *Executor) Execute(ctx context.Context, cmd entity.Command) (any, error) {
	switch cmd.Type {
	case "DEPLOY":
		var req entity.RunWorkloadRequest
		if err := json.Unmarshal(cmd.Payload, &req); err != nil {
			return nil, fmt.Errorf("decode DEPLOY payload: %w", err)
		}
		return e.runUC.Run(ctx, req)
	case "STOP":
		var req entity.WorkloadIDRequest
		if err := json.Unmarshal(cmd.Payload, &req); err != nil {
			return nil, fmt.Errorf("decode STOP payload: %w", err)
		}
		return e.stopUC.Stop(ctx, req.ID)
	case "DELETE":
		var req entity.WorkloadIDRequest
		if err := json.Unmarshal(cmd.Payload, &req); err != nil {
			return nil, fmt.Errorf("decode DELETE payload: %w", err)
		}
		return e.deleteUC.Delete(ctx, req.ID)
	default:
		return nil, fmt.Errorf("unknown command type: %s", cmd.Type)
	}
}
