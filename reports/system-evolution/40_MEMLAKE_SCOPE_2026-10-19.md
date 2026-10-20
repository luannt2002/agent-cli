# 40 — Memlake trong ảnh người dùng và phạm vi agent-cli

Ngày đọc nguồn: 19/10/2026. Bổ sung sau khi người dùng gửi ảnh có tên **Memlake**.
Các báo cáo 28–30 trước đó phân tích Hindsight và lựa chọn memory cho agent-cli,
chưa có mục riêng về Memlake. Không coi kết luận cũ là benchmark Memlake.

## Phân biệt các thành phần

| Thành phần | Vai trò | Trạng thái trong ứng viên |
|---|---|---|
| SQLite `crush.db` | Session, messages và dữ liệu dự án của agent-cli. | Đang dùng; code ở `internal/db/connect.go`. |
| M0 | Context file + lịch sử session + tự tóm tắt. | Đang dùng; chưa có kho bài học tự sinh xuyên session. |
| Hindsight open source | Hệ thống memory; bản mở tiếp tục chạy Postgres. | Chưa tích hợp. [Công bố chính chủ](https://hindsight.vectorize.io/blog/2026/10/13/memlake). |
| Hindsight Memlake | Engine memory bằng Rust, lưu bền trên object storage; query node giữ cache, indexer xây index. Thuộc Enterprise. | Chưa tích hợp, chưa chạy thử. [Tài liệu chính chủ](https://docs.hindsight.vectorize.io/enterprise/memlake/). |

Memlake chuyển nơi lưu memory, links/indexes và tài liệu liên quan sang bucket;
phần vận hành Hindsight như bank settings, job queue và audit log vẫn ở database.
Nó giữ API Hindsight, không phải driver thay trực tiếp SQLite của agent-cli.
[Tài liệu về phạm vi](https://docs.hindsight.vectorize.io/enterprise/memlake/).

## Các số trong video là bằng chứng gì?

“47×” và “100 triệu memory, dưới 100 ms p99” là số do nhà cung cấp công bố,
không phải kết quả đã đo cho agent-cli. Tài liệu benchmark ghi rõ phép so sánh
head-to-head không dùng phần cứng giống hệt; bài engine-level không đi qua toàn
bộ Hindsight API, bài 100M dùng dữ liệu tổng hợp; ước lượng chi phí là mô hình.
Không suy ra mọi workload sẽ nhanh hơn 47× hoặc agent sẽ trả lời đúng hơn.
[Điều kiện và caveats](https://docs.hindsight.vectorize.io/enterprise/memlake/performance/).

## Quyết định trong lượt tiếp quản

Giữ SQLite và M0. Đây là quyết định phạm vi dựa trên kiến trúc agent-cli hiện có,
không phải kết luận Memlake kém hơn: chưa có lớp Hindsight để chuyển backend,
chưa đo nghẽn truy hồi memory, chưa đánh giá triển khai/chi phí Enterprise.
Nếu mở hướng memory mới: xác định bài toán xuyên session, corpus và tiêu chí
đúng trước; tách phép đo chất lượng recall khỏi độ trễ storage, rồi so với M0/M1.
Không cài thêm dịch vụ, đổi database hay chạy benchmark có phí trong nhiệm vụ này.
