# 04 — Ma trận thực tế tính năng (Phase 1)

Trạng thái ở HEAD `4a0b25b6` (= baseline), trước mọi sửa của đợt này. Cột "Sau sửa" chỉ ghi khi worktree sửa đã có test đỏ→xanh và đột biến bị bắt. Bằng chứng: CHẠY (lệnh/probe/test), ĐỌC (đọc mã trọn đường), SUY (chưa đọc hết, có cách kiểm). Verifier = lượt soát đối kháng riêng cho từng cặp nhóm. Kết quả: g1+g2 38/38 xác nhận; g3+g4 44 xác nhận, 3 sửa; g5+g6 30 xác nhận, 1 sửa (đều đã áp dưới đây).

## Session · Context · Lưu trữ

| Năng lực | Trạng thái | BC | Khoảng trống chính | Sau sửa |
|---|---|---|---|---|
| Session (tạo/liệt kê/đổi tên/xoá, session con) | PARTIAL | CHẠY | xoá cha bỏ sót session con + transcript; `run -C` local có thể chọn session con | — |
| Resume | PARTIAL | CHẠY | khôi phục model bằng cách GHI scope workspace; model mất → âm thầm dùng model khác; không lưu effort/mode/cwd/branch | — |
| Chat (lưu tin, hàng đợi, huỷ) | PARTIAL | CHẠY | hàng đợi chỉ trong RAM; prompt xếp hàng chạy trước lượt làm tiếp sau tóm tắt | — |
| Context (thành phần system prompt) | PARTIAL | CHẠY | không thứ tự ưu tiên giữa context file; tên thường (agents.md) không nạp trên Linux; chỉ đọc ở cwd; không giới hạn kích thước; plan/agent con không có context | — |
| Compaction (tự/tay tóm tắt) | **BROKEN** | CHẠY | **lặp vô hạn** khi phần cố định ≥ 80% window (P-04); bản tóm tắt bị cắt vẫn được nhận; không có compaction phản ứng khi window=0 | dừng có lỗi sau 1 lần không tiến triển; bản bị cắt = hỏng |
| History (phiên bản tệp) | PARTIAL | CHẠY | chỉ 5 tool ghi; không có undo dù lưu đủ nội dung | — |
| Persistence (schema) | IMPLEMENTED | CHẠY | không có model/effort/cwd/branch theo session; không retention | — |
| Khôi phục sau sập (mức session) | PARTIAL | CHẠY | lời trả lời chỉ chữ/rỗng và bản tóm tắt dở không bao giờ được đóng | đóng là "Interrupted" khi mở |

## Agent · Run · Vòng đời

| Năng lực | Trạng thái | BC | Khoảng trống chính | Sau sửa |
|---|---|---|---|---|
| Task (todos) | IMPLEMENTED (tự khai) | ĐỌC | "completed" không gắn bằng chứng; reminder luôn nói danh sách rỗng | — |
| Task Graph | MISSING | ĐỌC | không có phụ thuộc giữa việc | — |
| Agent (vòng lượt, điều kiện dừng) | PARTIAL | CHẠY | không trần bước/thời gian mỗi lượt; loop detection dừng mà rc=0 | loop detection = lỗi |
| Agent con (`agent`, `agentic_fetch`) | PARTIAL | CHẠY/ĐỌC | không context file/skill; agentic_fetch bỏ qua `disabled_tools` và tự duyệt session con; mỗi agent con tốn một lời gọi tiêu đề | agentic_fetch theo `disabled_tools`, không tự duyệt |
| Agent Map | MISSING (chỉ dòng "running" trong RAM) | CHẠY | — | — |
| Run (RunID, hàng đợi, huỷ, mã thoát) | IMPLEMENTED | CHẠY | JSON không có provider/model; token là bước cuối | — |
| Job nền | PARTIAL | ĐỌC | tool agent không kiểm owner; ID trùng sau khởi động lại; teardown giết job mọi workspace | — |
| Vòng đời server/workspace | PARTIAL | CHẠY | "kẹt" xét cả workspace; teardown đóng MCP/job toàn tiến trình | — |
| Khôi phục sau server SIGKILL | PARTIAL | ĐỌC | job mồ côi không theo dõi; prompt xếp hàng mất không dấu vết | — |
| Nhiều session một workspace | PARTIAL | CHẠY (test) | hộp thoại quyền của A chặn B, không huỷ được | B huỷ được; session tự duyệt không xếp hàng |
| Review tích hợp | MISSING | ĐỌC | — | — |

## Routine · Loop · Git · Verify · Evidence

| Năng lực | Trạng thái | BC | Khoảng trống chính | Sau sửa |
|---|---|---|---|---|
| Routine new/list/show/run | PARTIAL | CHẠY | lượt bị kill bị giấu sau lượt GREEN cũ | lượt bị kill hiện ABANDONED; `show` liệt kê mọi lượt |
| Routine schedule `--print` | PARTIAL (verifier sửa từ IMPLEMENTED) | CHẠY | lịch in ra thiếu biến của launcher → rc=2 mỗi lần chạy | mang `CRUSH_GLOBAL_CONFIG/DATA/CACHE_DIR…` |
| Loop engine | IMPLEMENTED | CHẠY (test) | GREEN có thể đến từ verify không đo gì (0 test, ống dẫn che lỗi, không đổi gì) | pipefail; cờ `BaselinePass`, `Unchanged` |
| Chế độ không người trực | PARTIAL (không phải sandbox) | CHẠY | `git -C <repo khác>` lọt; viết tắt option dài (`--har`) lọt | `-C/--git-dir/--work-tree` ra ngoài workspace bị chặn; tiền tố option nguy hiểm bị chặn |
| Thao tác git trong Go | IMPLEMENTED | CHẠY | fingerprint vẫn chạy filter clean/smudge của repo (không vượt quyền: DEFER) | — |
| Diff | IMPLEMENTED (diff của tool sửa) | ĐỌC | không có trình xem git diff | — |
| Worktree | PARTIAL | CHẠY | không retention/GC (giữ để duyệt là chủ ý; verifier hạ còn LOW-MEDIUM) | — |
| Checkpoint | MISSING | ĐỌC | — | — |
| Rollback/undo | MISSING | ĐỌC | — | — |
| Verification | PARTIAL | ĐỌC/CHẠY | chỉ ở loop/routine; Stop hook opt-in và fail-open; agent có thể sửa test/cổng để GREEN | báo cáo routine liệt kê tệp đổi, cờ `touched_tests`/`touched_gate` |
| Evidence | PARTIAL | CHẠY | gắn với dấu vân tay cây làm việc, không gắn revision; commit:false thì nhánh không có commit | — |
| Lịch sử lượt chạy | PARTIAL | CHẠY | chỉ lượt mới nhất có báo cáo đọc được | mọi lượt, kể cả ABANDONED |

## Provider · Cấu hình · Lưu trữ · An ninh

| Năng lực | Trạng thái | BC | Khoảng trống chính | Sau sửa |
|---|---|---|---|---|
| Provider/catalog | IMPLEMENTED | ĐỌC | binary trần tự cập nhật catalog; biến key nào được export cũng bật provider đó | — |
| Model (large/small theo vai trò) | IMPLEMENTED | CHẠY (P-01) | small khác provider vẫn được dùng cho tiêu đề/agentic_fetch | small khác provider → dùng model large |
| Khoá provider/model (allowlist) | PARTIAL | CHẠY (P-02) | opt-in; chỉ kiểm nhãn; chặn cả lượt khi small nằm ngoài danh sách | chạy được khi chỉ liệt kê model của provider chính |
| Chính sách fallback | **BROKEN** | ĐỌC | fallback chéo provider lúc khởi động, ghi vào lớp đè crushrc, không báo; có thể khoá người dùng ngoài | — (cần thiết kế) |
| Model hiển thị vs model chạy | PARTIAL | CHẠY | small không hiện; JSON `run` không có model | sổ ghi LLM cho bằng chứng từng lời gọi |
| Cấu hình (lớp, crushrc, env) | PARTIAL | ĐỌC | cấu hình dự án chạy mã không qua tin cậy | — |
| Lưu trữ | PARTIAL | ĐỌC | không retention; xoá session sót session con | — |
| Log | PARTIAL | CHẠY (test) | ghi nguyên env client khi lệch cờ | chỉ ghi tên biến lệch |

## Skill · Tool · Hook · MCP · Quyền · UI · Test


| Năng lực | Trạng thái | BC | Khoảng trống chính | Sau sửa |
|---|---|---|---|---|
| Phát hiện skill | PARTIAL | CHẠY (P-05) | quét đệ quy cả `.trash` (8 skill đã xoá) và `synced` (14); 48 skill = 31.860 B mỗi request | bỏ thư mục ẩn → 40 skill |
| Hợp lệ skill | IMPLEMENTED | CHẠY | UI không nói lý do lỗi | — |
| Chọn/kích hoạt skill | PARTIAL / PARTIAL (verifier sửa từ IMPLEMENTED) | CHẠY (log thật) | không ngân sách, không lọc liên quan; 41 lượt thật chỉ 2 lượt nạp skill; `skills_paths` dạng `~`/`$VAR` không được `view` nhận là skill | — |
| Quan sát skill đã dùng | PARTIAL | CHẠY | tracker theo workspace, không theo session | — |
| Skill cho agent con | MISSING | ĐỌC | — | — |
| Skill gọi từ palette | PARTIAL (nhánh chết) | ĐỌC | không skill nào `user-invocable` | — |
| Tool / `disabled_tools` | PARTIAL | ĐỌC | agentic_fetch bỏ qua; tên lạ bị bỏ qua im lặng | agentic_fetch theo `disabled_tools` |
| Hook | PARTIAL | CHẠY | chỉ PreToolUse/Stop; fail-open; không áp trong agent con | — |
| MCP | PARTIAL | ĐỌC | CallTool không timeout; không bước tin cậy | — |
| Quyền | PARTIAL | CHẠY | máy này: bash/edit/write không bao giờ hỏi (cấu hình người dùng) | hàng đợi huỷ được |
| UI: model / busy / icon tool / todos / Modified Files / context % / hộp thoại quyền / trạng thái nền / agent con / skill | PARTIAL phần lớn; busy, todos, trạng thái nền IMPLEMENTED | CHẠY | job nền lỗi vẫn ✓; token là bước cuối; `~` ước lượng mất ở chế độ server; hộp thoại đóng trước khi grant chắc chắn | — |
| Review UI / Evidence UI | MISSING / PARTIAL | ĐỌC | — | — |
| Bộ test Go | IMPLEMENTED (85 gói, 68 có test) | CHẠY | history, pubsub không có test | — |
| Cassette VCR | PARTIAL | CHẠY | một cặp model, 13 kịch bản thuận | — |
| Script E2E | PARTIAL | ĐỌC | không có ca compaction, đổi provider, server SIGKILL giữa tool | — |
| Harness LLM giả | PARTIAL → có `scripts/eval/fakellm.py` | CHẠY | 4 bản chép; trùng từ khoá với prompt tiêu đề | LLM giả dùng chung, nhận vai trò từ system prompt |
| Eval harness | MISSING | ĐỌC | không có bộ việc, runner, chấm điểm | (Phase 8) |
| "Cái gì chi phối phiên này" | PARTIAL | CHẠY | không chỗ nào liệt kê context file, tool, chính sách | sổ ghi LLM (P-06) |
