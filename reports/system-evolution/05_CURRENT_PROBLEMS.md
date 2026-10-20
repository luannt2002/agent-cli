# 05 — Vấn đề hiện tại (Phase 1, sổ tổng)

Mỗi dòng là một vấn đề đã gộp trùng từ 6 nhóm audit, 3 verifier và các probe tự chạy. Mức sau soát. BC: CHẠY (lệnh/test/probe), ĐỌC, SUY. "Đã sửa (wt)" = sửa trong worktree `scratchpad/evo/wt` (chưa vào cây chính, chưa commit), có test đỏ→xanh và đột biến bị bắt; danh sách test ở [36_IMPLEMENTATION_PLAN.md](36_IMPLEMENTATION_PLAN.md).

## A. Đúng/sai và vòng đời (ưu tiên 1–3 của Phase 26)

| ID | Vấn đề | Mức | BC | Trạng thái |
|---|---|---|---|---|
| PRB-01 | Tự tóm tắt lặp vô hạn khi phần cố định ≥ 80% context window (755+755 lời gọi / 90 s, P-04) | HIGH | CHẠY | Đã sửa (wt) |
| PRB-02 | Routine bị kill giữa chừng: không báo cáo, `list/show` hiện lượt GREEN cũ | HIGH | CHẠY | Đã sửa (wt) |
| PRB-06 | Hộp thoại quyền của một session chặn session khác, không thấy, không huỷ được | HIGH | CHẠY (test) | Đã sửa (wt) |
| PRB-07 | Lịch routine in ra thiếu biến của launcher → mỗi lần chạy theo lịch rc=2, không chạy routine | MEDIUM | CHẠY | Đã sửa (wt) |
| PRB-12 | Bản tóm tắt bị cắt (finish=length) được nhận làm điểm cắt lịch sử | MEDIUM | CHẠY | Đã sửa (wt) |
| PRB-13 | Sau tự tóm tắt, prompt xếp hàng chạy trước lượt làm tiếp; lỗi của nó bị báo là "tóm tắt hỏng" | MEDIUM | CHẠY | Đã sửa (wt) |
| PRB-14 | Sau sập, lời trả lời chỉ chữ/rỗng và bản tóm tắt dở không bao giờ được đóng | MEDIUM | CHẠY | Đã sửa (wt) |
| PRB-15 | Teardown một workspace giết job nền và đóng MCP của mọi workspace cùng server (A-01) | HIGH (có điều kiện) | CHẠY (test) | Đã sửa (wt): App đóng chỉ dừng job của gốc mình; App cuối mới dừng hết + đóng MCP. Còn: MCP đóng rồi không mở lại được trong cùng tiến trình (broker), hai App cùng gốc trong cửa sổ khởi tạo |
| PRB-16 | Sau server SIGKILL: tiến trình job mồ côi không theo dõi, prompt xếp hàng mất không dấu vết, TUI không dựng lại server | MEDIUM | ĐỌC | Mở |
| PRB-17 | "Kẹt" xét cả workspace: session chờ quyền làm run của session khác bị huỷ sau 30 phút | MEDIUM | ĐỌC | Mở |
| PRB-63 | `run`/routine/loop chạy local: không ai duyệt được quyền của agent con → treo tới `--run-timeout` (verifier g2) | MEDIUM | SUY | Mở, cần test |
| PRB-64 | Race lúc tóm tắt: prompt đến giữa chừng mở lượt thứ hai song song trên cùng session (verifier g2) | MEDIUM | SUY | Mở, cần test |
| PRB-65 | Tiến trình local dùng chung DB với server đang chạy; settle có thể đánh dấu tool đang chạy là Interrupted (verifier g2) | LOW | SUY | Mở |
| PRB-42 | Model không có `context_window` không bao giờ tự tóm tắt; lỗi tràn context không được xử lý | MEDIUM | ĐỌC | Mở |
| PRB-43 | Agent con có thể được gọi trước khi system prompt/tool của nó dựng xong | LOW | SUY | Mở |
| PRB-44 | Chi phí agent con mất khi lỗi; cộng song song bị ghi đè | LOW | ĐỌC | Mở |

## B. Cô lập provider và cấu hình (ưu tiên 4)

| ID | Vấn đề | Mức | BC | Trạng thái |
|---|---|---|---|---|
| PRB-03 | Model small khác provider: tiêu đề (gửi nguyên prompt đầu) và agentic_fetch đi provider khác; small trống → provider khác (P-01) | HIGH (riêng tư, chi phí) | CHẠY | Đã sửa (wt): small phải cùng provider với large |
| PRB-04 | Fallback chéo provider lúc khởi động/reload, ghi vào global data đè crushrc, không báo; có thể khoá người dùng. Đọc lại 16/10: provider TỰ ĐỊNH NGHĨA đang được chọn mà thiếu `base_url` đã báo lỗi to (`load.go` selectedModelSlot); đường còn lặng lẽ: provider CÓ SẴN (catwalk) thiếu API key, và ID model không còn trong provider | HIGH | ĐỌC | Đã sửa: không ghi xuống đĩa (Daily G-01), không bao giờ đổi provider; provider không dùng được thì dừng có lỗi (vòng 6, faults ca `selected-no-key`: trước rc=0 + 2 request sang glm, sau rc=1 + 0 request) |
| PRB-05 | Resume / đổi session / `run --model` (server) ghi model vào scope workspace → mọi session sau đổi model | MEDIUM | CHẠY | Mở — mâu thuẫn quyền sở hữu (03) |
| PRB-30 | Allowlist chỉ kiểm nhãn `provider:model`, không ràng buộc `base_url`/`extra_body` | MEDIUM | ĐỌC | Mở |
| PRB-45 | `run --json` không có provider/model; chế độ local chèn cả bản tóm tắt vào `text` | LOW | CHẠY | Đã sửa (wt): có `provider`/`model`; `text` không còn lẫn bản tóm tắt (vòng 6, `qwen-only.sh`) |
| PRB-46 | Mỗi agent con tốn một lời gọi tiêu đề | LOW | CHẠY | Đã sửa (wt) |
| PRB-74 | Tiêu đề bằng LLM: 26/97 lời gọi baseline (27%), 26/26 kết thúc `length`, không lần nào ra tiêu đề; s12: 8/21 (21) | MEDIUM (chi phí) | CHẠY | Đã sửa (wt): không thử lại sau `length`, chỉ thử large khi khác small, không gọi cho session con |
| PRB-51 | Chính sách đọc một lần mỗi tiến trình | LOW | ĐỌC | Mở |

## C. Thành công giả và bằng chứng (ưu tiên 5)

| ID | Vấn đề | Mức | BC | Trạng thái |
|---|---|---|---|---|
| PRB-24 | Loop/routine GREEN dù verify không đo gì (0 test, `| cat` che lỗi, không đổi gì) | MEDIUM | CHẠY | Đã sửa (wt): pipefail + cờ; chưa nhận diện "0 test" |
| PRB-66 | Agent sửa test/cổng để đạt GREEN; chỉ có một câu trong prompt (verifier g3) | MEDIUM | ĐỌC | Đã sửa (wt): báo cáo liệt kê tệp + cờ |
| PRB-27 | Loop detection dừng lượt mà rc=0 (C-06) | MEDIUM | CHẠY | Đã sửa (wt) |
| PRB-31 | Lệnh bash nền lỗi (fast-fail, `job_output` exit≠0) vẫn hiện ✓ | MEDIUM | CHẠY | Đã sửa (wt): icon lỗi + `exit=N`; chưa xem trong TUI thật |
| PRB-32 | Bộ đếm token chỉ giữ bước cuối; `run --json`/`stats` không phải lượng tiêu thụ | MEDIUM | CHẠY | Giảm nhẹ: sổ ghi LLM (P-06) |
| PRB-33 | Cờ ước lượng `~` mất ở chế độ server và sau khởi động lại; `$0.00` không phân biệt miễn phí/không biết | MEDIUM | CHẠY | Mở |
| PRB-37 | `run` rc=0 dù lệnh trong lượt thất bại; không có tín hiệu verify mặc định | MEDIUM | CHẠY | Mở — đường đúng là `loop --verify` |
| PRB-36 | Stop hook chạm trần: TUI không hiện lỗi cho session đang xem | MEDIUM | ĐỌC | Mở |
| PRB-38 | Hook fail-open (timeout, crash, mã lạ, JSON hỏng) | MEDIUM | CHẠY | Mở (có tài liệu) |
| PRB-22 | Reminder "todo list is currently empty" gửi mọi lượt kể cả khi có todo | LOW | CHẠY | Mở (đụng cassette VCR) |
| PRB-75 | Model trả lời rỗng (không chữ, không tool call, finish=stop) → `run` rc=0, `text` rỗng; cả bản gốc lẫn bản sửa (33, ca `empty-reply`) | LOW-MEDIUM | CHẠY | Đã sửa (wt-b); hồi quy của bản sửa đầu (chặn nhánh tự làm tiếp khi bị cắt, đổi lý do từ chối) đã sửa ở đợt 3 |

## D. Skill, tool, quyền (ưu tiên 6)

| ID | Vấn đề | Mức | BC | Trạng thái |
|---|---|---|---|---|
| PRB-08 | Quét skill đệ quy vào `.trash` và `synced`: 48 skill, 31.860 B mỗi request, 41 lượt thật chỉ 2 lượt nạp skill | HIGH (chi phí, nhiễu) | CHẠY | Một phần (wt): bỏ thư mục ẩn; `synced` chờ quyết định |
| PRB-09 | agentic_fetch bỏ qua `disabled_tools`; session con tự duyệt → đọc tệp ngoài + gửi ra web | HIGH | ĐỌC | Đã sửa (wt); phần bỏ tự duyệt mới ĐỌC |
| PRB-21 | Tracker skill theo workspace: "loaded_this_session" sai, mất khi khởi động lại | MEDIUM | CHẠY | Mở |
| PRB-40 | Thư mục skill của dự án cho `view` không hỏi quyền, không giới hạn; symlink `.agents/skills -> /` phủ mọi tệp | MEDIUM | ĐỌC | Mở |
| PRB-41 | PreToolUse hook không chạy trong agent con | MEDIUM | ĐỌC | Mở |
| PRB-39 | MCP CallTool không timeout (B-08) | MEDIUM | ĐỌC | Mở |
| PRB-55 | Ưu tiên skill trùng tên theo thứ tự đường dẫn; `code-en-doc-vi` bản ~/.claude (1.185 B) che bản ~/.agents (4.246 B) | LOW | CHẠY | Mở |
| PRB-54 | Skill builtin crush-config chỉ `~/.config/crush` trong khi launcher dùng `~/.config/agent-cli` | LOW | ĐỌC | Mở |
| PRB-59 | Sự kiện hook lạ (PostToolUse) được nhận im lặng | LOW | ĐỌC | Mở |
| PRB-60 | Whitelist MCP docker theo tên server | LOW | ĐỌC | Mở |
| PRB-18 | `job_kill`/`job_output` vượt session/workspace, không hỏi quyền, ID trùng sau khởi động lại | MEDIUM | CHẠY (test) | Đã sửa (wt): chỉ job của session đang gọi; job của session khác trả "not found". Không thêm hỏi quyền |
| PRB-25 | Không người trực: `git -C <repo khác>` lọt | MEDIUM | CHẠY | Đã sửa (wt) |
| PRB-67 | Không người trực: viết tắt option dài (`--har`, `--delet`) lọt (verifier g3) | MEDIUM | CHẠY (test) | Đã sửa (wt) |
| PRB-68 | Routine chạy mã cấu hình của repo (crushrc, `$(...)`, hook) ngoài lớp chặn (verifier g3) | MEDIUM | ĐỌC | Giới hạn đã biết: chỉ chạy routine trên repo tin cậy |

## E. Context và dữ liệu

| ID | Vấn đề | Mức | BC | Trạng thái |
|---|---|---|---|---|
| PRB-19 | Context file tên thường (agents.md, crush.md) không nạp trên Linux; dialog Initialize bị tắt | MEDIUM | CHẠY | Mở |
| PRB-20 | Context file chỉ đọc ở cwd (không ở gốc git); plan mode và agent con không có context dự án | MEDIUM | CHẠY | Mở — quyết định thiết kế |
| PRB-11 | Xoá session sót session con + transcript (có thể chứa secret) | MEDIUM | CHẠY | Mở (cần query sqlc) |
| PRB-10 | Log ghi nguyên env client (API key) khi lệch cờ (A-09) | MEDIUM | CHẠY (test) | Đã sửa (wt) |
| PRB-49 | Không retention cho DB, journal, routine run/worktree, projects.json; panic dump vào cwd | LOW | ĐỌC | Mở |
| PRB-23 | Worktree/nhánh routine tích luỹ (giữ để duyệt là chủ ý) | LOW-MEDIUM | CHẠY | Mở |
| PRB-34 | "Modified Files" bỏ sót thay đổi bằng bash/download/giao việc | MEDIUM | CHẠY | Mở |
| PRB-35 | Hộp thoại quyền đóng trước khi grant chắc chắn; IO không timeout trong `Update` (D-03) | MEDIUM | ĐỌC | Mở |

## F. An ninh mặc định

| ID | Vấn đề | Mức | BC | Trạng thái |
|---|---|---|---|---|
| PRB-28 | Cấu hình dự án chạy mã khi mở (crushrc như Bash, `$(...)`, MCP stdio) không qua tin cậy (B-02) | HIGH (repo lạ) | ĐỌC | Mở — thiết kế |
| PRB-29 | API server không xác thực; chế độ tcp mở; socket mặc định `/tmp` có thể bị chiếm (wrapper tránh được) | MEDIUM | ĐỌC | Mở |
| PRB-47 | `.env` của repo bật được pprof | LOW | ĐỌC | Mở |
| PRB-48 | Telemetry bật với binary trần | LOW | ĐỌC | Mở |
| PRB-26 | Fingerprint chạy filter clean/smudge của repo (không vượt quyền agent vốn có) | LOW | CHẠY | DEFER |
| PRB-61 | Chế độ không người trực không phải sandbox (python/node, redirect, `cd` trước `rm`) | — | ĐỌC | Giới hạn đã biết |
| PRB-62 | Test backend nạp skill thật trong `~/.claude/skills` của máy chạy test dù đã cô lập HOME | LOW | CHẠY | Mở (vệ sinh test) |
| PRB-69 | `crush.json` của repo (chỉ dữ liệu, không cần crushrc) nối thêm `allowed_tools: [bash, edit, write]` → mọi lệnh không hỏi (verifier g5) | MEDIUM | ĐỌC | Mở — cùng thiết kế tin cậy với PRB-28 |
| PRB-70 | Không người trực: edit/write/fetch không qua lớp chặn (chỉ bash) → ghi ra ngoài worktree, vd `~/.config/agent-cli/crushrc` chạy như Bash lần sau (verifier g5) | MEDIUM | CHẠY (test) | Đã sửa (wt): lớp quyền từ chối ghi/download ra ngoài thư mục làm việc khi không người trực, trước cả YOLO và `allowed_tools` |
| PRB-71 | `skills_paths` viết `~/…`/`$VAR` được quét nhưng `view` không nhận là skill (hỏi quyền, giới hạn cỡ, không theo dõi) | LOW | ĐỌC | Mở |
| PRB-72 | Probe "busy" của client không timeout; một probe treo làm trạng thái busy/model cũ mãi | LOW | ĐỌC | Mở |
| PRB-73 | Cộng chi phí agent con không khoá, `Save` ghi lại mọi cột → có thể đảo ngược todos khi chạy song song | LOW | ĐỌC | Mở |

## H. Phát hiện của lượt soát độc lập trên worktree (16/10, 3 người soát chỉ-đọc)

| ID | Vấn đề | Mức | BC | Trạng thái |
|---|---|---|---|---|
| PRB-76 | Không người trực: git bị đổi repo bằng biến môi trường (`GIT_DIR=… git rm`, `env GIT_DIR=…`, trong `sh -c`; `GIT_CONFIG_COUNT` như `-c`) (SEC-1; lượt 2 tìm thêm: export trong ngoặc, gán trước hàm, `env -S`, `find -exec`, `eval`) | MEDIUM | CHẠY (overlay của người soát) | Đã sửa (wt-b, đợt 3): luật theo chữ, chặn mọi lệnh nhắc tới biến git khi không người trực; 15 dạng end-to-end. Còn: tên ghép lúc chạy, script từ tệp/stdin (ghi trong comment) |
| PRB-77 | `lsp_replace_symbol`/`lsp_rename` ghi tệp mà yêu cầu quyền không có Action/đường dẫn → qua mặt kiểm tra ghi ra ngoài khi không người trực (SEC-2, F3) | MEDIUM | CHẠY | Đã sửa (wt-b) |
| PRB-79 | `disabled_tools: [web_fetch]` mà agentic_fetch chế độ URL vẫn tự tải URL (SEC-4) | LOW | CHẠY (test) | Đã sửa (wt-b) |
| PRB-80 | Lịch routine đọc biến sau khi `.env` của dự án đã nạp; allowlist provider không đi theo lịch (SEC-5) | LOW | CHẠY (test) | Đã sửa (wt-b); allowlist cách bằng dấu cách đổi thành phẩy (SEC2-5) |
| PRB-81 | Sổ ghi LLM chép tới 300 ký tự lỗi provider (server local hay chép mẩu prompt vào lỗi) (SEC-6) | LOW | CHẠY (test) | Đã sửa (wt-b): chỉ ghi loại lỗi + mã HTTP |
| PRB-82 | Routine lỗi trước khi ghi báo cáo (tạo worktree hỏng) hiện ABANDONED "chết trước khi ghi" thay vì ERROR kèm nguyên nhân (LC-1, F5) | MEDIUM | CHẠY | Đã sửa (wt-b) |
| PRB-83 | Lượt mới nhất chưa có báo cáo mà khoá đang giữ hiện ABANDONED (LC-2) | LOW | CHẠY | Đã sửa (wt-b); chỉ thử khoá khi cần (LC2-5) |
| PRB-84 | Chế độ server: App cuối đóng MCP (broker không mở lại được), khách quay lại trong thời gian chờ nhận MCP chết (LC-3) | LOW | ĐỌC | Giới hạn đã biết: có từ trước, bản sửa PRB-15 làm hẹp lại; sửa đúng = đóng MCP khi tiến trình server dừng |
| PRB-85 | `KillOwned` theo gốc: App cũ đang dừng có thể giết job của App mới cùng gốc trong cửa sổ teardown (LC-4) | LOW | ĐỌC | Giới hạn đã biết (cửa sổ hẹp; sửa đúng = chủ job theo App) |
| PRB-86 | `pipefail` làm verify kiểu `cmd \| grep -q X` / `\| head` FAIL exit 141 (SIGPIPE) dù kiểm đạt (F1) | LOW | CHẠY | Đã sửa (wt-b): giữ pipefail, thêm gợi ý khi exit 141 |
| PRB-87 | Gốc skill tên ẩn viết kèm "/" cuối bị bỏ cả cây (hồi quy của bản sửa PRB-08) (F2) | LOW | CHẠY (test) | Đã sửa (wt-b) |
| PRB-63 (cập nhật) | `run` local: yêu cầu quyền của session con (agent con, agentic_fetch sau khi bỏ tự duyệt) không ai trả lời → treo (F4) | MEDIUM | CHẠY (test) | Đã sửa (wt-b): như `run` chế độ server (duyệt session con của lượt); tra lỗi thì từ chối |
| PRB-88 | Tự tóm tắt không có trần cho một prompt: model làm lại việc sau mỗi lần tóm tắt → 1.338 lời gọi main / 120 s (qwen-only.sh, cả hai binary) | MEDIUM | CHẠY | Đã sửa (wt-b): tối đa 10 lần/prompt khi việc còn dở, rồi dừng có lỗi |
| — | Comment sai: permission.go (agentic_fetch còn tự duyệt, `requestMu`), agentic_fetch_tool.go ("phải hỏi người dùng"), comment PRB-70 ngụ ý bash cũng bị chặn (F4a, SEC-3) | LOW | ĐỌC | Đã sửa (wt-b) |
| — | Dây nối không có test: cổng đăng ký agentic_fetch, danh sách tool con, bỏ tự duyệt, `provider`/`model` của `run --json` (F6) | LOW | ĐỌC | Đã sửa (wt-b): 4 test, 4/4 đột biến |

## G. Nhỏ / dọn dẹp

PRB-50 crush_logs đọc log client ở chế độ server · PRB-52 client ngắn hạn đặt lại đồng hồ mồ côi · PRB-53 `session delete/rename` CLI bỏ qua guard bận · PRB-56 đổi tên làm nhảy `updated_at` · PRB-57 `--cwd` tương đối lồng thư mục (local) · PRB-58 mã chết (CreateTitleSession, ListNewFiles với cột không tồn tại, FormatInvocation, PublishStates).

## Tổng

88 ID (gồm PRB-76..88 từ lượt soát độc lập và probe). Đã sửa trong worktree (`scratchpad/evo/wt-b`): PRB-01, 02, 03, 06, 07, 09, 10, 12, 13, 14, 15, 18, 24, 25, 27, 31, 46, 63, 66, 67, 70, 74, 75, 76, 77, 79, 80, 81, 82, 83, 86, 87, 88 (33) + PRB-04, PRB-45 (vòng 6) + một phần PRB-08, PRB-32. Giới hạn đã biết mới: PRB-84, PRB-85. Toàn bộ chi tiết bằng chứng nằm ở báo cáo thô của từng nhóm (`scratchpad/evo/p1-results/`) và [evidence/runtime-probes.md](evidence/runtime-probes.md).
