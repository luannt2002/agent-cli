# 24 — Sổ cái bằng chứng Hindsight (Phase 9.1)

Nguồn duy nhất: `/home/luannt/application/opensource/luannt-hindsight-1610.md` (audit 2026-10-16, Hindsight commit `e001544`, đọc trọn 852 dòng). Tệp này **không** chạy lại thí nghiệm nào của Hindsight; nó chép lại độ mạnh bằng chứng mà audit đã ghi, đổi sang thang chung của đợt tiến hoá, và **không nâng hạng** bằng chứng.

## Thang quy đổi

| Thang đợt này | Nghĩa | Nhãn trong audit Hindsight tương ứng |
|---|---|---|
| E0 | chỉ có lời khai (docs, README, issue, suy luận `[nghi]`) | `[nghi]`, số trong README, issue chưa lần ra code |
| E1 | đã đối chiếu mã nguồn | `✅`/`[TĨNH ✅]` (tác giả tự mở code) · `🔎`/`[TĨNH 🔎]` (agent đọc, tác giả **chưa** kiểm) → ghi **E1🔎** |
| E2 | test tồn tại | "FIXED_VERIFIED" ở §13.4 = có commit + test tên khớp, **test chưa được chạy** |
| E3 | chạy thật với LLM/embedding/rerank **stub** | `[CHẠY ✅]` (tác giả chạy lại) · `[CHẠY 🔎]` (agent chạy, tác giả đọc log) → **E3🔎** |
| E4 | chạy thật toàn phần ở local | không có mục nào: mọi lượt chạy đều dùng stub LLM |
| E5 | xác minh với LLM thật | **không có** (audit tự nói: không có API key) |
| E6 | benchmark | **không có** |
| E7 | xác minh trong hệ thống agent-cli | **không có** — chỉ đạt được bằng thí nghiệm của chính đợt này |

Quy ước khi cơ chế không phụ thuộc chất lượng LLM (khoá DB, FK, route, worker): vẫn ghi E3, kèm chữ "cơ chế không phụ thuộc LLM". Ghi như vậy để không có chỗ nâng hạng ngầm.

## Sổ cái

Cột cuối là **giả thuyết** về bề mặt tương tự ở agent-cli. Phase 10 sẽ kiểm, chưa phải kết luận.

### A. Việc dài / worker / khôi phục

| ID | Khẳng định (Hindsight) | Bằng chứng | Hạng | Bề mặt nghi ở agent-cli |
|---|---|---|---|---|
| H-01 | Worker chết, khởi động lại với **id khác** → việc kẹt `processing` vô hạn; client thấy `pending`, không lỗi; chỉ `decommission-worker` gỡ được. Maintainer: chủ ý (#2359, #3194) | E3b, log có, §14 | **E3🔎** (cơ chế không phụ thuộc LLM) | routine/loop bị `kill -9` giữa chừng; run của server khi server chết |
| H-02 | Khởi động lại **cùng id** → tự thu hồi, `retry_count=1`, xong sau 15,6 s | E3a | E3🔎 | — |
| H-03 | Ghi trạng thái cuối không kiểm `worker_id` (poller.py:931, 1062, 1117) → worker zombie có thể ghi đè | mã: TĨNH ✅; hậu quả: `[nghi]` | E1 (hậu quả E0) | hai tiến trình cùng ghi journal/report? |
| H-04 | Một `task_payload` JSON hỏng làm rollback mọi lần claim của cả schema (poller.py:700-706) | TĨNH ✅; lặp mãi: `[nghi]` | E1 | tệp định nghĩa/journal hỏng chặn routine? |
| H-05 | Chỉ một số loại việc có trần thời gian; import/export/clone/graph/vector/webhook không giới hạn | TĨNH 🔎 | E1🔎 | lệnh/hook không có timeout |
| H-06 | `_recover_batch_operations` reset mọi batch `processing` cũ hơn lúc khởi động, bất kể worker | `[nghi]` | E0 | — |
| H-07 | Semaphore LLM chung 32; consolidation và retain tranh nhau (#4130 mở) | TĨNH 🔎 + issue | E1🔎 | — |

### B. Thành công giả (transport ≠ semantic)

| ID | Khẳng định | Bằng chứng | Hạng | Bề mặt nghi ở agent-cli |
|---|---|---|---|---|
| H-08 | "0 fact mà vẫn `completed`": ≥4 lối vào, bịt từng lối; lối #5582 (dict thiếu khoá `facts`) **mở, tái lập ở HEAD** | chạy hàm thuần | **E3✅** | loop GREEN khi verify chạy 0 test; `run` rc=0 khi việc chưa xong |
| H-09 | Đường batch API: chunk lỗi → `fact_count=0`, operation `completed` (cờ chặn tắt mặc định, #2700) | TĨNH 🔎 + issue | E1🔎 | cờ an toàn tắt mặc định |
| H-10 | Lỗi một phần trong streaming: bản cũ bị xoá, bản mới dở, webhook `retain.completed` đã xếp hàng trước khi op `failed` | mã: TĨNH ✅ (ORCH:2822-2894, 3472); hậu quả chưa chạy | E1 (hậu quả E0) | báo cáo/sự kiện "xong" phát trước khi biết kết quả cuối |
| H-11 | Một chunk bị content-policy refusal → hỏng cả retain, không retry (#3690 PARTIAL) | TĨNH 🔎 + issue | E1🔎 | — |
| H-12 | Chunk < 500 ký tự mà vẫn `OutputTooLong` → bỏ chunk, một dòng warning | TĨNH 🔎 | E1🔎 | output bị cắt mà không báo |
| H-13 | Recall trả 200, **không cờ** khi graph timeout / reranker failover / analyzer lỗi / fact bị cắt theo token | TĨNH 🔎 | E1🔎 | kết quả suy giảm không có cờ |
| H-14 | Vòng retry lỗi kết nối của recall gần như mã chết (lỗi đã bị bọc `RuntimeError`) | TĨNH ✅ | E1 | — |
| H-15 | `run_consolidation` trả toàn số 0 dù DB có observation (lệch tên khoá) | E2 trong §14 | **E3✅** | số báo cáo lấy từ khoá sai |
| H-16 | `ExcludeNoneRoute` không áp ở route nào (100/100 là `UnknownParamsRoute`) | E1 trong §14 | **E3✅** (cơ chế không phụ thuộc LLM) | — |

### C. Tri thức dẫn xuất / consolidation / vòng đời

| ID | Khẳng định | Bằng chứng | Hạng | Bề mặt nghi ở agent-cli |
|---|---|---|---|---|
| H-17 | `consolidated_at` đóng dấu cả batch kể cả action bị loại (CON:2729); fact lẻ bị loại rơi khỏi hàng đợi vĩnh viễn | TĨNH ✅ (phần fact lẻ: 🔎) | E1 | dấu "đã xử lý" đặt khi không có gì mang nó |
| H-18 | Consolidation từng xoá sạch bank (#3876); sửa bằng một transaction | issue + commit + test tồn tại | E2 | — |
| H-19 | Lỗi ngoài LLM (recall/embed/dedup/DB) hỏng cả job, retry vô hạn, lấy cũ-nhất-trước → một fact độc chặn cả bank | TĨNH 🔎, `[nghi]` | E1🔎 / E0 | việc độc chặn việc khác |
| H-20 | Dedup chỉ chạy **sau** LLM chính; ngưỡng 0,97 tốn thêm một lời gọi LLM mỗi cặp | TĨNH 🔎 | E1🔎 | — |
| H-21 | Observation mồ côi: bản sửa chặn mồ côi mới, phần tồn vẫn được recall (#5419 mở); `source_memory_ids` là mảng, không FK | issue | E1🔎 | — |
| H-22 | Không có `valid_from/valid_to/superseded_by`; fact cũ và mới cùng tồn tại; thay thế chỉ ở tầng observation do LLM quyết | TĨNH 🔎 (grep) | E1🔎 | ghi chú/bộ nhớ cũ thắng sự thật hiện tại |
| H-23 | Xoá nguồn → xoá observation có nguồn bị đụng; mental model refresh tay vẫn nêu fact đã xoá | TĨNH 🔎 | E1🔎 | tri thức dẫn xuất sống lâu hơn nguồn |
| H-24 | Sửa một fact → xoá cứng observation dẫn xuất, không vết (#5638 mở) | issue + TĨNH 🔎 | E1🔎 | — |
| H-25 | Delta refresh để lại câu cũ mâu thuẫn (#5228); ngân sách token chỉ là gợi ý trong prompt (#5574) | issue | E0 | giới hạn chỉ nằm trong prompt |
| H-26 | Reflect không gắn nguồn từng câu; model không trích id thì `based_on` = **mọi** memory đã recall (ME:15738) | TĨNH 🔎 | E1🔎 | "bằng chứng" rộng hơn thứ thật sự dùng |
| H-27 | Tool `recall` trong reflect khác recall HTTP (query do LLM viết lại, fact type, max_tokens, `all_strict`) (#2670) | TĨNH 🔎 + issue | E1🔎 | — |

### D. Tích hợp coding agent (gần agent-cli nhất)

| ID | Khẳng định | Bằng chứng | Hạng | Bề mặt nghi ở agent-cli |
|---|---|---|---|---|
| H-28 | Vòng nhân bản (#826): dòng ghi công `From Hindsight memory — <facts>` mà plugin **yêu cầu** agent viết không bị cắt → memory được retain ngược lại vào bank. Không test | TĨNH ✅ (`core/inject.ts:103`, `MEMORY_TAG_RE`) | E1 | chỉ áp nếu agent-cli thêm retain cuối lượt |
| H-29 | Cursor ở `tmpdir()` mất khi reboot → replace toàn bộ; Stop cuối gặp server chết thì mất đuôi | TĨNH 🔎 | E1🔎 | trạng thái tạm ở /tmp |
| H-30 | Harness không gửi `sessionId` → mọi session chung `conversation:no-session`, replace lẫn nhau | `[nghi]` | E0 | khoá tài liệu không duy nhất |
| H-31 | Hook lỗi đều fail-open; Stop lỗi chỉ ghi tệp diag, người dùng không thấy | TĨNH 🔎 | E1🔎 | Stop hook fail-open (agent-cli có sẵn, audit C-01) |
| H-32 | Một bank cho mỗi git project (`coding-agent::{gitProject}`), mọi agent và mọi nhánh dùng chung | TĨNH 🔎 | E1🔎 | phạm vi nhánh/worktree (MEM-012/013) |
| H-33 | Prompt reflect cấm văn mệnh lệnh vì từng biến chuyện đã làm thành chỉ thị | TĨNH 🔎 | E1🔎 | memory dạng mệnh lệnh = prompt injection |
| H-34 | `sdebench`: corrections/task Claude Code 0,85 → 0,36 | chỉ README | **E0** | — |

### E. An ninh / dữ liệu lưu

| ID | Khẳng định | Bằng chứng | Hạng | Bề mặt nghi ở agent-cli |
|---|---|---|---|---|
| H-35 | Mặc định không auth, bind `0.0.0.0` (#5544 mở) | E4 trong §14 | E3🔎 (cơ chế không phụ thuộc LLM) | transport của server agent-cli |
| H-36 | `llm_requests` lưu **nguyên văn** prompt/response (trace bật mặc định) | E5 trong §14 | E3🔎 | DB tin nhắn, `crush.log` |
| H-37 | `async_operations.task_payload` giữ nguyên văn nội dung; retention = 0 → giữ (mới quan sát 6 phút) | E6 trong §14 | E3🔎 (phần "vĩnh viễn" E0) | journal loop, báo cáo routine |
| H-38 | `MCP_AUTH_TOKEN` so bằng `!=`, bỏ qua tenant extension | `[nghi]` | E0 | — |
| H-39 | `hindsight fs` lưu `api_token` plaintext | `[nghi]` | E0 | — |
| H-40 | Đọc bank config lỗi → âm thầm dùng config toàn cục | `[nghi]` | E0 | cấu hình lỗi rơi về mặc định lặng lẽ (audit A-02) |
| H-41 | Memory Defense OSS chỉ có detector `sensitive_data` (regex), không quét lại dữ liệu cũ | TĨNH 🔎 | E1🔎 | lọc secret trước khi lưu |
| H-42 | FK `invalidated_mu_document_fkey` chỉ có ở schema migrate đầu tiên | E7 trong §14 | **E3✅** (cơ chế không phụ thuộc LLM) | — |
| H-43 | `RequestContext.allowed_bank_ids` khai báo nhưng không thực thi (#5448 mở) | issue | E0 | ranh giới khai báo mà không ép |
| H-44 | `tags_match` gõ sai → nới phạm vi thành "any" (#5371 mở, "tái lập ở HEAD") | issue; cách tái lập không ghi lại | E1🔎 | đầu vào sai ở biên tin cậy bị nới thay vì từ chối |
| H-45 | `DELETE /memories` bỏ qua body, xoá cả bank (#4337 not_planned) | issue | E0 | — |
| H-46 | Thiếu `DATABASE_URL` → âm thầm chạy pg0 nhúng (#1202, #3728) | `DEFAULT_DATABASE_URL="pg0"`: grep ✅; mất dữ liệu: E0 | E1 / E0 | thiếu cấu hình → mặc định lặng lẽ |

### F. Retrieval

| ID | Khẳng định | Bằng chứng | Hạng | Ghi chú |
|---|---|---|---|---|
| H-47 | "4 chiến lược song song" là sai: dense+BM25 một câu UNION, temporal tuần tự, chỉ graph song song | TĨNH 🔎 | E1🔎 | — |
| H-48 | Câu fallback của graph không có timeout, không có test | TĨNH 🔎 | E1🔎 | — |
| H-49 | Unit không có ngày được temporal = 0,5 → có thể vượt fact đúng cửa sổ ở mép | `[nghi]` | E0 | — |
| H-50 | Reranker mặc định chỉ tiếng Anh, hỏng recall tiếng Trung (#5519 mở) | issue | E0 | liên quan **tiếng Việt**, chưa kiểm |

### G. Bài học lịch sử issue (11 invariant, mỗi cái ≥ 2 issue làm chứng)

Dữ liệu: 1.534/1.534 issue cào được; 59 issue lần tới commit/code/test; 40 issue đọc comment. Hạng E1: số issue và commit có thật trong `git log`, nhưng test chưa chạy. Nguyên văn ở audit §13.8:

1. Lỗi trích xuất phải nổi lên; không bao giờ `[]` rồi `completed`.
2. "Rỗng hợp lệ" phải phân biệt được với "rỗng do lỗi" trên status, metadata, sự kiện, metric.
3. Dấu "đã xử lý" commit **cùng transaction** với write nó đại diện.
4. Xoá/đổi scope nguồn → quét dữ liệu dẫn xuất ở điểm nghẽn chung, kèm đường sửa phần tồn.
5. Ghi dựa trên snapshot đọc trước lời gọi LLM phải kiểm lại trạng thái hiện tại (CAS, địa chỉ theo id).
6. Mọi operation có đường tới trạng thái cuối; không dòng `pending/processing` nào vĩnh viễn.
7. Phân loại lỗi: tất định → fail ngay; tạm thời → retry/defer; không đảo.
8. Đầu vào sai ở biên tin cậy bị **từ chối**, không bỏ qua hay nới phạm vi.
9. Phạm vi (tag/scope) đi theo cả dữ liệu dẫn xuất; mặc định strict/any phải lộ ra.
10. Sửa một quy tắc phải lan tới mọi bản sao của nó.
11. Trạng thái phía client không bền → chỉ ack sau khi ghi thành công.

Chỉ dấu định lượng (E1, phân cụm thô theo tiêu đề): họ tiêu đề "silent/swallow/drops" khớp 136 issue; cụm consolidation có tỉ lệ `not_planned` 32/140 ≈ 23%; cụm integration chiếm 390/1.534 ≈ 25% số issue.

## Điều KHÔNG được suy ra từ sổ cái này

- Không mục nào là E5/E6. Chất lượng trích fact, consolidation và reflect với LLM thật **chưa biết**.
- Test hồi quy "FIXED_VERIFIED" mới được xác nhận là **tồn tại**, chưa chạy.
- Số benchmark trong README của Hindsight (vd `sdebench`) là E0.
- Mọi cột "bề mặt nghi ở agent-cli" là giả thuyết. Kết luận áp dụng hay không nằm ở `26_HINDSIGHT_FAILURE_INVARIANTS.md` (Phase 10).
- Audit có hạn dùng: Hindsight nhận khoảng 270 issue/tháng; các mục OPEN (#5582, #5419, #5371, #5544, #4130, #5497) cần kiểm lại sau vài tuần.
