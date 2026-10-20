# 26 — Lỗi của Hindsight → bất biến của agent-cli (Phase 10)

Câu hỏi bắt buộc: **agent-cli có cùng bề mặt hỏng không?** Không → NOT_APPLICABLE. Có → nguyên nhân gốc → bất biến chung → bề mặt ở agent-cli → test → hành vi bắt buộc. ID `H-xx` ở [24](24_HINDSIGHT_EVIDENCE_LEDGER.md); `PRB-xx` ở [05](05_CURRENT_PROBLEMS.md). Chúng ta học từ lỗi của Hindsight, không nhận backlog của nó.

## Áp dụng (agent-cli có bề mặt tương tự)

| Hindsight | Bất biến chung | Bề mặt ở agent-cli | Test / bằng chứng | Trạng thái |
|---|---|---|---|---|
| H-01 worker chết, id khác → `processing` mãi | **Mọi việc dài có đường tới trạng thái cuối; người thay thế bất kỳ thu hồi được việc bỏ dở** | routine bị kill → bị giấu sau lượt GREEN cũ (PRB-02); lời trả lời dở sau sập quay mãi (PRB-14) | `TestLastReport_AbandonedRunsAreShownNotHidden`, `TestExecuteRoutine_RunningReportIsVisibleDuringTheRun`, `TestSettleInterruptedCalls_ClosesUnfinishedTextReplies` | Đã sửa (wt). Khác Hindsight: "lease" của routine là flock, kernel tự nhả khi chết, nên **mọi** tiến trình sau (khác danh tính) thu hồi được — không cần id worker |
| H-02 cùng id khởi động lại thì tự lành | Đọc lại phải tự sửa trạng thái mồ côi | settle tool call mồ côi khi mở session (có sẵn) | test có sẵn | Đã có |
| H-08, H-09 "0 kết quả mà `completed`" | **Rỗng/hỏng không bao giờ thành thành công; rỗng hợp lệ phân biệt được với rỗng do lỗi** | loop GREEN với verify không đo gì (PRB-24); loop detection rc=0 (PRB-27); tóm tắt bị cắt được nhận (PRB-12); tiêu đề `length` bị bỏ qua lặng | `TestEngine_NoChangeCarriesBaselineAndUnchangedFlags`, `TestShellVerify_PipelineFailureIsFail`, `TestLoopDetectionStopIsAnError`, `TestTruncatedSummaryKeepsHistory` | Đã sửa (wt); "0 test" chưa nhận diện |
| H-10 báo "xong" trước khi biết kết quả cuối | Sự kiện hoàn tất phát SAU trạng thái cuối, đúng một lần | RunComplete phát một lần có lỗi (có sẵn); báo cáo routine: RUNNING lúc đầu, kết quả lúc cuối, ghi nguyên tử | `writeReport` qua tệp tạm + rename | Đã sửa (wt) |
| H-05 chỉ một số việc có trần thời gian | Mọi lời gọi ra ngoài có trần | MCP CallTool không timeout (PRB-39); client probe không timeout (PRB-72) | — | Mở |
| H-13 kết quả suy giảm không cờ | Trạng thái suy giảm phải mang cờ | ước lượng usage `~` mất (PRB-33); "Modified Files" thiếu nguồn (PRB-34) | — | Mở |
| H-15 số báo cáo lấy sai khoá | Số đo phải đúng ngữ nghĩa | token `run --json` chỉ bước cuối (PRB-32) | sổ ghi LLM (P-06) | Giảm nhẹ |
| H-19 một việc độc chặn việc khác | Một yêu cầu kẹt không được chặn yêu cầu không liên quan | hộp thoại quyền một session chặn session khác (PRB-06); "kẹt" xét cả workspace (PRB-17) | `TestWaitingRequestIsCancellable`, `TestAutoApprovedSessionDoesNotWaitForOtherPrompts` | PRB-06 đã sửa (wt); PRB-17 mở |
| H-21..H-24 dữ liệu dẫn xuất sống lâu hơn nguồn | **Xoá nguồn phải quét dữ liệu dẫn xuất** | xoá session sót session con + transcript (PRB-11) | — | Mở |
| H-31 hook lỗi chỉ ghi tệp diag | Thất bại phải hiện ra | Stop/PreToolUse fail-open (PRB-38); trần Stop hook không hiện trong TUI (PRB-36) | — | Mở (có tài liệu) |
| H-36, H-37 lưu nguyên văn prompt/payload, giữ mãi | **Secret không được lưu bền nếu tránh được; có chính sách giữ** | log từng ghi nguyên env (PRB-10); DB lưu đủ output tool, không retention (PRB-49) | `TestFirstWinsMismatch_NeverLogsEnvValues` | PRB-10 đã sửa (wt); PRB-49 mở |
| H-40, H-46 đọc cấu hình lỗi → âm thầm dùng mặc định | **Đầu vào sai ở biên tin cậy bị từ chối/báo, không âm thầm nới** | fallback chéo provider lúc khởi động (PRB-04); small khác provider (PRB-03); tên lạ trong `disabled_tools` bị bỏ qua | `TestBuildAgentModelsKeepsSmallOnTheMainProvider`, `TestStartupFallbackIsNotPersisted`, `TestResolveSelectedModels_NoCrossProviderFallback`, `TestBuildAgentModelsFailsClosedWhenSelectedProviderHasNoKey`, `TestBuildAgentModelsRefusesADisabledProvider`, `TestReloadDoesNotSwitchProviderWhenTheKeyDisappears` | PRB-03 đã sửa (wt); PRB-04 đã sửa (Daily + vòng 6–7: không đổi provider, dừng có lỗi); tên lạ trong `disabled_tools` vẫn bị bỏ qua (mở) |
| H-43, H-44 ranh giới khai báo mà không ép / gõ sai thì nới | **Phạm vi đi theo cả thao tác dẫn xuất** | `disabled_tools` không áp vào agentic_fetch (PRB-09); allowlist chỉ kiểm nhãn (PRB-30); hook không chạy trong agent con (PRB-41) | `TestFilterDisabledToolsAppliesToInnerTools`, `TestAgenticFetchUsableNeedsAWebTool` | PRB-09 đã sửa (wt) |
| H-35 mặc định không xác thực | Mặc định an toàn có chủ ý | socket trong thư mục 0700 (wrapper, có sẵn); chế độ tcp không xác thực (PRB-29) | — | Mở (wrapper giảm nhẹ) |
| Bài học issue #10 "sửa phải tới mọi bản sao" | Một luật áp cho mọi bản sao của nó | hàng đợi câu hỏi có trần nhưng hàng đợi quyền thì không (PRB-06); allowlist cho large nhưng small đi provider khác (PRB-03) | như trên | Đã sửa (wt) |
| Bài học issue #11 "client không bền → ack sau khi ghi" | Chỉ báo đã nhận sau khi đã ghi bền | `send` (không `--wait`) thoát 0 trước khi prompt được ghi; hàng đợi mất khi sập (PRB-16) | — | Mở |

## Không áp dụng hôm nay (không có thành phần tương ứng)

| Hindsight | Vì sao NOT_APPLICABLE | Thành áp dụng khi |
|---|---|---|
| H-17, H-18, H-20 consolidation đóng dấu / xoá sạch / dedup | không có consolidation | thêm trí nhớ dẫn xuất bằng LLM |
| H-26, H-27 reflect `based_on` quá rộng, recall khác nhau | không có reflect | thêm reflect |
| H-28 vòng nhân bản qua dòng ghi công | không có retain cuối lượt | thêm memory có retain từ transcript → phải có test "memory đã tiêm không được retain lại" |
| H-29, H-30 cursor ở /tmp, sessionId thiếu | agent-cli lưu ở data dir; session luôn có id | — |
| H-32 một bank mỗi repo cho mọi nhánh | không có bank | thêm memory → phạm vi phải gồm nhánh/worktree (MEM-012/013) |
| H-33 memory dạng mệnh lệnh | không có memory tự sinh | thêm memory → chỉ lưu câu khai báo có nguồn |
| H-42 FK thiếu ở tenant 2+ | một DB mỗi dự án, không tenant | — |
| H-47..H-50 các tầng recall | không có recall | — |
| H-03, H-04, H-06 | lượt ghi báo cáo luôn giữ khoá; định nghĩa hỏng hiện INVALID, báo cáo hỏng thành ABANDONED | — |
