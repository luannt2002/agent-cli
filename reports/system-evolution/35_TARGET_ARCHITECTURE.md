# 35 — Kiến trúc đích (Phase 25)

Đích là **sửa chủ sở hữu**, không thêm tầng. Mỗi dòng: hiện tại → đích → vì sao → trạng thái. Nguồn: [16](16_CAPABILITY_OWNERSHIP.md), [17](17_AUTHORITY_CONFLICTS.md), [34](34_COHERENCE_GATE.md).

## Mô hình công việc (giữ, không thêm tầng)

```
người dùng → agent-cli (TUI / run / loop / routine)
  → Workspace  (một App mỗi gốc dự án; job nền, MCP, cấu hình của dự án)
    → Session  (SQLite: tin nhắn, todos, model đã chạy)
      → Run    (RunID: một lượt; hàng đợi prompt; huỷ theo RunID)
        → bước model + tool call (quyền, lớp chặn, hook)
  → Verification (lệnh verify tất định của loop/routine)
  → Evidence    (journal loop, báo cáo routine có base_sha + tệp đổi, sổ ghi LLM)
Provider = backend thực thi, chọn qua một điểm chặn: buildAgentModels + allowlist execpolicy.
Memory = context file (M0). Không có kho memory riêng.
```

Không thêm "task graph / attempt" như sơ đồ north star (§93): `todos` (model khai) và lượt thử của `loop` (journal) đã đóng vai đó. Thêm engine điều phối thứ hai là non-goal (§69).

## Chủ sở hữu: hiện tại → đích

| Năng lực | Hiện tại | Đích | Vì sao | Trạng thái |
|---|---|---|---|---|
| Provider của lời gọi ẩn (tiêu đề, agentic_fetch, tóm tắt, agent con) | small có thể ở provider khác | luôn provider của model chính | cô lập provider; faults 8/8 → 0/8 | **xong (wt)** |
| Chọn provider (C-1) | máy ghi lựa chọn vào global data khi fallback lúc khởi động | **chỉ người dùng** chọn. Thiếu khoá → báo lỗi to, hoặc fallback trong **cùng** provider; không bao giờ ghi bền | PRB-04: một lần mất khoá thành đổi provider vĩnh viễn | mở — slice S-4 |
| Model của session (C-2) | resume / `run --model` ghi vào scope workspace → mọi session đổi theo | session dùng model của tin assistant cuối, áp **trong RAM** cho lượt của nó; không ghi cấu hình | PRB-05 | mở — slice S-5 (cần đọc giao thức server) |
| Skill được liệt kê (C-3) | mọi thứ quét được, kể cả `.trash`, `synced` | agent-cli quyết định qua `disabled_skills`; luật "luôn bật" là context file toàn cục (chính sách), không phải skill | 60% system prompt; 2/41 lượt nạp skill | thư mục ẩn **xong (wt)**; phần còn lại chờ người dùng duyệt đề xuất |
| "Xong" (C-4) | Stop hook và verify của loop là hai định nghĩa | `loop`/`routine` là chủ của "xong có bằng chứng". `run` rc=0 nghĩa là "lượt chạy hết không lỗi runtime", **không** phải "đã kiểm chứng". Stop hook là cố vấn | §80: MODEL SAYS DONE ≠ AGENT CLI COMPLETED | tài liệu hoá ở Daily; PRB-75 (lượt rỗng = lỗi) mở |
| Context dự án | chỉ ở cwd, tên viết hoa | thêm gốc git; nhận cả tên thường | PRB-19/20 | mở — slice S-6 |
| Vòng đời job nền | teardown một workspace giết job mọi workspace | chủ là workspace (gốc), App cuối dọn hết | PRB-15 | **xong (wt)** |
| Job tool | đọc/dừng job bất kỳ | chỉ job của session đang gọi | PRB-18 | **xong (wt)** |
| Long-run | báo cáo cuối hoặc không gì | RUNNING → kết quả; ABANDONED suy từ khoá flock | PRB-02 | **xong (wt)** |
| Quan sát chi phí | token bước cuối | sổ ghi LLM opt-in (metadata, không nội dung) | PRB-32 | **xong (wt)** |
| Memory | context file | giữ (M0); M1 chỉ khi Daily I/J chứng minh cần | 28–30 | giữ |

## Không đổi (đúng sẵn — KEEP AS-IS)

- Một server mỗi dự án qua socket theo hash, trong thư mục 0700 (wrapper).
- Hàng đợi prompt một lượt/session, huỷ theo RunID.
- BlockFunc ở lớp thực thi, trước YOLO.
- Allowlist model (`execpolicy`) đọc env gốc, dự án không đổi được.
- `loop` fail-closed: UNMEASURED không bao giờ là xanh.
- Routine chạy trong worktree riêng, khoá chống chạy chồng, chỉ in lịch (không tự cài).
- Settle tool call mồ côi khi mở session.
