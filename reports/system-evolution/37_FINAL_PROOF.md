# 37 — Bằng chứng cuối (Phase 29)

Mọi số ở mục "Kết quả kiểm cuối" chạy trên **một revision duy nhất**: ứng viên cuối `scratchpad/evo/r8-final.patch`
(sha256 `e4081c0b…`, 88 tệp, +5925/−306; bản lưu bền: `evidence/r8-final.patch`) trên `main` @ `4a0b25b6`. Các bản vòng 6
(`final-verify2.txt`) và vòng 7 (`final-verify3.txt`) cũng đã qua đủ bộ kiểm này; mỗi lượt soát độc lập sau đó tìm thêm lỗi
nên có vòng kế tiếp.

Lệnh: `bash scratchpad/evo/final-verify4.sh` (build + 3 E2E + 3 cổng LLM giả + race + suite so theo tên). Log đầy đủ:
`scratchpad/evo/final-verify4.txt` và các tệp `*-final4*` cạnh nó.

## Kết quả kiểm cuối (16/10, 23:38–23:53)

| Bước | Kết quả |
|---|---|
| Build | `bin/agent-cli-final4`, sha256 `b67375ce…`; `git diff HEAD --stat`: 88 tệp, +5925/−306 (khớp patch đã chốt) |
| E2E `test-agent-cli-run-json.sh` (binary + server thật, LLM giả) | rc=0, PASS, 0 FAIL, 57 s |
| E2E `test-agent-cli-loop.sh` | rc=0, PASS, 0 FAIL, 16 s |
| E2E `test-agent-cli-routine.sh` (có ca SIGKILL → `ABANDONED`, khoá nhả) | rc=0, PASS, 0 FAIL, 19 s |
| `scripts/eval/faults.sh` (11 ca, gồm `selected-no-key` local và `selected-no-key-server` qua server, mỗi ca kèm probe `login`) | `FAULTS: PASS`: không ca nào có request sang `glm`; `ok` thoát 0 có request; hai ca thiếu key thoát 1, 0 request, lỗi nêu `"openai"`, `login` ra "Unknown platform" |
| `scripts/eval/qwen-only.sh` | `QWEN-ONLY: PASS`: 5 vai trò đều ở qwen, glm = 0 (server lẫn sổ ghi), `text` không lẫn bản tóm tắt, đường loop dừng `NO_CHANGE` rc=1 |
| `scripts/eval/workspaces.sh` | `WORKSPACES: PASS` |
| `go test -race` (12 nhóm gói: backend, agent, cmd, ui, workspace, projects, permission, server, shell, app, routine, loop) | rc=0, 413 s, 0 `DATA RACE`, 0 dòng FAIL |
| `go test -json ./...` so theo TÊN với cây sạch `4a0b25b6` | cây sạch 4.501 test (4.489 pass, 12 skip); ứng viên 4.637 test (4.625 pass, 12 skip); **mất 0, đổi trạng thái 0, mới 136 (đều pass), lỗi 0**; 68 gói pass, 17 gói không có test |

Cây sạch chạy bằng `suite-compare.sh` (cùng lệnh, `scratchpad/evo/suite-base.jsonl`); so bằng `suite-compare-py.sh`
(theo cặp gói + tên, không đếm số).

## "Đỏ trước", chạy lại cơ học (`scratchpad/evo/red-before2.sh`, trên bản vòng 6)

Chép mọi tệp test mới/đổi của ứng viên lên cây sạch `4a0b25b6` (wt-c đặt lại về HEAD), chạy đúng các test đó trên mã
CŨ. Gói không biên dịch thì thử từng tệp riêng. Log: `scratchpad/evo/red-before2.txt` (cây sạch trước và sau: 0 dòng
status).

| Gói | Test mới/đổi | Đỏ do assert | Xanh | Dòng không biên dịch (gói hoặc tệp) |
|---|---|---|---|---|
| internal/agent | 41 | 5 | 5 | 7 |
| internal/agent/tools | 8 | 4 | 1 | 2 |
| internal/app | 3 | 0 | 0 | 3 |
| internal/backend | 1 | 1 | 0 | 0 |
| internal/cmd | 23 | 3 | 1 | 4 |
| internal/config | 6 | 5 | 1 | 0 |
| internal/log | 2 | 0 | 0 | 2 |
| internal/loop | 5 | 0 | 0 | 2 |
| internal/permission | 4 | 3 | 1 | 0 |
| internal/routine | 5 | 0 | 0 | 3 |
| internal/shell | 8 | 6 | 1 | 2 |
| internal/skills | 2 | 1 | 1 | 0 |
| internal/ui/chat | 1 | 1 | 0 | 0 |
| **Tổng** | **109** | **29** | **11** | — |

- Test trong tệp không biên dịch được trên mã cũ (gọi API mới như `noFallbackError`, `StopNoChange`, `ledgerEntry`):
  "đỏ do không biên dịch". Với các test của vòng 6, người viết đã chạy đỏ bằng overlay thay hàm mới bằng bản rỗng (32).
- 11 test xanh trên mã cũ, đều có lý do:
  - 7 test **có sẵn**, chỉ bị liệt kê vì test mới được chèn ngay sau chúng (tiêu đề hunk mang tên chúng):
    `TestAcceptEditsDoesNotWaitForOtherPrompts`, `TestBashBlocksDestructiveCommandsWhenUnattended`,
    `TestEmptySummaryKeepsHistory`, `TestLengthContinueCapResetsWhenTheRunMakesProgress`,
    `TestModelSelectionYieldsToDiskWhenUnchosen`, `TestRepeatedSummaryDoesNotNestThePrompt`, `TestRunStream_RunCompleteExits`.
  - 4 test **mới là đối chứng có chủ đích**: `TestDiscoverKeepsAHiddenBaseWrittenWithATrailingSlash` (chặn hồi quy do
    chính ứng viên gây ra, PRB-87), `TestSettleInterruptedCalls_LeavesFinishedRepliesAlone` (lời trả lời đã xong không
    bị đụng), `TestTitleRetriesTheLargeModelAfterAnError` (giữ hành vi thử lại sau lỗi), `TestUnattendedGitAllowsUnrelatedEnv`
    (biến không liên quan vẫn qua).

Test thêm ở vòng 8: `TestInitiateAgentProcessingDecodesErrorBody` đỏ trước khi sửa (lỗi chỉ còn "status code 500"); test
provider bị tắt viết lại để đột biến bỏ kiểm `Disable` đỏ ở mọi cách chạy. 5 test thêm ở vòng 7: 3 test đỏ trước khi sửa (chạy lúc TDD: `TestCreateWorkspaceStartsWhenSelectedModelIsUnavailable` báo
"failed to initialize coder agent", `TestUnavailableSelectedModelOpensOnboarding` ra màn khởi tạo, `TestBuildAgentModelsRefusesADisabledProvider`
không có lỗi); `TestSelectedModelUnavailableCoversOnlyUnusableSelections` gọi hàm mới (không biên dịch trên mã cũ), bị bắt bởi 2
đột biến; `TestReloadDoesNotSwitchProviderWhenTheKeyDisappears` xanh trên mã vòng 6 (mã đúng), đỏ khi khôi phục fallback cũ.

## Ca hỏng tái lập được bằng binary thật: trước ↔ sau

Binary trước = ứng viên trước vòng 6 (`bin/agent-cli-final`) hoặc binary gốc; binary sau = ứng viên cuối.

| Ca | Trước | Sau | Bằng chứng |
|---|---|---|---|
| Lời gọi ẩn sang provider của model nhỏ (faults, 8 ca) | binary gốc: 8/8 ca có request sang glm | 0 | `faults-base.txt`, `faults-final4.txt` |
| Chỉ Qwen, đi qua mọi đường gọi LLM | binary gốc: 799 request sang glm, hết giờ sau 2.659 request | 12/12 ở qwen, xong | `qwen-only-base/`, `qwen-only-final4/` |
| Model đang chọn thiếu key | ứng viên trước vòng 6: rc=0, 2 request sang glm, `"provider":"glm"` | `run` rc=1, 0 request, lỗi nêu provider/model ở local lẫn qua server | `f1-red-faults.txt`, `faults-final4.txt` |
| Cùng trạng thái, qua server | bản vòng 7: rc=1, 0 request nhưng lỗi chỉ còn "status code 500" | lỗi nêu `provider "openai", model "gpt-4o"` | `review8/faults-final3.txt` (cổng cuối trên binary vòng 7: FAIL vì lỗi không nêu provider), `r8-faults-new.txt` |
| Cùng trạng thái: `agent-cli login` | bản vòng 6: hỏng ("failed to initialize coder agent") | chạy được ("Unknown platform" với nền tảng giả), local và server | `r7b-faults-old.txt`, `faults-final4/selected-no-key*/login.txt` |
| Cùng trạng thái: mở TUI | bản vòng 6: thoát 1 với lỗi | màn chọn model, 0 request (local: lead; server: người soát vòng 7) | `f1-tui/`, `r7-srv/tui.txt`, `review7/p-tui-srv` |
| `text` dính bản tóm tắt | `## Current State\nSUMMARY-OF-SESSIONDONE-AFTER-SUMMARY` | `DONE-AFTER-SUMMARY` | `r6-qwen-old.txt`, `qwen-only-final4.txt` |
| Loop có cổng xanh sẵn, agent không đổi gì | `stop: GREEN`, rc=0 | `stop: NO_CHANGE`, rc=1 | như trên |
| Model trả lời rỗng | binary gốc: rc=0, `text` rỗng | rc=1 | `faults-base.txt`, `faults-final4.txt` |

## Eval LLM thật (16/10, 23:04–23:14, binary vòng 7) và smoke LLM thật trên binary cuối

Eval đầy đủ chạy trên `bin/agent-cli-final3` (bản vòng 7). Giữa vòng 7 và vòng 8, mã sản phẩm chỉ đổi ở
`internal/client/proto.go` (`InitiateAgentProcessing`, đường client của chế độ server); bộ eval chạy chế độ local
(`CRUSH_CLIENT_SERVER=0` trong `scripts/eval/run.sh`), nên đường mã nó đi qua giống hệt giữa hai bản. Thêm một smoke LLM thật
**qua server** trên binary cuối `bin/agent-cli-final4` để chạy chính đường đã đổi: `bash scratchpad/evo/real-smoke-server.sh`
(kịch bản s1, socket riêng, dữ liệu tạm): `run` rc=0 trong 15 s, verify (`go test`) rc=0, 7 lời gọi đều `main/qwen3.8` (title
1, main 6), 0 lỗi; có log server ở `real-smoke-final4/cache/server-unix___s.sock/crush.log`.

Lệnh eval: `bash scratchpad/evo/run-suite2.sh bin/agent-cli-final3 final-real3` (13 kịch bản của `scripts/eval/fixtures.py`,
provider thật `main/qwen3.8` của người dùng; s13 bước 2 `-m glm/glm-5.3-flash`). Log: `scratchpad/evo/final-real3.log`.

- 12/13 đạt, 0 DONE giả. s13 trượt: lời gọi duy nhất tới `glm` (bước 2, đúng provider được chọn) không nhận dữ liệu trong
  120 s, kết thúc `timeout` (sổ ghi LLM: `duration_ms` 120062, `error: timeout`, 0 token); `run` thoát 1 với lỗi đó, không
  đổi sang provider khác.
- Phân loại: **đỏ do môi trường** (provider ngoài treo). Kiểm: chạy lại riêng s13 trên cùng binary
  (`final-real3-s13b/`): đạt, 34,3 s, 4 lời gọi `main` rồi 5 lời gọi `glm` đều kết thúc bình thường.
- So với trước: [15 (Daily)](../daily-work/15_BEFORE_AFTER_METRICS.md).

## Phần mã không đổi sau khi chốt

Sau `r8-final.patch` chỉ sửa tài liệu nếu có (README theo lượt soát README 5, báo cáo).
`git -C scratchpad/evo/wt-b diff HEAD -- . ':(exclude)*.md' | sha256sum` lúc chốt vòng 8 = `841f5edb22f291ce…`; giá trị cuối ghi
ở FINAL_DAILY_WORK_AUDIT.

## Đường dẫn `scratchpad/evo/` không còn (19/10)

Máy khởi động lại ngày 19/10 xoá `/tmp`, kéo theo mọi đường dẫn `scratchpad/evo/…` ở trên (worktree `wt-b`/`wt-c`, binary
`bin/agent-cli-final*`, `suite-base.jsonl`, các log). Bản lưu bền: patch `evidence/r8-final.patch` (sha256 kiểm lại 19/10:
OK), log then chốt `evidence/final-logs/` (20 tệp, gồm `final-verify4.txt`, `faults-final4.txt`, `qwen-only-final4.txt`,
`race-final4.txt`, `suite-compare-final4.txt`, `final-real3.log`). Bộ kiểm chạy lại trên bản tiếp quản:
[39](39_TAKEOVER_2026-10-19.md), receipt ở `evidence/takeover-2026-10-19/`.

Thay cho mục "Phần mã không đổi sau khi chốt": dựng cây từ `r8-final.patch` và từ `takeover-final.patch` trên index tạm
(`GIT_INDEX_FILE=… git read-tree HEAD && git apply --cached …`), rồi `git diff --stat` hai cây: chỉ khác `README.md`,
`docs/agent-cli-local-setup.md`, `scripts/eval/workspaces.sh`; 0 tệp `_test.go`, 0 tệp Go. Mã sản phẩm của bản tiếp quản
giống hệt vòng 8, nên eval LLM thật và race 12 nhóm gói ở trên vẫn áp cho nó.
