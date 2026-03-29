#!/usr/bin/env bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../deploy/bootstrap" && pwd)"
BOOTSTRAP_SCRIPT="${SCRIPT_DIR}/cloud-bootstrap.sh"

get_free_port() {
  python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()'
}

# start_mock_server PORT STATUS_CODE BODY
# Launches a Python HTTP server that returns the given status and body for all requests.
# Stores PID in MOCK_PID for teardown.
start_mock_server() {
  local port="$1" status="$2" body="$3"
  python3 -c "
import http.server, sys, json

class Handler(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        self.send_response(${status})
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        self.wfile.write('''${body}'''.encode())
    def do_GET(self):
        self.do_POST()
    def log_message(self, *args):
        pass

s = http.server.HTTPServer(('127.0.0.1', ${port}), Handler)
s.serve_forever()
" &
  MOCK_PID=$!
  # Wait for server to be ready
  for i in $(seq 1 20); do
    if curl -s "http://127.0.0.1:${port}/" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.1
  done
  echo "mock server failed to start on port ${port}" >&2
  return 1
}

kill_mock_server() {
  if [ -n "${MOCK_PID:-}" ]; then
    kill "$MOCK_PID" 2>/dev/null || true
    wait "$MOCK_PID" 2>/dev/null || true
    unset MOCK_PID
  fi
}
