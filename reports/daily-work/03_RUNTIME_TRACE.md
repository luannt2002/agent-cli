# 03 — Dấu vết runtime, tiến hoá theo git, dữ liệu lưu (§23, §24, §54, §55)

## 1. Một lượt làm việc hằng ngày đi qua đâu (khớp mã ứng viên)

Chi tiết từng tầng ở [02_RUNTIME_FLOW_MAP](../system-evolution/02_RUNTIME_FLOW_MAP.md) và [03_AUTHORITATIVE_STATE](../system-evolution/03_AUTHORITATIVE_STATE.md). Tóm tắt:

```
agent-cli (wrapper: env, socket theo hash dự án)
  → server của dự án (backend: workspace = App theo gốc dự án)
    → session (SQLite .crush/crush.db của dự án)
      → Run (RunID; hàng đợi prompt; huỷ theo RunID)
        → coordinator.buildAgentModels  ← điểm chặn provider duy nhất (allowlist execpolicy; small ép về provider của large)
          → bước model (fantasy) → tool (quyền → lớp chặn shell → thực thi)
          → tự tóm tắt (tối đa 10 lần/prompt) / tự làm tiếp khi bị cắt (tối đa 3)
        → RunComplete (lỗi nếu: lặp, tràn context, quá trần tóm tắt, chuỗi không ra gì, bị từ chối)
  → loop/routine: verify tất định sau lượt; routine trong worktree riêng, chế độ không người trực
```

Phân loại (§23) cho các năng lực hằng ngày: IMPLEMENTED — workspace/session/run, resume, compaction, job nền có chủ, loop, routine, provider allowlist, cô lập lời gọi ẩn, không đổi provider khi model đang chọn không dùng được (ứng viên; vòng 6–7). PARTIAL — model của session (vẫn là của workspace, PRB-05), context file (cwd, tên viết hoa, PRB-19/20). MISSING (cố ý) — task graph, Agent Map, memory riêng. UNVERIFIED — các phím Alt+W/S/N trong terminal thật, giao việc Claude/Codex khi bật.

## 2. Tiến hoá theo git (§24)

Nguồn: agent phân tích chỉ-đọc, mọi số từ lệnh git (`rev-list --count`, `diff --shortstat` từng khoảng). Điểm fork `a73e1bfa` (upstream #4067); 65 commit của fork (63 + 2 merge nội bộ), 14/10 → 16/10; 0 lần merge upstream, 0 revert; `origin/main` = `8568a886` (v0.2.0), `main` đi trước 8 commit chưa push.

| # | Thời kỳ | Khoảng | Commit | Diff | Áp lực thiết kế → trách nhiệm mới |
|---|---|---|---|---|---|
| E1 | Khởi dựng | c330628e..28259e14 | 7 | 150 tệp +19.192/−1.410 | nhiều session × nhiều dự án trong một terminal → Home, server mỗi dự án, `run/send/ps`, Stop hook, giao việc |
| E2 | Vá theo audit trước khi gộp | 9186c66b..d27ce38e | 9 | 38 tệp +1.138/−151 | ranh giới tin cậy → hỏi quyền khi giao việc, installer có sao lưu, DECISIONS.md |
| E3 | Ma sát hằng ngày | 133eb476..d4ccc579 | 5 | 27 tệp +2.064/−93 | dùng GLM thật mỗi ngày → chọn provider rồi model, Alt+P/M/E, hộp thoại quyền mặc định Deny |
| E4 | Cổng WIN + ranh giới | 461b540f..7a7fda8f | 7 | 219 tệp +19.807/−1.116 | job có chủ, run bị gián đoạn, `execpolicy` (allowlist), BlockFunc ở lớp thực thi |
| E5 | Chống output hỏng của model | a4b0286a..3a1cb9d6 | 5 | 9 tệp +808/−19 | model local/OpenAI-compat là agent chính → làm sạch tool call, chính sách vá |
| E6 | Đánh bóng UI → v0.1.0 | dd50cae7..87233bc8 | 12 | 50 tệp +1.439/−193 | "xong = verify" (hook mẫu), tên session đặt tay |
| E7 | Audit sâu → v0.2.0 | b7f02e4b..8568a886 | 12 | 23 tệp +1.266/−36 | X-01 CRITICAL (bash tự duyệt đọc được tệp env), SemVer + CHANGELOG |
| E8 | Chạy không người trực | fdfcbbbb..4a0b25b6 | 8 | 27 tệp +4.197/−12 | làm khi vắng mặt → loop, routine, chế độ không người trực |

Lệch hướng có bằng chứng (đã xác minh bằng commit, phần "ý định" là suy luận):

- **Phạm vi:** E8 thêm automation mà bảng KEEP/DEFER (plan multi-session, Phần K :582) xếp vào DEFER, làm trước khi cổng WIN đóng và khi A-01 (HIGH) còn mở. Repo không có ghi chép mở lại phạm vi. Ứng viên đã đóng A-01 (PRB-15).
- **Ba bộ phân tích lệnh shell song song:** `safe.go` (198 dòng), `execpolicy` (618), `protect.go` + `unattended.go` (734); 15 commit chạm chúng, phần lớn đuổi theo lối vòng qua danh sách đen. Đợt này thêm 3 lượt vá nữa cho đúng lớp lỗi đó (PRB-25/67/76). DECISIONS.md và CHANGELOG đã thừa nhận cần gộp hoặc dùng sandbox → ghi ở [07](07_DAILY_WORK_GAPS.md) như một khoảng trống kiến trúc, không sửa trong đợt này.
- **Hai cổng "xong":** Stop hook và `loop --verify` (C-4, có tài liệu vai trò).
- **Chuỗi "sửa lại cái vừa sửa":** 8 chuỗi (giao việc 4 bước, chặn đổi model → ghim theo lượt, regex → parser bash…). Bài học: lượt soát độc lập luôn bắt thêm, như chính đợt này (4 lượt).

Lý do fork phải giữ: nhiều session × nhiều dự án trong một terminal; GLM/Qwen/model local là agent chính, Claude/Codex chỉ opt-in; WIN chỉ tính khi đi trọn việc hằng ngày trên bản cài; merge upstream phải rẻ (hiện `agent.go` +583/−156 qua 11 commit, chi phí đang tăng).

## 3. Dữ liệu lưu và rủi ro secret (§54)

| Dữ liệu | Nơi | Nội dung | Giữ bao lâu | Bảo vệ | Rủi ro secret |
|---|---|---|---|---|---|
| Session DB | `<dự án>/.crush/crush.db` (thư mục 0700, `db/connect.go:118`) | toàn bộ tin nhắn, output tool (có thể chứa nội dung tệp agent đã đọc) | không có retention (PRB-49); xoá session sót session con (PRB-11) | quyền thư mục | **cao** nếu agent đọc tệp chứa secret |
| Log | server: `~/.cache/agent-cli/server-<host>/crush.log`; local: `<data>/logs/crush.log` | sự kiện, lỗi; không còn giá trị env (PRB-10) | xoay 10 MB, giữ 30 ngày (`log/log.go:25-29`) | lumberjack | thấp |
| Panic log | `crush-panic-<tên>-<thời điểm>.log` **ở thư mục hiện tại** (`log/log.go:72-75`, `os.Create`) | stack trace | vĩnh viễn | quyền mặc định | thấp, nhưng dễ bị commit nhầm vào repo (PRB-49) |
| Journal loop | `<data>/loops/<tên>-<thời điểm>.jsonl` (0600, `loop/journal.go:45`) | đuôi output verify | không retention | 0600 | trung bình (test có thể in secret) |
| Báo cáo + worktree routine | `<data>/routines/<tên>/runs/<…>/report.json` (0600), worktree giữ lại | báo cáo; bản sao repo | giữ để duyệt (PRB-23) | 0600 | như repo |
| Sổ ghi LLM | đường dẫn do người dùng đặt, opt-in (0600) | metadata, loại lỗi; không nội dung (PRB-81) | người dùng tự xoá | 0600 | thấp |
| Bản nháp prompt | `drafts.json` (0600, `ui/model/drafts.go:280`) | prompt chưa gửi | tới khi gửi | 0600 | trung bình |
| projects.json | data dir | đường dẫn dự án | vĩnh viễn | — | thấp |
| Tệp tạm agentic_fetch | thư mục tạm | trang web đã tải | CHƯA KIỂM có dọn không | — | thấp |
| MCP stdio | tiến trình con theo cấu hình | tuỳ server | — | — | tuỳ cấu hình người dùng |
| Memory riêng | không có | — | — | — | — |

## 4. Hiển thị theo lớp (§55)

Luôn thấy: chat của session, trạng thái lượt, footer phím. Thấy khi cần: danh sách session (Alt+S), provider/model/effort (Alt+P/M/E), job nền (Alt+J), Home. Qua CLI khi gỡ rối: `logs`, `ps`, `jobs`, `session show --json`, `run --json`, sổ ghi LLM. Không có dashboard luôn hiện, không có Agent Map/Task Graph (đúng non-goal §69). Khoảng trống: TUI hiện model của workspace chứ không phải model đã chạy lượt (F-01, PRB-05).
