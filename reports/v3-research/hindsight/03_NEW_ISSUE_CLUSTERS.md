# 03 — Họ issue Hindsight: phần mới và phần audit cũ bỏ sót

Ngày: 2026-10-19. Agent nghiên cứu V3, chỉ đọc. Clone `/home/luannt/application/opensource/v3-src/hindsight`, HEAD `5331da2f4` (2026-10-16). Không chạy test, docker, uv; không có lệnh git ghi.

Ký hiệu nguồn:
- **[E0]**: trang issue/PR GitHub đọc qua WebFetch ngày 19/10, tức lời khai của người báo hoặc maintainer.
- **[E1]**: tôi tự đọc commit/diff hoặc mã HEAD; có ghi tệp:dòng.
- **[E2]**: test có tồn tại trong cây (tôi chưa chạy).
- **[ĐO]**: lệnh tôi vừa chạy trong phiên này.
- **[SUY LUẬN]**: suy luận, có ghi cách kiểm.

Đường dẫn rút gọn: `api/` = `hindsight-api-slim/hindsight_api/`, `ca/` = `hindsight-integrations/coding-agents/src/core/`. Mọi IssueRecord đều có `project = vectorize-io/hindsight`. `reproducibility` mặc định là "không tái lập (spec cấm chạy)". Ba trường `architectural_lesson`, `agent_cli_analog`, `proposed_agent_cli_regression_test` ghi ở cấp họ vì chúng giống nhau cho mọi issue trong họ; issue nào khác thì ghi riêng. Ở đây **không** kết luận agent-cli có phơi nhiễm hay không.

---

## 0. Tóm tắt

- **Ngân sách đã dùng: 35/35 trang issue/PR và 6/6 trang fork** (danh sách ở §1). Không trang nào lỗi hay đòi đăng nhập. Có 2 trang compare của fork không sinh được diff (xem tệp 04). Trang #3989 ẩn bớt comment.
- **8 họ lỗi.** Bốn họ có **bug lặp được xác nhận bằng commit + trang issue**:
  1. cursor retain của coding-agents, gồm bản lặp mới #5735 mở ngày 18/10;
  2. hook/ngữ cảnh của host lọt vào tiến trình dẫn xuất (#1751 → #5542, cộng #5555 còn mở);
  3. tag scope mặc định `all_strict` làm rỗng phạm vi (#2808 tháng 7 → #5728 ngày 18/10);
  4. worker đổi danh tính thì việc kẹt (#2165 → #2359 → #3194 → PR #3779 bị đóng không merge).
- **Lặp do bản sửa không lan sang bản sao.** Ngày 12/10 hermes đã sửa "probe phiên bản thất bại bị cache vĩnh viễn" (#5221, `10b14dd0d`, có test). Client coding-agents vẫn còn đúng lỗi đó ở HEAD (`ca/hindsight.ts:540-546`), và 6 ngày sau nó được báo thành #5735.
- **Cursor phình theo từng sự cố [ĐO].** `RetainCursor` có 4 trường ở `aaa58f247` (12/08) và 8 trường ở `d3f0001ed` (12/10). Bốn bản sửa, mỗi bản thêm đúng một trường: `pending` (#3989), `appendSupported` (#4560), `paths` (#4493), `ops` (#5251). Client vẫn **không đọc lại** trạng thái document từ server: grep `ca/chat.ts` chỉ thấy `/operations/`.
- **Audit cũ hiểu sai #3989.** Sự cố mất fact (197 → 131) có gốc **ở server**: một append bị chia slice làm cụt body document (`fda969777`). Phía client (cursor dirty) chỉ là nửa còn lại.
- **#3136 là PR, không phải issue.** Bản sửa `aaa58f247` sửa một bug **khác**, tất định: prompt hook xoá cursor ở mọi phiên. Maintainer tìm ra bug này khi đang port PR đó.
- **Worker recovery là giới hạn có chủ ý.** PR #3779 (lease + heartbeat + đếm retry, do cộng đồng viết) bị đóng ngày 31/08 kèm câu "set stable worker id via env." Ở HEAD, thu hồi việc vẫn chỉ theo `worker_id` của chính mình (`api/worker/poller.py:1454,1475`).
- **#2347 (lợi ích): có người đo, nhưng không đo lợi ích.** README eval trong repo ghi "1360 user turns, 7 searches (0.5% of turns)". Harness đó đo tỉ lệ agent **có đọc** memory, không có nhánh không-memory, không có cổng CI. Vì vậy câu hỏi lợi ích vẫn ở mức INSUFFICIENT_EVIDENCE.
- **Issue mở sau 16/10 thuộc các họ trên:** #5735, #5728, #5731, #5730, #5729, #5713, #5721. Không issue nào bàn triage "chatter" mới (đã tìm, 0 kết quả liên quan).
- **Upstream đã chạy tiếp sau HEAD.** Các issue nhắc bản 1.0.0 (commit `39c65ffe2`) và commit ngày 19/10 trên #3876 (`bc0b8c1`, `11ed416`, PR #5744). Tôi không fetch, nên mọi trạng thái mã dưới đây là **ở `5331da2f4`**.

## 1. Cách làm và ngân sách

| Loại | Trang (theo thứ tự đọc) | Số |
|---|---|---|
| Danh sách issue | `is:issue created:>2026-10-16` trang 1 (25 dòng) và trang 2 (8 dòng), tổng 33; tìm `triage OR chatter consolidation` (23 kết quả) | 3 |
| Issue/PR | PR #3136, #3989, #4560, #4493, #5735, #1751, #5542, #5030, #5371, #5417, #5728, #5713, #5721, #3876, #5731, #2359, #3194, #1671, #3002, #3820, #5434, #5582, #5221, #2985, #5054, #2165, #2808, #5729, #5730, #5555, #5589, PR #3779 | 32 |
| **Tổng issue/PR** | | **35/35** |
| Fork | xem `04_FORK_DELTA.md` | 6/6 |

Không đọc lại các trang tệp 02 đã đọc (#5251, #2413, #2347, PR #5508). Với chúng tôi dùng IssueRecord của tệp 02.

Lệnh cục bộ chính [ĐO]: `git show --stat`/`git show <c> -- <path>` cho mọi commit nêu dưới; `git log --all -S'_reclaim_expired_processing_tasks'` và `-S'WORKER_ORPHAN_LEASE'` (đều rỗng); `git cat-file -t` cho commit của fork (đều "Not a valid object name"); `grep -n` trên mã HEAD; `git shortlog -sn`.

---

## 2. Các họ lỗi

### F1. Cursor retain bất đồng bộ của tích hợp coding-agent — CURSOR_ADVANCE_BEFORE_DURABILITY, ACCEPTED ≠ DURABLE ≠ EXTRACTED

#### IssueRecord

**PR #3136** (không phải issue) — "fix(claude-code): lock the state read-modify-write where flock is unavailable (Windows)" [E0]
- date 01/08 (Calderein, có ghi là do agent "Cipher" viết). Đóng **không merge** ngày 12/08 vì bị #3412 thay thế.
- affected: Windows 11, `hindsight-integrations/claude-code/scripts/lib/state.py`.
- symptom/impact: hook chạy đồng thời làm mất ghi; bộ đếm lượt "lagged its true value by 3–35"; retain tự động không bao giờ bắn [E0].
- root_cause: không có `fcntl` thì rơi vào nhánh ghi không khoá. Comment trong mã nói "minor races here are harmless". Độ tin: cao cho Windows; test của tác giả không có hiệu lực trên Linux.
- **phát hiện phụ của maintainer:** cursor là một *trường* của session cache, và prompt hook ghi đè cả object, nên "Not a race; deterministic, every session" [E0]. Message của `aaa58f247` ghi bug này tắt append ở "seven of eleven" harness [E1].
- fixing_commit: `aaa58f247` (12/08). Cursor có tệp riêng; ghi atomic bằng temp + rename. Khoá O_EXCL **không** được port ("machinery without a failure to prevent") [E0/E1].
- regression_test: `ca/session-cache.test.ts:54` "survives the prompt hook rewriting the session cache" [E2].
- current_code_status [E1, ĐO grep]: comment "races here are harmless" vẫn còn ở **5** tệp `state.py` của các tích hợp hook cũ: claude-code, codex, copilot-cli, cursor-cli, zcode. Ví dụ `hindsight-integrations/claude-code/scripts/lib/state.py:100`.
- state: PARTIALLY_FIXED. Bug tất định đã sửa trong coding-agents; race trên Windows ở tích hợp cũ để nguyên có chủ ý.

**#3989** — "coding-agents: a failed retain leaves the cursor dirty, forcing full transcript re-extraction on every subsequent Stop" [E0]
- date 01/09 (dvinas). Đóng completed 08/09 qua PR #4210 và #4229.
- affected: plugin 0.4.3, API 0.9.2.
- symptom/impact: một lần retain lỗi biến **mọi** lần Stop sau thành replace. Theo người báo, document tăng 212.335 → 267.311 ký tự trong khi fact giảm 197 → 131, và mỗi Stop tốn khoảng 22 chunk trích [E0, số người báo tự đo].
- root_cause: **hai gốc** [E1].
  - Client: `dirty` được đặt trước khi gửi và chỉ xoá khi thành công.
  - Server: append quá cỡ bị chia slice, mỗi slice báo "tail" như thể toàn bộ document. `original_text` bị cụt, và lần append **kế tiếp** tombstone các fact không còn khớp body. Message `fda969777` ghi: "the loss is here, on the ordinary append path, silent".
  - Độ tin: cao (message giải thích đúng chữ ký số liệu của người báo).
- fixing_commit [E1]:
  - `e655d889b`: đệm append lỗi rồi phát lại dưới **operation_id cũ**; `dirty` chỉ còn dùng cho replace; có trần `PENDING_MAX_AGE_MS`/`PENDING_MAX_BYTES`.
  - `fda969777`: một định nghĩa "body của document" (`merge_json_array_parts`), cộng `assert_append_extends_stored_body` raise `AppendWouldTruncateDocument`.
- regression_test [E2]: `ca/chat.test.ts:368` (nhánh không dirty); `api/../tests/test_oversized_append_truncates_document.py`, `test_append_body_invariants.py`.
- later_regression: lớp "replace thay cho append" quay lại ở #4560 và #5735.
- state: FIXED. Lỗi server này audit cũ không ghi.

**#4560** — "coding-agents: retain is hardcoded to extraction_mode "verbose" … re-sends the full transcript every Stop" [E0]
- date 20/09 (MattJackson). Đóng completed 22/09.
- affected: plugin 0.6.1, Hindsight Cloud.
- symptom/impact: một hội thoại bị retain 14 lần trong 10 phút, mỗi lần khoảng 183k token. Người báo **ước** khoảng $150/giờ; `diag.jsonl` có khoảng 47 timeout và 6 lần retain lỗi [E0].
- root_cause: theo maintainer, nguyên nhân chính là `GET /version` gọi ở **mỗi** Stop; timeout thì Stop đó replace toàn bộ. Chế độ verbose nằm trong strategy ghi vào bank chứ không gửi theo từng request [E0]. Độ tin: trung bình–cao.
- fixing_commit: `a0e2583b0`. Cursor nhớ "yes" (`appendSupported`); "A 'no' is not remembered" (`ca/chat.ts:330`); mặc định đổi sang concise [E1].
- regression_test: `ca/chat.test.ts:295` "remembers a confirmed append capability, so a failed probe on a later Stop still appends (#4560)" [E2].
- later_regression: người báo xác nhận bản 0.7.0 đỡ hẳn: khoảng $5,30/24 giờ, 24/09 [E0]. Lớp lỗi quay lại ở **#5735** qua cache của client.
- state: FIXED cho đường hook (mỗi Stop một tiến trình mới).

**#4493** — "[coding-agents][Codex] Switching transcript_path within one session replaces earlier retained history" [E0]
- date 18/09 (feather-snow). Đóng completed 29/09 qua PR #4711.
- affected: Codex Desktop trên Windows, plugin 0.6.1, API 0.10.0.
- symptom/impact: rollout tiếp nối nằm ở tệp mới nhưng cùng session id, và nó replace document. Fact cũ biến mất (số fact 1, 2, 1) [E0].
- root_cause: cursor không ghi tệp nguồn, nên prefix khác bị đọc là "transcript bị viết lại", và luật cho trường hợp đó là replace [E1 message `1238dbdbc`]. koriyoshi2041 tái lập trên `main` bằng một test đỏ [E0]. Độ tin: cao.
- fixing_commit: `1238dbdbc` (`paths` bị chặn trần; không append lại tệp đã ghi), cộng `04f77b5` [E0/E1].
- regression_test: `ca/retain-cursor.test.ts:103,136`, `ca/retain-hook.test.ts:288` [E2].
- state: FIXED.

**#5251**: xem `02_UPSTREAM_DELTA.md` §6.1. Tôi không đọc lại trang. Trạng thái PARTIALLY_FIXED (`d3f0001ed` thêm trường `ops`). Các chỗ hở (a)–(f) ở tệp 02 vẫn đứng vì HEAD không đổi `ca/`.

**#5735** (MỚI, 18/10) — "coding-agents 0.8.1: a failed version probe disables append for the life of the client" [E0]
- affected: plugin 0.8.1 chạy trong runtime **pi**, API 1.0.0. Đang **OPEN**; không có comment của maintainer.
- symptom/impact: người báo dựng proxy trả 503 cho lần `/version` đầu, rồi gửi 4 prompt và nhận `replace, replace, replace, replace`; không có proxy thì ra `replace, append, append, append`. Một lần API khởi động lại khoảng 30 giây kéo theo 2 giờ replace toàn bộ một phiên 683k token [E0].
- root_cause [E1 xác nhận ở HEAD]:
  - `ca/hindsight.ts:540-546` `supportsIdempotentRetain()` cache **mọi** câu trả lời đầu tiên, kể cả "không liên lạc được".
  - `probeIdempotentRetain()` (`ca/hindsight.ts:552-564`) trả `false` khi gặp bất kỳ exception hay status không ok nào.
  - Docstring `ca/hindsight.ts:535-538` khẳng định "costs efficiency, never correctness".
  - Comment ở `ca/chat.ts:330` ("A 'no' is not remembered") chỉ đúng với cursor, sai với client.
  - Runtime pi giữ cursor trong RAM (`ca/runtime.ts:46` `memoryCursorStore()`), còn client sống bằng tuổi tiến trình. Với hook harness, mỗi Stop là một tiến trình, nên cache âm chỉ kéo dài một Stop [SUY LUẬN từ mã; kiểm bằng cách lặp lại proxy 503 trên harness hook].
  - Độ tin: cao.
- liên hệ: người báo dẫn #5221 (hermes). Trang không nhắc #4560/#3989/#5251 [E0].
- regression_test: không có. Test coding-agents chỉ kiểm nhánh `true` (`ca/hindsight.pages.test.ts:94,106`) [E2]. Hermes có `test_an_unreachable_server_is_asked_again_not_cached` (`hindsight-integrations/hermes/tests/test_provider.py:559`) [E2].
- state: **REGRESSED** (cùng lớp với #4560, lần này ở tầng client), OPEN.

**#5221** — "hermes plugin: a single failed /version probe disables update_mode='append' for the entire process (no negative-entry TTL)" [E0]
- date 04/10. Đóng completed 12/10 qua PR #5538.
- symptom: probe lỗi một lần là append tắt tới khi khởi động lại; phiên bị chia mảnh [E0].
- fixing_commit: `10b14dd0d`. Chỉ cache câu trả lời thật của server; "unreachable … is asked again after 60s". Review sửa thêm: "a probe with one failed path stays unknown, not 'old'". Message ghi probe đi `/v1/version` trước "like the coding-agents client (#5286)" [E1].
- regression_test: `hermes/tests/test_provider.py:559,573,581,590` [E2].
- state: FIXED trong hermes. Bản sao trong coding-agents **không** được sửa (→ #5735).

#### Phân tích họ F1

- **Vì sao mắc.** Client phải đoán document trên server đang chứa gì, nhưng chỉ có tín hiệu cục bộ. Tín hiệu đó là `turns`, `fingerprint`, `bank`, `paths`, `appendSupported`, `dirty`, `pending`, `ops` (`ca/retain-cursor.ts:89-126`) [ĐO]. Chưa từng có lần đọc lại "document có N lượt / phiên bản V" từ server: grep `ca/chat.ts` chỉ ra `/operations/` ở dòng 150 [ĐO]. Đề xuất ở #3989 "Record the attempted turn count and verify it against the server-side document" [E0] không được làm.
- **Sửa thế nào.** Mỗi sự cố một trường mới. Số trường `RetainCursor` theo commit [ĐO `git show <c>:…retain-cursor.ts`]:

| Commit | Ngày | Trường | Thêm do |
|---|---|---|---|
| `aaa58f247` | 08-12 | turns, fingerprint, bank, dirty | — |
| `e655d889b` | 09-08 | + pending | #3989 |
| `a0e2583b0` | 09-22 | + appendSupported | #4560 |
| `1238dbdbc` | 09-29 | + paths | #4493 |
| `d3f0001ed` | 10-12 | + ops | #5251 |

- **Cái nào sống:**
  - Tách tệp cursor (`aaa58f247`): mã HEAD còn nguyên.
  - Assert append đơn điệu ở server (`fda969777`): một lớp chặn chạy lúc runtime.
  - Phát lại dưới operation_id cũ: dựa vào tính idempotent ở server.
- **Cái nào hồi quy:**
  - "probe lỗi → replace": #4560 sửa ở cursor; #5221 sửa ở hermes; #5735 là cùng lớp ở tầng client của coding-agents.
  - "fact mất mà không báo": #3989 (server cụt body), rồi #5251 (accept rồi fail).
- **Phân loại bản sửa:**

| Commit | Phân loại | Ghi chú |
|---|---|---|
| `aaa58f247` | ARCHITECTURAL_FIX ở cỡ nhỏ | message: "makes the invariant structural rather than a convention" |
| `fda969777` | SHARED_ABSTRACTION + assert | một định nghĩa body + chặn lúc runtime |
| `e655d889b` | đổi giao thức phục hồi | replace → replay |
| `a0e2583b0`, `1238dbdbc`, `d3f0001ed`, `10b14dd0d` | LOCAL_PATCH | |

- **Vì sao lớp bug cứ quay lại** (overlay §17):
  1. Không có chủ sở hữu duy nhất cho câu "server đang có gì". Chính client tự suy ra, và mỗi suy luận sai lại cần thêm một trường.
  2. Cùng một luật (probe phiên bản, cache âm) có bản sao ở hai tích hợp. Mỗi bản có test riêng, hoặc không có test.
  3. Cùng một mã nhưng vòng đời khác nhau theo host (tiến trình mỗi Stop so với pi chạy lâu), nên một cache vô hại ở host này lại là lỗi ở host kia.
  4. Lỗi chỉ đi vào tệp diag (fail-open; xem `retain-hook.ts:166-176` ở tệp 02). Người dùng chỉ phát hiện qua **hoá đơn** (#4560) hoặc **tự đếm** (#3989, #5251, #5735).
- **Bất biến tổng quát:**
  - Một con trỏ tiến độ chỉ tiến theo trạng thái mà **bên giữ dữ liệu xác nhận**.
  - "Không liên lạc được" ≠ "không"; câu trả lời âm do lỗi vận chuyển không được cache.
  - Mọi luật suy thoái hoặc probe phải có **một** cài đặt, hoặc có test ở **mọi** bản sao.
  - Tuổi thọ cache phải được nêu rõ theo host.
- **agent_cli_analog** (giả thuyết, không kết luận): mọi chỗ agent-cli lưu "đã gửi / đã nhận" để quyết định lần sau làm gì. Ví dụ: hàng prompt `send`, trạng thái resume, kết quả probe provider hay model được cache trong tiến trình server sống lâu.
- **proposed tests:**
  - **T-F1a**: probe khả năng của một backend lỗi tạm 1 lần, rồi gọi N lần trong **cùng** tiến trình sống lâu. Kỳ vọng: lần sau probe lại; không giữ "không" vĩnh viễn.
  - **T-F1b**: chạy cùng kịch bản trên hai vòng đời: tiến trình mỗi lượt và tiến trình chạy lâu.
  - **T-F1c**: đếm số bản cài đặt của cùng một luật probe (grep), và mỗi bản phải có test âm.

### F2. Hook/ngữ cảnh của host lọt vào tiến trình dẫn xuất — HOST_ADAPTER_MISMATCH, SESSION_CONTAMINATION

#### IssueRecord

**#1751** — "claude-code provider spawns subprocesses that recursively retain to the same bank" [E0]
- date 26/05 (bchumerch). Đóng completed 28/05 qua PR #1825.
- affected: hindsight-all 0.6.2, claude-agent-sdk 0.1.72, Windows 11.
- symptom/impact: khoảng 5M token/ngày; 4.614 document "cons-loop" (khoảng 144M ký tự); ngày 24/05 có 556/559 document mở đầu bằng prompt nội bộ [E0].
- root_cause: tiến trình con `claude` nạp plugin của host, và Stop hook retain transcript của chính nó vào bank. Độ tin: cao.
- fixing_commit: `bcae23d9f`, đặt `CLAUDE_CONFIG_DIR` vào thư mục tạm [E1]. Người báo đề xuất `HINDSIGHT_SKIP_RETAIN`, nhưng bản sửa **không** làm cách đó [E1 diff].
- regression_test: `api/../tests/test_claude_code_llm_isolation.py` [E2].
- later_regression: **#5542**.
- state: REGRESSED rồi FIXED lại.

**#5542** — "claude-code provider: CLAUDE_CONFIG_DIR isolation (#1751) is bypassed when the daemon's cwd is $HOME, causing a hook feedback loop" [E0]
- date 12/10 (Takkmee). Đóng completed 16/10 qua PR #5691.
- affected: coding-agents 0.8.1, API 0.10.3 chạy daemon qua hindsight-embed, macOS.
- symptom/impact: 100 lần `deepen` cho bank `$HOME` so với 2 lần cho repo thật; khoảng 30 tiến trình `claude` cùng lúc; reflect 503 "at capacity (160 concurrent)"; SIGTERM không dừng được [E0].
- root_cause: cwd = `$HOME`, nên `~/.claude/settings.json` được nạp như project settings. Người báo tự nhận ra đây là cùng vòng lặp với #1751 [E0]. Độ tin: cao.
- fixing_commit: `c7b17f0ad`, đặt `cwd` vào thư mục cô lập và `setting_sources=[]` (`api/engine/providers/claude_code_llm.py:419-426, 754-755`) [E1].
- regression_test: `test_claude_code_llm_isolation.py::_assert_isolation`. Test chỉ kiểm tham số được truyền, không kiểm CLI thật (tệp 02 §3.2) [E2].
- state: FIXED.

**#5555** (mở 13/10) — "[hermes integration] auto_retain writes kanban-worker (and other non-primary) sessions into the bank" [E0]
- affected: plugin hermes 1.2.2/1.2.3. Đang OPEN, không có comment của maintainer.
- symptom: worker kanban chạy `hermes -p … --cli` được core gán `primary`, nên mọi lượt bị retain. halr9000 dựng ma trận 24 kiểm tra: 10 đỏ, trong đó 6 đỏ là retain ở ngữ cảnh không-primary (`subagent`, `flush`) [E0].
- root_cause: plugin chỉ đặc trị `cron` (`_apply_retain_policy`); ngữ cảnh được **suy** từ `platform`; biến `HERMES_KANBAN_TASK` có đặt nhưng không ai đọc [E0].
- liên hệ: `10b14dd0d` (12/10) vừa thêm "cron runs … are no longer auto-retained" (#4945) [E1 message], tức lại một bản vá cho một ngữ cảnh.
- state: OPEN.

**#5729** (MỚI, 18/10) — "coding-agents: legacy-plugin warning ignores enabledPlugins set in project or local settings" [E0]
- symptom: plugin đã tắt ở project settings vẫn bị báo "recalled and retained twice" [E0].
- root_cause [E1 xác nhận]: `ca/legacy.ts:49` chỉ đọc `settings.json` ở thư mục user. Đây là cùng kiểu hiểu sai với #5542: tầng settings của host gồm user, project, local, và plugin tự dựng lại thứ tự ưu tiên chỉ một phần.
- impact: chỉ là cảnh báo sai, không ảnh hưởng dữ liệu [E0].
- state: OPEN.

#### Phân tích họ F2

- **Vì sao mắc.** "Tiến trình này có phải phiên người dùng chính không" được **suy** từ cấu hình hay môi trường của host (thư mục config, cwd, `platform`), thay vì được **đánh dấu** lúc sinh tiến trình.
- **Các cơ chế chống đệ quy ở HEAD** [E1, ĐO grep]. Có bốn cơ chế độc lập, không cơ chế nào đọc tín hiệu của cơ chế kia:

| # | Cơ chế | Vị trí |
|---|---|---|
| 1 | `CLAUDE_CONFIG_DIR` + cwd + `setting_sources=[]` | `claude_code_llm.py` |
| 2 | workspace rỗng + `CURSOR_CONFIG_DIR` | `cursor_llm.py:72-85, 275-281` |
| 3 | `HINDSIGHT_DISABLE_HOOKS` (chỉ survey của chính plugin đặt) | `ca/survey.ts:215`; được đọc ở `ca/retain-hook.ts:186`, `ca/hook.ts:433`, `ca/session-start.ts:398`, `ca/runtime.ts:123,169,248,270` |
| 4 | `agent_context == "cron"` | hermes |

  Env do server truyền cho CLI con chỉ có hai khoá `CLAUDE_CONFIG_DIR` và `CLAUDE_SECURESTORAGE_CONFIG_DIR` (`claude_code_llm.py:103-106`), tức **không** có `HINDSIGHT_DISABLE_HOOKS`, dù hook của chính Hindsight đã biết đọc biến này. [SUY LUẬN: chỉ thêm biến đó đã chặn được vòng #5542 từ phía hook; cách kiểm là chạy daemon từ `$HOME` với bản `bcae23d9f` cộng biến này.]
- **Cái nào sống:** cách ly bằng tham số (#1751, #5542).
- **Cái nào hồi quy:** #1751 → #5542, sau khoảng 4,5 tháng. Phía hermes: cron (#4945) đã sửa thì kanban/subagent/flush (#5555) lại lộ ra.
- **Phân loại:** cả `bcae23d9f` lẫn `c7b17f0ad` đều là LOCAL_PATCH. Chưa có ARCHITECTURAL_FIX, tức chưa có một dấu "tiến trình dẫn xuất" được truyền xuống mọi tiến trình con và được mọi hook kiểm.
- **Vì sao quay lại:** mỗi bản sửa bịt **một kênh** host dùng để nạp cấu hình (thư mục user, rồi project settings, rồi platform), trong khi host còn những kênh khác (local settings, managed, `--settings`; #5729 liệt kê). Danh sách kênh do host quyết và đổi theo phiên bản host.
- **Bất biến tổng quát:** tiến trình phụ do hệ thống tự sinh phải mang **dấu dương tường minh** ("tôi là tiến trình dẫn xuất, đừng ghi"). Mọi điểm ghi bền phải kiểm dấu đó. Không được suy ngữ cảnh từ cách host xếp lớp cấu hình.
- **agent_cli_analog** (giả thuyết): tiến trình con mà agent-cli sinh, như CLI `claude`/`codex` qua tool giao việc, hoặc bash. Chúng có thể nạp hook hay settings của host rồi ghi ngược vào session/memory. CLAUDE.md của agent-cli có ghi phải bỏ `CLAUDECODE`/`CLAUDE_CODE_*` cho test giao việc; tôi không kiểm.
- **proposed tests:** T4 ở tệp 02 (hook đánh dấu ở `$HOME/.claude/settings.json`, kiểm bằng CLI thật), cộng **T-F2b**: sinh tiến trình con với `cwd` ở mỗi tầng settings (user, project, local) và kiểm hook đánh dấu không bắn ở cả ba.

### F3. Phạm vi tag: nới lặng hoặc rỗng lặng — SCOPE_FAILURE, WRONG_SCOPE_RETRIEVAL

#### IssueRecord

**#5030** — "Recall chunks and reflect expand ignore the tag filter on source documents" [E0]
- date 01/10. Maintainer tự mở và đóng cùng ngày qua PR #5036.
- symptom: người đọc bị lọc `user:dan` vẫn nhận văn bản riêng của Kate qua `include.chunks` và `expand` [E0].
- root_cause: văn bản nguồn được kiểm theo tag của **fact**, không theo tag của **document**. Độ tin: cao.
- fixing_commit: `c71cd0437` [E1 tsv].
- regression_test: `api/../tests/test_source_text_tag_scope.py` (`test_visible_document_ids_fails_closed_on_a_missing_document`, dòng 109) [E2].
- current: nhánh mental-model refresh ghi "Not reproduced yet" [E0]; tôi chưa kiểm.
- state: FIXED.

**#5371**: tệp 02 §3.3 đã có diff. Bổ sung từ trang [E0]:
- người báo: JayOfTheKeyboard, 07/10;
- koriyoshi2041 tìm thêm lỗ ở `GET /documents`; người báo vá cả `/graph` trên nhánh fork (`b32c4a1`, `5d3e721`);
- upstream sửa bằng commit riêng `89594d22d`, không có trailer co-author [ĐO];
- **còn mở trong issue:** `DEFAULT_MCP_RECALL_DESCRIPTION` cố định, nên sửa docstring không tới được client MCP.

State FIXED, còn một parser inline (tệp 02).

**#5417**: tệp 02 §3.4 đã có diff. Bổ sung [E0]:
- người báo rodrigoedu94-png đưa commit fork `f70c1cfde`; upstream viết lại thành `3f8fe203f` (không co-author);
- issue liệt kê đúng 3 cặp bản sao (hàm "active", parser `tag_groups`, validate của proxy);
- trang **không** dẫn #5030/#5371, tức người báo không nối họ lỗi.

State FIXED, còn `KnowledgeTagFilter.active` inline.

**#2808** — "Mental model refresh returns empty content when model has multiple tags (all_strict default filters out all memories)" [E0]
- date 19/07 (Robs87). Đóng completed 21/07 qua PR #2858. #2670 bị đánh duplicate vào đây.
- symptom: refresh ra "No records found…" trong khi reflect cùng câu hỏi ra 18 hit [E0].
- root_cause: thiếu `tags_match` thì mặc định `all_strict`. Contributor ebarkhordar chỉ ra `compute_mental_model_is_stale` lại mặc định `any`, tức hai luật lệch nhau [E0].
- fixing_commit: `41d71a981`, chỉ **làm cho cấu hình được** `tags_match` ở MCP, TS client, CLI. **Mặc định không đổi** [E1 `--stat`: 14 tệp, không tệp nào thuộc `memory_engine.py`].
- current [E1]: `api/engine/memory_engine.py:1866` vẫn `"all_strict" if model_tags else "any"`; docstring 1856 ghi "(security isolation)".
- later_regression: **#5728** (F4).
- state: PARTIALLY_FIXED (mặc định vẫn còn, có chủ ý).

#### Phân tích họ F3

- **Vì sao mắc.** Có hai chiều sai đối xứng:
  - giá trị lạ thì **nới** (#5371);
  - mặc định chặt thì **rỗng**, mà rỗng lại không phân biệt được với "không có dữ liệu" (#2808, #5728).

  Cả hai cùng gốc: luật phạm vi có nhiều bản sao (#5417, #2808 refresh vs staleness), và kết quả rỗng do phạm vi không được báo khác kết quả rỗng thật.
- **Cái nào sống:** từ chối giá trị lạ (`89594d22d`); một decoder (`3f8fe203f`); seam store raise `TypeError` khi bị yêu cầu lọc mà không lọc được (tệp 02).
- **Cái nào hồi quy:** triệu chứng "refresh ra 'no records' vì `all_strict`" quay lại 3 tháng sau, ở #5728.
- **Phân loại:** `89594d22d` và `3f8fe203f` là SHARED_ABSTRACTION; `41d71a981` là LOCAL_PATCH (bề mặt cấu hình); `c71cd0437` là LOCAL_PATCH với luật "fail closed".
- **Vì sao quay lại:** mặc định `all_strict` được giữ vì an ninh. Lựa chọn đó đúng, nhưng hệ quả "phạm vi rỗng" không có kênh báo riêng, nên mỗi tầng dùng phạm vi (refresh, reflect, staleness) tự gặp lại nó.
- **Bất biến tổng quát:**
  - Giá trị phạm vi lạ thì từ chối.
  - Phạm vi giải ra **0 nguồn** là một trạng thái có tên (`SCOPE_EMPTY`), khác "nguồn có nhưng không khớp" và khác "đã đọc, không có gì mới".
  - Mặc định chặt phải đi kèm báo cáo đã chặt bao nhiêu.
- **agent_cli_analog** (giả thuyết): bộ lọc phiên, dự án, worktree hay provider khi chọn context; "không thấy gì" vì lọc sai bị đọc thành "không có gì".
- **proposed tests:** T6 (tệp 02), cộng **T-F3b**: bộ lọc hợp lệ nhưng khớp 0 nguồn → kết quả phải mang cờ `scope_empty`, không được ra câu trả lời "không có dữ liệu".

### F4. Trạng thái dẫn xuất ghi đè bằng "không tìm thấy" — FALSE_SUCCESS, DERIVED_STATE_STALENESS, CURSOR_ADVANCE

#### IssueRecord

**#5728** (MỚI, 18/10) — "Mental-model refresh: cursor and tag scope can zero retrieval, and the `no records` report overwrites a good model" [E0]
- affected: 1.0.0, Windows 11, PG 18.1; 7.667 memory; DeepSeek. Đang OPEN, không có comment của maintainer.
- symptom/impact:
  - Mọi tool truy xuất đều trả 0, nhưng refresh vẫn ghi một báo cáo "nothing found" dài khoảng 2.078 ký tự đè lên nội dung tốt. Kết quả lại ghi `content_unchanged`.
  - Ở chế độ delta, lần sau xây tiếp trên nội dung hỏng đó.
  - Lùi tay `last_memory_seen_at` thì refresh ra lại được tài liệu 6.642 ký tự [E0].
- root_cause (ba nhánh):
  1. **Cursor dữ liệu:** người báo nói `last_memory_seen_at` theo `max(updated_at)` của bank, không theo document mới nhất đã đọc, nên cửa sổ lần sau rỗng [E0, chưa kiểm ở mã]. `memory_engine.py:1839` ghi watermark được persist là `newest_in_scope` [E1], nên nhánh này còn tranh cãi; tôi để là UNKNOWN.
  2. **Tag:** mặc định `all_strict` (= F3, `memory_engine.py:1866`) [E1].
  3. **Cổng "có nguồn":** koriyoshi2041 tái lập trên `5331da2f4`. Một mental model anh em đọc được sẽ đặt `has_sources=True` dù phạm vi memory rỗng [E0]. Xác nhận ở `api/engine/memory_engine.py:17798-17801` [E1].
- **xung đột thiết kế [E2]:** HEAD có test **ghim đúng hành vi đó**: `tests/test_mental_models.py:3576` `test_sibling_document_counts_as_a_source_unless_excluded` ("a sibling document with real content is readable, so this refresh had something to reflect over"), nằm trong `TestRefreshSkipsEmptyScope` (dòng 3343, dẫn #3875). Vậy #5728 không phải "quên". Đây là hai mục tiêu va nhau: "đừng chạy reflect trên phạm vi rỗng" (#3875) và "trang anh em là nguồn hợp lệ".
- liên hệ: #4860 (cổng ghi chặn rỗng hoặc "OK", không chặn "nothing found" dài) [E0]; #4567 (`e00154496`, chính revision audit cũ): reflect thêm cờ `answer_found` bắt buộc, vì "Prompt rules (#5250…) … did not move it" [E1].
- state: OPEN. Phân loại: DESIGN_LIMITATION + FALSE_SUCCESS.

**#3820** — "parse_delta_operation_list rejects a top-level array of ops … deterministic wedge on thinking-disabled models" [E0]
- date 26/08 (superafun). Đóng completed 31/08 qua PR #3829.
- symptom: ở nhiệt độ 0, lần nào cũng `delta_ops_failed`, thành vòng `MentalModelRefreshError` vĩnh viễn; nội dung đứng yên [E0].
- root_cause: hai kiểu hỏng gộp chung một thông báo lỗi; bản #3424 chỉ sửa kiểu thứ nhất [E0].
- fixing_commit: `7fe6a5e79` (31/08, tác giả Ehsan), rồi `9e9384107` (16/10, sửa lần hai cho JSON đã parse; tệp 02 §3.9) [E1].
- regression_test: `tests/test_delta_operation_parse.py` [E2].
- later_regression: chính `9e9384107` (cùng luật, một biểu diễn khác).
- state: FIXED (hai lần).

#### Phân tích họ F4

- **Vì sao mắc.** Trạng thái dẫn xuất (trang mental model) được thay bằng **mọi** câu trả lời không rỗng. Lời "tôi không tìm thấy gì" là một câu trả lời hợp lệ về cú pháp, nên nó lọt cổng ghi.
- **Sửa thế nào:**
  - từng cổng một: rỗng hoặc "OK" (#4860);
  - với reflect, buộc recall trước khi kết luận "không thấy" (#4567);
  - bỏ qua refresh khi phạm vi rỗng (#3875).

  Mỗi bản sửa nhắm một **dạng** của lời "không có gì".
- **Vì sao quay lại:** "câu trả lời không có bằng chứng" không phải một trạng thái máy kiểm được. Cờ `answer_found` chỉ có ở `done` của reflect, refresh không đọc nó (nếu có thì #5728 đã không xảy ra) [SUY LUẬN; kiểm bằng cách grep `answer_found` trong `_execute_mental_model_refresh`].
- **Phân loại:** `e00154496` là SHARED ở cấp vòng reflect; cổng #4860 và #3875 là LOCAL_PATCH.
- **Bất biến tổng quát:**
  - Trạng thái dẫn xuất chỉ được **thay** khi bản mới có bằng chứng (≥1 nguồn trong phạm vi được yêu cầu).
  - "Không tìm thấy" giữ bản cũ và báo FAILED_RETRIEVAL.
  - Watermark chỉ tiến tới mốc đã thật sự đọc.
- **agent_cli_analog** (giả thuyết): tóm tắt hoặc compact phiên, hay digest, được tạo lại từ một nguồn rỗng hoặc lọc sai, rồi đè lên bản tốt.
- **proposed test T-F4:** tái tạo một bản dẫn xuất khi nguồn giải ra rỗng. Kỳ vọng: bản cũ giữ nguyên, có trạng thái lỗi, watermark không tiến.

### F5. Dấu "đã consolidate" ≠ "đã được biểu diễn" — FALSE_SUCCESS, OBSERVABILITY, PARTIAL_SUCCESS

#### IssueRecord

**#3876** — "Consolidation silently wipes bank: schema-invalid LLM output deletes source observations, merged units never created (v0.9.2)" [E0]
- date 29/08 (IliaKondratiev). Đóng completed 31/08 qua PR #3923.
- symptom/impact: hai lần bank bị xoá sạch: 1.646 memory unit còn 0, rồi còn 4. Operation vẫn `completed`, `failed_operations: 0`, `audit_log` = 0 [E0]. zunami gặp cùng lỗi `observation_id` với model 3B, khoảng 3/54 lần chạy [E0].
- root_cause: delete, update, create và stamp không cùng transaction [E1 message `ca38687cd`]. Độ tin: cao.
- fixing_commit: `ca38687cd`, gom mọi ghi của một response vào một transaction; tách prepare (không giữ kết nối) và apply [E1].
- regression_test: `tests/test_consolidation_batch_atomicity.py` [E2].
- later_regression: hình dạng ngược lại ở #5054 (cả batch ghi xong nhưng không biểu diễn fact nào). Trang ghi thêm commit ngày 19/10 (`bc0b8c1`, `11ed416`, PR #5744; refactor/perf, sau HEAD, chưa kiểm).
- state: FIXED.

**#5054** — "Consolidation stamps skipped source facts as consolidated, silently excluding them from rebuild" [E0]
- date 01/10 (aprj-dev). Đóng **completed** 05/10; maintainer chỉ viết "this is by design."
- symptom: 3.177/9.309 fact (48%) bị đóng dấu mà không observation nào trỏ tới. Người báo sau đó tự quy tỉ lệ skip 58% cho một mission tuỳ biến, sửa mission thì còn 21% [E0].
- root_cause: `mark_consolidated_ids` là cả batch đầu vào [E0]. Ở HEAD vẫn vậy: `api/engine/consolidation/consolidator.py:2752` [E1].
- state: DESIGN_LIMITATION. Đóng "completed" mà không có commit: closed ≠ fixed.

**Triage "chatter"** (`0d8e58b3e`, 16/10; không có issue):
- Fact "chatter" trả `skipped/triage_chatter` (`consolidator.py:2894`) và đi qua cùng khối stamp (`:2752`) [E1].
- Định nghĩa chatter có "a momentary activity, what is on a screen right now" (`api/engine/consolidation/triage.py:57-58`).
- Dưới một mission riêng của bank thì "nothing is dropped" (`triage.py:289-290`). Bank do coding-agents dựng có `observations_mission` (`ca/missions.ts:629`), nên triage **không** bỏ fact ở đó, với điều kiện mission vẫn còn [SUY LUẬN; kiểm bằng `GET /banks/{id}/config` trên bank coding].
- Comment ở `triage.py` thừa nhận lỗi phân loại đã đo được: "probed, both jev-latest and hindsight-decision dropped facts the mission wanted" [E1, số không kèm].
- Tìm `triage OR chatter consolidation`: 23 kết quả, **0** issue về triage mới [E0].

**#5731** (MỚI, 18/10) — "Explicit source observation clear does not reset consolidated sources when deleted_count is zero" [E0]
- affected: 1.0.0, commit `39c65ffe2`. OPEN.
- koriyoshi2041 tái lập trên `main` và có bản sửa ở fork `1d6058d`, nhưng không mở được PR vì "credential was blocked" [E0].
- root_cause [E1 xác nhận]: `clear_observations_for_memory` (`api/engine/memory_engine.py:12014`) chỉ requeue khi `deleted_count > 0` (dòng 12060, 12067). Route này là **lối reset duy nhất** có tài liệu cho một fact đã bị đóng dấu mà không được biểu diễn (#5054), nên chặn ở đây làm #5054 không có lối thoát.
- state: OPEN.

**#5730** (MỚI, 18/10) — "GET …/memories/{id} omits consolidated_at and consolidation_failed_at" [E0]
- root_cause [E1 xác nhận]: `select_cols` của `get_memory_unit` (`api/engine/memories/pg/curation.py:332-336`) thiếu hai cột mà đường list có (`curation.py:238, 295-297`). Hai reader cho cùng một thực thể, khác trường.
- state: OPEN.

**#5434** (mở 08/10) — "Consolidation op whose retry is skipped because another consolidation is pending stays failed forever (no superseded state)" [E0]
- symptom: op bị bỏ retry thì thành `failed` với `retry_count=0`, nên monitor `failed_operations` báo động giả. Người báo tự viết job đêm để retry [E0].
- root_cause [E1 xác nhận]: nhánh chống retry storm ở `api/engine/memory_engine.py:4838-4853` re-raise, và poller đánh `failed`. Không có trạng thái `superseded`.
- state: OPEN. Một bản vá chống RETRY_STORM đã tạo ra **thất bại giả** (OBSERVABILITY).

#### Phân tích họ F5

- **Vì sao mắc.** `consolidated_at` gánh hai nghĩa:
  - "đã được xét" (để loại khỏi hàng chờ);
  - "đã được biểu diễn" (người đọc hiểu thế).

  `failed` cũng gánh hai nghĩa: "hỏng" và "bị thay thế" (#5434).
- **Cái nào sống:** transaction một-response (`ca38687cd`) giữ chắc kiểu hỏng "xoá nửa vời".
- **Cái nào hồi quy:** không có hồi quy nguyên dạng. Nhưng mỗi lần thêm lý do skip (`no_durable_knowledge`, `triage_chatter`) là thêm một đường "đóng dấu mà không biểu diễn". Lối reset thì bị chặn (#5731), và trạng thái không đọc được ở endpoint chi tiết (#5730).
- **Phân loại:** `ca38687cd` là ARCHITECTURAL_FIX (ranh giới transaction). #5054 không sửa. #5434, #5730, #5731 chưa sửa.
- **Vì sao quay lại:** một trạng thái nhị phân (`consolidated_at` NULL hay không) được dùng cho nhiều câu hỏi, nên mỗi câu hỏi mới lại cần một ngoại lệ.
- **Bất biến tổng quát:**
  - Dấu xử lý phải **mang lý do** (`represented | skipped:<reason> | superseded_by:<id>`).
  - Mọi reader phải đọc cùng một tập trường.
  - Thao tác "reset" do người vận hành yêu cầu phải luôn reset, không phụ thuộc có gì bị xoá hay không.
- **agent_cli_analog** (giả thuyết): trạng thái task/run/job đơn trị (`done`, `failed`), trong khi lý do thật là "bị thay thế", "bỏ qua có chủ ý" hay "hỏng".
- **proposed test T-F5:** một job bị huỷ vì job khác thay thế. Kỳ vọng: trạng thái cuối khác `failed`, monitor lỗi không đếm nó; endpoint chi tiết và endpoint danh sách trả cùng trạng thái.

### F6. Worker chết hoặc đổi danh tính, hàng đợi kẹt mà health xanh — WORKER_FAILURE, ZOMBIE_OPERATION, OBSERVABILITY

#### IssueRecord

**#2165** — "Stuck operations with dead worker reservations after OOM restart — no Cancel or auto-recovery" [E0]
- date 12/06 (scwall, 0.8.1). **Chính người báo đóng** cùng ngày, không có bản sửa.
- symptom: `processing` 35 giờ trở lên; 37k mục chờ consolidation; API cancel trả 404; chỉ gỡ được bằng `DELETE` SQL [E0].
- root_cause: worker id mặc định = container id, đổi sau mỗi lần restart. Maintainer hỏi: "did you set the worker_id?" [E0].
- state: DESIGN_LIMITATION.

**#2359** — "Consolidation Task Claiming Bug in v0.8.3 - Worker Never Claims Pending Tasks" [E0]
- date 22/06. Đóng **not planned** 25/06.
- symptom: tiến độ 34/22.510 sau 24 giờ trở lên; op `processing` không cancel được; `POST /consolidate` dedupe vào chính op kẹt [E0].
- maintainer: "did you set HINDSIGHT_API_WORKER_ID?" [E0].
- commit: `680305aea` (23/06), chỉ thêm **cảnh báo** khi chạy trong container mà không đặt `WORKER_ID` [E1].
- regression_test: `tests/test_container_detection.py` [E2].
- state: DESIGN_LIMITATION.

**#3194** — "Consolidation task still never claimed on v0.8.6 — observations stall forever" [E0]
- date 06/08 (GSL-R). Đóng **not planned** 10/08.
- Người báo **đã** đặt `HINDSIGHT_API_WORKER_ID=hindsight-prod`, nhưng vẫn kẹt sau op mồ côi của **worker id cũ** (`c8f8045891d1:1`, `33ee967ffda4:5`) [E0].
- Người báo tự đánh `failed` 6 op, sau đó pending giảm 299 → 159 và observation tăng 20 → 69 [E0].
- maintainer: không có lời giải thích; chỉ một 👍 [E0].
- state: DESIGN_LIMITATION. Đặt id ổn định **không** cứu được việc còn treo dưới id cũ.

**PR #3779** — "fix(worker): reclaim expired 'processing' leases from departed workers" [E0]
- tác giả sanastasiou, khoảng 24/08. Đóng **không merge** 31/08.
- đề xuất: quét lúc khởi động mọi dòng `processing` của **bất kỳ** owner có `claimed_at` và `updated_at` đều cũ hơn `HINDSIGHT_API_WORKER_ORPHAN_LEASE_SECONDS` (mặc định 86400; 0 = tắt), có đếm `retry_count` và cộng dồn lên cha.
- review:
  - strix-security[bot] báo race CWE-362 (đặt lại một consolidation còn đang chạy), dẫn tới bản sửa dùng heartbeat;
  - Sanderhoff-alt chỉ ra lỗi Oracle và thiếu trần retry.
- maintainer: "set stable worker id via env." [E0].
- current [ĐO]: `git log --all -S'_reclaim_expired_processing_tasks'` và `-S'WORKER_ORPHAN_LEASE'` đều rỗng. HEAD vẫn chỉ thu hồi `worker_id = $1` (`api/worker/poller.py:1454,1475`).
- state: REJECTED, tức DESIGN_LIMITATION có chủ ý.

**#1671** — "Async operations need poison quarantine / degraded queue health" [E0]
- date 20/05. Đóng **not planned** 27/05, không có lý do.
- symptom: `/health` trả `healthy` trong khi lane async retry việc độc (vector dim 0, index hỏng, `task_payload` null) [E0].
- maintainer: "the broader poison-quarantine + health story … is its own follow-up" [E0].
- current [E1]:
  - Có thêm các lớp nhỏ: đếm retry khi phục hồi (`234f5a062`, tác giả handnewb), trần thời gian cho retain/consolidation/refresh (`api/worker/poller.py:98-111`), watchdog `[STUCK?]` chỉ ghi log (`poller.py:2271`).
  - **Không** có trạng thái quarantine và **không** có health suy giảm: `api/worker/main.py:93-112` (`/health` = kết nối DB); `/health/live` có `seconds_since_last_poll` nhưng không bao giờ đổi mã trạng thái (`main.py:115-135`).
- state: DESIGN_LIMITATION.

**#2985** — "Hard kill during batch retain leaves parent ops pending with task_payload=NULL — silent document loss…" [E0]
- date 27/07 (preyevates, 0.8.5). Đóng completed 28/07 qua PR #2986.
- symptom: 8/77 document không bao giờ được tạo; `failed_operations` = 0; retry trả 409; người báo chỉ phát hiện nhờ tự đếm [E0].
- root_cause: chính người báo xếp nó cùng lớp với #1671 và #2413 [E0].
- fixing_commit: `4708a3661` (`_reconcile_orphaned_parents`, `poller.py:1676`), cộng `ac4df7eb8` [E1].
- regression_test: `tests/test_worker.py`, `tests/test_operation_status.py` [E2].
- **fork:** `sanastasiou` `ddc3f2d` tham chiếu issue này (tệp 04).
- state: FIXED cho đường cha/con. Hàm reconcile chỉ chạy trong `recover_own_tasks`, tức chỉ với worker id của chính mình [E1, tệp 02].

**#3002** — "Retain workers wedge until restart after a bulk_insert_entities deadlock" [E0]
- date 28/07 (fyko, 0.8.5, EKS).
- symptom: 22 slot bị giữ; 399 retain chết đói 50 phút; một op 2.405 s; API không retry hay cancel được op `processing` [E0].
- fixing_commits (4, theo tsv `LAP=4`) [E1]:
  - `8133c5ab7`: `RETAIN_WALL_TIMEOUT`; huỷ cả producer lẫn consumer; `ORDER BY LOWER(name)`; DB acquire timeout có hiệu lực;
  - `f6b3ce3e3`;
  - `698620786` (tác giả Sanderhoff-alt): trần **idle** cho consolidation;
  - `3531e82f9`.
- regression_test: `tests/test_worker_wall_timeout.py`, `tests/test_llm_stage_breadcrumbs.py` [E2].
- current [E1]: chỉ retain, consolidation, `refresh_mental_model` có trần (`poller.py:98-111`); "Unmapped task types remain unbounded" (`poller.py:163-164`).
- state: FIXED (đã có trần), PARTIAL cho các loại task khác.

**#2413**: xem tệp 02 §6.2 (DESIGN_LIMITATION; test bị formatter vô hiệu).

#### Phân tích họ F6

- **Vì sao mắc.** Quyền sở hữu một việc được gắn với `worker_id`, mà id này mặc định là hostname. Không có lease hay heartbeat ràng buộc owner với "vẫn còn sống". "Vẫn chạy" và "đã chết dưới tên cũ" không phân biệt được nếu chỉ nhìn dòng dữ liệu.
- **Sửa thế nào:** **xung quanh** gốc, không động vào gốc.
  - cảnh báo cấu hình (`680305aea`);
  - đếm retry khi phục hồi (`234f5a062`);
  - reconcile việc của chính mình khi dừng (`c15b565c8`);
  - reconcile cha mồ côi (`4708a3661`);
  - trần thời gian theo loại task (`8133c5ab7`, `698620786`);
  - watchdog chỉ log.

  Bản sửa gốc từ cộng đồng (PR #3779) bị từ chối.
- **Cái nào sống:** mọi bản vá xung quanh đều còn ở HEAD [E1].
- **Cái nào hồi quy:** triệu chứng "kẹt `processing` dưới id cũ" lặp ở #2165 (06), #2359 (06), #3194 (08). Mỗi lần đều được trả lời bằng "đặt worker id".
- **Phân loại:** tất cả là LOCAL_PATCH hoặc SHARED (`_reclaim_own_processing_tasks` dùng chung cho 3 lối vào, `c15b565c8`). Chưa có ARCHITECTURAL_FIX; cái duy nhất được đề xuất (lease) đã bị bác.
- **Vì sao quay lại:**
  1. Giải pháp của maintainer là **một cấu hình người vận hành phải nhớ**. Lỗi xảy ra đúng ở những bản triển khai quên nó, và #3194 cho thấy nhớ rồi vẫn còn việc treo dưới id cũ.
  2. Health chỉ đo "DB tới được", nên không ai thấy hàng đợi đứng.
- **Bất biến tổng quát:**
  - Quyền sở hữu một việc phải **hết hạn** nếu không được gia hạn, vì danh tính có thể đổi.
  - "Tiến trình sống" ≠ "hàng đợi chảy". Bề mặt trạng thái phải có tuổi việc cũ nhất và số việc kẹt.
  - Mọi việc có trạng thái cuối trong thời gian hữu hạn.
- **agent_cli_analog** (giả thuyết): run hoặc job gắn với một server hay tiến trình mà sau khi khởi động lại có danh tính khác (socket theo hash, PID). Trạng thái "đang chạy" của run có tự hết hạn không? Chưa kiểm.
- **proposed tests:** T3 (tệp 02), cộng **T-F6b**: giết tiến trình sở hữu, khởi động lại với **danh tính khác**. Kỳ vọng: việc cũ tới trạng thái cuối hoặc được nhận lại trong N giây; lệnh `ps`/`status` báo việc kẹt; không cần thao tác tay.

### F7. Trích xuất trả rỗng hoặc hỏng mà vẫn "thành công"; một phần thành công đè dữ liệu cũ — FALSE_SUCCESS, PARTIAL_SUCCESS

#### IssueRecord

**#5582** (mở 14/10) — "retain: missing top-level facts silently succeeds with zero memories" [E0]
- affected: 1.0.0 (commit `39c65ffe2`). Vẫn **OPEN** ngày 19/10; không có comment của maintainer.
- symptom: retain trả 200 và `items_count: 1`, nhưng `memory_unit_count: 0`; `FAIL_ON_EXTRACTION_ERRORS` không có tác dụng [E0].
- root_cause [E1 xác nhận ở HEAD]: `_coerce_fact_response` nhận mọi dict (`api/engine/retain/fact_extraction.py:2005-2011`); `.get("facts", [])` ở dòng 2171 và 3153.
- fork: `dandomin` nhánh `fix/5582-extraction-envelope`. Người báo tự báo: trên main 34 regression đỏ; sau khi vá 172 xanh [E0].
- state: OPEN. Audit cũ đã có (§13.4); điểm mới là **có bản sửa ở fork mà không ai mở PR**.

**#5589** (mở 15/10) — "Retain commits document tracking … even when chunk extraction failed, leaving 0-unit documents" [E0]
- affected: 1.0.0. OPEN.
- symptom: người báo đếm 883 document có 0 unit sau một lần hết quota embeddings [E0].
- root_cause [E1 xác nhận]: consumer chạy `handle_document_tracking` (`api/engine/retain/orchestrator.py:3069, 3281`) rồi commit, còn `producer_error` chỉ được raise ở cuối (`orchestrator.py:3486-3487`).
- fork: `choi138` nhánh `fix/retain-atomic-document-tracking` (tệp 04).
- state: OPEN. Audit cũ đã có (§13.7); điểm mới là có bản sửa ở fork.

Ngoài ra #3820 (F4) cho thấy parser structured output được sửa hai lần. #2413 (tệp 02) cho thấy lỗi parse làm worker kẹt mà health xanh.

#### Phân tích họ F7

Audit cũ §13.5 đã ghi họ này "KHÔNG hội tụ" (phễu "danh sách rỗng → commit → `completed`"). Phần **mới**:
- cả hai issue mở đều có bản sửa sẵn ở fork kèm test (E0), nhưng không có PR upstream;
- `d80927b79` (#5565, 15/10) vừa bịt thêm một lối vào (lời "no facts" không phải JSON), trong khi lối dict-không-có-`facts` (#5582) vẫn mở [E1 tsv].

Lối thứ 5+ của cùng phễu, đúng như audit cũ dự báo.

- **Phân loại:** từng lối là LOCAL_PATCH. Đề xuất ở fork #5582 là "envelope phải có `facts` là list", tức chặn ở biên parser, gần SHARED hơn.
- **Bất biến** (giữ của audit cũ, nhấn thêm): ghi đè trạng thái bền chỉ xảy ra **sau khi** mọi phần của lượt ghi đã thành công; rỗng do lỗi không bao giờ thành `completed`.
- **proposed test:** T7 (tệp 02).

### F8. Truy hồi và tiêm memory cho coding agent; câu hỏi lợi ích — RETRIEVAL_NOISE, STALE_CONTEXT, CONTEXT_INJECTION

#### IssueRecord

**#5713** (MỚI, 17/10) — "coding-agents: auto-inject runs once per session and is never refreshed, so later decisions never reach the agent" [E0]
- affected: HEAD `5331da2`, self-hosted. OPEN.
- symptom: tiêm memory chỉ ở prompt đầu, sau đó dùng lại bản cache. Theo người báo, reflect p50 khoảng 14 s, và 92/298 lần (31%) chạm timeout 20 s rồi fallback; recall p50 khoảng 2,7 s [E0, số người báo tự đo].
- root_cause [E1 xác nhận]: mọi nhánh tiêm đều có điều kiện `reflectAnswer === undefined` (`ca/hook.ts:264, 280, 293`).
- state: OPEN. DESIGN_LIMITATION (STALE_CONTEXT trong phiên).

**#5721** (MỚI, 17/10) — "coding-agents: the page search tool promises a 'matching passage' … with no relevance cutoff every page comes back for any query" [E0]
- affected: plugin 0.8.1, API 0.10.1. OPEN.
- số người báo tự đo [E0]:
  - 4 bank có 5–11 trang; mọi lần tìm đều trả mọi trang tới `limit`, kể cả câu vô nghĩa;
  - "94 of 94 snippets were the first 280 characters";
  - "none of the 8 real searches was followed by a `hindsight_read_knowledge_page` call".
- root_cause [E1 một phần]: plugin bỏ `score` có chủ ý. Comment `ca/knowledge-tools.ts:222-226` giải thích RRF ra khoảng 0,03 nên model sẽ hiểu sai. Kết quả là agent không có tín hiệu "không khớp gì". Phần server (nhánh vector không có ngưỡng) tôi chưa mở dòng.
- liên hệ: #5503 (đóng "working as designed"), PR #4324 ("A relevance cutoff … could be a follow-up").
- state: OPEN.

**#2347**: tệp 02 §6.3 (INSUFFICIENT_EVIDENCE). **Bổ sung "có ai đo không"** [E1 đọc tệp; số là E0 với chúng ta vì không chạy]:
- `hindsight-system-evals/hindsight_system_evals/coding_agents/README.md:3`: chỉ một số đo, "searches per user turn".
- Dòng 7: "1360 user turns, 7 searches (0.5% of turns), 37 memory *writes*". Đây là đo trên transcript thật của một developer (commit `01bb474d6`, 18/09).
- Dòng 287: "No pass/fail gate and no CI job".
- `hindsight-system-evals/evals/test_09_coding_agent_reflect.py:1-30`: so **hai chế độ reflect với nhau** trên bank sde-bench (`boltons-budget-001`, 06/10), tức chấm "quyết định đúng" và chi phí.
- grep `no memory|without memory|baseline|ablation` trong `coding_agents/` chỉ ra "baseline" theo nghĩa lần chạy mốc của chính harness [ĐO]. **Không có nhánh không-memory hay repo-native.**

Kết luận: upstream đo **mức sử dụng** (agent có đọc memory không) và **độ đúng giữa các chế độ**. Upstream không đo **lợi ích so với không có memory**. Con số sử dụng do chính upstream công bố (0,5% lượt có tìm) là tín hiệu **ngược** với giả định "memory tự được dùng".

#### Phân tích họ F8

- **Vì sao mắc.** Tiêm một lần thì giảm độ trễ nhưng để context cũ (#5713). Tìm hybrid không có ngưỡng thì luôn trả "có kết quả", nên agent không phân biệt được "có liên quan" với "không có gì" (#5721). Không có bằng chứng lợi ích, nên mọi chỉnh sửa được chấm bằng tỉ lệ sử dụng.
- **Cái nào sống / hồi quy:** chưa có bản sửa nào.
- **Vì sao quay lại:** RETRIEVAL_NOISE và STALE_CONTEXT là hai mặt của một đánh đổi (độ trễ, chi phí so với độ tươi, độ chính xác). Upstream đang tối ưu "agent có đọc không", chứ không phải "agent làm đúng hơn không".
- **Bất biến tổng quát:**
  - Kết quả truy hồi phải mang tín hiệu "khớp hay không" mà agent dùng được; "rỗng" phải là một câu trả lời khả dĩ.
  - Tính năng memory phải chứng minh lợi ích so với baseline không-memory và repo-native, trước khi tối ưu mức sử dụng.
- **agent_cli_analog** (giả thuyết): context file hay bài học được tiêm một lần đầu phiên (M0/M1 của SE 29).
- **proposed tests:** T5 (tệp 02), cộng **T-F8b**: truy vấn vô nghĩa vào kho bài học hay context → kỳ vọng 0 kết quả hoặc một cờ "không khớp"; **T-F8c**: quyết định mới ghi giữa phiên phải tới được lượt kế tiếp.

---

## 3. Bản đồ ranh giới trạng thái (overlay §7)

| Ranh giới | Issue cho thấy hai trạng thái bị gộp | Trạng thái HEAD |
|---|---|---|
| ACCEPTED ≠ DURABLE | #2985 (cha `pending` mất document), #5251 | cha mồ côi: có reconcile, nhưng chỉ cho worker id của chính mình |
| DURABLE ≠ EXTRACTED | #5582, #5589, #3876 | #5582, #5589 OPEN; #3876 FIXED |
| EXTRACTED ≠ INDEXED/REPRESENTED | #5054, triage chatter, #5731, #5730 | DESIGN_LIMITATION + 2 OPEN |
| INDEXED ≠ SEARCHABLE (đúng phạm vi) | #2808, #5728 (rỗng do `all_strict`), #5030, #5371 | rỗng vẫn không có cờ |
| SEARCHABLE ≠ CURRENT | #5713 (tiêm một lần), #5728 (watermark), #4560 | OPEN |
| CURRENT ≠ VERIFIED | #5721 (không có ngưỡng; snippet là đầu trang), #2347 (không đo lợi ích) | OPEN / INSUFFICIENT_EVIDENCE |
| "không liên lạc được" ≠ "không" | #5221, #5735 | hermes đã sửa, coding-agents mở |
| tiến trình sống ≠ hàng đợi chảy | #1671, #2413, #3002, #3194 | DESIGN_LIMITATION |
| `failed` ≠ `superseded` | #5434 | OPEN |

## 4. Bổ sung và sửa audit cũ (`luannt-hindsight-1610.md`) và tệp 02

| Chỗ cũ | Sửa / bổ sung | Nguồn |
|---|---|---|
| 02 §4, chuỗi cursor "#3136 `aaa58f247`" | **#3136 là PR** (đóng không merge, bị #3412 thay). `aaa58f247` sửa một bug **khác**, tất định: prompt hook xoá cursor ở mọi phiên, trên 7/11 harness | trang PR #3136 [E0]; message `aaa58f247` [E1] |
| Audit §12.7, SE 24 H-29..H-31: #3989 là lỗi cursor phía client | #3989 có gốc **server**: append chia slice làm cụt `original_text`, và lần append sau tombstone fact (`fda969777`). Đây mới là chỗ fact 197 → 131 | `fda969777` [E1] |
| Audit §12.7: chỉ ghi lỗ "Stop cuối" và `tmpdir()` | Thêm **#5735** (mở 18/10): client cache câu trả lời âm vĩnh viễn trong runtime pi; cursor của pi nằm trong RAM (`ca/runtime.ts:46`) | [E0] + `ca/hindsight.ts:540-546` [E1] |
| Audit §13.8 bất biến 10 ("lan tới mọi bản sao") | Bằng chứng mới, rõ nhất: hermes sửa #5221 ngày 12/10 kèm 4 test; bản sao coding-agents không được sửa và bị báo lại 6 ngày sau (#5735) | `10b14dd0d` [E1], `hermes/tests/test_provider.py:559` [E2] |
| Audit §13.4: "#2359, #3194 NOT_PLANNED, gốc là không có lease" | Có **PR lease cụ thể của cộng đồng** (#3779, có heartbeat, đếm retry, đã sửa cho Oracle) bị đóng 31/08 với "set stable worker id via env." #3194 cho thấy đặt id ổn định rồi vẫn kẹt dưới id cũ. Chuỗi: #2165 → #2359 → #3194 → #3779 | [E0] ×4; `git log --all -S` rỗng [ĐO] |
| Audit §13.5: consolidation "đang hội tụ" | Chỉ hội tụ cho kiểu "áp nửa vời". Kiểu "đóng dấu mà không biểu diễn" thì #5054 đóng "by design"; thêm một lối qua triage `0d8e58b3e`; lối reset bị chặn (#5731, mở); trạng thái không đọc được ở endpoint chi tiết (#5730, mở) | [E0] + `consolidator.py:2752`, `memory_engine.py:12060` [E1] |
| Audit §13.4: #3876 FIXED_VERIFIED | Vẫn FIXED. Trang ghi thêm commit 19/10 (`bc0b8c1`, `11ed416`, PR #5744) **sau HEAD**, chưa kiểm | [E0] |
| Audit §13.3: #2670 là `duplicate`, hành vi `all_strict` vẫn còn | Nối được chuỗi: #2670 → #2808 (sửa = chỉ cho cấu hình được, `41d71a981`) → **#5728** (18/10, cùng triệu chứng "no records"). HEAD vẫn mặc định `all_strict` (`memory_engine.py:1866`) | [E0] + [E1] |
| Audit, SE: refresh mental model "rỗng thì giữ" | Test HEAD **ghim** việc "trang anh em là nguồn": `test_mental_models.py:3576`. Vì vậy #5728 là xung đột thiết kế, không phải sót | [E2] |
| 02 §3.4 so sánh claude-mem #3608, "nhiều bản sao" | Họ hook có **4** cơ chế chống đệ quy (02 ghi 3). Server không truyền `HINDSIGHT_DISABLE_HOOKS` mà chính hook của Hindsight đã biết đọc | `claude_code_llm.py:103-106`, `ca/retain-hook.ts:186` [E1] |
| Audit §13.4, §13.7: #5582, #5589 OPEN | Vẫn OPEN ngày 19/10, nhưng **có bản sửa ở fork kèm test** (`dandomin`, `choi138`) mà không có PR upstream | [E0] |
| 02 §6.3, #2347: "không có con số nào" | Repo **có** một số đo do upstream công bố: 0,5% lượt có tìm memory (1.360 lượt). Harness đo mức sử dụng, không đo lợi ích, không có cổng CI | `coding_agents/README.md:3,7,287` [E1, số là E0] |
| Audit §13.6: liệt kê `not_planned` | #2165 do **người báo** tự đóng (không phải maintainer); #5054 đóng "completed" mà chỉ có "by design" và không commit | [E0] |
| Audit, 02: phạm vi "Upstream sau `5331da2f4`: không biết" | Issue mới nhắc **1.0.0** (`39c65ffe2`). Mọi trạng thái mã ở đây là của `5331da2f4` | [E0] |
| Không có trong audit | #5713, #5721 (relevance cho coding), #5729 (thứ tự settings), #5555 (ngữ cảnh không-primary của hermes), #5434 (`failed` ≠ `superseded`) | [E0] + [E1] |

---

## Hợp đồng đầu ra (spec §63)

- **SCOPE:** issue/PR Hindsight thuộc các họ ở overlay §7. Trọng tâm là phần audit 16/10 chưa có và các issue mở sau 16/10. Gồm 8 họ, 32 issue/PR có IssueRecord (cộng 3 cái dẫn từ tệp 02).
- **REVISION AUDITED:** mã ở `5331da2f4db369528bb122075341a7de9c3df6df`; tracker đọc ngày 19/10.
- **FILES/COMMITS/ISSUES INSPECTED:**
  - Trang (35): xem §1.
  - Commit: `aaa58f247`, `e655d889b`, `fda969777`, `a0e2583b0`, `1238dbdbc`, `d3f0001ed`, `10b14dd0d`, `bcae23d9f`, `c7b17f0ad`, `41d71a981`, `ca38687cd`, `0d8e58b3e`, `680305aea`, `234f5a062`, `c15b565c8`, `4708a3661`, `8133c5ab7`, `698620786`, `e00154496`.
  - Mã HEAD: `ca/{hindsight,chat,retain-cursor,runtime,hook,knowledge-tools,legacy,missions}.ts`, `claude_code_llm.py`, `cursor_llm.py`, `memory_engine.py` (1844-1880, 4838-4853, 12014-12067, 17798-17801), `consolidator.py`, `triage.py`, `curation.py`, `fact_extraction.py`, `orchestrator.py`, `worker/{poller,main}.py`; test `test_mental_models.py`, `test_consolidation_triage.py`, `hermes/tests/test_provider.py`; README eval coding-agents.
- **VERIFIED FACTS:**
  - Mọi dòng có nhãn [E1] hoặc [ĐO] ở trên.
  - Đặc biệt: cache âm ở `ca/hindsight.ts:540-546`; cursor 4 → 8 trường; PR lease không có trong mọi nhánh của clone; `deleted_count > 0` chặn reset; `select_cols` thiếu hai cột; nhánh bỏ retry → `failed`; test ghim "anh em là nguồn"; 5 tệp `state.py` còn "harmless".
- **INFERENCES:**
  - Lý do lặp của từng họ (§2 "vì sao quay lại").
  - Cache âm chỉ kéo dài một Stop trên harness hook.
  - Bank coding-agents tránh được triage chatter nhờ có mission.
  - Một biến env đã chặn được #5542 từ phía hook.
- **UNKNOWNs:**
  - Upstream sau HEAD (1.0.0, commit 19/10).
  - Nhánh "cursor = `max(updated_at)`" của #5728.
  - Test có xanh không (không chạy).
  - Nội dung các comment bị ẩn trên #3989.
  - Phần server của #5721.
  - PR #1674 (quarantine) có merge không.
  - Trạng thái #5419, #4130, #5497, #5544 (không đọc lại).
- **FAILURE MODES:** CURSOR_ADVANCE_BEFORE_DURABILITY, HOST_ADAPTER_MISMATCH, SESSION_CONTAMINATION, SCOPE_FAILURE, WRONG_SCOPE_RETRIEVAL, FALSE_SUCCESS, PARTIAL_SUCCESS, DERIVED_STATE_STALENESS, STALE_CONTEXT, RETRIEVAL_NOISE, WORKER_FAILURE, ZOMBIE_OPERATION, RETRY_STORM → false failed, OBSERVABILITY_FAILURE.
- **AGENT CLI RELEVANCE:** chỉ là giả thuyết tương tự, ghi theo từng họ. Không kết luận phơi nhiễm.
- **PROPOSED TESTS:** T-F1a/b/c, T-F2b, T-F3b, T-F4, T-F5, T-F6b, T-F8b/c, cộng T1–T7 của tệp 02.
- **NO-CHANGE FINDINGS:**
  - Không có issue nào về triage "chatter" (đã tìm).
  - Không thấy báo cáo hồi quy nào sau bản sửa #3876, #4493, #4560 (đường hook), #5542, #2985.
  - #5582, #5589 vẫn như audit cũ ghi (OPEN).
