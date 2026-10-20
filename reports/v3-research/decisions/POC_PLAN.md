# V3 — Kế hoạch POC (sau soát điểm mù, 19/10)

Nguồn: `cross-project/FIT_GAP.md` (viết trước đợt 2) + soát điểm mù (KHONG-DAT, WORKFLOW_PROGRESS 28) + phơi nhiễm đã
soát DAT (`experiments/00_EXPOSURE_TESTS.md`) + chạy lại độc lập (`~/.cache/v3ev/m1.out`, `m2.out`). Bảng này THAY phần xếp
hạng POC trong FIT_GAP; các dòng FIT_GAP §1 dòng 1, 9, H1 (KEEP_V2), điều kiện của A1/B1 đã bị số đợt 2 bác.

## Luật chung cho mọi POC

- Mỗi POC một worktree/nhánh riêng từ `v2-final`, có 6 tệp test phơi nhiễm; không commit khi chưa được phép; đầu ra là
  patch + sha256 ở `experiments/poc/` và báo cáo cùng tên.
- Cổng xong: test phơi nhiễm của nó đỏ → xanh; đối chứng vẫn xanh; suite cũ không đỏ thêm (so theo TÊN); POC đồng thời
  thêm `-race`; kiểm răng hai chiều (hoàn bản sửa → đỏ lại); race thì chạy e2e ≥2 lần. Rồi soát độc lập (không Edit).
- Mở rộng chủ cũ; nếu POC thêm cơ chế mới thì tiêu chí xong phải gồm "xoá cơ chế cũ" (nguy cơ ở P1-a và P2-c).

## Bảng xếp hạng

| Hạng | POC | Phơi nhiễm | Chủ cũ (đã kiểm, tệp:dòng) | Cần người dùng quyết? | Soát |
|---|---|---|---|---|---|
| P0-1 | Race `readyWg` | T20 | `internal/agent/coordinator.go:179` (errgroup chung), `Wait` `:313`, `Go` `:843,:852` trong `buildAgent`; kích hoạt mỗi lượt qua `:347/:1558` → `buildTools` `:1571` → `agent_tool.go:26-36` | Không | Đồng thời (Opus) — **ĐANG LÀM** |
| P0-2 | Chính sách của client sau không bị bỏ im lặng | C5 (+ phần T21b: `.env` client đầu thành env server) | `internal/backend/backend.go:446` `CreateWorkspace`, nhánh trùng `:480,:587` → `logFirstWinsMismatch` `:1382-1400`; tiền lệ fail-closed `ErrChannelOptInMismatch` `:476-479,:582-586`; `execpolicy.Current` `policy.go:181`; env server = env client dựng (`root.go:1047`) | **Có (Q1)** | An ninh |
| P0-3 | `agent-cli-tmux rm` thấy session BUSY ở thư mục con | T4 | `scripts/agent-cli-tmux.sh:106` (so bằng → so tiền tố) | Không | Nhẹ |
| P1-a | Ghi bền trước khi ack (`send`, hàng đợi) | T11, T13 | `internal/backend/agent.go:32-70` (`go runAgent` `:68`); `enqueueCall` `agent.go:556-581`; đường bỏ hàng đợi `:603-635`, `:673-680`, `:2659-2690`, `:2704`, `:1765-1803`; fold `:1096`; `persistCanceledTurn` `:719` | Mặc định đã đặt (D1) | Vòng đời + đồng thời |
| P1-b | Kết quả MCP: trần + spill + `isError` | T9, T19 | `internal/agent/tools/mcp-tools.go:135-157`; `mcp/tools.go:50-115` không đọc `IsError`; dùng lại `shell.TruncateForPersist` (`truncate.go:57`) như `persist_message.go:32` | Không | Nhẹ |
| P1-c | Model không bị ghi thành cấu hình workspace ngoài ý người dùng | T17 | `cmd/run.go:935,:950,:962,:1010,:1019,:1050`; TUI `model_restore.go:56,:59`; `ui.go:2505,:2817,:3158`; proto `proto.go:144-158` không có trường model; `coordinator.go:347` dựng model từ cấu hình mỗi lượt | **Có (Q4)** | Đồng thời + provider |
| P2-a | Văn bản prompt theo đúng tool set | T5, T6 | `agent.go:955,:961-973`; `preparePrompt` `:2049-2060`; `plan.md.tpl:10,23`; `coder.md.tpl:91,:306` | Không (cassette cần provider thật — Q3) | — |
| P2-b | `--cwd` tuyệt đối | T2 | `cmd/root.go:1238-1252`; `ensureServer` `:547` trước `ResolveCwd` `:556`; `config/load.go:624` | Không | — |
| P2-c | Một khoá data dir + workspace lồng nhau | T1, T3, T1M | `projects.go:120-147`; `load.go:615-624`, `projectBoundary` `:1419`; `fsext/lookup.go:205-247`; Home union `home.go:376-427` | Mặc định đã đặt (D3) | Đồng thời + dữ liệu |
| P3-a | Nhãn nguồn cho tóm tắt | T12 | `agent.go:2241-2252` | Không (hiệu ứng thật cần LLM thật — Q3) | Nhẹ |
| P3-b | Lỗi ghi sổ LLM thấy được | T15 | `llm_ledger.go:72-90` (`:88`) | Mặc định đã đặt (D2) | — |

Không làm POC: C9 (R14 đã có tài liệu là denylist; R13 chưa có đường hại), T23 (NOT_EXPOSED), H3 (viết T16 trước).

## Mặc định do điều phối đặt (gắn cờ; người dùng có thể đảo)

- **D1 (P1-a):** prompt đã ack mà còn chờ khi crash → hiện là "đã huỷ/Interrupted" qua `settle.go` sẵn có, KHÔNG tự chạy
  lại (tự chạy lại = hàng đợi bền, đã REJECT ở FIT_GAP §3).
- **D2 (P3-b):** sổ ghi LLM lỗi → cảnh báo thấy được ở mọi lượt bị ảnh hưởng (log WARN + trường trong `run --json`), không
  dừng lượt (sổ là opt-in, dùng để chứng minh; dừng lượt làm hỏng việc chính vì công cụ phụ).
- **D3 (P2-c):** repo có ≥2 `.crush` → không tự gộp; cảnh báo + dùng `.crush` ở gốc dự án; `repo/sub` thuộc workspace của
  repo gốc khi mở bằng launcher. Gộp dữ liệu là việc tay của người dùng (có lệnh hướng dẫn).
- **D4 (`.env`, T21b):** giữ tự nạp `.env` của cwd cho tiến trình client (tương thích upstream, README đã ghi); trong P0-2
  chặn việc `.env` của client đầu thành env chung của server cho client sau.

## Câu hỏi cho người dùng

- **Q1 (P0-2):** allowlist/YOLO/env thuộc server hay từng client; client sau đòi chặt hơn thì từ chối hay dựng server riêng.
- **Q2:** quyền commit các nhánh POC V3 (local, không push).
- **Q3 (V3-U1):** provider `main` local hay trả tiền; ngân sách cho baseline §48 và ghi lại cassette.
- **Q4 (P1-c):** model thuộc session hay workspace; TUI mở session có đổi model của workspace không; `run -m` có ghi vĩnh viễn.

## Trả lời của người dùng (19/10)

- **Q1 → Từ chối rõ ràng.** Client có allowlist/YOLO khác server đang chạy → dừng với lỗi nêu khác biệt + cách sửa (thoát
  client cũ để server dựng lại), theo mẫu `ErrChannelOptInMismatch`.
- **Q2 → Cho commit** từng nhánh POC V3, chỉ sau khi POC qua cổng test + soát độc lập DAT; local, không push, không đụng
  `main`/`v2-final`.
- **Q3 → `main`/qwen3.8 là local, miễn phí:** chạy baseline §48 (V2) và biến thể V3 cùng ngày. Thứ tự: sau các P0 (server
  còn panic thì baseline nhiều session là nhiễu). Ghi lại cassette `TestCoderAgent` cần `CRUSH_HYPER_API_KEY` (provider
  khác) — CHƯA hỏi; P2-a tránh đổi template nếu được, hoặc báo chặn.
- **Q4 → Model thuộc workspace, nhưng không tự ghi:** chỉ đổi khi người dùng chọn (alt+p/alt+m, `-m`); mở/tiếp session cũ
  (`run -s`, TUI) không ghi model của session vào cấu hình dự án.
- **D5 (P1-a, điều phối 19/10):** prompt huỷ/lỗi khi còn xếp hàng KHÔNG vào lịch sử gửi model (giữ ngữ nghĩa V2 với model); UI
  vẫn hiện dấu vết đã huỷ. Lượt bị huỷ giữa chừng có nội dung giữ nguyên hành vi V2.

## Bổ sung sau đợt POC đầu (19/10)

- **P2-d (mới, Trung bình):** helper bóc `*hookedTool` dùng ở `filterToolsForChannel` (`agent.go:257`), `:284`, `filterDisabledMCPTools` (`:2202`). Có hook PreToolUse thì (a) lượt channel thấy tool của channel khác, (b) MCP đã tắt bằng Toggle vẫn gọi được. Test channel + Toggle có hook.
- **Chặn bởi cassette (hỏi người dùng cuối đợt):** T6 todo reminder (`P2a-toolprompt-todo.patch` sẵn), template `coder.md.tpl:91` `find_references`, `:306` `web_fetch`, `plan.md.tpl:10,23` `question` — ghi lại cassette `TestCoderAgent` cần provider Charm Hyper (`CRUSH_HYPER_API_KEY`).
