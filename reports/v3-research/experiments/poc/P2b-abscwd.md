# P2b: `-c/--cwd` tương đối đặt workspace và crush.db sai chỗ (T2)

Worktree `/home/luannt/application/opensource/agent-cli-p2b`, nhánh `v3-poc-p2b-abscwd`, gốc `v2-final` (`7cba42ff`). Chưa commit, chưa push.
**Bản hiện hành là Vòng 2 (mục 8).** Mục 1–7 là Vòng 1; chỗ nào mục 8 nói khác thì mục 8 thắng.
Patch `P2b-abscwd.patch`: 6 tệp, +119/−4 (`git apply --stat`). Patch KHÔNG gồm 6 tệp phơi nhiễm `??` (5 `*_v3exposure_test.go` và `scripts/eval/exposure.sh`).
`sha256sum -c P2b-abscwd.patch.sha256` → `39d2579ecd2d8bb45f9aec3ecfb9cc914943cbca4e760fc3f77d9f3ce0feb3c5` (Vòng 1 là `79901fd5…`, đã bị thay).
Đã chạy `git apply --check` trên bản xuất sạch của `7cba42ff` (`git archive`): rc=0. Áp xong, cây khớp worktree (`diff -rq`, trừ 6 tệp phơi nhiễm).

## 1. Gốc lỗi (đọc lại ở `7cba42ff` trước khi sửa)

- `internal/cmd/root.go:1238` `ResolveCwd`: `os.Chdir(cwd)` rồi trả NGUYÊN chuỗi `--cwd`, có thể là đường tương đối. Mọi lệnh dùng hàm này đều nhận chuỗi đó: TUI/`run`/`send` qua `setupLocalWorkspace` (:423) và `connectToServer` (:556), cùng `models`, `stats`, `session`, `dirs`, `loop`.
- Chế độ server: `connectToServer` gọi `ensureServer` (:547) trước `ResolveCwd` (:556). `startDetachedServer` (:1022–1047) không đặt `c.Dir`, nên server thừa hưởng cwd của client đã dựng nó. Server ghép `Path=".."` theo cwd CỦA NÓ ở hai chỗ: `resolveWorkspaceKey` (`backend.go:455`) và `config.Init(args.Path…)` (`backend.go:515`).
- Data dir bị cộng gốc hai lần (`internal/config/load.go:621` rồi `:624`): `Join(workingDir, ".crush")`, sau đó `SmartJoin(workingDir, …)` ghép thêm lần nữa khi chuỗi còn tương đối. Nhánh nạp lại workspace config (`load.go:76`) đưa giá trị đã ghép vào làm `dataDir`, nên với gốc tương đối nó bị ghép lần ba.
- Chế độ local, `-c foo`: chdir vào `r/foo` rồi `config.Init("foo")` ra `foo/foo/.crush`, mà đường này lại tính từ `r/foo`, nên DB nằm ở `r/foo/foo/foo/.crush`.
- `logs` (`logs.go:30,56`) đọc nguyên chuỗi cờ `--cwd`, không chdir.

## 2. Sửa (2 chỗ mã)

| Tệp | Đổi |
|---|---|
| `internal/cmd/root.go` `ResolveCwd` | `filepath.Abs(cwd)` theo cwd của lệnh, TRƯỚC khi chdir. Sau đó chdir vào và trả đường tuyệt đối đó. Không thêm resolver mới. |
| `internal/config/load.go` `setDefaults` | Nhánh mặc định gán `defaultDataDirectory` thay cho `Join(workingDir, …)`, để `SmartJoin` ở dòng sau là lần ghép gốc DUY NHẤT. Sau đó `filepath.Abs`, nên lần nạp lại (`load.go:76`) không ghép thêm. Lỗi `Abs` (chỉ xảy ra khi `Getwd` hỏng) thì giữ giá trị `Clean` như cũ. |

Cố ý KHÔNG đổi:

- **`-D/--data-dir`:** không cùng lớp lỗi. Upstream định nghĩa `-D` tương đối là tính theo working dir (test `resolves relative flag data directory from working directory`). Khi working dir đã tuyệt đối, server ghép `-D` theo `Path` tuyệt đối, không phụ thuộc cwd của server. Đo được: local `-c foo -D x` → `r/foo/x`, qua server → `r/foo/x`, hai đường cho cùng kết quả (bảng mục 4). Chuẩn hoá `-D` theo cwd của shell sẽ làm đổi nghĩa khi có `-c`.
- **Thứ tự `ensureServer`/`ResolveCwd`, `c.Dir` của server con, `logs.go`:** khi `Path` đã tuyệt đối thì các chỗ này không còn ảnh hưởng tới kết quả. Sửa chúng là phạm vi thêm.

## 3. Test mới

| Test | Kiểm |
|---|---|
| `internal/cmd/cwd_test.go` `TestResolveCwdMakesRelativeFlagAbsolute` (MỚI) | Đứng ở `repo/internal`, `-c ..` phải trả `<root>/repo` (tuyệt đối) và vẫn chdir. Ca con `absolute` là đối chứng. |
| `internal/config/load_test.go` `TestConfig_setDefaults/relative_working_directory_gets_the_data_directory_once` (MỚI) | Đứng ở `root` với gốc `"foo"`: data dir phải trỏ tới `root/foo/.crush`. Gọi lại `setDefaults("foo", <giá trị vừa ra>)` (y như nhánh nạp lại của `Load`) vẫn phải ra `root/foo/.crush`. |

ĐỎ trước khi sửa (đã chạy):
- `ResolveCwd` trả `".."`.
- `setDefaults` ra `…/foo/foo/.crush`.
- Ca đối chứng `absolute` xanh.

## 4. Bằng chứng e2e (binary thật, LLM giả, HOME cô lập)

Các binary nằm ở `~/.cache/v3ev/p2b-bin/`, đều build bằng `CGO_ENABLED=0 GOEXPERIMENT=greenteagc`:
- `agent-cli-before`: `7cba42ff` cộng các tệp `??`.
- `agent-cli-after`: bản sửa. Mã Go không đổi sau lần build này; đã đối chiếu `git diff -- *.go` với patch đã lưu.
- `agent-cli-teeth-root` và `agent-cli-teeth-load`: mỗi binary hoàn một nửa bản sửa (mục 6).

**T2** (`bash scripts/eval/exposure.sh <dir-mới> <bin> T2`):

| biến thể | before | after | chỉ sửa config (teeth-root) | chỉ sửa ResolveCwd (teeth-load) |
|---|---|---|---|---|
| a: server chung `--host` dựng từ A | WRONG `path='..'`, `data_dir='../../.crush'` → `w/.crush/crush.db` | OK `w/s/repo/.crush` | WRONG `w/s/.crush` | OK |
| b1: client tự dựng server | WRONG → `w/s/.crush` (cộng hai lần) | OK | **OK** | OK |
| b2: repo đã có `.crush` | OK | OK | OK | OK |
| c: qua launcher thật | WRONG → `w/.crush` | OK | WRONG `w/s/.crush` | OK |
| ctl: `-c` tuyệt đối | OK | OK | OK | OK |
| **Kết luận** | **EXPOSED** (a b1 c) | **NOT_EXPOSED** | EXPOSED (a c) | NOT_EXPOSED |

Sau khi sửa, workspace mà server lưu là `path='<ca>/w/s/repo'` và `data_dir='<ca>/w/s/repo/.crush'`, cả hai tuyệt đối. Thư mục chứa bằng chứng: `~/.cache/v3ev/p2b-t2-{before,after,teethroot,teethload}/T2/` (`variants.txt`, `*-workspaces.json`).

**Local và `-D`** (`bash <scratch>/p2b/local-cfoo.sh <dir-mới> <bin> <repo>`; script nằm ở scratchpad của phiên, không vào patch):

| ca | before | after |
|---|---|---|
| L1 local `cd r; -c foo run` | `r/foo/foo/foo/.crush/crush.db`, projects `data_dir=foo/foo/.crush` | `r/foo/.crush/crush.db`, `data_dir=<ca>/r/foo/.crush` |
| L2 đối chứng `-c <tuyệt đối>` | `r/foo/.crush` | `r/foo/.crush` (không đổi) |
| L3 đối chứng không `-c` (`cd r/foo`) | `r/foo/.crush` | `r/foo/.crush` (không đổi) |
| L5 local `-c foo -D x` | `r/foo/foo/x` | `r/foo/x` |
| S5 server `-c foo -D x` | `r/foo/x` (đúng nhờ trùng hợp: server được dựng từ `r`) | `r/foo/x` |

## 5. Hồi quy (đều chạy với `agent-cli-after`)

- `scripts/eval/faults.sh` → `FAULTS: PASS`.
- `scripts/eval/qwen-only.sh` → `QWEN-ONLY: PASS`.
- `scripts/eval/workspaces.sh` → `WORKSPACES: PASS`.
- `AGENT_CLI_BIN=… scripts/test-agent-cli-run-json.sh` → `PASS`.
- `scripts/test-agent-cli-wrapper.sh` → `PASS`. Script tự cài vào `mktemp -d` với binary giả, nên không chạm tới mã đã sửa.
- Log của cả năm script ở `~/.cache/v3ev/p2b-reg/`.
- `CGO_ENABLED=1 go test -race -count=1 -p 2 ./internal/cmd/... ./internal/config/... ./internal/backend/...`: 0 `DATA RACE`. `cmd`, `cmd/clientserverrace` và `config` ok. `backend` FAIL chỉ vì `TestV3Exposure_T1_IdentityResolversAgree` (vốn đỏ). Log: `~/.cache/v3ev/p2b-race.log`.
- Toàn kho theo TÊN (`go test -json -count=1 -p 2 ./...`), so worktree với bản xuất `7cba42ff` cộng 6 tệp phơi nhiễm:
  - Base: 4684 test, 85 package, 4 package FAIL.
  - After: 4688 test, 85 package, 4 package FAIL.
  - **0 test mất, 0 test đổi trạng thái, 4 test mới đều pass.**
  - Đỏ ở cả hai bên đúng 5 test `TestV3Exposure_*`: T12, T15, C9 tools, T1, C9 shell.
  - **T1 không đổi:** output của T1 base/after giống hệt sau khi che đường dẫn và thời gian. Ca con `relative` pass ở cả hai. Các ca đỏ đều là symlink, khác lớp lỗi này.
  - Kết quả thô: `~/.cache/v3ev/p2b-suite-{base,after}.json`.
- `gofmt -l` (GOROOT) trên 4 tệp Go: rỗng. `go vet ./internal/cmd ./internal/config`: ok.

## 6. Kiểm răng (hoàn bản sửa → đỏ lại)

Hoàn từng nửa bằng `git apply -R`, rồi áp lại. Sha256 của `git diff` toàn cây trước và sau khi kiểm răng trùng nhau: `ae4a6ef0…`.

- **Hoàn `root.go`:** `TestResolveCwdMakesRelativeFlagAbsolute/relative` FAIL. T2 ra EXPOSED (a, c). L1 ra `r/foo/foo/.crush`. L5 ra `r/foo/foo/x`.
- **Hoàn `load.go`:** `TestConfig_setDefaults/relative_working_directory_gets_the_data_directory_once` FAIL. E2e vẫn xanh (T2 NOT_EXPOSED, L1 đúng), vì sau khi sửa phía client thì không còn đường tương đối nào tới config qua `ResolveCwd`.
  - Răng e2e riêng của phần config là cột teeth-root ở mục 4: chỉ sửa config thì b1 hết cộng hai lần (`w/s/.crush` → `w/s/repo/.crush`).
  - Như vậy phần config là lớp chặn thứ hai. Nó có tác dụng khi đường tương đối tới config từ chỗ khác: `logs -c` (đọc nguyên cờ), hoặc một client không đi qua `ResolveCwd`.

## 7. Giả định, giới hạn, chưa kiểm

- **Nghĩa của `..` qua symlink:** `filepath.Abs` tính theo kiểu chữ (logical, như `cd ..` của shell và như launcher, vốn dùng `cd -- "$cwd"`). Mã cũ chdir theo kernel (physical). Khi `$PWD` đi qua symlink, `-c ..` nay vào cha theo `$PWD`. Chưa có test riêng cho ca này. Khoá workspace trên server vẫn `EvalSymlinks`.
- **Server vẫn nhận `Path` tương đối từ client lạ** (client cũ hoặc gọi API trực tiếp) mà không báo lỗi. Bản sửa config giúp data dir không cộng hai lần, nhưng workspace vẫn tính theo cwd của server. Đề xuất (chưa làm): `CreateWorkspace` từ chối `Path` không tuyệt đối. Đó là lỗi to tiếng thay cho hỏng âm thầm.
- **Câu CHANGELOG ban đầu sai quy nhân, đã sửa:** câu đầu viết `logs -c foo tìm đúng log` là nhờ sửa config. Kiểm răng cho thấy điều đó đúng nhờ DB nằm đúng chỗ (`LookupClosestBounded` trả đường tuyệt đối), không nhờ config, nên đã bỏ câu đó.
- **Docs:** chỉ thêm chú thích `(-c tương đối tính theo thư mục hiện tại)` ở `docs/agent-cli-local-setup.md`, mục Sử dụng. Không câu nào khác trong docs nói sai về `-c`/`--cwd`.
- **Phạm vi chạy:** không gọi LLM thật, không đụng `~/.config/agent-cli` thật, không commit.

## 8. Vòng 2 (sau lượt soát độc lập chấm KHÔNG ĐẠT)

### 8.1 Lỗi Vòng 1 bỏ sót: `loop` gọi `ResolveCwd` hai lần

- **Đường gọi:** lần 1 ở `internal/cmd/loop.go:87` (`p.Cwd`, nơi chạy verify và lấy fingerprint). Lần 2 ở `loop.go:388` (`connectToServer` → `root.go:556`) hoặc ở `loop.go:421` (`setupLocalWorkspace` → `root.go:423`).
- **Vì sao sai:** đứng ở `repo/internal` gọi `-c ..`. Lần 1 ra `repo` và chdir vào đó. Lần 2 tính `Abs("..")` từ `repo`, ra thư mục CHA. Kết quả: verify chạy trong `repo` nhưng agent, vốn tự duyệt mọi tool, làm việc ở thư mục cha. Sai âm thầm.
- **Lỗi có từ trước:** binary gốc `7cba42ff` chdir hai lần nên hỏng tương tự. Ở chế độ local còn ra `GREEN` cho một thư mục cách repo hai cấp (bảng 8.3).
- **Soát lại toàn bộ đường gọi** (`grep -n "ResolveCwd\|setupWorkspace(\|connectToServer(\|setupLocalWorkspace(\|sessionSetup(" internal/cmd/*.go`): chỉ `loop` gọi hai lần. `run`, `send`, `jobs`, `logout` và TUI gọi một lần qua `connectToServer`/`setupLocalWorkspace`; `session`, `stats`, `models`, `dirs` gọi một lần. `logs` đọc nguyên cờ, không qua `ResolveCwd`. `routine run` chỉ xét `Changed("cwd")` (`rejectCwdFlag`). Lệnh `exec` duy nhất (`home.go:154`) không mang `-c`.

### 8.2 Sửa

- **`ResolveCwd`:** sau khi chdir thành công thì ghi đường tuyệt đối trở lại cờ (`_ = cmd.Flags().Set("cwd", abs)`), kèm một dòng chú thích lý do. Lần gọi sau trả đúng đường đó.
  - `Changed` vốn đã true khi có `-c`; không có `-c` thì hàm đi nhánh `Getwd` và không `Set`. Vì vậy `rejectCwdFlag` không đổi hành vi (`TestRejectCwdFlag` xanh).
- **CHANGELOG:**
  - Thêm ý `loop -c ..`.
  - Câu "kể cả khi server nhận đường tương đối" đổi cho đúng phạm vi: khi đó workspace và data dir VẪN tính theo thư mục của server, chỉ không còn bị ghép gốc hai lần.
  - Thêm **Lưu ý nâng cấp**: session tạo bằng `-c` tương đối ở chỗ sai cũ vẫn còn trên đĩa nhưng không được mở, không có cảnh báo; mở lại bằng `-D <đường cũ>`.
  - Đã chạy thử lưu ý này: binary sau sửa, `-c foo session list --json` ra `[]`; thêm `-D foo/foo/.crush` thì ra đúng session `bd1b359e…` do binary gốc tạo ở `r/foo/foo/foo/.crush`.
- **Không làm trong POC này** (theo điều phối): `CreateWorkspace` từ chối `Path` không tuyệt đối. Việc này để PR riêng.

### 8.3 Test và bằng chứng (đều từ lệnh vừa chạy)

Hai test đổi/thêm trong `internal/cmd/cwd_test.go`:

- **`TestResolveCwdMakesRelativeFlagAbsolute`:** thêm bước gọi `ResolveCwd` LẦN HAI, phải vẫn ra `repo`.
  - Trước khi sửa: ca con `relative` ĐỎ, `expected …/repo, actual …/001` (thư mục cha), thông điệp `a second call must not resolve the flag again`.
  - Sau khi sửa: xanh.
- **`TestResolveCwdFollowsLogicalPwd` (MỚI, chỉ ghim):** `t.Chdir(link)` (link → `real/repo/internal`, PWD=link) rồi `-c ..` phải ra cha theo `$PWD`, không phải `real/repo`. Test này xanh từ đầu, không phải test đỏ→xanh.

E2E mới `scratchpad/p2b/loop-cwd.sh <dir-mới> <bin> <repo>`:
- LLM giả gọi bash `pwd > agent-pwd.txt`, verify chạy `test -f agent-pwd.txt`.
- Đo bốn thứ: chỗ tệp được ghi, nội dung `pwd`, dòng `Working directory` trong system prompt mà LLM giả nhận, và `crush.db` chứa session.
- Bốn ca: `LS` qua server, `LL` local (đều `cd repo/internal; -c ..`), và đối chứng `CS`/`CL` với `-c <tuyệt đối>`.
- Bằng chứng: `~/.cache/v3ev/p2b-loop-{before,r1,r2,teethset}/rows.txt`.

| ca | gốc `7cba42ff` | Vòng 1 | Vòng 2 | Vòng 2 bỏ `Set` (răng) |
|---|---|---|---|---|
| LS server `-c ..` | WRONG: MAX_REPAIRS, agent ở `w/`, DB `.crush` ở gốc ca | WRONG: STALLED, agent và prompt ở `w/`, DB `w/.crush` | **OK**: GREEN, agent, prompt và DB đều ở `w/repo` | WRONG (như Vòng 1) |
| LL local `-c ..` | WRONG: **GREEN giả**, agent ở gốc ca (cách repo 2 cấp) | WRONG: STALLED, `w/` | **OK** | WRONG |
| CS / CL `-c` tuyệt đối | OK | OK | OK | OK |
| Kết luận | EXPOSED | EXPOSED | **NOT_EXPOSED** | EXPOSED |

Binary vòng 2 là `~/.cache/v3ev/p2b-bin/agent-cli-after2`. Mã Go không đổi sau lần build này (đối chiếu `git diff -- *.go` với bản lưu). Kết quả chạy lại trên binary này:

- **T2:** `exposure.sh ~/.cache/v3ev/p2b-t2-r2 … T2` ra NOT_EXPOSED; a, b1, b2, c, ctl đều OK.
- **Ca local** (`local-cfoo.sh`, `~/.cache/v3ev/p2b-local-r2`): giống Vòng 1 sau sửa. L1 ở `r/foo/.crush`, L2/L3 không đổi, L5/S5 ở `r/foo/x`.
- **Script hồi quy:** `faults.sh` FAULTS: PASS; `qwen-only.sh` QWEN-ONLY: PASS; `workspaces.sh` WORKSPACES: PASS; `test-agent-cli-run-json.sh` PASS; `test-agent-cli-loop.sh` PASS (mới so với Vòng 1); `test-agent-cli-wrapper.sh` PASS. Log ở `~/.cache/v3ev/p2b-reg2/`.
- **`-race` cho cmd, config, backend:** 0 `DATA RACE`. `cmd`, `cmd/clientserverrace`, `config` ok. `backend` FAIL chỉ vì `TestV3Exposure_T1_IdentityResolversAgree`, vốn đỏ (`~/.cache/v3ev/p2b-race2.log`).
- **Suite theo TÊN** (`~/.cache/v3ev/p2b-suite-after2.json`) so với base `7cba42ff` cộng 6 tệp phơi nhiễm:
  - Base dùng lại lượt chạy Vòng 1 cùng phiên. Cây base không đổi.
  - After: 4689 test. **0 mất, 0 đổi trạng thái, 5 test mới đều pass.** Đỏ đúng 5 `TestV3Exposure_*` như base.
  - Output của T1 vẫn giống hệt base.
- **Format và vet:** `gofmt -l` rỗng; `go vet ./internal/cmd ./internal/config` ok.

### 8.4 Kiểm răng Vòng 2

Bỏ hai dòng chú thích và lệnh `Set`:
- `TestResolveCwdMakesRelativeFlagAbsolute/relative` FAIL (`a second call must not resolve the flag again`).
- Binary `agent-cli-teeth-set` cho `loop-cwd.sh` ra EXPOSED (LS, LL đều STALLED, agent ở `w/`).

Sau đó chép lại tệp gốc. Sha256 của `git diff` toàn cây trước và sau trùng nhau (`bb183b1b…`). Răng của Vòng 1 (mục 6) vẫn đứng, vì hai chỗ sửa đó không đổi.

### 8.5 Còn lại

- `CreateWorkspace` vẫn nhận `Path` tương đối mà không báo lỗi. Việc này dành cho PR riêng.
- Ca symlink cộng `..` giờ đã có test ghim. Hành vi đó vẫn khác bản gốc (logical thay vì physical) và đã ghi ở mục 7.
- Chưa chạy `task lint`.
