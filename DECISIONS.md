# DECISIONS — sổ quyết định (APPEND-ONLY, không sửa dòng cũ)
Nạp tự động vào mọi phiên Claude. Mục đích: không bàn lại cái đã chốt, không thử lại cái đã hỏng.

## 2026-10-14 — Tool claude_code/codex: xin quyền mỗi lần giao, mức quyền CLI con qua env
**Trạng thái:** đã làm
- **Chốt:** (commit 9186c66b; có test và đã chạy thật với CLI thật) Mỗi lần giao việc cho Claude Code/Codex đều qua `permissions.Request`: đây là ranh giới tin cậy duy nhất vì CLI con chạy không hỏi lại ai. Mức quyền CLI con mặc định giữ toàn quyền (claude `--dangerously-skip-permissions`, codex `sandbox_mode="danger-full-access"`), đổi được bằng env của tiến trình: `AGENT_CLI_CLAUDE_PERMISSION_MODE`, `AGENT_CLI_CODEX_SANDBOX`; `AGENT_CLI_CLAUDE_BIN`, `AGENT_CLI_CODEX_BIN` cho CLI không nằm trong PATH. Chỉ đăng ký tool khi agent được phép dùng (không bị `permissions deny`) và máy có CLI; `*_BIN` đặt mà sai thì dựng agent lỗi (fail-fast, lỗi hiện ngay ở prompt đầu, nêu tên biến). `resume` chỉ nhận dạng ID và đi thành một token (`--resume=<id>`).
- **Vì:** bản trước chạy CLI con toàn quyền mà không hỏi (P0, hai người soát độc lập cùng tìm ra). Mặc định toàn quyền là quyết định sản phẩm đã chốt từ trước và cần để CLI con chạy được test/build. Đo ngày 14/10: trước sửa, tool có mặt trong mọi request kể cả khi máy không có CLI; sau sửa, thiếu CLI thì tool vắng (chạy thật), `*_BIN` sai thì `run` rc 1 kèm tên biến.
- **Đã thử và bỏ:** (a) đặt cấu hình thành `option` của crushrc: bỏ vì phải sửa schema/options của upstream, merge upstream đắt hơn; (b) đọc biến từ `~/.config/agent-cli/env`: không được, tệp đó chỉ được crushrc `source` trong trình thông dịch riêng, biến không tới tiến trình; (c) mặc định an toàn (`acceptEdits`/`workspace-write`): bỏ vì Claude Code không chạy được bash/test và Codex không ghi được Go build cache ngoài workspace, mất phần lớn giá trị giao việc; (d) `*_BIN` sai thì chỉ cảnh báo và bỏ tool: bỏ vì luật "cấu hình hỏng thì chặn", tool biến mất âm thầm khó phát hiện; (e) truyền sandbox bằng `-s` cho cả resume: `codex exec resume` không nhận `-s` (codex-cli 0.159.2 báo `unexpected argument '-s'`), nên dùng `-c sandbox_mode=...`; (f) `--resume <id>` hai token: model đặt `resume:"--dangerously-skip-permissions"` là chèn được cờ, nên dùng một token.
- **Lật lại khi:** cần đổi mức quyền theo từng dự án mà không khởi động lại server (biến env là của tiến trình server, phải chờ server cũ tắt mới nạp lại) thì chuyển sang option của crushrc; hoặc upstream có cơ chế quyền chung cho tool gọi agent ngoài.

## 2026-10-14 — Kiểm thật giao việc ở mức toàn quyền từ phiên Claude Code
**Trạng thái:** bác bỏ — bộ phân loại auto-mode của Claude Code chặn ("Create Unsafe Agents")
- **Chốt:** Phiên Claude chỉ kiểm thật tool giao việc ở mức hạn chế: `dontAsk`/`acceptEdits` cho Claude, `read-only`/`workspace-write` cho Codex; `scripts/test-agent-cli-delegation.sh` mặc định như vậy. Mức toàn quyền mặc định chỉ có unit test (so cờ); người dùng tự chạy thật ở terminal của mình.
- **Vì:** ngày 14/10, lệnh `claude -p ... --dangerously-skip-permissions` qua được, nhưng ngay sau `codex exec -s danger-full-access` thì lệnh kế tiếp bị chặn với lý do "Create Unsafe Agents". Luật là không lách.
- **Đã thử và bỏ:** chạy CLI con toàn quyền trực tiếp bị chặn; chạy gián tiếp qua agent-cli là lách đúng điều bị chặn nên không làm.
- **Lật lại khi:** người dùng cấp quyền rõ ràng (luật permission trong settings) cho phiên Claude chạy agent toàn quyền trong thư mục nháp.

## 2026-10-14 — Ca kiểm "không ghi được tệp" của Claude Code dùng plan mode
**Trạng thái:** bác bỏ — plan mode ghi tệp kế hoạch vào `~/.claude/plans` thật của người dùng
- **Chốt:** Ca kiểm mức quyền hạn chế của Claude dùng `--permission-mode dontAsk`: chặn cả tool Write lẫn lệnh shell dự phòng, không sinh tệp nào ngoài thư mục test.
- **Vì:** mỗi lần chạy ca `plan` để lại `~/.claude/plans/create-a-file-named-*.md` (2 tệp ngày 14/10, đã xoá). `dontAsk` đo được: không tạo tệp, số tệp trong `~/.claude/plans` không đổi.
- **Đã thử và bỏ:** `plan` có tác dụng phụ ngoài thư mục test; `default`/`manual` chưa thử làm ca chặn ghi, không cần vì `dontAsk` đã đủ.
- **Lật lại khi:** Claude Code đổi ngữ nghĩa `dontAsk` (vd bắt đầu tự duyệt). Kiểm lại bằng ca `sandbox` của script.

## 2026-10-15 — Đổi model hai tầng: alt+p chọn provider, alt+m chỉ hiện model của provider đang dùng
**Trạng thái:** đã làm
- **Chốt:** (commit 133eb476, 6e67f1f0) alt+p mở hộp thoại Providers, liệt kê theo `options.visible_providers` (crushrc `option visible-provider`, đúng thứ tự khai; không khai thì lấy các provider đã cấu hình), provider đang dùng đứng đầu, mỗi dòng có trạng thái `current`/`ready`/lệnh đăng nhập/`needs API key`. Chọn một provider thì mở bảng model chỉ gồm model của provider đó. alt+m mở thẳng bảng lọc theo provider của model lớn đang dùng; nếu provider không còn model nào thì mở bảng đầy đủ kèm cảnh báo. alt+e chọn mức effort (chỉ khi model có reasoning levels). alt+. / alt+, đổi session vòng tròn.
- **Vì:** ngày 15/10, khi đang dùng GLM, bảng alt+m cũ hiện model của mọi provider trong catalog (DeepSeek…). User yêu cầu tách provider và model như app Codex. Alt+Tab không dùng được vì Cinnamon chiếm phím (`switch-windows ['<Alt>Tab']`), phím không tới terminal. Đã kiểm trong tmux 15/10: alt+p hiện Z.AI (current), OpenAI (`agent-cli login openai`), Anthropic (`needs API key`); alt+m hiện "Switch Model · Z.AI" chỉ gồm GLM; alt+e hiện Low/High/Max; với hai session, alt+. đi tới rồi quay vòng, alt+, lùi lại.
- **Đã thử và bỏ:**
  - (a) `option default-providers false` để giấu provider khác: bỏ vì như vậy là bỏ luôn catalog, OpenAI/Anthropic không còn model nào để chọn kể cả khi đã đăng nhập.
  - (b) alt+tab để đổi session: hệ điều hành nuốt phím.
  - (c) Bảng lọc theo provider dùng chung bước dọn "model gần đây không tìm thấy": bỏ vì bước này xoá model gần đây của provider khác khỏi cấu hình. Chỉ bảng đầy đủ mới dọn (`TestModelsDialogScopedToProvider`).
- **Lật lại khi:** upstream có bộ chọn provider riêng; hoặc user muốn thấy mọi model trong một bảng thường xuyên hơn đổi provider.

## 2026-10-15 — Mặc định an toàn cho hộp thoại xin quyền và hộp thoại init
**Trạng thái:** đã làm
- **Chốt:** (commit 133eb476)
  - Hộp thoại xin quyền:
    - Mở ra với Deny được chọn sẵn, nên Enter ngay là từ chối.
    - Không chữ cái nào cấp quyền: cho phép một lần là `ctrl+a`, cả session là `ctrl+s` (hoặc chọn nút bằng ←/→ rồi Enter). `d`/Esc là từ chối.
    - Chỉ mũi tên trái/phải chuyển nút; h/l/Tab không chuyển.
  - Hộp thoại init (dự án chưa có AGENTS.md) chọn sẵn "Nope".
- **Vì:**
  - Sự cố: 15/10 08:12, user mở agent-cli trong `~/application`, chưa gõ gì mà init đã chạy. Kết quả: 10 lệnh bash, 0 lần từ chối, ghi `AGENTS.md` 10,6 KB, đọc tệp ngoài dự án.
  - Nguyên nhân: dựng lại cho thấy TUI vẫn hỏi quyền. Phím lạc rơi vào nút mặc định (Enter = Yep/Allow, `s` = allow session).
  - Soát độc lập E (P1): gõ chữ khi hộp thoại đang mở (vd "fix the bug") thì h/l/Tab chuyển lựa chọn sang Allow, rồi Enter là cho phép (`TestPermissions_TypingThenEnterDenies`).
  - Kiểm trên bản cài 15/10: GLM gọi tool write thì hộp thoại tô sẵn Deny; nhấn Enter ra "User denied permission" và tệp không được tạo.
- **Đã thử và bỏ:**
  - (a) Giữ Allow làm mặc định cho nhanh (hành vi upstream): bỏ vì chỉ một lần Enter lạc là lệnh chạy.
  - (b) Đổi mặc định nhưng giữ h/l/Tab: bỏ vì gõ chữ vẫn đưa lựa chọn về Allow.
  - (c) Giữ `a` = cho phép một lần, `S` = cả session (bản đầu của mục này): bỏ vì hộp thoại chỉ nuốt phím trong tối đa 1,5 giây đầu (dialog.go grace), gõ tiếp một câu có chữ `a` ("and add tests") là Allow; câu mở đầu bằng "Sửa..." có `S`.
- **Lật lại khi:** có allowlist/yolo dùng thường xuyên tới mức hộp thoại quyền hiếm khi hiện; hoặc user thấy `a` chậm hơn Enter đáng kể.

## 2026-10-15 — GLM đi qua provider dựng sẵn `zai` của catalog, không qua provider "main" tự khai
**Trạng thái:** đã làm
- **Chốt:** (commit 6e67f1f0) crushrc mẫu đọc `AGENT_CLI_PROVIDER_ID`. Nếu đặt một ID có trong catalog (vd `zai`), crushrc chỉ ghi `api_key`; endpoint, model, tên và mức effort lấy từ catalog nhúng (catwalk v0.52.49: `https://api.z.ai/api/coding/paas/v4`, glm-5.3, glm-5.3-flash…). Nếu bỏ trống, giữ provider "main" tự khai bằng `AGENT_CLI_BASE_URL`, cho model local hoặc endpoint không có trong catalog.
- **Vì:**
  - Provider "main" tự khai không có `can_reason`, nên việc đặt tiêu đề session chỉ được 40 token. GLM suy luận dùng hết số token đó, nên mọi session đều tên "Untitled Session".
  - Không có reasoning levels thì alt+e không có gì để chọn.
  - Qua `zai` (đo 15/10): session có tiêu đề thật ("Kiểm tra GLM qua provider Z.AI"), alt+e có Low/High/Max, `agent-cli run` trả PONG, việc code có tool call chạy đúng.
- **Đã thử và bỏ:**
  - (a) Chỉ thêm `--can-reason` cho "main": tiêu đề hết lỗi, nhưng vẫn không có mức effort và phải tự khai context/max tokens. Cách này giữ lại cho endpoint ngoài catalog (`AGENT_CLI_CAN_REASON`).
  - (b) Đặt `--name` cho provider catalog qua `AGENT_CLI_PROVIDER_NAME`: tên trong catalog thắng nên `--name` bị bỏ qua. Biến này chỉ áp cho "main".
  - (c) Thêm `option default-providers false`: bỏ việc gộp catalog, nên `zai` mất endpoint và danh sách model.
- **Lật lại khi:** catalog nhúng bỏ hoặc đổi `zai`. Catalog đang khoá (`option provider-auto-update false`), nên chỉ đổi khi nâng catwalk. Hoặc z.ai đổi endpoint của gói Coding.

## 2026-10-15 — Mỗi lượt chạy ghim model và bộ tool của nó; bỏ chặn đổi model/effort khi đang chạy
**Trạng thái:** đã làm
- **Chốt:** `SessionAgentCall.pinnedModel` (coordinator gán đúng model mà ProviderOptions được tính cho) và `pinnedTools` (gán lúc xếp hàng). Mỗi bước lấy model qua `stepModel`: chỉ nhận bản dựng lại của CHÍNH provider/model đó (làm mới token OAuth), không thì giữ model của lượt. Bộ tool qua `stepTools`: bản mới nhất nếu model chung chưa đổi (để nhận tool MCP mới), không thì bộ tool của lượt (tool `agent` mang agent con của đúng model). Prompt xếp hàng chỉ được gộp vào lượt đang chạy khi cùng model. TUI bỏ cổng "Agent is busy" khi đổi model/effort và khi mở lại session (khôi phục model chạy nền trong `tea.Cmd`, lần lượt, lần sau thắng).
- **Vì:** fantasy gọi `ModelProvider` ở MỌI bước và nó đọc model chung của workspace; `coordinator.run` còn gọi `updateAgentModels` ở đầu mỗi lượt gửi, nên chỉ cần gửi một prompt ở session B (hay `agent-cli run --model`) là bước kế của run ở session A đổi provider. Test chứng minh: đổi model giữa bước 1 và 2 thì bước 2 vẫn chạy model cũ (đỏ khi bỏ bản sửa).
- **Đã thử và bỏ:**
  - (a) Chặn ở TUI (bản trước): không phủ đường `run`/`send`/backend, và chặn luôn việc đổi model ở session khác khi có run nền.
  - (b) Chặn ở backend (`UpdatePreferredModel` trả bận khi có run): an toàn nhưng làm hỏng đa session (không đổi được model ở B khi A chạy).
  - (c) Model riêng từng session ở coordinator: đúng nhất nhưng phải đổi API model của UI, backend, CLI.
- **Lật lại khi:** cần model khác nhau chạy CÙNG LÚC ở nhiều session theo cấu hình riêng (không chỉ "lượt giữ model lúc gửi") thì làm (c).

## 2026-10-15 — Huỷ theo RunID; lượt của `agent-cli run` gắn với đời tiến trình `run`
**Trạng thái:** đã làm
- **Chốt:** `CancelRun(session, runID)` qua agent → coordinator (cả Stop gate) → backend → `POST .../agent/sessions/{sid}/runs/{rid}/cancel` → client. Lượt đang chạy đúng RunID thì huỷ; còn trong hàng đợi thì gỡ đúng nó (kèm RunComplete cancelled); chưa tới thì ghi dấu 2 phút, tới là huỷ ngay lúc vào. `run` (Ctrl+C, SIGTERM, SIGHUP) huỷ theo RunID cả với `-s/-C`; server cũ (404) thì lùi về cách cũ. `run` gửi kèm `bind_to_client`: client mất claim (thoát, kill -9, mất kết nối quá detach grace) thì server huỷ lượt đó. `send` cố ý không gắn.
- **Vì:** `Cancel(session)` huỷ cả lượt của TUI lẫn hàng đợi người khác, nên trước đây `run -s` bấm Ctrl+C chỉ "thôi chờ" và để prompt chạy sau không ai giám sát; `run` bị kill -9 để lại lượt chạy tiếp tới 30 phút. E2E thật (`test-agent-cli-run-json.sh`): kill -9 → `canceled`; Ctrl+C `run -s` → prompt bị gỡ, lượt `send` vẫn xong.
- **Đã thử và bỏ:** (a) "thôi chờ" cho `-s/-C` (bản trước); (b) huỷ cả session khi Ctrl+C; (c) đánh dấu "không sống khi client chết" theo session thay vì theo lượt: TUI và `send` dùng chung session.
- **Lật lại khi:** coordinator có huỷ theo lượt sẵn từ upstream, hoặc cần huỷ một lượt `send` từ xa (thêm lệnh, không đổi ngữ nghĩa này).

## 2026-10-15 — Trần mồ côi 30 phút chỉ tính khi run kẹt; thêm trần tuyệt đối 6 giờ
**Trạng thái:** đã làm
- **Chốt:** `CRUSH_SERVER_ORPHAN_RUN_TIMEOUT` (30 phút) chỉ đếm khi run mồ côi đang chờ lời xin quyền/câu hỏi (`Permissions.ActiveRequest`, `Questions.Pending`); chạy lại thì đồng hồ tính lại. Run vẫn tiến triển sống tới `CRUSH_SERVER_ORPHAN_MAX_RUN` (mặc định 6 giờ, `0` = không trần).
- **Vì:** đồng hồ cũ tính từ lúc client rời nên việc giao bằng `send` dài hơn 30 phút bị huỷ giữa chừng, chỉ có log Warn.
- **Đã thử và bỏ:** (a) một trần 30 phút từ lúc client rời (bản trước); (b) không trần cho run tiến triển: run lặp vô tận giữ server mãi.
- **Lật lại khi:** có việc giao hợp lệ dài hơn 6 giờ (đặt biến), hoặc có tín hiệu "run lặp" tốt hơn đồng hồ.

## 2026-10-15 — Tool đọc (`grep`, `glob`, `view`, `ls`) hỏi quyền khi chạm ngoài dự án, kể cả qua symlink
**Trạng thái:** đã làm
- **Chốt:** `internal/agent/tools/outside.go`: giải symlink cả hai phía rồi mới so; `grep` (action read) và `glob` (action list) nhận permission service và hỏi như `view`/`ls`.
- **Vì:** `grep`/`glob` không có cổng nào, nên model đọc được `~/.config/agent-cli/env` (chứa key GLM) mà không hỏi; `view`/`ls` so đường dẫn không giải symlink nên `docs -> ~/.ssh` trong một repo lạ qua mặt được. Test đỏ khi bỏ bản sửa.
- **Đã thử và bỏ:** (a) chỉ khuyên lấy key từ kho bí mật (`pass show`): không chặn được tool; (b) allowlist đường dẫn trong crushrc: người dùng phải tự liệt kê, và `permissions allow` bỏ luôn lời hỏi.
- **Lật lại khi:** upstream thêm cổng đường dẫn chung cho mọi tool đọc.

## 2026-10-15 — `agent-cli run` qua server là lượt phi tương tác; KHÔNG tách agent (plan/coder) theo lượt
**Trạng thái:** đã làm
- **Chốt:** `run` gửi `non_interactive`: lượt không có tool `question` và chờ MCP khởi động xong (có trần) như đường local. Chế độ plan của workspace vẫn áp cho `run` (ghi rõ trong `run --help`, server ghi Warn).
- **Vì:** trước đây `run` qua server dùng coordinator tương tác: model gọi `question` là run hỏng (rc 1), lượt đầu sau khi server lên có thể thiếu tool MCP.
- **Đã thử và bỏ:** cho `run` luôn chạy agent coder dù workspace đang plan: `Cancel`/`IsBusy`/hàng đợi của coordinator chỉ hỏi agent chính, nên lượt chạy trên agent khác vô hình với Ctrl+C và với bước teardown.
- **Lật lại khi:** coordinator theo dõi lượt trên mọi agent (khi đó tách agent theo lượt được).

## 2026-10-15 — Câu hỏi chờ tối đa 30 giây sau câu hỏi của session khác
**Trạng thái:** đã làm
- **Chốt:** dịch vụ câu hỏi vẫn hiện một câu một lúc; câu thứ hai chờ tối đa `askQueueTimeout` (30 giây) rồi trả `ErrQuestionBusy` cho model. `Cancel` lấy kênh ra dưới khoá (hai client huỷ cùng lúc không còn panic).
- **Vì:** câu hỏi xếp hàng sau câu hỏi của một session không ai xem thì vô hình (không sự kiện, không WAITING) và chặn run mãi.
- **Đã thử và bỏ:** nhiều câu hỏi chờ song song theo session: phải đổi API trả lời/huỷ (không có ID lô) ở server, client, TUI.
- **Lật lại khi:** cần hai session cùng hỏi người dùng thường xuyên.

## 2026-10-15 — Kiểm tra bản mới của upstream mặc định TẮT trong binary
**Trạng thái:** đã làm
- **Chốt:** chỉ bật khi `CRUSH_DISABLE_UPDATE_CHECK` là giá trị false của `strconv.ParseBool`; không đặt, rỗng, true hay giá trị lạ đều không gọi GitHub.
- **Vì:** chạy binary không qua launcher vẫn gọi GitHub của upstream mỗi lần mở và mời "nâng cấp" lên bản Crush; `0`/`false` trước đây lại bị hiểu là tắt.
- **Đã thử và bỏ:** giữ mặc định bật, chỉ tắt qua launcher (bản trước).
- **Lật lại khi:** fork có kênh phát hành riêng để so phiên bản.

## 2026-10-15 — Provider đang được chọn mà không có endpoint thì dừng lúc khởi động
**Trạng thái:** đã làm
- **Chốt:** `config/load.go`: provider tự khai không có `base_url` (vd gõ sai `AGENT_CLI_PROVIDER_ID`) mà đang là model large/small thì báo lỗi nêu tên provider; provider không ai chọn thì giữ hành vi cũ (cảnh báo rồi bỏ).
- **Vì:** trước đây provider bị bỏ lặng lẽ, agent tự chọn model khác rồi ghi đè lựa chọn xuống đĩa (luật: cấu hình hỏng thì chặn lúc khởi động).
- **Đã thử và bỏ:** chặn mọi provider thiếu endpoint: làm hỏng cấu hình không liên quan và chặn `agent-cli login`.
- **Lật lại khi:** catalog có cách khai provider tạm thời không endpoint mà vẫn hợp lệ.

## 2026-10-15 — Launcher dừng khi không dựng được socket riêng cho dự án
**Trạng thái:** đã làm
- **Chốt:** đường dẫn socket quá dài hoặc thiếu `sha1sum`/`shasum` thì wrapper thoát mã 1 kèm cách sửa (trừ `--host` tường minh, `ps`, Home, `CRUSH_CLIENT_SERVER=0`).
- **Vì:** trước đây wrapper chỉ cảnh báo rồi chạy không `--host`, rơi về server mặc định dùng chung mọi dự án: phá cách ly theo dự án mà người dùng không biết.
- **Đã thử và bỏ:** chọn thư mục ngắn hơn (`/tmp`): bị loại có chủ ý vì ai cũng ghi được và tên socket đoán được.
- **Lật lại khi:** có thư mục runtime riêng ngắn hơn trên mọi máy dùng.

## 2026-10-15 — Port sớm ba bản vá upstream cho P1.4/P1.5
**Trạng thái:** đã làm
- **Chốt:** lấy nguyên văn `fix/dangling-tool-calls` (settle tool call bị bỏ dở khi đọc session, `settle.go`), `fix/shutdown-records-work` (server nhận SIGTERM/SIGHUP thì dừng từng workspace) và ý của `fix/background-job-budget` (`0ac04e4f`: chỉ job đang chạy tính vào trần 50). Fork thêm: session con chỉ settle khi cả chuỗi session cha rảnh (`BusyWithAncestors`), và câu ghi cho model không mời chạy lại mù quáng.
- **Vì:** cổng WIN "Đóng/mở client" và "Context/mạng lỗi" (không hiện trạng thái giả, không lặp mutation); trần job tính cả job đã xong làm mọi lệnh bash hỏng khi sub-agent chạy song song.
- **Đã thử và bỏ:** tự viết bản sửa riêng: lệch upstream, merge sau tốn hơn; settle chỉ theo `IsSessionBusy` như upstream: sub-agent chạy trên agent riêng nên mở session đang chạy sub-agent sẽ ghi "Interrupted" lên tool call còn sống; settle không khoá: hai người đọc song song (TUI và `run`) ghi trùng kết quả (fork thêm khoá theo session).
- **Còn hở:** kiểm "bận" và lúc ghi chưa nguyên tử: một lượt mới bắt đầu và gọi tool trong vài mili-giây giữa hai bước có thể bị ghi "Interrupted" kèm kết quả thật (soát độc lập xếp P3, hiếm).
- **Lật lại khi:** upstream merge các nhánh đó (lấy bản của upstream, giữ hai chỗ fork nếu upstream chưa có).

## 2026-10-15 — Lượt kết thúc thế nào thì prompt xếp hàng phía sau vẫn chạy
**Trạng thái:** đã làm
- **Chốt:** lượt xong, lỗi (kể cả lỗi provider), bị CancelRun (Ctrl+C của `run`, client `run` bị giết), bị huỷ ngay lúc vào Run hay lúc đang tự tóm tắt đều chuyển tiếp cho prompt xếp hàng kế (`handOff`/`nextQueued` dưới khoá session, có bộ đếm prompt "đang chuyển" để giữ thứ tự); lượt lỗi vẫn phát `TypeAgentError` của nó. Huỷ cả session (Esc) vẫn bỏ các prompt mà cancel mark phủ. `ClearQueue` lấy khoá session.
- **Vì:** hai lượt soát độc lập 15/10: prompt phía sau kẹt tới lượt sau, người chờ RunID của nó không nhận RunComplete, session không xoá được; huỷ lúc tóm tắt bị nuốt; Stop hook đánh dấu nhầm prompt chuyển tiếp là Cancelled.
- **Đã thử và bỏ:** chỉ chuyển tiếp khi bị huỷ, giữ hàng đợi khi lỗi provider với lý do "bộ điều phối thử lại lượt lỗi": sai, `coordinator.run` gọi Run một lần (thử lại khi lỗi xác thực nằm trong fantasy).
- **Lật lại khi:** cần dừng cả hàng đợi khi provider hỏng hẳn (khi đó báo lỗi cho từng prompt thay vì chạy chúng).

## 2026-10-15 — Everforest Dark v2 là theme mặc định cho mọi bề mặt
**Trạng thái:** đã làm
- **Chốt:** `styles.DefaultThemeName = everforest-dark-v2` (`internal/ui/styles/themes.go`); tên rỗng, tên sai và mọi provider (kể cả hyper) đều ra theme này; màn ngoài TUI chính (Home, banner thoát, `login`/`logout`, help CLI, dòng in của `dirs`/`session`/`update-providers`) dùng `styles.Color*` lấy từ cùng token. `charmtone-panther` và `gruvbox-dark` giữ nguyên, chọn được trong Themes.
- **Vì:** người dùng chốt theme mới làm mặc định, không phải một tuỳ chọn thêm.
- **Đã thử và bỏ:** chỉ thêm theme như một lựa chọn (bản đầu): người dùng phải tự chọn, Home và banner vẫn màu cũ.
- **Lật lại khi:** người dùng chọn lại theme khác làm mặc định.

## 2026-10-15 — Jobs mở bằng Alt+J, dừng job luôn hỏi lại
**Trạng thái:** đã làm
- **Chốt:** hộp thoại Jobs (Alt+J, mục "Jobs" trong Ctrl+P) đọc `ListJobs`/`JobOutput`/`KillJob` trong `tea.Cmd`, làm tươi 1 giây; `Ctrl+K` hỏi "Stop job?" mặc định No, chỉ ←/→ đổi nút. Message của hộp thoại đi theo ID (`TargetedMsg`) nên vòng làm tươi sống cả khi hộp thoại xin quyền đè lên.
- **Vì:** roadmap P1.4 (DoD: panel có xem output và cancel); phím chữ hay phím lặp không được thành xác nhận dừng job.
- **Đã thử và bỏ:** bấm Ctrl+K lần hai để xác nhận: giữ phím sẽ tự lặp thành xác nhận; chỉ báo "N jobs running" trên sidebar: cần poll liên tục và đổi bố cục.
- **Lật lại khi:** cần chỉ báo job ngoài hộp thoại (thêm vào Activity của sidebar ở Phase 2).

## 2026-10-15 — Ranh giới provider/model: allowlist, giao việc mặc định tắt
**Trạng thái:** đã làm
- **Chốt:** `internal/execpolicy` là ranh giới duy nhất, đọc từ env đã export rồi tệp env của agent-cli (chỉ hai khoá, không qua cấu hình dự án). `AGENT_CLI_ALLOWED_MODELS` (provider:model) giới hạn model chính/nhỏ khi có provider chính trong danh sách; `AGENT_CLI_DELEGATION=on` mới đăng ký `claude_code`/`codex`, chỉ với model cụ thể của chúng trong danh sách, luôn truyền `--model`/`-c model=`. Điểm chặn: `buildAgentModels` (mọi lượt, sub-agent, tiêu đề), `provider.NewTools` + lúc gọi tool (trước khi hỏi quyền), hộp thoại Alt+M, tool bash (chặn gọi thẳng CLI agent).
- **Vì:** người dùng 15/10: session GLM/Qwen tự giao việc sang Claude Code vì CLI có trên máy ("discovered = executable"), và CLI con chạy model mặc định của nó (có thể là Fable). Cài/đăng nhập/có key không phải là cho phép.
- **Đã thử và bỏ:** chỉ đọc env của tiến trình: tệp env chỉ được crushrc `source` trong trình thông dịch riêng nên biến không tới Go; đưa qua crushrc thành `options.allowed_models`/`options.delegation`: cấu hình của dự án gộp vào nên repo lạ tự bật được giao việc; đọc env lúc dựng agent: `.env` (godotenv autoload) và `env` trong `crush.json` của dự án đều `os.Setenv` trước đó (soát độc lập 15/10, P1-1) — nay chụp env tin cậy đầu `main` (`execpolicy.Snapshot`), nạp `.env` sau, server con nhận ảnh chụp; denylist (fable, astra): model mới lọt qua; nhận alias (opus, latest, [1m]): CLI tự đổi sang model mới nhất.
- **Còn hở:** so theo ID provider nên cấu hình dự án vẫn đổi được `base_url` của provider có sẵn (upstream tin cấu hình dự án); chặn lệnh bash theo tên không bắt được bản chép đổi tên.
- **Lật lại khi:** Claude/Codex thành agent chính trực tiếp (Phase 2): khi đó provider của session là claude_code/codex và ranh giới theo provider của session.

## 2026-10-15 — Tự tóm tắt hỏng thì làm tiếp, không dừng run
**Trạng thái:** đã làm
- **Chốt:** request tóm tắt có hạn chờ gấp 3 hạn thường (`summaryRequestTimeoutScale`, mặc định 6 phút vì `request-timeout` mặc định 2 phút). Tóm tắt vẫn hỏng (không phải bị huỷ) thì lượt KHÔNG dừng: lịch sử giữ nguyên (bản tóm tắt hỏng không thành điểm cắt, không gửi lại cho model), còn khoảng 20% context nên lượt làm tiếp được xếp ngay; tự tóm tắt tạm tắt cho session tới prompt mới của người dùng. Huỷ lúc tóm tắt vẫn dừng lượt.
- **Vì:** người dùng 15/10: stream tóm tắt im 2 phút → timeout → lỗi trả thẳng ra, run dừng giữa việc (tái hiện bằng test, đỏ trên mã cũ).
- **Đã thử và bỏ:** thử lại ngay một lần: provider im vì đầu vào lớn thì thử lại cũng im, người dùng chờ gấp đôi; bản tóm tắt cục bộ (cắt lịch sử) khi hỏng: mất ngữ cảnh mà vẫn còn chỗ để làm tiếp.
- **Lật lại khi:** có lỗi "context đầy" từ provider được bắt riêng (roadmap P1.5) để tóm tắt bằng model nhỏ/ngân sách nhỏ thay vì chờ provider báo lỗi.

## 2026-10-15 — Trả lời bị cắt vì hết ngân sách output thì tự làm tiếp (có trần)
**Trạng thái:** đã làm
- **Chốt:** lượt kết thúc với finish `max_tokens` thì tự xếp lượt làm tiếp cùng RunID với lời nhắc "làm tiếp đúng chỗ dừng, chia nhỏ việc"; tối đa `maxLengthContinues` = 3 lần cho mỗi prompt người dùng. crushrc mẫu đặt ngân sách output mặc định của provider `main` = min(32000, context/4) thay cho 8192.
- **Vì:** người dùng 15/10: phiên Qwen hết ngân sách output giữa việc rồi dừng; model có suy luận tiêu phần lớn ngân sách 8192 cho phần nghĩ.
- **Đã thử và bỏ:** làm tiếp không giới hạn: model cứ viết quá dài sẽ lặp mãi.
- **Lật lại khi:** provider cho biết trước trần output thật của model (khi đó đặt ngân sách theo model thay vì theo context).

## 2026-10-15 — Không cho agent tự giết tiến trình agent-cli hay tắt máy
**Trạng thái:** đã làm
- **Chốt:** `internal/shell/protect.go` là BlockFunc chạy trong trình thực thi shell (trước cả YOLO và quyền đã cấp cho session): chặn `kill`/`/bin/kill` nhắm PID thuộc cây tiến trình agent-cli (chính nó + tổ tiên, hoặc tiến trình khác chạy cùng binary qua `/proc/<pid>/exe`), `pkill`/`killall` khớp tên crush/agent-cli, và `shutdown`/`reboot`/`halt`/`poweroff`. Bóc lớp bọc env/nohup/timeout trước khi xét. Ghi log Warn khi chặn.
- **Vì:** sự cố 15/10 ~14:26 (máy runtime, bản 8eff139e): agent coi client (954803) + server (946242) là "mồ côi" rồi `/bin/kill -9` cả hai → mất lượt đang chạy. Cùng UID/cùng tên/bị coi mồ côi không đủ để được giết.
- **Đã thử và bỏ:** chỉ dặn trong prompt hoặc chặn tên lệnh: YOLO/quyền-cấp-session bỏ qua cổng quyền, và đường dẫn tuyệt đối/lớp bọc lách tên; đặt ở lớp thực thi (BlockFunc) mới chắc. Job do chính session tạo vẫn dừng được qua `jobs`/`job_kill`.
- **Còn hở (ghi docs):** chặn theo PID giải qua `/proc` (Linux) và theo tên; `bash -c "kill ..."` lồng sâu, bản chép binary đổi tên, hay systemctl/docker/pm2 stop chưa bắt — tiến trình khác vẫn qua cổng quyền thường. PID bị tái dùng: giải `/proc/<pid>/exe` ngay lúc xét nên đúng tại thời điểm đó.

## 2026-10-15 — Arguments tool call hỏng không được gửi tới provider (HTTP 400)
**Trạng thái:** đã làm
- **Chốt:** thêm cổng `sanitizeOutgoingToolCalls` trong PrepareStep của lượt chính: mọi `ToolCallPart` có `Input` không phải JSON hợp lệ bị thay `{}` ngay trước khi gửi provider (ghi log Warn). Trước đó chỉ có hai cổng không đủ: `sanitizeToolInput` (OnToolCall → chỉ sửa bản lưu DB) và `toolCallInput` (ToAIMessage → chỉ chạy khi đọc lại từ DB). Messages của bước kế trong cùng stream do fantasy dựng từ step result trong bộ nhớ (`prepared.Messages = options.Messages`), không qua hai cổng kia.
- **Vì:** sự cố 15/10 ~14:38 (máy runtime, 8eff139e): model trả tool call `write` arguments JSON bị cắt (input_len=5976); request bước kế mang nguyên chuỗi hỏng → LiteLLM/vLLM trả HTTP 400 "Unterminated string" → lượt dừng, todo 2/8. Đã đọc code xác nhận đường in-memory không được làm sạch (đúng giả thuyết: B là lỗ; A = model/provider sinh JSON hỏng là nguồn, chưa rõ cắt do token limit hay ghép stream).
- **Đã thử và bỏ:** đoán phần JSON thiếu rồi chạy tool: cấm (chạy nội dung suy đoán). Sau sửa: request gửi `{}` hợp lệ → provider không 400 → model nhận tool result lỗi "arguments were not valid JSON" → tạo tool call mới (hữu hạn qua trần max_tokens + loop detection sẵn có).
- **Còn chưa xác minh:** nguồn JSON hỏng (A) và liệu có luôn do token limit; cần payload runtime đã che secret để chốt. Cổng này xử lý phần B (agent-cli không gửi rác đi) bất kể A.

## 2026-10-15 — Không tự vá (jsonrepair) arguments của tool tác động
**Trạng thái:** đã làm
- **Chốt:** cấu hình `fantasy.WithRepairToolCall(repairToolCall)` cho agent chính. Tool tác động (write, edit, multiedit, bash, download, job_kill) có arguments JSON hỏng thì KHÔNG vá — trả nil để fantasy đánh Invalid, model nhận lỗi và gọi lại đúng. Tool chỉ-đọc vẫn được `jsonrepair` như mặc định (tiện khi thừa dấu phẩy), nhưng chỉ chạy nếu khớp schema (fantasy validate lại sau vá); JSON đã hợp lệ thì không đụng.
- **Vì:** mặc định fantasy (không cấu hình RepairToolCall) tự `jsonrepair.RepairJSON`. Arguments bị cắt như `{"file_path":"a","content":"nửa chừng` vá được thành JSON hợp lệ nhưng nội dung CỤT → `write` ghi tệp thiếu nội dung mà không báo (mất dữ liệu âm thầm). Phát hiện khi đọc fantasy v0.45.1 `validateAndRepairToolCall` (agent.go:1176).
- **Đã thử và bỏ:** tắt repair cho MỌI tool: làm tool đọc kém bền khi model lỡ sai JSON nhẹ; giữ repair cho tool đọc là đủ an toàn vì chúng không ghi/chạy.
- **Lật lại khi:** cần một tool tác động cụ thể tự vá (khi đó thêm RepairToolCall riêng có kiểm nội dung, không chỉ vá cú pháp).


## 2026-10-16 — Tên cá nhân cho theme Everforest
**Trạng thái:** đã triển khai trong mã nguồn, chờ bản cài được cập nhật.
- **Chốt:** tên chính `luannt-theme-everforest-dark`; `everforest-dark-v2` là alias tương thích cấu hình cũ. Giữ nguyên bảng màu và các theme khác.
- **Vì:** người dùng muốn nhận biết ngay theme do mình đặt.
- **Lật lại khi:** người dùng yêu cầu đổi tên hoặc màu.

## 2026-10-16 — bash tự duyệt: đối số ngoài workspace phải hỏi quyền
**Trạng thái:** đã làm (v0.2.0)
- **Chốt:** Lệnh chỉ-đọc trong danh sách an toàn vẫn hỏi quyền khi đối số trỏ ra ngoài thư mục làm việc: đường dẫn tuyệt đối, `~`, `..`, `$VAR`, giá trị của `--cờ=`, `git diff --no-index`, hoặc `working_dir` ngoài workspace. Bỏ `printenv` và `git config --list` khỏi danh sách an toàn. Kiểm sau khi bỏ nháy và `\`, chặn brace expansion (`internal/agent/tools/safe.go`, `commandLeavesWorkspace`).
- **Vì:** audit 16/10 (X-01) chạy được `isSafeCommand("git diff --no-index /dev/null ~/.config/agent-cli/env") == true`: nội dung tệp env vào ngữ cảnh model mà không hỏi, trái với chính crushrc mẫu ("chỉ hỏi khi đọc NGOÀI dự án"). `printenv VAR` và `echo $VAR` đo được là lộ biến môi trường không hỏi (`set` thì không lộ nên giữ).
- **Đã thử và bỏ:** chỉ kiểm token có bắt đầu bằng `/`, `~`, `..`: bị phá ngay 5/12 dạng (`ls \/etc`, `ls {/etc,/tmp}`, `ls .''./etc`...) vì shell ghép chuỗi trước khi chạy. Cấm hẳn mọi dấu nháy: bỏ vì `echo "a b"` là dùng hợp lệ; đổi lại bỏ nháy rồi mới kiểm, chấp nhận hỏi thừa.
- **Lật lại khi:** hỏi thừa gây phiền thật (đo được) thì nới có bằng chứng; symlink trong dự án trỏ ra ngoài còn lọt (ghi `shortcut:` ở `safe.go`), xử lý bằng `outsideWorkingDir` nếu cần.

## 2026-10-16 — Stop hook vẫn chặn sau 3 lần chạy lại: lượt báo lỗi
**Trạng thái:** đã làm (v0.2.0)
- **Chốt:** Chạm trần chạy lại (3) mà hook vẫn nói "block" thì `RunComplete` cuối của lượt mang lỗi `Stop hook still blocking after 3 re-runs: <lý do>` (`run --json` rc=1). Timeout, crash, exit khác 2/49 và halt giữ nguyên fail-open.
- **Vì:** audit C-01: trước đây lượt kết thúc như thành công, chỉ có `slog.Warn`, trong khi cổng verify đang đỏ.
- **Đã thử và bỏ:** làm cả timeout/crash thành lỗi: bỏ vì là giao thức Claude Code (hook không quyết định được thì không cản), README đã ghi, và lỗi hạ tầng của hook không nên đồng nghĩa với "việc hỏng".
- **Lật lại khi:** cần cổng fail-closed hẳn (thêm tuỳ chọn `fail_closed` cho hook Stop).

## 2026-10-16 — Mã thoát bash đi vào metadata, không đặt IsError
**Trạng thái:** đã làm (v0.2.0)
- **Chốt:** `BashResponseMetadata.ExitCode` (`*int`, omitempty); UI hiện icon lỗi và `(exit=N)` khi ≠ 0. Không đặt `ToolResponse.IsError`.
- **Vì:** audit X-02: `go test` thoát 1 vẫn hiện ✓ xanh. `IsError` đổi nội dung gửi model (và cassette VCR `TestCoderAgent`); metadata chỉ phục vụ UI.
- **Đã thử và bỏ:** parse chữ "Exit code N" trong UI: mong manh, không có ở job nền.
- **Lật lại khi:** muốn model thấy lệnh lỗi là lỗi công cụ (quyết định sản phẩm, phải thu lại cassette).

## 2026-10-16 — Phiên bản: SemVer + CHANGELOG + tag, mốc cơ sở v0.1.0
**Trạng thái:** đã làm
- **Chốt:** `CHANGELOG.md` theo Keep a Changelog; tag có chú thích `vX.Y.Z` trên `main`. `v0.1.0` gắn vào `87233bc8` (trạng thái trước audit, nối tiếp dòng `0.1.0-lat1/lat1.1/lat2` trong `~/.local/lib/agent-cli/VERSIONS.log`); `v0.2.0` là đợt sửa audit. Quy trình ở CLAUDE.md P6.
- **Vì:** binary trước đây báo `devel` hoặc pseudo-version, không có chỗ tra "bản nào có gì". Go 1.27 gắn version từ tag nên không cần `-ldflags`.
- **Đã thử và bỏ:** `-ldflags -X version.Version=…` trong installer: thêm một nguồn sự thật thứ hai cần giữ khớp với tag.
- **Lật lại khi:** build từ thư mục không có `.git` (VCS stamping không hoạt động): khi đó mới cần `-ldflags`.

## 2026-10-16 — bash tự duyệt: bổ sung sau lượt soát độc lập
**Trạng thái:** đã làm (v0.2.0); bổ sung cho mục "bash tự duyệt: đối số ngoài workspace phải hỏi quyền" cùng ngày
- **Chốt:** Bỏ thêm `ps` và `git config --get` khỏi danh sách an toàn; mọi cờ chứa `/` hoặc `~` (kể cả dính liền cờ ngắn như `-f/etc/x`) bị coi là đường dẫn. Câu "không bao giờ bỏ sót" ở commit `7cee200b` và comment cũ trong `safe.go` là sai và đã bỏ: đây là danh sách đen theo cú pháp.
- **Vì:** lượt soát độc lập (agent riêng, không có Edit) chạy thật ba đường còn lọt: `date -f/etc/hostname` (nội dung tệp hiện qua stderr), `git config --get-regexp .` (tương đương `--list`, thứ vừa bị gỡ), `ps eww` (lộ env như `printenv`). Mỗi đường đã thành test đỏ rồi xanh.
- **Đã thử và bỏ:** lọc riêng `ps` có cờ `e`: cú pháp BSD (`ps eww`, `ps auxe`, `ps e`) quá nhiều dạng, bỏ hẳn `ps` đơn giản hơn. Giữ `git config --get <key>` đúng một khoá: `--get-regexp .` lấy được tất cả nên không phân biệt được.
- **Lật lại khi:** có danh sách trắng cờ cho từng lệnh an toàn (cách sửa tận gốc), khi đó có thể cho lại `ps`/`git config --get` với cờ cụ thể.

## 2026-10-16 — Self-kill guard: phân tích cú pháp script `-c`, không tách chuỗi
**Trạng thái:** đã làm (v0.2.0)
- **Chốt:** `internal/shell/protect.go` parse script sau `sh/bash/... -c` và `eval` bằng `syntax.NewParser` rồi duyệt mọi `CallExpr`; từ chỉ gồm chữ và nháy được giải thành chuỗi, từ có expansion là "động". Tên lệnh động cạnh chữ kill/shutdown thì chặn.
- **Vì:** lượt soát độc lập chạy mô phỏng và phá bản đầu (tách bằng regex `[;&|\n(){}]`) ở `for ...; do kill $p; done`, `if ...; then kill`, `bash -c -- 'kill N'`, `exec/eval/builtin kill`, `x=1 kill N`, `ki""ll`, backtick. Regex không hiểu từ khoá, phép gán, nháy.
- **Đã thử và bỏ:** vá regex từng dạng (đuổi theo cú pháp bash mãi); gọi `execpolicy.commandString` (logic gần giống nhưng phục vụ mục đích khác, gộp là một việc riêng).
- **Lật lại khi:** cần phủ script từ tệp/stdin (cần sandbox chứ không phải BlockFunc) hoặc hợp nhất với `execpolicy`.

## 2026-10-16 — Loop: driver phía CLI, verify do driver chạy, mỗi lượt một session mới
**Trạng thái:** đã làm (lát 1: `agent-cli loop`; chưa lên lịch, chưa UI)
- **Chốt:** `internal/loop` là engine thuần logic (agent và verify được tiêm vào); `agent-cli loop` là driver phía CLI gọi `runNonInteractive`/`RunNonInteractive` mỗi lượt trong session MỚI, tự chạy lệnh verify bằng `shell.Run` trong thư mục workspace, và tin mã thoát của chính lệnh đó. Trạng thái chạy nằm trong journal JSONL ngoài worktree, không có bảng DB mới.
- **Vì:** audit trục runtime: server không phải daemon, trần mồ côi 6h và tự thoát sau 60s nên engine trong server chết giữa chừng; không có bảng run và `sqlc` chưa cài; "lượt xong" không phải bằng chứng (loop-detection dừng với rc=0, Stop hook fail-open, MCP không timeout). Mỗi lượt session mới (prompt tự chứa đủ) tránh auto-summarize lặp lại và ngữ cảnh nhiễm sau lần sửa hỏng.
- **Đã thử và bỏ:** (a) engine trong server (cần proto/endpoint/bảng DB/khôi phục khi khởi động, và chịu trần 6h); (b) khái quát hoá Stop hook (trạng thái chỉ ở RAM, không có pause, áp lên cả lượt TUI); (c) trần token/cost (cost đo ra 0 nên là cổng giả: không bao giờ chặn ai; chốt thật là thời gian và số lần sửa); (d) tiếp tục cùng một session qua các lượt sửa.
- **Lật lại khi:** cần Routine UI/Agent Map dùng chung runId (khi đó cần tệp trạng thái bền, xem audit R-02) hoặc chạy theo lịch không người trực (cần R-01 trước).

## 2026-10-16 — Routine: định nghĩa ngoài repo, chạy in-process trong worktree riêng, lịch chỉ in
**Trạng thái:** đã làm (lát 1-3 runtime của `agent-cli routine`; chưa có UI)
- **Chốt:** (1) định nghĩa nằm ở thư mục cấu hình của người dùng, mode tường minh `maintenance|goal`, lịch CHỈ cho `maintenance`, `verify` bắt buộc; (2) `routine run` luôn chạy agent IN-PROCESS (không qua server dùng chung) trong worktree riêng `routine/*` ngoài repo, với chế độ không người trực bật trong chính tiến trình; (3) `routine schedule --print` chỉ IN unit systemd/crontab, người dùng tự cài; (4) khoá flock, thoát 75 khi bận; (5) không commit trừ khi `"commit": true`, không bao giờ push/merge, worktree và nhánh được giữ lại.
- **Vì:** audit R-01 (`run` tự duyệt mọi quyền, lớp chặn không có `git push`): chạy theo lịch thành toàn quyền không ai nhìn. Lớp chặn là trạng thái của tiến trình nên nếu agent chạy qua server dùng chung (first-wins) thì cờ có thể bị bỏ qua lặng lẽ: chạy in-process loại trừ khả năng đó. Từ vựng của `learn_claudecode` (L14): routine chỉ nhận việc cơ học + kiểm được + vô tận, việc làm tính năng chạy tay. `agent` bị cấm `crontab`/`systemctl`/`at` ở bannedCommands nên người dùng mới là người cài lịch. Repo lạ không được thêm việc chạy theo lịch (B-02): nên định nghĩa không nằm trong repo, và lịch ghim sha256.
- **Đã thử và bỏ:** (a) tự cài timer/crontab (agent-cli không nên sửa lịch của người dùng; Linger và env của cron chưa kiểm được); (b) engine trong server (xem mục Loop); (c) allowlist lệnh thay denylist (không khả thi cho việc lập trình thật, tài liệu của chủ dự án cũng nói allowlist là hướng đúng nhưng cần danh sách cờ từng lệnh); (d) trần token/cost (cost đo ra 0 là cổng giả); (e) UI Alt+R/Agent Map làm trước (cần tệp trạng thái bền; Agent Map chưa tồn tại).
- **Lật lại khi:** cần Routine View/Agent Map dùng chung runId (dựng UI trên tệp báo cáo/journal), chạy nhiều worker ghi song song (cần worktree + server cho từng worker và trần RAM), hoặc có danh sách trắng cờ cho từng lệnh để thay denylist.

## 2026-10-16 — Chế độ không người trực: git theo danh sách trắng, công tắc tới tiến trình con (sau lượt soát độc lập)
**Trạng thái:** đã làm (bổ sung cho mục "Routine" cùng ngày; chưa phát hành)
- **Chốt:** Trong chế độ không người trực `git` chỉ chạy lệnh con nằm trong danh sách trắng (lệnh con lạ, rỗng hoặc động bị chặn; `-c`/`--config-env` cấm; `git config` chỉ đọc); `routine run` xuất `AGENT_CLI_UNATTENDED` và `AGENT_CLI_UNATTENDED_COMMIT` cho tiến trình con rồi trả lại; `loop`/`routine` thêm vào `selfAgentSubcommands`; tool `claude_code`/`codex` không đăng ký; tool bash từ chối `working_dir` ngoài workspace; mọi `rm` phải ở trong workspace; gitFingerprint tắt `core.fsmonitor`, `diff.external`, textconv; `routine run` từ chối `--cwd`; lỗi dựng agent thì dọn worktree/nhánh và ghi báo cáo ERROR.
- **Vì:** lượt soát độc lập chạy `git` thật và chứng minh các đường vòng của danh sách đen: alias (`git -c alias.p=push p`, `git config alias.p push`), cờ toàn cục lạ (`--attr-source`), biến động trong `bash -c`, launcher chưa bóc (`timeout -s`, `taskset -c 0,2`, `flock`, `systemd-run --user`), `xargs git`; công tắc chỉ là biến atomic nên `crush loop` từ bash không được bảo vệ; tool giao việc chạy CLI con ở mức quyền rất rộng. Cả bản trước tự khẳng định "không thể bị bỏ qua" là sai.
- **Đã thử và bỏ:** vá danh sách đen thêm từng tên lệnh con/cờ (đuổi theo git mãi, alias vòng được); quét mọi hậu tố argv tìm `git`/`rm` (chặn oan `grep git file`, `man git`); coi `--scheduled` là rào bảo vệ (chỉ là cờ do người gọi truyền).
- **Lật lại khi:** có sandbox thật (user namespace, đĩa chỉ-đọc ngoài worktree, mạng bị cắt) thì lớp này chỉ còn là lớp phụ; hoặc khi cần thêm lệnh git vào danh sách trắng cho một routine cụ thể (thêm theo bằng chứng, kèm test hai chiều).

## 2026-10-16 — Mọi lời gọi ẩn ở lại provider của model chính
**Trạng thái:** đã làm (chưa phát hành)
- **Chốt:** `buildAgentModels` (`internal/agent/coordinator.go`) là điểm chặn duy nhất: model nhỏ khác provider với model lớn thì dùng chính model lớn (`sameProviderSmall`), log cảnh báo một lần mỗi cặp provider mỗi tiến trình; cùng provider thì giữ model nhỏ. Allowlist kiểm sau bước này. Tiêu đề, `agentic_fetch`, tóm tắt, agent con vì vậy đều đi provider của model chính. Kèm theo: session con không gọi LLM đặt tiêu đề; tiêu đề bị cắt (`length`) thì không thử lại; chỉ thử model lớn khi nó khác model nhỏ. Test: `TestSameProviderSmall`, `TestBuildAgentModelsKeepsSmallOnTheMainProvider`, `TestNoTitleCallForChildSessions`, `TestTitleStopsAfterTokenLimit`.
- **Vì:** probe P-01 (provider giả đếm request): phiên chọn qwen gửi lời gọi tiêu đề kèm nguyên văn prompt đầu sang provider của model nhỏ (glm), thêm một lần cho mỗi agent con, không cảnh báo. Bộ tiêm lỗi `faults.sh` (`scripts/eval/` trên cây chính, chưa vào git): lọt ở 8/8 ca với bản gốc, 0/8 sau sửa. Tiêu đề baseline: 26/97 lời gọi, 26/26 kết thúc `length`, không lần nào ra tiêu đề. Phiên "chỉ Qwen" mà gọi provider khác là lỗi đúng/sai (riêng tư, chi phí), không phải tối ưu.
- **Đã thử và bỏ:** chưa thử phương án khác.
- **Lật lại khi:** cần model nhỏ ở provider khác một cách có chủ đích; khi đó phải là lựa chọn tường minh của người dùng, thấy được trong UI và sổ ghi, không phải mặc định.

## 2026-10-16 — Sổ ghi lời gọi LLM: opt-in, chỉ metadata
**Trạng thái:** đã làm (chưa phát hành)
- **Chốt:** `AGENT_CLI_LLM_LEDGER=<tệp>` là biến tin cậy (chụp từ env gốc đầu main; `.env` và cấu hình dự án không bật hay chuyển được). Sổ bọc model ở lớp ngoài cùng nên mỗi lần thử của fantasy là một dòng JSON: vai trò, session, provider/model thật, số tin, kích thước và sha (6 byte) system prompt, số và tên tool, token của chính lời gọi, finish, lỗi (chỉ loại lỗi và mã HTTP). Tệp mode 600, mở một lần mỗi tiến trình. Không bật thì model không bị bọc. Test: `TestLedgerRecordsEachCallWithRoleProviderAndUsage`, `TestLedgerRecordsTheErrorKindNotItsText`.
- **Vì:** bộ đếm token của session chỉ giữ bước cuối, nên `run --json`/`stats` không phải lượng tiêu thụ; cần bằng chứng độc lập cho "phiên chọn provider X không gọi provider khác". Probe P-06: sổ khớp 5/5 lời gọi với server giả về thứ tự, vai trò, provider, model. Bài học từ Hindsight (H-36: trace lưu nguyên văn prompt): provenance chỉ phục vụ bằng chứng, không lưu nội dung.
- **Đã thử và bỏ:** bản đầu chép tới 300 ký tự văn bản lỗi của provider: bỏ vì server local hay chép mẩu prompt vào thông báo lỗi (lượt soát độc lập, SEC-6).
- **Lật lại khi:** bộ đếm token của session đếm đúng tổng (sổ chỉ còn để soát provider), hoặc cần ghi thêm trường có thể chứa nội dung (phải là quyết định riêng).

## 2026-10-16 — Trần 10 lần tự tóm tắt cho một prompt
**Trạng thái:** đã làm (chưa phát hành)
- **Chốt:** trong chuỗi làm tiếp của một prompt, tự tóm tắt tối đa `maxAutoSummaries` = 10 lần khi việc còn dở (tin assistant có tool call, kể cả viết dở); cần thêm thì lượt dừng với lỗi `stopped: auto-summarized 10 times for one prompt without finishing; send a new prompt to keep going`. Riêng trường hợp tóm tắt xong mà riêng phần đầu vào (prompt) của bước đầu ở lượt làm tiếp đã vượt ngưỡng và bước đó còn gọi tool: dừng ngay với lỗi `context window too small…`. Chỉ xét đầu vào vì output lớn (ghi tệp lớn) thì tóm tắt tiếp vẫn giải phóng được chỗ (red team cuối, L1). Test: `TestAutoSummarizeHasACapPerPrompt`, `TestAutoSummarizeStopsWhenSummaryCannotFreeRoom`, `TestLargeOutputRightAfterSummaryIsNotAnOverflow`.
- **Vì:** probe P-04: phần cố định của prompt chiếm hơn 80% window làm tóm tắt lặp vô hạn (755 lời gọi main + 755 summary trong 90 s). `scripts/eval/qwen-only.sh` (cây chính, chưa vào git) với Qwen thật: model làm lại việc sau mỗi lần tóm tắt, 1.338 lời gọi main trong 120 s, ở cả binary gốc lẫn bản đã có sửa P-04. Test trước sửa: 1.000 lần tóm tắt trong 15 s, không lỗi.
- **Đã thử và bỏ:** chỉ dừng khi tóm tắt không giải phóng được chỗ: không đủ, vì tóm tắt vẫn giải phóng được chỗ mà model không tiến lên. Bản trần đầu bị vòng qua khi bước đầy context bị cắt giữa tool call (lượt soát 2, LC2-3): trần nay dùng cùng điều kiện với chỗ xếp lượt làm tiếp.
- **Lật lại khi:** việc thật cần quá 10 lần tóm tắt cho một prompt và gửi prompt mới để tiếp là phiền (khi đó cho cấu hình con số, không bỏ trần).

## 2026-10-16 — Lượt trả lời rỗng là lỗi
**Trạng thái:** đã làm (chưa phát hành)
- **Chốt:** xét cả chuỗi của một prompt người dùng (prompt và các lượt tự làm tiếp sau tóm tắt hay sau khi bị cắt): chuỗi kết thúc mà model chưa ra chữ hay tool call nào thì là lỗi `the model returned an empty reply (no text and no tool call)` (finish "Empty reply"; `run` thoát 1). Bước cuối `length`: nhánh tự làm tiếp chạy trước, hết 3 lần mà vẫn chưa ra gì mới là lỗi. Bộ lọc nội dung chặn trước khi có gì: lỗi riêng `the provider's content filter stopped the reply before any text or tool call`, finish vẫn là content_filter (TUI hiện Refused). Chuỗi đã ra chữ hay gọi tool thì lượt im lặng sau đó không phải lỗi. Cùng đợt: lượt dừng vì loop detection cũng là lỗi. Test: `TestEmptyTurnIsAnError`, `TestToolWorkThenSilentEndIsNotAnError`, `TestLengthCutWithoutTextIsContinuedNotEmpty`, `TestLengthCutChainThatNeverProducesIsAnError`, `TestContentFilterRefusalIsNotReportedAsEmpty`, `TestSilentContinuationAfterSummaryIsNotAnError`, `TestLoopDetectionStopIsAnError`.
- **Vì:** `faults.sh` ca `empty-reply`: rc=0 với `text` rỗng ở cả bản gốc lẫn bản sửa đầu, tức `run` báo thành công cho một lượt không làm gì. Loop detection dừng lượt mà rc=0 (C-06).
- **Đã thử và bỏ:** bản đầu xét từng lượt riêng lẻ, coi mọi lượt không chữ, không tool call là rỗng: chặn nhánh tự làm tiếp khi bị cắt giữa tool call (fantasy không phát ToolCall khi finish=length), đổi lý do của bộ lọc nội dung thành "Empty reply" (lượt soát 2, LC2-1 và LC2-2), và coi lượt làm tiếp im lặng là lỗi (LC2-4). Bản sửa kế tiếp chỉ xét lượt có bước cuối `stop` và bỏ qua mọi lượt làm tiếp; đã thay bằng cách xét cả chuỗi như trên.
- **Lật lại khi:** có provider thật trả lời rỗng một cách hợp lệ.

## 2026-10-16 — `agentic_fetch`: session con không tự duyệt trong TUI; `run` duyệt session con của chính lượt
**Trạng thái:** đã làm (chưa phát hành)
- **Chốt:** session con của `agentic_fetch` không còn được tự duyệt. Việc thường của nó (web_fetch, web_search, view/grep trong thư mục tạm) không cần quyền; đọc ra ngoài thì TUI hỏi người dùng. `run` local (`app.autoApproveRun`) và `run` qua server (`rootOf` ở `internal/cmd/run.go`) duyệt mọi yêu cầu của session có gốc là session của lượt; không tra được gốc thì không duyệt (local từ chối yêu cầu, qua server thì `run` dừng với lỗi). `disabled_tools` áp cho tool con; tắt cả web_fetch lẫn web_search thì không đăng ký `agentic_fetch`; tắt web_fetch thì chế độ URL bị từ chối trước khi tải. Lớp từ chối ghi ngoài thư mục làm việc khi không người trực chạy trước bước duyệt. Test: `TestAgenticFetchChildSessionNotAutoApproved`, `TestRunNonInteractiveGrantsDescendantSessions`.
- **Vì:** đọc mã (PRB-09): `agentic_fetch` bỏ qua `disabled_tools` và session con tự duyệt, nên nội dung web có thể bảo agent đọc tệp ngoài thư mục tạm rồi gửi đi bằng web_fetch. Bỏ tự duyệt thì `run` local treo tới `--run-timeout` vì không ai trả lời quyền của session con (lượt soát 1, F4). Lượt soát 1 (SEC-4): tắt web_fetch mà chế độ URL vẫn tự tải URL.
- **Đã thử và bỏ:** chỉ bỏ tự duyệt mà không cho `run` duyệt session con (hồi quy treo, F4); giữ tự duyệt session con (lỗ ở trên).
- **Lật lại khi:** TUI hỏi quá nhiều cho việc fetch thường ngày, hoặc có cách hỏi quyền cho lượt không có người trả lời.

## 2026-10-16 — Không người trực: chặn theo chữ mọi lệnh nhắc tới biến git đổi repo
**Trạng thái:** đã làm (chưa phát hành)
- **Chốt:** khi không người trực, mọi lệnh mà chữ của nó khớp `GIT_DIR`, `GIT_WORK_TREE`, `GIT_COMMON_DIR`, `GIT_INDEX_FILE`, `GIT_OBJECT_DIRECTORY`, `GIT_NAMESPACE`, `GIT_EXEC_PATH`, `GIT_CONFIG_PARAMETERS|COUNT|GLOBAL|SYSTEM|KEY_*|VALUE_*` đều bị chặn, bất kể lệnh gì và giá trị gì. "Chữ" gồm argv, script `sh -c`, chuỗi `env -S`, `eval`, và tiền tố `env NAME=…` mà trình thực thi thêm cho biến đã export (xét cả tên lẫn giá trị). Biến kế thừa nguyên giá trị từ env của chính agent-cli được bỏ qua. Fail-closed: chấp nhận báo nhầm. Cùng đợt: `git -C`/`--git-dir`/`--work-tree` ra ngoài workspace bị chặn; option dài viết tắt so theo tiền tố như git. Test: `TestUnattendedGitDeniesRedirectingEnvEndToEnd` và các test `unattended_*_test.go`.
- **Vì:** lượt soát 1 (SEC-1) đổi được repo đích bằng `GIT_DIR=… git rm`, `env GIT_DIR=…`, trong `sh -c`, và `GIT_CONFIG_COUNT` (tương đương `-c`). Bản sửa đầu phân tích phép gán; lượt soát 2 vòng qua bằng `export 'GIT_DIR=…'` có nháy, `declare -x`, gán trước khi gọi hàm, tên động trong `sh -c`, `env -S`, `find -exec`, `eval`: 7 dạng đều xoá được tệp ngoài workspace.
- **Đã thử và bỏ:** phân tích phép gán (bản đợt 2, `protect.go` đã trả về nguyên bản): cú pháp gán của shell có quá nhiều dạng. Báo nhầm được chấp nhận: biến shell thường tên `GIT_DIR` trong script, chỉ nhắc tên trong một chuỗi, lệnh không phải git mang biến git (`env GIT_DIR=/o ls`, `sh -c 'GIT_DIR=x make'` nay bị chặn). Còn hở, ghi trong comment: tên ghép lúc chạy (`"GIT_""DIR"`), script đọc từ tệp hoặc stdin.
- **Lật lại khi:** có sandbox thật (như mục chế độ không người trực ở trên), hoặc báo nhầm chặn việc thường ngày của một routine thật (khi đó thu hẹp theo bằng chứng, kèm test hai chiều).

## 2026-10-16 — Không có hệ memory riêng: context file là memory (M0)
**Trạng thái:** chốt (không làm mã)
- **Chốt:** không thêm kho memory, không trích fact tự động, không reflect hay consolidation. Tri thức giữa các phiên là context file (`AGENTS.md`, `CRUSH.md`, `CLAUDE.md`… của dự án; `~/.config/agent-cli/CRUSH.md` và `~/.config/AGENTS.md` toàn cục), lịch sử session (`-s`/`-C`) và tự tóm tắt. Bài học do người hoặc agent ghi vào context file, người duyệt qua git. Thứ tự tin: nguồn/runtime hiện tại > bằng chứng vừa kiểm > lịch sử > suy luận của model; memory không đè sự thật hiện tại của repo.
- **Vì:** không lớp vấn đề nào của agent-cli đòi memory dài hạn có bằng chứng. M0 đạt hai kịch bản với LLM thật, mỗi kịch bản chạy một lần: tiếp cùng session (s10) và đổi provider giữa phiên (s13). Lỗ đo được nằm ở kênh nạp context file (chỉ đọc ở cwd, tên chữ thường không nạp, agent con không nhận: PRB-19/20), sửa rẻ hơn mọi nấc memory cao hơn. Hệ kiểu Hindsight có kiểu hỏng đã ghi nhận: 0 fact mà báo `completed` (H-08/H-09), consolidation đóng dấu sai hoặc xoá sạch bank (H-17..H-24), trace lưu nguyên văn prompt (H-36).
- **Đã thử và bỏ:** không thử bằng mã. Đánh giá từng khái niệm: REJECT Memory Bank, retain tự động, reflect, consolidation, mental model tự làm mới, hook memory; DEFER recall nhiều chiến lược và hiệu lực thời gian (`reports/system-evolution/28_HINDSIGHT_DECISIONS.md`, chưa vào git).
- **Lật lại khi:** kịch bản Daily B (tiếp việc hôm qua), I (bài học lịch sử) hoặc J (memory sai) thất bại ở M0 vì thiếu tri thức giữa phiên. Bước kế khi đó là M1: một quy ước tệp bài học có cấu trúc (một bài một mục, có nguồn, có ngày), nạp như context file, không cần mã.

## 2026-10-16 — Chính sách skill: đề xuất chờ người dùng duyệt
**Trạng thái:** đề xuất, chưa áp (chờ người dùng duyệt)
- **Chốt:** mã chỉ đổi một chỗ: không quét thư mục ẩn dưới gốc skill (vd `.trash`). Phần còn lại là đề xuất (`reports/system-evolution/proposals/agent-cli-skill-policy.md`, chưa vào git): tắt 21 skill bằng `disabled_skills` (skill "luôn bật" mà agent-cli không có cơ chế bật; skill app Claude ở `synced` dùng công cụ agent-cli không có), và đưa luật "luôn bật" vào `~/.config/agent-cli/CRUSH.md`, context toàn cục nạp mọi phiên. Agent không tự sửa cấu hình thật của người dùng.
- **Vì:** khối skill được gửi lại ở mọi lời gọi main. Đo bằng provider giả với skill thật của máy: 31.860 B / 48 skill ở bản gốc; 27.169 B / 40 skill sau khi bỏ thư mục ẩn; 9.656 B / 19 skill nếu áp danh sách tắt. Log 41 lượt dùng thật: 2 lượt có nạp skill. Luật phải luôn áp dụng nên là context file (tất định), không phải skill mà model có thể không đọc. CLAUDE.md P7 cấm agent đụng `~/.config/agent-cli` thật.
- **Đã thử và bỏ:** chưa (đề xuất chưa áp).
- **Lật lại khi:** người dùng duyệt (áp, rồi A/B 2–3 kịch bản có và không có danh sách tắt với LLM thật, đo token và đạt/không) hoặc từ chối (đóng đề xuất).

## 2026-10-16 — Fallback model lúc khởi động chỉ cho tiến trình đó, không ghi bền
**Trạng thái:** đã lật trong cùng đợt (phần fallback chéo provider trong RAM), xem mục "Model đang chọn không dùng được: dừng, không đổi provider"; phần "không ghi bền" giữ nguyên
- **Chốt:** khi model đang chọn không dùng được lúc `Load` (provider dựng sẵn thiếu API key, hoặc ID model không còn), agent-cli vẫn khởi động và dùng tạm model mặc định của provider khác còn cấu hình, ghi `slog.Warn` nêu cả model đã cấu hình lẫn model đang dùng, và KHÔNG ghi lựa chọn tạm xuống global data. Reload vốn đã không ghi; nay `Load` cũng vậy. Test: `TestStartupFallbackIsNotPersisted`.
- **Vì:** đọc mã + test (PRB-04): lựa chọn tạm được ghi đè vào `~/.local/share/agent-cli/crush.json`, nên một lần thiếu key thành đổi provider vĩnh viễn, và với allowlist (`AGENT_CLI_ALLOWED_MODELS`) thì mọi lần mở sau đều dừng `MODEL_NOT_ALLOWED` dù key đã có lại. Không thể dừng hẳn lúc khởi động vì khi đó không chạy được `agent-cli login` để có key (`TestConfigureProviders_SelectedKnownProviderWithoutCredentialsStillStarts`).
- **Đã thử và bỏ:** dừng khởi động khi provider dựng sẵn đang chọn thiếu key (khoá người dùng khỏi `login`).
- **Lật lại khi:** có chính sách "cross-provider = deny" tường minh: khi đó fallback chéo provider trong RAM cũng phải bị từ chối với lỗi nêu cách sửa, thay vì dùng tạm.

## 2026-10-16 — Model đang chọn không dùng được: dừng, không đổi provider
**Trạng thái:** đã làm (chưa phát hành)
- **Chốt:** `resolveSelectedModels` (dùng cho cả `Load` lẫn nạp lại) không bao giờ thay lựa chọn bằng model của provider khác. Chỉ mất ID model mà provider vẫn dùng được: model mặc định của CÙNG provider (catalog, không có thì model đầu), log cảnh báo, không ghi xuống đĩa. Provider không dùng được (thiếu key, không còn, bị tắt): giữ nguyên lựa chọn, cờ `LargeUnresolved`/`SmallUnresolved`, `Load` vẫn thành công (để `agent-cli login` chạy), và `buildAgentModels` trả lỗi nêu provider/model kèm cách gỡ dùng được lúc đó. Test: `TestStartupFallbackIsNotPersisted` (đổi kỳ vọng), `TestSelectedModelGoneFallsBackWithinProvider`, `TestResolveSelectedModels_SameProviderCatalogDefault`, `TestResolveSelectedModels_NoCrossProviderFallback`, `TestBuildAgentModelsFailsClosedWhenSelectedProviderHasNoKey`, `TestNoFallbackErrorKeepsSentinel`, `TestBuildAgentModelsRefusesADisabledProvider` (provider dựng sẵn `"disable": true` mà model vẫn biết: trước đây vẫn bị gọi), `TestReloadDoesNotSwitchProviderWhenTheKeyDisappears` (đường nạp lại); E2E `scripts/eval/faults.sh` ca `selected-no-key`.
- **Vì:** luật cứng của workflow (§32: fallback chéo provider phải fail closed mặc định) và red team cuối (F1 HIGH): không cần mượn provider khác để app khởi động được; chỉ cần app không đòi dựng agent lúc khởi động (bản vòng 6 dựa vào giả định "nạp được cấu hình là login chạy được", hoá ra sai vì `login` dựng cả workspace; xem "Đã thử và bỏ"). Đo bằng provider giả, chọn `openai/gpt-4o` không có key: bản trước `run` thoát 0 sau 2 request sang provider `glm` (`"provider":"glm"` trong `--json`); bản cuối thoát 1, 0 request, lỗi nêu provider/model ở cả local lẫn qua server (`faults.sh` ca `selected-no-key`, `selected-no-key-server`), `login` chạy được, TUI vào màn chọn model.
- **Đã thử và bỏ:** (1) gợi ý `-m provider/model` trong thông báo lỗi: đo bằng binary, `run -m qwen/qwen3-coder` vẫn dừng ở cùng lỗi (lỗi chặn trước khi `-m` được áp). (2) Để `app.New` coi lỗi này là chí mạng: TUI không mở, và `agent-cli login` cũng hỏng vì `login` dựng cả workspace (đọc `internal/cmd/login.go`, đo bằng probe `login no-such-platform`). Nay `app.New` khởi động không agent khi lỗi là "model đang chọn không dùng được" (`agent.SelectedModelUnavailable`, chỉ 4 lỗi provider/model không có; allowlist chặn vẫn dừng khởi động), TUI vào màn chọn model (đường onboarding sẵn có dựng agent sau khi chọn), `run`/`loop`/`routine` vẫn dừng ở `InitCoderAgentNonInteractive`. Test: `TestCreateWorkspaceStartsWhenSelectedModelIsUnavailable`, `TestUnavailableSelectedModelOpensOnboarding`, `TestSelectedModelUnavailableCoversOnlyUnusableSelections`; E2E `faults.sh` (probe `login`).
- **Lật lại khi:** có đường gọi LLM mới không đi qua `buildAgentModels` (khi đó app không agent có thể không còn là trạng thái an toàn).

## 2026-10-16 — Loop/routine: verify xanh sẵn và cây không đổi là `NO_CHANGE`, không phải `GREEN`
**Trạng thái:** đã làm (chưa phát hành)
- **Chốt:** engine dừng `NO_CHANGE` khi verify xanh ở baseline VÀ dấu vân tay cây làm việc (HEAD, diff, tệp chưa theo dõi) y như baseline. `loop` thoát 1; `routine run` thoát 0 cho routine `maintenance` (không có việc là kết quả hợp lệ của lượt theo lịch), 1 cho `goal`. Chỉ một trong hai điều kiện thì vẫn `GREEN` kèm cảnh báo trong `detail`. Test: `TestEngine_NoChangeCarriesBaselineAndUnchangedFlags` (đổi kỳ vọng), `TestEngine_BaselinePassWithAChangeIsGreenWithAWarning`, `TestEngine_BaselineFailWithoutAChangeIsGreenWithAWarning`, `TestExecuteLoop_NoChangeWhenTheGateAlreadyPassesAndNothingChanged`, `TestLoopExitErr_OnlyGreenExitsZero`, `TestRoutineExitErr_NoChangeIsZeroOnlyForMaintenance`, `TestExecuteRoutine_NoChangeIsReportedAndListed`, `TestExecuteRoutine_NoChangeInGoalModeExitsNonZero`.
- **Vì:** red team cuối (F4): kịch bản "chỉ Qwen" cho `stop: GREEN (verify already passed before any change)` với `loop.rc`=0, tức "xong" mà không có gì chứng minh việc đã làm (DONE giả). Cảnh báo trên stderr không đổi được mã thoát mà script đọc.
- **Đã thử và bỏ:** chỉ cảnh báo (bản trước trong đợt này).
- **Lật lại khi:** dấu vân tay bỏ sót thay đổi thật (tệp gitignore, thay đổi ngoài repo) làm `NO_CHANGE` oan thường xuyên.

## 2026-10-16 — Allowlist của lượt theo lịch chỉ có một chủ: tệp env
**Trạng thái:** đã làm (chưa phát hành)
- **Chốt:** `routine schedule --print` không chép `AGENT_CLI_ALLOWED_MODELS` vào unit/crontab nữa; lượt theo lịch đọc allowlist từ tệp env (`AGENT_CLI_ENV_FILE` vẫn được chép). Allowlist chỉ đặt ở shell thì in cảnh báo ra stderr. Test: `TestScheduleEnv_CarriesTheProviderBoundary`, `TestScheduleEnv_DropsASpaceSeparatedAllowlist`, `TestScheduleEnv_IgnoresProjectDotenv` (đổi kỳ vọng), `TestWarnAllowlistNotScheduled`.
- **Vì:** red team cuối (L6): biến export thắng tệp env (`execpolicy`), nên giá trị chép vào lịch lúc in sẽ che mọi lần siết allowlist sau đó trong tệp env. Hai chủ cho một quyết định.
- **Đã thử và bỏ:** chép allowlist vào lịch, đổi khoảng trắng thành dấu phẩy (bản trước trong đợt này).
- **Lật lại khi:** có người dùng không có tệp env mà vẫn cần allowlist cho lượt theo lịch.

## 2026-10-19 — Model thuộc workspace và chỉ đổi khi người dùng chọn; mở/tiếp session cũ không tự ghi model
**Trạng thái:** đã làm (chưa phát hành)
- **Chốt:** `run -s`/`run -C` (local lẫn qua server) và TUI mở session cũ không còn khôi phục model của session đó: lượt chạy model hiện tại của workspace, không ghi gì vào `.crush/crush.json`; khác model lần cuối của session thì báo một dòng nêu cả hai model (stderr ở CLI, thanh trạng thái ở TUI). Lựa chọn tường minh vẫn ghi như trước: `-m`, `--small-model`, `--reasoning-effort`, alt+p/alt+m, thinking/effort trong TUI, `agent-cli-tmux new --model`. Thay phần "khôi phục model chạy nền trong `tea.Cmd`" của mục 2026-10-15 ở trên.
- **Vì:** đo được (binary thật, LLM giả; `scripts/eval/session-model.sh` giữ ca này): `run -s X` qua server ghi model của X thành cấu hình dự án, mọi session sau (kể cả session đang dùng model khác) đi theo mà không ai chọn. Người dùng chốt 19/10: model thuộc workspace nhưng chỉ đổi khi người dùng chọn.
- **Đã thử và bỏ:** (a) model riêng từng session (mục 2026-10-15, phương án c) — đúng nhất nhưng đổi API model của UI, backend, CLI và proto; (b) giữ khôi phục trong RAM cho chế độ local — làm cùng lệnh `run -s` ra provider khác nhau tuỳ local hay server.
- **Lật lại khi:** cần model khác nhau theo session một cách bền (không chỉ "lượt giữ model lúc gửi") thì làm phương án (a).
