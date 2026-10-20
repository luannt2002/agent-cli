# 09 — Kiến trúc đích cho việc hằng ngày (§70, §93)

So mô hình đích của workflow (§93) với agent-cli sau ứng viên vòng 6 (`r6-final.patch`). Mỗi tầng ghi: ở đâu trong mã, ai là chủ, khớp hay không, và quyết định. Quyết định theo §63/§94.14: chỉ thêm khi có vấn đề agent-cli đã chứng minh. Bảng chủ sở hữu đầy đủ ở [08](08_CAPABILITY_OWNERSHIP.md).

## Tầng theo mô hình đích

| Tầng (§93) | Ở agent-cli | Chủ | Khớp? | Quyết định |
|---|---|---|---|---|
| USER → AGENT CLI | TUI (`agent-cli`), CLI cho script (`run`, `send`, `ps`, `jobs`, `logs`, `loop`, `routine`) | agent-cli | khớp | KEEP AS-IS |
| WORKSPACE | gốc git của dự án; mỗi dự án một server nền (`--host` theo hash), DB `.crush/crush.db`, job, log | workspace | khớp | KEEP AS-IS |
| SESSION | lịch sử + To-Do trong DB; tiếp bằng danh sách, `-s`, `-C` | session | khớp, trừ model: chọn model là trạng thái của workspace (C-2, PRB-05) | Hoãn G-02 (cần đổi giao thức client/server), ghi giới hạn |
| RUN | một prompt → câu trả lời cuối; hàng đợi một lượt/session; `RunID` cho huỷ đúng lượt | `sessionAgent` | khớp | KEEP; vòng này thêm luật lượt rỗng/bị từ chối/lặp/tràn là lỗi |
| TASK GRAPH / TASK | không có. Gần nhất: tool `todos` (model tự khai) | — | khác mô hình | Không thêm: không có vấn đề đo được đòi nó; 13/13 kịch bản đạt mà không cần |
| ATTEMPT | lượt của `loop` (mỗi lượt một session mới, journal JSONL ghi trước mỗi bước) | engine `loop` | khớp cho việc có verify | KEEP |
| AGENT | model chính; agent con chỉ khi model gọi tool `agent` (chỉ đọc) | coordinator | khớp | KEEP; session con không gọi LLM đặt tiêu đề nữa |
| SKILLS | `SKILL.md` quét từ đĩa; tên + mô tả vào system prompt; model tự `view` | model | khớp; chọn lọc là chính sách chờ duyệt | Đề xuất `disabled_skills` + context file toàn cục ([11](11_SKILL_EFFECTIVENESS.md)) |
| TOOLS | tool trong `internal/agent/tools`, qua lớp quyền | permission service | khớp | KEEP; không người trực: chặn ghi ra ngoài trước YOLO |
| GIT | repo của người dùng; routine dùng worktree riêng `routine/*` | người dùng / routine | khớp | KEEP |
| VERIFICATION | lệnh `--verify` của `loop`/`routine` (mã thoát); Stop hook là cố vấn trong lượt | driver `loop` | khớp | Vòng 6: verify xanh sẵn + cây không đổi = `NO_CHANGE` |
| EVIDENCE | journal loop, báo cáo routine (0600), sổ ghi LLM opt-in | driver / người dùng bật sổ | khớp một phần: báo cáo routine chưa ghi provider/model | Ghi giới hạn |
| CURRENT TRUTH | repo + test + runtime | — | khớp | — |
| HISTORICAL MEMORY → advisory | context file + lịch sử session (M0); không kho memory | người/agent qua git | khớp: memory không đè sự thật (s14 đạt) | KEEP M0 ([14](14_MEMORY_DAILY_VALUE.md)) |
| EXECUTION BACKENDS | provider qua `charm.land/fantasy`; một điểm chặn `buildAgentModels`; allowlist `execpolicy` | người dùng chọn | khớp: lời gọi ẩn ở lại provider chính; không bao giờ đổi provider | Vòng 6: fail closed khi model đang chọn không dùng được |

## Nguyên tắc §94 đối chiếu

| Nguyên tắc | Trạng thái | Bằng chứng |
|---|---|---|
| 1. agent-cli là nơi làm việc, provider thực thi | khớp | README; 13 kịch bản chạy trong agent-cli |
| 2. Workspace sở hữu bối cảnh; provider không định danh workspace | khớp | `workspaces.sh` (nhiều workspace một server) |
| 3. Session sở hữu liên tục; provider thực thi lượt | khớp, trừ model của workspace (PRB-05) | s13-provider-switch đạt |
| 4. Ranh giới provider cứng; fallback chéo không bao giờ lặng lẽ | khớp sau vòng 6 | faults ca `selected-no-key`; qwen-only |
| 5. Cấu hình provider không phải bằng chứng thực thi | khớp: sổ ghi LLM ghi provider/model thật đã gọi; nhãn `run --json` được ghi rõ là không phải bằng chứng | `llm_ledger_test.go`; docs |
| 6. Số agent tối thiểu đủ dùng | khớp | [12](12_MULTI_AGENT_RESULTS.md) |
| 7. Skill dẫn đường, không tranh runtime | khớp (không skill nào chạy mã của runtime) | SE [07](../system-evolution/07_SKILL_RUNTIME_FLOW.md) |
| 8. Routine dùng lại primitive | khớp | [13](13_ROUTINE_RECOVERY_RESULTS.md) |
| 9. Memory nhớ lịch sử, không định nghĩa sự thật | khớp | s14 |
| 10. Verification chứng minh việc | khớp cho `loop`/`routine`; `run` chỉ báo lỗi runtime | README mục "Xong" |
| 11. Bằng chứng phải hiện hành cho revision tuyên bố sẵn sàng | áp dụng: kiểm cuối chạy lại trên `r6-final.patch` | SE [37](../system-evolution/37_FINAL_PROOF.md) |
| 12–15. Ít nghi lễ, độ phức tạp phải tự biện minh, không thêm vì dự án khác có | áp dụng | [16](16_PRUNING_DECISIONS.md); không thêm memory, task graph, dashboard |

## Hai chỗ lệch còn lại (có chủ đích)

1. **Model của session (C-2).** Đích: session sở hữu model của nó. Hiện tại: workspace. Cần mang model theo prompt qua giao thức client/server. Hoãn, ghi giới hạn.
2. **Ba bộ phân tích lệnh shell (G-05).** Đích: một ranh giới thật (sandbox hệ điều hành cho routine). Hiện tại: ba danh sách chặn song song. Hoãn, ghi rủi ro kiến trúc; tài liệu nói rõ "không phải sandbox".
