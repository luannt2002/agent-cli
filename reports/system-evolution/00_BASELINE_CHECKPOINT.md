# 00 — Checkpoint hệ thống trước đợt tiến hoá (Phase 0)

Chụp ngày 2026-10-16. Mọi số dưới đây đến từ lệnh chạy trong phiên này; lệnh ghi ngay cạnh.

## Mốc khôi phục

| Mục | Giá trị | Lệnh |
|---|---|---|
| Commit | `4a0b25b698fbc2ad838f62d9990a93e9154cbe46` | `git rev-parse HEAD` |
| Nhánh | `main`, đi trước `origin/main` 8 commit, chưa push | `git rev-list --left-right --count origin/main...HEAD` → `0 8` |
| Tag baseline (local, chưa push) | `evo/baseline-20261016` → `4a0b25b6` | `git tag evo/baseline-20261016 4a0b25b6` |
| Tệp bẩn / chưa track | 0 / 0 | `git status --porcelain=v1` |
| Tệp ignored có mặt | `.crush.json` (cấu hình provider của user), `.crush/` (dữ liệu dự án), `.superpowers/` | `git status --ignored` |
| Worktree | 1 (cây chính) | `git worktree list` |
| Stash | `stash@{0}` (nhánh `feat/personal-rename-providers`, 4 tệp untracked: `agent-cli.config.example.json`, `docs/agent-cli-local-setup.md`, `docs/progress/PHASE1_STATUS.md`, `scripts/install-agent-cli.sh`) — **giữ nguyên, không đụng** | `git stash show --include-untracked --name-status 'stash@{0}'` |
| Tags phát hành | `v0.1.0` = `87233bc8`, `v0.2.0` = `8568a886` | `git tag --list` |

**Cổng "khôi phục được" — đã chứng minh:** `git worktree add --detach <tạm> evo/baseline-20261016` → HEAD `4a0b25b6`, 0 tệp bẩn, rồi `git worktree remove`.

Cách quay về (chỉ khi cần, phá thay đổi sau mốc): `git diff evo/baseline-20261016` để xem; `git switch -c <nhánh-cứu> && git reset --hard evo/baseline-20261016` để về hẳn. Đợt tiến hoá **không commit khi user chưa cho phép**, nên mỗi lát sửa để trong cây làm việc, kèm tệp patch ở `evidence/` để gỡ riêng.

## Môi trường chạy thật (chỉ đọc tên/metadata, không đọc secret)

| Mặt | Trạng thái |
|---|---|
| Binary đã cài | `~/.local/lib/agent-cli/crush` build từ `335709e4` (trước v0.2.0), wrapper `~/.local/bin/agent-cli`. **Đang được một phiên khác dùng → không cài đè** |
| `~/.config/agent-cli/` | `crush.json` = symlink → `<repo>/.crush.json`; `crushrc` (khoá: `model`, `small_model`, `base_url`, `context_window`, `max_tokens`, `env_file`); `env` (0600, secret, KHÔNG đọc); `themes/`; `credentials/` (KHÔNG đọc) |
| `~/.local/share/agent-cli/` | 24K: `crush.db`, `projects.json`, `config-backups/` |
| Dữ liệu từng dự án | `<dự án>/.crush/` (SQLite riêng mỗi dự án) |
| Máy | i7-8750H 12 luồng, RAM 15,9 GB (MemAvailable lúc chụp 7,3 GB), swap 8 GB (đã dùng 5,1 GB). Trần: tối đa 3 agent, Go qua `go.sh` (flock + chờ RAM ≥ 3 GB) |

### Cấu hình provider/model (đã che secret bằng `jq`, khoá chứa key/token/auth → `<redacted>`)

| Provider | Kiểu | Model | Ghi chú |
|---|---|---|---|
| `glm` ("Z.ai Coding Plan") | `openai-compat`, `https://api.z.ai/api/coding/paas/v4` | `glm-5.3`, `glm-5.3-flash` (context 1M) | provider duy nhất đang bật |
| `anthropic` | `anthropic` | — | `disable: true` |

Tuỳ chọn: `disable_default_providers: true`, `disable_provider_auto_update: true`, `disable_metrics: true`, `disabled_tools: [sourcegraph, web_search, web_fetch, fetch, download]`, theme `github-dark-dim`.
**`permissions.allowed_tools: [view, ls, glob, grep, rg, edit, write, bash]`** → theo cấu hình của user, bash/edit/write không hỏi quyền; lớp chặn còn lại là BlockFunc ở trình thực thi. Tác động thật sẽ kiểm ở Phase 1 (mục Permissions), chưa kết luận ở đây.

## Skill có trên máy

| Nơi | Số | Nội dung |
|---|---|---|
| `~/.agents/skills/` | 20 | superpowers (brainstorming, writing-plans, executing-plans, TDD, systematic-debugging, verification-before-completion, …), ponytail (+ audit/debt/gain/review), `superpower`, `code-en-doc-vi` |
| `~/.claude/skills/` | 7 | `code-en-doc-vi`, `flow-tu-dong`, `glossary-keeper`, `retro`, `superpower`, `synced`, `writing-go-postgres-repositories` |
| Builtin của agent-cli | `internal/skills/builtin/` | (kiểm kê ở Phase 2) |

agent-cli có thật sự nạp skill nào trong số này không: **chưa biết**, là câu hỏi của Phase 2/3.

## Test baseline (chạy lúc chụp)

`bash go.sh test ./... -count=1 -p 2` → **68 ok, 17 không có test, 0 FAIL, rc=0**, 2m15s. Danh sách theo TÊN gói ở [evidence/00_test_baseline.txt](evidence/00_test_baseline.txt); đối chiếu với `go list ./...` (85 gói): lệch 0. So sánh về sau phải theo tên, không theo số.

E2E đã chạy ở commit này (trước Phase 0, cùng phiên): `scripts/test-agent-cli-loop.sh` 17/17, `scripts/test-agent-cli-routine.sh` 24/24, đều đỏ với binary đột biến. Chưa chạy lại ở checkpoint.

## Kiến trúc hiện tại (con trỏ, chi tiết ở Phase 1)

`AGENTS.md` mục Architecture · `docs/reviews/2026-10-16-deep-audit-agent-cli.md` §2 (đo ở `87233bc8`) · thêm sau đó: `internal/loop`, `internal/routine`, `internal/cmd/{loop,routine}.go`, `internal/shell/unattended.go` (v0.2.0 → HEAD).
Quy mô: 121.920 dòng Go production + 90.962 dòng test trong `internal/` (`find … | xargs cat | wc -l`).

## Lỗi/giới hạn đã biết lúc chụp

Không có test đỏ. Các điểm hở đã ghi (nguồn: `CHANGELOG.md` mục "Chưa làm" của 0.2.0 và mục Unreleased, `DECISIONS.md`):

- A-01 tắt một workspace giết job nền + MCP của workspace khác cùng server.
- B-02 `.crushrc` của dự án chạy như shell, không qua mô hình tin cậy.
- A-02/A-03 model mất khỏi catalog bị thay lặng lẽ; resume không báo khi không khôi phục được model.
- D-03 IO HTTP đồng bộ trong `Update` của TUI; D-01 footer cắt; F-02 "Modified Files" chỉ đếm tool ghi.
- Safe-list là danh sách đen theo cú pháp; `git ls-remote <url>` tự duyệt; symlink trong dự án trỏ ra ngoài.
- Chế độ không người trực KHÔNG phải ranh giới an ninh (python/node, `find -delete`, redirect, `cd` trước `rm`, script từ tệp); worktree routine dùng chung refs/config/hooks; guard chưa qua lượt soát thứ ba sau lần sửa cuối.
- Chưa có: Agent Map, UI cho loop/routine, pause/resume, lịch tự cài.
