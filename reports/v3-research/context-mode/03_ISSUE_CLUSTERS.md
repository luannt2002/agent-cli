# Context Mode — 03 Cụm lỗi từ issue (BẢN NHÁP OFFLINE)

> **Bản nháp offline.** Chưa đọc thảo luận issue nào: trang issue/PR GitHub cần đăng nhập hoặc `gh` (01 ghi nhận WebFetch trả
> 404 khi không đăng nhập). Nguồn issue ở đây là dữ liệu upstream tự lưu trên nhánh `v2-design` + mã/commit trong clone. Sẽ bổ sung
> nội dung thảo luận, ngày đóng, nhãn và liên kết PR↔issue khi có `gh`.

- Repo: `mksglu/context-mode`, clone `/home/luannt/application/opensource/v3-src/context-mode`, chỉ đọc.
- **REVISION:** `origin/v2-design` @ `7a99ece2` (nguồn issue + mã v2), `origin/main` @ `6006a351` (mã v1).
- Ký hiệu nguyên nhân gốc: **E0** = lời trong bản đồ/ADR/kết luận của upstream · **E1** = đã đọc mã/diff (tệp:dòng hoặc commit) ·
  **E2** = có tệp test (đọc, không chạy). `v2:` = `origin/v2-design:`, `main:` = `origin/main:`.

## 0. Nguồn và giới hạn

| Nguồn | Nội dung | Mức |
|---|---|---|
| `v2:tickets/v2-issue-map.md` | 351 dòng = 173 issue + 178 PR, mỗi dòng: tiêu đề rút gọn, host/OS, lớp, yêu cầu v2, lý do, cột "v2 test" | E0 |
| `v2:tickets/v2-not-closed.md:3` | danh sách lấy bằng `gh issue/pr list --state open` lúc 2026-10-14 01:36 +03 ⇒ cả 351 dòng là mục **ĐANG MỞ** lúc quét | E0 |
| `v2:tickets/data/{is2,pr2}.json` | `createdAt` của 174 issue mở / 182 PR mở; `isall.json`/`prall.json`: tiêu đề đầy đủ + trạng thái (590 issue, 659 PR) | E0 (dữ liệu máy chụp) |
| `v2:docs/v2/review/ISSUE-OUTCOMES-r4.md` | 329 dòng kết luận (closed-by-design / covered-by-test / wont-fix …) + "ARCHAEOLOGY", "INTRODUCED_BY" do agent của upstream viết | E0 |
| `issue-refs.tsv` (thư mục này) | ref `#N` trong commit `main` | E1 |
| Tệp test v2 (`e2e/cases`, `tests/v2`, `e2e/live/cases`) | tên ca GP và số `#N` được nhắc | E1 (tồn tại), không chạy |

Số đo (lệnh chạy 2026-10-19, script ở scratchpad `parse_map.py`, `enrich.py`, `families.py`):
- 351 dòng, 351 số khác nhau. Ngày tạo issue: 2026-03-03 → 2026-10-10; theo tháng: 03: 1 · 06: 11 · 07: 47 · 08: 45 · 09: 53 ·
  10: 16. **172/173 issue tạo sau khi `main` ngừng phát triển (v1.0.169, 2026-06-29)** ⇒ không issue nào trong bản đồ có commit
  sửa trên `main`: giao giữa số trong bản đồ và ref trong `issue-refs.tsv` = 0 (ref lớn nhất có commit fix trên `main` ≈ #875).
- Bảng chéo lớp × verdict (đếm từ bản đồ): A → `REMOVED` 106 · B → `GP` 124 · C → `N/A` 35 · D → `N/A` 12 · Q → `N/A` 13 ·
  M/M+B (PR "merge-now") → `GP` 25 / `N/A` 7. Suy ra (không thấy chú giải chính thức): A = đóng vì v2 bỏ cơ chế; B = yêu cầu hợp
  đồng có ca test; C = tính năng/phạm vi; D = trùng/không phải lỗi; Q = câu hỏi mở; M = PR nên merge vào v1.
- Cột "v2 test": GP 170 · LIVE 2 · REMOVED 110 · N/A 69 (khớp con số upstream tự ghi ở ADR-0003). Trong 170 ca GP, **159 tên
  ca xuất hiện trong ít nhất một tệp test v2, 11 không** (so chuỗi con; xuất hiện ≠ chạy xanh). 116/351 số issue/PR được nhắc
  bằng `#N` trong tệp test v2.
- **Mâu thuẫn trong nguồn upstream:** bản đồ ghi "Agents read bodies+comments" (`v2:tickets/v2-issue-map.md:3`), nhưng bằng chứng
  vòng 4 tự ghi "The cause-to-issue mapping is my reading of the code, not text from the issue bodies (no bodies available)"
  (`v2:tickets/round4-evidence.json:581`). ⇒ mọi "nguyên nhân gốc" chỉ lấy từ bản đồ được coi là E0 yếu.
- **Kết luận của upstream bị chính nó vô hiệu một phần:** ISSUE-OUTCOMES-r4 đóng #1037/#1055/#1053 với lý do "D10 removes hard
  denies" (`v2:docs/v2/review/ISSUE-OUTCOMES-r4.md:117,127`); sau đó bản đồ ghi "removed D10 … from every row (34 places). ADR-0008
  keeps hard denies" (`v2:tickets/v2-issue-map.md:567`). Lý do đóng của các dòng đó đã lỗi thời.

Phân loại họ lỗi: **làm tay** theo tiêu đề đầy đủ (không chỉ từ khoá), mỗi mục một họ chính + họ phụ; bảng đầy đủ ở Phụ lục A
(issue) và B (PR) để đếm lại được. Đây là phán đoán của người đọc tiêu đề, không phải của upstream.

## 1. Số đếm theo họ (từ Phụ lục A/B)

| Họ (overlay §4) | issue (có họ này) | issue (họ chính) | PR (có họ này) |
|---|---|---|---|
| HOST_ADAPTER_MISMATCH | **39** | 17 | 25 |
| (ngoài danh sách ưu tiên) INSTALL/UPGRADE | 24 | 22 | 23 |
| ROUTING_FAILURE | **21** | 19 | 14 |
| FALSE_SUCCESS | **18** | 6 | 9 |
| PROMPT_INTERFERENCE | **16** | 10 | 14 |
| ZOMBIE_OPERATION | **14** | 11 | 12 |
| CONTEXT_OVERHEAD | 13 | 12 | 16 |
| (ngoài) feature/phạm vi | 13 | 13 | 8 |
| PROJECT_IDENTITY / SESSION_CONTAMINATION | 9 | 8 | 6 |
| CAPABILITY_MISMATCH | 9 | 3 | 8 |
| DELETION / mất dữ liệu | 9 | 5 | 8 |
| (ngoài) stats/OBSERVABILITY | 8 | 7 | 7 |
| SECURITY_BOUNDARY_CONFUSION | 7 | 4 | 7 |
| STARTUP_RACE | 5 | 4 | 5 |

Năm họ ưu tiên nhiều issue nhất: **HOST_ADAPTER_MISMATCH 39, ROUTING_FAILURE 21, FALSE_SUCCESS 18, PROMPT_INTERFERENCE 16,
ZOMBIE_OPERATION 14**. Nếu tính cả họ ngoài danh sách, INSTALL/UPGRADE (24) đứng thứ hai — khớp 02 (110 fix "install/heal" trên
`main`).

## 2. Mười họ ưu tiên

Phân loại bản sửa (overlay §18): **LOCAL** = LOCAL_PATCH · **SHARED** = SHARED_ABSTRACTION · **ARCH** = ARCHITECTURAL_FIX.
"Bản sửa v2" = mã trên `v2-design`, chưa phát hành.

### 2.1 HOST_ADAPTER_MISMATCH — 39 issue, 25 PR

- **Issue:** hình dạng hook/plugin theo host #1206 (agy Stop dạng nhóm), #1199/#1187 (OpenCode 2 plugin API), #1201 (OpenClaw import
  động), #1179 (Pi thiếu timestamp), #993 (PowerShell thiếu `&`), #1085 (OpenCode chèn system message, Qwen chỉ nhận một); tên tool
  theo host #1018, #1028, #1052, #1212, #1222; đường dẫn/env theo host #1168, #1210, #1032, #1217, #1208, #1282, #1255; khả năng
  host khác nhau #1048 (plan mode), #1152 (Codex bật hooks mặc định), #874 (OMP không thay được output); bản sửa một host không lan
  #1221, #1069; còn lại ở Phụ lục A.
- **Nguyên nhân gốc:** cùng một hàm routing, ngữ nghĩa thực thi khác nhau theo host; cờ `PlatformCapabilities` khai mà không đọc
  (01 mục 9; `main:src/adapters/types.ts:27-41`) [E1]. Bản sửa riêng host: `5235f3fd` "fix(pi): curl/wget escape hatch + quoted-arg
  false positives (closes #625)" chỉ đụng Pi, sau đó #1221 "OMP adapter hard-blocks … (Pi fix from #625 never ported)" [E1 commit +
  E0 tiêu đề]; `4164d3de` chỉ bảo vệ 1/9 hook SessionStart (01 F5) [E1]. Phần lớn issue còn lại: E0 (tiêu đề + bản đồ).
- **Bản sửa:** v1 — vá từng host (#529: 6 commit "partial — <host>" ngày 05-12; #547: `675fec69`, `16e2bfc8`) → LOCAL. 25 PR mở,
  đa số LOCAL. v2 — một hợp đồng chuẩn + bảng adapter + một entry hook + matcher sinh từ bảng (`v2:src/adapters/table.ts:28-98`,
  `v2:src/core/hooks/dispatch.ts:50-74`, `v2:build/manifests.ts:40-47`); cắt 17 → 10 host → **ARCH + SHARED** [E1].
- **Test hồi quy:** 25/39 issue có ca GP; 23 tên ca có trong tệp test, 2 không có (`GP11/codex-hooks-default-true` #1152,
  `GP3/injected-message-required-fields` #1179) [E1 đếm].
- **Hồi quy về sau (trong v2):** D-M3-F-5 — hook Claude Code chạy với cwd = dự án ⇒ live `MODULE_NOT_FOUND`, phải học lại cách v1
  dùng `${CLAUDE_PLUGIN_ROOT}` (`v2:docs/v2/impl/DEVIATIONS.md:284`) [E1]; matcher sinh từ bảng làm nhánh "MCP ngoài" không tới được
  (02b V2-F5) [E1]; tiền tố tool Claude Code khoá cứng kiểu cài plugin (02b V2-F9) [E1].
- **Bài học tổng quát:** hợp đồng host (tên tool, hình dạng payload/đầu ra, khả năng) là DỮ LIỆU ở một bảng mà mọi đường đọc; chỉ
  khai capability khi có code đọc nó; ca test chạy trên payload thật đã ghi, qua đúng matcher đã sinh.
- **Bug lặp:** CÓ — (a) #625 → #1221 (sửa một adapter, adapter anh em giữ lỗi); (b) #529 → #547 → #1222 (matcher MCP ngoài).

### 2.2 ROUTING_FAILURE — 21 issue, 14 PR

- **Issue:** regex curl/wget #1106 ("residual of #63"), #1125 (bỏ qua pipe), #1269 (hỏng cả lệnh ghép), #1221 (OMP, trong quote);
  WebFetch deny vô điều kiện #938, #984, #1006, #908; nhịp hint #1107, #1267, #1026, #1212; matcher #1222, #960, #1000; lách/lấn
  #1003 (lô song song lọt), #979 (headless vẫn bị chặn), #1142 (sửa `cwd` phá allowlist quyền), #1134 (Codex coi lệnh `echo` thay thế
  là "instruction drift"), #1052, #1069.
- **Nguyên nhân gốc:** regex trên chuỗi đã bỏ quote, quyết định cả lệnh ghép một lần (`main:hooks/core/routing.mjs:733-818`, 01 §2)
  [E1]; nhắc Read luôn bắn vì `guidanceOnce(...) ?? {action:"context",…}` (`main:hooks/core/routing.mjs:854-855`) — đúng mô tả #1267
  [E1]; khoá throttle rơi về `process.ppid` khi không có session (`main:hooks/core/routing.mjs:110-118`) — cơ chế #1107 [E1], "không
  bao giờ throttle dưới Claude Code" [E0]; matcher trần `"mcp__"` (`main:hooks/hooks.json:100`) [E1], việc Claude Code 2.1.195 đổi
  ngữ nghĩa matcher [E0]; ánh xạ "regex → issue" của upstream là đọc mã, không phải đọc issue (`v2:tickets/round4-evidence.json:581`).
- **Bản sửa:** v1 — `3383299c` (#63), `2731ca25` (#166), `5235f3fd` (#625, chỉ Pi), `f8d23289` (#463), `752d5236` (#593) → LOCAL.
  PR mở cạnh tranh cùng chỗ: #1086, #1124, #1128, #1189, #1227 (5 bản vá parser curl), #927/#945 (WebFetch), #1231 (matcher),
  #897/#1265 (PowerShell), #1270 (Read). v2 — lexer + parser + unwrap không regex, quyết theo từng pipeline, `NEVER_DENY_HOSTS`,
  deny không trạng thái, hint khoá theo `session_id` trong DB (`v2:src/core/policy/{lexer,parse,unwrap,decide}.ts`;
  `v2:src/core/hooks/dispatch.ts:223-227`) → **ARCH** [E1].
- **Test hồi quy:** 9/21 có ca GP, cả 9 tên có trong tệp test. Dù bản đồ ghi #1269/#1125/#1221 là `REMOVED` (không cần test), đầu vào
  golden của chúng có trong `tests/v2/policy/{parse,unwrap,decide}.test.mjs` (`npm ci && curl`, `head -c 200`, `git log --grep`)
  [E2].
- **Hồi quy về sau (v2):** bỏ cổng sẵn sàng không thay ⇒ deny vẫn bắn khi server không lên (02b V2-F1/F10); nhánh hint/case C cho
  MCP ngoài không tới được do matcher (02b V2-F5) [E1].
- **Bài học:** hiểu lệnh shell bằng phân tích cú pháp, không regex; quyết định chặn không mang trạng thái; trạng thái throttle khoá
  theo danh tính bền; kiểm luật qua matcher đã sinh của host.
- **Bug lặp:** CÓ, xác nhận bằng tiêu đề tự nhận: #63 (`3383299c`, 03-05) → #166 → #625 (`5235f3fd`, 05-19) → #1106 (08-31,
  "residual of #63") và #1221 (09-29, "Pi fix from #625 never ported"); #593 (`752d5236`, 05-16) → #1026 (08-01), #1107 (08-31),
  #1267 (10-05) cùng cơ chế throttle hint.

### 2.3 FALSE_SUCCESS — 18 issue, 9 PR

- **Issue:** cài nền thất bại im lặng #1280, #1174; nhận diện host âm thầm sai #1255; matcher khớp không gì #1222; doctor báo sai
  #1152 (báo tắt khi đang bật), #1014 (PASS trên index rỗng), #889 (luôn FAIL), #901 (upgrade sửa cây này, doctor đọc cây kia);
  số liệu sai #894 (statusline cứng "~98%", renderer nuốt lỗi import), #1151 (đọc nhị phân tính là tiết kiệm), #1023 (tỉ lệ byte
  trình bày như thời gian), #874 (bytes_avoided luôn 0), #915; cổng nuốt redirect #1053; lách #1003; bỏ qua proxy im lặng #1039;
  cache-heal kiểm sai khoá #1078; tài liệu khẳng định chặn mà không chặn #1130.
- **Nguyên nhân gốc:** kiểm HÌNH DẠNG thay vì HIỆU ỨNG — tiền lệ `4bc292fc` trên `main`: redirect Bash "silently no-op'd in
  production" trong khi test khẳng định đúng JSON (02) [E1]; còn lại E0 (tiêu đề).
- **Bản sửa:** v1 `4bc292fc` (re-pin 5 ca integration) → LOCAL + test; PR mở #1281, #1153, #1250, #1121, #952, #931, #899, #932, #996
  → LOCAL. v2 — doctor so cấu hình như dữ liệu, báo index rỗng (ADR-0010 D3, D9); test tầng E2E chạy bundle thật như tiến trình con
  (ADR-0003) → **ARCH** cho tầng test [E1 khung].
- **Test hồi quy:** 7 có ca GP; 5 tên có trong test, 2 KHÔNG có (`GP10/binary-bytes-excluded-from-savings` #1151,
  `GP11/codex-hooks-default-true` #1152). #1014 được nhắc ở `e2e/cases/GP11-doctor-upgrade.e2e.mjs` và
  `tests/v2/ops/doctor-db.test.mjs` [E1].
- **Hồi quy về sau (trong v2, cùng phương pháp):** test đơn vị gọi thẳng `handleHook` nên xanh cho nhánh MCP ngoài mà host không bao
  giờ gửi tới (02b V2-F5); ca live T-SUB mớm sẵn giao thức cho subagent (02b V2-F6); test "case A" thực chất chạy case C (02b mục b);
  GP6 tự cấp `_meta.sessionId` (02b V2-F3) [E1/E2].
- **Bài học:** xanh phải đo ở ranh giới nơi người dùng thấy hiệu ứng; fixture không được cấp thứ mà môi trường thật không có.
- **Bug lặp:** CÓ ở cấp phương pháp — `4bc292fc` (05-31) rồi lại xuất hiện trong bộ test v2 viết mới [E1].

### 2.4 PROMPT_INTERFERENCE — 16 issue, 14 PR

- **Issue:** classifier của host chặn #946, #967 ("~40% of subagent spawns"), #911; Codex auto-review #1134; vai trò sai #1263 (Pi
  `role:"user"`), #1085; chỉ dẫn cũ phát lại như mệnh lệnh #1259; chỉ dẫn mâu thuẫn #1131, #1130; tiêm sang phiên khác #1274; phá
  prompt cache #1112; văn bản lái quá dài #1031; muốn tắt #964; #941, #903, #1173.
- **Nguyên nhân gốc:** tiêm khối routing vào prompt Agent và SessionStart (`main:hooks/core/routing.mjs:894-916`,
  `main:hooks/sessionstart.mjs:166`) [E1]; giọng CAPS/MANDATORY (01 mục 3) [E1]; upstream tự nhận "classifier tripping … is a symptom
  of the routing-block design itself" (`v2:docs/v2/review/ISSUE-OUTCOMES-r4.md:168`) [E0]; INTRODUCED_BY `e19bf0cc` (khối XML
  SessionStart) theo r4 #967 (dòng 155) [E0].
- **Bản sửa:** v1 không có commit cho #946 (01). PR mở — các phương án cộng đồng: #918 (bỏ khung giống injection), #1034 (khối tự
  xưng tên + opt-out), **#1272 (chuyển chỉ dẫn subagent sang sự kiện `SubagentStart` thay vì sửa prompt Agent)**, #864/#1002/#1264
  (Pi đổi sang `role:"system"`), #1122/#1126 (Pi chỉ tiêm 1 lần/bỏ "anchor" chưa kiểm). v2 — XOÁ tính năng: không tiêm; chỉ dẫn qua
  `instructions` + mô tả tool; lint PR1–PR12 trên output đã build (`v2:.github/workflows/ci.yml:73`) → **ARCH (xoá)** [E1].
- **Test hồi quy:** 5 có ca GP (vd `GP2/pi-hint-not-user-role`, `GP2/stable-prefix-across-requests`), cả 5 có trong test; 11 dòng
  `REMOVED` không có test theo thiết kế; lint prompt là cổng CI [E1].
- **Hồi quy về sau:** upstream tự ghi REGRESSION_RISK MEDIUM vì `CLAUDE.md` của repo vẫn giọng CAPS (r4:168) [E0]. Trên v2: `CLAUDE.md`
  gốc đã xoá nhưng 9 tệp routing trong `configs/` còn (02b mục a 0004); v2 vẫn tiêm hint/`RESUME_HINT`/con trỏ compact qua
  `additionalContext` (`v2:src/core/hooks/dispatch.ts:107-122`) — ngắn, kiểu thường [E1].
- **Bài học:** hạ tầng không viết vào prompt nó không sở hữu; văn bản tiêm phải có chủ, có ngân sách, giọng thường, đi qua kênh host
  hỗ trợ (ý của #1272).
- **Bug lặp:** CÓ — "thêm văn bản → host phản ứng → vá văn bản": 500-word → caveman → gỡ (#482, 02); #233 → #724 → #1034 → #946/#967
  [E1 commit + E0 issue].

### 2.5 ZOMBIE_OPERATION — 14 issue, 12 PR

- **Issue:** server không thoát khi phiên đóng #1100/#1101, #982 (Windows), #983 (Pi reload), #953 (Codex Desktop giữ một server mỗi
  task), #1058 (gỡ cài vẫn còn tiến trình); busy-loop #999 ("23 concurrent instances at 284%+ CPU"), #985; treo không hạn #936, #947;
  không huỷ được #959, #1175; tích tụ worker #951; vòng compact vô hạn #1173.
- **Nguyên nhân gốc:** không có timeout server-side trừ agy (`main:src/server.ts:1493-1503`, 01 §1) [E1]; vòng đời dựa ppid poll
  (`main:src/lifecycle.ts:81-120`) [E1], "Linux/Bun-only" (ADR-0002 r3) [E0]; idle-shutdown thêm rồi revert `a888cac6` (02) [E1].
- **Bản sửa:** v1 — chuỗi #103 → #236 (revert) → #311 → #388 → #565/#568/#595 → revert #602 (`a888cac6`) → #854 → #862 (02) → LOCAL
  nối tiếp. PR mở #987, #1009, #975, #968, #1147, #904/#1093/#1182, #1228. v2 — thoát chỉ bằng stdin EOF, không timer/ps
  (`v2:src/core/lifecycle/exit.ts:11-27`); timeout mặc định = tối đa 600 000 ms, giết cả cây (`v2:src/core/contracts/constants.ts:50`,
  `v2:src/core/tools/execute.ts:182,207`); timeout trong cấu hình hook sinh ra (`v2:build/manifests.ts:52-58`) → **ARCH** [E1].
- **Test hồi quy:** 11/14 có ca GP, cả 11 tên có trong test (vd `GP9/stdin-eof-exit` nhắc #1100 trong
  `e2e/cases/GP9-lifecycle.e2e.mjs`) [E1].
- **Hồi quy về sau:** chính chuỗi v1 (bỏ-thêm-bỏ, #602 phải revert để "restore tools on 12 hosts") [E1]. v2: chưa đo; Windows phải chạy
  CI thật (ADR-0002 r3) [E0].
- **Bài học:** gắn vòng đời vào MỘT tín hiệu không phụ thuộc OS (EOF stdin); mọi thao tác dài có hạn chót phía server; gỡ cài phải dừng
  tiến trình.
- **Bug lặp:** CÓ (02 chuỗi vòng đời; #999 tạo 07-26 sau `#862` 06-22) [E1 commit + E0 issue].

### 2.6 CONTEXT_OVERHEAD — 13 issue, 16 PR

- **Issue:** #1022 (snapshot "advertises <2KB, injects ~196KB"), #1173, #1267, #1107, #1112, #1031 (~6,2K token mô tả tool trên Pi),
  #1004 (thân luật lặp), #1077, #961, #878 (chunk vượt cap), #964, #940, #1047.
- **Nguyên nhân gốc:** snapshot "reference-based" không ngân sách (`main:src/session/snapshot.ts:30,470`, `a0b53c0f`; 01/02) [E1];
  fallback hint (2.2) [E1]; 5/11 mô tả tool > 2048 ký tự (ADR-0005, upstream đo bằng lint) [E0].
- **Bản sửa:** PR mở #1123 (ép ngân sách snapshot), #1270, #898, #921/#922/#1275, #980, #1076, #1097, #1127 → LOCAL. v2 — một trần
  `OUTPUT_CAP=16384`, `STATE_BUDGET=2048` + con trỏ, Claude Code chỉ nhận 1 dòng con trỏ; lint ≤ 1024/mô tả, ≤ 8000 tổng
  (`v2:src/core/contracts/constants.ts:8-12`; ADR-0005 PR7) → **ARCH** [E1].
- **Test hồi quy:** 7 có ca GP, cả 7 có trong test; #1022 nhắc trong `e2e/cases/GP6-journal-resume.e2e.mjs` (khối ≤ 2048 B với 10 000
  hàng) [E2].
- **Hồi quy về sau:** v1: #233 bỏ `<ctx_commands>` rồi #724 làm khối subagent lớn lại (4637 B > 4631 B, đo 01) [E1]. v2: chưa thấy.
- **Bài học:** mỗi văn bản trả về/tiêm có hằng ngân sách, test ở ranh giới; "không cắt" phải đi kèm "chọn trong ngân sách".
- **Bug lặp:** CÓ (#233 → #724; 500-word → caveman → gỡ) [E1].

### 2.7 CAPABILITY_MISMATCH — 9 issue, 8 PR

- **Issue:** #1037 (deny WebFetch/curl cho subagent không với tới MCP), #946, #967, #1055, #1099 (OMP deny chỉ sang `ctx_execute`
  không đăng ký), #1048 (plan mode chặn `ctx_batch_execute`), #1018 (tên tool giả định cài plugin), #1028 (snapshot dùng `ctx_search`
  trần), #1006 (công cụ thay thế không làm được việc của công cụ bị chặn: URL artifact claude.ai).
- **Nguyên nhân gốc:** ba chỗ quyết capability với ba giả định (01 F3: `main:hooks/core/mcp-ready.mjs:73-108`,
  `main:hooks/pretooluse.mjs:167`, `main:hooks/core/routing.mjs:894-916`) [E1]; tiền tố plugin cứng (`main:hooks/core/tool-naming.mjs:20`)
  [E1]; #1006, #1048: E0.
- **Bản sửa:** v1 `85f64d2c` (#794/#834: cho qua khi có `agent_id`/`agent_type`, chỉ Claude Code) → LOCAL; `b1ba5cc9` (#724) → LOCAL.
  PR mở #1256 (readiness theo phiên), #1034, #1272, #1126, #913/#1161/#1043 (tên tool), #1076 (allow-list tool). v2 — một resolver
  tool-set + case A/B/C (`v2:src/core/policy/subagent.ts:28-53`) → **SHARED**; bỏ tiêm → **ARCH**; `NEVER_DENY_HOSTS` cho #1006
  (`v2:src/core/contracts/constants.ts:210`) → LOCAL (bảng); #1018 đóng bằng một kiểu cài duy nhất (`v2:src/adapters/claude-code.ts:6`).
- **Test hồi quy:** #1037 = `LIVE:T-SUB` (agent fixture có lời dặn giao thức, 02b V2-F6); `tests/v2/policy/subagent.test.mjs` 17 ca
  (A/B/C, `--agent` main thread, `disallowedTools`) [E2]; không tệp test nào nhắc #1037, #1055, #946, #967, #1006, #1018 bằng số [E1].
- **Hồi quy về sau (trong v2):** D-M3-F-7 — deny case B ban đầu nêu `ctx_execute`, live subagent gọi và thất bại, đã sửa
  (`v2:docs/v2/impl/DEVIATIONS.md:286`) [E1]; hint, deny case C và dòng `more:` case C vẫn nêu `ctx_*` cho subagent không có (02b V2-F2)
  [E1/E2].
- **Bài học:** capability của ĐÚNG agent đang chạy được giải một lần và truyền cho CẢ chỗ quyết định lẫn chỗ viết văn bản (INV-CM-1/2).
- **Bug lặp:** CÓ, xác nhận: #233 (`245cd8af`) → #724 (`b1ba5cc9`) → #1034 (PR, "INTRODUCED_BY" bản sửa #724 theo r4, E0); #794 →
  `85f64d2c` → #1037 (08-06); v2: D-M3-F-7 → V2-F2.

### 2.8 DELETION / mất dữ liệu — 9 issue, 8 PR

- **Issue:** #1140 (`cleanupOldSessions(7)` xoá phiên đang chạy), #1156 (eviction xoá sự kiện quan trọng nhất trước), #902 (eviction
  xoá sự kiện mới trước), #1024 (dọn DB nội dung cũ xoá phiên sống-nhưng-rảnh, chỉ nhìn mtime WAL), #880 (race dọn với server Pi song
  song làm hỏng store), #1214 (server nuốt `events.md` của mọi dự án), #1079 (nguồn hết hạn theo tuổi dù tệp không đổi), #867 (corrupt
  giữa phiên), #1226 (upgrade âm thầm hoàn tác timeout người dùng sửa).
- **Nguyên nhân gốc:** `getOldSessions` lọc theo `started_at` (`main:src/session/db.ts:1096-1097`) — đúng mô tả #1140 [E1];
  `evictLowestPriority` `ORDER BY priority ASC` (`main:src/session/db.ts:970-974`) trong khi đọc CLAUDE.md/rule được gán `priority: 1`
  (`main:src/session/extract.ts:170-206`) — đúng mô tả #1156 [E1]; dọn content DB theo mtime (`main:src/store.ts:231-232`, gọi tại
  `main:src/server.ts:729`) — cơ chế #1024 [E1], "live-but-idle" [E0]; `4164d3de` chỉ 1/9 hook (01 F5) [E1].
- **Bản sửa:** v1 `4164d3de` (05-10, chỉ Claude Code) → LOCAL. PR mở #1143 (TTL theo hoạt động cuối), #1181/#1158 (đảo cực ưu tiên),
  #871 (heal không mất), #1239/#1235/#1160/#1271 (giữ cấu hình người dùng) → LOCAL. v2 — một DB, `events` không bao giờ bị dọn tự
  động, nguồn hết hạn theo truy cập cuối và bỏ qua hẳn khi còn owner sống, dự án chỉ xoá khi không owner + 14 ngày, corrupt không tự
  xoá (`v2:src/core/store/cleanup.ts:32-133`, `v2:src/core/store/db.ts:143-145`) → **ARCH** [E1].
- **Test hồi quy:** chỉ 1 issue có ca GP trong bản đồ; phần lớn `REMOVED`. Tên `GP8/v22-project-cleanup` (bản đồ, cho PR #1143/#970)
  KHÔNG có trong test, nhưng nội dung V22 có dưới tên khác: `tests/v2/store/cleanup.test.mjs:37-145` (`GP8/cleanup-live-owner-kept
  (V22)`, `GP8/cleanup-dead-owner-old-deleted`, project corrupt bị bỏ qua) [E2]. Không có ca race giữa `sweepDeadProjects` và server mới
  mở dự án.
- **Hồi quy về sau (v2, mới nghi):** khe `rmSync` ngoài khoá (02b V2-F7); `events` tăng không giới hạn; ghi journal detached lỗi
  không đếm (02b mục c).
- **Bài học:** điều kiện xoá dùng độ sống + hoạt động cuối, chạy trong cùng khoá với lệnh xoá; không xoá theo nhãn ưu tiên suy diễn;
  test xoá phải gieo dữ liệu "sống nhưng rảnh".
- **Bug lặp:** CÓ ở cấp lớp — bốn đường dọn khác nhau, mỗi đường xoá dữ liệu sống: orphan (`4164d3de`, 05-10), phiên (#1140, 09-09),
  eviction (#1156, 09-13; #902, 07-01), content DB (#1024, 08-01; #880, 06-26) [E1 cơ chế, E0 triệu chứng].

### 2.9 SECURITY_BOUNDARY_CONFUSION — 7 issue, 7 PR

- **Issue:** #1141 (script debug in secret trong `settings.json` vào issue công khai), #1075 (mẫu quyền neo `//`, `~/` không bao giờ
  khớp ⇒ cổng deny lọt), #1142 (sửa `cwd` của `ctx_execute` phá allowlist), #954 (giới hạn tệp không nhất quán giữa hai tool), #1017
  (Copilot lấy gốc = thư mục plugin), #944 (Codex đọc sai tệp cấu hình quyền), #1225 (phụ thuộc có lỗ hổng).
- **Nguyên nhân gốc:** mô phỏng `permissions.deny` của host bằng so chuỗi/regex (01 mục 8; `main:src/security.ts:354-395`) [E1];
  từng issue cụ thể: E0.
- **Bản sửa:** PR mở #1144/#1155 (redact), #1087 (mở rộng neo), #1019 (quyền dự án Pi), #1165 (Codex sandbox), #1294 (symlink), #1245
  (deps) → LOCAL. v2 — báo cáo redaction theo kiểu enum (`v2:src/core/ops/report.ts:23-40`), gốc giới hạn từ `git worktree list`
  (`v2:src/core/project/resolve.ts:132-138`), `run.mjs` chạy lại luật deny của người dùng (DEVIATIONS D-M3-X1-1) → **SHARED** [E1].
- **Test hồi quy:** 6/7 có ca GP, cả 6 tên có trong test (vd `GP3/v17-read-glob-deny-anchors` #1075; #1141 nhắc trong
  `tests/v2/ops/report.test.mjs`) [E1].
- **Hồi quy về sau (v2):** denylist env tiến trình con thu hẹp từ ~90 biến xuống nhóm `NODE_*_OPTIONS` (02b V2-F8) [E1]; vẫn không cô
  lập FS/mạng (như v1).
- **Bài học:** mô phỏng quyền của host phải chuẩn hoá neo đúng như host và test theo ngữ nghĩa host; đầu ra debug là allowlist có kiểu.
- **Bug lặp:** CHƯA xác nhận offline (không thấy chuỗi sửa → hỏng lại); thu hẹp env của v2 là ứng viên hồi quy mới.

### 2.10 STARTUP_RACE — 5 issue, 5 PR

- **Issue:** #1055 (isMCPReady đếm server của phiên anh em), #1053 (cổng nuốt mọi redirect ở OpenCode nhúng), #1099, #880, #1159 (Windows
  dò `where` 30 s trước `initialize`).
- **Nguyên nhân gốc:** sẵn sàng là trạng thái MÁY (01 F3, `main:hooks/core/mcp-ready.mjs:73-108`, EPERM coi là sống) [E1]; INTRODUCED_BY
  `f4e8d2ac` cho #1055 theo r4 (dòng 117) [E0].
- **Bản sửa:** v1 — `b4edff18` → revert `415ce57d` → `04569b12` (#230, 04-13) → `35e264ca`/`e56f17c1` (#347, 04-26) → `f4e8d2ac` (#354,
  05-02) → `ff5b0cff` (#844, 06-22) → LOCAL ×4 vòng. PR mở #1256 (theo phiên), #1113, #1167, #1056, #888. v2 — xoá cổng, KHÔNG thay
  (02b mục b) → "ARCH bằng xoá", nhưng mở lại lớp gốc #230 (deny khi server không có).
- **Test hồi quy:** #1159 có `GP4/cold-start-no-path-probes` (có trong test); #1055/#1053 `REMOVED`; ca golden ADR-0008 "#173 server chưa
  nối → tool chạy" và ca "disconnect" của TECH-SPEC §3.8 không tồn tại (`git grep` = 0) [E1].
- **Hồi quy về sau:** #1055 (08-15), #1053 (08-14) mở SAU vòng sửa thứ 4 (`ff5b0cff`, 06-22) [E1 commit + E0 issue].
- **Bài học:** "sẵn sàng" phải là thuộc tính của ĐÚNG phiên/agent; bỏ một cổng phải kèm ca test cho trạng thái "dịch vụ không lên".
- **Bug lặp:** CÓ — mạnh nhất: 4 vòng sửa + 2 issue sau vòng cuối + bản viết lại bỏ cơ chế mà không có test cho lớp gốc.

## 3. IssueRecord chi tiết (issue then chốt)

Trường `state` theo overlay §3: trạng thái GitHub lúc quét là OPEN cho mọi mục (v2:tickets/v2-not-closed.md:3); trạng thái thật sự
của lỗi được ghi theo `main` (v1) và `v2-design`. `agent_cli_analog` để agent khác điền.

```
IssueRecord { project: context-mode, issue_id: 1055,
  title: "isMCPReady() counts sibling sessions' servers: WebFetch/curl denied in concurrent sessions whose own MCP server never started",
  date: tạo 2026-08-15 (is2.json), affected_version: v1.0.169 (main đóng băng; suy luận),
  symptom (E0 tiêu đề): phiên có server không lên vẫn bị deny vì server của phiên KHÁC còn sống,
  user_impact: deny ngõ cụt — chỉ sang ctx_* mà phiên đó không có, subsystem: PreToolUse readiness gate,
  root_cause: sentinel toàn máy, quét mọi /tmp/context-mode-mcp-ready-<pid> (main:hooks/core/mcp-ready.mjs:73-108) [E1],
  root_cause_confidence: cao cho cơ chế (E1), trung bình cho kịch bản (E0),
  state: main = OPEN/không sửa; v2 = cổng bị xoá, KHÔNG có cổng thay (v2:tickets/v2-issue-map.md:149; 02b V2-F1) ⇒ DESIGN_LIMITATION,
  fixing_pr: #1256 (mở, "scope MCP readiness to the calling Claude Code session"), fixing_commit: không có,
  regression_test: không (REMOVED); golden #173 của ADR-0008 không có test,
  later_regression: là hồi quy của chuỗi #230→#347→#354→#844 (02); v2 có thể mở lại #230 (deny khi server không có),
  current_code_status: main:hooks/core/mcp-ready.mjs:73-108; v2:src/core/policy/decide.ts:52-75 (không kiểm sẵn sàng),
  reproducibility: 2 phiên cùng máy, phiên B tắt server ⇒ PreToolUse curl ở B bị deny,
  architectural_lesson: trạng thái dịch vụ phải gắn danh tính phiên/agent; bỏ cổng phải kèm test "dịch vụ không lên",
  agent_cli_analog: (agent khác), proposed_agent_cli_regression_test: T11 (02b) }

IssueRecord { project: context-mode, issue_id: 1037,
  title: "PreToolUse routing denies WebFetch/curl for subagents that cannot reach the MCP server", date: tạo 2026-08-06,
  affected_version: ≥ v1.0.163 (sau `85f64d2c`; suy luận), symptom: subagent bị deny và được bảo dùng ctx_* nó không có,
  user_impact: subagent kẹt/bỏ việc, subsystem: routing × capability subagent,
  root_cause: `85f64d2c` chỉ xét agent_id/agent_type trong hook Claude Code; host/đường khác vẫn deny (01 F3) [E1 cơ chế, E0 kịch bản],
  root_cause_confidence: trung bình, state: main OPEN; v2 PARTIALLY_FIXED — case B sửa (D-M3-F-7), case C/hint vẫn nêu ctx_* (02b V2-F2),
  fixing_pr: không, fixing_commit: v2 `dispatch.ts` caseB (v2:src/core/hooks/dispatch.ts:244-252),
  regression_test: LIVE:T-SUB (v2:e2e/live/cases/subagent-cap.json — fixture mớm giao thức, 02b V2-F6); dispatch.test.mjs:263-278 (B) [E2],
  later_regression: lý do đóng r4 "D10 removes hard denies" (ISSUE-OUTCOMES-r4.md:127) đã lỗi thời (v2-issue-map.md:567),
  current_code_status: v2:src/core/hooks/dispatch.ts:214-228, 311-326; v2:src/core/index/select.ts:59-62,
  reproducibility: subagent `tools: Read, WebFetch` ⇒ WebFetch URL thường ⇒ đọc lý do deny,
  architectural_lesson: INV-CM-1 cho cả văn bản, agent_cli_analog: (agent khác), proposed_agent_cli_regression_test: T10 (02b) }

IssueRecord { project: context-mode, issue_id: 967,
  title: "Auto-mode permission classifier denies ~40% of subagent spawns due to injected <context_window_protection> block",
  date: tạo 2026-07-17, affected_version: ≤ v1.0.169, symptom (E0): host chặn lời gọi subagent,
  user_impact: PROMPT_INTERFERENCE với agent bên thứ ba, subsystem: tiêm prompt Agent,
  root_cause: nối khối routing vào mọi prompt Agent (main:hooks/core/routing.mjs:894-916) [E1]; INTRODUCED_BY e19bf0cc, ec53f516 theo r4 [E0],
  root_cause_confidence: cao cho cơ chế, trung bình cho tỉ lệ 40 % (E0, chưa đo),
  state: main OPEN; v2 FIXED bằng xoá (không nhánh Agent, v2:src/core/policy/decide.ts:52-75),
  fixing_pr: #1034, #1272, #918 (mở, không merge), fixing_commit: v2 (không commit riêng; ADR-0004/0008 mục 8),
  regression_test: REMOVED (không cần) + lint prompt (v2:.github/workflows/ci.yml:73),
  later_regression: chưa (v2 chưa phát hành), current_code_status: như trên,
  reproducibility: Claude Code auto-mode + một plugin bên thứ ba gọi Agent, so tỉ lệ bị từ chối khi bật/tắt context-mode,
  architectural_lesson: INV-CM-3, agent_cli_analog: (agent khác), proposed_agent_cli_regression_test: T3 (01) }

IssueRecord { project: context-mode, issue_id: 1267,
  title: "guidanceOnce(\"read\") ?? {...} re-injects the Read tip on every Read of a file over 50 KB", date: tạo 2026-10-05,
  affected_version: ≤ v1.0.169, symptom: nhắc Read mỗi lần đọc tệp > 50 KB, user_impact: CONTEXT_OVERHEAD lặp,
  subsystem: PreToolUse Read hint, root_cause: toán tử `??` biến "đã nhắc" (null) thành nhắc lại (main:hooks/core/routing.mjs:854-855) [E1],
  root_cause_confidence: cao, state: main OPEN; v2 FIXED (deny Read lớn không offset/limit + hint 1 lần/phiên khoá DB:
  v2:src/core/policy/decide.ts:245-255; v2:src/core/hooks/dispatch.ts:223-227),
  fixing_pr: #1270 (mở), fixing_commit: v2 (không commit riêng),
  regression_test: tests/v2/hooks/throttle.test.mjs:46 `GP3/hint-once-per-session` [E2],
  later_regression: thuộc chuỗi #593 → #1026/#1107/#1267, current_code_status: như trên,
  reproducibility: hai Read tệp 60 KB cùng phiên ⇒ đếm additionalContext,
  architectural_lesson: trạng thái "đã báo" phải bền và có test hai lần gọi, agent_cli_analog: (agent khác),
  proposed_agent_cli_regression_test: "gọi hai lần, nhắc một lần" cho mọi hint }

IssueRecord { project: context-mode, issue_id: 1222,
  title: "Claude Code: bare `mcp__` hook matchers match nothing since CC v2.1.195 (exact-match semantics)", date: tạo 2026-09-29,
  affected_version: v1.0.169 trên CC ≥ 2.1.195 (E0), symptom: hook MCP ngoài không bao giờ chạy — routing im lặng tắt,
  user_impact: FALSE_SUCCESS (tưởng có bảo vệ), subsystem: cấu hình hook Claude Code,
  root_cause: matcher trần `"mcp__"` (main:hooks/hooks.json:100) [E1] + host đổi ngữ nghĩa matcher [E0],
  root_cause_confidence: trung bình, state: main OPEN; v2: matcher sinh từ bảng, CHỈ native + tiền tố của chính nó ⇒ tool MCP ngoài
  cố ý không có hook (#960) nhưng ADR-0008/0016 vẫn hứa route chúng (02b V2-F5) ⇒ DESIGN_LIMITATION,
  fixing_pr: #1231 (mở, khôi phục `mcp__.*`), fixing_commit: v1 trước đó `16e2bfc8`, `675fec69` (#547),
  regression_test: GP11/hook-matcher-valid-regex (tên có; tests/v2/ops/doctor-config.test.mjs nhắc #1222); GP3/unrelated-mcp-tool-no-hook
  (v2:e2e/cases/GP3-hooks.e2e.mjs:151-161) [E2],
  later_regression: chuỗi #529 → #547 → #1222, current_code_status: v2:build/manifests.ts:40-43; v2:src/core/policy/decide.ts:73,
  reproducibility: Claude Code ≥ 2.1.195, gọi một tool MCP bên thứ ba, xem hook có chạy không,
  architectural_lesson: nhánh luật nào cũng phải tới được qua matcher đã sinh (T12), agent_cli_analog: (agent khác),
  proposed_agent_cli_regression_test: T12 (02b) }

IssueRecord { project: context-mode, issue_id: 1106,
  title: "PreToolUse curl/wget regex still matches unquoted argument words, not the command (residual of #63)", date: tạo 2026-08-31,
  affected_version: ≤ v1.0.169, symptom: lệnh có chữ "curl" ở vị trí đối số bị coi là curl, user_impact: deny/redirect sai,
  subsystem: Bash routing, root_cause: regex `\s(curl|wget)\s` trên chuỗi (01 §2; dẫn chứng upstream v2:tickets/round4-evidence.json:484) [E1/E0],
  root_cause_confidence: cao, state: main OPEN; v2 FIXED bằng parser theo argv0 (v2:src/core/policy/decide.ts:183-189),
  fixing_pr: #1189, #1086, #1124, #1128 (mở, cạnh tranh), fixing_commit: v1 `3383299c` (#63, 03-05) không đủ,
  regression_test: tests/v2/policy/{parse,decide}.test.mjs (đầu vào `git log --grep="curl"`) [E2],
  later_regression: chính nó là hồi quy của #63, current_code_status: như trên,
  reproducibility: `git log --grep curl` qua hook v1, architectural_lesson: phân tích cú pháp thay regex,
  agent_cli_analog: (agent khác), proposed_agent_cli_regression_test: bảng đầu vào shell (quote, heredoc, pipe, wrapper) cho mọi luật chặn lệnh }

IssueRecord { project: context-mode, issue_id: 1140,
  title: "cleanupOldSessions(7) evicts active sessions — started_at is never refreshed by ensureSession/resume", date: tạo 2026-09-09,
  affected_version: ≤ v1.0.169, symptom: phiên dài > 7 ngày bị xoá khi đang dùng, user_impact: mất dữ liệu phiên âm thầm,
  subsystem: SessionDB cleanup, root_cause: `WHERE started_at < datetime('now', -7 days)` (main:src/session/db.ts:1096-1097; gọi qua
  cleanupOldSessions db.ts:1696-1705) [E1], root_cause_confidence: cao,
  state: main OPEN; v2 FIXED theo cấu trúc (không bảng session_meta; nguồn hết hạn theo truy cập cuối, bỏ qua khi còn owner:
  v2:src/core/store/cleanup.ts:32-62), fixing_pr: #1143 (mở), fixing_commit: không trên main,
  regression_test: tests/v2/store/cleanup.test.mjs:54 ("keeps a row whose session is tied to a live owner") [E2],
  later_regression: cùng lớp với 4164d3de, #1156, #1024 (2.8), current_code_status: như trên,
  reproducibility: phiên started_at 8 ngày trước còn ghi sự kiện ⇒ khởi động phiên mới ⇒ đếm sự kiện,
  architectural_lesson: điều kiện xoá theo hoạt động cuối + độ sống, agent_cli_analog: (agent khác),
  proposed_agent_cli_regression_test: gieo phiên "cũ nhưng còn hoạt động" rồi chạy dọn }

IssueRecord { project: context-mode, issue_id: 1156,
  title: "evictLowestPriority deletes the most critical events first (ORDER BY priority ASC on a 1=critical scale)", date: tạo 2026-09-13,
  affected_version: ≤ v1.0.169, symptom: khi phiên > 1000 sự kiện, sự kiện quan trọng (rule/CLAUDE.md) bị xoá trước,
  user_impact: resume mất ngữ cảnh quan trọng, subsystem: SessionDB eviction,
  root_cause: `ORDER BY priority ASC` (main:src/session/db.ts:970-974) + rule/CLAUDE.md gán priority 1 (main:src/session/extract.ts:170-206) [E1],
  root_cause_confidence: cao, state: main OPEN; v2 OBSOLETE (bỏ eviction theo nhãn; khối compact chọn theo điểm nguyên, v2:src/core/contracts/constants.ts:29-31),
  fixing_pr: #1181, #1158 (mở), fixing_commit: không, regression_test: e2e/cases/GP6-journal-resume.e2e.mjs `[M4] GP6/priority-polarity`
  (ca bản đồ gán cho PR #1181, không nhắc #1156 bằng số) [E2],
  later_regression: #902 (07-01) cùng cơ chế, current_code_status: như trên, reproducibility: 1001 sự kiện gồm 1 rule priority 1 ⇒ xem cái nào bị xoá,
  architectural_lesson: thang ưu tiên phải có một chiều được test ở biên, agent_cli_analog: (agent khác),
  proposed_agent_cli_regression_test: "tràn dung lượng giữ mục quan trọng nhất" }

IssueRecord { project: context-mode, issue_id: 1022,
  title: "Resume snapshot ignores byte budget — advertises <2KB, injects ~196KB", date: tạo 2026-08-01, affected_version: ≥ `a0b53c0f` (04-04),
  symptom: SessionStart compact tiêm ~196 KB, user_impact: CONTEXT_OVERHEAD, có thể góp vào #1173 (vòng compact; suy luận),
  subsystem: snapshot resume, root_cause: snapshot "reference-based", maxBytes bị bỏ qua (main:src/session/snapshot.ts:30,470) [E1],
  root_cause_confidence: cao, state: main OPEN; v2 FIXED (STATE_BUDGET 2048, con trỏ, v2:src/core/compaction/render.ts:16-18),
  fixing_pr: #1123 (mở), fixing_commit: v2, regression_test: e2e/cases/GP6-journal-resume.e2e.mjs (nhắc #1022; khối ≤ 2048 B với 10 000 hàng) [E2],
  later_regression: chưa, current_code_status: như trên, reproducibility: phiên 10 000 sự kiện ⇒ compact ⇒ đo byte tiêm,
  architectural_lesson: "không cắt" phải đi với "chọn trong ngân sách", agent_cli_analog: (agent khác), proposed_agent_cli_regression_test: T-budget cho mọi văn bản tiêm }

IssueRecord { project: context-mode, issue_id: 1100,
  title: "Session close (SIGTERM/SIGHUP/Ctrl+D) does not terminate the MCP server — orphan bun process leaks CPU and grows SQLite …",
  date: tạo 2026-08-29, affected_version: ≤ v1.0.169, symptom: server mồ côi ăn CPU, user_impact: ZOMBIE_OPERATION, subsystem: vòng đời server,
  root_cause: phát hiện cha chết bằng poll ppid (main:src/lifecycle.ts:81-120) [E1] không phủ mọi đường đóng [E0],
  root_cause_confidence: trung bình, state: main OPEN; v2 FIXED (thoát theo stdin EOF, v2:src/core/lifecycle/exit.ts:11-27),
  fixing_pr: #987 (Windows PID probe, mở), fixing_commit: v2, regression_test: e2e/cases/GP9-lifecycle.e2e.mjs (`GP9/stdin-eof-exit`, nhắc #1100) [E2],
  later_regression: chuỗi vòng đời v1 (02), current_code_status: như trên, reproducibility: đóng phiên bằng Ctrl+D ⇒ `ps` tìm server,
  architectural_lesson: một tín hiệu vòng đời không phụ thuộc OS, agent_cli_analog: (agent khác), proposed_agent_cli_regression_test: T8 (02) }

IssueRecord { project: context-mode, issue_id: 1014,
  title: "No supported way to assert a source is populated: doctor says PASS on an empty index, ctx_stats measures session usage, …",
  date: tạo 2026-07-29, affected_version: ≤ v1.0.169, symptom: doctor xanh khi index rỗng, user_impact: FALSE_SUCCESS,
  subsystem: ctx_doctor, root_cause: kiểm hình dạng, không kiểm nội dung [E0], root_cause_confidence: thấp (chưa đọc mã doctor v1),
  state: main OPEN; v2 FIXED theo thiết kế (D9 báo index rỗng, ADR-0010), fixing_pr: không, fixing_commit: v2,
  regression_test: e2e/cases/GP11-doctor-upgrade.e2e.mjs, tests/v2/ops/doctor-db.test.mjs (nhắc #1014) [E2],
  later_regression: chưa, current_code_status: v2:src/core/tools/doctor.ts, reproducibility: dự án mới ⇒ ctx_doctor,
  architectural_lesson: kiểm sức khoẻ phải nhìn dữ liệu thật, agent_cli_analog: (agent khác),
  proposed_agent_cli_regression_test: health-xanh-mà-rỗng (overlay §15) }

IssueRecord { project: context-mode, issue_id: 1075,
  title: "permission patterns anchored with // or ~/ never match — ctx_execute_file blocks allowed paths and the deny gate …",
  date: tạo 2026-08-24, affected_version: ≤ v1.0.169, symptom: luật deny của người dùng không áp ⇒ đọc được tệp bị cấm,
  user_impact: SECURITY_BOUNDARY_CONFUSION, subsystem: mô phỏng permissions.deny,
  root_cause: khớp chuỗi không mở rộng neo như host (01 mục 8; main:src/security.ts:354-395) [E1 khung, E0 chi tiết],
  root_cause_confidence: trung bình, state: main OPEN; v2 FIXED theo ADR-0008 golden (user deny `Read(//etc/**)`, `Read(~/secret/**)`),
  fixing_pr: #1087 (mở), fixing_commit: v2 `v2:src/core/policy/user-deny.ts`, regression_test: `GP3/v17-read-glob-deny-anchors` (tên có trong test) [E1],
  later_regression: chưa, current_code_status: như trên, reproducibility: deny `Read(//etc/**)` rồi đọc /etc/hosts,
  architectural_lesson: mô phỏng quyền host phải test theo ngữ nghĩa host, agent_cli_analog: (agent khác),
  proposed_agent_cli_regression_test: bảng mẫu quyền × đường dẫn (tuyệt đối, ~, tương đối, symlink) }
```

## 4. Ứng viên bug lặp (overlay §17)

| Chuỗi | Bằng chứng | Mức xác nhận | Vì sao lớp lỗi cứ quay lại |
|---|---|---|---|
| Sẵn sàng MCP: #230 → #347 → #354 → #844 → #1055/#1053 → v2 bỏ cổng không thay | 6 commit `main` (2.10) + 2 issue tạo sau vòng 4 + mã v2 | **XÁC NHẬN** (E1 commit/mã + E0 tiêu đề) | không có chủ sở hữu "danh tính phiên"; trạng thái dịch vụ đặt ở cấp máy |
| Capability subagent: #233 → #724 → #1034; #794 → #834 → #1037; v2 D-M3-F-7 → V2-F2 | commit `245cd8af`, `b1ba5cc9`, `85f64d2c`; DEVIATIONS.md:286; mã v2 | **XÁC NHẬN** (E1) | mỗi bản sửa special-case một nhánh; chỗ VIẾT văn bản không đọc chỗ QUYẾT capability |
| Regex lệnh curl: #63 → #166 → #625 → #1106 ("residual of #63"), #1221 ("never ported") | commit `3383299c`, `2731ca25`, `5235f3fd` + tiêu đề tự nhận | **XÁC NHẬN** (E1 + E0 tiêu đề) | regex trên chuỗi; bản sửa nằm trong một adapter |
| Matcher MCP ngoài: #529 → #547 → #1222/#960 → v2 matcher bỏ MCP ngoài trong khi luật còn | `855e330d`…, `675fec69`, `16e2bfc8`; `main:hooks/hooks.json:100`; 02b V2-F5 | **XÁC NHẬN** (E1) | luật và matcher là hai nguồn sự thật |
| Throttle hint: #463/#593 → #1026, #1107, #1267, #1212 | `f8d23289`, `752d5236`; `main:hooks/core/routing.mjs:110-118,854-855` | **XÁC NHẬN** cơ chế (E1), triệu chứng E0 | trạng thái "đã nhắc" lưu theo tiến trình/ppid, không bền |
| Vòng đời server: #103 → #236 (revert) → … → revert #602 → #854/#862 → #999, #1100 | 02 chuỗi | **XÁC NHẬN** (E1 + E0) | nhiều cơ chế phát hiện chết (ppid, idle, sibling) chồng nhau |
| Dọn dữ liệu xoá dữ liệu sống: `4164d3de` → #1140, #1156, #902, #1024, #880 | mã `main` (2.8) | ứng viên mạnh (khác đường, cùng lớp) | mỗi đường dọn tự định nghĩa "cũ"; không có định nghĩa độ sống chung |
| Tiêm văn bản: 500-word → caveman → gỡ (#482); routing block → #946/#967 → v2 xoá | 02; r4:155,168 | **XÁC NHẬN** (E1 commit) | tính năng sống bằng văn bản tiêm vào prompt người khác |
| Kiểm theo hình dạng: `4bc292fc` (v1) → test v2 bỏ qua matcher/tự cấp `_meta`/mớm fixture | 02; 02b V2-F3, F5, F6 | **XÁC NHẬN** ở cấp phương pháp (E1/E2) | test đứng ở tầng trong, không ở ranh giới người dùng |
| Cài/heal: #644 → #1096 ("the one instance #644 missed"), #181…#814 (02) | `11474111`, `79e7bb52`; tiêu đề #1096 | XÁC NHẬN (E1 + E0) — ngoài 10 họ ưu tiên | đường dẫn cài trôi theo host/runtime |

## 5. Phát hiện chéo (không kết luận phơi nhiễm agent-cli)

- **"Đóng vì bỏ tính năng" ≠ "đã sửa".** 110 dòng đóng bằng `REMOVED` không có test theo thiết kế. Ít nhất một chỗ (cổng sẵn sàng) việc
  bỏ cơ chế mở lại lớp lỗi gốc mà không có ca test (2.10). Khi đọc bản đồ upstream, coi `REMOVED` là "chưa kiểm" cho tới khi có ca
  test cho trạng thái mà cơ chế cũ từng bảo vệ.
- **PR cộng đồng mở nhưng không ai merge:** 178 PR mở lúc quét; riêng lô P0 (45 PR) bản đồ ghi "All authors are external
  contributors … CI: no check runs on any PR (fork workflows wait for maintainer approval). Only #1228 has a review approval"
  (`v2:tickets/v2-issue-map.md:277`) [E0]; repo tắt PR từ v2 (ADR-0014). Nhiều PR cạnh tranh sửa cùng chỗ (5 bản parser curl; 3 bản Pi `role`). Đây là nguồn "phương án thay thế của cộng đồng"
  (overlay §2) — ví dụ #1272 chuyển chỉ dẫn subagent sang `SubagentStart` thay vì sửa prompt.
- **Kiểu sửa v1 vs v2:** theo phân loại ở mục 2, 8/10 họ ưu tiên có bản sửa v2 loại ARCH (STARTUP_RACE là "ARCH bằng xoá", chưa có
  thay thế); CAPABILITY và SECURITY chủ yếu SHARED; bản sửa v1 và PR cộng đồng gần như toàn LOCAL. Phần còn hở nằm đúng ở chỗ nối giữa các abstraction mới (quyết định ↔ văn bản, hook ↔ server, luật ↔ matcher).

---

## Phụ lục A — 173 issue: phân loại tay

Mã họ: CAP CAPABILITY_MISMATCH · ROUTE ROUTING_FAILURE · PI PROMPT_INTERFERENCE · OVH CONTEXT_OVERHEAD · RACE STARTUP_RACE · ZOMB
ZOMBIE_OPERATION · FALSE FALSE_SUCCESS · DEL DELETION/mất dữ liệu · HOST HOST_ADAPTER_MISMATCH · SEC SECURITY_BOUNDARY_CONFUSION ·
IDENT PROJECT_IDENTITY/SESSION_CONTAMINATION · STALE STALE_CONTEXT · RETR RETRIEVAL · STORAGE SQLite/lưu trữ · EXEC executor ·
FETCH fetch/parser · INSTALL cài/nâng cấp · STATS OBSERVABILITY (số liệu) · PERF chi phí CPU/IO · ENC mã hoá ký tự · TOOLC hợp đồng
tham số tool · WORKER WORKER_FAILURE · DOC tài liệu · FEAT tính năng/phạm vi · NA không phải lỗi. "tên ca trong tệp test": có = chuỗi
tên ca GP xuất hiện trong `e2e/cases`/`tests/v2`/`e2e/live/cases`; KHÔNG = không thấy; — = dòng không có ca GP. "r4" = kết luận
`ISSUE-OUTCOMES-r4.md` (— = không có dòng).

| # | tạo | tiêu đề (rút gọn, theo bản đồ) | họ (chính; phụ) | lớp | v2 test | tên ca trong tệp test | r4 |
|---|---|---|---|---|---|---|---|
| 1282 | 2026-10-10 | Cursor MCP spawn fails, bare npx not on PATH | HOST; INSTALL | B | `GP1/absolute-node-path-cursor-windows` | có | wont-fix |
| 1280 | 2026-10-08 | turndown missing, silent bg install fails | FALSE; INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1274 | 2026-10-08 | OpenCode resume snapshot injected into other session | IDENT; PI | A | `REMOVED:D9` | — | covered-by-test |
| 1269 | 2026-10-06 | curl detector aborts whole compound command | ROUTE | A | `REMOVED:ADR-0008` | — | closed-by-design |
| 1268 | 2026-10-06 | Native ZCode adapter | FEAT | Q | `N/A:deferred-post-v2.0` | — | wont-fix |
| 1267 | 2026-10-05 | Read tip re-injected every Read >50 KB (`??` fallback) | ROUTE; OVH | A | `REMOVED:D11` | — | covered-by-test |
| 1263 | 2026-10-04 | Pi injects memory as role:"user" | PI; HOST | A(partlyB) | `GP2/pi-hint-not-user-role` | có | wont-fix |
| 1262 | 2026-10-04 | parent-liveness spawns `ps` every 30 s | PERF | B | `GP9/no-periodic-ps-spawn` | có | - |
| 1261 | 2026-10-04 | Pre/PostToolUse hooks cost 55-70 ms CPU per Bash | PERF | B | `GP3/hook-perf-posttool-async` | có | - |
| 1259 | 2026-10-04 | intent classifier replayed as live directive on Pi | PI; STALE | A | `REMOVED:D9` | — | covered-by-test |
| 1258 | 2026-10-04 | DeepSeek Harness support | FEAT | Q | `N/A:open-question` | — | wont-fix |
| 1255 | 2026-10-03 | opencode missing from clientInfo map, wrong store | HOST; IDENT,FALSE | A | `REMOVED:D3` | — | covered-by-test |
| 1254 | 2026-10-03 | ctx_stats skips OpenCode (v1 layout only) | STATS; HOST | A | `REMOVED:D3` | — | closed-by-design |
| 1251 | 2026-10-03 | ensure-deps npm retry storm, console windows | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1248 | 2026-10-02 | stats heartbeat reopens every session DB each minute | PERF | B | `GP8/stats-heartbeat-no-timer-db-open` | có | closed-by-design |
| 1242 | 2026-10-02 | ctx-debug.sh fails under POSIX sh | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1226 | 2026-09-30 | hooks.json has no default timeouts | INSTALL; DEL | B | `GP3/hook-timeout-present` | có | covered-by-test |
| 1225 | 2026-09-30 | vulnerable ip-address dep chain | SEC | B | `GP11/npm-audit-zero-high-critical` | có | covered-by-test |
| 1222 | 2026-09-29 | bare `mcp__` matcher matches nothing (CC 2.1.195) | ROUTE; FALSE,HOST | B | `GP11/hook-matcher-valid-regex` | có | closed-by-design |
| 1221 | 2026-09-29 | OMP hard-blocks curl in quotes/grep | ROUTE; HOST | A | `REMOVED:ADR-0008` | — | wont-fix |
| 1219 | 2026-09-28 | No docs for Codex Desktop install | DOC | B | `GP1/install-docs-per-host` | có | wont-fix |
| 1217 | 2026-09-28 | ctx_execute shell has PATH stripped to /usr/bin:/bin | HOST; EXEC | B | `GP4/windows-path-inherit` | có | - |
| 1215 | 2026-09-28 | project-scope plugin enabled globally by heal | INSTALL | A | `REMOVED:D2` | — | covered-by-test |
| 1214 | 2026-09-27 | server takes every project's events.md | IDENT; DEL | A | `REMOVED:D8` | — | closed-by-design |
| 1213 | 2026-09-27 | stats opens every session DB read-write | PERF | B | `GP8/sibling-db-readonly` | có | - |
| 1212 | 2026-09-27 | PowerShell tool not routed; Bash nudge once | ROUTE; HOST | B | `GP3/powershell-routed-like-bash` | có | covered-by-test |
| 1210 | 2026-09-27 | Codex Windows project dir falls back to plugin cache | IDENT; HOST | B | `GP7/codex-session-project-dir` | có | closed-by-design |
| 1208 | 2026-09-26 | ctx_execute python/node fails with shell-script shims | HOST; EXEC | B | `GP4/windows-pyenv-shim` | có | - |
| 1206 | 2026-09-26 | agy rejects hooks.json (grouped Stop) | HOST | B | `GP11/agy-hooks-schema-valid` | có | wont-fix |
| 1205 | 2026-09-26 | Pi bridge on Bun spins, disk I/O error | STORAGE; HOST | A | `REMOVED:D2` | — | covered-by-test |
| 1204 | 2026-09-25 | index-time preview + ctx_page tool | FEAT | Q | `N/A:open-question` | — | - |
| 1201 | 2026-09-24 | OpenClaw capture breaks computed import paths | HOST | B | `GP1/adapter-bundle-static-imports` | có | wont-fix |
| 1200 | 2026-09-24 | ctx_search rejects singular `query` | TOOLC | B | `GP5/search-param-schema-match` | có | covered-by-test |
| 1199 | 2026-09-24 | OpenCode 2.x plugin fails to load | HOST | B | `GP1/opencode-v2-plugin-api` | có | - |
| 1198 | 2026-09-24 | ctx_* events credited to latest session | IDENT | A | `REMOVED:D8` | — | - |
| 1196 | 2026-09-23 | ensure-deps re-codesigns .node every hook | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1193 | 2026-09-22 | Bun seed writes ABI-mismatched cache | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1192 | 2026-09-22 | integrate with pi-blackhole | FEAT | C | `N/A:deferred-post-v2.0` | — | wont-fix |
| 1191 | 2026-09-22 | healScript version filter unanchored | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1188 | 2026-09-21 | ensure-deps dlopen then rebuild, SIGSEGV | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1187 | 2026-09-21 | OpenCode V2 support | HOST | D | `N/A:duplicate-of-#1199` | — | - |
| 1186 | 2026-09-21 | Windows ctx_stats 4 s + 1.6 s execSync probes; coexistence detector | PERF; HOST | B | `GP10/windows-stats-no-execsync-probes` | có | wont-fix |
| 1185 | 2026-09-20 | multi-agent context isolation per worktree | IDENT | C | `GP7/worktree-label-session-first` | có | - |
| 1184 | 2026-09-19 | fetch via Firecrawl API | FEAT | Q | `REMOVED:D15` | — | covered-by-test |
| 1179 | 2026-09-18 | Pi context message without timestamp → 400 | HOST | B | `GP3/injected-message-required-fields` | KHÔNG | covered-by-test |
| 1175 | 2026-09-18 | ctx_execute ignores Esc / no progress | ZOMB | B | `GP4/cancel-kills-child-tree-1s` | có | covered-by-test |
| 1174 | 2026-09-17 | start.mjs background npm install dies silently | INSTALL; FALSE | A | `REMOVED:D2` | — | closed-by-design |
| 1173 | 2026-09-17 | infinite compaction loop burns quota | OVH; PI,ZOMB | A | `REMOVED:D9` | — | covered-by-test |
| 1168 | 2026-09-16 | Pi paths hardcoded to legacy ~/.pi dirs | HOST | B | `GP11/pi-discovery-dirs-doctor` | có | - |
| 1163 | 2026-09-15 | snippet slicing makes lone surrogates → 500s | ENC | B | `GP5/snippet-code-point-safe` | có | - |
| 1162 | 2026-09-15 | start-http.mjs lacks turndown install (systemd) | INSTALL | Q | `N/A:open-question` | — | closed-by-design |
| 1159 | 2026-09-14 | Windows runtime probes take 30 s before initialize | RACE; HOST | B | `GP4/cold-start-no-path-probes` | có | covered-by-test |
| 1156 | 2026-09-13 | eviction deletes critical events first | DEL; STALE | A | `REMOVED:D8` | — | closed-by-design |
| 1152 | 2026-09-13 | doctor says Codex hooks off when default on | FALSE; HOST | B | `GP11/codex-hooks-default-true` | KHÔNG | - |
| 1151 | 2026-09-13 | binary/HTTP bytes counted as saved tokens | STATS; FALSE | B | `GP10/binary-bytes-excluded-from-savings` | KHÔNG | closed-by-design |
| 1150 | 2026-09-13 | Pi renderCall drops args | HOST | B | `GP5/call-row-language-code-line` | có | covered-by-test |
| 1142 | 2026-09-09 | hook cwd rewrite defeats permission allowlist | ROUTE; SEC | B | `GP3/no-updated-input-rewrite` | có | covered-by-test |
| 1141 | 2026-09-09 | ctx-debug.sh dumps secrets; template requires pasting | SEC | B | `GP11/redact-secrets-in-doctor-output` | có | covered-by-test |
| 1140 | 2026-09-09 | cleanupOldSessions evicts active sessions | DEL | A | `REMOVED:D6` | — | closed-by-design |
| 1139 | 2026-09-09 | Windows npm install better-sqlite3 crashes | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1136 | 2026-09-08 | empty `<table>` crashes turndown | FETCH | B | `GP5/rowless-tables-survive` | có | covered-by-test |
| 1135 | 2026-09-08 | fetch error returns raw Node crash stack | FETCH | B | `GP4/fetch-error-one-line-cause` | có | - |
| 1134 | 2026-09-08 | Codex auto-review rejects echo rewrite | PI; ROUTE | A | `REMOVED:ADR-0008` | — | closed-by-design |
| 1131 | 2026-09-08 | CLAUDE.md continuity contradicts routing block | PI | A | `REMOVED:D11` | — | closed-by-design |
| 1130 | 2026-09-08 | CLAUDE.md says blocked, code does not block | PI; FALSE | A | `REMOVED:D11` | — | closed-by-design |
| 1125 | 2026-09-06 | curl pipe to head still redirected, block dropped | ROUTE | A | `REMOVED:ADR-0008` | — | closed-by-design |
| 1119 | 2026-09-03 | ctx_search negative limit | TOOLC | B | `GP5/negative-limit-rejected` | có | closed-by-design |
| 1117 | 2026-09-02 | NODE_OPTIONS prefix breaks `for`/`while` | EXEC | B | `GP4/env-via-spawn-env` | có | covered-by-test |
| 1112 | 2026-09-01 | Pi ephemeral tail injection breaks prompt cache | OVH; PI | B | `GP2/stable-prefix-across-requests` | có | covered-by-test |
| 1107 | 2026-08-31 | guidanceOnce keyed on ppid never throttles; 2149 tmp dirs | OVH; ROUTE | B | `GP3/hint-once-per-session` | có | closed-by-design |
| 1106 | 2026-08-31 | curl regex matches `which curl` | ROUTE | A | `REMOVED:ADR-0008` | — | closed-by-design |
| 1105 | 2026-08-31 | better-sqlite3 ABI rebuild loop | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1101 | 2026-08-29 | same as 1100 | ZOMB | D | `N/A:duplicate-of-#1100` | — | closed-by-design |
| 1100 | 2026-08-29 | Server orphaned after session close (100% CPU) | ZOMB | B | `GP9/stdin-eof-exit` | có | - |
| 1099 | 2026-08-28 | OMP deny points to ctx_execute that is not registered | CAP; RACE | A | `REMOVED:ADR-0008` | — | closed-by-design |
| 1098 | 2026-08-28 | Pi bridge failure after init is sticky | WORKER | B | `GP1/pi-bridge-retry-next-turn` | có | covered-by-test |
| 1096 | 2026-08-27 | healScript statSync ELOOP | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1094 | 2026-08-27 | Pi collapsed view shows only code fence | HOST | B | `GP4/pi-collapsed-view-plain-text-first-line` | có | covered-by-test |
| 1090 | 2026-08-27 | plugin.json command pinned to versioned node path | INSTALL | B | `GP1/plugin-json-unpinned-node` | có | - |
| 1088 | 2026-08-26 | Show input/progress live for ctx_execute | FEAT | C(later) | `N/A:deferred-post-v2.0` | — | wont-fix |
| 1085 | 2026-08-25 | OpenCode extra system message breaks Qwen single-system templates | PI; HOST | B | `GP2/opencode-single-system-message` | có | closed-by-design |
| 1083 | 2026-08-25 | Codex agent greps ~/.codex and ingests own history | FEAT | C(later) | `N/A:deferred-post-v2.0` | — | wont-fix |
| 1081 | 2026-08-24 | timeout unit (ms) missing from schema | TOOLC | B | `GP2/tool-schema-timeout-unit` | có | covered-by-test |
| 1079 | 2026-08-24 | cleanupStaleSources(14) drops unchanged sources by age | DEL | Q | `N/A:open-question` | — | closed-by-design |
| 1078 | 2026-08-24 | cache-heal gates on wrong plugin key | INSTALL; FALSE | A | `REMOVED:D2` | — | closed-by-design |
| 1077 | 2026-08-24 | ctx_* calls verbose in Codex; batch repeats sections per query | OVH | B | `GP5/batch-execute-dedup-section` | có | - |
| 1075 | 2026-08-24 | `//` and `~/` permission anchors never match; deny fails open | SEC | B | `GP3/v17-read-glob-deny-anchors` | có | - |
| 1069 | 2026-08-21 | Pi does not route read/grep, only bash HTTP | ROUTE; HOST | B | `GP3/pi-read-hint-parity` | có | covered-by-test |
| 1068 | 2026-08-21 | cm-fs preload marker leaks to grandchild stderr | EXEC | B | `GP4/no-preload-marker-leak-grandchild` | có | closed-by-design |
| 1067 | 2026-08-19 | pi collapsed row shows ```javascript fence | HOST | B | `GP6/code-echo-default` | có | - |
| 1065 | 2026-08-19 | Hosted Insights D1 SQL vars / owner team | FEAT | C(rejectforOSS) | `N/A:rejected` | — | closed-by-design |
| 1063 | 2026-08-18 | "Is the project abandoned?" | NA | D | `N/A:not-a-bug` | — | wont-fix |
| 1058 | 2026-08-17 | Uninstall leaves cache-heal hook, cache dir, live MCP | ZOMB | A+B | `GP1/clean-uninstall-no-residue` | có | covered-by-test |
| 1055 | 2026-08-15 | isMCPReady counts sibling sessions → WebFetch denied | RACE; CAP | A | `REMOVED:ADR-0008` | — | closed-by-design |
| 1053 | 2026-08-14 | isMCPReady swallows redirects in OpenCode embedded mode | RACE; FALSE,HOST | A | `REMOVED:ADR-0008` | — | closed-by-design |
| 1052 | 2026-08-14 | OpenCode `webfetch` alias missing | ROUTE; HOST | B | `GP3/tool-name-map-opencode-webfetch` | có | closed-by-design |
| 1050 | 2026-08-13 | Pi single-row collapsed rendering | FEAT | C(later) | `N/A:deferred-post-v2.0` | — | wont-fix |
| 1049 | 2026-08-12 | cache-heal re-injects duplicate absolute-path hook | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1048 | 2026-08-12 | Plan mode blocks ctx_batch_execute | CAP; HOST | Q | `N/A:open-question` | — | wont-fix |
| 1047 | 2026-08-12 | Configurable execution echo + summary arg | OVH | Q | `GP6/code-echo-default` | có | covered-by-test |
| 1046 | 2026-08-12 | LongHorizon-Harness integration | FEAT | C(reject) | `N/A:rejected` | — | wont-fix |
| 1045 | 2026-08-11 | neuforge-pay token metering | FEAT | C(reject) | `N/A:rejected` | — | wont-fix |
| 1039 | 2026-08-07 | ctx_fetch_and_index strips HTTP(S)_PROXY | FETCH; FALSE | B | `GP5/http-proxy-honored` | có | closed-by-design |
| 1037 | 2026-08-06 | WebFetch denied for subagents without ctx_* | CAP | A | `LIVE:T-SUB` | — | closed-by-design |
| 1036 | 2026-08-06 | OpenCode cumulative cost appended per step | STATS | B | `GP10/usage-events-idempotent-per-turn` | có | - |
| 1032 | 2026-08-04 | VS Code Remote-WSL root = VS Code install dir | IDENT; HOST | B | `GP7/workspace-folder-remote-wsl` | có | covered-by-test |
| 1031 | 2026-08-03 | Tool descriptions ~6.2K tokens | OVH; PI | Q/B | `GP2/tool-description-token-budget` | có | closed-by-design |
| 1028 | 2026-08-02 | PreCompact snapshot uses bare ctx_search | HOST; CAP | A | `REMOVED:D9` | — | closed-by-design |
| 1026 | 2026-08-01 | Bash nudge once-per-session vs periodic | ROUTE | B | `GP3/hint-once-per-session` | có | wont-fix |
| 1025 | 2026-08-01 | reduction_pct double-counts returned bytes | STATS | B | `GP10/reduction-pct-denominator` | có | - |
| 1024 | 2026-08-01 | Stale DB cleanup deletes live idle sessions (WAL mtime) | DEL | A | `REMOVED:D6` | — | closed-by-design |
| 1023 | 2026-08-01 | "ran N× longer" is byte ratio shown as time | STATS; FALSE | B | `GP10/stats-wording-bytes-not-time` | có | closed-by-design |
| 1022 | 2026-08-01 | Resume snapshot ignores maxBytes (196 KB) | OVH | A | `REMOVED:D9` | — | covered-by-test |
| 1021 | 2026-08-01 | README says Pi hooks High not Full | DOC | A | `REMOVED:D16` | — | closed-by-design |
| 1018 | 2026-07-30 | CC tool naming assumes plugin install | HOST; CAP | B | `GP3/tool-prefix-standalone-mcp` | có | closed-by-design |
| 1017 | 2026-07-30 | Copilot CLI / Codex root = plugin install dir | IDENT; SEC | B | `GP7/adapter-root-path-containment` | có | closed-by-design |
| 1016 | 2026-07-30 | Clean uninstall command | INSTALL | C(v2.0) | `N/A:deferred-v2.0-scope` | — | closed-by-design |
| 1014 | 2026-07-29 | No way to count docs per source; doctor PASS on empty index | FALSE | C(v2.0) | `N/A:deferred-v2.0-scope` | — | closed-by-design |
| 1006 | 2026-07-27 | WebFetch deny breaks claude.ai artifact URLs | ROUTE; CAP | A | `REMOVED:ADR-0008` | — | closed-by-design |
| 1005 | 2026-07-27 | ctx upgrade writes old-version paths (bun) | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1004 | 2026-07-27 | buildRulesSection inlines AGENTS.md into resume | OVH | A | `REMOVED:D9` | — | closed-by-design |
| 1003 | 2026-07-27 | Denies defeated by parallel batches; no WebFetch opt-out | ROUTE; FALSE | A(part)/Q | `LIVE:T-PAR` | — | wont-fix |
| 1000 | 2026-07-26 | PreToolUse double-fires; no PostToolUse timeout | ROUTE; PERF | B | `GP1/hooks-json-one-matcher-per-tool` | có | closed-by-design |
| 999 | 2026-07-26 | start.mjs busy-loop, 23 instances | ZOMB | B | `GP9/idle-cpu-stdin-exit` | có | covered-by-test |
| 993 | 2026-07-23 | Windows hook commands fail under PowerShell | HOST | Q | `N/A:open-question` | — | closed-by-design |
| 992 | 2026-07-23 | Bun SQLITE_IOERR wedge; doctor green | STORAGE | A | `REMOVED:D2` | — | closed-by-design |
| 985 | 2026-07-22 | Multi-session contention on shared DB, busy-wait | STORAGE; ZOMB | A | `REMOVED:D3` | — | closed-by-design |
| 984 | 2026-07-22 | WebFetch no exemption for artifact URLs | ROUTE | D | `N/A:duplicate-of-#1006` | — | closed-by-design |
| 983 | 2026-07-21 | Pi reload leaks MCP helper | ZOMB | B | `GP9/pi-reload-no-leak` | có | closed-by-design |
| 982 | 2026-07-21 | Windows MCP orphans spin after session end | ZOMB; HOST | B | `GP9/windows-client-kill-exit` | có | closed-by-design |
| 979 | 2026-07-20 | Headless CC still denied (dead formatter) | ROUTE; HOST | A | `REMOVED:D7` | — | covered-by-test |
| 967 | 2026-07-17 | Injected <context_window_protection> trips auto-mode classifier | PI; CAP | A | `REMOVED:D11` | — | closed-by-design |
| 964 | 2026-07-16 | Env knob to disable SessionStart routing block | OVH; PI | A | `REMOVED:D11` | — | closed-by-design |
| 962 | 2026-07-16 | RFC: bound trigram/vocab growth for logs/JSONL | STORAGE | Q | `N/A:open-question` | — | closed-by-design |
| 961 | 2026-07-16 | RFC: indexed byte/chunk budgets for batch | OVH | B | `GP5/batch-index-byte-budget` | có | covered-by-test |
| 960 | 2026-07-15 | PreToolUse `mcp__` matcher hits all MCP tools | ROUTE | B | `GP3/unrelated-mcp-tool-no-hook` | có | - |
| 959 | 2026-07-15 | Pi: hung ctx_execute cannot be aborted | ZOMB | B | `GP4/hung-execute-cancel-kills-tree` | có | - |
| 956 | 2026-07-14 | prebuild-install deprecated warning | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 954 | 2026-07-13 | execute_file rejects external worktrees, batch accepts | IDENT; SEC | B | `GP7/worktree-containment-consistent` | có | closed-by-design |
| 953 | 2026-07-13 | Codex Desktop Win keeps one MCP server per task, os error 32 | ZOMB; HOST | B | `GP9/windows-stdio-close-plugin-dir-deletable` | có | wont-fix |
| 951 | 2026-07-12 | Stale cm-fs-preload-<pid>.js via NODE_OPTIONS leak | EXEC; ZOMB | B | `GP4/node-options-no-leak-across-servers` | có | covered-by-test |
| 950 | 2026-07-11 | ctx_stats self-contradicts (part > whole) | STATS | B | `GP10/stats-totals-consistent` | có | - |
| 949 | 2026-07-11 | sessions/ has no retention | STORAGE | A | `REMOVED:D6` | — | covered-by-test |
| 947 | 2026-07-10 | batch timeout does not bound indexing/search | ZOMB | B | `GP4/batch-timeout-bounds-whole-handler` | có | - |
| 946 | 2026-07-10 | Agent-prompt injection blocked by auto-mode classifier | PI; CAP | A | `REMOVED:ADR-0008` | — | closed-by-design |
| 944 | 2026-07-10 | Codex external-file check reads wrong settings | SEC; HOST | Q | `N/A:open-question` | — | closed-by-design |
| 941 | 2026-07-09 | Codex guidance: JS default, not only | PI; HOST | A | `REMOVED:D11` | — | closed-by-design |
| 940 | 2026-07-09 | OS-specific Codex guidance overlay | OVH | A | `REMOVED:D11` | — | closed-by-design |
| 938 | 2026-07-09 | WebFetch redirect breaks claude.ai artifact URLs | ROUTE | A | `REMOVED:ADR-0008` | — | closed-by-design |
| 936 | 2026-07-09 | stdio ctx_execute hangs, no server timeout | ZOMB | B | `GP4/host-rpc-timeout-governs-no-default` | có | - |
| 925 | 2026-07-06 | Wrapper breaks commands starting with if/for/( | EXEC | B | `GP4/shell-syntax-with-spawn-env` | có | covered-by-test |
| 923 | 2026-07-06 | start.mjs chdir breaks terminal cwd | EXEC | B | `GP9/no-chdir-server-cwd-stable` | có | covered-by-test |
| 919 | 2026-07-05 | Support CommandCode CLI | FEAT | C | `N/A:deferred-post-v2.0` | — | closed-by-design |
| 915 | 2026-07-04 | runPool `capped` true when jobs < concurrency | FALSE | B | `GP8/runpool-capped-flag` | có | covered-by-test |
| 914 | 2026-07-04 | index --source bare label vs indexDirectory | TOOLC | Q | `N/A:open-question` | — | wont-fix |
| 911 | 2026-07-03 | session_continuity framing trips auto-mode classifier | PI | A | `REMOVED:D9` | — | covered-by-test |
| 908 | 2026-07-02 | Configurable redirect exclusions per domain | ROUTE | A | `REMOVED:ADR-0008` | — | closed-by-design |
| 905 | 2026-07-02 | Pi bridge Bun SQLITE_IOERR | STORAGE | A | `REMOVED:D2` | — | closed-by-design |
| 903 | 2026-07-02 | SessionStart truncation splits UTF-16 surrogates | ENC; PI | B | `GP4/code-point-safe-truncation-cut` | có | covered-by-test |
| 902 | 2026-07-01 | Resume snapshot stale, priority eviction | DEL; STALE | A | `REMOVED:D8` | — | closed-by-design |
| 901 | 2026-07-01 | upgrade updates one tree, doctor reads other (Win) | INSTALL; FALSE | Q | `N/A:open-question` | — | closed-by-design |
| 895 | 2026-06-29 | ctx_search ranks stale cross-session above fresh | RETR; STALE | A | `GP5/session-worktree-project-rank` | có | wont-fix |
| 894 | 2026-06-29 | Statusline stuck, build/ missing | FALSE; STATS | A | `REMOVED:D17` | — | closed-by-design |
| 891 | 2026-06-28 | Emacs Lisp runtime | FEAT | C | `N/A:deferred-post-v2.0` | — | wont-fix |
| 890 | 2026-06-27 | omp: `with {type:"json"}` import breaks validator | INSTALL | B | `GP1/bundle-no-import-attributes` | có | covered-by-test |
| 889 | 2026-06-27 | tarball missing scripts/*.mjs | INSTALL; FALSE | A/B | `GP1/npm-pack-includes-runtime-files` | có | covered-by-test |
| 880 | 2026-06-26 | cleanupStaleContentDBs race poisons store (Pi) | RACE; DEL | A | `REMOVED:D3` | — | - |
| 878 | 2026-06-25 | chunkMarkdown oversized paragraphs | OVH | B | `GP5/chunk-byte-cap` | có | covered-by-test |
| 874 | 2026-06-24 | OMP tool_result never replaces output | FALSE; HOST | Q | `N/A:open-question` | — | wont-fix |
| 873 | 2026-06-24 | omp install does not copy SYSTEM.md | HOST | A | `REMOVED:D11` | — | closed-by-design |
| 867 | 2026-06-22 | Self-heal only at open-time, mid-session corrupt | STORAGE; DEL | Q | `N/A:open-question` | — | closed-by-design |
| 865 | 2026-06-22 | ctx_stats ignores $CLAUDE_CONFIG_DIR | STATS | A | `REMOVED:D3` | — | closed-by-design |
| 45 | 2026-03-03 | Beta testers wanted | NA | D | `N/A:not-a-bug` | — | closed-by-design |

## Phụ lục B — 178 PR theo họ chính

Cùng mã họ như Phụ lục A. PR là bản sửa đề xuất, ĐANG MỞ lúc quét, chưa merge vào `main`.

| họ (chính) | PR |
|---|---|
| INSTALL (22) | #1293 #1278 #1271 #1253 #1247 #1243 #1239 #1235 #1224 #1220 #1216 #1197 #1183 #1160 #1148 #1129 #1089 #1035 #973 #958 #929 #863 |
| HOST (22) | #1276 #1266 #1240 #1218 #1209 #1194 #1180 #1171 #1166 #1161 #1154 #1149 #1116 #1095 #1073 #1043 #995 #972 #971 #913 #910 #906 |
| OVH (15) | #1275 #1270 #1170 #1127 #1123 #1122 #1097 #1076 #991 #980 #955 #922 #921 #898 #877 |
| ROUTE (13) | #1265 #1231 #1227 #1189 #1128 #1124 #1104 #1086 #1084 #1066 #945 #927 #897 |
| ZOMB (11) | #1228 #1182 #1164 #1147 #1093 #1029 #1009 #987 #975 #968 #904 |
| PI (9) | #1272 #1264 #1137 #1034 #1002 #948 #918 #876 #864 |
| FEAT (8) | #1284 #1178 #1092 #1044 #1010 #957 #939 #935 |
| FALSE (8) | #1281 #1250 #1153 #1121 #996 #932 #931 #899 |
| STORAGE (8) | #1236 #1118 #1056 #1030 #988 #986 #970 #871 |
| SEC (7) | #1294 #1245 #1165 #1155 #1144 #1087 #1019 |
| STATS (7) | #1287 #1252 #1042 #1040 #1001 #952 #866 |
| EXEC (7) | #1286 #1237 #1177 #974 #969 #934 #924 |
| IDENT (6) | #1279 #1246 #1241 #1234 #1229 #1033 |
| RETR (5) | #1295 #1292 #1291 #1091 #963 |
| NA (5) | #1283 #1244 #930 #928 #892 |
| PERF (4) | #1290 #1277 #1211 #1111 |
| RACE (4) | #1256 #1167 #1113 #888 |
| FETCH (4) | #1238 #1172 #1138 #1041 |
| TOOLC (4) | #1230 #1207 #1082 #920 |
| ENC (3) | #1249 #1176 #907 |
| DEL (3) | #1181 #1158 #1143 |
| DOC (2) | #1285 #884 |
| CAP (1) | #1126 |

---

## Hợp đồng đầu ra (spec §63)

**SCOPE** — Bản nháp offline: dựng IssueRecord và cụm họ lỗi (overlay §3, §4, §17, §18) cho 351 issue/PR (173 + 178) đang mở mà
upstream gom trong `tickets/v2-issue-map.md`, ưu tiên 10 họ được giao. Không đọc trang/thảo luận issue (cần `gh`), không đào fork,
không chạy test.

**REVISION AUDITED** — `origin/v2-design` @ `7a99ece2` (nguồn issue, mã v2, test v2); `origin/main` @ `6006a351` (mã v1, commit sửa).

**FILES/COMMITS/ISSUES INSPECTED** — `v2:tickets/{v2-issue-map.md, v2-not-closed.md, round4-evidence.json}`,
`v2:tickets/data/{is,is2,isall,pr2,prall}.json`, `v2:docs/v2/review/ISSUE-OUTCOMES-r4.md`, `v2:docs/v2/impl/DEVIATIONS.md`; tệp test v2
(`e2e/cases/*`, `tests/v2/**`, `e2e/live/cases/*`, quét chuỗi); mã v1 `main:hooks/core/{routing,mcp-ready,tool-naming}.mjs`,
`main:hooks/hooks.json`, `main:src/session/{db,extract,snapshot}.ts`, `main:src/store.ts`, `main:src/server.ts`; mã v2 như 02b. Commit:
`3383299c`, `2731ca25`, `5235f3fd`, `f8d23289`, `752d5236`, `855e330d`, `675fec69`, `16e2bfc8`, `b4edff18`, `415ce57d`, `04569b12`,
`35e264ca`, `e56f17c1`, `f4e8d2ac`, `ff5b0cff`, `245cd8af`, `b1ba5cc9`, `85f64d2c`, `4bc292fc`, `4164d3de`, `a888cac6`, `11474111`,
`79e7bb52`. IssueRecord đầy đủ: #1055, #1037, #967, #1267, #1222, #1106, #1140, #1156, #1022, #1100, #1014, #1075 (+ #233/#724/#946/#794
ở 01).

**VERIFIED FACTS** — Số đếm ở mục 0–1 (lệnh: `parse_map.py`, `enrich.py`, `families.py` trong scratchpad, trên dữ liệu `git show`);
172/173 issue tạo sau khi `main` ngừng; giao với commit `main` = 0; 159/170 tên ca GP có trong tệp test; mâu thuẫn "đọc body" trong
nguồn upstream; lý do đóng dựa D10 đã lỗi thời; cơ chế gốc E1 của #1267, #1107, #1222, #1140, #1156, #1024, #1055/#1053, #1106; các
chuỗi bug lặp ở mục 4 có commit `main` thật.

**INFERENCES** — Phân loại họ (làm tay theo tiêu đề); ý nghĩa các lớp A/B/C/D/Q/M (suy từ bảng chéo); triệu chứng/kịch bản lấy từ tiêu
đề (E0); phiên bản bị ảnh hưởng; v2 "FIXED" nghĩa là mã trên nhánh chưa phát hành, chưa chạy.

**UNKNOWNs** — Nội dung thảo luận, người báo, nhãn, liên kết PR↔issue chính thức, ngày đóng, phản hồi của maintainer cho từng mục;
issue nào tái hiện được; số người dùng bị ảnh hưởng; hành vi runtime của mọi bản sửa v2.

**FAILURE MODES** — HOST_ADAPTER_MISMATCH, ROUTING_FAILURE, FALSE_SUCCESS, PROMPT_INTERFERENCE, ZOMBIE_OPERATION, CONTEXT_OVERHEAD,
CAPABILITY_MISMATCH, DELETION_PROPAGATION_FAILURE/mất dữ liệu âm thầm, SECURITY_BOUNDARY_CONFUSION, STARTUP_RACE; ngoài danh sách:
INSTALL/UPGRADE, PROJECT_IDENTITY/SESSION_CONTAMINATION, OBSERVABILITY (số liệu sai), STALE_CONTEXT.

**AGENT CLI RELEVANCE** — Không kết luận phơi nhiễm. Câu hỏi chuyển cho agent khác: agent-cli có cơ chế nào được "sửa" bằng cách xoá mà
không thêm ca test cho trạng thái cơ chế đó từng bảo vệ không; chỗ quyết quyền/tool của subagent và chỗ viết chỉ dẫn có cùng nguồn
không; luật chặn lệnh có đi qua parser hay chuỗi; trạng thái "đã nhắc/đã sẵn sàng/đang sống" khoá theo danh tính nào; mỗi đường dọn dữ
liệu định nghĩa "cũ/chết" thế nào và có test "sống nhưng rảnh" không.

**PROPOSED TESTS** — Gộp với 01 (T1–T8) và 02b (T9–T13). Thêm: T14 removed-mechanism guard: mỗi cơ chế bị xoá ⇒ một ca tái hiện trạng
thái nó từng bảo vệ (vd dịch vụ không lên). T15 deletion-liveness matrix: mọi đường dọn × dữ liệu {mới, cũ-đang-dùng, cũ-rảnh,
cũ-chết} ⇒ chỉ ô cũ-chết bị xoá. T16 once-means-once: mọi lời nhắc gọi hai lần cùng phiên, hai tiến trình khác nhau ⇒ một lần.
T17 cross-adapter port: mỗi bản sửa luật ở một adapter ⇒ cùng bảng đầu vào chạy trên mọi adapter.

**NO-CHANGE FINDINGS** — Hướng sửa v2 cho ROUTING (parser), ZOMBIE (EOF + deadline), CONTEXT_OVERHEAD (hằng ngân sách + lint) và
PROMPT_INTERFERENCE (bỏ tiêm) đơn giản hơn chuỗi vá v1 và có test; không thấy lý do đề xuất khác cho các họ đó. Dữ liệu offline đủ để
gom cụm; không đủ để khẳng định nguyên nhân gốc của các issue chỉ có tiêu đề.
