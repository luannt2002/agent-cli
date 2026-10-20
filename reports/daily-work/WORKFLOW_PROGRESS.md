# Tiến độ workflow (§84)

Nhật ký sự kiện kỹ thuật của đợt System Evolution → Daily Workspace, ngày 16/10. Chỉ ghi sự thật có bằng chứng: tệp hoặc lệnh ở cột cuối. Không ghi suy luận dở. Giờ lấy theo mtime của tệp bằng chứng, nên chỉ là gần đúng.

| # | Loại | Sự kiện | Bằng chứng |
|---|---|---|---|
| 1 | CHECKPOINT | Mốc xuất phát: `main` ở `4a0b25b6`, cây sạch. Toàn bộ thay đổi của đợt nằm trong worktree `scratchpad/evo/wt-b`; không commit. | `git rev-parse HEAD`, SE [00](../system-evolution/00_BASELINE_CHECKPOINT.md) |
| 2 | VERIFIED_RESULT | Baseline với LLM thật trên binary gốc: 10/10 kịch bản đạt, 0 false DONE (18:38). | `scratchpad/evo/baseline-real.log`, SE [21](../system-evolution/21_BASELINE_RESULTS.md) |
| 3 | DISCOVERY | Lời gọi ẩn (tiêu đề, `agentic_fetch`) đi sang provider của model nhỏ: 8/8 ca tiêm lỗi có request lọt sang provider thứ hai (19:12). | `scratchpad/evo/faults-base.txt` |
| 4 | DISCOVERY | Tự tóm tắt không có trần: 1.338 lời gọi main trong 120 s, ở cả binary gốc lẫn ứng viên đầu. | SE [33](../system-evolution/33_FAULT_AND_LONG_RUN.md), PRB-88 |
| 5 | DECISION | Không thêm subsystem memory (M0: context file + lịch sử session + tự tóm tắt). | SE [28](../system-evolution/28_HINDSIGHT_DECISIONS.md), [29](../system-evolution/29_MEMORY_COMPETITION.md) |
| 6 | VERIFIED_RESULT | Lượt soát độc lập 1–3 trên ứng viên: tổng 13 + 10 + 5 phát hiện, đều sửa có test hoặc ghi giới hạn. | SE [38](../system-evolution/38_RED_TEAM.md) |
| 7 | ROLLBACK | Bỏ cơ chế chặn biến git bằng phân tích phép gán (lượt soát 2 vòng qua bằng 7 dạng); `protect.go` trả về nguyên bản; thay bằng luật theo chữ. | SE 38 lượt 2, [16](16_PRUNING_DECISIONS.md) |
| 8 | CHECKPOINT | Mốc SE: `evidence/wt-final.patch`, sha256 `1d963689…`, 72 tệp (21:06). | `reports/system-evolution/evidence/wt-final.patch.sha256` |
| 9 | PHASE_STARTED | Daily: tận dụng bằng chứng SE, chỉ đo lại phần bị đổi. | [00_PREVIOUS_WORKFLOW_HANDOFF](00_PREVIOUS_WORKFLOW_HANDOFF.md) |
| 10 | DECISION | Daily sửa G-01 (fallback không ghi xuống đĩa) và G-08 (panic log vào thư mục log, 0600); hoãn G-02..G-07, G-09..G-11 có lý do. | [07](07_DAILY_WORK_GAPS.md), `evidence/daily-delta-1.patch` (20:57) |
| 11 | VERIFIED_RESULT | Kiểm cuối trước vòng 6: 3 E2E PASS, 0 data race, suite 4.614 test so theo tên với cây sạch: 0 mất, 0 đổi trạng thái. | `scratchpad/evo/final-verify.txt`, `suite-compare-final.txt` |
| 12 | VERIFIED_RESULT | Eval LLM thật trên ứng viên trước vòng 6: 13/13 đạt, 0 false DONE (21:15). | `scratchpad/evo/final-real2.log` |
| 13 | REGRESSION | Red team cuối KHONG-DAT: F1 HIGH (vẫn đổi provider lặng lẽ lúc khởi động), F2 bằng chứng cũ hơn mã, F3 cổng Qwen-only ngoài repo, F4 GREEN giả thoát 0, cùng L1–L9. | `scratchpad/evo/redteam-final.txt` |
| 14 | ASSUMPTION_INVALIDATED | "Dừng lúc khởi động thì không chạy được `agent-cli login`" là sai: login chỉ cần nạp được cấu hình. Quyết định "fallback trong RAM" bị lật. | DECISIONS.md (mục 16/10 "Model đang chọn không dùng được…") |
| 15 | PHASE_STARTED | Vòng 6: hai agent ở hai worktree không giao tệp (P: F1+L5; Q: F4+L6); lead làm L1, L2, F3, docs. | workflow `wf_4dd0f31f-164` |
| 16 | VERIFIED_RESULT | F1 bằng binary thật: chọn `openai/gpt-4o` không key. Bản trước thoát 0 sau 2 request sang `glm`. Bản mới thoát 1, 0 request, ở `run` và khi mở TUI (chế độ local: hai thư mục `f1-server/`, `f1-tui/` đặt tên nhầm, xem mốc 30). | `scratchpad/evo/f1-red-faults.txt`, `f1-green-faults.txt`, `f1-server/`, `f1-tui/` |
| 17 | DEVIATION | Gợi ý `-m provider/model` trong thông báo lỗi F1 là sai: đo bằng binary, `run -m` vẫn dừng ở cùng lỗi. Đã sửa câu chữ, ghim bằng test. | `scratchpad/evo/f1-hint2/out.json`, `TestNoFallbackErrorKeepsSentinel` |
| 18 | VERIFIED_RESULT | L2 + F4 bằng binary thật (`qwen-only.sh` bản mới): binary trước FAIL (`text` dính bản tóm tắt; loop GREEN thoát 0). Binary vòng 6 PASS (`text` sạch; loop `NO_CHANGE` thoát 1; 12/12 request đi qwen). | `scratchpad/evo/r6-qwen-old.txt`, `r6-qwen-new.txt` |
| 19 | NEW_UNKNOWN | Repo chưa có commit thì dấu vân tay "unknown": loop không bao giờ ra `NO_CHANGE` ở đó (GREEN kèm cảnh báo). Ghi vào tài liệu; kịch bản đo phải commit fixture. | `scratchpad/evo/qwen-only-final/loop.err`, `docs/agent-cli-local-setup.md` |
| 20 | CHECKPOINT | Ứng viên vòng 6 đã chốt: `scratchpad/evo/r6-final.patch`, sha256 `2af05925…`, 82 tệp, +5611/−301 (21:48). Delta vòng 6: `r6-delta.patch`, 32 tệp. | `sha256sum r6-final.patch` |

| 21 | VERIFIED_RESULT | Kiểm cuối trên bản vòng 6: 3 E2E PASS, faults/qwen-only/workspaces PASS, race 0, suite 4.631 test: 0 mất, 0 đổi trạng thái, 130 mới. | `scratchpad/evo/final-verify2.txt` |
| 22 | DISCOVERY | `agent-cli login` dựng cả workspace (`app.New` → `InitCoderAgent`), nên ở trạng thái fail closed chính `login` cũng hỏng: cách gỡ mà thông báo lỗi gợi ý lại dùng không được. Lead tìm ra khi đọc mã; người soát độc lập vòng 6 tìm ra độc lập và đo bằng binary (local + server). | `internal/cmd/login.go:53`; `scratchpad/evo/review6/p-login-*` |
| 23 | REGRESSION | Người soát vòng 6 KHONG-DAT: 1 HIGH (login), 2 MEDIUM (provider dựng sẵn bị tắt vẫn bị gọi; `text` local nối mọi chữ assistant), 3 LOW. Người soát README lượt 2 và 3 NOT READY. | SE [38](../system-evolution/38_RED_TEAM.md), [README_REVIEW](README_REVIEW.md) |
| 24 | ASSUMPTION_INVALIDATED | Giả định của F1 "Load thành công là `login` chạy được" sai; quyết định cũ "dừng khởi động thì khoá `login`" hoá ra đúng một nửa: dừng thì khoá, nhưng không cần đổi provider để khỏi dừng. | DECISIONS.md (mục "Model đang chọn không dùng được") |
| 25 | DECISION | Vòng 7: `app.New` khởi động không agent khi lỗi là "model đang chọn không dùng được" (như lúc chưa cấu hình); TUI vào màn chọn model; `run`/`loop`/`routine` vẫn dừng. Provider bị tắt bị từ chối ở `buildAgentModels`. `text` local: ghi tài liệu thay vì đổi mã. | `scratchpad/evo/r7-delta.patch` |
| 26 | VERIFIED_RESULT | Vòng 7 bằng binary: faults PASS (probe `login` → "Unknown platform"); binary vòng 6 FAIL đúng chỗ đó. `login` qua được, `run` rc=1, 0 request; TUI: màn "To start, let's choose a provider and model.", 0 request (chế độ local, xem mốc 30). | `scratchpad/evo/r7b-faults-new2.txt`, `r7b-faults-old.txt`, `r7-srv/` |
| 27 | CHECKPOINT | Ứng viên vòng 7: `scratchpad/evo/r7-final.patch`, sha256 `6035d0fc…`, 86 tệp, +5883/−304. Delta vòng 6 → 7: 18 tệp. | `sha256sum r7-final.patch` |
| 28 | DEVIATION | Một lệnh kiểm của lead viết sai (`A && B && (C) & (D) &` cho build chạy nền chung nhóm), nên lần chạy faults đầu trên binary mới ra rc=127; đã chạy lại riêng. Báo cáo dùng lần chạy lại. | `scratchpad/evo/r7b-faults-new.txt` (lỗi), `r7b-faults-new2.txt` (dùng) |

| 29 | VERIFIED_RESULT | Kiểm cuối lần 3 trên bản vòng 7: 3 E2E PASS, 3 cổng PASS, race 0, suite 4.636 test (0 mất, 0 đổi, 135 mới). Eval LLM thật: 12/13 đạt; s13 trượt vì GLM không gửi dữ liệu 120 s (lỗi rõ, không đổi provider), chạy lại riêng: đạt. | `final-verify3.txt`, `final-real3.log`, `final-real3-s13b/` |
| 30 | DEVIATION | Lead ghi nhầm "chế độ server" cho các probe `f1-server/`, `r7-srv/`: binary chạy thẳng mặc định là local (`CRUSH_CLIENT_SERVER` không đặt; chỉ launcher bật server). Người soát vòng 7 kiểm đúng chế độ server. Các báo cáo đã sửa nhãn. | `internal/cmd/root.go` `useClientServer`, `scripts/install-agent-cli.sh:83` |
| 31 | REGRESSION | Người soát vòng 7 KHONG-DAT: 1 MEDIUM (qua server, `run`/`loop`/`send` chỉ in "status code 500": client bỏ nội dung lỗi của server), 3 LOW (test provider bị tắt phụ thuộc thứ tự chạy; chưa có test cho nhánh `app.New` dừng ở lỗi khác; vài câu docs). Người soát README lượt 4: NOT READY (phạm vi của lời hứa provider). | SE [38](../system-evolution/38_RED_TEAM.md), [README_REVIEW](README_REVIEW.md) |
| 32 | VERIFIED_RESULT | Vòng 8: client giữ nội dung lỗi (`checkStatus`); `faults.sh` thêm ca qua server thật. Binary vòng 7 FAIL ca đó (lỗi không nêu provider), binary vòng 8 PASS (lỗi nêu `provider "openai", model "gpt-4o"`; `login` qua server chạy). | `r8-faults-old.txt`, `r8-faults-new.txt` |
| 33 | CHECKPOINT | Ứng viên vòng 8: `scratchpad/evo/r8-final.patch`, sha256 `e4081c0b…`, 88 tệp, +5925/−306. Delta vòng 7 → 8: 6 tệp. | `sha256sum r8-final.patch` |

| 34 | VERIFIED_RESULT | Kiểm cuối lần 4 trên bản vòng 8: 3 E2E PASS; faults (11 ca, gồm ca qua server) PASS; qwen-only, workspaces PASS; race 0 (413 s); suite 4.637 test, 0 mất, 0 đổi trạng thái, 136 mới. | `final-verify4.txt` |
| 35 | VERIFIED_RESULT | Smoke LLM thật qua server trên binary cuối: `run` rc=0, verify rc=0, 7 lời gọi đều `main/qwen3.8`, 0 lỗi; log server có trong thư mục smoke. | `real-smoke-final4.txt` |
| 36 | VERIFIED_RESULT | Soát độc lập vòng 8: KET-LUAN: DAT. 3 câu README mức LOW đã sửa. | SE [38](../system-evolution/38_RED_TEAM.md) |

Kết quả soát README lượt 5 và trạng thái cuối: [README_REVIEW](README_REVIEW.md), [FINAL_DAILY_WORK_AUDIT](FINAL_DAILY_WORK_AUDIT.md).

## Tiếp quản ngày 19/10/2026

| # | Loại | Sự kiện | Bằng chứng |
|---|---|---|---|
| 37 | CHECKPOINT | Khôi phục vòng 8 trên worktree `codex/audit-r8-takeover`, base `4a0b25b6`; khôi phục 15 sửa README cuối từ transcript, không chạy lệnh trong transcript. Không commit/push. | SE [39](../system-evolution/39_TAKEOVER_2026-10-19.md) |
| 38 | VERIFIED_RESULT | Reviewer mới chỉ đọc README/guide/help: READY, 16/16 câu hỏi trả lời được; đồng bộ giới hạn Git của loop trong guide. | [Review 19/10](README_REVIEW_2026-10-19.md) |
| 39 | REGRESSION | Lượt workspace đầu đỏ vì helper tìm nhãn không xử lý dòng trống đầu prompt. Chỉ sửa chọn nhãn bằng `lstrip()`, giữ nguyên assertion; reviewer DAT. | SE 39; `takeover-delta.patch` |
| 40 | VERIFIED_RESULT | Build, suite 4.637 test (4.628 pass/9 skip; 136 mới pass, 0 mất), 3 E2E và faults/qwen-only/workspaces đều PASS ở lượt chạy mới có log bền. `go vet` vẫn FAIL copylocks trên file giống base. Race sau đó PASS, exit 0, 579,21 s, không phát hiện data race. | SE 39; `evidence/takeover-2026-10-19/` |
| 41 | DECISION | Xác minh Memlake là engine memory Enterprise riêng của Hindsight, không phải SQLite mà agent-cli đang dùng. Giữ M0/SQLite, chưa tích hợp hoặc benchmark Memlake. | SE [40](../system-evolution/40_MEMLAKE_SCOPE_2026-10-19.md) |
| 42 | VERIFIED_RESULT | Phía Claude kiểm độc lập kết quả tiếp quản bằng lệnh, không dựa vào lời khai: `r8-final.patch` sha256 OK; dựng hai cây trên index tạm, vòng 8 và bản tiếp quản chỉ khác `README.md`, guide, `workspaces.sh` (0 tệp Go, 0 `_test.go`); tự đếm `suite.log`: 4.637 test, 4.628 pass, 9 skip, 0 fail; race 28 gói `ok`, 0 `DATA RACE`; vet chỉ còn `csync/maps.go:156`, tệp không đổi so với HEAD; 6 cổng tích hợp có dòng PASS; worktree Codex vẫn ở `4a0b25b6` (không commit); bản cài `~/.local/lib/agent-cli/crush` vẫn mang mtime 16/10. Race của Codex chạy 8 nhóm gói, còn bản vòng 8 chạy 12 nhóm: mã giống hệt nên kết quả 12 nhóm vẫn áp dụng. | SE [37](../system-evolution/37_FINAL_PROOF.md) mục cuối |
| 43 | VERIFIED_RESULT | Soát README lượt 6b, người soát phía Claude không có Edit, làm trên README bản tiếp quản: READY, 16/16, 0 MUST-FIX, 14 NICE-TO-HAVE. N11 (comment trong `agent-cli.crushrc.example` sai so với mã) cần sửa. | [README_REVIEW](README_REVIEW.md) lượt 6b |
| 44 | REGRESSION → VERIFIED_RESULT | Sửa N11 bằng patch chỉ đổi comment `reports/system-evolution/evidence/followup-2026-10-19/crushrc-followup.patch`, áp lên bản tiếp quản. Người soát không có Edit chấm KHONG-DAT hai lần, cả hai lần đúng: (1) bản đầu xếp routine vào nhóm tự duyệt, nhưng routine không có tool giao việc (`delegate.go:104-107`); (2) vế `AGENT_CLI_UNATTENDED=1` không có tác dụng khi ghi trong tệp env. Lần ba chấm DAT: 2 hunk, 0 dòng không phải comment. Mỗi lần đều chạy lại `git apply --cached --check` trên index tạm, `bash -n`, `go test ./internal/config/ ./internal/shellconfig/`: rc=0 cả ba. | sha256 `e607f9de…` |
| 45 | CHECKPOINT | Gộp `reports/` của Codex vào repo chính (Codex chỉ thêm, không xoá dòng nào; sau khi chép `diff -rq` = 0), cộng phần phía Claude. Ghi trạng thái cuối §96. | [FINAL_DAILY_WORK_AUDIT](FINAL_DAILY_WORK_AUDIT.md) |
