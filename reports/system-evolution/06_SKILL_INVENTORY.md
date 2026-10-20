# 06 — Kiểm kê skill (Phase 2)

Đo 2026-10-16. Nguồn: `scratchpad/evo/p2-skill-inventory.tsv` (quét đĩa), probe P-05 (khối `<available_skills>` thật trong request), audit Phase 2 (3 agent chỉ-đọc), log `Skill turn summary` của binary đã cài.

## Nguồn skill agent-cli quét (mã: `internal/config/load.go:1382-1418`, `internal/skills/skills.go:236-300`)

builtin (3) · `~/.config/crush/skills` · `~/.config/agents/skills` · `~/.agents/skills` · `~/.claude/skills` · và trong dự án: `.agents|.crush|.claude|.cursor/skills` ở cwd và ở gốc git. Quét đệ quy, theo symlink. Trùng tên: bản cuối theo thứ tự đường dẫn chữ thường thắng. **Không quét `~/.config/agent-cli/skills`.** `CRUSH_SKILLS_DIR` thay cả 4 thư mục toàn cục.

## Trên máy này

| Nhóm | Số đang hoạt động trong prompt | Byte trong prompt | Nguồn | Ghi chú |
|---|---|---|---|---|
| builtin | 3 (crush-config, crush-hooks, jq) | ~1.200 | embed | crush-config còn chỉ `~/.config/crush` (PRB-54) |
| superpowers v7.0.0 | 13 + `using-superpowers` | (xem 09) | `~/.agents/skills` ← vendor `bb92a77` | cập nhật bằng `~/.agents/update-skills.sh` (ghi đè) |
| ponytail v5.2.0 | 5 (ponytail, -audit, -debt, -gain, -review) | 2.425 | `~/.agents/skills` ← vendor `2961b52` | `ponytail-help` bị loại lúc cài |
| cá nhân | superpower, code-en-doc-vi, flow-tu-dong, glossary-keeper, retro (ẩn khỏi prompt: `disable-model-invocation`), writing-go-postgres-repositories | (xem 10) | `~/.claude/skills` (+ symlink từ `~/.agents`) | `code-en-doc-vi`: 2 bản khác nhau, bản `~/.claude` (1.185 B) che bản `~/.agents` (4.246 B) |
| Claude app "synced" | 14 tên (2 thư mục UUID trùng nhau) | 14.506 | `~/.claude/skills/synced/` | hướng dẫn tool của app Claude (Docs connector, browser…) |
| `.trash` | 8 (trước sửa) → 0 | 4.630 → 0 | `~/.claude/skills/.trash/` | skill người dùng đã xoá; worktree bỏ thư mục ẩn |
| **Tổng** | **48 → 40** | **31.860 → 27.169** | | 60% system prompt trước sửa |

Byte từng nhóm: cộng các mục `<skill>` trong system prompt bắt được ở P-05, theo đường dẫn `location` (đo lại 16/10; bản trước ghi trash 6.100, synced ~13.000 là ước sai). Tổng sau sửa đo bằng probe trên binary worktree.

## Hiệu quả thật (log của binary đã cài, 41 lượt, 3 ngày, provider thật)

- Lượt có danh sách skill: 41/41 (48–51 skill mỗi lượt).
- Lượt có nạp skill: **2/41**. Lượt thứ nhất nạp `redamon-runtime-reference` (skill của dự án khác). Lượt thứ hai nạp **5 skill cùng lúc**: brainstorming, dispatching-parallel-agents, subagent-driven-development, superpower, test-driven-development.
- Không có cơ chế nào ghi skill được **làm theo** hay **có ích** (USED/EFFECTIVE: không đo được).
