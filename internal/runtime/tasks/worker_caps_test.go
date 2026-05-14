package tasks

import (
	"strings"
	"testing"
)

// TestBuildCreateArgs_CapabilitiesLocked Phase 51 QUAL-06 / 闭 Phase 49 GAP-1：
// 锁定 worker 容器启动参数 capability 现状。
//
// 契约：
//   - 必须含 --cap-add NET_ADMIN（sing-box tun 设备依赖）
//   - 不得含 --cap-add SYS_ADMIN（业务不依赖，docker 默认 OK）
//   - 必须显式 --cap-drop NET_RAW（docker 默认带，必须显式 drop 才能去掉；
//     去掉后容器内 SOCK_RAW 创建即时 PermissionDenied，闭 LEAK-06 攻击面）
func TestBuildCreateArgs_CapabilitiesLocked(t *testing.T) {
	w := &Worker{}
	req := minimalCreateHostRequest("h-caps")

	args, err := w.buildCreateArgs(req, "c-caps", "c-caps", nil)
	if err != nil {
		t.Fatalf("buildCreateArgs: %v", err)
	}

	hasFlagPair := func(flag, val string) bool {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == flag && args[i+1] == val {
				return true
			}
		}
		return false
	}

	if !hasFlagPair("--cap-add", "NET_ADMIN") {
		t.Errorf("expected --cap-add NET_ADMIN; got args=%s", strings.Join(args, " "))
	}
	if hasFlagPair("--cap-add", "SYS_ADMIN") {
		t.Errorf("--cap-add SYS_ADMIN must be removed (Phase 51 QUAL-06); got args=%s",
			strings.Join(args, " "))
	}
	if !hasFlagPair("--cap-drop", "NET_RAW") {
		t.Errorf("expected --cap-drop NET_RAW (Phase 51 QUAL-06 / Phase 49 GAP-1); got args=%s",
			strings.Join(args, " "))
	}
}
