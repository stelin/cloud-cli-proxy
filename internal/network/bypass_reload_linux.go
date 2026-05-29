//go:build linux

package network

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// workerNetNSPIDLookup 通过 host id 解析对应 worker 容器的 init pid（State.Pid）。
// 默认实现用 `docker inspect -f '{{.State.Pid}}' cloudproxy-<hostID>`，与
// internal/runtime/tasks/worker_bypass_reload.go::verifyBypassHealthyDefault 同源。
//
// 抽包级 var 是为了让 host 上 nft 命令通过 `nsenter -t <pid> -n -- nft ...` 真正
// 落到 worker 容器 netns（worker netns 与 host netns 是两个 namespace，host 上
// 直接跑 nft 看不到 worker 内的 cloudproxy 表 / whitelist_v4 set —— Phase 47
// verification BLOCKER-2 根因）。单测注入 fake 返回固定 pid，避开 docker 依赖。
var workerNetNSPIDLookup = func(ctx context.Context, hostID string) (int, error) {
	containerName := workerContainerName(hostID)
	cmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Pid}}", containerName)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("docker inspect pid for %s: %w", containerName, err)
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return 0, fmt.Errorf("docker inspect returned empty pid for %s", containerName)
	}
	pid, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("parse pid %q for %s: %w", s, containerName, err)
	}
	if pid <= 0 {
		return 0, fmt.Errorf("worker container %s not running (pid=%d)", containerName, pid)
	}
	return pid, nil
}

// nftRunner 跑 `nsenter -t <pid> -n -- nft -f -` 把 stdin 当成事务批量下发到
// worker 容器 netns；返回 stdout/stderr 合并输出。
//
// 必须经过 nsenter 是因为白名单 set 由 ConfigureBypassFirewall 在 worker netns
// 的 `ip cloudproxy` 表内创建，host 上的 nft 看不见该 namespace —— Phase 47
// verification BLOCKER-2 修复的核心要点。
//
// 单独抽包级 var 是为了让单测可以注入 fake，不依赖宿主机存在 nft / nsenter 二进制。
var nftRunner = func(ctx context.Context, netNSPID int, stdin string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "nsenter", "-t", strconv.Itoa(netNSPID), "-n", "--", "nft", "-f", "-")
	cmd.Stdin = strings.NewReader(stdin)
	return cmd.CombinedOutput()
}

// nftJSONLister 跑 `nsenter -t <pid> -n -- nft -j list set ip cloudproxy <setName>`
// 返回 JSON bytes，作为 VerifyBypassConsistency 的事实源。
//
// 同 nftRunner，必须 nsenter 进 worker netns 才能看到真实 set 内容；单测注入
// fake 输出避开宿主机 nft 依赖。
var nftJSONLister = func(ctx context.Context, netNSPID int, setName string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "nsenter", "-t", strconv.Itoa(netNSPID), "-n", "--",
		"nft", "-j", "list", "set", bypassNftFamily, bypassNftTable, setName)
	return cmd.Output()
}

// ApplyBypassRuleSet 把 snapshot 的 cidrsJSON / domainsJSON 落盘 +
// 同步更新 nft @whitelist_v4 set。
//
// 严格顺序（任意一步失败立即返回 error 并清理 tmp 文件，原文件 + nft set 都保持旧值）：
//  1. 解析 cidrsJSON，抽出全部 ip_cidr → []string；解析失败立即返回，**不落盘 / 不下发 nft**。
//  2. 通过 workerNetNSPIDLookup 取 worker 容器 init pid（`docker inspect -f '{{.State.Pid}}'`）。
//     pid 取不到立即返回，原文件不动。
//  3. 拼出 nft 事务 stdin：`flush set ip cloudproxy whitelist_v4` + `add element ... { ... }`；
//     通过 `nsenter -t <pid> -n -- nft -f -` 落到 worker netns；失败立即返回，原文件不动。
//  4. 把 cidrsJSON 原始字节通过 tmpfile + os.Rename 原子写到
//     <GatewayConfigDir>/whitelist-cidrs.json；失败先 os.Remove(tmp)。
//  5. 同步 domainsJSON 原始字节到 whitelist-domains.json（同样 tmpfile + rename）。
//
// 这样保证一旦失败：nft set 仍为旧值 + 两个文件仍为旧值，sing-box 看到的世界自洽。
func ApplyBypassRuleSet(ctx context.Context, hostID string, cidrsJSON, domainsJSON json.RawMessage) error {
	cidrs, err := extractCIDRsFromRuleSetJSON(cidrsJSON)
	if err != nil {
		return fmt.Errorf("parse cidrs json: %w", err)
	}

	// 1. 拿 worker netns pid（host 上的 nft 看不到 worker netns 内的表）
	pid, err := workerNetNSPIDLookup(ctx, hostID)
	if err != nil {
		return fmt.Errorf("lookup worker netns pid: %w", err)
	}

	// 2. nft 事务（失败立即 abort，不写文件）
	stdin := buildNftWhitelistUpdateScript(cidrs)
	if out, err := nftRunner(ctx, pid, stdin); err != nil {
		return fmt.Errorf("nft -f update set %s failed: %s: %w", bypassNftSetName, strings.TrimSpace(string(out)), err)
	}

	// 3. 文件原子写盘
	dir := GatewayConfigDir(hostID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir gateway dir: %w", err)
	}

	if err := atomicWriteFile(filepath.Join(dir, "whitelist-cidrs.json"), cidrsJSON); err != nil {
		return fmt.Errorf("atomic write whitelist-cidrs.json: %w", err)
	}
	if err := atomicWriteFile(filepath.Join(dir, "whitelist-domains.json"), domainsJSON); err != nil {
		return fmt.Errorf("atomic write whitelist-domains.json: %w", err)
	}
	return nil
}

// VerifyBypassConsistency 读取 <GatewayConfigDir>/whitelist-cidrs.json 与
// `nsenter -t <worker pid> -n -- nft -j list set ip cloudproxy whitelist_v4` 输出，
// 分别按「CIDR 集合归一化 → sha256」计算两侧 hash 并比较；一致返回 OK=true，
// 不一致返回 OK=false + Detail。
//
// 任何子步骤失败（读文件 / 解析 / 取 pid / 调 nft）都返回 error；hash 不一致不算 error，
// 只在 ConsistencyResult.OK 上反映。
func VerifyBypassConsistency(ctx context.Context, hostID string) (ConsistencyResult, error) {
	// 1. ruleset 文件侧
	cidrsPath := filepath.Join(GatewayConfigDir(hostID), "whitelist-cidrs.json")
	fileBytes, err := os.ReadFile(cidrsPath)
	if err != nil {
		return ConsistencyResult{}, fmt.Errorf("read whitelist-cidrs.json: %w", err)
	}
	fileCIDRs, err := extractCIDRsFromRuleSetJSON(fileBytes)
	if err != nil {
		return ConsistencyResult{}, fmt.Errorf("parse whitelist-cidrs.json: %w", err)
	}
	fileHash := normalizedSHA256(fileCIDRs)

	// 2. nft set 侧（必须 nsenter 进 worker netns 才看得到真实 set）
	pid, err := workerNetNSPIDLookup(ctx, hostID)
	if err != nil {
		return ConsistencyResult{}, fmt.Errorf("lookup worker netns pid: %w", err)
	}
	nftOut, err := nftJSONLister(ctx, pid, bypassNftSetName)
	if err != nil {
		return ConsistencyResult{}, fmt.Errorf("nft -j list set %s failed: %w", bypassNftSetName, err)
	}
	nftCIDRs, err := extractCIDRsFromNftJSON(nftOut)
	if err != nil {
		return ConsistencyResult{}, fmt.Errorf("parse nft -j output: %w", err)
	}
	nftHash := normalizedSHA256(nftCIDRs)

	res := ConsistencyResult{
		RuleSetSHA256: fileHash,
		NftSetSHA256:  nftHash,
		OK:            fileHash == nftHash,
	}
	if !res.OK {
		res.Detail = fmt.Sprintf("cidr set mismatch: file=%d entries, nft=%d entries", len(fileCIDRs), len(nftCIDRs))
	}
	return res, nil
}
