# 23 — Ma trận test hệ thống (Phase 8)

Loại test theo workflow → cái đang có → cái thêm trong đợt này. "wt" = test mới ở worktree sửa (đều đỏ trước sửa, xanh sau, có đột biến bị bắt).

| Loại | Có sẵn (baseline) | Thêm trong đợt (wt / harness) |
|---|---|---|
| Unit | 68 gói có test, 2.617 hàm Test | ~30 test mới (xem 36) |
| Integration | test backend/agent với store thật (SQLite) | `TestBuildAgentModelsKeepsSmallOnTheMainProvider` (coordinator + config thật), `TestExecuteRoutine_*` (git thật) |
| Phiên mới | không test nội dung system prompt | probe P-03/P-05 (thành phần), sổ ghi (P-06) |
| Resume | E2E `-s/-C`; UI restore | kịch bản thật s10, s13 |
| Provider/model | allowlist (`TestCheckModelPolicy`), ghim model theo lượt | `TestSameProviderSmall`, P-01/P-07 (2 provider giả, đếm phía server), s13 thật |
| Kích hoạt skill | tracker, metadata view | log thật (41 lượt), audit Phase 2; KHÔNG có test hành vi "skill được làm theo" |
| Kế thừa agent con | test runSubAgent với mock | P-01 (agent con dùng model large, system 1 KB, 8 tool) |
| Tương tác | — | ma trận 18 (đo/suy) |
| Âm (negative) | safe-list, guard | `git -C` ra ngoài, viết tắt option, ghi ngoài workspace khi không người trực, tool bị tắt trong agentic_fetch |
| Tiêm lỗi | client SIGKILL (E2E), server SIGTERM | P-04 (window nhỏ → vòng lặp), routine bị kill (overlay g3), lời trả lời dở sau sập |
| False DONE | — (lượt thường không có verify) | `false_done` trong harness; s2/s6/s10 chặn sửa test; loop detection = lỗi; cờ GREEN không đổi / cổng xanh sẵn; báo cáo routine `touched_tests/touched_gate` |
| Độ tươi bằng chứng | không có khái niệm | báo cáo routine có `base_sha` + danh sách tệp đổi; chưa gắn revision cuối (mở) |
| Sập/khôi phục | settle tool call mồ côi | đóng lời trả lời chữ/rỗng; routine ABANDONED; journal fsync (có sẵn) |

## Chưa phủ (mang sang Phase 21–22 hoặc ghi giới hạn)

- Server SIGKILL giữa tool call với provider thật (thủ công trong CLAUDE.md P8; chưa tự động).
- Kịch bản "chỉ Qwen" đầy đủ mọi đường (tóm tắt, agentic_fetch, loop ở cấu hình 2 provider) — mới phủ main/title/subagent.
- Biến thiên của LLM thật (một lượt mỗi kịch bản).
- Test hành vi skill (có/không skill trên cùng việc).
