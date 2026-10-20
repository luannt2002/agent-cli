# FINAL V3 AUDIT — agent-cli

**Trạng thái đề xuất (spec §77, chọn đúng một): `V3 PARTIALLY ADOPTED — VERIFIED`** — CHỐT sau khi chạy nốt phần phơi nhiễm binary (T4…C5) trên `300c8dd1`; xem `HANDOFF_2026-10-20.md`.

- Ngày 20/10/2026. Ứng viên: nhánh local `v3-candidate` @ `300c8dd1` (19 commit trên `v2-final` `7cba42ff`). Chưa
  merge vào `main`, chưa push.
- "Adopted": các bất biến rút từ lỗi của ba dự án ngoài, mỗi bất biến ép ở chủ sẵn có của agent-cli. Mỗi cái có test
  tái hiện đỏ trên V2 và xanh trên ứng viên, cộng một lượt soát độc lập DAT.
- "Partially": mọi lớp trí nhớ/kiến trúc mới của giả thuyết Minimum-V3 bị DEFER hoặc REJECT (§B Q22–Q28). Một phần T6
  bị chặn vì cần ghi lại cassette.
- Quyết định tích hợp chi tiết: `V3_INTEGRATION_DECISION.md`. Nhật ký: `WORKFLOW_PROGRESS.md` (sự kiện 1–99).

Viết tắt: CM = Context Mode, MEM = claude-mem, HS = Hindsight, MAP = `cross-project/ISSUE_TO_INVARIANT_MAP.md`,
FIT = `cross-project/FIT_GAP.md`, WP = `WORKFLOW_PROGRESS.md`.

---

## A. Số đo cuối (đều từ lệnh đã chạy; tái lập bằng `scratchpad/verify-final.sh <commit> <thư-mục-mới>`)

### A.1 Kiểm cơ học trên ứng viên cuối `300c8dd1`

Lượt 3 (`scratchpad/verify-final.sh 300c8dd1`; DỪNG giữa bước phơi nhiễm binary theo lệnh người dùng 20/10):

| Bước | Kết quả |
|---|---|
| build | rc 0, `agent-cli version v0.2.1-0.20261020030742-300c8dd141db` |
| Toàn kho `go test -json -count=1 -p 2 ./...`, so theo TÊN với v2-final | v2 4.637 tên (4.625 pass, 12 skip) → ứng viên 4.782 tên; 0 FAIL hai bên; 155 tên mới; 0 đổi trạng thái; 10 tên mất, đều do POC xoá có chủ đích (1 `yolo` của P0-2; 3 `TestRestoreModel*` của P1-c; 6 test của cơ chế `Locations`/Home gộp DB và một test đổi tên, của P2-c) |
| `-race` 31 package (P5 + mọi package ứng viên chạm) | rc 0, 0 cảnh báo DATA RACE |
| E2E binary + LLM giả: run-json, loop, routine, tmux, wrapper | rc 0 cả 5 |
| Cổng eval: faults, qwen-only, workspaces, session-model, ledger | rc 0 cả 5 |
| Phơi nhiễm unit | T1 PASS (9/9 hàng), T12 PASS, T15 PASS; C9 FAIL (không làm POC, đúng dự kiến) |
| Phơi nhiễm binary | T1M, T2, T3 NOT_EXPOSED; các ca sau T3 CHƯA chạy lại trên `300c8dd1` (bị dừng) — kết quả gần nhất của chúng là lượt 2 trên `474a51f5` (bảng A.3) |

Lượt 1 (`642fe075`) và lượt 2 (`474a51f5`): mọi bước rc 0 như trên, race 0 (`experiments/final-verify/`).

### A.2 Eval LLM thật (provider `main/qwen3.8`, local)

| | v2-final `7cba42ff` | ứng viên `73ce25eb` (trước P2-d, P3-a, INTEG3, P2-c) |
|---|---|---|
| passed | 12/12 | 12/12 |
| false_done | 0 | 0 |
| lọt provider | 0 | 0 |
| tokens vào | 1.525.887 | 1.581.972 |
| giây | 343 | 376 |

- Xen kẽ A/B theo kịch bản; mỗi kịch bản 1 lượt; bỏ s13 (gọi GLM).
- Kết luận được: không hồi quy pass/false_done trên mẫu này.
- Không kết luận được: chi phí (n=1, dao động hai chiều: s3 ×2,9, s16 ×0,44).
- Ứng viên cuối CHƯA chạy lại eval LLM thật. Bốn thay đổi sau `73ce25eb` đã qua cổng LLM giả (A.1).
- Nguồn: `experiments/eval/ab1-COMPARE.md`, `ab1.log`.

### A.3 Bộ phơi nhiễm: V2 → ứng viên cuối

| Ca | V2 (`7cba42ff`) | Ứng viên | Lượt đo trên ứng viên | POC |
|---|---|---|---|---|
| T1 symlink vào thư mục con | EXPOSED | NOT_EXPOSED | lượt 3 (unit) | P2-c |
| T1M hai workspace một DB | EXPOSED | NOT_EXPOSED | lượt 3 | P2-c |
| T2 `-c` tương đối | EXPOSED | NOT_EXPOSED | lượt 3 | P2-b |
| T3 2 DB theo thứ tự mở | EXPOSED | NOT_EXPOSED | lượt 3 | P2-c |
| T4 `agent-cli-tmux rm` thư mục con | EXPOSED | NOT_EXPOSED | lượt 2 | P0-3 |
| T5 `<mcp-instructions>` cho agent con | EXPOSED | NOT_EXPOSED | lượt 2 | P2-a |
| T6 todo reminder / template nêu tool vắng | EXPOSED | **EXPOSED** (phần còn lại chờ cassette) | lượt 2 | P2-a (một phần) |
| T9 MCP 5 MiB | EXPOSED | NOT_EXPOSED | lượt 2 | P1-b |
| T11 `send` ack rồi mất | EXPOSED (7/8) | NOT_EXPOSED (8/8 trong DB) | lượt 2 | P1-a |
| T12 tóm tắt thành lời user | EXPOSED | NOT_EXPOSED | lượt 3 (unit) | P3-a |
| T13 prompt xếp hàng mất khi SIGTERM | EXPOSED | NOT_EXPOSED | lượt 2 | P1-a |
| T15 lỗi ghi sổ LLM im lặng | EXPOSED | NOT_EXPOSED | lượt 3 (unit) | P3-b |
| T17 `run -s` ghi model workspace | EXPOSED | NOT_EXPOSED | lượt 2 | P1-c |
| T19 MCP `isError` | EXPOSED | NOT_EXPOSED | lượt 2 | P1-b |
| T20 panic khi `run` đồng thời | EXPOSED | NOT_EXPOSED (0/10) | lượt 2 | P0-1 |
| T23 xoá session bận | NOT_EXPOSED | NOT_EXPOSED | lượt 2 | — |
| C5 chính sách client sau | EXPOSED | phần allowlist NOT_EXPOSED; phần env (biến thường của `base_url`) EXPOSED theo thiết kế, có tài liệu | lượt 2 | P0-2 |
| T21b `.env` của cwd vào tiến trình | EXPOSED | EXPOSED (thiết kế D4, có tài liệu) | lượt 2 | — |
| C9 tool/shell ra ngoài thư mục làm việc | EXPOSED | EXPOSED (không làm POC) | lượt 3 (unit) | — |
| T21 secret lưu bền | MEASURED | MEASURED (3 tệp server, 4 tệp local `-d`) | lượt 2 | — |
| Hook che MCP đã tắt (P2-d, phát hiện giữa đợt) | EXPOSED (3/3 request) | NOT_EXPOSED (0/3) | e2e riêng của P2-d | P2-d |
| `--yolo` cho run/send/loop (hồi quy của P0-2) | — (V2 cảnh báo) | đúng: từ chối có hướng dẫn, khớp được | e2e `evidence-integ3/yolo-repro.sh` | INTEG3 |

---

## B. Trả lời 55 câu (spec §78)

### Q1. V2 sau khi xong là gì?
- Nhánh local `v2-final` `7cba42ff`: 8 commit trên `main` `4a0b25b6`; chưa push.
- Trạng thái V2: SE "SYSTEM IMPROVED — WITH KNOWN LIMITATIONS"; Daily "DAILY WORKSPACE READY — WITH KNOWN LIMITATIONS".
- Kiến trúc dựng lại từ mã: `01_POST_V2_ARCHITECTURE.md`.
  - Mỗi dự án một server (socket theo hash của gốc git); TUI/`run`/`send`/`loop` là client.
  - Một điểm dựng model với allowlist mỗi lượt, fail closed.
  - Session trong SQLite của dự án; tóm tắt bằng LLM khi đầy context.
  - Spill output bash ra `<data>/shell-output`.
  - Loop có journal fsync; routine có flock.
- Không có kho trí nhớ dài hạn, worker nền hay observer LLM.

### Q2. Giả định nào của V2 sai?
Đều đã tái hiện trên binary/unit của V2 (WP 20, 24, 30–35, 78, 83):
- "Chờ agent dựng xong" an toàn khi đồng thời: sai. 10 `run` song song làm server panic 27/200 lượt (T20).
- Cảnh báo first-wins là đủ cho chính sách client sau: sai. Allowlist chặt hơn vẫn chạy provider ngoài danh sách (C5).
- `.env` của repo chỉ ảnh hưởng tiến trình đã nạp nó: sai. Nó tới CLI con giao việc, `!cmd`, server do Home dựng.
- `send` rc 0 nghĩa là prompt đã được ghi: sai. 7/8 prompt mất khi server bị giết ngay sau (T11); prompt xếp hàng mất khi
  SIGTERM (T13).
- Mở/tiếp session cũ không đổi gì của dự án: sai. `run -s` ghi model của session thành cấu hình workspace (T17).
- Kết quả tool nào cũng có trần: sai. MCP 5 MiB đi nguyên vào request kế và DB; `isError` bị bỏ (T9, T19).
- `-c` tương đối an toàn: sai. DB rơi ra ngoài repo; `loop -c ..` ra GREEN cho thư mục khác (T2).
- Bộ lọc tool theo kiểu thấy mọi tool: sai. Có hook PreToolUse thì MCP đã tắt vẫn gọi được.
- Văn bản prompt khớp tool set: sai. Agent con nhận chỉ dẫn cho tool nó không có (T5); một phần T6 còn mở.
- Tóm tắt là dữ liệu của assistant: sai. Nó tới model dưới vai user không nhãn (T12).
- Sổ ghi LLM là bằng chứng: sai khi ghi lỗi, vì lỗi bị nuốt (T15).
- `agent-cli-tmux rm` chỉ xoá worktree rỗi: sai khi session bận ở thư mục con (T4).
- Data dir/workspace có một danh tính cho mỗi repo: sai. Có 2 DB theo thứ tự mở (T3), DB ngoài repo qua symlink (T1),
  hai workspace chung một DB chạy hai lượt cùng session (T1M).

> Ghi chú: trạng thái "ĐO ĐƯỢC / ĐANG ĐO" của T# trong Q5, Q9, Q13 là lúc nghiên cứu (trước POC). Trạng thái sau V3 xem bảng A.3.

### Q3. Context Mode thực sự giải được gì?

- Chứa output của đường `ctx_*`: chỉ stdout, preview hoặc kết quả search vào context; `ctx_execute` không `intent` mà stdout > 100 KiB thì index FTS5 và trả con trỏ; có `intent` và output > 5 000 B thì BM25, trả tiêu đề + dòng đầu ≤ 120 ký tự của ≤ 5 section (E1; chưa chạy). [CM01 §1]
- Bỏ smartTruncate, index hết: sửa lỗi #197 (cắt phần giữa trước khi index làm mất dữ liệu vĩnh viễn); báo cáo đánh giá quyết định này hợp lý về dữ liệu, nhược điểm nằm ở snapshot không đặt lại ngân sách (E1 commit `ee628da5`). [CM02 Thời kỳ 3]
- Tìm kiếm: FTS5 kép porter + trigram, BM25, RRF, sửa chính tả Levenshtein; thiết kế đơn giản, có test (E2); audit không thấy lý do đề xuất semantic search. [CM01 Hợp đồng đầu ra]
- Fetch/executor: SSRF guard, redirect thủ công tối đa 5, cap 50 MiB, stream cap 100 MiB + kill cây tiến trình được đánh giá "đúng mức" (E1); chưa chạy fault-injection nào. [CM01 §7]
- Chặn cứng một số đường lấy dữ liệu của native tool (curl/wget, inline HTTP, build tool, WebFetch) và redirect sang `ctx_*` (E1); số "bytes avoided" là hằng số 8192/16384, không đo thật. [CM01 §1]
- Giữ trạng thái qua compact: SessionDB + snapshot "reference-based", tiêm directive + auto-injection ≤ 500 token ước lượng (E1). [CM01 §5]

### Q4. Context Mode không giải được gì?

- Containment chỉ phủ đường `ctx_*`: Read/Grep/Bash/MCP ngoài chỉ nhận lời nhắc, PostToolUse không sửa output, nhắc Read chỉ bắn 1 lần/phiên (E1, F2). [CM01 §1]
- `ctx_execute` không `intent` mà stdout ≤ 100 KiB thì trả nguyên stdout vào context (E1). [CM01 §1]
- Không có cô lập FS, mạng, env: `cwd` = project root, `HOME` thật, env chỉ bỏ denylist ~90 biến nên secret khác như `GITHUB_TOKEN` đi vào tiến trình con (suy luận từ denylist, chưa chạy); README tự mâu thuẫn "isolated" và "not a full OS sandbox". [CM01 §8]
- Capability của agent/subagent: ba chỗ quyết với ba giả định khác nhau (F3, E1); không có khái niệm subagent read-only/reviewer ngoài chuỗi `"Bash"` (E1); cờ `PlatformCapabilities` khai mà không ai đọc. [CM01 §2] [CM01 §4]
- Mất dữ liệu âm thầm: F1 nhãn `execute:<language>` ghi đè nhau (suy luận từ mã, chưa chạy); F5 bản sửa `4164d3de` chỉ ở 1/9 hook SessionStart, 8 hook còn lại vẫn DELETE vô điều kiện (E1), việc cùng khe hở thời gian trên các host đó là "mới nghi". [CM01 §1] [CM01 §5]
- Danh tính phiên/worktree: F6 `currentAttribution()` lấy session "mới nhất", F7 ContentStore không theo worktree (cơ chế E1, hệ quả suy luận, chưa chạy). [CM01 §5]
- Cổng "MCP sẵn sàng": 4 vòng sửa (#230, #347, #354, #844) rồi #1055/#1053 mở sau vòng cuối; v2 bỏ cổng mà không thay (xác nhận E1 commit/mã + E0 tiêu đề). [CM03 §2.10]
- Snapshot resume không có ngân sách byte (E1); upstream có #1022 "advertises <2KB, injects ~196KB" (E0). [CM02 Thời kỳ 3]

### Q5. Họ issue nào của Context Mode quan trọng với agent-cli?

- Giới hạn bằng chứng: CM03 là bản nháp offline; 172/173 issue tạo sau khi `main` ngừng phát triển, giao với commit fix trên `main` = 0, và "nguyên nhân gốc" chỉ lấy từ bản đồ upstream là E0 yếu. [CM03 §0]
- CAPABILITY_MISMATCH (9 issue; chuỗi #233 → #724 → #1034, #794 → #1037) ↔ họ F05: ĐO ĐƯỢC T5, T6 (E3) trên agent-cli (mcp-instructions và todo reminder không đọc `agentTools`); ứng viên POC P2. [MAP F05]
- FALSE_SUCCESS (18 issue; tiền lệ `4bc292fc` test khớp hình dạng JSON mà redirect no-op ở production) ↔ F14: ĐO ĐƯỢC T15 (thu hẹp, unit), ĐANG ĐO T19. [MAP F14]
- ROUTING_FAILURE (21 issue; regex curl #63 → #1106/#1221) ↔ F06: SUY LUẬN (Bash tự duyệt theo tiền tố, BlockFunc theo mẫu), ĐANG ĐO C9. [MAP F06]
- HOST_ADAPTER_MISMATCH (39 issue, họ lớn nhất, phân loại tay theo tiêu đề) ↔ F09: SUY LUẬN dạng "nhiều bản sao" (local so với server, ba cấu hình discovery); V3 test T17 (thêm nhánh local), T24. [MAP F09]
- PROMPT_INTERFERENCE (16 issue; #946, #967 "~40% of subagent spawns" là E0) ↔ F07: KHÔNG (E1) với prompt giao việc truyền nguyên văn; SUY LUẬN với context file của repo lạ; T8. [MAP F07]
- ZOMBIE_OPERATION (14) và STARTUP_RACE (5) ↔ F11 và F10: F11 KHÔNG cho lớp "kẹt mãi" (E1), ĐANG ĐO T4, T23, thêm T28; F10 SUY LUẬN S5, S7, ĐANG ĐO T20, thêm T25. [MAP F11] [MAP F10]
- DELETION / mất dữ liệu (9 issue; bốn đường dọn đều từng xoá dữ liệu sống) ↔ F17: SUY LUẬN (xoá session không xoá session con, `session delete` qua CLI không kiểm bận), ĐANG ĐO T23, T10 chưa viết. [MAP F17]

### Q6. Fork/nhánh nào (CM và MEM) rẽ hướng có ý nghĩa, và vì sao?

- CM `origin/v2-design` (737 commit chưa vào `main`, 2026-10-14 → 10-19): viết lại song song, 18 ADR; bỏ tiêm routing vào prompt, một DB, capability của subagent giải ở một module, index append-only (F1 biến mất), thoát theo stdin EOF. Lý do theo upstream (E0): classifier host chặn khối tiêm (#946, #967), 13 bản routing lệch nhau, ~10 issue ABI native; mã khớp ADR mức "LÀM" cho 6/18, "MỘT PHẦN" cho 10/18 (E1). [CM02b §0] [CM02b §(a)]
- Nhưng v2 bỏ `isMCPReady` mà không thay (V2-F1, E1), và văn bản hint/deny case C/`more:` vẫn nêu `ctx_*` cho subagent không có (V2-F2, E1+E2); hai danh tính phiên là "mới nghi" (V2-F3). [CM02b Phát hiện mới trên v2]
- Tình trạng v2: M7 tự ghi "not done" (GS5/GS10/GS11 FAIL, E0 tự báo); không thấy dấu hiệu merge vào `main` (ADR-0011: `main` đóng băng, v2 phát hành qua nhánh `release`, chưa có trong `git branch -r`). [CM02b §(e)]
- Fork ngoài của Context Mode: Nguồn không trả lời (đã soát CM01, CM02, CM02b, CM03; các tệp đều ghi không đào fork). Chỉ có phương án cộng đồng dạng PR chưa merge, ví dụ #1272 chuyển chỉ dẫn subagent sang sự kiện `SubagentStart`. [CM03 §5]
- MEM `tt12b/claude-mem` (DEPLOYMENT_VARIANT / ARCHITECTURAL_FORK, E0): tóm tắt theo chu kỳ N phút thay cho một lời gọi provider mỗi sự kiện, watermark chỉ tiến theo phần đã nằm trong prompt, claim bỏ dở được thu hồi về outbox; vì chi phí free-tier và con trỏ tiến trước kết quả. Quyết định: ADAPT_IDEA (ứng viên, chưa audit agent-cli). [MEM04 §2.1]
- MEM `rodboev` (SCOPE_FIX, #2882 monorepo mất memory): heuristic khoá repo-relative đảo chiều ngay trong cùng chuỗi (fork bỏ nhánh "nested package roots", edge giữ); không đổi kiến trúc. Quyết định: IGNORE heuristic, giữ bài học "khai báo tường minh thắng suy đoán". [MEM04 §2.2]
- MEM nhánh upstream rẽ hướng nhưng không merge: `branch-memory` (lọc theo tổ tiên git; PR #1246 đóng "Closing during backlog consolidation", lý do không làm lại là UNKNOWN), `canary` (bản sửa #1911 "default project" chỉ nằm ở đây), `bugfix/security-hardening` (khung "unverified"), `enhancement/supersession`. [MEM04 §3.2]
- MEM dòng `community-edge` (tag `archive/community-edge`) là nơi upstream đặt thay đổi danh tính rủi ro; khoá chuẩn `107ef946e` trên `main` bị revert sau ~14 giờ (suy luận: đổi danh tính bị đẩy khỏi stable). [MEM04 §3.3]

### Q7. claude-mem thực sự giải được gì?

- Đường ống hoạt động đầy đủ (E1): hook ghi spool bền → worker → observer LLM → SQLite + FTS5/Chroma → `ContextBuilder` fit 10.000 ký tự → `additionalContext`; hook lỗi bất ngờ thì no-op + exit 0, không chặn người dùng. [MEM01 §1]
- Hook không còn chặn người dùng: trước kia exit 2 gây vòng chặn (#2966, #3481, #3523), nay `grep -rn "process.exit(2)" src` = 0 ngoài test. [MEM03 §2.2] [MEM03 Hợp đồng đầu ra]
- Ngân sách và hợp đồng đầu ra có một chủ duy nhất: `ContextBudget` (có test E2 `context-budget-3802.test.ts`), `hook-io` một lối stdout, `hardened-options`, SSE `/api/ready`; từ khi các chủ này xuất hiện chưa thấy hồi quy (chưa đo dài hạn, các mốc mới từ 09-10/2026). [MEM03 §17] [MEM01 §7]
- Cold start: SSE `/api/ready` (`starting → db_ready → routes_ready → ready|failed`) cộng context cache trên đĩa, có test (E2), thay chuỗi chỉ nâng timeout. [MEM03 §8]
- NO MEMORY tách được khỏi NOT READY/FAILED ở mức mã (E1). [MEM01 §5]
- Observer chạy không có tool và không persist session (`--no-session-persistence`), có test E2; đây là mục "không đổi" của báo cáo. [MEM01 §3] [MEM03 Hợp đồng đầu ra]
- Giảm mất memory do đổi khoá: phía đọc hợp nhất khoá (`projectReadKeys`, `merged_into_project`, `COLLATE NOCASE`); upstream chọn "hợp nhất lúc đọc" thay cho khoá chuẩn lúc ghi (suy luận của báo cáo). [MEM01 §4.4] [MEM02 §3]
- Khung "data, not instructions" có ở các đường search, progressive search, corpus, TeamBrain, Grok bot index (E1). [MEM01 §11]

### Q8. claude-mem không giải được gì?

- Danh tính project: không có resolver duy nhất, #3608 OPEN, còn ít nhất 6 đường vòng (A3-A8), doctor không có hàng danh tính, lần thử khoá chuẩn `107ef946e` bị revert sau ~14 giờ (E1). [MEM01 §4.4]
- Thiếu `project` ở MCP search/timeline/get_observations/mem_search thì tìm toàn cục (#3764, hồi quy của #1911), không có bản sửa trên `main`; test `checkout-search-scope.test.ts:21-24` khoá hành vi đó (E1, E2). [MEM01 §12] [MEM03 §2.3]
- Không có danh tính branch (E1; `grep abbrev-ref src` = 0); thử nghiệm `branch-memory` không merge. [MEM01 §4.2] [MEM04 §3.2]
- ACCEPTED ≠ DURABLE còn hở: spool coi là đã giao khi sự kiện vào buffer RAM; worker chết thì mất; đường replay transcript tự động cho host Claude Code là "mới nghi: chưa thấy" (cách kiểm: giết worker sau `markHandedOff`, đếm `observations` và `tool_uses`). [MEM01 §3]
- NOT READY và RETRIEVAL FAILED gộp thành cùng một khối rỗng với model; SAI SCOPE trông y như NO MEMORY (E1). [MEM01 §5]
- SessionStart, kênh tiêm chính, không có khung untrusted; thử nghiệm đối kháng "chưa làm". [MEM01 §11]
- Không lọc theo session ở SessionStart: maintainer coi trộn observation giữa session là "working as designed" (#2909, PR #2919 đóng không merge, E0). [MEM03 §7]
- Chưa đo: số lời gọi LLM, latency, chi phí, tốc độ tăng log; log hằng ngày không thấy giới hạn kích thước/số ngày giữ (E1); `main` không có eval lợi ích memory (nhánh `evals-locomo` là prototype, F1 trung bình 0,041 trên 20 câu). [MEM01 §3] [MEM01 §9] [MEM04 §3.2]

### Q9. Lỗi scope/lifecycle/worker nào của claude-mem quan trọng (với agent-cli)?

- Danh tính đổi luật không migration (#2663 → #2882, `daf6d9dc0`; 23 lần đổi luật, issue là E0) ↔ F01: agent-cli ĐO ĐƯỢC T2, T3, T1 (thu hẹp về symlink-vào-thư-mục-con), ĐANG ĐO T4, C5/S10. [MEM02 §3] [MAP F01]
- Thiếu scope thì toàn cục (#1911 → #3764) ↔ F04: KHÔNG ÁP DỤNG hôm nay vì agent-cli chưa có đường truy hồi tri thức; T30 là cổng thiết kế chỉ viết khi V3 thêm truy hồi. [MAP F04]
- Nhiễm session/agent/tác giả (#2909/#2976, #3274 ~76% observation tiêm là của subagent theo người báo, #4215 sửa 2 lần) ↔ F02: KHÔNG thấy đường (đọc mã, chưa có test); T26 mới. [MEM03 §7] [MAP F02]
- Cold start 4-5 lần (#1341 → #1505 → #1785 → #2117 → #2795; năm bản sửa đầu không có tệp test, E1) ↔ F10: SUY LUẬN S5, S7, ĐANG ĐO T20, thêm T25. [MEM03 §8] [MAP F10]
- ACCEPTED ≠ DURABLE (`markHandedOff` trước khi bền; #3653 13 observation từ ~44k sự kiện là E0) ↔ F13: ĐO ĐƯỢC T11 (`send` rc 0 rồi SIGKILL, 7/8 lần mất tin user) và T13; thêm T14. [MEM03 §9] [MAP F13]
- Liveness worker: #3603 (plan-15) OPEN với 5 tín hiệu liveness; chuỗi reclaim `fc5f15843` → `8009b0dab` → `cf0c7cdcc`, mỗi bước sửa lỗi do bước trước sinh ra (E1) ↔ F11: KHÔNG cho lớp "kẹt mãi" ở các đường đã đọc (E1, chưa đo lại), ĐANG ĐO T4, T23, thêm T28. [MEM03 §10] [MAP F11]
- Retry/log storm (#4087 770k dòng/111 MB, #3928 8,9M lỗi/3,8 GB là E0; lý do gỡ hàng đợi bền: "that was the retry storm") ↔ F12: KHÔNG cho retention log (lumberjack 10 MB, 30 ngày), SUY LUẬN `DropCount` không ai đọc; T29 mới. [MEM03 §11] [MAP F12]
- Observer tự bắt phiên của chính nó (#3425, ≥3 lớp chặn) ↔ F03: SUY LUẬN (CLI con giao việc nạp settings/hook người dùng); T27 mới (phép đo, chưa phải cổng). [MEM03 §16] [MAP F03]

### Q10. Fork nào đổi kiến trúc của claude-mem một cách có ý nghĩa?

- Chỉ `tt12b/claude-mem` được xếp ARCHITECTURAL_FORK trong 13 fork đã soi (1 kiến trúc, 1 sửa scope, 7 nguồn PR đã hấp thụ, 2 mirror, 2 không đủ bằng chứng). [MEM04 Hợp đồng đầu ra]
- Thay đổi (E0, từ README + thân commit): từ "một sự kiện → một lời gọi" sang "cửa sổ thời gian → một lời gọi", thêm outbox + watermark chỉ tiến theo phần đã nằm trong prompt, đệm sự kiện hook tại chỗ khi server sập. [MEM04 §2.1]
- Chi phí và rủi ro: Postgres + Valkey + 4 container, dashboard không auth (README tự cảnh báo), độ trễ memory bằng chu kỳ N phút (suy luận); không thấy test, không có runtime evidence; base của fork là suy luận (dựng trên server-beta). [MEM04 §2.1]
- Tương đương upstream: gom lô opt-in `d55d20f7e` (mặc định batch=1), HookSpool bền `fe1709c30`, spool tới worker từ xa; báo cáo không có bằng chứng upstream lấy từ fork (tiến hoá song song). [MEM04 §2.1]
- `rodboev` không đổi kiến trúc (thêm heuristic vào resolver chung), quyết định IGNORE heuristic. [MEM04 §2.2]
- `escooterclinic` (một renderer, nhiều nguồn hàng) và `surfingdoggo` (migration dựa bằng chứng checkout đã ghi nhận) có ý tưởng kiến trúc nhưng đã vào upstream: OBSOLETE, bài học giữ lại. [MEM04 §2.3]
- Kết luận của báo cáo: không fork nào trong số đã soi giải được bài danh tính bằng một resolver chuẩn. [MEM04 Hợp đồng đầu ra]
- Giới hạn bằng chứng: 24/25 SHA fork lấy từ timeline không có trong clone; ba trang fork không hiện ahead/behind nên chưa đếm commit chỉ-có-ở-fork. [MEM04 §1]

### Q11. Hindsight đổi gì kể từ audit trước?

- Delta `e00154496..5331da2f4`: 48 commit, tất cả ngày 2026-10-16, 327 tệp, +18.846/-2.603 dòng (đo bằng `git diff --shortstat`); 47 commit của Nicolò Boschi, 1 commit blog. [HS02 §0]
- `hindsight-integrations/` không đổi dòng nào (0 tệp); ba commit "codex/claude-code" là provider LLM phía server, không phải plugin hook của coding agent. [HS02 §0]
- Worker và hàng đợi không đổi ngữ nghĩa: `poller.py` chỉ đổi đồng hồ sang `perf_counter()`, vẫn không có lease hay heartbeat. [HS02 §0]
- Phân nhóm 48 commit: Retain 6, Scope/tag/quyền 5, Control plane 5, Mental models 4, Transfer/import 4, Provider coding-agent 4, Provider khác 4, Worker/quan sát 3, Helm/CI 3, Consolidation 2, Reflect 2, Recall 2, Documents API 2, Khác 2. [HS02 §2]
- Sửa lỗi: #5371 (`89594d22d`), #5448 (`c7acb488a`), #5228 giảm nhẹ bằng prompt (`0433604c1`); bộ lọc tag gom về một luật (`3f8fe203f`, #5417). [HS02 §0] [HS02 §3.4]
- Thêm tính năng ở đúng phần SE 28 đã REJECT: reflect "decision-orchestrated" (`198947912`, +2.988 dòng), triage consolidation bằng decision model (`0d8e58b3e`, bật mặc định khi có cấu hình), full refresh theo lịch cho mental model; số eval upstream nêu (ví dụ trang stale 10/16 → 3/16) là E0 với agent-cli. [HS01 §5]
- Số đếm mốc đổi: `memory_engine.py` 23.477 → 23.784 dòng, migration 111 → 113, tệp test API 631 → 648, system story 79 → 87 (đo `wc -l`, `git ls-tree`). [HS02 §5]
- Không commit nào trong delta mâu thuẫn với 7 bất biến SE 28; không có bằng chứng mới đổi quyết định REJECT/DEFER; chưa chạy lại các kết quả E3 của audit cũ. [HS01 §4] [HS01 §5]

### Q12. Kết luận nào của audit Hindsight trước đã lỗi thời?

- #5371 "OPEN, tái lập ở HEAD" nay FIXED ở parser chung và 6 lối vào (`89594d22d`); còn một parser inline ở `list_mental_models`. [HS02 §5]
- #5448 `allowed_bank_ids` khai báo mà không ép nay FIXED bằng cách xoá trường và chặn `dry-run-extract` (`c7acb488a`). [HS02 §5]
- #5228 delta refresh để lại câu cũ nay PARTIALLY_FIXED bằng một dòng footer trong prompt (`0433604c1`), phụ thuộc LLM; số eval của upstream là E0. [HS02 §5]
- H-26 `based_on` = mọi memory đã recall: PARTIALLY, một nguồn gây ra (fact qua `expand`) đã gỡ (`45616ffe3`), nhánh "rỗng thì lấy tất cả" vẫn còn. [HS02 §5]
- Mô tả cursor coding-agents (audit §12.7, SE 24 H-29..H-31) thiếu ngay tại revision được audit: không nhắc trường `ops` của bản sửa #5251 (PR #5508, `d3f0001ed` là tổ tiên của `e00154496`); đây là bỏ sót, không do delta. [HS02 §5] [HS01 §7]
- Mô tả pipeline đã đổi: consolidation có thêm bước triage, reflect có chế độ fast/decision-orchestrated, delta retain có luật nhường streaming cho document viết dở; danh sách provider và mặc định codex đổi, số đếm §1/§8 lệch (tổng số provider "chưa đếm lại"). [HS02 §5]
- Bổ sung/sửa thêm từ họ issue mới: #3989 có gốc ở server (append chia slice làm cụt `original_text`), #3136 là PR chứ không phải issue, và consolidation chỉ "hội tụ" cho kiểu áp nửa vời (#5054 đóng "by design"). [HS03 §4]
- Không lỗi thời (delta không chạm vùng mã; grep diff rỗng, không chạy lại E1/E2/E7): 9/10 kết luận chính, gồm không lease (H-01), #5582, `ExcludeNoneRoute`, `run_consolidation` trả số 0, FK đa tenant, mặc định mở, recall "4 song song". [HS01 §3]

### Q13. Lỗi mới nào của Hindsight quan trọng?

- #5735 (mở 18/10): probe phiên bản thất bại bị cache âm vĩnh viễn ở client coding-agents (`ca/hindsight.ts:540-546`, E1); hermes đã sửa cùng lỗi #5221 ngày 12/10 nhưng bản sao không được sửa, 6 ngày sau bị báo lại. Liên hệ F09 (bản sửa không lan sang bản sao) và F10 (T20, T25). [HS03 F1] [MAP F10]
- #5251 PARTIALLY_FIXED với 6 chỗ hở (a)-(f) (E1, chưa chạy): `retain_ok` ghi ngay lúc accept, `settleOps` chỉ chạy ở write-back kế tiếp của cùng session, 404 bị coi là xong, cursor ở `tmpdir()`; chuỗi cursor phình 4 → 8 trường. Liên hệ F13: agent-cli ĐO ĐƯỢC T11, T13, thêm T14. [HS02 §6.1] [MAP F13]
- #5582 (retain trả 200 với 0 memory, OPEN) và #5589 (883 document 0 unit theo người báo, OPEN): cùng phễu "rỗng → completed", mỗi cái có bản sửa ở fork mà không có PR upstream (E0). Liên hệ F14: ĐO ĐƯỢC T15 (thu hẹp), ĐANG ĐO T19. [HS03 F7] [MAP F14]
- #5728 (mở 18/10): refresh mental model ghi báo cáo "nothing found" đè lên trang tốt; HEAD có test ghim hành vi "trang anh em là nguồn" (`test_mental_models.py:3576`), nên là xung đột thiết kế. Chuỗi tag scope #5030 → #5371 → #5417 trong 15 ngày. Liên hệ F04: KHÔNG ÁP DỤNG hôm nay, T30 cổng thiết kế; agent-cli đã đúng ở chỗ tóm tắt rỗng không thay lịch sử. [HS03 F4] [MAP F04]
- #5542 (#1751 lặp sau ~4,5 tháng), #5555 còn mở: hook của host lọt vào tiến trình con LLM; có 4 cơ chế chống đệ quy độc lập, server không truyền `HINDSIGHT_DISABLE_HOOKS` dù hook của Hindsight đọc biến này (suy luận). Liên hệ F03: SUY LUẬN với tool giao việc của agent-cli, T27 (phép đo). [HS03 F2] [MAP F03]
- Dấu "đã consolidate" gánh nhiều nghĩa: #5054 (48% fact bị đóng dấu mà không được biểu diễn, theo người báo) đóng "by design"; triage "chatter" thêm một lối đóng dấu; #5731 reset bị chặn khi `deleted_count = 0`; #5434 `failed` ≠ `superseded` (OPEN). Liên hệ F17: SUY LUẬN, ĐANG ĐO T23, thêm T10. [HS03 F5] [MAP F17]
- #2413 (đóng not planned): `/health` chỉ đo kết nối DB nên xanh khi hàng đợi kẹt, và test hồi quy bị formatter vô hiệu, không chạy trong CI (E1 `00968a1ce`); việc thu hồi vẫn theo `worker_id` (PR #3779 lease bị bác). Liên hệ F11: KHÔNG cho "kẹt mãi" ở agent-cli (E1), ĐANG ĐO T4, T23, thêm T28. [HS02 §6.2] [MAP F11]
- #5713 (tiêm một lần mỗi phiên, không làm mới) và #5721 (tìm trang không có ngưỡng liên quan) đều OPEN, số đo là của người báo (E0). Liên hệ F15: SUY LUẬN, test T8; mọi trạng thái mã ở `5331da2f4`, upstream đã chạy tiếp lên 1.0.0 nên chưa kiểm. [HS03 F8] [MAP F15]


### Q14. Trí nhớ dài hạn có cải thiện việc code của agent-cli không?
Chưa có bằng chứng nào cho thấy có, và V3 không làm nó.
- Trên 12 kịch bản eval thật, gồm `s10-resume` và `s14-wrong-context`, M0 (không trí nhớ dài hạn) đạt 12/12 ở cả V2 lẫn
  ứng viên.
- Không lỗi nào trong 22 họ MAP của agent-cli truy về việc thiếu trí nhớ dài hạn (FIT F, SE28).
- Bằng chứng ngoài đều E0 hoặc âm: HS #2347 có 7/1.360 lượt có tìm; MEM không có eval lợi ích trên `main`.

### Q15. So với baseline nào?
- `v2-final` `7cba42ff`, chạy cùng ngày, cùng provider `main/qwen3.8`, xen kẽ A/B theo kịch bản.
- Cùng bộ cổng LLM giả và cùng bộ phơi nhiễm cho cả hai binary.

### Q16. Current truth là gì?
Tệp trong repo, git, và cấu hình của dự án. Agent đọc qua tool ở mỗi lượt. Không có bản sao dẫn xuất nào được coi là
thẩm quyền:
- model của session không còn ghi thành cấu hình (P1-c);
- tóm tắt mang nhãn là dữ liệu do assistant viết (P3-a).

### Q17. Working context là gì?
Prompt của một lượt, dựng trong `sessionAgent.Run` / `preparePrompt`:
- system prompt (template + context file ở gốc dự án + `<mcp-instructions>` chỉ của server có tool trong lượt, P2-a);
- tin của session từ điểm tóm tắt (tóm tắt có nhãn nguồn);
- tool set của lượt.

### Q18. Session continuity là gì?
Bảng session/messages trong `<gốc>/.crush/crush.db`. Prompt được ghi bền trước khi ack (P1-a). Tool call bỏ dở được ghi
"Interrupted" khi mở lại. Tóm tắt thay lịch sử khi đầy context; tóm tắt rỗng hoặc lỗi không thay.

### Q19. Historical knowledge là gì?
Không có lớp riêng. Hiện chỉ có session cũ, context file (`AGENTS.md`…), git history, và journal của loop. Store tri thức
đã kiểm bị DEFER (FIT F).

### Q20. Ai sở hữu từng phần?
| Phần | Chủ trong mã |
|---|---|
| Current truth | repo/git; cấu hình qua `internal/config` (một `ProjectRoot`) |
| Working context | `internal/agent` (`sessionAgent.Run`, `prompt.Build`) |
| Session continuity | `internal/message`, `internal/session`, `internal/backend` (ghi bền, settle) |
| Chính sách tin cậy / provider | `internal/execpolicy`, `coordinator.buildAgentModels` |
| Bằng chứng "xong" | `internal/loop` (journal + verify), `scripts/eval/*` |

### Q21. Mô hình scope cuối là gì?
- Gốc dự án là gốc git vật lý (`config.ProjectRoot`), ngoài git là thư mục vật lý.
- Mỗi gốc dự án ứng với một server, một workspace, một data dir `<gốc>/.crush`. Áp cho mọi chế độ, kể cả khi mở từ thư mục
  con (quyết định Q5 của người dùng, P2-c).
- Session nằm trong DB của dự án.
- Worktree liên kết là gốc riêng.
- Branch không phải chiều của scope; revision cho tóm tắt DEFER vì chưa viết T8.
- Không dùng `ExecutionScope` 11 chiều (FIT B2 DEFER).

### Q22. Engineering Event Model có đáng không?
Không, DEFER. Không lỗi nào truy về việc thiếu event log. Lỗi quan sát được (T15) sửa bằng tín hiệu ở chủ sẵn có (P3-b).

### Q23. Artifact Store có đáng không?
Không tạo store mới. Đường spill sẵn có (`<data>/shell-output`, trần + marker có đường dẫn) được mở cho MCP (P1-b).

### Q24. Context Compiler có đáng không?
Không. Chỉ cần một điểm quyết: văn bản prompt đọc tool set của lượt (P2-a). Bộ chọn theo ngân sách DEFER (FIT C2).

### Q25. Mức truy hồi nào thắng?
Không cần truy hồi mới: spill + `grep`/`view` sẵn có. FTS và semantic DEFER (FIT A2, A3).

### Q26. Observation có đáng không?
Không, REJECT: MEM tốn 1 lời gọi LLM mỗi sự kiện (#2201, #3653); HS #5582.

### Q27. Graph có đáng không?
Không, DEFER. Không lỗi nào đòi; bằng chứng hỏng riêng của graph chỉ có HS #5417.

### Q28. Reflect có đáng không?
Không, REJECT: HS #5728 ("nothing found" đè trang tốt), #5228, H-26.

### Q29. Phát hiện trí nhớ cũ thế nào?
Không có kho trí nhớ. Dẫn xuất duy nhất là bản tóm tắt; nó mang nhãn nguồn và không thay lịch sử khi rỗng/lỗi. Trường
revision cho tóm tắt DEFER (chờ T8).

### Q30. Sửa trí nhớ sai thế nào?
Tóm tắt được tạo lại ở lần nén sau; người dùng mở session mới. Không có tri thức bền nào cần đính chính.

### Q31. Vô hiệu hoá item dẫn xuất thế nào?
Dẫn xuất không bao giờ được nâng thành thẩm quyền:
- model suy từ session không ghi vào cấu hình (P1-c, T17);
- tóm tắt không thành lời người dùng (P3-a, T12);
- model của lượt cuối trong ghi chú bỏ qua placeholder (`message.LastReply`, INTEG2).

### Q32. Ngăn secret lưu bền thế nào?
**Chưa ngăn.** T21 (MEASURED, ứng viên): canary trong output tool nằm ở 3 tệp khi chạy qua server (`crush.db*`,
`shell-output`, journal) và 4 tệp khi chạy local `-d` (thêm log).
- Đã có: quyền 0600/0700 cho spill; sổ ghi LLM chỉ lưu metadata.
- Là giới hạn đã biết; không POC nào nhắm tới.

### Q33. Trí nhớ có ảnh hưởng chính sách được không?
Không qua đường nào đã đo:
- chính sách lấy từ env gốc chụp đầu `main` cộng tệp env (`internal/execpolicy`); `.env` của repo không đổi được nó (P0-2b);
- client có chính sách khác bị từ chối (P0-2);
- session không ghi model (P1-c);
- tóm tắt có nhãn (P3-a).

### Q34. Cô lập provider qua "trí tuệ nền" ép thế nào?
Không có worker LLM nền. Mọi lời gọi LLM nội bộ (tiêu đề, tóm tắt, agent con, `agentic_fetch`) đi qua một điểm dựng model
có allowlist. Cổng `qwen-only` (LLM giả đếm theo provider và vai trò): 5 vai trò đều đi qwen, glm = 0 (A.1).

### Q35. Scope chỉ Qwen local/miễn phí có sinh lời gọi trả phí trái phép nào không?
Không, trên các đường đã đo:
- cổng `qwen-only` và `faults`: 0 request sang provider khác (A.1);
- eval LLM thật: 0 lọt provider;
- giao việc Claude Code/Codex mặc định TẮT.

Chưa đo: giao việc khi bật (`AGENT_CLI_DELEGATION=on`) với CLI thật.

### Q36. Cold start ra sao?
- Spawn single-flight, phân loại socket cũ.
- Đua "agent dựng xong" khi nhiều `run` đồng thời đã sửa (P0-1, T20: 0/10 lỗi).
- Server khác bản mà đang bận thì lệnh chạy model bị từ chối, không chạy nhầm bản cũ.

### Q37. Worker crash ra sao?
Không có worker nền.
- Server bị giết thì tool call dở được ghi "Interrupted" khi mở lại.
- Prompt đã ack còn trong DB (T11 8/8) và hiện là đã huỷ, không tự chạy lại (D1).
- Routine bị SIGKILL ra `ABANDONED` và nhả khoá (E2E routine).

### Q38. Worker thay thế đổi danh tính ra sao?
Không áp dụng. flock nhả khi tiến trình chết, tương đương lease (FIT X5 KEEP_V2).

### Q39. Thành công một phần (async) ra sao?
- `send` ghi bền rồi mới ack (P1-a).
- Lỗi ghi sổ LLM thành `ledger_error` cộng WARN (P3-b).
- MCP `isError` được giữ (P1-b).
- Tóm tắt lỗi không thành điểm cắt.

### Q40. Backlog ra sao?
Prompt xếp hàng là hàng trong DB. Prompt bị huỷ khi còn chờ không gửi model nhưng để lại dấu trong UI (D5). Không có retry
vô hạn, không có hàng đợi nền.

### Q41. Trí nhớ không có thì sao?
Không áp dụng (không có kho). Phần phụ hỏng (sổ ghi LLM) không làm hỏng lượt chính (D2).

### Q42. Xuyên workspace?
- Mỗi dự án một server và workspace; client có chính sách hay `--yolo` khác bị từ chối kèm lời nói cách khớp (INTEG3).
- Cổng `workspaces` (Daily E: A→B→A→B) rc 0: context file và lịch sử không lọt.

### Q43. Xuyên session?
Cổng `workspaces` (Daily F) rc 0: lịch sử session khác không lọt; `-s` nối đúng. Xoá session đang bận không để tin mồ côi
(T23).

### Q44. Xuyên branch/worktree?
- Worktree liên kết cho đúng danh tính (T1 hàng `wt`).
- `agent-cli-tmux rm` không xoá worktree có session bận (T4).
- Branch không thuộc scope; độ tươi của branch trong system prompt DEFER (T8).

### Q45. Context của người soát độc lập tới đâu?
- Mỗi lượt soát là một agent riêng (loại Plan, không có Edit), context mới, chỉ nhận diff, báo cáo người viết và quyết
  định đã chốt.
- WP có 16 sự kiện REGRESSION, đa số là lượt soát KHONG-DAT bắt được lỗi.
- Giới hạn: cùng họ model với người viết. Người soát P2-c tự nhận đã sót một lỗi CHẶN ở vòng 1. Lỗi tích hợp INTEG3 do lượt
  soát docs bắt được, không phải lượt soát mã.

### Q46. V3 làm tăng hay giảm số agent?
Sản phẩm: không thêm agent, worker hay tiến trình nền nào. Quy trình làm V3: tối đa 3 agent cùng lúc (trần RAM của máy).

### Q47. Tăng hay giảm token?
- Chưa kết luận được: 1,53M → 1,58M với n=1, dao động hai chiều.
- Thay đổi có hướng biết trước:
  - agent con không còn nhận `<mcp-instructions>` thừa (P2-a);
  - session đã nén thêm câu nhãn khoảng 45 token mỗi lượt (P3-a; ước lượng đếm chữ, chưa đo bằng tokenizer).

### Q48. Có giảm khám phá lại không?
Chưa đo.

### Q49. Có giảm DONE giả không?
- Eval thật: false_done 0 → 0.
- Đã sửa hai đường DONE/STALLED giả có tái hiện:
  - `loop -c ..` ra GREEN cho thư mục khác (P2-b);
  - `loop` từ thư mục con STALLED oan (P2-c).

### Q50. Ý tưởng nào bị bác?
Reflect, mental models, auto-recall mỗi lượt, observation LLM mỗi sự kiện, nhiều DB/worker/injector, worker hay hàng đợi bền
cho trí nhớ, fallback chéo provider mặc định, cloud sync, dùng benchmark LoCoMo làm bằng chứng (FIT §3).

### Q51. Độ phức tạp nào bị cắt?
- Toàn bộ lớp trí nhớ của giả thuyết Minimum-V3 trừ phần bất biến.
- Trong POC:
  - P2-c xoá cơ chế Home gộp nhiều DB (`Locations`/`DataDirs`), thay bằng một DB;
  - INTEG2 gom ba chỗ tự tính "trả lời thật cuối" về `message.LastReply`;
  - P2-d gom việc bóc lớp hook về `unwrapTool`.
- Không POC nào thêm bảng, worker hay store.

### Q52. Cái gì còn thử nghiệm?
- Nhãn nguồn của tóm tắt (P3-a): hiệu ứng lên model thật chưa đo.
- Một workspace cho mỗi repo khi mở từ thư mục con (P2-c): mới chạy với LLM giả và unit; chưa có ngày dùng thật.

### Q53. Cái gì còn chưa biết?
- Hiệu ứng P3-a lên LLM thật.
- Chi phí token V3 so với V2 (cần n > 1).
- Giao việc thật khi bật.
- Phần T6 cần cassette mới (`CRUSH_HYPER_API_KEY`).
- `task lint` chưa chạy.
- Filetracker của server ghép đường theo cwd của tiến trình server (có từ trước, P2-c báo 11.4).
- Phần env của C5: biến thường như `$VAR` trong `base_url` theo client đã dựng server (có tài liệu).
- T21b: `.env` của cwd vào tiến trình (thiết kế, D4).
- C9: tool ra ngoài thư mục làm việc; không làm POC.

### Q54. README/docs có đúng không?
Soát độc lập đối chiếu mã, 3 vòng, DAT ở `ef2b8050`. P2-c tự cập nhật CHANGELOG và docs theo quyết định Q5. Lượt soát
docs cuối sau P2-c: chưa có lượt soát docs riêng sau P2-c; CHANGELOG/docs của P2-c được soát trong 4 vòng soát P2-c (mục 6/f).

### Q55. V3 có tốt hơn V2 một cách đo được không?
- **Có, ở phạm vi đã đo:**
  - mọi ca phơi nhiễm mà POC nhắm tới từ EXPOSED thành NOT_EXPOSED (A.3);
  - bộ test toàn kho không mất test ngoài các xoá có chủ đích;
  - race 0;
  - 5 E2E và 5 cổng xanh;
  - eval thật không hồi quy (12/12).
- **Chưa đo được tốt hơn** ở chi phí, khám phá lại và hiệu ứng trên model thật.

---

## C. Vì sao chọn trạng thái này, và việc cần người dùng

- Không chọn ADOPTED vì:
  - các lớp kiến trúc của giả thuyết V3 không được làm, đúng theo số đo;
  - ứng viên chưa merge;
  - một phần T6 bị chặn.
- Không chọn DEFERRED/REJECTED vì có cải thiện đo được và đã kiểm.
- Không chọn REGRESSED vì không có hồi quy còn mở; hồi quy INTEG3 đã sửa.
- Cần người dùng:
  1. Cho phép merge `v3-candidate` vào `main`, hoặc không.
  2. `CRUSH_HYPER_API_KEY` để ghi lại cassette, nếu muốn đóng phần còn lại của T6.
  3. Bật đầu pha DEW (`spec/DEW_RECONSTRUCTION_WORKFLOW.md`).
