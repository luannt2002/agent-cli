# P1b: kết quả tool MCP không có trần (T9) và mất cờ `isError` (T19)

Worktree `/home/luannt/application/opensource/agent-cli-p1b`, nhánh `v3-poc-p1b-mcpresult`, gốc `v2-final` (`7cba42ff`). Chưa commit, chưa push.
Patch: `P1b-mcpresult.patch`, 4 tệp (numstat): `CHANGELOG.md` +2/−0, `internal/agent/tools/mcp-tools.go` +9/−1, `internal/agent/tools/mcp/tools.go` +6/−1, `internal/agent/tools/mcp-tools_test.go` (mới) +106/−0.
Patch không gồm 6 tệp phơi nhiễm `??`. mtime của chúng là 17:16–17:30, trước phiên này, và chúng không bị sửa.
`sha256sum -c P1b-mcpresult.patch.sha256` → `4467d3648b98e2b5c9038ff2a3ce3c7d10e54081d619671def22e744e2e14754`, OK.
`git apply --cached --check` trên index tạm dựng từ `7cba42ff` (`GIT_INDEX_FILE` riêng, không đụng index hay cây làm việc) → rc=0.

## 1. Vấn đề

Đo lại bằng `scripts/eval/exposure.sh` trên binary dựng từ `7cba42ff` (`crush-before`, sha256 `cd384b73…`):

- **T9:** tool MCP trả 5 MiB văn bản. Request kế tiếp nặng thêm **+5.327.624 byte** và `crush.db*` tăng **+5.341.184 byte**. `tool_result` lưu đủ 5.242.880 ký tự. Đối chứng: bash in 5 MiB thì request kế tiếp chỉ +30.455 byte (bị cắt, có spill).
- **T19:** tool MCP trả `isError=true`, nhưng `tool_result` lưu `is_error=False`. Đối chứng: `view` một tệp không có → `is_error=True`.

## 2. Kiểm lại chủ cũ (dòng theo `7cba42ff`)

| Khẳng định trong brief | Kết quả kiểm | Kiểm bằng |
|---|---|---|
| Response được trả ở `internal/agent/tools/mcp-tools.go:135-157` | Đúng. Nhánh `default` gọi `fantasy.NewTextResponse(result.Content)`, không cắt, không có cờ lỗi | đọc mã |
| `mcp/tools.go` `RunTool` (`:42-115`) không đọc `result.IsError` và không có trần | Đúng. `ToolResult` không có trường lỗi; `CallToolResult.IsError` của go-sdk bị bỏ | đọc mã; go-sdk `protocol.go:305` |
| Đường spill sẵn có là `shell.TruncateForPersist` (`truncate.go:57`), bash dùng ở `bash.go:518-535` | **Lệch một chỗ.** Bash KHÔNG dùng `TruncateForPersist`. Bash dùng `tools.TruncateOutput` (`bash.go:519`, trần `MaxOutputLength` = 30000 ô, dấu `... [N lines truncated, full output: <tệp>] ...`), hàm này gọi `shell.SpillOutput`. `job_output.go:75` cũng dùng nó. `TruncateForPersist` chỉ phục vụ chế độ bang (`persist_message.go:32`), trần khác (50 KiB / 2000 dòng) và dấu khác. Cả hai cùng ghi vào `<data>/shell-output/` qua `SpillOutput`. | `grep -rn "TruncateForPersist\|SpillOutput\|TruncateOutput"` |

Brief yêu cầu "cùng trần, cùng thư mục spill, cùng thông điệp như bash". Vì vậy bản sửa dùng `TruncateOutput`, đúng hàm của bash và `job_output`. Không có đường spill thứ ba.

## 3. Thiết kế

1. `mcp.ToolResult` có thêm `IsError bool`. `RunTool` chép `result.IsError` của server vào cả 4 điểm trả về (rỗng, ảnh, audio, chữ). Lớp dữ liệu giữ đúng sự thật, còn quyết định dùng cờ này nằm ở `Tool.Run`.
2. Nhánh chữ của `Tool.Run` (`default`):
   - Gọi `TruncateOutput(result.Content, m.cfg.Config().Options.DataDirectory)`. Thư mục spill là cùng biểu thức coordinator truyền cho bash (`coordinator.go:901`) và `job_output` (`:904`), nên spill nằm ở cùng `<data>/shell-output/`. Không đổi chữ ký `GetMCPTools`, vì `Tool` đã giữ sẵn `cfg`.
   - Nếu `result.IsError` thì trả `fantasy.NewTextErrorResponse(content)`, không thì `NewTextResponse(content)`. Fantasy đổi `IsError` thành `ToolResultOutputContentError` (`fantasy@v0.45.1/agent.go:861`). `convertToToolResult` đặt `is_error=true` cho kiểu đó (`agent.go:2802`), giống đường của `view`.
3. **Ảnh/media giữ nguyên hành vi** (có comment tại chỗ). Hai lý do:
   - Brief yêu cầu giữ nguyên.
   - Đặt `IsError` cho response ảnh thì fantasy bỏ hẳn ảnh, chỉ gửi chữ (`agent.go:861-866`). Đó là đổi hành vi, chưa ai yêu cầu.

   Chữ đi kèm ảnh cũng chưa bị cắt (xem mục 7).
4. Lỗi giao thức (`err != nil` từ `RunTool`) vẫn đi qua `NewTextErrorResponse` như cũ.

## 4. Test mới: `internal/agent/tools/mcp-tools_test.go`

Helper `runMCPStub` dựng một MCP server go-sdk thật qua HTTP (`httptest` + `NewStreamableHTTPHandler`). Server được nối bằng `mcp.InitializeSingle` như lúc chạy thật. Sau đó helper lấy tool qua `GetMCPTools` và gọi `Tool.Run`, nên test đi trọn đường production từ go-sdk đến `ToolResponse`.

| Test | Khẳng định |
|---|---|
| `TestMCPTool_LargeTextIsCappedAndSpilled` | 5 MiB chữ → `len(Content) < MaxOutputLength+1024`. Dấu `full output: <tệp>` có mặt. Tệp nằm đúng `<data>/shell-output`. Nội dung tệp == toàn bộ 5 MiB. `IsError=false` |
| `TestMCPTool_SmallTextIsUnchanged` (đối chứng) | `"echo"` đi nguyên văn, `IsError=false`, không tạo `<data>/shell-output` |
| `TestMCPTool_IsErrorReachesToolResponse` | `isError=true` → `IsError=true`, `Content` giữ nguyên chữ lỗi |

Các test này cố ý KHÔNG dùng `t.Parallel()`. Registry MCP là toàn cục. Khi dọn, test gọi `mcp.Reinitialize` với config rỗng để gỡ hẳn server (`DisableSingle` vẫn để lại mục `disabled`).

**Sự cố đã gặp và đã sửa:** bản test đầu có `t.Parallel()` và dọn bằng `DisableSingle`. Lượt suite đầu sau khi sửa vì thế ra **2 đỏ mới, do chính mình gây ra**: `TestCrushInfo_EmptySectionsOmitted` và `TestCrushInfo_MinimalConfig` thấy `[mcp] p1b-big = connected…`. Đã đổi sang chạy tuần tự và dọn hẳn. Lượt suite thứ hai ra 0 đỏ mới (mục 5).

## 5. Bảng chân trị

### E2E (`exposure.sh`, binary thật, LLM giả)

| Ca | Trước (`crush-before`, `/home/luannt/.cache/v3ev/p1b-before2`) | Sau (`crush-after`, `/home/luannt/.cache/v3ev/p1b-after`) |
|---|---|---|
| T9: MCP 5 MiB, request kế tiếp | +5.327.624 B | **+31.302 B** |
| T9: MCP 5 MiB, `crush.db*` | +5.341.184 B | **+32.768 B** |
| T9: `tool_result` lưu | 5.242.880 ký tự | **30.625 ký tự**. Dấu `... [84071 lines truncated, full output: …/proj/.crush/shell-output/output-2459254105.log] ...`; tệp 5.242.880 B (đủ) |
| T9: đối chứng bash 5 MiB | +30.455 B / db +65.536 | +30.451 B / db +65.536 (không đổi) |
| T9 verdict | EXPOSED | **NOT_EXPOSED** |
| T19: MCP `isError=true` → `is_error` | `False` | **`True`** (`'V3-MCP-FAIL boom'`) |
| T19: đối chứng `view` tệp không có | `True` | `True` |
| T19 verdict | EXPOSED | **NOT_EXPOSED** |
| TC (đối chứng P1b): MCP nhỏ, thành công | `('mcp_v3stub_stub_echo', False, 'echo')`, spill 0 → OK | giống hệt → OK (`/home/luannt/.cache/v3ev/p1b-ctl-{before,after}`) |

TC chạy bằng một **bản sao** `exposure.sh` trong scratch, có thêm hàm `tc()` gọi `stub_echo`. Bản sao không vào patch: repo không có ca này và brief không giao sửa harness.

`crush-after` và `crush-final` (dựng lại sau mọi chỉnh sửa test và CHANGELOG) cùng sha256 `ab91a225…`, nên số E2E ứng với đúng mã cuối.

### Unit

| Trạng thái | LargeText | SmallText (đối chứng) | IsError |
|---|---|---|---|
| Trước sửa (ĐỎ) | FAIL `"5242906" is not less than "31024"` | PASS | FAIL `Should be true` |
| Sau sửa | PASS | PASS | PASS |
| Kiểm răng: `git apply -R` bản sửa, giữ test | FAIL (cùng thông điệp) | PASS | FAIL (cùng thông điệp) |
| Áp lại (sha256 diff `b787d881…` trùng trước/sau), có `-race` | PASS | PASS | PASS, 0 DATA RACE |

### Hồi quy

| Bộ | Kết quả |
|---|---|
| `go test -count=1 -p 2 -json ./...`, so theo TÊN với cây sạch | Trước: 4684 tên (4658 pass, 14 fail, 12 skip). Sau: 4687 tên (4661 pass, 14 fail, 12 skip). **Đỏ mới: 0. Tên mất: 0.** Tên thêm: 3 `TestMCPTool_*`, đều pass. 14 tên đỏ trùng hệt baseline: 5 `TestV3Exposure_*` cấp cao (T12, T15, C9_ToolsOutsideWorkingDir, T1, C9_ShellLeavesWorkspace) và subtest của chúng |
| `go test -race -count=1 -p 2 ./internal/agent/...` | 0 DATA RACE. FAIL chỉ ở `TestV3Exposure_T12`, `TestV3Exposure_T15` (`internal/agent`) và `TestV3Exposure_C9_ToolsOutsideWorkingDir` (`internal/agent/tools`), tức baseline. `hyper`, `prompt`, `tools/mcp`, `tools/provider` ok |
| `scripts/eval/faults.sh` | rc=0, `FAULTS: PASS` |
| `scripts/eval/qwen-only.sh` | rc=0, `QWEN-ONLY: PASS` |
| `scripts/eval/workspaces.sh` | rc=0, `WORKSPACES: PASS` |
| `gofmt -l`, `gofumpt -l`, `go vet` (2 package bị sửa) | sạch |

## 6. Lệnh tái lập

Mọi lệnh `go` chạy ở gốc worktree qua `flock /tmp/claude-1000/agent-go.lock`, với `GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off`, và chỉ khi `free -m` available ≥ 3000.

```bash
cd /home/luannt/application/opensource/agent-cli-p1b
G='flock /tmp/claude-1000/agent-go.lock env GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off'
# Unit (xanh); muốn thấy đỏ: git apply -R <patch chỉ 2 tệp .go> rồi chạy lại
$G go test -count=1 -p 2 -run 'TestMCPTool_' -v ./internal/agent/tools/
# E2E: binary, rồi T9/T19 vào thư mục MỚI
$G go build -p 2 -o /tmp/p1b-crush .
bash scripts/eval/exposure.sh /home/luannt/.cache/v3ev/<thư-mục-mới> /tmp/p1b-crush T9 T19
# Hồi quy
for s in faults qwen-only workspaces; do bash scripts/eval/$s.sh /home/luannt/.cache/v3ev/<mới>-$s /tmp/p1b-crush; done
$G go test -race -count=1 -p 2 ./internal/agent/...
$G go test -count=1 -p 2 -json ./... > after.json   # so theo tên với lượt chạy trên 7cba42ff
```

Bằng chứng thô của phiên này:

- `/tmp/claude-1000/-home-luannt-application/1ba1a44b-c74c-4a9d-b148-f58af7166a87/scratchpad/p1b/logs/`: `red-unit.txt`, `green-unit2.txt`, `teeth-red.txt`, `teeth-green.txt`, `base.json`, `after2.json`, `suite-compare2.txt`, `race2.txt`, `e2e-*.log`, `exp-*.log`. Scratch có thể bị dọn.
- Script so theo tên: `…/p1b/cmp.py`.
- Thư mục E2E: `/home/luannt/.cache/v3ev/p1b-*`.

## 7. Rủi ro

- **CPU khi cắt output lớn:** `TruncateOutput` duyệt độ rộng ANSI trên toàn chuỗi. Trong unit test, 5 MiB tốn khoảng 0,6 s (LargeText 0,92 s khi đã sửa, 0,30 s khi chưa) và 9,6 s dưới `-race`. Bash cũng chịu đúng chi phí này. MCP trả 50 MiB thì ước khoảng 6 s (**chưa đo**).
- **Bộ nhớ:** chuỗi đầy đủ vẫn nằm trong RAM (go-sdk đã giải mã xong trước khi tới mã này). Bản sửa không giảm đỉnh bộ nhớ, chỉ giảm phần đi vào context và DB.
- **Spill MCP giữ ĐỦ, spill bash chỉ giữ đầu/đuôi:** bash có `cappedBuffer` (`shell/background.go:34-35`, 256 KiB đầu + 1 MiB đuôi), nên tệp spill bash ở T9 chỉ có 1.310.763 B. Tệp spill MCP là 5.242.880 B. Thư mục spill có trần 256 MiB (`spillDirLimit`), xoá tệp cũ trước.
- **Spill hỏng âm thầm:** nếu `SpillOutput` lỗi (vd đĩa đầy), dấu chỉ còn `[N lines truncated]`, không có đường dẫn. Phần giữa mất, chỉ ghi `slog.Debug`. Bash cũng vậy.
- **`isError` đổi cách model thấy kết quả:** chữ lỗi giờ đi dạng tool error. Có provider hiển thị khác cho tool error so với text. Đây là hành vi đúng như tool nội bộ.

## 8. Chưa kiểm / ngoài phạm vi (mới nghi, kèm cách kiểm)

- **`read_mcp_resource` vẫn không có trần** (`read_mcp_resource.go:79-100` nối toàn bộ `Text`/`Blob`). Cùng lớp lỗi với T9. Brief bảo giữ resource nguyên. Cách kiểm: stub có resource 5 MiB, đo request kế tiếp như T9. Cách sửa một dòng là `TruncateOutput` ở dòng trả về.
- **Ảnh/media:** chữ đi kèm (`response.Content`) và `Data` không có trần. Ảnh trả kèm `isError=true` vẫn lưu `is_error=false`. Cách kiểm: stub trả `ImageContent` + 5 MiB chữ.
- `-race` chỉ chạy `./internal/agent/...` sau khi sửa, không chạy trên cây sạch. Đối chiếu dùng danh sách đỏ của suite không-race.
- Không chạy chế độ server (`--host`) cho T9/T19. `exposure.sh` dùng chế độ local; đường mã `Tool.Run` là chung.
- Docs: `grep -n MCP docs/agent-cli-local-setup.md README.md` không có câu nào về giới hạn output MCP, nên không sửa docs. CHANGELOG thêm một mục ở `[Unreleased]` → `### Sửa lỗi` (một gạch đầu dòng, xuống dòng theo khổ của tệp).
- Không dùng LLM thật, không đụng `~/.config/agent-cli`, không commit/push.

## Sau soát (điều phối, 19/10)

- Soát độc lập: **KET-LUAN: DAT**. Điều phối chạy lại trên binary tự build: T9, T19 NOT_EXPOSED (`~/.cache/v3ev/p1b-mine-r1.out`).
- Điều phối sửa 3 mục người soát yêu cầu: CHANGELOG thêm "Chưa sửa:"; dời `t.Cleanup(Reinitialize)` lên trước
  `InitializeSingle`; comment ảnh/media theo dạng `shortcut:`. Chạy lại gói `tools`: chỉ `TestV3Exposure_C9_*` (vốn đỏ)
  đỏ; 3 test MCP mới xanh; `TestCrushInfo*` xanh.
- Commit `9c08f3e8` trên nhánh `v3-poc-p1b-mcpresult`; patch cuối thay bản cũ cùng tên.
- Việc sau: `read_mcp_resource` chưa có trần; chữ kèm ảnh chưa có trần; ảnh kèm `isError` lưu `is_error=false`.
