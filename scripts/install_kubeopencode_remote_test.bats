#!/usr/bin/env bats

setup() {
  TEST_ROOT="$(mktemp -d)"
  export TEST_ROOT
  SCRIPT_UNDER_TEST="$BATS_TEST_DIRNAME/install_kubeopencode_remote.sh"
}

teardown() {
  [ -d "$TEST_ROOT" ] && rm -rf "$TEST_ROOT"
}

install_ssh_stub() {
  mkdir -p "$TEST_ROOT/bin"
  cat > "$TEST_ROOT/bin/ssh" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
count_file="${TEST_ROOT}/ssh-count"
count=0
if [[ -f "$count_file" ]]; then
  count="$(cat "$count_file")"
fi
count=$((count + 1))
printf '%s' "$count" > "$count_file"
printf '%s\n' "$*" >> "${TEST_ROOT}/ssh-args"
if [[ "$count" -eq 3 ]]; then
  printf 'kubeopencode-test\n'
fi
STUB
  chmod +x "$TEST_ROOT/bin/ssh"
  export PATH="$TEST_ROOT/bin:$PATH"
}

@test "rejects host values beginning with dash before ssh" {
  install_ssh_stub

  run bash "$SCRIPT_UNDER_TEST" --host "-oProxyCommand=evil" --port 2222

  [ "$status" -eq 2 ]
  echo "$output" | grep -q "Invalid --host"
  [ ! -f "$TEST_ROOT/ssh-args" ]
}

@test "rejects shell metacharacters in host before ssh" {
  install_ssh_stub

  run bash "$SCRIPT_UNDER_TEST" --host "user@example.com;touch /tmp/pwned" --port 2222

  [ "$status" -eq 2 ]
  echo "$output" | grep -q "Invalid --host"
  [ ! -f "$TEST_ROOT/ssh-args" ]
}

@test "normal user host and port use ssh option terminator" {
  install_ssh_stub

  run bash "$SCRIPT_UNDER_TEST" --host "user@example.com" --port 2222 --namespace test-ns --release test-release

  [ "$status" -eq 0 ]
  [ "$(cat "$TEST_ROOT/ssh-count")" -eq 5 ]
  grep -q -- "^-p 2222 -- user@example.com bash -s -- test-ns$" "$TEST_ROOT/ssh-args"
  grep -q -- "^-p 2222 -- user@example.com bash -s -- test-release test-ns$" "$TEST_ROOT/ssh-args"
}

@test "normal bare host is accepted" {
  install_ssh_stub

  run bash "$SCRIPT_UNDER_TEST" --host "kube-node-01.internal" --port 2200

  [ "$status" -eq 0 ]
  grep -q -- "^-p 2200 -- kube-node-01.internal bash -s -- kubeopencode-system$" "$TEST_ROOT/ssh-args"
}
