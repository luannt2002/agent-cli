# INTEG2: một helper `message.LastReply` cho "lời trả lời thật cuối", gỡ bẫy `GetLastAssistantMessage`

Worktree `/home/luannt/application/opensource/agent-cli-cand`, nhánh `v3-candidate`, HEAD `92182451` (bản INTEG1 đã commit). Chưa commit, chưa push, không dùng `git stash`, không đổi index.

Patch `INTEG2-lastreply.patch` gồm `git diff` cộng `git diff --no-index /dev/null internal/message/last_reply_test.go` (tệp mới, chưa track). 8 tệp:

| Tệp | Dòng (+/−) |
|---|---|
| `internal/app/app.go` | +7/−9 |
| `internal/cmd/run.go` | +4/−9 |
| `internal/cmd/run_reply_model_test.go` | +9/−27 |
| `internal/db/messages.sql.go` | +1/−0 |
| `internal/db/sql/messages.sql` | +1/−0 |
| `internal/message/content.go` | +11/−0 |
| `internal/ui/model/model_restore.go` | +19/−21 |
| `internal/message/last_reply_test.go` (mới) | +71 |

Kiểm patch:

- `sha256sum -c INTEG2-lastreply.patch.sha256` → `5dbb4fa9d5e9e20828a9e6457fde1985642c199b39f220452130af24342db22e`, OK.
- `git apply` lên bản `git archive 92182451` sạch → rc=0. Sau đó so `cmp` cả 8 tệp với worktree: trùng hết.

## 1. Thay đổi

1. **`message.LastReply(msgs []Message) (Message, bool)`** nằm trong `content.go`, cạnh `IsPromptPlaceholder`. Hàm lùi từ cuối và lấy tin đầu tiên thoả `Role == Assistant && !IsSummaryMessage && Model != "" && !IsPromptPlaceholder(m)`.

   Ba chỗ gọi:
   - TUI `sessionModelNote`: vòng lặp được thay bằng `LastReply`. Thân hàm lùi một mức thụt lề, nên diff 19/21 dòng. Hành vi không đổi ngoài việc thêm `Model != ""`.
   - `app.sessionModelNote`: `List` + `LastReply`.
   - `cmd.lastReplyModel`: còn 2 dòng, gọi `LastReply`. `protoReplies` giữ nguyên, chỉ sửa comment.

   Cả 3 chỗ giờ cùng một ngữ nghĩa. Khác với INTEG1: TUI và local giờ cũng bỏ qua tin assistant có `Model == ""` và lùi tiếp về lời trả lời có model. Trước đây hai chỗ này dừng ở tin đó và im lặng. Cả 4 chỗ tạo tin assistant đều đặt `Model`, nên khác biệt chỉ chạm dữ liệu cũ.
2. **SQL `GetLastAssistantMessageBySession`** thêm `AND COALESCE(provider, '') != 'agent-cli-pending' -- message.PendingReplyProvider: prompt chưa chạy`, đúng dạng P1-a dùng ở `stats.sql`.
   - Đã sửa cả `internal/db/sql/messages.sql:80` lẫn **sửa tay** `internal/db/messages.sql.go:94`. Máy không có `sqlc` (`which sqlc` rỗng).
   - Đối chiếu bằng script: thân truy vấn sau dòng `SELECT` ở hai tệp trùng nhau từng ký tự (`same body: True`).
   - Hàm vẫn không có caller trong mã chạy. Giờ nó không trả tin giữ chỗ nữa, nhưng vẫn KHÔNG lọc `Model == ""` như `LastReply`. Ngoài brief nên không thêm.
3. **Comment** `model_restore.go:11`: đổi "tin assistant cuối" thành "lời trả lời thật cuối … (message.LastReply)". Comment của `app.sessionModelNote` cũng được sửa: lý do cũ ("GetLastAssistantMessage trả cả tin giữ chỗ") không còn đúng sau mục 2.

## 2. Test

- **`internal/message/last_reply_test.go`** (mới):
  - `TestLastReply`, 5 ca: không có tin; chỉ có tin giữ chỗ; tin giữ chỗ sau lời trả lời; bản tóm tắt sau lời trả lời; tin không có model sau lời trả lời. Hai ca giữa chuyển từ `TestLastReplyModelSkipsPromptPlaceholders` sang; ba ca còn lại là mới.
  - `TestGetLastAssistantMessageSkipsPromptPlaceholders`: SQLite thật. Lời trả lời pp/m-pp, rồi u2, rồi tin giữ chỗ; `created_at` đặt rõ 1000/1001/1002. Hàm phải trả pp/m-pp.
- **`internal/cmd/run_reply_model_test.go`**: giữ tên `TestLastReplyModelSkipsPromptPlaceholders`, thân chỉ còn đường server (`protoReplies` chép đủ trường để `LastReply` nhận ra tin giữ chỗ). Các ca trên `message.Message` đã chuyển sang `TestLastReply`. **Lệch nhỏ so với brief:** tôi không xoá hẳn test này, vì phần proto chỉ kiểm được trong gói `cmd`. Giữ tên cũng giúp suite so theo tên không mất tên nào.
- 12 ca tích hợp của INTEG1 giữ nguyên, không sửa dòng nào.

**ĐỎ.**

1. Trước khi có `LastReply`: `go test ./internal/message/` → rc=1, `undefined: LastReply`.
2. Có `LastReply`, SQL chưa sửa: `TestLastReply` 5/5 PASS; `TestGetLastAssistantMessageSkipsPromptPlaceholders` FAIL đúng lý do (`expected: "pp"`, `actual: "agent-cli-pending"`).

**XANH.** `-run 'TestLastReply|TestGetLastAssistantMessageSkipsPromptPlaceholders|TestOpeningSessionKeepsTheWorkspaceModel|TestContinuedRunKeepsTheWorkspaceModel|TestSessionModelNote'` trên `message`, `ui/model`, `cmd`, `app` → rc=0, 33 dòng PASS, 0 FAIL.

**Kiểm răng.** Mỗi đột biến xong thì chép lại tệp từ bản giữ; `sha256sum -c` OK:

| Đột biến | Kết quả |
|---|---|
| `LastReply` bỏ `&& !IsPromptPlaceholder(m)` | rc=1. 15 ca FAIL: `TestLastReply/{placeholder_after_a_reply,only_a_placeholder}`, cả 12 ca tích hợp (4 TUI, 4 CLI, 4 local) và `TestLastReplyModelSkipsPromptPlaceholders` |
| `LastReply` bỏ `&& m.Model != ""` | rc=1. `TestLastReply/no_model_after_a_reply` FAIL |
| `messages.sql.go` bỏ dòng `agent-cli-pending` (chuỗi SQL thật sự chạy) | rc=1. `TestGetLastAssistantMessageSkipsPromptPlaceholders` FAIL |

Sửa riêng `.sql` không đổi hành vi lúc chạy (đó chỉ là đầu vào của sqlc). Vì vậy hai tệp được giữ khớp nhau bằng phép đối chiếu ở mục 1.2.

## 3. Lệnh đã chạy

Môi trường: 19/10, `flock /tmp/claude-1000/agent-go.lock`, `GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off`, `-p 2`, RAM trống ≥ 8,3 GB trước mỗi lệnh Go.

| Lệnh | Kết quả |
|---|---|
| `go test -count=1 -json ./internal/message/ ./internal/ui/model/ ./internal/cmd/ ./internal/app/ ./internal/db/...` | rc=0; 5 gói `pass` |
| So bộ test theo TÊN: cùng lệnh trên `git archive 92182451` | gốc 773, sau 780. **0 mất, 0 fail ở cả hai, 0 đổi trạng thái.** 7 tên mới pass: `TestLastReply` + 5 ca con và `TestGetLastAssistantMessageSkipsPromptPlaceholders`. Theo gói: message 46→53; app 57, cmd 220, db 12, model 438 không đổi |
| `go build` (CGO_ENABLED=0, GOEXPERIMENT=greenteagc) → `crush2` sha256 `a7c2167e09067553…` | rc=0 |
| `bash scripts/eval/session-model.sh /home/luannt/.cache/v3ev/integ2-lr-sm <crush2>` | rc=0, `SESSION-MODEL: PASS` (11 ok) |
| `repro.sh` của INTEG1 (`INTEG-p1a-p1c.md` mục 6) trên `/home/luannt/.cache/v3ev/integ2-lr-repro` | rc=0, `REPRO: PASS` (6 ok). X báo `last ran on pp/m-pp; continuing on the project model qq/m-qq` ở cả local lẫn server; Y không báo |
| `AGENT_CLI_BIN=<crush2> bash scripts/test-agent-cli-run-json.sh` (`lastReplyModel` của `run --json` đã đổi) | rc=0, `PASS`, 39 ok |
| `go vet ./internal/message/ ./internal/db/ ./internal/app/ ./internal/cmd/ ./internal/ui/model/` | rc=0 |
| `$(go env GOROOT)/bin/gofmt -l` các tệp `.go` đã đổi; `git diff --check` | rỗng; rc=0 |

## 4. Đã chứng minh / Mới nghi / Chưa kiểm

**Đã chứng minh:**

- Cả 3 chỗ báo model dùng một helper, cùng một ngữ nghĩa.
- Bỏ điều kiện tin giữ chỗ trong helper thì cả test đơn vị lẫn 12 ca tích hợp đều đỏ.
- `GetLastAssistantMessage` không còn trả tin giữ chỗ, có test SQLite thật làm cổng.
- E2E session-model, repro và run-json đều xanh trên binary mới.

**Mới nghi:**

- `GetLastAssistantMessage` vẫn khác `LastReply` ở chỗ không lọc tin có `Model == ""`. Hiện không có caller nên vô hại. Nếu có người dùng lại, nên thêm `AND COALESCE(model, '') != ''`.
- `sqlc generate` sau này sẽ sinh lại `messages.sql.go` từ `.sql`. Hai tệp đã khớp nên chưa thấy rủi ro, nhưng chưa chạy sqlc thật.

**Chưa kiểm:**

- TUI chưa chạy trên màn hình thật.
- Chưa chạy test toàn kho, `-race`, `task lint`.
- Không thêm CHANGELOG, vì việc này là refactor cộng gỡ bẫy trong bản chưa phát hành.
