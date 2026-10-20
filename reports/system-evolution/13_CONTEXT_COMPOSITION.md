# 13 — Thành phần context (Phase 3)

Đo trên request thật mà agent-cli gửi (bắt bằng `scripts/eval/fakellm.py`), binary baseline `4a0b25b6`. Byte là UTF-8 của chính request; token thật (provider `main/qwen3.8`) chỉ có cho cả request: 18.605 token cho request đầu ở môi trường thật.

## Lượt chính (coder)

| Thành phần | Byte | Nguồn | Ghi chú |
|---|---|---|---|
| Template coder (19 khối luật, trước `<env>`) | ≈ 18.900 | `internal/agent/templates/coder.md.tpl` | khối lớn nhất: `critical_rules` 2.830, `editing_files` 2.707, `decision_making` 1.723 |
| `<env>` + snapshot git | 337–452 | `prompt.go:208-288` | git status ≤ 20 dòng + log 3 commit, chụp lúc dựng coordinator |
| `<available_skills>` | **1.197 → 31.860** | `skills.go:299-322` | 3 skill builtin → 48 skill thật; trong đó 19.136 B đến từ `.trash` + `synced` |
| `<skills_usage>` + luật 14 | 1.557 + 412 | `coder.md.tpl:19, 390-410` | luôn có khi có skill |
| Context file dự án | 0 trong fixture; **28.752** ở repo agent-cli (AGENTS.md 10.879 + CLAUDE.md 17.873) | `prompt.go:111-164` | đọc ở cwd, không giới hạn cỡ, thứ tự ngẫu nhiên |
| Context người dùng toàn cục | ~223 | `~/.config/agent-cli/CRUSH.md`, `~/.config/AGENTS.md` | |
| Reminder todo (tin user) | 358 | `agent.go:1971-1981` | luôn có, nội dung có thể sai |
| Schema tool | 25.775 (26) / ít hơn khi tắt tool | `internal/agent/tools/*.md` | `bash` một mình 8.388 B |

## Vai trò khác

| Vai trò | System prompt | Tool | Ghi chú |
|---|---|---|---|
| Agent con (`agent`) | 1.031 B (`task.md.tpl`) | 8 chỉ-đọc, 5.426 B | không context file, không skill |
| Tiêu đề | 492 B | 0 | gửi nguyên prompt đầu; `max_tokens=40` |
| Tóm tắt | 1.752 B | 0 | model large của lượt |
| agentic_fetch | template riêng | tối đa 6 | model small (trước sửa) |

## Đánh giá

1. **Phần cố định chiếm ưu thế.** Ở môi trường thật, khoảng 79 KB trước khi có việc gì; skill là phần lớn nhất (40%), tool là phần thứ hai (33%).
2. **Skill là phần rẻ nhất để cắt mà không đổi hành vi lõi.** Bỏ `.trash` (đã làm ở worktree) giảm 4.732 B/request. Bỏ thêm 14 skill `synced` (các skill này hướng dẫn tool mà agent-cli không có) sẽ giảm thêm khoảng 14 KB. Đây là quyết định của người dùng (ghi ở tệp 28).
3. **Phần cố định quyết định ngưỡng tràn context.** Vùng lặp tóm tắt vô hạn (PRB-01) là [0,8·W, W). Với 79 KB ≈ 20k token, mọi model có window ≤ 25k token đều rơi vào vùng nguy hiểm (đã chặn ở worktree).
4. **Không có ngân sách context.** Không cơ chế nào giới hạn tổng skill/context file/MCP trong prompt.
