# 11 — Hiệu quả skill (Daily K)

Câu hỏi: skill có làm việc hằng ngày tốt hơn không, tốn bao nhiêu, và quy ước nào nên thành chính sách thay vì skill?
Dấu vết đầy đủ: [05](05_SKILL_TRACE.md).

## Chi phí (E3, đo được)

Khối skill đi kèm mọi lời gọi main: 31.860 B (binary gốc, 48 skill) → 27.169 B (ứng viên, 40 skill) → 9.656 B nếu áp danh
sách tắt đề xuất (19 skill). Mỗi lượt có vài lời gọi main, nên khối này nhân lên theo số lời gọi.

## Có ích không (E5)

| | Baseline | Ứng viên trước vòng 6 | Binary cuối |
|---|---|---|---|
| Kịch bản có đọc skill | 5/10, đều đạt | 7/13, đều đạt | 4/13, đều đạt |
| Kịch bản không đọc skill | 5/10, đều đạt | 6/13, đều đạt | 9/13, đều đạt (s13 tính lần chạy lại; lần 1 trượt vì provider GLM treo, không liên quan skill) |

Kịch bản đạt cả khi có lẫn không nạp skill, nên bộ đo **không phân biệt được** skill có ích hay không. Không có A/B cùng việc
có và không có skill; chạy A/B trên bộ này chỉ đo được chi phí, mà chi phí đã đo được không cần LLM (SE [31](../system-evolution/31_SKILL_EFFECTIVENESS.md)).

## Quyết định

| Nhóm | Quyết định | Căn cứ |
|---|---|---|
| Skill trong thư mục ẩn (`.trash`) | Bỏ (trong mã) | người dùng đã xoá; −4.691 B mỗi request |
| Skill quy trình được chọn đúng việc (`systematic-debugging`, TDD, `ponytail-review`) | Giữ | được nạp đúng loại việc trong eval |
| Skill "luôn bật" (`using-superpowers`, `superpower`, `ponytail`, `verification-before-completion`) | Đề xuất tắt, đưa luật vào `~/.config/agent-cli/CRUSH.md` | agent-cli không có cơ chế luôn bật; nạp lác đác; luật bắt buộc phải là chính sách tất định |
| Skill app Claude (`synced`: computer-use, trình duyệt…) | Đề xuất tắt | trỏ tới tool agent-cli không có (red team L9) |
| Skill dựa trên agent con ghi mã (SDD, dispatching) | Đề xuất tắt | agent con của agent-cli chỉ đọc |
| Áp dụng | **Chờ người dùng duyệt** | đụng cấu hình thật `~/.config/agent-cli`, CLAUDE.md P7 cấm agent tự sửa ([đề xuất](../system-evolution/proposals/agent-cli-skill-policy.md)) |

## Quy ước thành chính sách (trong mã, không phải skill)

"Xong" là mã thoát verify (`loop`/`routine`); `NO_CHANGE`; lượt rỗng/bị từ chối/lặp là lỗi; không đổi provider. Đây là các
luật mà skill (vd `verification-before-completion`) chỉ khuyên, nay máy ép.
