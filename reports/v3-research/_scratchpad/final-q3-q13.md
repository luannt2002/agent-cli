# Q3–Q13 báo cáo cuối V3 (chỉ tổng hợp từ báo cáo nghiên cứu có sẵn)

Quy ước: E0 lời khai/tiêu đề issue/thân commit · E1 đọc mã/diff · E2 có test (chưa chạy) · E3 chạy với stub. "T#" trong Q5/Q9/Q13 là ca test agent-cli theo MAP/FIT; các đề xuất T1–T17 nằm trong CM01/CM02b/CM03 là đánh số riêng của báo cáo Context Mode, không dùng ở đây. [MAP §0]

## Q3. Context Mode thực sự giải được gì?

- Chứa output của đường `ctx_*`: chỉ stdout, preview hoặc kết quả search vào context; `ctx_execute` không `intent` mà stdout > 100 KiB thì index FTS5 và trả con trỏ; có `intent` và output > 5 000 B thì BM25, trả tiêu đề + dòng đầu ≤ 120 ký tự của ≤ 5 section (E1; chưa chạy). [CM01 §1]
- Bỏ smartTruncate, index hết: sửa lỗi #197 (cắt phần giữa trước khi index làm mất dữ liệu vĩnh viễn); báo cáo đánh giá quyết định này hợp lý về dữ liệu, nhược điểm nằm ở snapshot không đặt lại ngân sách (E1 commit `ee628da5`). [CM02 Thời kỳ 3]
- Tìm kiếm: FTS5 kép porter + trigram, BM25, RRF, sửa chính tả Levenshtein; thiết kế đơn giản, có test (E2); audit không thấy lý do đề xuất semantic search. [CM01 Hợp đồng đầu ra]
- Fetch/executor: SSRF guard, redirect thủ công tối đa 5, cap 50 MiB, stream cap 100 MiB + kill cây tiến trình được đánh giá "đúng mức" (E1); chưa chạy fault-injection nào. [CM01 §7]
- Chặn cứng một số đường lấy dữ liệu của native tool (curl/wget, inline HTTP, build tool, WebFetch) và redirect sang `ctx_*` (E1); số "bytes avoided" là hằng số 8192/16384, không đo thật. [CM01 §1]
- Giữ trạng thái qua compact: SessionDB + snapshot "reference-based", tiêm directive + auto-injection ≤ 500 token ước lượng (E1). [CM01 §5]

## Q4. Context Mode không giải được gì?

- Containment chỉ phủ đường `ctx_*`: Read/Grep/Bash/MCP ngoài chỉ nhận lời nhắc, PostToolUse không sửa output, nhắc Read chỉ bắn 1 lần/phiên (E1, F2). [CM01 §1]
- `ctx_execute` không `intent` mà stdout ≤ 100 KiB thì trả nguyên stdout vào context (E1). [CM01 §1]
- Không có cô lập FS, mạng, env: `cwd` = project root, `HOME` thật, env chỉ bỏ denylist ~90 biến nên secret khác như `GITHUB_TOKEN` đi vào tiến trình con (suy luận từ denylist, chưa chạy); README tự mâu thuẫn "isolated" và "not a full OS sandbox". [CM01 §8]
- Capability của agent/subagent: ba chỗ quyết với ba giả định khác nhau (F3, E1); không có khái niệm subagent read-only/reviewer ngoài chuỗi `"Bash"` (E1); cờ `PlatformCapabilities` khai mà không ai đọc. [CM01 §2] [CM01 §4]
- Mất dữ liệu âm thầm: F1 nhãn `execute:<language>` ghi đè nhau (suy luận từ mã, chưa chạy); F5 bản sửa `4164d3de` chỉ ở 1/9 hook SessionStart, 8 hook còn lại vẫn DELETE vô điều kiện (E1), việc cùng khe hở thời gian trên các host đó là "mới nghi". [CM01 §1] [CM01 §5]
- Danh tính phiên/worktree: F6 `currentAttribution()` lấy session "mới nhất", F7 ContentStore không theo worktree (cơ chế E1, hệ quả suy luận, chưa chạy). [CM01 §5]
- Cổng "MCP sẵn sàng": 4 vòng sửa (#230, #347, #354, #844) rồi #1055/#1053 mở sau vòng cuối; v2 bỏ cổng mà không thay (xác nhận E1 commit/mã + E0 tiêu đề). [CM03 §2.10]
- Snapshot resume không có ngân sách byte (E1); upstream có #1022 "advertises <2KB, injects ~196KB" (E0). [CM02 Thời kỳ 3]

## Q5. Họ issue nào của Context Mode quan trọng với agent-cli?

- Giới hạn bằng chứng: CM03 là bản nháp offline; 172/173 issue tạo sau khi `main` ngừng phát triển, giao với commit fix trên `main` = 0, và "nguyên nhân gốc" chỉ lấy từ bản đồ upstream là E0 yếu. [CM03 §0]
- CAPABILITY_MISMATCH (9 issue; chuỗi #233 → #724 → #1034, #794 → #1037) ↔ họ F05: ĐO ĐƯỢC T5, T6 (E3) trên agent-cli (mcp-instructions và todo reminder không đọc `agentTools`); ứng viên POC P2. [MAP F05]
- FALSE_SUCCESS (18 issue; tiền lệ `4bc292fc` test khớp hình dạng JSON mà redirect no-op ở production) ↔ F14: ĐO ĐƯỢC T15 (thu hẹp, unit), ĐANG ĐO T19. [MAP F14]
- ROUTING_FAILURE (21 issue; regex curl #63 → #1106/#1221) ↔ F06: SUY LUẬN (Bash tự duyệt theo tiền tố, BlockFunc theo mẫu), ĐANG ĐO C9. [MAP F06]
- HOST_ADAPTER_MISMATCH (39 issue, họ lớn nhất, phân loại tay theo tiêu đề) ↔ F09: SUY LUẬN dạng "nhiều bản sao" (local so với server, ba cấu hình discovery); V3 test T17 (thêm nhánh local), T24. [MAP F09]
- PROMPT_INTERFERENCE (16 issue; #946, #967 "~40% of subagent spawns" là E0) ↔ F07: KHÔNG (E1) với prompt giao việc truyền nguyên văn; SUY LUẬN với context file của repo lạ; T8. [MAP F07]
- ZOMBIE_OPERATION (14) và STARTUP_RACE (5) ↔ F11 và F10: F11 KHÔNG cho lớp "kẹt mãi" (E1), ĐANG ĐO T4, T23, thêm T28; F10 SUY LUẬN S5, S7, ĐANG ĐO T20, thêm T25. [MAP F11] [MAP F10]
- DELETION / mất dữ liệu (9 issue; bốn đường dọn đều từng xoá dữ liệu sống) ↔ F17: SUY LUẬN (xoá session không xoá session con, `session delete` qua CLI không kiểm bận), ĐANG ĐO T23, T10 chưa viết. [MAP F17]

## Q6. Fork/nhánh nào (CM và MEM) rẽ hướng có ý nghĩa, và vì sao?

- CM `origin/v2-design` (737 commit chưa vào `main`, 2026-10-14 → 10-19): viết lại song song, 18 ADR; bỏ tiêm routing vào prompt, một DB, capability của subagent giải ở một module, index append-only (F1 biến mất), thoát theo stdin EOF. Lý do theo upstream (E0): classifier host chặn khối tiêm (#946, #967), 13 bản routing lệch nhau, ~10 issue ABI native; mã khớp ADR mức "LÀM" cho 6/18, "MỘT PHẦN" cho 10/18 (E1). [CM02b §0] [CM02b §(a)]
- Nhưng v2 bỏ `isMCPReady` mà không thay (V2-F1, E1), và văn bản hint/deny case C/`more:` vẫn nêu `ctx_*` cho subagent không có (V2-F2, E1+E2); hai danh tính phiên là "mới nghi" (V2-F3). [CM02b Phát hiện mới trên v2]
- Tình trạng v2: M7 tự ghi "not done" (GS5/GS10/GS11 FAIL, E0 tự báo); không thấy dấu hiệu merge vào `main` (ADR-0011: `main` đóng băng, v2 phát hành qua nhánh `release`, chưa có trong `git branch -r`). [CM02b §(e)]
- Fork ngoài của Context Mode: Nguồn không trả lời (đã soát CM01, CM02, CM02b, CM03; các tệp đều ghi không đào fork). Chỉ có phương án cộng đồng dạng PR chưa merge, ví dụ #1272 chuyển chỉ dẫn subagent sang sự kiện `SubagentStart`. [CM03 §5]
- MEM `tt12b/claude-mem` (DEPLOYMENT_VARIANT / ARCHITECTURAL_FORK, E0): tóm tắt theo chu kỳ N phút thay cho một lời gọi provider mỗi sự kiện, watermark chỉ tiến theo phần đã nằm trong prompt, claim bỏ dở được thu hồi về outbox; vì chi phí free-tier và con trỏ tiến trước kết quả. Quyết định: ADAPT_IDEA (ứng viên, chưa audit agent-cli). [MEM04 §2.1]
- MEM `rodboev` (SCOPE_FIX, #2882 monorepo mất memory): heuristic khoá repo-relative đảo chiều ngay trong cùng chuỗi (fork bỏ nhánh "nested package roots", edge giữ); không đổi kiến trúc. Quyết định: IGNORE heuristic, giữ bài học "khai báo tường minh thắng suy đoán". [MEM04 §2.2]
- MEM nhánh upstream rẽ hướng nhưng không merge: `branch-memory` (lọc theo tổ tiên git; PR #1246 đóng "Closing during backlog consolidation", lý do không làm lại là UNKNOWN), `canary` (bản sửa #1911 "default project" chỉ nằm ở đây), `bugfix/security-hardening` (khung "unverified"), `enhancement/supersession`. [MEM04 §3.2]
- MEM dòng `community-edge` (tag `archive/community-edge`) là nơi upstream đặt thay đổi danh tính rủi ro; khoá chuẩn `107ef946e` trên `main` bị revert sau ~14 giờ (suy luận: đổi danh tính bị đẩy khỏi stable). [MEM04 §3.3]

## Q7. claude-mem thực sự giải được gì?

- Đường ống hoạt động đầy đủ (E1): hook ghi spool bền → worker → observer LLM → SQLite + FTS5/Chroma → `ContextBuilder` fit 10.000 ký tự → `additionalContext`; hook lỗi bất ngờ thì no-op + exit 0, không chặn người dùng. [MEM01 §1]
- Hook không còn chặn người dùng: trước kia exit 2 gây vòng chặn (#2966, #3481, #3523), nay `grep -rn "process.exit(2)" src` = 0 ngoài test. [MEM03 §2.2] [MEM03 Hợp đồng đầu ra]
- Ngân sách và hợp đồng đầu ra có một chủ duy nhất: `ContextBudget` (có test E2 `context-budget-3802.test.ts`), `hook-io` một lối stdout, `hardened-options`, SSE `/api/ready`; từ khi các chủ này xuất hiện chưa thấy hồi quy (chưa đo dài hạn, các mốc mới từ 09-10/2026). [MEM03 §17] [MEM01 §7]
- Cold start: SSE `/api/ready` (`starting → db_ready → routes_ready → ready|failed`) cộng context cache trên đĩa, có test (E2), thay chuỗi chỉ nâng timeout. [MEM03 §8]
- NO MEMORY tách được khỏi NOT READY/FAILED ở mức mã (E1). [MEM01 §5]
- Observer chạy không có tool và không persist session (`--no-session-persistence`), có test E2; đây là mục "không đổi" của báo cáo. [MEM01 §3] [MEM03 Hợp đồng đầu ra]
- Giảm mất memory do đổi khoá: phía đọc hợp nhất khoá (`projectReadKeys`, `merged_into_project`, `COLLATE NOCASE`); upstream chọn "hợp nhất lúc đọc" thay cho khoá chuẩn lúc ghi (suy luận của báo cáo). [MEM01 §4.4] [MEM02 §3]
- Khung "data, not instructions" có ở các đường search, progressive search, corpus, TeamBrain, Grok bot index (E1). [MEM01 §11]

## Q8. claude-mem không giải được gì?

- Danh tính project: không có resolver duy nhất, #3608 OPEN, còn ít nhất 6 đường vòng (A3-A8), doctor không có hàng danh tính, lần thử khoá chuẩn `107ef946e` bị revert sau ~14 giờ (E1). [MEM01 §4.4]
- Thiếu `project` ở MCP search/timeline/get_observations/mem_search thì tìm toàn cục (#3764, hồi quy của #1911), không có bản sửa trên `main`; test `checkout-search-scope.test.ts:21-24` khoá hành vi đó (E1, E2). [MEM01 §12] [MEM03 §2.3]
- Không có danh tính branch (E1; `grep abbrev-ref src` = 0); thử nghiệm `branch-memory` không merge. [MEM01 §4.2] [MEM04 §3.2]
- ACCEPTED ≠ DURABLE còn hở: spool coi là đã giao khi sự kiện vào buffer RAM; worker chết thì mất; đường replay transcript tự động cho host Claude Code là "mới nghi: chưa thấy" (cách kiểm: giết worker sau `markHandedOff`, đếm `observations` và `tool_uses`). [MEM01 §3]
- NOT READY và RETRIEVAL FAILED gộp thành cùng một khối rỗng với model; SAI SCOPE trông y như NO MEMORY (E1). [MEM01 §5]
- SessionStart, kênh tiêm chính, không có khung untrusted; thử nghiệm đối kháng "chưa làm". [MEM01 §11]
- Không lọc theo session ở SessionStart: maintainer coi trộn observation giữa session là "working as designed" (#2909, PR #2919 đóng không merge, E0). [MEM03 §7]
- Chưa đo: số lời gọi LLM, latency, chi phí, tốc độ tăng log; log hằng ngày không thấy giới hạn kích thước/số ngày giữ (E1); `main` không có eval lợi ích memory (nhánh `evals-locomo` là prototype, F1 trung bình 0,041 trên 20 câu). [MEM01 §3] [MEM01 §9] [MEM04 §3.2]

## Q9. Lỗi scope/lifecycle/worker nào của claude-mem quan trọng (với agent-cli)?

- Danh tính đổi luật không migration (#2663 → #2882, `daf6d9dc0`; 23 lần đổi luật, issue là E0) ↔ F01: agent-cli ĐO ĐƯỢC T2, T3, T1 (thu hẹp về symlink-vào-thư-mục-con), ĐANG ĐO T4, C5/S10. [MEM02 §3] [MAP F01]
- Thiếu scope thì toàn cục (#1911 → #3764) ↔ F04: KHÔNG ÁP DỤNG hôm nay vì agent-cli chưa có đường truy hồi tri thức; T30 là cổng thiết kế chỉ viết khi V3 thêm truy hồi. [MAP F04]
- Nhiễm session/agent/tác giả (#2909/#2976, #3274 ~76% observation tiêm là của subagent theo người báo, #4215 sửa 2 lần) ↔ F02: KHÔNG thấy đường (đọc mã, chưa có test); T26 mới. [MEM03 §7] [MAP F02]
- Cold start 4-5 lần (#1341 → #1505 → #1785 → #2117 → #2795; năm bản sửa đầu không có tệp test, E1) ↔ F10: SUY LUẬN S5, S7, ĐANG ĐO T20, thêm T25. [MEM03 §8] [MAP F10]
- ACCEPTED ≠ DURABLE (`markHandedOff` trước khi bền; #3653 13 observation từ ~44k sự kiện là E0) ↔ F13: ĐO ĐƯỢC T11 (`send` rc 0 rồi SIGKILL, 7/8 lần mất tin user) và T13; thêm T14. [MEM03 §9] [MAP F13]
- Liveness worker: #3603 (plan-15) OPEN với 5 tín hiệu liveness; chuỗi reclaim `fc5f15843` → `8009b0dab` → `cf0c7cdcc`, mỗi bước sửa lỗi do bước trước sinh ra (E1) ↔ F11: KHÔNG cho lớp "kẹt mãi" ở các đường đã đọc (E1, chưa đo lại), ĐANG ĐO T4, T23, thêm T28. [MEM03 §10] [MAP F11]
- Retry/log storm (#4087 770k dòng/111 MB, #3928 8,9M lỗi/3,8 GB là E0; lý do gỡ hàng đợi bền: "that was the retry storm") ↔ F12: KHÔNG cho retention log (lumberjack 10 MB, 30 ngày), SUY LUẬN `DropCount` không ai đọc; T29 mới. [MEM03 §11] [MAP F12]
- Observer tự bắt phiên của chính nó (#3425, ≥3 lớp chặn) ↔ F03: SUY LUẬN (CLI con giao việc nạp settings/hook người dùng); T27 mới (phép đo, chưa phải cổng). [MEM03 §16] [MAP F03]

## Q10. Fork nào đổi kiến trúc của claude-mem một cách có ý nghĩa?

- Chỉ `tt12b/claude-mem` được xếp ARCHITECTURAL_FORK trong 13 fork đã soi (1 kiến trúc, 1 sửa scope, 7 nguồn PR đã hấp thụ, 2 mirror, 2 không đủ bằng chứng). [MEM04 Hợp đồng đầu ra]
- Thay đổi (E0, từ README + thân commit): từ "một sự kiện → một lời gọi" sang "cửa sổ thời gian → một lời gọi", thêm outbox + watermark chỉ tiến theo phần đã nằm trong prompt, đệm sự kiện hook tại chỗ khi server sập. [MEM04 §2.1]
- Chi phí và rủi ro: Postgres + Valkey + 4 container, dashboard không auth (README tự cảnh báo), độ trễ memory bằng chu kỳ N phút (suy luận); không thấy test, không có runtime evidence; base của fork là suy luận (dựng trên server-beta). [MEM04 §2.1]
- Tương đương upstream: gom lô opt-in `d55d20f7e` (mặc định batch=1), HookSpool bền `fe1709c30`, spool tới worker từ xa; báo cáo không có bằng chứng upstream lấy từ fork (tiến hoá song song). [MEM04 §2.1]
- `rodboev` không đổi kiến trúc (thêm heuristic vào resolver chung), quyết định IGNORE heuristic. [MEM04 §2.2]
- `escooterclinic` (một renderer, nhiều nguồn hàng) và `surfingdoggo` (migration dựa bằng chứng checkout đã ghi nhận) có ý tưởng kiến trúc nhưng đã vào upstream: OBSOLETE, bài học giữ lại. [MEM04 §2.3]
- Kết luận của báo cáo: không fork nào trong số đã soi giải được bài danh tính bằng một resolver chuẩn. [MEM04 Hợp đồng đầu ra]
- Giới hạn bằng chứng: 24/25 SHA fork lấy từ timeline không có trong clone; ba trang fork không hiện ahead/behind nên chưa đếm commit chỉ-có-ở-fork. [MEM04 §1]

## Q11. Hindsight đổi gì kể từ audit trước?

- Delta `e00154496..5331da2f4`: 48 commit, tất cả ngày 2026-10-16, 327 tệp, +18.846/-2.603 dòng (đo bằng `git diff --shortstat`); 47 commit của Nicolò Boschi, 1 commit blog. [HS02 §0]
- `hindsight-integrations/` không đổi dòng nào (0 tệp); ba commit "codex/claude-code" là provider LLM phía server, không phải plugin hook của coding agent. [HS02 §0]
- Worker và hàng đợi không đổi ngữ nghĩa: `poller.py` chỉ đổi đồng hồ sang `perf_counter()`, vẫn không có lease hay heartbeat. [HS02 §0]
- Phân nhóm 48 commit: Retain 6, Scope/tag/quyền 5, Control plane 5, Mental models 4, Transfer/import 4, Provider coding-agent 4, Provider khác 4, Worker/quan sát 3, Helm/CI 3, Consolidation 2, Reflect 2, Recall 2, Documents API 2, Khác 2. [HS02 §2]
- Sửa lỗi: #5371 (`89594d22d`), #5448 (`c7acb488a`), #5228 giảm nhẹ bằng prompt (`0433604c1`); bộ lọc tag gom về một luật (`3f8fe203f`, #5417). [HS02 §0] [HS02 §3.4]
- Thêm tính năng ở đúng phần SE 28 đã REJECT: reflect "decision-orchestrated" (`198947912`, +2.988 dòng), triage consolidation bằng decision model (`0d8e58b3e`, bật mặc định khi có cấu hình), full refresh theo lịch cho mental model; số eval upstream nêu (ví dụ trang stale 10/16 → 3/16) là E0 với agent-cli. [HS01 §5]
- Số đếm mốc đổi: `memory_engine.py` 23.477 → 23.784 dòng, migration 111 → 113, tệp test API 631 → 648, system story 79 → 87 (đo `wc -l`, `git ls-tree`). [HS02 §5]
- Không commit nào trong delta mâu thuẫn với 7 bất biến SE 28; không có bằng chứng mới đổi quyết định REJECT/DEFER; chưa chạy lại các kết quả E3 của audit cũ. [HS01 §4] [HS01 §5]

## Q12. Kết luận nào của audit Hindsight trước đã lỗi thời?

- #5371 "OPEN, tái lập ở HEAD" nay FIXED ở parser chung và 6 lối vào (`89594d22d`); còn một parser inline ở `list_mental_models`. [HS02 §5]
- #5448 `allowed_bank_ids` khai báo mà không ép nay FIXED bằng cách xoá trường và chặn `dry-run-extract` (`c7acb488a`). [HS02 §5]
- #5228 delta refresh để lại câu cũ nay PARTIALLY_FIXED bằng một dòng footer trong prompt (`0433604c1`), phụ thuộc LLM; số eval của upstream là E0. [HS02 §5]
- H-26 `based_on` = mọi memory đã recall: PARTIALLY, một nguồn gây ra (fact qua `expand`) đã gỡ (`45616ffe3`), nhánh "rỗng thì lấy tất cả" vẫn còn. [HS02 §5]
- Mô tả cursor coding-agents (audit §12.7, SE 24 H-29..H-31) thiếu ngay tại revision được audit: không nhắc trường `ops` của bản sửa #5251 (PR #5508, `d3f0001ed` là tổ tiên của `e00154496`); đây là bỏ sót, không do delta. [HS02 §5] [HS01 §7]
- Mô tả pipeline đã đổi: consolidation có thêm bước triage, reflect có chế độ fast/decision-orchestrated, delta retain có luật nhường streaming cho document viết dở; danh sách provider và mặc định codex đổi, số đếm §1/§8 lệch (tổng số provider "chưa đếm lại"). [HS02 §5]
- Bổ sung/sửa thêm từ họ issue mới: #3989 có gốc ở server (append chia slice làm cụt `original_text`), #3136 là PR chứ không phải issue, và consolidation chỉ "hội tụ" cho kiểu áp nửa vời (#5054 đóng "by design"). [HS03 §4]
- Không lỗi thời (delta không chạm vùng mã; grep diff rỗng, không chạy lại E1/E2/E7): 9/10 kết luận chính, gồm không lease (H-01), #5582, `ExcludeNoneRoute`, `run_consolidation` trả số 0, FK đa tenant, mặc định mở, recall "4 song song". [HS01 §3]

## Q13. Lỗi mới nào của Hindsight quan trọng?

- #5735 (mở 18/10): probe phiên bản thất bại bị cache âm vĩnh viễn ở client coding-agents (`ca/hindsight.ts:540-546`, E1); hermes đã sửa cùng lỗi #5221 ngày 12/10 nhưng bản sao không được sửa, 6 ngày sau bị báo lại. Liên hệ F09 (bản sửa không lan sang bản sao) và F10 (T20, T25). [HS03 F1] [MAP F10]
- #5251 PARTIALLY_FIXED với 6 chỗ hở (a)-(f) (E1, chưa chạy): `retain_ok` ghi ngay lúc accept, `settleOps` chỉ chạy ở write-back kế tiếp của cùng session, 404 bị coi là xong, cursor ở `tmpdir()`; chuỗi cursor phình 4 → 8 trường. Liên hệ F13: agent-cli ĐO ĐƯỢC T11, T13, thêm T14. [HS02 §6.1] [MAP F13]
- #5582 (retain trả 200 với 0 memory, OPEN) và #5589 (883 document 0 unit theo người báo, OPEN): cùng phễu "rỗng → completed", mỗi cái có bản sửa ở fork mà không có PR upstream (E0). Liên hệ F14: ĐO ĐƯỢC T15 (thu hẹp), ĐANG ĐO T19. [HS03 F7] [MAP F14]
- #5728 (mở 18/10): refresh mental model ghi báo cáo "nothing found" đè lên trang tốt; HEAD có test ghim hành vi "trang anh em là nguồn" (`test_mental_models.py:3576`), nên là xung đột thiết kế. Chuỗi tag scope #5030 → #5371 → #5417 trong 15 ngày. Liên hệ F04: KHÔNG ÁP DỤNG hôm nay, T30 cổng thiết kế; agent-cli đã đúng ở chỗ tóm tắt rỗng không thay lịch sử. [HS03 F4] [MAP F04]
- #5542 (#1751 lặp sau ~4,5 tháng), #5555 còn mở: hook của host lọt vào tiến trình con LLM; có 4 cơ chế chống đệ quy độc lập, server không truyền `HINDSIGHT_DISABLE_HOOKS` dù hook của Hindsight đọc biến này (suy luận). Liên hệ F03: SUY LUẬN với tool giao việc của agent-cli, T27 (phép đo). [HS03 F2] [MAP F03]
- Dấu "đã consolidate" gánh nhiều nghĩa: #5054 (48% fact bị đóng dấu mà không được biểu diễn, theo người báo) đóng "by design"; triage "chatter" thêm một lối đóng dấu; #5731 reset bị chặn khi `deleted_count = 0`; #5434 `failed` ≠ `superseded` (OPEN). Liên hệ F17: SUY LUẬN, ĐANG ĐO T23, thêm T10. [HS03 F5] [MAP F17]
- #2413 (đóng not planned): `/health` chỉ đo kết nối DB nên xanh khi hàng đợi kẹt, và test hồi quy bị formatter vô hiệu, không chạy trong CI (E1 `00968a1ce`); việc thu hồi vẫn theo `worker_id` (PR #3779 lease bị bác). Liên hệ F11: KHÔNG cho "kẹt mãi" ở agent-cli (E1), ĐANG ĐO T4, T23, thêm T28. [HS02 §6.2] [MAP F11]
- #5713 (tiêm một lần mỗi phiên, không làm mới) và #5721 (tìm trang không có ngưỡng liên quan) đều OPEN, số đo là của người báo (E0). Liên hệ F15: SUY LUẬN, test T8; mọi trạng thái mã ở `5331da2f4`, upstream đã chạy tiếp lên 1.0.0 nên chưa kiểm. [HS03 F8] [MAP F15]
