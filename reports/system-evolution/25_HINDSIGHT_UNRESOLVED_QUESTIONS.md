# 25 — Câu hỏi Hindsight chưa giải (Phase 9.2)

Đây là đầu vào hạng nhất cho mọi quyết định áp dụng ở Phase 11–14. Mỗi dòng là một chỗ audit Hindsight **chưa** đo. Agent-cli **không** được lấp chỗ trống bằng giả định. Nguồn: `luannt-hindsight-1610.md` §9, §10, §12 `[nghi]`, §15, §17.

## 1. Khoảng trống cấu trúc của nghiên cứu

| # | Chưa có | Hệ quả cho agent-cli |
|---|---|---|
| U-01 | Không chạy với LLM thật: không biết trích fact đúng bao nhiêu, consolidation gộp đúng hay làm hỏng, reflect có bịa không | Mọi tính năng memory có xác suất (trích fact bằng LLM, observation, reflect) **phải** có thí nghiệm LLM thật của chính agent-cli trước khi thành mặc định. Không có thí nghiệm thì tối đa DEFER |
| U-02 | Không có ablation: semantic-only vs +BM25 vs +temporal vs +graph vs +reranker vs +observation | Không tầng retrieval nào được coi là "đã chứng minh có giá trị" |
| U-03 | Không chạy AMB (LoComo/LongMemEval/BEAM) | Benchmark của Hindsight không phải bằng chứng cho agent-cli |
| U-04 | Chưa tách "lõi thuật toán" khỏi "vỏ sản phẩm" bằng số | Không ước lượng được "Hindsight tối thiểu" nặng bao nhiêu |
| U-05 | 1.534 issue mới phân cụm theo tiêu đề; chỉ 59 issue lần tới commit/code/test, 40 issue đọc comment; cụm entity/graph, temporal, knowledge page, multi-bank, Hermes/OpenClaw chưa khảo cổ sâu | Bài học ở các cụm đó là E0 |
| U-06 | Test hồi quy chỉ xác nhận tồn tại; chưa chạy bộ 631 tệp test API (mới chạy 4 tệp system-test = 18 test) | "FIXED_VERIFIED" không có nghĩa là test xanh ở HEAD |
| U-07 | Không thử bank lớn (10k/100k/1M), chưa đo độ trễ retain/recall/reflect, chưa đo chi phí LLM mỗi thao tác | Không có số nào về chi phí/độ trễ để so với agent-cli |
| U-08 | Oracle, store-owned, provider ngoài OpenAI-compatible chưa chạy | — |
| U-09 | Hạn dùng: khoảng 270 issue/tháng; audit đúng với `e001544` ngày 2026-10-16 | Kiểm lại các mục OPEN trước khi trích như sự thật |

## 2. Nghi vấn Tier-1 chưa có repro (audit §17.6)

| # | Nghi vấn | Cách kiểm đề xuất (của audit) |
|---|---|---|
| T-01 | Retain lỗi một phần: bản cũ mất, bản mới dở, webhook `retain.completed` đã bắn, op `failed` | tài liệu 3 chunk, stub ném `ProviderContentPolicyError` ở chunk 2, có webhook; đếm `memory_units` + đọc `webhook_delivery` |
| T-02 | Fact lẻ có action bị loại rơi khỏi hàng đợi consolidation vĩnh viễn | batch 1 fact, LLM trả id sai |
| T-03 | Một `task_payload` hỏng chặn claim cả schema, lần poll nào cũng gặp lại | chèn dòng payload hỏng cũ nhất, chạy worker |
| T-04 | Vòng nhân bản qua dòng ghi công `From Hindsight memory` | transcript có dòng ghi công → Stop → recall |
| T-05 | Xoá document đồng thời với delta retain làm document "sống lại một phần" | race có kiểm soát |

## 3. Nghi vấn khác còn mở

| # | Nghi vấn | Trạng thái |
|---|---|---|
| Q-01 | `hindsight fs` lưu `api_token` plaintext | nghi, chưa chạy |
| Q-02 | `MCP_AUTH_TOKEN` so bằng `!=`, bỏ qua tenant extension | nghi |
| Q-03 | Đọc bank config lỗi → dùng config toàn cục lặng lẽ | nghi |
| Q-04 | `report-pr-status` là mã chết | nghi (không liên quan agent-cli) |
| Q-05 | `generate-clients.sh` bỏ qua Go lặng lẽ | nghi (không liên quan) |
| Q-06 | Graph timeout làm recall bank lớn kém lặng lẽ; fallback không timeout, không test | mạnh hơn, vẫn tĩnh |
| Q-07 | `security_events` / `receipt_uri` chỉ có trong docs | không đổi |
| Q-08 | Số `sdebench` và mọi benchmark README | chưa chạy |
| Q-09 | E2: job báo `observations_created: 2` nhưng DB có 1 → đếm theo action, không theo dòng ghi | nghi, cần đọc `consolidator.py` ~1971-1984 |
| Q-10 | `invalidated_memory_units` không có FK `chunk_id` → restore lỗi FK hoặc gắn sai chunk | nghi |
| Q-11 | Chỉ số nhân quả kiểm trên danh sách thô, nối vào danh sách đã lọc → `caused_by` nối sai | nghi |
| Q-12 | Fact không ngày (temporal 0,5) vượt fact đúng cửa sổ khi CE ngang | nghi |
| Q-13 | Reranker chỉ tiếng Anh (#5519) làm hỏng recall tiếng Việt | nghi, **chưa kiểm**, liên quan trực tiếp người dùng |
| Q-14 | Harness không gửi `sessionId` → replace lẫn nhau | nghi |
| Q-15 | 2 dòng log "No stub rule matched … search_observations" mà test vẫn xanh | chưa tra |

## 4. Luật dùng tài liệu này

- Nếu một quyết định ở Phase 12 dựa vào một dòng trong tệp này thì quyết định đó tối đa là **DEFER**, trừ khi agent-cli tự đo bằng thí nghiệm của mình.
- Tự đo trên agent-cli tạo bằng chứng E7 cho agent-cli. Nó không lấp khoảng trống của nghiên cứu Hindsight, và cũng không được ghi ngược thành "Hindsight đã chứng minh".
