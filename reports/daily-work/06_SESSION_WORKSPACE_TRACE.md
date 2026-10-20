# 06 — Dấu vết workspace / session (Daily E, F)

Câu hỏi: nhiều dự án và nhiều session song song có lẫn ngữ cảnh, lịch sử hay danh sách session không?

## Cách đo

`bash scripts/eval/workspaces.sh <dir> <binary>`: binary thật, LLM giả ghi lại **chính request** model nhận (system prompt theo sha, đầu từng tin user). Hai repo A và B, mỗi repo có `AGENTS.md` chứa dấu riêng (`PROJECT-A-MARKER`, `PROJECT-B-MARKER`).

- E (A → B → A): `run` ở A, `run` ở B, rồi `run -s <session A>` ở A.
- F (cùng workspace A): session X, session Y, rồi `run -s <X>`.
- Sau đó `session list --json` ở từng dự án.

## Kết quả (E3, chế độ local, binary ứng viên `agent-cli-wtb2`, 16/10)

| Kiểm tra | Kết quả |
|---|---|
| 6 lượt `run` đều trả session id | ok |
| Mỗi lượt ở A thấy dấu của A, không thấy dấu của B (và ngược lại) — kiểm trong system prompt thật | ok (3/3 lượt) |
| `-s` ở A nối đúng session A (cùng id; request mang lịch sử của A1, không có gì của B) | ok |
| Session X tiếp tục mang lịch sử X1, không có Y1 | ok |
| `session list` của A có A1, X, Y và không có session của B; của B chỉ có B1 | ok |
| **Tổng** | **11/11 đạt** |

Lượt chạy lại trên binary cuối: xem [37](../system-evolution/37_FINAL_PROOF.md).

## Chủ sở hữu (khớp mã)

| Thứ | Chủ | Ghi chú |
|---|---|---|
| Context dự án (AGENTS.md…) | workspace (cwd khi nạp) | chỉ nạp ở cwd, chưa ở gốc git (PRB-20) |
| Lịch sử, todos | session (SQLite của dự án, `.crush/`) | mỗi dự án một DB |
| Job bash nền | workspace (gốc) + session (chủ) | job tool chỉ thấy job của session đang gọi (PRB-18 đã sửa); đóng một workspace chỉ dừng job của nó (PRB-15 đã sửa) |
| Model của lượt | lượt (ghim lúc bắt đầu) | đổi model ở session khác không đổi lượt đang chạy |
| Model "đang chọn" | **workspace** (chưa phải session) | resume/đổi session ghi vào workspace → session khác đổi theo (PRB-05 mở) |

## Chưa đo

- Chế độ server (một server mỗi dự án qua socket theo hash) với hai client cùng lúc: E2E `test-agent-cli-run-json.sh` phủ session id mỗi tiến trình ở chế độ server (chạy ở 37), chưa phủ hai dự án cùng lúc.
- Model rò giữa session (PRB-05): đã tái lập ở đợt System Evolution (05: BC = CHẠY), chưa có kịch bản tự động trong harness Daily.
