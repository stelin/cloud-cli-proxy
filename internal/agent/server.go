package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"os/exec"
	"strings"

	"github.com/zanel1u/cloud-cli-proxy/internal/agentapi"
	"github.com/zanel1u/cloud-cli-proxy/internal/network"
	runtimetasks "github.com/zanel1u/cloud-cli-proxy/internal/runtime/tasks"
)

type Server struct {
	socketPath string
	worker     *runtimetasks.Worker
	logger     *slog.Logger
}

func NewServer(socketPath string, repo runtimetasks.WorkerRepo, provider network.Provider) *Server {
	return &Server{
		socketPath: socketPath,
		worker:     runtimetasks.NewWorker(repo, provider),
		logger:     slog.Default(),
	}
}

func (s *Server) Serve(ctx context.Context) error {
	if err := os.MkdirAll(filepath.Dir(s.socketPath), 0o755); err != nil {
		return fmt.Errorf("ensure socket directory: %w", err)
	}
	_ = os.Remove(s.socketPath)

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen on host-agent socket: %w", err)
	}
	defer listener.Close()
	defer os.Remove(s.socketPath)

	if err := os.Chmod(s.socketPath, 0o660); err != nil {
		return fmt.Errorf("chmod host-agent socket: %w", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("GET /v1/containers/{name}/status", func(w http.ResponseWriter, r *http.Request) {
		containerName := r.PathValue("name")

		cmd := exec.CommandContext(r.Context(), "docker", "container", "inspect",
			"-f", "{{.State.Running}}", containerName)
		out, err := cmd.Output()
		if err != nil {
			writeJSON(w, http.StatusOK, agentapi.ContainerStatusResponse{
				Name:    containerName,
				Exists:  false,
				Running: false,
			})
			return
		}

		running := strings.TrimSpace(string(out)) == "true"
		writeJSON(w, http.StatusOK, agentapi.ContainerStatusResponse{
			Name:    containerName,
			Exists:  true,
			Running: running,
		})
	})

	mux.HandleFunc("POST /v1/host-actions", func(w http.ResponseWriter, r *http.Request) {
		var request agentapi.HostActionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		running := agentapi.TaskStatusUpdate{
			TaskID: request.TaskID,
			Status: "running",
		}
		if err := s.worker.UpdateTaskStatus(r.Context(), running); err != nil {
			s.logger.Error("UpdateTaskStatus to running failed", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		update := s.worker.Execute(r.Context(), request)
		if err := s.worker.UpdateTaskStatus(r.Context(), update); err != nil {
			s.logger.Error("UpdateTaskStatus final write failed", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		statusCode := http.StatusAccepted
		if update.Status == "failed" {
			statusCode = http.StatusInternalServerError
		}

		writeJSON(w, statusCode, agentapi.HostActionResponse{
			TaskID:        request.TaskID,
			Action:        request.Action,
			ContainerName: request.ContainerName,
			Update:        update,
		})
	})

	server := &http.Server{Handler: mux}
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		_ = server.Shutdown(context.Background())
		return ctx.Err()
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
