#!/usr/bin/env bats

setup() {
  load 'test_helper/common'
  MOCK_PORT=$(get_free_port)
  export BOOTSTRAP_API="http://127.0.0.1:${MOCK_PORT}"
  export POLL_INTERVAL=1
  export POLL_TIMEOUT=3
}

teardown() {
  kill_mock_server
}

@test "auth_invalid error_code exits 10" {
  start_mock_server "$MOCK_PORT" 200 '{"error_code":"auth_invalid","message":"用户名或密码错误"}'
  run bash -c "printf 'user\npass\n' | bash \"$BOOTSTRAP_SCRIPT\""
  [ "$status" -eq 10 ]
  [[ "$output" == *"用户名或密码错误"* ]]
}

@test "account_disabled error_code exits 11" {
  start_mock_server "$MOCK_PORT" 200 '{"error_code":"account_disabled","message":"账号已被停用，请联系管理员"}'
  run bash -c "printf 'user\npass\n' | bash \"$BOOTSTRAP_SCRIPT\""
  [ "$status" -eq 11 ]
  [[ "$output" == *"账号已被停用"* ]]
}

@test "account_expired error_code exits 12" {
  start_mock_server "$MOCK_PORT" 200 '{"error_code":"account_expired","message":"账号已过期，请联系管理员续期"}'
  run bash -c "printf 'user\npass\n' | bash \"$BOOTSTRAP_SCRIPT\""
  [ "$status" -eq 12 ]
  [[ "$output" == *"账号已过期"* ]]
}

@test "host_not_found error_code exits 13" {
  start_mock_server "$MOCK_PORT" 200 '{"error_code":"host_not_found","message":"未找到可用主机，请联系管理员分配"}'
  run bash -c "printf 'user\npass\n' | bash \"$BOOTSTRAP_SCRIPT\""
  [ "$status" -eq 13 ]
  [[ "$output" == *"未找到可用主机"* ]]
}

@test "connection refused exits 2" {
  run bash -c "printf 'user\npass\n' | bash \"$BOOTSTRAP_SCRIPT\""
  [ "$status" -eq 2 ]
  [[ "$output" == *"连接控制面失败"* ]]
}

@test "HTTP 500 exits 2" {
  start_mock_server "$MOCK_PORT" 500 '{"error":"internal server error"}'
  run bash -c "printf 'user\npass\n' | bash \"$BOOTSTRAP_SCRIPT\""
  [ "$status" -eq 2 ]
}

@test "unknown error_code exits 1" {
  start_mock_server "$MOCK_PORT" 200 '{"error_code":"unknown_xyz","message":"未知错误"}'
  run bash -c "printf 'user\npass\n' | bash \"$BOOTSTRAP_SCRIPT\""
  [ "$status" -eq 1 ]
}
