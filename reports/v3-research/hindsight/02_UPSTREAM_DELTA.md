# 02 — Delta upstream Hindsight `e00154496..5331da2f4`

Ngày: 2026-10-19. Chỉ đọc mã trong clone `/home/luannt/application/opensource/v3-src/hindsight`. Không chạy test nào.
Ký hiệu: **[ĐO]** = lệnh tôi vừa chạy. **[TĨNH ✅]** = tôi tự mở mã ở HEAD hoặc đọc diff. **[WEB]** = trang HTML GitHub đọc qua WebFetch ngày 19/10. **[SUY LUẬN]** = suy luận, có ghi cách kiểm. "Test" ở tệp này chỉ có nghĩa là test **tồn tại** trong diff; tôi không chạy test nào.

## 0. Tóm tắt

- 48 commit, đều ngày 2026-10-16, trong khoảng từ 10:47 tới 21:58 giờ +0200. 47 commit của Nicolò Boschi, 1 commit blog của Ben. Tổng 327 tệp, +18.846/−2.603 dòng [ĐO `git diff --shortstat`].
- **0 tệp** thuộc `hindsight-integrations/`, tức tích hợp coding-agents không đổi [ĐO `git diff --stat … -- hindsight-integrations/` ra rỗng]. Ba commit "codex/claude-code" (074e3fe60, dcc3ee421, c7b17f0ad) là **provider LLM phía server**: server gọi CLI hoặc tài khoản Codex/Claude Code để trích fact, consolidation, reflect. Chúng không đụng tới plugin hook của coding agent.
- Worker và hàng đợi không đổi về ngữ nghĩa: `poller.py` chỉ đổi đồng hồ. Vẫn không có lease hay heartbeat.
- Hai issue mà audit cũ ghi "OPEN, tái lập ở HEAD" hoặc "OPEN" nay đã có bản sửa: **#5371** (89594d22d) và **#5448** (c7acb488a). #5228 được giảm nhẹ bằng prompt (0433604c1).
- **#5251** đã được sửa **trước** revision mà audit cũ dùng (PR #5508, `d3f0001ed`, 12/10), và audit cũ bỏ sót bản sửa đó. Ở HEAD bản sửa vẫn nguyên nhưng **còn hở** (§6.1).
- Họ bug lặp lộ rõ trong delta: hook của host lọt vào tiến trình con LLM (#1751 tháng 5 → #5542 tháng 10); bộ đọc structured output cho delta-op được sửa lần hai (#3820); một PR bị revert rồi tách ra merge lại (#5513).

## 1. Cách đo

```
cd /home/luannt/application/opensource/v3-src/hindsight
git rev-list --count e00154496..5331da2f4                      # 48
git log --reverse --format='%h|%ci|%an|%s' e00154496..5331da2f4
git show --stat <commit> ; git show <commit> -- <path>          # đọc từng diff
git diff --stat e00154496..5331da2f4 -- hindsight-integrations/ # rỗng
git diff e00154496..5331da2f4 -- <tệp>                          # chạm/không chạm vùng mã
git log --grep / -S                                             # nhắc issue (xem issue-refs.tsv)
```
Lệnh `git log -S'#5251'` trên toàn repo mất hơn 120 s; tôi cho chạy nền rồi đọc kết quả, không có lệnh nào ghi vào repo.

## 2. Phân nhóm 48 commit theo hệ con

Mỗi commit được xếp vào đúng một nhóm chính. Cột "BB" đánh dấu commit liên quan tới bất biến của overlay §7 và có mục riêng ở §3.

| Nhóm | Số | Commit | BB |
|---|---|---|---|
| **Retain** | 6 | `c8b7b3341` append async thiếu document_id bị từ chối lúc submit (#5415) · `d7beb8e4b` thu hồi attachment khi bị kiểm write-scope từ chối (#5416) · `06a964dd2` resume tài liệu quá cỡ viết dở theo từng slice (#5485) · `dcdcf6bd5` giữ `event_date` "unset" qua reprocess/edit (#5473) · `e1f43dfa0` upload hiện tới khi retain mà nó xếp hàng xong (#5310) · `e92bf3543` chuyển connector LlamaParse sang API v2 (#5447/#5513) | c8b, d7b, 06a, e1f |
| **Consolidation** | 2 | `0d8e58b3e` triage bằng decision model trước lời gọi LLM (feat) · `b991f96fb` probe bank cần consolidation mà không seq-scan (#5425; phần merge lại của PR #5513 đã bị revert) | 0d8, b99 |
| **Reflect** | 2 | `198947912` reflect "decision-orchestrated"/fast mode (feat, +2.988 dòng) · `45616ffe3` fact đọc qua `expand` được giữ trong citation/`based_on` (#5474) | 45616 |
| **Recall** | 2 | `c4c572c40` tool MCP recall trả được chunk/entity/source fact (#5470) · `315182acb` báo store reranker mà recall dùng; export id fact của store-owned | — |
| **Scope / tag / quyền / biên tin cậy** | 5 | `89594d22d` từ chối `tags_match` lạ thay vì nới thành "any" (#5371) · `3f8fe203f` một luật lọc tag cho knowledge base, entity, control plane (#5417) · `c7acb488a` vá lỗ bank-validator (#5448) · `d8dd624b3` PATCH `observation_scopes` cho document đã có (#5309) · `b5ed4c406` kiểm bank id khi import, giữ tiền tố `_` (#5540) | cả 5 |
| **Mental models** | 4 | `9e9384107` nhận mảng delta-op mà provider đã parse (#3820) · `0433604c1` delta refresh sửa giá trị đã đổi ở mọi chỗ trên trang (#5228) · `f6294cd25` nhiệt độ riêng cho lời gọi delta/retraction (#5500) · `ed369f9e9` full refresh theo lịch cho model delta (#4340) | 9e9, 043, ed3 |
| **Documents API** | 2 | `3234aca57` tìm theo nội dung document và báo trường nào khớp (#5214) · `e73381c63` giới hạn tìm kiếm và báo khi tổng chỉ là cận dưới | e73 |
| **Transfer / import / export** | 4 | `8d197a582` extension metering thấy và chặn được import · `4b5f045c7` remap evidence của mental model khi import (#5298) · `e661e0dff` import bank theo lô có giới hạn (#5418) · `40cebcb46` import archive không giữ cả archive trong RAM (#5524) | 4b5 |
| **Worker / vận hành / quan sát** | 3 | `29698acb6` đo thời gian bằng đồng hồ đơn điệu, ruff cấm `time.time()` (#5445) · `3d169f5c2` giữ cached/thinking token trong tổng reflect/retain (#5487) · `11cf2c1a4` không nạp module nặng lúc khởi động (#5479) | 296 |
| **Provider qua tài khoản coding agent (claude-code / codex / chatgpt)** | 4 | `074e3fe60` codex: model mặc định dùng được với tài khoản ChatGPT, bớt tốn quota · `c7b17f0ad` claude-code: không để hook của host lọt vào tiến trình con LLM (#5542) · `dcc3ee421` báo token thật cho Codex/Claude Code/Anthropic (#5541) · `e00ca6d3b` provider `chatgpt` qua Sign in with ChatGPT | c7b, dcc, 074 |
| **Provider khác** | 4 | `a23e4b559` service tier của OpenAI (#5439) · `3ca0fe110` cảnh báo sentence-transformers (#5446) · `8c5fabf4d` Anthropic Workload Identity Federation (#5502) · `2ecc73cc1` NVIDIA NIM reranker + embeddings (#5551) | — |
| **Control plane** | 5 | `e6620f6bc` 401 của dataplane không đẩy session hợp lệ về login (#5452) · `f4da88255` tải trang/mental model dạng PDF/MD (#5457) · `87fc9d5ae` nút mới/cũ trên timeline · `643568791` retry upload hỏng và retry mọi operation hỏng · `a1dd98a1f` tab Memories chuyển ngay trên bank lớn (#5456/#5548) | — |
| **Helm / deploy / CI** | 3 | `a578c4973` thôi publish plugin hermes lên PyPI · `f8d9683f5` Helm `extraObjects` · `68f9c425a` Helm `revisionHistoryLimit` + Gateway API HTTPRoute | — |
| **Khác** | 2 | `e1a630d5c` embed: áp `--profile` khi chạy qua `python -m` (#5543) · `5331da2f4` blog Vercel Eve | — |
| **Tích hợp coding-agents (`hindsight-integrations/`)** | **0** | — | — |

Tổng: 6+2+2+2+5+4+2+4+3+4+4+5+3+2 = 48.

## 3. Commit then chốt: vấn đề, cách sửa, test, phân loại

Phân loại theo overlay §18: LOCAL_PATCH (vá một chỗ), SHARED_ABSTRACTION (gom về một chủ chung), ARCHITECTURAL_FIX (đổi cấu trúc, hoặc chặn cả một lớp lỗi bằng cổng máy).

### 3.1 `06a964dd2` — resume tài liệu quá cỡ viết dở (#5485) — DURABLE ≠ EXTRACTED, partial success, worker recovery

- **Vấn đề** [TĨNH ✅ diff + orchestrator.py:2500-2570]: một document quá cỡ được cắt thành nhiều slice. Ngay slice đầu, hàng `documents` đã lưu `original_text` = **toàn bộ** body (#1838) cùng hash của toàn body. Nếu retain bị ngắt giữa chừng, DB có một document "mang hash của bản đầy đủ" nhưng mới có một phần chunk. Khi retry cùng body, slice 1 đi đường **delta**: delta diff cả document rồi trích **mọi** chunk chưa ghi trong **một** bước, và chỉ commit ở cuối. Hệ quả là hàng trăm lời gọi LLM dồn vào hàng đợi trước mọi bank khác, và nếu lần retry cũng bị ngắt thì mất hết.
- **Sửa**: nếu hash đã lưu bằng hash body và vẫn còn chunk khác thì delta trả `None` để nhường cho đường streaming (orchestrator.py:3895). Đường streaming có `recovery_chunk_hashes` để bỏ qua các chunk đã commit, nên nó resume theo từng slice.
- **Test**: `tests/test_delta_oversized_replacement.py::test_recovering_partially_written_oversized_document_resumes_slice_by_slice`. Test dùng spy monkeypatch ném "crash" sau 3 lời gọi, rồi kiểm lần retry gọi trích từng chunk một và phủ mọi chunk còn thiếu. Test cần DB thật (fixture `memory`).
- **Phân loại**: LOCAL_PATCH. Một điều kiện chặn trong một hàm, dựa trên probe khôi phục có sẵn; không thêm trạng thái.
- **Còn hở** [TĨNH ✅]: (a) trạng thái "viết dở" không được lưu tường minh. Nó được **suy ra** từ "hash bằng nhau mà chunk khác nhau". (b) Probe khôi phục bọc `except Exception: pass` (orchestrator.py:2570). Probe lỗi thì xử lý lại mọi chunk: tốn tiền nhưng không mất dữ liệu. (c) [SUY LUẬN] trong lúc viết dở, reader thấy một document có `original_text` đầy đủ nhưng chỉ một phần fact, tức tìm được một phần mà không có cờ nào báo. Cách kiểm: ngắt retain ở slice k, gọi recall và `GET /documents/{id}`.

### 3.2 `c7b17f0ad` — hook của host lọt vào tiến trình con LLM (#5542; cùng họ với #1751) — host hook interference

- **Vấn đề** [TĨNH ✅ diff]: provider `claude-code` sinh tiến trình con `claude` để gọi LLM. Bản sửa #1751 (`bcae23d9f`, 2026-05-28) chỉ đặt `CLAUDE_CONFIG_DIR` vào một thư mục tạm, và biến này chỉ chuyển **user** settings. CLI vẫn nạp `./.claude/settings.json` ở cwd như **project** settings. Daemon chạy từ `$HOME` thì cwd chính là `$HOME`, nên tệp đó chính là `~/.claude/settings.json` có hook của host (Stop hook của plugin Hindsight). Kết quả: lời gọi LLM để retain/reflect/consolidation lại kích Stop hook, và hook đó retain transcript của tiến trình con ngược vào bank. Vòng lặp #1751 quay lại.
- **Sửa**: hai đường gọi (`call` và `call_with_tools`) đặt `cwd=<thư mục cô lập>` và `setting_sources=[]` (claude_code_llm.py:419-426, 754-755).
- **Test**: `tests/test_claude_code_llm_isolation.py::_assert_isolation` kiểm `options.cwd` và `options.setting_sources` trên SDK giả (`_FakeOptions`). Test chỉ chứng minh **tham số được truyền**, không chứng minh CLI thật bỏ qua settings. Lượt review ghi 18 test anh em đỏ vì tham số mới, và đã sửa fake của chúng.
- **Phân loại**: LOCAL_PATCH. Ở HEAD có **ba** cơ chế chống đệ quy độc lập nhau, không có chủ chung: (1) claude_code_llm (env + cwd + setting_sources); (2) `cursor_llm.py` tự cô lập bằng workspace rỗng và `CURSOR_CONFIG_DIR` (comment cursor_llm.py:67-85 viện dẫn #1751 và ghi hook headless "không tái lập hôm nay"); (3) plugin coding-agents dùng biến `HINDSIGHT_DISABLE_HOOKS` cho phiên survey headless của chính nó (retain-hook.ts:186).
- **Lặp**: cùng một lớp lỗi dưới hai số issue khác nhau, cách nhau khoảng 4,5 tháng (#1751 ngày 2026-05-28 → #5542 ngày 2026-10-16). Cột `LAP` trong `issue-refs.tsv` không bắt được ca này vì message của c7b17f0ad không nhắc #1751.

### 3.3 `89594d22d` — `tags_match` lạ bị nới thành "any" (#5371) — scope/tag filter

- **Vấn đề** [TĨNH ✅ diff]: MCP recall/reflect và `GET /documents`, `/graph` nhận `tags_match: str`. Parser dùng chung `_parse_tags_match` gặp giá trị lạ thì rơi về `"any"`, tức nhận cả dòng không tag. Một lỗi gõ như `any-strict` âm thầm **nới** phạm vi.
- **Sửa**: 4 chữ ký MCP và 2 route đổi sang kiểu `TagsMatch` (Literal), nên giá trị lạ bị từ chối khi validate. Parser nay `raise ValueError` (engine/search/tags.py:74-89).
- **Test**: `test_tags_visibility.py::test_parse_tags_match_rejects_unknown_mode` (unit), cộng `test_mcp_tools.py` và `test_graph_filtering.py`.
- **Phân loại**: sửa tại parser chung (SHARED_ABSTRACTION đã có sẵn) + LOCAL typing ở 6 lối vào.
- **Còn hở** [TĨNH ✅]: `list_mental_models` (memory_engine.py:16888-16894) có parser **inline riêng**, nhánh `else: # any`. Qua HTTP chỉ nhận `Literal["any","all","exact"]` (http.py:7050), nên giá trị lạ bị FastAPI chặn; lời gọi qua Python API thì không được bảo vệ. Mặc định của MCP recall vẫn là `"any"` (gồm dòng không tag). Control plane bỏ lặng `tags_match` lạ và rơi về mặc định dataplane `any_strict`, tức **hẹp** đi chứ không rộng ra.

### 3.4 `3f8fe203f` — một luật lọc tag (#5417) — scope/tag, "nhiều resolver"

- **Vấn đề** [TĨNH ✅ diff]: cùng một luật có nhiều bản sao, và các bản lệch nhau:
  - hai hàm "bộ lọc có hiệu lực không" (`source_scope.tag_filter_is_active`, `search.tags.tag_filter_active`);
  - hai parser `tag_groups`: knowledge base trả 422 và bỏ qua giá trị rỗng, entity trả 400 và từ chối giá trị rỗng;
  - hai bản SQL "entity của các unit" (`curation.py`, `graph.py`);
  - proxy control plane chuyển tiếp `tags` nguyên văn.

  Hậu quả: dưới bộ lọc, entity mà chỉ observation nhắc tới thì **biến mất**.
- **Sửa**: xoá các bản sao; dùng một decoder `_decode_tag_groups_query` (http.py:274); một rowset `visible_entity_mentions_sql` có kế thừa observation → source cho list/detail/graph. Seam của store chỉ nhận tham số tag khi bộ lọc thật sự có hiệu lực: store cũ vẫn chạy (không lọc), còn store bị yêu cầu lọc mà không lọc được thì `TypeError` ồn ào thay vì trả dòng ngoài phạm vi (base.py:258-265).
- **Test**: `test_entity_tag_filter.py`, `test_knowledge_base.py`, `test_source_text_tag_scope.py`, `tests/lib/tag-filter-params.test.ts` (control plane), system story `test_71_entity_tag_scope.py`.
- **Phân loại**: SHARED_ABSTRACTION.
- **So với claude-mem #3608** (chỉ theo mô tả ở overlay §6; tôi chưa đọc claude-mem): cùng hình dạng, tức nhiều đường tự tính một luật rồi trôi lệch nhau. Hindsight giải bằng cách **xoá bản sao và cho mọi caller đi qua một hàm có sẵn**, không dựng thêm một tầng resolver mới.
- **Còn hở** [TĨNH ✅]: luật vẫn chưa thật sự "một". `KnowledgeTagFilter.active` (memory_engine.py:1800) viết lại cùng biểu thức inline, hôm nay cùng ngữ nghĩa; `list_mental_models` vẫn có parser riêng (§3.3). Họ lỗi tag-scope ra 3 bản sửa trong 15 ngày: #5030 (`c71cd0437`, 01/10) → #5371 → #5417.

### 3.5 `c8b7b3341` — accept một yêu cầu không bao giờ thành công (#5415) — ACCEPTED ≠ hợp lệ

- **Vấn đề**: đường đồng bộ kiểm "append phải có `document_id`" và "append cần `store_document_text`", đường async thì không. API trả 200 + operation, rồi worker fail sau vài phút retry.
- **Sửa**: một hàm `_validate_retain_update_modes` cho cả hai đường (memory_engine.py:22663; gọi ở 6667 và 22758). `InvalidRetainRequestError` (một `ValueError`, HTTP 400) được xếp non-retryable (memory_engine.py:1603).
- **Test**: `test_retain_append_mode.py` và system `test_12_append_document.py::test_an_append_without_a_document_is_refused_up_front` (tham số hoá cả hai đường; kiểm 400 và `operations.total == 0`).
- **Phân loại**: SHARED_ABSTRACTION.

### 3.6 `e1f43dfa0` — "completed" của bước 1 ≠ kết quả của chuỗi (#5310) — ACCEPTED ≠ EXTRACTED

- **Vấn đề**: upload chạy `file_convert_retain`. Bước này xếp hàng một retain riêng rồi tự đánh `completed`. UI chỉ đọc bước chuyển đổi, nên khi trích xuất fail, upload biến mất mà không để lại document hay dòng failed nào.
- **Sửa**: retain con mang `document_id`/`original_filename` của upload (memory_engine.py:4293); UI lấy trạng thái từ retain con.
- **Test**: system `test_63_file_upload_failures.py` (theo message: fail trên main), `pending-uploads.test.ts`.
- **Phân loại**: LOCAL_PATCH (nối metadata + UI).

### 3.7 `d7beb8e4b` — ghi trước khi kiểm, bị từ chối thì sót blob (#5416) — partial success

- **Vấn đề**: attachment được lưu ở ingress **trước** mọi kiểm tra. Chỉ lời từ chối của validator mới thu hồi; lời từ chối của kiểm write-scope ngay sau đó trả 403 mà giữ blob mồ côi, vẫn fetch được. Ngoài ra, lời từ chối ở một nhóm strategy sau từng thu hồi luôn attachment của nhóm đã ghi xong.
- **Sửa**: một `try` phủ cả validator lẫn `_check_retain_writes` trên cả hai đường; thu hồi chỉ trong phạm vi document của lời gọi bị từ chối.
- **Test**: unit `test_retain_validator_attachments.py`, system `test_85_refused_write_keeps_no_attachment.py`.
- **Phân loại**: LOCAL_PATCH. Đây là ví dụ cho bài học #10 của audit cũ: một bước bù trừ phải có ở **mọi** nhánh từ chối.

### 3.8 `0d8e58b3e` — triage consolidation bằng decision model (feat)

- **Thay đổi** [TĨNH ✅]: trước LLM chính có thêm một lời gọi decision model, phân fact thành "restates" (thành UPDATE giữ nguyên văn bản), "new" (CREATE nguyên câu) hoặc "chatter" (bỏ). Tính năng **bật mặc định** khi có cấu hình decision model; mỗi bank tắt bằng `consolidation_use_decision_model`. Một request triage lỗi thì fact đó được chuyển cho LLM (triage.py:266-272).
- **Liên quan audit cũ H-17**: fact "chatter" trả `skipped/triage_chatter` (consolidator.py:2894). `stamp_ids` vẫn là **cả batch** khi LLM không fail (consolidator.py:2752), nên fact chatter bị đóng dấu `consolidated_at` và không bao giờ được xét lại. Một phân loại sai của decision model là một lối mất fact khỏi tầng observation mà không báo [SUY LUẬN về hệ quả; cách kiểm: stub decision model trả chatter cho một fact có nội dung, rồi đếm observation].
- **Test**: `test_consolidation_triage.py`. **Phân loại**: không áp (tính năng). Tính năng này thêm một đường provider (decision model, `HINDSIGHT_API_DECISION_MODEL_*`) vào consolidation và reflect.

### 3.9 `9e9384107` — structured output: lần sửa thứ hai cho cùng một luật (#3820)

- **Vấn đề**: lần sửa trước (`7fe6a5e79`, 2026-08-31) cho bộ đọc nhận "mảng delta-op ở cấp cao nhất" dạng **văn bản**. Provider OpenAI-compatible với `response_format` trả JSON **đã parse**; một list đã parse rơi vào `(raw or "").strip()` và ném `AttributeError`. Nhánh "hỏi lại" không bắt lỗi này, nên refresh kết thúc ở `delta_ops_failed` mà không có lượt retry sửa sai.
- **Sửa**: list đã parse được validate như mảng dạng văn bản; mọi trả lời không phải văn bản khác ném `TypeError` để retry vẫn áp dụng.
- **Test**: `test_delta_operation_parse.py`. **Phân loại**: LOCAL_PATCH. Đây là bài học #10 lặp lại: luật được sửa cho một **biểu diễn** mà quên biểu diễn kia.

### 3.10 `074e3fe60` và `dcc3ee421` — provider codex / claude-code

- `074e3fe60`: model mặc định `gpt-5.4-mini` bị backend Codex từ chối với tài khoản ChatGPT (HTTP 400), nên bản triển khai không cấu hình model **fail mọi lời gọi**. Mặc định đổi thành `gpt-5.6-luna` (config.py, `PROVIDER_DEFAULT_MODELS`). `prompt_cache_key` trước là `uuid4` mới cho **mỗi** request (luôn trượt cache); nay ổn định theo operation. Bỏ yêu cầu reasoning summary và `encrypted_content` vì không ai đọc mà vẫn tốn quota. Test `test_codex_reasoning_effort.py` (mock). LOCAL_PATCH.
- `dcc3ee421` (#5541): số token của Codex và Claude Code trước đây là **ước lượng ký tự/4** (Codex còn trả 0) nhưng được ghi vào metric/trace như số thật. Nay đọc usage thật từ SSE `response.completed` hoặc `ResultMessage` của SDK. Anthropic cộng cache read/write vào `input_tokens`. Test là các tệp `test_codex_*`, `test_claude_code_*`, `test_anthropic_*` (mock). LOCAL_PATCH theo từng provider.
- **Còn hở** [TĨNH ✅]: khi không có khối usage, cả hai vẫn rơi về ước lượng ký tự/4, và `TokenUsage` (response_models.py:108-114) **không có cờ** "ước lượng". Ngược lại, `e73381c63` trong cùng delta lại thêm cờ `total_is_estimate` cho tổng tìm kiếm document bị cắt. Một delta mà hai chính sách khác nhau cho cùng câu hỏi "số này là đo hay ước lượng".

### 3.11 Các commit then chốt khác

| Commit | Vấn đề → sửa | Test | Phân loại | Bất biến |
|---|---|---|---|---|
| `45616ffe3` (#5474) | fact đọc qua `expand` không được coi là citable. Còn 0 citation hợp lệ thì `based_on` rơi về **mọi** fact đã recall, ghi công cả fact không dùng. Nay `returned_memories()` là một hàm cho 3 reader (allowlist citation, dựng `based_on`, đếm retrieval của refresh) | system `test_47_expanded_source_provenance.py`, unit `test_reflect_agent.py` | SHARED_ABSTRACTION | VERIFIED/provenance. **Còn**: nhánh "rỗng thì lấy tất cả" vẫn ở memory_engine.py ~15933-15936 (comment "or all if none specified") [TĨNH ✅] |
| `d8dd624b3` (#5309) | đổi `observation_scopes` của document đã có. Chỉ unit đổi scope thật mới bị xếp lại consolidation (luật `write_scopes_changed`, #5593, dùng chung với PATCH tags); đóng dấu unit đã đổi để consolidation đang chạy bỏ kết quả cũ (#4831) | `test_document_observation_scopes_patch.py`, `test_observation_invalidation.py`, `test_tag_scope.py` | SHARED_ABSTRACTION | scope đi theo dữ liệu dẫn xuất; CURRENT |
| `c7acb488a` (#5448) | `RequestContext.allowed_bank_ids` khai báo mà không ai đọc → **xoá**; `dry-run-extract` đi qua `validate_bank_read`; lời từ chối của validator giữ mã trạng thái thay vì thành 500 | `test_bank_validator_coverage.py` quét **mọi** route per-bank với validator từ chối hết | SHARED_ABSTRACTION + test quét cả lớp | ranh giới khai báo phải được ép |
| `29698acb6` (#5445) | `time.time()` lùi (NTP) làm thời lượng âm, OTel bỏ mẫu. Đổi sang `perf_counter`; **ruff TID251 cấm `time.time()`** trong `hindsight_api`; gom 6 bản sao jitter về `remote_retry.jittered_backoff` | `test_metrics.py`, `test_remote_retry.py` | ARCHITECTURAL_FIX ở mức lớp lỗi (cổng lint) + SHARED | OBSERVABILITY |
| `b991f96fb` (#5425) | `banks_needing_consolidation()` seq-scan `memory_units`; đổi sang probe từng bank qua index. Thêm luật "một fact không bao giờ vừa consolidated vừa failed", importer ép luật này. Là phần **merge lại** của PR #5513 (merge `13b53c4f6` 12/10, revert `65ef1bdee` 13/10 không ghi lý do, tách lại thành `82ad30eab` 15/10, `b991f96fb` và `e92bf3543` 16/10) | `test_maintenance_routines.py` | LOCAL_PATCH | consolidation đúng hạn |
| `0433604c1` (#5228) | delta refresh sửa khối có fact mới mà để câu tóm tắt cũ nói giá trị cũ. Thêm một dòng footer trong prompt bảo model đọc lại cả trang. Upstream tự báo eval: trang stale 10/16 → 3/16 (gemini-2.5-flash-lite), 2/16 → 0/16 (3.1-flash-lite) | `hindsight-system-evals/evals/test_10_page_stays_consistent.py` (LLM thật, nightly), `test_delta_rereads_summary_llm` | LOCAL_PATCH (prompt; phụ thuộc LLM) | CURRENT. Số eval là **E0** với chúng ta |
| `ed369f9e9` (#4340) | model delta có thể full refresh theo lịch (`full_refresh_cron`) hoặc một lần; full refresh lỗi thì giữ nội dung cũ | `test_mental_model_scheduled_refresh.py` + migration `d4f7a2c9e1b5` | không áp (feat) | DERIVED_STATE_STALENESS: thêm cần gạt cho người vận hành, không tự phát hiện |
| `dcdcf6bd5` (#5473) | `_build_retain_params` bỏ mọi giá trị `None`, nên document "undated" khi reprocess bị gán `utcnow()`; control plane edit cũng bỏ nhiều tham số | system `test_14_reprocess_document.py`, `edited-retain-item.test.ts` | LOCAL_PATCH | CURRENT (thời gian sai khi phát lại) |
| `4b5f045c7` (#5298) | import document phát lại fact dưới id mới nhưng giữ nguyên `based_on` của mental model, nên evidence trỏ id của bank nguồn | system `test_52_transfer.py` | LOCAL_PATCH (dùng lại hàm remap của restore cả bank) | PROVENANCE_LOSS khi chuyển |
| `e73381c63` | đếm hết kết quả tìm document → giới hạn 1.000, response có `total_is_estimate` | `test_list_documents.py`, system `test_10_documents.py` | LOCAL_PATCH | kết quả suy giảm **có cờ** (ngược với H-13 của audit cũ) |

## 4. Ứng viên bug lặp

Nguồn: `issue-refs.tsv` (cột `loai` chứa `LAP=k`; ref ở ≥2 commit fix khác nhau; 131 ref) cộng các họ mà tôi ghép tay vì chúng mang **số khác nhau**. "Ứng viên" nghĩa là chưa đọc thảo luận issue để khẳng định là hồi quy.

| Họ | Chuỗi commit | Nguồn | Bất biến |
|---|---|---|---|
| Hook của host lọt vào tiến trình con LLM | #1751 `bcae23d9f` (05-28) → #5542 `c7b17f0ad` (10-16) | ghép tay | host hook interference |
| Cursor retain của coding-agents | #3136 `aaa58f247` (08-12) → #3989 `e655d889b` (09-08) → #4560 `a0e2583b0` (09-22) → #4493 `1238dbdbc` (09-29) → #5251 `d3f0001ed` (10-12) | ghép tay (#3989 có LAP=2) | CURSOR_ADVANCE_BEFORE_DURABILITY |
| Phạm vi tag | #5030 `c71cd0437` (10-01) → #5371 `89594d22d` → #5417 `3f8fe203f` (10-16) | ghép tay | SCOPE_FAILURE |
| Thành công giả của retain | #1833: `6e734e1af`, `a23187a45`, `d8a71c313` | LAP=3 | EXTRACTED ≠ completed |
| Tài liệu quá cỡ / slice | #1888: `364ccf17c`, `81865bf87`, `288a9b7fc` · #1838: `dee9a7b9d`, `fda969777` · #5485 `06a964dd2` | LAP=3 / 2 | partial success |
| Worker slot kẹt / LLM treo | #3002: `8133c5ab7`, `f6b3ce3e3`, `698620786`, `3531e82f9` | LAP=4 | WORKER_FAILURE |
| batch_retain cha/con | #1795: `74525cc04`, `8cf0dcbf8`, `ac4df7eb8` · #2985: `4708a3661`, `ac4df7eb8` | LAP=3 / 2 | ZOMBIE_OPERATION |
| Đọc structured output delta-op | #3820: `7fe6a5e79` (08-31), `9e9384107` (10-16) | LAP=2 | structured-output parse |
| Consolidation sửa giữa lời gọi LLM | #4831: `ff7f96ce2`, `b80910713` (+ `d8dd624b3` feat) | LAP=2 | CAS sau LLM |
| PR bị revert rồi merge lại | #5513: `13b53c4f6` → revert `65ef1bdee` → `82ad30eab`, `b991f96fb`, `e92bf3543` | LAP=2 | — |
| Khác (ngoài phạm vi overlay §7) | #2638 (routine maintenance theo schema, 5 commit), #2645 (vector index theo bank, 4), #3365 (max_tokens, 4), #4465 (4) | LAP≥4 | đa tenant / giới hạn token |

[SUY LUẬN] Ba họ đầu là ví dụ đúng kiểu overlay §18: "bug lặp là tín hiệu thiếu chủ sở hữu hoặc bất biến". Họ hook: ba cơ chế chống đệ quy riêng, không có chủ chung. Họ cursor: 5 bản sửa trên một tệp trạng thái ở `tmpdir()`. Họ tag: nhiều bản sao của luật, đến #5417 mới gom một phần.

## 5. Kết luận nào của audit cũ đã lỗi thời (spec §78 câu 12)

| Kết luận cũ (vị trí) | Nay | Vì sao |
|---|---|---|
| #5371 "OPEN, tái lập ở HEAD" (audit §13.4, §13.7; SE 24 H-44) | **FIXED** ở parser chung và 6 lối vào; còn một parser inline (§3.3) | 89594d22d |
| #5448 `allowed_bank_ids` khai báo mà không ép (audit §13.7; SE 24 H-43) | **FIXED bằng cách xoá trường** và chặn `dry-run-extract` | c7acb488a |
| #5228 delta refresh để lại câu cũ mâu thuẫn (audit §13.7; SE 24 H-25 nửa đầu) | **PARTIALLY_FIXED**: dòng footer trong prompt, phụ thuộc LLM; số eval của upstream là E0 | 0433604c1 |
| H-26 `based_on` = mọi memory đã recall khi model không trích id | **PARTIALLY**: một nguồn gây ra (fact qua `expand`) đã gỡ; nhánh "rỗng thì tất cả" vẫn còn | 45616ffe3 |
| §12.7 / SE 24 H-29..H-31: mô tả cursor coding-agents (đệm `pending`, `dirty`) | **Thiếu ngay tại revision được audit**: không nhắc trường `ops` của bản sửa #5251 (d3f0001ed là tổ tiên của e00154496) | bỏ sót, không do delta |
| §3.4 pipeline consolidation (fact → recall → LLM → áp action) | Có thêm bước **triage** bằng decision model, bật mặc định khi có cấu hình | 0d8e58b3e |
| §3.3 reflect = một vòng agent | Có thêm chế độ fast / decision-orchestrated | 198947912 |
| §3.1 delta retain | Có thêm luật "nhường streaming khi document viết dở" | 06a964dd2 |
| §4.10 danh sách provider | thêm `chatgpt`, NVIDIA NIM (embeddings + reranker), Anthropic WIF; đổi mặc định của codex. Tổng số tên provider: **chưa đếm lại** | e00ca6d3b, 2ecc73cc1, 8c5fabf4d, 074e3fe60 |
| §4.12 quan sát | thời lượng nay đơn điệu (cổng lint); token của codex/claude-code là số thật khi có usage | 29698acb6, dcc3ee421 |
| Số đếm §1/§8 | `memory_engine.py` 23.477 → **23.784** dòng; migration 111 → **113**; tệp test API 631 → **648**; system story 79 → **87** [ĐO `wc -l`, `git ls-tree`] | delta |

**Không lỗi thời** (delta không chạm vùng mã; [ĐO] grep diff ra rỗng; **không chạy lại** E1/E2/E7): không lease (H-01), #5582, `ExcludeNoneRoute` (E1), `run_consolidation` (E2), FK đa tenant (E7), mặc định mở (E4–E6), recall "4 song song" (H-47). Không commit nào trong **toàn** lịch sử tới HEAD nhắc #5582, #5419, #5519, #4130, #5497, #5544, #5638, #5589, #5434, #5574, #5590, #5442, #5433. "Không có commit nhắc" không có nghĩa là "chưa sửa"; riêng #5582 tôi đã đối chiếu diff `fact_extraction.py`: đường xử lý đó không đổi.

## 6. Issue then chốt

Ba trang issue và trang PR #5508 đọc qua WebFetch ngày 19/10, dùng 4/4 lượt cho phép; không trang nào trả 404 hay đòi đăng nhập. Tiêu đề nguyên văn của #5251 dài quá giới hạn trích của công cụ, nên dưới đây ghi bản diễn ý.

### 6.1 IssueRecord #5251

| Trường | Giá trị |
|---|---|
| project | vectorize-io/hindsight |
| issue_id | #5251 |
| title | (diễn ý) Một retain fail sau khi server đã accept làm mất vĩnh viễn các turn của nó; các lần append sau dựng document thiếu chúng [WEB] |
| date | mở 2026-10-05 (sambatia), đóng 2026-10-12 "completed" qua PR #5508 [WEB] |
| affected_version | `@vectorize-io/hindsight-coding-agents` 0.8.0 + `hindsight-api` 0.10.2, self-hosted macOS; provider LLM qua thuê bao CLI, lỗi do chạm hạn mức [WEB] |
| symptom | Stop hook đẩy cursor ngay khi server **accept** retain async. Sau đó trích xuất fail và các turn đó không bao giờ được lưu hay gửi lại. Stop kế tiếp chỉ gửi turn mới với `update_mode:"append"`; server dựng document `conversation:<session>` từ mảnh đó. Hook vẫn ghi `retain_ok` [WEB] |
| user_impact | Người báo tự đếm: document 1.180/1.197 dòng, thiếu header và turn 0–15; 3 document có lỗ khớp với operation fail (một của Codex, một Claude Code, một Antigravity). Số do người báo tự đo, **E0** [WEB] |
| subsystem | `hindsight-integrations/coding-agents` (retain cursor) + ngữ nghĩa append của server |
| root_cause | Cursor tiến theo ACCEPTED rồi được coi như DURABLE/EXTRACTED; server cho append vào document chưa tồn tại (tạo document từ mảnh) |
| root_cause_confidence | cao cho phần client (bản sửa và test khớp đúng kịch bản báo cáo); phần server là đề xuất **chưa làm** |
| state | **PARTIALLY_FIXED** (issue đóng "completed"; xem các chỗ còn hở) |
| fixing_pr / fixing_commit | PR #5508, merge `d3f0001ed` ngày 2026-10-12. Một commit gộp sửa 6 issue (#4963, #5079, #5081, #5251, #5306, #5052); PR không có reviewer [WEB] |
| regression_test | `src/core/chat.test.ts`: "replaces when an accepted write's operation later failed (#5251)". Vitest, client giả với bảng trạng thái giả; chưa chạy |
| later_regression | không thấy: delta không đổi tệp nào của integrations; `git log -S'#5251'` chỉ ra `d3f0001ed` |
| reproducibility | tôi không tái lập (không chạy). Báo cáo có các bước |
| architectural_lesson | ACCEPTED ≠ DURABLE ≠ EXTRACTED. Một con trỏ tiến độ phải tiến theo đúng trạng thái mà nó tuyên bố. Nếu xác minh bị hoãn thì cửa sổ hoãn phải có giới hạn và phải có ai đó chắc chắn chạy nó |
| agent_cli_analog | (giả thuyết, **chưa kiểm, không kết luận phơi nhiễm**) mọi chỗ agent-cli ghi tiến độ/ack khi mới "được nhận" thay vì khi đã xong bền. SE 28 tự ghi PRB-16 (`send` không `--wait` thoát 0 trước khi prompt được ghi) còn mở |
| proposed_agent_cli_regression_test | xem §7, T1–T2 |

**Trạng thái mã ở HEAD `5331da2f4`** [TĨNH ✅; mọi đường dẫn dưới `hindsight-integrations/coding-agents/src/core/` trừ khi ghi khác]:

1. `hindsight.ts:500-530` — `retain()` **luôn** gửi `async: true` kèm `operation_id` của client; không chờ trích xuất.
2. Server: `operation_id` của client chính là id của operation **cha** `batch_retain` (`hindsight-api-slim/hindsight_api/engine/memory_engine.py:22828`). Cha thành `failed` khi bất kỳ con nào failed (memory_engine.py:5677, 5687). Vì vậy `GET /operations/{id}` phản ánh được lỗi của con.
3. `retain-cursor.ts:121-125` — trường `ops`: "Accepted is not stored: extraction can still fail". `retain-cursor.ts:130` — `MAX_CURSOR_OPS = 8`, kèm comment "ponytail: an op still running after 8 more write-backs goes unchecked".
4. `chat.ts:140-165` — `settleOps`: GET trạng thái từng op với timeout 5 s (`chat.ts:131`). `failed|cancelled|error` (`chat.ts:132`) thì đánh `dirty`, và kế hoạch chuyển sang **replace** cả transcript. **404 coi như xong** (`chat.ts:155, 163`). Đọc không được thì op vẫn được theo dõi, không coi là failed.
5. `chat.ts:336-337` — `settleOps` **chỉ** chạy khi `appendSupported && cursor.ops.length && !cursor.dirty`, và **chỉ** vào lúc write-back **kế tiếp** của **cùng** session.
6. `chat.ts:407-410, 423-427, 446, 464` — op id được ghi vào cursor **sau khi accept**. Replace thì đặt `dirty` trước khi gửi, rồi xoá `ops` cũ và ghi op mới.
7. `retain-hook.ts:166` — vẫn ghi `diag … "retain_ok"` ngay sau khi `retainLiveSession` trả về, tức lúc **accept**, đúng triệu chứng người báo nêu. `retain-hook.ts:167-176` — lỗi chỉ thành `log.warn` + `diag retain_failed` (fail-open).
8. `session-cache.ts:183-185` — tệp cursor nằm ở `tmpdir()/hindsight-<harness>/<session>.retain.json`.
9. `settleOps` chỉ được gọi ở `chat.ts:337` (grep). SessionStart (`session-start.ts`) không kiểm `ops` của session trước.
10. Phía server, đề xuất thứ hai của #5251 ("từ chối append vào document chưa tồn tại") **không được làm**: orchestrator.py:1832-1834 ghi rõ append vào document chưa tồn tại là hợp lệ và được gộp; sentinel `_APPEND_BASE_ABSENT` (orchestrator.py:511-514) là trạng thái hợp lệ "chưa ai ghi document này".

**Chỗ còn hở ở HEAD** (suy ra từ các dòng trên; chưa chạy):
- (a) **Write-back cuối của session**: op được accept rồi mới fail mà không còn write-back nào sau đó, nên turn mất và không được phát hiện (điểm 5, 9).
- (b) Op **pruned** (404) bị coi là xong, dù nó có thể đã fail trước khi bị dọn (điểm 4). Mặc định server giữ operation mãi (`OPERATION_RETENTION_DAYS=0`), nên ca này chỉ xảy ra khi người vận hành bật retention.
- (c) Trên 8 op chưa xác nhận thì op cũ nhất bị quên (điểm 3).
- (d) Mất tệp ở `tmpdir()` (khởi động lại) làm mất danh sách `ops`. Lần sau không có cursor thì replace, nên lỗ được vá **nếu** còn write-back.
- (e) **"completed" mà 0 fact** vẫn được coi là thành công: thành công giả ở phía server (audit cũ H-08/H-09, #5582) không bị client phát hiện.
- (f) Trong cửa sổ từ lúc accept tới write-back kế tiếp, document có lỗ mà vẫn tìm được (ACCEPTED ≠ SEARCHABLE-đầy-đủ).

### 6.2 IssueRecord #2413

| Trường | Giá trị |
|---|---|
| title | "local_embedded worker can remain stuck on recovered async retain batch after structured JSON parse failures" [WEB] |
| date | mở 2026-06-25 (BooRy95), đóng 2026-06-30 **not planned** (nicoloboschi) [WEB] |
| affected_version | 0.7.2 (hindsight-client, hindsight-embed); chế độ `local_embedded`, Ollama `kimi-k2.6:cloud` |
| symptom | Batch retain async được khôi phục giữ worker sau nhiều lần lỗi parse JSON (LLM trả văn xuôi; `retain_extract_facts.attempt=4/4`). **`/health` vẫn xanh** trong khi hàng đợi suy giảm: 1 con `processing`, 3 cha `batch_retain` payload null ở `pending`, 2 con `pending`. Phải cancel tay 6 dòng rồi khởi động lại [WEB] |
| root_cause | người báo không xác nhận. Maintainer cho rằng operation "không thật sự kẹt, nó đang retry", và Ollama ép JSON schema kém; đã thêm log debug [WEB] |
| root_cause_confidence | thấp |
| state | NOT_PLANNED ở tracker. Phân loại của tôi: **DESIGN_LIMITATION** cho phần health; một phần được giảm nhẹ bằng test và thông báo lỗi |
| fixing_pr / commit | không có PR. Comment dẫn `37fb1dd` (chỉ nằm trên nhánh `origin/feat/stats-include-entity-links-optout`, **không** là tổ tiên của HEAD [ĐO `merge-base`]) và `00968a1ce` (main, 2026-06-30: giữ thông báo exception trong tóm tắt lỗi trích xuất) |
| regression_test | `815d99f5b` (2026-06-26, PR #2418): `test_retain_transient_extraction_failure.py::test_extraction_failure_at_retry_cap_fails_terminally`, docstring dẫn #2413. Message của `00968a1ce` ghi test này "ran for the first time on this branch (its original PR's test-api job was skipped)" và đã bị formatter **âm thầm vô hiệu**: test tồn tại mà không chạy trong CI |
| later_regression | cùng họ "việc kẹt mà client thấy `pending`": audit cũ E3b (H-01), #2359/#3194 not_planned |
| current_code_status | [TĨNH ✅] `hindsight-api-slim/hindsight_api/worker/main.py:93-112`: `/health` = `/health/ready` = kết nối DB. `main.py:115-135`: `/health/live` có `seconds_since_last_poll` nhưng **không bao giờ đổi mã trạng thái** (cố ý; docstring). Thu hồi việc vẫn theo `worker_id` (poller.py:1505). Health của daemon `hindsight-embed`: **chưa đọc** |
| reproducibility | không tái lập |
| architectural_lesson | HEALTHY (tiến trình sống + DB tới được) ≠ PROGRESSING (hàng đợi đang chảy) ≠ CORRECT. Suy giảm chỉ hiện ở metric/log. Một test hồi quy có thể tồn tại mà CI không chạy |
| agent_cli_analog | (giả thuyết) bề mặt trạng thái/health của routine/loop so với tiến độ thật; không kết luận |
| proposed test | §7 T3 |

### 6.3 IssueRecord #2347

| Trường | Giá trị |
|---|---|
| title | "Clarify benefit model and evaluation criteria for coding-agent memory integrations" [WEB] |
| date | mở 2026-06-22 (Sanderhoff-alt), đóng 2026-08-10 **completed** (nicoloboschi: thảo luận đã định hình Hindsight 0.9.0, kèm link một blog; nội dung blog **chưa kiểm**) [WEB] |
| symptom | không phải lỗi, mà là câu hỏi: "khi nào memory dài hạn thật sự cải thiện workflow của coding agent, và chứng minh thế nào?" Đề xuất 5 nhánh: không memory · markdown trong repo · memory gốc của agent · Hindsight auto recall/retain · Hindsight chuyên cho code. Metric: hoàn thành task, test pass, context lặp lại, thời gian resume, overhead token/độ trễ, recall lạc đề/cũ, tuân thủ sở thích, tỉ lệ người dùng sửa [WEB] |
| thảo luận | maintainer: benchmark nên đặt ở repo `agent-memory-benchmark`, đo chất lượng/tốc độ/chi phí. Người khác đề xuất miền tổng hợp `cross_tool_same_user`, `team_shared_bank`, `dead_end_avoidance`; đo riêng stale recall, harmful recall, chất lượng provenance. **Không có kết quả thực nghiệm hay con số nào trong issue** [WEB] |
| state | đóng "completed" nhưng không có bằng chứng lợi ích trong repo. Phân loại của tôi: **INSUFFICIENT_EVIDENCE** cho câu hỏi lợi ích |
| fixing_commit | không commit nào nhắc #2347 (cả message lẫn pickaxe) [ĐO] |
| current_code_status | không áp. Chỉ dấu liên quan: số `sdebench` trong README (audit cũ H-34, E0); 198947912 nhắc một ca "sde-bench seqledger-history" như giai thoại |
| architectural_lesson | lợi ích phải đo so với **cả** baseline repo-native và agent-native, có metric tác hại; trùng spec V3 §11.9 và SE 29 (M0 làm baseline) |
| proposed test | §7 T5 |

## 7. Ứng viên test cho agent-cli (overlay §15–17; không kết luận phơi nhiễm)

| # | Từ | Test đề xuất | Kỳ vọng |
|---|---|---|---|
| T1 | #5251 | **accept-rồi-hỏng**: một sink trả ack ngay rồi fail bất đồng bộ; chạy hai lượt ghi liên tiếp | con trỏ tiến độ không vượt qua đơn vị đã fail, **hoặc** lượt sau phát hiện rồi gửi lại toàn bộ; không có log "ok" trước trạng thái cuối |
| T2 | #5251 (a) | như T1 nhưng **không có lượt sau** (lượt cuối của session) | lỗi phải hiện ở một chỗ người dùng thấy được; không được chỉ nằm trong tệp diag |
| T3 | #2413 | **health xanh mà worker kẹt**: một job luôn lỗi parse rồi bị claim lại | bề mặt trạng thái báo suy giảm hoặc backlog trong N giây; job tới `failed` sau trần retry; test phải **được CI chạy** (kiểm tên test có trong kết quả, đúng bài học `00968a1ce`) |
| T4 | c7b17f0ad / #1751 | tiến trình con LLM hoặc CLI của agent-cli được sinh từ `$HOME` có `.claude/settings.json` chứa hook đánh dấu | hook đánh dấu không bắn; kiểm bằng CLI thật, không chỉ bằng tham số |
| T5 | #2347 | A/B theo spec §11.9: không memory · repo-native · agent-native · structured continuity · kiểu Hindsight | metric: task xong có kiểm, false DONE, rediscovery, stale recall, token |
| T6 | 89594d22d / 3f8fe203f | giá trị phạm vi lạ ở mọi lối vào; đếm số nơi tự tính luật phạm vi | lạ thì từ chối, không nới; mọi lối vào đi qua một hàm |
| T7 | 06a964dd2 | ngắt một lượt ghi nhiều phần ở phần k, rồi retry | phần đã commit không làm lại; không có trạng thái "đầy đủ" giả cho reader trong lúc dở |

---

## Hợp đồng đầu ra (spec §63)

- **SCOPE:** 48 commit `e00154496..5331da2f4`; các bất biến overlay §7; IssueRecord #5251, #2413, #2347; trạng thái mã của retain async/cursor ở HEAD.
- **REVISION AUDITED:** `5331da2f4db369528bb122075341a7de9c3df6df` (2026-10-16 15:58 -0400). Upstream sau mốc này: không biết (không fetch).
- **FILES/COMMITS/ISSUES INSPECTED:** diff đầy đủ của 06a964dd2, c7b17f0ad, 89594d22d, 3f8fe203f, c8b7b3341, e1f43dfa0, d7beb8e4b, c7acb488a, 074e3fe60, dcc3ee421 (phần mã), 0d8e58b3e (consolidator), d3f0001ed (chat/cursor), 815d99f5b; message của toàn bộ 48 commit; mã HEAD: `retain-cursor.ts`, `retain-hook.ts`, `chat.ts`, `hindsight.ts`, `session-cache.ts`, `orchestrator.py`, `tags.py`, `claude_code_llm.py`, `cursor_llm.py`, `memory_engine.py` (phần cha/con, validator, list_mental_models, based_on), `consolidator.py`, `triage.py`, `worker/main.py`, `worker/poller.py`, `response_models.py`; issue #5251, #2413, #2347, PR #5508 [WEB].
- **VERIFIED FACTS:** §0–§3 có tệp:dòng; delta không chạm `hindsight-integrations/`; #5371/#5448 đã sửa; #5251 sửa ở `d3f0001ed` trước revision của audit cũ; 10 chỗ trạng thái mã §6.1.
- **INFERENCES:** chỗ hở (a)–(f) của #5251; triage làm mất fact "chatter" vĩnh viễn; tài liệu viết dở tìm được một phần; ba họ bug lặp là do thiếu chủ sở hữu.
- **UNKNOWNs:** test có xanh ở HEAD không; health của daemon `hindsight-embed`; nội dung blog đóng #2347; lý do revert `65ef1bdee`; các issue OPEN mà không commit nào nhắc; tổng số provider hiện tại.
- **FAILURE MODES:** CURSOR_ADVANCE_BEFORE_DURABILITY, PARTIAL_SUCCESS, FALSE_SUCCESS ("retain_ok" lúc accept), HOST_ADAPTER_MISMATCH / PROMPT_INTERFERENCE (hook của host), SCOPE_FAILURE (nới phạm vi), OBSERVABILITY_FAILURE (ước lượng không có cờ, health xanh), DERIVED_STATE_STALENESS.
- **AGENT CLI RELEVANCE:** 7 ứng viên test ở §7; không kết luận phơi nhiễm.
- **PROPOSED TESTS:** T1–T7.
- **NO-CHANGE FINDINGS:** worker/lease, #5582, E1/E2/E7, mặc định mở, recall "4 song song": delta không chạm. Tích hợp coding-agents: 0 tệp đổi.
