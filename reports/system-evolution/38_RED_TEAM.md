# 38 — Soát độc lập / red team (Phase 30)

Người soát: phiên riêng, loại agent không có quyền Edit/Write, chỉ nhận **yêu cầu** (ý định của từng thay đổi) + patch + mã, không nhận kết luận của người làm. Mỗi phát hiện phải có kịch bản hỏng cụ thể; người soát tự chạy overlay test để xác nhận khi được.

## Lượt 1 — toàn bộ ứng viên (`evidence/wt-candidate-1.patch`, 45 tệp)

Ba góc nhìn chạy song song: an ninh + cô lập provider · vòng đời + đồng thời · toàn vẹn test + thành công giả.

| Góc | Kết luận | Phát hiện |
|---|---|---|
| An ninh | KHONG-DAT | 2 MEDIUM (đều xác nhận bằng overlay), 4 LOW |
| Vòng đời | KHONG-DAT | 1 MEDIUM (xác nhận), 3 LOW |
| Toàn vẹn | DAT | 6 LOW (4 xác nhận) |

Gộp trùng: 13 phát hiện khác nhau.

| Phát hiện | Mức | Xử lý | Bằng chứng sau xử lý |
|---|---|---|---|
| SEC-1 git đổi repo qua biến môi trường (`GIT_DIR=… git rm`, `env …`, `sh -c`) → PRB-76 | MEDIUM | Sửa: biến git đi qua cả 3 dạng, chặn khi không người trực | 8 ca end-to-end trên repo git thật; đối chứng có người trực thì tệp mất; 9/9 đột biến |
| SEC-2 / F3 tool LSP ghi tệp không qua kiểm tra ghi ra ngoài → PRB-77 | MEDIUM | Sửa: Action write + đường dẫn; `lsp_rename` hỏi từng tệp khi không người trực | 3 test, 3/3 đột biến |
| LC-1 / F5 lỗi sớm của routine hiện ABANDONED → PRB-82 | MEDIUM | Sửa: RUNNING ngay khi có thư mục lượt, lỗi sớm ghi ERROR kèm nguyên nhân | test tái lập đúng kịch bản của người soát |
| LC-2 chưa có báo cáo mà khoá giữ → PRB-83 | LOW | Sửa | test |
| SEC-3 comment PRB-70 ngụ ý bash cũng bị chặn | LOW | Sửa comment; bash ghi ra ngoài là giới hạn đã biết (PRB-61, "không phải sandbox") | — |
| SEC-4 agentic_fetch URL khi tắt web_fetch → PRB-79 | LOW | Sửa | test + đột biến |
| SEC-5 lịch đọc biến sau `.env`; allowlist không theo lịch → PRB-80 | LOW | Sửa: ảnh chụp env trước `.env` (`OriginalEnv`), mang theo `AGENT_CLI_ENV_FILE`/`AGENT_CLI_ALLOWED_MODELS` | 3 test (một test chạy tiến trình con mô phỏng đúng `main`) |
| SEC-6 sổ ghi chép văn bản lỗi provider → PRB-81 | LOW | Sửa: chỉ loại lỗi + mã HTTP | test |
| LC-3 MCP chết sau App cuối trong thời gian server chờ → PRB-84 | LOW | **Giới hạn đã biết**: có từ trước, bản sửa PRB-15 làm hẹp lại | — |
| LC-4 `KillOwned` theo gốc trong cửa sổ teardown → PRB-85 | LOW | **Giới hạn đã biết**: cửa sổ hẹp, sửa đúng cần chủ job theo App | — |
| F1 pipefail làm `\| grep -q` FAIL 141 → PRB-86 | LOW | Giữ pipefail (đóng an toàn), thêm gợi ý SIGPIPE | test |
| F2 gốc skill ẩn có "/" cuối mất hết skill (hồi quy) → PRB-87 | LOW | Sửa | test |
| F4 comment sai + `run` local treo vì quyền session con (hồi quy do bỏ tự duyệt agentic_fetch; cũng là PRB-63) | LOW→MEDIUM | Sửa: `run` local duyệt session con của lượt như chế độ server; TUI vẫn hỏi | test + đột biến |
| F6 dây nối không có test | LOW | Thêm 4 test | 4/4 đột biến |

Người soát cũng ghi các vùng đã kiểm mà không thấy lỗi: điểm chặn provider duy nhất ở `buildAgentModels`; sổ ghi không bật được từ `.env` của dự án; semaphore quyền không rò slot; vòng tóm tắt không vô hạn ở kịch bản phần cố định quá lớn; test mới đều đỏ trên mã cũ (đọc đường cũ); test có sẵn bị đổi không yếu đi.

Ba agent sửa trong ba worktree không giao tệp; mỗi bản sửa theo TDD (đỏ → xanh), so tên test trước/sau, đột biến cho code an ninh. Gộp lại: toàn bộ `go test ./...` so theo tên với cây sạch: 0 test biến mất, 0 đổi trạng thái, 76 test mới đều xanh (`evidence/suite-compare-1.txt`).

## Phát hiện tự tìm thêm trong lúc sửa (không do người soát)

| Phát hiện | Cách thấy | Xử lý |
|---|---|---|
| PRB-75 model trả lời rỗng → rc=0 | `faults.sh` ca `empty-reply` (cả bản gốc lẫn bản sửa) | Sửa (S-9) |
| PRB-88 tự tóm tắt không có trần cho một prompt | `qwen-only.sh`: model làm lại việc sau mỗi lần tóm tắt → 1.338 lời gọi main / 120 s, rc=124, ở cả hai binary | Sửa: tối đa 10 lần/prompt khi việc còn dở |

## Lượt 2 — phần sửa sau lượt 1 (`evidence/wt-delta-2.patch`, 30 tệp)

| Góc | Kết luận | Phát hiện |
|---|---|---|
| An ninh | KHONG-DAT | 2 MEDIUM + 3 LOW, 4 xác nhận bằng overlay trên repo git thật |
| Vòng đời + toàn vẹn | KHONG-DAT | 1 MEDIUM + 4 LOW (đọc mã; người soát này không được tạo tệp) |

| Phát hiện | Mức | Xử lý (đợt 3, `evidence/wt-delta-3.patch`) |
|---|---|---|
| SEC2-1 `export 'GIT_DIR=…'` có ngoặc / `declare -x "…"` / tên động trong `sh -c` lọt | MEDIUM | **Đổi cơ chế**: bỏ phân tích phép gán (protect.go về nguyên bản), thay bằng luật theo chữ: lệnh nào có chữ nhắc tới biến git (kể cả trong script `sh -c`, chuỗi `env -S`, `eval`, tiền tố `env NAME=…` cho biến đã export, kể cả biến có GIÁ TRỊ nhắc tới tên đó) đều bị chặn khi không người trực, bất kể là lệnh gì |
| SEC2-2 gán trước khi gọi hàm trong `sh -c` lọt | MEDIUM | Như trên |
| SEC2-3 `env -S`, `find -exec`, `eval "$X"`, `env $a` lọt | LOW | Như trên. Còn lại (ghi trong comment): tên ghép lúc chạy (`"GIT_""DIR"`), script đọc từ tệp/stdin |
| SEC2-4 báo nhầm: biến shell thường tên `GIT_DIR` trong script | LOW | **Chấp nhận** (nghiêng về chặn), ghi trong comment; luật mới còn chặn cả lệnh không phải git mang biến git, vì biến đi theo tiến trình con |
| SEC2-5 allowlist cách bằng dấu cách làm `routine schedule` hỏng | LOW | Sửa: đổi khoảng trắng thành phẩy (cùng nghĩa với parser của policy) |
| LC2-1 kiểm "lượt rỗng" chặn nhánh tự làm tiếp khi bị cắt giữa tool call (fantasy không phát ToolCall khi finish=length) — **hồi quy do S-9** | MEDIUM | Sửa: chỉ coi là rỗng khi bước cuối kết thúc `stop` |
| LC2-2 bộ lọc nội dung từ chối bị đổi thành "Empty reply" | LOW | Sửa: như trên (giữ finish content_filter) |
| LC2-3 trần tóm tắt bị vòng qua khi bước đầy context bị cắt giữa tool call | LOW | Sửa: trần dùng cùng điều kiện với chỗ xếp lượt làm tiếp (tool call của tin assistant, kể cả viết dở) |
| LC2-4 lượt làm tiếp im lặng bị coi là lỗi | LOW | Sửa: bỏ qua kiểm rỗng ở lượt làm tiếp |
| LC2-5 `listRuns` thử khoá mỗi lần gọi (cửa sổ làm lượt theo lịch thoát 75) | LOW | Sửa: chỉ thử khi báo cáo mới nhất thiếu hoặc RUNNING |

Bằng chứng đợt 3: test mới đỏ trên cơ chế cũ (7 dạng vòng qua của lượt 2 đều xoá được tệp ngoài workspace; 3 test lượt rỗng; 2 test routine), xanh sau sửa; đột biến 3/3 (agent) + 4/4 (shell). Ba kỳ vọng của test cũ đổi theo chính sách chặt hơn, có ghi chú trong test: `env GIT_DIR=/o ls`, `sh -c 'GIT_DIR=x make'` nay bị chặn; `GIT_DIR=x echo` thay bằng `MY_GIT_DIR=x git status` (phải qua).

## Lượt 3 — đợt sửa 3 (`evidence/wt-delta-3.patch`, 10 tệp)

Một người soát, phạm vi hẹp. Kết luận **KHONG-DAT** (1 MEDIUM + 4 LOW, cả 5 xác nhận bằng probe; probe e2e dùng chính provider openai-compat của fantasy với server SSE giả).

| Phát hiện | Mức | Xử lý (đợt 4, `evidence/wt-delta-4.patch`) | Bằng chứng |
|---|---|---|---|
| F1 kiểm "lượt rỗng" chỉ bắt finish `stop`; GLM `network_error`/`sensitive`, DeepSeek `insufficient_system_resource` (openai-compat ánh xạ thành `unknown`), google `error`/`other` vẫn rc=0. Từ chối không ra gì cũng rc=0 | MEDIUM | Đổi luật: chuỗi của prompt chưa ra chữ hay tool call nào (`call.produced`, mang qua mọi lượt làm tiếp) thì: `content_filter` → lỗi "bị từ chối" (giữ finish content_filter cho TUI), `length` → để nhánh tự làm tiếp tới khi hết trần, mọi finish khác → lỗi lượt rỗng | 3 test mới; probe của người soát: cả 5 finish nay ra lỗi đúng loại |
| F2 chuỗi bị cắt mà chưa ra gì vẫn thành công | LOW | Như trên (`produced` thay cho miễn trừ mọi lượt làm tiếp) | 3 test; probe: cả 3 chuỗi ra lỗi |
| F3 chốt tràn context sau tóm tắt không tính tool call viết dở | LOW | Dùng `currentAssistant.ToolCalls()` (cùng điều kiện với trần tóm tắt và chỗ xếp lượt làm tiếp) | test biến thể; probe: 12 lời gọi → 3 lời gọi + lỗi "context window too small" |
| F4 lệnh bị chặn không cho model biết lý do | LOW | Mỗi lời gọi tool bash gom lý do của lớp chặn không người trực và lớp bảo vệ tiến trình, nối "Blocked: <lý do>" vào output | test tool bash |
| F5 script có shebang nhận biến git không qua lượt soát env | LOW | Hàm dùng chung `blockedByGitEnv` chạy ở cả `blockHandler` và nhánh shebang | test e2e 2 dạng; probe của người soát: cả 4 dạng (make, shebang gán, shebang export, script không shebang) bị chặn |

Người soát cũng ghi: 21 lệnh việc thường ngày khi không người trực không bị chặn (git status/add/diff/log/commit -m, `GIT_AUTHOR_*`, `GIT_PAGER`…, make, script, `git commit -F`); biến git thừa kế (devcontainer/CI) được cho qua; chế độ có người trực không đổi trên 23 lệnh mẫu. Ngoài phạm vi, chỉ ghi lại: `GIT_EXTERNAL_DIFF`, `GIT_EDITOR`, `GIT_SSH_COMMAND` chạy được lệnh và không nằm trong danh sách (cùng lớp với "ngôn ngữ khác").

Đột biến đợt 4: 5/5 bị bắt (F1–F5, mỗi cái làm đỏ đúng test của nó).

## Lượt 4 — xác minh đợt sửa 4

Hai người soát chỉ-đọc (workflow `wf_5bfcf926-d1d`): một cho shell + tool bash, một cho agent. Cả hai **KHONG-DAT**. Cả 5 phát hiện của lượt 3 được xác nhận đã đóng.

| Phát hiện | Mức | Xử lý (đợt 5) | Bằng chứng |
|---|---|---|---|
| R4-M1 dòng "Blocked: …" không có trần, nằm ngoài bước cắt output: 1.500 lý do → 147.581 ký tự; một lý do 200 KB → 200.327 ký tự vào context | MEDIUM | Trần 5 lý do × 300 ký tự, phần còn lại ghi "and N more blocked commands" | probe của người soát (overlay); `TestBashBlockReasonsAreBounded` |
| R4-A1 bước bị cắt `length` khi chỉ đang nghĩ, đúng bước vượt ngưỡng tự tóm tắt: lượt thành công với câu trả lời rỗng | MEDIUM | Bước cuối `length` mà không có lượt làm tiếp nào được xếp (đang tóm tắt và không có tool call) thì là lỗi lượt rỗng | `TestThinkingCutWhileSummarizingIsAnError` |
| R4-A2 `turnProduced` dùng `ResponseContent.Text()` (chỉ phần chữ đầu): phần đầu toàn khoảng trắng thì câu trả lời thật bị bỏ qua | LOW | Duyệt mọi phần chữ (`fantasy.AsContentType[TextContent]`) | `TestAnswerInALaterTextBlockIsNotEmpty` |
| R4-A3 lỗi bị từ chối lưu `Message="Refused"` + chi tiết lỗi, làm TUI mất lời giải thích của chính nó | LOW | Lưu finish `content_filter` với Message/Details rỗng | `TestContentFilterRefusalIsNotReportedAsEmpty` (thêm assert) |
| R4-A4 agent con lỗi thì chi phí không cộng vào session cha | LOW | Cộng chi phí trước khi xét lỗi (PRB-44) | `TestRunSubAgent` (thêm ca) |
| R4-L1 job chạy nền (`run_in_background` còn chạy, hoặc tự chuyển nền) không mang dòng lý do chặn | LOW | **Giới hạn**: chỉ đường đồng bộ và fast-fail có dòng này; `job_output` không đọc | — |

## Lượt cuối — red team ngữ cảnh mới trên toàn ứng viên (`evidence/wt-final.patch`)

Một người soát (workflow `wf_16ae3314-bc6`), nhận yêu cầu + kiến trúc + mã + test + bằng chứng, không nhận kết luận của người làm. **KHONG-DAT**: 1 HIGH, 3 MEDIUM, 9 LOW. Nguyên văn: `scratchpad/evo/redteam-final.txt`.

| Phát hiện | Mức | Xử lý (vòng 6) |
|---|---|---|
| F1 provider vẫn bị đổi lặng lẽ lúc khởi động/nạp lại (thiếu key, ID model không còn); allowlist không chặn được khi model mặc định của provider thay thế nằm trong danh sách | HIGH | Sửa: không bao giờ đổi provider; provider không dùng được thì dựng agent lỗi nêu provider/model; mất ID model thì model mặc định của cùng provider |
| F2 bằng chứng (suite, faults, qwen-only) cũ hơn mã cuối | MEDIUM | Chạy lại toàn bộ trên ứng viên đã chốt (37) |
| F3 cổng Qwen-only/faults nằm ngoài repo | MEDIUM | `scripts/eval/` vào ứng viên; faults/qwen-only thoát 1 khi hỏng |
| F4 GREEN mà không có bằng chứng việc vẫn thoát 0 | MEDIUM | `NO_CHANGE` |
| L1 chốt tràn context đếm cả output; docs nói thoát 1 cả khi bước là câu trả lời cuối | LOW | Sửa: chỉ xét đầu vào; sửa docs |
| L2 `text` dính bản tóm tắt | LOW | Sửa |
| L3 nhãn provider của `run --json` lấy từ tin đã lưu | LOW | Giới hạn, ghi tài liệu |
| L4 `--small-model` chỉ định rõ ở provider khác cũng bị thay | LOW | Giới hạn, ghi tài liệu |
| L5 tắt `fetch` không tắt `web_fetch` của agentic_fetch | LOW | Sửa |
| L6 lịch chép allowlist từ shell: hai chủ | LOW | Sửa: không chép |
| L7 `routine list/show` thử khoá có thể làm mất một lượt theo lịch | LOW | Giới hạn (cửa sổ vài micro giây), ghi tài liệu |
| L8 làm mới danh sách model ChatGPT/Grok có kết nối ra ngoài kể cả phiên Qwen | LOW | Giới hạn, ghi tài liệu; đọc mã: chỉ khi đã login mà chưa có danh sách |
| L9 skill đồng bộ từ app Claude trỏ tới tool agent-cli không có | LOW | Giới hạn; đề xuất tắt đã có (chính sách skill chờ duyệt) |

Cùng lượt có một người soát README ngữ cảnh mới: README lúc đó **NOT READY** (thiếu cài/lần chạy đầu, "provider chính" chưa định nghĩa, ID nội bộ, giới hạn allowlist chưa nói). Đã sửa, soát lại ở [README_REVIEW](../daily-work/README_REVIEW.md).

## Lượt 5 — soát độc lập phần sửa vòng 6 (`scratchpad/evo/r6-delta.patch`, 32 tệp)

Một người soát chỉ-đọc, tự chạy binary (local, server qua `--host unix://…`, TUI dưới `script(1)`), đột biến và probe trong
`scratchpad/evo/review6/`. **KHONG-DAT**.

| Phát hiện | Mức | Xử lý (vòng 7) | Bằng chứng sau xử lý |
|---|---|---|---|
| `agent-cli login` hỏng ở chính trạng thái fail closed (login dựng cả workspace, `app.New` dừng); cả local lẫn server; thông báo lỗi lại bảo chạy `agent-cli login` | HIGH | `app.New` khởi động không agent khi lỗi là "model đang chọn không dùng được" (`agent.SelectedModelUnavailable`, 4 lỗi); TUI vào màn chọn model; `run`/`loop`/`routine` vẫn dừng | `TestCreateWorkspaceStartsWhenSelectedModelIsUnavailable`, `TestUnavailableSelectedModelOpensOnboarding`, `TestSelectedModelUnavailableCoversOnlyUnusableSelections`; faults probe `login` (binary vòng 6 FAIL, vòng 7 PASS); `login` qua, `run` rc=1 0 request, TUI vào màn chọn model (lead đo ở chế độ local; người soát vòng 7 đo lại cả local lẫn server) |
| Provider dựng sẵn bị tắt (`"disable": true`) mà model vẫn biết thì vẫn bị gọi (8 request); docs vòng 6 lại nói "bị tắt" thì dừng | MEDIUM (có từ trước) | `buildAgentModels` coi `Disable` như không có provider | `TestBuildAgentModelsRefusesADisabledProvider` (đỏ trước sửa) |
| `text` của `run --json` ở chế độ local nối mọi chữ assistant (câu dẫn trước tool call + câu trả lời); docs nói "chỉ câu trả lời cuối, cả local lẫn server" | MEDIUM | Ghi tài liệu: qua server là câu trả lời cuối, local nối mọi chữ (trừ bản tóm tắt) | README, guide, CHANGELOG |
| `faults.sh` xanh cả khi không request nào tới server giả | LOW | Ca `ok` phải thoát 0 và có request main tới qwen | probe của người soát (`fake-broken-bin`) |
| Đường L2 local và đường reload không có test trong repo | LOW | Reload: nhận test của người soát (`TestReloadDoesNotSwitchProviderWhenTheKeyDisappears`, đỏ khi khôi phục fallback cũ). L2 local: chỉ E2E `qwen-only.sh` (giữ) | đột biến của lead |
| Lặt vặt: gợi ý lỗi sai cho ca "mất ID model"; log nói "not configured" cả khi provider bị tắt; hai tên test đọc ngược nghĩa; CHANGELOG nói `workspaces.sh` chạy cùng server; "vài micro giây" chưa đo | LOW | Sửa hết | — |

Người soát cũng xác nhận: 4 test bị đổi kỳ vọng ở vòng 6 là theo yêu cầu mới, không phải "sửa test cho xanh"; 8/9 đột biến của họ
bị bắt; F4 đúng ở 7 tình huống (thư mục không phải git, repo chưa commit, chỉ ghi tệp gitignore, tệp chưa theo dõi, không
đổi gì, routine maintenance, routine goal); L6 đúng bằng binary.

Cùng lúc, lượt soát README ngữ cảnh mới thứ 2 và thứ 3 chấm NOT READY (6 rồi 4 mục phải sửa); đã sửa, xem
[README_REVIEW](../daily-work/README_REVIEW.md).

## Lượt 6 — soát độc lập phần sửa vòng 7 (`scratchpad/evo/r7-delta.patch`, 18 tệp)

Cùng người soát (không viết mã), chạy binary ở cả local lẫn server (`CRUSH_CLIENT_SERVER=1`, socket riêng), TUI dưới tmux
riêng, đột biến trong `scratchpad/evo/review7/`. **KHONG-DAT**. Cả 3 lỗi của vòng 6 đã đóng ở chế độ local; 4/6 đột biến bị
bắt.

| Phát hiện | Mức | Xử lý (vòng 8) | Bằng chứng sau xử lý |
|---|---|---|---|
| Qua server (mặc định của launcher), `run`/`loop`/`send` chỉ in "status code 500": client bỏ nội dung lỗi mà server trả | MEDIUM (mới ở vòng 7: lỗi dời từ tạo workspace sang dựng agent) | `InitiateAgentProcessing` dùng `checkStatus` như các lời gọi khác | `TestInitiateAgentProcessingDecodesErrorBody` (đỏ trước sửa); faults ca `selected-no-key-server`: binary vòng 7 FAIL, vòng 8 PASS |
| `TestBuildAgentModelsRefusesADisabledProvider` qua được cả khi bỏ kiểm `Disable`, tuỳ thứ tự chạy (catalog provider cache theo tiến trình) | LOW | Đặt `Disable` trực tiếp trên provider tự khai sau khi nạp cấu hình | đột biến bỏ kiểm `Disable`: đỏ ở `-run TestBuildAgentModels`, chạy riêng, và cả gói |
| Không test nào chứng minh `app.New` vẫn dừng ở lỗi khác (vd allowlist chặn) | LOW | Không thêm: trong test `execpolicy.Current()` luôn trả policy rỗng, cần móc test trong mã sản phẩm. Hàm phân loại có test; nhánh `app.New` có probe binary của người soát (MODEL_NOT_ALLOWED vẫn dừng khởi động) | — |
| Docs: `run -m` chỉ ghi model ở chế độ server; `dirs` in thư mục, không in tệp; skill của repo còn ở `.crush/skills`, `.cursor/skills`; một gạch "Vì" trong DECISIONS còn nói "login chỉ cần nạp cấu hình" | LOW | Sửa hết | — |

Người soát cũng xác nhận: `login`/`logout` qua được ở local và server; TUI vào màn chọn model ở cả hai, chọn `qwen3-coder`
xong thì prompt kế tiếp chỉ đi qwen; provider bị tắt: 0 request ở cả hai chế độ; mọi chỗ dùng `AgentCoordinator` ngoài gói
agent đều kiểm nil hoặc chỉ chạy sau khi dựng thành công; allowlist chặn vẫn dừng khởi động như trước.

Phát hiện phụ của lead khi đọc báo cáo này: các probe lead ghi là "chế độ server" (`f1-server/`, `r7-srv/`) thật ra chạy local
(binary chạy thẳng mặc định local). Đã sửa nhãn trong các báo cáo.

## Lượt 7 — xác minh phần sửa vòng 8 (`scratchpad/evo/r8-delta.patch`, 6 tệp)

Cùng người soát. **KET-LUAN: DAT.** Cả ba phát hiện mã của vòng 7 đã đóng: qua server, `run`/`loop`/`send --new --wait` thoát 1,
0 request, lỗi nêu `provider "openai", model "gpt-4o"`; `login` qua được; ca `selected-no-key-server` của `faults.sh` chạy qua
server thật (có log server trong thư mục ca, socket biến mất sau khi server rảnh tự thoát). Đột biến c1 (client trở lại so mã
trạng thái trần) và n2 (bỏ kiểm `Disable`) bị bắt, n2 ở cả 3 cách chạy. `checkStatus` không đổi luồng điều khiển nào: không
nơi gọi nào xét `errors.Is` cho lời gọi này, mã thoát vẫn 1. Người soát chấp nhận để nhánh `app.New` chỉ có probe binary.

Còn 3 câu README mức LOW, đã sửa: `routine show` không in đường dẫn tệp định nghĩa (nay ghi
`~/.config/agent-cli/routines/<tên>.json`, in ra lúc `routine new`); `--commit` chỉ cho phép commit, không tự commit; `.env`
được nạp là của thư mục đang đứng khi chạy, không phải của `-c <dir>`. Thêm một ghi chú bằng chứng: `r8-faults-old.txt` dùng
thông báo cổng cũ; chạy lại bằng cổng cuối trên binary vòng 7 cho đúng lý do "the error does not name the unavailable
provider" (`scratchpad/evo/review8/faults-final3.txt`).

Tổng: 8 lượt soát mã độc lập (lượt 1–4, red team cuối, lượt 5–7); lượt cuối DAT. Soát README: 5 lượt, xem
[README_REVIEW](../daily-work/README_REVIEW.md).

