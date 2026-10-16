# Changelog

Theo [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) và [SemVer](https://semver.org/lang/vi/).
Mỗi bản có một tag git có chú thích `vX.Y.Z`. `agent-cli --version` hiện đúng số đó khi build từ commit
được gắn tag (Go gắn version từ tag, không cần `-ldflags`; build từ commit sau tag sẽ hiện dạng
`vX.Y.Z-0.<ngày>-<hash>`, cây bẩn có đuôi `+dirty`).

Quy trình phát hành: xem [CLAUDE.md](CLAUDE.md), mục P6.

## [Unreleased]

### Thêm

- **`agent-cli loop --verify "<lệnh>" "<việc>"`**: chạy agent, rồi lệnh verify, sửa tiếp cho tới khi verify
  thoát 0 hoặc gặp điều kiện dừng. "Xong" là mã thoát của `--verify`, không phải lời agent. Mỗi lượt là một
  session mới với prompt tự chứa đủ (đề bài + đuôi output verify vừa đo, kèm lệnh cấm sửa test/cổng để qua).
  Lý do dừng: `GREEN`, `MAX_REPAIRS` (mặc định 2 lượt sửa, tổng 3 lượt, không có lần vá thứ ba), `STALLED` (cùng
  chữ ký lỗi và cây làm việc không đổi sau một lượt sửa), `TIMEOUT` (`--max-duration`, kiểm trước mỗi lượt),
  `RUN_ERROR` (lượt agent lỗi hoặc quá `--run-timeout`), `CANCELLED` (Ctrl+C/SIGTERM), `UNMEASURED` (verify
  không chạy được: không tìm thấy lệnh, lỗi cú pháp, quá hạn; không đốt lượt agent nào). Verify đã xanh sẵn
  trước mọi thay đổi thì ghi cảnh báo "cổng có thể không đo đúng mục tiêu" nhưng vẫn chạy.
  Journal JSON Lines (`<data>/loops/<tên>-<thời điểm>.jsonl`, 0600) được ghi và `Sync` TRƯỚC mỗi bước.
  Mã thoát: 0 `GREEN`, 1 mọi lý do dừng khác, 2 dùng sai. `--json` in một object ở cuối. Không bao giờ commit,
  push hay merge. Cùng mô hình quyền như `run` (tự duyệt tool): dành cho việc bạn khởi chạy và trông.
  Chưa có: lên lịch / chạy không người trực (cần danh sách chặn lệnh theo lượt, xem audit R-01), UI.

### Sửa lỗi

- **Tiêu đề markdown (H1) vô hình trong theme tuỳ chỉnh.** Theme tuỳ chỉnh (vd `github-dark-dim` trong
  `~/.config/agent-cli/themes/`) không nhận lớp "yên tĩnh" của theme dựng sẵn, nên H1 rơi về kiểu `quickStyle`
  gốc: chữ `warning_subtle` trên nền `primary`. Với `#ffa657` trên `#79c0ff` đó là 1,00:1 (đo bằng công thức
  WCAG). Nay theme tuỳ chỉnh dùng chữ `fg_base` cho H1, heading và inline code, nền `bg_less_visible`, H6 dùng
  `fg_subtle`; chỉ dùng token của chính theme nên theme sáng vẫn đúng. Theme dựng sẵn (Charmtone, Gruvbox,
  mặc định) không đổi. Phải cài lại binary mới thấy (`scripts/install-agent-cli.sh`).
  Chưa sửa: logo gradient (`secondary → primary`) và các style khác của `quickStyle` trong theme tuỳ chỉnh.

## [0.2.0] - 2026-10-16

Đợt sửa theo [audit 16/10](docs/reviews/2026-10-16-deep-audit-agent-cli.md) (mã nguồn `87233bc8`).
Plan: [docs/superpowers/plans/2026-10-16-audit-fixes-v0.2.0.md](docs/superpowers/plans/2026-10-16-audit-fixes-v0.2.0.md).

### Bảo mật

- **bash tự duyệt không còn đọc được ngoài workspace** (audit X-01). Lệnh chỉ-đọc có đối số là đường dẫn
  tuyệt đối, `~`, `..`, `$VAR`, `--cờ=/đường/dẫn`, `git diff --no-index`, hoặc chạy với `working_dir` ngoài
  workspace nay HỎI quyền. Đối số được kiểm sau khi bỏ nháy, `\` và chặn brace expansion, vì shell ghép chuỗi
  trước khi chạy (`ls \/etc`, `ls {/etc,}`, `ls .''./etc` từng lọt ở bản thử đầu).
  `printenv`, `ps` (cờ `e` in biến môi trường) và mọi `git config` không còn trong danh sách an toàn;
  giá trị dính liền cờ ngắn (`date -f/etc/hostname`) cũng bị coi là đường dẫn.
- **Self-kill guard kín hơn** (B-05): script sau `sh/bash/zsh/dash/ksh/rbash -c` (kể cả `-c --`) và `eval` được
  PHÂN TÍCH CÚ PHÁP bằng `mvdan.cc/sh/v3/syntax` rồi soi từng lệnh đơn, kể cả trong `for/if/while`, hàm,
  `$(...)`, backtick; nháy và `\` chen giữa từ được giải (`ki""ll`). Bản đầu dùng regex tách chuỗi và bị lượt
  soát độc lập phá ngay (`for p in N; do kill $p; done`). Thêm: chặn `xargs kill` (PID từ stdin), `kill -1` /
  pid 1, `kill` không PID tường minh (trừ `kill -l`); bóc thêm `exec`, `builtin`, `time`, `taskset`, `busybox`
  và đối số thời gian `5s` của `timeout`; tên lệnh động (`$k X`, `"$@"`) cạnh chữ kill thì chặn. Tối đa 3 lớp
  `sh -c` lồng nhau, sâu hơn thì chặn. Có test nối `blockFuncs()` vào shell thật (C-05): xoá dòng nối ở `bash.go`
  thì test đỏ.
  Cố ý KHÔNG phủ: script nạp từ tệp hoặc stdin (`bash x.sh`, `echo 'kill N' | sh`), `env -S`, `find -exec`,
  `script -c`, ngôn ngữ khác (`python -c "os.kill"`). Chặn oan đã biết: `kill $VAR` bên trong `bash -c` (không
  biết giá trị lúc phân tích); dùng `/bin/kill $VAR` ở lớp ngoài (đã expand thành số) hoặc tool `job_kill`.

### Sửa lỗi

- **Lệnh bash thoát mã ≠ 0 không còn hiện ✓ xanh** (X-02). `BashResponseMetadata.ExitCode` mới; UI đổi icon
  thành lỗi và ghi `(exit=N)`. Không đặt `IsError`, nên nội dung gửi model và cassette VCR giữ nguyên.
- **Stop hook vẫn chặn sau 3 lần chạy lại thì lượt báo lỗi** (C-01): `RunComplete` cuối mang
  `Stop hook still blocking after 3 re-runs: …`; `run` coi `RunComplete.Error` là lỗi (`agent run failed: …`,
  `internal/cmd/run.go`) nên thoát khác 0. Test chỉ phủ phần `RunComplete`; chưa có test `run` end-to-end.
  Timeout, crash và halt vẫn kết thúc như thành công (hook không quyết định được).
- **Giới hạn duyệt tệp theo workspace, không theo cwd tiến trình** (A-04): server thừa hưởng cwd của client
  đầu tiên nên mở Home từ `$HOME` từng làm mọi repo git sau đó bị giới hạn `ls` depth 2 / 100 mục.

### Tài liệu

- `verify-gate.sh` chỉ đúng chỗ đặt `AGENT_CLI_VERIFY_CMD` (inline trong lệnh hook; tệp
  `~/.config/agent-cli/env` không tới được hook) và cảnh báo ra stderr khi biến rỗng (C-02).
- `docs/agent-cli-local-setup.md` và `docs/hooks/README.md` cập nhật theo hành vi mới.
- Thêm báo cáo audit và plan sửa; thêm `CHANGELOG.md`.

### Chưa làm (đã biết, có trong audit)

- Tắt một workspace giết job nền + MCP của workspace khác cùng server (A-01).
- `.crushrc` của dự án chạy như shell không hỏi (B-02, mã upstream); cần mô hình tin cậy thư mục.
- Model cấu hình mất khỏi catalog bị thay lặng lẽ và ghi vào config global (A-02); resume không báo khi không
  khôi phục được model (A-03).
- IO HTTP đồng bộ trong `Update` của TUI, không timeout (D-03).
- Footer cắt `ctrl+g more` ở terminal <162 cột (D-01); "Modified Files" chỉ đếm tool edit/write (F-02).
- Safe-list chưa nhận ra đường dẫn tương đối đi qua symlink trong dự án trỏ ra ngoài (ghi `shortcut:` ở `safe.go`).
- `git ls-remote <url>` vẫn tự duyệt: có thể đẩy dữ liệu ra mạng qua URL hoặc liệt kê repo ngoài (có từ trước).
- Danh sách an toàn là danh sách đen theo cú pháp; hướng đúng là danh sách trắng cờ cho từng lệnh.
- Stop hook chạm trần: lỗi đặt lên `RunComplete` của lượt chính và của prompt xếp hàng xả ra giữa chuỗi chạy
  lại, không đặt lên prompt xả ở lượt cuối; chưa có test cho trường hợp prompt xếp hàng.
- Sau khi cài lại binary (`mv -f`), server cũ có `/proc/<pid>/exe` kết thúc bằng ` (deleted)` nên tiến trình
  mới không nhận ra nó là "cùng binary" để bảo vệ (`pkill`/`killall` theo tên vẫn bị chặn).
- Mọi mục LOW của audit.

### Kiểm chứng

- `go test ./...`: 66 gói ok, 17 không có test, 0 FAIL; cùng tập gói và trạng thái với baseline trước khi sửa.
  `go test -race` sạch cho `internal/agent`, `internal/agent/tools`, `internal/config`, `internal/shell`,
  `internal/ui/chat`.
- Soát độc lập (agent riêng, không có quyền sửa, Opus): lượt 1 bắt được 3 lỗ còn lại của safe-list (đã sửa, mỗi
  đòn thành test); lượt 2 bắt được các lỗ của guard (đã sửa bằng cách phân tích cú pháp). Bản guard sau khi viết
  lại CHƯA qua lượt soát thứ ba; bảng tấn công của reviewer đã được chuyển thành test, và guard là lớp chặn thêm
  (bash vẫn hỏi quyền cho lệnh không an toàn, trừ `run`/YOLO).
- Chưa chạy: `run --json` end-to-end với Stop hook chạm trần; build có `GOEXPERIMENT=greenteagc` như installer.

### Lưu ý nâng cấp

- Một số lệnh chỉ-đọc từng tự duyệt nay hỏi quyền (đối số ngoài workspace, `printenv`, `ps`, `git config`).
  Đó là chủ ý; cấp quyền theo phiên nếu cần.

## [0.1.0] - 2026-10-16

Mốc cơ sở, gắn vào commit `87233bc8`: trạng thái fork cá nhân của [Crush](https://github.com/charmbracelet/crush)
sau Phase 1 (45 commit từ điểm fork `a73e1bfa`, 14–16/10/2026). Tóm tắt từ `git log`; chi tiết ở
`DECISIONS.md` và `docs/reviews/`.

### Thêm

- Đổi tên thành agent-cli, logo và nhãn riêng, tắt kiểm tra cập nhật mặc định, bản cài cục bộ có sao lưu và lệnh
  quay lại (`scripts/install-agent-cli.sh`).
- Nhiều session và nhiều dự án trong một terminal: Home, server dùng chung theo dự án, run sống khi đóng TUI,
  khôi phục tool call bỏ dở, `run --json` / `send --wait` / `ps` / `jobs` / `logs` / `session` cho script.
- Phím Alt: `Alt+W/S/N/M/P/E/J`, `Alt+.` / `Alt+,`, `F2` đổi tên session, bảng Jobs, chọn provider rồi model.
- Ranh giới provider/model (`AGENT_CLI_ALLOWED_MODELS`), giao việc Claude Code/Codex tắt mặc định và hỏi quyền
  mỗi lần.
- Bảo vệ tiến trình điều khiển (không tự giết agent-cli, không tắt máy) ở lớp thực thi shell.
- Stop hook "cổng nghiệm thu" tuỳ chọn (`docs/hooks/examples/verify-gate.sh`).
- Theme deep-charcoal, chữ và accent sáng hơn; tool output dùng hết bề ngang chat.

### Sửa

- Không gửi JSON tool call hỏng cho provider; tiếp tục khi auto-summarize lỗi; nối lại câu trả lời bị cắt ở giới
  hạn output; giữ lịch sử và yêu cầu thật khi tiếp tục lượt.
- Không để tiêu đề LLM ghi đè tên đặt tay.

[Unreleased]: https://github.com/luannt2002/agent-cli/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/luannt2002/agent-cli/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/luannt2002/agent-cli/releases/tag/v0.1.0
