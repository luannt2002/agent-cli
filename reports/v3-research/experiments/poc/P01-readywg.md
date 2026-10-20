# P01: server panic `sync: WaitGroup is reused before previous Wait has returned` khi nhiều `run` đồng thời

Worktree `/home/luannt/application/opensource/agent-cli-p01`, nhánh `v3-poc-p01-readywg`, gốc `v2-final` (`7cba42ff`). Chưa commit, chưa push.
Patch: `P01-readywg.patch` (2 tệp: `internal/agent/coordinator.go` +51/−2, `internal/agent/coordinator_readiness_test.go` +135/−0; không gồm 6 tệp phơi nhiễm `??`, mtime của chúng 17:16–17:30 tức trước phiên này, không bị sửa).
`sha256sum -c P01-readywg.patch.sha256` → `c63b7eee5adf8d35b3326114ad67ac0e6b91a6c59cb279c7a783c3e99dbede4b`. `git apply --cached --check` trên index dựng từ `7cba42ff` (index tạm, không đụng cây làm việc) cho rc=0.

## 1. Vấn đề

- **Triệu chứng (ca T20 của `scripts/eval/exposure.sh`):** khoảng 10 lệnh `run` vào cùng một server thì server panic, mọi client đang nối cùng chết. Stack: `coordinator.go:313` ← `coordinator.go:304` (`RunAccepted`).
- **Gốc:** `readyWg` là một `errgroup.Group` dùng chung cho cả coordinator. Bên trong là `sync.WaitGroup`. Luật của WaitGroup: một `Add` đưa bộ đếm từ 0 lên phải happens-before mọi `Wait`. Code ở đây gọi `Go` (tức `Add`) từ nhiều goroutine không đồng bộ, trong lúc một lượt khác đang `Wait`.

## 2. Phân tích đã kiểm (dòng theo `7cba42ff`)

Phân tích trong brief **đúng**. Mình đã đọc lại mã và chạy `-race` để xác nhận.

| Vị trí | Vai trò | Kiểm bằng |
|---|---|---|
| `internal/agent/coordinator.go:179` | `readyWg errgroup.Group`, field của coordinator, dùng chung | đọc mã |
| `coordinator.go:313` (trong `run`) | `c.readyWg.Wait()`, câu lệnh đầu tiên của mọi lượt | đọc mã; stack panic T20; stack `-race` |
| `coordinator.go:304` `RunAccepted` ← `internal/backend/agent.go:131` `runAgent` | lượt từ server | đọc mã; stack panic T20 |
| `coordinator.go:843`, `:852` (trong `buildAgent`, `:808`) | `c.readyWg.Go(...)`: dựng system prompt và bộ tool | đọc mã; stack `-race` (Read ở `:843`) |
| `coordinator.go:1544` `UpdateModels` → `:1558` `updateAgentModels` → `:1571` `buildTools` → `agent_tool.go:36` `agentTool` → `buildAgent` | mỗi `run` của CLI gọi `UpdateAgent` (`internal/cmd/run.go:441` → `backend/agent.go:203` → `app.go:528` → `UpdateModels`) | đọc mã; stack `-race`: `agent_tool.go:36` → `coordinator.go:843` |
| `coordinator.go:347` (trong `run`, sau `Wait`) → `updateAgentModels` → … → `buildAgent` | lượt nào cũng khởi việc dựng sub-agent mới | đọc mã |

Bổ sung khi kiểm:

- **Cơ chế panic** (`$GOROOT/src/sync/waitgroup.go:206-213`, Go 1.27.0):
  1. Lượt A đang chờ trong `Wait`.
  2. `Done` cuối cùng đặt state về 0 và đánh thức A.
  3. Trước khi A kịp đọc lại state, lượt B gọi `Go`, tức `Add` từ 0.
  4. A thấy state ≠ 0 và panic.
- **Không phải nguồn lỗi:** `Go` lồng, gọi từ BÊN TRONG một goroutine dựng (lúc dựng bộ tool của coder, `agentTool` lại gọi `buildAgent`). Lúc đó bộ đếm ≥ 1 nên hợp lệ. Bản sửa vẫn phải chờ được các việc lồng này.
- **Vì sao `run` chờ:** `buildAgent` trả agent về ngay, còn system prompt và tool được gắn sau, trong goroutine. Nếu không chờ, lượt đầu tiên (agent vừa dựng ở `NewCoordinator`) có thể chạy khi chưa có prompt và tool. Nếu một việc dựng lỗi, lỗi đó được giữ lại và làm mọi lượt sau trả lỗi (ngữ nghĩa của errgroup).

## 3. Thiết kế bản sửa

`readyWg` đổi kiểu sang `readyGroup`, một kiểu nhỏ đặt ngay dưới struct `coordinator`. Tên field và API `Go(func() error)` / `Wait() error` giữ nguyên, nên các nơi gọi và các test cũ (`readyWg.Wait()` ở `coordinator_readiness_test.go`, `stop_hook_test.go:489`) không phải sửa. Import `errgroup` được bỏ khỏi `coordinator.go`.

- Trạng thái gồm `pending int` và kênh `idle`, cả hai do `sync.Mutex` canh giữ.
  - `Go` tăng `pending`. Nếu `pending` đi lên từ 0 thì tạo kênh `idle` mới.
  - Goroutine xong thì giảm `pending`. Về 0 thì đóng `idle`.
  - `Wait` lấy `idle` hiện hành dưới khoá rồi chờ ngoài khoá, sau đó trả lỗi đầu tiên.
- **Giữ ngữ nghĩa:**
  - Lượt chờ tới khi không còn việc dựng nào dở, giống WaitGroup "bộ đếm về 0".
  - Việc dựng lồng tăng `pending` trước khi việc cha xong, nên `Wait` chờ cả việc lồng.
  - Lỗi đầu tiên được giữ mãi, như errgroup.
  - Zero value dùng được ngay, khớp các test dựng `&coordinator{...}` trần.
- **Không deadlock:** không ai giữ khoá trong lúc chờ. Việc dựng lồng chỉ giữ khoá một lúc ngắn để tăng `pending`.
- **Không còn dùng sai:** `Go` và `Wait` có thể đan xen tuỳ ý. Mọi chuyển trạng thái đều đi qua mutex hoặc đóng kênh, nên `-race` không còn gì để báo.

Phương án đã cân nhắc và bỏ:

| Phương án | Lý do bỏ |
|---|---|
| Giữ khoá trong suốt `Wait`, và `Go` cũng lấy khoá đó | Deadlock: `Go` lồng từ trong việc dựng sẽ chờ khoá mà `Wait` đang giữ, còn `Wait` thì chờ chính việc dựng đó. |
| `sync.Cond` + `for pending > 0 { Wait }` | Lượt có thể chờ mãi: dưới tải liên tục, bộ đếm có thể lên lại trước khi lượt đang chờ thức dậy. Kênh theo "thế hệ" trả về ngay khi bộ đếm đã chạm 0, đúng ngữ nghĩa WaitGroup. |
| Mỗi agent tự giữ readiness, `run` chỉ chờ agent của mình | Đổi interface `SessionAgent` và các mock, và đổi ngữ nghĩa (lỗi giữ mãi, chờ việc dựng của lượt khác). Rộng hơn mức cần để hết panic. Ghi ở mục 7 như hướng cải tiến. |
| Dựng đồng bộ trong `buildAgent`, bỏ hẳn readyWg | Chậm lúc khởi động (`NewCoordinator` đợi dựng prompt), đổi hành vi upstream. |
| Bọc `errgroup` sau mutex và tráo group theo thế hệ | Vẫn phải tự đếm `pending` để biết lúc tráo, nên phức tạp hơn tự viết thẳng. |

## 4. Test mới

`TestRunWaitIsSafeAgainstConcurrentAgentBuild`, trong `internal/agent/coordinator_readiness_test.go`, kèm 2 helper `holdReadiness` và `waitParked`. Test không chạy `t.Parallel`, để dump stack nhỏ và không bị test khác chen vào.

Cửa sổ panic quá hẹp, không dựng lại chắc chắn được nếu không có hook bên trong `sync`. Vì vậy test dựng **đúng thứ tự xấu** rồi để `-race` bắt vi phạm. WaitGroup tự có chú thích race: `Add` từ 0 ghi nhận như một Read, `Wait` chặn ghi nhận như một Write, trên `&wg.sema`. Hai truy cập này không có happens-before thì là DATA RACE. Phát hiện kiểu này dựa trên quan hệ happens-before, không dựa vào thời điểm, nên tất định.

Các bước của test:

1. **Coordinator:** dựng tay với cấu hình kín (provider giả `127.0.0.1:9`, không gọi mạng) và main agent là `mockSessionAgent` trả lời ngay.
2. **Goroutine lượt (W):**
   - Tạo trước, rồi chỉ dò `runtime.Stack` cho tới khi thấy việc dựng đang đứng. Không đụng `t.*`, khoá hay kênh chung, vì những thứ đó tạo happens-before và che mắt `-race`.
   - Sau đó gọi `coord.Run(...)`.
3. **Goroutine test (T):**
   - Gọi `coord.agentTool(ctx)`. Đây là `buildAgent` → `Go` từ 0, đúng đường `buildTools` đi qua.
   - Sau đó thêm một việc dựng chặn trên kênh (`holdReadiness`), mô phỏng dựng prompt chậm.
   - Giữa `Go` và lúc W chờ, T chỉ làm việc nhẹ. Lý do: `-race` chỉ báo khi còn khôi phục được stack của truy cập trước. Bản đầu tiên gọi `UpdateModels`; phần còn lại của `buildTools` đẩy truy cập đó khỏi lịch sử và test xanh trên mã lỗi. Đặt `GORACE=history_size=7` thì bản đó lại đỏ, đã chạy để xác nhận nguyên nhân.
4. **Ngữ nghĩa:**
   - W phải đứng trong `run` ở một hàm `.Wait(` và chưa trả về khi việc dựng còn dở.
   - Thả việc dựng ngoài. Nó khởi một việc dựng lồng, và W vẫn phải chờ.
   - Thả việc lồng thì W trả về `nil`.
   - Cuối cùng `readyWg.Wait()` trả `nil`.
5. **Fail-closed:** nếu sau này đổi tên hàm hoặc khung stack, `waitParked` trả false và test đỏ, không xanh giả.

Khi đỏ, report trỏ đúng vị trí lỗi. Write ở `coordinator.go:313` (trong `run`, gọi từ `Run`). Previous read ở `coordinator.go:843` (trong `buildAgent`), gọi từ `agent_tool.go:36`.

## 5. Bảng chân trị

Mọi lệnh `go` chạy qua `flock /tmp/claude-1000/agent-go.lock`, với `GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off` và `-p 2`. RAM trống lúc chạy từ 6.978 đến 8.070 MB.

| Phép đo | Trước sửa (`7cba42ff` + test) | Sau sửa |
|---|---|---|
| Unit test mới, `-race`, binary test chạy 30 tiến trình riêng | **30/30 FAIL**, 30/30 có `WARNING: DATA RACE` | **30/30 PASS**, 0 race (chạy kèm `TestBuildAgentReadinessSurvivesCallerCancellation` và `TestNewCoordinatorWithStopHooks`) |
| Unit test mới, không `-race`, 30 tiến trình | 30/30 PASS (giới hạn: không `-race` thì test không bắt được lỗi, xem mục 7) | 30/30 PASS |
| `go test -json -race -count=1 -p 2 ./internal/agent/... ./internal/backend/... ./internal/cmd/...` | 1.335 pass. Đỏ: test mới (race), cùng 4 top-level `TestV3Exposure_*` (T12, T15, C9_Tools, T1) | 1.336 pass, **0 DATA RACE**. Đỏ: chỉ còn 4 `TestV3Exposure_*` như trước. So theo tên: 1.355 tên trước/sau, mất 0, mới 0 |
| `go test -json -count=1 -p 2 ./...` | 4.659 pass. Đỏ: 5 top-level `TestV3Exposure_*` (T12, T15, C9_ToolsOutsideWorkingDir, T1, C9_ShellLeavesWorkspace) | 4.659 pass. Tập đỏ **trùng khít theo tên** với trước (`diff` rỗng). 4.685 tên trước/sau, mất 0 |
| T20, `T20_TRIALS=10` (10 lạnh + 10 ấm + 10 tuần tự đối chứng mỗi lượt), binary `agent-cli-before` | **8/10 lượt EXPOSED**; 27/200 trial có `panic: sync: WaitGroup is reused…` ở `coordinator.go:313` | — |
| T20, binary sau sửa (`agent-cli-after`, 14 lượt xen kẽ với before) | — | **14/14 NOT_EXPOSED**; 0/280 trial panic; đối chứng tuần tự 10/10 rc 0 ở mọi lượt |
| Kiểm răng 2 chiều: hoàn bản sửa (`git show HEAD:internal/agent/coordinator.go`, giữ test) | unit `-race`: **FAIL**, có DATA RACE. Binary dựng lại có sha256 trùng `agent-cli-before`. T20: **3/3 lượt EXPOSED**, 4/60 trial có panic, thêm 1 trial lạnh (revert-3 `c8`: 9/10 client rc 1, server khởi động 2 lần, stderr bị lần dựng lại ghi đè) | trả bản sửa (sha256 tệp `050c6acb…` như trước khi hoàn): unit `-race` PASS (3 test readiness). Binary dựng lại có sha256 trùng `agent-cli-after`. T20 `p01-restore-1`: NOT_EXPOSED |

Ghi chú: binary T20 trước và sau đều dựng bằng `CGO_ENABLED=0 go build -p 2` (thêm các biến luật máy). Bằng chứng thô ở `/home/luannt/.cache/v3ev/p01-{before,after,revert,restore}-*/` (`verdicts.txt`, `T20/summary.txt`, `T20/w*-server.err`). Tỉ lệ panic trước sửa thay đổi theo lúc chạy: 4/120 ở 6 lượt đầu, 23/80 ở 4 lượt sau. Brief cũ ghi 4–7/10.

## 6. Lệnh tái lập

```bash
cd /home/luannt/application/opensource/agent-cli-p01
G='flock /tmp/claude-1000/agent-go.lock env GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off'
# Unit, tất định dưới -race: chạy N tiến trình riêng (-race khử trùng báo cáo theo PC trong một tiến trình, nên -count>1 chỉ đỏ lần đầu)
$G go test -race -c -o /tmp/agent-race.test ./internal/agent/
(cd internal/agent && for i in $(seq 30); do GOMAXPROCS=2 /tmp/agent-race.test -test.run '^TestRunWaitIsSafeAgainstConcurrentAgentBuild$' >/dev/null 2>&1 && echo PASS || echo FAIL; done | sort | uniq -c)
# Bộ race 3 gói + toàn kho, so tên đỏ
$G go test -json -race -count=1 -p 2 ./internal/agent/... ./internal/backend/... ./internal/cmd/... > race3.json
$G go test -json -count=1 -p 2 ./... > full.json
# E2E
$G env CGO_ENABLED=0 go build -p 2 -o /tmp/agent-cli-p01 .
T20_TRIALS=10 bash scripts/eval/exposure.sh /home/luannt/.cache/v3ev/p01-<mới> /tmp/agent-cli-p01 T20
grep -rl '^panic:' /home/luannt/.cache/v3ev/p01-<mới>/T20 --include=*.err --include=stderr.log | wc -l
# Kiểm răng: hoàn bản sửa, giữ test
cp internal/agent/coordinator.go /tmp/coordinator.go.fixed && git show HEAD:internal/agent/coordinator.go > internal/agent/coordinator.go
# … chạy lại unit -race + T20 (kỳ vọng đỏ/EXPOSED), rồi: cp /tmp/coordinator.go.fixed internal/agent/coordinator.go
```

## 7. Rủi ro và giới hạn

- **Test chỉ tất định dưới `-race`.** Không `-race` thì test xanh cả trên mã lỗi (đo được 30/30). Repo chạy `-race` ở `task test` (`Taskfile.yaml:85`) và CI upstream (`.github/workflows/build.yml:30`); CI có chạy cho fork này hay không thì CHƯA XÁC MINH (CLAUDE.md P6). Nếu ai đó thêm việc nặng giữa `agentTool` và lúc lượt chờ, test có thể xanh giả. Comment trong test đã ghi rõ điều này.
- **Test đọc chuỗi `runtime.Stack`** (`(*coordinator).run(`, `.Wait(`, `agent.holdReadiness(`). Nếu đổi tên thì test đỏ (fail-closed), không xanh giả.
- **Ngữ nghĩa cũ được giữ, kể cả những chỗ dở có sẵn:**
  - Một lượt vẫn chờ cả việc dựng sub-agent do lượt KHÁC khởi. Dưới tải đều, độ trễ của `Wait` có thể dài hơn cần thiết. CHƯA ĐO độ trễ này; T20 với 10 lượt ấm đồng thời cho rc 0 ở 280/280 trial.
  - Lỗi dựng đầu tiên vẫn dính mãi, làm hỏng mọi lượt sau trên coordinator đó.
- **Mới nghi, có sẵn từ trước, không sửa:** lượt không chờ sub-agent mà chính nó vừa dựng ở `updateAgentModels` (`coordinator.go:347`, nằm SAU `Wait` ở `:313`). Nếu model gọi tool `agent` trước khi prompt/tool của sub-agent được gắn, sub-agent sẽ chạy với system prompt rỗng.
  - Cách kiểm: cho `prompt.Build` của sub-agent chậm (chặn bằng kênh trong test) rồi gọi tool `agent` ngay; xem `SystemPrompt` của lượt sub-agent.
  - Hướng sửa nếu có thật: mỗi agent giữ readiness riêng.

## 8. Chưa kiểm

- Chưa chạy `task lint` / golangci-lint. Đã chạy: `gofmt -l` sạch, `gofumpt -l` sạch trên 2 tệp, `go vet ./internal/agent/` sạch.
- `-race` mới chạy cho `internal/agent/...`, `internal/backend/...`, `internal/cmd/...`. Chưa chạy cho `ui`, `workspace`, `server`, `permission`.
- Chưa thử với LLM thật hay TUI tương tác thật. Chưa thử pha T20 với `T20_N` khác 10.
- Chưa đo độ trễ của `Wait` dưới tải.

## Sau soát (điều phối, 19/10)

- Soát đồng thời độc lập: **KET-LUAN: DAT** (không đóng kênh hai lần, không chờ nhầm kênh cũ, không deadlock, starvation
  không tệ hơn errgroup cũ, lỗi đầu tiên giữ mãi đúng như errgroup).
- Điều phối tự kiểm: test mới `-race` 5/5 PASS trên bản sửa; đặt tạm `coordinator.go` của `v2-final` (giữ test) → `--- FAIL`
  + `WARNING: DATA RACE`, trả lại bản sửa (sha `050c6acb…` trước = sau); binary tự build T20 NOT_EXPOSED 2/2 lần.
- Thêm dòng CHANGELOG (người soát: bắt buộc trước khi gộp main). Commit trên nhánh `v3-poc-p01-readywg`; patch cuối
  (`git diff v2-final HEAD`) thay bản cũ cùng tên.
- Việc sau (không chặn): đưa bước giảm `pending` vào `defer` (Goexit trong `f` làm `Wait` treo — chỉ gặp ở test);
  ticket riêng: sub-agent có thể chạy với system prompt rỗng (`coordinator.go:402`, `agent_tool.go:36`, `:1842`; T25) và
  lỗi dựng đầu tiên làm hỏng mọi lượt sau tới khi InitAgent (có từ trước).
