# FINAL SYSTEM AUDIT — System Evolution của agent-cli

> Báo cáo dưới giữ nguyên kết quả lịch sử ngày 16/10. Bản tiếp quản, README
> đã soát lại và receipt ngày 19/10: [báo cáo 39](39_TAKEOVER_2026-10-19.md).

Revision đánh giá: ứng viên cuối `scratchpad/evo/r8-final.patch` (sha256 `e4081c0b…`, 88 tệp, +5925/−306; bản lưu bền
`evidence/r8-final.patch`) trên `main` @ `4a0b25b6`. Chưa commit (CLAUDE.md P1: chờ người dùng cho phép). Bằng chứng chạy trên đúng revision này:
[37](37_FINAL_PROOF.md). Thang bằng chứng: E1 đọc mã · E2 có test · E3 binary thật + LLM giả · E4 chạy thật local ·
E5 LLM thật · E6 benchmark · E7 cả hệ thống.

## Kết luận một dòng

agent-cli tốt hơn đo được ở **tính đúng và an toàn**: lời gọi lọt sang provider khác từ 799 xuống 0 trong kịch bản chỉ
Qwen, ba lớp "thành công giả" thành lỗi, hai vòng lặp vô hạn có trần, provider không còn bị đổi lặng lẽ. **Hiệu quả**
(token, thời gian) thì chưa kết luận được: mỗi kịch bản LLM thật chỉ chạy một lần. Bộ kịch bản cũng đã đạt 10/10 ngay từ
đầu, nên không đo được "giải việc giỏi hơn".

## Những gì phải xác lập (Phase 24)

**SAI Ở ĐÂU (bản gốc).** Đã chứng minh (E3–E5): lời gọi ẩn (tiêu đề, `agentic_fetch`) đi sang provider của model nhỏ;
tự tóm tắt lặp vô hạn ở hai dạng; `run` thoát 0 khi model trả lời rỗng, khi dừng vì lặp tool call, khi bản tóm tắt bị
cắt; lịch routine in ra không chạy được; routine bị kill bị che sau lượt GREEN cũ; chế độ không người trực để lọt
`git -C`, option viết tắt, biến `GIT_*`, tool ghi tệp ra ngoài; hộp thoại quyền của một session chặn session khác; `run`
local treo vì quyền của session con; job nền không có chủ; đóng một workspace giết job/MCP của workspace khác; model
đang chọn thiếu key thì lặng lẽ chạy bằng provider khác và ghi lựa chọn đó xuống đĩa; provider dựng sẵn bị tắt vẫn bị
gọi; skill trong `.trash` vẫn vào prompt; panic log rơi vào thư mục hiện tại; không có cách chứng minh lời gọi nào đi
provider nào. Danh sách đủ: [05](05_CURRENT_PROBLEMS.md) (88 ID).

**CÁI GÌ VỐN ĐÃ TỐT.** Một điểm chặn provider duy nhất (`buildAgentModels`); session/workspace tách đúng (11/11 kiểm tra
E3); tool call bỏ dở được ghi "Interrupted" khi mở lại; `RunComplete` phát đúng một lần; hàng đợi một lượt/session; agent
con chỉ chạy khi model gọi; context file + lịch sử session đủ cho việc tiếp tục (s10 đạt ngay ở bản gốc).

**GIỮ.** Toàn bộ kiến trúc lõi trên; M0 cho memory; engine `loop` làm nền cho `routine`.

**BỎ.** Lời gọi tiêu đề cho session con; thử lại tiêu đề bằng model lớn sau `length`; quét thư mục ẩn; ghi fallback xuống
đĩa; fallback chéo provider (cả trong RAM); `mcp.Close` của mọi App; ghi giá trị env vào log; chép allowlist vào lịch;
cơ chế chặn biến git bằng phân tích phép gán (quay lại, thay bằng luật theo chữ). Chi tiết: [16 (Daily)](../daily-work/16_PRUNING_DECISIONS.md).

**THÍCH NGHI (adapt).** 7 bất biến rút từ lỗi của Hindsight ([26](26_HINDSIGHT_FAILURE_INVARIANTS.md), [28](28_HINDSIGHT_DECISIONS.md)).

**LÀM LẠI (reimplement).** Provenance → sổ ghi LLM chỉ metadata; "mọi việc có trạng thái cuối" → `RUNNING`/`ABANDONED`/
`ERROR` của routine; "rỗng ≠ thành công" → lỗi lượt rỗng/bị từ chối, `NO_CHANGE`.

**BÁC BỎ.** Memory Bank, retain tự động, reflect, consolidation, mental model, hook memory, benchmark memory làm bằng
chứng ([28](28_HINDSIGHT_DECISIONS.md)).

**HINDSIGHT ĐÓNG GÓP GÌ.** Kho lỗi, không phải kiến trúc: các bất biến vòng đời và "rỗng ≠ thành công" dẫn thẳng tới các
sửa của PRB-02, PRB-12, PRB-24, PRB-27, PRB-75, PRB-82; bài học "đầu vào sai ở biên tin cậy không được âm thầm nới" khớp
đúng PRB-04 (đóng ở vòng 6–7). Không thành phần nào của Hindsight được nhận nguyên khối.

**PONYTAIL ĐÓNG GÓP GÌ.** Trong sản phẩm: không đo được. 0/41 lượt dùng thật nạp nó; trong eval, `ponytail-review` được nạp
đúng ở việc review (s7, s16), nhưng kịch bản đạt cả khi không nạp. Trong quy trình của đợt này: khẩu vị "thay đổi nhỏ
nhất đủ đúng" được áp khi viết mã. Đó là lời khai về quy trình, không có số đo.

**SUPERPOWERS ĐÓNG GÓP GÌ.** Trong sản phẩm: `systematic-debugging` được nạp đúng ở việc gỡ lỗi, TDD ở việc sửa/tính năng;
không đo được tác dụng (kịch bản đạt cả khi không nạp). Các skill dựa trên agent con ghi mã (SDD, dispatching) không chạy
được ở agent-cli vì agent con chỉ đọc. Đề xuất tắt chờ người dùng duyệt ([09](09_SUPERPOWERS_AUDIT.md)).

**CÁI GÌ CÒN LÀ NGUỒN SỰ THẬT.** Repo + test + hành vi binary; cấu hình (crushrc/crush.json/env) cho lựa chọn; DB của dự
án cho lịch sử; sổ ghi LLM và báo cáo routine cho bằng chứng.

**MEMORY LÀM ĐƯỢC / KHÔNG.** Làm được: giữ mạch qua tiến trình (s10), qua đổi provider (s13), nén lịch sử (tự tóm tắt).
Không: không tự học, không tự sửa, không có thẩm quyền trên mã/test; context file là lời dặn model, không phải luật máy ép.

**TEST NÀO BẢO VỆ KIẾN TRÚC.** [32](32_REGRESSION_SUITE.md): test theo từng lỗi (đỏ trước, đột biến cho mã an ninh), cộng
cổng bằng binary `scripts/eval/{faults,qwen-only,workspaces}.sh` và E2E `scripts/test-agent-cli-{run-json,loop,routine}.sh`.

**CÒN CHƯA BIẾT.** Câu 25 dưới.

**SỐ TRƯỚC / SAU.** Mục cuối.

**HỆ THỐNG CÓ THẬT SỰ TỐT HƠN.** Câu 30 dưới.

## 30 câu nghiệm thu

1. **Bản gốc thật sự có vấn đề gì?** Như mục SAI Ở ĐÂU. 88 ID ở [05](05_CURRENT_PROBLEMS.md); 35 đã sửa trong ứng viên, một
   phần thêm vài ID, số còn lại mở có lý do.
2. **Vấn đề nào đã chứng minh, vấn đề nào chỉ giả định?** Chứng minh bằng chạy (E3–E5): mọi mục trong SAI Ở ĐÂU. Chỉ đọc mã
   (E1), chưa tái lập: PRB-30 (allowlist chỉ so nhãn), L8 (làm mới danh sách model ChatGPT/Grok), ba bộ phân tích lệnh shell
   song song (G-05), hàng đợi prompt chỉ ở RAM (PRB-16), xoá session sót session con (PRB-11). Cột BC ở [05](05_CURRENT_PROBLEMS.md).
3. **Ponytail đóng góp gì?** Xem trên: không đo được trong sản phẩm.
4. **Superpowers đóng góp gì?** Xem trên: chọn đúng loại việc, tác dụng không đo được; phần agent con ghi mã không chạy được.
5. **Skill nào thật sự được nạp và có ích?** Được nạp trong eval LLM thật: `systematic-debugging`, `test-driven-development`,
   `superpower`, `ponytail-review`, `verification-before-completion`, `ponytail`, `using-superpowers`
   ([05 Daily](../daily-work/05_SKILL_TRACE.md)). Có ích: không phân biệt được, vì kịch bản đạt cả khi không nạp.
6. **Hindsight dạy gì?** Lỗi vòng đời và "rỗng mà báo xong" là lỗi thật của loại hệ thống này, và agent-cli có cùng bề mặt
   ([26](26_HINDSIGHT_FAILURE_INVARIANTS.md)).
7. **Khái niệm nào được nhận nguyên?** Không có.
8. **Khái niệm nào được thích nghi?** 7 bất biến: provenance cho bằng chứng, phạm vi đi theo thao tác dẫn xuất, không lưu
   secret, ack sau khi ghi bền (chưa sửa), rỗng ≠ thành công, mọi việc có trạng thái cuối, sửa phải tới mọi bản sao.
9. **Khái niệm nào được làm lại?** Sổ ghi LLM, trạng thái routine, lỗi lượt rỗng, `NO_CHANGE`.
10. **Khái niệm nào bị bác?** Memory Bank, retain, reflect, consolidation, mental model, hook memory.
11. **Lỗi nào của Hindsight thành test hồi quy của agent-cli?** H-01 → `TestLastReport_AbandonedRunsAreShownNotHidden` + E2E
    routine SIGKILL; H-08/H-09 → `TestEmptyTurnIsAnError`, `TestLoopDetectionStopIsAnError`, `TestTruncatedSummaryKeepsHistory`,
    `TestEngine_NoChangeCarriesBaselineAndUnchangedFlags`; H-19 → `TestWaitingRequestIsCancellable`; H-36 →
    `TestFirstWinsMismatch_NeverLogsEnvValues`, `TestLedgerRecordsTheErrorKindNotItsText`; H-40/H-46 →
    `TestBuildAgentModelsFailsClosedWhenSelectedProviderHasNoKey`, `TestBuildAgentModelsRefusesADisabledProvider`; H-43/H-44 →
    `TestFilterDisabledToolsAppliesToInnerTools`, `TestDisabledFetchAlsoGatesAgenticFetch`.
12. **Cái gì còn là sự thật runtime hiện tại?** Mã + test + binary; không có lớp nào đè lên chúng.
13. **Memory có thẩm quyền gì?** Chỉ tư vấn: context file do người viết, duyệt qua git; lịch sử là bản ghi; bản tóm tắt là
    bản nén. Không cái nào đè mã hay kết quả test.
14. **Phát hiện memory cũ bằng cách nào?** Không có máy tự phát hiện. Dựa vào thứ tự tin (mã > bằng chứng > lịch sử) và việc
    kiểm chứng (test chạy trên mã hiện tại). s14 (context file cũ, sai và có chỉ thị độc) đạt với LLM thật, một lần; không
    phải bảo đảm.
15. **Sửa tri thức lịch sử thế nào?** Sửa context file hay DECISIONS.md qua git (mục "Lật lại khi"); không có memory dẫn xuất
    nào phải sửa.
16. **Khôi phục sau sập thế nào?** Tool call bỏ dở → "Interrupted"; lời trả lời dở được đóng khi mở session; routine bị kill →
    `ABANDONED`, khoá flock nhả do kernel; server SIGTERM dừng từng workspace. Mất: prompt đang xếp hàng khi server bị SIGKILL (PRB-16).
17. **Người thay thế có thu hồi được việc bỏ dở không?** Có, với routine: khoá là flock nên tiến trình nào sau đó cũng lấy
    được; lượt sau chạy worktree mới; worktree/nhánh của lượt bỏ dở giữ lại để xem (E2E routine mục 8).
18. **Phiên mới có giải thích được skill/memory nào đã ảnh hưởng không?** Một phần: skill được nạp hiện thành lời gọi `view`
    trong transcript; context file nằm trong system prompt; sổ ghi LLM có `system_sha`. Không có bản ghi "ảnh hưởng".
19. **Người soát có đủ độc lập không?** Trong quy trình: có. Người soát không có Edit, không nhận kết luận của người viết, và
    lượt nào cũng bắt thêm lỗi thật cho tới lượt cuối DAT ([38](38_RED_TEAM.md): 8 lượt soát mã, kể cả một HIGH do chính bản sửa trước gây ra). Trong sản
    phẩm: s16 (người soát nhận kết luận sai của tác giả) vẫn tìm ra lỗi, một lần chạy.
20. **Đổi provider có sống được không?** Đổi có chủ ý giữ mạch (s13); provider không dùng được thì dừng có lỗi, không đổi
    lặng lẽ (faults `selected-no-key`); app vẫn khởi động để login hay chọn model khác (vòng 7).
21. **Memory hỏng có sống được không?** NOT_APPLICABLE: không có dịch vụ memory. Context file thiếu thì bị bỏ qua (E1).
22. **Chạy không người trực lâu có sống được không?** Có rào: khoá, worktree riêng, chặn lệnh, `--run-timeout`, trần tóm tắt
    10, phát hiện lặp, `NO_CHANGE`. Chưa chạy routine nhiều giờ với LLM thật.
23. **Thêm bao nhiêu độ phức tạp?** Mã Go sản phẩm: 30 tệp, +1.357/−192 dòng (`git diff HEAD --numstat` trên bản vòng 7;
    riêng vòng 7: 4 tệp, +30/−9). Test: 36 tệp, +3.107/−16. Script: 8 tệp, +708 (đo ở vòng 6). Tính năng mới: sổ ghi LLM,
    `loop`/`routine` có trạng thái và `NO_CHANGE`, luật chặn không người trực, `scripts/eval`.
24. **Bớt bao nhiêu?** Như mục BỎ. Không xoá mã chết có sẵn (luật sửa phẫu thuật; danh sách ở PRB-58).
25. **Còn gì chưa biết?** U-1 (provider `main` là local hay trả tiền); chất lượng tóm tắt với LLM thật; hiệu quả skill
    (A/B); routine nhiều giờ; macOS/Windows; kiểm bằng mắt các hành vi TUI; hai client cùng server cùng lúc; hiệu quả token/
    thời gian (cần nhiều lần chạy).
26. **Tỉ lệ việc code thành công có tăng không?** Không đo được mức tăng: bản gốc đã đạt 10/10, bản cuối giữ đạt (mục cuối).
27. **DONE giả có giảm không?** Trên bộ LLM thật: 0 → 0. Trên các ca hỏng tái lập được: model trả lời rỗng (rc 0 → 1), dừng vì
    lặp (rc 0 → 1), tóm tắt bị cắt (nhận → từ chối), verify xanh sẵn mà không đổi gì (GREEN rc 0 → `NO_CHANGE` rc 1).
28. **Lỗi lặp lại có giảm không?** Không có kịch bản đo lỗi lặp giữa các lần. Một dạng lặp trong lượt đã có trần: model làm lại
    việc sau mỗi lần tóm tắt (1.338 lời gọi / 120 s → tối đa 10 lần tóm tắt rồi dừng có lỗi).
29. **Tiếp việc (resume) có tốt hơn không?** Giữ nguyên: s10 đạt ở cả hai. Thêm: lời trả lời dở sau sập được đóng (PRB-14).
30. **TOÀN HỆ THỐNG có tốt hơn đo được không?** Có, ở trục đúng/an toàn (các số trên, đều chạy lại trên revision cuối). Ở trục
    hiệu quả: chưa kết luận. Ở trục giải việc: giữ nguyên (đã ở trần của bộ kịch bản).

## Số trước / sau

Chi tiết: [15 (Daily)](../daily-work/15_BEFORE_AFTER_METRICS.md). Tóm tắt:

| Đo | Trước (binary gốc) | Sau (binary cuối) |
|---|---|---|
| Lời gọi lọt sang provider khác, kịch bản chỉ Qwen (E3) | 799 | 0 |
| Ca tiêm lỗi có request lọt provider khác (E3) | 8/8 | 0/11 |
| Model trả lời rỗng (E3) | rc=0 (DONE giả) | rc=1 |
| Tóm tắt lặp khi model làm lại việc (E3) | hết giờ, 2.659 request | dừng có lỗi sau 10 lần |
| Suite Go so theo tên | 4.501 test | 4.637 test, 0 mất, 0 đổi trạng thái |
| Eval LLM thật, 9 kịch bản chung (bỏ s13) (E5) | 9/9 đạt · 87 lời gọi · 1.196.842 token vào · 325,8 s | 9/9 đạt · 69 lời gọi · 1.108.932 token vào · 325,0 s |
| Lời gọi đặt tiêu đề trong 9 kịch bản đó | 24 | 9 |
| s13 đổi provider có chủ ý (E5) | đạt | lần 1 trượt vì GLM treo 120 s (lỗi rõ, không đổi provider); chạy lại đạt |

Mỗi kịch bản E5 chạy một lần mỗi binary; chênh lệch token/thời gian nằm trong dao động giữa các lần chạy, nên không kết luận
về hiệu quả. Riêng số lời gọi tiêu đề giảm do thay đổi cấu trúc. Eval E5 đầy đủ chạy trên binary vòng 7; bản cuối (vòng 8)
chỉ khác ở đường client của chế độ server, đã chạy smoke LLM thật qua server trên chính binary cuối ([37](37_FINAL_PROOF.md)).

## Trạng thái cuối

**SYSTEM IMPROVED — WITH KNOWN LIMITATIONS**

Vì sao không phải "VERIFIED": hiệu quả chưa kết luận được (một lần chạy mỗi kịch bản); một số vấn đề chỉ ở mức đọc mã (câu 2);
chưa có routine nhiều giờ với LLM thật; hành vi TUI chưa kiểm bằng mắt. Vì sao không phải "INCONCLUSIVE": có bằng chứng
trước/sau trên cùng bộ đo, chạy lại trên revision cuối, cho các trục đúng/an toàn; không trục nào đo được tệ đi.
