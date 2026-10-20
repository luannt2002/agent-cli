# Kế hoạch đổi tên agent-cli → DEW

Theo `reports/v3-research/spec/DEW_RECONSTRUCTION_WORKFLOW.md` §39–43. Người dùng chốt đổi tên ngày 20/10/2026.

**Đợt 1 (20/10):** đổi phần git và tên hiển thị. Phần chạm tới máy và cấu hình người dùng để cho pha DEW, vì cần cách
chuyển đổi.

## Số đo (20/10, `git grep` ngoài `reports/`)

| Thứ | Số lượng |
|---|---|
| Tệp có chữ `agent-cli` | 155 |
| Tệp `.go` có chữ `agent-cli` (chuỗi hiển thị, help, lỗi, test) | 111 |
| Biến `AGENT_CLI_*` | 30 biến khác nhau, 543 lần xuất hiện |
| Tệp nhắc `~/.config` / `~/.local/share` / `~/.cache` `/agent-cli` | 30 |
| Tệp mang tên `agent-cli` (script cài, launcher tmux, 6 script E2E, 2 tệp mẫu cấu hình, docs) | 14 |
| Module Go | `github.com/charmbracelet/crush` |

## Phân loại

| Loại (spec §39) | Hiện tại | Đích | Quyết định (spec §40) | Ghi chú |
|---|---|---|---|---|
| REPOSITORY_NAME | `luannt2002/agent-cli` | `luannt2002/dew` | **SAFE_NOW**, cần người làm trên GitHub | Settings → Repository name. GitHub chuyển hướng URL cũ (clone/fetch/push) |
| Thư mục trên máy | `opensource/agent-cli` | `opensource/dew` | **SAFE_NOW** (đợt 1) | Sau khi đổi: `git worktree repair` cho worktree phụ. Registry dự án giữ mục đường cũ cho tới lần mở mới |
| DISPLAY_NAME trong docs | "agent-cli" | "DEW — Daily Engineering Workspace" | **SAFE_NOW** ở README và docs đồng bộ (đợt 1); phần docs còn lại để pha DEW | Docs phải nói rõ lệnh vẫn là `agent-cli` |
| BINARY_NAME | `agent-cli` (wrapper) + `crush` (binary trong `~/.local/lib/agent-cli/`) | `dew` | **MIGRATION_REQUIRED**, DEFER | Cần giai đoạn alias `agent-cli` → `dew` có cảnh báo; script cài, launcher, `agent-cli-tmux` và 6 script E2E đổi theo |
| CONFIG_PATH | `~/.config/agent-cli`, `~/.local/share/agent-cli`, `~/.cache/agent-cli` | `…/dew` | **MIGRATION_REQUIRED**, DEFER | Đụng env có key, credentials, registry dự án, log server. Cần đọc đường cũ khi đường mới chưa có, không chép key tự động |
| ENV_PREFIX | `AGENT_CLI_*` (30 biến) | `DEW_*` | **BACKWARD_COMPATIBILITY_REQUIRED**, DEFER | Gồm các khoá `AGENT_CLI_*` của chính sách tin cậy (`internal/execpolicy`, danh sách 11 khoá có cả `HOME`): đọc cả hai tên một thời gian; hai tên mang giá trị khác nhau thì từ chối như chính sách lệch |
| Chuỗi hiển thị trong mã Go (help, lỗi, TUI) | "agent-cli" | "dew" | DEFER, đi cùng BINARY_NAME | 111 tệp `.go`; nhiều test so chuỗi |
| MODULE_PATH | `github.com/charmbracelet/crush` | giữ | **DO_NOT_RENAME** | Invariant CLAUDE.md P3: giữ để merge upstream rẻ |
| `crush.json`, `CRUSH_*`, system prompt "Crush" | giữ | giữ | **DO_NOT_RENAME** | Invariant P3; đổi system prompt làm đỏ cassette `TestCoderAgent` |
| DATABASE_NAMESPACE (`.crush/crush.db`) | giữ | giữ | **DO_NOT_RENAME** | Dữ liệu người dùng; theo upstream |
| USER_AGENT / telemetry | upstream | — | DEFER (kiểm ở pha DEW) | Chưa audit |
| Tên tệp lịch sử (`docs/reviews/*agent-cli*`, `docs/superpowers/plans/*agent-cli*`, `reports/`) | giữ | giữ | **DO_NOT_RENAME** | Là hồ sơ lịch sử; đổi tên làm gãy liên kết trong báo cáo |

## Đã làm ở đợt 1

- Thư mục trên máy đổi thành `dew`; worktree phụ đã sửa đường.
- README có tiêu đề DEW và lưu ý lệnh vẫn là `agent-cli`.
- Thêm `docs/dew-sync.md`: lấy code, đồng bộ, đổi remote cho bản clone cũ.
- `CLAUDE.md` P3 trỏ đường mới.

## Cần người dùng làm

1. Đổi tên repo trên GitHub: `luannt2002/agent-cli` → `dew`. Máy này không có `gh`, và lệnh đẩy lên GitHub bị bộ phân
   loại quyền chặn.
2. Sau đó, trên máy này:
   `git remote set-url origin https://github.com/luannt2002/dew.git`, rồi push qua alias SSH
   `git@github.com-luannt2002:luannt2002/dew.git`.

## Đã làm ở pha DEW (20/10, nhánh `dew-rename`)

Người dùng chốt "đổi tên toàn bộ, đừng để agent-cli nữa": KHÔNG có giai đoạn alias và không đọc song song `AGENT_CLI_*`
(khác bảng trên, ghi ở DECISIONS.md mục 2026-10-20).

- [x] BINARY_NAME: `dew`, `dew-tmux`; binary ở `~/.local/lib/dew/crush`.
- [x] CONFIG_PATH: `~/.config/dew`, `~/.local/share/dew`, `~/.cache/dew`; socket `dew-<hash>.sock`.
- [x] ENV_PREFIX: `DEW_*`. Biến tin cậy tên cũ còn trong env gốc hay tệp env thì `internal/execpolicy` dừng với lỗi nêu
  cả hai tên.
- [x] Chuỗi trong mã Go: "dew" (lệnh), "DEW" (header, logo, Home, banner thoát, trang OAuth, thông báo). `userAgent`
  = `dew/<version>`: cassette VCR không so header User-Agent (`TestCoderAgent` xanh).
- [x] Tệp: `scripts/install-dew.sh`, `scripts/dew-tmux.sh`, `scripts/test-dew-*.sh`, `dew.{crushrc,env}.example`,
  `docs/dew-local-setup.md`.
- [x] Chuyển đổi trong `scripts/install-dew.sh`, test trong `scripts/test-dew-wrapper.sh`; "Lưu ý nâng cấp" ở CHANGELOG.
- Còn chữ agent-cli có chủ đích: giá trị `agent-cli-pending` (DB), danh sách chặn `selfNames`/`ProcessGuard`,
  `.gitignore` `/agent-cli.env`, mã và test chuyển đổi, hồ sơ lịch sử (`docs/reviews`, `docs/superpowers`, `reports/`,
  mục đã phát hành của CHANGELOG, mục cũ của DECISIONS).
- Chưa làm: chạy trình cài thật trên máy (điều phối làm sau khi soát).

## Pha DEW (kế hoạch ban đầu, đã thay bằng mục trên)

BINARY_NAME, CONFIG_PATH, ENV_PREFIX và chuỗi trong mã theo bảng trên, mỗi phần kèm:
- test chuyển đổi (máy đang có cấu hình cũ vẫn chạy);
- giai đoạn alias;
- một dòng "Lưu ý nâng cấp" trong CHANGELOG.
