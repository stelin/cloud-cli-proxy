package tasks

import (
	"context"
	"log/slog"

	"github.com/zanel1u/cloud-cli-proxy/internal/agentapi"
)

type EmbeddedDispatcher struct {
	worker *Worker
}

func NewEmbeddedDispatcher(worker *Worker) *EmbeddedDispatcher {
	return &EmbeddedDispatcher{worker: worker}
}

func (d *EmbeddedDispatcher) Dispatch(ctx context.Context, request agentapi.HostActionRequest) (agentapi.HostActionResponse, error) {
	slog.Info("embedded: executing host action in-process",
		"task_id", request.TaskID,
		"host_id", request.HostID,
		"action", request.Action,
	)

	update := d.worker.Execute(ctx, request)

	if err := d.worker.UpdateTaskStatus(ctx, update); err != nil {
		slog.Error("embedded: failed to update task status", "error", err)
	}

	return agentapi.HostActionResponse{Update: update}, nil
}
