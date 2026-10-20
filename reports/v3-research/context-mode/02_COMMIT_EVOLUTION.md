# Context Mode — 02 Tiến hoá qua commit

- **REVISION AUDITED:** `main` @ `6006a351` (2026-10-18). Repo clone đầy đủ (`git rev-parse --is-shallow-repository` = false).
- Số liệu (lệnh chạy 2026-10-19 trong clone):
  - `git rev-list --count HEAD` = **2239** commit trên `main`; 165 merge; 2074 non-merge; 557 commit "ci: update …" (555 của
    `github-actions[bot]`).
  - **198** tag, từ `v0.6.0` (2026-02-24) tới `v1.0.169` (2026-06-29); 168 tag từ `v1.0.0` (2026-03-05) trở đi.
  - Commit theo tháng (`git log --format=%ad --date=format:%Y-%m | sort | uniq -c`): 02: 97 · 03: 544 · 04: 412 · **05: 740** ·
    06: 268 · 07: 53 · 08: 51 · 09: 41 · 10: 33. Từ 2026-06-30 tới HEAD: 179 commit, **tất cả** là "ci: update install stats".
  - Loại commit non-merge (tiền tố subject, bỏ "ci: update"): fix 664 · feat 242 · khác 195 · docs 108 · test 104 · chore 87 ·
    refactor 43 · perf 12. Revert thật (subject bắt đầu "Revert"/"revert:"): **9**.
  - Tác giả non-merge (`git shortlog -sn --no-merges HEAD`): Mert Koseoglu 1201 (+13 dưới tên "Mert Köseoğlu"), bot 555,
    còn lại ≤ 33/người.
- Nhánh khác: `origin/next` (11 commit trước `main`, tới 2026-09-28); `origin/v2-design` (737 commit không có trên `main`,
  2026-10-14 → 2026-10-19, kèm ~280 nhánh `impl/*`, `wip/*`).
- Quy mô theo tag (đếm bằng `git ls-tree` + `git show | wc -l`):

| Mốc | Ngày | Tệp test | Dòng `.ts` trong src | Hook `.mjs` | Thư mục adapter |
|---|---|---|---|---|---|
| `60a1a254` (đầu) | 02-23 | 2 | 1 691 | 0 | 0 |
| `v0.6.0` | 02-24 | 3 | 2 505 | 0 (hook bash `hooks/task-inject.sh`) | 0 |
| `v1.0.0` | 03-05 | 32 | 10 074 | 22 | 5 |
| `v1.0.30` | 03-19 | 39 | 14 875 | 30 | 10 |
| `v1.0.76` | 04-13 | 44 | 16 134 | 37 | 10 |
| `v1.0.112` | 05-10 | 103 | 27 113 | 53 | 14 |
| `v1.0.152` | 05-31 | 167 | 35 440 | 63 | 15 |
| `v1.0.169` | 06-29 | 210 | 41 617 | 79 | 17 |

## Thời kỳ 0 — MCP thuần, cắt ngắn output (2026-02-23)

- **Mô hình ban đầu** (`60a1a254`, v0.4.0): 5 tool (`execute` 10 ngôn ngữ, `execute_file`, `index`, `search`,
  `fetch_and_index`), FTS5 BM25, `SKILL.md` hướng dẫn model; executor **smart truncation** 100 KiB, giữ 60% đầu / 40% cuối
  (`git show 60a1a254:src/executor.ts` dòng 30-39, 140-178). Không hook, không session. Giả định: model tự chọn tool đúng nhờ
  skill.
- Trong ngày: intent-driven search (`15f00d8d`), source-scoped search "to prevent cross-source contamination" (`fc6c3401`).

## Thời kỳ 1 — Ép buộc bằng hook (2026-02-24 → 03-03)

- **Áp lực:** model không theo skill; subagent "inherit MCP tools but NOT skill instructions" (`9941db01`, 02-24 — giả định gốc
  của mọi lỗi subagent về sau).
- **Thêm:** PreToolUse cho Task (tiêm routing vào prompt subagent, `9941db01`/`9b45392a`), whitelist Bash (`eefacd9b`), hook
  PreToolUse hợp nhất chặn Bash lấy dữ liệu (`fbf402fd`), chặn WebFetch (`cc6d97a6`), batch_execute + throttle search
  (`103b41dd`), **ngân sách 500 từ cho output subagent** (`c0e93adb`, v0.7.3), hook bash/jq → Node (`4511152b`, 03-01),
  SessionStart tiêm khối XML (`e19bf0cc`, 03-02), tiền tố `MANDATORY` trong mô tả tool (`906989e4`, 03-02), gom văn bản về
  `routing-block.mjs` (`d32a5aeb`, 03-03), enforcement `permissions.deny` cho tool sandbox (`b3f93eba`, 03-03).
- **Bỏ:** test subagent-budget bị gỡ khỏi CI ngay khi đổi runtime hook (`d521587f`, 03-01: "requires hook script").
- **Trở thành:** giọng mệnh lệnh (MANDATORY, XML) là lựa chọn có chủ đích — sau thành gốc của #946/#967 (thời kỳ 7).

## Thời kỳ 2 — Session continuity + đa host (2026-03-04 → 03-21)

- **Áp lực:** compact làm mất trạng thái; người dùng các host khác (#46).
- **Thêm:** capture sự kiện + "zero-truncation injection" (`af03f710`, 03-04); adapter pattern (`49e0d7f1`, 03-05); v1.0.0
  (`31f96c01`, 03-05); background mode `ctx_execute` (`117e15f2`); dedup FTS theo nhãn (`80c07c7d`, #67 — nguồn của F1 ở 01);
  Cursor/OpenClaw/Kiro/Zed/Antigravity adapter; worktree suffix cho SessionDB (`6643f617`, 03-15, #128); MCP `clientInfo`
  detection (`b40d6bdb`, 03-16); RRF thay fallback tầng (`bce072b5`, 03-18); TTL cache fetch 24 h (`8d93b646`); ContentStore
  bền theo dự án (`4d7a2f4d`, 03-19); tên tool theo host (`2b5cfef7`, 03-21).
- **Abstraction bị xoá:** monorepo 3 package `packages/{shared,session,core}` tạo `649d2ab8` (03-04) rồi **xoá ngay hôm sau**
  (`49e0d7f1`: "Migrate from 3-package monorepo … to flat src/").
- **"Tạm" rồi bị gỡ:** tự ghi tệp routing (AGENTS.md/CLAUDE.md/GEMINI.md) vào cây dự án người dùng cho host không hook
  (`55c0b863`, 03-06) → tắt (`2b5cfef7`, 03-21, #158 #164) → xoá khỏi interface (`6dae20c9`, 04-02, #195: "dirtied user
  workspaces … OpenCode loaded AGENTS.md as active instructions, mutating user source of truth").
- **An ninh nới ra:** env executor allowlist 80 biến → denylist 27 biến (`0a8bb7b7`, 03-15) để hết vòng PR thêm biến
  (#22, #29, #86) — từ đây secret trong env cha đi vào tiến trình con (xem 01 mục 8).
- **Native module bắt đầu xoáy:** better-sqlite3 ABI cache (`c85a11cf`, 03-19), fallback `bun:sqlite` (`e4cf8a67`, #45),
  #163 bốn commit (03-21/22). Tiếp: `node:sqlite` vì SIGSEGV (`302bb7b8`, 04-13, #228), re-exec bằng Bun trên Linux
  (`f985afa0`, 05-14, #564). Upstream v2 định bỏ `bun:sqlite` (D2, E0).

## Thời kỳ 3 — "Không cắt, index hết" + MANDATORY (2026-03-22 → 04-13)

- **Áp lực:** #197 — smartTruncate "discarding the middle of large outputs BEFORE FTS5 indexing, making that data permanently
  lost and unsearchable" (`ee628da5`, 04-02).
- **Viết lại:** bỏ smartTruncate, output > 100 KiB tự index và trả con trỏ (`ee628da5`); dọn nốt `maxOutputBytes`
  (`a6a1fc06`); snapshot "reference-based", **không ngân sách byte**, lưu FULL event data (`a0b53c0f`, 04-04). Comment cũ
  "executor stdout truncation (100KB limit)" còn sót ở mã fetch (`server.ts:3002`). Upstream sau đó có #1022 "Resume snapshot
  ignores maxBytes (196KB)" (E0) — cái giá của "zero truncation".
- **Thêm:** "Think in Code — MANDATORY paradigm across all 12 platforms" (`071ded7a`, 04-04), `ctx_purge`, DB nội dung tách
  theo host (`bfd7a95c`), cảnh báo phiên bản cũ cho mọi host (`e2a5ce1d`).
- **Thêm rồi revert cùng ngày:** `inheritEnvKeys` (`4b88a82c` → `a2e9c719` 04-04: "unverified claim (#226)").
- **MCP readiness ra đời:** redirect làm agent kẹt khi server chưa lên ⇒ sentinel + PID (`04569b12`, 04-13, #230), sau một
  bản sửa bị revert (`b4edff18` → `415ce57d`). Đây là vòng 1/4 của lỗi "có sẵn sàng không" (xem bug lặp).
- **#233** (`245cd8af`, 04-13): bỏ `<ctx_commands>` khỏi khối subagent.

## Thời kỳ 4 — Bung host + vá tầng cài đặt (2026-04-14 → 05-31)

- **Áp lực:** 14→15 host, cài qua marketplace/npm/bun, Windows; tháng 5 là đỉnh (740 commit).
- **Thêm:** BaseAdapter (`c6cc02ac`, 04-25); JetBrains, OMP, Pi, Kimi adapter; per-event project attribution (#325);
  stats/insight/analytics nhiều đợt; concurrency opt-in (`b392c2fe`, 05-02); "algo defense 1..6" cho plugin cache
  (05-13); kill sibling MCP + lockfile một writer (#559/#560, 05-14); route MCP ngoài qua PreToolUse cho 6 host (#529, 05-12);
  `ctx_index` thư mục (#687); bridge sự kiện ra platform (thời kỳ 5).
- **Tiêm phong cách → gỡ:** "caveman-style response rules across all 14 adapters" thay 500-word limit (`7e35caf4`, 04-25) →
  **gỡ toàn bộ** 22 chỗ (`d05a3758`, 05-10, #482: ép ngắn làm model "drops assumptions, caveats, verification evidence, failure
  modes, and security warnings"; phương án "config switch" bị bác vì "switches default-on become dead code").
- **Vá cài đặt dạng nhiều mảnh:** #523 (8 mảnh, 05-11), #531 (10 mảnh + "fix-of-fix", 05-12), #545 env-leak (7 mảnh, 05-12),
  #542 detect (6 mảnh), #535 i18n (merge rồi **revert** `bad8c9ea`, làm lại theo "structural rule").
- **Thêm rồi revert:** lifecycle idle-shutdown (#568, 05-14; #595, 05-17) → revert "restore tools on 12 hosts" (`a888cac6`,
  05-18, #602) — server tự tắt khi rảnh làm mất tool ở 12 host.
- **Lỗi im lặng nghiêm trọng:** `4bc292fc` (05-31): trên Claude Code, redirect Bash dạng allow+`updatedInput.command` "silently
  no-op'd in production: users ran `curl https://example.com`, got the raw HTML" — test cũ khẳng định đúng hình dạng JSON nên
  vẫn xanh; phải re-pin 5 case integration. Hai bản formatter Claude Code song song phải vá cùng lúc.
- **Bản sửa một host, tám host không:** `4164d3de` (05-10) bảo vệ sự kiện UUID khỏi DELETE mồ côi chỉ trong hook Claude Code
  (01 F5).
- **#724** (`b1ba5cc9`, 05-31, PR #725, contributor): bootstrap ToolSearch cho subagent Claude Code.

## Thời kỳ 5 — Platform/telemetry, vá subagent (2026-06-01 → 06-29)

- **Thêm:** `hooks/platform-bridge.mjs` gửi sự kiện tới `platform_url` khi có `~/.context-mode/platform.json` (`043c38f8`,
  06-01, opt-in); "worktree-invariant project identity in bridge" (`13bfdd3d`); bắt token/cost theo lượt cho 8+ host
  (06-26); trang landing có Pricing (06-06).
- **Subagent/capability:** #794 qua PR #834 (`85f64d2c`, 06-21, contributor): `mcpToolsAvailable=false` khi payload có
  `agent_id`/`agent_type`; **gỡ** env `CONTEXT_MODE_DISABLE_AGENT_INJECTION` vì "gates no remaining behavior". #769 flood-guard
  theo agent (`3a45eb1c`, 06-21).
- **MCP readiness vòng 4:** #844 PID namespace (`ff5b0cff`, 06-22): sentinel mới < 90 s coi là sống.
- **An ninh:** `ctx_execute_file` giới hạn trong project root + tiêu đề phê duyệt "Run code…" (#852).
- **Phát hành cuối:** `v1.0.169` (`589d8214`, 06-29).

## Thời kỳ 6 — `main` đóng băng (2026-06-30 → 2026-10-18)

- 179 commit, toàn bộ bot "ci: update install stats". `origin/next` có thêm 11 commit (site chuyển repo khác, 09-28).
- Suy luận: phát triển chuyển sang thiết kế lại (thời kỳ 7); các issue số > ~900 được xử lý trên giấy ở nhánh v2, không ở `main`.

## Thời kỳ 7 — Thiết kế lại v2, CHƯA merge (2026-10-14 → 10-19, nhánh `origin/v2-design`)

Chỉ là E0 (ý định), dẫn để biết hướng upstream tự sửa:
- 18 ADR (`docs/v2/adr/0001…0018`), bản đồ issue 351 dòng (`tickets/v2-issue-map.md`; commit `e4873926`: "every issue-map
  row has a test verdict (351/351: GP 170, LIVE 2, REMOVED 110, N/A 69)").
- **Gỡ tiêm routing:** ADR-0004 "no skills/mcp instructions", D11 "block removed"; #946/#967/#1034/#1272 đóng bằng "REMOVED".
- **Deny cứng có ngân sách byte** (ADR-0008), "deny unranged Read of files over 16 KB" (DECISIONS-r8, `9e3f0165`).
- **Subagent theo capability:** ADR-0016 ba trường hợp A (có ctx) / B (có Bash) / C (không có cả hai) — tool-set của subagent
  đọc từ frontmatter/`--agents`/manifest; "until LV8 runs, a built-in type is treated as having no ctx tools".
- **Một DB** (ADR-0002), test chỉ E2E/golden (ADR-0003), không gate theo trạng thái server (P2: `isMCPReady` biến mất).
- Mã v2 nằm ở `src/core`, `e2e/golden`, `tests/v2` (`git diff --name-only main origin/v2-design`); mã v1 trong `hooks/` và
  `src/server.ts` chưa đổi trên nhánh đó.

## Trả lời các câu hỏi §8

| Câu hỏi | Trả lời (có bằng chứng) |
|---|---|
| Mô hình ban đầu? | MCP 5 tool + skill + cắt ngắn 60/40 (thời kỳ 0). |
| Áp lực nào dẫn tới kiến trúc sau? | Model không theo skill → hook (TK1); compact mất trạng thái + đa host → session/adapter (TK2); mất dữ liệu do cắt (#197) → index hết (TK3); cài đặt/host vỡ hàng loạt → vá nhiều mảnh (TK4); subagent kẹt (#724, #794) → special-case (TK4-5); classifier/overhead (#946, #967, #233) → v2 gỡ tiêm (TK7). |
| Thêm gì? | hook 6 loại, SessionDB, 17 adapter, RRF/trigram/fuzzy, TTL cache, security deny, stats/insight, platform bridge. |
| Bỏ gì? | smartTruncate (TK3), monorepo (TK2), tự ghi routing file (TK2→TK4), 500-word + caveman (TK1→TK4), inheritEnvKeys (TK3), idle-shutdown (TK4), env `CONTEXT_MODE_DISABLE_AGENT_INJECTION` (TK5), Task matcher → Agent (#241). |
| Viết lại gì? | executor output path (TK3), snapshot (TK3), hook bash→Node (TK1), formatter Claude Code Bash modify→deny (TK4). |
| Tổng quát hơn? | `routePreToolUse` + formatter registry chung (TK2-3), BaseAdapter (TK4). |
| Riêng host hơn? | bootstrap ToolSearch chỉ Claude Code; `mcpToolsAvailable` chỉ Claude Code; codex-caps theo `codex --version`; agy biến nhắc thành deny; OpenCode chống trùng; timeout mặc định chỉ agy (01 mục 9). |
| Bất đồng bộ? | concurrency fetch/batch (TK4); platform bridge fire-and-forget (TK5). |
| Bền vững hoá? | ContentStore theo dự án (TK2), SessionDB 7 ngày, stats persist (TK4). |
| Nhạy cảm an ninh? | env denylist (TK2), security bundle fail-open/fail-closed (#558, #468), SSRF (TK4), #852, bridge gửi ra ngoài (TK5). |
| Đắt lên? | hook mỗi tool call (upstream #1261 "55-70ms CPU per Bash", E0; test "<3ms p95" `fafef116` chỉ đo routing thuần), khối 4.6 KB mỗi Agent (01 mục 3), snapshot không ngân sách. |
| Hỏng ở production? | redirect Bash no-op trên Claude Code (`4bc292fc`); mất >1000 sự kiện sau mất điện (`4164d3de`); tool biến mất ở 12 host do idle-shutdown (`a888cac6`); subagent kẹt (#724, #794/#832). |
| Phải đơn giản hoá về sau? | monorepo, tự ghi routing file, tiêm phong cách, v2 bỏ cả routing block và `isMCPReady`. |

## "Tạm" thành vĩnh viễn

- **Self-heal trong mỗi PreToolUse:** khối sửa thư mục cache/`installed_plugins.json`/`settings.json` chạy ở hook mọi tool call
  (chặn bằng marker `/tmp/context-mode-healed-<ver>`), `hooks/pretooluse.mjs:44-155`; phần phá huỷ đã gỡ (#415) nhưng khối vẫn
  ghi cấu hình người dùng. Cộng `hooks/heal-partial-install.mjs` (712 dòng), `hooks/cache-heal-utils.mjs` (379 dòng). Các họ
  #181, #411, #523, #531, #604, #620, #814 là nhiều vòng vá cùng lớp "đường dẫn cài đặt trôi".
- **Ước lượng byte cố định** 8192/16384 dùng cho thống kê "bytes avoided" (`routing.mjs:779,886`).
- **Static exports "backward compat"** của routing block (`routing-block.mjs:98-105`) và sentinel PPID "@deprecated … one release
  cycle" (`mcp-ready.mjs:49-56`) vẫn còn ở HEAD.

## Test chỉ thêm sau hồi quy (mẫu)

`git log -i --grep=regression` (bỏ bot) = 157 commit nhắc "regression". Ví dụ có diff test:
- `4bc292fc` — test formatter + 5 case integration re-pin sau lỗi no-op production.
- `4164d3de` — `tests/session/cleanup-preserves-live-uuid-events.test.ts` sau sự cố mất điện (chỉ cho hook Claude Code).
- `f4e8d2ac` (05-02) — "cross-platform regression matrix for MCP readiness (#354)" sau #347.
- `a9a2b85c` (05-12) — "15x14x3 leak-matrix regression — MUST-3 invariant" sau #545.
- `139431d2` (05-31) — "cross-adapter hook path parity guard (#712)".
- `53899008` (05-11) — regression guard toán tử ghép cho SAFE_COMMAND_PATTERNS (#517).
- `85f64d2c` — test subagent (#794), đồng thời xoá test chết của #832.

## Bug lặp (ứng viên, từ `issue-refs.tsv`)

`issue-refs.tsv` có 1075 dòng (ref × commit), 501 ref khác nhau, 677 commit nhắc ref; **106 ref** xuất hiện ở ≥ 2 commit
fix có subject khác nhau. Phần lớn là bản sửa nhiều mảnh của cùng một issue ("partial — k of n"), không phải hồi quy. Các họ
có dấu hiệu **sửa → hỏng lại → sửa** (đọc subject + diff ở các commit dẫn):

| Họ | Chuỗi | Ghi chú |
|---|---|---|
| MCP readiness | #230 (04-13, kèm 1 revert) → #347 (04-26, PPID→quét thư mục) → #354 (ma trận hồi quy) → #844 (06-22, PID namespace) → upstream #1055 "isMCPReady counts sibling sessions" (E0) | 4 vòng; v2 bỏ hẳn cơ chế (P2) |
| Vòng đời MCP server | #103 (03-13) → #236 (04-13, sửa rồi **revert**) → #311 (04-25) → #388 (05-02) → #565/#568/#595 (05-14..17) → **revert #602** (05-18) → #854 (06-22) → #862 (06-22); upstream #999 "start.mjs busy-loop, 23 instances" (E0) | bỏ-thêm-bỏ |
| Đường dẫn cài đặt | #181, #253, #411, #415, #523, #531 (+"fix-of-fix" `85188304`), #604, #620, #636, #814 | 110 fix khớp regex install/upgrade/heal |
| Danh tính dự án | #45, #521, #542, #545, #561, #645, #737; revert `16d7b34f`/`38aa9244` ("default projectDir to cwd") | 68 fix khớp regex project-dir/env-leak/workspace/cwd/detect |
| Subagent/routing | #233 → #724 → (#1034 do bản sửa #724 gây ra, E0) ; #794 → #832 → #834 → (#1037, E0) | sửa bằng special-case, mỗi bản sửa sinh issue mới |
| SQLite native | #45, #163, #215, #228, #564 | đổi driver 3 lần |

Đếm fix theo từ khoá trên 665 subject fix (không loại trừ nhau; lệnh `git log --no-merges --format=%s main | grep -E '^fix'`
rồi `grep -ciE <regex>`): install/upgrade/heal… 110 · stats/analytics/insight/cost 91 · project-dir/env-leak/workspace/cwd/detect
68 · windows/msys/cmd 64 · sqlite/abi/native 49 · security/deny/ssrf/permission 28 · subagent/agent_id/routing 22 ·
lifecycle/orphan/idle/shutdown 21.

---

## Hợp đồng đầu ra (spec §63)

**SCOPE** — Thời kỳ tiến hoá của Context Mode theo `git log` trên `main` (+ ghi chú E0 từ `origin/v2-design`); đọc message và
diff các commit then chốt. Không đào trang issue (404), không đào fork.

**REVISION AUDITED** — `main` @ `6006a3511ea2afde3e413cf635ba1da85ac8a8aa`; `origin/v2-design` @ `7a99ece2`; `origin/next` @ `46213649`.

**FILES/COMMITS/ISSUES INSPECTED** — Commit (message, nhiều cái cả diff): `60a1a254`, `9941db01`, `fbf402fd`, `c0e93adb`,
`d521587f`, `4511152b`, `e19bf0cc`, `906989e4`, `649d2ab8`, `49e0d7f1`, `af03f710`, `80c07c7d`, `55c0b863`, `2b5cfef7`,
`6dae20c9`, `0a8bb7b7`, `ee628da5`, `a6a1fc06`, `a0b53c0f`, `071ded7a`, `4b88a82c`, `a2e9c719`, `04569b12`, `415ce57d`,
`245cd8af`, `7e35caf4`, `d05a3758`, `4164d3de`, `a888cac6`, `2e44cf6b`, `4bc292fc`, `b1ba5cc9`, `85f64d2c`, `3a45eb1c`,
`ff5b0cff`, `043c38f8`, `589d8214`; tệp `docs/v2/adr/0016…`, `tickets/v2-issue-map.md`, `docs/v2/review/ISSUE-OUTCOMES-r4.md`
(nhánh v2-design). Issue chỉ qua commit/doc.

**VERIFIED FACTS** — số commit/tag/tháng/loại/tác giả ở đầu tệp (lệnh đã ghi); bảng quy mô theo tag; 9 revert; `main` chỉ có
commit bot từ 2026-06-30; các pivot đã dẫn commit (bỏ truncation, gỡ tiêm phong cách, gỡ tự ghi routing file, monorepo tồn tại
1 ngày, allowlist→denylist, idle-shutdown revert, Bash modify→deny no-op production, fix orphan chỉ 1 host).

**INFERENCES** — phát triển chuyển sang v2; "zero truncation" dẫn tới snapshot quá cỡ (#1022 là E0); mỗi special-case
subagent sinh issue mới (#1034, #1037 là E0, chưa đọc issue).

**UNKNOWNs** — nội dung, ngày, trạng thái GitHub của các issue (404); v2 có được merge không, khi nào; số người dùng chịu
lỗi no-op `4bc292fc` (commit chỉ nói "every bun-installed Claude Code user").

**FAILURE MODES** — FALSE_SUCCESS (enforcement no-op, test theo hình dạng), PARTIAL fix (1/9 host), ZOMBIE_OPERATION /
STARTUP_RACE (vòng đời, readiness), PROJECT_IDENTITY_FAILURE (68 fix), CONTEXT_OVERHEAD + PROMPT_INTERFERENCE (tiêm), STALE_CONTEXT
(snapshot không ngân sách), SECURITY_BOUNDARY_CONFUSION (denylist env).

**AGENT CLI RELEVANCE** — Bài học quy trình (không kết luận phơi nhiễm): một tính năng "bật mặc định rồi thêm công tắc" bị
upstream bác vì công tắc thành dead code; test theo hình dạng thay vì hiệu ứng đã che lỗi production; sửa riêng từng adapter
để lại bug ở adapter khác; cơ chế "server sẵn sàng" toàn máy cần 4 vòng vá rồi bị bỏ.

**PROPOSED TESTS** — xem 01 (T1–T7). Thêm: T8 revert-safety — mỗi cơ chế vòng đời mới (tự tắt khi rảnh, dọn tiến trình anh em)
phải có kịch bản "nhiều host/nhiều phiên cùng máy" trước khi bật mặc định.

**NO-CHANGE FINDINGS** — Quyết định bỏ smartTruncate để index toàn bộ (không mất phần giữa) là hợp lý về dữ liệu; nhược điểm
nằm ở chỗ không đặt lại ngân sách cho snapshot, không ở chỗ bỏ cắt. Việc gỡ tiêm phong cách (#482) có lập luận và bằng chứng
ngoài, không có gì cần đảo ngược.
