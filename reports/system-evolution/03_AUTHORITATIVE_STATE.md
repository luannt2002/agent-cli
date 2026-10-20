# 03 — Trạng thái và nguồn sự thật (Phase 1)

Mỗi dòng: dữ liệu → nơi là sự thật → ai ghi → ai đọc → sống bao lâu. "RAM" = mất khi tiến trình chết. Đo ở HEAD `4a0b25b6`.

| Dữ liệu | Nguồn sự thật | Ghi | Đọc | Bền | Ghi chú / vấn đề |
|---|---|---|---|---|---|
| Model đang chọn (large/small) | Cấu hình gộp; lớp cao nhất thắng: `<DataDir>/crush.json` (workspace) > dự án > **global data `crush.json`** > global crushrc | TUI chọn model, `run/loop --model` (server: scope workspace), khôi phục khi resume (scope workspace), **fallback lúc khởi động (scope global)** | coordinator mỗi run, sidebar | tệp | Hai lớp máy ghi (global data, workspace) đè lớp người viết (crushrc) → PRB-04, PRB-05 |
| Model của một lượt | `pinnedModel` của lượt | coordinator lúc gửi prompt | lượt đó | RAM | đúng thiết kế; tin assistant lưu provider/model thật đã chạy |
| Provider/model đã chạy (lịch sử) | `messages.provider`, `messages.model` | session agent | resume, `session show` | SQLite | session KHÔNG có cột model/effort/mode/cwd/branch |
| Chính sách model (allowlist, giao việc, không người trực) | env chụp đầu `main` + `~/.config/agent-cli/env` | người dùng | `execpolicy.Current()` | đọc 1 lần/tiến trình | đổi tệp phải khởi động lại server |
| Quyền tool | `permissions.allowed_tools` (mảng nối qua các lớp), YOLO, grant theo session | cấu hình, hộp thoại | permission service | grant: RAM | trên máy này bash/edit/write luôn tự duyệt |
| Phiên làm việc + tin nhắn | `<dự án>/.crush/crush.db` (sessions, messages) | session agent (debounce 33 ms, flush khi kết thúc) | mọi reader qua `ReadSettledMessages` | SQLite | xoá session không xoá session con (PRB-11) |
| Phiên bản tệp do tool ghi | bảng `files` (đủ nội dung) | 5 tool ghi | sidebar "Modified Files" | SQLite | không có undo dùng nó |
| Todo | `sessions.todos` (JSON) | tool `todos` (model tự khai) | pill UI, prompt tóm tắt | SQLite | "completed" không gắn bằng chứng; reminder luôn nói rỗng (PRB-22) |
| Hàng đợi prompt, RunID đang chạy | `messageQueue`, `activeRequests` | backend/agent | agent | RAM | mất khi sập, không dấu vết (PRB-16) |
| Job bash nền | singleton `BackgroundShellManager` | tool bash | `jobs`, Alt+J, `job_kill/job_output` | RAM (tiến trình con sống sót) | ID đếm lại từ 001 mỗi tiến trình; tool agent không kiểm owner (PRB-18) |
| Skill | thư mục skill (quét lúc dựng) + builtin | người dùng | system prompt coder (snapshot), tracker | snapshot RAM | quét cả `.trash`, `synced` (PRB-08, đã sửa phần `.trash`) |
| Skill đã nạp | tracker theo **workspace** + metadata kết quả `view` | tool view | `session show`, crush_info, log | tracker RAM; metadata SQLite | "loaded_this_session" thật ra là cả workspace (PRB-21) |
| Context file (AGENTS.md, CLAUDE.md…) | đĩa, đọc ở **cwd** | người / agent | system prompt coder (snapshot) | tệp | chỉ tên hoa trên Linux; không đọc ở gốc git; plan/agent con không có (PRB-19, PRB-20) |
| Journal loop | `<data>/loops/<tên>-<ts>.jsonl` hoặc `runs/<stamp>/*.jsonl` | engine (fsync trước mỗi bước) | người đọc | tệp 0600 | không dọn |
| Báo cáo routine | `routines/<tên>/runs/<stamp>/report.json` | `executeRoutine` | `routine list/show` | tệp 0600 | lượt bị kill không có báo cáo → bị giấu (PRB-02, đã sửa) |
| Worktree routine | repo người dùng (`git worktree`), nhánh `routine/*` | `executeRoutine` | người duyệt | git | không dọn, tích luỹ (PRB-23) |
| Khoá routine | flock `routines/<tên>/<tên>.lock` | `routine run` | `routine run`, (sau sửa) `list/show` | kernel tự nhả khi chết | đây là "lease" duy nhất trong hệ |
| Log | `<data>/logs/crush.log` (client/local), `~/.cache/agent-cli/server-<host>/crush.log` | slog | `agent-cli logs` | lumberjack 10 MB, giữ 30 ngày | từng ghi nguyên env client (PRB-10, đã sửa) |
| Secret provider | global data `crush.json` (0600, plaintext), `~/.config/agent-cli/env` | người dùng, OAuth | config | tệp | không keychain |

## Hai mâu thuẫn quyền sở hữu cần giải quyết (đưa sang Phase 5)

1. **Model của session vs model của workspace.** Session ghi lại model đã chạy nhưng không sở hữu model sẽ chạy; resume "mượn" workspace để khôi phục rồi để lại hậu quả cho mọi session khác. Một chủ duy nhất phải được chọn.
2. **Lớp cấu hình người viết vs lớp máy ghi.** Fallback tự động ghi vào lớp đè lên lựa chọn của người dùng, nên một lần provider mất khoá biến thành đổi provider vĩnh viễn.
