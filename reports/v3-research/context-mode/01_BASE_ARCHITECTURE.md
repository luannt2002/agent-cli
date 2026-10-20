# Context Mode — 01 Kiến trúc hiện tại (từ mã)

- Repo: `mksglu/context-mode`, clone tại `/home/luannt/application/opensource/v3-src/context-mode` (chỉ đọc).
- **REVISION AUDITED:** `main` @ `6006a3511ea2afde3e413cf635ba1da85ac8a8aa` (2026-10-18, "ci: update install stats").
  Mã nguồn tại HEAD = tag `v1.0.169` (2026-06-29) + bundle CI dựng lại: `git diff --stat v1.0.169 HEAD` chỉ đổi
  `server.bundle.mjs`, `stats.json`. Mọi commit trên `main` sau 2026-06-29 là bot "ci: update install stats".
- Nhánh `origin/v2-design` (737 commit chưa vào `main`, bắt đầu 2026-10-14) là bản viết lại đang làm — chỉ dùng làm bằng
  chứng E0 về ý định của upstream, KHÔNG phải kiến trúc hiện tại (xem 02_COMMIT_EVOLUTION, thời kỳ 7).
- Ký hiệu bằng chứng (spec §5): **E0** doc/claim · **E1** đọc mã · **E2** có test (không chạy: cấm `npm install`) ·
  **E4\*** chạy hàm thuần tại chỗ (chỉ tạo chuỗi, không cài gì) để đo kích thước.

## 0. Bản đồ một trang

Context Mode = **một MCP server** (`src/server.ts`, 4991 dòng) + **bộ hook theo host** (`hooks/*.mjs`, `hooks/<host>/`) +
**hai CSDL SQLite** (ContentStore FTS5 và SessionDB) + **17 adapter host** (`src/adapters/*`).

| Thành phần | Vai trò | Tệp chính |
|---|---|---|
| MCP tools (11) | `ctx_execute`, `ctx_execute_file`, `ctx_index`, `ctx_search`, `ctx_fetch_and_index`, `ctx_batch_execute`, `ctx_stats`, `ctx_doctor`, `ctx_upgrade`, `ctx_purge`, `ctx_insight` | `src/server.ts:1647,2042,2236,2553,3412,3678,3936,4139,4270,4445,4816` [E1] |
| Executor | chạy code 11 ngôn ngữ trong tiến trình con, cap byte, kill cây tiến trình | `src/executor.ts` [E1] |
| ContentStore | FTS5 kép (porter + trigram), BM25, RRF, fuzzy | `src/store.ts` [E1] |
| SessionDB | sự kiện phiên (capture từ PostToolUse), snapshot resume | `src/session/db.ts`, `src/session/snapshot.ts` [E1] |
| Routing | quyết định PreToolUse thuần, chung mọi host | `hooks/core/routing.mjs` (1017 dòng) [E1] |
| Formatter | dịch quyết định chuẩn → JSON riêng từng host | `hooks/core/formatters.mjs` [E1] |
| Văn bản tiêm | `<context_window_protection>` + các `<context_guidance>` | `hooks/routing-block.mjs` [E1] |

## 1. Luồng tool-output (spec §9.1) — đo ở từng ranh giới

```
tool request → PreToolUse routing → (native tool | ctx_* tool) → execution → raw output → containment/store
→ transform/index → search/filter → selected result → model context
```

| Ranh giới | Mã làm gì | Hằng số / giới hạn | Bằng chứng |
|---|---|---|---|
| Native tool đi qua | PreToolUse chỉ CHẶN (deny) hoặc NHẮC (additionalContext); PostToolUse **không** sửa output (`posttooluse.mjs` không in `hookSpecificOutput`). Output của Read/Grep/Bash được phép vào context **nguyên vẹn**. | — | `hooks/posttooluse.mjs:224` [E1]; `grep updatedToolOutput` trong hooks = 0, đã mở `posttooluse.mjs` đọc xác nhận |
| Chặn cứng native | curl/wget không ghi file, inline HTTP, gradle/mvn/sbt, WebFetch → redirect sang ctx_* | ước lượng "bytes avoided" cố định 8192 (curl), 16384 (WebFetch) — **hằng số, không đo thật** | `hooks/core/routing.mjs:733-818,875-890,779,886` [E1] |
| Nhắc mềm native | Bash không bị chặn → nhắc 1 lần/phiên; Read > 50 000 B → nhắc + đếm byte; Grep nhắc 1 lần; MCP ngoài nhắc mỗi 10 lần | `50_000`, `EXTERNAL_MCP_NUDGE_DEFAULT=10` | `routing.mjs:840,853,871,64,1010-1013` [E1] |
| Executor stream | gộp stdout+stderr, vượt cap → kill cây tiến trình | `hardCapBytes = 100 MiB` | `src/executor.ts:251,510-547` [E1] |
| Timeout | KHÔNG có timer server-side trừ host `antigravity-cli` | `AGY_DEFAULT_EXEC_TIMEOUT_MS = 120_000`; fetch subprocess 30 000 ms | `src/server.ts:1493-1503,3323` [E1] |
| Echo mã nguồn | chép mã đã chạy vào đầu kết quả | `CODE_ECHO_MAX = 2000` ký tự, `COMMAND_ECHO_MAX = 500` | `server.ts:1477,1511` [E1] |
| ctx_execute không `intent`, stdout ≤ 100 KiB | **trả nguyên stdout vào context** | `LARGE_OUTPUT_THRESHOLD = 102_400` | `server.ts:1980,1939` [E1] |
| ctx_execute không `intent`, stdout > 100 KiB | index vào FTS5, trả con trỏ | như trên | `server.ts:1923-1935` [E1] |
| ctx_execute có `intent`, output > 5 000 B | index + BM25 theo intent, trả **tiêu đề + dòng đầu ≤120 ký tự** của ≤5 section | `INTENT_SEARCH_THRESHOLD = 5_000`, `maxResults = 5`, `slice(0,120)` | `server.ts:1979,1982-2033,2023` [E1] |
| Chunking | markdown theo heading; plain text 20 dòng/chunk; chunk tối đa 4096 B | `MAX_CHUNK_BYTES = 4096`, tiêu đề ≤ 80 ký tự | `src/store.ts:153-170,966-990` [E1] |
| ctx_search | throttle theo cửa sổ 60 s: sau 3 lần → 1 kết quả/query, sau 8 lần → BLOCKED; tối đa 2 kết quả/query; tổng ≤ 40 KiB | `SEARCH_WINDOW_MS=60_000`, `MAX_RESULTS_AFTER=3`, `BLOCK_AFTER=8`, `MAX_TOTAL=40*1024` | `server.ts:2457-2459,2655-2672` [E1] |
| ctx_fetch_and_index | trả preview ≤ 3072 ký tự (batch: 384/URL), phần còn lại phải `ctx_search` | `FETCH_PREVIEW_LIMIT=3072`, `FETCH_BATCH_PREVIEW_LIMIT=384` | `server.ts:3119,3401-3402,3618` [E1] |
| ctx_batch_execute | index toàn bộ output dưới một nhãn, trả inventory section + kết quả query | nhãn `batch:<labels>`.slice(0,80) | `server.ts:3827-3870` [E1] |

**Phát hiện F1 — nhãn `execute:<language>` ghi đè nhau (mất dữ liệu âm thầm theo thiết kế).** Mọi output lớn của
`ctx_execute` cùng ngôn ngữ dùng chung nhãn `execute:javascript` / `execute:shell` (`server.ts:1887,1897,1917,1924`), và
`#insertChunks` xoá mọi source cùng nhãn trước khi chèn (`store.ts:1046-1052`, lý do: #67 tránh kết quả cũ trong vòng
build-fix-build). Hệ quả (suy luận từ mã, chưa chạy): hai lần `ctx_execute` JS khác mục đích, cả hai > 100 KiB → lần sau
xoá chỉ mục của lần trước; `ctx_search` không còn tìm thấy output thứ nhất. Đối chiếu: `ctx_fetch_and_index` đã phải
đổi sang khoá ghép `(source,url)` sau một hồi quy va chạm (comment `server.ts:3296-3299`, commit `1f1243e`) — chính sách
nhãn không nhất quán giữa hai tool. Test "re-indexing same source replaces previous entry" tồn tại (`tests/store.test.ts:558`) [E2]
— tức hành vi ghi đè được test như tính năng. Kiểm: index 2 output JS > 100 KiB khác nội dung, `ctx_search` từ khoá chỉ
có ở output 1 → kỳ vọng 0 kết quả.

**Phát hiện F2 — containment chỉ phủ đường ctx_*.** Native Read/Grep/Bash/MCP ngoài chỉ nhận lời nhắc; PostToolUse không
thay output (`posttooluse.mjs:224`). Lời nhắc Read chỉ bắn 1 lần/phiên (`guidanceOnce`, `routing.mjs:121-144`). [E1]

## 2. Routing (spec §9.2)

| Câu hỏi | Trả lời từ mã |
|---|---|
| Ai quyết? | Hàm thuần `routePreToolUse(toolName, toolInput, projectDir, platform, sessionId, options)` (`routing.mjs:670`), gọi từ hook PreToolUse của từng host (Claude Code `hooks/pretooluse.mjs:170`; Codex `hooks/codex/pretooluse.mjs:30`; agy `hooks/antigravity-cli/pretooluse.mjs:30`; OpenCode in-process `src/adapters/opencode/plugin.ts:493`). [E1] |
| Khi nào? | Trước khi tool chạy. Claude Code đăng ký matcher Bash, WebFetch, Read, Grep, Agent, 3 tên ctx_* đầy đủ và `mcp__` (`hooks/hooks.json:28-108`). [E1] |
| Luật gì? | (1) cổng fail-closed tuỳ chọn `CONTEXT_MODE_REQUIRE_SECURITY=1` (`:680`); (2) Bash: chính sách deny/ask của người dùng → curl/wget → inline HTTP → build tool → allowlist lệnh "bounded" → ngưỡng byte tuỳ chọn → nhắc 1 lần (`:704-841`); (3) Read/Grep nhắc; (4) WebFetch deny; (5) Agent sửa prompt; (6) ctx_execute*/batch: kiểm deny + ghim `cwd` = projectDir trên Claude Code (`:920-997`); (7) MCP ngoài nhắc định kỳ. [E1] |
| Toàn cục hay theo agent? | Phần lớn **toàn cục**. Biến duy nhất theo agent là `options.mcpToolsAvailable`, và chỉ hook Claude Code truyền nó (`pretooluse.mjs:167-172`); Codex/agy/OpenCode không truyền → mặc định `true`. [E1] |
| Biết host không? | Có: `platform` đổi tên tool trong văn bản (`tool-naming.mjs:19-37`) và formatter đổi ngữ nghĩa (mục 9). [E1] |
| Biết capability không? | Một phần, và **mâu thuẫn** (F3 dưới). Cờ `PlatformCapabilities` của adapter (`src/adapters/types.ts:27-41`) **không được đọc ở runtime**: `grep -rn "capabilities\."` trong `src/` + `hooks/` = 0 kết quả; đã mở `claude-code/index.ts:68`, `codex/index.ts:309`, `opencode/index.ts:94` — chỉ là khai báo. [E1] |
| Biết quyền không? | Đọc `permissions.deny/ask/allow` dạng Claude Code từ `.claude/settings.local.json`, `.claude/settings.json`, `~/.claude/settings.json` (`src/security.ts:354-395`). Khớp chuỗi/regex, tách lệnh nối chuỗi (`splitChainedCommands`, `:174`). [E1] |
| Vượt được không? | Có, nhiều đường: (a) `isMCPReady()` false → mọi redirect thành passthrough (`routing.mjs:28-32`); (b) mọi native tool trừ curl/wget/HTTP/build/WebFetch chỉ bị nhắc; (c) code không phải shell trong `ctx_execute` chỉ bị quét bằng regex chuỗi literal (`security.ts:793-883`) — suy luận: ghép chuỗi `"su"+"do"` lọt; (d) security module không nạp được → fail-open + cảnh báo (`routing.mjs:385-443`). [E1] |
| Lấn quyền (overreach) không? | Có: (a) subagent `subagent_type: "Bash"` bị đổi thành `"general-purpose"` (`routing.mjs:911-913`) — đổi loại agent của người gọi sang loại có nhiều tool hơn; (b) trên agy, lời nhắc mềm (`context`) bị đổi thành **deny** (`formatters.mjs:253-256`, lý do: agy không nhận additionalContext) → lần Read đầu mỗi phiên bị chặn rồi phải thử lại. [E1] |

**Phát hiện F3 — "MCP sẵn sàng" là trạng thái của MÁY, không phải của agent đang gọi.**
`isMCPReady()` quét mọi sentinel `/tmp/context-mode-mcp-ready-<pid>` (`hooks/core/mcp-ready.mjs:73-108`): bất kỳ server
context-mode nào còn sống — kể cả của dự án khác, hay của **người dùng khác** (`EPERM` ⇒ coi là sống, `:93`), hay sentinel
mới chạm < 90 s (`:28,100`) — đều làm redirect có hiệu lực. Ngược lại, trong subagent Claude Code, hook đặt
`mcpToolsAvailable=false` cho MỌI subagent có `agent_id` hoặc `agent_type` (`pretooluse.mjs:167`) — kể cả subagent
general-purpose có thể dùng ctx_*. Cùng lúc, nhánh Agent vẫn tiêm vào prompt subagent lời dặn "dùng ctx_*" + bootstrap
ToolSearch (`routing.mjs:894-916`) **không** qua `mcpRedirect` (không kiểm MCP sẵn sàng, không kiểm capability).
Ba chỗ quyết capability với ba giả định khác nhau. [E1] Upstream tự ghi nhận: ADR-0016 nhánh `v2-design` nói "`agent_type`
có cả ở main thread khi khởi động `--agent <type>` … chỉ `agent_id` mới đánh dấu subagent" [E0] ⇒ suy luận: HEAD xếp nhầm
main thread `--agent` thành subagent.

**Phát hiện F4 — guidance mềm không gate theo capability.** Nhắc Bash/Read/Grep (`routing.mjs:840,854-866,871`) không đi
qua `mcpRedirect` ⇒ subagent không gọi được ctx_* vẫn được bảo "dùng ctx_execute". [E1]

## 3. Soạn prompt/context và chỗ đụng nhau (spec §9.3)

Các nguồn chỉ dẫn context-mode đưa vào phiên, đo bằng `node` chạy hàm thuần trong `hooks/routing-block.mjs`
(script `scratchpad/measure-blocks.mjs`, lệnh: `node measure-blocks.mjs <repo>`) [E4\*]:

| Nguồn | Khi nào | Kích thước đo | Mã |
|---|---|---|---|
| `ROUTING_BLOCK` (`<context_window_protection>` đầy đủ + `<ctx_commands>`) | SessionStart mọi nguồn (startup/compact/resume/clear) | **4631 B** | `hooks/sessionstart.mjs:166` |
| Khối subagent Claude Code (`includeCommands:false`, `toolSearchBootstrap:true`) | nối vào CUỐI prompt của MỌI lời gọi Agent | **4637 B** | `routing.mjs:904-915` |
| Khối subagent Codex (không bootstrap) | như trên trên Codex | 3327 B | — |
| `BASH_GUIDANCE` / `READ` / `GREP` / `EXTERNAL_MCP` | PreToolUse, 1 lần/phiên (MCP ngoài: mỗi 10 lần) | 681 / 405 / 505 / 628 B | `routing-block.mjs:82-96` |
| Directive compact + auto-injection | SessionStart `compact` | auto-injection ≤ 500 token ước lượng | `sessionstart.mjs:191-206`, `hooks/auto-injection.mjs:61` |
| Snapshot resume | sau compact | **không có ngân sách byte** ("maxBytes KEPT … IGNORED") | `src/session/snapshot.ts:30,470` |
| Cảnh báo security fail-open | SessionStart nếu security không nạp | — | `routing.mjs:468-491`, `sessionstart.mjs:175-183` |
| Mô tả tool MCP | luôn (tools/list) | mô tả `ctx_execute` dài nhiều đoạn có ví dụ | `server.ts:1660-1700` |

Chỗ đụng nhau đã thấy:

1. **Bản sửa #233 bị #724 ăn lại.** #233 bỏ `<ctx_commands>` khỏi khối subagent; #724 thêm `<deferred_tool_bootstrap>`.
   Đo tại HEAD: khối subagent (4637 B) **lớn hơn** khối phiên chính (4631 B). [E4\*]
2. **Hai bộ luật cùng chủ đề, độ mạnh khác nhau.** `CLAUDE.md` gốc repo (dành cho người bảo trì) dùng giọng
   `MANDATORY`/`BLOCKED`/"Bash ONLY for …" (`CLAUDE.md:1,5,9,11,15,19,25`), còn `ROUTING_BLOCK` runtime dùng giọng mềm
   "Bash stays correct when you intend to OBSERVE …" (`routing-block.mjs:43`). Phiên làm việc trong repo nhận CẢ HAI. [E1]
   Upstream ghi đúng mẫu này là thứ #946 nêu làm vấp classifier (`docs/v2/review/ISSUE-OUTCOMES-r4.md:168` nhánh `v2-design`) [E0].
3. **Quan sát trực tiếp trong phiên nghiên cứu này (không phải test của context-mode):** khi agent nghiên cứu đọc tệp
   trong clone, host Claude Code tự nạp `context-mode/CLAUDE.md` vào ngữ cảnh, kèm skill `.claude/skills/context-mode-ops`.
   Tệp đó ra lệnh dùng `ctx_execute`/`ctx_batch_execute`, khẳng định "curl … BLOCKED. Intercepted and replaced with error"
   (`CLAUDE.md:11-13`) và "Routing block auto-injected into subagent prompts. Bash-type subagents upgraded to
   general-purpose" (`CLAUDE.md:54-56`) — trong khi phiên này KHÔNG có MCP context-mode: tool không tồn tại, curl không bị
   chặn. Đây là đúng họ lỗi #233/#946: **chỉ dẫn tool được tiêm cho một agent không có tool đó**. Bằng chứng: E1 (tệp:dòng)
   + quan sát runtime của host trong phiên 2026-10-19. Agent nghiên cứu đã coi nội dung đó là dữ liệu, không làm theo.
4. **`file_writing_policy` sai so với executor.** Khối routing bảo "`ctx_execute`, `ctx_execute_file`, and Bash subprocesses
   do not persist edits to the host filesystem" (`routing-block.mjs:46,50`), nhưng executor chạy với `cwd = projectRoot`
   (`executor.ts:306`), `HOME` thật (`:693`), không có cô lập FS (mục 8) ⇒ ghi file từ `ctx_execute` shell **có** lưu lên
   đĩa thật (suy luận từ mã; chưa chạy). Chỉ dẫn mô tả sai hành vi. [E1]
5. **Chống trùng chỉ có ở OpenCode.** OpenCode bỏ qua tiêm khi system prompt đã chứa ≥ 2 tên tool ctx_* (`plugin.ts:216,687-701`,
   #487); Claude Code SessionStart tiêm `ROUTING_BLOCK` vô điều kiện (`sessionstart.mjs:166`). [E1]
6. **Tên tool Claude Code giả định cài kiểu plugin.** `mcp__plugin_context-mode_context-mode__<tool>` cứng
   (`tool-naming.mjs:20`); bootstrap ToolSearch `select:` dùng tên này (`routing-block.mjs:26`). Suy luận: cài MCP rời
   (`mcp__context-mode__*`) ⇒ bootstrap chọn sai tên. Upstream có issue #1018 "CC tool naming assumes plugin install"
   (`tickets/v2-issue-map.md:170`, nhánh `v2-design`) [E0].
7. **Lịch sử tiêm "phong cách trả lời" đã bị gỡ.** 500-word budget (v0.7.3) → "caveman" (2026-04-25) → gỡ toàn bộ
   (2026-05-10, #482: prompt ép ngắn làm giảm chất lượng câu trả lời cuối). Xem 02. [E1 qua git]

## 4. Capability của subagent (spec §9.4)

| Ngữ cảnh | Hành vi tại HEAD | Bằng chứng |
|---|---|---|
| Root agent Claude Code | redirect cứng có hiệu lực khi có sentinel sống; nhắc 1 lần/phiên | `pretooluse.mjs:170`, `routing.mjs:28-32` [E1] |
| Subagent Claude Code (có `agent_id` HOẶC `agent_type`) | `mcpToolsAvailable=false` ⇒ curl/HTTP/build/WebFetch **đi thẳng** (không chặn, không giới hạn byte); nhắc mềm vẫn bắn | `pretooluse.mjs:167-172`, test `tests/hooks/core-routing.test.ts:448-535` [E2] |
| Lời gọi Agent (cha → con) | nối khối routing 4637 B vào prompt, mọi `subagent_type`, kể cả agent bên thứ ba; `Bash` → `general-purpose` | `routing.mjs:894-916` [E1] |
| Subagent read-only / reviewer / background worker | **không có khái niệm**: không phân biệt loại subagent ngoài chuỗi `"Bash"` | `grep subagent_type` trong `hooks/` chỉ thấy `routing.mjs:895,911-912` [E1] |
| Host có hook nhưng không đổi được args (Codex < 0.141, Kimi, Kiro, agy) | modify → deny hoặc null; context → null/deny | `formatters.mjs:155-256` [E1] |
| Host không hook (antigravity IDE, Zed, Pi, OMP) | chỉ MCP; routing hoàn toàn dựa vào văn bản người dùng tự chép | capability `mcp-only` trong `src/adapters/{antigravity,zed,pi,omp}/index.ts` [E1] |

Không có đường "context gets ≤ budget" cho subagent: hoặc bị chặn với gợi ý tool nó không gọi được (trước #834), hoặc được
cho qua không giới hạn (sau #834). Upstream gọi thẳng hai lựa chọn này là "dead end" và "context floods" (ADR-0016,
nhánh `v2-design`) [E0].

## 5. Vòng đời session (spec §9.5)

| Bước | Mã | Ghi chú |
|---|---|---|
| create | `db.ensureSession(sessionId, projectDir)` tại SessionStart `startup` | `sessionstart.mjs:296-336` [E1] |
| touch/persist | PostToolUse/UserPromptSubmit/Stop ghi `session_events`; dedup 5 sự kiện gần nhất; > 1000 sự kiện ⇒ xoá sự kiện ưu tiên thấp nhất | `src/session/db.ts:639-642,1170-1200` [E1] |
| compact | PreCompact dựng snapshot; SessionStart `compact` tiêm directive + auto-injection ≤ 500 token | `hooks/precompact.mjs`, `sessionstart.mjs:191-258`, `auto-injection.mjs:61` [E1] |
| restore | snapshot "reference-based": mỗi mục kèm lời gọi `ctx_search` mẫu; không ngân sách byte | `snapshot.ts:463-500` [E1] |
| expire/cleanup | startup: `cleanupOldSessions(7)` + xoá sự kiện mồ côi | `sessionstart.mjs:306-331` [E1] |
| purge | `ctx_purge` (theo phiên từ #520) là cơ chế xoá duy nhất; `clear` không reset | `sessionstart.mjs:444`, `src/session/purge.ts` [E1] |
| crash/restart | server MCP tự thoát khi cha/ông chết (ppid poll) | `src/lifecycle.ts:81-120` [E1] |
| ContentStore cleanup | DB > 14 ngày bị xoá khi mở store; source > 14 ngày bị xoá | `server.ts:727-733` [E1] |

**Phát hiện F5 — bản sửa mất dữ liệu chỉ áp cho MỘT host.** Commit `4164d3de` (2026-05-10) sửa việc startup xoá
"sự kiện mồ côi" làm mất >1000 sự kiện thật sau mất điện (khe hở giữa `insertEvent` và `ensureSession`); bản sửa chỉ bảo
vệ session_id dạng UUID trong `hooks/sessionstart.mjs:326-331`. Tám hook SessionStart khác vẫn chạy câu DELETE vô điều kiện:
`hooks/codex/sessionstart.mjs:105`, `cursor/sessionstart.mjs:83`, `copilot-cli/sessionstart.mjs:78`,
`vscode-copilot/sessionstart.mjs:88`, `jetbrains-copilot/sessionstart.mjs:86`, `gemini-cli/sessionstart.mjs:86`,
`kiro/agentspawn.mjs:82`, và `kimi/sessionstart.mjs:99` qua `SessionDB.pruneOrphanedEvents()` (`db.ts:1718-1725`). [E1]
Mới nghi (chưa chứng minh): các host đó có cùng khe hở thời gian không — kiểm bằng cách ghi sự kiện cho session chưa có
`session_meta` rồi gọi hook startup của host đó, đếm sự kiện còn lại.

**Phát hiện F6 — danh tính phiên phía MCP là phỏng đoán.** `currentAttribution()` lấy session_id "mới nhất" trong DB dự án
(`server.ts:448-495`, câu `ORDER BY created_at DESC LIMIT 1` tại `:487`, cache 2 s). Flood-guard "theo agent" (#769) dùng
khoá này (`server.ts:2479-2485`). Suy luận: nhiều phiên/subagent song song cùng dự án ⇒ chunk gắn nhầm phiên và chung một
xô throttle. Kiểm: hai phiên song song cùng projectDir, mỗi phiên `ctx_index` rồi đọc cột `session_id` của chunk.

**Phát hiện F7 — phạm vi lưu trữ không đồng nhất.** SessionDB theo dự án + hậu tố worktree (`db.ts:297-386`); ContentStore
theo dự án, **không** theo worktree ("No worktree suffix — the FTS5 store is per-project", `db.ts:450`). Suy luận:
`ctx_search` ở worktree B thấy output index ở worktree A. projectDir lấy qua chuỗi dò: env theo host → `CONTEXT_MODE_PROJECT_DIR`
→ transcript Claude Code mới nhất trong 5 phút → PWD → cwd (`server.ts:597-656`); `_store` mở một lần theo projectDir
đầu tiên (`server.ts:699-705`). [E1]

## 6. Search/index FTS5/BM25 (spec §9.6)

| Mục | Mã | Bằng chứng |
|---|---|---|
| Bảng | `sources(label, chunk_count, …, file_path, content_hash)`, FTS5 `chunks` tokenizer `porter unicode61`, FTS5 `chunks_trigram` tokenizer `trigram`, `vocabulary` | `store.ts:465-505` [E1] |
| Trọng số | `bm25(chunks, 5.0, 1.0)` (tiêu đề ×5) | `store.ts:592-770` [E1] |
| Pipeline | refresh nguồn file cũ → RRF(porter OR + trigram OR) → rerank theo khoảng cách từ → nếu 0: sửa chính tả Levenshtein (≤1/2/3 theo độ dài) → RRF lại | `store.ts:1340-1394,141-148` [E1] |
| Escape query | xoá `'"(){}[]*:^~`, bỏ AND/OR/NOT/NEAR, bỏ stopword, bọc từng từ trong `"…"`; trigram yêu cầu từ ≥ 3 ký tự | `store.ts:90-128` [E1] |
| Lọc nguồn | `LIKE %source%` có escape wildcard (#646) | `store.ts:1110-1119` [E1] |
| Trùng lặp | replace theo nhãn (F1) | `store.ts:1046-1052` [E1] |
| Cũ | nguồn có `file_path`: so SHA-256, tự index lại khi search; nguồn > 14 ngày bị xoá theo tuổi (upstream #1079 nói xoá cả nguồn không đổi [E0]) | `store.ts:1411-1460`, `server.ts:730` [E1] |
| Đã xoá | chỉ qua `ctx_purge`/dọn theo tuổi | — |
| Hỏng/lớn/nhị phân | chunk > 4096 B tách theo đoạn; plain text bị ép cap từ #789 (`6f699fac`, 2026-06-13); fetch trả nội dung không phải JSON/HTML qua `resp.text()` ⇒ nhị phân (PDF, ảnh) bị decode UTF-8 và index như văn bản (suy luận từ `server.ts:3105-3108`) | [E1] |
| Phạm vi phiên | `project:` lọc qua allow-set session_id; chunk cũ `session_id=''` thấy ở mọi dự án | `store.ts:1356,1396-1405` [E1] |

## 7. Fetch/ingest (spec §9.7)

`ctx_fetch_and_index` chạy fetch trong một tiến trình con JS do executor sinh (`server.ts:3318-3324`):
SSRF guard trước khi fetch và lúc connect (DNS), redirect thủ công tối đa 5 bước (`:3013-3057`), cap 50 MiB qua
Content-Length và sau khi đọc (`:3066-3077`), JSON → pretty-print, HTML → Turndown + GFM, bỏ `script/style/nav/header/footer/noscript`
(`:3083-3108`), cache TTL 24 h theo khoá `(source,url)` (`:3118,3294-3313`).

| Lỗi tiêm | Hành vi tại HEAD | Bằng chứng |
|---|---|---|
| HTTP lỗi | `console.error("HTTP "+status)`, exit 1 → `isError: true`, kèm stderr thô | `server.ts:3081,3325-3337,3597-3606` [E1] |
| trang rỗng | `fetch_error reason:"empty"` → `isError: true` | `:3356-3358` [E1] |
| timeout | executor 30 s → exit ≠ 0 → isError | `:3323` [E1] |
| ngoại lệ parser | exception trong tiến trình con → stack trace trong stderr → trả về agent nguyên văn trong `Failed to fetch …: <stderr>`, nhưng **có** `isError: true` (không thành "thành công ngữ nghĩa") | `:3330-3337` [E1] |
| bảng khổng lồ / HTML hỏng | Turndown xử lý; không có cap riêng ngoài 50 MiB | [E1] |
| encoding lạ | `resp.text()` (UTF-8 mặc định của fetch) | suy luận |

Chưa chạy fault-injection nào (cấm cài phụ thuộc). Đề xuất test ở cuối.

## 8. An ninh — containment ≠ cô lập (spec §9.8)

| Lớp | Có không | Bằng chứng |
|---|---|---|
| **Context containment** | Có cho ctx_*: chỉ stdout/preview/kết quả search vào context | mục 1 [E1] |
| **Cô lập tiến trình** | Một phần: tiến trình con riêng, process group riêng (Unix), kill cả cây khi timeout/cap | `executor.ts:443-446,196-210` [E1] |
| **Cô lập FS** | **Không.** `cwd` = project root (shell), `HOME` thật, không mount namespace; `grep -rn "unshare\|bwrap\|sandbox-exec\|seccomp\|landlock\|firejail\|nsjail\|chroot"` trong `src/` + `hooks/` = 0 (đã mở `executor.ts:573-740` đọc xác nhận) | `executor.ts:306,692-693` [E1] |
| **Cô lập mạng** | **Không** cho `ctx_execute` ("full network access", tự quảng bá trong mô tả tool). Chỉ `ctx_fetch_and_index` có SSRF guard | `routing.mjs:771`, `server.ts:2985-3057` [E1] |
| **Cô lập env** | Kế thừa env cha, chỉ xoá denylist ~90 biến tiêm mã (BASH_ENV, NODE_OPTIONS, LD_PRELOAD, GIT_SSH_COMMAND…) + `BASH_FUNC_*` + `COMPlus_*`; **token/secret khác (vd `GITHUB_TOKEN`, AWS_*) đi vào tiến trình con** (suy luận từ denylist, `executor.ts:573-690`). Allowlist cũ 80 biến bị thay bằng denylist ngày 2026-03-15 (`0a8bb7b7`). | [E1] |
| **Thực thi quyền** | Mô phỏng `permissions.deny` của Claude Code bằng regex trên chuỗi lệnh; ngôn ngữ khác shell chỉ quét literal; `ctx_execute_file` giới hạn trong project root (#852, `server.ts:1169-1205`); security không nạp ⇒ fail-open trừ khi `CONTEXT_MODE_REQUIRE_SECURITY=1` | [E1] |
| **Cô lập provider** | Không áp dụng: context-mode không gọi LLM. Có đường gửi sự kiện ra ngoài **tuỳ chọn**: `hooks/platform-bridge.mjs` POST mọi sự kiện tới `platform_url` khi có `~/.context-mode/platform.json` với key `ctxm_*` (`:1-5,49-56,286`); mặc định không có tệp ⇒ không gửi. Server gọi npm registry kiểm phiên bản mới (`server.ts:777-790` (`fetchLatestVersion` tại `:780`)). | [E1] |

Tự mô tả trái nhau trong README: "spawns an isolated subprocess" (`README.md:1178`), "processed in complete isolation"
(`:1522`), nhưng cũng "still inherit the process's filesystem access … not a full OS sandbox" (`:1575`). [E0 vs E1]

## 9. Adapter nhiều host (spec §9.9)

17 `PlatformId` (`src/adapters/types.ts:474-493`), 3 paradigm: `json-stdio` (hook tiến trình riêng), `ts-plugin` (OpenCode,
OpenClaw: chạy trong tiến trình host), `mcp-only` (antigravity IDE, Zed, Pi, OMP). Nhận diện host: MCP `clientInfo.name` →
`CONTEXT_MODE_PLATFORM` → env/thư mục cấu hình (`src/adapters/detect.ts:367-430`). [E1]

| | Claude Code | Codex | OpenCode | antigravity-cli |
|---|---|---|---|---|
| Hook | 6 loại, `hooks/hooks.json` | 7 script `hooks/codex/*` | in-process `tool.execute.before/after`, `experimental.chat.system.transform`, `experimental.session.compacting` | Pre/Post/Stop |
| deny | `permissionDecision:"deny"` | như CC | `throw new Error` | `{decision:"deny"}` |
| modify Bash | **đổi thành deny** (CC bỏ qua `updatedInput.command`, `formatters.mjs:21-63`) | allow+updatedInput nếu `codex --version` ≥ 0.141.0, không thì deny (`hooks/core/codex-caps.mjs`, `formatters.mjs:172-195`) | `Object.assign(output.args, updatedInput)` | deny |
| modify Agent | allow+updatedInput (giữ) | như trên | như trên | deny |
| context (nhắc) | `additionalContext` | chỉ khi ≥ 0.141.0, không thì null | **ghi vào `output.args.additionalContext`** — tức nhét field lạ vào tham số tool (`plugin.ts:510-513`, dòng gán `:512`; suy luận: model có thể không bao giờ thấy) | **deny** kèm lời nhắc |
| ask | có | null (không hỗ trợ) | throw (như deny) | có |
| tiêm phiên | SessionStart `additionalContext` | SessionStart | `system.splice(1,0,…)` 1 lần/phiên, chống trùng | không (`sessionStart:false`) |
| mcpToolsAvailable | truyền theo `agent_id/agent_type` | không truyền | không truyền | không truyền |
| bootstrap ToolSearch cho subagent | có | không | không | không |

Kết luận phần này (E1): **cùng hàm routing, khác ngữ nghĩa thực thi**. Cùng một quyết định `context` là lời nhắc (CC), là
im lặng (Codex cũ, Kimi), là tham số lạ (OpenCode), là chặn (agy). Cờ `PlatformCapabilities` khai báo nhưng không ai đọc.
Bản sửa ở một host không lan sang host khác (F5; và hai bản formatter Claude Code song song `hooks/formatters/claude-code.mjs`
và `hooks/core/formatters.mjs` phải vá cùng lúc trong `4bc292fc`).

## Issue then chốt (IssueRecord, overlay §3)

**Giới hạn bằng chứng:** WebFetch tới `github.com/mksglu/context-mode/issues/{233,946,724}` và `pull/725` đều trả **HTTP 404**
khi không đăng nhập (đã dùng 4/6 lượt cho phép, dừng). Nội dung dưới đây lấy từ (a) commit message + diff + test trên
`main` [E1/E2] và (b) bảng issue-map chính upstream viết trên nhánh `origin/v2-design` (`tickets/v2-issue-map.md`,
`docs/v2/review/ISSUE-OUTCOMES-r4.md`, ADR-0004/0008/0016) [E0 — lời upstream, chưa đối chiếu trang issue].

```
IssueRecord {
  project: context-mode, issue_id: 233,
  title: INSUFFICIENT_EVIDENCE (trang 404); chủ đề theo commit: "omit ctx_commands from subagent routing block",
  date: fix 2026-04-13 (ngày mở: UNKNOWN), affected_version: < v1.0.76 (tag đầu chứa fix),
  symptom: "injecting ~400 tokens of user-facing help is wasted context" vào prompt subagent (commit 245cd8af),
  user_impact: CONTEXT_OVERHEAD ở mỗi lời gọi Agent,
  subsystem: PreToolUse Agent routing / routing-block,
  root_cause: một khối chỉ dẫn duy nhất cho mọi người nhận; không có khái niệm "người nhận dùng được gì",
  root_cause_confidence: cao (diff 245cd8af: createRoutingBlock(t,{includeCommands:false})),
  state: FIXED cho đúng triệu chứng; lớp vấn đề còn (khối subagent đo tại HEAD 4637 B > khối chính 4631 B),
  fixing_pr: UNKNOWN, fixing_commit: 245cd8af,
  regression_test: tests/hooks/core-routing.test.ts:538-582 (E2),
  later_regression: không hồi quy ctx_commands; overhead quay lại qua #724 (bootstrap),
  current_code_status: routing.mjs:893-916, routing-block.mjs:62-78,
  reproducibility: tái lập bằng gọi routePreToolUse("Agent",{prompt:"x"},…,"claude-code") và đo độ dài phần nối thêm,
  architectural_lesson: văn bản tiêm phải được soạn theo capability của người nhận, và có ngân sách,
  agent_cli_analog: (để agent khác đánh giá),
  proposed_agent_cli_regression_test: xem PROPOSED TESTS T1, T2 }

IssueRecord {
  project: context-mode, issue_id: 724,
  title: INSUFFICIENT_EVIDENCE (trang 404); chủ đề theo commit/comment: subagent được bảo dùng ctx_* nhưng tool là
    DEFERRED trong Claude Code (schema nạp qua ToolSearch) ⇒ "stalls" (routing.mjs:900-903),
  date: fix 2026-05-31, affected_version: < v1.0.152,
  symptom: subagent gọi ctx_* thất bại "tool not found"/InputValidationError rồi kẹt (routing-block.mjs:25),
  subsystem: Agent prompt injection × host deferred-tool loading,
  root_cause: chỉ dẫn giả định "tool có trong danh sách" = "tool gọi được ngay"; host Claude Code hoãn nạp schema MCP,
  root_cause_confidence: trung bình (comment mã + test; chưa đọc issue),
  state: FIXED trên main bằng special-case host; upstream v2 coi là "closed-by-design" vì bỏ hẳn tiêm (E0),
  fixing_pr: #725 (theo subject commit), fixing_commit: b1ba5cc9 (tác giả Ed Fricker),
  regression_test: tests/core/routing.test.ts:74-88 (E2) — chỉ kiểm CHUỖI có "ToolSearch"/"select:mcp__plugin_…", không kiểm
    subagent thật nạp được tool,
  later_regression: upstream ghi #1034 "self-identify subagent routing block" có INTRODUCED_BY = bản sửa #724
    (ISSUE-OUTCOMES-r4.md:362, E0); #1018 "CC tool naming assumes plugin install" (E0) ⇒ suy luận: chuỗi select: sai với cài MCP rời,
  current_code_status: routing.mjs:900-908 (isClaudeCode = !platform || platform==="claude-code"), routing-block.mjs:23-29,
  architectural_lesson: "tool có trên máy/đã đăng ký" ≠ "agent này gọi được ngay bây giờ"; capability (kể cả trạng thái nạp)
    phải được giải trước khi soạn chỉ dẫn,
  agent_cli_analog: (để agent khác), proposed_agent_cli_regression_test: T2 }

IssueRecord {
  project: context-mode, issue_id: 946,
  title (theo bảng upstream, E0): "Agent-prompt injection blocked by auto-mode classifier" — host Claude Code
    (tickets/v2-issue-map.md:226; map-I3.md:17); họ hàng #967 "Injected <context_window_protection> trips auto-mode classifier",
  date: UNKNOWN, affected_version: UNKNOWN (mọi bản có tiêm Agent, ít nhất từ v0.6.0 `9941db01`),
  symptom (E0): khối routing giọng mệnh lệnh (caps, MANDATORY/BLOCKED) nối vào prompt Agent làm classifier auto-mode của host chặn,
  user_impact: lời gọi subagent bên thứ ba bị host từ chối/đổi hành vi — PROMPT_INTERFERENCE,
  subsystem: Agent prompt injection,
  root_cause (E0, upstream tự nhận): "classifier tripping on injected prompt text is a symptom of the routing-block design itself",
  root_cause_confidence: trung bình (lời upstream; chưa thấy issue gốc),
  state: trên main = DESIGN_LIMITATION/OPEN — `git log --grep '#946'` trên main = 0 commit; tiêm vẫn vô điều kiện
    (routing.mjs:894-916); trên v2-design = kế hoạch xoá (ADR-0004 + D11/ADR-0008 item 8), CHƯA merge,
  fixing_pr/commit: không có trên main,
  regression_test: không có; upstream v2 ghi "REMOVED:ADR-0008" (không cần test vì bỏ tính năng),
  later_regression: upstream tự ghi REGRESSION_RISK MEDIUM: CLAUDE.md của repo vẫn giọng caps (ISSUE-OUTCOMES-r4.md:168) —
    xác nhận E1: CLAUDE.md:1-25 tại HEAD; và quan sát trong phiên này (mục 3.3),
  architectural_lesson: hạ tầng context tự nó thay đổi hành vi agent; không được viết vào prompt của agent mà mình không sở hữu,
  agent_cli_analog: (để agent khác), proposed_agent_cli_regression_test: T1, T3 }

IssueRecord (liên quan bắt buộc, overlay §5 "special-case routing cho subagent") {
  project: context-mode, issue_id: 794 (PR #834),
  symptom (commit 85f64d2c): subagent Claude Code bị deny WebFetch/curl và bảo dùng ctx_* mà nó không gọi được; #832 hang subagent foreground,
  root_cause: redirect không biết người gọi có ctx_* hay không,
  state: PARTIALLY_FIXED — sửa bằng cho qua toàn bộ (không ngân sách) khi có agent_id/agent_type; upstream sau đó có #1037
    "WebFetch denied for subagents without ctx_*" (E0, không xác định được phiên bản bị),
  fixing_commit: 85f64d2c (2026-06-21, v1.0.163), regression_test: tests/hooks/core-routing.test.ts:448-535 (E2),
  current_code_status: pretooluse.mjs:167-172 (chỉ Claude Code), routing.mjs:28-32 }
```

**Thiết kế HIỆN TẠI còn logic đặc biệt nào cho subagent? — Có, sáu chỗ (E1):**
1. `routing.mjs:904-908` — bootstrap ToolSearch chỉ cho Claude Code (special-case host).
2. `routing.mjs:911-913` — `subagent_type: "Bash"` bị đổi thành `"general-purpose"` (đổi capability của người gọi).
3. `routing.mjs:897` — đoán tên field prompt trong 6 tên (`prompt|request|objective|question|query|task`).
4. `routing.mjs:905-907` + `routing-block.mjs:62` — bỏ `<ctx_commands>` cho subagent (#233).
5. `hooks/pretooluse.mjs:167-172` — mọi payload có `agent_id`/`agent_type` ⇒ `mcpToolsAvailable=false` (chỉ hook Claude Code).
6. `hooks/core/formatters.mjs:30-47` — trên Claude Code, `modify` cho Agent giữ allow+updatedInput, `modify` cho Bash thành deny.

**Bất biến tổng quát rút ra (overlay §11; KHÔNG kết luận phơi nhiễm agent-cli):**
- INV-CM-1 Capability-first: capability của ĐÚNG agent đang thực thi (tool có thật, đã nạp, được phép) phải được giải
  TRƯỚC khi soạn chỉ dẫn hay quyết định chặn/redirect. Tool tồn tại toàn cục ≠ agent hiện tại dùng được (F3, #724, #794).
- INV-CM-2 Một nguồn capability: mọi nơi quyết theo capability đọc cùng một bản ghi; không để ba chỗ ba giả định (F3).
- INV-CM-3 Không viết vào prompt mình không sở hữu; văn bản tiêm có chủ, có ngân sách đo được, tắt được (#233, #946, #482).
- INV-CM-4 Kiểm enforcement bằng HIỆU ỨNG trên host, không bằng hình dạng JSON phát ra (`4bc292fc`: redirect Bash
  "silently no-op'd in production" trong khi test khẳng định shape allow+updatedInput).
- INV-CM-5 Sửa ở lớp chung: bản sửa dữ liệu/vòng đời ở một adapter phải đi qua mã dùng chung (F5).
- INV-CM-6 Containment ≠ isolation; tài liệu và chỉ dẫn không được hứa điều executor không làm (mục 3.4, mục 8).
- INV-CM-7 Danh tính (project/session/worktree/agent) giải một lần ở biên rồi truyền xuống; không đoán lại ở từng hệ con (F6, F7).
- INV-CM-8 Ghi đè theo nhãn chỉ khi nhãn mang danh tính nội dung/lời gọi; nếu không, mặc định là append (F1).

---

## Hợp đồng đầu ra (spec §63)

**SCOPE** — Kiến trúc hiện tại của Context Mode từ mã: §9.1–§9.9 của V3_SPEC, overlay §5 (họ issue subagent/routing),
IssueRecord cho #233/#724/#946 (+#794). Không đào fork, không đào issue ngoài 3 trang (404), không chạy test.

**REVISION AUDITED** — `main` @ `6006a3511ea2afde3e413cf635ba1da85ac8a8aa` (= `v1.0.169` + bundle). Tham chiếu E0 từ
`origin/v2-design` @ `7a99ece2` (2026-10-19).

**FILES/COMMITS/ISSUES INSPECTED** — Tệp: `hooks/pretooluse.mjs`, `hooks/core/{routing,formatters,mcp-ready,tool-naming,codex-caps}.mjs`,
`hooks/routing-block.mjs`, `hooks/sessionstart.mjs`, `hooks/{codex,antigravity-cli}/pretooluse.mjs`, 9 hook SessionStart theo host,
`hooks/platform-bridge.mjs`, `hooks/hooks.json`, `src/server.ts` (các đoạn đã dẫn), `src/executor.ts`, `src/store.ts`,
`src/security.ts` (mục lục), `src/session/{db,snapshot}.ts`, `src/lifecycle.ts`, `src/adapters/{types,detect}.ts`, 17 `index.ts`
(capabilities), `src/adapters/opencode/plugin.ts`, `CLAUDE.md`, `README.md` (dòng đã dẫn), `tests/hooks/core-routing.test.ts`,
`tests/core/routing.test.ts`. Commit: `245cd8af`, `b1ba5cc9`, `85f64d2c`, `4bc292fc`, `4164d3de`, `04569b12`, `3a45eb1c`.
Issue (qua commit/doc upstream, trang GitHub 404): #233, #724, #946, #794/#834, #967, #1018, #1034, #1037, #1055.

**VERIFIED FACTS** (E1 trừ khi ghi khác) — F1 ghi đè nhãn `execute:<lang>`; F2 containment chỉ cho ctx_*; F3 MCP-ready là
trạng thái máy (EPERM = sống); F4 nhắc mềm không gate capability; F5 bản sửa orphan-delete chỉ ở 1/9 hook SessionStart;
F7 ContentStore không theo worktree; cờ `PlatformCapabilities` không được đọc ở runtime; executor không có cô lập FS/mạng;
6 special-case subagent; kích thước khối tiêm (E4\*: 4631/4637/3327/681/405/505/628 B).

**INFERENCES** — ghi đè nhãn làm mất kết quả search; `ctx_execute` shell ghi được đĩa thật (trái `file_writing_policy`);
secret trong env cha lọt vào tiến trình con; bootstrap ToolSearch sai với cài MCP rời; main thread `--agent` bị coi là subagent;
attribution/flood-guard gộp sai khi song song; OpenCode `context` không tới model; nhị phân bị index như văn bản.

**UNKNOWNs** — nội dung gốc, ngày, trạng thái GitHub của #233/#724/#946 (404 không token); host Claude Code có dedupe hook
khi hai matcher (`mcp__plugin_…ctx_execute` và `mcp__`) cùng khớp không (upstream #1000 nói "double-fires", E0); các host
ngoài Claude Code có khe hở `insertEvent`/`ensureSession` không (F5); mọi số đo runtime (latency hook, bytes thật) — chưa chạy.

**FAILURE MODES** — ROUTING_FAILURE, CAPABILITY_MISMATCH (F3, F4, #724, #794), HOST_ADAPTER_MISMATCH (mục 9, F5),
CONTEXT_OVERHEAD (#233, khối 4.6 KB/Agent), PROMPT_INTERFERENCE (#946, #967, #482, mục 3.3), CONTEXT_INJECTION_FAILURE (#724),
FALSE_SUCCESS (`4bc292fc`), SESSION_CONTAMINATION/WORKTREE_CONTAMINATION (F6, F7), DELETION_PROPAGATION/ silent loss (F1, F5),
SECURITY_BOUNDARY_CONFUSION (mục 8, README tự mâu thuẫn), STALE_CONTEXT (snapshot không ngân sách).

**AGENT CLI RELEVANCE** — Câu hỏi để agent khác kiểm trên agent-cli (không kết luận ở đây): agent-cli soạn chỉ dẫn tool cho
subagent/reviewer từ danh sách tool TOÀN CỤC hay từ tool-set của chính agent đó? có nơi nào nối văn bản vào prompt của
agent/host không do agent-cli sở hữu? enforcement có được kiểm bằng hiệu ứng không? lưu output lớn có ghi đè theo nhãn không?
danh tính phiên/workspace giải một lần hay mỗi hệ con tự đoán? "sandbox"/"không người trực" có bị mô tả mạnh hơn thực tế không?

**PROPOSED TESTS** (ứng viên cho agent-cli, overlay §15) —
T1 restricted-subagent prompt: tạo subagent có tool-set hẹp (không có tool X); assert prompt/chỉ dẫn của nó KHÔNG nhắc X và
không vượt ngân sách byte cố định. T2 deferred/unloaded tool: tool đăng ký nhưng chưa nạp cho agent ⇒ hoặc nạp trước, hoặc
không được nhắc. T3 third-party prompt untouched: lời gọi tới agent/host bên thứ ba ⇒ byte prompt gửi đi == byte người dùng viết.
T4 enforcement-by-effect: chặn một lệnh ⇒ assert lệnh KHÔNG chạy (side-effect file), không chỉ assert JSON phản hồi.
T5 label collision: hai output lớn liên tiếp cùng loại ⇒ cả hai vẫn truy xuất được. T6 cross-adapter fix parity: một bản sửa
vòng đời ⇒ chạy cùng kịch bản trên mọi adapter. T7 worktree scope: index ở worktree A, search ở worktree B ⇒ kỳ vọng tách.

**NO-CHANGE FINDINGS** — FTS5 dual-tokenizer + RRF + fuzzy là thiết kế đơn giản, có test (`tests/store.test.ts`); không thấy
lý do đề xuất semantic search từ audit này. SSRF guard + cap 50 MiB + redirect thủ công của fetch hợp lý. Cap stream 100 MiB
+ kill cây tiến trình là đúng mức cho executor. Không thấy gì cần "sao chép" nguyên văn.

## Kiểm lại của điều phối (19/10)

- Khớp: 8 hook SessionStart khác vẫn xoá sự kiện mồ côi vô điều kiện — 7 hook gọi thẳng `DELETE FROM session_events WHERE
  session_id NOT IN (SELECT session_id FROM session_meta)` (codex, cursor, gemini-cli, copilot-cli, vscode-copilot,
  jetbrains-copilot, `kiro/agentspawn.mjs`), `kimi/sessionstart.mjs:99` gọi `db.pruneOrphanedEvents()` chạy đúng câu đó
  (`src/session/db.ts:1718-1725`). Điều phối từng ghi "7, không phải 8" vì grep chuỗi nguyên văn bỏ sót đường gọi qua hàm;
  đã rút lại.
- Khớp: `hooks/pretooluse.mjs:167` (`isSubagentContext = input.agent_id != null || input.agent_type != null` →
  `mcpToolsAvailable: !isSubagentContext`); `origin/main..origin/v2-design` = 737 commit.
