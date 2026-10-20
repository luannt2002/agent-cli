# 02 — Bản đồ ma sát hằng ngày (§56)

Mỗi dòng là một chỗ người dùng phải tự làm thay agent-cli: rời agent-cli, chép ngữ cảnh, lặp lời dặn, dựng lại lịch sử, soi nội tạng provider, tự kiểm lời khai. Nguồn: log 3 ngày dùng thật (41 lượt), 10 kịch bản baseline + 3 kịch bản Daily với LLM thật, probe với LLM giả, và sổ vấn đề [05](../system-evolution/05_CURRENT_PROBLEMS.md). Trạng thái tính theo ứng viên `scratchpad/evo/wt-b`.

| # | Người dùng phải… | Bằng chứng | Phân loại | Nguyên nhân gốc | Trạng thái |
|---|---|---|---|---|---|
| F-01 | rời agent-cli để biết provider/model nào đã thật sự trả lời | TUI hiện model của workspace, không phải model đã chạy (21 mục 4); `run --json` không có provider/model | OBSERVABILITY_PROBLEM | model là trạng thái chung của workspace (C-2) | Một phần: `run --json` có `provider`/`model`; sổ ghi LLM opt-in. TUI vẫn hiện model workspace (PRB-05 mở) |
| F-02 | sửa tay `~/.local/share/agent-cli/crush.json` sau một lần provider thiếu khoá | fallback chéo provider được ghi bền, đè crushrc (`load.go:166-180`) | RELIABILITY_PROBLEM | máy ghi lựa chọn provider như tác dụng phụ (C-1) | Đã sửa: không ghi xuống đĩa (G-01) và không bao giờ đổi provider; model không dùng được thì dừng có lỗi nêu provider/model (vòng 6, faults ca `selected-no-key`) |
| F-03 | lặp lời dặn của dự án khi đứng ở thư mục con, hoặc khi đặt tên tệp thường `agents.md` | context file chỉ nạp ở cwd, tên viết hoa (PRB-19/20) | CONFIGURATION_PROBLEM | đường nạp context file hẹp | Mở (slice S-12) |
| F-04 | nhắc lại quy ước cá nhân (đòi bằng chứng, không commit…) mỗi phiên | quy ước nằm trong skill "luôn bật" mà model chỉ nạp ở 2/41 lượt | CONFIGURATION_PROBLEM | agent-cli không có cơ chế "luôn bật" cho skill; quy ước nên là context file toàn cục | Đề xuất chờ duyệt ([proposal](../system-evolution/proposals/agent-cli-skill-policy.md)) |
| F-05 | trả tiền cho 79 KB ngữ cảnh cố định mỗi request, 60% là skill không dùng | P-05; 4.630 B skill đã xoá trong `.trash` vẫn được gửi | UNNECESSARY_FEATURE | quét skill đệ quy mọi thư mục | Một phần: bỏ thư mục ẩn (−4.691 B/request); danh sách tắt 21 skill trong đề xuất chờ duyệt (−17.513 B nữa, đo bằng probe) |
| F-06 | tự kiểm "xong" của `run` | `run` rc=0 chỉ nghĩa là lượt chạy hết; model trả lời rỗng cũng rc=0 (PRB-75) | RELIABILITY_PROBLEM | không có tín hiệu verify trong `run` (đúng thiết kế) + lượt rỗng không thành lỗi | Lượt rỗng: đã sửa. "Xong có bằng chứng" thuộc `loop`/`routine` (tài liệu hoá ở README) |
| F-07 | đứng nhìn hộp thoại quyền của session A chặn session B | PRB-06 (test) | ARCHITECTURE_PROBLEM | một mutex toàn cục cho hộp thoại | Đã sửa |
| F-08 | kill/khởi động lại khi một lượt `run` treo vì session con chờ quyền | PRB-63 | RELIABILITY_PROBLEM | `run` local chỉ tự duyệt session gốc | Đã sửa (như chế độ server) |
| F-09 | đoán vì sao routine chạy theo lịch "không chạy gì" | lịch in ra thiếu biến launcher → rc=2 mỗi lần (PRB-07); lượt bị kill bị che sau GREEN cũ (PRB-02); lỗi tạo worktree hiện ABANDONED (PRB-82) | OBSERVABILITY_PROBLEM | báo cáo chỉ ghi lúc cuối; lịch không mang env | Đã sửa |
| F-10 | dọn worktree/nhánh routine | tích luỹ (PRB-23) | REQUIRED_COMPLEXITY | giữ để người duyệt là chủ ý | Giữ; ghi trong tài liệu routine |
| F-11 | đợi/huỷ tay khi model "quên tiến độ" sau tóm tắt và làm lại mãi | qwen-only.sh: 1.338 lời gọi / 120 s (PRB-88) | RELIABILITY_PROBLEM | không có trần tự tóm tắt mỗi prompt | Đã sửa (tối đa 10 lần) |
| F-12 | bọc `timeout` quanh `run` trong script | provider treo → `run` treo (faults ca `hang`) | REQUIRED_COMPLEXITY | `run` không có trần thời gian; `loop`/`routine` có `--run-timeout` | Giữ; ghi tài liệu |
| F-13 | thấy job nền "✓" dù lệnh đã lỗi | PRB-31 | UX_PROBLEM | header job không đọc exit code | Đã sửa (chưa xem trong TUI thật) |
| F-14 | lo session khác đọc/dừng nhầm job của mình | PRB-18 | ARCHITECTURE_PROBLEM | tool job không kiểm chủ | Đã sửa |
| F-15 | tự nhận ra prompt đầu của mình bị gửi sang provider thứ hai (tiêu đề) | P-01; faults 8/8 | ARCHITECTURE_PROBLEM | model small lấy provider riêng | Đã sửa (0/8; qwen-only: 5/5 vai trò ở qwen) |
| F-16 | chép bối cảnh sang phiên mới để tiếp việc hôm qua | — (s10 resume đạt bằng `-s`; TUI có danh sách session) | — | — | Không thấy ma sát đo được với M0; kịch bản B ở 01 |

Không thấy (đã tìm): phải quản lý agent con bằng tay (agent con chỉ chạy khi model gọi tool `agent`); hai engine điều phối chạy chồng nhau lúc runtime (Stop hook chạy trong lượt, verify của loop chạy sau lượt — hai định nghĩa "xong" nhưng không tranh nhau, C-4).
