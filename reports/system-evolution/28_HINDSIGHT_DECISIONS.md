# 28 — Quyết định cho từng khái niệm Hindsight (Phase 12)

Luật: quyết định dựa trên vấn đề ĐÃ CHỨNG MINH ở agent-cli ([27](27_PROBLEM_FIT_MAP.md)). Dựa vào một dòng của [25](25_HINDSIGHT_UNRESOLVED_QUESTIONS.md) thì tối đa DEFER. Thang bằng chứng: E0 lời khai · E1 đọc mã · E2 có test · E3 chạy với stub · E4 chạy thật local · E5 LLM thật · E6 benchmark · E7 cả hệ thống agent-cli.

| Khái niệm Hindsight | Quyết định | Căn cứ | Ở agent-cli nghĩa là gì |
|---|---|---|---|
| Memory Bank (kho memory riêng theo dự án) | **REJECT** (hôm nay) | 27: không lớp vấn đề nào đòi memory dài hạn có bằng chứng; U-01 (chưa ai đo chất lượng với LLM thật) | Giữ context file + lịch sử session (M0) |
| Retain: LLM trích fact từ transcript | **REJECT** | H-08/H-09 (0 fact mà `completed`), H-28 (vòng nhân bản qua dòng ghi công), U-01 | Không trích tự động. Bài học do người hoặc agent ghi vào context file, người duyệt qua git |
| Recall nhiều chiến lược (dense+BM25+temporal+graph+rerank) | **DEFER** | U-02 (không ablation), H-47 (mô tả "4 song song" sai), H-50/Q-13 (reranker chỉ tiếng Anh, chưa kiểm tiếng Việt) | Chỉ xét khi M1 (bài học có cấu trúc) đã có và đo được là tìm không ra |
| Reflect (LLM tổng hợp trả lời kèm `based_on`) | **REJECT** | H-26 (`based_on` = mọi memory đã recall khi model không trích id), H-27 | Không có tầng tổng hợp; model chính đọc thẳng nguồn |
| Observation / consolidation | **REJECT** | H-17..H-24 (đóng dấu sai, xoá sạch bank, mồ côi, không vết) | — |
| Mental model / knowledge page tự làm mới | **REJECT** | H-23, H-25 (câu cũ mâu thuẫn sau delta refresh) | DECISIONS.md, CLAUDE.md do người viết là "trang tri thức" duy nhất |
| Hàng đợi async + worker | **NOT_APPLICABLE** | agent-cli không có worker nền cho memory | Bài học vòng đời (H-01..H-10) được **ADAPT** thành bất biến (26) cho routine/run/hàng đợi prompt |
| Hook retain-ở-Stop / recall-ở-prompt cho coding agent | **REJECT** | H-28, H-29, H-30, H-31 | Hook Stop/PreToolUse của agent-cli giữ vai trò cổng, không vai trò memory |
| Provenance (`source_memory_ids`, trace) | **ADAPT** — làm cho bằng chứng, không cho memory | Lỗi H-36: trace lưu nguyên văn prompt | Sổ ghi LLM (`AGENT_CLI_LLM_LEDGER`, opt-in) chỉ lưu metadata: vai trò, provider, model, kích thước, sha system prompt, tên tool, token. Không lưu nội dung. Báo cáo routine có `base_sha` + tệp đổi. Test: `llm_ledger_test.go` |
| Scope/tag đi theo dữ liệu dẫn xuất | **ADAPT** (bất biến) | H-43, H-44, bài học #9 | `disabled_tools` áp cả vào tool con của agentic_fetch (`TestFilterDisabledToolsAppliesToInnerTools`) |
| Memory Defense (lọc secret) | **ADAPT** (bất biến "không lưu secret") | H-36, H-37, H-41 | Log không còn ghi giá trị env (`TestFirstWinsMismatch_NeverLogsEnvValues`). PRB-49 (DB giữ đủ output tool, không retention) để mở |
| Ack sau khi ghi bền | **ADAPT** (bất biến), chưa sửa | bài học #11 | PRB-16 (`send` không `--wait` thoát 0 trước khi prompt được ghi) để mở, ghi ở 36 |
| Rỗng hợp lệ ≠ rỗng do lỗi | **ADAPT** (bất biến) | H-08, bài học #1-2 | Tóm tắt bị cắt → lỗi; loop detection → lỗi; GREEN mang cờ `baseline_pass`/`unchanged` |
| Mọi việc có đường tới trạng thái cuối | **ADAPT** (bất biến) | H-01, bài học #6 | routine ABANDONED, đóng lời trả lời dở sau sập |
| Hiệu lực thời gian (`valid_from/to/superseded_by`) | **DEFER** | H-22 | Chỉ cần khi có memory dẫn xuất |
| Webhook | **NOT_APPLICABLE** | — | — |
| Benchmark memory (LoComo/LongMemEval, `sdebench`) | **REJECT làm bằng chứng** | U-03, H-34 (E0) | Đo bằng kịch bản của chính agent-cli (Phase Daily: B, I, J) |

## Kết quả

- **ADOPT nguyên khối:** không có.
- **ADAPT:** 7 bất biến (provenance cho bằng chứng, scope dẫn xuất, không lưu secret, ack sau ghi, rỗng ≠ thành công, trạng thái cuối, sửa lan tới mọi bản sao). 5 cái đã có test ở worktree, 2 cái mở (PRB-16, PRB-49).
- **KEEP_EXISTING:** context file, lịch sử session, tóm tắt tự động, journal loop.
- **DEFER:** recall nhiều tầng, hiệu lực thời gian, đưa bài học giữa các lần chạy routine.
- **REJECT:** retain tự động, reflect, consolidation, mental model, hook memory.

Điều kiện mở lại: kịch bản Daily B (tiếp việc hôm qua), I (bài học lịch sử) hoặc J (memory sai) thất bại ở M0 vì thiếu tri thức giữa phiên. Khi đó mới thử M1 ([29](29_MEMORY_COMPETITION.md)).
