# 18 — Ma trận tương tác (Phase 5 + Phase 15)

Phân loại: SYNERGY · NEUTRAL · OVERLAP · INTERFERENCE · CONFLICT. "Đo" = có lệnh/log/test; "suy" = từ mã.

| Cặp | Phân loại | Bằng chứng | Ghi chú |
|---|---|---|---|
| Ponytail × Superpowers | INTERFERENCE (suy, chờ 08/09) | luật của người dùng xếp superpowers trên ponytail; ponytail "skip ceremony", superpowers "brainstorm/plan trước" | ở agent-cli không có thứ bậc nào được nạp (CLAUDE.md toàn cục không vào prompt) |
| Skill × hệ thống prompt agent-cli | CONFLICT | C-3, log 5 skill/lượt | hai luật bắt buộc |
| Skill × agent con | NEUTRAL (thiếu) | agent con không có skill (P-03) | skill dispatching/subagent-driven giả định agent con ghi được → không làm được ở agent-cli |
| Skill × provider | NEUTRAL | cùng danh sách cho mọi provider | chưa đo khác biệt hành vi theo model |
| Skill × hook | NEUTRAL | không có hook nào được cấu hình trên máy | — |
| Routine × không người trực × tool ghi | INTERFERENCE | verifier g5: edit/write không qua lớp chặn | PRB-70 |
| Routine × cấu hình dự án | CONFLICT | crushrc/hook của repo chạy trong lượt không người trực | PRB-68 (giới hạn đã biết) |
| Compaction × hàng đợi prompt | INTERFERENCE | g1 test: prompt xếp hàng chạy trước lượt làm tiếp | PRB-13 |
| Compaction × phần cố định của prompt | CONFLICT | P-04 vòng vô hạn | đã chặn (worktree) |
| Quyền × nhiều session | CONFLICT | test đỏ trước sửa | đã sửa (worktree) |
| Teardown workspace × job/MCP workspace khác | CONFLICT | g2 đọc ×2 | PRB-15 |
| Resume × model workspace | CONFLICT | g1 chạy | PRB-05 |
| Tiêu đề × model suy luận | INTERFERENCE | baseline thật: 2/2 lời gọi tiêu đề `finish=length` | tốn 2 lời gọi mỗi phiên, không ra tiêu đề |
| Sổ ghi LLM × mọi vai trò | SYNERGY | P-06 | một nguồn bằng chứng cho provider, token, vai trò |
| Memory × (mọi thứ) | NOT_APPLICABLE | không có thành phần memory | xem Phase 9–13 |
| Worktree × nhiều agent | NOT_APPLICABLE | agent con không ghi; routine một khoá mỗi routine | — |
