# 01 — Taxonomy lỗi chung (spec §12)

Ánh xạ 16 họ tối thiểu của spec §12 sang 22 họ ở [ISSUE_TO_INVARIANT_MAP.md](ISSUE_TO_INVARIANT_MAP.md) (gọi tắt **MAP**; F01–F22).
Bảng chỉ tóm tắt; bằng chứng đầy đủ, mức bằng chứng và tệp:dòng nằm trong mục F## tương ứng của MAP.

Nguồn phơi nhiễm agent-cli: `WORKFLOW_PROGRESS.md` mục 20, cụ thể:
- 9 ca EXPOSED: T2, T3, T5, T6, T11, T12, T13, T15 (thu hẹp), T17.
- T1 thu hẹp về symlink-vào-thư-mục-con.
- T21 chỉ là phép đo.
- Đợt 2 ĐANG ĐO: T4, T9, T19, T20, T23, C5/S10, C9, T21b/c.

Viết tắt dự án: CM = Context Mode, MEM = claude-mem, HS = Hindsight.

## Bảng

| # | Họ spec §12 | → MAP | Bằng chứng CM | Bằng chứng MEM | Bằng chứng HS | Agent-cli V2 tương đương | Bất biến V3 (rút gọn) | Test hồi quy V3 |
|---|---|---|---|---|---|---|---|---|
| 1 | SCOPE_FAILURE | F01, F02, F04 | F6 attribution lấy "session mới nhất"; F7 ContentStore không theo worktree (E1) | #3608 OPEN, #2882, #3437, #1911→#3764 (E0+E1) | tag scope #5030→#5371→#5417; `all_strict` #2808→#5728 (E1) | **ĐO ĐƯỢC** T2, T3, T1 thu hẹp; worktree liên kết KHÔNG; truy hồi: KHÔNG áp dụng | Mọi resolver cho cùng khoá và data dir; thiếu scope ⇒ lỗi, không toàn cục | T1–T4, T26, T30 |
| 2 | AUTHORITY_FAILURE | F20, F18, F16, F07 | context file của repo ra lệnh cho tool không tồn tại (quan sát runtime, WP 5) | `sdk_sessions.project` first-write-wins, trong khi khoá observation theo session-init gần nhất (L3, E1) | "nothing found" đè lên trang tốt #5728 (E0+E1) | **ĐO ĐƯỢC** T17 (model dẫn xuất từ lịch sử bị ghi thành cấu hình workspace); T12 (tóm tắt LLM tới model với vai user) | Tầng thấp (spec §13) chỉ được gợi ý, không được ghi đè tầng cao; dẫn xuất không bao giờ thành cấu hình thẩm quyền | T17, T12 |
| 3 | INJECTION_FAILURE | F05, F18, F07 | #233, #724, #794, #1037; v2 V2-F2 (E1/E2) | #4141; SessionStart không khung untrusted (E1) | — | **ĐO ĐƯỢC** T5, T6 (chỉ dẫn cho tool vắng), T12 | Văn bản chỉ nêu tool có thật trong lượt; văn bản dẫn xuất có nhãn nguồn ở mọi kênh | T5, T6, T7, T12, T25 |
| 4 | DELIVERY_FAILURE | F08, F13 | #1022 (snapshot ~196 KB, E0+E1) | #3802 (vượt 10.000 ký tự → stub, hook vẫn báo thành công); #2547 | #4560 (gửi lại toàn transcript, E0) | **ĐO ĐƯỢC** T11 (rc 0 nhưng tin không tới DB); SUY LUẬN: MCP/giao việc không trần (T9 ĐANG ĐO) | Trần kênh là một phần hợp đồng, đo trên bản render; ack chỉ sau khi đã bền | T9, T11 |
| 5 | LIFECYCLE_FAILURE | F11, F17, F10 | chuỗi vòng đời server (revert #602); dọn xoá dữ liệu sống (E1) | reclaim worker 3 bước, bước sau sửa bước trước (E1) | worker id đổi → kẹt `processing` (#2165→#3194); lease bị bác | KHÔNG "kẹt mãi" (routine ABANDONED, Interrupted, trần mồ côi — E1); SUY LUẬN: xoá session không lan sang session con; ĐANG ĐO: T4, T23 | Mọi việc tới trạng thái cuối trong thời gian hữu hạn; dọn chỉ xoá dữ liệu cũ-chết | T4, T23, T28, T10 |
| 6 | RECOVERY_FAILURE | F11, F13 | `4164d3de` mất hơn 1000 sự kiện sau mất điện (E1) | buffer RAM mất khi worker restart (`d55d20f7e`, E1 + E0) | #5251 (lượt ghi cuối hỏng thì mất), #2985 (E0+E1) | **ĐO ĐƯỢC** T13 (prompt xếp hàng mất khi SIGTERM) | Sau crash/restart, mọi đơn vị đã ack có bản ghi chờ hoặc huỷ | T13, T11, T28 |
| 7 | COMPOSITION_FAILURE | F05, F08, F07 | 3 chỗ quyết capability với 3 giả định (v1); v2 quyết một chỗ nhưng chỗ viết không đọc (E1) | `ContextBudget` một renderer (cách giải); một renderer nhiều nguồn (fork escooterclinic) | — | **ĐO ĐƯỢC** T5, T6; SUY LUẬN: không ai lưu context cuối cùng gửi model | Một chỗ soạn tiêu thụ quyết định capability; mỗi khối có chủ và ngân sách | T5, T6, T8 |
| 8 | PORTABILITY_FAILURE | F09 | 39 issue HOST_ADAPTER; bản sửa chỉ ở 1/9 hook (E1) | #3678 OpenCode, #3611 plan-23 OPEN | #5221 sửa ở hermes, #5735 còn ở coding-agents (E1) | SUY LUẬN: local và server khác ngữ nghĩa (T17 mới đo server); 3 bản discovery skill; ĐANG ĐO: C9 | Một luật một cài đặt; cùng bảng đầu vào chạy ở mọi chế độ | T17 (thêm nhánh local), T24, C9 |
| 9 | SECURITY_BOUNDARY_FAILURE | F19, F03, F06 | containment ≠ isolation; denylist env; #1075 (E1) | bearer auth bị gỡ; observer tự bắt chính nó (#3425) | không auth, bind 0.0.0.0 (audit E3🔎); hook của host lọt vào tiến trình con #1751→#5542 | PHÉP ĐO T21: `.env` của cwd vào env tiến trình (`main.go:22`); SUY LUẬN: CLI con kế thừa env (`provider/exec.go:35`); ĐANG ĐO: C9 | Mọi phép "trong workspace" đồng nhất; tiến trình con mang dấu nội bộ; bề mặt lưu bền được kê khai | C9, T21, T27 |
| 10 | STALE_STATE_FAILURE | F15 | v2 không làm mới nguồn cũ (mới nghi) | không có danh tính branch; cache ≤24 h (E1) | #5713 tiêm một lần không làm mới (E1) | SUY LUẬN: system prompt là ảnh chụp lúc InitAgent; session không có branch/revision | Lượt sau checkout không nhận ảnh chụp cũ như sự thật hiện tại | T8 |
| 11 | DERIVED_STATE_FAILURE | F16 | nhãn `execute:<lang>` ghi đè chỉ mục (E1) | #4730 Chroma giữ bản cũ (E0+E1) | #5054, #5728, #5731, `consolidated_at` gánh hai nghĩa (E1) | **Đã đúng:** tóm tắt rỗng hoặc bị cắt không thành điểm cắt (E1) | Dẫn xuất chỉ thay bản cũ khi có ≥1 nguồn trong phạm vi và trạng thái ok | test V2 có sẵn (SE 28); T12 |
| 12 | PROVENANCE_FAILURE | F16, F18 | — | observation→tool_uses không 1:1; không gắn revision (E1) | `based_on` rộng hơn nguồn thật dùng (`45616ffe3`); import giữ id evidence cũ (`4b5f045c7`) | **ĐO ĐƯỢC** T12 (mất nhãn nguồn); SUY LUẬN: tóm tắt không có revision | Bản dẫn xuất mang nguồn, model, revision; nhãn không mất khi vào prompt | T12 |
| 13 | PARTIAL_SUCCESS_FAILURE | F13, F14 | v2 index nhiều lô không nguyên tử (mới nghi) | #3653 13 obs/44k sự kiện (E0) | #5589, #3876, `06a964dd2`, #5582 (E0+E1) | **ĐO ĐƯỢC** T15 (lỗi ghi sổ bị nuốt); SUY LUẬN: tóm tắt lưu 2 bước; T19 ĐANG ĐO | Không ghi đè trạng thái bền khi một phần đã biết là hỏng; rỗng/lỗi không bao giờ là thành công | T15, T19, T14 |
| 14 | BACKPRESSURE_FAILURE | F12 | #999 busy-loop 23 instance, #985 (E0) | #4087 111 MB/ngày, #3928 3,8 GB/3 ngày (E0); log không retention (E1) | #5434 (chống retry storm sinh `failed` giả) | **Đã đúng:** log xoay 10 MB, giữ 30 ngày (E1); SUY LUẬN: `DropCount()` không ai đọc | Lỗi vĩnh viễn ⇒ log bị chặn trên; retry có trần và phân loại | T29 |
| 15 | PROVIDER_POLICY_FAILURE | F20 | không gọi LLM; egress opt-in | #4727 fallback Anthropic mặc định (rủi ro, E1); #2201 worker kế thừa cấu hình (E0+E1) | `0d8e58b3e` thêm đường provider decision model (E1) | **ĐO ĐƯỢC** T17; **KHÔNG** cho đường LLM nội bộ (`buildAgentModels`, fail closed); SUY LUẬN: giao việc và hook ngoài sổ ghi; ĐANG ĐO: C5/S10 | Mọi lời gọi LLM qua một điểm dựng có allowlist và sổ ghi; không fallback chéo provider | T16, T17, T18 |
| 16 | OBSERVABILITY_FAILURE | F21 | số liệu sai #894/#1151; lỗi ghi detached không đếm | NOT_READY ≡ FAILED với model (E1) | health = kết nối DB (#2413, #1671); token ước lượng không có cờ | **ĐO ĐƯỢC** T15; SUY LUẬN: R21 `crush_logs` đọc nhầm tệp, `DropCount` | Lỗi fail-open có bộ đếm đọc được; "đang chạy" kèm tuổi | T15, T24, T28 |

## Nhận xét

- Họ spec §12 không có trong overlay §4: **AUTHORITY**, **LIFECYCLE**, **RECOVERY**, **COMPOSITION**, **PORTABILITY**. Cả năm đều ánh xạ
  được vào họ đã có của MAP, không cần họ mới.
  - AUTHORITY là họ mà agent-cli có bằng chứng rõ nhất ngoài danh sách overlay. Hai test cho cùng một hình dạng "dẫn xuất được nâng thành
    thẩm quyền": T17 (lịch sử → cấu hình) và T12 (văn bản LLM → lời người dùng).
- Ba họ spec có phơi nhiễm ĐO ĐƯỢC mạnh nhất ở agent-cli, xét theo số ca test đỏ:
  - SCOPE: T1, T2, T3;
  - INJECTION / COMPOSITION: T5, T6, T12;
  - DELIVERY / RECOVERY: T11, T13.
- Họ agent-cli đã làm đúng (E1, chưa đo lại trong V3): DERIVED_STATE (tóm tắt rỗng không thay bản cũ), BACKPRESSURE phần retention log,
  LIFECYCLE phần "mọi việc có trạng thái cuối", PROVIDER_POLICY phần đường LLM nội bộ. Bằng chứng ở MAP F11, F12, F16, F20.

---

## Hợp đồng đầu ra (spec §63)

- **SCOPE:** ánh xạ 16 họ của spec §12 sang 22 họ của MAP. Mỗi họ có bằng chứng CM/MEM/HS, tương đương ở agent-cli V2, bất biến và test.
- **REVISION AUDITED:** như MAP: Context Mode `6006a351` / `7a99ece2`; claude-mem `3fae1eb45`; Hindsight `5331da2f4`; agent-cli `7cba42ff`.
- **FILES/COMMITS/ISSUES INSPECTED:** MAP (tệp anh em) cùng các báo cáo nguồn ghi trong MAP §0; spec §12–14.
- **VERIFIED FACTS:** chỉ những gì MAP đã ghi kèm tệp:dòng hoặc phán quyết test. Tệp này không thêm khẳng định mới.
- **INFERENCES:** cách ánh xạ họ spec sang họ MAP; câu bất biến rút gọn; nhận xét về AUTHORITY.
- **UNKNOWNs:** kết quả đợt 2 (T4, T9, T19, T20, T23, C5/S10, C9, T21b/c).
- **FAILURE MODES:** 16 họ ở bảng trên.
- **AGENT CLI RELEVANCE:**
  - 11/16 họ có ít nhất một ca ĐO ĐƯỢC: SCOPE, AUTHORITY, INJECTION, DELIVERY, RECOVERY, COMPOSITION, PROVENANCE, PARTIAL_SUCCESS,
    PROVIDER, OBSERVABILITY, và SECURITY ở dạng phép đo.
  - Còn lại là SUY LUẬN hoặc ĐANG ĐO.
- **PROPOSED TESTS:** không có test mới ngoài T26–T30 của MAP.
- **NO-CHANGE FINDINGS:** không họ nào của spec §12 cần thêm họ mới vào MAP.
