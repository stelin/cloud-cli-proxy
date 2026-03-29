#!/usr/bin/env bash
set -euo pipefail

BOOTSTRAP_API="${BOOTSTRAP_API:-http://127.0.0.1:8080}"
POLL_INTERVAL="${POLL_INTERVAL:-3}"
POLL_TIMEOUT="${POLL_TIMEOUT:-120}"

_tmpfile="/tmp/cloud-bootstrap-resp.$$"
trap 'stty echo 2>/dev/null || true; rm -f "$_tmpfile"' EXIT INT TERM

printf "用户名: "
read -r username
printf "密码: "
read -r -s password
printf "\n"

http_code=$(curl --silent --show-error \
  -o "$_tmpfile" -w '%{http_code}' \
  -X POST "${BOOTSTRAP_API}/v1/bootstrap/sessions" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${username}\",\"password\":\"${password}\"}" 2>/dev/null) || {
  printf "\n连接控制面失败，请检查网络或稍后重试\n"
  printf "重试命令: curl -sSL %s/v1/bootstrap/script | bash\n" "$BOOTSTRAP_API"
  exit 2
}
response=$(cat "$_tmpfile")

if [ "$http_code" -ge 400 ] 2>/dev/null; then
  error_code=$(echo "$response" | grep -o '"error_code":"[^"]*"' | cut -d'"' -f4 || true)
  message=$(echo "$response" | grep -o '"message":"[^"]*"' | cut -d'"' -f4 || true)
  if [ -n "$error_code" ]; then
    printf "\n错误: %s\n" "$message"
    case "$error_code" in
      auth_invalid)           exit 10 ;;
      account_disabled)       exit 11 ;;
      account_expired)        exit 12 ;;
      host_not_found)         exit 13 ;;
      start_failed)           exit 14 ;;
      ssh_not_ready)          exit 15 ;;
      egress_binding_missing) exit 16 ;;
      *)                      exit 1 ;;
    esac
  fi
  printf "\n服务器返回错误（HTTP %s），请稍后重试或联系管理员\n" "$http_code"
  printf "重试命令: curl -sSL %s/v1/bootstrap/script | bash\n" "$BOOTSTRAP_API"
  exit 2
fi

error_code=$(echo "$response" | grep -o '"error_code":"[^"]*"' | cut -d'"' -f4 || true)
if [ -n "$error_code" ]; then
  message=$(echo "$response" | grep -o '"message":"[^"]*"' | cut -d'"' -f4 || true)
  printf "\n错误: %s\n" "$message"
  case "$error_code" in
    auth_invalid)           exit 10 ;;
    account_disabled)       exit 11 ;;
    account_expired)        exit 12 ;;
    host_not_found)         exit 13 ;;
    start_failed)           exit 14 ;;
    ssh_not_ready)          exit 15 ;;
    egress_binding_missing) exit 16 ;;
    *)                      exit 1 ;;
  esac
fi

task_id=$(echo "$response" | grep -o '"task_id":"[^"]*"' | cut -d'"' -f4 || true)
printf "认证通过，主机启动中 (任务: %s)\n" "$task_id"

elapsed=0
last_stage=""
while [ "$elapsed" -lt "$POLL_TIMEOUT" ]; do
  poll_code=$(curl --silent --show-error \
    -o "$_tmpfile" -w '%{http_code}' \
    "${BOOTSTRAP_API}/v1/bootstrap/tasks/${task_id}" 2>/dev/null) || {
    printf "查询启动状态失败，请检查网络后重试\n"
    printf "重试命令: curl -sSL %s/v1/bootstrap/script | bash\n" "$BOOTSTRAP_API"
    exit 2
  }
  status_resp=$(cat "$_tmpfile")
  if [ "$poll_code" -ge 400 ] 2>/dev/null; then
    printf "\n查询启动状态失败（HTTP %s），请稍后重试\n" "$poll_code"
    printf "重试命令: curl -sSL %s/v1/bootstrap/script | bash\n" "$BOOTSTRAP_API"
    exit 2
  fi

  task_status=$(echo "$status_resp" | grep -o '"task_status":"[^"]*"' | cut -d'"' -f4 || true)
  stage_text=$(echo "$status_resp" | grep -o '"stage_text":"[^"]*"' | cut -d'"' -f4 || true)

  if [ -n "$stage_text" ] && [ "$stage_text" != "$last_stage" ]; then
    printf "  → %s\n" "$stage_text"
    last_stage="$stage_text"
  fi

  if [ "$task_status" = "succeeded" ]; then
    break
  fi

  if [ "$task_status" = "failed" ]; then
    err_code=$(echo "$status_resp" | grep -o '"error_code":"[^"]*"' | cut -d'"' -f4 || true)
    err_msg=$(echo "$status_resp" | grep -o '"error_message":"[^"]*"' | cut -d'"' -f4 || true)
    printf "\n启动失败: %s\n" "$err_msg"
    printf "请重试命令: curl -sSL %s/v1/bootstrap/script | bash\n" "$BOOTSTRAP_API"
    case "$err_code" in
      start_failed)           exit 14 ;;
      ssh_not_ready)          exit 15 ;;
      egress_binding_missing) exit 16 ;;
      *)                      exit 1 ;;
    esac
  fi

  sleep "$POLL_INTERVAL"
  elapsed=$((elapsed + POLL_INTERVAL))
done

if [ "$elapsed" -ge "$POLL_TIMEOUT" ]; then
  printf "\n等待 %s 秒后主机仍未就绪，请稍后重试\n" "$POLL_TIMEOUT"
  printf "重试命令: curl -sSL %s/v1/bootstrap/script | bash\n" "$BOOTSTRAP_API"
  exit 15
fi

handoff_code=$(curl --silent --show-error \
  -o "$_tmpfile" -w '%{http_code}' \
  "${BOOTSTRAP_API}/v1/bootstrap/tasks/${task_id}/handoff" 2>/dev/null) || {
  printf "获取 SSH 接入信息失败，请检查网络后重试\n"
  printf "重试命令: curl -sSL %s/v1/bootstrap/script | bash\n" "$BOOTSTRAP_API"
  exit 2
}
handoff_resp=$(cat "$_tmpfile")
if [ "$handoff_code" -ge 400 ] 2>/dev/null; then
  printf "\n获取 SSH 接入信息失败（HTTP %s），请稍后重试\n" "$handoff_code"
  printf "重试命令: curl -sSL %s/v1/bootstrap/script | bash\n" "$BOOTSTRAP_API"
  exit 2
fi

handoff_error=$(echo "$handoff_resp" | grep -o '"error_code":"[^"]*"' | cut -d'"' -f4 || true)
if [ -n "$handoff_error" ]; then
  handoff_msg=$(echo "$handoff_resp" | grep -o '"message":"[^"]*"' | cut -d'"' -f4 || true)
  printf "\n获取接入信息失败: %s\n" "$handoff_msg"
  printf "请重试命令: curl -sSL %s/v1/bootstrap/script | bash\n" "$BOOTSTRAP_API"
  exit 15
fi

ssh_host=$(echo "$handoff_resp" | grep -o '"host":"[^"]*"' | cut -d'"' -f4 || true)
ssh_port=$(echo "$handoff_resp" | grep -o '"port":[0-9]*' | grep -o '[0-9]*$' || true)
ssh_user=$(echo "$handoff_resp" | grep -o '"user":"[^"]*"' | cut -d'"' -f4 || true)

if [ -z "$ssh_host" ] || [ -z "$ssh_port" ] || [ -z "$ssh_user" ]; then
  printf "\nSSH 接入信息不完整，请联系管理员\n"
  exit 15
fi

printf "\n正在连接 SSH 会话...\n"
exec ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
  -p "$ssh_port" "${ssh_user}@${ssh_host}"
