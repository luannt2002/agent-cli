# 04 — Fork của Hindsight: có fork nào đổi kiến trúc không?

Ngày: 2026-10-19. Chỉ đọc. Upstream ở clone: `5331da2f4` (2026-10-16). Nhãn nguồn như tệp 03: [E0] trang GitHub, [E1] commit/mã tôi đọc, [E2] test tồn tại, [ĐO] lệnh vừa chạy, [SUY LUẬN].

## 0. Kết luận thẳng

- **Ngân sách fork: 6/6 trang.**
  1. Danh sách fork, trang 1 (lọc active, 1 năm, sắp theo lần cập nhật gần nhất);
  2. `MillionthOdin16/hindsight`;
  3. `karljsamuel/hindsight`;
  4. commit `sanastasiou/hindsight@ddc3f2d`;
  5. compare `choi138:fix/retain-atomic-document-tracking`;
  6. compare `dandomin:fix/5582-extraction-envelope`.

  Không trang nào đòi đăng nhập. Hai trang compare báo "taking too long to generate", nên **không có diff**, chỉ có danh sách commit và message.
- **Phạm vi đã soi:**
  - 25 fork ở trang 1 (danh sách có 33 trang; tôi không đếm tổng);
  - 2 trang chủ fork;
  - 3 trang commit/compare;
  - cộng 7 nhánh hay commit fork lộ ra qua 35 trang issue/PR ở tệp 03 (không tính vào ngân sách fork).
- **Kết quả: 0 fork có thay đổi kiến trúc được chứng minh.**
  - Mọi thay đổi chỉ-có-ở-fork mà tôi thấy được đều là **một nhánh vá cho đúng một issue**, tức TRACKING_FORK dùng làm chỗ dựng bản sửa.
  - Thay đổi gần kiến trúc nhất là **thu hồi việc theo lease** (PR #3779, `sanastasiou`). Nó **bị upstream bác** ngày 31/08 và không có trong bất kỳ nhánh nào của clone [ĐO].
- **Vì sao fork chỉ là chỗ đậu bản vá [E1].** Từ 2026-10-05 (`bb099497e`), `CONTRIBUTING.md:7` ghi "We don't accept pull requests from outside the Hindsight team. PRs opened by external contributors are closed without review." Hệ quả quan sát được:
  - Ba bản sửa cho issue **đang mở** nằm trên nhánh fork mà không có PR: #5582 (`dandomin`), #5589 (`choi138`), #5731 (`koriyoshi2041`) [E0].
  - Hai bản sửa có nhánh fork (#5371, #5417) được upstream **viết lại** thành commit của maintainer, không có trailer co-author [ĐO `git log --format=%(trailers)`].

## 1. Cách làm và giới hạn

- Commit của fork **không có** trong clone: `git cat-file -t` cho `f70c1cfde`, `b32c4a1`, `5d3e721`, `1d6058d`, `62052e5`, `ddc3f2d`, `e208bb17`, `18cbf457` đều ra "Not a valid object name" [ĐO]. Vì vậy các trường CHANGED FILES và NEW TESTS dưới đây lấy từ trang GitHub (E0), không phải từ diff.
- Trang danh sách fork không hiện ahead/behind. Hai trang chủ fork tôi mở cũng **không** có dòng "N commits ahead/behind" [E0]. Do đó tôi không phân biệt được MIRROR với TRACKING_FORK cho 25 fork ở danh sách.
- Theo spec, tôi **không** xếp fork theo sao. Mọi fork ở trang 1 có 0–1 sao [E0], nên thứ tự sao cũng vô nghĩa ở đây.

## 2. Trang 1 danh sách fork (active, 1 năm, sắp theo cập nhật) [E0]

25 fork, tạo từ 26/02 tới 17/10, đều cập nhật trong khoảng 18–19/10. Không fork nào có mô tả riêng. Các fork có số PR hoặc issue khác 0 trong chính fork:

| Fork | Đặc điểm hiện trên danh sách | Mở thêm | Kết luận |
|---|---|---|---|
| `MillionthOdin16/hindsight` | 107 PR mở trong fork | trang chủ: 3.120 commit, README là của upstream, không có dòng ahead/behind, không nêu mục đích [E0] | INSUFFICIENT_EVIDENCE. [SUY LUẬN] 107 PR nội bộ có thể là một luồng tự động; cần trang `/pulls` để kiểm |
| `karljsamuel/hindsight` | 7 issue, 1 PR | trang chủ: 3.495 commit, không có dòng ahead/behind, không nêu mục đích [E0] | INSUFFICIENT_EVIDENCE |
| `davidhsv`, `Korayem`, `taomylife521/AI-hindsight`, `ZhuQingping` | 1–2 PR | không mở | chưa soi |
| `koriyoshi2041/hindsight` | 0/0 | thấy qua issue (§3, FC4) | TRACKING_FORK |
| 19 fork còn lại | 0/0 | không mở | chưa soi; [SUY LUẬN] phần lớn là mirror hoặc fork để báo issue |

## 3. ForkChange

Mỗi bản ghi theo overlay §10. `decision` ở đây là **ứng viên cho ý tưởng**, chưa phải quyết định V3, vì spec cấm kết luận agent-cli có phơi nhiễm trước khi audit.

### FC1 — `sanastasiou/hindsight`: thu hồi việc `processing` theo lease (PR #3779 + `ddc3f2d`) — WORKER_FIX

| Trường | Giá trị |
|---|---|
| fork_base | không biết (trang commit chỉ cho parent `ced4ee0`) [E0] |
| commit_or_series | PR #3779 (khoảng 24/08 → đóng không merge 31/08); `ced4ee0` (sửa log owner), `ddc3f2d` (Oracle-safe, trần retry, env/docs) [E0] |
| original_upstream_behavior | chỉ thu hồi `processing` của **chính** `worker_id` lúc khởi động (`api/worker/poller.py:1454,1475`) [E1] |
| changed_behavior | quét dòng `processing` của **mọi** owner có `claimed_at` **và** `updated_at` cũ hơn `HINDSIGHT_API_WORKER_ORPHAN_LEASE_SECONDS` (mặc định 86400; 0 = tắt); đếm `retry_count`, quá trần thì `failed` và cộng dồn lên cha; chạy **sau** `_reconcile_orphaned_parents` để lỗi của nó không chặn bước kia [E0] |
| problem_addressed | worker đổi danh tính (hostname container) làm việc kẹt mãi: #2165, #2359, #3194; tác giả dẫn thêm #3448, #3709, #3594, #3720 [E0] |
| architecture_change | **có, nhỏ**: quyền sở hữu việc thành lease có hạn. Đây là thay đổi duy nhất mang tính kiến trúc mà tôi thấy ở fork |
| complexity_added | `poller.py` +100/−42; 1 biến env; 2 tệp `env.example` và 1 tệp docs [E0] |
| tests_added | **không có** trong `ddc3f2d`; tác giả không chạy được test sync template [E0] |
| runtime_evidence | không có (E0: trang không ghi lần chạy nào) |
| new_failure_modes | strix-security[bot]: race CWE-362, đặt lại một consolidation **còn sống**, dẫn tới chạy trùng. Đã giảm nhẹ bằng điều kiện `updated_at`. SQL chỉ chạy được trên Postgres, gặp Oracle thì làm **hỏng lặng** bước reconcile cha mồ côi vì chung một `try`; reviewer Sanderhoff-alt bắt được [E0]. [SUY LUẬN] lease 24h vẫn để việc kẹt tới một ngày |
| current_upstream_equivalent | **không có.** `git log --all -S'_reclaim_expired_processing_tasks'` và `-S'WORKER_ORPHAN_LEASE'` rỗng [ĐO]. Upstream chỉ có các lớp xung quanh: cảnh báo `WORKER_ID` (`680305aea`), đếm retry khi phục hồi (`234f5a062`), trần thời gian theo loại task (`8133c5ab7`, `698620786`) [E1] |
| still_relevant | có. Lý do upstream đưa ra ("set stable worker id via env.") không phủ ca #3194, nơi id đã ổn định mà việc vẫn treo dưới id cũ [E0] |
| agent_cli_equivalent_problem | (giả thuyết) run/job gắn với danh tính server hay tiến trình mà danh tính đổi sau khi khởi động lại. Chưa kiểm |
| decision (ứng viên) | **ADAPT_IDEA**: lease có hạn + điều kiện heartbeat + trần retry + chạy độc lập với các bước dọn khác. Chỉ áp dụng nếu audit agent-cli thấy cùng ràng buộc |

### FC2 — `choi138/hindsight` nhánh `fix/retain-atomic-document-tracking` (#5589) — OTHER (độ bền retain)

| Trường | Giá trị |
|---|---|
| fork_base | không biết |
| commit_or_series | 1 commit (choi138, 15/10) "fix(retain): do not commit document tracking when extraction already …failed"; 2 tệp; trang không hiện diff [E0] |
| original_upstream_behavior | consumer chạy `handle_document_tracking` (xoá unit cũ, ghi `original_text`) rồi commit; `producer_error` chỉ raise ở cuối (`api/engine/retain/orchestrator.py:3069, 3281, 3486-3487`) [E1] |
| changed_behavior | producer ghi nhận chunk lỗi **ngay khi** nó lỗi; consumer kiểm ở mỗi điểm commit và "discards the batch instead of writing it"; batch đã commit trước lỗi được giữ cho lần gửi lại resume [E0 message] |
| problem_addressed | người báo đếm 883 document có 0 unit sau một lần hết quota embeddings [E0] |
| architecture_change | không. Chỉ thêm một cổng tại điểm commit có sẵn |
| tests_added | theo trang issue: một test có DB, đỏ trên main [E0]. Không thấy tên test |
| new_failure_modes | [SUY LUẬN từ message] nếu lỗi tới **sau** batch đầu đã commit thì document cũ đã bị xoá và chỉ còn một phần bản mới, tức chưa đạt "document cũ nguyên vẹn tới khi retain thành công" như issue đòi. Cách kiểm: cho chunk 3 lỗi sau khi batch 1 đã commit, rồi đếm unit |
| current_upstream_equivalent | không có (#5589 OPEN ngày 19/10) [E0] |
| still_relevant | có |
| agent_cli_equivalent_problem | (giả thuyết) ghi đè một bản ghi bền (session, artifact) trước khi mọi phần của lượt ghi thành công |
| decision (ứng viên) | **ADAPT_IDEA** cho bất biến "không ghi đè trạng thái bền khi một phần đã biết là hỏng". Bản vá cụ thể là IGNORE (gắn chặt với pipeline streaming của Hindsight) |

### FC3 — `dandomin/hindsight` nhánh `fix/5582-extraction-envelope` (#5582) — OTHER (thành công giả)

| Trường | Giá trị |
|---|---|
| fork_base | người báo nói bản vá áp được lên commit `39c65ffe2` (1.0.0) [E0] |
| commit_or_series | 3 commit ngày 14/10: `c90e976` (dandomin) "validate extraction envelopes before reading facts"; `642a4d1` và `9116145` (tác giả "Hermes") "abort malformed batch envelopes before document replacement" và format; 6 tệp, 2 contributor [E0]. [SUY LUẬN] "Hermes" có thể là danh tính của một agent |
| original_upstream_behavior | `_coerce_fact_response` nhận mọi dict; `.get("facts", [])` biến khoá thiếu thành 0 fact thành công (`api/engine/retain/fact_extraction.py:2005-2011, 2171, 3153`) [E1] |
| changed_behavior | chỉ nhận dict có `facts` là list; hỏi lại có giới hạn; batch hỏng thì ghi vào kênh lỗi trích xuất và **huỷ trước khi thay document** [E0] |
| architecture_change | không. Chặn ở biên parser (gần SHARED) |
| tests_added | regression trên PostgreSQL. Người báo tự báo: main có 34 test đỏ và 4 test đối chứng xanh; bản vá cho 172 xanh, 7 skip [E0] |
| runtime_evidence | chỉ lời khai của người báo |
| new_failure_modes | người báo tự ghi: streaming nhiều chunk vẫn không atomic; lỗi batch vẫn theo chính sách không-fatal [E0] |
| current_upstream_equivalent | không có (#5582 OPEN, không có comment của maintainer). Upstream vừa bịt một lối anh em là lời "no facts" không phải JSON (`d80927b79`, #5565) [E1 tsv] |
| still_relevant | có |
| agent_cli_equivalent_problem | (giả thuyết) đầu ra cấu trúc của model thiếu trường bắt buộc mà bị đọc thành "rỗng hợp lệ" |
| decision (ứng viên) | **ADAPT_IDEA**: validate envelope ở biên; "thiếu" ≠ "rỗng" |

### FC4 — `koriyoshi2041/hindsight` (contributor tái lập và vá) — TRACKING_FORK

| Trường | Giá trị |
|---|---|
| commit_or_series | `fix/source-clear-zero-observations-5731` @ `1d6058d` (19/10; báo 37 + 45 test xanh, ruff/type xanh; "could not open the upstream PR because their GitHub credential was blocked") [E0]; `18cbf457` (05/08, "review fixes: keep .queued stage until permits, stamp .backoff…", trên timeline #3002) [E0] |
| hoạt động ở upstream | tái lập trên `main` cho #4493, #5371, #5728 (trên `5331da2f4`), #5731; review PR #3779 [E0] |
| original → changed (1d6058d) | `clear_observations_for_memory` chỉ requeue khi `deleted_count > 0` (`api/engine/memory_engine.py:12060, 12067`) [E1] → reset và enqueue theo yêu cầu tường minh, kể cả khi không xoá dòng nào [E0] |
| architecture_change | không |
| current_upstream_equivalent | không có (#5731 OPEN) |
| decision (ứng viên) | IGNORE cho bản vá. Bất biến "thao tác reset tường minh luôn reset" đã ghi ở tệp 03 F5 |

### FC5 — `rodrigoedu94-png/hindsight` @ `f70c1cfde` (#5417) — SCOPE_FIX

| Trường | Giá trị |
|---|---|
| changed_behavior | gom luật lọc tag cho knowledge base, entity, proxy control plane (theo issue) [E0] |
| current_upstream_equivalent | **upstream đã nhận ý tưởng nhưng viết lại**: `3f8fe203f` (16/10, Nicolò Boschi, **không** co-author) [ĐO]. Cùng người (email `rodrigoedu94@gmail.com`) có 5 commit Oracle **được merge với tên tác giả** ngày 05–06/10, PR #4804..#5164 [ĐO `git log --author`] |
| [SUY LUẬN] | khớp với chính sách `bb099497e` (05/10): PR mở trước chính sách thì merge giữ tên tác giả; bản sửa #5417 (issue mở 07/10) thì được viết lại. Mới thấy ở một người, chưa đủ để gọi là quy luật |
| decision | OBSOLETE (upstream đã có) |

### FC6 — `JayOfTheKeyboard` nhánh `fix/mcp-recall-reflect-tags-match` @ `b32c4a1`, `5d3e721` (#5371) — SCOPE_FIX

| Trường | Giá trị |
|---|---|
| changed_behavior | `GET /documents` và `/graph` trả 422 cho `tags_match` lạ [E0] |
| current_upstream_equivalent | `89594d22d` (16/10, Nicolò Boschi, không co-author) phủ 6 lối vào và parser chung [E1/ĐO] |
| decision | OBSOLETE |

### FC7 — `Sanderhoff-alt` @ `e208bb17` (trần idle cho consolidation, #3002/#3726) — WORKER_FIX

| Trường | Giá trị |
|---|---|
| current_upstream_equivalent | **merge giữ tên tác giả**: `698620786` (31/08, tác giả Sanderhoff-alt) [ĐO]. Đây là trước chính sách 05/10 |
| ghi chú | Sanderhoff-alt cũng là người mở #2347 (câu hỏi lợi ích) và là tác giả thứ 5 theo số commit: 89 commit [ĐO `git shortlog -sn`]. Họ không có commit nào đo lợi ích (grep `bench|eval|coding` trong commit của họ chỉ ra sửa tích hợp và docs) [ĐO] |
| decision | OBSOLETE |

`sanastasiou` cũng có `ddc3f2d` tham chiếu #2985 (xem FC1). Upstream đã có reconcile cha mồ côi (`4708a3661`), nhưng không có lease.

## 4. Trả lời các câu hỏi overlay §8–10

| Câu hỏi | Trả lời |
|---|---|
| Nỗi đau nào sinh ra fork? | Ở các nhánh có nội dung: worker kẹt khi đổi danh tính (FC1); thành công giả và thành công một phần khi retain (FC2, FC3); reset không tới được (FC4); phạm vi tag (FC5, FC6); slot consolidation bị giữ (FC7) |
| Upstream không muốn hay không thể? | **Không muốn** với lease (FC1, maintainer chọn cấu hình `WORKER_ID`). **Không nhận PR ngoài** từ 05/10 (`CONTRIBUTING.md:7`). Với FC2/FC3/FC4 thì chưa phản hồi (issue mở, không có comment của maintainer) |
| Riêng của host? | Không fork nào trong mẫu là HOST_PORT |
| Upstream có nhận ý tưởng về sau không? | Có, bằng cách **viết lại**: FC5, FC6. Có, **merge nguyên**: FC7 (trước 05/10). **Bác**: FC1. Chưa: FC2, FC3, FC4 |
| Fork chết chưa? | Không xác định: trang không hiện ahead/behind. Các nhánh FC2–FC4 mới 14–19/10 |
| Fork thêm bug mới không? | FC1 có (race với consolidation còn sống; hỏng lặng trên Oracle), cả hai đều bị review bắt trước khi đóng. FC2: nghi còn hở khi lỗi tới sau batch đầu [SUY LUẬN] |
| Agent CLI có cùng ràng buộc không? | Không kết luận ở đây. Câu cần kiểm cho từng FC đã ghi ở trường `agent_cli_equivalent_problem` |

**Lưu ý về số đo tác giả [ĐO].** Từ 05/10 tới HEAD có 200 commit của Nicolò Boschi và 64 của Chris Bartholomew, cộng 21 tên tác giả khác (mỗi tên 1–19 commit; có Ben 9 commit). Trailer co-author trong cùng khoảng có 38 dòng Nicolò Boschi, 5 dòng "Claude Opus 5.5" và 1 dòng "Hermes Bot". Vậy chính sách "không nhận PR ngoài" **không** có nghĩa là không có commit của người ngoài trong lịch sử. Tôi không xác định được cơ chế: PR mở trước chính sách, cherry-pick giữ tên, hay thành viên nhóm mà tôi không biết.

---

## Hợp đồng đầu ra (spec §63)

- **SCOPE:** fork của `vectorize-io/hindsight`, tìm thay đổi có ý nghĩa (không xếp theo sao); ForkChange cho các nhánh thấy được; upstream đã nhận ý tưởng chưa.
- **REVISION AUDITED:** upstream `5331da2f4`; trang fork và issue đọc ngày 19/10.
- **FILES/COMMITS/ISSUES INSPECTED:**
  - 6 trang fork (§0) cùng trang issue/PR của tệp 03 (#3779, #5371, #5417, #5582, #5589, #5731, #3002, #3820).
  - `CONTRIBUTING.md` và commit `bb099497e`.
  - `git log` theo tác giả và trailer; `git cat-file` cho 8 commit fork; `git log --all -S` cho mã lease.
- **VERIFIED FACTS:**
  - Commit fork không có trong clone.
  - Mã lease không có trên mọi nhánh của clone.
  - `CONTRIBUTING.md:7` cấm PR ngoài từ 05/10.
  - `89594d22d` và `3f8fe203f` không có trailer co-author.
  - `698620786` có tác giả là Sanderhoff-alt.
  - Các dòng mã HEAD mà FC2–FC4 nhắm tới vẫn còn nguyên (`orchestrator.py:3069/3281/3486`, `fact_extraction.py:2005-2011`, `memory_engine.py:12060`).
- **INFERENCES:**
  - Phần lớn fork ở danh sách là mirror hoặc fork để báo issue.
  - FC2 còn hở khi lỗi tới sau batch đầu.
  - Quan hệ giữa thời điểm mở PR và chính sách 05/10.
  - "Hermes" là danh tính agent.
- **UNKNOWNs:**
  - ahead/behind của mọi fork;
  - nội dung 107 PR trong `MillionthOdin16`;
  - diff thật của FC2/FC3 (trang compare quá tải);
  - fork ở trang 2–33 của danh sách;
  - fork nào là ARCHITECTURAL_FORK nằm ngoài mẫu.
- **FAILURE MODES:** WORKER_FAILURE / ZOMBIE_OPERATION (FC1), PARTIAL_SUCCESS (FC2), FALSE_SUCCESS (FC3), SCOPE_FAILURE (FC5, FC6), OBSERVABILITY_FAILURE (bước dọn hỏng lặng vì chung `try`, FC1).
- **AGENT CLI RELEVANCE:** ba ý tưởng ứng viên ADAPT_IDEA: lease có hạn + heartbeat + trần retry (FC1); không ghi đè trạng thái bền khi đã biết một phần hỏng (FC2); validate envelope, "thiếu" ≠ "rỗng" (FC3). Không kết luận phơi nhiễm.
- **PROPOSED TESTS:** T-F6b (tệp 03) cho FC1; T7 (tệp 02) cho FC2 và FC3. Không có test mới riêng cho fork.
- **NO-CHANGE FINDINGS:** soi 25 fork ở danh sách + 2 trang fork + 3 trang commit/compare + 7 nhánh fork thấy qua issue: **0 fork có thay đổi kiến trúc được chứng minh, 0 HOST_PORT, 0 RETRIEVAL_EXPERIMENT**. Thay đổi kiến trúc duy nhất được đề xuất (lease, FC1) đã bị upstream bác và không sống tiếp ở nhánh nào tôi đọc được.
