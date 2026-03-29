package http

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/zanel1u/cloud-cli-proxy/internal/store/repository"
)

type stubBindingStore struct {
	host      repository.Host
	hostErr   error
	binding   repository.HostBinding
	bindErr   error
	unbindErr error
	hostID    string
	hostIDErr error
}

func (s *stubBindingStore) GetHost(_ context.Context, _ string) (repository.Host, error) {
	return s.host, s.hostErr
}

func (s *stubBindingStore) BindEgressIPToHost(_ context.Context, _, _ string) (repository.HostBinding, error) {
	return s.binding, s.bindErr
}

func (s *stubBindingStore) UnbindEgressIPFromHost(_ context.Context, _ string) error {
	return s.unbindErr
}

func (s *stubBindingStore) GetBindingHostID(_ context.Context, _ string) (string, error) {
	return s.hostID, s.hostIDErr
}

func TestAdminBindingsHandler(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	stoppedHost := repository.Host{ID: "h1", Status: "stopped", CreatedAt: now, UpdatedAt: now}
	runningHost := repository.Host{ID: "h2", Status: "running", CreatedAt: now, UpdatedAt: now}
	sampleBinding := repository.HostBinding{
		BindingID: "b1", HostID: "h1", EgressIPID: "ip1", CreatedAt: now,
	}

	tests := []struct {
		name       string
		method     string
		path       string
		body       any
		store      *stubBindingStore
		wantStatus int
		wantField  string
	}{
		{
			name:   "Bind 201 success",
			method: "POST",
			path:   "/v1/admin/bindings",
			body:   map[string]string{"host_id": "h1", "egress_ip_id": "ip1"},
			store: &stubBindingStore{
				host: stoppedHost, binding: sampleBinding,
			},
			wantStatus: 201,
			wantField:  "binding",
		},
		{
			name:       "Bind missing host_id 400",
			method:     "POST",
			path:       "/v1/admin/bindings",
			body:       map[string]string{"egress_ip_id": "ip1"},
			store:      &stubBindingStore{},
			wantStatus: 400,
		},
		{
			name:       "Bind missing egress_ip_id 400",
			method:     "POST",
			path:       "/v1/admin/bindings",
			body:       map[string]string{"host_id": "h1"},
			store:      &stubBindingStore{},
			wantStatus: 400,
		},
		{
			name:   "Bind host not found 404",
			method: "POST",
			path:   "/v1/admin/bindings",
			body:   map[string]string{"host_id": "missing", "egress_ip_id": "ip1"},
			store: &stubBindingStore{
				hostErr: pgx.ErrNoRows,
			},
			wantStatus: 404,
		},
		{
			name:   "Bind running host 409",
			method: "POST",
			path:   "/v1/admin/bindings",
			body:   map[string]string{"host_id": "h2", "egress_ip_id": "ip1"},
			store: &stubBindingStore{
				host: runningHost,
			},
			wantStatus: 409,
		},
		{
			name:   "Unbind 204 success",
			method: "DELETE",
			path:   "/v1/admin/bindings/b1",
			store: &stubBindingStore{
				hostID: "h1", host: stoppedHost,
			},
			wantStatus: 204,
		},
		{
			name:   "Unbind binding not found 404",
			method: "DELETE",
			path:   "/v1/admin/bindings/missing",
			store: &stubBindingStore{
				hostIDErr: pgx.ErrNoRows,
			},
			wantStatus: 404,
		},
		{
			name:   "Unbind running host 409",
			method: "DELETE",
			path:   "/v1/admin/bindings/b2",
			store: &stubBindingStore{
				hostID: "h2", host: runningHost,
			},
			wantStatus: 409,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := &stubEventRecorder{}
			router := adminTestRouter(t, Dependencies{
				Logger:        slog.Default(),
				AdminBindings: tt.store,
				EventRecorder: events,
			})
			srv := httptest.NewServer(router)
			defer srv.Close()

			var body []byte
			if tt.body != nil {
				body, _ = json.Marshal(tt.body)
			}

			req, _ := nethttp.NewRequest(tt.method, srv.URL+tt.path, bytes.NewReader(body))
			req.Header.Set("Authorization", "Bearer "+validAdminToken(t))
			req.Header.Set("Content-Type", "application/json")

			resp, err := nethttp.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				var respBody map[string]any
				json.NewDecoder(resp.Body).Decode(&respBody)
				t.Errorf("status = %d, want %d; body = %v", resp.StatusCode, tt.wantStatus, respBody)
				return
			}

			if tt.wantField != "" && resp.StatusCode != 204 {
				var respBody map[string]any
				if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if _, ok := respBody[tt.wantField]; !ok {
					t.Errorf("response missing field %q: %v", tt.wantField, respBody)
				}
			}
		})
	}
}
