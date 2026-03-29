package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/zanel1u/cloud-cli-proxy/internal/agentapi"
	"github.com/zanel1u/cloud-cli-proxy/internal/store/repository"
)

type mockReconcileStore struct {
	runningHosts   []repository.Host
	runningHostErr error
	updatedHosts   []struct{ ID, Status string }
	staleMarked    []repository.Task
	staleErr       error
	recordedEvents []repository.RecordEventParams
}

func (m *mockReconcileStore) ListRunningHosts(_ context.Context) ([]repository.Host, error) {
	return m.runningHosts, m.runningHostErr
}

func (m *mockReconcileStore) UpdateHostStatus(_ context.Context, id, status string) error {
	m.updatedHosts = append(m.updatedHosts, struct{ ID, Status string }{id, status})
	return nil
}

func (m *mockReconcileStore) MarkStaleTasks(_ context.Context, _ time.Duration) ([]repository.Task, error) {
	return m.staleMarked, m.staleErr
}

func (m *mockReconcileStore) RecordEvent(_ context.Context, params repository.RecordEventParams) (repository.Event, error) {
	m.recordedEvents = append(m.recordedEvents, params)
	return repository.Event{}, nil
}

type mockInspector struct {
	results map[string]agentapi.ContainerStatusResponse
	errors  map[string]error
}

func (m *mockInspector) InspectContainer(_ context.Context, name string) (agentapi.ContainerStatusResponse, error) {
	if err, ok := m.errors[name]; ok {
		return agentapi.ContainerStatusResponse{}, err
	}
	if resp, ok := m.results[name]; ok {
		return resp, nil
	}
	return agentapi.ContainerStatusResponse{}, nil
}

func TestReconciler(t *testing.T) {
	t.Run("Run_AllHealthy_NoUpdates", func(t *testing.T) {
		store := &mockReconcileStore{
			runningHosts: []repository.Host{
				{ID: "h1"},
				{ID: "h2"},
			},
		}
		inspector := &mockInspector{
			results: map[string]agentapi.ContainerStatusResponse{
				"cloudproxy-h1": {Exists: true, Running: true},
				"cloudproxy-h2": {Exists: true, Running: true},
			},
		}
		r := NewReconciler(slog.Default(), store, inspector, 10*time.Minute)

		err := r.Run(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(store.updatedHosts) != 0 {
			t.Errorf("expected no host updates, got %d", len(store.updatedHosts))
		}
		if len(store.recordedEvents) != 0 {
			t.Errorf("expected no events, got %d", len(store.recordedEvents))
		}
	})

	t.Run("Run_ContainerStopped_UpdatesStatus", func(t *testing.T) {
		store := &mockReconcileStore{
			runningHosts: []repository.Host{{ID: "h1"}},
		}
		inspector := &mockInspector{
			results: map[string]agentapi.ContainerStatusResponse{
				"cloudproxy-h1": {Exists: true, Running: false},
			},
		}
		r := NewReconciler(slog.Default(), store, inspector, 10*time.Minute)

		err := r.Run(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(store.updatedHosts) != 1 {
			t.Fatalf("expected 1 host update, got %d", len(store.updatedHosts))
		}
		if store.updatedHosts[0].ID != "h1" || store.updatedHosts[0].Status != "stopped" {
			t.Errorf("update = %+v, want {ID:h1 Status:stopped}", store.updatedHosts[0])
		}

		foundDrift := false
		for _, ev := range store.recordedEvents {
			if ev.Type == "reconcile.host.drift" {
				foundDrift = true
				actual, _ := ev.Metadata["actual_status"].(string)
				if actual != "stopped" {
					t.Errorf("actual_status = %q, want stopped", actual)
				}
			}
		}
		if !foundDrift {
			t.Error("expected reconcile.host.drift event, not found")
		}
	})

	t.Run("Run_ContainerNotFound_UpdatesStatus", func(t *testing.T) {
		store := &mockReconcileStore{
			runningHosts: []repository.Host{{ID: "h1"}},
		}
		inspector := &mockInspector{
			results: map[string]agentapi.ContainerStatusResponse{
				"cloudproxy-h1": {Exists: false, Running: false},
			},
		}
		r := NewReconciler(slog.Default(), store, inspector, 10*time.Minute)

		err := r.Run(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(store.updatedHosts) != 1 {
			t.Fatalf("expected 1 host update, got %d", len(store.updatedHosts))
		}
		if store.updatedHosts[0].Status != "stopped" {
			t.Errorf("status = %q, want stopped", store.updatedHosts[0].Status)
		}

		foundDrift := false
		for _, ev := range store.recordedEvents {
			if ev.Type == "reconcile.host.drift" {
				foundDrift = true
				actual, _ := ev.Metadata["actual_status"].(string)
				if actual != "not_found" {
					t.Errorf("actual_status = %q, want not_found", actual)
				}
			}
		}
		if !foundDrift {
			t.Error("expected reconcile.host.drift event, not found")
		}
	})

	t.Run("Run_InspectError_SkipsHost", func(t *testing.T) {
		store := &mockReconcileStore{
			runningHosts: []repository.Host{{ID: "h1"}},
		}
		inspector := &mockInspector{
			errors: map[string]error{
				"cloudproxy-h1": errors.New("agent unreachable"),
			},
		}
		r := NewReconciler(slog.Default(), store, inspector, 10*time.Minute)

		err := r.Run(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(store.updatedHosts) != 0 {
			t.Errorf("expected no host updates when inspect fails, got %d", len(store.updatedHosts))
		}
	})

	t.Run("Run_StaleTasks_MarkedAndRecorded", func(t *testing.T) {
		hostID := "h1"
		store := &mockReconcileStore{
			staleMarked: []repository.Task{
				{ID: "t1", HostID: &hostID, Kind: "start_host"},
				{ID: "t2", HostID: &hostID, Kind: "stop_host"},
			},
		}
		inspector := &mockInspector{}
		r := NewReconciler(slog.Default(), store, inspector, 10*time.Minute)

		err := r.Run(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		staleCount := 0
		for _, ev := range store.recordedEvents {
			if ev.Type == "reconcile.task.stale" {
				staleCount++
			}
		}
		if staleCount != 2 {
			t.Errorf("stale event count = %d, want 2", staleCount)
		}
	})

	t.Run("Run_NoRunningHosts_NoAction", func(t *testing.T) {
		store := &mockReconcileStore{
			runningHosts: []repository.Host{},
		}
		inspector := &mockInspector{}
		r := NewReconciler(slog.Default(), store, inspector, 10*time.Minute)

		err := r.Run(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(store.updatedHosts) != 0 {
			t.Errorf("expected no updates, got %d", len(store.updatedHosts))
		}
		if len(store.recordedEvents) != 0 {
			t.Errorf("expected no events, got %d", len(store.recordedEvents))
		}
	})
}
