# 22 — Harness đánh giá hệ thống (Phase 8)

Mục tiêu: đo hành vi của CẢ hệ thống (binary thật, provider giả hoặc thật), ra kết quả máy đọc được, lặp lại được. Không thay bộ test Go; bổ sung tầng mà unit test không phủ.

## Thành phần (đều nằm trong repo, chưa commit)

| Tệp | Vai trò |
|---|---|
| `scripts/eval/fakellm.py` | LLM giả tương thích OpenAI, nhiều provider trên một cổng (đoạn đường dẫn trước `/v1`), ghi mọi request (provider, model, vai trò đoán từ system prompt, kích thước, tool), lưu system prompt theo sha; luật kịch bản theo bước (`steps[n]` = số tin tool sau tin user cuối); usage ≈ byte/4 để kích compaction |
| `scripts/eval/fixtures.py` | 10 kịch bản (repo Go nhỏ có lỗi cài sẵn, test ẩn ngoài repo của agent, lệnh verify, cách chấm) |
| `scripts/eval/run.sh` | chạy một kịch bản: dựng repo, chạy `run --json` từng bước (resume bằng `-s`, đổi model bằng `-m`), chạy verify NGOÀI tầm agent, `session show`, ghi kết quả |
| `scripts/eval/score.py` | `metrics.json` cho từng lượt (đạt, báo xong giả, giây, lời gọi theo vai trò/provider, token thật từ sổ ghi, tool call) và `summary` ra bảng |
| `AGENT_CLI_LLM_LEDGER` (trong binary, worktree) | sổ ghi lời gọi LLM — nguồn token/provider THẬT cho cả provider thật |
| scratch: `probe-provider-roles.sh`, `probe-context.sh`, `run-suite.sh` | probe cô lập provider, thành phần context, chạy cả bộ |

## Chế độ

- **Provider giả** (`EVAL_GLOBAL_CONFIG=<thư mục có crush.json trỏ fakellm>`): kiểm cơ chế (định tuyến provider, compaction, false DONE, luồng resume). Rẻ, tất định, chạy được trong CI. Đã dùng: P-01..P-07, chạy khô harness (`evalfake-s1`: bắt đúng `false_done=true` khi agent giả trả lời mà verify đỏ).
- **Provider thật** (cấu hình thật của người dùng, dữ liệu tách riêng, in-process): đo thành công việc và hiệu quả. Đã dùng cho baseline (21).

## Thiết kế chống tự lừa

- Verify chạy sau khi agent xong, ngoài tầm agent; test ẩn chép vào lúc verify rồi xoá.
- Mỗi lệnh verify được kiểm ĐỎ trên fixture chưa sửa (8/8).
- s2/s6/s10 kiểm cả "tệp test không bị đổi" → bắt kiểu "sửa test cho xanh".
- Token lấy từ sổ ghi (usage từng lời gọi), không từ `run --json` (chỉ bước cuối).
- So sánh luôn theo cùng fixture, cùng cấu hình, ghi rõ binary.

## Giới hạn đã biết

- Một lượt mỗi kịch bản ở baseline (chưa đo biến thiên).
- Fixture chỉ Go; repo nhỏ — không đo hành vi trên repo lớn.
- Chấm review/đa agent bằng từ khoá (yếu).
- `run.sh` dùng `env -i` + `HOME` thật để có skill thật; không chạy được trên máy không có cấu hình agent-cli.
- Chưa có kịch bản tự động cho: compaction với model thật, server SIGKILL giữa tool call với provider thật, routine dài.
