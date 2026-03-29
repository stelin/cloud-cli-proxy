package http

import (
	"log/slog"
	nethttp "net/http"
	"os"
)

const DefaultBootstrapScriptPath = "deploy/bootstrap/cloud-bootstrap.sh"

type BootstrapScriptDependencies struct {
	Logger     *slog.Logger
	ScriptPath string
}

type bootstrapScriptHandler struct {
	logger     *slog.Logger
	scriptPath string
}

func NewBootstrapScriptHandler(deps BootstrapScriptDependencies) nethttp.Handler {
	path := deps.ScriptPath
	if path == "" {
		path = DefaultBootstrapScriptPath
	}
	return &bootstrapScriptHandler{
		logger:     deps.Logger,
		scriptPath: path,
	}
}

func (h *bootstrapScriptHandler) ServeHTTP(w nethttp.ResponseWriter, r *nethttp.Request) {
	content, err := os.ReadFile(h.scriptPath)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("read bootstrap script failed", "path", h.scriptPath, "error", err)
		}
		writeJSON(w, nethttp.StatusInternalServerError, map[string]string{
			"error": "bootstrap script unavailable",
		})
		return
	}

	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	w.WriteHeader(nethttp.StatusOK)
	w.Write(content)
}
