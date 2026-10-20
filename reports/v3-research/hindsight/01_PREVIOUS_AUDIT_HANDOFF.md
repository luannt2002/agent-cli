# 01 — Bàn giao audit Hindsight trước (đã kiểm lại)

Ngày: 2026-10-19. Agent nghiên cứu V3, chỉ đọc. Không chạy docker/uv/test, không chạy lệnh git ghi.
Nguồn được bàn giao: `/home/luannt/application/opensource/luannt-hindsight-1610.md` (852 dòng, `wc -l`) và SE 24, 25, 26, 28, 29, 30, 40 trong `reports/system-evolution/`.
Ký hiệu: **[ĐO]** là lệnh tôi vừa chạy trong phiên này. **[AUDIT]** là audit cũ khai, tôi chưa chạy lại. **[SUY LUẬN]** là suy luận, có ghi cách kiểm.

---

## 1. Revision và phạm vi

| Mục | Giá trị | Bằng chứng |
|---|---|---|
| Revision được audit | `e00154496d3f66d0f6aa63138b1ce63a7bd391c6`. Audit ghi 7 ký tự `e001544`, cùng một commit | [ĐO] `git rev-parse e001544 e00154496` trả cùng một hash |
| Commit đó | 2026-10-16 10:33 +0200, `fix(reflect): a "nothing found" answer needs recall first (#4567) (#5641)` | [ĐO] `git log -1 e00154496` |
| Clone cũ | `/home/luannt/application/opensource/hindsight` vẫn ở `e00154496`, không đổi gì | [ĐO] `git log --oneline -1` ra `e00154496`; `git status --short \| wc -l` ra `0` |
| Số đếm audit dùng làm mốc | 631 tệp test API, 79 system story, 111 migration | [ĐO] `git ls-tree` tại `e00154496`, khớp audit §8 |
| Lịch sử git lúc audit | audit khai 3.652 commit. Clone mới có 3.700 = 3.652 + 48 commit delta | [ĐO] `git rev-list --count 5331da2f4` ra 3700, `e00154496..5331da2f4` ra 48 |

Audit làm hai lượt trong cùng ngày 16/10, cùng một commit:
- **Lượt 1 (§1–10):** bản đồ đọc mã: engine (retain, recall, reflect, consolidation, mental model, transfer, maintenance), vỏ API, worker, MCP, cấu hình, extension, đa tenant, provider, CI, SDK, CLI, control plane, coding-agents. Phần lớn có nhãn 🔎, tức 3 agent chỉ-đọc báo về và tác giả **chưa** tự kiểm.
- **Lượt 2 (§11–17):** đào sâu retain, consolidation, vòng đời, recall, reflect, worker, coding-agents. Khảo cổ issue (1.534 issue; 59 issue lần tới commit/code/test; đọc comment của 40 issue). Chạy thật bằng Docker với stub LLM.

Audit tự ghi những chỗ **chưa** làm (§10, §17): không có LLM thật; không benchmark hay ablation; mới chạy 18 test system (chưa chạy bộ 631 tệp test API); năm nghi vấn Tier-1 chưa có repro; chưa thử bank lớn; chưa chạy Oracle hay store-owned. Audit cũng tự ghi **hạn dùng** khoảng vài tuần, vì nhịp issue là khoảng 270 issue/tháng.

## 2. Những gì đã chạy thật (audit §14)

Môi trường [AUDIT]: `docker run python:3.11`, repo mount `:ro`, `uv sync --frozen --extra embedded-db`, Postgres là pg0 nhúng thật, LLM/embedding/rerank là `hindsight_system_tests/stub_server.py` (stub tất định của repo). Không có API key nên **không có lượt nào dùng LLM thật**.

| # | Thí nghiệm | Kết quả audit khai | Nhãn audit | Hạng SE 24 |
|---|---|---|---|---|
| L1 | import, dựng app | đạt | CHẠY 🔎 | E3🔎 |
| L2 | system test `test_01/20/40/60` | 18/18 xanh trong 48,6 s. Có 2 dòng log "No stub rule matched … search_observations" mà test vẫn xanh, chưa tra | CHẠY 🔎 | E3🔎 |
| E1 | `ExcludeNoneRoute` (bỏ `null`) có hiệu lực không | không: 100/100 route là `UnknownParamsRoute` | **CHẠY ✅** | E3✅ |
| E2 | `run_consolidation` | trả toàn số 0 trong khi DB có observation | **CHẠY ✅** | E3✅ |
| E3a | `kill -9` worker rồi khởi động lại **cùng** id | tự phục hồi, `retry_count=1` | CHẠY 🔎 | E3🔎 |
| E3b | `kill -9` rồi khởi động lại với id **khác** | dòng kẹt `processing`; client thấy `pending`, không lỗi; chỉ `decommission-worker` gỡ được | CHẠY 🔎 | E3🔎 |
| E4 | auth/bind mặc định | không auth vẫn 200; bind `0.0.0.0` | CHẠY 🔎 | E3🔎 |
| E5 | `llm_requests` | lưu nguyên văn prompt/response | CHẠY 🔎 | E3🔎 |
| E6 | `async_operations` khi retention = 0 | sau khoảng 6 phút dòng vẫn còn, payload nguyên văn | CHẠY 🔎 | E3🔎 (phần "vĩnh viễn" là E0) |
| E7 | FK `invalidated_mu_document_fkey` khi migrate 2 schema | FK chỉ có ở schema đầu | **CHẠY ✅** | E3✅ |

Lần này tôi **không chạy lại** thí nghiệm nào: spec cấm docker/uv, và log gốc đã mất (mục 6).

## 3. Mức bằng chứng của 10 kết luận chính (audit §11.3) và trạng thái sau delta

Thang lấy từ SE 24: E0 lời khai · E1 đọc mã · E2 có test · E3 chạy với stub · E5 LLM thật (không có mục nào). Cột cuối nói delta `e00154496..5331da2f4` có chạm tới kết luận đó không. Chi tiết nằm ở `02_UPSTREAM_DELTA.md`.

| # | Kết luận | Hạng | Delta chạm tới? [ĐO bằng `git diff`/`git log` trên tệp liên quan] |
|---|---|---|---|
| 1 | Worker chết rồi lên lại với id khác → việc kẹt `processing` mãi (#2359, #3194 not_planned) | E3🔎 | Không. `worker/poller.py` trong delta chỉ đổi `time.time()` sang `perf_counter()` (29698acb6). `recover_own_tasks` vẫn lọc theo `worker_id` (poller.py:1505) |
| 2 | "0 fact mà `completed`" có ≥4 lối vào; lối #5582 còn mở | E3✅ (chỉ hàm thuần) | Không. `fact_extraction.py` trong delta chỉ đổi đồng hồ và cách đọc usage. Không commit nào trong toàn lịch sử nhắc `#5582` |
| 3 | Retain lỗi giữa streaming → bản cũ bị xoá, bản mới dở, webhook `retain.completed` đã xếp hàng | E1 (hậu quả là E0) | Không chạm khối `producer_error`/webhook: grep diff orchestrator ra rỗng. 06a964dd2 chỉ đổi lựa chọn delta hay streaming khi retry |
| 4 | `consolidated_at` đóng dấu cả batch, kể cả action bị loại | E1 | Không chạm khối `stamp_ids` (grep diff ra rỗng). Delta thêm **triage** (0d8e58b3e); fact bị xếp "chatter" cũng đi qua cùng khối đóng dấu này (consolidator.py:2752) |
| 5 | Observation mồ côi tồn đọng vẫn được recall (#5419 mở) | E1🔎 | Không có commit nào nhắc |
| 6 | FK đa tenant thiếu ở schema 2+ | E3✅ | Delta thêm 2 migration (`b2429c65cc20`, `d4f7a2c9e1b5`); grep `invalidated` trong diff alembic ra 0 dòng |
| 7 | `ExcludeNoneRoute` không áp ở route nào | E3✅ | Không. `route_class` vẫn bị gán hai lần (http.py:5388 và 5428; số dòng đổi vì tệp dài thêm) |
| 8 | `run_consolidation` trả số 0 | E3✅ | Không đổi khoá đọc/trả (grep diff ra rỗng) |
| 9 | Mặc định mở: không auth, `0.0.0.0`, trace nguyên văn (#5544 mở) | E3🔎 | Không có commit nào nhắc #5544 |
| 10 | "4 chiến lược recall song song" là mô tả sai | E1🔎 | `retrieval.py` trong delta chỉ đổi đồng hồ |

Một số kết luận **khác** đã lỗi thời. Có cái do delta (#5371, #5448, #5228, một phần H-26). Có cái do audit bỏ sót ngay tại revision được audit (#5251). Danh sách đầy đủ ở `02_UPSTREAM_DELTA.md` §5.

## 4. Bảy bất biến SE 28 giữ lại (ADAPT)

Nguồn: SE 28 mục "Kết quả". Trạng thái phía agent-cli dưới đây là **lời khai của SE 28/26**; tôi không kiểm lại agent-cli (ngoài phạm vi).

| # | Bất biến (SE 28) | Bằng chứng Hindsight gốc | Trạng thái agent-cli theo SE 28 | Trạng thái overlay §7 tương ứng |
|---|---|---|---|---|
| 1 | Provenance dùng cho **bằng chứng**, không dùng cho memory; trace chỉ giữ metadata | H-36 (trace lưu nguyên văn) | sổ ghi LLM opt-in, có `llm_ledger_test.go` | VERIFIED phải có nguồn |
| 2 | Phạm vi (scope/tag) đi theo cả thao tác dẫn xuất | H-43, H-44, bài học #9 | `TestFilterDisabledToolsAppliesToInnerTools` | SCOPE |
| 3 | Không lưu bền secret nếu tránh được; phải có chính sách giữ | H-36, H-37, H-41 | sửa PRB-10; **PRB-49 còn mở** | DURABLE ≠ an toàn |
| 4 | Chỉ ack sau khi đã ghi bền | bài học #11 | **PRB-16 còn mở** | ACCEPTED ≠ DURABLE |
| 5 | Rỗng hợp lệ ≠ rỗng do lỗi; rỗng/hỏng không bao giờ thành thành công | H-08, bài học #1–2 | có test (tóm tắt bị cắt, loop detection, cờ GREEN) | EXTRACTED ≠ completed |
| 6 | Mọi việc có đường tới trạng thái cuối | H-01, bài học #6 | routine ABANDONED, đóng lời trả lời dở | worker recovery |
| 7 | Sửa một luật phải lan tới mọi bản sao của nó | bài học #10 | PRB-06, PRB-03 đã sửa | (xem 3f8fe203f ở tệp 02) |

SE 28 tự ghi: 5/7 có test ở worktree, 2 còn mở (PRB-16, PRB-49).

Delta 48 commit thêm chứng cứ cho bất biến 4, 5, 7 ở phía Hindsight (c8b7b3341, e1f43dfa0, 3f8fe203f). Không commit nào trong delta mâu thuẫn với một bất biến nào trong 7 cái.

## 5. Những gì bị bác hoặc hoãn, và vì sao

| Khái niệm Hindsight | Quyết định | Lý do ghi trong SE 28/29/30 |
|---|---|---|
| Memory Bank | REJECT (hôm nay) | SE 27 không tìm thấy lớp vấn đề nào ở agent-cli có bằng chứng đòi memory dài hạn; U-01 (chưa ai đo với LLM thật) |
| Retain: LLM trích fact từ transcript | REJECT | H-08/H-09 (0 fact mà `completed`), H-28 (vòng nhân bản), U-01 |
| Recall nhiều chiến lược | DEFER | U-02 (không có ablation), H-47, Q-13 (reranker chỉ tiếng Anh; tiếng Việt chưa kiểm) |
| Reflect | REJECT | H-26 (`based_on` rộng hơn thứ thật sự dùng), H-27 |
| Observation / consolidation | REJECT | H-17..H-24 |
| Mental model / knowledge page | REJECT | H-23, H-25 |
| Hook retain-ở-Stop / recall-ở-prompt | REJECT | H-28..H-31 |
| Hàng đợi async + worker, webhook | NOT_APPLICABLE | agent-cli không có worker nền cho memory; bài học vòng đời được ADAPT thành bất biến |
| Hiệu lực thời gian (`valid_from/to`) | DEFER | H-22; chỉ cần khi có memory dẫn xuất |
| Benchmark LoComo/LongMemEval/`sdebench` | REJECT làm bằng chứng | U-03, H-34 (E0) |
| Memlake (SE 40) | không tích hợp; giữ SQLite + M0 | số "47×", "100M <100 ms p99" do nhà cung cấp công bố, khác phần cứng, dữ liệu tổng hợp; chưa đo nghẽn truy hồi ở agent-cli |
| Phương án thắng (SE 29) | M0 + sửa kênh nạp context file; M1 (tệp bài học theo quy ước) chỉ dự bị | M0 đạt hai kịch bản E5 (s10, s13); lỗ đo được nằm ở kênh nạp |

Điều kiện mở lại (SE 28): kịch bản Daily B, I hoặc J thất bại ở M0 vì thiếu tri thức giữa phiên.

Delta không cung cấp bằng chứng nào làm đổi các quyết định trên: không có số đo LLM thật, không ablation. Delta **thêm** độ phức tạp ở đúng những phần đã bị REJECT. Ba ví dụ: reflect chế độ "decision-orchestrated" (198947912, +2.988 dòng), triage consolidation bằng decision model (0d8e58b3e), full refresh theo lịch cho mental model (ed369f9e9). Số eval mà upstream nêu trong các commit này (ví dụ 0433604c1: trang stale 10/16 → 3/16) là **E0** đối với chúng ta.

## 6. Dữ liệu thô của lần cào 1.534 issue còn trên đĩa không?

Audit §11.1 ghi dữ liệu thô nằm trong "thư mục scratchpad phiên `…/scratchpad/`": `issues.jsonl` (1.534 dòng), `issues_clustered.json`, `issue_commit_refs.json`, `analyze_issues.py`, `crawl_issues.py`, `archaeology/repro_pure.py`, `runtime-lab/*.py`, `runtime-lab/lab-run3.log`, `runtime-lab/verify-lead.log`. Audit tự cảnh báo đó là thư mục tạm.

**Kết luận [ĐO]: KHÔNG còn trên đĩa.**

| Lệnh | Kết quả |
|---|---|
| `find / -xdev \( -name issues.jsonl -o -name issues_clustered.json -o -name issue_commit_refs.json -o -name crawl_issues.py -o -name analyze_issues.py -o -name verify_lead.sh -o -name verify-lead.log -o -name lab-run3.log -o -name repro_pure.py \)` | 0 kết quả |
| `find /home/luannt -xdev \( -iname '*issue*commit*' -o -iname '*crawl*issue*' -o -iname 'issues*.jsonl' -o -iname '*hindsight*issue*' \)` | 0 kết quả |
| `find /tmp /home/luannt -maxdepth 6/7 -type d -name runtime-lab` / `-name archaeology` | 0 kết quả |
| `df -h -x tmpfs …` | chỉ có `/` (nvme0n1p2) và `/boot/efi`, nên `find / -xdev` đã phủ mọi ổ dữ liệu |
| `uptime -s` | `2026-10-19 12:51:32`. Mục cũ nhất trong `/tmp` có mtime `2026-10-19 12:51:39`; 26 thư mục phiên trong `/tmp/claude-1000/-home-luannt-application/` đều tạo sau giờ khởi động đó |
| `ls reports/system-evolution/evidence/` | không có tệp issue nào |

[SUY LUẬN] Máy khởi động lại ngày 19/10 đã xoá `/tmp`, nơi chứa scratchpad của phiên 16/10. Image docker `python:3.11` có thể còn, nhưng nó không chứa dữ liệu (dữ liệu nằm trên volume mount từ scratchpad). Tôi không chạy `docker images` để kiểm vì spec cấm docker.

Hệ quả:
- Không tái lập được các con số sau nếu không cào lại: 1.534 issue, bảng phân cụm §13.2, "136 issue họ silent", tỉ lệ `not_planned` 32/140, nhịp issue theo tháng.
- Bảng 59 issue (§13.4) vẫn **đối chiếu được một phần** qua `git log`, vì hash commit có thật trong clone. Trạng thái issue (open/closed/lý do đóng) thì không đối chiếu được.
- `issue-refs.tsv` (tệp 3 của đợt này) chỉ thay **một phần**: mọi số `#N` mà commit message nhắc tới (3.700 commit, 3.974 số khác nhau). Tệp không có trạng thái issue và không có issue nào chưa từng được commit nhắc.

## 7. Audit cũ bỏ sót gì (phát hiện lúc kiểm, không do delta)

- **#5251 đã có bản sửa ngay trong revision được audit.** Bản sửa là PR #5508, merge commit `d3f0001ed` ngày 2026-10-12, và [ĐO] `git merge-base --is-ancestor d3f0001ed e00154496` đúng. Audit §12.7 tả cursor của coding-agents (đệm `pending`, cờ `dirty`, cursor ở `tmpdir()`) mà không nhắc trường `ops`. Trường này kiểm operation đã được accept nhưng sau đó fail. SE 24 (H-29..H-31) cũng không nhắc. Grep `5251` trong audit và SE ra 0 kết quả (hit duy nhất là chuỗi "23477").
- **#2413 và #2347** không có trong audit hay SE. Cả hai đều đã đóng (xem IssueRecord ở tệp 02).

---

## Hợp đồng đầu ra (spec §63)

- **SCOPE:** kiểm lại audit Hindsight trước và các báo cáo SE suy ra từ nó; tìm dữ liệu thô của lần cào issue.
- **REVISION AUDITED:** audit cũ ở `e00154496`; đối chiếu với clone `5331da2f4`.
- **FILES/COMMITS/ISSUES INSPECTED:** `luannt-hindsight-1610.md` (đọc trọn 852 dòng); SE 24, 25, 26, 28, 29, 30, 40; `git ls-tree`/`git diff` trên poller, fact_extraction, orchestrator, consolidator, retrieval, http, alembic; commit `d3f0001ed`.
- **VERIFIED FACTS:** revision khớp; số đếm mốc khớp (631/79/111); dữ liệu thô không còn trên đĩa; delta không chạm 9/10 kết luận chính ở mức mã (riêng #4 có thêm một lối đóng dấu mới qua triage); audit bỏ sót bản sửa #5251.
- **INFERENCES:** dữ liệu thô mất do khởi động lại xoá `/tmp`.
- **UNKNOWNs:** các kết quả E3 có còn tái lập ở `5331da2f4` không (không chạy lại); trạng thái hiện tại của các issue OPEN mà không commit nào nhắc (#5419, #5519, #4130, #5497, #5544, #5638, #5589, #5434, #5574, #5590, #5442, #5433), vì không đọc được tracker; upstream sau `5331da2f4` (không fetch).
- **FAILURE MODES:** bằng chứng tạm (scratchpad) mất sau khi khởi động lại. Kết luận "issue OPEN" cũ dần mà không ai báo.
- **AGENT CLI RELEVANCE:** 7 bất biến ADAPT vẫn đứng; không có bằng chứng mới đổi REJECT/DEFER.
- **PROPOSED TESTS:** không có test mới ở tệp này (xem tệp 02 §7).
- **NO-CHANGE FINDINGS:** 9/10 kết luận chính không bị delta chạm ở mức mã; mọi quyết định REJECT/DEFER của SE 28 giữ nguyên.
