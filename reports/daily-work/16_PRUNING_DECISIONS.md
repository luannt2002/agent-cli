# 16 — Quyết định cắt bỏ (§68, §88)

Mỗi dòng: thứ gì đã bỏ, đã quay lại, hay cố ý không thêm; vì sao; bằng chứng. "Bỏ" ở đây là trong patch ứng viên (`wt-final.patch`), không xoá mã có sẵn ngoài phạm vi (luật "không xoá dead code có sẵn").

## Đã bỏ (giảm hành vi hoặc mã)

| Thứ | Vì sao | Bằng chứng |
|---|---|---|
| Lời gọi tiêu đề cho session con | không lần nào ra tiêu đề; 8/21 lời gọi ở s12 | `TestNoTitleCallForChildSessions` |
| Thử lại tiêu đề bằng model large sau `length` (khi large == small) | 26/26 lời gọi tiêu đề baseline kết thúc `length` | `TestTitleStopsAfterTokenLimit`; s1 cuối: title 1 (baseline 2) |
| Quét skill trong thư mục ẩn (`.trash`) | skill người dùng đã xoá vẫn vào prompt; −4.691 B/request | probe P-05, `TestDiscoverSkipsHiddenDirectories` |
| Ghi fallback model lúc khởi động xuống đĩa | biến một lần thiếu khoá thành đổi provider vĩnh viễn | `TestStartupFallbackIsNotPersisted` |
| Tự duyệt session con của `agentic_fetch` trong TUI | nội dung web có thể bảo agent đọc tệp ngoài rồi gửi đi | `TestAgenticFetchChildSessionNotAutoApproved` |
| Ghi giá trị env vào log khi lệch cờ | lộ API key | `TestFirstWinsMismatch_NeverLogsEnvValues` |
| `mcp.Close` trong cleanup của MỌI App | đóng MCP của mọi workspace khi một workspace đóng | `TestShutdownStopsOnlyItsOwnWorkspace` |

## Thử rồi quay lại (rollback trong đợt)

| Thử | Vì sao bỏ | Thay bằng |
|---|---|---|
| Chặn biến git bằng phân tích phép gán (peel assigns, `scriptCommands` mang env, `DenyEnv`) | lượt soát 2 vòng qua bằng 7 dạng (export trong ngoặc, gán trước hàm, `env -S`, `find -exec`, `eval`…) | luật theo chữ; `protect.go` trả về nguyên bản (ít mã hơn, phủ rộng hơn) |
| "Lượt rỗng" xét từng lượt riêng | chặn nhánh tự làm tiếp khi bị cắt; đổi lý do từ chối (lượt soát 2) | xét theo `stop` |
| "Lượt rỗng" chỉ khi finish = `stop` | GLM `network_error`/`sensitive` → `unknown` lọt (lượt soát 3) | xét cả chuỗi prompt (`produced`), chỉ miễn `length` còn lượt làm tiếp và tách lỗi từ chối |
| Dòng lý do chặn không trần | 147.581 / 200.327 ký tự vào context (lượt soát 4) | trần 5 lý do × 300 ký tự |
| Fallback model chéo provider trong RAM lúc khởi động (để `login` chạy được) | red team cuối F1: vi phạm luật cứng §32; giả định "dừng thì không login được" sai | không đổi provider; provider không dùng được thì dừng có lỗi; `Load` vẫn thành công |
| Chép allowlist vào lịch routine, đổi khoảng trắng thành dấu phẩy | red team cuối L6: biến export thắng tệp env, che việc siết allowlist sau này | không chép; tệp env là chủ duy nhất; cảnh báo khi allowlist chỉ có ở shell (bớt một nhánh mã) |
| Gợi ý `-m provider/model` / `alt+m` trong lỗi "model không dùng được" | đo bằng binary: `run -m` vẫn dừng ở cùng lỗi, TUI không mở | gợi ý key/login/cấu hình; `alt+m` chỉ khi TUI đang chạy |
| GREEN kèm cảnh báo khi verify xanh sẵn và cây không đổi | red team cuối F4: thoát 0 mà không có gì chứng minh | `NO_CHANGE` |

## Cân nhắc và KHÔNG thêm (§68: "can it disappear?")

| Ứng viên | Vì sao không |
|---|---|
| Subsystem memory (kho, retain, recall, reflect) | không vấn đề nào chứng minh được (28–30) |
| Task graph / Agent Map | không có vấn đề agent-cli nào đã chứng minh đòi nó (§63, §94.14); `todos` + journal loop đủ cho 13 kịch bản đo được |
| Trần thời gian riêng cho `run` | script bọc `timeout`; `loop`/`routine` đã có `--run-timeout` |
| Hỏi quyền cho `job_kill` | kiểm chủ đã chặn lấn session |
| Ghi nhớ "model này luôn `length` khi đặt tiêu đề" | tiết kiệm nhỏ, thêm state |
| Gộp ba bộ phân tích lệnh shell / sandbox | lớn, cần thiết kế riêng (G-05) |

## Còn để nguyên (dead code / phức tạp có sẵn, không thuộc phạm vi xoá)

PRB-58 liệt kê mã chết có sẵn (`CreateTitleSession`, `ListNewFiles` với cột không tồn tại, `FormatInvocation`, `PublishStates`): chỉ ghi lại, không xoá (luật sửa phẫu thuật). Bài test mô phỏng ghi fallback trong `load_test.go` ("resolve and persist fallback under writeMu does not deadlock") nay kiểm một mẫu mà `Load` không còn dùng: vô hại, để nguyên, đề nghị xoá khi có yêu cầu review.
