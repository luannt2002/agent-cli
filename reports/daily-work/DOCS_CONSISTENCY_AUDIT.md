# Kiểm tính nhất quán của tài liệu (§85)

So tài liệu của ứng viên cuối (`scratchpad/evo/wt-b`) với mã, runtime và test. Hai cách: (1) soát cơ học bằng lệnh,
phủ mọi chỗ tài liệu nhắc tới một thứ kiểm được; (2) đối chiếu ngữ nghĩa từng hành vi mới với mã và binary, do lead và
hai lượt soát độc lập làm.

## 1. Soát cơ học

Tài liệu được soát: `README.md`, `docs/agent-cli-local-setup.md`, `CHANGELOG.md`, `DECISIONS.md`, `CLAUDE.md`,
`AGENTS.md`, `internal/ui/AGENTS.md`, `docs/hooks/README.md`, `docs/config/README.md`.

| Kiểm | Cách | Số chỗ soát | Lệch | Ghi chú |
|---|---|---|---|---|
| Tên test được nhắc có tồn tại | regex `Test[A-Z]\w+` trong docs, so với `func Test…(` của mọi tệp `.go` (2.696 test) | 46 | 1, báo nhầm | `TestYourFunction` ở `AGENTS.md` là ví dụ mẫu |
| Đường dẫn được nhắc có tồn tại | regex các đường `internal/…`, `scripts/…`, `docs/…` trong backtick | 72 | 1, báo nhầm | `docs/progress/PHASE1_STATUS.md`: CLAUDE.md ghi rõ tệp này chỉ nằm trong `stash@{0}` |
| Biến env được nhắc có trong mã/script | regex `AGENT_CLI_*`, `CRUSH_*` | 83 | 1, báo nhầm | `AGENT_CLI_VERIFY_CMD` nằm ở `docs/hooks/examples/verify-gate.sh`, mục 0.2.0 của CHANGELOG |
| Lệnh con và cờ README nhắc có trong `--help` | `--help` của binary cuối, môi trường cô lập (`scratchpad/evo/docs-audit/help-*.txt`) | 33 cờ dài, 6 cờ ngắn, 10 lệnh con | 0 | |
| Link và anchor trong README | dựng slug kiểu GitHub cho mọi tiêu đề của tệp đích | 20 | 0 | chạy lại sau lần sửa README cuối |

Script soát nằm trong lịch sử phiên (lệnh `python3 -I` đọc trực tiếp các tệp trên); không thêm vào repo vì chỉ dùng
một lần.

## 2. Đối chiếu ngữ nghĩa

| Hành vi | Mã | Runtime (binary thật) | Test | README | Guide | Config/mẫu |
|---|---|---|---|---|---|---|
| Lời gọi ẩn ở lại provider chính | `sameProviderSmall` trong `buildAgentModels` | qwen-only: 12/12 ở qwen | `TestSameProviderSmall`, `TestBuildAgentModelsKeepsSmallOnTheMainProvider` | khớp | khớp | — |
| Model đang chọn không dùng được → không đổi provider; `run`/`loop`/`routine` dừng; app vẫn khởi động (`login`, TUI vào màn chọn model) | `resolveSelectedModels`, `noFallbackError`, `agent.SelectedModelUnavailable`, `app.New`, onboarding ở `ui.go` | faults `selected-no-key` (local) và `selected-no-key-server` (qua server): rc=1, 0 request, lỗi nêu provider, `login` → "Unknown platform"; TUI: màn chọn model, 0 request (local: lead; server: người soát vòng 7) | 10 test (SE 32, vòng 6 + 7) | khớp (sau lượt soát README 2: cách gỡ đúng là key, `login <nền tảng>`, hoặc xoá `models` ở `.crush/crush.json`) | khớp | — |
| Gợi ý trong thông báo lỗi trên | `noFallbackError` | đo: `run -m` KHÔNG gỡ được | `TestNoFallbackErrorKeepsSentinel` khẳng định không còn gợi ý `-m provider/model` | khớp | khớp | — |
| Allowlist: có mục không phải `claude_code:`/`codex:` thì large và small phải nằm trong danh sách | `execpolicy.Parse` (`restricted`), `checkModelPolicy(large, small)` | `MODEL_NOT_ALLOWED` (CLAUDE.md P8, 15/10) | `policy_test.go` | **đã sửa**: bản trước viết "có ít nhất một provider chính", mơ hồ (người soát README 2) | khớp | mẫu env ghi luật không điều kiện; đúng với trường hợp thường gặp |
| `NO_CHANGE` và mã thoát | `loop.Engine`, `loopExitErr`, `routineExitErr` | qwen-only: `stop: NO_CHANGE`, rc=1 | 8 test (SE 32) | **đã sửa**: bản trước nói "`loop`/`routine` thoát 1" rồi lại nói routine maintenance thoát 0 | khớp | help của `loop`, `routine run` khớp |
| Lịch không chép allowlist | `scheduleEnv`, `warnAllowlistNotScheduled` | — (không cài lịch thật, CLAUDE.md cấm đụng crontab) | 4 test | khớp | khớp | — |
| `text` của `run --json`: qua server là câu trả lời cuối; local nối mọi chữ assistant; bản tóm tắt không có ở cả hai | `runStream.handle` (server), `app.RunNonInteractive` (local) | qwen-only (local): `DONE-AFTER-SUMMARY`; người soát vòng 6: câu dẫn + câu trả lời nối lại ở local, chỉ câu trả lời qua server | `TestRunStream_SkipsTheAutoSummary` | **đã sửa** ở vòng 7 (bản vòng 6 nói "chỉ câu trả lời cuối" cho cả hai) | đã sửa | — |
| Provider dựng sẵn bị tắt thì không gọi | `buildAgentModels` (`Disable`) | người soát vòng 6: bản trước gửi 8 request tới provider đã tắt | `TestBuildAgentModelsRefusesADisabledProvider` | khớp | khớp | — |
| Nhãn provider của `run --json` không phải bằng chứng | `runJSON` đọc tin đã lưu | ca `provider-down`: 0 request vẫn `"provider":"qwen"` | — | khớp | khớp | — |
| Chốt tràn context chỉ xét đầu vào | `agent.go` (StopWhen) | — | `TestLargeOutputRightAfterSummaryIsNotAnOverflow` + 2 biến thể cũ | khớp ("mà việc còn dở") | khớp | — |
| `glob`/`grep` hỏi quyền khi ra ngoài dự án | `requestOutsideAccess` trong `glob.go`, `grep.go` | — | test có sẵn của tool | khớp | khớp | **đã sửa**: comment trong `agent-cli.crushrc.example` nói "không có lời hỏi nào" |
| Shift+Tab xoay tới "code + YOLO" cho cả workspace | TUI | — | — | **đã sửa**: bản trước chỉ ghi "đổi mode" | khớp | — |
| `send` không tự duyệt, cần TUI mở session | `cmd/send.go` | E2E run-json | — | **đã thêm** | khớp | — |
| Kiến trúc: catwalk là catalog, config là `*config.ConfigStore`, `common.Common` giữ `workspace.Workspace` | đọc mã | — | — | — | — | `AGENTS.md`, `internal/ui/AGENTS.md` đã sửa |

## 3. Còn lệch hoặc chưa kiểm

- Help của `run --json` (trong `run --help`) không liệt kê `provider`/`model` dù output có hai trường này (người soát README 2 thấy). Đây là help text tụt sau mã; ghi lại, chưa sửa (đổi help là đổi mã, phải kiểm lại).
- Thông báo lỗi "model không dùng được": đã sửa ở vòng 7 (`agent-cli login <platform>`, nêu cả ca bị tắt/mất ID model).
- `docs/agent-cli-local-setup.md` còn mã PRB-xx (giải thích ở đầu mục "Giới hạn đã biết"), một commit hash và tên ca của `faults.sh`: tài liệu nâng cao, chấp nhận.
- Hành vi chỉ thấy trong TUI (hộp thoại, Jobs, mode): không kiểm bằng mắt (luật không đụng màn hình); kiểm bằng unit test và tmux riêng.
