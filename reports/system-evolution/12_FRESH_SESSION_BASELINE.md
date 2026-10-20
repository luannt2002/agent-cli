# 12 — Baseline của một phiên mới (Phase 3)

Phiên mới thật sự (session mới, tiến trình mới, không truyền ngữ cảnh tay). Đo bằng binary baseline có gắn sổ ghi (`agent-cli-baseline-ledger` = `4a0b25b6` + chỉ phần sổ ghi, 5 tệp / 9 dòng thêm, không đổi hành vi).

## 1. Môi trường thật của người dùng, provider thật (kịch bản s1, 2026-10-16 18:29)

Cấu hình thật (`CRUSH_GLOBAL_CONFIG=~/.config/agent-cli`, skill thật trong HOME), dữ liệu tách riêng, chạy in-process. Dự án: repo Go nhỏ, không có context file.

| Đo | Giá trị |
|---|---|
| Provider/model thật (sổ ghi) | `main / qwen3.8` cho **mọi** lời gọi (crushrc chọn provider `main`; giá trị `base_url` nằm trong tệp env bí mật nên không biết là local hay trả phí) |
| Lời gọi LLM | 7: **2 tiêu đề** (đều `finish=length`, trần 40 token với model suy luận → không ra tiêu đề, mất 5,7 s) + 5 main |
| Token vào lời gọi main đầu tiên | **18.605** (tokenizer thật của provider) |
| System prompt | 48.259 B; 23 tool (cấu hình người dùng tắt sourcegraph/fetch/download) |
| Tổng token cho một sửa một dòng | 112.685 vào / 748 ra |
| Kết quả | verify xanh, 18,5 s, 7 tool call |

## 2. Thành phần (HOME cô lập vs skill thật), provider giả — P-03, P-05

| | HOME cô lập (3 skill builtin) | Skill thật của máy |
|---|---|---|
| Request main đầu tiên | 48.119 B | 78.972 B |
| System prompt | 22.040 B | 52.685 B |
| Khối skill | 1.197 B | 31.860 B (48 skill) |
| Schema tool | 25.785 B (26 tool) | 25.775 B |

## 3. Câu trả lời cho "cái gì chi phối phiên mới này"

- **Model:** provider/model của slot large cho lượt chính và agent con; slot **small** cho tiêu đề và agentic_fetch, kể cả khi small thuộc provider khác (P-01, đã sửa ở worktree).
- **Skill:** 48 skill (60% system prompt) dù 41 lượt thật chỉ 2 lượt nạp skill.
- **Context file:** AGENTS.md/CLAUDE.md… ở cwd (không có trong fixture; repo agent-cli sẽ thêm 28.752 B).
- **Reminder todo:** câu "todo list is currently empty" ở đầu lịch sử mọi lượt (PRB-22).
- **Phần tool:** 23–26 schema, khoảng 25 KB, gửi mọi bước.

## 4. Hệ quả cho các phase sau

- Mỗi bước của agent chính tốn khoảng 18–24k token vào trước khi đọc dòng code nào → chi phí nằm ở phần cố định (skill + tool), không ở việc.
- Tiêu đề bằng LLM thất bại 100% với model suy luận hiện tại nhưng vẫn tốn 2 lời gọi mỗi phiên (và mỗi agent con) → ứng viên cắt (PRB-46).
- Số đo token chỉ đáng tin qua sổ ghi; `run --json` báo bước cuối.
