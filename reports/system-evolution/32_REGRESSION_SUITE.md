# 32 — Bộ hồi quy (Phase 20)

Mỗi lỗi đã sửa có ít nhất một test chặn nó quay lại. Test đều nằm trong bộ `go test ./...` thường, không cần cờ hay môi trường đặc biệt. "Đỏ trước" = test chạy trên mã chưa sửa thì đỏ. "Đột biến" = bỏ từng phần của bản sửa bằng `go test -overlay` thì test đỏ (tệp đột biến ở `scratchpad/evo/mut*`, ngoài repo).

| Lỗi | Bất biến | Test (gói) | Đỏ trước | Đột biến |
|---|---|---|---|---|
| PRB-01 vòng tóm tắt vô hạn | tóm tắt không giải phóng được chỗ thì dừng có lỗi, không lặp | `TestAutoSummarizeStopsWhenSummaryCannotFreeRoom`, `TestAutoSummarizeStillRepeatsWhenItFreesRoom`, `TestAutoSummarizeKeepsFinalAnswerAfterSummary` (agent) | có | có |
| PRB-12 tóm tắt bị cắt được nhận | bản cắt dở không thành điểm cắt lịch sử | `TestTruncatedSummaryKeepsHistory` (agent) | có | có |
| PRB-27 loop detection rc=0 | dừng vì lặp là lỗi | `TestLoopDetectionStopIsAnError` (agent) | có | có |
| PRB-13 prompt xếp hàng chen trước lượt làm tiếp | làm tiếp sau tóm tắt chạy trước hàng đợi | `TestSummaryContinuationRunsBeforeQueuedUserPrompt` (agent) | có | — |
| PRB-14 lời trả lời dở sau sập | mở session thì đóng mọi lời trả lời chưa kết thúc | `TestSettleInterruptedCalls_ClosesUnfinishedTextReplies`, `…_LeavesFinishedRepliesAlone` (agent) | có | có |
| PRB-03 small khác provider | mọi lời gọi ẩn ở lại provider của model chính | `TestSameProviderSmall`, `TestBuildAgentModelsKeepsSmallOnTheMainProvider` (agent) | có | có |
| PRB-46, PRB-74 chi phí tiêu đề | session con không gọi tiêu đề; không thử lại sau `length` | `TestNoTitleCallForChildSessions`, `TestTitleStopsAfterTokenLimit`, `TestTitleRetriesTheLargeModelAfterAnError` (agent) | có | — |
| PRB-09 agentic_fetch bỏ qua `disabled_tools` | phạm vi đi theo tool con | `TestFilterDisabledToolsAppliesToInnerTools`, `TestAgenticFetchUsableNeedsAWebTool` (agent) | có | có |
| PRB-32 (giảm nhẹ) token không thật | sổ ghi mỗi lời gọi: vai trò, provider, model, usage | `TestLedgerRecordsEachCallWithRoleProviderAndUsage`, `TestLedgerOffReturnsTheModelUnchanged` (agent) | — (tính năng mới) | — |
| PRB-06 hộp thoại một session chặn session khác | chờ quyền huỷ được; phiên tự duyệt không xếp hàng | `TestWaitingRequestIsCancellable`, `TestAutoApprovedSessionDoesNotWaitForOtherPrompts` (permission) | có | có |
| PRB-70 ghi ra ngoài khi không người trực | từ chối cứng trước YOLO/allowlist | `TestUnattendedRefusesWritesOutsideTheWorkingDir` (permission) | có | có |
| PRB-25 `git -C` ra repo khác | cờ trỏ repo khác bị chặn khi ra ngoài workspace | `TestUnattendedGitDeniesFlagsThatPointAtAnotherRepo` (shell) | có | có |
| PRB-67 viết tắt option dài | so khớp tiền tố như git | `TestUnattendedGitDeniesAbbreviatedLongOptions` (shell) | có | có |
| PRB-08 (một phần) skill trong `.trash` | không quét thư mục ẩn | `TestDiscoverSkipsHiddenDirectories` (skills) | có | — |
| PRB-10 log ghi giá trị env | chỉ ghi tên khoá khác nhau | `TestFirstWinsMismatch_NeverLogsEnvValues` (backend) | có | — |
| PRB-02 routine bị kill bị giấu | lượt bỏ dở hiện ABANDONED; RUNNING thấy được khi đang chạy | `TestLastReport_AbandonedRunsAreShownNotHidden`, `TestExecuteRoutine_RunningReportIsVisibleDuringTheRun` (cmd) | có | có |
| PRB-07 lịch routine thiếu biến launcher | lịch mang biến launcher an toàn, từ chối giá trị lạ | `TestScheduleEnvCarriesTheLauncherVariables` (cmd), `TestScheduleText_CarriesLauncherEnv`, `TestScheduleText_RefusesUnsafeEnvValues` (routine) | có | có |
| PRB-66 agent sửa test/cổng | báo cáo liệt kê tệp đổi, cờ `touched_tests`/`touched_gate` | `TestExecuteRoutine_ReportFlagsTestAndGateChanges`, `TestExecuteRoutine_ReportNoFlagsForOrdinaryChanges` (cmd) | có | có |
| PRB-24 GREEN không đo gì | pipefail; cờ `baseline_pass`/`unchanged` | `TestShellVerify_PipelineFailureIsFail` (cmd), `TestEngine_NoChangeCarriesBaselineAndUnchangedFlags`, `TestEngine_GreenAfterARealChangeHasNoFlags`, `TestEngine_UnknownFingerprintIsNotUnchanged` (loop) | có | có |

### Đợt 2: sửa theo lượt soát độc lập (38) và phát hiện mới

| Lỗi | Bất biến | Test (gói) | Đỏ trước | Đột biến |
|---|---|---|---|---|
| PRB-75 lượt rỗng rc=0 | lượt không chữ, không tool call ở mọi bước là lỗi; làm tool rồi im lặng thì không | `TestEmptyTurnIsAnError`, `TestToolWorkThenSilentEndIsNotAnError` (agent) | có | — |
| PRB-88 tự tóm tắt không trần | tối đa 10 lần tự tóm tắt mỗi prompt khi việc còn dở | `TestAutoSummarizeHasACapPerPrompt` (agent) | có (1.000 lần tóm tắt, 15 s, không lỗi) | — |
| PRB-81 sổ ghi chép văn bản lỗi | chỉ loại lỗi + mã HTTP | `TestLedgerRecordsTheErrorKindNotItsText` (agent) | có | — |
| PRB-87 gốc skill ẩn có "/" cuối | gốc nhận bằng `filepath.Rel == "."` | `TestDiscoverKeepsAHiddenBaseWrittenWithATrailingSlash` (skills) | có | — |
| PRB-76 git đổi repo qua biến môi trường | không người trực: git/git-*/tên động có biến `GIT_DIR`… bị chặn ở cả 3 dạng | `TestUnattendedGitDeniesRedirectingEnvEndToEnd` (8 ca), `TestUnattendedGitEnvGuardUnit`, `TestUnattendedGitAllowsUnrelatedEnv`, `TestUnattendedGitAllowsInheritedEnv` (shell) | có | 9/9 |
| PRB-77 LSP ghi ra ngoài | `lsp_replace_symbol` có Action write + FilePath; `lsp_rename` hỏi từng tệp khi không người trực | `TestReplaceSymbolRefusedOutsideWhileUnattended`, `TestRenameRefusedWhenAnyFileOutsideWhileUnattended`, `TestRenameAttendedAsksNoPerFilePermission` (tools) | có | 3/3 |
| PRB-79 agentic_fetch URL khi tắt web_fetch | chế độ URL từ chối trước khi hỏi quyền/tải | `TestAgenticFetchURLModeRefusedWhenWebFetchDisabled` (agent) | có | 1/1 |
| PRB-63 `run` local treo vì quyền session con | duyệt session có gốc là session của lượt; tra lỗi thì từ chối | `TestRunNonInteractiveGrantsDescendantSessions` (app) | có | 1/1 |
| F6 dây nối | cổng đăng ký agentic_fetch, tool con, không tự duyệt, `provider`/`model` của `run --json` | `TestBuildToolsGatesAgenticFetchOnWebTools`, `TestAgenticFetchInnerToolsRespectDisabledTools`, `TestAgenticFetchChildSessionNotAutoApproved` (agent), `TestRunJSONReportsProviderAndModel` (cmd) | xanh sẵn (mã đúng) | 4/4 |
| PRB-82 lỗi sớm của routine thành ABANDONED | RUNNING ngay sau khi có thư mục lượt; lỗi sớm ghi ERROR + nguyên nhân | `TestExecuteRoutine_WorktreeFailureWritesAnErrorReport` (cmd) | có | 1/1 |
| PRB-83 chưa có báo cáo mà khoá giữ | lượt mới nhất không báo cáo + khoá giữ = RUNNING | `TestListRuns_NewestRunWithoutReportIsRunningWhileLocked` (cmd) | có | 1/1 |
| PRB-80 lịch đọc biến sau `.env` | đọc từ ảnh chụp trước `.env` (`execpolicy.OriginalEnv`); mang theo allowlist (phần allowlist **đã lật ở vòng 6**, L6) | `TestScheduleEnv_IgnoresProjectDotenv`, `TestScheduleEnv_CarriesTheProviderBoundary` (cmd), `TestScheduleText_EnvAllowsModelAllowlist` (routine) | có | 3/3 |
| PRB-86 exit 141 với pipefail | FAIL exit 141 kèm gợi ý SIGPIPE | `TestShellVerify_SigpipeFailureExplainsItself` (cmd) | có | 1/1 |

### Đợt 3–4: sửa theo lượt soát 2 và 3

| Lỗi | Bất biến | Test (gói) | Đỏ trước | Đột biến |
|---|---|---|---|---|
| PRB-76 (lượt 2) git qua env trong script lồng, `env -S`, `find -exec`, `eval` | không người trực: lệnh nào nhắc biến git (argv, script, giá trị biến export) đều bị chặn | 7 ca mới trong `TestUnattendedGitDeniesRedirectingEnvEndToEnd` (tổng 15), `TestUnattendedGitEnvGuardUnit` (thêm ca lệnh không phải git) | có (7 ca xoá được tệp ngoài trên cơ chế cũ) | 4/4 |
| PRB-76 (lượt 3, F5) script có shebang | lượt soát env chạy cả ở nhánh shebang | `TestUnattendedGitEnvDoesNotReachShebangScripts` (shell) | có | 1/1 |
| F4 lý do chặn | output tool bash có "Blocked: <lý do>" | `TestBashTellsTheModelWhyAnUnattendedCommandWasBlocked` (tools) | có | 1/1 |
| PRB-75 (LC2-1/2/4, F1/F2) lượt rỗng và chuỗi chưa ra gì | xem 38 | `TestLengthCutWithoutTextIsContinuedNotEmpty`, `TestContentFilterRefusalIsNotReportedAsEmpty`, `TestSilentContinuationAfterSummaryIsNotAnError`, `TestEmptyTurnWithOtherFinishReasonsIsAnError`, `TestLengthCutChainThatNeverProducesIsAnError` (agent) | có | 3/3 (đợt 3) + 2/2 (đợt 4) |
| PRB-88 (LC2-3, F3) trần tóm tắt và chốt tràn context với tool call viết dở | cùng điều kiện "tool call của tin assistant" | `TestAutoSummarizeHasACapPerPrompt` (2 biến thể), `TestAutoSummarizeStopsWhenSummaryCannotFreeRoom` (2 biến thể) | có | 1/1 + 1/1 |
| PRB-80 (SEC2-5) allowlist cách bằng dấu cách | đổi thành phẩy trong lịch (**đã lật ở vòng 6**: allowlist không vào lịch nữa; test giữ tên, nay khẳng định không có allowlist) | `TestScheduleEnv_DropsASpaceSeparatedAllowlist` (cmd) | có | — |
| PRB-83 (LC2-5) thử khoá mỗi lần `listRuns` | chỉ thử khi cần | `TestListRuns_DoesNotProbeTheLockForAFinishedRun` (cmd) | có | — |

### Daily và vòng 6 (sửa theo red team cuối)

| Lỗi | Bất biến | Test (gói) | Đỏ trước | Đột biến |
|---|---|---|---|---|
| G-01 / PRB-04 fallback ghi xuống đĩa | lựa chọn tạm không ghi xuống global data | `TestStartupFallbackIsNotPersisted` (config) | có | — |
| G-08 panic log ở thư mục hiện tại | panic log vào thư mục log, 0600; dọn dẹp chạy cả khi không tạo được tệp | `TestRecoverPanicWritesToTheLogDirNotTheWorkingDir`, `TestRecoverPanicRunsCleanupWhenTheFileCannotBeCreated` (log) | có | — |
| F1 / PRB-04 đổi provider lặng lẽ | không bao giờ thay bằng model của provider khác; mất ID model thì model mặc định của cùng provider; provider không dùng được thì dựng agent lỗi nêu provider/model | `TestStartupFallbackIsNotPersisted` (đổi kỳ vọng), `TestSelectedModelGoneFallsBackWithinProvider`, `TestResolveSelectedModels_SameProviderCatalogDefault`, `TestResolveSelectedModels_NoCrossProviderFallback`, `TestConfig_configureSelectedModels` (đổi kỳ vọng ca reload) (config); `TestBuildAgentModelsFailsClosedWhenSelectedProviderHasNoKey`, `TestNoFallbackErrorKeepsSentinel` (agent); E2E `faults.sh` ca `selected-no-key` | có (5 test config đỏ do assert; 2 test agent đỏ qua overlay hàm rỗng); E2E: binary trước rc=0 + 2 request sang glm | 3/3 (config) |
| L5 tắt `fetch` không tắt `web_fetch` của agentic_fetch | `fetch` trong `disabled_tools` tính là tắt `web_fetch` ở cả 3 chỗ | `TestDisablingFetchCountsAsDisablingWebFetch`, `TestDisabledFetchAlsoGatesAgenticFetch` (agent) | có (overlay hàm rỗng) | 1/1 |
| F4 / PRB-24 GREEN giả | verify xanh ở baseline + cây không đổi = `NO_CHANGE`; `loop` thoát 1; routine maintenance 0, goal 1 | `TestEngine_NoChangeCarriesBaselineAndUnchangedFlags` (đổi kỳ vọng), `TestEngine_BaselinePassWithAChangeIsGreenWithAWarning`, `TestEngine_BaselineFailWithoutAChangeIsGreenWithAWarning` (loop); `TestExecuteLoop_NoChangeWhenTheGateAlreadyPassesAndNothingChanged`, `TestLoopExitErr_OnlyGreenExitsZero`, `TestRoutineExitErr_NoChangeIsZeroOnlyForMaintenance`, `TestExecuteRoutine_NoChangeIsReportedAndListed`, `TestExecuteRoutine_NoChangeInGoalModeExitsNonZero` (cmd); E2E `qwen-only.sh` (đường loop) | có (engine + 2 test routine); E2E: binary trước `stop: GREEN` rc=0 | 2/2 |
| L6 allowlist hai chủ | lịch không chép allowlist; cảnh báo khi allowlist chỉ có ở shell | `TestScheduleEnv_CarriesTheProviderBoundary`, `TestScheduleEnv_DropsASpaceSeparatedAllowlist`, `TestScheduleEnv_IgnoresProjectDotenv` (đổi kỳ vọng), `TestWarnAllowlistNotScheduled` (cmd) | có | 2/2 |
| L1 chốt tràn context đếm cả output | chỉ xét phần đầu vào của bước đầu sau tóm tắt | `TestLargeOutputRightAfterSummaryIsNotAnOverflow`, `TestAutoSummarizeStopsWhenSummaryCannotFreeRoom` (agent) | có | 1/1 (bỏ chốt → 2 biến thể cũ đỏ) |
| L2 / PRB-45 `text` dính bản tóm tắt | in câu trả lời, không in tin tóm tắt (local + đường server dự phòng) | `TestRunStream_SkipsTheAutoSummary` (cmd); E2E `qwen-only.sh` (`text` = câu trả lời cuối) | E2E: binary trước `text` có `SUMMARY-OF-SESSION` | 1/1 |

### Vòng 7 (sửa theo lượt soát độc lập vòng 6)

| Lỗi | Bất biến | Test (gói) | Đỏ trước | Đột biến |
|---|---|---|---|---|
| `agent-cli login` hỏng khi model đang chọn không dùng được | app khởi động không agent cho đúng 4 lỗi "không dùng được"; lỗi khác (allowlist, chưa chọn model) vẫn dừng | `TestCreateWorkspaceStartsWhenSelectedModelIsUnavailable` (backend, qua `CreateWorkspace` → `app.New`), `TestSelectedModelUnavailableCoversOnlyUnusableSelections` (agent); E2E `faults.sh` probe `login` | có (`CreateWorkspace` lỗi "failed to initialize coder agent"; binary vòng 6 FAIL probe) | 2/2 (hàm luôn đúng / luôn sai) |
| TUI không mở được để chọn model khác | model chính đã chọn mà không dùng được → onboarding | `TestUnavailableSelectedModelOpensOnboarding` (ui/model) | có | — |
| Provider dựng sẵn bị tắt vẫn bị gọi | `Disable` = không gọi | `TestBuildAgentModelsRefusesADisabledProvider` (agent; vòng 8: đặt `Disable` trong bộ nhớ để không phụ thuộc thứ tự chạy) | có | 1/1 (ở cả 3 cách chạy) |
| Đường nạp lại chưa có test | nạp lại cũng không đổi provider | `TestReloadDoesNotSwitchProviderWhenTheKeyDisappears` (config; do người soát viết) | xanh sẵn trên mã vòng 6 (mã đúng) | 1/1 (khôi phục fallback cũ → đỏ) |

### Vòng 8 (sửa theo lượt soát độc lập vòng 7)

| Lỗi | Bất biến | Test (gói) | Đỏ trước | Đột biến |
|---|---|---|---|---|
| Qua server, lỗi dựng agent chỉ còn "status code 500" | client giữ nội dung lỗi của server | `TestInitiateAgentProcessingDecodesErrorBody` (client); E2E `faults.sh` ca `selected-no-key-server` | có (test; binary vòng 7 FAIL ca server) | — |

"Đỏ trước" và "Đột biến" của vòng 6: P và Q tự chạy (báo cáo của họ trong journal workflow `wf_4dd0f31f-164`, tệp đột biến ở `scratchpad/evo/mut-p/`, `mut-q/`); L1, L2 do lead chạy (`mut-l1/`, `mut-l2/`). Người soát độc lập vòng 6 chạy lại một phần: xem [38](38_RED_TEAM.md).

Cột "Đột biến" ghi "—" khi không chạy đột biến cho mục đó. CLAUDE.md P2 của dự án chỉ đòi đột biến cho code an ninh. Cột "Đỏ trước" và "Đột biến" là kết quả lúc viết từng bản sửa (TDD trong phiên này). Cột "Đỏ trước" được chạy lại cơ học ở [37](37_FINAL_PROOF.md): chép tệp test lên cây sạch `4a0b25b6`, ghi từng test là đỏ do assert, đỏ do không biên dịch, hay xanh.

## Test có sẵn bị đổi (phải biết khi soát)

| Test | Đổi gì | Vì sao không phải "sửa test cho xanh" |
|---|---|---|
| `TestRepeatedSummaryDoesNotNestThePrompt` (continuation_test.go) | Kịch bản cho phần làm tiếp sau tóm tắt có tiến triển (thêm `summaryFinish`) | Kịch bản cũ dựa vào chính lỗi PRB-01: làm tiếp không tiến triển mà vẫn tóm tắt lại. Ý định của test (không lồng prompt khi tóm tắt nhiều lần) giữ nguyên; bản sửa PRB-01 dừng đúng kịch bản cũ |
| `routine_test.go` | Chỉ thêm tham số `nil` vào các lời gọi `ScheduleText` | Đổi chữ ký hàm |
| `TestStartupFallbackIsNotPersisted` (model_pin_test.go) | `NotEqual("openai")` → `Equal("openai")` + `Equal("gpt-4")` | Test cũ khẳng định chính hành vi bị cấm (đổi sang provider khác trong RAM, F1). Phần "không ghi xuống đĩa" giữ nguyên |
| `TestConfig_configureSelectedModels`, ca "reload mode should not persist fallback defaults" (load_test.go) | `True(LargeFallback)` → `False`; kỳ vọng `openai/large-model` → `ghost/missing` | Như trên: provider `ghost` không có thì giữ nguyên lựa chọn, không đổi sang openai. Phần đĩa giữ nguyên |
| `TestEngine_NoChangeCarriesBaselineAndUnchangedFlags` (flags_test.go) | `StopGreen` → `StopNoChange` | Đúng tình huống F4 gọi là GREEN giả. Cờ và `Detail` vẫn được kiểm |
| `TestScheduleEnv_CarriesTheProviderBoundary`, `…NormalisesASpaceSeparatedAllowlist`, `…IgnoresProjectDotenv` (routine_status_test.go) | "có allowlist" → "không có allowlist" | Đúng thay đổi L6. `AGENT_CLI_ENV_FILE`, `/evil`, `CRUSH_GLOBAL_DATA` vẫn được kiểm. Hai test này đổi tên theo nghĩa mới ở vòng 7 (`TestScheduleEnv_DropsASpaceSeparatedAllowlist`; còn `TestEngine_NoChangeCarriesBaselineAndUnchangedFlags`) |

## Ngoài Go test

| Kiểm tra | Lệnh | Ghi chú |
|---|---|---|
| E2E run/send/loop/routine | `scripts/test-agent-cli-run-json.sh`, `test-agent-cli-loop.sh`, `test-agent-cli-routine.sh` | chạy một lần ở cuối (37) |
| Cô lập provider với binary thật | `scratchpad/evo/probe-provider-roles.sh` | P-01, P-07: đếm phía server giả |
| Harness hệ thống | `scripts/eval/run.sh` | so baseline ↔ ứng viên (37) |
| Cổng cô lập provider + tiêm lỗi | `scripts/eval/faults.sh`, `scripts/eval/qwen-only.sh` | thoát 1 khi: request lọt provider khác, ca thiếu key không dừng, `text` lẫn tóm tắt, loop xanh sẵn không ra `NO_CHANGE` |
| Nhiều workspace một server | `scripts/eval/workspaces.sh` | thoát 1 khi hỏng |
