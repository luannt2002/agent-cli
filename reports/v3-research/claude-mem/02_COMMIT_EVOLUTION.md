# claude-mem — tiến hoá qua commit (spec §8)

- Repo: `/home/luannt/application/opensource/v3-src/claude-mem` (chỉ đọc). HEAD `3fae1eb45ba1a7fbd4818527f48b439991a273da` (2026-10-19, `v14.0.1-62`).
- Nguồn: `git log` / `git show` / `git for-each-ref` / `CHANGELOG.md` / `plans/`. Không dùng API GitHub. 7 trang issue đọc ở `01_BASE_ARCHITECTURE.md` §12.
- Ngày trong tệp này là **author date** (`%ad`). Có nhiều squash-merge dồn cục, ví dụ 169 commit mang author date 2026-09-30, 221 commit mang committer date 2026-09-30. Vì vậy "ngày" chỉ đáng tin ở mức tuần.

## 0. Số liệu gốc (lệnh vừa chạy)

| Số | Giá trị | Lệnh |
|---|---|---|
| Tổng commit | 3479 | `git rev-list --count HEAD` |
| Khoảng thời gian | 2025-09-06 → 2026-10-19 | `git log --format=%ad --date=short \| sort \| sed -n '1p;$p'` |
| Tag | 393 (major đầu tiên v3.3.8 → v14.0.1) | `git tag \| wc -l`; `git for-each-ref --sort=creatordate refs/tags` |
| Nhánh remote | 763 | `git branch -r \| wc -l` |
| Commit merge | 298 | `git log --merges --oneline \| wc -l` |
| Có trailer `Co-Authored-By: Claude` | 1805 | `git log --format=%H -i --grep='Co-Authored-By: Claude' \| wc -l` |
| Có `Claude-Session:` | 544 | `git log --format=%H --grep='Claude-Session:' \| wc -l` |
| Tác giả hàng đầu | Alex Newman 2426, Rudy Mizrahi Celekli 223, Rod Boev 78, posthog[bot] 61, copilot-swe-agent[bot] 39 | `git shortlog -sn HEAD` |
| Theo tiền tố subject | fix 1317 · chore 408 · docs 364 · other 359 · feat 348 · merge 286 · MAESTRO 156 · refactor 140 · test 54 · build 25 · perf 9 · ci 7 · revert 6 | sed/awk trên `git log --format=%s` (cộng lại đúng 3479) |
| Commit fix có sửa `tests/` | 887/1317 | awk trên `git log --format='@@%h %s' --name-only` |

Commit theo tháng: 2025-09 26 · 10 228 · 11 220 · 12 481 · 2026-01 116 · 02 381 · 03 82 · 04 289 · 05 123 · 06 165 · 07 261 · 08 134 · 09 398 · 10 575.

Commit fix có kèm test, theo tháng (`fix` / `có sửa tests/`):

| Tháng | 2025-10 | 2025-11 | 2025-12 | 2026-01 | 2026-02 | 2026-03 | 2026-04 | 2026-05 | 2026-06 | 2026-07 | 2026-08 | 2026-09 | 2026-10 |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| fix | 38 | 31 | 79 | 34 | 78 | 43 | 101 | 37 | 58 | 94 | 37 | 271 | 416 |
| có test | 1 | 2 | 1 | 10 | 15 | 13 | 37 | 21 | 39 | 77 | 27 | 248 | 396 |

→ Năm 2025 gần như không có test đi kèm fix (4/148). Từ 2026-07 trở đi thì ≥82%. "Test chỉ được thêm sau hồi quy" đúng cho cả nửa đầu vòng đời dự án (E1, số đo từ lệnh trên).

---

## 1. Các thời kỳ

### E0 — v3.x: hook script + chroma-mcp (2025-09-06 → 2025-10-14)
- **Mô hình ban đầu:** gói npm phát hành liên tục (v3.3.8 → v3.9.x trong một tháng). Cây ở `v3.3.8` gồm `hooks/session-start.js`, `hooks/pre-compact.js`, `hooks/session-end.js`, `hooks/shared/*`, `install.sh` và tài liệu tham khảo chroma-mcp (`git ls-tree -r --name-only v3.3.8`) — E1.
- **Áp lực:** hook đọc/ghi trực tiếp, nén lúc pre-compact. Không có tiến trình nền.

### E1 — v4: SDK agent + worker (2025-10-15 → 2025-11-02)
- **Viết lại:** "Phase 1/2/3 of SDK agent architecture" (`e81ea6914`, `78fd1368d`, `047298a18`, 2025-10-15). Chuyển sang `bun:sqlite` (`2663121d9`). Worker service có session + Agent SDK (`372854948`, 2025-10-17). Worker chạy dưới PM2 (`be0ab1278`, 2025-10-19). MCP search server (`56167c47a`). v4.0.0 "Plugin data directory and auto-starting worker" (`002f7a94b`).
- **Thành bất đồng bộ / bền:** observation do một LLM observer chạy nền sinh ra. Đây là quyết định gốc kéo theo mọi họ lỗi worker về sau.
- **Danh tính:** context hook lấy project từ cwd (`7c5e5b194`, 2025-10-16), tức basename(cwd).
- Đổi summary từ mức session sang mức request (`ef572ec03`, 2025-10-21).
- Matcher SessionStart bỏ `startup` để tiêm cả khi resume/clear/compact (`f499810c7`, 2025-10-25). Gần một năm sau phải đảo lại: #4061, không tiêm khi `resume`.

### E2 — v5–v6: Chroma lai + viewer + "Endless Mode" (2025-11-03 → 2025-12-04)
- **Thêm:** tìm kiếm lai Chroma + SQLite (v5.0.0, CHANGELOG `:8217`, E0; `633f89a5f`). Viewer web (`79ff1849f`). Worker v2 (`980151a50`). Skill tìm kiếm với progressive disclosure (`0475a57fb`). Endless Mode (`928582654`, 2025-11-16, nhánh `origin/beta/endless-mode`).
- **Đắt lên:** thêm phụ thuộc Python/uv cho Chroma.

### E3 — v7–v9: hook thành HTTP client, mô-đun hoá, Live Context (2025-12-05 → 2026-01-31)
- **Viết lại:** mọi hook thành HTTP client, bỏ hết SQL trong hook (`8e0b1ee4e`, 2025-12-05). v7.0.0 "Major Architectural Refactor". Worker từ hơn 1600 dòng tách thành routes (CHANGELOG `:6619`, E0).
- **Bỏ:** PM2, chuyển sang runtime Bun (`807d1d610`, 2025-12-11, gỡ `"pm2"` khỏi `package.json`).
- **Thêm:** mode system và đa ngôn ngữ (v8). Live Context: CLAUDE.md sinh theo thư mục, có worktree (`e1ab73dec`, 2026-01-05). Đây là lần đầu `getProjectContext` có `allProjects=[parent, primary]` với primary = basename của worktree. Tách memory session id khỏi content session id (v9, E0).
- **Theo host:** sửa Windows Terminal tab tích tụ bằng cách xoá thư mục `src/hooks/*` cũ (`601596f5c`, 2026-01-09).

### E4 — MAESTRO và mở rộng host (2026-02-01 → 2026-03-31)
- 156 commit tiền tố `MAESTRO:` (2026-02-04 → 2026-02-23): agent tự động xử lý PR/issue (`git log --format=%s | grep -c '^MAESTRO'`). Có cả các thí nghiệm không merge, ví dụ `origin/branch-memory` (2026-03-16, thêm `branch`/`commit_sha` cho observation; `git merge-base --is-ancestor origin/branch-memory HEAD` → NOT_MERGED; `grep commit_sha src` = 0) — E1.
- **Thêm:** OpenClaw (v10). Embedded Process Supervisor (`80a8c90a1`, 2026-03-16). Cô lập nguồn Claude/Codex bằng `platform_source` (`2b60dd293`, 2026-03-24).

### E5 — v11–v12: tính năng dồn dập và "Issue Blowout" (2026-04-01 → 2026-04-30)
- **Thêm:** tier routing theo độ phức tạp của hàng đợi (`0fcc07887`). Semantic injection qua Chroma trên UserPromptSubmit (`876cc4d83`, nay mặc định `false`). SSH sync nhiều máy `claude-mem-sync` (`5a2742080`). File-read gate (v12).
- **Worktree:** ghi theo tên parent (`16a0737df`, #1820) → composite `parent/worktree` (`040729bee`) → đọc gồm cả parent (`dc198d567`) → remap cwd một lần lúc boot (`193e7e071`) + cột `merged_into_project` (`3d1dfcc26`) + adoption engine (`a7c3c4af2`). Tất cả cùng ngày 2026-04-15/16.
- **Áp lực / thất bại:**
  - "Issue Blowout 2026 — 25 bugs" (`ba1ef6c42`, 2026-04-20) → rollback (`bfc7de377`) → revert của rollback (`8d166b47c`), cùng ngày.
  - Cùng ngày gỡ bộ lọc `platform_source` của context (`03748acd6`, #2081).
  - "Cynical deletion: close 27 issues by removing defenders + tolerators" (`d13662d5d`, 2026-04-25).
  - Revert "overengineered summary salvage logic" (`d0fc68c63`).

### E6 — v13: server-beta, đổi giấy phép, bỏ hàng đợi bền (2026-05-01 → 2026-06-30)
- **Thêm:** server-beta với Postgres + BullMQ + `/v1` (`36b0929fa`, 2026-05-08), đổi giấy phép AGPL→Apache-2.0 (CHANGELOG `:2368`). Skill `oh-my-issues` để gom issue theo nguyên nhân gốc (`ea057feee`, 2026-05-21).
- **Bỏ (abstraction bị xoá):** "remove retry/observation queue, replace with SessionMessageBuffer" (`61fe70a20`, 2026-05-28). Xoá `PendingMessageStore`, `SessionQueueProcessor`, `BullMqObservationQueueEngine`, `ObservationQueueEngine`, `RestartGuard` và 5 tệp test. Kết quả: +482/−3400 dòng (`git show --stat`). Lý do ghi trong mã: hàng đợi bền là "retry storm" (`SessionMessageBuffer.ts:30-45`) — E1.
- **Danh tính:** chuyển sang git root (#2663, `daf6d9dc0`, 2026-05-28; ra mắt ở 13.4.0 theo #2882) **không kèm migration** → #2882 (monorepo mất memory).
- **Thất bại thực tế:** cold-start lần 3 (#2795, `7babbf0fb`, chỉ nâng timeout). Observer tạo hàng chục tệp `.jsonl` (#2909, `641ff245f`, `f278aec69`).
- **Phục hồi phạm vi:** "scope memories by platform source" (`348d9ee40`, 2026-06-29), tức bật lại cái đã gỡ ngày 04-20.

### E7 — Ba nhánh phát hành, revert danh tính, plan-14…24 (2026-07-01 → 2026-08-31)
- **Theo host:** gỡ Gemini CLI (`26fe40a6c`, 2026-07-03), thay bằng Antigravity (plan 2026-07-03).
- **Quy trình:** plan 3 nhánh `main`/`core-dev`/`community-edge` (`plans/2026-07-05-three-release-branches.md`). Có tag `archive/community-edge` (2026-07-09) và `archive/core-dev` (2026-10-11).
- **Danh tính:** "use canonical project identity" (`107ef946e`, 2026-07-05 01:27) bị revert lúc 15:24 cùng ngày (`77869056e`), thông điệp không nêu lý do (§3).
- **An toàn hook:** lỗi bất ngờ không còn chặn (exit 2 → 0) theo plan-17 #3161 (`hook-command.ts:151-160`).
- **Gom issue:** mở 10 issue tracker plan-14 → plan-24 (#3602–#3611, #3618) ngày 2026-08-16 (E0, trang #3608). Bản design doc vào repo ở `6c1ecb54a` (2026-09-12).

### E8 — "oh-my-issues" và merge dồn (2026-09-01 → 2026-10-02)
- Ngày 2026-09-30 có 169 commit (author date), PR từ #3203 tới #4310 vào main cùng lúc. Gồm phần lớn các bản sửa danh tính: #3311, #3525, #3643, #3536, #2827, #4291, #4297, #4303 (§3).
- **Thêm:** CLAUDE_PROJECT_DIR (`e2d309b37`, 2026-09-11). Cloud sync qua hub với outbox/rev (các op `remap_project`).
- Họ lỗi worker/process: 58 commit fix có từ khoá zombie/orphan/kill/reclaim/ghost/wedged/respawn/storm (§4).

### E9 — Liveness, spool bền, context cache, v14 (2026-10-03 → 2026-10-19)
- **Viết lại:** "Liveness over deadlines: stop guessing timeouts, stop paying twice" (`fe1709c30`, 2026-10-03):
  - SSE `/api/ready` thay cho việc poll;
  - `PaidSendBudget` (never pay twice);
  - stream request của OpenRouter/OpenAI-compatible;
  - track B: Supabase sync;
  - HookSpool bền và context cache trên đĩa (hai thứ xuất hiện đầu tiên trong `git log -i --grep='hook spool\|precomputed'`).
- **Thêm:** progressive memory search có khung "data, not instructions" (`5c125f3a1`, 2026-10-08). `CLAUDE_MEM_CONTEXT_PROJECTS=all` (opt-in, `9624d0a70`, 2026-10-19).
- **Sản phẩm:** v14.0.0 "My Work" và "TeamBrain" (trả phí theo seat), 2026-10-17 (CHANGELOG `:11`).

---

## 2. Trả lời các câu hỏi §8 (gom theo chủ đề)

| Câu hỏi | Trả lời (bằng chứng) |
|---|---|
| Mô hình ban đầu? | Script hook + chroma-mcp (E0) → observer LLM chạy nền + worker HTTP (E1) |
| Áp lực sinh kiến trúc kế? | Hook nặng, SQL trùng lặp → hook thành HTTP client (E3). Retry storm của hàng đợi bền → buffer RAM (E6). Đoán timeout sai → liveness/SSE (E9). Lỗi danh tính lặp lại → plan-20 (E7–E8, chưa xong) |
| Thêm gì? | Chroma, viewer, mode, Live Context, server/Postgres, cloud sync, spool, context cache, environment/git-remote, TeamBrain |
| Bỏ gì? | PM2 (`807d1d610`), hàng đợi bền/BullMQ engine (`61fe70a20`), Gemini CLI (`26fe40a6c`), lọc `platform_source` (`03748acd6`, sau bật lại), salvage summary (`d0fc68c63`), "defenders + tolerators" (`d13662d5d`) |
| Viết lại gì? | Toàn bộ ở v4. Hook ở v7. `worker-utils` ở `3030f518b` (2025-11-06). Đường thời gian chờ ở E9 |
| Cái gì thành chung hơn? | Read keys (`projectReadKeys`) và `buildProjectWhereFilter`, dùng chung cho mọi đường Chroma/SQLite (`84fc2d8f9`, `c29675b16`) |
| Cái gì thành riêng theo host? | Adapter cho từng host (`src/cli/adapters/` có 10 tệp: 7 adapter host + `codex-file-context`, `errors`, `index`). `CLAUDE_PROJECT_DIR` chỉ Claude Code dùng. Codex có ngân sách khởi động riêng và compaction gate. Cursor/Windsurf ghi tệp rules. DSH/OpenCode tiêm trong tiến trình |
| Thành async/bền? | Observer (E1). Spool hook bền (E9). Ngược lại, hàng đợi observation từ bền chuyển về RAM (E6) |
| Thành nhạy cảm an ninh? | Cloud sync/server (token, quarantine). Observer không được có tool (test an ninh). Tag `<private>`. Khung "untrusted" chỉ ở đường tìm kiếm (E9) |
| Thành đắt? | Một lời gọi LLM cho mỗi sự kiện tool (mặc định Claude batch=1). Có cả tier routing (E5) và quota fallback/gateway (2026-09-30) để bù |
| Hỏng thật ngoài đời? | Cold start (#1785/#2117/#2795). Mất memory do danh tính (#2882, #3437 "26% store gắn sai", #3531, #3641). Rò chéo dự án (#1911→#3764). Context bị host thay bằng stub (#3802). Hook chặn prompt (#2966/#3481/#3523) |
| Về sau phải đơn giản hoá? | Bỏ hàng đợi bền. "Cynical deletion". Revert canonical identity. Liveness thay hàng loạt timeout |

---

## 3. Lịch sử logic project identity (chú ý riêng)

Lệnh: `git log --follow -- src/utils/project-name.ts` (23 commit), `git log -- src/utils/worktree.ts`, `git log -S'show-toplevel'|-S'CLAUDE_PROJECT_DIR'|-S'COLLATE NOCASE'|-S'runOneTimeCwdRemap'`, `git log -i --grep='#<issue>'`.

| # | Ngày | Commit | Thay đổi luật | Issue | Loại sửa |
|---|---|---|---|---|---|
| 1 | 2025-10-16 | `7c5e5b194` | project = basename(cwd) ở context hook | — | — |
| 2 | 2025-12-17 | `bff10d49c` | tách `project-name.ts`: basename(cwd), ổ đĩa gốc → `drive-X` | #378 | LOCAL_PATCH |
| 3 | 2026-01-05 | `e1ab73dec` | `getProjectContext`: worktree → `allProjects=[parent, worktree]` | #556 | SHARED_ABSTRACTION |
| 4 | 2026-03-30 | `f86be1ef2` | mở rộng `~` | — | LOCAL_PATCH |
| 5 | 2026-04-15 | `16a0737df` | worktree GHI theo tên parent | #1820 | LOCAL_PATCH |
| 6 | 2026-04-16 | `040729bee`, `dc198d567` | composite `parent/worktree`; đọc gồm parent | — | đảo (5) |
| 7 | 2026-04-16 | `193e7e071`, `3d1dfcc26`, `a7c3c4af2` | remap cwd **một lần** (marker `.cwd-remap-applied-v1`) + `merged_into_project` + adoption | — | MIGRATION |
| 8 | 2026-05-28 | `daf6d9dc0` | **git toplevel** thay basename(cwd), không migration | #2663 (PR #2701) | đổi luật âm thầm → #2882 |
| 9 | 2026-07-05 | `107ef946e` → `77869056e` | "canonical project identity": khoá repo-relative cho thư mục con + override `.claude-mem.json`; **revert sau ~14 h** | — | ARCHITECTURAL (bị bỏ) |
| 10 | 2026-07-18 | `ec09f9db3` | worktree phát hiện từ toplevel, giữ composite khi đứng ở thư mục con | #3262 | LOCAL_PATCH |
| 11 | 2026-09-11 | `e2d309b37` | neo `CLAUDE_PROJECT_DIR` (chỉ Claude Code) | #3437/#4055 | LOCAL_PATCH ở biên adapter |
| 12 | 2026-09-30 | `e192903e8` | marker `.claude-mem-project`/`.claude-mem.json`, chỉ áp **ngoài git** | #3194 (#3311) | LOCAL_PATCH |
| 13 | 2026-09-30 | `f08f145fb` | submodule → `parent/<path>`; reconcile worktree/submodule mồ côi | #2842, #2864 (#3525) | MIGRATION |
| 14 | 2026-09-30 | `cc064d8c2` | gộp khoá worktree bị lặp `<repo>/<repo>` (Codex); tìm kiếm tôn trọng `merged_into_project` | #3641 (#3643) | LOCAL_PATCH + alias |
| 15 | 2026-09-30 | `411093dfc` | so khoá `COLLATE NOCASE` | #3531 (#3536) | SHARED (đọc) |
| 16 | 2026-09-30 | `276f845d2` | opt-in `git-remote` slug `org/repo` | #2827 | luật mới (opt-in) |
| 17 | 2026-09-30 | `5beeceedd` | `keySource`: chỉ tin "checkout đã xoá" với khoá dẫn từ đường dẫn | #4291 | MIGRATION guard |
| 18 | 2026-09-30 | `c29675b16`, `84fc2d8f9` | đọc **mọi** khoá của checkout (gate P2-5); một scope NOCASE cho mọi đường Chroma | #4303 | SHARED (đọc) |
| 19 | 2026-09-30 | `5d454a749` | adoption chạy ngoài event loop; `project merge` chạy trong worker | #4297 | — |
| 20 | 2026-10-01 | `a27e248e3` | environment đặt tên theo glob đường dẫn, thắng mọi khoá dẫn xuất | #2737 | luật mới (opt-in) |
| 21 | 2026-10-16 | `137360788` | placeholder `'unknown'` khi payload không có cwd (tránh sync quarantine) | #4772 (#4774) | LOCAL_PATCH |
| 22 | 2026-10-16 | `2b4bb4377` | chỉ merge trọn mới nâng thành read key (chặn lộ từ adoption một phần) | #4782 (#4789) | SHARED (đọc) |
| 23 | 2026-10-19 | `9624d0a70` | `CLAUDE_MEM_CONTEXT_PROJECTS=all` (SessionStart chéo dự án, opt-in) | #4817 | luật mới (opt-in) |

Đếm tay từ bảng (không phải lệnh):
- Luật **mặc định** của khoá ghi đổi 6 lần: (1), (3), (5→6), (8), (11), (12/13/14).
- Có 4 luật opt-in chồng lên: (16), (20), (23), và marker (12).
- 1 lần thử hợp nhất bị revert: (9).
- Phía đọc nở dần thành "tập khoá": (6), (15), (18), (22).

Nhận xét (suy luận):
- Mỗi bản sửa vá một bố cục cụ thể (monorepo, worktree, submodule, Codex, hoa/thường, ngoài git, CLAUDE_PROJECT_DIR) ở đúng chỗ phát hiện. Không bản nào xoá được các quy tắc dẫn xuất song song đã liệt kê ở `01` §4.2 (A3–A8).
- Upstream chọn cách **hợp nhất lúc đọc**: alias + `merged_into_project` + NOCASE + read keys. Họ không chọn **một khoá chuẩn lúc ghi**.
- Lần duy nhất thử khoá chuẩn ở lúc ghi (9) lại đổi khoá mặc định của mọi thư mục con, và bị gỡ trong ngày.
- #3608 vẫn OPEN. Bước 1 (resolver duy nhất) và bước 5 (doctor so khớp đọc/ghi) chưa có ở HEAD (`01` §4.4).
- Lịch sử nhánh có thêm các thử nghiệm danh tính không merge, ví dụ:
  - `origin/fix/dynamic-project-name-detection` (2025-11-20)
  - `origin/copilot/fix-default-project-name` (2025-12-14)
  - `origin/branch-memory` (2026-03-16)
  - `origin/fix/worktree-sweep-key-derivation` (2026-09-30)
  - `origin/fix/sync-pull-identity-collision` (2026-10-02)

  Cả 5 cho kết quả `NOT_MERGED` với `git merge-base --is-ancestor <branch> HEAD`. Riêng `origin/thedotmack/worktree-remap` đã merge. Đào sâu để pha fork sau.

---

## 4. Họ lỗi lặp (sửa → hồi quy → sửa)

| Họ | Chuỗi | Bằng chứng | Hiện trạng |
|---|---|---|---|
| Cold start / race SessionStart | #1785 → #2117 → #2795 (13.4.0) → SSE `/api/ready` + cache | tiêu đề #2795 "Regression of #1785/#2117"; `7babbf0fb` không test; `fe1709c30` | NOT_READY vẫn gộp với FAILED với model (`01` §5) |
| Rò chéo dự án khi thiếu project | #1911 sửa (`ba1ef6c42`, 2026-04-20) → #3764 (2026-08-27, "regression of #1911") → đóng not planned | WebFetch #3764; `mcp-server.ts:646,665,685` | CÒN (E1) |
| Phạm vi `platform_source` | thêm `2b60dd293` (03-24) → gỡ `03748acd6` (04-20) → thêm lại `348d9ee40` (06-29) → cấu hình được `0d825988e` (09-30) | `git log -S'platformSource' -- src/cli/handlers/context.ts` | dao động 3 lần |
| Worktree | ghi theo parent (04-15) → composite (04-16) → thư mục con (#3262) → khoá lặp (#3641) → reconcile (#3525) → `keySource` (#4291) | §3 | còn khác nhau giữa A1 và A3 (bare repo, submodule) |
| Tiêm SessionStart khi resume | bật cho mọi nguồn (`f499810c7`, 2025-10-25) → #4061 bắn đôi khi resume → matcher bỏ `resume` | `hooks.json:33`; `context.ts:180-183` | FIXED (E1) |
| Hook chặn người dùng | exit 2 chặn prompt/Read/Stop (#2966, #3481, #3523) → plan-17 #3161 không bao giờ chặn | `worker-utils.ts:2182-2184`; `hook-command.ts:151-160` | FIXED (E1) |
| Process/zombie/reclaim | 58 commit fix khớp `^fix.*(zombie\|orphan\|kill\|reclaim\|ghost\|stale worker\|wedged\|respawn\|restart storm\|retry storm)` (2026-02 → 2026-10-19, lần gần nhất `cf0c7cdcc` #4868 vào chính ngày HEAD) | lệnh trên `git log --format=%s` | còn đang sửa |
| Giao context (kích thước/hợp đồng) | #1899/#3802 (vượt 10.000 ký tự → stub, "hook vẫn báo thành công"); #2547/#2972 (Codex JSON) | `ContextBudget.ts:3-9`; `hook-command.ts:34-48` | đã có test #3802 (E2) |

Số họ đếm thô theo subject (regex, có thể đếm dư hoặc thiếu):
- cold/race: 30 commit, `^fix.*(cold|race|before init|not ready|readiness)`;
- project/worktree identity: 29 commit, `^fix.*(project name|project key|project-name|worktree|project identity|submodule|merged_into)`.

---

## 5. Abstraction bị xoá · "tạm" thành vĩnh viễn · đi vòng kiến trúc · test sau hồi quy

- **Abstraction bị xoá:**
  - Hàng đợi bền + BullMQ engine + RestartGuard (`61fe70a20`);
  - PM2 (`807d1d610`);
  - `src/hooks/*` cũ (`601596f5c`);
  - adapter Gemini CLI (`26fe40a6c`);
  - canonical identity (`77869056e`);
  - test của các phần đã xoá (5 tệp trong `61fe70a20`).
- **"Tạm" thành vĩnh viễn:**
  - Remap cwd "one-time" (marker `-v1`, 2026-04-16) vẫn gọi mỗi lần boot ở HEAD (`worker-service.ts:749`), song song với sweep adoption (`:781`).
  - Đọc `pending_messages` "only for rows still on disk" sau khi v13.10.0 ngừng ghi (`WorktreeAdoption.ts:713-714`).
  - Literal `'server-beta'` và các khoá `CLAUDE_MEM_SERVER_BETA_*` (`runtime-selector.ts:41-44,62-73`).
  - `getProjectName` vẫn export nhưng `src` không còn gọi, chỉ test dùng (`project-name.ts:269`).
  - Nhãn `error_mode: 'blocking_error'` giữ lại để chuỗi telemetry liên tục dù hook không còn chặn (`hook-command.ts:159-166`).
- **Đi vòng kiến trúc đã quảng bá:** "hooks are HTTP clients, worker là nguồn sự thật duy nhất" (v7), nhưng:
  - runtime server gọi thẳng từ hook, không qua worker (`context.ts:93-101`);
  - DSH/transcript watcher/ingest memory tự suy project (`01` §4.2, A4–A5);
  - MCP `search` ở runtime server có lọc thì đọc SQLite local (`mcp-server.ts:617-646`).
- **Test chỉ thêm sau hồi quy:**
  - Bảng ở §0: 2025 có 4/148 fix kèm test.
  - Các block test danh tính đặt tên theo issue (`tests/utils/project-name.test.ts:93,133,268,412,460,593`).
  - Bản sửa cold-start #2795 không có test.
  - Test bảng 5 resolver mà plan-20 tự yêu cầu thì chưa có.

---

## 6. Ref bị nhiều commit fix nhắc (ứng viên bug lặp, overlay mục 17)

Từ `issue-refs.tsv`, quy tắc chặt: ref nằm trong SUBJECT hoặc ở dạng closes/fixes của ≥2 commit fix, không tính số PR của chính commit. Có 18 ref (cách lỏng, tính cả body: 208, nhiễu rất cao vì body hay liệt kê hàng loạt ref):
- #1641 (3)
- #4745, #4682, #4215, #4069, #3926, #3917, #3678, #3609, #3362, #3078, #3070, #2302, #984, #531, #527, #517, #511 (mỗi ref 2)

Lọc tay:
- **Review vòng lặp của cùng một PR, không phải bug lặp:** #1641, #2302, #3070, #3078 ("address PR review", "Greptile #…").
- **Một commit gộp và bản squash của nó:** #511/#517/#527/#531.
- **Ứng viên thật:**
  - **#4215** — Qwen `submitted_prompt`: `3b2e5829a` 2026-10-01, rồi `fa8453e62` 2026-10-05 "skip Qwen sends that omit submitted_prompt entirely". Đây là sửa → sửa tiếp cùng lớp "có mặt ≠ giá trị thật" (`claude-code.ts:21-37`).
  - **#3609** — plan-21 schema: slice A/B.
  - **#3362** — Chroma: pin + doctor.
  - **#3678** — OpenCode: 2 lát.
  - **#3926** — tree-sitter: fix, rồi fix test CI.

Các bug lặp lớn nhất (cold start, rò chéo dự án, platform scope) **không lộ ra** bằng cách đếm ref, vì chúng mang số issue khác nhau ở mỗi lần (§4). Đếm ref chỉ là chỉ báo yếu.

---

## Hợp đồng đầu ra (spec §63)

- **SCOPE:** các thời kỳ tiến hoá, lịch sử logic project identity, họ lỗi lặp, abstraction bị xoá, ref lặp.
- **REVISION AUDITED:** `3fae1eb45ba1a7fbd4818527f48b439991a273da`.
- **FILES/COMMITS/ISSUES INSPECTED:**
  - `git log` toàn bộ (subject + body), `CHANGELOG.md` (đầu mỗi bản major), `plans/20-*.md`, `plans/2026-07-05-three-release-branches.md`.
  - `git show` các commit `107ef946e`, `77869056e`, `daf6d9dc0`, `61fe70a20`, `fe1709c30`, `7babbf0fb`, `e1ab73dec`, `03748acd6`, `ba1ef6c42`, `d13662d5d`, `641ff245f`, `f278aec69`.
  - Các nhánh remote danh tính nêu ở §3.
- **VERIFIED FACTS:** các số ở §0 (lệnh vừa chạy); các dòng commit/ngày ở §1–§5 (`git log`/`git show`).
- **INFERENCES:** lý do revert canonical identity (đổi khoá mặc định của thư mục con); "hợp nhất lúc đọc" là chiến lược thực tế của upstream; phân loại LOCAL_PATCH/SHARED/ARCHITECTURAL.
- **UNKNOWNs:** lý do thật của revert `77869056e`; nội dung 763 nhánh remote (để pha fork); vì sao 2026-09-30 có merge dồn (giả thuyết: gom backlog oh-my-issues); bản sửa trên `core-dev`/`community-edge` có khác `main` không.
- **FAILURE MODES:** STARTUP_RACE lặp; WRONG_SCOPE_RETRIEVAL hồi quy; scope dao động (platform_source); PROJECT_IDENTITY_FAILURE vá cục bộ nhiều lần; RETRY_STORM (lý do xoá hàng đợi bền); ZOMBIE_OPERATION đang sửa tới ngày HEAD.
- **AGENT CLI RELEVANCE:** câu hỏi kiểm cho pha sau:
  1. Agent CLI có chỗ nào cũng có "luật danh tính" đổi theo thời gian mà không có migration/alias không?
  2. Có scope nào bật/tắt qua lại không?
  3. Có fix nào chỉ nâng timeout không?

  Pha này KHÔNG kết luận phơi nhiễm.
- **PROPOSED TESTS:**
  1. Đổi luật danh tính phải kèm test "khoá cũ vẫn đọc được".
  2. Test chống dao động: scope theo host/session luôn có test giữ chiều.
  3. Cold start phải có test, không chỉ hằng số timeout.
- **NO-CHANGE FINDINGS:** không có danh tính branch trên main (thử nghiệm `branch-memory` không merge); không có namespace dream trong lịch sử `src`.
