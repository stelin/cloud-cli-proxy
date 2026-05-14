//go:build linux

package network

import (
	"encoding/binary"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// nsConfig 控制 GetContainerNetNS 重试窗口与上限。Phase 51 QUAL-04 把原本
// 硬编码的「5 次重试 × 300ms 间隔」抽出，允许 e2e harness 用更短窗口加速测试。
// 默认值与 Phase 51 之前的硬编码保持一致，确保零回归。
type nsConfig struct {
	probeWindow time.Duration
	maxRetries  int
}

// Option 是 GetContainerNetNS 的 functional option 类型。
type Option func(*nsConfig)

// WithProbeWindow 覆盖 netns 探测两次尝试之间的等待时长（默认 300ms）。
func WithProbeWindow(d time.Duration) Option {
	return func(c *nsConfig) {
		if d > 0 {
			c.probeWindow = d
		}
	}
}

// WithMaxRetries 覆盖 netns 探测最大重试次数（默认 5）。
func WithMaxRetries(n int) Option {
	return func(c *nsConfig) {
		if n > 0 {
			c.maxRetries = n
		}
	}
}

// defaultNsConfig 返回 Phase 51 QUAL-04 锁定的默认探测配置。
func defaultNsConfig() nsConfig {
	return nsConfig{
		probeWindow: 300 * time.Millisecond,
		maxRetries:  5,
	}
}

// GetContainerNetNS retrieves the network namespace handle and init PID
// for a running Docker container identified by name.
//
// Phase 51 QUAL-04：暴露 functional option（WithProbeWindow / WithMaxRetries）
// 以便 e2e 加速；默认行为不变。
func GetContainerNetNS(containerName string, opts ...Option) (netns.NsHandle, uint32, error) {
	cfg := defaultNsConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	out, err := exec.Command("docker", "inspect", "-f", "{{.State.Pid}}", containerName).Output()
	if err != nil {
		return 0, 0, &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("inspect container %s pid: %v", containerName, err),
		}
	}

	pid, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 32)
	if err != nil {
		return 0, 0, &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("parse container pid: %v", err),
		}
	}
	if pid == 0 {
		return 0, 0, &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("container %s not running (pid=0)", containerName),
		}
	}

	var lastErr error
	for attempt := 1; attempt <= cfg.maxRetries; attempt++ {
		runningOut, runErr := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", containerName).Output()
		if runErr != nil || strings.TrimSpace(string(runningOut)) != "true" {
			if attempt < cfg.maxRetries {
				time.Sleep(cfg.probeWindow)
				continue
			}
		}

		ns, nsErr := netns.GetFromPid(int(pid))
		if nsErr == nil {
			return ns, uint32(pid), nil
		}
		lastErr = nsErr
		if attempt < cfg.maxRetries {
			time.Sleep(cfg.probeWindow)
		}
	}

	statusOut, _ := exec.Command("docker", "inspect", "-f", "{{.State.Status}}|{{.State.ExitCode}}", containerName).Output()
	statusInfo := strings.TrimSpace(string(statusOut))
	if statusInfo == "" {
		statusInfo = "unknown"
	}

	return 0, 0, &NetworkError{
		Type:    ErrTunnelSetupFailed,
		Message: fmt.Sprintf("get netns from pid %d after %d attempts (container status=%s): %v", pid, cfg.maxRetries, statusInfo, lastErr),
	}
}

// InjectManagementVeth creates a veth pair between the host and container
// network namespaces for SSH management access. The container side intentionally
// has no default gateway to prevent it from becoming an egress bypass path.
func InjectManagementVeth(hostNS, containerNS netns.NsHandle, hostID string) (hostIP, containerIP string, err error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	idx := mgmtSubnetIndex(hostID)
	block := idx / 128
	offset := (idx % 128) * 2

	hostAddr := fmt.Sprintf("10.99.%d.%d/30", block+1, offset+1)
	containerAddr := fmt.Sprintf("10.99.%d.%d/30", block+1, offset+2)

	hostVethName := fmt.Sprintf("mgmt-%s", truncateID(hostID, 8))
	containerVethName := "mgmt0"

	if err := netns.Set(hostNS); err != nil {
		return "", "", &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("set host netns: %v", err),
		}
	}

	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{Name: hostVethName},
		PeerName:  containerVethName,
	}
	if err := netlink.LinkAdd(veth); err != nil {
		return "", "", &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("create veth pair: %v", err),
		}
	}

	peer, err := netlink.LinkByName(containerVethName)
	if err != nil {
		return "", "", &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("find peer veth %s: %v", containerVethName, err),
		}
	}
	if err := netlink.LinkSetNsFd(peer, int(containerNS)); err != nil {
		return "", "", &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("move peer to container netns: %v", err),
		}
	}

	hostLink, err := netlink.LinkByName(hostVethName)
	if err != nil {
		return "", "", &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("find host veth %s: %v", hostVethName, err),
		}
	}

	hostIPNet, err := netlink.ParseAddr(hostAddr)
	if err != nil {
		return "", "", &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("parse host addr %s: %v", hostAddr, err),
		}
	}
	if err := netlink.AddrAdd(hostLink, hostIPNet); err != nil {
		return "", "", &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("add addr to host veth: %v", err),
		}
	}
	if err := netlink.LinkSetUp(hostLink); err != nil {
		return "", "", &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("bring up host veth: %v", err),
		}
	}

	if err := netns.Set(containerNS); err != nil {
		return "", "", &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("set container netns: %v", err),
		}
	}
	defer netns.Set(hostNS) //nolint:errcheck

	containerLink, err := netlink.LinkByName(containerVethName)
	if err != nil {
		return "", "", &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("find container veth %s: %v", containerVethName, err),
		}
	}

	containerIPNet, err := netlink.ParseAddr(containerAddr)
	if err != nil {
		return "", "", &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("parse container addr %s: %v", containerAddr, err),
		}
	}
	if err := netlink.AddrAdd(containerLink, containerIPNet); err != nil {
		return "", "", &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("add addr to container veth: %v", err),
		}
	}
	if err := netlink.LinkSetUp(containerLink); err != nil {
		return "", "", &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("bring up container veth: %v", err),
		}
	}

	hostIPOnly, _, _ := net.ParseCIDR(hostAddr)
	containerIPOnly, _, _ := net.ParseCIDR(containerAddr)

	return hostIPOnly.String(), containerIPOnly.String(), nil
}

// mgmtSubnetIndex derives a unique /30 subnet index from hostID
// to avoid address collisions across concurrent containers.
func mgmtSubnetIndex(hostID string) uint16 {
	b := []byte(hostID)
	if len(b) < 4 {
		padded := make([]byte, 4)
		copy(padded, b)
		b = padded
	}
	return binary.BigEndian.Uint16(b[:2]) ^ binary.BigEndian.Uint16(b[2:4])%16382
}
