package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/zanel1u/cloud-cli-proxy/internal/agentapi"
	"github.com/zanel1u/cloud-cli-proxy/internal/store/repository"
)

type ReconcileStore interface {
	ListRunningHosts(context.Context) ([]repository.Host, error)
	UpdateHostStatus(context.Context, string, string) error
	MarkStaleTasks(context.Context, time.Duration) ([]repository.Task, error)
	RecordEvent(context.Context, repository.RecordEventParams) (repository.Event, error)
}

type ContainerInspector interface {
	InspectContainer(context.Context, string) (agentapi.ContainerStatusResponse, error)
}

type Reconciler struct {
	logger         *slog.Logger
	store          ReconcileStore
	inspector      ContainerInspector
	staleThreshold time.Duration
}

func NewReconciler(logger *slog.Logger, store ReconcileStore, inspector ContainerInspector, staleThreshold time.Duration) *Reconciler {
	if staleThreshold == 0 {
		staleThreshold = 10 * time.Minute
	}
	return &Reconciler{
		logger:         logger,
		store:          store,
		inspector:      inspector,
		staleThreshold: staleThreshold,
	}
}

func (r *Reconciler) Run(ctx context.Context) error {
	if err := r.reconcileHosts(ctx); err != nil {
		r.logger.Error("host reconciliation failed", "error", err)
	}

	if err := r.reconcileStaleTasks(ctx); err != nil {
		r.logger.Error("stale task reconciliation failed", "error", err)
	}

	return nil
}

func (r *Reconciler) reconcileHosts(ctx context.Context) error {
	hosts, err := r.store.ListRunningHosts(ctx)
	if err != nil {
		return fmt.Errorf("list running hosts: %w", err)
	}

	for _, host := range hosts {
		containerName := fmt.Sprintf("cloudproxy-%s", host.ID)

		status, err := r.inspector.InspectContainer(ctx, containerName)
		if err != nil {
			r.logger.Warn("inspect container communication failed, skipping host",
				"host_id", host.ID, "container", containerName, "error", err)
			continue
		}

		if status.Exists && status.Running {
			continue
		}

		actualStatus := "stopped"
		if !status.Exists {
			actualStatus = "not_found"
		}

		if err := r.store.UpdateHostStatus(ctx, host.ID, "stopped"); err != nil {
			r.logger.Error("update drifted host status failed", "host_id", host.ID, "error", err)
			continue
		}

		r.store.RecordEvent(ctx, repository.RecordEventParams{
			HostID:  &host.ID,
			Level:   "warn",
			Type:    "reconcile.host.drift",
			Message: "对账发现主机状态漂移",
			Metadata: map[string]any{
				"operator":      "system",
				"db_status":     "running",
				"actual_status": actualStatus,
				"host_id":       host.ID,
			},
		})

		r.logger.Info("reconciled drifted host",
			"host_id", host.ID, "db_status", "running", "actual_status", actualStatus)
	}

	return nil
}

func (r *Reconciler) reconcileStaleTasks(ctx context.Context) error {
	staleTasks, err := r.store.MarkStaleTasks(ctx, r.staleThreshold)
	if err != nil {
		return fmt.Errorf("mark stale tasks: %w", err)
	}

	for _, task := range staleTasks {
		r.store.RecordEvent(ctx, repository.RecordEventParams{
			HostID:  task.HostID,
			Level:   "warn",
			Type:    "reconcile.task.stale",
			Message: "对账发现陈旧任务",
			Metadata: map[string]any{
				"operator":   "system",
				"task_id":    task.ID,
				"kind":       task.Kind,
				"old_status": "pending/running",
				"threshold":  r.staleThreshold.String(),
			},
		})

		r.logger.Info("marked stale task", "task_id", task.ID, "kind", task.Kind)
	}

	if len(staleTasks) > 0 {
		r.logger.Info("stale task reconciliation completed", "stale_count", len(staleTasks))
	}

	return nil
}
