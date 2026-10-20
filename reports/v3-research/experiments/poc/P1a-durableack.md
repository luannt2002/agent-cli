# P1-a: ack ⇒ dấu vết bền (`send` không `--wait`, prompt xếp hàng)

Worktree `/home/luannt/application/opensource/agent-cli-p1a`, nhánh `v3-poc-p1a-durableack`, gốc `v2-final` (`7cba42ff`).
Chưa commit, chưa push. Tác giả bản sửa tự viết báo cáo này; **chưa qua người soát độc lập** (Opus, vòng đời + đồng thời).

**Patch hiện hành = vòng 4** (mục 12). `P1a-durableack.patch`, 31 tệp: 29 tệp của vòng 3 (mục 11 liệt kê), thêm
`internal/cmd/run.go` (+23/−12) và `internal/cmd/run_reply_model_test.go` (mới); `internal/message/message.go` nay
+91/−16.
Không gồm 5 tệp test phơi nhiễm `??` và `scripts/eval/exposure.sh` (so byte với bản chụp lúc bắt đầu: không đổi).
`sha256sum -c P1a-durableack.patch.sha256` → `e3ea0ef511bc1e63a8638b0887af77377180d5d44792a3f7d4ecae615eef808c`.
`git apply --check` rồi `git apply` trên cây `git archive 7cba42ff` sạch: rc 0; 31 tệp sau khi apply trùng byte với worktree.
Mục 1–9 là vòng 1, mục 10 là vòng 2, mục 11 là vòng 3 (giữ để đối chiếu); vòng 4 ở mục 12.

## 1. Vấn đề (đã tái hiện lại, không dùng số cũ)

| Ca | Binary gốc `7cba42ff` (`~/.cache/v3ev/p1a-red1`) |
|---|---|
| T11: `send --new` (rc 0) rồi SIGKILL server ngay | EXPOSED: 7/8 lần DB không có tin user; đối chứng kill sau 2 s có |
| T13: A chậm, B xếp hàng (rc 0), SIGTERM server | EXPOSED: DB chỉ có `user A, assistant canceled`; không có byte `T13-B` trong `crush.db*`; B chưa tới LLM |

Gốc (dòng theo `7cba42ff`, đã đọc lại):
- `internal/backend/agent.go:68` `go b.runAgent` rồi trả nil. Tin user chỉ được ghi ở `sessionAgent.Run` `agent.go:1014`,
  sau readiness, `UpdateModels`, `sessions.Get`, `getSessionMessages`.
- Session bận: `enqueueCall` `agent.go:556-581` chỉ giữ call trong `messageQueue` (RAM). Mọi đường bỏ hàng đợi chỉ phát
  `RunComplete` huỷ cho call có RunID (`publishCanceledQueueDrops` `:646`), không ghi gì vào DB: `drainQueueForStep`
  `:603-635`, `clearQueueAndNotify` `:675-680` (Cancel `:2594`, ClearQueue `:2704`), `CancelRun` `:2659-2690`,
  `nextQueued` `:1765-1803`. SIGTERM: `Workspace.Shutdown` huỷ ctx rồi `CancelAll` → Cancel xoá hàng đợi, hoặc lượt A
  thoát trước và B nằm lại trong RAM tới khi tiến trình chết. Cả hai đều không để lại gì.
- Gộp `:1096` ghi tin user lúc gộp; `persistCanceledTurn` `:719` ghi user + assistant huỷ cho đường huỷ-lúc-vào.
- `settle.go` (`repairInterruptedToolCalls`) chỉ sửa tin **trợ lý** dở (đóng "Interrupted"); không biết gì về tin user
  không có trả lời.

## 2. Thiết kế

**Dấu vết = một cặp tin ghi TRƯỚC khi ack:** tin user + ngay sau nó một tin trợ lý giữ chỗ (rỗng, chưa xong, model lúc gửi).

- `backend.SendMessage`: sau `BeginAccepted`, gọi `accept.Record(ws.ctx, call)`; lỗi ghi → nhả accept, `runWG`,
  `inflightRuns` và trả lỗi (không ack). `bindRun` dời xuống sau `Record` để lần gửi bị từ chối không gắn run ma vào client.
- `AcceptedRun.Record` → `sessionAgent.recordPrompt` (dùng lại `createUserMessage`, rồi `messages.Create` tin giữ chỗ; tạo
  tin giữ chỗ lỗi thì xoá tin user vừa ghi). Kết quả (`recordedPrompt{userMsgID, pendingMsgID}`) nằm trên handle;
  `Run` chép sang `call.recorded`; `enqueueCall` chép `call` nên hàng đợi mang theo.
- **Lượt chạy tiếp quản** (không ghi tin user lần hai): `Run` bỏ cặp tin khỏi `msgs` (lịch sử gửi model và kiểm "session
  chưa có tin user" để đặt tiêu đề). Cặp còn ở cuối session thì giữ nguyên; bị tin khác vượt qua (prompt đã chờ sau lượt
  khác) thì `moveRecordedPrompt` ghi lại cặp ở cuối rồi xoá cặp cũ (ghi trước, xoá sau). Tin giữ chỗ chỉ bị xoá trong
  `PrepareStep` **sau khi** lời trả lời thật của lượt đã được tạo, nên không có khe nào prompt chỉ còn tin user trần.
- **Gộp:** prompt có cặp tin được `moveRecordedPrompt` tới điểm gộp (thứ tự DB = thứ tự model thấy, như cũ), tin giữ chỗ
  mới bị xoá ngay sau khi tin trợ lý của bước được tạo.
- **Bỏ khỏi hàng đợi** (Cancel/Esc, ClearQueue, CancelRun, cancel mark ở `drainQueueForStep`/`nextQueued`): cả 4 đường
  vốn đã đi qua `publishCanceledQueueDrops`; hàm này nay đóng tin giữ chỗ là huỷ (`persistCanceledTurn` nhánh mới: call có
  cặp tin thì `AddFinish(Canceled)` lên tin giữ chỗ thay vì tạo cặp mới) trước khi phát `RunComplete`.
  `drainQueueForStep`/`nextQueued` trả cả call có cặp tin (không chỉ call có RunID) cho hàm đó.
- **Huỷ lúc vào / huỷ trước khi có trả lời** (`persistCanceledTurn` ở `Run`): call còn `recorded` thì đóng tin giữ chỗ.
- **Server tắt/chết:** không thêm gì. Tin giữ chỗ còn dở; `ReadSettledMessages` (TUI mở session, server
  `ListSessionMessages`) đóng nó là `error "Interrupted"` bằng đúng luật sẵn có của `settle.go` (tin trợ lý dở không có
  tool call). `settle.go` không đổi dòng nào. SIGTERM thường đi đường `CancelAll` → Cancel nên ra "huỷ" ngay.

Đã cân nhắc và bỏ:

| Phương án | Vì sao bỏ |
|---|---|
| Chỉ ghi tin user, thêm luật settle "tin user cuối không có trả lời" | Tin user của prompt xếp hàng nằm GIỮA lượt A; sau crash có tin trợ lý của A đứng sau nó nên luật theo vị trí không bắt được (B trông như đã được A trả lời) |
| Đánh dấu trên tin user (part/cột mới) | Đổi mô hình dữ liệu hoặc schema; renderer hiện có không hiện trạng thái trên tin user |
| Dùng lại tin giữ chỗ làm tin trợ lý bước 1 | `UpdateMessage` không ghi `model`/`provider` (`messages.sql:38-48`); lượt có thể chạy model khác lúc gửi → nhãn model sai |
| `SendMessage` chờ goroutine tới lúc `Run` ghi tin user | Ack bị trễ theo readiness/MCP (tới `InitWaitBudget`), lỗi trước `Run` phải dắt thêm tín hiệu; vẫn không phủ hàng đợi |
| Hàng đợi bền / tự chạy lại | D1: REJECT |

**Xoá cơ chế cũ:** không có cơ chế cũ nào bị thay thế mà còn để lại. Không thêm luồng phục hồi thứ hai (crash dùng nguyên
`settle.go`); hàng đợi vẫn chỉ trong RAM. Đường "ghi tin user lúc chạy" (`createUserMessage` trong `Run`/gộp) vẫn là
đường của người gọi trong tiến trình (agent con, loop/routine, stop hook, TUI chế độ in-process): họ không có ack qua
ranh giới tiến trình nên không có cặp tin; với call đi qua `SendMessage` đường đó không còn chạy.

## 3. Trả lời ba điểm của người soát điểm mù

| Điểm | Trả lời | Bằng chứng |
|---|---|---|
| (i) TUI không RunID, Esc/ClearQueue, fold, panic | Ghi ở `SendMessage`, không phụ thuộc RunID (TUI chế độ client gửi RunID rỗng qua đúng hàm này). Esc = Cancel, ClearQueue, CancelRun, cancel mark đều qua `publishCanceledQueueDrops` → đóng huỷ. Fold: `moveRecordedPrompt`. Panic: tiến trình chết → nhánh crash (settle); nếu có recover thì tin giữ chỗ dở và được settle khi session rảnh | Test `Canceled…` (Esc, RunID rỗng), `ClearedQueue…`, `CancelRun…`, `Folded…`, `TestQueueMarkDropClosesRecordedPrompt`. **Panic: chưa kiểm** (không có test, `runAgent` không có recover) |
| (ii) ghi trùng tin user ở fold | Fold ghi lại cặp ở điểm gộp và xoá cặp cũ; mỗi prompt đúng 1 tin user | `TestSendMessage_FoldedPromptNoDuplicate` (2 tin user, B tới LLM đúng 1 lần ở bước 2); đột biến M4 làm nó đỏ |
| (iii) ai đóng tin chờ sau crash | `settle.go` sẵn có, qua `ReadSettledMessages`; tin giữ chỗ là tin trợ lý dở nên luật cũ áp nguyên | `TestSendMessage_QueuedPromptSurvivesCrash`, `…Shutdown`; e2e mở lại qua server mới: `error "Interrupted"` đọc thẳng từ `crush.db` (mục 5) |

## 4. Test

Mới (`internal/backend/durable_ack_test.go`, đi qua `Backend.SendMessage` thật; đọc lại bằng `agent.ReadSettledMessages`
như tiến trình mới mở session; `saCoordinator` nối backend với `sessionAgent` thật + LLM giả; T11 dùng coordinator thật
có cổng chờ sẵn trong `accepted_run_integration_test.go`):

| Test | Khẳng định | Gốc `7cba42ff` | Bản sửa |
|---|---|---|---|
| `AckPersistsUserMessageBeforeRun` | ack khi lượt còn đứng trước `RunAccepted` ⇒ đã có 1 tin user; "crash" lúc đó → mở lại thấy Interrupted; huỷ sau đó không ghi thêm tin user (2 tin) | FAIL | PASS |
| `QueuedPromptSurvivesShutdown` | thứ tự `Workspace.Shutdown`; B: 1 tin user + trả lời huỷ/Interrupted; LLM chỉ 1 request | FAIL | PASS |
| `QueuedPromptSurvivesCrash` | DB lúc B còn trong RAM, mở lại: A và B đều Interrupted, đúng 2 tin trợ lý | FAIL | PASS |
| `QueuedPromptCanceledLeavesTrace` | Esc, RunID rỗng: DB thô ngay sau đó có B + huỷ | FAIL | PASS |
| `ClearedQueueLeavesTrace` | ClearQueue: B huỷ; A xong `end_turn` | FAIL | PASS |
| `QueuedPromptCancelRunLeavesTrace` | CancelRun theo RunID của B đang chờ: B huỷ | FAIL | PASS |
| `SinglePromptNoDuplicate` (đối chứng) | 1 prompt ⇒ đúng `[user, assistant end_turn]`, request chứa prompt 1 lần | PASS | PASS |
| `FoldedPromptNoDuplicate` (đối chứng) | gộp: 2 tin user, không tin trợ lý huỷ/lỗi, B chỉ ở request bước 2 | PASS | PASS |
| `QueuedRunIDPromptRunsInOrder` (đối chứng) | `[A, A trả lời, B, B trả lời]` | PASS | PASS |
| `QueuedRunIDPromptAfterMultiStepTurn` (đối chứng) | A nhiều bước: B và trả lời của B đứng sau MỌI tin của A | PASS | PASS |
| `TestQueueMarkDropClosesRecordedPrompt` (`internal/agent`, 2 subtest) | cancel mark phủ prompt có cặp tin ở `drainQueueForStep` và `nextQueued` → đóng huỷ | (API mới) | PASS |

"Gốc" = cây `7cba42ff` + đúng tệp test backend này: 6 FAIL, 4 đối chứng + test cũ PASS.

**Kiểm răng** (`teeth.py`: mỗi đột biến làm hỏng một mảnh trên bản sao cây đã sửa, chạy bộ test mới; tất cả đều đỏ):

| Đột biến | Test đỏ |
|---|---|
| M1 bỏ `accept.Record` trong `SendMessage` | 6 test phơi nhiễm backend |
| M2 bỏ đóng huỷ trong `publishCanceledQueueDrops` | mark-drop ×2, Canceled, ClearedQueue, CancelRun |
| M3 không bao giờ dời cặp tin | AfterMultiStepTurn |
| M4 gộp ghi tin user mới (bỏ qua cặp tin) | FoldedPromptNoDuplicate |
| M5 để cặp tin trong lịch sử gửi model | SinglePrompt, RunsInOrder, AfterMultiStepTurn |
| M6 không xoá tin giữ chỗ | 6 test (SinglePrompt, Folded, RunsInOrder, …) |
| M7 `persistCanceledTurn` bỏ qua cặp tin | mark-drop ×2, AcceptedCancelRace (test cũ), AckPersists…, Canceled, ClearedQueue, CancelRun |

Lần đầu M2 chỉ làm đỏ test mức agent: test backend đọc qua settle nên "Interrupted" che "huỷ". Đã thêm
`requireCanceledNow` (đọc DB thô) cho 3 đường bỏ hàng đợi; chạy lại M2 thì đỏ cả 3.

## 5. Bảng chân trị e2e (binary thật, LLM giả)

| Kịch bản | Gốc | Bản sửa |
|---|---|---|
| T11 `exposure.sh` (8 lần kill ngay + 1 đối chứng) | 7/8 mất tin user (EXPOSED) | 3 lượt (`p1a-green3..5`): 24/24 có tin user (NOT_EXPOSED); cả 27 session đều có ≥1 tin trợ lý sau tin user (25 có 1, 2 có 2: chết giữa lúc tạo trả lời thật và lúc xoá tin giữ chỗ) |
| T13 `exposure.sh` | EXPOSED | 3/3 NOT_EXPOSED: `user A, canceled, user B, canceled`; B không tới LLM; đối chứng 2 tin user cho 2 prompt |
| Mở lại sau SIGTERM, B xếp hàng (`reopen-e2e.sh`) | B không có | B `canceled` ngay cả trước khi mở lại |
| Mở lại sau SIGKILL, B xếp hàng | B không có | trước: tin giữ chỗ `UNFINISHED`; sau `run -s` (server mới, `ListSessionMessages`): `error "Interrupted"` |
| Mở lại sau T11 (4 lần) | 2/4 mất hẳn prompt | 4/4: tin trợ lý dở → `error "Interrupted"` sau khi mở lại |

Bản sửa đầu tiên (xoá tin giữ chỗ ngay lúc `Run` tiếp quản, `p1a-green1..2`) vẫn NOT_EXPOSED theo `exposure.sh` nhưng
2/16 session chỉ còn tin user trần (chết giữa lúc tiếp quản và lúc `PrepareStep` tạo trả lời): có dấu vết nhưng không
có trạng thái. Đã dời việc xoá tin giữ chỗ ra sau khi tạo trả lời thật; các số ở bảng trên là của bản cuối (`bin-fix2`).

## 6. Hồi quy

| Cổng | Kết quả |
|---|---|
| Suite theo TÊN, `go test -p 2 -json ./...` | Gốc 4684 tên (4658 pass/14 fail/12 skip), sửa 4697 (4671/14/12). Mất 0 tên, đổi trạng thái 0, 13 tên mới đều pass. Tập fail giống hệt: 5 `TestV3Exposure_*` (T1, T12, T15, C9 tools, C9 shell) + subtest |
| `TestCoderAgent` (cassette) | pass |
| `-race` `./internal/agent/... ./internal/backend/... ./internal/cmd/...` | 0 `DATA RACE`; chỉ đỏ 4 `TestV3Exposure_*` vốn đỏ của các gói này. Test mới thêm `-race -count=3`: ok |
| `faults.sh`, `qwen-only.sh`, `workspaces.sh` | PASS / PASS / PASS |
| `test-agent-cli-loop.sh`, `test-agent-cli-routine.sh` (`AGENT_CLI_BIN`) | PASS / PASS |
| `test-agent-cli-run-json.sh` (`AGENT_CLI_BIN`) | Bản gốc của script: binary gốc PASS (57 s), binary sửa **1 FAIL** |

FAIL đó là đỏ **mới do mình, đúng ý đồ**: check "lượt của send vẫn xong sau Ctrl+C của send và của run -s" khớp chính xác
danh sách lời trả lời của session là `end_turn|ECHO(..SLOW send;`. Prompt của `run -s` đã được server nhận (xếp hàng sau
lượt send), Ctrl+C gỡ nó; nay nó để lại `canceled|` ngay sau. Đó chính là bất biến được yêu cầu. Đã sửa đúng một
assertion thành `…SLOW send;canceled|;` (kèm comment) và giữ check kế bên "prompt đã gỡ không chạy". Script mới: binary
sửa PASS, binary gốc FAIL đúng check đó. **Người soát nên xét riêng thay đổi test này.**

## 7. Lệnh tái lập

`P` = scratch `.../scratchpad/p1a`. Mọi lệnh `go` chạy qua `$P/go.sh` (`flock /tmp/claude-1000/agent-go.lock`, chờ RAM
≥ 3000 MB, `GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off`), luôn có `-p 2`.

```bash
# unit đỏ trên gốc / xanh trên bản sửa
cd <cây> && GOMAXPROCS=2 go test -p 2 -count=1 -v -run 'TestSendMessage_|TestQueueMarkDropClosesRecordedPrompt' ./internal/backend/ ./internal/agent/
# binary
CGO_ENABLED=0 GOEXPERIMENT=greenteagc GOMAXPROCS=2 go build -p 2 -o $P/bin-fix2 .
# phơi nhiễm (T11 lặp vì là race)
for i in 3 4 5; do bash scripts/eval/exposure.sh ~/.cache/v3ev/p1a-green$i $P/bin-fix2 T11 T13; done
# mở lại qua server mới (nguồn: $P/reopen-e2e.sh, dùng lại hàm của exposure.sh; tóm tắt ở dưới)
bash $P/reopen-e2e.sh ~/.cache/v3ev/p1a-reopen1 $P/bin-fix2 4
# kiểm răng
python3 -I $P/teeth.py $P/teeth $P/go.sh $P/teeth.txt
# hồi quy
bash scripts/eval/faults.sh ~/.cache/v3ev/p1a-faults $P/bin-fix2   # tương tự qwen-only.sh, workspaces.sh
AGENT_CLI_BIN=$P/bin-fix2 bash scripts/test-agent-cli-run-json.sh   # tương tự loop, routine
```

`reopen-e2e.sh`: cắt phần hàm của `exposure.sh` (trước vòng `for c in $CASES`), `source` nó, rồi với mỗi ca: `send --new`
A (LLM giả ngủ 10 s), `send -s` B, `kill -TERM|-KILL` server, `session show --json` (đọc thô), `run -s <sid> "reopen
check"` (server mới; `restoreModelFromSession` gọi `ListMessages` → `ReadSettledMessages` → settle), `session show` lại.
Ca T11: `send --new` rồi `kill -9` ngay, 4 lần. Trạng thái `Interrupted` kiểm thẳng trong `crush.db` (`session show
--json` không in trường `message` của finish).

## 8. Rủi ro

1. **"Interrupted" giả tạm thời.** Settle chỉ hỏi `IsSessionBusy` (= `activeRequests`). Prompt đã ack mà chưa active
   (đang qua readiness; lượt phi tương tác chờ MCP tới `InitWaitBudget`), khe giữa hai lượt, hay lúc tự tóm tắt (`Run`
   gỡ `activeRequests` của session) thì người đọc có settle (TUI mở session, `run -s`) sẽ đóng tin giữ chỗ còn sống là
   Interrupted. Lượt sau đó vẫn chạy và xoá tin giữ chỗ nên trạng thái cuối đúng, nhưng người đọc thấy sai một lúc. Sửa
   tận gốc cần cho người đọc biết accepted/queued (interface `Coordinator` chưa có), nên chưa làm.
2. **Hai tin Interrupted cho một prompt** khi chết giữa lúc tạo trả lời thật và lúc xoá tin giữ chỗ (đo: 2/27 session
   T11). Chết trong khe dời cặp tin (sau khi ghi cặp mới, trước khi xoá cặp cũ) thì prompt hiện 2 lần. Chưa đo ca này.
3. **Lỗi trước khi có trả lời** (readiness, `UpdateModels`, provider chưa cấu hình, lỗi không phải huỷ trước
   `PrepareStep`): mở lại thấy "Interrupted", không thấy lỗi thật. Lỗi vẫn báo trực tiếp qua notification như cũ.
   Trước đây prompt mất hẳn.
4. **Prompt bị huỷ khi còn xếp hàng nằm ở chỗ nó được gửi** (có thể giữa các tin của lượt A) và chữ của nó vào lịch sử
   gửi model ở các lượt sau, như prompt huỷ-lúc-vào vẫn vào từ trước (tin trợ lý rỗng bị `preparePrompt` bỏ, tin user
   giữ lại). Trước đây prompt xếp hàng bị huỷ biến mất, không vào lịch sử.
5. **UX TUI chưa xem bằng mắt** (luật: không đụng màn hình người dùng). Prompt xếp hàng nay hiện trong chat ngay khi gửi,
   kèm tin trợ lý rỗng đang chờ, và vẫn hiện ở pill hàng đợi (hai chỗ). Khi gộp hay chạy sau lượt khác thì có cặp sự
   kiện Deleted/Created (prompt "nhảy" xuống cuối).
6. **Chế độ in-process** (`CRUSH_CLIENT_SERVER=0`: TUI gọi thẳng `coordinator.Run`) không qua `SendMessage` nên không có
   cặp tin; prompt xếp hàng ở đó vẫn bị bỏ im lặng khi Esc/ClearQueue. Launcher đặt `CRUSH_CLIENT_SERVER=1`.
7. `session show` của CLI đọc DB thô, không settle (nó không biết tiến trình khác có đang chạy lượt hay không). Sau crash
   nó hiện tin giữ chỗ là tin trợ lý rỗng chưa xong, tới khi session được mở qua server/TUI. Giới hạn này có từ trước.
8. Mỗi prompt qua `SendMessage` thêm 2 lần ghi (tạo + xoá tin giữ chỗ); prompt phải dời cặp thêm 2 lần nữa.

## 9. Chưa kiểm

- Panic trong `Run` (không có test, không có recover để thử).
- TUI thật (cả client lẫn in-process); LLM thật; cassette ngoài `TestCoderAgent`.
- `go vet` chỉ chạy cho `./internal/agent/ ./internal/backend/` (sạch); `task lint`/gofumpt toàn kho chưa chạy
  (`gofumpt -l` sạch trên 3 tệp Go chính, `gofmt -l` sạch trên `internal/agent`, `internal/backend`).
- `test-agent-cli-run-json.sh`, `-loop.sh`, `-routine.sh` tự tạo thư mục tạm ở `/tmp/acl*.XXXXXX` (cứng trong script),
  không nằm dưới `~/.cache/v3ev/` như các script nhận thư mục.
- Chết đúng khe dời cặp tin (rủi ro 2) và khe accepted-chưa-active với người đọc song song (rủi ro 1): chưa dựng ca đo.

## 10. Vòng 2 (sau lượt soát vòng đời độc lập: KHONG-DAT)

Cùng worktree, vẫn chưa commit. Bản cuối: `bin-fix4` (scratch `p1a/`). Đỏ trước khi sửa: chạy bộ test mới trên mã vòng 1
(`$P/round2-red.txt`).

### 10.1 Năm điểm phải sửa

| # | Lỗi (người soát) | Sửa | Test (đỏ trên mã vòng 1 → xanh) |
|---|---|---|---|
| 1 | Cặp tin đã ghi mà chưa được tiếp quản lọt vào request LLM của lượt khác (TUI: `[A, C, B, C]`), vào bản tóm tắt; session có prompt đang chờ thì không được đặt tiêu đề | Luật suy ra từ DB `withoutUnsentPrompts`: tin user đứng ngay trước một **tin giữ chỗ** thì cả hai không thuộc lịch sử gửi model. Áp ở `Run` (lịch sử + kiểm "đã có prompt" để đặt tiêu đề) và `summarize`. Tin giữ chỗ nay được đánh dấu bằng provider `agent-cli-pending` (không model). Không dùng dấu "model rỗng": test lộ ra rằng lời trả lời thật cũng có thể có model rỗng (model giả) nên sẽ bị bỏ nhầm. Kiểm "cặp tin ở cuối" nới thành "sau nó chỉ còn prompt khác đang chờ", nên không phải dời cặp khi chưa cần | `PendingPromptsFoldInOrder` (đỏ: `[A, C, B, C]`), `PendingRunIDPromptsInOrder` (đỏ: lượt B thấy `[A, C, B]`), `RecordedPromptStaysOutOfEarlierTurn` (B ghi trước khi A đọc lịch sử; tiêu đề; đỏ: không có tiêu đề), `SummarizeLeavesOutPendingPrompts` (đỏ: request tóm tắt có "pending B") |
| 2 | (Quyết định của điều phối) prompt huỷ/lỗi khi còn xếp hàng không vào lịch sử model | Cùng luật trên: tin giữ chỗ đã đóng huỷ, lỗi hay Interrupted vẫn là tin giữ chỗ nên bị bỏ khỏi request; UI vẫn hiện. Lượt bị huỷ giữa chừng có lời trả lời thật (provider thật) giữ nguyên như V2; `persistCanceledTurn` của người gọi trong tiến trình (provider thật) cũng như V2 | `DroppedQueuedPromptNotInHistory/{cancel,clear_queue}` (đỏ: `[A, B, D]` → xanh `[A, D]`, B vẫn hiện là huỷ) |
| 3 | Lượt hỏng trước khi có trả lời: TUI quay mãi, mở lại thấy "Interrupted" thay lỗi thật | `failRecordedPrompt`: đóng tin giữ chỗ còn mở bằng `error "Prompt not run"` + lỗi. Gọi từ: `AcceptedRun.Fail` trong `backend.runAgent` (lỗi trước `sessionAgent.Run`: readiness, MCP, `updateAgentModels`/provider tắt hay thiếu key, provider chưa cấu hình); một `defer` trong `Run` cho các lối thoát sớm; và nhánh `currentAssistant == nil` (trước `afterError`, vì lượt kế có thể ghi đè `retErr`). Chỉ đóng khi còn mở: không ghi đè huỷ hay Interrupted, không làm gì khi lượt đã tiếp quản | `PreRunErrorClosesPromptWithError` (coordinator thật, provider bị tắt → cùng `noFallbackError` "unavailable (missing key or login…)" như khi thiếu key; đỏ: tin giữ chỗ không đóng), `RunEarlyErrorClosesRecordedPrompt` |
| 4 | Settle ghi "Interrupted" lên tin giữ chỗ còn sống (đã nhận chưa active, khe giữa hai lượt, trước tóm tắt) | `HasPendingPrompts` (interface `SessionAgent` + `Coordinator`) trả true khi còn prompt xếp hàng hay `acceptedRuns > 0` (gồm cả transit). `SettleBusy(c) = IsSessionBusy ∥ HasPendingPrompts` được dùng ở 2 chỗ đọc có settle (`backend.ListSessionMessages`, `AppWorkspace.ListMessages`). `IsSessionBusy` giữ nguyên, nên ngữ nghĩa dispatch, UI và `is_busy` không đổi. Sau crash, tiến trình mới không giữ prompt nào nên settle vẫn đóng. Chọn method của interface thay vì assertion tuỳ chọn: lớp bọc (`stopHookCoordinator`, coordinator có cổng trong test) nhúng interface nên assertion sẽ âm thầm hụt. Cái giá là thêm 1 dòng ở 7 type giả | `SettleLeavesAcceptedPromptOpen` (đỏ: đã ghi Interrupted), `SettleBusyKeepsLivePromptsOpen` (đã nhận; xếp hàng mà không lượt nào active; tiến trình "khởi động lại" thì đóng) |
| 5 | Ghi DB (và `PublishMustDeliver`) của `publishCanceledQueueDrops` nằm dưới khoá dispatch | `clearQueueAndNotify` → `takeQueue` (chỉ lấy ra). Cancel, ClearQueue, CancelRun, nextQueued đăng ký `defer publishCanceledQueueDrops(drops)` TRƯỚC `Lock`, nên nó chạy sau `Unlock` (defer chạy ngược thứ tự). Việc phát `RunComplete` huỷ (vốn có từ trước, cũng dưới khoá) giờ cũng nằm ngoài khoá | `QueueDropWritesOutsideDispatchLock/{Cancel,ClearQueue,CancelRun,nextQueued}` (dò `TryLock` ngay trong lúc ghi; đỏ 4/4) |

NÊN:
- #3b/#6 (hai bản Interrupted / hai bản prompt khi chết giữa hai lần ghi): **chưa làm**. `message.Service` không có API
  transaction; thêm vào là đổi lớp store. Đo lại: 3/27 session T11 ở vòng 2 có hai tin trợ lý (`uaa`), 24 có một (`ua`).
- #8: CHANGELOG ghi rõ phạm vi client/server (in-process không đổi), dùng "Ctrl+C trên `run`", nói prompt chưa tới
  model thì không vào lịch sử, và ghi lỗi thật. Hướng dẫn: `send` không `--wait` và PRB-16 nêu phạm vi client/server.
- #3d (TUI hiện prompt xếp hàng ở cả pill lẫn chat): ghi nhận, chưa sửa (ngoài phạm vi; chưa xem bằng mắt).

### 10.2 Kết quả chạy lại (mã cuối)

| Cổng | Kết quả |
|---|---|
| Test mới (29 tên mới so với gốc, gồm subtest) | đỏ trên mã vòng 1 đúng các ca trên; xanh trên mã cuối |
| Suite theo TÊN (`go test -p 2 -json ./...`, mã cuối) | gốc 4684 tên, sửa 4713 (4687 pass/14 fail/12 skip). Mất 0, đổi trạng thái 0, 29 mới đều pass, tập fail giống hệt gốc (5 `TestV3Exposure_*`). `TestCoderAgent` pass |
| `-race` agent/backend/cmd/server/workspace | 0 `DATA RACE`; chỉ đỏ 4 `TestV3Exposure_*` vốn đỏ của các gói này |
| `exposure.sh T11 T13` | 3 lượt (`p1a-r2-exp1`, `-exp2` trên `bin-fix3`; `p1a-r2-final` trên `bin-fix4`): T11 24/24 có tin user, T13 NOT_EXPOSED (B huỷ, không tới LLM) |
| Mở lại qua server mới (`reopen-e2e.sh`, `p1a-r2-reopen`) | giống vòng 1: SIGTERM → B huỷ; SIGKILL → A, B `error "Interrupted"`; T11 4/4 Interrupted. Script thoát 143 do `wait` lên LLM giả bị kill ở bước dọn cuối (`finish`); log đủ 6 ca |
| faults, qwen-only, workspaces, run-json, loop, routine | đều PASS (run-json với assertion đã đổi ở vòng 1) |

`bin-fix3` được build trước một sửa nhỏ cuối cùng: thêm điều kiện `!IsSummaryMessage` trong `withoutUnsentPrompts`, để bản
tóm tắt (mang vai user ở đầu lịch sử) không bao giờ bị coi là prompt. Mọi chạy Go ở trên đều trên mã cuối; e2e chạy lại
T11/T13 trên `bin-fix4`.

### 10.3 Kiểm răng vòng 2 (`teeth2.py`, 13 đột biến; mỗi đột biến làm đỏ ít nhất một test)

| Đột biến | Test đỏ (rút gọn) |
|---|---|
| M1 bỏ `Record` | 11 test backend (ack, các đường bỏ, settle, lỗi trước khi chạy…) |
| M2 bỏ đóng huỷ ở drop | mark-drop ×2, lock-probe ×4, Canceled, ClearedQueue, CancelRun |
| M3 luôn coi cặp tin ở cuối | `RecordedPromptStaysOutOfEarlierTurn`, `QueuedRunIDPromptAfterMultiStepTurn` |
| M4 gộp ghi tin user mới | `FoldedPromptNoDuplicate` |
| M6 không xoá tin giữ chỗ | 12 test (thứ tự, trùng lặp, lịch sử) |
| M7 `persistCanceledTurn` bỏ qua cặp tin | 8 test |
| N1 `Run` không lọc prompt chưa gửi | 10 test (lịch sử, thứ tự, tiêu đề, tóm tắt…) |
| N2 `summarize` không lọc | `SummarizeLeavesOutPendingPrompts` |
| N3 bỏ `defer` đóng lỗi trong `Run` | `RunEarlyErrorClosesRecordedPrompt` |
| N4 bỏ `accept.Fail` trong `runAgent` | `PreRunErrorClosesPromptWithError` |
| N5 `SettleBusy` bỏ `HasPendingPrompts` | `SettleLeavesAcceptedPromptOpen`, `SettleBusyKeepsLivePromptsOpen` |
| N6a Cancel ghi dưới khoá | lock-probe/Cancel |
| N6b ClearQueue + CancelRun ghi dưới khoá | lock-probe/ClearQueue, lock-probe/CancelRun |

### 10.4 Rủi ro sau vòng 2 (thay mục 8)

Đã hết hoặc đã đổi: rủi ro 1 (Interrupted giả) đã sửa ở #4; rủi ro 3 (lỗi thành Interrupted) đã sửa ở #3; rủi ro 4 (prompt
huỷ vào lịch sử) đã đổi theo quyết định #2. Vẫn còn: 2 (hai bản khi chết giữa hai lần ghi, 3/27), 5 (UX TUI), 6
(in-process không đổi), 7 (`session show` đọc thô), 8 (thêm lần ghi). Mới:

1. **Provider `agent-cli-pending` là tên dành riêng.** Nếu người dùng tự đặt một provider có đúng ID này thì mọi lời trả
   lời của nó bị coi là tin giữ chỗ.
2. **Thống kê theo model** (`GetUsageByModel`) đếm tin giữ chỗ đã đóng (prompt không chạy) thành một dòng
   `model "" / provider agent-cli-pending`.
3. **Khôi phục model theo tin trợ lý cuối** (TUI `ui.go:3030`, `run.go:989`): nếu tin cuối là tin giữ chỗ (model rỗng)
   thì không khôi phục, thay vì lùi về lời trả lời thật trước đó.
4. **Prompt đã ghi mà bị huỷ ngay khi vào** (đường accepted, client/server) nay không vào lịch sử model. Ở V2, prompt
   huỷ-lúc-vào có vào; đường in-process vẫn như V2. Khớp quyết định #2 ("chưa từng tới model"), nhưng khác V2 ở đường này.
5. Ghi `RunComplete` huỷ của prompt bị bỏ nay chạy sau khi nhả khoá dispatch. Thứ tự giữa "prompt bị gỡ" và "sự kiện
   phát ra" không đổi với người gọi `CancelRun`/`Cancel` (vẫn phát xong trước khi hàm trả về), nhưng không còn được
   xếp tuần tự với các thao tác khác trên khoá.

Chưa kiểm (thêm vào mục 9): ca gộp mà ghi lại cặp tin bị lỗi giữa vòng gộp (để lại tin giữ chỗ mới, sau đó Interrupted);
TUI thật với luật lịch sử mới; e2e cho lỗi thiếu key qua `send` (chỉ có unit test với coordinator thật).

## 11. Vòng 3 (soát vòng đời: KHONG-DAT; 5 mục vòng 2 đã đóng đúng)

Cùng worktree, chưa commit. Binary cuối: `bin-fix5`. Đỏ trước khi sửa (chạy trên mã vòng 2): `$P/round3-red.txt`.

### 11.1 Mục chặn: luật "tin user đứng liền trước tin giữ chỗ" vỡ khi ghi đồng thời

Người soát chỉ ra hai lỗi. Thứ nhất, tin user và tin giữ chỗ là hai lần insert rời nhau. Hai `send` cùng lúc trên một
kết nối DB (`MaxOpenConns(1)`) có thể ghi ra thứ tự `[U_A, U_B, P_A, P_B]`, và tin của lượt đang chạy có thể chen giữa
`U_B` và `P_B`. Thứ hai, luật lọc dựa vào vị trí: khi các tin xen kẽ như trên, nó bỏ nhầm tin, hoặc để prompt đã huỷ ở
lại lịch sử mãi.

Thiết kế vòng 3 sửa từ gốc, có hai phần, mỗi phần chặn một kiểu hỏng:

1. **Liên kết thay cho vị trí.** Tin giữ chỗ lưu ID tin user của nó trong cột `model` (provider vẫn là
   `message.PendingReplyProvider`). `unsentPrompt(msgs)` bỏ khỏi lịch sử gửi model mọi tin giữ chỗ và mọi tin user mà
   một tin giữ chỗ trỏ về, bất kể nằm ở đâu.
   - Kiểm "cặp tin ở cuối" thành: sau tin user của lượt chỉ còn tin thuộc prompt chưa gửi.
   - Đúng dưới mọi thứ tự xen kẽ, kể cả khi `ORDER BY created_at` (độ phân giải 1 giây, hoà không có thứ tự định sẵn)
     đảo hai tin cùng giây.
   - Chỉ dùng transaction thì không đủ vì lý do đó: transaction giữ hai lần insert liền nhau trong DB, nhưng không giữ
     chúng liền nhau trong kết quả đọc.
2. **Transaction cho mọi lần ghi nhiều tin.** `message.Service.WriteBatch(ctx, session, create, deleteIDs)` làm đúng mẫu
   của `session.Delete`: `BeginTx`, `WithTx`, `Rollback` hoãn, `Commit`; sự kiện phát sau khi commit.
   - `*sql.DB` lấy qua `(*db.Queries).BeginTx` (tệp tay `internal/db/tx.go`), nên không phải đổi chữ ký
     `message.NewService` ở 13 nơi gọi.
   - `CreateMessageParams.ID` (tuỳ chọn) để tin giữ chỗ trỏ được về tin user tạo trong cùng lần ghi.
   - Dùng ở 3 chỗ:
     - (a) ghi cặp tin lúc nhận (`recordPrompt`);
     - (b) dời cặp: ghi cặp mới và xoá cặp cũ trong một lần;
     - (c) `PrepareStep` tạo lời trả lời thật và xoá tin giữ chỗ (của lượt và của các prompt vừa gộp) trong một lần.
   - Hết khe chết giữa hai lần ghi: không còn "hai bản Interrupted", "hai bản prompt", hay "tin user trần không tin giữ
     chỗ".

| Test | Thứ tự / lỗi tiêm | Đỏ trên mã vòng 2 | Mã cuối |
|---|---|---|---|
| `TestUnsentPromptsInterleavedPairs` | dựng thẳng `[U_A, U_B, P_A, P_B]` | request của A là `[A, A]` | `[A]`, `[A, B]`; DB `A, trả lời, B, trả lời` |
| `TestUnsentPromptWithToolResultBetween` | `[U_B, tool result, P_B]`, B huỷ | `[B, D]` (B huỷ vẫn vào lịch sử) | `[D]` |
| `TestSendMessage_ConcurrentSendsStayDistinct` | hai `SendMessage` thật cùng lúc (goroutine + barrier) khi A bận | (không ép được thứ tự trên mã cũ) | mỗi request chứa mỗi prompt ≤ 1 lần; 3 user / 3 trả lời; chạy `-count=3` |
| `TestSendMessage_ReplyReplacesPlaceholderAtomically` | trigger SQLite làm mọi lệnh xoá tin hỏng | 2 bản trả lời cho 1 prompt | 1 bản (tin giữ chỗ mang lỗi thật) |
| `TestSendMessage_MoveRecordedPromptAtomically` | như trên, lúc dời cặp tin của B | 2 bản prompt B | 1 bản, `error "Prompt not run"` |
| `TestWriteBatchIsAllOrNothing` | trigger làm insert thứ hai hỏng | (API mới) | không tạo gì, không xoá gì |

Đo lại T11 (3 lượt `p1a-r3-exp1..3` trên `bin-fix5`): 27/27 session đúng một tin trợ lý sau tin user (`ua`). Vòng 2 đo
được 3/27 session có hai bản; nay 0.

### 11.2 NÊN (đã làm)

- **#2 dời cặp hỏng.** `call.recorded` chỉ đổi khi dời thành công, nên `defer` đóng tin giữ chỗ cũ bằng lỗi thật (test
  MoveRecorded… kiểm `Message == "Prompt not run"`). Ở bước gộp: dời hỏng thì đóng cặp cũ của prompt đó; bước hỏng sau
  khi đã gộp (dời prompt kế, hay tạo lời trả lời) thì đóng tin giữ chỗ mới của các prompt đã gộp.
- **#3 cờ `agent-cli-pending` rò.**
  - Export `message.IsPromptPlaceholder` cùng `message.PendingReplyProvider`, theo ý điều phối. Chưa sửa khôi phục model
    ở `ui.go`/`run.go`, vì nhánh P1-c đã bỏ phần đó.
  - `GetUsageByModel` và `GetAverageResponseTime` loại provider này (`TestStatsSkipPromptPlaceholders`, đỏ trước khi
    sửa).
  - Config từ chối provider tự đặt có ID này (`TestConfig_configureProvidersRejectsReservedID`).
- **#4 khe sau `takeQueue`, trước khi ghi "canceled": chưa đóng, ghi giới hạn.** Trong khe (một lần Get + Update) một
  người đọc có thể settle ghi "Interrupted". Việc ghi "canceled" sau đó dùng `AddFinish` nên thay nhãn. Nếu `Update` của
  settle tới sau thì nhãn cuối là "Interrupted": vẫn là trạng thái đã đóng, chỉ sai nhãn. Muốn đóng khe thì phải giữ
  prompt "đang bỏ" trong `HasPendingPrompts` (thêm một bộ đếm), hoặc ghi dưới khoá (trái #5).

### 11.3 Kết quả chạy lại (mã cuối)

| Cổng | Kết quả |
|---|---|
| Suite theo TÊN | gốc 4684 tên, sửa 4721 (4695 pass/14 fail/12 skip). Mất 0, đổi trạng thái 0, 37 tên mới đều pass, tập fail giống hệt gốc (5 `TestV3Exposure_*`). `TestCoderAgent` pass |
| `-race` agent/backend/cmd/server/workspace/message | 0 `DATA RACE`; chỉ đỏ 4 `TestV3Exposure_*` vốn đỏ của các gói này |
| `exposure.sh T11 T13` ×3 | NOT_EXPOSED ×3; T11 24/24 có tin user; 0 session hai bản |
| Mở lại qua server mới (`p1a-r3-reopen`) | SIGTERM → B huỷ; SIGKILL → A, B "Interrupted"; T11 4/4 "Interrupted". Thoát 143 là do `wait` lên LLM giả lúc dọn, như vòng 2 |
| faults, qwen-only, workspaces, run-json, loop, routine | đều PASS |

### 11.4 Kiểm răng vòng 3 (`teeth3.py`; tất cả đỏ)

| Đột biến | Test đỏ |
|---|---|
| X1 lọc theo vị trí thay vì liên kết | `UnsentPromptsInterleavedPairs`, `UnsentPromptWithToolResultBetween` |
| X2 `WriteBatch` không transaction | `WriteBatchIsAllOrNothing`, `ReplyReplacesPlaceholderAtomically`, `MoveRecordedPromptAtomically` |
| X3 dời hỏng làm mất cặp cũ (`call.recorded = nil`) | `MoveRecordedPromptAtomically` |
| X4 thống kê tính tin giữ chỗ | `StatsSkipPromptPlaceholders` |
| X5 cho phép provider ID dành riêng | `configureProvidersRejectsReservedID` |
| X6 tạo trả lời rồi xoá tin giữ chỗ ở lần ghi riêng | `ReplyReplacesPlaceholderAtomically` |
| X7 dời cặp: ghi mới rồi xoá cũ ở lần ghi riêng | `MoveRecordedPromptAtomically` |
| M1, M2, N1, N5 (vòng trước, chạy lại trên mã cuối) | vẫn đỏ (M1: 12 test; M2: 11; N1: 13; N5: 2) |

Lưu ý cho người soát: điều phối dự đoán "bỏ transaction thì test xen kẽ đỏ". Với thiết kế này, test xen kẽ đỏ khi bỏ
**liên kết** (X1). Bỏ **transaction** (X2) làm đỏ các test lỗi tiêm và test toàn-hoặc-không, vì transaction chặn kiểu
hỏng khác (chết hay lỗi giữa hai lần ghi).

### 11.5 Rủi ro sau vòng 3 (thay 10.4)

Đã hết: hai bản khi chết giữa hai lần ghi (rủi ro 2 cũ), thống kê đếm tin giữ chỗ, provider ID trùng. Còn lại:

1. **Khe sau `takeQueue`** (11.2 #4): nhãn có thể là "Interrupted" thay vì "canceled".
2. **UX TUI chưa xem bằng mắt.** Prompt xếp hàng hiện ở cả pill lẫn chat.
3. **In-process không đổi.**
4. **`session show` đọc thô.**
5. **Khôi phục model** bỏ qua khi tin cuối là tin giữ chỗ. Để lại cho lúc ghép P1-c.
6. **Prompt huỷ-lúc-vào trên đường accepted** không vào lịch sử model (khác V2 ở đường này, khớp quyết định #2).
7. **`WriteBatch` cần `*db.Queries` gắn với `*sql.DB`.** Service dựng trên `Querier` khác sẽ trả lỗi rõ ràng, không
   lặng lẽ ghi không transaction. Mọi nơi dựng hiện tại dùng `db.New(conn)`.
8. **`internal/db/stats.sql.go` được sửa tay** cho khớp `stats.sql` (không có `sqlc` trên máy). Lần chạy `sqlc generate`
   sau sẽ ra cùng nội dung, vì chỉ thêm điều kiện WHERE.
9. **Transaction giữ kết nối duy nhất** trong lúc ghi 2–4 tin. Ghi debounce và đọc của goroutine khác phải chờ, cỡ vài
   mili giây. Chưa đo dưới tải.

Chưa kiểm: TUI thật; tải cao với nhiều `send` đồng thời (chỉ có test 2 `send`); prompt đính kèm ảnh lớn đi qua
`WriteBatch` (cùng đường với `Create`).

## 12. Vòng 4 (soát vòng đời: còn một chỗ do vòng 3 gây ra)

**Phải sửa.** Vòng 3 cho tin giữ chỗ mang `model` = ID tin user. Phần báo provider/model của `run --json`
(`internal/cmd/run.go`, cả đường server lẫn local) chọn tin trợ lý cuối có `model` khác rỗng, nên chọn trúng tin giữ
chỗ. Hậu quả: (a) lượt hỏng trước khi chạy báo `"provider":"agent-cli-pending","model":"<uuid>"` thay vì để trống;
(b) `run -s --json` thành công mà sau lời trả lời của nó đã có prompt khác được ghi thì báo nhãn `agent-cli-pending`.

Sửa:
- Hai vòng lặp trùng nhau gom thành `lastReplyModel(msgs)`. Đường server dùng `protoReplies` để chép các trường cần từ
  `proto.Message`. Bước này tách riêng, không đổi hành vi; chạy test trên đó thì đỏ.
- Sau đó thêm điều kiện `!message.IsPromptPlaceholder(m)`.

| Test | Đỏ (sau khi tách hàm, trước điều kiện) | Sau sửa |
|---|---|---|
| `TestLastReplyModelSkipsPromptPlaceholders`: (a) chỉ có tin giữ chỗ đóng lỗi; (b) trả lời thật rồi tới prompt khác có tin giữ chỗ; (b) qua tin `proto` | (a) báo `agent-cli-pending` (`$P/round4-red.txt`) | rỗng; `openai`/`gpt-5` ở cả hai đường |

Đột biến Y1 (bỏ điều kiện): test đỏ.

**Nên (đã làm).**
- #4: hướng dẫn (mục `send` không `--wait`) ghi rằng `session show --json` hiện tin giữ chỗ là `assistant` với
  `provider` `agent-cli-pending` và `model` là ID tin user.
- #5: `WriteBatch` từ chối lô không có tin nào để tạo, để câu đầu của transaction luôn là câu ghi. Transaction mở bằng
  câu đọc rồi nâng lên ghi có thể gặp `SQLITE_BUSY` khi tiến trình khác đang ghi. Mã hiện tại không có nơi gọi nào với
  `create` rỗng. Test `TestWriteBatchNeedsCreate`; đột biến Y2 (bỏ chặn) làm nó đỏ.

**Chạy lại** (binary `bin-fix6`):

| Cổng | Kết quả |
|---|---|
| Suite theo TÊN `./internal/cmd/...` | gốc 196 tên (196 pass), sửa 197 (197 pass): mất 0, đổi 0, mới 1 (`TestLastReplyModelSkipsPromptPlaceholders`) pass |
| Bộ test P1-a (agent, backend, message, config) | pass |
| `test-agent-cli-run-json.sh` | PASS (0 FAIL) |
| `exposure.sh T11 T13` (`p1a-r4-exp`) | NOT_EXPOSED; T11: 9/9 session đúng một tin trợ lý (`ua`) |

**Chưa kiểm.** Chưa chạy e2e cho ca (a) với binary thật (cần cấu hình làm lượt hỏng trước khi chạy mà config vẫn nạp
được); chỉ có unit test trên hàm chọn, phủ cả hai đường.

**Ghi nhận ngoài phạm vi.** Hướng dẫn (mục server bị dừng) nói tool call dở được ghi "Interrupted" khi mở lại bằng TUI,
`session show`, `run -s`. Thực ra `session show` đọc DB thô, không settle (mục 8.7). Câu này có từ trước P1-a, chưa sửa.


## Sau soát (điều phối, 19/10)

- Soát vòng đời: KHONG-DAT ×3 (vòng 1: prompt chờ lọt vào request; vòng 2: luật vị trí vỡ dưới `send` đồng thời; vòng 3:
  nhãn `run --json`) → **DAT** (vòng 4). Điều phối chạy lại mỗi vòng: T11+T13 NOT_EXPOSED, run-json 0 FAIL.
- Commit `3c888208` trên nhánh `v3-poc-p1a-durableack`. Patch cuối thay bản cũ cùng tên.
- Khi ghép với P1-c: `sessionModelNote` phải bỏ qua placeholder (`message.IsPromptPlaceholder`).
- Còn mở: khe sau `takeQueue` trước khi ghi "canceled" (nhãn Interrupted thay canceled); TUI hiện prompt xếp hàng cả pill lẫn
  chat (chưa xem bằng mắt); chế độ in-process giữ nguyên; `stats.sql.go` sửa tay (không có sqlc) — chạy `sqlc generate` khi có.
