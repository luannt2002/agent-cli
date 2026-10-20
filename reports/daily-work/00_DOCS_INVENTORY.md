# 00 — Kiểm kê tài liệu (§73)

Người kiểm: agent chỉ-đọc riêng (không có quyền ghi), đối chiếu từng khẳng định với mã của ứng viên `scratchpad/evo/wt-b` (HEAD `4a0b25b6` + thay đổi chưa commit). 59 tệp `.md` được git theo dõi; không tệp nào nằm trong testdata. Số dòng mã là của wt-b.

**Nguyên tắc sửa (§79):** hành vi chỉ có trong ứng viên (small model ép về provider chính, sổ ghi LLM, `provider`/`model` trong `run --json`, trần tóm tắt…) chỉ được ghi vào tài liệu **cùng patch** với mã, tức là sửa trong wt-b, không sửa ở cây chính. Cây chính giữ tài liệu khớp HEAD.

## Bảng

| Tệp | Mục đích | Lớp | Xung đột chính (bằng chứng) | Việc |
|---|---|---|---|---|
| README.md | Trang đầu của fork | PARTIALLY_STALE | `ps --all` không phải "liệt kê session đang chạy": `ps` liệt kê mọi session (BUSY/WAITING/IDLE) của dự án có server; `--all` chỉ bỏ trần 9 session mỗi dự án (`ps.go:28,67-69,86`). `claude_code`/`codex` mặc định TẮT, chỉ có khi `AGENT_CLI_DELEGATION=on` + ID trong allowlist + có CLI (`execpolicy/policy.go:31-35`). Không nhắc loop/routine/jobs/session/logs | Viết lại (Daily §75) |
| docs/agent-cli-local-setup.md | Hướng dẫn đầy đủ | PARTIALLY_STALE | Sai 8 chỗ, gồm: token trong `run --json` là của bước cuối, chỉ cost cộng dồn (`agent.go:2530,2547-2550`); `view` ngoài dự án CÓ hỏi quyền (`tools/view.go:122-132`); danh sách lệnh lồng bị chặn thiếu `loop`/`routine` (`policy.go:406`); bảng màu theme lệch `everforest_dark_v2.go:58-64`; ngưỡng tự tóm tắt 20% chỉ cho window ≤200k (`agent.go:66-68`). Thiếu PRB-04 trong "Đã biết" | Sửa 8 chỗ; thêm mục hành vi của ứng viên trong cùng patch |
| CLAUDE.md | Luật agent + sự thật dự án (agent-cli tự nạp) | CONTRADICTORY | P5 dùng `taskset -c …13-15` (máy có CPU 0-11, `lscpu`); "65 package ok" (nay 68 ok / 17 không test). P1/P2 còn "Phase 1", "ngoài phạm vi Phase 2–5" dù loop/routine (Phase 4) đã ra. P2 nói sổ `.superpowers/sdd` "đã mất" nhưng thư mục còn trên đĩa (không theo dõi, loại trừ ở `.git/info/exclude`), `docs/progress/PHASE1_STATUS.md` nằm trong `stash@{0}`. `ui.go` ~7.000 dòng (không phải 5.700) | Sửa P5 (lệnh cho máy này), P1/P2/P7 chỗ lệch. **Không** đụng stash |
| CHANGELOG.md | Nhật ký phát hành | CURRENT cho HEAD | `[Unreleased]` phủ đủ 8 commit sau v0.2.0. Câu "lên lịch routine chạy được" chỉ đúng với ứng viên (HEAD: lịch thiếu biến launcher, rc=2 — PRB-07) | Thêm mục cho ứng viên trong cùng patch |
| AGENTS.md | Hướng dẫn dev (upstream + ghi chú fork) | PARTIALLY_STALE | Trỏ tới `HOOKS.md`, `internal/llm/prompt`, `internal/tui/...`, `config.Service`: đều không tồn tại; gọi catwalk là công cụ snapshot test (thực ra là catalog provider) | Sửa tham chiếu gãy |
| DECISIONS.md | Sổ quyết định append-only | CURRENT | Chưa có mục cho quyết định của đợt này | Thêm mục cùng patch |
| docs/config/README.md | Tham chiếu crushrc (upstream) | MISSING_CONTEXT | `request-timeout` "(default 60)" sai: mặc định 2 phút (`config.go:490`); đường dẫn `~/.config/crush` | Sửa giá trị mặc định + dòng ghi chú fork |
| docs/hooks/README.md | Giao thức hook | PARTIALLY_STALE | Hành vi đúng; chữ còn `crush run`, `~/.config/crush` | Sửa chữ |
| internal/ui/AGENTS.md | Luật TUI | PARTIALLY_STALE | `common.Common` giữ `Workspace`, không phải `*app.App` (`ui/common/common.go:133-136`) | Sửa nhỏ |
| docs/superpowers/plans/2026-10-14-agent-cli-personal-roadmap.md | Roadmap Phase 1–5 | PARTIALLY_STALE | loop/routine (Phase 4) đã ra mà không có ghi chú sửa đổi | Thêm ghi chú trỏ DECISIONS |
| docs/superpowers/plans/2026-10-14-core-mvp-lat1-lat2.md | Plan lát 1–2 | STALE | Bắt mọi lệnh Go qua `.superpowers/sdd/…/go.sh`, mà CLAUDE.md gọi là "plan hiện hành" | Gỡ khỏi P2 của CLAUDE.md |
| docs/reviews/2026-10-14-phase1-open-issues.md | Vấn đề mở Phase 1 | SHOULD_ARCHIVE | Phần lớn đã sửa; chỉ C1 (= PRB-04) còn mở lúc kiểm kê (nay đã đóng, vòng 6–7) | Giữ tệp như lịch sử, ghi chú "lỗi thời" ở đầu (đã làm) |
| docs/superpowers/plans/2026-10-16-master-status-and-batches.md | Bảng trạng thái | PARTIALLY_STALE | Chụp trước v0.1.0, không biết loop/routine | Ghi chú "lịch sử" |
| 10 tệp lịch sử khác (docs/reviews/*, plans/* đã xong, specs/*, CLA.md, docs/UPSTREAM-README.md, config/hooks FUTURE.md) | Lịch sử / upstream | HISTORICAL_ONLY | — | Giữ nguyên |
| LICENSE.md, .agents/skills/*, internal/cmd/stats/AGENTS.md, internal/oauth/callback/AGENTS.md | | CURRENT | — | Giữ |
| Template prompt và mô tả tool (`internal/agent/templates/*.md`, `tools/*.md`, skill builtin) | Nội dung system prompt | CURRENT | Sửa = đổi system prompt = phải ghi lại cassette VCR | Không đụng |
| reports/system-evolution/, reports/daily-work/ (chưa theo dõi) | Báo cáo kỹ thuật của đợt này | GENERATED_REPORT | Nơi duy nhất liệt kê PRB còn mở và bản đồ thay đổi | Không commit nguyên khối; tóm tắt sang CHANGELOG/DECISIONS; hỏi người dùng có giữ `reports/` trong repo không |

## Tính năng hằng ngày có trong mã mà tài liệu chưa nói

`agent-cli loop` (cờ, trường `--json`), `agent-cli routine` (cờ, trạng thái RUNNING/ABANDONED, cờ báo cáo), chế độ không người trực dùng tay (`AGENT_CLI_UNATTENDED=1`), `jobs` (CLI), `logs -f -t`, các cờ của `run` (`-m provider/model`, `--small-model`, `--reasoning-effort`, `-q`, `-v`, nhận stdin), `session list/show/last/delete/rename --json`, lệnh slash từ `.claude/commands`, `CRUSH_SERVER_IDLE_TIMEOUT`. Hành vi của ứng viên: small model ép về provider chính, sổ ghi LLM, `provider`/`model` trong `run --json`, allowlist trong lịch phải ngăn bằng phẩy (ứng viên tự đổi khoảng trắng thành phẩy).

## Đã kiểm, đúng

33 khẳng định trong README khớp mã (cài đặt, đường dẫn cấu hình/dữ liệu, launcher, phím tắt Home và chat, Alt+…, F2, `run` tự duyệt, `send` không tự duyệt, hộp thoại quyền mặc định Deny, tự tóm tắt + tự làm tiếp, `--session/--continue`). Không tài liệu nào khẳng định "không có fallback chéo provider" vô điều kiện.

Nguồn đầy đủ: báo cáo của agent kiểm kê (lưu trong transcript phiên); mọi dòng "sai" đều có tệp:dòng.

Tự kiểm lại bằng lệnh (16/10, không đọc nội dung, không đụng stash): `ls .superpowers/sdd/` → 3 mục; `.git/info/exclude:7` = `/.superpowers/`; `git stash show --include-untracked --name-only stash@{0}` có `docs/progress/PHASE1_STATUS.md` (4 tệp); `nproc` = 12; `wc -l internal/ui/model/ui.go` = 7000.
