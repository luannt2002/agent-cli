# Deep audit agent-cli — 16/10/2026

Audit chỉ đọc, HEAD `87233bc8`, cây git sạch khi bắt đầu. Không sửa mã production. Báo cáo này là tệp duy nhất được ghi.

**Cách làm.** 6 agent chỉ-đọc (loại không có Write/Edit) chia theo domain, chạy song song tối đa 3, mọi lệnh Go qua một cổng khoá + chờ RAM. Một agent phản biện độc lập thử bác bỏ các phát hiện của domain B và C. Người điều phối (tôi) tự tái lập hai phát hiện nặng nhất bằng `go test -overlay` (không ghi vào repo), tự đọc lại A-01/A-04/D-03, và dò mâu thuẫn giữa các báo cáo.

**Mức bằng chứng dùng trong bảng:**

| Ký hiệu | Nghĩa |
|---|---|
| **CHẠY** | Có lệnh tái lập đã chạy, kết quả nằm ở §Phụ lục |
| **ĐỌC×2** | Hai người độc lập đọc cùng mã, kết luận khớp |
| **ĐỌC** | Một người đọc mã, có file:line; chưa ai bác bỏ |
| **NGHI** | Suy luận, kèm cách kiểm |

Chưa phát hiện nào được chạy động trên binary thật (không build, không mở TUI, không chạy script E2E — theo luật phiên).

---

## 1. Kết luận

**Daily-use: READY WITH RISKS**, cho một chủ sở hữu duy nhất dùng hằng ngày. **Chưa đủ** để mở repo lạ hoặc để script tin `rc=0` là "việc xong".

**Điểm 6,5/10** là điểm *độ phủ năng lực P0 hằng ngày* theo phương pháp của agent F (10 mục, PASS=1, PARTIAL=0,5, MISSING=0, không trọng số). Đây không phải điểm an ninh: phát hiện CRITICAL B-01 không nằm trong con số này.

**Baseline test (CHẠY):** 83 gói, 66 ok, 17 không có test, 0 FAIL, rc=0, 1m51s. Đã đối chiếu từng gói trong `go list ./...` với dòng kết quả.

### Năm rủi ro lớn nhất

| # | Rủi ro | Mức | Bằng chứng |
|---|---|---|---|
| 1 | Bash tự duyệt đọc được tệp bất kỳ ngoài workspace (`git diff --no-index … ~/.config/agent-cli/env`) | CRITICAL | CHẠY |
| 2 | Không có cổng xác minh mặc định: lệnh bash lỗi vẫn hiện ✓ xanh, `run --json` rc=0, todo tự khai | HIGH (hệ thống) | ĐỌC×3 |
| 3 | Tắt một workspace giết bash job nền + MCP của mọi workspace cùng server | HIGH (có điều kiện) | ĐỌC×2 (A + người điều phối) |
| 4 | `.crushrc` của dự án chạy như shell không hỏi, có thể nới ranh giới provider | MEDIUM cá nhân / HIGH repo lạ | ĐỌC×2 |
| 5 | Resume / model biến mất → đổi model lặng lẽ, ghi đè cấu hình (invariant G hỏng) | MEDIUM | ĐỌC×3 |

### Thứ KHÔNG tồn tại (đã kiểm độc lập bằng 2 agent + người điều phối)

- **Agent Map** (`Alt+A`, `/agents`, modal): 0 kết quả trong 5 nhánh, toàn lịch sử (4344 commit), stash, và binary đang cài. Ảnh "Agent map" không phải từ build của repo này (xem F-01).
- **Review / Verify / Evidence / READY**: không có trong Go. Chỉ có Stop hook opt-in.
- **8 lệnh daily** `/status /changes /verify /history /checkpoint /undo /doctor /agents`: không lệnh nào có.
- **Git dirty/staged/diff/rollback/worktree trong Go**: không có (chỉ `scripts/agent-cli-tmux.sh`).

---

## 2. Kiến trúc thực tế (rút gọn)

```
main.go ─ execpolicy.Snapshot() ─ godotenv.Load() ─ cmd.Execute (cobra/fang)
  rootCmd.RunE ─┬─ Home (internal/cmd/home.go, internal/ui/home)
                └─ setupWorkspace ─┬─ local:  app.App in-process (AppWorkspace)
                                   └─ server: connectToServer → ensureServer → startDetachedServer
                                              (ClientWorkspace ⇄ HTTP/SSE ⇄ backend ⇄ app.App)
backend: CreateWorkspace (first-wins theo đường dẫn), claim refcount, detachGrace 10s,
         linger 60s, trần mồ côi 30 phút / 6 giờ
agent:   coordinator → session agent → tool → permission; sub-agent = "task session"
         (id messageID$$toolCallID, parent_session_id), tool đọc-chỉ
db:      SQLite + sqlc + goose; session KHÔNG lưu provider/model/cwd/branch
config:  system < ~/.config < crushrc < data < dự án (ngoài→trong) < <datadir>/crush.json
policy:  AGENT_CLI_ALLOWED_MODELS đọc RIÊNG bởi internal/execpolicy, KHÔNG qua config.Load
```

Hai cài đặt song song của `workspace.Workspace` (`app_workspace.go` 637 dòng, `client_workspace.go` 1577 dòng). Wrapper mặc định chạy chế độ server; binary chạy trần là local (đường legacy).

**Lệch phiên bản:** binary cài (`335709e4`) cũ hơn `main` 3 commit (`9446e056` sửa glob, `de515363` chữ sáng hơn, `87233bc8` doc). Mọi kết luận UI ở đây theo `87233bc8`.

---

## 3. Ma trận tính năng

| Tính năng | Trạng thái | Mức | Ghi chú |
|---|---|---|---|
| Chat / draft / history / huỷ | PASS | — | draft theo session, huỷ theo RunID |
| Server dùng chung, claim, teardown, trần mồ côi | PASS | — | test backend ok |
| Run sống khi TUI thoát; settle tool call bỏ dở | PASS | — | |
| Khoá model theo allowlist | PASS (khi đặt biến) | — | opt-in; mọi đường dựng model qua `coordinator.go:1036` |
| Tool giao việc `claude_code` / `codex` | PARTIAL | MEDIUM | tắt mặc định; CLI thật bản cuối chưa chạy |
| Bash auto-approve | **BROKEN một phần** | CRITICAL | B-01 |
| Self-kill guard | PARTIAL | MEDIUM | B-05 |
| Ranh giới workspace view/ls/glob/grep | PASS | — | giải symlink hai phía |
| Sub-agent kế thừa quyền; lỗi con → cha | PASS | — | tool đọc-chỉ; `NewTextErrorResponse` |
| Job nền cô lập theo workspace | BROKEN | HIGH | A-01 |
| Resume giữ model (invariant G) | BROKEN | MEDIUM | A-03 |
| Cấu hình sai thì dừng rõ ràng | PARTIAL | MEDIUM | A-02, A-11 |
| Context %, token, cost | PARTIAL | MEDIUM | A-07, D-07 |
| Test / lint / typecheck | MISSING | — | model tự gọi bash |
| Verification gate | PARTIAL, opt-in | MEDIUM | C-01, C-02 |
| Review / Evidence / READY | MISSING | — | |
| To-Do | PARTIAL | — | model tự khai, không gắn kết quả lệnh |
| "Modified Files" | PARTIAL | MEDIUM | F-02 |
| Git changes / diff trong TUI | MISSING | MEDIUM | F-03 |
| Agent Map | MISSING | — | F-01 |
| Checkpoint / undo / rollback | MISSING | P2 | `files` đủ nội dung để làm, git thay được |
| Worktree isolation (Go) | MISSING | P3 | |
| Footer / shortcut-help | PARTIAL | MEDIUM | D-01 |
| Theme mặc định + test tương phản | PASS | — | >100 cặp được test gate |
| Theme sáng | MISSING | LOW | chỉ 3 theme tối |
| Skills sidebar | PARTIAL | LOW | chỉ "khả dụng", không có "đã dùng" |
| MCP: lỗi init không hỏng session | PASS | — | |
| MCP: timeout lúc gọi tool | PARTIAL | MEDIUM | B-08 |

---

## 4. Phát hiện (đã hợp nhất, mức = sau phản biện)

### CRITICAL

**X-01 · Bash tự duyệt đọc tệp ngoài workspace** (B-01) — **CHẠY**
- `bash.go:233` chỉ gọi `permissions.Request` khi `isSafeCommand` trả `false`. Tái lập bằng test inject qua `-overlay`: `isSafeCommand` trả **true** cho `git diff --no-index /dev/null ~/.config/agent-cli/env`, `printenv`, `git config --list`. Ca đối chứng đúng chiều: `git status`=true, `rm -rf x`=false, `git diff --output=…`=false.
- Hệ quả: nội dung tệp (kể cả tệp env chứa API key) vào ngữ cảnh model → DB → provider, không có hộp thoại. Trái với chính `agent-cli.crushrc.example` ("chỉ hỏi khi đọc NGOÀI dự án").
- Nguồn gốc: tiền tố `git diff`/`printenv` có từ lúc import upstream; bản vá S1 `isReadOnlyGit` là của fork (`c330628e`, 14/10) và vẫn để lọt `--no-index`.
- Giới hạn: đã chứng minh ở hàm thuần + đọc cổng `bash.go:233`; chưa chạy tool bash end-to-end. Dưới `run`/YOLO mọi thứ vốn đã tự duyệt.
- Fix: chặn `--no-index` và đối số đường dẫn tuyệt đối / `~` / `..` trong lệnh an toàn; kiểm `working_dir` thuộc workspace; bỏ `printenv`. Test: bảng bad-case trong `bash_autoapprove_test.go`.

### HIGH

**X-02 · Không có cổng xác minh mặc định; thành công giả hiển thị** (C-07 + D-02 + F-07 + C-01 + C-02) — **ĐỌC×3**
- `bash.go:408-440`: exit≠0 chỉ thành chữ `Exit code N` trong output, `BashResponseMetadata` không có `ExitCode`, không phải `IsError`. UI chỉ nhìn `IsError` (`chat/tools.go:451-458`) → lệnh `go test` fail vẫn hiện ✓ xanh. `grep "Exit code"` trong `internal/ui` chỉ khớp lệnh `!` (`chat/shell.go:228`).
- `todos.go:49-52` chỉ kiểm tên trạng thái; "completed" không gắn với kết quả lệnh nào.
- Không có `pipefail`: `go test ./... | tail` trả exit của `tail`.
- `run --json` rc phản ánh *lượt chạy*, không phản ánh *công việc* (có tài liệu).
- Cổng duy nhất là Stop hook opt-in. Xem C-01/C-02 dưới đây.
- Mức: từng mảnh do agent C chấm MEDIUM; tôi giữ HIGH vì là lỗ hổng hệ thống ảnh hưởng mọi phiên không bật hook.
- Fix: thêm `ExitCode` vào metadata; UI đổi icon khi ≠0; `IsError` hoặc cờ cho exit≠0 tuỳ quyết định sản phẩm.

**X-03 · Tắt một workspace giết job nền + MCP của workspace khác** (A-01) — **ĐỌC×2** (A truy đủ đường; người điều phối đọc lại `app.go:944-947` và `background.go:350-368`; chưa phản biện bởi agent thứ ba, chưa chạy động)
- `App.Shutdown` gọi `KillAll` trên singleton toàn tiến trình; `KillAll` huỷ mọi shell và `Reset` bảng. `mcp.Close` đóng mọi session MCP toàn tiến trình.
- Điều kiện: ≥2 workspace (thư mục con khác nhau) cùng git root → chung server. Đóng TUI `fe` thì dev-server nền ở `be` bị giết.
- Fix: `KillAll` lọc theo `owner.Root`; đóng MCP theo workspace. Test: hai Workspace trong một Backend, job ở B, teardown(A), job B còn chạy.

### MEDIUM

| ID | Nội dung | Bằng chứng | Fix ngắn |
|---|---|---|---|
| B-02 | `.crushrc`/`crushrc` của dự án (và thư mục cha tới gốc git) chạy như shell với `os.Environ()`, không trust. Có thể ghi `~/.config/agent-cli/env` để nới `AGENT_CLI_ALLOWED_MODELS`/`DELEGATION` cho lần mở server sau → vòng qua ranh giới chính của fork. **Mức: MEDIUM cho chủ cá nhân, HIGH nếu mở repo tải về.** Không phải do fork thêm (mã Charm: `987ba71f`, `650bf3c2`); docs upstream ghi "trusted code" | ĐỌC×2 | Trust thư mục trước khi nạp crushrc/hook/MCP/base_url |
| A-02 = B-03 | Model đã ghim mất khỏi catalog → `load.go:858-866` chọn provider đầu tiên đang bật, **ghi vào config global** (`:166-179`), không thông báo. Allowlist vẫn chặn model ngoài danh sách. `slog.Warn` ở `:771,786` dùng printf trong slog nên log mất tên model | ĐỌC×3 | Lỗi nêu tên model, không persist |
| A-03 = B-06 = F-06 | Resume: model không khôi phục được thì `slog.Debug` rồi chạy model khác. `run -s` / `run --model X` persist vào `.crush/crush.json` (scope workspace), đổi model mọi phiên mới. JSON của `run` không có trường provider/model. Sidebar hiện model *đang chọn*, không phải model đã chạy. Có sẵn `OverridePreferredModel` (in-memory) nhưng `run` không nối | ĐỌC×3 | Cảnh báo rõ; thêm provider/model vào JSON; không persist khi chỉ xem |
| C-01 | Stop hook fail-open: timeout (mặc định 30s), crash, exit≠2/49, JSON hỏng, script thiếu, và sau 3 lần chạy lại vẫn bị chặn → lượt kết thúc như thành công, chỉ `slog.Warn`. **Có tài liệu và test khoá** (đã hạ từ HIGH). Mâu thuẫn thật: `verify-gate.sh` hứa "không chạy được = NOT READY" | ĐỌC×2 | Khi chạm trần/timeout: đặt `RunComplete.Error`, `run --json` rc=1 + `gate_failed` |
| C-02 | `verify-gate.sh` bảo đặt `AGENT_CLI_VERIFY_CMD` trong `~/.config/agent-cli/env` nhưng tệp đó chỉ được source trong interpreter riêng của crushrc → biến không tới hook → script thấy rỗng → `exit 0` → **cổng tắt lặng lẽ**. Tác giả đã ghi nhận ở `DECISIONS.md`. Chỉ `export` ở shell rc hoặc `.env` ở cwd mới tới hook | ĐỌC×2 | Sửa doc; script in cảnh báo stderr khi biến rỗng |
| C-06 | Loop detection (`agent.go:1314`) dừng lượt mà không đặt lỗi: `fantasy@v0.45.1/agent.go:1077-1081` trả `nil` → `RunComplete` không `Error` → rc=0. Đã xác nhận bằng đọc mã fantasy | ĐỌC×2 | Gắn finish reason `loop_detected` |
| C-05 | `shell.ProtectControlProcess` nối ở `bash.go:178`; xoá dòng này thì **không test nào đỏ** (đã tìm mọi tham chiếu). Không test chiều YOLO | ĐỌC×2 | Test `NewShell{BlockFuncs: blockFuncs()}` + `kill <pid test>` |
| B-05 | Self-kill guard lọt: `xargs kill -9`, `bash -c "kill <pid>"`, `sh -c "kill -9 <pid>"`, `kill -9 -1`. **CHẠY** (overlay): cả 4 trả `Blocked=false`; ca đối chứng `kill <pid>` bị chặn. Bash vẫn phải hỏi quyền trừ `run`/YOLO. Phụ: sau `cp -p` cài lại, `/proc/<pid>/exe` thành "(deleted)" nên guard server anh em sai (ĐỌC) | CHẠY | Bóc shell lồng; chặn `kill` không PID tường minh |
| B-08 | `mcp/tools.go:52` `CallTool(ctx,…)` không timeout; `timeout` config chỉ cho kết nối. TUI có Esc nên nhẹ; `run`/`send` treo đến Ctrl-C | ĐỌC×2 | Deadline cho CallTool |
| D-03 | IO HTTP đồng bộ trên goroutine `Update`: `AgentCancel` (phím Esc, `ui.go:5937`), `PermissionGrant` (`resolved, _ :=` nuốt lỗi), `SetCompactMode`… `http.Client{Timeout: 0}` (`client.go:60-63`) + `context.Background()`. Server treo → TUI đứng, kể cả Esc. Grant thất bại thì dialog vẫn đóng, agent kẹt chờ quyền. **Treo thật là NGHI** (cần workspace giả có Sleep) | ĐỌC×2 | Gói vào `tea.Cmd`, thêm timeout |
| A-04 | `isInsideWorktree()` (`load.go:1322`) chạy `git rev-parse` không đặt `cmd.Dir`; server thừa hưởng cwd của client lúc spawn → chạy Home từ `$HOME` thì mọi repo git trên server đó bị giới hạn `ls` depth 2 / 100 mục | ĐỌC×2 | `cmd.Dir = workingDir` |
| A-06 | Env của client không tới tool bash của agent (`proto.Workspace.Env` chỉ dùng cho lệnh `!`); bash dùng env của terminal đã spawn server đầu tiên → venv/nvm/PATH/AWS_PROFILE ở terminal sau không tới agent | ĐỌC | Merge env client |
| A-07 | `EstimatedUsage` không qua `proto` → ở chế độ client/server mặc định, model local không trả usage thì sidebar hiện `NN%` như số thật; cost ước lượng luôn 0; mất sau resume | ĐỌC | Thêm field vào proto |
| A-08 | `agent-cli session delete|rename` (`cmd/session.go:127`) mở DB thẳng, qua mặt guard `ErrSessionBusy` của server | ĐỌC | Đi qua server |
| A-09 | `logFirstWinsMismatch` (`backend.go:1380-1410`) ghi nguyên `os.Environ()` của client vào `crush.log` khi YOLO khác nhau (tệp 0600) | ĐỌC | Chỉ log tên biến lệch |
| F-02 | "Modified Files" lấy từ bảng `files` của session; chỉ 5 tool ghi vào (edit, write, multiedit, lsp_rename, lsp_replace_symbol). `sed -i`, `go fmt`, `git checkout`, codegen, `npm install` không xuất hiện; không phải git; session mới luôn "None" | ĐỌC | Đổi nhãn ("Edited by agent") hoặc thêm nguồn git |
| F-03 | Không có chỗ xem git status/diff trong TUI; `git status` chỉ chụp một lần vào system prompt | ĐỌC | Lệnh palette chạy `git status --short` + `git diff --stat` |
| D-01 | Footer cắt từ cuối: `ctrl+g more` (đường duy nhất tới toàn bộ phím) chỉ hiện từ ~162 cột; ở 80/100/120/140/160 cột mục cuối còn thấy lần lượt là `alt+w`, `alt+s`, `alt+m`, `ctrl+j`, `ctrl+c`. Dấu "…" dùng màu border (1,45:1) nên không thấy là bị cắt. Home: `esc quit` mất khi <94 cột | CHẠY (python) | Đặt Help/Quit đầu danh sách; Ellipsis dùng `fgSubtle` |

### LOW (gọn)

| ID | Nội dung |
|---|---|
| B-07 | `job_kill` không kiểm owner; ID `%03X` đoán được. Hạ từ MEDIUM: chỉ giết job do agent khởi động, chủ duy nhất |
| B-04 | Tệp env ghi được bằng tool agent dưới YOLO. Hạ: thường vẫn hỏi quyền vì ngoài working dir; hậu quả chỉ khi server mở lại. Phát hiện thêm: `protectedNames` có `.crushrc` nhưng không có `env` hay `.env` |
| B-09 | `echo claude \| sh` lọt chặn CLI agent theo tên |
| B-10 | filetracker độ phân giải 1 giây, không khoá; `relpath` dùng `os.Getwd()` của tiến trình (A-05) |
| B-11 | `permissions deny sourcegraph` không lan vào tool cố định của `agentic_fetch` |
| B-12 | Allowlist chỉ khoá khi có ≥1 provider chính; nạp một lần mỗi tiến trình; `:exacto` openrouter thêm sau khi kiểm |
| B-13 | NGHI: model nội bộ của CLI con (`claude_code`) không bị allowlist ràng buộc; mức quyền mặc định "toàn quyền", chưa chạy thật |
| C-08 | LSP diagnostics rỗng không phân biệt được với "sạch" (timeout/không có server) |
| C-09 | Stop hook không phân biệt lượt code với lượt hỏi đáp |
| C-10 | `TestBackgroundShellManager_Start` skip vô điều kiện ("flaky"); test backend skip dưới `-race` (race có sẵn). NGHI: `execpolicy` `depth > 4` fail-open, không test lồng sâu hơn |
| C-11 | Mã chết: nhánh `exitCode==0 && execErr!=nil` (`bash.go:282,368`), `hooked_tool.go:57` |
| A-10 | `CreateWorkspace` lỗi ở `app.New` không giải phóng conn + MCP/herdr đã khởi động (đường thường gặp: `MODEL_NOT_ALLOWED`) |
| A-11 | Schema `request_timeout` lệch code (default 60 vs 2 phút; số âm = tắt); `permissions deny <tên sai>` bỏ qua lặng |
| A-13 | Mã chết: `createDotCrushDir`, `HasInitialDataConfig`, `event.SessionSwitched`… (không chạy `unused`, danh sách chưa đầy đủ) |
| A-14/F-09 | `GetLastSession` không lọc parent, nhưng chỉ dùng ở `run --continue` (`app.go:274`); TUI không dính. NGHI |
| A-15/A-16 | `-c` đường dẫn tương đối ở chế độ server; tiêu đề phiên lấy từ stdin |
| D-04 | NGHI mạnh: output tool đi qua `StripCursorControl` giữ lại DCS/APC/OSC; ultraviolet phát lại DCS/APC cho terminal thật. Jobs dialog sạch, đường chat thì không |
| D-05 | `ansi[0]` = 1,21:1 (SGR 30), `ansi[8]` = 3,08:1 (SGR 90 — cargo/npm/git dùng cho chữ phụ). Test tương phản không gate hai ô này |
| D-06 | To-Do lớn: chiều cao không trần, completed-first nên phần bị cắt là in_progress/pending (NGHI về kết quả trực quan) |
| D-07 | Mẫu số context % lấy từ cấu hình (mặc định 128000), không hỏi server; "0%" ở sidebar vs ẩn ở header khi window=0; `$0.00` không phân biệt miễn phí/không biết giá |
| D-08 | `elements.go:91` `strings.Contains(firstLine, "via")` — tên model chứa "via" thì mất dòng provider |
| D-09/D-10/D-11/D-12/D-13 | Fork theme mất toàn bộ override khi sửa 1 màu; `synString == success == diffAdd`; tiêu đề LLM không strip ESC/`\r`; độ rộng grapheme lipgloss vs `WcWidth` (NGHI); Skills không có "đã dùng" |
| D-15/D-16 | OSC 11 bị bỏ qua trên terminal nền sáng → chữ sáng không đọc được (NGHI); sidebar dựng lại mỗi frame (NGHI hiệu năng, chưa đo) |
| F-04/F-05/F-08/F-10 | Trạng thái sub-agent chỉ trong RAM TUI; không có truy vấn con theo parent; todo pill biến mất khi xong; `files` giữ đủ nội dung để undo nhưng không có lệnh |
| Khác | Telemetry PostHog bật mặc định nếu chạy binary trần; `go b.runAgent` không có `recover` (NGHI) |

### Đã thử bác bỏ và không bác bỏ được / bị bác bỏ

- "Provider lock có đường vòng": soi 14 đường gọi LLM/CLI ngoài; mọi đường dựng model đi qua `coordinator.go:1036`, `store.go:515` chặn ghi đĩa. **Không tìm được đường vòng trong mã Go.** Hai giới hạn: không test nào chạy cả coordinator dưới policy hạn chế (`execpolicy.Current()` luôn mở trong test); `.crushrc` repo lạ (B-02) vòng qua ở tầng *nạp policy*, không phải tầng dựng model.
- "Tool error bị ẩn khỏi `run --json`": **bị bác bỏ** — exit≠0 hiện trong text, lỗi provider/run cho rc=1.
- "Stop hook chặn vô hạn": **bị bác bỏ** — trần 3 lần.
- "Stop hook bỏ qua cổng với prompt xếp hàng": **bị bác bỏ** (2 test ok).
- "Quét lỗi nuốt": `_ =` đếm được: agent 11, cmd 18, app 4, shell 3; đã đọc phần lớn — progress bar, `Close`, dọn dẹp best-effort. Không thấy nuốt lỗi trên đường ghi file/commit-session/cấp quyền. `message.go:297` flush lỗi nhưng `flushOne` khôi phục cờ dirty.

---

## 5. Invariant

| | Invariant | Kết luận |
|---|---|---|
| A | Verify bắt buộc đỏ thì không thể READY | **Không áp dụng** — không có khái niệm READY. Bảo vệ thực tế = Stop hook, chỉ giữ khi được cấu hình đúng (C-02), kịp timeout, sửa xong trong 3 lần (C-01) |
| B | Model không được phép không thể chạy | **ĐẠT khi có allowlist** (opt-in); giới hạn: B-12, B-02 |
| C | Agent con lỗi → cha không báo thành công | **ĐẠT** (`runSubAgent` trả `NewTextErrorResponse`; test pass) |
| D | UI phản ánh trạng thái runtime thật | **KHÔNG ĐẠT toàn phần** — model/busy/permission tươi (probe có TTL); sai ở D-02 (✓ xanh cho lệnh lỗi), D-03, D-07 |
| E | Task completed có bằng chứng hợp lệ | **MISSING** — `todos.go` nhận trạng thái tự do |
| F | Bằng chứng ứng với revision mới nhất | **Không áp dụng** — không có bằng chứng; ví dụ gate chạy lại verify mỗi lần nên không cache cũ |
| G | Resume không đổi provider/model lặng lẽ | **KHÔNG ĐẠT** (A-03/B-06). Điểm cộng: run đang chạy giữ model (`pinnedModel`, test ok) |
| H | Agent song song không dùng chung workspace ghi được | **ĐẠT theo thiết kế** — sub-agent chỉ có tool đọc, session riêng; giữa session xem B-10 |
| I | Màu ngữ nghĩa nhất quán | **ĐẠT một phần** — có test chặn; vi phạm nhỏ D-10 |
| J | Dọn dẹp lỗi không phá việc người dùng | **PARTIAL** — `agent-cli-tmux.sh rm` an toàn (từ chối khi dirty/BUSY; đọc tĩnh); app: A-01 |

---

## 6. Theme

**Palette thực** (`internal/ui/styles/everforest_dark_v2.go:51-110`; tên file giữ "everforest" nhưng là deep-charcoal):

| Role | Hex | | Role | Hex |
|---|---|---|---|---|
| bg.base | `#141619` | | accent | `#93CFD1` |
| bg.sunken | `#0F1113` | | success = diffAdd = synString | `#9CC093` |
| bg.raised | `#1B1E22` | | warning = synNumber | `#D4B87C` |
| bg.overlay | `#23272D` | | error = diffDelete | `#DD9090` |
| fg.primary | `#E6E9E7` | | info | `#9BB6D0` |
| fg.normal | `#C8CCCA` | | keyword | `#A392B2` |
| fg.muted | `#B2B8B6` | | border | `#30343B` |
| fg.subtle | `#99A09E` | | | |

Bão hoà HSV: accent 0,30 · success 0,23 · warning 0,42 · error 0,35 · info 0,25 · keyword 0,18. Không `#FFFFFF`/`#000000`, không neon, không gradient trong theme này (logo/header một màu).

**Tương phản** (python3, công thức sRGB; ngưỡng chữ ≥4,5, UI ≥3):

| FG | BG | Tỉ lệ | |
|---|---|---|---|
| primary | base / sunken / raised / overlay | 14,82 / 15,47 / 13,68 / 12,27 | PASS |
| muted | base / sunken / raised / overlay | 9,00 / 9,39 / 8,30 / 7,45 | PASS |
| subtle | base / sunken / raised / overlay | 6,80 / 7,10 / 6,27 / 5,63 | PASS |
| accent / success / warning / error / info | base | 10,43 / 8,95 / 9,45 / 7,31 / 8,63 | PASS |
| keyword | base / overlay | 6,31 / 5,22 | PASS |
| diff+ trên nền ins / diff− trên nền del | | 6,74 / 6,23 | PASS |
| keyword trên nền ins | | 4,75 | PASS (sát) |
| fg.primary trên nút inactive `#5E6665` | | 4,82 | PASS (sát) |
| border | base | 1,45 | FAIL — chủ ý trang trí, **nhưng** dùng cho dấu "…" của footer (D-01) |
| `ansi[0]` `#23272D` (SGR 30) | base | 1,21 | FAIL (D-05) |
| `ansi[8]` `#5E6665` (SGR 90) | base / sunken | 3,08 / 3,21 | chỉ đạt mức UI |

Quan điểm "đủ chứ không tối đa": chữ chính 14,8:1 là cao nhưng nằm trên nền `#141619` (không đen tuyền), fg.primary `#E6E9E7` không trắng tinh — không có dấu hiệu chói theo số đo. **Chưa đo** độ chói thị giác thực tế.

**Màu hard-code ngoài theme** (`grep` có đếm): `themes.go` 54 hex, `everforest_dark_v2.go` 40, `diffview/style.go` 12 (fallback; sản xuất luôn ghi đè bằng `.Style(s.Diff)`, phần light chỉ dùng trong test), `anim.go` đỏ→xanh mặc định (không dùng trong repo). Không còn màu hard-code rải trong `chat/`, `dialog/`, `model/`, `home/`. Màu chết: `diffChange`, `Header.Charm`, `Logo.CharmColor`, `VersionColor`.

**Mật độ** (trong `everforestDarkV2Overrides`, đếm token): fgMuted 45, fgSubtle 32, fgBase 21, accent 17, border 11, info 4, warning 2, success 0, error 0 → trung tính ≈85%, accent ≈11%, trạng thái ≈2%. **Chỉ là đo trong lớp override**; mật độ trên màn hình thực chưa đo.

### Lo ngại "inline-code, path, command bị tô đỏ/cam" — đã truy

- `quickStyle` gốc (`quickstyle.go:331-345`) tô inline code bằng `o.destructive` = **đỏ** (`#DD9090`).
- Theme gruvbox (`themes.go:247`) tô inline code **cam** `#fe8019`.
- Theme mặc định (`restyleMarkdown`, `everforest_dark_v2.go:427`) đặt `Markdown.Code.Color = fgBase` (**trung tính**). Test `inline_code_khong_cam_khong_do` (`everforest_dark_v2_test.go:372`) khoá điều này. Test được thêm ở `461b540f` (15/10), là tổ tiên của commit binary đang cài `335709e4`.
- Kết luận: ở theme mặc định, cả HEAD lẫn binary cài **đều không tô đỏ/cam inline code**. Nếu màn hình bạn thấy đỏ/cam thì (1) bạn đang chọn theme khác (gruvbox/theme riêng), hoặc (2) ảnh không từ build này. **Chưa đo:** theme bạn đang dùng thật — tôi không đọc cấu hình của bạn. Cách kiểm: `agent-cli` → Themes, hoặc xem khoá theme trong `~/.config/agent-cli/crush.json`.
- Phần "path/command/sidebar nhiều màu / logo gradient" chưa có agent nào đo bằng ảnh; D xác nhận logo một màu và override "sidebar yên tĩnh" tồn tại trong theme mặc định.
- Palette và layout 75/25 trong các đề xuất bạn dán (`#14181B`, `#E0E4E1`, `#7FB3A8`…) **chưa được đo tương phản**; layout hiện tại: sidebar chỉ hiện khi ≥120 cột (`Toggle Sidebar`), tỉ lệ thực chưa đo.

---

## 7. Chất lượng test

| Khu | Đánh giá |
|---|---|
| Stop hook (logic) | Mạnh: 13 test với script shell thật |
| Stop hook nối vào `app.go:848` | **Không test** — thay `NewCoordinatorWithStopHooks` bằng `NewCoordinator` thì mọi test xanh (C-04, suy luận, chưa chạy đột biến) |
| `ProtectControlProcess` nối vào bash | **Không test** (C-05) |
| Bash auto-approve | Mạnh ở hành vi, nhưng không có corpus âm cho `--no-index`, `printenv` (X-01 sống sót vì thế) |
| Coordinator dưới policy hạn chế | **Không test** (`Current()` mở trong test) |
| `load.go` fallback model | Test mã hoá hành vi hiện tại; B và A trích hai test khác nhau cho B-03 — phản biện sửa: dòng 1940-1941 kiểm hàm thuần "disk unchanged", persist nằm ở `Load` (`:166-179`) |
| VCR `TestCoderAgent` | Cassette thật phát lại, chỉ một cặp model (deepseek-v4), không có lệnh lỗi |
| `run --json` E2E | Phủ lỗi provider, SIGTERM, kill -9, tool-call thành công; **không có ca tool exit≠0** |
| Skip | 50 `t.Skip*` (28 tệp); đa số guard hệ điều hành/thiếu binary. 2 đáng chú ý ở C-10. 26 TODO/FIXME phi-test, không nằm trên đường C |
| Tổng | 416 tệp test, 2539 hàm `Test*` (đếm bằng grep) |

Không chạy đột biến thật (luật phiên: không sửa tệp). Mọi nhận xét "thay X thì test vẫn xanh" là suy luận từ đọc test.

---

## 8. Session, sub-agent, và nền cho Agent Map

Cây agent của một session **dẫn xuất được** từ dữ liệu đã lưu, không đổi schema (F, ĐỌC): duyệt tool call `agent`/`agentic_fetch` trong tin nhắn cha → ghép id con `messageID$$toolCallID` → `GetSession` + `ListMessages` (UI đã làm đúng thế ở `ui.go:2007`).

| Trường | Phân loại |
|---|---|
| id, tên, created/updated | PERSISTED (tên task session luôn "New Agent Session") |
| status cha | REAL RUNTIME (`AgentSessionStatus`) |
| status con | DERIVED trong RAM của TUI (`childRuns`) — chỉ có "running", mất khi đóng TUI |
| failed / cancelled | suy ra được từ `Finish.Reason`, nhưng không chỗ nào lưu/hiển thị |
| queued (sub-agent) | không tồn tại |
| elapsed | xấp xỉ `updated_at − created_at` |
| provider / model | PERSISTED theo từng assistant message, **không theo session**; sidebar hiện model đang chọn |
| token | cộng dồn ở session, không theo agent; cost con cộng vào cha, token thì không |
| cwd / branch / worktree | không lưu theo session; branch poll 5s |

- Sub-agent song song là thật (`NewParallelAgentTool`, semaphore 5 của fantasy).
- `ListSessions` lọc `parent_session_id IS NULL` nên task session không rò vào sidebar.
- `Alt+A` không bị `keys.go` hay textarea (bubbles v2.2.1) chiếm; phím global xử lý trước textarea. Việc terminal có gửi Alt+A hay không: **NGHI**, chưa thử.
- Ảnh "Agent map" của bạn ghi root "Sonnet 5.5" trong khi session "qwen3.8 via main". Dữ liệu agent-cli không thể sinh ra hai model khác nhau như thế (sub-agent dùng cùng `SelectedModelTypeLarge` với cha). Nhiều khả năng ảnh đến từ một chương trình khác. Cần bạn xác nhận nguồn ảnh.
- Khuyến nghị của F (tôi đồng ý, có bằng chứng từ khoảng trống đã đo): Agent Map **không** thuộc bộ tối thiểu cho workflow hằng ngày; hàng "└ ● running" dưới session và item `agent` trong transcript đã đủ. Ưu tiên: nguồn thay đổi trung thực (F-02/F-03) và bước verify mặc định (X-02).

---

## 9. Lời khai lệch thực tế

| Nơi | Ghi | Thực tế |
|---|---|---|
| `CLAUDE.md` P5 "Máy" | Ryzen 7 9700X 8 nhân/16 luồng, RAM 60 GiB, swap 2 GiB | i7-8750H, `nproc`=12, RAM 15,5 GiB, swap 8 GiB (đo hôm nay) |
| `CLAUDE.md` P5 `taskset -c …13-15` | chạy trên 16 luồng | rc=0 nhưng CPU 13-15 không tồn tại bị bỏ lặng lẽ |
| `CLAUDE.md` P5 | 65 gói ok, ~45s | 66 gói ok, 1m51s (đo hôm nay, `-p 3`) |
| `CLAUDE.md` P4 Home | "MỤC TIÊU, đang làm (H2)" | đã cài đặt và có test |
| `docs/agent-cli-local-setup.md:~183` | mọi lần ghi cấu hình (vd chọn model) nạp lại toàn bộ | `UpdatePreferredModel` không reload (`store.go:441-457`); chỉ `SetConfigField(s)`/`RemoveConfigField` |
| `docs/reviews/…phase1-open-issues.md` C2 | `recent_models` phình | đã cắt ở `maxRecentModelsPerType` (`store.go:1198`) |
| `schema.json:659` / `config.go:482` | `request_timeout` default 60, âm = không hợp lệ | default 2 phút, âm = tắt |
| `verify-gate.sh` header | đặt biến trong `~/.config/agent-cli/env` | biến không tới hook (C-02) |
| `docs/reviews/2026-10-16-ux-theme-continuation-changes.md` §A2 | palette cũ `#D0D2D1`/`#929997`/`#7D8585` | code là `#E6E9E7`/`#B2B8B6`/`#99A09E` |
| `DECISIONS.md` 15/10 | `everforest-dark-v2` | đã giải thích ở mục 16/10; tên file giữ nguyên |
| Binary cài | — | `335709e4`, cũ hơn main 3 commit |

---

## 10. Kế hoạch sửa (thứ tự theo phụ thuộc, không theo độ dễ)

**Wave 0 — chặn rò rỉ và thành công giả**
1. X-01: chặn `--no-index`/đường dẫn ngoài workspace/`printenv` trong safe-list; test âm.
2. X-02: `ExitCode` trong `BashResponseMetadata`; UI đổi icon; quyết định `IsError` cho exit≠0.
3. C-01: trần/timeout của Stop hook đặt `RunComplete.Error`, `run --json` rc=1 + `gate_failed`.
4. C-05 + B-05: test nối `blockFuncs()` và bổ sung bad-case (`xargs kill`, `sh -c`, `kill -9 -1`).

**Wave 1 — đúng đắn**
5. A-02/A-03: không persist khi fallback; cảnh báo khi resume không khôi phục model; thêm provider/model vào JSON của `run`.
6. A-04, A-06, A-07: cwd cho `git rev-parse`; env client tới bash; `EstimatedUsage` qua proto.
7. D-03: IO `Update` → `tea.Cmd` + timeout HTTP.
8. C-02: sửa doc/script `verify-gate.sh`.

**Wave 2 — cô lập / độ tin cậy**
9. A-01 (+B-07): `KillAll`/MCP theo workspace, owner cho `job_kill`.
10. B-02: trust thư mục cho crushrc/hook/MCP/base_url.
11. B-08, A-08, A-09, A-10.

**Wave 3 — UX/TUI/theme**
12. D-01 (footer), F-02/F-03 (nhãn + nguồn git), D-05, D-04, D-06, D-10.

**Wave 4 — dọn dẹp:** A-13, C-11, D-14, đồng bộ doc lệch (§9).

**Chưa nên làm:** worktree (P3), checkpoint/undo (P2, git thay được), Agent Map (P2 sau khi biết ảnh từ đâu), dashboard.

---

## 11. Phụ lục bằng chứng

### Lệnh do người điều phối chạy

| Lệnh | Kết quả |
|---|---|
| `go.sh test ./... -count=1 -p 3` | 66 ok / 17 no-test / 0 FAIL, rc=0, 1m51s; `go list ./…` = 83 gói, khớp 83 dòng kết quả |
| `git status --short --ignored` trước/sau, `diff` với bản chụp đầu | giống hệt (trước khi ghi báo cáo này) |
| `go.sh test -overlay=… ./internal/agent/tools -run TestZZAuditB01 -v` | `isSafeCommand`: `git status`=true, `rm -rf x`=false, `git diff --no-index /dev/null /etc/hostname`=true, `… ~/.config/agent-cli/env`=true, `printenv`=true, `git config --list`=true, `git diff --output=/tmp/x`=false |
| `go.sh test -overlay=… ./internal/shell -run TestZZAuditB05 -v` | `kill <pid>` bị chặn; `xargs kill -9`, `bash -c "kill …"`, `sh -c "kill -9 …"`, `kill -9 -1` → `Blocked=false` |
| `git log -S` / `merge-base --is-ancestor` | `isReadOnlyGit` do `c330628e`; test inline-code do `461b540f`, là tổ tiên của `335709e4` |
| quét 5 nhánh + binary cài (`strings`) cho "agent map" | 0 / 0 |
| đọc `load.go:80-100,1315-1335`, `app.go:940-950`, `background.go:350-368`, `client.go:55-68`, `client_workspace.go:283-285,400-420`, `ui.go:5930-5942`, `bash.go:225-245`, `safe.go:125-160`, `quickstyle.go:328-348,596-620`, `everforest_dark_v2.go:421-430` | khớp báo cáo agent |

Hai tệp test inject (`zz_audit_*_test.go`) và `overlay.json`, `go.sh` nằm trong thư mục scratchpad của phiên, ngoài repo. `-overlay` không ghi vào repo.

### Agent và phạm vi

| Agent | Domain | Test đã chạy (qua cổng) |
|---|---|---|
| A | runtime / state / config / dead code | backend, agent, cmd, config (nhắm trúng) — ok |
| B | provider / multi-agent / quyền / skill / MCP / git | execpolicy, provider, shell(Protect), tools, agent(Policy…), permission, filetracker, skills, gitutil, shellconfig, mcp — ok |
| C | workflow / verify / test quality / lỗi | agent(StopHook…), hooks, shell(Protect), tools(Bash/Safe), execpolicy — ok |
| D | TUI + theme | ui/styles, home, common, diffview, exitbanner, dialog, chat, model(nhóm) — ok; tính tương phản bằng `python3 -I` |
| E | phản biện B/C | không chạy test; đọc mã (kể cả `fantasy@v0.45.1`) |
| F | năng lực hằng ngày + Agent Map | ui/model(TestChildRun/SessionActivity/SessionsPanel) — ok |

Danh sách lệnh/tệp chi tiết của từng agent nằm trong báo cáo của agent đó; không được lưu lại thành tệp (đúng quy ước một tệp output).

### Tự khai quy trình

- Agent A gọi `go version`, `go list ./... | wc -l` trực tiếp (không qua cổng) và `ls` thư mục cache — chỉ đọc metadata.
- Agent C vô tình chạy hai lô test giống nhau (một lô ra `/dev/null`) — không ghi tệp.
- Agent F `ls ~/.config/agent-cli` (chỉ tên tệp, không đọc nội dung).
- Không agent nào ghi tệp; xác nhận cuối bằng `git status` (xem dưới).

### Chưa kiểm được

- Không chạy động bất kỳ phát hiện nào ở trên trên binary thật (không build, không E2E, không `-race`, không mở TUI, không chụp màn hình — luật phiên).
- Không đo mật độ màu thực trên màn hình, độ chói thị giác, hạ cấp 16/256 màu, terminal <40 cột, Unicode ZWJ/emoji.
- Không chạy `claude_code`/`codex` thật, không chạy `agent-cli-tmux.sh`.
- Không so sánh trực tiếp với upstream (remote `upstream` có khai báo nhưng chưa `fetch`); nguồn gốc "upstream hay fork" dựa trên tác giả/ngày commit.
- Không đo hiệu năng (đọc lại message hai lần ở `ReadSettledMessages`, sidebar dựng mỗi frame, goroutine/watcher).
- Không đọc cấu hình/log thật của người dùng; theme đang chọn thật chưa biết.
- Không chạy `unused`/lint; danh sách mã chết chưa đầy đủ.
- A-01, A-04, A-06, A-07, A-08, A-09, A-10 mới được truy tĩnh; A-01 và A-04 được người điều phối đọc lại, các mục còn lại chưa ai bác bỏ.
