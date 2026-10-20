# 00 — Bàn giao V2 → V3

## Mốc V2 (cổng §2 của spec V3)

- Nhánh local `v2-final` (worktree `/home/luannt/application/opensource/agent-cli-v2`), 8 commit trên `main` @ `4a0b25b6`,
  CHƯA push. Cây cuối `128d744a`: trùng từng byte với cây đã kiểm (bản tiếp quản `be56dccb` + patch comment crushrc đã soát
  DAT), so bằng `git rev-parse HEAD^{tree}` với cây dựng trên index tạm.
- Mỗi commit qua `go build ./...` và `go test -run '^$' ./...` (biên dịch mọi test) trước khi commit.
- Trạng thái V2: SE **SYSTEM IMPROVED — WITH KNOWN LIMITATIONS**; Daily **DAILY WORKSPACE READY — WITH KNOWN LIMITATIONS**
  (`reports/daily-work/FINAL_DAILY_WORK_AUDIT.md`).
- Không tag (người dùng chỉ cho commit nhánh riêng).

## Bằng chứng V2 dùng lại được

| Loại | Ở đâu | Revision |
|---|---|---|
| Suite 4.637 test, race, 3 E2E, faults, qwen-only, workspaces | `reports/system-evolution/evidence/takeover-2026-10-19/` | bản tiếp quản = mã `v2-final` |
| Eval LLM thật 13 kịch bản (bảng) | `reports/daily-work/15_BEFORE_AFTER_METRICS.md`, log `evidence/final-logs/final-real3.log` | vòng 7 (đường mã local giống `v2-final`) |
| Audit Hindsight trước | SE 24–30, `opensource/luannt-hindsight-1610.md`, SE 40 (Memlake) | Hindsight `e00154496` (16/10) |
| Quyết định memory | SE 28 (7 bất biến giữ lại; bác memory bank/retain/reflect/consolidation/mental model), SE 29 (M0) | — |

`metrics.json` từng kịch bản của eval V2 nằm ở `/tmp` và đã mất khi máy khởi động lại 19/10: chỉ còn bảng tổng hợp và log.
Baseline V3 (§48) vì vậy phải chạy lại eval trên binary `v2-final`.

## Giới hạn V2 mang sang

Cổng WIN chưa có người dùng đi; model của workspace chứ không của session (PRB-05); không người trực là danh sách chặn,
không phải sandbox; lời hứa provider chỉ phủ lời gọi LLM do agent-cli phát ra; `go vet` copylocks có sẵn; lint chưa chạy.

## Đầu vào V3

`spec/V3_SPEC.md` (bản chép của tệp người dùng, sha256 `1e596ba2…`), `spec/OVERLAY_ISSUE_FORK_ARCHAEOLOGY.md`.
Nguồn nghiên cứu clone ở `/home/luannt/application/opensource/v3-src/` (bền, không ở `/tmp`).

## Ràng buộc môi trường

- Không có `gh`; API GitHub không token: 60 request/giờ, search 10/giờ (đo 19/10). Người dùng sẽ cài `gh` + login.
  Trước đó chỉ làm phần không cần API (clone, git log, đọc mã).
- Máy: tối đa 3 agent; lệnh Go qua `flock` + RAM trống ≥3 GB.
- Không push; POC V3 ở nhánh riêng từ `v2-final`.
