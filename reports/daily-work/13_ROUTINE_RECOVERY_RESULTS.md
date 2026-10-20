# 13 — Routine và khôi phục (Daily H, §43, §44)

## Routine dùng lại primitive, không có runtime thứ hai (§43)

`routine run` → đọc định nghĩa (sha ghim) → khoá flock → worktree riêng → engine `loop` → agent chạy in-process bằng chính coordinator/`sessionAgent` của `run` → lệnh verify tất định → báo cáo 0600. Không có hàng đợi, worker hay trạng thái task riêng. Đã xác minh khi đọc mã (`internal/cmd/routine.go`, `internal/loop/`) và qua kịch bản "chỉ Qwen": đường `loop` đi qua cùng điểm chặn provider (2 request loop, đều qwen).

## Chết giữa chừng và thu hồi (§44, binary thật, LLM giả, 16/10)

Lệnh: `AGENT_CLI_BIN=<binary cuối> bash scripts/test-agent-cli-routine.sh` → 26 kiểm tra `ok`, `PASS` (19 giây).

| Kiểm tra | Kết quả |
|---|---|
| SIGTERM giữa lượt → `CANCELLED`, rc=1, báo cáo vẫn được ghi, khoá nhả | ok |
| **TEST A/B:** SIGKILL giữa lượt → lượt cuối hiện `ABANDONED` (không bị che sau báo cáo CANCELLED cũ) | ok (mới) |
| **TEST B:** sau SIGKILL khoá đã nhả: tiến trình khác (danh tính khác) chạy được | ok (mới; `flock -n` rc=0) |
| Đang giữ khoá → lượt thứ hai thoát 75, không tạo nhánh | ok |
| Lỗi sớm (tạo worktree hỏng) → `ERROR` kèm nguyên nhân, không phải `ABANDONED` | unit test `TestExecuteRoutine_WorktreeFailureWritesAnErrorReport` |
| Lượt mới chưa kịp ghi báo cáo mà khoá đang giữ → `RUNNING` | unit test |
| Verify xanh từ trước và agent không đổi gì → `NO_CHANGE` (maintenance thoát 0, goal thoát 1); `list`/`show` hiện `NO_CHANGE` (vòng 6, red team F4) | `TestExecuteRoutine_NoChangeIsReportedAndListed`, `TestExecuteRoutine_NoChangeInGoalModeExitsNonZero`; với `loop` bằng binary thật: `qwen-only.sh` (`stop: NO_CHANGE`, rc=1) |

Kết quả hợp lệ theo §44: thu hồi được (RECLAIMABLE: khoá nhả khi tiến trình chết, kernel tự làm, không cần id worker), thử lại được (RETRYABLE: lượt sau chạy worktree mới), thất bại có bằng chứng (FAILED WITH EVIDENCE: báo cáo ABANDONED/ERROR giữ worktree + nhánh để xem). Không có trạng thái "processing mãi".

Khác Hindsight (H-01): ở đó worker thay thế mang id khác thì việc kẹt `processing` vô hạn. Ở agent-cli "lease" là flock, nên bất kỳ tiến trình nào sau đó cũng lấy được.

## Lượt agent dài (chung cho run/loop/routine)

| Hỏng | Trước | Sau | Bằng chứng |
|---|---|---|---|
| Phần cố định ≥ 80% window | tóm tắt lặp vô hạn (755+755 lời gọi / 90 s) | dừng sau 1 lần tóm tắt, lỗi "context window too small" | P-04, test (2 biến thể) |
| Model làm lại việc sau mỗi lần tóm tắt | vô hạn (1.338 lời gọi / 120 s) | dừng sau 10 lần, lỗi rõ | qwen-only.sh, `TestAutoSummarizeHasACapPerPrompt` |
| Chuỗi không ra gì (rỗng, bị cắt khi nghĩ, bị từ chối) | rc=0, `text` rỗng | lỗi (rỗng / bị từ chối) | faults.sh ca `empty-reply`, probe openai-compat của người soát |
| Lặp cùng tool call | rc=0 | lỗi | test |
| Server chết giữa tool call | tool call "Interrupted" khi mở lại (có sẵn) | + lời trả lời chữ/rỗng dở cũng được đóng | test |

## Chưa chạy

Routine nhiều giờ với LLM thật; routine theo lịch thật (agent-cli chỉ IN lịch, không tự cài; chạy theo lịch thật cần người dùng cài unit/crontab).
