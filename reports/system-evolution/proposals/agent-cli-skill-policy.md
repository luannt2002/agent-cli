# Đề xuất: chính sách skill cho agent-cli (CHỜ NGƯỜI DÙNG DUYỆT)

Không tự áp: cả hai thay đổi nằm trong cấu hình thật của bạn (`~/.config/agent-cli/`), mà CLAUDE.md P7 của dự án cấm đụng vào. Căn cứ: Phase 2 (06–10), log 41 lượt thật, probe P-05.

## 1. Tắt skill không dùng được hoặc chồng chéo

Thêm vào `options` trong `~/.config/agent-cli/crush.json` (tệp này đang symlink tới `<repo>/.crush.json`):

```json
"disabled_skills": [
  "using-superpowers", "subagent-driven-development", "dispatching-parallel-agents", "verification-before-completion",
  "ponytail", "ponytail-gain",
  "superpower", "flow-tu-dong", "code-en-doc-vi",
  "docs", "docx", "pptx", "xlsx", "google-workspace", "computer-use", "chrome-browser",
  "built-in-browser", "deep-research", "import-memory", "morning", "skill-creator"
]
```

Đo 16/10 (E3: provider giả, skill THẬT của máy, `EXTRA_OPTS=',"disabled_skills":[…]' bash scratchpad/evo/probe-context.sh <dir> <binary>`):

| | Request main | Khối skill | Số skill |
|---|---|---|---|
| Binary gốc `4a0b25b6` | 78.972 B | 31.860 B | 48 |
| Binary sửa (bỏ thư mục ẩn) | 74.240 B | 27.169 B | 40 |
| Binary sửa + danh sách trên | 56.621 B | 9.656 B | 19 |

Còn lại 19 skill: brainstorming, crush-config, crush-hooks, executing-plans, finishing-a-development-branch, glossary-keeper, jq, pdf, ponytail-audit, ponytail-debt, ponytail-review, receiving-code-review, requesting-code-review, systematic-debugging, test-driven-development, ui-ux-pro-max, using-git-worktrees, writing-go-postgres-repositories, writing-plans. Request đầu mỗi lượt nhỏ đi 22.351 B (28%) so với bản gốc.

Rủi ro: một skill bị tắt mà model thật cần thì sẽ không được nạp nữa. Đối chứng: 41 lượt thật chỉ 2 lượt nạp skill. Trong các skill được nạp ở 2 lượt đó, `superpower`, `dispatching-parallel-agents` và `subagent-driven-development` nằm trong danh sách tắt. Phương pháp của chúng được đưa vào mục 2.

## 2. Context toàn cục `~/.config/agent-cli/CRUSH.md` (mới, khoảng 1,7 KB)

agent-cli nạp tệp này vào mọi phiên, xem `load.go:603-608`. Nó thay cho các skill "luôn bật" mà agent-cli không có cơ chế bật.

```markdown
# Luật làm việc (mọi dự án)

Thứ bậc: hướng dẫn của dự án (CLAUDE.md/AGENTS.md) > tệp này > skill. Việc nhỏ (sửa một chỗ, câu hỏi) bỏ
brainstorming/writing-plans/worktree; sửa logic vẫn viết test đỏ trước.

## Đòi bằng chứng
- Không nói "xong/được/pass" khi chưa chạy lệnh kiểm trong lượt này; nêu lệnh và kết quả. `rc=0` là máy có chạy, chưa chắc đúng.
- Số trong báo cáo phải từ lệnh vừa chạy; chưa đo thì ghi "chưa đo". `grep` ra 0 chưa phải "không có": mở tệp đọc.

## Sửa gọn
- Thay đổi nhỏ nhất giải đúng bài; không thêm abstraction/config/tính năng ngoài yêu cầu; dùng lại mã sẵn có.
- Sửa phẫu thuật: không refactor chỗ không hỏng, không xoá dead code có sẵn (thấy thì nói).
- Giới hạn biết trước: được thêm đúng một comment `shortcut: <giới hạn>, <khi nào nâng>` (ngoại lệ duy nhất của luật không comment).

## Ngôn ngữ
- Identifier, tên tệp, log: tiếng Anh. Comment, tài liệu, giải thích: tiếng Việt, giữ keyword kỹ thuật tiếng Anh.
- Không đổi tên API có sẵn chỉ vì ngôn ngữ.

## Git
- Không commit, không push khi người dùng chưa nói rõ trong phiên này; lời cho phép chỉ áp cho đúng lần đó.
- Plan (khi cần): brief gồm mục tiêu · ràng buộc · tiêu chí nghiệm thu; người thi hành là chính agent (executing-plans), không giao việc ghi cho agent con.

## Review
- Người chấm khác người làm: review bằng tool `agent` (chỉ đọc) với diff + yêu cầu, không đưa kết luận của người làm;
  dòng đầu `KET-LUAN: DAT` hoặc `KET-LUAN: KHONG-DAT`, rồi: sửa gì · làm gì · mục đích · rủi ro.

## Hỏi hay tự quyết
- Phiên có người (TUI): phân vân về sản phẩm thì hỏi một câu; còn lại tự quyết và nói giả định.
- Chạy không người trực (routine/loop): không hỏi; ghi giả định vào câu trả lời cuối.
```

## 3. Việc khác (không phải cấu hình agent-cli)

- `~/.agents/skills/code-en-doc-vi` là bản cũ (4.246 B, bắt commit tiếng Việt), khác bản trong `~/.claude` (1.185 B). Nên thay bằng symlink tới bản `~/.claude` để ZCode/Codex dùng cùng một bản.
- `retro`: thêm `user-invocable: true` để gọi được từ palette của agent-cli. Khi đó vị trí log trong skill cần trỏ thêm tới `agent-cli logs`.
