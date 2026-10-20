# 00 — Bàn giao từ System Evolution sang Daily Workspace (§22)

## Mốc

| | Giá trị |
|---|---|
| Revision xuất phát | `main` @ `4a0b25b6`, cây sạch (SE [00](../system-evolution/00_BASELINE_CHECKPOINT.md)) |
| Mốc cuối của SE | `reports/system-evolution/evidence/wt-final.patch` trên `4a0b25b6`, sha256 `1d963689…`, 72 tệp. Patch chưa commit; cây làm việc `scratchpad/evo/wt-b` |
| Ứng viên cuối của Daily (sau vòng 8) | `scratchpad/evo/r8-final.patch` trên `4a0b25b6`, sha256 `e4081c0b…`, 88 tệp, +5925/−306 (bản lưu bền `reports/system-evolution/evidence/r8-final.patch`) |

## Phát hiện quan trọng của SE (có bằng chứng)

| Phát hiện | Bằng chứng |
|---|---|
| Lời gọi đặt tiêu đề (kèm nguyên văn prompt đầu) và `agentic_fetch` đi sang provider của model nhỏ | faults 8/8 ca có request lọt (binary gốc) |
| Tự tóm tắt lặp vô hạn: khi phần cố định quá lớn (755 + 755 lời gọi / 90 s), và khi model làm lại việc sau mỗi lần tóm tắt (1.338 lời gọi / 120 s) | P-04, `qwen-only.sh`, PRB-88 |
| `run` thoát 0 khi model trả lời rỗng hoặc bị bộ lọc nội dung chặn | faults ca `empty-reply`, PRB-75 |
| Routine: lịch in ra không chạy được (thiếu biến launcher, thoát 2); lượt bị kill bị che sau GREEN cũ | PRB-07, PRB-02 |
| Chế độ không người trực để lọt `git -C`, option viết tắt, biến `GIT_*`, tool ghi tệp ra ngoài workspace | lượt soát 1–3 (SE [38](../system-evolution/38_RED_TEAM.md)) |
| 60% system prompt là skill; 2/41 lượt dùng thật có nạp skill | P-05, SE [31](../system-evolution/31_SKILL_EFFECTIVENESS.md) |

## Thay đổi được nhận (trong ứng viên)

Danh sách đủ ở `CHANGELOG.md` mục `[Unreleased]` (Sửa lỗi) và SE [32](../system-evolution/32_REGRESSION_SUITE.md). Theo nhóm: cô lập provider cho lời gọi ẩn; trần tự tóm tắt và chốt tràn context; lượt rỗng/bị từ chối/lặp là lỗi; routine `RUNNING`/`ABANDONED`/`ERROR` và lịch mang biến launcher; chế độ không người trực bịt thêm lỗ (luật theo chữ cho biến git, chặn tool ghi ra ngoài); sổ ghi LLM opt-in; skill trong thư mục ẩn bị bỏ; hộp thoại quyền không chặn chéo session; `run` local duyệt session con; job nền có chủ.

## Bị loại, hoặc thử rồi quay lại

| Thứ | Vì sao | Nguồn |
|---|---|---|
| Chặn biến git bằng phân tích phép gán | bị vòng qua bằng 7 dạng | [16](16_PRUNING_DECISIONS.md) |
| "Lượt rỗng" xét từng lượt / chỉ finish `stop` | chặn nhánh tự làm tiếp; finish `unknown` lọt | [16](16_PRUNING_DECISIONS.md) |
| Memory Bank, retain, reflect, consolidation, mental model | không vấn đề nào chứng minh được | SE [28](../system-evolution/28_HINDSIGHT_DECISIONS.md) |
| Fallback model chéo provider trong RAM (quyết định SE cuối) | lật ở vòng 6 (F1): vi phạm luật cứng §32 | DECISIONS.md |

## Quyết định theo mảng

| Mảng | Quyết định khi bàn giao | Thay đổi trong Daily |
|---|---|---|
| Provider | một điểm chặn `buildAgentModels`; model nhỏ khác provider thì thay bằng model chính; allowlist `execpolicy` | G-01: fallback không ghi xuống đĩa; vòng 6: không bao giờ đổi provider, provider không dùng được thì dừng có lỗi |
| Skill | bỏ thư mục ẩn; chính sách tắt 21 skill là đề xuất chờ duyệt | không đổi (chờ người dùng) |
| Routine | dùng lại engine `loop` + coordinator, khoá flock, worktree riêng, lịch chỉ in | vòng 6: `NO_CHANGE`; allowlist không vào lịch |
| Memory | M0 | đo s14 (context file sai) |
| Verification | "xong" = mã thoát verify của `loop`/`routine`; `run` rc=0 không phải đã kiểm | vòng 6: verify xanh sẵn + cây không đổi = `NO_CHANGE` |

## Giới hạn đã biết khi bàn giao

PRB-05 (model là của workspace), PRB-16 (hàng đợi prompt chỉ ở RAM), PRB-19/20 (context file chỉ ở cwd), PRB-30 (allowlist so nhãn), PRB-84/85 (vòng đời MCP/job hẹp), "không người trực không phải sandbox". Trạng thái mới nhất của từng mục: `docs/agent-cli-local-setup.md`, mục "Giới hạn đã biết".

## Chưa biết khi bàn giao

U-1: provider `main` (qwen3.8) là local hay trả tiền: người dùng chưa trả lời; bộ eval LLM thật chạy qua nó. Chất lượng tóm tắt với LLM thật: chưa đo. Hiệu quả skill (A/B): chưa đo được trên bộ kịch bản (10/10 đạt dù có nạp skill hay không).

## Test hồi quy liên quan

SE [32](../system-evolution/32_REGRESSION_SUITE.md) (bảng test theo PRB, cột đỏ-trước và đột biến), cộng các cổng bằng binary: `scripts/test-agent-cli-{run-json,loop,routine}.sh`, `scripts/eval/{faults,qwen-only,workspaces}.sh`.

## Dùng lại bằng chứng (§22)

| Bằng chứng SE | Code liên quan có đổi sau đó? | Xử lý |
|---|---|---|
| Baseline LLM thật 10 kịch bản (`baseline-real`) | binary gốc, không đổi | REUSE làm "trước" |
| faults/qwen-only/workspaces trên ứng viên SE | đổi (Daily + vòng 6) | REVALIDATE trên binary cuối |
| Suite so theo tên (`suite-compare-1.txt`, 19:57) | đổi sau đó (red team F2) | REVALIDATE: chạy lại `./...` trên ứng viên cuối |
| Eval LLM thật trên ứng viên (`final-real2`) | đổi (vòng 6) | REVALIDATE trên binary cuối |
| Đo kích thước khối skill (P-05) | không đổi trong Daily | REUSE |
| Thí nghiệm memory (SE 30) | không đổi | REUSE; thêm s14 |
| Probe đọc Hindsight (SE 24–26) | không liên quan Daily | DO NOT REPEAT |
