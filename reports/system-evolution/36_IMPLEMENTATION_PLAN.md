# 36 — Kế hoạch thi hành (Phase 26–27)

Thứ tự theo §64: đúng/sai workspace-session → khôi phục → cô lập provider → đúng trạng thái → bằng chứng → skill → routine. Mỗi slice: một vấn đề, test đỏ trước, sửa nhỏ nhất, chạy gói liên quan, so tên test với baseline. Code an ninh/vòng đời qua soát độc lập.

## Đã làm (worktree `scratchpad/evo/wt`, chưa commit)

| Slice | Vấn đề | Tệp chính | Test (chi tiết ở [32](32_REGRESSION_SUITE.md)) |
|---|---|---|---|
| S-1 Vòng đời lượt | PRB-01, 12, 13, 14, 27 | `agent.go`, `errors.go`, `settle.go` | summarize_overflow, false_success, continuation, unfinished_reply |
| S-2 Cô lập provider + chi phí ẩn | PRB-03, 46, 74, 09 | `coordinator.go`, `agent.go`, `agentic_fetch_tool.go`, `agent_tool.go` | provider_isolation, title_cost, agentic_fetch_policy |
| S-3 Quan sát | PRB-32 (giảm nhẹ), PRB-45 (một phần) | `llm_ledger.go`, `execpolicy/policy.go`, `cmd/run.go` | llm_ledger; E2E run-json |
| S-4a Quyền | PRB-06, 70 | `permission.go` | permission_test (3 test mới) |
| S-5a Không người trực | PRB-25, 67 | `shell/unattended.go` | unattended_dirflag, unattended_abbrev |
| S-6a Routine/loop | PRB-02, 07, 24, 66 | `cmd/routine.go`, `cmd/loop.go`, `loop/loop.go`, `routine/routine.go` | routine_status, verify_pipefail, flags, schedule_env; E2E routine mục 8 |
| S-7a Skill/log | PRB-08 (một phần), 10 | `skills/skills.go`, `backend/backend.go` | hidden_dirs, env_log |
| S-8 Job nền | PRB-15, 18, 31 | `app/app.go`, `shell/background.go`, `tools/job_*.go`, `ui/chat/bash.go` | shutdown, kill_owned, job_tools, job_exit |

## Tiếp theo (theo thứ tự)

| Slice | Vấn đề | Tiêu chí nghiệm thu | Chỗ chưa chắc → cách kiểm |
|---|---|---|---|
| S-9 Lượt rỗng là lỗi | PRB-75 | `faults.sh` ca `empty-reply` ra rc=1; lượt có tool call mà chữ cuối rỗng vẫn rc=0; cassette `TestCoderAgent` xanh | provider thật có trả lời rỗng hợp lệ không → chỉ coi là lỗi khi **mọi** bước không có chữ lẫn tool call |
| S-10 Fallback provider lúc khởi động | PRB-04 (C-1) | provider cấu hình thiếu khoá → lỗi rõ, không ghi `models` vào global data; fallback chỉ trong cùng provider | đường `load.go` fallback còn ai dựa vào (lần chạy đầu không có cấu hình?) → đọc `load.go:166-180` + test cấu hình trống |
| S-11 Model thuộc session | PRB-05 (C-2) | resume/đổi session không đổi model của session khác; `run --model` chỉ áp cho lượt đó | đường server mang model theo prompt có cần đổi giao thức → đọc `backend` + `client` trước khi code |
| S-12 Context file | PRB-19, 20 | `agents.md` tên thường được nạp; context file ở gốc git được nạp khi cwd là thư mục con | thứ tự ưu tiên khi cả cwd lẫn gốc git có tệp → giữ cwd thắng, gốc git bổ sung; test bằng probe P-03 |
| S-13 Xoá session sạch | PRB-11 | xoá session xoá session con + transcript | cần query sqlc mới; `sqlc` chưa cài trên máy → viết query tay đúng mẫu `internal/db` hoặc cài sqlc |

Không làm trong đợt này (lý do ở 34/28): memory, trần thời gian cho `run`, hỏi quyền `job_kill`, PRB-22 (đụng cassette VCR, LOW).

## Cách chứng minh xong (mỗi slice)

1. Test mới đỏ trên mã cũ, xanh trên mã mới (dán lệnh + dòng FAIL/PASS).
2. `go test` các gói bị đụng, so **tên** test với `evidence/00_test_baseline.txt`: không tên nào biến mất, không tên nào đổi trạng thái ngoài test mới.
3. Code an ninh/vòng đời: đột biến bằng `-overlay`, và một lượt soát độc lập không có quyền Edit.
4. Cuối đợt: toàn bộ `go test ./...` một lần, race một lần, ba E2E một lần ([37](37_FINAL_PROOF.md)).
