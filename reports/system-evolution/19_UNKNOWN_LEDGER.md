# 19 — Sổ cái Known/Unknown (Phase 6, cập nhật dần ở Phase 23)

Không phải một bộ lập kế hoạch: đây là chỗ thách thức độ tự tin. Mỗi dòng có trạng thái theo vòng đời UNKNOWN → GIẢ THUYẾT → ĐANG KIỂM → BẰNG CHỨNG → ĐÃ BIẾT. Loại: KK (biết mình biết), KU (biết mình chưa biết), UK (đã có dữ liệu mà chưa dùng), UU (phát hiện bất ngờ — ghi lại cách nó lộ ra).

## Đã biết (KK) — có lệnh tái lập

| # | Điều đã biết | Bằng chứng |
|---|---|---|
| K-1 | Provider/model thật của người dùng là `main/qwen3.8` cho mọi vai trò hiện tại | sổ ghi, baseline thật s1 |
| K-2 | Tự tóm tắt lặp vô hạn khi phần cố định ≥ 80% window (trước sửa) | P-04 |
| K-3 | Việc phụ đi theo provider của slot small (trước sửa) | P-01 |
| K-4 | 48 skill (31.860 B) trong mọi request; 41 lượt thật chỉ 2 lượt nạp skill | P-05, log server |
| K-5 | Tiêu đề bằng LLM thất bại (`length`) với model hiện tại, tốn 2 lời gọi/phiên | baseline thật |
| K-6 | Bộ test Go baseline 68 ok / 17 không test / 0 FAIL | evidence/00 |

## Chưa biết, biết là chưa biết (KU)

| # | Câu hỏi | Vì sao quan trọng | Cách kiểm | Trạng thái |
|---|---|---|---|---|
| U-1 | Provider `main` là local hay trả phí? | quyết định chi phí của eval LLM thật và kịch bản "Qwen local/free" | người dùng xác nhận (giá trị nằm trong tệp env bí mật) | cần người dùng |
| U-2 | Cắt 14 skill `synced` có làm giảm chất lượng việc nào không? | 14 KB mỗi request | A/B trên bộ kịch bản với `disabled_skills` | chưa đo |
| U-3 | Model có thật sự LÀM THEO skill đã nạp không? | "nạp" ≠ "dùng" ≠ "hiệu quả" | so hành vi có/không skill trên cùng kịch bản | chưa đo |
| U-4 | Agent con trong `run` local có treo khi cần quyền không (PRB-63)? | độ tin cậy routine | test: agent con `view` ra ngoài workspace trong `run` local | chưa chạy |
| U-5 | Race lúc tóm tắt (PRB-64) có xảy ra thật? | hỏng lịch sử | test overlay hai prompt đồng thời | chưa chạy |
| U-6 | Fallback chéo provider lúc khởi động (PRB-04) có xảy ra trên máy này không? | khoá người dùng / tốn tiền âm thầm | đọc global data hiện tại (không chứa secret? có key → không đọc); thử với HOME giả | chưa chạy |
| U-7 | Biến thiên kết quả giữa các lần chạy LLM thật | baseline 1 lần/kịch bản có thể nhiễu | chạy lặp 3 lần một vài kịch bản | chưa đo |
| U-8 | Khôi phục khi server bị SIGKILL giữa tool call với provider thật | PRB-16 | kịch bản kill có kiểm soát | chưa chạy |

## Có dữ liệu mà chưa khai thác (UK)

- Log `Skill turn summary` của server cũ (đã dùng một phần) còn chứa `prompt_tokens` theo lượt — có thể ước lượng chi phí skill trên dữ liệu thật mà không tốn lời gọi mới.
- Bảng `files` lưu đủ nội dung mọi phiên bản → undo/checkpoint làm được mà không thêm lưu trữ (Phase 25 sẽ cân).

## Phát hiện bất ngờ (UU) và cách nó lộ ra

| # | Phát hiện | Lộ ra nhờ |
|---|---|---|
| X-1 | Vòng tóm tắt vô hạn | thử đặt window nhỏ để kích tóm tắt trong probe vai trò (định đo việc khác) |
| X-2 | Skill trong `.trash` vẫn được nạp | output log của một test backend đỏ (test không cô lập home) |
| X-3 | Provider thật là `main/qwen3.8`, không phải GLM như CLAUDE.md P8 ghi | sổ ghi ở lượt chạy thật đầu tiên |
| X-4 | Viết tắt option dài lọt lớp chặn git | verifier đối kháng g3 |
| X-5 | Mọi lượt tiêu đề với model suy luận đều cắt ở 40 token | sổ ghi (`finish=length`) |
