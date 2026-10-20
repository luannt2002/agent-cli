# 05 — Dấu vết skill (§58, Daily K)

Trạng thái một skill: DISCOVERED (quét thấy trên đĩa) → LISTED (tên + mô tả vào system prompt) → ACTIVATED (model gọi
`view …/SKILL.md`) → USED/EFFECTIVE (không có cơ chế ghi; xem SE [07](../system-evolution/07_SKILL_RUNTIME_FLOW.md)).

## DISCOVERED → LISTED (E3: provider giả, skill thật trên máy tác giả)

| Binary | Khối skill mỗi request main | Số skill liệt kê | Nguồn |
|---|---|---|---|
| gốc `4a0b25b6` | 31.860 B | 48 | SE [31](../system-evolution/31_SKILL_EFFECTIVENESS.md), probe P-05 |
| ứng viên (bỏ thư mục ẩn như `.trash`) | 27.169 B | 40 | như trên |
| ứng viên + danh sách tắt đề xuất (chưa áp, chờ người dùng) | 9.656 B | 19 | [đề xuất](../system-evolution/proposals/agent-cli-skill-policy.md) |

Khối skill được gửi lại ở mọi lời gọi main. Đổi duy nhất trong mã: không quét thư mục ẩn (`TestDiscoverSkipsHiddenDirectories`,
`TestDiscoverKeepsAHiddenBaseWrittenWithATrailingSlash`).

## LISTED → ACTIVATED (E5: eval LLM thật)

Cách đếm: tên skill trong các đường `…/<skill>/SKILL.md` xuất hiện trong transcript của kịch bản (`session.json` mà
`scripts/eval/run.sh` lưu), tức model đã gọi `view` tệp đó. Lệnh: regex `([A-Za-z0-9_.-]+)/SKILL\.md` trên từng
`session.json`.

| Kịch bản | Baseline (binary gốc) | Ứng viên trước vòng 6 | Binary cuối |
|---|---|---|---|
| s1-simple-bug | systematic-debugging | systematic-debugging, verification-before-completion | systematic-debugging |
| s2-multi-file-bug | systematic-debugging | — | superpower, systematic-debugging |
| s3-feature | superpower, test-driven-development | — | — |
| s4-refactor | — | ponytail, using-superpowers | — |
| s5-debug-unknown | systematic-debugging | systematic-debugging, test-driven-development | systematic-debugging |
| s6-test-integrity | — | — | systematic-debugging |
| s7-review | ponytail-review | ponytail-review | — |
| s10-resume | — | systematic-debugging | — |
| s12-multi-agent | — | — | — |
| s13-provider-switch | — | superpower, test-driven-development, verification-before-completion | — (cả lần chạy lại) |
| s14-wrong-context | (chưa có) | — | — |
| s16-review-shared | (chưa có) | ponytail-review | — |
| s17-single-agent | (chưa có) | — | — |
| **Kịch bản có đọc skill** | **5/10** | **7/13** | **4/13** |

Log 3 ngày dùng thật của tác giả (binary đã cài, E5): 2/41 lượt có nạp skill (SE 31).

## Nhận xét

- Lựa chọn khớp loại việc khi có: gỡ lỗi nạp `systematic-debugging`, review nạp `ponytail-review`, tính năng/sửa lỗi có lúc
  nạp TDD.
- Cùng một kịch bản, hai lần chạy nạp khác nhau (vd s2, s3, s4, s10, s13): việc nạp skill không ổn định giữa các lần chạy.
- Skill "luôn bật" (`using-superpowers`, `superpower`, `ponytail`, `verification-before-completion`) chỉ được nạp lác
  đác; không có cơ chế bật luôn ở agent-cli (SE [08](../system-evolution/08_PONYTAIL_AUDIT.md), [09](../system-evolution/09_SUPERPOWERS_AUDIT.md)).
