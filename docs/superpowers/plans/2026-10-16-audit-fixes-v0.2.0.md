# Plan: sửa theo audit 16/10 và phát hành v0.2.0

Nguồn: `docs/reviews/2026-10-16-deep-audit-agent-cli.md` (HEAD `87233bc8`). Plan này là brief, không phải đặc tả từng dòng.

**Người thi hành:** phiên hiện tại, được quyền phản biện plan (plan do chính phiên này viết sau khi đọc mã). Mã an ninh qua đúng MỘT lượt soát độc lập: agent riêng, không có Edit, ghi `KET-LUAN: DAT|KHONG-DAT` ở dòng đầu.

**Cấm:** không viết số chưa chạy · không nở phạm vi · không push upstream (`charmbracelet/crush`) · không cài đè binary đang chạy `~/.local/lib/agent-cli/crush` (có phiên agent-cli đang dùng) · không sửa test cho xanh mà không nói rõ test đó mã hoá lỗi.

## Mục tiêu

Đóng các lỗi Wave 0 của audit và những sửa một-dòng đã chứng minh, rồi đặt version + changelog + tag để dùng được.

| # | Audit | Việc | Tệp chính |
|---|---|---|---|
| T1 | X-01 | Bash tự duyệt không được đọc ngoài workspace | `agent/tools/safe.go`, `bash.go` |
| T2 | X-02 | Exit code bash vào metadata; UI không hiện ✓ xanh cho lệnh lỗi | `agent/tools/bash.go`, `ui/chat/bash.go` |
| T3 | B-05, C-05 | Self-kill guard: bóc `sh -c`, `xargs kill`, `kill -1`; test nối `blockFuncs()` | `shell/protect.go`, `agent/tools/bash_policy_test.go` |
| T4 | C-01 | Stop hook chạm trần 3 lần mà vẫn chặn → lượt báo lỗi, không báo xong | `agent/stop_hook.go` |
| T5 | A-04 | `isInsideWorktree` dùng thư mục workspace, không dùng cwd tiến trình | `config/load.go` |
| T6 | C-02 | `verify-gate.sh` + doc chỉ đúng chỗ đặt biến; cảnh báo khi biến rỗng | `docs/hooks/examples/verify-gate.sh`, README |
| T7 | — | `CHANGELOG.md`, cập nhật `CLAUDE.md` (P2, P6), `DECISIONS.md`, tag `v0.1.0` (baseline `87233bc8`) và `v0.2.0` | gốc repo |

**Không làm lượt này** (lý do ghi trong CHANGELOG mục "Chưa làm"): A-01 (job nền theo workspace) và B-02 (trust crushrc) vì đổi vòng đời/mô hình tin cậy cần quyết định sản phẩm; D-03 (IO trong `Update`) vì chạm ≥8 điểm trong `ui.go` 7000 dòng; D-01 footer vì test đóng băng thứ tự phím; A-02/A-03 resume/model vì đổi hành vi cấu hình; mọi LOW.

## Ràng buộc

- Mỗi sửa: test ĐỎ trước (chạy thấy đỏ), sửa, test XANH. Một commit mỗi mục T1–T7, hoàn tác riêng được.
- Go chỉ qua `scratchpad/go.sh` (một lệnh nặng một lúc, chờ RAM ≥3GB). `-race` + `./...` chạy một lần ở cuối.
- Không đổi system prompt / mô tả tool / SKILL.md builtin (đỏ cassette VCR `TestCoderAgent`). T2 không đặt `IsError` vì nó đổi nội dung gửi model.
- Khớp style sẵn có; comment tiếng Việt, identifier tiếng Anh.

## Chỗ tôi không chắc (kèm cách kiểm)

| Điều | Cách kiểm |
|---|---|
| Siết safe-list làm đỏ test đang mã hoá hành vi cũ (`printenv HOME`, `echo $VAR`) | Chạy `go.sh test ./internal/agent/tools`; test nào đỏ phải tách: mã hoá lỗi (sửa test, ghi rõ) hay hồi quy thật (sửa mã) |
| Chặn token `/`, `~` làm các lệnh hay dùng (`git diff HEAD~1`, `git log a..b`, `ls -la`) bị hỏi quyền oan | Bảng dương trong test: các lệnh này PHẢI vẫn `true` |
| T4 đổi một hành vi có tài liệu (README hook) | Đọc `stop_hook.go` + test `LoopGuard`; chỉ đổi nhánh chạm trần, giữ fail-open cho timeout/crash; cập nhật README đúng chỗ |
| `go build` có gắn tag vào `--version` không | Sau tag: build vào thư mục scratchpad, chạy `--version`, không cài |
| Quyền push main qua SSH alias | Đã `push --dry-run` OK; push thật xác nhận |

## Cách chứng minh là xong

1. Từng T: lệnh test đỏ → xanh, in kết quả thật.
2. Cổng phải chặn được ít nhất một lần: T1 bảng âm (`git diff --no-index`, `printenv`, `ls /etc`, `cat ~/x`, `working_dir` ngoài) trả `false` SAU sửa và `true` TRƯỚC sửa.
3. `go.sh test ./... -count=1 -p 3`: 0 FAIL, so theo TÊN với baseline (66 ok / 17 no-test); `go.sh test -race` cho `agent/tools`, `shell`, `agent`, `config`.
4. Soát độc lập T1+T3+T4 → `KET-LUAN: DAT`.
5. `--version` của binary build thử ở scratchpad hiện `v0.2.0`.
6. `git push` chỉ tới `git@github.com-luannt2002:luannt2002/agent-cli.git`; sau push, `git ls-remote` thấy `main` và hai tag.
