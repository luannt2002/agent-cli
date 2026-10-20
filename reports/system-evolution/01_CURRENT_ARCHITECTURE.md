# 01 — Kiến trúc hiện tại (Phase 1)

Đo ở HEAD `4a0b25b6` (= `evo/baseline-20261016`). Nguồn: 6 nhóm audit chỉ-đọc, mỗi nhóm có một lượt soát đối kháng. Kết quả thô ở `scratchpad/evo/p1-results/*.json`; các probe tự chạy ở [evidence/runtime-probes.md](evidence/runtime-probes.md). Mã là sự thật; README/docs/comment chỉ là giả thuyết.

## 1. Tiến trình và đường vào

```
~/.local/bin/agent-cli (wrapper bash)
  │  đặt CRUSH_GLOBAL_CONFIG=~/.config/agent-cli, CRUSH_GLOBAL_DATA=~/.local/share/agent-cli,
  │  CRUSH_CACHE_DIR, tắt metrics/auto-update; CRUSH_CLIENT_SERVER=1
  │  socket = /run/user/$UID/agent-cli-<sha1(git toplevel)[:8]>.sock  (mỗi repo một server)
  ▼
binary (main.go) ─ execpolicy.Snapshot() ─ godotenv.Load() ─ cobra (internal/cmd/root.go)
  ├─ Home (không tham số, có TTY)                     internal/ui/home
  ├─ TUI/run/send/loop… chế độ server  → internal/client ⇄ HTTP/SSE trên unix socket ⇄ internal/server → internal/backend
  └─ chế độ local (CRUSH_CLIENT_SERVER=0, routine)  → internal/app chạy trong tiến trình
backend: một Workspace cho mỗi đường dẫn cwd (first-wins), claim refcount, detach 10s,
         linger 60s, trần mồ côi 30 phút (kẹt chờ người) / 6 giờ (đang chạy)
```

## 2. Thành phần (mỗi trách nhiệm một chủ)

| Thành phần | Gói | Trách nhiệm | Trạng thái chính |
|---|---|---|---|
| Cấu hình | `internal/config` | Gộp lớp: `/etc` < global `crush.json` < global crushrc (chạy như shell) < **global data `crush.json` (máy ghi: models, key, OAuth)** < dự án (gốc git → cwd; crush.json < .crush.json < crushrc < .crushrc) < `<DataDir>/crush.json` (scope workspace) | file + bộ nhớ, tự reload khi ghi |
| Chính sách thực thi | `internal/execpolicy` | Allowlist model `AGENT_CLI_ALLOWED_MODELS`, bật/tắt giao việc, chế độ không người trực; đọc từ env chụp đầu `main` rồi tệp `env` | đọc 1 lần mỗi tiến trình |
| Điều phối agent | `internal/agent/coordinator.go` | Dựng model large/small mỗi run (`buildAgentModels`, chốt allowlist duy nhất), dựng tool, system prompt, agent con | bộ nhớ, theo workspace |
| Session agent | `internal/agent/agent.go` | Vòng lặp một lượt (fantasy Stream), StopWhen = ngưỡng tự tóm tắt + loop detection, hàng đợi prompt, tiêu đề, tóm tắt | bộ nhớ + SQLite |
| Agent con | `agent_tool.go`, `agentic_fetch_tool.go` | `agent` (task, tool chỉ-đọc, model large), `agentic_fetch` (model small, tự duyệt) | session con `msgID$$toolCallID` |
| Tool | `internal/agent/tools` | 26 tool cho coder; bash qua `internal/shell` (mvdan/sh) với BlockFuncs | job nền trong bộ nhớ |
| Quyền | `internal/permission` | Thứ tự: YOLO → `allowed_tools` → hook allow/accept-edits → tự duyệt session → grant theo session → hộp thoại | bộ nhớ |
| Chặn ở trình thực thi | `internal/shell/protect.go`, `unattended.go` | Self-kill guard, chặn CLI agent khác, chế độ không người trực (KHÔNG phải sandbox) | tĩnh |
| Skill | `internal/skills`, `agent/prompt` | Quét builtin + 4 thư mục global + 4 thư mục dự án (cwd và gốc git), đệ quy; danh sách XML vào system prompt của coder; kích hoạt = model `view` SKILL.md | snapshot lúc dựng, tracker trong bộ nhớ theo workspace |
| Hook | `internal/hooks`, `stop_hook.go` | PreToolUse, Stop (opt-in, fail-open) | cấu hình |
| MCP | `internal/agent/tools/mcp` | Client stdio/http/sse; tool `mcp_<server>_<tool>`; không có MCP server | tiến trình con/kết nối |
| Lưu trữ | `internal/db` (SQLite WAL, sqlc, goose) | sessions, messages (đủ parts), files (đủ nội dung mọi phiên bản), read_files, mcp_* | `<dự án>/.crush/crush.db` |
| Loop / Routine | `internal/loop`, `internal/routine`, `internal/cmd/{loop,routine}.go` | Driver phía CLI: agent → verify tất định → sửa; routine = định nghĩa + worktree + khoá + báo cáo | journal JSONL, report.json |
| UI | `internal/ui` | TUI Bubble Tea v2; probe trạng thái qua `tea.Cmd` | bộ nhớ |

## 3. Ranh giới tin cậy (đo, không suy)

- **Socket server không xác thực**; an toàn nhờ quyền thư mục `/run/user/$UID` (0700) mà wrapper chọn. Binary chạy trần không wrapper dùng `$XDG_RUNTIME_DIR` rồi `/tmp/crush-<uid>.sock` (có thể bị chiếm). Có chế độ `tcp://` không xác thực, không TLS. (g4, ĐỌC)
- **Cấu hình dự án được tin hoàn toàn**: crushrc của repo chạy như Bash với toàn bộ env; `$(...)` trong giá trị JSON được chạy; MCP stdio của repo tự khởi động. (g4/g5, ĐỌC; B-02 cũ, vẫn mở)
- **Allowlist provider chỉ là nhãn**: kiểm `provider:model`, không ràng buộc `base_url`, `extra_body`. (g4)
- **Trên máy này**, `permissions.allowed_tools` hiệu lực = `[view, ls, glob, grep, rg, edit, write, bash]` (crushrc + `.crush.json` symlink, mảng nối nhau) → bash/edit/write **không bao giờ hỏi**; chỉ còn BlockFuncs. (g5, CHẠY; mã `permission.go:205-209`)

## 4. Thứ KHÔNG tồn tại (đã tìm kỹ)

Agent Map/Task Graph (chỉ danh sách agent con đang chạy, tối đa 3 dòng, trong RAM TUI) · reviewer/verifier tích hợp · checkpoint/undo (dù bảng `files` lưu đủ nội dung) · memory tool (chỉ có context file do người/agent tự sửa) · lease/heartbeat cho việc dài · retention/prune cho DB, journal, worktree.

## 5. Số đo quy mô

121.920 dòng Go production + 90.962 dòng test trong `internal/` · 85 gói (68 có test) · một request "hello" ở dự án rỗng với skill thật của máy: **78.972 B** (system prompt 52.685 B, trong đó khối skill 31.860 B; schema 26 tool 25.775 B) — P-05.
