# P1c: mở/tiếp session cũ tự ghi model của session thành model cả dự án (T17)

Worktree `/home/luannt/application/opensource/agent-cli-p1c`, nhánh `v3-poc-p1c-modelwrite`, gốc `v2-final` (`7cba42ff`). Chưa commit, chưa push.
**Bản hiện hành = sau Vòng 2 (mục 10).** Mục 1–9 là Vòng 1. Chỗ nào Vòng 2 thay đổi so với Vòng 1 thì mục 10 thắng.
Patch: `P1c-modelwrite.patch`, 13 tệp, +462/−339 (`git apply --stat`). Patch gồm cả `DECISIONS.md` (mục 2026-10-19 do điều phối thêm, giữ nguyên) và tệp mới `scripts/eval/session-model.sh`. Patch không gồm 6 tệp phơi nhiễm `??` (5 `*_v3exposure_test.go` + `scripts/eval/exposure.sh`).
`sha256sum -c P1c-modelwrite.patch.sha256` → `89602afa437f7d6844336e666083adceb6c4cb37dbfc18f21a092dc128fecf84`.
`git apply --check` trên bản xuất sạch của `7cba42ff` cho rc=0. Áp xong, cây khớp worktree (`diff -rq`, trừ 6 tệp phơi nhiễm), và `session-model.sh` giữ quyền thực thi.

## 1. Vấn đề

- **Triệu chứng (T17, tự tái hiện trên binary gốc):** X tạo bằng provider `pp`; dự án đổi sang `qq` (`crush.json` của repo); Y chạy `qq`. `run -s X` (qua server) ghi `models.large=pp` vào `<dự án>/.crush/crush.json` (tầng cấu hình cao nhất). Từ đó lượt sau của Y và session mới đều đi `pp`.
  ```
  T17: EXPOSED — run -s X: X1->['pp'] Y1->['qq'] X2->['pp'] Y2->['pp'] N1->['pp'] | <data>/crush.json before: <absent> | after: models={"large": {"model": "m-pp", "provider": "pp"}} | control send -s X -> NOT_EXPOSED ...
  ```
- **Gốc:** ba hàm "khôi phục model của session":
  - `internal/cmd/run.go:976` `restoreModelFromSession` (server): `UpdatePreferredModel(ScopeWorkspace, …)` = ghi tệp dự án.
  - `internal/ui/model/ui.go:3019` + `model_restore.go:48` (TUI mở session): ghi tệp dự án, trong `tea.Cmd`.
  - `internal/app/app.go:541` (local): chỉ đổi trong RAM (`OverridePreferredModel`), không ghi tệp, nhưng lượt `-s` chạy model cũ của session.
- **Quyết định người dùng (Q4):** model thuộc WORKSPACE, chỉ đổi khi NGƯỜI DÙNG chọn.

## 2. Mọi chỗ ghi/đổi model workspace (số dòng ở `7cba42ff`)

Soát bằng `grep -rn "UpdatePreferredModel\|OverridePreferredModel\|SetConfigField" internal`, rồi đọc từng hàm gọi. Kết quả: 21 chỗ, trong đó 6 chỗ bỏ (4 ghi tệp + 2 đổi RAM).

| # | Chỗ | Kích hoạt | Ghi vào | Loại | Xử lý |
|---|---|---|---|---|---|
| 1 | `cmd/run.go:935` `overrideModels` (large) | `run -m`, `loop -m` (server) | `<dự án>/.crush/crush.json` | tường minh | **giữ** |
| 2 | `cmd/run.go:950` (small) | `run --small-model` (server) | như trên | tường minh | **giữ** |
| 3 | `cmd/run.go:962` small mặc định theo provider của `-m` | hệ quả của `-m` | như trên | tường minh (hệ quả) | **giữ** |
| 4 | `cmd/run.go:1010` `restoreModelFromSession` (large) | `run -s/-C` không `-m` (server) | như trên | **tự ghi khi tiếp session** | **BỎ** (đổi thành dòng báo stderr) |
| 5 | `cmd/run.go:1019` (small, khi chưa có small) | như #4 | như trên | tự ghi | **BỎ** |
| 6 | `cmd/run.go:1050` `overrideReasoningEffort` | `run --reasoning-effort` (server) | như trên (model đang chọn + effort) | tường minh | **giữ**. Trước đây trên `-s` nó chạy SAU #4 (`deferredEffort`) nên ghi model CŨ của session + effort. Nay nó áp cho model của workspace (hoặc của `-m`); bỏ `deferredEffort` vì hết lý do tồn tại |
| 7 | `ui/model/model_restore.go:56` `requestModelRestore` (large) | TUI mở session: `alt+s`, `alt+.`/`alt+,`, `--session`, `--continue`, nạp lại sau reconnect (đều qua `loadSessionMsg`) | như trên | **tự ghi khi mở session** | **BỎ** (đổi thành thông báo ở thanh trạng thái) |
| 8 | `ui/model/model_restore.go:59` (small) | như #7 | như trên | tự ghi | **BỎ** |
| 9 | `ui/model/ui.go:2505` `ActionToggleThinking` | người dùng bật/tắt thinking | như trên | tường minh | **giữ** |
| 10 | `ui/model/ui.go:2817` `ActionSelectReasoningEffort` | `alt+e` / hộp thoại effort | như trên | tường minh | **giữ** |
| 11 | `ui/model/ui.go:3158` `handleSelectModel` | `alt+m`, `alt+p`, onboarding | như trên | tường minh | **giữ** |
| 12 | `ui/model/ui.go:3171` small mặc định khi chưa có | hệ quả của #11 | như trên | tường minh (hệ quả) | **giữ** |
| 13 | `app/app.go:566` `restoreModelFromSession` (large, RAM) | `run -s/-C` local | RAM | tự đổi khi tiếp session | **BỎ** (xem mục 3) |
| 14 | `app/app.go:572` (small, RAM) | như #13 | RAM | tự đổi | **BỎ** |
| 15 | `app/app.go:607,621,629` `overrideModelsForNonInteractive` | `run -m`, `loop -m` local; `routine` (model trong định nghĩa) | RAM | tường minh | **giữ** |
| 16 | `app/app.go:653` `overrideReasoningEffort` | `--reasoning-effort` local | RAM | tường minh | **giữ** (bỏ `deferredEffort` như #6) |
| 17 | `ui/dialog/models.go:558` `recent_models` | chọn trong hộp thoại model | `recent_models` GLOBAL (không phải `models`) | tường minh, lịch sử | **giữ** |
| 18 | `scripts/agent-cli-tmux.sh:146` `new --model` | người dùng | `<worktree>/.crush/crush.json` | tường minh | **giữ** |
| 19 | `cmd/send.go` (`send -s`) | — | không ghi model (đọc lại: không có lời gọi model nào; ca đối chứng T17ctl xác nhận) | — | không đổi |
| 20 | `backend/config.go:80`, `server/config.go:57` | endpoint chuyển tiếp | — | — | không đổi |
| 21 | `agent/agenttest/coordinator.go:58` | test helper | RAM | test | không đổi |

## 3. Thiết kế

- **Tiếp/mở session cũ chạy model nào:** model hiện tại của workspace (hoặc của `-m` nếu có). Không ghi gì, không đổi gì trong RAM.
- **Thông báo (chỉ khi model lượt cuối của session ≠ model lớn của workspace; trùng hoặc session chưa có trả lời thì im):**
  - CLI (`run -s/-C`, cả server lẫn local), stderr, đúng một dòng:
    `session <id> last ran on pp/m-pp; continuing on the project model qq/m-qq (pick another with -m)`.
  - TUI, thanh trạng thái (info):
    `This session last ran on pp/m-pp; continuing on the project model qq/m-qq (alt+m to change)`.
  - So sánh dùng chung một hàm `app.SessionModelNote(last, current)` (`internal/app/app.go`). Mỗi nơi gọi chỉ thêm tiền tố và gợi ý riêng.
- **Bỏ luôn khôi phục trong RAM ở chế độ local.** Không thấy lý do mạnh để giữ:
  - (a) Nếu giữ, cùng lệnh `run -s X` sẽ chạy `pp` ở local nhưng `qq` ở server, tuỳ `CRUSH_CLIENT_SERVER`. Đó là bẫy.
  - (b) Đổi provider giữa session đã được hỗ trợ (`alt+m`, `run -s -m`; README ghi đã thử với LLM thật), nên lịch sử khác provider không phải lý do.
  - (c) Ở server không thể khôi phục "chỉ một lượt" nếu không thêm model theo lượt vào proto, vì cấu hình workspace dùng chung giữa các client. Không thêm trường proto.
- **TUI:** xoá cả bộ `modelRestorer` (hàng đợi ghi lần lượt, lần sau thắng, `modelRestoredMsg`, trường `restorer`) vì không còn gì để ghi. `sessionModelNote` chỉ đọc `m.com.Config()` (bộ nhớ đệm, như mã cũ vẫn đọc trong `Update`) và trả `util.ReportInfo`. Không thêm IO trong `Update`/`View`.
- **Hệ quả phụ có chủ ý:**
  - Theme không còn tự đổi theo provider của session vừa mở (trước gọi `applyThemeForProvider`).
  - Sửa luôn A-03 trong CHANGELOG 0.2.0 ("resume không báo khi không khôi phục được model"): model cũ không còn trong catalog thì vẫn có dòng báo, thay cho log Debug câm (`run.go:999`).

## 4. Test

| Test | Tệp | Kiểm |
|---|---|---|
| `TestContinuedRunKeepsTheWorkspaceModel` (MỚI, 5 ca con) | `internal/cmd/run_session_model_test.go` | `runNonInteractive` qua server giả (httptest), ghi lại mọi `POST /config/model`. `run -s`, `run -C`: 0 lần ghi + đúng 1 dòng stderr nêu cả `pp/m-pp` và `qq/m-qq`. `run -s --reasoning-effort high`: đúng 1 lần ghi `qq/m-qq effort=high` (không phải `pp`). Đối chứng `run -s -m pp/m-pp`: ghi `pp`, không báo. `run -s` khi model trùng: 0 ghi, không báo |
| `TestOpeningSessionKeepsTheWorkspaceModel` (MỚI, 2 ca con) | `internal/ui/model/model_restore_test.go` | Đưa `loadSessionMsg` qua `Update` (đúng đường mở session thật), chạy mọi lá Cmd: 0 lần `UpdatePreferredModel`. Model khác thì có đúng 1 `InfoMsg` nêu cả `p/b` và `p/a`; model trùng thì không có |
| `TestContinuedLocalRunKeepsTheWorkspaceModel` (MỚI) | `internal/app/session_model_test.go` | Lúc ĐỎ: gọi hàm cũ, model trong RAM đổi sang `pp`. Lúc XANH: kiểm chữ của `sessionModelNote`/`SessionModelNote` (khác, trùng, chưa có trả lời) và cấu hình vẫn `qq` |
| `TestModelSelectionWritesWorkspaceScope` (SỬA: bỏ nửa sau) | `model_selection_scope_test.go` | Nửa sau từng khẳng định "mở session ghi ScopeWorkspace", đúng là hành vi bị bỏ theo Q4. Bất biến mới do `TestOpeningSessionKeepsTheWorkspaceModel` giữ. Nửa đầu (`alt+m` ghi ScopeWorkspace) giữ nguyên |
| `TestRestoreModelFromSessionWhileBusyStillRestores`, `TestRestoreModelLatestRequestWins`, `TestRestoreModelQuickBackAndForthEndsOnLastSession` (XOÁ) + helper `availableModels` | `model_restore_test.go` | Test cho bộ `modelRestorer` đã xoá (hàng đợi ghi, A→B→A). Không còn mã để test. **Người soát nên xác nhận ba test này bị xoá vì hành vi bị bỏ theo Q4, không phải vì đỏ** |
| Giữ nguyên, vẫn xanh | `TestModelSwitchAllowedWhileAnySessionBusy`, `TestModelTweaksWriteWorkspaceScope` | `alt+m`, thinking, effort vẫn ghi ScopeWorkspace (đối chứng "tường minh vẫn ghi") |

## 5. Bằng chứng (đều từ lệnh vừa chạy)

Binary: `~/.cache/v3ev/p1c-bin/agent-cli-before` (build từ `7cba42ff` + tệp `??`) và `agent-cli-after` (build 19:26, sau lần sửa mã cuối lúc 19:19; sau đó chỉ đổi CHANGELOG và tên/chú thích trong test). Cả hai báo `v0.2.1-0.20261019085449-7cba42ff3ca4+dirty`.

### ĐỎ, trước khi sửa
```
cmd:  --- FAIL run_-s, run_-C, run_-s_--effort (ghi pp/m-pp); PASS run_-s_-m, run_-s_same_model (đối chứng)
ui:   --- FAIL other_model: "Should be empty, but was [workspace]"; PASS same_model
app:  --- FAIL: "this run uses the workspace model" (actual m-pp/pp)
exposure.sh T17 (before): EXPOSED (dòng ở mục 1)
p1c-e2e.sh (before): fails=5
  local: X1=pp X2=pp X3=pp ; FAIL "run -s X goes to the workspace model qq", FAIL note
  server: X2=pp Y2=pp N1=pp ; data-models after-run-s={"large": {"model": "m-pp", "provider": "pp"}}
p1c-tui.sh (before): after open: data-models={"large": {"model": "m-pp", "provider": "pp"}}  (TUI mở -s X tự ghi pp)
```
Ở binary cũ, ca `alt+m` của `p1c-tui.sh` cũng FAIL. Đây không phải đối chứng hợp lệ: sau khi tự khôi phục, hộp thoại đang ở provider `pp` nên gõ `m-qq` không khớp gì.

### XANH, sau khi sửa
```
unit (worktree): TestContinuedRunKeepsTheWorkspaceModel 5/5 PASS, TestContinuedLocalRunKeepsTheWorkspaceModel PASS,
  TestOpeningSessionKeepsTheWorkspaceModel 2/2 PASS, TestModelSelectionWritesWorkspaceScope/TestModelSwitchAllowedWhileAnySessionBusy/
  TestModelTweaksWriteWorkspaceScope PASS; go vet 3 package: sạch; gofmt -l: rỗng
exposure.sh T17 (after): T17: NOT_EXPOSED — run -s X: X1->['pp'] Y1->['qq'] X2->['qq'] Y2->['qq'] N1->['qq'] | <data>/crush.json before: <absent> | after: models={} | control send -s X -> NOT_EXPOSED ...
p1c-e2e.sh (after): fails=0 (10/10 PASS)
  local:  X1=pp X2=qq X3=pp ; x2.err=session 1e0da26a-… last ran on pp/m-pp; continuing on the project model qq/m-qq (pick another with -m)
  server: X1=pp Y1=qq X2=qq Y2=qq N1=qq X3=pp N2=pp ; data-models before=<absent> after-run-s=<absent>
          after-run-m={"large": {"model": "m-pp", "provider": "pp"}, "small": {"model": "m-pp", "provider": "pp"}}   ← -m vẫn ghi
p1c-tui.sh (after, tmux socket riêng): fails=0
  pane: This session last ran on pp/m-pp; continuing on the project model qq/m-qq (alt+m to change)
  after open: data-models=<absent>
  after alt+m: data-models={"large": {"max_tokens": 1000, "model": "m-qq", "provider": "qq"}}   ← alt+m vẫn ghi
```

### Hồi quy
| Lệnh | Kết quả |
|---|---|
| `scripts/eval/faults.sh ~/.cache/v3ev/p1c-faults <after>` | `FAULTS: PASS`, rc=0 (11 ca, glm=0 ở mọi ca) |
| `scripts/eval/qwen-only.sh ~/.cache/v3ev/p1c-qwen <after>` | `QWEN-ONLY: PASS`, rc=0 |
| `scripts/eval/workspaces.sh ~/.cache/v3ev/p1c-ws <after>` | `WORKSPACES: PASS`, rc=0 |
| `AGENT_CLI_BIN=<after> bash scripts/test-agent-cli-run-json.sh` | `PASS`, rc=0, 39 dòng `ok`, 0 dòng FAIL |
| `go test -race -count=1 -p 2 ./internal/cmd/... ./internal/ui/... ./internal/app/...` | rc=0, 18 package ok, 0 `DATA RACE`, 7 phút 10 giây |
| `go test -json -count=1 -p 2 ./...`, so THEO TÊN với cùng lệnh trên bản xuất sạch `7cba42ff` + 5 test phơi nhiễm (cả hai chạy trong bản sao không có `.git`, cùng môi trường) | gốc 4684 tên, sau 4691. **Đổi kết quả: 0.** Mất: đúng 3 test xoá có chủ ý (mục 4). Mới: 10 (3 test + 7 ca con), đều pass. Top-level fail ở cả hai bên: đúng 5 `TestV3Exposure_*` (T12, T15, C9 tools, T1, C9 shell). Package fail: 4 ở cả hai bên (agent, agent/tools, backend, shell) |

Bộ đầy đủ chạy trên bản sao trước khi đổi tên một ca con (`run_-s_--effort` thành `run_-s_--reasoning-effort`) và chú thích trong test. Ca đổi tên đã chạy lại xanh trên worktree. Race chạy trên worktree trước lần đổi tên đó; mã sản phẩm không đổi.

### Kiểm răng (bản đột biến trong scratchpad, không đụng worktree; chạy 3 test mới)
| Bản | cmd | ui | app |
|---|---|---|---|
| sửa nguyên vẹn | 5/5 PASS | 2/2 PASS | PASS |
| r0: hoàn mã sản phẩm về `7cba42ff` (giữ test mới) | FAIL `run_-s`, `run_-C`, `run_-s_--effort` | FAIL `other_model` | build failed (test gọi hàm mới): răng YẾU, xem dưới |
| m3: bỏ dòng in stderr ở server | FAIL `run_-s`, `run_-C`, `run_-s_--effort` | PASS | PASS |
| m4: TUI không báo (`return nil`) | PASS | FAIL `other_model` | PASS |

**Đường local chỉ có răng ở e2e, không có ở unit.** `TestContinuedLocalRunKeepsTheWorkspaceModel` không đi qua `RunNonInteractive` (dựng cả App là quá nặng). Nếu ai đó thêm lại `OverridePreferredModel` vào `RunNonInteractive`, unit vẫn xanh. Chỉ phần `local` của `p1c-e2e.sh` bắt được: binary gốc chính là bản hoàn đó, cho `local: X2=pp` → FAIL.

## 6. Bảng chân trị (server = chế độ mặc định qua launcher; local = `CRUSH_CLIENT_SERVER=0`)

| Hành động | Trước: ghi `.crush/crush.json`? | Trước: lượt chạy | Sau: ghi? | Sau: lượt chạy | Sau: báo |
|---|---|---|---|---|---|
| `run -s X` / `run -C` (server), X từng chạy pp, workspace qq | CÓ (pp) | pp, và mọi session sau cũng pp | KHÔNG | qq | 1 dòng stderr |
| `run -s X` / `run -C` (local) | không (RAM) | pp | không | qq | 1 dòng stderr |
| `run -s X -m pp/m-pp` (server) | CÓ (pp) | pp | CÓ (pp) | pp | không |
| `run -s X -m pp/m-pp` (local) | không (RAM) | pp | không (RAM) | pp | không |
| `run -s X --reasoning-effort high` (server) | CÓ: pp + effort | pp | CÓ: qq + effort | qq | 1 dòng stderr |
| `run -s X` khi X cũng chạy qq | không | qq | không | qq | không |
| `send -s X` | không | qq | không | qq | không (không đổi) |
| TUI mở X (`-s`, `--continue`, `alt+s`, `alt+.`/`alt+,`) | CÓ (pp, chạy nền) | pp | KHÔNG | qq | thanh trạng thái |
| TUI `alt+m`/`alt+p` chọn model | CÓ | model chọn | CÓ | model chọn | "Large model changed to …" (không đổi) |
| TUI thinking / effort | CÓ | — | CÓ | — | không đổi |

## 7. Lệnh tái lập

Luật máy: mọi lệnh go qua `flock /tmp/claude-1000/agent-go.lock env GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off go … -p 2`, chỉ chạy khi `free -m` available ≥ 3000.
```
W=/home/luannt/application/opensource/agent-cli-p1c; P=reports/v3-research/experiments/poc
cd $W
# unit
go test -p 2 -count=1 -run 'TestContinuedRunKeepsTheWorkspaceModel|TestContinuedLocalRunKeepsTheWorkspaceModel|TestOpeningSessionKeepsTheWorkspaceModel|TestModelSelectionWritesWorkspaceScope|TestModelSwitchAllowedWhileAnySessionBusy|TestModelTweaksWriteWorkspaceScope' ./internal/cmd/ ./internal/app/ ./internal/ui/model/
# binary + phơi nhiễm
CGO_ENABLED=0 go build -p 2 -o ~/.cache/v3ev/<mới>/agent-cli .
bash scripts/eval/exposure.sh ~/.cache/v3ev/<thư-mục-mới> ~/.cache/v3ev/<mới>/agent-cli T17
# e2e local+server và TUI (nội dung ở Phụ lục A, B; lưu thành tệp rồi chạy)
bash p1c-e2e.sh $W ~/.cache/v3ev/<thư-mục-mới> <binary>     # after: fails=0; before: fails=5
bash p1c-tui.sh $W ~/.cache/v3ev/<thư-mục-mới> <binary>     # after: fails=0; before: 3 FAIL
# hồi quy
bash scripts/eval/faults.sh ~/.cache/v3ev/<mới> <binary>; bash scripts/eval/qwen-only.sh ~/.cache/v3ev/<mới> <binary>
bash scripts/eval/workspaces.sh ~/.cache/v3ev/<mới> <binary>; AGENT_CLI_BIN=<binary> bash scripts/test-agent-cli-run-json.sh
go test -race -count=1 -p 2 ./internal/cmd/... ./internal/ui/... ./internal/app/...
# patch
sha256sum -c $P/P1c-modelwrite.patch.sha256
mkdir <dir> && git archive 7cba42ff | tar -x -C <dir> && (cd <dir> && git apply --check <đường>/P1c-modelwrite.patch)
```
Thư mục bằng chứng đã chạy: `~/.cache/v3ev/p1c-{red1,green1,e2e-before,e2e-after,tui-before,tui-after4,faults,qwen,ws}`.

## 8. Rủi ro

- **Đổi hành vi mà người dùng có thể đang dựa vào:**
  - Ai quen "`run -s X` tự quay về model của X" giờ chạy model của dự án.
  - Giảm thiểu: một dòng stderr nêu cả hai model kèm cách đổi (`-m`).
  - Với `--json`, dòng này nằm ở stderr. Script chỉ đọc stdout vẫn thấy model thật qua trường `provider`/`model` của JSON.
- **Dòng báo in ra stderr khi spinner có thể đang chạy** (chỉ khi stderr là TTY và không `--quiet`): dòng có thể lẫn với spinner. Mã cũ cũng in kiểu này ("… is pending on session …"). Chưa kiểm bằng mắt.
- **TUI bỏ đổi theme theo provider** của session vừa mở, vì không còn đổi model.
- **Tệp `model_restore.go` giữ tên cũ** dù chỉ còn `sessionModelNote` + `appliesNextNote` (đổi tên = thêm diff xoá/tạo tệp). Người soát có thể muốn đổi tên.
- **`DECISIONS.md:69` còn mô tả cơ chế cũ** ("khi mở lại session (khôi phục model chạy nền trong `tea.Cmd`, lần lượt, lần sau thắng)"). Không sửa vì đó là bản ghi quyết định cũ. Đề xuất thêm một mục quyết định mới cho Q4 (ngoài phạm vi brief).

## 9. Chưa kiểm

- Đường local chưa có unit test chặn được hồi quy (mục 5, kiểm răng). Chỉ e2e chặn.
- `loop -m` ở server ghi workspace ở MỖI vòng (`loop.go:406` → `overrideModels`). Coi là tường minh, không đổi, không chạy e2e riêng.
- `routine` chạy local với model của định nghĩa (chỉ RAM). Không đụng.
- `alt+p` (đổi provider) trong TUI thật: chỉ có unit `handleSelectModel` (cùng hàm với `alt+m`). tmux chỉ chạy `alt+m`.
- `task lint` (gofumpt/golangci) chưa chạy; chỉ `gofmt -l` và `go vet` cho 3 package.
- Ghi IO trong `Update` của `ActionSelectReasoningEffort` (`ui.go:2817`) và `handleSelectModel` (`ui.go:3158`) có từ trước (D-03), không đụng.
- Không chạy LLM thật.

## 10. Vòng 2 (sau soát độc lập: V1, V2, V3, V5, V4)

### Đã sửa
| Mục | Sửa | Tệp |
|---|---|---|
| V1 | Gợi ý trong TUI chọn theo provider. `alt+m` chỉ liệt kê model của provider đang dùng (`keys.go:112`), nên: khác provider thì gợi ý `(alt+p to change provider)`, cùng provider thì `(alt+m to change model)`. Dòng stderr của CLI giữ `(pick another with -m)` vì `-m` nhận `provider/model` nên đúng ở cả hai ca | `internal/ui/model/model_restore.go` |
| V5 | Chỉ báo khi mở session KHÁC hoặc lúc khởi động (`switched`, đã có sẵn trong `case loadSessionMsg`). Nạp lại đúng session đang xem (reconnect `ui.go:1981`) không báo lại, nên không đè "Reconnected…". Chỉ báo khi model thật sự khác (vốn đã đúng: `SessionModelNote` rỗng khi trùng) | `internal/ui/model/ui.go` |
| V2 | Chuyển ca local + server của e2e vào repo thành `scripts/eval/session-model.sh <thư-mục-mới> <binary>`, cùng khuôn với `faults.sh`/`workspaces.sh`: in `ok`/`FAIL`, dòng `SESSION-MODEL: PASS/FAIL`, thoát 1 khi hỏng, 2 khi sai cách dùng. Không thêm vào `test-agent-cli-run-json.sh` vì script đó dựng một provider giả duy nhất và dùng chung một `crush.json` dự án cho mọi ca, còn kịch bản này cần hai provider và đổi `crush.json` giữa chừng. Unit app: bỏ assert luôn-đúng (cấu hình không đổi sau một hàm chỉ đọc) và stub coordinator; đổi tên `TestContinuedLocalRunKeepsTheWorkspaceModel` → `TestSessionModelNote`, chỉ kiểm chữ của dòng báo | `scripts/eval/session-model.sh` (mới), `internal/app/session_model_test.go` |
| V3 | Viết lại mục CHANGELOG: nêu cả local (trước chạy model của session trong RAM); bỏ câu sai "`--reasoning-effort` mới đổi model"; liệt kê đủ lựa chọn tường minh (`-m`, `--small-model`, `alt+p`/`alt+m`, `agent-cli-tmux new --model`; thinking/effort qua `alt+e`, `--reasoning-effort` đổi tham số của model đang chọn); thêm "model của session không còn trong catalog nay cũng được báo" | `CHANGELOG.md` |
| V4 | Xuất lại patch và `.sha256` SAU CÙNG, gồm `DECISIONS.md` | mục đầu báo cáo |

### ĐỎ → XANH (lệnh vừa chạy)
```
TestOpeningSessionKeepsTheWorkspaceModel, TRƯỚC khi sửa V1/V5 (test đã thêm 2 ca):
  --- FAIL other_provider: "This session last ran on q/x; continuing on the project model p/a (alt+m to change)" does not contain "alt+p"
  --- FAIL reload_of_shown_session: Should be empty, but was [This session last ran on q/x; ...]
  --- PASS other_model, same_model
SAU khi sửa: 4/4 PASS (other_provider, other_model, same_model, reload_of_shown_session)
Unit khác (worktree): TestContinuedRunKeepsTheWorkspaceModel 5/5, TestSessionModelNote, TestModelSelectionWritesWorkspaceScope,
  TestModelSwitchAllowedWhileAnySessionBusy, TestModelTweaksWriteWorkspaceScope 2/2, TestLoadSessionReportsOnlyAfterSwitch,
  TestFailedLoadShowsErrorAndKeepsReportOnShownSession: PASS; go vet 3 package sạch; gofmt -l rỗng
```
Ca `other_provider` dùng provider `q` không có trong cấu hình, nên nó cũng chứng minh "model không còn trong catalog vẫn được báo".

### Script e2e mới `scripts/eval/session-model.sh` (binary `agent-cli-after2`, build 20:03, sau lần sửa mã cuối lúc 20:00)
| Binary | Kết quả |
|---|---|
| `agent-cli-before` (`7cba42ff`) | rc=1, `SESSION-MODEL: FAIL (5)`: local X2=pp, không có dòng báo; server ghi `{"large": pp}` khi `run -s`, X2/Y2/N1=pp, không có dòng báo |
| `agent-cli-after2` | rc=0, `SESSION-MODEL: PASS`, 11/11 ok: local X2=qq + dòng báo + không ghi; `-m` chạy pp. Server không ghi khi `run -s`, X2/Y2/N1=qq + dòng báo; `-m` ghi pp; session mới sau `-m` chạy pp |
| **Kiểm răng đường local:** `agent-cli-mutant-local` = bản sửa đầy đủ + thêm lại khôi phục model trong RAM ở `RunNonInteractive` (5 dòng, chỉ đường local) | rc=1, `SESSION-MODEL: FAIL (1)`: đúng `FAIL local: run -s X runs the project model qq` (X2=pp); mọi ca server vẫn ok |

### Chạy lại (binary `agent-cli-after2`)
| Lệnh | Kết quả |
|---|---|
| `exposure.sh ~/.cache/v3ev/p1c-green2 <after2> T17` | `T17: NOT_EXPOSED` (X2, Y2, N1 → qq; `<data>/crush.json` before = after = `<absent>`) |
| TUI thật, tmux socket riêng (`p1c-tui.sh`, phụ lục B) | fails=0. Thanh trạng thái: `This session last ran on pp/m-pp; continuing on the project model qq/m-qq (alt+p to change provider)`. Mở xong không ghi gì; `alt+m` chọn m-qq ghi `models.large=qq`. Đã mở thử `alt+p` trong tmux: hộp thoại liệt kê `qq current`, `pp ready`, nên gợi ý dùng được |
| `faults.sh`, `qwen-only.sh`, `workspaces.sh` | `FAULTS: PASS`, `QWEN-ONLY: PASS`, `WORKSPACES: PASS` (đều rc=0) |
| `AGENT_CLI_BIN=<after2> bash scripts/test-agent-cli-run-json.sh` | `PASS`, rc=0, 39 `ok`, 0 `FAIL` |
| `go test -race -count=1 -p 2 ./internal/cmd/... ./internal/ui/... ./internal/app/...` | rc=0, 18 package ok, 0 `DATA RACE` (6 phút 54 giây) |
| Suite theo TÊN (`go test -json -count=1 -p 2 ./...` trên bản sao không `.git` của worktree vòng 2, so với gốc `7cba42ff` + 5 test phơi nhiễm) | gốc 4684 tên, sau 4693. **Đổi kết quả: 0.** Mất 3: ba test `TestRestoreModel*` đã xoá ở Vòng 1. Mới 12, đều pass: `TestSessionModelNote` (thay `TestContinuedLocalRunKeepsTheWorkspaceModel` của Vòng 1), `TestContinuedRunKeepsTheWorkspaceModel` + 5 ca con, `TestOpeningSessionKeepsTheWorkspaceModel` + 4 ca con. Top-level fail: đúng 5 `TestV3Exposure_*` như gốc |

### Lệnh tái lập Vòng 2
```
cd /home/luannt/application/opensource/agent-cli-p1c
go test -p 2 -count=1 -run 'TestOpeningSessionKeepsTheWorkspaceModel|TestSessionModelNote|TestContinuedRunKeepsTheWorkspaceModel' ./internal/ui/model/ ./internal/app/ ./internal/cmd/
CGO_ENABLED=0 go build -p 2 -o <bin> .
bash scripts/eval/session-model.sh ~/.cache/v3ev/<thư-mục-mới> <bin>     # PASS rc=0; trên binary gốc FAIL (5) rc=1
```
Bằng chứng: `~/.cache/v3ev/p1c-{sm-before,sm-after,sm-mutant,green2,tui-r2,tui-altp,faults2,qwen2,ws2}`.

### Còn lại sau Vòng 2
- `scripts/eval/session-model.sh` chưa được nhắc trong `CLAUDE.md` P5 hay docs (bảng lệnh kiểm). Đề xuất thêm một dòng khi merge.
- Phụ lục A (`p1c-e2e.sh`) nay đã được thay bằng script trong repo; giữ lại làm lịch sử Vòng 1.
- Chưa kiểm tay trong tmux đường reconnect (chỉ unit `reload_of_shown_session`).

## Phụ lục A: `p1c-e2e.sh`

```bash
#!/usr/bin/env bash
# E2E P1c: tiếp session cũ không ghi model của nó vào cấu hình dự án; lượt chạy model của workspace; -m vẫn ghi.
# Binary thật + LLM giả (scripts/eval/fakellm.py), env cô lập như scripts/eval/exposure.sh.
# Dùng: bash p1c-e2e.sh <repo agent-cli> <thư-mục-mới ≤ 70 byte> <binary>
# In một dòng mỗi kiểm tra (PASS/FAIL) và thoát 1 nếu có FAIL.
set -u
R=$1; OUT=$2; BIN=$3
[ -e "$OUT" ] && { echo "exists: $OUT" >&2; exit 2; }
mkdir -p "$OUT" && OUT=$(cd "$OUT" && pwd)
FAILS=0
check() { if [ "$2" = 0 ]; then echo "PASS $1"; else echo "FAIL $1"; FAILS=$((FAILS + 1)); fi; }

begin() { # begin <ca> <cs 0|1>
  P="$OUT/$1"; CS=$2; HOSTARG=()
  mkdir -p "$P/home/.config/agent-cli" "$P/log" "$P/rt" "$P/proj"; chmod 700 "$P/rt"
  echo '[]' > "$P/rules.json"
  timeout 600 python3 -I "$R/scripts/eval/fakellm.py" "$P/port" "$P/log" "$P/rules.json" >/dev/null 2>&1 &
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
ac() { # ac <lệnh...>: binary trong thư mục dự án với env cô lập
  (cd "$P/proj" && env -i PATH=/usr/bin:/bin HOME="$P/home" XDG_CONFIG_HOME="$P/home/.config" \
    XDG_DATA_HOME="$P/home/.local/share" XDG_CACHE_HOME="$P/home/.cache" CRUSH_GLOBAL_CONFIG="$P/home/.config/agent-cli" \
    CRUSH_GLOBAL_DATA="$P/home/.local/share/agent-cli" CRUSH_CACHE_DIR="$P/home/.cache/agent-cli" AGENT_CLI_RUNTIME_DIR="$P/rt" \
    GIT_CEILING_DIRECTORIES="$OUT" CRUSH_DISABLE_METRICS=1 DO_NOT_TRACK=1 CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1 \
    CRUSH_CLIENT_SERVER=$CS CRUSH_SERVER_IDLE_TIMEOUT=60 timeout 60 "$BIN" "${HOSTARG[@]}" "$@")
}
finish() {
  local pid; for pid in $(pgrep -f -- "server --host unix://$P/"); do kill -TERM "$pid" 2>/dev/null; done
  kill "$FAKE" 2>/dev/null; wait "$FAKE" 2>/dev/null
}
sid() { python3 -I -c 'import json,sys; print(json.load(sys.stdin).get("session_id",""))' 2>/dev/null; }
prov() { python3 -I - "$P/log/requests.jsonl" "$1" <<'PY'
import json, sys
tag = sys.argv[2]
ps = sorted({r["provider"] for r in map(json.loads, open(sys.argv[1])) if r["role"] == "main" and tag in r["last_user"]})
print(",".join(ps) or "none")
PY
}
models() { python3 -I -c 'import json,sys
try: print(json.dumps(json.load(open(sys.argv[1])).get("models", {}), sort_keys=True))
except FileNotFoundError: print("<absent>")' "$P/proj/.crush/crush.json"; }
project_qq() { printf '%s' '{"models":{"large":{"provider":"qq","model":"m-qq"},"small":{"provider":"qq","model":"m-qq"}}}' > "$P/proj/crush.json"; }
note_ok() { grep -q 'last ran on pp/m-pp; continuing on the project model qq/m-qq' "$1"; }

# --- local (CRUSH_CLIENT_SERVER=0): trước bản sửa lượt -s chạy model cũ của session trong RAM.
begin local 0
x=$(ac run --json "L-X1 on pp" 2>>"$P/err" | sid)
project_qq
ac run --json -s "$x" "L-X2 continue X" > "$P/x2.json" 2> "$P/x2.err"
ac run --json -s "$x" -m pp/m-pp "L-X3 explicit pp" > "$P/x3.json" 2> "$P/x3.err"
echo "local: X1=$(prov L-X1) X2=$(prov L-X2) X3=$(prov L-X3) data-models=$(models) x2.err=$(grep -v '^$' "$P/x2.err" | head -1)"
[ -n "$x" ] && [ "$(prov L-X1)" = pp ]; check "local: setup X1 on pp" $?
[ "$(prov L-X2)" = qq ]; check "local: run -s X goes to the workspace model qq" $?
note_ok "$P/x2.err"; check "local: run -s X prints the model note on stderr" $?
[ "$(prov L-X3)" = pp ] && ! note_ok "$P/x3.err"; check "local: run -s X -m pp/m-pp uses pp, no note" $?
finish

# --- server (CRUSH_CLIENT_SERVER=1).
begin server 1
x=$(ac run --json "S-X1 on pp" 2>>"$P/err" | sid)
project_qq
y=$(ac run --json "S-Y1 on project default" 2>>"$P/err" | sid)
before=$(models)
ac run --json -s "$x" "S-X2 continue X" > "$P/x2.json" 2> "$P/x2.err"
after=$(models)
ac send -s "$y" --wait --json "S-Y2 next turn of Y" > "$P/y2.json" 2>>"$P/err"
ac run --json "S-N1 new session" > "$P/n1.json" 2>>"$P/err"
ac run --json -s "$x" -m pp/m-pp "S-X3 explicit pp" > "$P/x3.json" 2> "$P/x3.err"
explicit=$(models)
ac run --json "S-N2 new session after -m" > "$P/n2.json" 2>>"$P/err"
echo "server: X1=$(prov S-X1) Y1=$(prov S-Y1) X2=$(prov S-X2) Y2=$(prov S-Y2) N1=$(prov S-N1) X3=$(prov S-X3) N2=$(prov S-N2)"
echo "server: data-models before=$before after-run-s=$after after-run-m=$explicit x2.err=$(grep -v '^$' "$P/x2.err" | head -1)"
[ -n "$x" ] && [ -n "$y" ] && [ "$(prov S-X1)" = pp ] && [ "$(prov S-Y1)" = qq ]; check "server: setup X1 on pp, Y1 on qq" $?
[ "$before" = "$after" ]; check "server: run -s X leaves <data>/crush.json models unchanged" $?
[ "$(prov S-X2)" = qq ] && [ "$(prov S-Y2)" = qq ] && [ "$(prov S-N1)" = qq ]; check "server: X2, Y2, N1 all on the workspace model qq" $?
note_ok "$P/x2.err"; check "server: run -s X prints the model note on stderr" $?
[ "$(prov S-X3)" = pp ] && [[ $explicit == *'"large": {"model": "m-pp", "provider": "pp"}'* ]]; check "server: run -s X -m pp/m-pp writes pp (explicit choice)" $?
[ "$(prov S-N2)" = pp ]; check "server: after -m the project runs pp" $?
finish

echo "fails=$FAILS"
[ "$FAILS" = 0 ]
```

## Phụ lục B: `p1c-tui.sh`

```bash
#!/usr/bin/env bash
# TUI thật trong tmux socket RIÊNG (không đụng màn hình người dùng), server mode, LLM giả.
# 1) session X chạy pp; dự án đổi sang qq; mở TUI bằng `-s X`: thanh trạng thái phải báo, <data>/crush.json không có models.
# 2) alt+m (danh sách model của provider đang chọn, qq), chọn m-qq, Enter: <data>/crush.json phải có models.large = qq
#    (lựa chọn tường minh vẫn ghi, kể cả khi trùng model đang có từ crush.json của repo).
# Dùng: bash p1c-tui.sh <repo agent-cli> <thư-mục-mới ≤ 70 byte> <binary>
set -u
R=$1; OUT=$2; BIN=$3
[ -e "$OUT" ] && { echo "exists: $OUT" >&2; exit 2; }
mkdir -p "$OUT" && OUT=$(cd "$OUT" && pwd)
P=$OUT; FAILS=0
check() { if [ "$2" = 0 ]; then echo "PASS $1"; else echo "FAIL $1"; FAILS=$((FAILS + 1)); fi; }
mkdir -p "$P/home/.config/agent-cli" "$P/log" "$P/rt" "$P/proj"; chmod 700 "$P/rt"
echo '[]' > "$P/rules.json"
timeout 600 python3 -I "$R/scripts/eval/fakellm.py" "$P/port" "$P/log" "$P/rules.json" >/dev/null 2>&1 &
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
(cd "$P/proj" && git init -q && echo x > README.md && echo "# t" > AGENTS.md && git add -A && git -c user.name=t -c user.email=t@t commit -qm init)
ENVS=(env -i PATH=/usr/bin:/bin TERM=xterm-256color HOME="$P/home" XDG_CONFIG_HOME="$P/home/.config"
  XDG_DATA_HOME="$P/home/.local/share" XDG_CACHE_HOME="$P/home/.cache" CRUSH_GLOBAL_CONFIG="$P/home/.config/agent-cli"
  CRUSH_GLOBAL_DATA="$P/home/.local/share/agent-cli" CRUSH_CACHE_DIR="$P/home/.cache/agent-cli" AGENT_CLI_RUNTIME_DIR="$P/rt"
  GIT_CEILING_DIRECTORIES="$OUT" CRUSH_DISABLE_METRICS=1 DO_NOT_TRACK=1 CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1
  CRUSH_CLIENT_SERVER=1 CRUSH_SERVER_IDLE_TIMEOUT=60)
HOST=(--host "unix://$P/s.sock")
x=$(cd "$P/proj" && "${ENVS[@]}" timeout 60 "$BIN" "${HOST[@]}" run --json "T-X1 on pp" 2>>"$P/err" | python3 -I -c 'import json,sys; print(json.load(sys.stdin)["session_id"])')
printf '%s' '{"models":{"large":{"provider":"qq","model":"m-qq"},"small":{"provider":"qq","model":"m-qq"}}}' > "$P/proj/crush.json"
models() { python3 -I -c 'import json,sys
try: print(json.dumps(json.load(open(sys.argv[1])).get("models", {}), sort_keys=True))
except FileNotFoundError: print("<absent>")' "$P/proj/.crush/crush.json"; }

TM=(tmux -S "$P/tmux.sock" -f /dev/null)
printf '#!/bin/sh\ncd %q && exec %s %q --host %q -s %q\n' "$P/proj" "$(printf '%q ' "${ENVS[@]}")" "$BIN" "unix://$P/s.sock" "$x" > "$P/tui.sh"
"${TM[@]}" new-session -d -s t -x 160 -y 45 "sh $P/tui.sh"
seen=""
for _ in $(seq 60); do
  sleep 0.25
  "${TM[@]}" capture-pane -p -t t > "$P/pane1.txt" 2>/dev/null
  grep -q "last ran on" "$P/pane1.txt" && { seen=1; break; }
done
echo "pane: $(grep -o 'This session[^│]*' "$P/pane1.txt" | head -1)"
[ -n "$x" ] && [ -n "$seen" ]; check "TUI -s X shows the model note in the status bar" $?
after_open=$(models)
echo "after open: data-models=$after_open"
[ "$after_open" = "<absent>" ]; check "TUI -s X writes no model to <data>/crush.json" $?

"${TM[@]}" send-keys -t t M-m
sleep 1
"${TM[@]}" send-keys -t t -l "m-qq"
sleep 0.5
"${TM[@]}" capture-pane -p -t t > "$P/pane2.txt"
"${TM[@]}" send-keys -t t Enter
picked=""
for _ in $(seq 40); do
  sleep 0.25
  m=$(models)
  [[ $m == *'"large": {'*'"model": "m-qq", "provider": "qq"}'* ]] && { picked=1; break; }
done
"${TM[@]}" capture-pane -p -t t > "$P/pane3.txt"
echo "after alt+m: data-models=$(models)"
[ -n "$picked" ]; check "alt+m picking m-qq writes models.large=qq (explicit choice)" $?

"${TM[@]}" kill-server 2>/dev/null
for pid in $(pgrep -f -- "server --host unix://$P/"); do kill -TERM "$pid" 2>/dev/null; done
kill "$FAKE" 2>/dev/null; wait "$FAKE" 2>/dev/null
echo "fails=$FAILS"
[ "$FAILS" = 0 ]
```

## Sau soát (điều phối, 19/10)

- Soát độc lập: DAT (vòng 1, 4 mục phải sửa) → DAT (vòng 2, đã đóng cả 4 + V5). Điều phối chạy lại: `session-model.sh` PASS
  trên bản sửa, FAIL trên `v2-final`; T17 NOT_EXPOSED.
- Điều phối thêm mục `DECISIONS.md` 2026-10-19 và một dòng `CLAUDE.md` P5. Commit `ef5c774e` trên nhánh
  `v3-poc-p1c-modelwrite`; patch cuối (`git diff v2-final HEAD`, gồm CLAUDE.md) thay bản cũ cùng tên.
