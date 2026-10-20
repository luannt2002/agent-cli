#!/usr/bin/env bash
# Chạy các test mới của P0-2b từ một bộ test binary đã build sẵn, với HOME/XDG cô lập
# (không đụng ~/.cache, ~/.config thật). In kết quả theo tên test, không in env.
# Dùng: bash runtb.sh <thư mục test binary> <thư mục HOME mới>
set -u
TB=$1; H=$2
W=/home/luannt/application/opensource/agent-cli-p02b
[ -e "$H" ] && { echo "exists: $H" >&2; exit 2; }
mkdir -p "$H"
run() {
  local bin=$1 pkg=$2; shift 2
  (cd "$W/$pkg" && env -u CRUSH_CACHE_DIR -u CRUSH_GLOBAL_DATA -u CRUSH_GLOBAL_CONFIG \
    HOME="$H" XDG_CACHE_HOME="$H/.cache" XDG_CONFIG_HOME="$H/.config" XDG_DATA_HOME="$H/.local/share" \
    GOMAXPROCS=2 "$bin" -test.count=1 -test.v "$@" 2>&1)
}
run "$TB/cmd.test" internal/cmd -test.run '^(TestChildEnvsIgnoreProjectDotenv|TestEnsureServer_RefusesModelCommandOnBusyStaleServer|TestConnectToServerSendsNoAgentRuns)$'
run "$TB/workspace.test" internal/workspace -test.run '^TestShellCommandIgnoresProjectDotenvTrustedKeys$'
run "$TB/backend.test" internal/backend -test.run '^TestRunShellCommandIgnoresProjectDotenvTrustedKeys$'
run "$TB/provider.test" internal/agent/tools/provider -test.run '^TestRunCommandIgnoresProjectDotenv$'
echo "== files written under isolated HOME:"
find "$H" -type f | sed "s|^$H|HOME|"
