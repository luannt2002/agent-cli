# P2d: có hook PreToolUse thì hai bộ lọc tool MCP bỏ sót tool MCP

- Worktree: `/home/luannt/application/opensource/agent-cli-p2d`.
- Nhánh: `v3-poc-p2d-hookedtool`, gốc `v2-final` (`7cba42ff`). Chưa commit, chưa push. Không dùng `git stash`.
- Patch: `P2d-hookedtool.patch`, 4 tệp (numstat):
  - `CHANGELOG.md` +6/−0
  - `internal/agent/agent.go` +2/−2
  - `internal/agent/hooked_tool.go` +13/−0
  - `internal/agent/hooked_tool_test.go` +76/−0
- `sha256sum -c P2d-hookedtool.patch.sha256` → `5161876cb0e1e98f4529d1b64e10fa126a192e3d746f4450fc0a11e937d682aa`, OK.
- `git apply --check` trên bản `git archive 7cba42ff` mới giải nén → rc=0.
- Cây làm việc không có tệp untracked. `gofmt -l` và `gofumpt -l` trên 3 tệp Go đều rỗng.

## 1. Lỗi đã được tái hiện bằng chạy

Brief ghi "chưa tái hiện bằng chạy". Nay đã tái hiện được bằng hai cách: unit test (mục 3) và binary thật (mục 4).

`wrapToolsWithHooks` (`coordinator.go:998`) bọc mọi tool của agent cấp cao trong `*hookedTool`. `hookedTool` chỉ chuyển tiếp `Info/Run/ProviderOptions/SetProviderOptions`, không chuyển tiếp `MCP()`. Vì vậy:

- **(a)** `filterToolsForChannel` (`agent.go:257`) dùng `agentTool.(interface{ MCP() string })`. Khi có hook, phép assert này luôn trượt, nên mọi tool bị coi là tool thường và được giữ lại. Lượt từ channel `signal` thấy cả `telegram_send`.
- **(b)** `filterDisabledMCPTools` (`agent.go:2180`; brief ghi `:2202`, số dòng lệch) dùng `t.(*tools.Tool)`, cũng luôn trượt. Server đã tắt vẫn đưa tool cho model. Fantasy chạy tool lấy từ chính `prepared.Tools` (`fantasy@v0.45.1/agent.go:490, 1446, 1772`), nên model **gọi được** tool đó.

## 2. Bản sửa

Thêm một helper `unwrapTool` trong `hooked_tool.go`. Hàm bóc mọi lớp `*hookedTool` (dùng vòng lặp, nên bọc lồng hai lớp cũng bóc hết). Hai bộ lọc gọi nó trước khi type-assert:

```go
mcpTool, ok := unwrapTool(agentTool).(interface{ MCP() string })   // agent.go:257
if mcpTool, ok := unwrapTool(t).(*tools.Tool); ok {                 // agent.go:2180
```

Hai bộ lọc vẫn trả về tool **đã bọc**, không trả bản đã bóc, nên hook vẫn chạy với các tool được giữ lại. E2E `fix-hook` chứng minh điều này: `hook.log` có `mcp_p2don_send`.

Khi không có hook, `unwrapTool(t) == t`, nên hành vi không đổi. Ca đối chứng `no hook` trong unit test và hai ca `*-nohook` của e2e đều cho cùng kết quả trước và sau sửa.

Ngoài `*hookedTool` không có lớp bọc nào khác. Lệnh `grep -rn '^\s\+\w\+\s\+fantasy\.AgentTool\s*$'` trên `internal/` (bỏ test) chỉ ra `hooked_tool.go:19`. Vì vậy không thêm interface `Unwrap()` chung.

## 3. Unit test (TDD): đỏ trước, xanh sau, có kiểm răng

Hai test mới nằm trong `internal/agent/hooked_tool_test.go`. Mỗi test có hai ca con: `no hook` (đối chứng) và `with hook`.

| Test | Khẳng định |
|---|---|
| `TestFilterToolsForChannel_SeesThroughHookedTool` | Có hai server channel `signal` và `telegram`. Lượt `signal` chỉ thấy `[signal_send, plain]`. Lượt local thấy đủ 3 tool. |
| `TestFilterDisabledMCPTools_SeesThroughHookedTool` | Dùng `*tools.Tool` thật: MCP server go-sdk chạy qua HTTP trong tiến trình, nối bằng `mcp.InitializeSingle`, lấy tool bằng `tools.GetMCPTools`. Server `p2d-off` bị tắt bằng `SetMCPServerDisabled` trên DB session thật. Kết quả lọc phải là `[plain]`. |

Test (b) cố ý không gọi `t.Parallel`, vì registry MCP là toàn cục. Khi xong, test gỡ hẳn server bằng `mcp.Reinitialize` với config rỗng, theo cách P1b đã làm. Test dùng `*tools.Tool` thật thay vì đổi assert thành interface, để bản sửa vẫn chỉ là thêm `unwrapTool`.

Các lệnh dưới đây chạy trong worktree. Mọi lệnh `go` đều đi qua `flock /tmp/claude-1000/agent-go.lock`, `GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off`, `-p 2`, và chỉ chạy khi RAM available ≥ 3000 MB.

| Bước | Lệnh | Kết quả |
|---|---|---|
| ĐỎ (mã `7cba42ff` + test) | `go test -count=1 -v -run 'TestFilterToolsForChannel_SeesThroughHookedTool\|TestFilterDisabledMCPTools_SeesThroughHookedTool\|TestFilterToolsForChannel$' ./internal/agent/` | Cả 2 ca `with_hook` FAIL: thừa `telegram_send`; thừa `mcp_p2d-off_send`. Cả 2 ca `no_hook` và `TestFilterToolsForChannel` cũ PASS. |
| XANH | Cùng lệnh, thêm `TestWrapToolsWithHooks\|TestHookedTool` | Tất cả PASS. |
| Kiểm răng 1 | Bỏ `unwrapTool` ở `:257` (sed, diff 1/1 dòng) | Chỉ `TestFilterToolsForChannel_SeesThroughHookedTool/with_hook` FAIL. Test (b) PASS. |
| Kiểm răng 2 | Bỏ `unwrapTool` ở `:2180` | Chỉ `TestFilterDisabledMCPTools_SeesThroughHookedTool/with_hook` FAIL. Test (a) PASS. |

Sau khi kiểm răng, `agent.go` được khôi phục và kiểm bằng `sha256sum -c` → OK.

## 4. E2E: binary thật, LLM giả, 2 MCP stub, hook (chứng minh (b))

Harness gồm `e2e.sh`, `mcp_stub.py` và `cmp.py`, có bản chép ở `/home/luannt/.cache/v3ev/p2d-harness/`. Cách dùng: `e2e.sh <thư-mục-mới> <binary> <hook 0|1>`.

Thiết lập:
- LLM giả là `scripts/eval/fakellm.py`.
- Có hai server MCP stdio là `p2don` và `p2doff`, mỗi server có một tool `send`.
- Hook `PreToolUse` là một script ghi `$CRUSH_TOOL_NAME` vào `hook.log`.
- LLM giả gọi lần lượt `mcp_p2doff_send`, rồi `mcp_p2don_send`, rồi trả chữ.

**Cách dựng ca tắt server:** lúc `initialize`, stub `p2doff` tự ghi tên mình vào `mcp_disabled_servers` trong `proj/.crush/crush.db`. Việc này mô phỏng một tiến trình agent-cli khác trên cùng repo vừa tắt server bằng "Toggle MCPs", trong khi tiến trình này vẫn giữ kết nối.

Lý do phải dựng như vậy: trong một tiến trình `run`, server đã tắt từ trước sẽ không được khởi động (`app.go:165`, `mcp.Initialize` bỏ qua `localDisabled`), nên không thể tái hiện bằng cấu hình tĩnh.

Binary được build bằng `CGO_ENABLED=0 go build`:
- `crush-base` build từ `git archive 7cba42ff`, sha256 `53a22f56…af10`.
- `crush-fix` build từ worktree, sha256 `60af75da…2c65`.

Hai binary nằm trong scratch `p2d/`.

| Ca (`/home/luannt/.cache/v3ev/p2d-<ca>`) | rc | `mcp_p2doff_send` trong 3 request main | `tools/call` tới p2doff | `hook.log` |
|---|---|---|---|---|
| `base-hook` | 0 | **3/3** (30 tool) | **có** | `mcp_p2doff_send`, `mcp_p2don_send` |
| `base-nohook` (đối chứng) | 0 | 0/3 (29 tool) | không | (không có hook) |
| `fix-hook` | 0 | **0/3** (29 tool) | **không** | `mcp_p2don_send` |
| `fix-nohook` | 0 | 0/3 (29 tool) | không | (không có hook) |

Trong cả 4 ca, `mcp_p2don_send` đều có mặt ở 3/3 request và `mcp.log` đều có `disabled-in-db p2doff`. Như vậy MCP thật sự đã nạp và dòng tắt đã được ghi, nên cổng không xanh vì rỗng.

## 5. Grep mọi type-assert trên tool

Lệnh: `grep -rn '\.(interface{\|\.(\*tools\.Tool)\|\.(\*hookedTool)\|hookedTool'` và `grep '\.(type)'` trên `internal/`, cùng `grep` trong fantasy v0.45.1.

| Chỗ | Trên giá trị gì | Cùng lớp lỗi? |
|---|---|---|
| `internal/agent/agent.go:257` `.(interface{ MCP() string })` | tool của lượt (đã bọc) | **Có, đã sửa** |
| `internal/agent/agent.go:2180` `.(*tools.Tool)` | tool của bước (đã bọc) | **Có, đã sửa** |
| `internal/agent/hooked_tool.go` `.(*hookedTool)` | chính helper `unwrapTool` | — |
| `internal/agent/hooked_tool_test.go:128,138` `.(*hookedTool)` | test khẳng định có bọc | Không |
| `internal/agent/coordinator.go:979` `tool.MCP()` | `*tools.Tool` TRƯỚC khi bọc (bọc ở `:998`) | Không |
| `internal/ui/chat/messages.go:174` `.(interface{ Bump() })`, `internal/ui/completions/completions.go:283` `.(interface{ Text() string })` | item của UI, không phải tool | Không |
| `.(type)` trong `internal/agent` (`channelreply.go`, `usage_fallback.go`, `tools/*`) | token, content, sym… | Không |
| fantasy `tool.go:128` `.(interface{ SetParallel(bool) })` | lúc dựng tool, trước khi bọc; cờ được đọc qua `Info().Parallel` (`agent.go:1773`), mà `Info()` thì hookedTool có chuyển tiếp | Không |
| fantasy `agent.go:1358` `.(ExecutableProviderTool)` | provider-defined tools, không phải `AgentTool` | Không |

**Chỗ thứ ba (`agent.go:284`):**
- Ở `v2-final`, `:284` nằm trong interface `SessionAgent`. Vòng nạp MCP instructions (`agent.go:960`) lặp trên `mcp.GetStates()`, không lặp trên tool. Vậy nhánh này không có chỗ thứ ba.
- Đã thử ghép: áp patch P2d rồi áp phần mã của `P2a-toolprompt.patch` (sha OK) lên một bản `7cba42ff` mới. Kết quả rc=0, chỉ `CHANGELOG.md` xung đột vì cả hai patch cùng chèn vào cuối mục.
- Sau khi ghép, bóc lớp riêng của P2-a nằm ở `agent.go:284-287` (`if h, ok := t.(*hookedTool); ok { t = h.inner }`). Lúc tích hợp nên thay nó bằng `unwrapTool(t).(interface{ MCP() string })`. Bóc riêng một lớp vẫn đúng với cách bọc hiện tại, nhưng để hai nơi bóc thì sau này dễ lệch nhau.

## 6. Hồi quy

| Kiểm | Lệnh | Kết quả |
|---|---|---|
| Suite đầy đủ, so theo TÊN | `go test -p 2 -count=1 -json ./...` trên `git archive 7cba42ff` và trên worktree, rồi `cmp.py` | base: 4722 tên (4693 pass, 0 fail, 29 skip). fix: 4728 tên (4699 pass, 0 fail, 29 skip). **Đỏ mới: 0.** Mất tên: 0. Thêm 6 tên, đều là 2 test mới cộng 4 ca con. |
| `TestCoderAgent` | nằm trong suite trên | 15/15 pass ở cả base và fix |
| `-race` | `CGO_ENABLED=1 go test -race -p 2 -count=1 ./internal/agent/...` | rc=0, 6 gói ok, `DATA RACE`: 0 |
| `faults.sh` | `bash scripts/eval/faults.sh /home/luannt/.cache/v3ev/p2d-faults-fix <crush-fix>` | rc=0, `FAULTS: PASS`, glm = 0 ở 11/11 ca |
| `qwen-only.sh` | `bash scripts/eval/qwen-only.sh /home/luannt/.cache/v3ev/p2d-qwen-fix <crush-fix>` | rc=0, `QWEN-ONLY: PASS`, đủ 5 vai trò đi qwen, `loop_rc=1` (NO_CHANGE) |

## 7. Đã chứng minh / Mới nghi

**Đã chứng minh** (có lệnh tái lập ở mục 3, 4, 6):
- Khi có hook, (a) và (b) hỏng trên `7cba42ff`.
- Server đã tắt vẫn bị gọi thật (`tools/call` tới p2doff).
- Bản sửa chặn được cả hai.
- Khi không có hook, hành vi không đổi.
- Không có đỏ mới theo tên.

**Mới nghi** (chưa chạy):
1. Ở (a), lệnh gửi chéo channel có lẽ vẫn bị `Tool.Run` từ chối lúc chạy (`mcp-tools.go`, nhánh `state.Channel && ch != "" && ch != m.mcpName`). Nếu đúng thì tác hại của (a) là lộ tool cho model và tốn một bước, không phải gửi tin thật. Mới đọc mã. Cách kiểm: gọi `Tool.Run` với ctx channel `A` trên tool của channel `B`.
2. Ngoài ca hai tiến trình đã dựng ở mục 4, (b) có thể còn lộ ở hai đường khác:
   - Server nhiều workspace dùng chung kết nối MCP: workspace A tắt server X, workspace B vẫn bật X.
   - `stepTools` giữ bộ tool đã ghim của lượt khi model chung đổi.

   Chưa đo cả hai.
3. E2E chỉ phủ (b). (a) mới có unit test, vì e2e channel cần server channel thật và lượt đến từ channel, tốn nhiều công dựng.
