# 34 — Cổng mạch lạc cho mọi thứ thêm vào (Phase 24)

Gánh chứng minh nằm ở phần phức tạp mới. Mỗi thứ mà đợt này THÊM được hỏi đủ các câu ở §63. "Không" là câu trả lời tốt.

| Thứ thêm vào | Vấn đề đã chứng minh | Chủ hiện tại | Quyền thứ hai? | State bền mới? | LLM call / context mới? | Worker / bảng DB mới? | Kiểu hỏng mới | Cải thiện đo được |
|---|---|---|---|---|---|---|---|---|
| Lỗi "context window too small" sau tóm tắt | PRB-01 (755+755 lời gọi / 90 s) | vòng agent (`StopWhen`) | không | không | **bớt** lời gọi | không | lượt dừng có lỗi khi window quá nhỏ (đúng ý) | P-04: vòng vô hạn → 1 lần tóm tắt + lỗi |
| Tóm tắt bị cắt = lỗi | PRB-12 | `summarize` | không | không | không | không | người dùng thấy lỗi thay vì mất lịch sử lặng lẽ | test |
| Dừng vì lặp = lỗi | PRB-27 | vòng agent | không | không | không | không | rc=1 thay vì 0 | test |
| Không gọi tiêu đề cho session con, không thử lại sau `length` | PRB-46, PRB-74 (26/97 lời gọi) | `GenerateTitle` | không | không | **bớt** lời gọi | không | không | đo lại ở 37 |
| small phải cùng provider với large | PRB-03, faults 8/8 → 0/8 | `buildAgentModels` (điểm chặn duy nhất) | không: dùng chính điểm chặn có sẵn | không | không | không | small khác provider bị thay bằng large (cảnh báo một lần) | faults.sh: 0 lời gọi lọt |
| `disabled_tools` áp vào agentic_fetch | PRB-09 | coordinator | không | không | không | không | agentic_fetch biến mất khi tắt cả hai tool web | test |
| Sổ ghi LLM (`AGENT_CLI_LLM_LEDGER`) | PRB-32 (token chỉ bước cuối); cần nguồn thật cho đánh giá | **mới**, bọc model ở `buildAgentModels` | không: chỉ ghi, không ai đọc để quyết định | **có**: tệp JSONL, chỉ khi bật env | không | không | ghi đĩa lỗi → bỏ qua, không làm hỏng lượt | P-06: khớp 5/5 với server giả; là nguồn số cho 21/37 |
| Hộp thoại quyền: semaphore huỷ được | PRB-06 | `permission.Service` | không (thay mutex) | không | không | không | không | test |
| Từ chối ghi ra ngoài khi không người trực | PRB-70 | `permission.Request` | không | không | không | không | routine muốn ghi ngoài worktree sẽ bị từ chối (đúng ý) | test |
| Chặn `git -C`, viết tắt option | PRB-25, PRB-67 | lớp chặn không người trực | không | không | không | không | không | test |
| Bỏ thư mục ẩn khi quét skill | PRB-08 | `skills.Discover` | không | không | **bớt** context (4.691 B/request) | không | skill đặt trong thư mục ẩn không còn được thấy | probe P-05 |
| Báo cáo routine RUNNING/ABANDONED, ghi nguyên tử, tệp đổi + cờ | PRB-02, PRB-66 | `cmd/routine` (tệp báo cáo có sẵn) | không: trạng thái suy ra từ báo cáo + khoá flock có sẵn | thêm trường trong tệp có sẵn | không | không | không | test + E2E mục 8 |
| Biến launcher trong lịch in ra | PRB-07 | `routine.ScheduleText` | không | không | không | không | giá trị lạ bị từ chối | test |
| Sổ App còn sống trong tiến trình | PRB-15 | **mới**, `app` (map + mutex) | không: chỉ quyết "App cuối" | trong RAM | không | không | MCP đóng rồi không mở lại được trong cùng tiến trình (có từ trước, nay hẹp hơn) | test |
| Kiểm chủ job | PRB-18 | tool job (dùng `Owner` có sẵn) | không | không | không | không | job không có session chủ không còn với tới qua tool | test |
| `exit_code` trong metadata job_output | PRB-31 | tool job + UI | không | thêm trường JSON trong metadata | không | không | không | test |
| `provider`/`model` trong `run --json` | PRB-45 | `cmd/run` | không | không | không | không | không | test E2E (37) |
| Harness `scripts/eval/`, `faults.sh` | cần đo hệ thống | công cụ dev, ngoài binary | không | không | không | không | không | là nguồn của 21, 33 |

## Thứ được cân nhắc và KHÔNG thêm

- Subsystem memory (28–30): không có vấn đề chứng minh được.
- Trần thời gian riêng cho `run`: script bọc `timeout`; `loop`/`routine` đã có `--run-timeout`.
- Hỏi quyền cho `job_kill`: kiểm chủ đã chặn lấn session; hỏi quyền thêm ma sát mà không có vấn đề đo được.
- Ghi nhớ "model này luôn `length` khi đặt tiêu đề" để bỏ hẳn lời gọi: tiết kiệm nhỏ, thêm state.

Kết luận: chỉ hai thứ thêm state mới (sổ ghi LLM, opt-in; sổ App trong RAM). Không thêm LLM call, worker, bảng DB hay context nào. Ba thay đổi **bớt** lời gọi/context.
