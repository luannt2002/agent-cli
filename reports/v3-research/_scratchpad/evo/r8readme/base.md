# agent-cli

Trợ lý lập trình AI chạy trong terminal, bản fork cá nhân của [Crush](https://github.com/charmbracelet/crush) (Charm). Nối với model local (server tương thích OpenAI, không cần key) hoặc provider qua API key, cấu hình bằng các biến `AGENT_CLI_*` dùng chung.

Fork giữ nguyên giấy phép FSL-1.1-MIT của upstream, xem [LICENSE.md](LICENSE.md). Bản gốc chưa sửa: [docs/UPSTREAM-README.md](docs/UPSTREAM-README.md), hữu ích để tra cú pháp cấu hình.

## Cài và dùng

```bash
bash scripts/install-agent-cli.sh          # build vào ~/.local, tạo crushrc + env mẫu nếu chưa có
export PATH="$HOME/.local/bin:$PATH"

cd /path/to/project
agent-cli                                  # giao diện TUI
agent-cli run "Đọc repo và tóm tắt kiến trúc, chưa sửa code."   # CẢNH BÁO: run TỰ DUYỆT mọi quyền
agent-cli --continue                       # nối phiên gần nhất
```

> **Cảnh báo:** `agent-cli run` tự duyệt MỌI lời xin quyền (bash, sửa tệp, giao việc cho Claude Code/Codex) của session nó chạy, kể cả session có sẵn qua `-s`/`-C`; "chưa sửa code" trong prompt chỉ là lời dặn model, không phải rào chắn. Muốn được hỏi thì dùng TUI hoặc `agent-cli send` (send không tự duyệt).

Cần sẵn `git` và `go` để cài; `sha1sum`/`shasum` cho launcher, `jq` cho installer và `agent-cli-tmux`, `tmux` cho `agent-cli-tmux`, `python3` và `script` để chạy các script kiểm (chi tiết ở hướng dẫn bên dưới).

Hướng dẫn đầy đủ (cấu hình model, kiểm chứng, cập nhật): [docs/agent-cli-local-setup.md](docs/agent-cli-local-setup.md).

## Cấu hình

| Việc | Ở đâu |
|---|---|
| Provider, model, key | biến `AGENT_CLI_*` trong `~/.config/agent-cli/env` (mode 600, mẫu: `agent-cli.env.example`); key chữ thường đặt trong nháy đơn, hoặc `"$(pass show <entry> \| head -n1)"` |
| Cấu hình chính, quyền công cụ | `~/.config/agent-cli/crushrc` (mẫu: `agent-cli.crushrc.example`) |
| Màu giao diện | `~/.config/agent-cli/themes/*.json`, chọn bằng `options.tui.active_theme` |
| Dữ liệu phiên, log | `~/.local/share/agent-cli`, `.crush/` trong từng project |

Tên tệp và biến môi trường bên trong (`crush.json`, `CRUSH_*`, `.crush/`) giữ nguyên để dễ gộp thay đổi từ upstream. Launcher `agent-cli` đã đặt sẵn các biến đó, tắt metrics và tắt cập nhật catalog provider; kiểm tra phiên bản upstream thì tắt sẵn ngay trong binary (launcher còn đặt `CRUSH_DISABLE_UPDATE_CHECK=1`; chỉ chạy thẳng binary với `CRUSH_DISABLE_UPDATE_CHECK=false` mới bật được).

## Phím tắt và luồng dùng

Phím hiển thị ngay dưới chân màn hình; `ctrl+g` xem thêm. Bảng dưới là các phím hay dùng (nguồn: `internal/ui/model/keys.go`).

**Màn Home (chọn/mở dự án):** gõ để tìm · `↑↓` chọn · `enter` mở · `ctrl+p` ghim/bỏ ghim · `ctrl+x` bỏ dự án đã mất khỏi danh sách · `esc` thoát.

**Màn chat:**

| Phím | Việc |
|---|---|
| `enter` | gửi tin · `ctrl+j` xuống dòng · `ctrl+o` mở editor ngoài |
| `@` | chèn file vào ngữ cảnh · `/` hoặc `ctrl+p` bảng lệnh |
| `ctrl+f` | đính kèm ảnh · `ctrl+v` dán ảnh từ clipboard |
| `tab` | đổi focus (ô nhập ↔ khung chat) · `shift+tab` đổi mode · `ctrl+y` bật/tắt YOLO |
| `alt+n` | session mới · **`f2` đổi tên session hiện tại** · `alt+s` danh sách session |
| `alt+.` / `alt+,` | session kế / trước |
| `alt+p` | chọn provider · `alt+m` model của provider đó · `alt+e` mức effort |
| `alt+w` | về Home · `alt+j` bảng Jobs (tiến trình bash nền) |
| `ctrl+b` | ẩn/hiện sidebar · `ctrl+t` ẩn/hiện To-Do · `ctrl+d` chi tiết |
| `esc` | huỷ lượt đang chạy · `ctrl+c` thoát · `ctrl+z` treo |

> `f2` đổi tên session đổi tên đúng session đang mở; tên bạn đặt KHÔNG bị tiêu đề do model sinh ghi đè. Vài terminal tự chiếm `f2`; khi đó dùng `alt+s` rồi `ctrl+r`.

**Luồng dùng thường ngày:**

1. `agent-cli` ở thư mục dự án → màn Home → `enter` mở dự án (hoặc chọn "Open current folder").
2. Gõ yêu cầu, `enter`. Model chạy, gọi tool, hỏi quyền (mặc định Deny — bấm cho phép từng cái hoặc "Allow for Session").
3. `f2` đặt tên session cho dễ tìm. Mở việc khác song song: `alt+n` tạo session mới, `alt+.`/`alt+,` qua lại.
4. Context gần đầy thì tự tóm tắt; trả lời bị cắt vì hết token thì tự làm tiếp (đặt `AGENT_CLI_MAX_TOKENS` đủ lớn trong env cho model suy luận).
5. Đóng TUI vẫn để run chạy tiếp; mở lại `agent-cli --session <id>` hoặc `agent-cli --continue` để nối. `agent-cli ps --all` liệt kê session đang chạy.

Cổng nghiệm thu tuỳ chọn (chặn model báo "xong" khi test/lint fail): bật Stop hook `verify-gate.sh`, xem [docs/hooks/examples/verify-gate.sh](docs/hooks/examples/verify-gate.sh).

## Khác upstream

- Tên và giao diện: wordmark AGENT-CLI, banner thoát, thông báo, trợ giúp lệnh.
- Không kiểm tra bản mới của upstream: mặc định tắt trong binary, không gọi GitHub (chỉ bật khi chạy thẳng binary với `CRUSH_DISABLE_UPDATE_CHECK=false`).
- Crushrc mẫu tắt dòng `Assisted-by: Crush` / `Generated with Crush` mà upstream bảo model thêm vào commit và PR (cách bật lại: [docs](docs/agent-cli-local-setup.md#attribution-trong-commit-và-pr)).
- Nhiều session/dự án trong một TUI (Home, Alt+W/S/N, Alt+. / Alt+, chuyển session), draft riêng từng session, run tiếp tục khi đóng TUI; `run --json`, `send --wait`, `ps`, `logs` cho script.
- Đổi provider (Alt+P), model của provider đang dùng (Alt+M), mức effort (Alt+E). Hộp thoại quyền mặc định Deny.
- Tool `claude_code`, `codex`: giao việc cho Claude Code / Codex đã đăng nhập trên máy, mỗi lần giao đều hỏi quyền trừ khi bạn chủ động cho qua (`run`, `--yolo`, "Allow for Session"; xem [docs](docs/agent-cli-local-setup.md#giao-việc-cho-claude-code--codex-tool-claude_code-codex)).
- Chưa đổi: tên module Go, system prompt (còn tự xưng "Crush"), skill và công cụ nội bộ `crush-config`, `crush_info`.

## Phát triển

```bash
go build .
go test ./...        # máy dùng chung: đặt GOMAXPROCS=2 và go test -p 2
```

Quy ước code và kiến trúc: [AGENTS.md](AGENTS.md).
