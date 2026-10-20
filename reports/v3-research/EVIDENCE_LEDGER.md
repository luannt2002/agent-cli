# V3 — Sổ bằng chứng

Mỗi dòng: kết luận · mức bằng chứng (E0–E7 theo spec §5) · lệnh/tệp tái lập.

| # | Kết luận | Mức | Tái lập |
|---|---|---|---|
| 1 | Cây `v2-final` = cây đã kiểm | E1 (so cây) + E3/E4 (bộ kiểm trên cây đó) | `git -C agent-cli-v2 rev-parse HEAD^{tree}` = `128d744a…`; receipt `evidence/takeover-2026-10-19/` |
| 2 | Context Mode: bản sửa mất sự kiện (`4164d3de`) chỉ áp cho hook Claude Code; 8 hook host khác vẫn DELETE mồ côi vô điều kiện (7 gọi thẳng, kimi qua `pruneOrphanedEvents()`) | E1 | `grep -rln 'NOT IN (SELECT session_id FROM session_meta)' hooks src`; `hooks/kimi/sessionstart.mjs:99`; `src/session/db.ts:1718` |
| 3 | Context Mode: payload có `agent_id`/`agent_type` bị coi là không dùng được ctx_* | E1 | `v3-src/context-mode/hooks/pretooluse.mjs:167` |
| 4 | Context Mode: `main` đóng băng từ 30/06 (chỉ commit bot CI); phát triển thật ở `origin/v2-design` (+737 commit) | E1 | `git rev-list --count origin/main..origin/v2-design` |
| 5 | agent-cli: MCP instructions tiêm vào agent không có tool MCP | E1 | `agent-cli-v2/internal/agent/agent.go:961-973`; `internal/config/config.go:1127,1138` |
| 6 | agent-cli: `run -s <session cũ>` ghi model session đó vào cấu hình workspace | E1 | `agent-cli-v2/internal/cmd/run.go:1010` (`ScopeWorkspace`) |
| 7 | claude-mem: plugin DSH ghi project theo `basename(cwd)`, đọc bằng cách gửi cả `cwd` cho worker tự giải | E1 (phía ghi); phía đọc phụ thuộc resolver của worker | `v3-src/claude-mem/src/integrations/dsh-plugin/hooks.ts:96-103` vs `:50-55` |
| 8 | claude-mem: spool coi là đã giao (`markHandedOff`) ngay khi sự kiện vào SessionManager, trước việc bền; hàng đợi bền đã bị xoá ở `61fe70a20` (theo báo cáo) | E1 | `v3-src/claude-mem/src/services/worker/http/shared.ts:102-111,335-336` |
| 9 | Hindsight #5251 PARTIALLY_FIXED ở HEAD `5331da2f`: lỗi chỉ phát hiện ở lượt ghi kế tiếp (`settleOps` chỉ chạy khi append), theo dõi tối đa 8 op, `retain_ok` ghi ngay khi accept → lượt ghi cuối của session hỏng thì mất không ai biết | E1 | `v3-src/hindsight/hindsight-integrations/coding-agents/src/core/{chat.ts:336-337,retain-cursor.ts:130,retain-hook.ts:166}` |
| 10 | agent-cli phơi nhiễm, bản ĐÃ SOÁT: đợt 1 giữ 9 ca (T2, T3, T5, T6, T11, T12, T13, T15 thu hẹp, T17), T1 thu hẹp, T21 chỉ là phép đo (WORKFLOW_PROGRESS 20); đợt 2 xem dòng 15–16 và `experiments/00` (đang soát lại) | E3 (binary + LLM giả) / E2 (unit) | `go test -run TestV3Exposure_ …`; `bash scripts/eval/exposure.sh <dir-mới> <bin>` trên worktree `agent-cli-v3`; số lấy từ `verdicts.txt` của thư mục chạy (`~/.cache/v3ev/r2`, `r3`, `m1`), không lấy từ bảng tóm tắt |
| 11 | Hindsight: `RetainCursor` 4 trường (`aaa58f247`) → 8 trường (`d3f0001ed`), mỗi bản sửa thêm một trường; client không đọc lại trạng thái document từ server | E1 | `git show <c>:hindsight-integrations/coding-agents/src/core/retain-cursor.ts` đếm trường của `interface RetainCursor` |
| 12 | Hindsight: tag filter mặc định `all_strict` khi có `model_tags` (gốc của chuỗi #2670/#2808 → #5728) | E1 | `v3-src/hindsight/.../memory_engine.py:1866` |
| 13 | claude-mem: observer mặc định fallback sang Anthropic khi provider được chọn không phục vụ được (PR #4727) — đúng kiểu fallback chéo provider mà agent-cli V2 đã gỡ | E1 | `git -C v3-src/claude-mem show 3dc125f2d` |
| 14 | claude-mem: bản sửa scope mặc định (#1911) chỉ ở `canary`, chưa vào `main`; HEAD có test khoá "tìm không scope vẫn không scope" | E1/E2 | `git merge-base --is-ancestor 64fc4f406 origin/main` → không; `tests/servers/checkout-search-scope.test.ts:21-24` |
| 15 | agent-cli V2: server panic khi 10 `run` đồng thời | E3 | `bash scripts/eval/exposure.sh <dir-mới> <bin> T20` (worktree `agent-cli-v3`); `~/.cache/v3ev/m1.out` 4/10 |
| 16 | agent-cli V2: allowlist/env của client sau bị bỏ qua khi server đã dựng | E3 | `exposure.sh … C5`; `~/.cache/v3ev/m1.out` |
