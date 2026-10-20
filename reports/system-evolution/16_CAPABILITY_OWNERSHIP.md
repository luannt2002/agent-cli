# 16 — Chủ sở hữu năng lực (Phase 5)

Một năng lực, một chủ chính. Cột "Ghi" là ai ghi nguồn sự thật; "Xung đột" trỏ sang [17_AUTHORITY_CONFLICTS.md](17_AUTHORITY_CONFLICTS.md). Đo ở HEAD `4a0b25b6`.

| Năng lực | Chủ chính | Cố vấn | Nguồn sự thật | Ghi | Đọc | Chính sách khi xung đột |
|---|---|---|---|---|---|---|
| Lập kế hoạch | model chính (coder; plan mode khi bật) | skill writing-plans/brainstorming (nếu được nạp) | transcript | model | người | không có chủ máy; plan mode không có context dự án (PRB-20) |
| Chia việc | model chính qua `todos` | skill superpowers | `sessions.todos` | model | UI, prompt tóm tắt | tự khai, không gắn bằng chứng |
| Điều phối lượt | `sessionAgent.Run` + coordinator | — | RAM (activeRequests, queue) | agent | backend | một lượt/session, prompt xếp hàng |
| Sinh agent con | tool `agent` / `agentic_fetch` | skill dispatching-parallel-agents (giả định agent con ghi được — sai ở agent-cli) | session con | coordinator | cha (chỉ chữ cuối) | — |
| Chọn provider | **chưa rõ chủ**: lớp cấu hình cao nhất thắng | allowlist (`execpolicy`) chặn | cấu hình gộp | người (crushrc), **máy** (global data, workspace) | coordinator mỗi run | C-1 |
| Chọn model của lượt | coordinator (`pinnedModel`) | — | RAM + cấu hình | coordinator | lượt | đúng |
| Chọn model của session | **không ai** (session chỉ ghi lịch sử) | resume "mượn" workspace | messages.model | — | resume | C-2 |
| Kích hoạt skill | model (đọc SKILL.md bằng `view`) | luật bắt buộc trong system prompt + `using-superpowers` | metadata kết quả `view` | model | tracker, `session show` | C-3 (hai luật "bắt buộc" chồng nhau) |
| Git | người dùng; agent qua bash | routine (worktree riêng) | repo | người/agent | mọi bên | không người trực: danh sách trắng git |
| Worktree | `routine run` | — | `git worktree` | routine | người | giữ lại để duyệt; không dọn |
| Context | prompt builder (một lần mỗi coordinator) | context file, skill | đĩa | người/agent | coder | ảnh chụp, không reload (PRB-20) |
| Compaction | `sessionAgent` (ngưỡng 20%) | — | messages + `summary_message_id` | agent | lịch sử gửi model | đã chặn lặp vô hạn (worktree) |
| Memory dài hạn | **context file** (AGENTS.md…) do người/agent tự sửa | luật "memory files" trong template | đĩa | người/agent | coder | không có memory riêng |
| Review | **không có** | skill requesting/receiving-code-review | — | — | — | MISSING |
| Verification | **`loop`/`routine`** (lệnh verify tất định) | Stop hook (opt-in, fail-open) | mã thoát verify | driver | report/journal | lượt thường không có verify |
| Evidence | journal loop + `report.json` routine | sổ ghi LLM (mới) | tệp | driver | người | gắn dấu vân tay, không gắn revision |
| Quyền | permission service | hook PreToolUse (allow/deny), BlockFunc (chặn cứng) | cấu hình + RAM | người | tool | BlockFunc > YOLO/allowlist > hook allow > hỏi |
| Lưu trữ | `internal/db` (SQLite mỗi dự án) | — | `.crush/crush.db` | agent | mọi reader | một server mỗi data dir (flock); local không khoá (PRB-65) |
| Khôi phục | reader (`ReadSettledMessages`) | — | messages | reader | UI | settle khi không bận; chữ dở nay cũng được đóng (worktree) |
| Long-run | `routine` (khoá flock = lease) | `loop` engine | report/journal | driver | `list/show` | ABANDONED khi khoá nhả mà báo cáo còn RUNNING (worktree) |

## Kế thừa của agent con (Phase 17, đo)

| Thứ | Agent con (`agent`) | agentic_fetch |
|---|---|---|
| Chính sách gốc (template) | `task.md.tpl` riêng, chỉ env | template riêng |
| Context file dự án | **không** | không |
| Skill | **không** (view đọc được nếu biết đường dẫn) | không |
| Provider/model | slot large, ghim theo lượt | slot small (sau sửa: cùng provider) |
| Quyền | dịch vụ chung; KHÔNG kế thừa tự duyệt của cha (trừ đường `run` qua RootID) | trước: tự duyệt session con; sau: không |
| Memory/ngữ cảnh | chỉ chuỗi prompt cha đưa | prompt + nội dung web |
| Phạm vi ghi | không có tool ghi | thư mục tạm |
| Worktree | cùng thư mục với cha | thư mục tạm |
| Hook | **không** chạy PreToolUse | không |
