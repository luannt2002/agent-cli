# 11 — Manifest của một phiên (Phase 3)

Câu hỏi của cổng Phase 3: **"Chính xác cái gì đã chi phối phiên này?"** Tệp này liệt kê từng mục của manifest, nơi lấy được nó HÔM NAY, và mức tin cậy. Nguyên tắc: cấu hình không phải bằng chứng; ưu tiên dữ liệu ghi lúc chạy.

## Công cụ đã dựng trong đợt này

1. **Sổ ghi lời gọi LLM** — `AGENT_CLI_LLM_LEDGER=<tệp>` (biến tin cậy, chụp đầu `main`; `.env` của repo không bật được). Mỗi lời gọi một dòng JSON: `time, duration_ms, role (main|subagent|title|summary|agentic_fetch), session_id, provider, model, stream, messages, system_bytes, system_sha, tools, tool_names, input_tokens, output_tokens, cache_read_tokens, finish, error`. Không chép nội dung prompt; tệp 0600. Mã: `internal/agent/llm_ledger.go` (worktree). Đã kiểm khớp 5/5 với server giả (P-06).
2. **LLM giả có ghi request** — `scripts/eval/fakellm.py`: lưu system prompt đầy đủ theo sha để phân tích thành phần (chỉ dùng khi phát triển/đo).
3. **Bộ đo kịch bản** — `scripts/eval/{fixtures.py,run.sh,score.py}` (Phase 8).

## Manifest: mục → nguồn → mức tin

| Mục | Lấy từ đâu (phiên thật) | Mức |
|---|---|---|
| session_id, project | `run --json`, sổ ghi (`session_id`), `session show` | runtime |
| Provider + model THẬT từng lời gọi, theo vai trò | sổ ghi (`role, provider, model`) | runtime |
| Model hiển thị | sidebar (model đang chọn, không phải model đang chạy) | cấu hình — không dùng làm bằng chứng |
| Token thật từng lời gọi | sổ ghi (`input/output/cache_read_tokens`); `run --json` chỉ có bước cuối (PRB-32) | runtime |
| Kích thước + hash system prompt | sổ ghi (`system_bytes`, `system_sha`) | runtime |
| Bộ tool đưa cho model | sổ ghi (`tools`, `tool_names`) | runtime |
| Skill được liệt kê | nội dung system prompt (chỉ thấy qua LLM giả); số đếm ở log `Skill turn summary active_total` | dev / log |
| Skill đã nạp | `session show` (Skills), metadata kết quả `view`, log `loaded_this_turn` | runtime (tracker theo workspace, PRB-21) |
| Skill thật sự được làm theo | không có | KHÔNG BIẾT |
| Context file đã nạp | chỉ thấy trong system prompt (LLM giả); không bề mặt nào liệt kê | dev |
| Hook | cấu hình (`hooks` trong crush.json/crushrc) | cấu hình |
| MCP | sidebar (trạng thái, số tool), cấu hình | cấu hình + trạng thái |
| Quyền | `permissions.allowed_tools` hiệu lực (mảng nối qua các lớp), YOLO, chế độ không người trực | cấu hình |
| Nguồn "memory" | context file + `~/.config/agent-cli/CRUSH.md`, `~/.config/AGENTS.md` | cấu hình |
| Kế thừa của agent con | cố định trong mã (P17, ở [16_CAPABILITY_OWNERSHIP.md](16_CAPABILITY_OWNERSHIP.md)) | mã |
| Chính sách provider/model | `AGENT_CLI_ALLOWED_MODELS` (env + tệp env), đọc một lần mỗi tiến trình | cấu hình |

## Kết luận cổng Phase 3

**ĐẠT MỘT PHẦN.** Với sổ ghi, một phiên thật trả lời được: gọi provider/model nào, vai trò gì, bao nhiêu token, system prompt to bao nhiêu (và hash để đối chiếu), tool nào được đưa. Chưa trả lời được cho phiên thật: **context file nào** và **skill nào nằm trong prompt** (chỉ thấy qua LLM giả lúc phát triển), skill có được **làm theo** không. Quyết định: không xây lệnh "manifest" đọc cấu hình riêng (cấu hình ≠ bằng chứng, thêm một nguồn sự thật thứ hai); nếu cần liệt kê context file cho phiên thật, thêm vào sổ ghi một trường `context_files` từ prompt builder — **DEFER** tới khi có nhu cầu đo cụ thể.
