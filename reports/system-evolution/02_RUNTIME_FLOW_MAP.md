# 02 — Bản đồ luồng runtime (Phase 1)

Mỗi luồng: bước → nơi xử lý (file:dòng ở HEAD `4a0b25b6`) → chỗ hỏng đã đo. Mã ID vấn đề (`PRB-xx`) xem [05_CURRENT_PROBLEMS.md](05_CURRENT_PROBLEMS.md). Nhãn: CHẠY = có lệnh/probe tái lập; ĐỌC = đọc mã trọn đường.

## F1. Khởi động + nạp cấu hình
1. `main.go` chụp env tin cậy (`execpolicy.Snapshot`) → `godotenv.Load()` (`.env` của cwd, không đè biến có sẵn) → cobra.
2. `config.Load` gộp lớp (01 §2); provider catalog (embedded catwalk + cache + fetch nếu auto-update bật).
3. `resolveSelectedModels`: model đã chọn mất/provider bị loại → **lấy mặc định của provider đầu tiên còn cấu hình, ghi vào global data `crush.json`**, không log, không UI (`load.go:159-180, 845-910`) → PRB-04 (ĐỌC, g4).
4. Slot small trống → small mặc định của **provider khác** (`load.go:847-851`) → PRB-03.

## F2. Prompt → lượt chạy → tool
1. Client gửi prompt (RunID với `run`/`send`) → backend → coordinator dựng lại model + tool (`coordinator.go:345-357`), chốt allowlist ở `buildAgentModels` (`:1023-1038`).
2. Session agent: tạo tin user, `preparePrompt` (luôn chèn "todo list is currently empty" — PRB-22), `fantasy.Stream`.
3. Tool call → `hooked_tool` (PreToolUse) → permission (thứ tự ở 01 §2) → thực thi; bash qua BlockFuncs.
4. StopWhen: còn ≤ 20% context (window ≤ 200k) → tự tóm tắt; lặp cùng tool call > 5/10 bước → dừng.
5. RunComplete phát đúng một lần; `run --json` in `session_id, text, prompt_tokens, completion_tokens, cost` (**không có provider/model**, P-02).

## F3. Tiêu đề session
Tin user đầu tiên (và tin đầu của **mỗi** session agent con) → model **small**, gửi kèm nguyên văn prompt (`agent.go:2235`); lỗi/cắt thì thử model large (`:2252-2283`). Probe P-01: large=qwen, small=glm → tiêu đề đi glm → PRB-03 (CHẠY).

## F4. Agent con
`agent` tool: session con mới, system prompt `task.md.tpl` (1.031 B: không context file, không skill), 8 tool chỉ-đọc, model large, chạy song song tối đa 5; chỉ chữ của bước cuối về cha (P-03, CHẠY). `agentic_fetch`: model small, session con được tự duyệt, tool cố định `web_fetch, web_search, glob, grep, sourcegraph, view` bỏ qua `disabled_tools` → PRB-09.

## F5. Tự tóm tắt (compaction)
Ngưỡng ở `agent.go:1292-1308` → `summarize()` bằng model large của lượt, đặt `PromptTokens=0` → nếu lượt còn tool call thì xếp lượt làm tiếp "The previous session was interrupted…". **Phần cố định ≥ 80% window → lặp vô hạn** (P-04: 755 main + 755 summary / 90 s) → PRB-01 (CHẠY). Bản tóm tắt bị cắt (finish=length) vẫn được nhận → PRB-12. Prompt xếp hàng chạy TRƯỚC lượt làm tiếp → PRB-13.

## F6. Resume / đổi session
Đọc tin từ điểm tóm tắt, `ReadSettledMessages` (settle tool call mồ côi khi session không bận). Khôi phục provider/model của tin assistant cuối **bằng cách ghi vào `.crush/crush.json` (scope workspace)** → mọi session mới sau đó đổi model (g1 CHẠY) → PRB-05. Model không còn: `slog.Debug`, chạy model hiện tại.

## F7. Sập / khôi phục
- Tool call không có kết quả → settle thành "Interrupted" khi mở (đã có test).
- Lời trả lời chỉ có chữ/rỗng, bản tóm tắt dở → **không bao giờ đóng** → PRB-14 (g1/g2, CHẠY qua overlay).
- Hàng đợi prompt, grant quyền, tracker skill, bảng job: mất (trong RAM); tiến trình job nền sống sót (Setsid) mà không ai theo dõi → PRB-16.

## F8. Vòng đời server/workspace
Teardown khi hết client và hết run; "kẹt" xét **cả workspace** → session chờ quyền làm run của session khác bị huỷ sau 30 phút (PRB-17); `App.Shutdown` giết **mọi** job nền và đóng mọi MCP của tiến trình (A-01, PRB-15).

## F9. Quyền giữa nhiều session
`Request` khoá `sync.Mutex` suốt lúc chờ người trả lời → phiên khác kẹt, không thấy hộp thoại, **không huỷ được** (PRB-06, đã tái lập bằng test).

## F10. Loop / Routine
`loop`: baseline verify → lượt agent (session mới) → verify (`shell.Run`) → sửa ≤ MaxRepairs; dừng GREEN/MAX_REPAIRS/STALLED/TIMEOUT/RUN_ERROR/CANCELLED/UNMEASURED; journal fsync trước mỗi bước.
`routine run`: khoá flock → worktree + nhánh `routine/<tên>-<stamp>` → bật không người trực → loop → `report.json`. Bị kill giữa chừng: khoá tự nhả, journal còn `attempt_begin`, **không có report, `list/show` hiện lượt GREEN cũ** → PRB-02 (CHẠY). Lịch in ra thiếu biến của launcher → chạy theo lịch luôn rc=2 (PRB-07, CHẠY).
