# Review README độc lập khi tiếp quản — 19/10/2026

KET-LUAN: DAT
README: READY

Reviewer: `readme_takeover_review`, ngữ cảnh mới, chỉ đọc; không sửa file, không
đọc source để bù chỗ thiếu, không gọi model/API. Ứng viên: base `4a0b25b6` +
patch vòng 8 + 15 sửa README cuối được khôi phục từ lịch sử phiên Claude.
Đây là review thay thế lượt 6 chưa có kết quả, không phải kết luận được lấy
lại từ reviewer cũ.

| # | Câu hỏi | Trả lời từ README | Dòng |
|---|---|---|---|
| 1 | Agent CLI? | Trợ lý lập trình terminal, fork Crush. | 3–9 |
| 2 | Vì sao tồn tại? | Công việc FE/BE/devops, nhiều dự án/việc, biết model và kết quả. | 6–7 |
| 3 | Daily Engineering Workspace? | Mở dự án → tiếp việc → giao → theo dõi → kiểm → tiếp ngày sau. | 39–40 |
| 4 | Bắt đầu? | Clone, installer, PATH, env, models, thử PONG ở thư mục trống, mở TUI. | 14–35 |
| 5 | Workspace? | Gốc git/thư mục thật, server/session/job/log riêng. | 91 |
| 6 | Session? | Mạch một việc; run là một lượt trong đó. | 92–93 |
| 7 | Provider? | Backend model local hoặc dịch vụ. | 94, 107–117 |
| 8 | Model? | Model thuộc provider; phân biệt cú pháp chọn và allowlist, large/small. | 95–96 |
| 9 | Nhiều provider? | Env cho provider chính, key/login cho provider phụ, visible-provider. | 107–117 |
| 10 | Đổi provider? | Cùng session giữ lịch sử; lựa chọn áp cả dự án, giới hạn bằng chứng rõ. | 119–126 |
| 11 | Cô lập provider? | Lời gọi ẩn theo chính, fail closed, allowlist theo tên; không khoá URL và không sandbox tool. | 128–174 |
| 12 | Skills? | SKILL.md, model tự chọn đọc; context cho luật luôn áp; không bắt buộc runtime. | 243–252 |
| 13 | Routine? | Loop lưu định nghĩa, worktree, maintenance/goal, lịch chỉ in. | 103, 200–227 |
| 14 | DONE? | Verify rc=0, GREEN không bảo đảm ngoài verify; NO_CHANGE và giới hạn tự sửa test. | 176–198 |
| 15 | Memory? | Context + lịch sử + tóm tắt; không có kho memory riêng hay cưỡng chế thứ tự tin. | 229–241 |
| 16 | Tài liệu sâu? | Guide, README upstream, AGENTS.md. | 9–10, 309, 330 |

MUST-FIX: không có.

NICE-TO-HAVE đã xử lý: câu về Git của loop trong guide (dòng 671 lúc review)
đã đồng bộ với README: driver không tự commit/push, nhưng model tự duyệt tool
vẫn có thể gọi Git; routine chỉ có bộ chặn theo mẫu, không phải sandbox.

Reviewer đã đọc README đánh số dòng, hai tệp mẫu env/crushrc và các đoạn guide
liên quan. Đã chạy binary ứng viên với `--version`, `--help`, `run --help`,
`loop --help`, `routine new --help`, `routine run --help`, `send --help`, `keys`
qua `env -i`, các thư mục HOME/config/data/cache tạm, tắt metrics và auto-update.
Version đọc được: `v0.2.1-0.20261016093845-4a0b25b698fb+dirty`.
Không chạy build/test, prompt, timer hoặc config thật trong lượt review.
