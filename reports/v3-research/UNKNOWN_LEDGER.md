# V3 — Sổ chưa biết (§60)

| ID | Câu hỏi | Vì sao quan trọng | Mức | Giả thuyết | Điều tra | Bằng chứng | Trạng thái |
|---|---|---|---|---|---|---|---|
| V3-U1 | Provider `main` (qwen3.8) của người dùng là local hay trả tiền? | Chạy lại eval LLM thật cho baseline V3 có tốn tiền không | U3 | local | hỏi người dùng | người dùng trả lời 19/10: local, miễn phí | ĐÓNG |
| V3-U2 | Không có `gh`: đào issue/fork có đủ sâu không? | Overlay coi issue/fork là bằng chứng hạng nhất | U3 | đủ khi có `gh` | chờ người dùng cài | quota 60/giờ | MỞ |
| V3-U3 | Pha DEW: tên `dew` có đụng namespace (GitHub, Go module, binary trên PATH) không? | DEW §42 bắt kiểm trước khi đổi | U2 | có va chạm (tên 3 chữ chung) | tra cứu khi tới pha DEW | — | MỞ |
| V3-U4 | Pha DEW: giữ hay đổi module path / `CRUSH_*` / `crush.json` (P3 giữ để merge upstream rẻ)? | Đổi = mất đường merge upstream rẻ; giữ = tên nội bộ lệch tên sản phẩm | U3 | giữ (DO_NOT_RENAME), chỉ đổi tên hiển thị/binary/docs | người dùng quyết ở pha DEW | CLAUDE.md P3 | MỞ |
| V3-U5 | Mốc ổn định V3 cần commit: lời cho phép commit V2 chỉ áp một lần | Cổng giữa V3 và DEW | U2 | — | hỏi 19/10 | người dùng: cho commit từng nhánh POC V3 sau khi soát DAT, local, không push | ĐÓNG (cho POC; mốc V3 cuối/merge vẫn hỏi) |
