# 27 — Vấn đề agent-cli × khái niệm Hindsight (Phase 11)

Bắt đầu từ VẤN ĐỀ ĐÃ CHỨNG MINH của agent-cli, không từ tính năng Hindsight. Không có vấn đề chứng minh được → không nhận tính năng.

| Lớp vấn đề | Bằng chứng ở agent-cli | Giải pháp hiện có | Khoảng trống | Giải pháp đơn giản nhất | Khái niệm Hindsight có thể giúp | Hỏng đã biết của nó | Kết luận |
|---|---|---|---|---|---|---|---|
| Mất tri thức giữa các phiên | Chưa đo. Baseline: 10/10 việc độc lập đều đạt, chưa có kịch bản phụ thuộc phiên trước | context file (AGENTS.md…) được nạp mọi phiên; template dặn cập nhật "memory files" | context file chỉ ở cwd, tên thường không nạp (PRB-19/20); không có context toàn cục trên máy | sửa PRB-19/20; thêm `~/.config/agent-cli/CRUSH.md` (đề xuất) | Memory Bank + retain/recall | thành công giả, trùng lặp, cũ thắng mới (H-08, H-22) | **Chưa đủ bằng chứng** → không nhận memory; sửa kênh có sẵn |
| Quên quyết định kiến trúc | Chưa đo | DECISIONS.md, CLAUDE.md của dự án (người viết) | không tự động | giữ: quyết định là tài liệu có người duyệt | knowledge pages / mental models | tài liệu dẫn xuất cũ, mâu thuẫn (H-23, H-25) | REJECT tự sinh |
| Lặp lỗi gỡ rối / cách đã hỏng | Chưa đo | journal loop có chữ ký lỗi trong 1 lần chạy; STALLED khi lặp | journal không đưa sang lần chạy routine sau | DEFER: đưa chữ ký lỗi lần trước vào prompt routine **chỉ khi** đo được routine lặp lỗi | observation "failed approach" | tri thức âm tính cũ không được kiểm lại (MEM-024) | DEFER |
| Mất ngữ cảnh khi compaction | P-04: vòng lặp (đã sửa); chất lượng bản tóm tắt chưa đo | summary.md có cấu trúc | bản bị cắt từng được nhận (đã sửa) | giữ; đo bằng kịch bản compaction thật sau | compaction recall | — | KEEP_EXISTING |
| Resume sau thời gian dài | s10/s13 đạt | session lưu trong SQLite, `-s`/`-C` | resume đổi model của workspace (PRB-05) | sửa PRB-05 | — | — | KEEP_EXISTING + sửa |
| Khám phá lại repo mỗi phiên | Đo: 18.605 token cố định trước khi làm gì; agent con không có context | context file; agent con chỉ có prompt | agent con khám phá lại | thử nghiệm: cho agent con context file (tốn token) — chưa có bằng chứng lợi | git-history seeding | seeding chạy nền, chưa có memory ở lượt đầu | REJECT seeding; DEFER context cho agent con |
| Reviewer thiếu ngữ cảnh lịch sử | Không có reviewer tích hợp | review bằng agent con chỉ-đọc (phương pháp) | — | reviewer nhận yêu cầu + diff + bằng chứng hiện tại, KHÔNG nhận giả định của người làm | memory cho reviewer | nhiễm giả định chung (Phase 18) | REJECT memory cho reviewer |
| Agent con khám phá trùng | Cấu trúc (P-03), không đo được hại (s12 đạt) | prompt do cha viết | — | không làm gì | memory chia sẻ | khuếch đại trùng lặp (H-28) | REJECT |
| Chi phí context lớn | **Đo:** 79 KB phần cố định, skill 60% system prompt | — | không ngân sách context | tắt skill không dùng (đề xuất), bỏ `.trash` (đã làm) | (không phải memory) | — | giải bằng chính sách skill |
| Khó tìm bằng chứng lịch sử | Chưa đo | `session show`, journal, report, sổ ghi LLM (mới) | không tìm kiếm xuyên phiên | DEFER | BM25/semantic recall trên lịch sử | — | DEFER |

**Kết luận Phase 11:** Không lớp vấn đề nào đòi memory dài hạn có bằng chứng ở agent-cli hôm nay. Các vấn đề đo được (chi phí context, cô lập provider, khôi phục, thành công giả) đều giải bằng sửa runtime/cấu hình có sẵn, không bằng thêm subsystem memory.
