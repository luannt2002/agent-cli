# 08 — Chủ sở hữu năng lực (§38, §39)

Bảng gốc (đo ở HEAD `4a0b25b6`): [16_CAPABILITY_OWNERSHIP](../system-evolution/16_CAPABILITY_OWNERSHIP.md). Ở đây chỉ ghi trạng thái sau ứng viên (`wt-final.patch`) và đối chiếu với mô hình đích §39. Một năng lực, một chủ chính.

| Năng lực | Chủ chính | Cố vấn | Nguồn sự thật | Ghi | Đọc | Chính sách khi xung đột | Thay đổi trong đợt |
|---|---|---|---|---|---|---|---|
| Lập kế hoạch / chia việc | model chính (`todos`) | skill (nếu nạp) | transcript, `sessions.todos` | model | UI, tóm tắt | tự khai, không gắn bằng chứng | — |
| Điều phối lượt | `sessionAgent.Run` + coordinator | — | RAM (hàng đợi, activeRequests) | agent | backend | một lượt/session; lượt làm tiếp chạy trước prompt mới | trần tự tóm tắt 10, lượt rỗng = lỗi |
| Sinh agent con | tool `agent`, `agentic_fetch` | — | session con | coordinator | cha | chỉ khi model gọi tool | session con không gọi tiêu đề; chi phí cộng cả khi lỗi |
| **Chọn provider** | **người dùng** (crushrc / TUI) | allowlist `execpolicy` | cấu hình gộp | người; máy ghi khi người chọn trong TUI | coordinator | lời gọi ẩn ở lại provider chính; không bao giờ đổi provider (model không dùng được thì dừng có lỗi) | C-1 đóng (G-01 + vòng 6) |
| Chọn model của lượt | coordinator (ghim theo lượt) | — | RAM | coordinator | lượt | đúng | — |
| Chọn model của session | **chưa có chủ** (workspace) | — | messages.model | — | resume | C-2 còn mở (PRB-05) | — |
| Kích hoạt skill | model (`view` SKILL.md) | luật trong system prompt | metadata `view` | model | tracker | C-3: đề xuất chính sách chờ duyệt | bỏ thư mục ẩn |
| Git / worktree | người dùng; routine (worktree riêng) | — | repo | người/agent | mọi bên | không người trực: danh sách trắng git + chặn biến git | chặn `-C` ra ngoài, viết tắt option, biến `GIT_*` |
| Context | prompt builder | context file | đĩa | người/agent | coder | chỉ cwd (PRB-20) | — |
| Compaction | `sessionAgent` | — | messages + summary | agent | lịch sử | bản cắt dở không thành điểm cắt; tràn → lỗi; trần 10 | ✓ |
| Memory | context file (M0) | — | đĩa (git) | người/agent | coder | memory không đè sự thật hiện tại | quyết định: không thêm kho memory |
| Review | không có trong sản phẩm | skill review | — | — | — | phương pháp: người soát ≠ người làm (dùng trong đợt này) | — |
| **Verification** | `loop`/`routine` (lệnh verify) | Stop hook (cố vấn) | mã thoát verify | driver | report | `run` rc=0 ≠ đã kiểm chứng | pipefail + gợi ý exit 141; verify xanh sẵn + cây không đổi = `NO_CHANGE` (vòng 6) |
| Evidence | journal loop + report routine | sổ ghi LLM (opt-in) | tệp 0600 | driver | người | report có base_sha, tệp đổi, cờ đụng test/cổng | ✓ |
| Quyền | permission service | hook PreToolUse, BlockFunc | cấu hình + RAM | người | tool | không người trực: từ chối ghi ra ngoài trước YOLO/allowlist | hộp thoại không chặn session khác; `run` local duyệt session con |
| Lưu trữ | `internal/db` | — | `.crush/crush.db` | agent | reader | một server mỗi data dir | — (retention mở, PRB-49) |
| Khôi phục | reader + routine | — | messages, report, flock | reader | UI, `routine show` | lời trả lời dở được đóng; routine bị kill = ABANDONED; lỗi sớm = ERROR | ✓ |
| Job nền | workspace (gốc) + session (chủ) | — | bảng job trong RAM | bash tool | Alt+J, `jobs`, job tools | chỉ session chủ đọc/dừng; đóng workspace chỉ dừng job của nó | ✓ |

## Đối chiếu mô hình đích §39

| Mô hình §39 | Ở agent-cli | Khớp? |
|---|---|---|
| Runtime: trạng thái thực thi hiện tại | backend/App/sessionAgent | khớp |
| Routine: điều phối việc dài | `routine` → engine `loop` → `run` in-process | khớp (dùng lại primitive, không có runtime thứ hai) |
| Skill Base: phương pháp tái dùng | skill quét từ đĩa | khớp; chính sách chọn lọc chờ duyệt |
| Ponytail / Superpowers | gói skill tuỳ chọn; runtime không phụ thuộc | khớp |
| Chính sách cá nhân | context file (đề xuất `~/.config/agent-cli/CRUSH.md`) | đề xuất, chưa áp |
| Memory lịch sử | context file + lịch sử session (M0) | khớp ở mức M0 |
| Provider: backend thực thi | một điểm chặn `buildAgentModels` + allowlist | khớp (vòng 6: fail closed khi model đang chọn không dùng được) |
| Git/nguồn/runtime: sự thật hiện tại | repo + test | khớp |
| Verification: chứng minh | lệnh verify của loop/routine | khớp; `run` không tự kiểm |
| Evidence: hồ sơ chứng minh | journal, report, sổ ghi | khớp; chưa gắn revision cuối vào report (mở) |

Hai chỗ còn hai chủ: model của session (C-2), và lớp phân tích lệnh shell có ba bộ song song ([03](03_RUNTIME_TRACE.md) §2).
