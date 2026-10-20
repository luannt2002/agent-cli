# 20 — Bộ kịch bản baseline (Phase 7)

Mã: `scripts/eval/fixtures.py` (kịch bản + repo Go nhỏ có lỗi cài sẵn + test ẩn), `scripts/eval/run.sh` (chạy một kịch bản), `scripts/eval/score.py` (chấm + gộp bảng). Mỗi lệnh verify được kiểm là **ĐỎ trên fixture chưa sửa** (cổng đo đúng mục tiêu, không xanh sẵn): 8/8 kịch bản dùng verify cho rc=1 lúc đầu.

## Kịch bản

| ID | Đại diện cho | Fixture | Các bước | Chấm |
|---|---|---|---|---|
| s1-simple-bug | lỗi đơn giản | `calc` (Add trả a-b) | 1 | `go test ./...` |
| s2-multi-file-bug | lỗi nhiều tệp | `temp` (công thức + định dạng sai ở 2 tệp) | 1 | test + test không bị sửa |
| s3-feature | thêm tính năng | `strutil` (Reverse theo README) | 1 | test ẩn (unicode) |
| s4-refactor | refactor | `users` (đổi tên hàm khắp nơi) | 1 | test + không còn tên cũ |
| s5-debug-unknown | gỡ lỗi chưa rõ gốc | `stats` (median sai khi số phần tử chẵn; test hiện có vẫn xanh) | 1 | test ẩn |
| s6-test-integrity | test đỏ / chống "sửa test cho xanh" | `stack` (Pop trả sai đầu) | 1 | test + `stack_test.go` không đổi |
| s7-review | review | `window` + `change.diff` có 2 lỗi | 1 | câu trả lời phải nêu cả hai lỗi (từ khoá) |
| s10-resume | resume | `temp` | 2 (điều tra, rồi `-s` sửa) | test + test không đổi |
| s12-multi-agent | đa agent | `errs` (6 hàm, 3 hàm trả lỗi không bọc) | 1 (bảo dùng tool `agent`) | nêu đủ 3 tên |
| s13-provider-switch | đổi provider giữa phiên | `calc` | 2 (bước 2 `-m glm/glm-5.3-flash`) | test |

Kịch bản trong workflow chưa phủ ở bộ thật này: **compaction/resume** (chạy bằng LLM giả ở P-04, vì kích tóm tắt với model thật phải bơm đầy context) · **architecture task** (không chấm tự động được) · **Routine long-run** (E2E `scripts/test-agent-cli-routine.sh` với LLM giả; chạy thật để Phase 22).

## Số đo (mỗi lượt, `metrics.json`)

`passed`, `false_done` (bước cuối rc=0 + có trả lời, nhưng verify đỏ), `seconds`, `llm_calls` và `by_role` (từ sổ ghi), `providers`, `input_tokens`/`output_tokens` (tổng THẬT từng lời gọi), `first_main_input` (kích thước context ở bước đầu), `tool_calls` (từ `session show`), `llm_errors`.

## Môi trường

Binary đo baseline: `agent-cli-baseline-ledger` = `evo/baseline-20261016` + chỉ phần sổ ghi (5 tệp, 9 dòng thêm, không đổi hành vi; cùng vẫn gửi tiêu đề sang provider của small như baseline — kiểm ở probe `probe-bl-1`). Cấu hình thật của người dùng, skill thật trong HOME, dữ liệu trong thư mục ra, chạy in-process (`CRUSH_CLIENT_SERVER=0`), không đụng server đang chạy.

Lệnh tái lập: `bash scratchpad/evo/run-suite.sh <binary> <out-root>`; tổng hợp: `python3 -I scripts/eval/score.py summary '<out-root>/*'`.
