# 12 — Kết quả nhiều agent (Daily G, §94.6)

Câu hỏi: khi nào nhiều agent có ích, và agent-cli có giữ số agent ở mức tối thiểu đủ dùng không?

## Cơ chế

Agent con duy nhất của agent-cli là tool `agent`: một agent chỉ đọc (glob, grep, ls, view), dùng model chính, chỉ chạy khi
model gọi tool này. Không có planner hay worker nền. `agentic_fetch` là một agent con đọc web, chạy model nhỏ (crushrc mẫu
tắt nó).

## Cùng việc, hai cách (E5, LLM thật, một lần mỗi binary)

Việc: liệt kê mọi hàm trả lỗi mà không bọc bằng `fmt.Errorf` + `%w` trong fixture `errs`. Chấm bằng từ khoá (`ReadConfig`,
`Remove`, `Stat`). s12 bảo model dùng tool `agent` để xem song song; s17 bảo model tự xem, không dùng `agent`.

| Kịch bản | Binary | Đạt | Lời gọi LLM | Theo vai trò | Token vào / ra | Giây | Tool call |
|---|---|---|---|---|---|---|---|
| s12 nhiều agent | gốc | ✓ | 21 | main 4, subagent 9, title 8 | 99.863 / 4.008 | 39,0 | 7 |
| s12 nhiều agent | ứng viên trước vòng 6 | ✓ | 14 | main 4, subagent 9, title 1 | 92.214 / 3.166 | 40,1 | 7 |
| s17 một agent | ứng viên trước vòng 6 | ✓ | 4 | main 3, title 1 | 53.896 / 865 | 16,8 | 5 |
| s12 nhiều agent | binary cuối | ✓ | 12 | main 4, subagent 7, title 1 | 88.133 / 2.787 | 43,0 | 5 |
| s17 một agent | binary cuối | ✓ | 4 | main 3, title 1 | 53.547 / 628 | 12,1 | 4 |

Nguồn: `scratchpad/evo/{baseline-real,final-real2,final-real3}/s1{2,7}-*/metrics.json`.

## Kết luận

- Ở việc cỡ này, nhiều agent **không có ích**: cùng kết quả, nhưng s12 tốn 14 lời gọi và 40,1 s so với 4 lời gọi và 16,8 s của
  s17 (ứng viên trước vòng 6); trên binary cuối 12 lời gọi và 43,0 s so với 4 lời gọi và 12,1 s. Chưa có kịch bản nào cho thấy nhiều agent đạt mà một agent trượt.
- Giảm agent thừa đo được: lời gọi đặt tiêu đề cho session con bị bỏ (s12: title 8 → 1, tổng 21 → 14 lời gọi).
- Mặc định đã là "tối thiểu đủ dùng": agent con chỉ chạy khi model gọi; không có vai trò agent cố định nào.
- Chưa đo: việc lớn thật sự (nhiều thư mục, đọc song song có lợi về thời gian thực); một lần chạy mỗi kịch bản nên con số
  thời gian chỉ để tham khảo.
