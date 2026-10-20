# 08 — Audit Ponytail (Phase 2)

Ponytail v5.2.0 (tag `v5.2.0`, commit `2961b52`, MIT), vendor ở `~/.agents/vendor/ponytail`, cài vào `~/.agents/skills` bằng `~/.agents/update-skills.sh` (ghi đè mỗi lần cập nhật; `ponytail-help` bị loại lúc cài). Audit chỉ-đọc bởi agent riêng; mọi khẳng định dưới đây có `file:dòng` trong báo cáo thô `scratchpad/evo/p2-results/ponytail.json`.

## Ponytail được bật thế nào

- **Ở Claude Code:** chạy như plugin. Hook SessionStart (`ponytail-activate.js`) chọn mức (env, config, `full`), ghi cờ, rồi tiêm thân skill theo mức vào mọi phiên. Nghĩa là nó **luôn bật**, không cần model tự nạp.
- **Ở agent-cli:** không có hook SessionStart. Ponytail chỉ là 5 dòng trong `<available_skills>`. Chính tài liệu của Ponytail nói cài chỉ-skill thì không có chế độ luôn bật (`INSTALL.md:206`, `README.md:148,167`). Log 41 lượt thật cho thấy chưa lần nào nạp.

## Từng skill

| Skill | Tương thích | Khuyến nghị | Lý do ngắn |
|---|---|---|---|
| ponytail | DEGRADED | MOVE_TO_POLICY | nó là luật "luôn bật"; ở agent-cli chỉ là dòng 557 B không bao giờ được nạp → đưa ~1 KB phần cốt lõi vào context toàn cục `~/.config/agent-cli/CRUSH.md` |
| ponytail-review | OK | KEEP_BUT_FIX | một lần, chỉ đọc, hợp agent-cli; v5 đã thành review chất lượng đầy đủ (không còn chỉ săn over-engineering như v4) — luật của người dùng vẫn mô tả theo v4 |
| ponytail-audit | OK | KEEP_BUT_FIX | như trên, cho cả repo; đọc cả repo có thể chạm ngưỡng tự tóm tắt |
| ponytail-debt | OK | KEEP | rẻ (391 B), chỉ khi được gọi; có marker `shortcut:` thật để thu |
| ponytail-gain | DEGRADED | DISABLE_IN_AGENT_CLI | bảng điểm cứng, trỏ tới lệnh slash không có ở agent-cli; 345 B mỗi request, không làm việc |
| ponytail-help | BROKEN | (đã loại lúc cài) | chỉ mô tả tính năng plugin/hook |

## Xung đột

1. **Với system prompt agent-cli** (`coder.md.tpl`):
   - Luật 8 "NEVER ADD COMMENTS" mâu thuẫn với luật comment `shortcut:` của Ponytail.
   - "Don't add … tests to codebases that don't have them" mâu thuẫn với luật "để lại một test/self-check nhỏ".
   - "Trả lời bằng ngôn ngữ của prompt" mâu thuẫn với báo cáo "Very simple English" của review/audit.
2. **Với thứ bậc của người dùng.** `~/.claude/CLAUDE.md` (CLAUDE.md → superpower → superpowers → ponytail; việc nhỏ bỏ nghi lễ; ponytail chỉ nới khối lượng code, không cắt test) **không được agent-cli nạp**. Hai tệp context toàn cục của agent-cli (`~/.config/agent-cli/CRUSH.md`, `~/.config/AGENTS.md`) chưa tồn tại.
3. **Lệch phiên bản.** Claude Code ghi plugin 4.13.0 trong khi agent-cli đọc bản 5.2.0. Chưa kiểm Claude Code thật sự chạy bản nào.

## Chi phí

Danh sách 5 skill tốn 2.425 B mỗi request (7,6% khối skill, khoảng 606 token theo byte/4). Khi nạp: 2,2–6,2 KB mỗi skill.

## Đóng góp thật cho agent-cli hôm nay

**Không có gì đo được.** Không lượt thật nào nạp Ponytail. Các quy tắc thay thế đã có sẵn trong system prompt agent-cli: "đơn giản", "đọc trước khi sửa", "find_references trước khi đổi code dùng chung".

## Việc đề xuất (cần người dùng duyệt vì đụng cấu hình thật)

- Thêm `options.disabled_skills: ["ponytail", "ponytail-gain"]` vào cấu hình agent-cli.
- Tạo `~/.config/agent-cli/CRUSH.md` gồm thứ bậc luật của người dùng và phần cốt lõi của Ponytail (khoảng 1 KB), dùng cách gỡ xung đột với luật 8 và luật test như `~/.claude/CLAUDE.md` đã làm.
- Không sửa SKILL.md trong `~/.agents/skills`, vì `update-skills.sh` sẽ ghi đè.
