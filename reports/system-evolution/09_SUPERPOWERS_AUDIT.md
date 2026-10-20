# 09 — Audit Superpowers (Phase 2)

Superpowers v7.0.0 (`obra/superpowers`, vendor `bb92a77`), 13 skill cài vào `~/.agents/skills` (+ `using-superpowers`). `diagnosing-superpowers` và `writing-skills` bị loại lúc cài. Audit chỉ-đọc bởi agent riêng; báo cáo thô kèm `file:dòng` ở `scratchpad/evo/p2-results/superpowers.json`.

## Cách Superpowers được thiết kế để chạy, và chỗ agent-cli khác

- Upstream tiêm `using-superpowers` lúc SessionStart rồi gọi tool `Skill`. agent-cli không có hai cơ chế này.
- Luật 14 + `<skills_usage>` của agent-cli **đã là** bộ khởi động skill (kiểu "danh mục skill hiện sẵn" mà hướng dẫn port của superpowers mô tả). Thêm `using-superpowers` chỉ chồng một luật bắt buộc thứ hai, với ngưỡng thấp hơn ("1% khả năng" thay cho "khớp").
- **Agent con của agent-cli chỉ đọc** (glob, grep, ls, view), không chọn model được, không resume được, không có skill hay context, và system prompt dặn "trả lời một từ". Vì vậy mọi mẫu "agent con làm code" đều không chạy được. Ngược lại, mẫu "agent con review chỉ đọc" lại rất hợp.
- 0 lần xuất hiện TodoWrite, AskUserQuestion, "Task tool", "Skill tool" trong 13 SKILL.md: các skill mô tả hành động nên phần lớn chuyển được sang tool của agent-cli.

## Từng skill

| Skill | Tương thích | Khuyến nghị | Lý do ngắn |
|---|---|---|---|
| using-superpowers | DEGRADED | DISABLE_IN_AGENT_CLI | luật 14 đã làm việc này; skill chỉ hạ ngưỡng nạp cho 47 skill khác và nhắc cơ chế không có |
| subagent-driven-development | BROKEN | DISABLE_IN_AGENT_CLI | không agent con nào ghi được code; mỗi lần nạp tốn 32,6 KB |
| dispatching-parallel-agents | BROKEN | DISABLE_IN_AGENT_CLI | agent con sửa song song là không thể; phần chỉ-đọc đã có trong prompt agent-cli |
| verification-before-completion | OK | MOVE_TO_POLICY | tự nhận là luôn bật nhưng 0/41 lượt được nạp → 3 dòng trong context toàn cục |
| brainstorming | DEGRADED | KEEP_BUT_FIX | đáng giữ cho thiết kế thật trong TUI; thu hẹp trigger bằng luật ngoài tệp vendor |
| writing-plans | DEGRADED | KEEP_BUT_FIX | dự án thật sự viết plan; người thi hành mặc định (SDD) hỏng → luật: dùng executing-plans, format brief |
| executing-plans | DEGRADED | KEEP_BUT_FIX | người thi hành duy nhất hợp agent-cli; commit theo từng task va với "không commit khi chưa được phép" |
| using-git-worktrees | DEGRADED | KEEP_BUT_FIX | `cd` không giữ qua các lệnh bash của agent-cli → mở session mới trong worktree |
| requesting-code-review | DEGRADED | KEEP_BUT_FIX | agent con không chạy được git → cha đưa diff cho agent con chỉ-đọc review; định dạng `KET-LUAN` của luật người dùng |
| finishing-a-development-branch | OK | KEEP | không merge/push/bỏ khi chưa được chọn |
| receiving-code-review | OK | KEEP | trigger hẹp, hợp luật "được phản biện" |
| systematic-debugging | OK | KEEP | đúng skill cho "fix this failing test" |
| test-driven-development | OK | KEEP | khớp luật "TDD vẫn giữ khi sửa logic" |

## Xung đột chính

1. Pipeline dựa trên commit (writing-plans, executing-plans, SDD, brainstorming, using-git-worktrees) va với mặc định không commit của agent-cli (luật 6) và luật dự án (P1).
2. Các điểm dừng chờ người (HARD-GATE của brainstorming, menu kết thúc, hỏi đồng ý tạo worktree, dừng sau 3 lần sửa hỏng) va với "đừng hỏi, tự quyết" của agent-cli, và với chạy không người trực.
3. Thứ bậc luật của người dùng (house rule > superpowers > ponytail; việc nhỏ bỏ nghi lễ) không tới được agent-cli.

## Chi phí

- 13 dòng trong danh sách tốn 4.143 B mỗi request.
- Tổng thân skill là 129.185 B.
- Chuỗi điển hình cho "fix this failing test" (systematic-debugging + TDD + verification) nạp 22.754 B. Thêm `using-superpowers` là +3.192 B; thêm requesting-code-review là +9.423 B.
- Lượt thật duy nhất có nạp superpowers đã nạp 5 skill cùng lúc.

## Đóng góp thật cho agent-cli hôm nay

- **Đo được:** 1/41 lượt có nạp superpowers (5 skill). Không có dữ liệu skill có được làm theo hay có ích (U-3).
- **Đóng góp tiềm năng thật:** systematic-debugging và TDD, nếu được nạp đúng lúc. Phương pháp review chỉ-đọc cũng hợp với agent con của agent-cli.
