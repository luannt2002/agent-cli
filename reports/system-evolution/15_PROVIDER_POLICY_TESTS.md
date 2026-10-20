# 15 — Test chính sách provider (Phase 4)

## Đã có (chạy được, xanh ở worktree)

| Test / probe | Chứng minh | Loại |
|---|---|---|
| `TestCheckModelPolicy` (có sẵn) | allowlist chặn model ngoài danh sách, không chặn khi không đặt | unit |
| `TestSameProviderSmall` (mới) | small khác provider → model large; cùng provider → giữ small | unit |
| `TestBuildAgentModelsKeepsSmallOnTheMainProvider` (mới) | đường dựng model thật của coordinator trả small thuộc provider large; đột biến bỏ dòng sửa → FAIL | unit (coordinator thật) |
| `TestLedgerRecordsEachCallWithRoleProviderAndUsage` (mới) | sổ ghi ghi đúng vai trò/provider/model/usage từng lời gọi, không chép prompt | unit |
| P-01/P-02 (baseline) và P-07 (sau sửa) | `probe-provider-roles.sh`: large=qwen, small=glm qua hai provider giả; đếm request theo provider ở phía server | E3 runtime |
| P-06 | sổ ghi khớp 5/5 với server giả | E3 runtime |

## Hồi quy "chỉ Qwen" (đặt hàng ở workflow Daily §35)

Kịch bản: provider = Qwen (local/miễn phí), cấm chéo provider. Đếm lời gọi trái phép phía server.

| Đường LLM | Có trong agent-cli? | Đã phủ | Kết quả sau sửa |
|---|---|---|---|
| lượt chính | có | P-07 | 0 lời gọi ngoài qwen |
| tiêu đề (session + agent con) | có | P-07 | 0 |
| agent con | có | P-07 | 0 |
| tóm tắt | có | P-04 (provider giả đơn) | cùng provider (model large) — chưa chạy lại ở cấu hình 2 provider |
| agentic_fetch | có | chưa chạy | theo mã: small → large cùng provider |
| plan mode | có (model large) | chưa chạy | theo mã: large |
| loop/routine | có (agent chính) | chưa chạy ở 2 provider | theo mã: large |
| reviewer/verifier LLM, memory LLM | KHÔNG tồn tại | — | NOT_APPLICABLE |

**Việc còn lại (Phase 8):** gộp các đường trên vào một script E2E `scripts/eval/qwen-only.sh`: hai provider giả, đếm lời gọi theo provider phía server, assert = 0 cho provider bị cấm. Thêm ca tóm tắt (window nhỏ), agentic_fetch (URL tới server giả) và `loop`. Chưa làm: ca fallback lúc khởi động (PRB-04) vì chưa có thiết kế sửa.
