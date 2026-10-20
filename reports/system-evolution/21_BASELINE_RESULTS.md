# 21 — Kết quả baseline (Phase 7)

Chạy 2026-10-16 18:29–18:38, model thật `main/qwen3.8` (bước 2 của s13 dùng `glm/glm-5.3-flash`), **một lần mỗi kịch bản** (chưa đo biến thiên giữa các lần — U-7 ở tệp 19). Dữ liệu thô: `scratchpad/evo/baseline-real/<kịch bản>/` (ledger.jsonl, step-*.json, verify.out, metrics.json).

| Kịch bản | Đạt | Báo xong giả | Giây | Lời gọi LLM | Token vào / ra | Tool call | Vai trò |
|---|---|---|---|---|---|---|---|
| s1-simple-bug | ✓ | — | 18,5 | 7 | 112.685 / 748 | 7 | main 5, title 2 |
| s2-multi-file-bug | ✓ | — | 28,0 | 8 | 138.569 / 1.633 | 9 | main 6, title 2 |
| s3-feature | ✓ | — | 38,5 | 9 | 160.030 / 2.321 | 10 | main 7, title 2 |
| s4-refactor | ✓ | — | 21,6 | 8 | 118.055 / 1.281 | 9 | main 6, title 2 |
| s5-debug-unknown | ✓ | — | 37,5 | 10 | 190.997 / 2.306 | 10 | main 8, title 2 |
| s6-test-integrity | ✓ | — | 21,4 | 7 | 97.546 / 946 | 6 | main 5, title 2 |
| s7-review | ✓ | — | 87,4 | 9 | 156.969 / 5.782 | 9 | main 7, title 2 |
| s10-resume | ✓ | — | 33,9 | 8 | 122.128 / 2.129 | 9 | main 6, title 2 |
| s12-multi-agent | ✓ | — | 39,0 | 21 | 99.863 / 4.008 | 7 | main 4, subagent 9, title 8 |
| s13-provider-switch | ✓ | — | 44,7 | 10 | 75.432 / 794 | 6 | main 8, title 2; qwen 5 + glm 5 |
| **Tổng** | **10/10** | **0** | **370,5** | **97** | **1.272.274 / 21.948** | 82 | title **26** |

## Đọc kết quả

1. **Model đủ mạnh cho bộ việc nhỏ này.** Tỉ lệ đạt 100%, không lần nào báo xong giả. Cải thiện của đợt này vì vậy không đo được bằng "tỉ lệ đạt" trên bộ này. Phải đo bằng hiệu quả (token, số lời gọi, thời gian) và bằng các kịch bản hỏng có chủ ý (kill, tràn context, provider) ở Phase 21–22.
2. **Tiêu đề bằng LLM: 26/97 lời gọi (27%), 26/26 kết thúc `length`.** Không lời gọi nào sinh được tiêu đề. Đây là chi phí thuần, và nó tăng theo số agent con: s12 có 8/21 lời gọi là tiêu đề.
3. **Phần cố định của context là chi phí lớn nhất.** Lời gọi main đầu tiên tốn 18.605–18.621 token vào trong mọi kịch bản, trước khi đọc code. Token vào trung bình 13.116 mỗi lời gọi.
4. **Đổi provider giữa phiên chạy được** (s13: 5 lời gọi qwen rồi 5 lời gọi glm, vẫn đạt). Sổ ghi cho thấy rõ ranh giới. Model hiển thị trong UI không phản ánh điều này (PRB-05 vẫn mở).
5. **Đa agent:** 9 lời gọi của agent con nhưng chỉ 99.863 token vào, vì system prompt của agent con chỉ 1 KB. Agent con không có skill hay context, và việc này vẫn đạt.

## Giới hạn

Một lần chạy mỗi kịch bản. Bộ việc nhỏ, chỉ có Go. Provider `main` chưa biết là local hay trả phí (U-1). Không có kịch bản compaction thật hay architecture.
