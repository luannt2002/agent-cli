# 31 — Hiệu quả skill (Phase 19)

Câu hỏi: skill có làm việc tốt hơn không, và tốn bao nhiêu? Trạng thái skill: DISCOVERED → LISTED (vào system prompt) → ACTIVATED (model `view` SKILL.md) → USED/EFFECTIVE (không có cơ chế đo, xem [07](07_SKILL_RUNTIME_FLOW.md)).

## Chi phí: đo được (E3, provider giả, skill thật của máy)

| | Request main đầu tiên | Khối skill | Số skill liệt kê |
|---|---|---|---|
| Binary gốc `4a0b25b6` | 78.972 B | 31.860 B | 48 |
| Binary sửa (bỏ thư mục ẩn như `.trash`) | 74.240 B | 27.169 B | 40 |
| Binary sửa + danh sách tắt đề xuất | 56.621 B | 9.656 B | 19 |

Khối skill được gửi lại ở **mọi** lời gọi main, không chỉ lời gọi đầu.

## Kích hoạt: đo được

| Nguồn | Lượt | Lượt có nạp skill | Skill được nạp |
|---|---|---|---|
| Log binary đã cài (3 ngày dùng thật, E5) | 41 | 2 | `redamon-runtime-reference` (skill dự án khác); 5 skill một lúc: brainstorming, dispatching-parallel-agents, subagent-driven-development, superpower, test-driven-development |
| Baseline 10 kịch bản ([21](21_BASELINE_RESULTS.md), LLM thật, E5) | 10 | 5 | s1, s2, s5: `systematic-debugging` · s3: `superpower` + `test-driven-development` · s7: `ponytail-review` |

Nhận xét từ baseline:

- Lựa chọn khớp loại việc: việc gỡ lỗi nạp `systematic-debugging`, review nạp `ponytail-review`, tính năng nạp TDD.
- s3: model đọc `superpower` ở `~/.claude/skills/`, rồi **đoán** TDD cũng ở đó. Lời gọi `view` trả "File not found" (system prompt ghi đúng vị trí `~/.agents/skills/…`). Model không thử lại. Đây là lỗi của model, tốn 1 lời gọi tool; không sửa.
- Hai việc không nạp skill (s4 refactor, s6 test integrity) cũng đạt. 10/10 kịch bản đạt dù có nạp hay không. Bộ kịch bản **không phân biệt được** skill có ích hay không.

## Hiệu quả: chưa đo được

- Không có thí nghiệm A/B cùng việc, có và không có skill, với LLM thật. Baseline đạt 10/10, nên A/B trên bộ này chỉ còn đo được chi phí, không đo được thành công.
- Không có cơ chế ghi "đã làm theo" (USED). Đây là giới hạn ghi nhận, không phải lỗi cần sửa.

## Quyết định

| Nhóm | Quyết định | Căn cứ |
|---|---|---|
| Skill trong thư mục ẩn (`.trash`) | **Bỏ** (đã sửa trong mã, `hidden_dirs_test.go`) | người dùng đã xoá; khối skill nhỏ đi 4.691 B mỗi request (31.860 → 27.169, đo ở bảng trên) |
| Skill quy trình thường được nạp (systematic-debugging, TDD, brainstorming, writing-plans, ponytail-review…) | **Giữ** | được chọn đúng loại việc ở baseline |
| Skill "luôn bật" mà agent-cli không có cơ chế bật (using-superpowers, superpower, ponytail, verification-before-completion) | **Đề xuất tắt; đưa luật vào `~/.config/agent-cli/CRUSH.md`** | luật nên là chính sách tất định nạp mọi phiên, không phải skill model có thể không đọc |
| Skill app Claude (`synced`: docs, docx, pptx, google-workspace, computer-use…) | **Đề xuất tắt** | công cụ mà agent-cli không có; 0 lần nạp |
| Áp dụng | **Chờ người dùng duyệt** | tệp cấu hình thật, CLAUDE.md P7 cấm tự sửa ([đề xuất](proposals/agent-cli-skill-policy.md)) |

Thí nghiệm còn nợ: A/B 2–3 kịch bản có và không có danh sách tắt, LLM thật, đo token + đạt/không. Chạy ở lượt đánh giá cuối nếu người dùng áp đề xuất. Không chạy trước, vì chỉ đo được chi phí, và chi phí đã đo được không cần LLM.
