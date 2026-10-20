# 17 — Xung đột quyền sở hữu (Phase 5)

Cổng Phase 5: **không năng lực quan trọng nào có hai chủ vô tình tranh nhau.** Kết quả: **CHƯA ĐẠT**. Có 4 xung đột, 2 đã giảm ở worktree.

## C-1 · Ai chọn provider: người viết cấu hình hay máy?

- Hiện tại: crushrc của người dùng nói `model=…`. Nhưng lớp **global data `crush.json`** (máy ghi, mức cao hơn crushrc) giữ `models` từ lần chọn trong TUI, và nhận cả **fallback âm thầm** khi provider mất khoá (`load.go:166-180`). Lớp `<DataDir>/crush.json` (workspace, cao nhất) nhận `run --model`/resume.
- Hậu quả: một lần provider mất khoá biến thành đổi provider vĩnh viễn (PRB-04); xem lại session cũ đổi model mọi session mới (PRB-05).
- Đề xuất (Phase 12): **người dùng là chủ duy nhất của provider đang chọn.** Máy không bao giờ ghi lựa chọn provider như một tác dụng phụ (fallback, resume, `run --model`). Lựa chọn tạm cho một lượt/một session là trạng thái của lượt/session, không phải của cấu hình.

## C-2 · Ai sở hữu model của một session?

- Session lưu model ĐÃ chạy (từng tin) nhưng không sở hữu model SẼ chạy; model là trạng thái chung của workspace. Resume khôi phục bằng cách ghi vào workspace → ảnh hưởng mọi session.
- Đề xuất: model của session = model của tin assistant cuối, áp **trong bộ nhớ** cho lượt của session đó (đường local `app.restoreModelFromSession` đã làm thế, `OverridePreferredModel`); không ghi cấu hình. (Đường server cần mang model theo prompt → thay đổi giao thức, đánh giá ở Phase 12.)

## C-3 · Hai luật "kích hoạt skill bắt buộc" chồng nhau

- System prompt agent-cli: "description khớp thì PHẢI `view` SKILL.md trước mọi tool" (`coder.md.tpl` luật 14 + `<skills_usage>`). Skill `using-superpowers` (nếu được nạp) yêu cầu gọi skill trước MỌI phản hồi, kể cả câu hỏi làm rõ.
- Bằng chứng: lượt thật duy nhất có nạp superpowers đã nạp **5 skill liền** trong một lượt (log 2026-10-16 09:45). Tổng nội dung các skill đó cỡ hàng chục KB đưa vào context.
- Chủ đề xuất: **agent-cli** quyết định skill nào được liệt kê (cấu hình `disabled_skills`, đường quét); skill không được tự nâng mình thành "luôn bật". Chi tiết ở Phase 2 (06–10).

## C-4 · Verification: Stop hook vs loop/routine

- Hai cơ chế: Stop hook (opt-in, fail-open, chạy trong mọi lượt) và lệnh verify của `loop`/`routine` (tất định, fail-closed: UNMEASURED không bao giờ là xanh).
- Không chồng nhau ở runtime (hook chạy trong lượt; verify chạy sau lượt), nhưng **hai định nghĩa "xong"**. Đề xuất: `loop`/`routine` là chủ của "xong có bằng chứng"; Stop hook là cố vấn (chặn kết thúc lượt), không phát bằng chứng.

## Đã gỡ trong đợt này (worktree)

- **Provider của việc phụ** (tiêu đề, agentic_fetch) không còn tách khỏi provider của lượt chính (C-1 một phần).
- **Quyền của agentic_fetch**: bỏ tự duyệt session con, theo `disabled_tools` của người dùng (một chủ: cấu hình người dùng).
