# P2a: prompt chỉ dẫn dùng tool mà agent không có (T5 `<mcp-instructions>`, T6 todo reminder)

- **Worktree:** `/home/luannt/application/opensource/agent-cli-p2a`, nhánh `v3-poc-p2a-toolprompt`, gốc `v2-final` (`7cba42ff`). Chưa commit, chưa push.
- **`P2a-toolprompt.patch`:** sửa T5. Gồm 3 tệp: `internal/agent/agent.go`, `internal/agent/tool_prompt_test.go` (mới), `CHANGELOG.md`. Không gồm 6 tệp phơi nhiễm `??`.
  - sha256 `c92444600c7711c595bb8723bc9d00e34883835f329828b2e7c760bfe10aceee` (bản sau soát, mục 8).
  - `git apply --check` trên `v2-final` sạch: rc=0. Áp xong, cả 3 tệp `cmp` trùng worktree.
- **`P2a-toolprompt-todo.patch`:** sửa phần todo reminder của T6, áp SAU patch trên. **Chặn bởi cassette**, xem mục 3.
  - sha256 `75972e42f754832d51518994805a19e7cb3ec02435d7f3ddb86b96a535e22832` (dựng lại trên patch chính sau soát, mục 8).
  - `git apply --check` sau patch 1: rc=0. Áp xong, cả 5 tệp `cmp` trùng cây đã đo.

## 0. Kết luận

| Mục | Kết quả (lệnh vừa chạy) |
|---|---|
| T5 | EXPOSED → **NOT_EXPOSED**. Agent con nhận `<mcp-instructions>` ở 1/1 request trước khi sửa, 0/1 sau khi sửa |
| T6, phần todo reminder | Đã sửa và đã đo (`agentic_fetch`, plan: reminder True→False; agent chính True→True). **Không nằm trong patch chính**: làm đỏ 13/13 cassette `TestCoderAgent` |
| T6, phán quyết chung | Vẫn EXPOSED do `find_references` và `question` (template, ngoài phạm vi) |
| Agent chính | System prompt giống từng byte khi có MCP (T5) và khi không có MCP (T5ctl, T6). Plan và agent con giống từng byte khi không có MCP (plan chỉ đo ở T6, không có MCP). Agent chính vẫn nhận đủ chỉ dẫn MCP |
| Hồi quy patch chính | `TestCoderAgent` 13/13 PASS. Suite toàn kho so theo tên: 0 test mất, 0 test đổi trạng thái, 1 test mới PASS. `-race ./internal/agent/...` 0 data race. `faults`, `qwen-only`, `workspaces`, `run-json` PASS |
| Kiểm răng | 3 đột biến của bản sửa MCP và 1 đột biến của bản sửa todo đều làm test đỏ lại |

## 1. Vấn đề và gốc

- **T5.** `sessionAgent.Run` (`agent.go:958-973` ở `v2-final`) duyệt `mcp.GetStates()`. Nó gắn chỉ dẫn của MỌI server đang kết nối vào system prompt, mà không nhìn `agentTools` của lượt đó (biến này đã tính sẵn ở dòng 955).
  - Agent con (`task`) và plan có `AllowedMCP: {}` (`config.go:1127,1138`), nên không có tool MCP nào. `agentic_fetch` dựng bộ tool riêng, cũng không có tool MCP.
  - Ba agent này vẫn nhận lời dặn kiểu "always call stub_echo".
- **T6, todo reminder.** `preparePrompt` (`agent.go:2049-2060`) gắn reminder "use the `todos` tool" khi `!a.isSubAgent`. Có hai đường lọt:
  - `agentic_fetch` không đặt `IsSubAgent`.
  - Plan là agent ở slot chính.
  - Cả hai đều không có tool `todos`.
- **Bất biến dùng để sửa:** bộ tool của request được quyết MỘT lần (`agentTools`). Mọi chỗ sinh văn bản prompt phụ thuộc tool phải đọc từ đó.

## 2. Cách sửa (patch chính)

- Thêm `mcpInstructions(states, agentTools) string` (`agent.go:276`). `Run` gọi một dòng: `systemPrompt := a.systemPrompt.Get() + mcpInstructions(mcp.GetStates(), agentTools)`.
- Hàm giữ nguyên vòng lặp và định dạng khối cũ. Định dạng có test so từng byte. Hàm chỉ thêm một điều kiện: server phải nằm trong tập "chạm tới được" của `agentTools`.
- **Giả định 1 (lệch khỏi chữ của brief, có chủ ý):** lượt có `list_mcp_resources` hoặc `read_mcp_resource` được tính là chạm tới MỌI server.
  - Lý do: hai tool này nhận `mcp_name` bất kỳ và không bị `AllowedMCP` giới hạn (`read_mcp_resource.go:72`, `list_mcp_resources.go:66`).
  - Agent chính có hai tool này mỗi khi cấu hình có MCP (`coordinator.go:951-957`). Vì vậy hành vi agent chính giữ nguyên cả ở hai ca:
    - server chỉ có resource (0 tool), hoặc có tool nhưng đã bị `disabled_tools` tắt hết;
    - lượt từ channel, khi `filterToolsForChannel` đã bỏ tool của channel khác.
  - Nếu chỉ tính "có tool của server X", agent chính sẽ mất chỉ dẫn trong hai ca trên.
  - Agent con và plan không có hai tool này (`resolveReadOnlyTools`, `resolvePlanTools`).
- **Giả định 2:** khi có hook PreToolUse, tool của agent chính được bọc bởi `hookedTool`, và `hookedTool` không lộ `MCP()`. Hàm mới bóc lớp bọc trước khi đọc tên server. Không đụng `hookedTool` hay `filterToolsForChannel`; xem lỗi có sẵn ở mục 6.

## 3. Chặn bởi cassette

`TestCoderAgent` so khớp NGUYÊN body request (`charm.land/x/vcr@v0.1.1/matcher.go`). Ghi lại cassette cần provider thật (lỗi khi không khớp gọi tới `hyper.charm.land`). Chưa đo cách ghi lại.

| Mục | Ở đâu | Vì sao chặn |
|---|---|---|
| **Todo reminder chỉ khi có `todos`** (brief không lường trước) | `P2a-toolprompt-todo.patch` | Fixture `coderAgent` (`internal/agent/common_test.go:175-187`) có 11 tool và KHÔNG có `todos`. Cả 13 cassette đều chứa reminder (`grep -c "todo list is currently empty"` ra 1–5 mỗi tệp). Tool list của cassette không có `todos` (`grep 'name\":\"todos'` ra 0). Sửa đúng bất biến thì fixture mất reminder, body lệch. Thêm `todos` vào fixture thì tool list lệch. Không cách nào giữ xanh |
| `find_references` (tool không tồn tại) | `templates/coder.md.tpl:91` | Sửa template là đổi system prompt agent chính |
| `question` trong plan phi tương tác | `templates/plan.md.tpl:10,23` | Như trên |
| `web_fetch` trong system prompt agent chính (agent chính không có tool này; `exposure.sh` không tính vào phán quyết) | `coder.md.tpl`, dòng 306 của system prompt đã render | Như trên |

**Bằng chứng nhân quả cho dòng đầu.**

| Cây | `go test ./internal/agent -run TestCoderAgent -count=1` |
|---|---|
| Patch 1 | ok, 13/13 PASS, 2,2 s |
| Patch 1 + todo patch | FAIL 13/13, 458 s (`Request interaction not found`) |
| Patch 1 + todo patch, ép `todoReminder = true` ngay trước `preparePrompt` | ok, 3,6 s |

Kết luận: thứ duy nhất làm lệch cassette là reminder bị bỏ khỏi fixture.

**Todo patch đã được chứng minh riêng, dù không vào patch chính:**
- Unit `TestTodoReminderOnlyWithTodosTool`: ĐỎ trước khi sửa (`without_todos`: expected false, actual true), XANH sau khi sửa.
- Agent package, bỏ riêng `TestCoderAgent`: 960 test → 964 test, 0 mất, 0 đổi trạng thái, 4 mới PASS.
- e2e T6: xem mục 4.
- Đột biến: mục 5.

**Quyết định trong todo patch.** Request tóm tắt không có tool nào, nên truyền `false`: request đó không còn reminder, tức đầu vào tóm tắt của agent chính bớt một tin user tổng hợp. Không muốn đổi thì sửa thành `true`.

**Việc người dùng cần quyết:** ghi lại cassette bằng provider thật rồi áp todo patch, hoặc chấp nhận phần todo của T6 còn mở.

## 4. Bằng chứng

Tệp gốc nằm trong `/tmp/claude-1000/-home-luannt-application/1ba1a44b-c74c-4a9d-b148-f58af7166a87/scratchpad/p2a/`. Thư mục e2e: `/home/luannt/.cache/v3ev/p2a-{red,B,12,B-faults,B-qwen-only,B-workspaces}`.

### ĐỎ, trước khi sửa

- **Unit** (đã tách hàm, giữ hành vi cũ):
  - `TestMCPInstructionsFollowRunTools` FAIL: `Should be empty, but was <mcp-instructions> ALPHA-INSTR BETA-INSTR` (agent không có tool MCP).
  - `TestTodoReminderOnlyWithTodosTool/without_todos` FAIL. Đối chứng `with_todos` PASS.
- **e2e** `exposure.sh /home/luannt/.cache/v3ev/p2a-red bin-base T5 T6`:
  - `T5: EXPOSED`. Agent con có `mcp-instructions in 1`, `mcp tools []`. Đối chứng không MCP: 0.
  - `T6: EXPOSED`. FINDINGS: `agentic_fetch gets the todo reminder without todos; non-interactive plan names question; main names find_references`. Dòng plan: `todo-reminder-without-todos=True`.

### XANH, patch chính (`bin-B`, thư mục `p2a-B`)

- `T5: NOT_EXPOSED`. Có MCP: `main=2 (mcp-instructions in 2) subagent=1 (mcp-instructions in 0)`.
- `T6: EXPOSED`. FINDINGS giữ nguyên ba mục, vì patch chính không gồm phần todo.

### XANH, patch chính + todo patch (`bin-1plus2`, thư mục `p2a-12`)

- `T5: NOT_EXPOSED`.
- `T6: EXPOSED`. FINDINGS chỉ còn `non-interactive plan names question; main names find_references`, tức hết phần todo.

### Đối chứng: so từng role với bản gốc

Lệnh: `python3 -I syscmp.py p2a-red <mới> <ca>`. Chuẩn hoá đường dẫn thư mục chạy và hash commit của repo thử, vì hai thứ này đổi theo mỗi lần chạy.

| Ca / role | system prompt | `<mcp-instructions>` | todo reminder | tool list |
|---|---|---|---|---|
| T5 main (có MCP), patch chính và 1+2 | giống từng byte, 21.997 B | True→True | True→True | giống |
| T5 subagent (có MCP) | chỉ mất khối MCP, 1.041→932 B | True→False | False→False | giống |
| T5ctl (không MCP), mọi role | giống từng byte | False | không đổi | giống |
| T6 main | giống, 21.888 B | — | True→True | giống |
| T6 plan | giống, 5.087 B | — | patch chính True→True; 1+2 True→**False** | giống |
| T6 agentic_fetch | chỉ khác tên thư mục tạm `crush-fetch-<số>` | — | patch chính True→True; 1+2 True→**False** | giống |

### Hồi quy, patch chính

- **`TestCoderAgent`:** ok, 13/13 PASS.
- **Suite toàn kho so theo TÊN:** `go test -json -count=1 -p 2 ./...`. Baseline chạy trên worktree tạm `7cba42ff` cộng đúng 6 tệp phơi nhiễm.
  - Base: 4.684 test (4.658 pass, 14 fail, 12 skip). Mới: 4.685 test (4.659, 14, 12).
  - Mất 0, đổi trạng thái 0. Thêm 1: `TestMCPInstructionsFollowRunTools` pass.
  - Tập fail trùng theo tên. Cả 14 là test phơi nhiễm V3 có sẵn (T12, T15, C9, T1), loại "có sẵn".
- **`-race -count=1 -p 2 ./internal/agent/...`:** 976 test, 0 dòng `DATA RACE`. 5 fail là T12 ×2, T15, C9 ×2, cùng tên với fail của baseline (baseline chạy không race; chưa chạy race trên baseline).
- **e2e với `bin-B`:**
  - `faults.sh`: PASS, rc=0, 164 s.
  - `qwen-only.sh`: PASS.
  - `workspaces.sh`: PASS.
  - `AGENT_CLI_BIN=bin-B scripts/test-agent-cli-run-json.sh`: PASS, rc=0, 59 s.
- **Tĩnh:** `go vet ./internal/agent/` sạch. `$(go env GOROOT)/bin/gofmt -l` sạch.

## 5. Kiểm răng: đột biến bản sửa phải làm test đỏ lại

Chạy trên worktree tạm, sau đó đã khôi phục và kiểm bằng `cmp`.

| Đột biến | Kết quả |
|---|---|
| Bỏ điều kiện lọc (`… \|\| false && !allServers && !reach[name]`) | FAIL `tool_prompt_test.go:43` "an agent with no MCP tool must not get <mcp-instructions>" |
| Bỏ bước bóc `hookedTool` | FAIL `:54` (ca hook) |
| Tắt luật `list/read_mcp_resource` | FAIL `:61` "list_mcp_resources" |
| Todo: `todoReminder && !a.isSubAgent` → `todoReminder \|\| !a.isSubAgent` | FAIL `without_todos` "todo reminder sent" |

## 6. Phát hiện ngoài phạm vi (chưa sửa)

Hai mục sau được chứng minh bằng probe chạy trên worktree tạm. Tệp probe đã xoá, không có trong patch.

- **`hookedTool` vô hiệu hoá phạm vi channel.** Khi có hook PreToolUse, `filterToolsForChannel` không nhận ra tool MCP đã bị bọc, nên lượt từ channel khác vẫn thấy tool gửi của `signal`. Probe đo trên lượt channel `switchboard`: bản không bọc giữ 0 tool, bản bọc giữ 1. Sửa có thể đi chung lối bóc lớp bọc như `mcpInstructions`.
- **Thứ tự khối `<mcp-instructions>` ngẫu nhiên khi có từ 2 server có chỉ dẫn**, vì vòng lặp duyệt map. Probe gọi 50 lần và ra 2 biến thể. System prompt đổi giữa các lượt thì prompt cache của provider bị trượt. Lỗi có sẵn: vòng cũ duyệt cùng map. **Đã sửa trong patch chính sau soát (mục 8).**
- **Mới nghi, chưa đo e2e:**
  - Agent chính tắt cả `list_mcp_resources` lẫn `read_mcp_resource` bằng `disabled_tools`, và có server có chỉ dẫn nhưng 0 tool ⇒ chỉ dẫn của server đó không còn được gửi. Đây là hành vi đúng theo bất biến, nhưng là thay đổi. Hiện chỉ unit phủ luật resource.
  - Custom agent có `AllowedMCP` hẹp sẽ chỉ nhận chỉ dẫn của server được phép, trừ khi nó có tool resource.
  - Cách kiểm: cấu hình như trên rồi chạy `exposure.sh` ca T5.

## 7. Lệnh tái lập

```
cd /home/luannt/application/opensource/agent-cli-p2a
GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off go test -p 2 -count=1 ./internal/agent -run 'TestMCPInstructionsFollowRunTools|TestCoderAgent' -v
CGO_ENABLED=0 GOMAXPROCS=2 go build -p 2 -o /tmp/x/bin .
bash scripts/eval/exposure.sh /home/luannt/.cache/v3ev/<thư-mục-mới> /tmp/x/bin T5 T6
```

## 8. Sau soát (soát độc lập DAT)

Worktree, luật chạy và thư mục scratch như trên. Patch chính vẫn gồm 3 tệp như cũ; cả `agent.go`, `tool_prompt_test.go` và `CHANGELOG.md` đều đổi. Todo patch được dựng lại trên patch chính mới.

| Việc | Đã làm | Bằng chứng (lệnh vừa chạy) |
|---|---|---|
| 1. `CHANGELOG.md` dẫn `scripts/eval/exposure.sh` (không có trong git) | Bỏ lời dẫn và số đo của script; thay bằng tên test trong repo. Todo patch cũng dẫn `TestTodoReminderOnlyWithTodosTool` thay cho script | `grep -c exposure.sh P2a-toolprompt.patch` → 0 |
| 2. Thứ tự `<mcp-instructions>` cố định | `mcpInstructions` duyệt `slices.Sorted(maps.Keys(states))`. Thêm 1 dòng `CHANGELOG` gộp vào mục P2-a | ĐỎ trước khi sửa: `TestMCPInstructionsOrderIsStable` (3 server, 50 lần gọi, so từng byte với chuỗi xếp theo tên) FAIL ngay lần gọi 0 (`BETA` trước `ALPHA`). XANH sau khi sửa. Ca 1 server: `TestMCPInstructionsFollowRunTools` vẫn so từng byte, PASS. `TestCoderAgent` 13/13 PASS |
| 3. Test khoá việc `Run` dùng `mcpInstructions(…, agentTools)` | `TestRunMCPInstructionsFollowRunTools`: MCP server HTTP thật trong test (`httptest` + go-sdk), đăng ký qua `mcp.InitializeSingle` với `config.NewTestStore`, gỡ bằng `mcp.DisableSingle`. Chạy `Run` thật, đọc system prompt từ request gửi model. Không song song, vì trạng thái MCP là của cả tiến trình | Agent không có tool MCP: không có chỉ dẫn. Agent có tool của server: có chỉ dẫn. Test này là test khoá, không đỏ trước khi sửa (patch chính đã đúng); răng của nó ở dòng đột biến dưới đây |
| 4. Câu §0 về plan | Sửa thành "giống từng byte khi không có MCP" | mục 0 |

**Đột biến (khôi phục xong, kiểm bằng `cmp`):**

| Đột biến | Kết quả |
|---|---|
| `Run` truyền `nil` thay cho `agentTools` | FAIL `TestRunMCPInstructionsFollowRunTools/tool_of_the_server` |
| `Run` bỏ hẳn `mcpInstructions` | FAIL `…/tool_of_the_server` |
| Bỏ điều kiện lọc trong `mcpInstructions` | FAIL `…/no_MCP_tool` (system prompt có `P2A-RUN-INSTR`) và FAIL `TestMCPInstructionsFollowRunTools` |
| Duyệt map, không sắp xếp | FAIL `TestMCPInstructionsOrderIsStable` (lần gọi 0) |

**Chạy lại:**

- **e2e** `exposure.sh /home/luannt/.cache/v3ev/p2a-B2 bin-B2 T5 T6`:
  - `T5: NOT_EXPOSED`. Agent con 0 khối, agent chính 2/2 request có khối.
  - `T6: EXPOSED`, cùng ba FINDINGS như patch chính trước soát.
- **So với bản gốc** (`syscmp.py p2a-red p2a-B2`): system prompt của agent chính giống từng byte ở T5 (21.997 B), T5ctl và T6. Plan và agent con ở T6 cũng giống. Agent con ở T5 chỉ mất khối MCP.
- **`go test -json -count=1 -p 2 ./internal/agent/...`, so theo TÊN với baseline `7cba42ff`:**
  - 975 → 980 test. Mất 0, đổi trạng thái 0.
  - Thêm 5 test, đều PASS: `TestMCPInstructionsFollowRunTools`, `TestMCPInstructionsOrderIsStable`, `TestRunMCPInstructionsFollowRunTools` và 2 subtest.
  - 5 fail là test phơi nhiễm V3 có sẵn (T12 ×2, T15, C9 ×2).
- **`-race -count=1 -p 2 ./internal/agent/...`:** 980 test, 0 dòng `DATA RACE`, cùng 5 fail phơi nhiễm có sẵn.
- **`go vet ./internal/agent/`** sạch. **`gofmt -l`** sạch.
- **Hai patch trên worktree tạm `7cba42ff` sạch:**
  - `git apply --check` rc=0 cho cả hai, áp nối tiếp; mọi tệp `cmp` trùng cây đã đo.
  - Todo patch: `./internal/agent/...` bỏ riêng `TestCoderAgent` cho 0 đổi trạng thái; 8 test và subtest mới đều PASS. Tệp phơi nhiễm không có trong cây sạch, nên 9 tên phơi nhiễm vắng mặt là đúng.
  - Worktree tạm đã gỡ.
- **Chưa chạy lại sau soát:** suite toàn kho (chỉ chạy `./internal/agent/...` như brief soát yêu cầu); `faults`, `qwen-only`, `workspaces`, `run-json`; `TestCoderAgent` trên todo patch (đã đo đỏ 13/13 ở bản trước; phần thay đổi của todo patch không đổi).
- **Không làm (điều phối mở POC riêng):** bóc `*hookedTool` cho `filterToolsForChannel` (`:257`), `:284`, `filterDisabledMCPTools` (`:2202`).
