# 14 — Ma trận provider / model (Phase 4)

Provider thật của người dùng (đo bằng sổ ghi, 2026-10-16): `main / qwen3.8` cho cả lượt chính lẫn tiêu đề. Cấu hình còn khai báo `glm` (glm-5.3, glm-5.3-flash) và `anthropic` (tắt). Lời gọi LLM chỉ được tạo ở `coordinator.go:1086,1090` (`LanguageModel`), `fantasy.NewAgent` ở `agent.go:966,1768,2226`; `buildAgentModels` được gọi từ `coordinator.go:809,1514` và `agentic_fetch_tool.go:150` (g4, verifier xác nhận). Không có đường gọi LLM nào khác trong tiến trình.

## Vai trò → model

| Vai trò | Slot | Chốt allowlist | Fallback | Trước (baseline) | Sau sửa (worktree) | Bằng chứng |
|---|---|---|---|---|---|---|
| Lượt chính (coder, plan) | large | có (`buildAgentModels`) | không | đúng provider đã chọn | như cũ | P-01, P-06 |
| Agent con (`agent`) | large | có | không | đúng | như cũ | P-01 seq 4 |
| Tóm tắt (tự/tay) | large của lượt / workspace | có | không (tóm tắt hỏng → làm tiếp không tóm tắt) | đúng | như cũ | P-04 |
| Tiêu đề session | small, rồi large nếu lỗi/cắt | có | small → large (slog.Error) | **provider của small** (P-01: glm) | small khác provider → dùng large | P-01, P-07 |
| Tiêu đề session agent con | như trên | có | như trên | **provider của small**, một lời gọi mỗi agent con | như trên | P-01 seq 3 |
| agentic_fetch | small (làm cả large) | có | không | **provider của small** | như trên | đọc mã `agentic_fetch_tool.go:150` |
| Loop/routine | large (`--model`/`def.Model`) | có | không | đúng; `loop` qua server ghi model vào scope workspace | như cũ | g4 |
| Giao việc claude_code/codex | model trong allowlist của backend | kiểm từng lần gọi | không | tắt mặc định | như cũ | g4 |

## Đường fallback (g4, verifier xác nhận)

| Đường | Hành vi | Người dùng biết? | Trạng thái |
|---|---|---|---|
| Khởi động: model đã chọn mất / provider bị loại | lấy mặc định của provider đầu tiên còn cấu hình (Hyper, anthropic, openai…) **ghi vào global data `crush.json`**, đè crushrc | KHÔNG | Mở (PRB-04) |
| Reload cấu hình (logout, sửa crushrc) | như trên, trong bộ nhớ; lượt sau dùng provider mới | KHÔNG | Mở |
| Slot small trống | small mặc định của provider khác | KHÔNG | Đã chặn hậu quả (small khác provider → large) |
| Tiêu đề lỗi/cắt | small → large → tiêu đề suy ra từ prompt | log Error | chấp nhận (cùng provider sau sửa) |
| Resume, model của session không còn | bỏ qua (slog.Debug), dùng model hiện tại | KHÔNG | Mở (PRB-05) |
| Lỗi provider lúc chạy | không đổi provider; lỗi tới lượt và RunComplete | có | đúng |
| Allowlist từ chối | fail-closed, không thay model | có (`MODEL_NOT_ALLOWED`) | đúng |

## Hiển thị vs thật

Sidebar hiện model **đang chọn** của workspace; lượt đang chạy giữ model đã ghim (có ghi chú). Model small không hiện ở đâu. Footer từng tin hiện provider/model thật đã lưu, nhưng chỉ khi kết thúc `end_turn`. Bằng chứng đúng cho "model nào đã chạy" là cột provider/model trong DB và sổ ghi.

## Bất biến và kiểm chứng

- **Không gọi chéo provider âm thầm.** Sau sửa, mọi vai trò dùng provider của slot large (P-07: glm = 0 lời gọi khi chọn qwen). Còn hở: fallback lúc khởi động/reload (PRB-04) và allowlist chỉ kiểm nhãn (PRB-30).
- **Model hiển thị = model chạy:** chỉ đúng cho lượt chính sau khi nó xong; small/agentic_fetch/tiêu đề không hiển thị.
