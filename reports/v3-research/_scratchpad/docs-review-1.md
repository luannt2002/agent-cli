KET-LUAN: KHONG-DAT

Phạm vi: HEAD 642fe075 so với 7cba42ff. Đã đọc diff CHANGELOG/README/docs/CLAUDE.md/DECISIONS và đối chiếu từng mục với mã ở HEAD (execpolicy, backend, client, cmd/{root,run,send,loop,home,jobs,login,logout}, provider/exec, workspace, app, ui/model_restore, mcp-tools, llm_ledger, agent-cli-tmux.sh, config/load).

## Đã chứng minh

1. **SAI / MÂU THUẪN — README.md:71-75 về `--yolo`**: "áp cho cả workspace (kể cả TUI khác và `send` vào dự án đó). `--yolo` lúc mở phải khớp workspace đang mở, cả hai chiều: dự án mở bằng `--yolo` thì client chạy model sau đó mà không có `--yolo` (TUI, `run`, `send`, `loop`) bị từ chối". Bằng chứng: `--yolo` chỉ khai ở `internal/cmd/root.go:67` `rootCmd.Flags().BoolP("yolo",...)` (cờ cục bộ, không PersistentFlags); `grep -rn '"yolo"' internal/cmd/*.go` → chỉ root.go:67/418/581. Vậy `run`/`send`/`loop` không nhận `--yolo`. `backend.go:493,604` từ chối mọi client `!NoAgentRuns && SkipPermissionRequests != args.YOLO`. Hệ quả: dự án mở bằng `agent-cli --yolo` thì MỌI `run`/`send`/`loop` (và Home, agent-cli-tmux vì chúng mở `-c` không `--yolo`) bị từ chối tới khi server tắt. THIẾU cùng gốc: CHANGELOG "Client không còn lặng lẽ…" (CHANGELOG.md:195-206, câu "`--yolo` khác workspace đang mở cũng bị từ chối thay cho cảnh báo cũ") và docs/agent-cli-local-setup.md:184 không nêu hệ quả.
   [ĐIỀU PHỐI: đã tái hiện trên binary; mã đang được sửa: run/send/loop nhận `--yolo` — xem brief.]

2. **MÂU THUẪN (có từ trước) — docs/agent-cli-local-setup.md** :353-354 "Server bị dừng (`kill <pid>`, SIGTERM, SIGHUP) thì dừng từng workspace trước: run đang chạy bị huỷ và được ghi lại, không bỏ dở giữa chừng" vs :402-403 "Dừng server (SIGTERM, `kill <pid>`) trong lúc nó đang giữ run cho client đã rời thì run bị cắt giữa lượt và session không ghi được dấu kết thúc." `git log -S`: :402 từ 1757a90d (14/10), :353 từ 461b540f (15/10) và khớp CLAUDE.md P4 + CHANGELOG mới → :402 LỖI THỜI.

3. **MÂU THUẪN (có từ trước) — README.md:127** "`alt+p`, hoặc `agent-cli run -s <id> -m …`. Cả hai đổi model của cả dự án, không chỉ một lượt" vs **:131** "`run -m` (ở chế độ server mặc định; chế độ local chỉ đổi trong lượt đó)". Mã khớp :131 (`app.overrideModelsForNonInteractive` chỉ ghi đè tạm).

4. **THIẾU — `scripts/eval/session-model.sh`, `ledger.sh`** không có trong mục "Thêm" `scripts/eval/` (CHANGELOG.md:87-91); cả hai có trên nhánh.

## Mới nghi (chưa chạy binary)

5. **LỖI THỜI — docs:390-393** ("Server bản cũ chưa có huỷ theo RunID: `run -s/-C` chỉ thôi chờ…"): với socket unix, `restartIfStale` trả `errStaleServerBusy` cho mọi lệnh chạy model khi server khác bản (root.go:997-1003, 731-733), nên `run` không còn tới server bản cũ (chỉ còn khi `--host` tcp).

6. **Có thể quá rộng — README.md:85-88, docs:637-639** ("`!agent-cli run …` … chụp biến thường của `.env` … nên CLI con của nó nhận chúng"): ở chế độ server, `run` lồng nối vào server sẵn có; CLI con do server dựng (`provider/exec.go:38` `OriginalEnviron()` của server). Có lẽ chỉ đúng in-process hoặc khi tiến trình lồng tự dựng server mới.

7. **Hơi lệch — CHANGELOG:204-205** ("Biến thường … vẫn theo client đã dựng server, như trước"): sau 213411fb server dựng bằng env gốc rồi tự nạp `.env` theo cwd của nó; `loop --cwd <dir>` → `.env` của `<dir>`. Mục `.env` (CHANGELOG:226-227) đã nói vậy → "như trước" không còn chính xác.

8. **Help text cũ:** `internal/cmd/send.go:28` "returns as soon as the server accepts the prompt" (README/docs nay "đã nhận và đã ghi"); `internal/cmd/loop.go:53-54` "The verify command runs in the workspace with your environment" (biến tin cậy nay theo env gốc, `loop.go:273` ChildEnv).

9. **THIẾU nhẹ:** CLAUDE.md P4 (dòng 64 "Khởi động", 67 "CLI cho script") chưa nhắc từ chối client khi chính sách tin cậy/`--yolo` khác, server khác bản đang bận, `ledger_error`, ghi bền trước ack. docs "Giới hạn đã biết" (:939+) thiếu giới hạn MCP còn lại (chữ kèm ảnh chưa có trần; ảnh kèm `isError` lưu `is_error=false`) — chỉ có trong CHANGELOG. CHANGELOG:108 dẫn "ca T11, T13 của bộ phơi nhiễm V3", DECISIONS:362 dẫn "T17" — bộ này không có trong repo (chỉ có `durable_ack_test.go:293,328`).

## Đã soát, không phát hiện
CHANGELOG 16 chỗ (0 SAI); README/docs/CLAUDE P5/help ~20 chỗ; trùng lặp 4 cặp (2 phát hiện ở mục 2, 3). Không soát docs/reviews, docs/superpowers, docs/hooks, mục [0.2.0].
