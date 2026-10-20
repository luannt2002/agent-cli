# 15 — Số trước / sau (§87)

TRƯỚC = binary gốc (`main` @ `4a0b25b6`). SAU, mục A = binary cuối `bin/agent-cli-final4` (ứng viên vòng 8,
`r8-final.patch`, `e4081c0b…`); mục B = `bin/agent-cli-final3` (vòng 7), vì bản vòng 8 chỉ khác ở đường client của chế độ
server mà bộ eval (chạy local) không đi qua; trên binary cuối có thêm smoke LLM thật qua server (SE [37](../system-evolution/37_FINAL_PROOF.md)).
Có thêm cột "trước vòng 6" (ứng viên lúc bàn giao Daily) để thấy phần do vòng 6–8 đổi.

## A. Đúng / an toàn (E3: binary thật + LLM giả, tái lập được)

| Đo | Trước | Sau | Lệnh |
|---|---|---|---|
| Lời gọi lọt sang provider của model nhỏ, kịch bản chỉ Qwen | 799 (title 533, agentic_fetch 266) | 0 | `scripts/eval/qwen-only.sh` |
| Tiêm lỗi provider: ca có request lọt sang glm | 8/8 | 0/11 | `scripts/eval/faults.sh` |
| Model đang chọn thiếu key | (ứng viên trước vòng 6) rc=0, 2 request sang glm | rc=1, 0 request, lỗi nêu provider (local và qua server); `login` chạy; TUI vào màn chọn model | `faults.sh` ca `selected-no-key`, `selected-no-key-server` |
| Model trả lời rỗng | rc=0, `text` rỗng (DONE giả) | rc=1 | `faults.sh` ca `empty-reply` |
| Tự tóm tắt lặp khi model làm lại việc | hết giờ 120 s, 2.659 request | dừng có lỗi sau 10 lần tóm tắt | `qwen-only.sh`, `faults.sh` ca `summary-loop` |
| Loop có cổng xanh sẵn, agent không đổi gì | (ứng viên trước vòng 6) GREEN rc=0 | `NO_CHANGE` rc=1 | `qwen-only.sh` (đường loop) |
| `text` của `run --json` (local) khi có tự tóm tắt | dính bản tóm tắt | chỉ câu trả lời | `qwen-only.sh` |
| Routine bị SIGKILL giữa lượt | bị che sau báo cáo cũ | `ABANDONED`, khoá nhả, lượt sau chạy được | `scripts/test-agent-cli-routine.sh` |
| Nhiều workspace / nhiều session | (chưa đo trên binary gốc) | 11/11 kiểm tra | `scripts/eval/workspaces.sh` |
| Suite Go so theo tên với cây sạch | 4.501 test | 4.637 test: 0 mất, 0 đổi trạng thái, 136 mới đều pass | `final-verify4.txt` |
| Race | — | 0 | như trên |

## B. Giải việc với LLM thật (E5, mỗi kịch bản một lần mỗi binary)

Nguồn: `scratchpad/evo/{baseline-real,final-real2,final-real3}/*/metrics.json`, `final-real3-s13b/` (chạy lại s13); tổng hợp
bằng `scratchpad/evo/compare-real.py` và một lệnh Python cùng logic. Provider: `main/qwen3.8` của người dùng; s13 bước 2
đổi có chủ ý sang `glm/glm-5.3-flash`.

| Kịch bản | Trước | Trước vòng 6 | Sau |
|---|---|---|---|
| s1-simple-bug | ✓ 18,5 s · 7 lời gọi | ✓ 35,1 s · 7 | ✓ 22,9 s · 7 |
| s2-multi-file-bug | ✓ 28,0 s · 8 | ✓ 21,1 s · 6 | ✓ 56,4 s · 7 |
| s3-feature | ✓ 38,5 s · 9 | ✓ 34,5 s · 9 | ✓ 16,9 s · 6 |
| s4-refactor | ✓ 21,6 s · 8 | ✓ 37,3 s · 8 | ✓ 28,2 s · 9 |
| s5-debug-unknown | ✓ 37,5 s · 10 | ✓ 57,0 s · 10 | ✓ 60,4 s · 9 |
| s6-test-integrity | ✓ 21,4 s · 7 | ✓ 20,7 s · 6 | ✓ 25,0 s · 7 |
| s7-review | ✓ 87,4 s · 9 | ✓ 124,9 s · 10 | ✓ 38,0 s · 5 |
| s10-resume | ✓ 33,9 s · 8 | ✓ 54,5 s · 9 | ✓ 34,2 s · 7 |
| s12-multi-agent | ✓ 39,0 s · 21 | ✓ 40,1 s · 14 | ✓ 43,0 s · 12 |
| s13-provider-switch | ✓ 44,7 s · 10 | ✓ 61,0 s · 11 | lần 1 ✗ (GLM không gửi dữ liệu 120 s, `timeout`; rc=1, không DONE giả, không đổi provider); chạy lại ✓ 34,3 s · 9 |
| s14-wrong-context | — | ✓ 18,3 s · 6 | ✓ 26,6 s · 6 |
| s16-review-shared | — | ✓ 41,4 s · 7 | ✓ 29,6 s · 5 |
| s17-single-agent | — | ✓ 16,8 s · 4 | ✓ 12,1 s · 4 |

Tổng trên 9 kịch bản có ở cả ba lần chạy, bỏ s13 (lần 1 của SAU bị provider treo):

| | Trước | Trước vòng 6 | Sau |
|---|---|---|---|
| Đạt / DONE giả | 9/9 · 0 | 9/9 · 0 | 9/9 · 0 |
| Lời gọi LLM | 87 | 79 | 69 |
| Lời gọi đặt tiêu đề | 24 | 9 | 9 |
| Token vào / ra | 1.196.842 / 21.154 | 1.315.511 / 23.711 | 1.108.932 / 15.799 |
| Tổng giây | 325,8 | 425,2 | 325,0 |
| Tool call | 76 | 82 | 70 |

Đọc số:

- Thành công giữ nguyên ở trần (mọi kịch bản đạt ở cả ba binary, tính cả lần chạy lại s13). Bộ này không đo được "giải giỏi
  hơn".
- Lời gọi tiêu đề giảm là thay đổi cấu trúc, lặp lại được: session con không gọi tiêu đề, không thử lại sau `length`
  (s12: 8 → 1).
- Token, thời gian, số lời gọi main dao động mạnh giữa các lần chạy cùng một binary (vd s2: 21,1 s ở lần trước vòng 6,
  56,4 s ở lần sau; s7: 124,9 s rồi 38,0 s). Mỗi kịch bản chỉ một lần chạy, nên chênh lệch tổng **chưa phải kết luận**.

## C. Checklist §87

| Câu hỏi | Trả lời | Căn cứ |
|---|---|---|
| Việc hằng ngày dễ hơn? | Một phần: ít ma sát hơn ở các ca hỏng (lỗi rõ thay vì im lặng, routine không bị che, hộp thoại không chặn chéo); README có luồng hằng ngày, lần chạy đầu | [02](02_FRICTION_MAP.md), [README_REVIEW](README_REVIEW.md) |
| Tính đúng cải thiện hay giữ nguyên? | Cải thiện ở các ca hỏng (mục A); giữ nguyên ở việc thường (mục B) | trên |
| Thành công đã kiểm cải thiện hay giữ? | Giữ (13/13 với lần chạy lại s13); thêm `NO_CHANGE` làm "xong" khó giả hơn | trên |
| Giảm chép ngữ cảnh tay? | Không đo được bằng số; resume giữ đạt (s10), đổi provider giữ mạch (s13) | [14](14_MEMORY_DAILY_VALUE.md) |
| Chặn lọt provider? | Có: 799 → 0; 8/8 → 0/10; thiếu key thì dừng | [10](10_PROVIDER_ISOLATION_RESULTS.md) |
| Resume chạy? | Có (s10 đạt ở cả ba) | trên |
| Nhiều workspace / nhiều session chạy? | Có (11/11) | [06](06_SESSION_WORKSPACE_TRACE.md) |
| Skill chọn lọc / có ích? | Chọn đúng loại việc khi được nạp; nạp dao động giữa các lần (5/10, 7/13, 4/13); có ích hay không không phân biệt được | [05](05_SKILL_TRACE.md), [11](11_SKILL_EFFECTIVENESS.md) |
| Giảm agent thừa? | Có: tiêu đề cho session con bỏ; nhiều agent không có ích ở việc cỡ này | [12](12_MULTI_AGENT_RESULTS.md) |
| Routine khôi phục? | Có | [13](13_ROUTINE_RECOVERY_RESULTS.md) |
| Kiểm chứng đáng tin? | Đáng tin hơn: pipefail, `NO_CHANGE`, cờ `touched_tests`/`touched_gate`; vẫn cần người duyệt diff vì agent sửa được test | README mục "Xong" |
| Bằng chứng còn tươi? | Có: mọi số của SAU chạy trên revision cuối | SE [37](../system-evolution/37_FINAL_PROOF.md) |
| Tri thức lịch sử giúp? | NOT_APPLICABLE (M0, không có memory dẫn xuất để so) | [14](14_MEMORY_DAILY_VALUE.md) |
| Memory không đè sự thật hiện tại? | s14 đạt (một lần); không có máy ép | [14](14_MEMORY_DAILY_VALUE.md) |
| Độ phức tạp có đáng? | Mã sản phẩm +1.357/−192 dòng trong 30 tệp đổi lấy các số ở mục A; không thêm hệ con mới (memory, task graph, dashboard) | [16](16_PRUNING_DECISIONS.md), SE FINAL_SYSTEM_AUDIT câu 23 |
