# 29 — Cạnh tranh các cách giữ tri thức (Phase 13)

Bài toán: giảm khám phá lại giữa các phiên mà không để tri thức cũ đè sự thật hiện tại. So từ nấc thấp nhất của thang M0–M9 (§51). Nấc cao hơn không tự động tốt hơn.

| Ứng viên | Nấc | Đang có? | Chi phí mỗi request | Cách cũ/sai bị phát hiện | Kiểu hỏng đã biết | Bằng chứng |
|---|---|---|---|---|---|---|
| Context file (`AGENTS.md`, `CRUSH.md`, `CLAUDE.md`…) nạp mọi phiên | M0 | Có | toàn văn tệp (repo này: CLAUDE.md 17.873 B + AGENTS.md 10.879 B, đo `wc -c` 16/10) | người duyệt qua git diff; agent đọc nguồn khi nghi | chỉ nạp ở cwd, không nạp tên thường (PRB-19/20); agent con không nhận | E1 (load.go) + E3 (P-03: agent con 1 KB system) |
| Lịch sử session + `-s`/`-C` | M0 | Có | toàn bộ lịch sử tới lúc tóm tắt | — (nguyên văn) | cùng session mới dùng được; resume đổi model workspace (PRB-05) | E5: s10, s13 đạt với LLM thật |
| Tóm tắt tự động khi gần đầy context | M0 | Có | bản tóm tắt thay lịch sử | — | bản bị cắt từng được nhận (đã sửa); vòng tóm tắt vô hạn (đã sửa) | E3: P-04 + test overflow |
| Journal của loop (chữ ký lỗi trong một lần chạy) | M0 | Có | nhỏ | xoá theo lần chạy | không mang sang lần chạy sau | E2 |
| Tệp bài học có cấu trúc (một bài một mục, có nguồn, có ngày) theo quy ước, nạp như context file | M1 | Không (chỉ cần quy ước, không cần mã) | tỉ lệ số bài học | người duyệt; mỗi mục ghi lệnh kiểm lại | phình to; cũ mà không ai xoá | E0 cho agent-cli. Chính người dùng đang dùng kiểu này cho Claude Code (`MEMORY.md` + tệp từng fact) |
| M1 + provenance + vòng đời (supersede/invalidate) | M2 | Không | như M1 | trường `source`, `verified_at`, lệnh kiểm | phải có người/agent bảo trì | E0 |
| Truy hồi đơn giản (grep/BM25 trên lịch sử session) | M3 | Không | theo truy vấn | — | trả nhiều kết quả nhiễu | E0 |
| Hindsight-like (M4–M9) | M4+ | Không | lời gọi LLM thêm cho retain/consolidate/reflect + dịch vụ ngoài | yếu (H-22, H-23, H-25) | H-08, H-17..H-28 | E0 cho agent-cli; E1–E3 cho Hindsight |

## Đánh giá

- M0 đã phủ hai tình huống đo được: resume cùng session (s10) và đổi provider giữa chừng (s13). Cả hai đạt với LLM thật.
- Lỗ đo được của M0 nằm ở **kênh nạp**, không nằm ở việc thiếu kho memory: context file không nạp ở git root, không nạp tên thường (PRB-19/20); agent con không nhận context. Sửa kênh nạp rẻ hơn mọi nấc M1+.
- M1 không cần mã: một quy ước tệp bài học, nạp qua cơ chế context file sẵn có. Nó là bước tiếp theo **nếu** kịch bản Daily I (bài học lịch sử) thất bại.
- M3+ không có vấn đề nào chứng minh được để biện minh.

**Thắng:** M0 + sửa kênh nạp context file. **Dự bị:** M1 bằng quy ước, chỉ khi Daily I/J chứng minh cần.
