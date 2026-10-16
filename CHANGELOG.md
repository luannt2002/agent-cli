# Changelog

Theo [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) và [SemVer](https://semver.org/lang/vi/).
Mỗi bản có một tag git có chú thích `vX.Y.Z`. `agent-cli --version` hiện đúng số đó khi build từ commit
được gắn tag (Go gắn version từ tag, không cần `-ldflags`; build từ commit sau tag sẽ hiện dạng
`vX.Y.Z-0.<ngày>-<hash>`, cây bẩn có đuôi `+dirty`).

Quy trình phát hành: xem [CLAUDE.md](CLAUDE.md), mục P6.

## [Unreleased]

## [0.2.0] - 2026-10-16

Đợt sửa theo [audit 16/10](docs/reviews/2026-10-16-deep-audit-agent-cli.md) (mã nguồn `87233bc8`).
Plan: [docs/superpowers/plans/2026-10-16-audit-fixes-v0.2.0.md](docs/superpowers/plans/2026-10-16-audit-fixes-v0.2.0.md).

### Bảo mật

- **bash tự duyệt không còn đọc được ngoài workspace** (audit X-01). Lệnh chỉ-đọc có đối số là đường dẫn
  tuyệt đối, `~`, `..`, `$VAR`, `--cờ=/đường/dẫn`, `git diff --no-index`, hoặc chạy với `working_dir` ngoài
  workspace nay HỎI quyền. Đối số được kiểm sau khi bỏ nháy, `\` và chặn brace expansion, vì shell ghép chuỗi
  trước khi chạy (`ls \/etc`, `ls {/etc,}`, `ls .''./etc` từng lọt ở bản thử đầu).
  `printenv` và `git config --list` không còn trong danh sách an toàn.
- **Self-kill guard kín hơn** (B-05): chặn `sh -c 'kill <pid>'` / `bash -c ...` (bóc script sau `-c`, tối đa
  3 lớp lồng), `xargs kill` (PID đến từ stdin không kiểm được), `kill -9 -1` và pid 1.
  Có test nối `blockFuncs()` vào shell thật (C-05): xoá dòng nối ở `bash.go` thì test đỏ.

### Sửa lỗi

- **Lệnh bash thoát mã ≠ 0 không còn hiện ✓ xanh** (X-02). `BashResponseMetadata.ExitCode` mới; UI đổi icon
  thành lỗi và ghi `(exit=N)`. Không đặt `IsError`, nên nội dung gửi model và cassette VCR giữ nguyên.
- **Stop hook vẫn chặn sau 3 lần chạy lại thì lượt báo lỗi** (C-01): `RunComplete` cuối mang
  `Stop hook still blocking after 3 re-runs: …`, `run --json` trả rc=1. Timeout, crash và halt vẫn kết thúc
  như thành công (hook không quyết định được).
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
- Mọi mục LOW của audit.

### Lưu ý nâng cấp

- Một số lệnh chỉ-đọc từng tự duyệt nay hỏi quyền (đối số ngoài workspace, `printenv`, `git config --list`).
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
