# P3b: lỗi ghi sổ ghi LLM không còn im lặng (T15, quyết định D2)

Worktree `/home/luannt/application/opensource/agent-cli-p3b`, nhánh `v3-poc-p3b-ledger`, gốc `v2-final` (`7cba42ff`). Chưa commit, chưa push.
**Bản hiện hành = sau Vòng 2 (mục 9).** Mục 1–8 là Vòng 1. Chỗ nào Vòng 2 thay đổi so với Vòng 1 thì mục 9 thắng.
Patch: `P3b-ledger.patch`, 20 tệp, +496/−18 (`git apply --stat`). Patch gồm tệp mới `scripts/eval/ledger.sh` (nối bằng `git diff --no-index`, không đụng index). Không gồm 6 tệp phơi nhiễm `??` (5 `*_v3exposure_test.go` + `scripts/eval/exposure.sh`); `grep -c "v3exposure\|exposure.sh"` trên patch = 0.
`sha256sum -c P3b-ledger.patch.sha256` → `e8ea61428de2cdedfd8fbddd6f87d360808e6d2490ee273ba02f81dc16ea1ef8`.
`git apply --check` trên bản xuất sạch của `7cba42ff` (`git archive`, thư mục `~/.cache/v3ev/p3b/apply-check-r2`) cho rc=0. Áp xong, từng tệp khớp worktree (`cmp`), kể cả `ledger.sh`.

## 1. Vấn đề

- `internal/agent/llm_ledger.go:88` (gốc): `_, _ = f.Write(...)`. Sổ bật (`AGENT_CLI_LLM_LEDGER`) mà ghi lỗi (đĩa đầy; `/dev/full` cho ENOSPC) thì không có tín hiệu nào ở lượt nào. Sổ thiếu dòng nhưng vẫn trông như bằng chứng "phiên chỉ dùng provider X".
- Tái hiện ĐỎ trên mã gốc: `TestV3Exposure_T15_LedgerFailureIsVisible` → `write-fails-dev-full turn 1/2: ledger_warn_logs=0 visible=false`, `T15 EXPOSED`.
- Ca "không mở được tệp → cảnh báo một lần" (`llm_ledger.go:85`) là thiết kế, KHÔNG đổi.

## 2. Thiết kế (D2: thấy được, không dừng lượt)

- **Theo dõi theo lượt.** `ledgerWriteErrs` (map session ID → lỗi ghi đầu tiên của lượt, dưới `ledgerMu` sẵn có).
  - `writeLedger`: lần ghi lỗi đầu tiên của session trong lượt thì ghi lại lỗi và log WARN `Failed to write the LLM ledger; calls of this turn are missing from it` (path, session_id, role, error). Các lần lỗi sau cùng lượt thì im.
  - Lời gọi không gắn session (đặt tiêu đề: ctx của nó lấy trước khi `Run` gắn session) thì WARN mỗi lần. Mỗi session mới chỉ có một lời gọi như vậy.
  - `sessionAgent.Run`, khi lượt thành active: `resetLedgerTurn(sessionID)`.
- **Mang lỗi ra ngoài.**
  - `notify.RunComplete.LedgerError` được điền ở cả hai chỗ dựng RunComplete có kết quả: defer của `Run` và `handOff`, tức lượt trước của một prompt đang xếp hàng.
  - Qua server: `proto.RunComplete.LedgerError` (`ledger_error,omitempty`), chép ở `server/events.go` và `workspace/client_workspace.go`, để giữ cầu nối đối xứng.
- **`run --json`.** Trường `ledger_error` (omitempty) đi qua `sessionUsage`, cùng đường với `provider`/`model`.
  - Local: `agent.LedgerWriteError(sessionID)`, đọc cùng tiến trình.
  - Qua server: `runStream` giữ `LedgerError` của RunComplete khớp RunID. `runNonInteractive` có thêm tham số `onLedgerError` (gọi trong defer). `loop.go` và 3 lời gọi trong `run_interrupt_test.go` truyền `nil`.
- **Không dừng lượt.** Không có `error`, mã thoát vẫn 0.
- **Lý do làm cả đường server:** launcher mặc định chạy qua server (README), và README hướng dẫn bật sổ cho server. Nếu chỉ làm local thì đường mặc định vẫn để script mù.

## 3. TDD

| Bước | Lệnh (qua `scratchpad/p3b/g.sh` = flock + RAM ≥3000 MB + `GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off`) | Kết quả |
|---|---|---|
| ĐỎ T15 (mã gốc) | `go test -p 2 -count=1 -run TestV3Exposure_T15 -v ./internal/agent/` | FAIL, `[write-fails-dev-full-1 write-fails-dev-full-2]` |
| ĐỎ test mới (mới thêm field rỗng, chưa có hành vi) | `-run 'TestLedgerWriteFailureIsReportedOncePerTurn\|TestV3Exposure_T15' ./internal/agent/`; `-run 'TestRunJSONReportsLedgerError\|TestRunStream_KeepsTheLedgerErrorOfItsOwnRun' ./internal/cmd/` | cả 4 FAIL ở assert (không phải lỗi biên dịch) |
| XANH | cùng lệnh | PASS. T15: `write-fails-dev-full turn 1/2: ledger_warn_logs=1 visible=true`; `unwritable-dir turn 2: visible=false` vẫn như cũ (thiết kế, không vào phán quyết; test phơi nhiễm KHÔNG sửa) |

Test mới:
- `internal/agent/llm_ledger_test.go`:
  - `TestLedgerWriteFailureIsReportedOncePerTurn`: hai lượt, mỗi lượt 2 lời gọi đều ghi lỗi. Kỳ vọng đúng 1 WARN mỗi lượt, `RunComplete.LedgerError` và `LedgerWriteError` có `no space left on device`, lượt không lỗi. Đối chứng tệp ghi được: 0 WARN, không có lỗi, sổ đủ 4 dòng.
  - `TestLedgerWriteErrorReachesRunCompleteOfAHandedOffTurn`: prompt xếp hàng; cả lượt trước (do `handOff` phát) lẫn lượt xếp hàng đều mang lỗi. Test này thêm SAU khi chạy suite đầy đủ; xem mục 5.
- `internal/cmd/run_json_test.go` `TestRunJSONReportsLedgerError`: có `ledger_error` mà không có `error`; sổ ổn thì không có trường.
- `internal/cmd/run_stream_test.go` `TestRunStream_KeepsTheLedgerErrorOfItsOwnRun`: RunComplete của RunID khác không được tính.

## 4. E2E bằng binary thật, LLM giả (`scripts/eval/fakellm.py`)

Binary: `~/.cache/v3ev/p3b/crush-fix` (worktree) và `crush-base` (`git archive 7cba42ff`), cùng `CGO_ENABLED=0 go build`.
Kịch bản: `scratchpad/p3b/e2e-ledger.sh <thư-mục-mới> <binary> <repo>`. Mỗi lượt có 1 tool call + 1 trả lời, tức 2 lời gọi main + 1 lời gọi tiêu đề.

| ca | fix: rc / text / `ledger_error` / reqs / dòng sổ / WARN (có sid/không sid) | base |
|---|---|---|
| local-devfull | 0 / DONE-LEDGER / `write /dev/full: no space left on device` / 3 / – / 1/1 | 0 / – / không có / 3 / – / 0/0 |
| local-file | 0 / DONE-LEDGER / không có / 3 / 3 / 0/0 | như fix |
| server-devfull | 0 / DONE-LEDGER / `write /dev/full: …` / 3 / – / 1/1 | 0 / – / không có / – / 0/0 |
| server-file | 0 / DONE-LEDGER / không có / 3 / 3 / 0/0 | như fix |

Kết quả cổng: fix `E2E-LEDGER: PASS`; base `FAIL` (thiếu `ledger_error` và WARN ở cả hai ca devfull).
Hai lượt trên CÙNG một server (`scratchpad/p3b/e2e-ledger-2turns.sh`, `run` rồi `run -s <id>`): fix PASS, cả hai lượt có `ledger_error`, 1 log server, 2 WARN có session + 1 WARN tiêu đề, 5 request. Base FAIL (0 WARN).

## 5. Hồi quy

- `scripts/eval/faults.sh`: `FAULTS: PASS` (11 ca, glm = 0).
- `scripts/eval/qwen-only.sh`: `QWEN-ONLY: PASS`. Sổ ghi khớp số request phía server theo từng vai trò: qwen title 2, main 7, subagent 1, agentic_fetch 1, summary 1. `out.json` không có `ledger_error`.
- `scripts/eval/workspaces.sh`: `WORKSPACES: PASS`.
- `AGENT_CLI_BIN=…/crush-fix bash scripts/test-agent-cli-run-json.sh`: rc=0, 39 dòng `ok`, `PASS`.
- `go test -race -json -count=1 -p 2 ./internal/agent/... ./internal/cmd/...`:
  - 0 `DATA RACE`; 1171 pass / 9 skip.
  - 6 fail, đều là test phơi nhiễm khác (T12, C9) đã đỏ ở baseline.
- **Suite đầy đủ theo TÊN** (`go test -json -count=1 -p 2 ./...`, có 6 tệp phơi nhiễm ở cả hai phía):
  - Baseline lấy bằng `git stash` (chỉ tệp tracked) trên cùng worktree rồi `stash pop`; sau pop, `git diff` trùng từng byte với bản trước.
  - Baseline 4769 tên, fix 4774. Chỉ 6 tên khác: +5 tên mới đều pass, và `TestV3Exposure_T15_LedgerFailureIsVisible` fail→pass. Không tên nào pass→fail, không tên nào biến mất.
  - Fail còn lại ở cả hai phía: T12, C9 (tools, shell), T1 (backend).
- **Sau suite** chỉ thêm `TestLedgerWriteErrorReachesRunCompleteOfAHandedOffTurn` (thuần test) và một dòng trống do gofumpt.
  - Chạy lại `./internal/agent/` đầy đủ: 385 pass, fail chỉ có T12.
  - Test mới chạy `-race -count=3`: PASS.
- `go vet` 5 gói đã đổi: sạch.
- `gofmt -l`: sạch. `gofumpt -l` trên tệp đã đổi chỉ còn báo `run_stream_test.go:88`, đoạn mã CÓ SẴN, không đụng.

## 6. Kiểm răng (hoàn từng phần của bản sửa → phải đỏ; mỗi lần khôi phục xong thì `cmp` bằng bản giữ)

| Đột biến | Kết quả |
|---|---|
| M1 nuốt lỗi ghi lại (`err != nil && false`) | T15 FAIL (`T15 EXPOSED`), `TestLedgerWriteFailureIsReportedOncePerTurn` FAIL (turn 1) |
| M2 bỏ `resetLedgerTurn` | T15 FAIL (lượt 2 im lặng), test mới FAIL (turn 2) |
| M3 bỏ `LedgerError` ở defer của `Run` | `TestLedgerWriteFailureIsReportedOncePerTurn` FAIL |
| M4 bỏ `s.ledgerError = …` trong `runStream` | `TestRunStream_KeepsTheLedgerErrorOfItsOwnRun` FAIL |
| M5 bỏ `res.LedgerError = …` trong `runJSON` | `TestRunJSONReportsLedgerError` FAIL |
| M6 bỏ `LedgerError` ở `handOff` | `TestLedgerWriteErrorReachesRunCompleteOfAHandedOffTurn` FAIL (`run first`) |
| M7 bỏ dây nối của `run --json` ở local (`agent.LedgerWriteError`) và server (`onLedgerError`), build binary `crush-m7` | `e2e-ledger.sh`: FAIL `local-devfull: no ledger_error; server-devfull: no ledger_error`; WARN vẫn có (1/1). Unit test không phủ hai dây này, e2e thì có |

## 7. Tài liệu

- `CHANGELOG.md` `[Unreleased] → ### Sửa lỗi`: một mục mới.
- `README.md` mục Quan sát sâu: `ledger_error` trong danh sách trường của `run --json`; câu về lỗi ghi sổ trong mục sổ ghi.
- `docs/agent-cli-local-setup.md`: mục "Sổ ghi LLM" thêm hành vi khi ghi lỗi và giới hạn; mục "`run --json`" thêm `ledger_error`.

## 8. Giới hạn và giả định (cần người soát / điều phối xem)

- `ledger_error` chỉ tính lời gọi gắn chính session của lượt. Lỗi ghi riêng ở agent con (session con) hay ở lời gọi tiêu đề chỉ có WARN trong log.
  - Lỗi bền (đĩa đầy, `/dev/full`) thì luôn trúng cả lời gọi main của lượt.
  - Lỗi thoáng qua chỉ ở lời gọi con thì không có trường.
- Không mở được tệp (thư mục không ghi được, tệp cũ bị chmod) vẫn như cũ: WARN một lần mỗi tiến trình, không có `ledger_error`. Với tệp cũ còn dòng của lần trước, script có thể đếm nhầm. Ca này đã khai là thiết kế nên không đổi; nếu muốn đổi thì đó là một quyết định riêng.
- Chỉ `run --json` có trường này. `send --json`, `loop --json`, `routine` và `run` dạng chữ chỉ có WARN trong log. Help của `run` (`run.go:73`) không liệt kê trường mới; nó cũng chưa liệt kê `provider`/`model`.
- `ledgerWriteErrs` giữ một mục cho mỗi session có lỗi, tới lượt kế của session đó. Server sống lâu với sổ hỏng thì map lớn theo số session (mỗi mục là một chuỗi ngắn).
- ~~Lần thử lại do coordinator (re-auth) gọi lại `Run`…~~ SAI, sửa ở Vòng 2: `coordinator.go:427` chỉ gọi `run()` một lần (fantasy tự thử lại khi lỗi xác thực, bên trong cùng lượt). Chỗ `Run` thật sự được gọi lại là Stop hook và các lượt tự làm tiếp, xem mục 9.
- Khi lấy baseline ở Vòng 1, tôi `git stash push`/`pop` ngắn trên worktree này. `refs/stash` là ref dùng chung giữa các worktree; `stash@{0}` có sẵn từ trước (`feat/personal-rename-providers`) còn nguyên. Vòng 2 không dùng `git stash`.
- Các thư mục chạy: `~/.cache/v3ev/p3b/` (binary, e2e, apply-check), `scratchpad/p3b/` (g.sh, e2e-*.sh, names.py, log, JSON của suite). Không đụng `~/.config/agent-cli`, không gọi LLM thật.

## 9. Vòng 2 (sau soát độc lập KHONG-DAT)

### 9.1 Sửa bắt buộc

**(1) Lượt tự làm tiếp xoá lỗi của đoạn trước.**
- Lỗi: `resetLedgerTurn` được gọi vô điều kiện mỗi lần vào `Run`. Lượt tự làm tiếp (sau tự tóm tắt, `agent.go` gán `call.continuation = true`; hay khi trả lời bị cắt vì hết output, `next.continuation = true`) vào lại `Run` cùng RunID, nên xoá lỗi của đoạn trước, kể cả lời gọi `summary`. Lỗi thoáng qua vì thế cho `ledger_error` rỗng trong khi sổ thiếu dòng.
- Sửa: chỉ reset khi `!call.continuation`, tức prompt mới. Đặt ngay sau `sessMu.Unlock()`, trong một `if` riêng, chứ không trong khối `if !call.continuation` có sẵn (đang giữ `sessMu`). Lý do: `ledgerMu` bị giữ trong lúc ghi đĩa, không nên bắt việc nhận lượt của session chờ đĩa. Ngữ nghĩa giống hệt cách soát đề nghị.
- Test ĐỎ trước: `TestLedgerWriteErrorSurvivesTheTurnsOwnContinuation`.
  - Script `["cut","done"]`.
  - `ledgerFiles[path]` đặt sẵn là `/dev/full`; wrapper `fixLedgerBeforeSecondCall` đổi sang tệp thường ngay trước lời gọi thứ hai.
  - Kiểm: model nhận 2 lời gọi `work`, sổ có đúng 1 dòng, `RunComplete.LedgerError` có `no space left on device`.
  - Trước khi sửa FAIL: `"" does not contain "no space left on device"`. Sau khi sửa PASS.

**(2) Dây nối chưa có lớp bảo vệ.**
- `TestRunCompleteToProto_RoundTrip` (`server/events_test.go`): thêm `LedgerError` vào nguồn và assert sau khi qua SSE.
- Harness SSE `TestRunQueuedBehindOtherTurnLeavesItsPromptsAlone` (`run_interrupt_test.go`): RunComplete mang `LedgerError`; `runNonInteractive` nhận `onLedgerError` khác nil.
  - Ca có RunComplete thì assert bằng giá trị đã phát.
  - Ca bỏ dở vì câu hỏi thì assert rỗng: callback vẫn được gọi, và không có RunComplete của lượt.
- Thêm `TestTranslateEvent_RunCompleteKeepsLedgerError` (`workspace/client_workspace_test.go`) cho chiều proto → domain.
- Thêm `scripts/eval/ledger.sh <thư-mục-mới> <binary>`, cạnh `faults.sh`.
  - Binary thật, LLM giả. Bốn ca: local-devfull, local-file, server-devfull (hai lượt trên cùng server: `run` rồi `run -s`), server-file.
  - Thoát 1 khi một lượt không chạy sạch, thiếu `ledger_error`, số WARN có session khác số lượt, hoặc sổ của tệp thường có số dòng khác số request.
  - Đây là lớp duy nhất phủ các closure trong `RunE` (`run.go`: usage qua server, callback, usage local).

### 9.2 Nên (đã làm)

- **#5:** `send --wait --json` có `ledger_error` (`send.go`, `waitForRun`: `res.LedgerError = p.LedgerError`). Test `TestSendWaitReportsLedgerError` cũng kiểm RunComplete của run khác không được tính. Trước khi sửa FAIL (`expected … actual ""`).
- **#6:** guide (mục sổ ghi LLM) và README (Quan sát sâu) ghi rõ những đường chỉ có WARN trong log: `send` không `--wait`, `loop`, `routine`, `run` dạng chữ; khi đĩa đầy thật, log cũng có thể không ghi được. Câu cũ "đếm từ sổ chỉ tin được khi không có trường" đổi thành:
  - Có `ledger_error` thì đừng dùng sổ làm bằng chứng cho lượt đó.
  - Không có `ledger_error` vẫn chưa chắc sổ đủ, kèm danh sách giới hạn bên dưới.
- **#7:** help của `run` (khối ví dụ) và mô tả cờ `--json` liệt kê `provider`, `model`, `ledger_error`. Không có golden nào chứa chuỗi này (`grep -rl`).
- **#4 (ghi giới hạn):** Stop hook chặn rồi chạy lại (`stop_hook.go`, `RunAccepted` → `Run` với `continuation=false`) xoá lỗi của lần chạy trước. Gate giữ RunComplete MỚI NHẤT theo RunID, nên lỗi thoáng qua chỉ ở lần trước bị mất khỏi `ledger_error`; WARN vẫn có trong log. Cách sửa rẻ (CHƯA làm, cần test với Stop hook): `stopGate.hold` giữ `LedgerError` cũ khi bản mới rỗng.
- **#3 (ghi giới hạn, không sửa):** nhánh thường nhả session (`a.activeRequests.Del`) trước defer và `handOff` đọc lỗi.
  - Một prompt MỚI vào cùng session đúng trong khe đó sẽ reset và xoá lỗi trước khi lượt cũ báo. Không có điểm móc để dựng test tất định cho khe này.
  - Có ba chỗ nhả session: nhánh thường, `afterError`, và trước tự tóm tắt. Muốn chụp lỗi trước mỗi chỗ thì phải thêm trường vào `SessionAgentCall` và sửa cả ba chỗ.
  - Đã ghi vào guide (dòng "Hiếm: …").
- Comment của `ledgerWriteErrs`, `resetLedgerTurn`, `LedgerWriteError` đổi "lượt" thành "prompt (gồm lượt tự làm tiếp)". CHANGELOG ghi `send --wait --json` và `scripts/eval/ledger.sh`.

### 9.3 Kiểm răng Vòng 2 (mỗi đột biến một dòng; khôi phục xong `git diff` trùng từng byte với bản trước đột biến)

| Đột biến | Kết quả |
|---|---|
| R1 reset vô điều kiện (`if true`) | `TestLedgerWriteErrorSurvivesTheTurnsOwnContinuation` FAIL |
| R2 bỏ `LedgerError` ở `wrapEvent` (`server/events.go`) | `TestRunCompleteToProto_RoundTrip` FAIL |
| R3 bỏ `LedgerError` ở `translateEvent` (`client_workspace.go`) | `TestTranslateEvent_RunCompleteKeepsLedgerError` FAIL |
| R4 `runNonInteractive` không gọi `onLedgerError` | `TestRunQueuedBehindOtherTurnLeavesItsPromptsAlone` FAIL (cả hai assert) |
| R5 `runStream` bỏ `s.ledgerError = …` | harness SSE + `TestRunStream_KeepsTheLedgerErrorOfItsOwnRun` FAIL |
| R6 `send` bỏ `res.LedgerError = …` | `TestSendWaitReportsLedgerError` FAIL |
| R7 `RunE` bỏ dây usage (`ledgerErr[:0]`, `agent.LedgerWriteError(id)[:0]`), binary `crush-r7b` | `ledger.sh`: `FAIL local-devfull: ledger_error missing…; server-devfull: ledger_error missing…`; WARN vẫn có. (Lần build R7 đầu lỗi biên dịch, biến không dùng, nên không tính; thư mục `r2-ledger-r7` của lần đó còn lại.) |

### 9.4 Chạy lại

- **Test mới:** đỏ → xanh như 9.1 và 9.2.
- **`scripts/eval/ledger.sh`:** binary V2 `crush-base` (`git archive 7cba42ff`) cho `LEDGER: FAIL`: thiếu `ledger_error` và 0 WARN ở cả local-devfull lẫn server-devfull. Binary cuối `crush-fix3` cho `LEDGER: PASS`.
  - local-devfull: rc 0, `ledger_error` có, 3 request, WARN 1/1.
  - server-devfull: hai lượt đều rc 0 và đều có `ledger_error`, 5 request, WARN 2/1.
  - Các ca tệp thường: không có trường; số dòng sổ = số request (3, 5).
- **Hồi quy với `crush-fix3`:**
  - `faults.sh`: PASS.
  - `qwen-only.sh`: PASS. Sổ khớp server: qwen title 2, main 7, subagent 1, agentic_fetch 1, summary 1.
  - `workspaces.sh`: PASS.
  - `test-agent-cli-run-json.sh`: 39 `ok`, PASS.
- **`-race` cho `./internal/agent/... ./internal/cmd/... ./internal/server/...`:** 0 `DATA RACE`; 1271 pass / 9 skip. Fail chỉ có T12, C9: đã đỏ ở baseline.
- **Suite theo TÊN:**
  - Baseline lấy mới, không stash: `git archive 7cba42ff` + chép 6 tệp phơi nhiễm vào `~/.cache/v3ev/p3b/base-git`, `git init` + commit riêng (repo cô lập, không đăng ký worktree). Chạy `go test -json -count=1 -p 2 ./...`.
  - Baseline 4769 tên, bản sửa 4778 tên. Chỉ khác: +9 tên mới đều pass; `TestV3Exposure_T15_LedgerFailureIsVisible` fail→pass. Không tên nào pass→fail hay biến mất.
  - Baseline Vòng 2 trùng hoàn toàn baseline Vòng 1 (0 tên khác).
  - Fail còn lại ở cả hai phía: T12, C9 (tools, shell), T1 (backend).
- **Kiểm tĩnh:** `go vet` 5 gói đã đổi: sạch. `gofmt -l`: sạch. `gofumpt -l` chỉ báo `run_stream_test.go:88`, mã có sẵn, không đụng.
- **Không đụng:** `refs/stash` (vẫn chỉ `stash@{0}` của người dùng), danh sách worktree, `~/.config/agent-cli`. Không gọi LLM thật.

### 9.5 Còn mở (người soát / điều phối quyết)

- #3 và #4 ở trên: đã ghi giới hạn, chưa sửa.
- `CLAUDE.md` mục P5 chưa liệt kê `scripts/eval/ledger.sh`. Tôi không sửa `CLAUDE.md`; đề nghị thêm vào dòng "Cô lập provider, tiêm lỗi".
- Môi trường phiên báo thư mục làm việc chính là `agent-cli-cand`. Tôi làm theo lệnh điều phối: chỉ worktree `agent-cli-p3b`, không đụng `agent-cli-cand`.

## Sau soát (điều phối, 19/10)

- Soát độc lập: KHONG-DAT (vòng 1) → **DAT** (vòng 2). Điều phối chạy lại: `ledger.sh` V2 FAIL, bản sửa PASS.
- Điều phối thêm dòng CLAUDE.md P5 cho `ledger.sh`. Commit `afaf96d2` trên nhánh `v3-poc-p3b-ledger`; patch cuối thay bản cũ.
- Còn mở (đã ghi giới hạn): race hẹp khi session rảnh trước khi đọc lỗi (#3); Stop hook chạy lại xoá lỗi lần trước (#4).
