# INTEG P1-a × P1-c: dòng báo model khi tiếp session đọc phải tin giữ chỗ

Worktree `/home/luannt/application/opensource/agent-cli-cand`, nhánh `v3-candidate`, HEAD `b5a487f1` (P1-a). Chưa commit, chưa push, không dùng `git stash`.
Patch `INTEG-p1a-p1c.patch`, 6 tệp (numstat): `internal/app/app.go` +14/−8, `internal/app/session_model_test.go` +58/−18, `internal/cmd/run.go` +7/−11, `internal/cmd/run_session_model_test.go` +21/−2, `internal/ui/model/model_restore.go` +3/−2, `internal/ui/model/model_restore_test.go` +20/−3.
`sha256sum -c INTEG-p1a-p1c.patch.sha256` → `d5a50c4ba9fbfcd1eef08a9351212e54808956bb08bd0b2090470c72ab38ea6e`, OK.
`git apply --check` trên bản `git archive b5a487f1` sạch → rc=0. Không đụng SQL, không đụng `*.sql.go`, không thêm tệp `*_v3exposure_test.go`.

## 1. Lỗi (đo trên binary thật, trước khi sửa)

Binary dựng từ `git archive b5a487f1` (`crush-base`, sha256 `5f39ab85c98ae815…`), LLM giả, tin giữ chỗ chèn vào DB đúng hình dạng `recordPrompt` ghi (mục 6):

```
local:  X2 stderr: session 6b1c… last ran on agent-cli-pending/5b0c8489-…; continuing on the project model qq/m-qq (pick another with -m)
local:  Y2 stderr: session c31b… last ran on agent-cli-pending/24655a71-…; continuing on the project model qq/m-qq (pick another with -m)
server: X2 stderr: session 94cc… last ran on agent-cli-pending/b6d90bb9-…; continuing on the project model qq/m-qq (pick another with -m)
server: Y2 stderr: session 4985… last ran on agent-cli-pending/58080c49-…; continuing on the project model qq/m-qq (pick another with -m)
REPRO: FAIL (4)
```

X trả lời thật bằng pp/m-pp, Y bằng qq/m-qq (trùng model của dự án), sau đó mỗi session có một tin giữ chỗ đã đóng (canceled) và một tin còn mở. Cả hai chế độ đều in UUID. Với Y thì dòng báo "khác model" là sai hẳn. Tin giữ chỗ đã đóng nằm lại vĩnh viễn trong session, nên lỗi không tự hết.

## 2. Sửa

| Chỗ | Trước | Sau |
|---|---|---|
| TUI `internal/ui/model/model_restore.go` `sessionModelNote` | vòng lặp bỏ qua tin không phải assistant và bản tóm tắt | thêm `\|\| message.IsPromptPlaceholder(msgs[i])` vào điều kiện bỏ qua (1 dòng) |
| CLI qua server `internal/cmd/run.go` `noteSessionModel` | vòng lặp riêng trên `proto.Message`, chỉ bỏ bản tóm tắt | dùng lại `lastReplyModel(protoReplies(msgs))` của P1-a, xoá vòng lặp |
| Local `internal/app/app.go` `(*App).sessionModelNote` | `Messages.GetLastAssistantMessage` (SQL `GetLastAssistantMessageBySession`, không biết tin giữ chỗ) | `Messages.List` rồi lùi từ cuối: assistant, không tóm tắt, không phải tin giữ chỗ |

Ghi chú thiết kế:

- **Không đụng SQL.** `List` đọc cả session. `run --json` local đã làm đúng việc đó ở cuối lượt (`run.go:241`), nên đây không phải kiểu truy cập mới.
- **Bỏ nhánh `sql.ErrNoRows`.** `List` trả slice rỗng khi session chưa có tin, không trả lỗi. Session chưa có trả lời vẫn không có dòng báo, test "only a pending prompt" phủ ca này. `errors` và `sql` vẫn được dùng ở chỗ khác trong `app.go`, nên import giữ nguyên.
- **`lastReplyModel` bỏ thêm tin assistant có `Model == ""`.** Vòng lặp cũ của P1-c dừng ở tin đó và im lặng. Bản mới lùi tiếp về lời trả lời có model. Cả 4 chỗ tạo tin assistant (`agent.go:781`, `:1235`, `:1985`, `:2165`) đều đặt `Model`, nên khác biệt này chỉ chạm dữ liệu cũ hoặc dữ liệu chèn tay. TUI và local giữ đúng điều kiện cũ của P1-c, chỉ thêm việc bỏ qua tin giữ chỗ.
- Comment tiếng Việt, ≤ 80 cột. Các dòng cũ trong bảng test không bị gióng lại: ca mới nằm sau một dòng trống, thành nhóm gofmt riêng.

## 3. TDD

**ĐỎ** (test mới, code cũ). Lệnh: `go test -run 'TestOpeningSessionKeepsTheWorkspaceModel|TestContinuedRunKeepsTheWorkspaceModel|TestSessionModelNote' ./internal/ui/model/ ./internal/cmd/ ./internal/app/` → rc=1. Đúng 12 ca mới đỏ, và đỏ đúng lý do (`"…last ran on agent-cli-pending/u2…" should not contain "agent-cli-pending"` / `expected: "" actual: "last ran on agent-cli-pending/u2…"`). Các ca cũ vẫn xanh.

```
TestContinuedRunKeepsTheWorkspaceModel/run_-C,_prompt_canceled_after
TestContinuedRunKeepsTheWorkspaceModel/run_-s,_only_a_pending_prompt
TestContinuedRunKeepsTheWorkspaceModel/run_-s,_prompt_pending_after
TestContinuedRunKeepsTheWorkspaceModel/run_-s_same_model,_prompt_pending_after
TestOpeningSessionKeepsTheWorkspaceModel/only_a_pending_prompt
TestOpeningSessionKeepsTheWorkspaceModel/other_model,_prompt_canceled_after
TestOpeningSessionKeepsTheWorkspaceModel/other_model,_prompt_pending_after
TestOpeningSessionKeepsTheWorkspaceModel/same_model,_prompt_pending_after
TestSessionModelNote/only_a_pending_prompt
TestSessionModelNote/other_model,_prompt_canceled_after
TestSessionModelNote/other_model,_prompt_pending_after
TestSessionModelNote/same_model,_prompt_pending_after
```

Ca mới ở mỗi chỗ (ID tin user `u2`, tin giữ chỗ `{assistant, agent-cli-pending, model=u2}`):

| Ca | Kỳ vọng |
|---|---|
| lời trả lời model khác → u2 + tin giữ chỗ **mở** | báo model của lời trả lời thật (TUI gợi ý `alt+m`), không chứa `agent-cli-pending` |
| lời trả lời model khác → u2 + tin giữ chỗ **đã đóng** (canceled) | như trên; ở CLI đi đường `run -C` |
| lời trả lời **cùng** model → u2 + tin giữ chỗ | không báo |
| chỉ có u2 + tin giữ chỗ | không báo |

Thêm vào test có sẵn:

- TUI: mọi `InfoMsg` không được chứa `agent-cli-pending`.
- CLI: stderr không được chứa `agent-cli-pending`. Ở ca không báo, điều kiện đổi từ `NotContains("pp/m-pp")` thành `NotContains("last ran on")`. Điều kiện cũ để lọt lỗi: dòng báo `agent-cli-pending/…` không chứa `pp/m-pp`. Điều kiện mới chặt hơn và các ca cũ vẫn xanh.
- `internal/app/session_model_test.go`: thay `message.Service` giả (chỉ có `GetLastAssistantMessage`) bằng DB SQLite thật (`db.Connect` + `message.NewService`, như `placeholder_stats_test.go`). `created_at` của từng tin được đặt cách nhau 1 giây, vì cột này tính bằng giây và thứ tự khi trùng giây không xác định. Nhờ vậy lúc ĐỎ, test chạy đúng truy vấn SQL cũ; lúc XANH, test chạy `List` thật.

**XANH**: cùng lệnh → rc=0. Kèm `TestLastReplyModelSkipsPromptPlaceholders` của P1-a: 8 (TUI) + 9 (CLI) + 5 (local) ca con PASS.

**Kiểm răng.** Mỗi lần bỏ đúng một điều kiện, chạy test, rồi chép lại tệp từ bản giữ. `sha256sum -c` cả 3 tệp sau đó → OK:

| Đột biến | Kết quả |
|---|---|
| TUI: bỏ `\|\| message.IsPromptPlaceholder(msgs[i])` | rc=1; 4 ca mới FAIL |
| `run.go` `lastReplyModel`: bỏ `&& !message.IsPromptPlaceholder(m)` | rc=1; 4 ca mới của `TestContinuedRunKeepsTheWorkspaceModel` FAIL, cùng `TestLastReplyModelSkipsPromptPlaceholders` |
| `app.go`: bỏ `&& !message.IsPromptPlaceholder(m)` | rc=1; 4 ca mới FAIL |

## 4. Lệnh đã chạy (máy 19/10, `flock /tmp/claude-1000/agent-go.lock`, `GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off`, `-p 2`; RAM trống ≥ 8,4 GB trước mỗi lệnh Go)

| Lệnh | Kết quả |
|---|---|
| `go test -count=1 -json ./internal/ui/model/ ./internal/cmd/ ./internal/app/ ./internal/message/` | rc=0; 4 gói `pass` |
| Bộ test so theo TÊN: cùng lệnh trên `git archive b5a487f1` (gốc) và trên cây đã sửa | gốc 748 test, sau 761; **0 mất, 0 fail ở cả hai, 0 đổi trạng thái**; 13 tên mới đều pass (12 ca placeholder + `TestSessionModelNote/other_model`, ca cũ chuyển thành ca con). Theo gói: app 52→57, cmd 216→220, message 46→46, model 434→438 |
| `go build` (CGO_ENABLED=0, GOEXPERIMENT=greenteagc) → `crush` sha256 `a92be774e1188981…` | rc=0 |
| `bash scripts/eval/session-model.sh /home/luannt/.cache/v3ev/integ-p1ac-sm <crush>` | rc=0, `SESSION-MODEL: PASS` (11/11 ok) |
| `AGENT_CLI_BIN=<crush> bash scripts/test-agent-cli-run-json.sh` | rc=0, `PASS`, 39 dòng `ok` |
| Repro ở mục 6, binary đã sửa (`…/integ-p1ac-repro-fix`) | rc=0, `REPRO: PASS` (6/6). X: `last ran on pp/m-pp; continuing on the project model qq/m-qq` (1 dòng, cả local lẫn server); Y: không báo. Lượt X2 vẫn chạy được: `provider=qq`, `error=''` |
| Repro ở mục 6, binary gốc (`…/integ-p1ac-repro-base`) | rc=1, `REPRO: FAIL (4)` (mục 1) |
| `go vet ./internal/app/ ./internal/cmd/ ./internal/ui/model/` | rc=0 |
| `$(go env GOROOT)/bin/gofmt -l` các tệp đổi; `git diff --check` | rỗng; rc=0 |
| `gofumpt -l` | không dùng được: bản cài cũ hơn Go 1.27, báo lỗi phân tích ở mã có sẵn `app.go:792`/`:822` (method có type parameter), không liên quan tới bản sửa |

## 5. Đã chứng minh / Mới nghi / Chưa kiểm

**Đã chứng minh** (có lệnh tái lập ở trên):

- Cả 3 chỗ bỏ qua tin giữ chỗ, mở lẫn đã đóng, và lùi về lời trả lời thật.
- Session chỉ có tin giữ chỗ thì không báo.
- Bỏ điều kiện ở bất kỳ chỗ nào thì test đỏ.
- Binary thật trước khi sửa in `agent-cli-pending/<uuid>` ở cả local lẫn server; sau khi sửa thì không.

**Mới nghi / để người soát quyết:**

1. `message.Service.GetLastAssistantMessage` (upstream, `b82d7d05`) giờ không còn caller nào trong mã chạy. Patch không xoá nó (luật không xoá mã có sẵn; upstream cũng giữ). Đây là bẫy: caller mới nào dùng nó cũng sẽ đọc phải tin giữ chỗ. Cách kiểm: `grep -rn GetLastAssistantMessage --include=*.go . | grep -v internal/db/`, hiện chỉ còn khai báo và hàm.
2. Điều kiện "lời trả lời thật cuối" giờ nằm ở 3 chỗ: vòng lặp TUI, vòng lặp `app`, và `cmd.lastReplyModel`. Lỗi lần này chính là kiểu lệch đó (P1-a sửa một chỗ, P1-c viết hai chỗ khác). Hướng gom: một helper ở `internal/message`, cạnh `IsPromptPlaceholder`. Patch không gom để sửa ở mức phẫu thuật, vì `lastReplyModel` nằm trong `package cmd` mà `app` và `ui/model` không import được.
3. Chi phí: `run -s/-C` local giờ đọc cả session một lần (`List` giải mã mọi parts). Chưa đo với session rất dài. `run --json` local đã làm đúng việc này ở cuối lượt.

**Chưa kiểm:**

- TUI chưa chạy thật trên màn hình. Mới chỉ có unit test qua `Update(loadSessionMsg)`, theo luật không đụng màn hình người dùng.
- Repro chèn tin giữ chỗ thẳng vào DB, không sinh nó qua hàng đợi hay huỷ thật. Đường ghi là của P1-a, đã soát riêng.
- Chưa soát xem các chỗ khác đọc tin assistant có gặp tin giữ chỗ không: `herdr/translate.go:22,36`, `ui/model/sessions_panel.go:446`, `ui/model/ui.go:2070`, `cmd/run.go:748`, `app/app.go:459`. Ngoài phạm vi brief.
- Chưa chạy: test toàn kho, `-race`, `task lint`.
- Không thêm mục CHANGELOG. Cả hai POC đều chưa phát hành, nên lỗi gộp này chưa từng tới tay người dùng.

## 6. Repro một lần (không vào repo)

Cách dùng: `bash repro.sh <thư-mục-mới ≤70 byte> <binary> <repo>`. Phần dựng môi trường chép từ `scripts/eval/session-model.sh`.

<details><summary>repro.sh</summary>

```bash
#!/usr/bin/env bash
# Repro một lần (không vào repo): tin giữ chỗ của P1-a nằm sau lời trả lời thật
# thì dòng báo model của P1-c (`run -s`) phải dựa vào lời trả lời thật.
# Binary thật + LLM giả (scripts/eval/fakellm.py); tin giữ chỗ được chèn thẳng
# vào DB đúng hình dạng recordPrompt ghi (user + assistant provider
# agent-cli-pending, model = ID tin user), một đóng (canceled), một còn mở.
# Dùng: bash repro.sh <thư-mục-mới ≤70 byte> <binary> <repo>  → 0 PASS, 1 FAIL.
set -u
OUT=$1; BIN=$2; R=$3
[ -e "$OUT" ] && { echo "exists: $OUT" >&2; exit 2; }
mkdir -p "$OUT" && OUT=$(cd "$OUT" && pwd)
FAILS=0
check() { if [ "$2" = 0 ]; then echo "ok   $1"; else echo "FAIL $1"; FAILS=$((FAILS + 1)); fi; }
begin() { # begin <ca> <CRUSH_CLIENT_SERVER 0|1>
  P="$OUT/$1"; CS=$2; HOSTARG=()
  mkdir -p "$P/home/.config/agent-cli" "$P/log" "$P/rt" "$P/proj"; chmod 700 "$P/rt"
  echo '[]' > "$P/rules.json"
  timeout 300 python3 -I "$R/scripts/eval/fakellm.py" "$P/port" "$P/log" "$P/rules.json" >/dev/null 2>&1 &
  FAKE=$!
  for _ in $(seq 50); do [ -s "$P/port" ] && break; sleep 0.1; done
  PORT=$(cat "$P/port")
  cat > "$P/home/.config/agent-cli/crush.json" <<JSON
{"providers":{
  "pp":{"type":"openai-compat","base_url":"http://127.0.0.1:$PORT/pp/v1","api_key":"x","models":[{"id":"m-pp","name":"m-pp","context_window":100000,"default_max_tokens":1000}]},
  "qq":{"type":"openai-compat","base_url":"http://127.0.0.1:$PORT/qq/v1","api_key":"x","models":[{"id":"m-qq","name":"m-qq","context_window":100000,"default_max_tokens":1000}]}},
 "models":{"large":{"provider":"pp","model":"m-pp"},"small":{"provider":"pp","model":"m-pp"}},
 "options":{"disable_provider_auto_update":true,"disable_metrics":true,"disable_default_providers":true}}
JSON
  (cd "$P/proj" && git init -q && echo x > README.md && git add -A && git -c user.name=t -c user.email=t@t commit -qm init)
  [ "$CS" = 1 ] && HOSTARG=(--host "unix://$P/s.sock")
}
ac() {
  (cd "$P/proj" && env -i PATH=/usr/bin:/bin HOME="$P/home" XDG_CONFIG_HOME="$P/home/.config" \
    XDG_DATA_HOME="$P/home/.local/share" XDG_CACHE_HOME="$P/home/.cache" CRUSH_GLOBAL_CONFIG="$P/home/.config/agent-cli" \
    CRUSH_GLOBAL_DATA="$P/home/.local/share/agent-cli" CRUSH_CACHE_DIR="$P/home/.cache/agent-cli" AGENT_CLI_RUNTIME_DIR="$P/rt" \
    GIT_CEILING_DIRECTORIES="$OUT" CRUSH_DISABLE_METRICS=1 DO_NOT_TRACK=1 CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1 \
    CRUSH_CLIENT_SERVER=$CS CRUSH_SERVER_IDLE_TIMEOUT=60 timeout 60 "$BIN" "${HOSTARG[@]}" "$@")
}
finish() {
  local pid; for pid in $(pgrep -f -- "server --host unix://$P/s.sock\$"); do kill -TERM "$pid" 2>/dev/null; done
  kill "$FAKE" 2>/dev/null; wait "$FAKE" 2>/dev/null
}
sid() { python3 -I -c 'import json,sys
try: print(json.load(sys.stdin).get("session_id",""))
except Exception: print("")'; }
project_qq() { printf '%s' '{"models":{"large":{"provider":"qq","model":"m-qq"},"small":{"provider":"qq","model":"m-qq"}}}' > "$P/proj/crush.json"; }
inject() { # inject <session>: u2 + tin giữ chỗ đã đóng (canceled), rồi u3 + tin giữ chỗ còn mở
  python3 -I - "$P/proj/.crush/crush.db" "$1" <<'PY'
import json, sqlite3, sys, time, uuid
db, sid = sys.argv[1], sys.argv[2]
c = sqlite3.connect(db, timeout=10)
t = int(time.time()) + 5
def row(role, parts, model, provider, at):
    i = str(uuid.uuid4())
    c.execute("INSERT INTO messages (id, session_id, role, parts, model, provider, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?)",
              (i, sid, role, json.dumps(parts), model, provider, at, at))
    return i
for k, closed in ((0, True), (2, False)):
    u = row("user", [{"type": "text", "data": {"text": "queued prompt"}}, {"type": "finish", "data": {"reason": "stop", "time": t + k}}], "", None, t + k)
    parts = [{"type": "finish", "data": {"reason": "canceled", "time": t + k + 1, "message": "User canceled request"}}] if closed else []
    row("assistant", parts, u, "agent-cli-pending", t + k + 1)
c.commit()
print(c.execute("SELECT role, provider, model FROM messages WHERE session_id = ? ORDER BY created_at", (sid,)).fetchall())
PY
}
nopending() { ! grep -q 'agent-cli-pending' "$1"; }

for mode in local:0 server:1; do
  name=${mode%%:*}; begin "$name" "${mode##*:}"
  x=$(ac run --json "R-X1 on pp" 2>>"$P/err" | sid)
  project_qq
  y=$(ac run --json "R-Y1 on qq" 2>>"$P/err" | sid)
  [ -n "$x" ] && [ -n "$y" ]; check "$name: setup X on pp, Y on qq" $?
  inject "$x"; inject "$y"
  ac run --json -s "$x" "R-X2 continue X" > "$P/x2.json" 2> "$P/x2.err"
  ac run --json -s "$y" "R-Y2 continue Y" > "$P/y2.json" 2> "$P/y2.err"
  echo "$name: X2 stderr: $(grep 'last ran on' "$P/x2.err")"
  echo "$name: Y2 stderr: $(grep 'last ran on' "$P/y2.err")"
  [ "$(grep -c 'last ran on pp/m-pp; continuing on the project model qq/m-qq' "$P/x2.err")" = 1 ] && nopending "$P/x2.err"
  check "$name: X (pp reply, then placeholders) notes pp/m-pp once, no agent-cli-pending" $?
  ! grep -q 'last ran on' "$P/y2.err"
  check "$name: Y (qq reply, then placeholders) prints no note" $?
  finish
done
echo "REPRO: $([ "$FAILS" = 0 ] && echo PASS || echo "FAIL ($FAILS)")"
[ "$FAILS" = 0 ]
```
</details>

## Sau soát (điều phối, 19/10)

- Soát độc lập: **DAT** (rà thêm herdr, sessions panel, `ui.go:2070`, `run.go:748`, `app.go:459`, stats, session show, title: không lỗi
  người dùng thấy, trừ herdr — ghi cho P1-a). Commit `92182451` trên `v3-candidate`.
- Tiếp nối (đang làm): helper `message.LastReply` cho 3 chỗ; `GetLastAssistantMessage` SQL loại placeholder (gỡ bẫy cho caller
  upstream sau merge); sửa comment `model_restore.go:11`.
- Ghi cho P1-a (chưa sửa, suy từ mã): pane herdr có thể kẹt "working" khi TUI chế độ server gửi prompt RunID rỗng và lượt hỏng
  trước `sessionAgent.Run` (không phát `RunComplete`).
