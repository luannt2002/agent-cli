# 33 — Tiêm lỗi và chạy dài (Phase 21–22)

Yêu cầu chung cho mọi ca lỗi: không báo xong giả, không fallback provider lặng lẽ, không zombie vĩnh viễn, không lây giữa workspace/nhánh.

## 1. Lỗi provider trên binary thật (E3, 16/10)

Lệnh: `bash scripts/eval/faults.sh <dir> <binary>`. Model chính ở provider `qwen`. Model nhỏ **cố ý** đặt ở provider `glm`, nên mọi lời gọi ẩn lọt sang provider khác hiện ở cột glm. Server giả đếm request, không tin vào log của agent-cli.

| Ca | Bản gốc `4a0b25b6`: rc / lời gọi glm | Bản sửa (worktree): rc / lời gọi glm | Lời gọi qwen (bản sửa) | Kết quả bản sửa |
|---|---|---|---|---|
| ok | 0 / **1** | 0 / 0 | title 1, main 1 | trả lời bình thường |
| provider-down (connection refused) | 1 / **1** | 1 / 0 | 0 | lỗi nêu địa chỉ + "connection refused" |
| http-500 | 1 / **1** | 1 / 0 | title 1, main 4 (1 + 3 lần thử lại) | lỗi "internal server error: upstream exploded" |
| http-400 | 1 / **1** | 1 / 0 | title 1, main 1 (không thử lại) | lỗi |
| malformed-stream (JSON hỏng) | 1 / **1** | 1 / 0 | title 1, main 1 | lỗi "invalid character…" |
| truncated-stream (đứt giữa chừng) | 1 / **1** | 1 / 0 | title 1, main 4 | lỗi "unexpected EOF"; phần chữ đã nhận giữ trong `text` |
| empty-reply (không chữ, không tool) | **0** / **1** | **0** / 0 | title 1, main 1 | **rc=0, `text` rỗng** → PRB-75 |
| hang (provider không trả lời 60 s) | 124 / **1** | 124 / 0 | title 1, main 1 | `run` không có trần thời gian riêng; bị `timeout 40` ngoài giết; JSON vẫn in, `error: context canceled` |

Đọc kết quả:

- **Cô lập provider:** bản gốc gửi lời gọi tiêu đề (kèm nguyên prompt đầu) sang provider khác ở **cả 8/8 ca**, kể cả khi provider chính chết. Bản sửa: 0/8.
- **Không fallback chéo provider khi lỗi:** cả hai bản đều không chuyển lời gọi main sang `glm` khi `qwen` hỏng. Lỗi lộ ra, rc=1.
- **Thành công giả:** 7/8 ca lỗi ra rc≠0. Ca `empty-reply` ra rc=0 ở cả hai bản. Mở PRB-75, sửa nhỏ ở Daily: lượt không có chữ lẫn tool call ở mọi bước thì là lỗi.
- **Treo:** `run` không có trần thời gian. Script phải tự bọc `timeout`; `loop`/`routine` có `--run-timeout`. Đây là giới hạn đã biết, ghi tài liệu.

## 2. Chết giữa chừng

| Ca | Cách kiểm | Kết quả | Hạng |
|---|---|---|---|
| Tự tóm tắt khi window nhỏ (lặp vô hạn) | P-04 (binary, LLM giả) + `summarize_overflow_test.go` | bản gốc 755+755 lời gọi / 90 s; bản sửa dừng có lỗi "context window too small" | E3 |
| Routine bị SIGTERM | E2E routine mục 7 (có sẵn) | CANCELLED, báo cáo ghi, khoá nhả | E3 (binary) |
| Routine bị SIGKILL | E2E routine **mục 8 (mới)** + `TestLastReport_AbandonedRunsAreShownNotHidden` | mục tiêu: lượt cuối ABANDONED, không hiện báo cáo cũ. Kết quả ở 37 | E2 → E3 khi E2E chạy |
| Server chết giữa tool call | CLAUDE.md P8 (15/10, binary cũ) | tool call bỏ dở ghi "Interrupted" khi mở lại | E4 (thủ công) |
| Lời trả lời chữ/rỗng dở sau sập | `unfinished_reply_test.go` | đóng bằng finish lỗi khi mở session | E2 |
| Client `run` bị SIGKILL (chế độ server) | E2E run-json (có sẵn) | run vẫn chạy trên server | chạy lại ở 37 |

## 3. Chạy dài và áp lực hàng đợi (§67)

| Hàng đợi | Có quan sát được không | Ghi chú |
|---|---|---|
| Prompt xếp hàng trong một session | có: nhãn "N Queued" trong TUI (E1, `styles.go` QueueLabel; chưa xem bằng mắt đợt này) | mất khi server SIGKILL (PRB-16, mở) |
| Hộp thoại quyền | có, mỗi session một hộp; chờ huỷ được (PRB-06 đã sửa) | — |
| Job bash nền | `agent-cli jobs`, Alt+J | PRB-15/18/31 đang sửa |
| Routine | `routine list/show` hiện RUNNING/ABANDONED/lượt gần nhất | không có hàng đợi: khoá chống chạy chồng trả rc=75 |
| Lời gọi LLM đồng thời | sổ ghi LLM (mới) cho biết từng lời gọi và vai trò | không có trần đồng thời theo provider (không có vấn đề đo được) |

Chưa chạy: routine dài nhiều giờ với LLM thật; tải nhiều session song song trên một server. Không có vấn đề đo được dẫn tới hai ca này. Nếu Daily F (nhiều session) cho thấy vấn đề thì chạy.
