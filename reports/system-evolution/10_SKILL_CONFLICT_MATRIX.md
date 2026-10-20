# 10 — Ma trận xung đột skill (Phase 2)

Tổng hợp 3 audit (Ponytail, Superpowers, cá nhân + synced) và log thật. Mức: CONFLICT (hướng dẫn trái nhau), INTERFERENCE (làm hỏng nhau ở runtime), OVERLAP (trùng việc), DRIFT (giả định harness khác).

## Với chính agent-cli

| Skill / nhóm | Đụng với | Mức | Ghi chú |
|---|---|---|---|
| using-superpowers | luật 14 + `<skills_usage>` | INTERFERENCE | hai luật bắt buộc chồng nhau; log: 5 skill/lượt |
| ponytail (`shortcut:` comment), TDD (thêm test) | luật 8 "NEVER ADD COMMENTS", "đừng thêm test vào codebase chưa có" | CONFLICT | gỡ bằng luật trong context toàn cục |
| superpower, writing-plans, executing-plans, SDD, writing-go-postgres-repositories, flow-tu-dong | luật 6 "chỉ commit khi được bảo", CLAUDE.md P1 | CONFLICT | luật chặt hơn thắng; skill nên viết "khi được phép" |
| brainstorming, glossary-keeper, receiving-code-review, systematic-debugging (hỏi/dừng) | luật 2 "đừng hỏi" | CONFLICT | TUI: theo luật người dùng (hỏi khi phân vân); không người trực: không hỏi |
| subagent-driven-development, dispatching-parallel-agents, deep-research | tool `agent` chỉ đọc | DRIFT → BROKEN | không ghi được qua agent con |
| docs, google-workspace, computer-use, chrome-browser, built-in-browser, import-memory, morning | không có connector/MCP/ToolSearch | DRIFT → BROKEN | `docs` còn cướp yêu cầu "viết tài liệu" rồi từ chối |
| docx, pptx, xlsx, skill-creator | giả định sandbox của claude.ai | DRIFT | thiếu phụ thuộc trên máy |
| ui-ux-pro-max | `${CLAUDE_PLUGIN_ROOT}` không được đặt | DRIFT | lệnh nguyên văn hỏng; sửa tại chỗ bị sync ghi đè |
| retro | `disable-model-invocation` + không `user-invocable` | DRIFT | không model lẫn người dùng gọi được ở agent-cli |
| crush-config (builtin) | launcher dùng `~/.config/agent-cli` | DRIFT | hướng dẫn sửa sai tệp (PRB-54) |

## Giữa các skill

| Cặp | Mức | Ghi chú |
|---|---|---|
| Ponytail × Superpowers | CONFLICT (chủ ý) | "bỏ nghi lễ / bản nhỏ nhất" vs "brainstorm/plan/TDD trước". Thứ bậc của người dùng gỡ được, nhưng không được agent-cli nạp |
| ponytail-review v5 × requesting-code-review | OVERLAP | v5 thành review chất lượng đầy đủ; luật người dùng còn xếp nó là lớp "lean" thứ hai (theo v4) |
| superpower × flow-tu-dong | OVERLAP + CONFLICT | trùng §4–§6; trái nhau về `--maxWorkers` |
| superpower (luật nhà) × máy này | DRIFT | mô tả box 4 vCPU, pm2, GitLab CE 11.6; máy này 12 luồng, GitHub |
| code-en-doc-vi (~/.claude 1.185 B) × code-en-doc-vi (~/.agents 4.246 B) | CONFLICT | bản cũ bắt commit tiếng Việt + chú thích bắt buộc; agent-cli dùng bản mới vì thứ tự đường dẫn, harness khác (ZCode) dùng bản cũ |
| docx/pptx/google-workspace × docs | INTERFERENCE | đều đẩy yêu cầu tài liệu sang `docs`, mà `docs` hỏng ở agent-cli |

## Phạm vi ảnh hưởng (byte mỗi request)

| Nhóm | Byte trong prompt | Đề xuất |
|---|---|---|
| synced hỏng/không cần (12) | ~12.000 | tắt |
| superpowers tắt (3) + chuyển luật (1) | ~600 | tắt / vào context |
| ponytail, ponytail-gain | ~900 | vào context / tắt |
| superpower, flow-tu-dong, code-en-doc-vi | ~1.100 | vào context / tắt |
| Còn lại sau đề xuất | ~13.000–14.000 | — |

Đề xuất cụ thể (cấu hình + nội dung context toàn cục) ở [proposals/agent-cli-skill-policy.md](proposals/agent-cli-skill-policy.md). Áp dụng là quyết định của người dùng vì đụng `~/.config/agent-cli`.
