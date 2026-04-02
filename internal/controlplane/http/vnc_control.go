package http

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func restartContainerVNC(containerName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", containerName, "bash", "-lc", "/usr/local/bin/restart-vnc")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker exec restart-vnc: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	return nil
}
