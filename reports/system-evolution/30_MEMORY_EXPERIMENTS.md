# 30 — Thí nghiệm memory (Phase 14–18)

Không nhận subsystem memory nào ([28](28_HINDSIGHT_DECISIONS.md)), nên phần lớn thí nghiệm memory không có đối tượng để chạy. Ghi rõ từng cái để không ai đọc "không có số" thành "đã đạt".

| Thí nghiệm | Trạng thái | Lý do / kết quả | Chạy khi |
|---|---|---|---|
| M0 giữ được liên tục cùng session (resume) | **ĐÃ CHẠY** | s10-resume đạt với LLM thật (bước 2 dùng lịch sử bước 1). E5, một lượt | — |
| M0 giữ liên tục khi đổi provider giữa chừng | **ĐÃ CHẠY** | s13-provider-switch đạt; lượt sau đi đúng provider mới. E5, một lượt | — |
| Tóm tắt không phá liên tục | **MỘT PHẦN** | Cơ chế (vòng lặp, bản bị cắt) kiểm bằng LLM giả (E3). Chất lượng nội dung tóm tắt với LLM thật: CHƯA ĐO | kịch bản compaction thật ở Daily |
| Chi phí context cố định | **ĐÃ CHẠY** | 79 KB phần cố định, skill chiếm 60% system prompt (P-05). E3 | đo lại sau khi áp chính sách skill |
| M1 (tệp bài học) có giảm khám phá lại | **CHƯA CHẠY** | không có vấn đề M0 chứng minh được để so | Daily I thất bại |
| Memory cũ/sai bị bắt khi đối chiếu nguồn | **CHƯA CHẠY** (cho memory) | Không có memory tự sinh. Với context file: Daily J sẽ đặt một khẳng định sai vào context file và kiểm agent có đối chiếu nguồn không | Daily J |
| Vòng nhân bản retain → recall | **NOT_APPLICABLE** | không có retain | thêm retain |
| Memory hỏng không làm hỏng CLI | **NOT_APPLICABLE** | không có dịch vụ memory. Context file thiếu/hỏng: hiện có đường bỏ qua (E1, load.go), chưa test | thêm memory ngoài tiến trình |
| Reviewer có memory vs không | **REJECT không chạy** | reviewer cố ý không nhận memory hay kết luận của người làm (luật lõi 1) | — |
| Recall tiếng Việt (Q-13) | **NOT_APPLICABLE** | không có reranker | thêm recall có reranker |

Kết luận: memory dài hạn hiện chưa có bằng chứng giá trị lẫn bằng chứng hại ở agent-cli, vì chưa có. Mọi tuyên bố "memory giúp" phải chờ Daily B/I/J.
