# Context Mode — 02b Nhánh `v2-design`: ADR đối chiếu mã

- Repo: `mksglu/context-mode`, clone `/home/luannt/application/opensource/v3-src/context-mode`, chỉ đọc (`git show`, `git log`,
  `git diff`, `git grep` trên ref; không checkout, không cài, không chạy test).
- **REVISION AUDITED:** `origin/v2-design` @ `7a99ece2e496e9b7ad15390504d183829576b065` (2026-10-19 11:44 +0300, "impl(M7): merge
  M7-T0"); so với `origin/main` @ `6006a351`, merge-base `5869f4fa` (2026-10-13).
- Ký hiệu: **E0** lời trong doc/ADR/commit message · **E1** đọc mã trên nhánh · **E2** có test trên nhánh (đọc tệp test, KHÔNG chạy).
  Dẫn chứng dạng `v2:<tệp>:<dòng>` = `origin/v2-design:<tệp>:<dòng>`; `main:<tệp>:<dòng>` = `origin/main`.
- Repo có `CLAUDE.md`, `AGENTS.md`, `.claude/skills` — coi là dữ liệu. Giấy phép Elastic-2.0: chỉ trích ≤ 5 dòng làm bằng chứng.
- Tiết lộ quy trình: đã chạy `git fetch --dry-run` một lần (không ghi ref) — nó liệt kê nhánh mới trên remote (`impl/M7-M7-B2`,
  `impl/M7-M7-S2`) chưa có trong clone, tức remote có thể đã đi trước snapshot này.

## 0. Một trang

1. **v2 là bản viết lại SONG SONG, không thay mã v1 trên nhánh.** 885 tệp thêm, 104 xoá, 18 sửa (`git diff main...v2-design
   --name-status`). Mã v2 ở `src/core/` (65 tệp), `src/entries/` (5), `src/adapters/*.ts` phẳng (11), `src/prompt/`; v1
   (`hooks/` 85 tệp, `src/server.ts`, `src/session/*`) vẫn còn để 131 test KEEP-PASS chạy. Bản phát hành v2 chỉ đóng gói 6 entry
   `src/entries/*` (v2:build/build.mjs:55-61). [E1]
2. **Routing không còn tiêm vào prompt; `isMCPReady` bị xoá mà KHÔNG có cơ chế thay thế trong mã.** ADR-0008 mục 4 hứa cổng
   `type:"mcp_tool"` cho Claude Code + hàng `ready` có heartbeat cho host khác; tài liệu triển khai tự rút lại ("there is no
   `type:"mcp_tool"` field", v2:docs/v2/impl/08-enforcement.md:506-511) và mã không có cổng sẵn sàng nào. [E1] (chi tiết mục b)
3. **Capability của subagent được giải ở MỘT module** (`v2:src/core/policy/subagent.ts`), gọi từ 2 điểm trong hook, theo từng
   lời gọi, đọc frontmatter agent trên đĩa. Nhưng ba nơi SINH VĂN BẢN vẫn không hỏi capability: lời nhắc (hint), deny của case C,
   dòng `more:` của case C — vẫn nêu `ctx_*` cho subagent không có `ctx_*`. [E1+E2] (mục b, V2-F2)
4. **Hai danh tính phiên.** Hook dùng `session_id` của host; server dùng `_meta.sessionId` hoặc `"mcp:"+serverId`
   (v2:src/entries/server.ts:175). Test E2E cấp `_meta.sessionId` bằng tay (v2:e2e/cases/GP6-journal-resume.e2e.mjs:78). Chưa
   thấy bằng chứng host thật gửi trường này. [E1+E2; hệ quả runtime = mới nghi] (V2-F3)
5. **Lớp mất dữ liệu `4164d3de` biến mất theo cấu trúc**: một DB, không còn `session_meta`, không có DELETE sự kiện lúc khởi động;
   chín hook v1 xoá "mồ côi" vẫn nằm trong cây nhưng không được đóng gói. [E1] (mục c)
6. **Nhánh chưa sẵn sàng merge:** tự ghi M7 "not done" — GS5/GS10/GS11 FAIL (v2:docs/v2/TECH-SPEC.md:46-64) [E0 tự báo];
   `.impl/progress.json` xong M0–M6. 97 tệp unit v2 + 20 tệp E2E-style + 30 ca live; v1 giữ 210 tệp test. [đếm tệp, E1]

## (a) 18 ADR: quyết định, lý do, mã có làm đúng không

Cột "Mã": **LÀM** = mã khớp quyết định chính · **MỘT PHẦN** = khớp phần chính, lệch/thiếu phần nêu · **Ý ĐỊNH** = chưa có mã
tương ứng (ADR mới là ý định) · **LỆCH** = mã làm khác ADR (thường có dòng DEVIATIONS giải thích).

| ADR (`v2:docs/v2/adr/`) | Quyết định | Lý do (theo ADR, E0) | Mã trên nhánh | Mức |
|---|---|---|---|---|
| 0001 `claude-code-reference-contract` | Hợp đồng hook chuẩn = schema Claude Code (4 sự kiện, allow/hint/deny); adapter chỉ decode/encode, 30–80 dòng, không prompt/routing/state | v1 xử lý khác biệt host bằng fallback runtime trong từng adapter → lỗi không truy được về 1 đường | **MỘT PHẦN.** Một `handleHook` cho mọi host (v2:src/core/hooks/dispatch.ts:50-74); adapter thuần decode/encode (v2:src/adapters/claude-code.ts:31-70). Lệch nhỏ: `copilot-cli.ts` 118, `gemini.ts` 93, `pi.ts` 86 dòng > 80 (đếm `wc -l`) | E1 |
| 0002 `single-db-git-distribution` | Một `context.db`/dự án tại `projects/<hash git-common-dir>/`; node:sqlite `timeout:5000`; mọi ghi `BEGIN IMMEDIATE`; không vòng retry; corrupt → chỉ `ctx_doctor` đổi tên; dọn dự án chết: kiểm PID sống trong cùng khoá; phân phối qua nhánh git `release` | v1 hai DB/host, ~10 issue ABI native, `withRetry` busy-loop #985 | **MỘT PHẦN.** Khớp: đường dẫn + hash (v2:src/core/project/resolve.ts:141-155), `BEGIN IMMEDIATE` (v2:src/core/store/db.ts:151-164), corrupt không đổi tên ngoài doctor (db.ts:143-145). Lệch: dọn dự án kiểm PID trong transaction nhưng `rmSync` thư mục SAU `closeDb`, ngoài khoá (v2:src/core/store/cleanup.ts:100-111); PID nằm ở tệp `owners/*.pid` ngoài DB (v2:src/core/store/owners.ts:21-24) nên khoá DB không chặn server mới ghi tệp owner. Tức "atomic" của quyết định 4 không trọn (mới nghi về race, xem V2-F7) | E1 |
| 0003 `e2e-only-golden-tests` | Ba tầng: 131 KEEP-PASS v1 + GP1–GP11 E2E-style (tiến trình con, payload ghi sẵn) + live keyless; `e2e/` cấm import `src/` | Phần lớn yêu cầu là hợp đồng ở ranh giới tiến trình | **LÀM.** CI chạy tầng 1 KEEP-PASS (v2:.github/workflows/ci.yml:84), tầng 2 GP (ci.yml:75), thêm unit v2 (ci.yml:89) và lint cấm `e2e/` import `src/` (ci.yml:93); tầng 3 live chạy tay. Đếm: 20 tệp `e2e/cases/*.e2e.mjs` (170 lời gọi `test(`), 97 tệp `tests/v2` (987 `test(`/`it(`), 97 golden JSON, 30 ca live | E1 (đếm), E2 |
| 0004 `no-skills-mcp-instructions` | Một nguồn prompt `src/prompt/`; hai kênh: MCP `instructions` + mô tả tool; xoá 11 SKILL routing, 13 tệp routing, mọi đường adapter ghi AGENTS.md/CLAUDE.md… | Tiêm/tệp routing ghi vào không gian người dùng, 13 bản lệch nhau; #946 | **MỘT PHẦN.** Nguồn prompt + inline vào bundle có (v2:src/core/tools/registry.ts:7,56; `instructions.md` 1015 B). Xoá CHƯA xong: trong `configs/` còn 9 tệp routing/chỉ dẫn (`claude-code/CLAUDE.md`, `codex/AGENTS.md`, `gemini-cli/GEMINI.md`, `kiro/KIRO.md`, `opencode/AGENTS.md`, `pi/AGENTS.md`, `qwen-code/QWEN.md`, `vscode-copilot/copilot-instructions.md`, `antigravity-cli/rules/context-mode.md`) và 2 SKILL.md v1 (`configs/{antigravity-cli,copilot-cli}/skills/context-mode/SKILL.md`) — `git ls-tree` | E1 |
| 0005 `prompt-style-opus` | Luật PR1–PR12 (không CAPS, không "if in doubt", ≤ 2048/1024/8000 ký tự, tên tool trần), lint trên output đã build | Hướng dẫn prompting: nhấn mạnh gây over-trigger | **LÀM.** CI chạy `lint.mjs --out release-out` (v2:.github/workflows/ci.yml:73); build chặn khi vượt `SURFACE_CAP` (v2:build/build.mjs:133) | E1 |
| 0006 `claude-code-tool-loading` | `ctx_execute` mang `_meta anthropic/alwaysLoad`; các tool khác hoãn nạp | Tool search ẩn mô tả tool thường (probe 2026-10-14) | **LÀM.** v2:src/core/tools/registry.ts:33; v2:src/entries/server.ts:237 | E1 |
| 0007 `mcp-2026-07-28` | Server MCP 2026-07-28 qua SDK `@modelcontextprotocol/server` v2, phục vụ cả client 2025-xx; stateless; danh tính agent từ cờ `--agent <id>` lúc cài | Spec mới bỏ handshake, `clientInfo` không mang hành vi | **LÀM, có ngoại lệ.** SDK + `instructions` + cache hint cho `tools/list`/`server/discover` (v2:src/entries/server.ts:7,218-227). `resolveAgent` có nhánh dự phòng `CLAUDE_PROJECT_DIR` → `claude-code` (server.ts:39), trái tinh thần "no fallback chain". Session lấy từ `_meta` hoặc `"mcp:"+serverId` (server.ts:175) — xem V2-F3 | E1 |
| 0008 `enforcement-rewrite-and-cap` | Giữ phạm vi deny v1 (WebFetch, curl/wget ra stdout, inline HTTP, build tool) + MỚI: deny Read > 16 384 B không offset/limit; parser shell không regex; deny thật thay `echo`; bỏ `isMCPReady` (mục 4); subagent cùng ngân sách (mục 5); deny không trạng thái (mục 7); không tiêm (mục 8) | Mỗi cơ chế v1 gắn với issue: #1221, #1269, #1134, #1055, #794, #1003, #967 | **MỘT PHẦN.** LÀM: lexer/parser/unwrap (v2:src/core/policy/{lexer,parse,unwrap}.ts), deny per-pipeline + `NEVER_DENY_HOSTS` (v2:src/core/policy/decide.ts:157-195, constants.ts:210), Read deny (decide.ts:245-255), không tiêm prompt Agent (không có nhánh Agent trong `decide`, decide.ts:52-75). **Ý ĐỊNH/LỆCH mục 4:** không có cổng sẵn sàng nào trong `src/core` (`git grep` "ready\|heartbeat" trong `src/core` = 0; đã đọc decide/dispatch); hook Claude Code là `type:"command"` (v2:build/manifests.ts:47). **Golden "#173 server chưa nối → tool chạy" (ADR dòng 104) không có test**: `git grep '173\|disconnect'` trong `e2e/ tests/v2` = 0. Hàng "MCP ngoài: routed case C" (ADR dòng 18) bị vô hiệu trên host dùng matcher regex — xem V2-F5 | E1, E2 |
| 0009 `compaction-and-memory` | Khối trạng thái sau compact = chọn tham lam có ngân sách 2048 B trên journal của phiên; không LLM, không cắt; con trỏ `ctx_resume(seq)`; Claude Code chỉ nhận 1 dòng con trỏ (`compactSummary:"host"`); ranking BM25 + RRF k=60 | v1 snapshot không ngân sách (#1022 196 KB), cắt chữ, eviction theo nhãn ưu tiên | **LÀM.** `STATE_BUDGET=2048`, `POINTER_MIN=640`, trọng số nguyên (v2:src/core/contracts/constants.ts:9,12,29-31); render (v2:src/core/compaction/render.ts:16-18); con trỏ cho CC (v2:src/core/hooks/dispatch.ts:102-105). RRF trong một câu SQL (v2:src/core/store/fts.ts:29-52). Rủi ro nối hai danh tính phiên: V2-F3 | E1, E2 (`e2e/cases/GP6-journal-resume.e2e.mjs`) |
| 0010 `doctor-and-upgrade` | `ctx_doctor` chỉ đọc, D1–D10 + D5b, chuỗi FAIL cố định/host; `ctx_upgrade` một đường/chế độ cài; D8 được phép một lần ghi (heal) | v1 doctor so chuỗi đường dẫn, upgrade ghi đường dẫn có version | **LÀM** (đọc khung). `DOCTOR_CHECK_ORDER` D1…D10 (constants.ts:306); D8 heal (v2:src/core/ops/db-heal.ts, DEVIATIONS D-M5-D5-2). Không chạy | E1 |
| 0011 `release-and-v1-transition` | Nhánh `main` đóng băng, `v2` phát triển, `release` chỉ chứa build; `detectV1` thuần + gỡ v1 theo bảng | v1 `/ctx-upgrade` so version `package.json` ⇒ v2 không làm vỡ v1 | **MỘT PHẦN.** `detect-v1.ts`, `v1-remove.ts` có (v2:src/core/ops/). Nhánh thực tế tên `v2-design`, không phải `v2` (ADR dòng 4, 12) | E1 |
| 0012 `keyless-live-harness` | Live E2E chạy bằng `claude -p` của chủ repo, không API key; env dựng từ allowlist | Chủ repo không cấp key | **MỘT PHẦN.** `e2e/live/run.mjs` mặc định Sonnet 5 effort low (v2:e2e/live/run.mjs:14-15,87). Thiếu `.claude/commands/e2e-live.md` (ADR dòng 27) — `git ls-tree` = không có | E1 |
| 0013 `release-automation` | Một lệnh `/release`; `release.yml` concurrency + staging → smoke → promote | Phát hành thường xuyên | **LÀM** (đọc khung). `release.yml` có `concurrency`, `release-staging`, smoke 3 agent, attestation (v2:.github/workflows/release.yml:7,19,67,85,115,138-142); `.claude/skills/release/SKILL.md` có | E1 |
| 0014 `issue-forms-and-debug-report` | Báo cáo `ctx-report: 1 … digest: <sha256>` ≤ 4096 B "by construction"; redaction theo kiểu; bảng `hook_error_day`, `deny_day` 7 ngày; không nhận PR | Debug v1 rò secret (#1141, #1144), báo xanh trên trạng thái hỏng | **MỘT PHẦN.** Redaction theo enum có (v2:src/core/ops/report.ts:23-40). LỆCH: báo cáo là bảng Markdown + dòng env, **cắt** ở `REPORT_CAP` (report.ts:55-59), không có `digest`; **không có** bảng `hook_error_day`/`deny_day` trong schema (v2:src/core/store/schema.ts:19-78; `git grep` = 0). Form `bug.yml`/`feature.yml` có, template PR đã xoá | E1 |
| 0015 `readme-and-repo-cleanup` | README ≈165 dòng (CI cap 200); xoá `web/`, 57 hook `.mjs`, v1 skills/configs, `tests/` cuối cùng; dọn `package.json` | 81,5 % byte repo là thừa | **MỘT PHẦN.** README 172 dòng, có `check-readme.mjs` (ci.yml:95); `web/` đã xoá. Chưa: `hooks/` còn 85 tệp, `tests/` 210 tệp v1, `package.json` còn `better-sqlite3`, script `heal-better-sqlite3` (v2:package.json:75,110); `check-removed.mjs` liệt 22 đường dẫn còn sót (TECH-SPEC.md §1.1, E0 tự báo) | E1 |
| 0016 `subagent-context-protection` | Cùng ngân sách 16 384 B ở mọi context; subagent chỉ khi có `agent_id`; tool-set từ frontmatter dự án/user/plugin/`--agents`; case A (có ctx) / B (có Bash) / C (không cả hai); built-in chưa LV8 ⇒ coi như không có ctx | Hai lựa chọn v1 đều hỏng: deny ngõ cụt (#794, #1037) hoặc cho qua làm ngập context | **MỘT PHẦN.** LÀM: `isSubagentCall` chỉ theo `agentId` (v2:src/core/policy/subagent.ts:28-30), A/B/C (subagent.ts:33-40), wrapper `run.mjs` cho B (dispatch.ts:244-252), replace hậu kỳ cho C (dispatch.ts:311-326). Ý ĐỊNH: nguồn `--agents` JSON và manifest OpenCode có trong hàm nhưng **không caller nào truyền** (`git grep cliAgents\|manifestTools` ngoài subagent.ts = 0; dispatch.ts:220,313 chỉ truyền `{projectDir, env}`); bảng built-in LV8 chưa có ⇒ mọi built-in (Explore, Plan, general-purpose) rơi về C (subagent.ts:34; test `tests/v2/policy/subagent.test.mjs:138`). "Index trên worker thread" làm bằng promise detached cùng tiến trình (dispatch.ts:329-338). Trạng thái ADR tự ghi "mechanism Proposed until LV8–LV10 pass" (ADR dòng 3) | E1, E2 |
| 0017 `e2e-every-step-and-v1-retention` | Mỗi bước có ca E2E; live gắn `covers`; R1–R8 giữ người dùng v1 | Chủ repo yêu cầu | **MỘT PHẦN.** 30 ca live có `covers` (`git grep -c '"covers"'`). R1/R2 "PENDING owner", R3–R8 FAIL (TECH-SPEC.md §1.1, E0 tự báo) | E1, E0 |
| 0018 `implementation-architecture` | `00-architecture.md` ràng buộc; CI chặn mọi cạnh import ngoài bảng module | Tránh trôi giữa các người viết module song song | **Ý ĐỊNH (phần CI).** Không thấy bước CI nào kiểm đồ thị import: đã đọc toàn bộ bước `run:` trong `ci.yml` (dòng 48-100) và `git grep` "Imports cell\|module graph\|allowedImports" trong `scripts e2e build tests/v2 .github` = 0. Chỉ có lint "e2e không import src" (ci.yml:93) | E1 |

Tóm lại: **LÀM** 0003, 0005, 0006, 0009, 0010, 0013 · **MỘT PHẦN** 0001, 0002, 0004, 0007, 0011, 0012, 0014, 0015, 0016, 0017 ·
**Ý ĐỊNH (phần cốt lõi chưa có mã)** 0008 mục 4 (cổng sẵn sàng), 0016 (bảng LV8 + nguồn `--agents`/manifest), 0018 (cổng
import CI), 0014 §4 (bảng lưu lỗi hook/deny).

## (b) Routing, `isMCPReady` và capability

### Cái thay việc tiêm routing

- v1: nối khối `<context_window_protection>` ~4,6 KB vào prompt MỌI lời gọi Agent (01 mục 3, `main:hooks/core/routing.mjs:894-916`).
- v2: **không có nhánh Agent nào** trong `decide()` (v2:src/core/policy/decide.ts:52-75); chỉ dẫn đi qua hai kênh MCP chuẩn:
  `instructions` (1015 B, v2:src/prompt/instructions.md) và mô tả tool (`ctx_execute.md` 732 B, `ctx_search.md` 716 B). Cộng hint
  1 lần/phiên khoá theo `session_id` trong DB (v2:src/core/hooks/dispatch.ts:223-227). [E1]
- Hệ quả: chỉ dẫn tới subagent giờ phụ thuộc host có chuyển `instructions`/mô tả tool MCP cho subagent hay không — không còn do
  context-mode quyết. Suy luận: đây là cách dứt điểm họ #946/#967 (không viết vào prompt mình không sở hữu).

### Cái thay `isMCPReady`

| | v1 (`main`) | ADR-0008 mục 4 (ý định) | Mã v2 |
|---|---|---|---|
| Claude Code | sentinel `/tmp/context-mode-mcp-ready-<pid>` toàn máy (01 F3) | hook `type:"mcp_tool"` chỉ chạy khi server của phiên nối | hook `type:"command"` (v2:build/manifests.ts:45-47); TECH-SPEC tự đính chính "no `type:"mcp_tool"` field" (v2:docs/v2/TECH-SPEC.md:229) |
| host khác | như trên | hàng `ready` (agent, worktree, pid, heartbeat 30 s) trong `context.db`, deny chỉ khi hàng < 60 s | không có bảng/hàng `ready` (schema.ts:19-78); không có kiểm tra sẵn sàng nào |

**Kết luận E1:** v2 bỏ cổng mà không thay. Bản đồ issue của upstream ghi đúng như vậy: "deny does not gate on server-readiness
state … isMCPReady and the readiness gate disappear" (v2:tickets/v2-issue-map.md:149). Hệ quả (mới nghi, chưa chạy): khi server
không lên hoặc chạy chế độ suy giảm — Node < 24.15.0 hoặc thiếu `node:sqlite`, server chỉ liệt kê `ctx_doctor`
(v2:src/entries/server.ts:80-82) — hook vẫn deny curl/WebFetch/Read lớn và chỉ sang `ctx_execute`/`ctx_index`: `decide()` không
mở DB, không kiểm Node (decide.ts:52-75; cổng Node chỉ nằm trong `openDb`, db.ts:116). Riêng deny Read còn nêu
`Read(offset,limit)` nên không ngõ cụt; deny curl/WebFetch thì ngõ cụt. **Kiểm:** chạy `hook.mjs --agent claude-code` dưới
Node 22 với payload `PreToolUse` Bash `curl -s https://x.io/a.json` → kỳ vọng deny nêu `ctx_execute`, trong khi
`server.mjs` cùng Node trả `tools/list` chỉ có `ctx_doctor`.

### Capability của agent/subagent được giải ở đâu, khi nào

| Câu hỏi | v1 `main` (01 F3) | v2 |
|---|---|---|
| Nơi quyết | 3 chỗ, 3 giả định: `isMCPReady` (máy), `pretooluse.mjs:167` (`agent_id` HOẶC `agent_type` ⇒ không có MCP), nhánh Agent tiêm mọi subagent | **1 module**: `toolSetForAgentType` + `pickSubagentCase` (v2:src/core/policy/subagent.ts:33-53), gọi từ 2 điểm cùng tệp (v2:src/core/hooks/dispatch.ts:220, 313) |
| Khi nào | mỗi hook, không có tool-set | lười: PreToolUse chỉ khi `decide()` đã ra deny VÀ là subagent (dispatch.ts:218); PostToolUse chỉ khi subagent VÀ output > 16 384 B (dispatch.ts:312). Không cache — đọc tệp agent mỗi lần |
| Ai là subagent | `agent_id` hoặc `agent_type` (xếp nhầm main thread `--agent`) | chỉ `agent_id` (subagent.ts:28-30; test `tests/v2/policy/subagent.test.mjs:59`) |
| Tool-set lấy từ | không có | frontmatter `.claude/agents/<type>.md` dự án → user → plugin (`installed_plugins.json`) → `--agents` (subagent.ts:62-83); `tools:` trừ `disallowedTools:` (subagent.ts:50-52); chưa biết ⇒ `null` ⇒ case C (subagent.ts:34) |
| Host nào có khái niệm subagent | chỉ hook Claude Code | chỉ adapter dạng Claude (decode `agent_id`, v2:src/adapters/claude-code.ts:37-40; dùng chung cho codex); Kiro/OpenCode/Pi ghi rõ "agentId stays undefined" (v2:src/adapters/{kiro.ts:47,opencode.ts:49,pi.ts:46}) |

**Những chỗ v2 vẫn soạn văn bản KHÔNG theo capability (E1, có test ghim một phần):**

1. **Hint** (Bash/Read/Grep/MCP) nêu `{prefix}ctx_execute` (v2:src/core/contracts/constants.ts:198-201); đường hint không kiểm
   subagent (dispatch.ts:223-227). Khoá 1 lần/phiên dùng chung `session_id` với phiên cha ⇒ nếu lời gọi đầu tiên của phiên là của
   subagent case B/C, chính subagent đó nhận hint về tool nó không có. (cùng mẫu F4 của v1)
2. **Deny case C giữ nguyên deny phiên chính** (dispatch.ts:219-221; DEVIATIONS D-M3-H2-4, v2:docs/v2/impl/DEVIATIONS.md:261).
   Với WebFetch, lời deny nêu `ctx_index` rồi `ctx_search` (decide.ts:81-90, constants.ts:175). Subagent case C theo định nghĩa
   không có `ctx_*` ⇒ ngõ cụt đúng kiểu #1037. Với built-in chưa có hàng LV8 (Explore/Plan/general-purpose ⇒ C), curl qua Bash
   cũng nhận deny nêu `ctx_execute`.
3. **Dòng `more:` của case C** là `ctx_search({queries:[…], source:…})` (v2:src/core/index/select.ts:59-62), được nối vào output
   thay thế cho subagent case C (dispatch.ts:326). Test ghim hành vi này: `tests/v2/hooks/dispatch.test.mjs:233-251` cắt chuỗi tại
   `"\nctx_search("`. [E2]
4. Test "case A subagent (no agent file, inherits ctx tools)" (`tests/v2/hooks/dispatch.test.mjs:280-285`) gửi `agent_id` không
   có `agent_type` ⇒ `toolSet=null` ⇒ thực ra là **case C**; test vẫn xanh vì A và C cùng giữ deny phiên chính. Test không phân
   biệt được A với C. [E2, đọc mã]

Case B đã được sửa đúng sau một lần hỏng live: deny ban đầu giữ lời "Tool: ctx_execute", subagent gọi `ctx_execute` và thất bại;
nay lời deny là `SUBAGENT_RUN_DENY`, chỉ nêu dòng `run.mjs` (DEVIATIONS D-M3-F-7, DEVIATIONS.md:286; constants.ts:255; test
dispatch.test.mjs:263-278 kiểm "không chứa `ctx_`"). Bản sửa chỉ phủ case B — đây là ví dụ "sửa một nhánh, nhánh anh em giữ lỗi".

So với ba chỗ lệch nhau trên `main`: v2 gom QUYẾT ĐỊNH về một chỗ (INV-CM-2 đạt cho quyết định), nhưng chưa gom VĂN BẢN (INV-CM-1
chưa đạt cho hint/deny C/`more:`).

## (c) Containment, index/search, vòng đời phiên, hook mất dữ liệu

| Khía cạnh | v1 (01) | v2 | Bằng chứng |
|---|---|---|---|
| Ngưỡng output `ctx_execute` | trả nguyên ≤ 100 KiB | trả nguyên khi không `intent` và cả reply ≤ `OUTPUT_CAP` 16 384 B; còn lại index toàn bộ rồi chọn chunk theo BM25 với `intent`, chèn dòng `more:` | v2:src/core/tools/execute.ts:145-160; constants.ts:8 [E1] |
| Cắt | "không cắt" | chọn chunk nguyên, nhưng vẫn `capBytes(text, OUTPUT_CAP)` ở cuối như chốt chặn | execute.ts:160 [E1] |
| Timeout server-side | không có (trừ agy) | mặc định và tối đa 600 000 ms, giết cả cây | constants.ts:50; execute.ts:182,207 [E1] |
| Nhãn nguồn (F1 ghi đè) | `execute:<lang>` + DELETE cùng nhãn | **append-only**: `ftsInsert` chỉ INSERT, không DELETE; nhãn = `ctx_execute:` + dòng đầu code | v2:src/core/store/fts.ts:60-99; execute.ts:120-125 [E1] — F1 biến mất |
| Ghi index | 1 transaction | chia lô `INDEX_WRITE_CHUNK`, MỖI lô một `withWrite` riêng; hàng `sources` ghi `chunk_count` đủ ngay ở lô đầu | fts.ts:63-97 [E1]. Mới nghi: tiến trình chết giữa các lô ⇒ nguồn khai N chunk nhưng chỉ có một phần, không lỗi nào báo |
| Tokenizer / fuzzy | porter + trigram + Levenshtein | một FTS5 `unicode61 remove_diacritics 2`, khớp tiền tố `"w"*`; không trigram/porter/fuzzy (`git grep` = 0) | v2:src/core/store/schema.ts:57-65; fts.ts:103-116 [E1]; lý do đo: ADR-0009 §9 (E0, recall@5 0,48 vs 0,27) |
| Phạm vi | ContentStore theo dự án, không theo worktree (F7) | mọi hàng mang `session_id, worktree, agent`; mặc định `scope` = worktree; xếp phiên hiện tại → worktree hiện tại → điểm | schema.ts:20-65; fts.ts:35-37,47; mô tả tool `ctx_search.md` [E1] |
| Nguồn cũ | so SHA-256, tự index lại | không cột hash, không làm mới (`git grep content_hash\|stale\|refresh` = 0); index lại = nguồn mới, nguồn cũ vẫn khớp | schema.ts:42-54 [E1]; mới nghi: kết quả cũ lẫn mới cùng trả (nguồn mới xếp trước khi đồng điểm, fts.ts:47) |
| Phạm vi cần git | mọi thư mục | ngoài repo git: tool `ctx_*` lỗi `NO_GIT_DIR`; hook cho qua TẤT CẢ (không deny, không hint, không DB) | v2:src/core/contracts/errors.ts:26; resolve.ts:87; dispatch.ts:61-65 [E1] |
| Env tiến trình con | denylist ~90 biến (BASH_ENV, LD_PRELOAD, GIT_SSH_COMMAND…) | chỉ bỏ `NODE_OPTIONS`, `NODE_*_OPTIONS`, `BUN_OPTIONS`, `DENO_OPTIONS`, 2 biến Node | v2:src/core/sandbox/env.ts:7-21; `main:src/executor.ts:581,655` [E1]. `git grep LD_PRELOAD\|BASH_ENV` trong `src/core` + `impl/03` + PRD = 0 ⇒ không thấy quyết định ghi lại (thu hẹp có chủ ý hay sót: chưa rõ) |

**Vòng đời phiên v2 (E1):**

| Bước | Mã |
|---|---|
| create | không có bước tạo phiên; hàng `events` tự mang `session_id` (schema.ts:20-38) — không có `session_meta` |
| ghi | hook ghi "detached" (fire-and-forget, lỗi chỉ ra stderr, v2:src/core/hooks/detached.ts:9-26); tiến trình hook không gọi `process.exit` để lần ghi kịp xong (v2:src/entries/hook.ts:21-26, D-M3-H2-8). Server ghi 1 hàng/lời gọi `ctx_*` (server.ts:137-157) |
| compact | CC: 1 dòng con trỏ cố định; host khác: khối ≤ 2048 B (dispatch.ts:102-122, 134-138); cờ pending xoá bằng `DELETE … RETURNING` (db.ts:200-204) |
| restore | `ctx_resume()` dựng khối theo `ctx.sessionId`; `ctx_resume(seq)` từ chối hàng khác phiên (v2:src/core/tools/resume.ts:34,43) |
| cleanup | chạy 1 lần/tiến trình server khi mở DB, không timer (server.ts:93-110). Nguồn: xoá theo tuổi truy cập 30 ngày + trần 500 MB — **bỏ qua hoàn toàn khi còn bất kỳ server sống** (cleanup.ts:35; constants.ts:74-75). Dự án: xoá cả thư mục khi không còn owner và 14 ngày không truy cập (cleanup.ts:100-133; constants.ts:76) |
| purge | `ctx_purge` theo session/nguồn/toàn bộ (v2:src/core/tools/purge.ts:49-55) — đường DUY NHẤT xoá `events` |
| crash/thoát | thoát khi stdin EOF, không timer/ps (v2:src/core/lifecycle/exit.ts:11-27) — đáp #1262, #1100 |

**Hook mất dữ liệu (`4164d3de` + 8 hook khác):** trên v2 lớp này **không tồn tại theo cấu trúc**: (1) không có bảng `session_meta`
nên không có khái niệm "sự kiện mồ côi" (schema.ts:19-78); (2) chỉ 3 câu `DELETE` trên dữ liệu trong `src/core` — cleanup nguồn
(cleanup.ts:27-28), lấy cờ meta (db.ts:202), `ctx_purge` (purge.ts:49-55) — không câu nào chạy ở SessionStart (`git grep "DELETE
FROM"` trong `src/core src/entries`); (3) một entry hook cho mọi host (dispatch.ts:107-122), nên không còn "sửa 1/9 hook". Chín hook
v1 với câu DELETE vô điều kiện **vẫn nằm trong cây** (`git grep -c 'NOT IN (SELECT session_id FROM session_meta)'` = 9 tệp dưới
`hooks/` + `src/session/db.ts`) nhưng không được đóng gói vào bản v2 (build.mjs:55-61). [E1]

Rủi ro mới thay chỗ (mới nghi): `events` không bao giờ bị cleanup tự động xoá khi dự án còn sống (cleanup.ts chỉ xoá `sources`)
⇒ journal tăng không giới hạn trong dự án dùng hằng ngày; ghi detached lỗi chỉ ra stderr, không có bộ đếm (ADR-0014 §4 định
có `hook_error_day` nhưng chưa có, mục a) ⇒ mất hàng journal mà không ai đếm được.

## (d) v2 bỏ hẳn gì, vì sao

| Bỏ | Lý do (nguồn) | Mã v2 xác nhận |
|---|---|---|
| Tiêm routing vào prompt Agent + SessionStart block 4,6 KB | #946, #967, #911: classifier của host chặn; "symptom of the routing-block design itself" (ADR-0004; v2:docs/v2/review/ISSUE-OUTCOMES-r4.md:168) [E0] | không nhánh Agent (decide.ts:52-75) [E1] |
| `isMCPReady` + sentinel | 4 vòng vá v1 (02); #1055, #1053 [E0] | không cơ chế thay (mục b) [E1] |
| 11 SKILL routing, 13 tệp routing, adapter ghi tệp người dùng | 13 bản lệch, sửa file của người dùng (#1235, #1239, #1160) [E0] | đã xoá `skills/context-mode/*` (`c814ef3e`); 9 tệp routing + 2 SKILL.md trong `configs/*` còn (mục a 0004) [E1] |
| Hai DB/host, better-sqlite3, bun:sqlite, ensure-deps, heal scripts | ~10 issue ABI mở; chuỗi self-heal 13+ commit (ADR-0002) [E0] | node:sqlite trong `src/core` (db.ts:49-57); `package.json` vẫn giữ dep v1 cho test v1 [E1] |
| Snapshot resume không ngân sách, nhãn ưu tiên, bộ phân loại intent/role | #1022, #1156, #856, #1259 [E0] | khối 2048 B, trọng số nguyên (mục a 0009) [E1] |
| Throttle `ctx_search` (flood-guard) | bộ đếm toàn cục sẽ chặn fan-out hợp lệ của 10 subagent (ADR-0015) [E0] | không có `BLOCK_AFTER`/throttle trong `search.ts` [E1] |
| Smart-truncate / build-tool `tail -30` | "owner rule: no truncation" (ADR-0008 mục 9) [E0] | chọn chunk (select.ts:34-57) [E1] |
| Agent bị bỏ: Cursor, OpenClaw, OMP, Kimi, Kilo, Zed, JetBrains, Antigravity IDE | PRD §17, D14 [E0] | 10 hàng adapter (v2:src/adapters/table.ts:28-98); `c814ef3e` xoá adapter OpenClaw/OMP/Kimi [E1] |
| Pull request từ cộng đồng | chủ repo: "PR istemiyorum" (ADR-0014) [E0] | `PULL_REQUEST_TEMPLATE.md` đã xoá [E1] |
| CLI `context-mode`, statusline, `ctx_insight`, `ctx_fetch_and_index` (gộp vào `ctx_index` URL) | ADR-0010 (9 tool), ADR-0015 [E0] | `TOOL_ORDER` 9 tool (constants.ts:5) [E1] |
| `monorepo`, tự ghi routing file (đã bỏ từ v1, 02) | — | — |

Ghi chú: "bỏ" ở đây phần lớn là bỏ khỏi bản PHÁT HÀNH v2; mã v1 tương ứng vẫn nằm trên nhánh cho tới bước xoá cuối (ADR-0015 thứ
tự 0–5, `tests/` xoá sau cùng).

## (e) Tình trạng nhánh

| Câu hỏi | Trả lời | Bằng chứng |
|---|---|---|
| Commit cuối | `7a99ece2`, 2026-10-19 11:44 +0300 | `git log -1` [E1] |
| Nhịp | 737 commit trong 6 ngày (10-14: 164, 10-15: 119, 10-16: 10, 10-17: 159, 10-18: 128, 10-19: 157); 275 nhánh `impl/*`/`wip/*` trên remote; commit rải 24 giờ (00 h: 136) | `git log --date`, `git branch -r` [E1]. Suy luận: chạy bằng agent tự động (chủ đề "workflow merger verifies merges…" `4949c3dd`; `OVERNIGHT-REPORT.md`) |
| Tác giả | một người dưới 3 tên (mksglu 407, Mert Köseoğlu 261, Mert Koseoglu 69) | `git shortlog -sn` [E1] |
| Phân loại commit | review 261, impl(M0–M7) 341, merge/impl(merge) 42, docs 34 … | đếm tiền tố subject [E1] |
| Test (đếm tệp, không chạy) | `tests/v2`: 97 tệp (987 `test(`/`it(`); `e2e/cases`: 20 tệp (170 `test(`); `e2e/golden`: 97 JSON; `e2e/live/cases`: 30 JSON; v1 `tests/`: 210 tệp (= `main`) | `git ls-tree` + `grep -c` [E1] |
| Cổng tự báo | M7 not done: GS5 FAIL (22 đường dẫn sót), GS10 R3–R8 FAIL, GS11 FAIL (live T-CMP); live 18/19 PASS | v2:docs/v2/TECH-SPEC.md:46-64 (`ded3c90e`) [E0] |
| Tiến độ | `.impl/progress.json` done: install, M0–M6, docs | [E1] |
| Dấu hiệu sắp merge vào `main` | **Không thấy**: ADR-0011 nói `main` giữ đóng băng, v2 phát hành qua nhánh `release` chứ không merge vào `main`; nhánh `release` chưa có trong `git branch -r`; `package.json`/`hooks/`/`tests/` v1 chưa dọn; M7 chưa xong; nhánh tên `v2-design` thay vì `v2` | [E1 + E0] |

## Phát hiện mới trên v2 (V2-F*)

**Đã chứng minh (đọc mã/test, có lệnh tái lập `git show`/`git grep`):**

- **V2-F1 Cổng sẵn sàng bị bỏ, không thay.** ADR-0008 mục 4 hứa cơ chế, impl/08 B3 rút lại, mã không có (mục b). Golden #173 không
  có test; TECH-SPEC.md:229 hứa thêm ca "disconnect the `context-mode` server mid-session" — `git grep -i 'disconnect\|not connected'`
  trong `e2e/ tests/v2` = 0. [E1]
- **V2-F2 Capability chỉ được hỏi khi QUYẾT, không khi VIẾT.** Hint, deny case C, `more:` case C nêu `ctx_*` cho subagent không có
  (mục b, 4 điểm). Một test ghim hành vi (dispatch.test.mjs:233-251), một test gắn nhãn sai A/C (dispatch.test.mjs:280-285). [E1, E2]
- **V2-F4 Nguồn tool-set khai mà không nối:** `cliAgents` (`--agents` JSON) và `manifestTools` (OpenCode 2) không caller nào truyền;
  bảng built-in LV8 chưa có ⇒ mọi built-in là case C. [E1]
- **V2-F5 Ba tài liệu mâu thuẫn về tool MCP ngoài.** ADR-0008 dòng 18 + ADR-0016 Q1: tool `mcp__<other>__*` được route (hint,
  case C replace). TECH-SPEC §3.8 luật 5 (v2:docs/v2/TECH-SPEC.md:230) + `buildMatcher` (v2:build/manifests.ts:40-43) chỉ khớp
  tool native + tiền tố của chính context-mode; GP3 khẳng định `mcp__grafana__get_datasource` KHÔNG sinh hook
  (v2:e2e/cases/GP3-hooks.e2e.mjs:151-161). Vậy trên host dùng matcher regex (Claude Code, Codex, Qwen, ZCode) nhánh MCP ngoài
  trong `decide.ts:73` và case C cho MCP ngoài không bao giờ được gọi; test đơn vị `GP3/external-mcp-tool-routing-hint`
  (dispatch.test.mjs:172) xanh vì gọi thẳng `handleHook`, bỏ qua matcher của host. [E1, E2]
- **V2-F6 Live test subagent tự mớm giao thức.** Ca `T-SUB` (v2:e2e/live/cases/subagent-cap.json) định nghĩa agent `bash-runner`
  có lời dặn "If context-mode denies the Bash call and the reason names a replacement `node .../run.mjs ...` command … run that
  replacement". Subagent thật của người dùng không có lời dặn này ⇒ ca live xanh không chứng minh subagent thông thường tự theo
  được deny case B. [E1 đọc tệp test]
- **V2-F8 Thu hẹp env denylist** so với v1 (mục c). [E1]
- **V2-F9 `claude-code` khoá cứng tiền tố cài plugin** `mcp__plugin_context-mode_context-mode__` (v2:src/adapters/claude-code.ts:6),
  một chế độ cài/agent (table.ts:30 `installMode:"marketplace"`). #1018 được "đóng" bằng cách chỉ hỗ trợ một kiểu cài; TECH-SPEC
  §3.8 vẫn liệt cả tên MCP rời (TECH-SPEC.md:210). [E1]

**Mới nghi (kèm cách kiểm):**

- **V2-F3 Hai danh tính phiên.** Hook: `session_id` của host (claude-code.ts:36). Server: `_meta.sessionId` hoặc `"mcp:"+serverId`
  (server.ts:175; quyết định A8, v2:docs/v2/impl/00-architecture.md:421). Test E2E luôn tự cấp `_meta: {sessionId}`
  (GP6-journal-resume.e2e.mjs:78,201,229,238); GP7 dựa vào fallback (GP7-scope.e2e.mjs:250-253). Không tìm thấy bằng chứng host thật
  gửi `_meta.sessionId` (`git grep` trong TECH-SPEC/PRD/spikes chỉ thấy Copilot `preMcpToolCall only edits _meta`). Nếu host không
  gửi: (a) `ctx_resume()` trên CC (con trỏ compact trỏ tới nó) đọc phiên `mcp:<serverId>` — chỉ có hàng `ctx_*`, không có prompt/edit
  do hook ghi; (b) con trỏ `ctx_resume(seq:N)` trong khối do hook dựng bị server từ chối vì khác phiên (resume.ts:43); (c) lỗi
  `ctx_execute` không bao giờ vào mục "open errors" của khối compact (hook bỏ qua tool của chính mình, dispatch.ts:297). **Kiểm:**
  phiên Claude Code thật, ghi log `req.mcpReq._meta` ở `dispatch` của server rồi so với `session_id` trong payload hook cùng lượt.
- **V2-F7 Race dọn dự án.** Kiểm PID trong `BEGIN IMMEDIATE` nhưng `rmSync` sau khi đóng DB, ngoài khoá; server mới chỉ ghi tệp
  owner, không giữ khoá DB khi ghi (cleanup.ts:100-111; owners.ts:21-24; server.ts:93-99). **Kiểm:** hai tiến trình, A chạy
  `sweepDeadProjects` có điểm dừng giữa `closeDb` và `rmSync`, B mở dự án đó trong khe → đếm tệp `context.db` của B sau khi A xong.
- **V2-F10 Suy giảm ⇒ ngõ cụt deny** (mục b, cách kiểm đã ghi).
- **V2-F11 Ghi index nhiều lô không nguyên tử** (mục c).

---

## Hợp đồng đầu ra (spec §63)

**SCOPE** — Nhánh `origin/v2-design` của Context Mode: 18 ADR đối chiếu mã; thay thế cho tiêm routing/`isMCPReady`; nơi/lúc giải
capability; containment, index/search, vòng đời phiên, lớp mất dữ liệu `4164d3de`; cái bị bỏ; tình trạng nhánh. Không chạy test,
không cài, không đọc trang issue.

**REVISION AUDITED** — `origin/v2-design` @ `7a99ece2` (2026-10-19); đối chiếu `origin/main` @ `6006a351`.

**FILES/COMMITS/ISSUES INSPECTED** — 18 ADR `docs/v2/adr/*`; `docs/v2/TECH-SPEC.md` (§1.1, §3.8), `docs/v2/impl/{00-architecture,08-enforcement,DEVIATIONS}.md`,
`tickets/v2-issue-map.md`, `docs/v2/review/ISSUE-OUTCOMES-r4.md`, `OVERNIGHT-REPORT.md`, `.impl/progress.json`; mã
`src/core/{policy/{subagent,decide}.ts,hooks/{dispatch,detached}.ts,store/{db,schema,cleanup,owners,fts}.ts,index/{text,select}.ts,
tools/{registry,execute,resume,search}.ts,sandbox/env.ts,project/resolve.ts,compaction/render.ts,lifecycle/exit.ts,ops/report.ts,
contracts/{constants,errors}.ts}`, `src/entries/{server,hook}.ts`, `src/adapters/{claude-code,table}.ts`, `build/{manifests.ts,build.mjs}`,
`.github/workflows/{ci,release}.yml`; test `tests/v2/{hooks/dispatch,policy/subagent}.test.mjs`, `e2e/cases/{GP3-hooks,GP6-journal-resume,GP7-scope}.e2e.mjs`,
`e2e/live/cases/subagent-cap.json`, `e2e/live/run.mjs`. Commit: `7a99ece2`, `ded3c90e`, `c814ef3e`, `4949c3dd`; v1 `main:src/executor.ts`.

**VERIFIED FACTS** — Mục (a) cột "Mã" với tệp:dòng; V2-F1, F2, F4, F5, F6, F8, F9; số đếm tệp/test/commit ở mục (e); v2 không còn
lớp xoá "sự kiện mồ côi" trong mã đóng gói; F1 (ghi đè nhãn) của v1 không còn (append-only).

**INFERENCES** — V2-F3 (hai danh tính phiên làm hỏng resume trên host thật), V2-F7 (race dọn dự án), V2-F10 (ngõ cụt khi suy giảm),
V2-F11 (index một phần), journal không giới hạn, kết quả cũ lẫn mới khi index lại; nhánh chạy bằng agent tự động; v2 phát hành qua
nhánh `release`, không merge `main`.

**UNKNOWNs** — Host nào gửi `_meta.sessionId`; kết quả LV8 (built-in subagent có thấy MCP không); remote đã đi trước snapshot bao xa
(dry-run thấy nhánh mới); thu hẹp env denylist là chủ ý hay sót; mọi hành vi runtime (không chạy gì).

**FAILURE MODES** — CAPABILITY_MISMATCH (V2-F2, F4), STARTUP_RACE/ROUTING_FAILURE (V2-F1, F10), SESSION_CONTAMINATION/IDENTITY
(V2-F3), FALSE_SUCCESS do test (V2-F5 unit bỏ qua matcher, V2-F6 live tự mớm, test A/C gắn nhãn sai), DELETION/RACE (V2-F7),
PARTIAL_SUCCESS (V2-F11), STALE_CONTEXT (không làm mới nguồn), SECURITY_BOUNDARY_CONFUSION (V2-F8), OBSERVABILITY_FAILURE (lỗi
ghi detached không đếm).

**AGENT CLI RELEVANCE** — Câu hỏi để kiểm trên agent-cli, không kết luận ở đây: chỗ QUYẾT capability và chỗ VIẾT chỉ dẫn có đọc cùng
một bản ghi không; danh tính phiên của tầng hook và tầng tool có cùng một nguồn không; test có tự cấp thứ mà môi trường thật không
cấp (`_meta`, lời dặn trong agent fixture) không; bỏ một cổng có kèm cơ chế thay hoặc ca test cho trạng thái "dịch vụ không lên" không.

**PROPOSED TESTS** — T9 two-layer session identity: một lượt thật đi qua hook và tool ⇒ assert cùng một khoá phiên ở cả hai tầng.
T10 capability-aware text: subagent không có tool X ⇒ mọi văn bản gửi nó (hint, lý do deny, dòng tiếp theo) không nêu X. T11
service-down enforcement: dịch vụ đích tắt/suy giảm ⇒ chặn không được chỉ sang dịch vụ đó. T12 host-matcher reachability: mỗi
nhánh luật trong policy ⇒ có ít nhất một payload đi qua matcher ĐÃ SINH tới được nó. T13 fixture-without-coaching: ca live dùng
agent fixture không chứa lời dặn về giao thức của chính hệ thống.

**NO-CHANGE FINDINGS** — Bỏ tiêm prompt Agent, một DB, append-only index, khối compact có ngân sách và không LLM, thoát theo stdin EOF,
timeout 600 s server-side: đều đơn giản hơn v1 và có test; không thấy lý do đề xuất ngược lại.
