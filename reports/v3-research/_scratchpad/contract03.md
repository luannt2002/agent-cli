
---

## Hợp đồng đầu ra (spec §63)

**SCOPE** — Bản nháp offline: dựng IssueRecord và cụm họ lỗi (overlay §3, §4, §17, §18) cho 351 issue/PR (173 + 178) đang mở mà
upstream gom trong `tickets/v2-issue-map.md`, ưu tiên 10 họ được giao. Không đọc trang/thảo luận issue (cần `gh`), không đào fork,
không chạy test.

**REVISION AUDITED** — `origin/v2-design` @ `7a99ece2` (nguồn issue, mã v2, test v2); `origin/main` @ `6006a351` (mã v1, commit sửa).

**FILES/COMMITS/ISSUES INSPECTED** — `v2:tickets/{v2-issue-map.md, v2-not-closed.md, round4-evidence.json}`,
`v2:tickets/data/{is,is2,isall,pr2,prall}.json`, `v2:docs/v2/review/ISSUE-OUTCOMES-r4.md`, `v2:docs/v2/impl/DEVIATIONS.md`; tệp test v2
(`e2e/cases/*`, `tests/v2/**`, `e2e/live/cases/*`, quét chuỗi); mã v1 `main:hooks/core/{routing,mcp-ready,tool-naming}.mjs`,
`main:hooks/hooks.json`, `main:src/session/{db,extract,snapshot}.ts`, `main:src/store.ts`, `main:src/server.ts`; mã v2 như 02b. Commit:
`3383299c`, `2731ca25`, `5235f3fd`, `f8d23289`, `752d5236`, `855e330d`, `675fec69`, `16e2bfc8`, `b4edff18`, `415ce57d`, `04569b12`,
`35e264ca`, `e56f17c1`, `f4e8d2ac`, `ff5b0cff`, `245cd8af`, `b1ba5cc9`, `85f64d2c`, `4bc292fc`, `4164d3de`, `a888cac6`, `11474111`,
`79e7bb52`. IssueRecord đầy đủ: #1055, #1037, #967, #1267, #1222, #1106, #1140, #1156, #1022, #1100, #1014, #1075 (+ #233/#724/#946/#794
ở 01).

**VERIFIED FACTS** — Số đếm ở mục 0–1 (lệnh: `parse_map.py`, `enrich.py`, `families.py` trong scratchpad, trên dữ liệu `git show`);
172/173 issue tạo sau khi `main` ngừng; giao với commit `main` = 0; 159/170 tên ca GP có trong tệp test; mâu thuẫn "đọc body" trong
nguồn upstream; lý do đóng dựa D10 đã lỗi thời; cơ chế gốc E1 của #1267, #1107, #1222, #1140, #1156, #1024, #1055/#1053, #1106; các
chuỗi bug lặp ở mục 4 có commit `main` thật.

**INFERENCES** — Phân loại họ (làm tay theo tiêu đề); ý nghĩa các lớp A/B/C/D/Q/M (suy từ bảng chéo); triệu chứng/kịch bản lấy từ tiêu
đề (E0); phiên bản bị ảnh hưởng; v2 "FIXED" nghĩa là mã trên nhánh chưa phát hành, chưa chạy.

**UNKNOWNs** — Nội dung thảo luận, người báo, nhãn, liên kết PR↔issue chính thức, ngày đóng, phản hồi của maintainer cho từng mục;
issue nào tái hiện được; số người dùng bị ảnh hưởng; hành vi runtime của mọi bản sửa v2.

**FAILURE MODES** — HOST_ADAPTER_MISMATCH, ROUTING_FAILURE, FALSE_SUCCESS, PROMPT_INTERFERENCE, ZOMBIE_OPERATION, CONTEXT_OVERHEAD,
CAPABILITY_MISMATCH, DELETION_PROPAGATION_FAILURE/mất dữ liệu âm thầm, SECURITY_BOUNDARY_CONFUSION, STARTUP_RACE; ngoài danh sách:
INSTALL/UPGRADE, PROJECT_IDENTITY/SESSION_CONTAMINATION, OBSERVABILITY (số liệu sai), STALE_CONTEXT.

**AGENT CLI RELEVANCE** — Không kết luận phơi nhiễm. Câu hỏi chuyển cho agent khác: agent-cli có cơ chế nào được "sửa" bằng cách xoá mà
không thêm ca test cho trạng thái cơ chế đó từng bảo vệ không; chỗ quyết quyền/tool của subagent và chỗ viết chỉ dẫn có cùng nguồn
không; luật chặn lệnh có đi qua parser hay chuỗi; trạng thái "đã nhắc/đã sẵn sàng/đang sống" khoá theo danh tính nào; mỗi đường dọn dữ
liệu định nghĩa "cũ/chết" thế nào và có test "sống nhưng rảnh" không.

**PROPOSED TESTS** — Gộp với 01 (T1–T8) và 02b (T9–T13). Thêm: T14 removed-mechanism guard: mỗi cơ chế bị xoá ⇒ một ca tái hiện trạng
thái nó từng bảo vệ (vd dịch vụ không lên). T15 deletion-liveness matrix: mọi đường dọn × dữ liệu {mới, cũ-đang-dùng, cũ-rảnh,
cũ-chết} ⇒ chỉ ô cũ-chết bị xoá. T16 once-means-once: mọi lời nhắc gọi hai lần cùng phiên, hai tiến trình khác nhau ⇒ một lần.
T17 cross-adapter port: mỗi bản sửa luật ở một adapter ⇒ cùng bảng đầu vào chạy trên mọi adapter.

**NO-CHANGE FINDINGS** — Hướng sửa v2 cho ROUTING (parser), ZOMBIE (EOF + deadline), CONTEXT_OVERHEAD (hằng ngân sách + lint) và
PROMPT_INTERFERENCE (bỏ tiêm) đơn giản hơn chuỗi vá v1 và có test; không thấy lý do đề xuất khác cho các họ đó. Dữ liệu offline đủ để
gom cụm; không đủ để khẳng định nguyên nhân gốc của các issue chỉ có tiêu đề.
