# P2c: một dự án một data dir và một workspace (T1, T3, T1M)

Worktree `/home/luannt/application/opensource/agent-cli-p2c`, nhánh `v3-poc-p2c-datadir`, gốc `v3-candidate` (`642fe075`). Chưa commit, chưa push, không stash.
**Bản hiện hành là Vòng 4 (mục 11).** Mục 1–8 là Vòng 1, mục 9 là Vòng 2, mục 10 là Vòng 3 (soát vòng 3: DAT, còn 2 mục NÊN SỬA); chỗ nào mục sau nói khác thì mục sau thắng.
Vòng 4: 29 tệp theo dõi, +733/−274 (`git diff --stat`), cộng tệp mới `internal/cmd/datadir_test.go`.
- `git diff | sha256sum` = `1d22627391065319fd35e6fbcbef2e750dfd40c55e99318353cd97e4987981d1`.
- `sha256sum internal/cmd/datadir_test.go` = `41225a0c89670a971f2414db2eb62820d72e35f5bab93cb02b32ec193924d9c1` (không đổi so với vòng 3).
Hai tệp phơi nhiễm chép tạm đã xoá; `git status`: 29 tệp `M` và `?? internal/cmd/datadir_test.go`.
Bằng chứng thô: `~/.cache/v3ev/p2c-*` (exposure, bảng chân trị, mutant, hồi quy); script đo và log: `~/.cache/v3ev/p2c-scripts/`.
Các binary ở `scratchpad/p2c/bin/` của phiên (build `CGO_ENABLED=0 GOEXPERIMENT=greenteagc`): `agent-cli-before` (`642fe075`), `agent-cli-after` (bản sửa), `agent-cli-teeth-key`, `agent-cli-teeth-datadir` (mục 5).

## 1. Gốc lỗi (đọc ở `642fe075`)

- **Data dir theo `.crush` gần nhất** (`internal/config/load.go:622`, `setDefaults`): `LookupClosestBounded(workingDir, projectBoundary, ".crush")`. Không thấy thì `workingDir/.crush`.
  - Mở `repo/sub` khi `repo` chưa có `.crush` thì tạo `sub/.crush`. Mở gốc sau đó tạo `repo/.crush`. Từ đó `sub` luôn thấy `sub/.crush` gần hơn (T3).
- **Đi lên theo chữ** (`internal/fsext/lookup.go`, `traverseUpBounded`): đi theo các cha của `Abs(dir)`. Biên `stopDir` lại so bằng đường đã giải symlink.
  - Symlink `T/link-sub → repo/sub`: cha theo chữ là `T`, không bao giờ gặp biên `repo`. Vòng lặp đi tới `/`: ra `T/.crush` (ngoài repo), hoặc không thấy gì thì ra `link/.crush` = `sub/.crush` (DB thứ hai). Cùng lỗi này làm `lookupConfigs` nạp `crush.json` ngoài dự án qua symlink.
- **Nhánh qua symlink** (`internal/gitutil/branch.go`, `findGitDir`): cũng đi lên theo chữ, nên ra `""`.
- **Khoá workspace theo đường** (`internal/backend/backend.go`, `resolveWorkspaceKey`): `EvalSymlinks(path)`. `repo` và `repo/sub` là hai khoá, tức hai workspace và hai coordinator trên một `crush.db`. Trạng thái bận nằm trong coordinator nên hai bên không thấy nhau (T1M).
- **Home gộp nhiều data dir** (`internal/cmd/home.go` `listHomeSessions` cùng `projects.Project.Locations`/`DataDirs`): cơ chế bù cho việc mỗi thư mục con có DB riêng.

## 2. Sửa

| Tệp | Đổi |
|---|---|
| `internal/fsext/lookup.go` | `traverseUpBounded` đi lên từ đường VẬT LÝ (`canonicalize(cwd)`), một dòng. |
| `internal/gitutil/branch.go` | `findGitDir` `EvalSymlinks` trước khi đi lên. |
| `internal/config/load.go` | Data dir mặc định = `projectBoundary(workingDir)/.crush`, bỏ việc tìm `.crush` gần nhất. `projectBoundary` ngoài git trả đường vật lý. Thêm `StrayDataDirs(workingDir, dataDir)`: các `.crush` giữa `workingDir` và gốc (không tính gốc), trừ data dir đang dùng. |
| `internal/backend/backend.go` | `resolveWorkspaceKey` = gốc dự án qua `projects.NormalizeWorkspace` (git toplevel vật lý, đúng hàm launcher/Home dùng). `CreateWorkspace` dựng workspace ở gốc đó (`config.Init(key…)`, `Path: key`). `-D` tương đối ghép theo `args.Path` trước, để giữ nghĩa "tính theo thư mục mở" như local. |
| `internal/cmd/root.go` | `warnStrayDataDirs`: in cảnh báo ra stderr và `slog.Warn` cho mỗi `.crush` cũ, kèm lệnh mở lại. Gọi ở `setupLocalWorkspace` và `connectToServer`. |
| `internal/projects/projects.go` | XOÁ `Location`, `Locations`, `maxLocations`, `DataDirs()`. `Register` giữ đúng một `DataDir` cho mỗi mục. |
| `internal/ui/home/home.go`, `internal/cmd/home.go` | XOÁ `home.Location`; `Workspace.DataDir` thay `Locations`. `listHomeSessions` đọc đúng một data dir. |
| `CHANGELOG.md`, `docs/agent-cli-local-setup.md` | Mục Sửa lỗi kèm **Lưu ý nâng cấp**; sửa đoạn Home từng nói "session ở mọi DB đều hiện". |

**Luật POC: cơ chế cũ bị thay đã xoá, không để hai đường song song.**
- Tìm `.crush` gần nhất cho data dir → thay bằng gốc dự án.
- Khoá workspace theo đường → khoá theo gốc dự án, sửa tại chỗ (cùng một hàm).
- Home gộp DB theo từng thư mục con (`Locations`) → một data dir.

Không xoá: `fsext.LookupClosestBounded` (hàm thư viện của upstream, commit `79b2d619`). Sau bản sửa nó không còn caller ngoài test. Giữ lại để merge upstream rẻ; điều phối quyết.

**Thứ tự gọi đã soát:**
- `grep resolveWorkspaceKey`: chỉ `CreateWorkspace` (cùng T1 test).
- `grep projects.Register`: `root.go:448`, `:522`, `home.go:259`. `Register` không đổi chữ ký.
- `grep 'DataDirs()\|Locations'`: chỉ Home và test.

## 3. Test

**Mới:**

| Test | Kiểm |
|---|---|
| `fsext` `TestLookupBounded/walks_the_physical_path_of_a_symlink_into_the_project` | Symlink ở thư mục khác trỏ vào `boundary/subdir`. Chỉ ra `boundary/target.txt`, không ra tệp ở thư mục chứa symlink. |
| `gitutil` `TestCurrentBranch/reads_the_branch_through_a_symlink_into_a_subdirectory` | Nhánh đọc qua symlink vào thư mục con = nhánh thật. |
| `config` `TestStrayDataDirs` | `setDefaults(link-vào-a/b)` ra `repo/.crush`. `StrayDataDirs` báo `a/b/.crush` từ `a/b` và từ symlink. Từ gốc không báo. `-D` trỏ đúng `.crush` cũ thì không báo. |
| `backend` `TestPathDedupe_SubdirSharesProjectWorkspace` | `CreateWorkspace(sub)` rồi `(repo)`: cùng ID, `Path=repo`, data dir `repo/.crush`, không tạo `sub/.crush`. Thêm ca `-D x` từ `sub` ra `sub/x`. |
| `projects` `TestRegisterKeepsOneDataDirPerWorkspace` | Registry cũ có `locations` vẫn đọc được. `Register(sub, repo/.crush)` giữ một data dir, và lần ghi sau bỏ trường `locations`. |

**Đổi (giải thích vì người soát sẽ hỏi):**
- `config` `TestConfig_setDefaults/does_not_climb_out_of_git_worktree_to_find_.crush` (upstream): assert cuối từng là `pkg/.crush`. Đó chính là hành vi T3 mà D3 đổi. Nay assert `worktree/.crush`. Assert chính của test ("không dùng `.crush` phía trên worktree") giữ nguyên.
- `cmd/home_test.go`: 3 chỗ `Locations: …` thành `DataDir: …`; `TestHomeSourceAdd` đọc `ws.DataDir`.

**Xoá** (test của cơ chế đã xoá; đối chiếu theo tên ở mục 7):
- `projects` `TestSubdirThenRootKeepsBothDataDirs`, `TestLocationsCappedOldestDropped`, `TestOldFormatEntryKeepsItsDataDirOnNextRegister`.
- `cmd` `TestHomeWorkspaceLocations`, `TestHomeSessionsFromEveryDataDir`.
- Phần "đọc DB qua server sống thì không mở DB" của test cuối vẫn được `TestHomeSessionsFromLiveServer` phủ (`require.NoDirExists(never)`).

Test phơi nhiễm không bị sửa.

## 4. Đỏ → xanh

**T1 (unit):** `flock … go test -p 2 -count=1 -v -run 'TestV3Exposure_T1_' ./internal/backend/`

| ca | `642fe075` | bản sửa |
|---|---|---|
| symlink-into-sub | data `T/.crush` (ngoài `T/real/repo`), nhánh `""` → LỆCH spelling_invariant, data_dir_inside_project, branch_equals_physical | data `T/real/repo/.crush`, nhánh `"main"` → ok |
| symlink-into-sub-no-parent-crush | data `U/link-sub/.crush` (= `sub/.crush`, DB thứ hai), nhánh `""` → LỆCH spelling_invariant, branch_equals_physical | `T/real/repo/.crush`, `"main"` → ok |
| 7 hàng đối chứng (repo-root, symlink-to-root, monorepo-sub, linked-worktree, trailing-slash, relative, outside-git) | ok | ok. Cột K của `sub` đổi từ `T/real/repo/sub` thành `T/real/repo`; data dir `symlink-to-root` đổi từ `T/link-root/.crush` thành `T/real/repo/.crush` (cùng thư mục vật lý) |
| Kết luận | `--- FAIL` (5 subtest) | `--- PASS: TestV3Exposure_T1_IdentityResolversAgree` |

**T1M, T3 (binary thật, LLM giả):** `bash scripts/eval/exposure.sh <dir-mới> <bin> T1M T3`

```
# agent-cli-before → ~/.cache/v3ev/p2c-before
T1M: EXPOSED — harm: second prompt sent 1.6s after the long turn started (long turn lasts 8s) -> RAN CONCURRENTLY; ps rows for S (path,busy)=[('repo', True), ('repo/sub', False)]; workspaces=['repo', 'repo/sub']; control: second prompt sent 8.0s after … -> queued; ps rows for S (path,busy)=[('repo', True)]; workspaces=['repo']; crush.db under repo=['repo/.crush/crush.db']
T3: EXPOSED — local|EXPOSED|order1 sub-first db=sub/.crush/crush.db root db=.crush/crush.db sub-again db=sub/.crush/crush.db | order2 (control) root db=.crush/crush.db sub db=.crush/crush.db | crush.db count repo1=2 repo2=1 | repo1 root sees 1/3, repo1/sub sees 2/3 server|EXPOSED|(giống local)
# agent-cli-after → ~/.cache/v3ev/p2c-after
T1M: NOT_EXPOSED — harm: second prompt sent 8.0s after the long turn started (long turn lasts 8s) -> queued; ps rows for S (path,busy)=[('repo', True)]; workspaces=['repo']; control: … -> queued; …; crush.db under repo=['repo/.crush/crush.db']
T3: NOT_EXPOSED — local|NOT_EXPOSED|order1 sub-first db=.crush/crush.db root db=.crush/crush.db sub-again db=.crush/crush.db | order2 (control) root db=.crush/crush.db sub db=.crush/crush.db | crush.db count repo1=1 repo2=1 | repo1 root sees 3/3, repo1/sub sees 3/3 server|NOT_EXPOSED|(giống local)
```

Đối chứng không đổi: T1M control xếp hàng ở cả hai binary; T3 order2 ra 1 DB ở cả hai.

## 5. Kiểm răng (mỗi cơ chế một mutant; khôi phục bằng bản sao tệp, không dùng git)

Lệnh: `bash ~/.cache/v3ev/p2c-scripts/mutants.sh`, log ở `mutants.log`. Sha256 của `git diff` trước và sau khi chạy: cùng `5086786d…` (bản lúc đó, trước khi sửa chú thích và CHANGELOG).

| Mutant | Test đỏ (nguyên văn rút gọn) |
|---|---|
| M1 fsext bỏ `canonicalize` | `TestLookupBounded` expected `…/001/target.txt`, actual `…/002/target.txt`; `TestStrayDataDirs` actual `…/002/link/.crush` |
| M2 gitutil bỏ `EvalSymlinks` | `TestCurrentBranch` expected `"viasymlink"`, actual `""` |
| M3 config về `.crush` gần nhất | `TestConfig_setDefaults` expected `…/worktree/.crush`, actual `…/worktree/pkg/.crush`; `TestStrayDataDirs` "through a symlink into the project too" |
| M4 `StrayDataDirs` trả nil | `TestStrayDataDirs` expected `[…/a/b/.crush]`, actual `nil` |
| M5 khoá workspace theo đường | `TestPathDedupe_SubdirSharesProjectWorkspace` "repo/sub and repo are one workspace" (hai ID khác nhau) |
| M6 `Path: args.Path` | cùng test: "rooted at the project root whichever opened first", actual `…/005/sub` |
| M7 bỏ ghép `-D` theo `args.Path` | cùng test: expected `…/sub/x`, actual `…/x` |
| M8 Home bỏ qua `ws.DataDir` | `TestHomeSessionsFromDatabaseReadOnly` "[]" should have 50 item(s) |

**Răng e2e:**
- `agent-cli-teeth-key` (chỉ hoàn M5; data dir vẫn sửa): T1M **EXPOSED** (`workspaces=['repo', 'repo/sub']`, RAN CONCURRENTLY); T3 NOT_EXPOSED.
- `agent-cli-teeth-datadir` (chỉ hoàn M3; khoá theo gốc vẫn sửa): T3 **EXPOSED** ở local (`sub-first db=sub/.crush/crush.db … repo1=2`). T3 server và T1M NOT_EXPOSED, vì server đã dựng workspace ở gốc.
- Kết luận: mỗi lỗi e2e có đúng một cơ chế chặn ở local. Ở server, T3 được chặn bởi cả hai cơ chế.
- Log: `~/.cache/v3ev/p2c-teeth-{key,datadir}/verdicts.txt`.

## 6. Bảng chân trị

Lệnh: `bash ~/.cache/v3ev/p2c-scripts/table.sh <dir-mới> <bin> agent-cli-before <worktree>`. Đường tính theo thư mục ca. "workspace" là dòng `Working directory` trong system prompt mà LLM giả nhận. Ca E dựng 2 `.crush` bằng binary GỐC (local, `sub` trước rồi gốc).

| ca | chế độ | `642fe075` | bản sửa |
|---|---|---|---|
| A gốc | local / server | `repo/.crush`, ws `repo` | như cũ |
| B `repo/sub` (chưa có `.crush`) | local | `repo/sub/.crush`, ws `repo/sub` | `repo/.crush`, ws `repo/sub` |
| | server | `repo/sub/.crush`, ws `repo/sub` | `repo/.crush`, ws **`repo`** |
| C `-c ~/code/link-sub` (`code/.crush` có sẵn) | local | `code/.crush` (NGOÀI repo), ws `code/link-sub` | `repo/.crush`, ws `code/link-sub` (vật lý = `repo/sub`) |
| | server | `code/.crush`, ws `code/link-sub` | `repo/.crush`, ws `repo` |
| D sub → gốc → sub | local | sub1, sub2 vào `sub/.crush`, gốc vào `repo/.crush`: **2 DB** | cả 3 vào `repo/.crush`: 1 DB; ws `repo/sub`, `repo`, `repo/sub` |
| | server | 2 DB | 1 DB, ws `repo` cả 3 lượt |
| E có sẵn 2 `.crush`, mở từ sub rồi gốc | local / server | sub vào `sub/.crush`, gốc vào `repo/.crush`, không cảnh báo | cả hai vào `repo/.crush`. Mở từ sub: **có cảnh báo**; từ gốc: không. `sub/.crush` cũ còn nguyên, không gộp |

**Câu cảnh báo (nguyên văn, ca E server):**

```
warning: …/s-two/repo/sub/.crush is not used: agent-cli now keeps one data directory per project, …/s-two/repo/.crush. Nothing was merged or deleted. To open the sessions stored there: CRUSH_CLIENT_SERVER=0 agent-cli -c …/s-two/repo/sub -D …/s-two/repo/sub/.crush (merging them into the project is up to you).
```

Đã chạy đúng lệnh hướng dẫn (`… -D <sub/.crush> session list --json`, binary sau sửa): ra session `cd3eae84…`, chính session mà binary gốc tạo trong `sub/.crush`.

## 7. Hồi quy

- **Gói đã chạm và gói được giao:**
  - Lệnh: `flock … go test -p 2 -count=1 ./internal/fsext/... ./internal/gitutil/... ./internal/config/... ./internal/projects/... ./internal/ui/home/... ./internal/backend/... ./internal/cmd/...`
  - Kết quả: tất cả `ok` (kể cả `cmd/clientserverrace`, cùng T1 phơi nhiễm đang có trong cây).
- **Race:**
  - Lệnh: `flock … CGO_ENABLED=1 … go test -race -count=1 -p 2 ./internal/backend/... ./internal/cmd/... ./internal/projects/... ./internal/ui/home/... ./internal/config/...`
  - `-race` cần cgo, nên đây là chỗ duy nhất không dùng `CGO_ENABLED=0`.
  - Kết quả: rc=0, 6 package ok, `grep -c "DATA RACE"` = 0 (`p2c-scripts/race.log`).
- **Toàn kho theo TÊN:**
  - Lệnh: `go test -json -count=1 -p 2 ./...`. Base là `git archive 642fe075` cộng tệp T1 phơi nhiễm; after là worktree. So bằng `python3 -I cmp.py`.
  - Base: 4794 test, 85 package; package FAIL chỉ `internal/backend` (6 fail = T1).
  - After: 4794 test, 85 package, 0 package FAIL, 0 test đỏ.
  - **Mất 5:** đúng 5 test đã xoá ở mục 3. **Mới 5:** đều pass. **Đổi trạng thái 6:** T1 và 5 subtest, fail → pass. Không test nào khác đổi trạng thái.
  - Kết quả thô: `~/.cache/v3ev/p2c-scripts/suite-{base,after}.json`.
- **Script e2e với `agent-cli-after`** (log ở `~/.cache/v3ev/p2c-reg/`): `scripts/eval/workspaces.sh` → `WORKSPACES: PASS`; `AGENT_CLI_BIN=… scripts/test-agent-cli-run-json.sh` → `PASS`; `scripts/eval/faults.sh` → `FAULTS: PASS`.
- **Format và vet:** `gofmt -l` (GOROOT) trên 14 tệp Go: rỗng. `go vet` 7 package đã chạm: ok.
- **Thứ tự build và sửa chú thích:** sau khi build `agent-cli-after` chỉ còn sửa chú thích Go, CHANGELOG và docs. Kiểm bằng cách so các dòng `+/-` không phải chú thích giữa `git diff -- '*.go'` lúc build và lúc bàn giao: giống hệt. Suite toàn kho chạy trên cây cuối.

## 8. Đã chứng minh / Mới nghi / Giới hạn

**Đã chứng minh** (lệnh ở trên):
- T1, T1M, T3 từ EXPOSED thành NOT_EXPOSED; đối chứng không đổi.
- Mỗi cơ chế có mutant làm đỏ test; T1M và T3 có răng e2e riêng.
- Repo có sẵn 2 `.crush`: không gộp, không xoá, dùng `.crush` ở gốc, cảnh báo khi mở từ thư mục con, và lệnh trong cảnh báo mở được DB cũ.
- **Hệ quả sản phẩm của D3, đo bằng `p2c-scripts/ctx.sh`** (`sub/AGENTS.md` mang dấu `SUBCTX-MARK`, gốc mang `ROOTCTX-MARK`):
  - Qua server, mở từ `repo/sub`: bản gốc nạp `sub/AGENTS.md` (`True`) và không nạp của gốc. Bản sửa ngược lại: `sub=False`, `gốc=True`.
  - Local không đổi: `sub=True`, `gốc=False`.
  - Như vậy qua launcher, agent mở từ thư mục con làm việc ở gốc repo và mang ngữ cảnh của gốc. Đây là hệ quả trực tiếp của "`repo/sub` thuộc workspace của repo gốc"; người dùng nên biết (đã ghi ở CHANGELOG).

**Mới nghi / chưa kiểm** (kèm cách kiểm):
- **Không cảnh báo từ gốc hay Home.** Bảng ca E: từ gốc không có cảnh báo. Home trước đây liệt kê cả session của `sub/.crush` (cơ chế gộp), nay không. Người chỉ mở từ gốc hay Home sẽ không thấy session cũ mà cũng không được báo, tức hỏng âm thầm về mặt nhìn thấy (dữ liệu vẫn còn).
  - D3 viết "repo có ≥2 `.crush` → cảnh báo". Bản này chỉ báo ở nơi hành vi đổi (mở từ thư mục có `.crush` cũ trên đường lên gốc), vì quét cả cây repo mỗi lần mở là tốn trên monorepo.
  - Phương án nếu cần: quét qua `git ls-files -o -i --directory`, hoặc Home đọc trường `locations` cũ một lần để báo. **Cần điều phối quyết.**
- **Home sau nâng cấp, trường hợp chuyển tiếp:** mục registry mà lần mở cuối là từ `sub` bằng bản cũ có `data_dir = sub/.crush`. Home liệt kê session của DB đó và mở chúng ở gốc, nên sẽ không thấy session (TUI báo thế nào thì chưa xem). Lần mở đầu tiên bằng bản mới ghi lại `data_dir` ở gốc. Kiểm: registry giả + `agent-cli` trần trong tmux.
- **`crush.json` và skill riêng của `sub` không được nạp qua server:** suy từ mã (`lookupConfigs(root)` bị chặn ở gốc; `ProjectSkillsDir(root)`), chưa chạy.
- **Cảnh báo chỉ ra stderr và log.** TUI dùng alt screen nên người dùng chỉ thấy cảnh báo sau khi thoát. Chưa có toast trong TUI.
- **Server đang chạy từ bản trước:** `restartIfStale` (`root.go:984`) so `Version` và `BuildID`. Server rảnh thì bị thay; server bận thì lệnh chạy model bị từ chối (`errStaleServerBusy`). Vì vậy client mới không chạy lượt trên server có khoá theo đường cũ. Đọc mã, chưa chạy riêng cho P2c.
- **`-D` chung cho hai dự án khác nhau** vẫn ra hai workspace trên một DB (lớp T1M), không đổi so với trước. Trong một dự án, first-wins của `-D` nay áp cho cả dự án (trước chỉ cùng đường), đã ghi ở CHANGELOG.
- **`options.data_directory` tương đối** trong `crush.json` vẫn ghép theo working dir. Ở local, mở từ `sub` thì ra `sub/<x>`, vẫn có thể chia đôi. Qua server thì ghép theo gốc. Không sửa (cấu hình người dùng tự đặt); chưa đo.
- **Ca biên chưa đo:**
  - `$HOME` là repo git (dotfiles): mọi thư mục ngoài git dưới `~` sẽ quy về `~`, data dir `~/.crush`, workspace `~`. Launcher vốn đã chọn server theo gốc này. Máy này `~` không phải repo.
  - Không có `git`: rơi về chính thư mục, như cũ.
- **Local vs server lệch nhau về working dir** (local: thư mục mở; server: gốc). Lệch này có chủ đích theo D3 "khi mở bằng launcher". Nếu muốn local cũng ở gốc thì phải sửa `ResolveCwd`, phạm vi rộng hơn.

**Phản biện brief:**
- **Không chỉ cảnh báo khi ≥2 `.crush`:** bản này cảnh báo khi đường từ thư mục mở lên gốc có `.crush` cũ, kể cả khi gốc chưa có `.crush`. Đó là ca T3 khi nâng cấp: bản mới sẽ tạo `.crush` ở gốc, nên lúc đó có 2.
- **Không cảnh báo khi mở ở gốc**, lý do ở mục Mới nghi.
- **Home union:** brief chỉ đường vào. Tôi xoá hẳn thay vì giữ, vì mọi session nó liệt kê từ `sub/.crush` đều không mở được nữa (Home mở `--cwd sub --session`, mà `sub` nay ra DB ở gốc). Còn giữ thì là hai đường song song.

**Phạm vi:** không gọi LLM thật; không đụng `~/.config/agent-cli`, `~/.local/lib/agent-cli`, crontab; không đọc `.env`. Không đụng `internal/agent/agent.go`. Chưa chạy `task lint`.

## 9. Vòng 2 (sau soát độc lập vòng 1 KHONG-DAT và quyết định của người dùng)

**Quyết định của người dùng (thay D3, phần "khi mở bằng launcher"):** một workspace cho mỗi repo ở MỌI chế độ. Mở từ thư mục con thì cả local lẫn server làm việc ở gốc repo. Mục 8 "local vẫn làm việc ở thư mục mở" và câu tương ứng ở CHANGELOG vòng 1 đã bỏ.

### 9.1 Sửa thêm

| Mục soát | Tệp | Đổi |
|---|---|---|
| 1d, local ≡ server | `internal/config/load.go` | `projectBoundary` đổi thành `ProjectRoot` (export). Đây là MỘT hàm tính gốc: `Abs` trước, kết quả git nhớ theo đường tuyệt đối, `git rev-parse` có hạn 5 giây (`worktreeRootTimeout`, trước không có hạn). Thêm `InitProject(dir, dataDir, debug)` = `Init(ProjectRoot(dir), SmartJoin(dir, -D))`. |
| 1d | `internal/projects/projects.go`, `internal/backend/backend.go` | `NormalizeWorkspace` và `resolveWorkspaceKey` gọi `config.ProjectRoot`, bỏ bản git riêng có timeout riêng. `CreateWorkspace` gọi `config.InitProject`. |
| local ở gốc | `internal/cmd/root.go` `setupLocalWorkspace` | `config.InitProject(cwd…)` thay `config.Init(cwd…)`. |
| 1a | `config/load.go` `setDefaults` | `options.data_directory` tương đối (và mặc định `.crush`) ghép theo `ProjectRoot(workingDir)`; `-D` ghép theo `workingDir`. Mọi caller nhận cùng quy tắc, kể cả `session`/`stats`/`models`/`logs` vẫn gọi `Init/Load(cwd)`. |
| 3 (CHẶN) | `projects.go` | Khôi phục `Locations`/`Location`/`DataDirs()` để ĐỌC. `Register` chép nguyên `locations`, không thêm mục mới; chỉ giữ lại `data_dir` cũ của mục kiểu cũ (chưa có `locations`) khi nó đổi. |
| 3 | `config/load.go` `LegacyDataDirs(dir, root, dataDir, known)` | Thay `StrayDataDirs`. Nguồn: sổ dự án (`known`) và các `.crush` trên đường từ `dir` lên gốc. Chỉ báo `.crush` của một thư mục con (không phải của gốc, không ngoài dự án), khác data dir đang dùng, và còn `crush.db`. Không quét cây. |
| 3 | `cmd/root.go` `warnLegacyDataDirs` | Gọi ở local và server, cả khi mở từ gốc; `known` lấy từ `projects.List()`. In ra `cmd.ErrOrStderr()` và `slog.Warn` (`warnUser`). |
| 3, Home | `cmd/home.go` `toHomeWorkspace`, `ui/home/{home,view}.go` | Data dir của Home là `.crush` ở gốc dự án (DB mà TUI mở); mục sổ có `data_dir` tên khác `.crush` thì tin sổ (chú thích `shortcut:`). `Workspace.Legacy` hiện một dòng ở danh sách session (`Not listed: older sessions in … · open: …`). Màn đó vẫn ưu tiên lỗi thật nếu có. |
| 1b | `cmd/root.go` `dataDirMismatchWarning` | Cảnh báo khi `abs(-D)` khác `ws.DataDir`. |
| 1b (mở rộng) | `internal/proto/proto.go`, `backend.go` | Thêm `proto.Workspace.DataDirFromFlag` (server trả true khi client tạo workspace chọn data dir bằng `-D`). Client sau KHÔNG có `-D` cũng được cảnh báo. Đo được trước khi thêm: client đó vào DB `sub/x` mà không có dòng nào (`p2c-dmis-r2`, `G-second`). Trường thêm vào cuối struct, `omitempty`, nên server/client cũ không vỡ. |
| 6, 4b | `CHANGELOG.md`, `docs/agent-cli-local-setup.md` | Thêm mục **Thay đổi** riêng: workspace ở gốc ở mọi chế độ và hệ quả (ngữ cảnh, `crush.json`/quyền/hook/MCP, `sub/.crush/crush.json`, `!`, phạm vi "trong dự án" của tự duyệt quyền và chặn ghi, env/cờ first-wins, `data_directory`/`-D`, `.crush` cố ý ở thư mục con) và Lưu ý nâng cấp (cảnh báo, giới hạn tìm, TUI che stderr, Home). Mục Sửa lỗi rút lại phần lỗi. Docs: "Một server cho mỗi dự án" nói mỗi dự án là một workspace ở gốc; mục model nói `<project>` là gốc repo; mục Home và PRB-19 khớp hành vi mới. |

### 9.2 Test mới và test đổi (vòng 2)

**Mới:**
- `config` `TestProjectRoot`: gốc qua thư mục con, qua symlink, ngoài git qua symlink ra đường vật lý; data dir ngoài git qua symlink ra đường vật lý.
- `config` `TestProjectRootGitTimeout`: `git` giả `sleep 10`, hạn 200 ms, phải trả trong dưới 3 giây.
- `config` `TestDataDirFromSubdirectory`: qua symlink ra `repo/.crush`; `data_directory: "state"` từ `a/b` ra `repo/state`; `-D x` ra `a/b/x`.
- `config` `TestLegacyDataDirs`: thay `TestStrayDataDirs`. Thấy trên đường đi lên và qua symlink; từ gốc thì chỉ thấy nhờ sổ; `-D` trỏ đúng nó thì không báo; xoá `crush.db` thì không báo; `.crush` rỗng hay ngoài dự án thì không báo.
- `projects` `TestRegisterKeepsKnownDataDirs`: thay test vòng 1 khẳng định `locations` bị xoá. Nay `locations` được chép nguyên, `data_dir` cũ của mục kiểu cũ được giữ đúng một lần.
- `cmd` (tệp mới `datadir_test.go`):
  - `TestDataDirMismatchWarning`.
  - `TestClientServerWarnsAboutDataDirs` (server giả): mở từ GỐC thấy cảnh báo `.crush` cũ qua sổ; `-D x` lệch thì cảnh báo; không còn `crush.db` thì im lặng; client không `-D` vào workspace dựng bằng `-D` thì cảnh báo.
  - `TestLocalWorkspaceFromSubdirIsTheProject`: `setupLocalWorkspace` thật từ `sub` cho `WorkingDir()=repo`, data dir `repo/.crush` có `crush.db`, có cảnh báo `.crush` cũ.
- `ui/home` `TestLegacyDataDirNote`: dòng báo hiện ở danh sách session của workspace có `Legacy`, không hiện ở workspace khác.
- `backend` `TestPathDedupe_SubdirSharesProjectWorkspace`: thêm assert `DataDirFromFlag` false/true.

**Đổi:**
- `cmd` `TestHomeSourceAdd`: mục sổ `data_dir = sub/.crush` thì Home dùng `repo/.crush` (DB mà TUI mở). `Legacy` rỗng khi chưa có `crush.db`, ra `[sub/.crush]` khi có. Vòng 1 khẳng định Home giữ `sub/.crush`, chính là ca "liệt kê session không mở được" ở mục soát 3.
- `projects` `TestPinRemoveNormalizeOutsideLocks`: Pin và Remove dùng hai thư mục khác nhau. Lý do: gốc dự án nay được nhớ trong tiến trình (`ProjectRoot`), nên thao tác thứ hai trên cùng thư mục không gọi `git` lần nữa, mà test dùng "git đã chạy" làm dấu hiệu. Ý của test (chuẩn hoá đường chạy TRƯỚC khi lấy khoá) giữ nguyên.

**Xoá** (so theo tên ở 9.4): đúng 5 test của vòng 1. Ba test `projects` kiểm việc `Register` THÊM/giới hạn `locations` (nay không thêm). Hai test `cmd` kiểm Home gộp session từ nhiều DB (nay một DB cộng dòng báo).

### 9.3 Bằng chứng e2e (binary thật, LLM giả)

Binary: `agent-cli-before` (`642fe075`) và `agent-cli-after4` (bản cuối; `git diff -- '*.go'` lúc build: `dac55431…`, sau đó không đổi mã Go). `agent-cli-after3` là bản trước đó: khác `after4` đúng một chỗ, hằng 5 giây đổi thành biến `worktreeRootTimeout` cùng giá trị (đã so `diff` các dòng `+/-`).

- **T1M, T3** (`bash scripts/eval/exposure.sh <dir> <bin> T1M T3`):
  - Before (`~/.cache/v3ev/p2c-r2-before`): `T1M: EXPOSED … RAN CONCURRENTLY; … workspaces=['repo/sub', 'repo']`; `T3: EXPOSED … crush.db count repo1=2 … root sees 1/3` (local và server).
  - Sau (`~/.cache/v3ev/p2c-r2-after4`): `T1M: NOT_EXPOSED … queued; … workspaces=['repo']`; `T3: NOT_EXPOSED … repo1=1 … root sees 3/3, repo1/sub sees 3/3` (local và server).
  - Đối chứng không đổi ở cả hai binary.
- **T1 unit:** trong suite theo tên (9.4), `TestV3Exposure_T1_IdentityResolversAgree` và 5 subtest: fail → pass.
- **Ca mới: mở từ `sub` ở chế độ LOCAL** (`table.sh`, `CRUSH_CLIENT_SERVER=0`, `~/.cache/v3ev/p2c-tbl-r2final/table.txt`). Bảng của bản cũ ở mục 6.

| ca | local (vòng 1) | local (vòng 2) | server (vòng 2) |
|---|---|---|---|
| `repo/sub` | DB `repo/.crush`, ws `repo/sub` | DB `repo/.crush`, ws **`repo`** | DB `repo/.crush`, ws `repo` |
| symlink vào sub | DB `repo/.crush`, ws `code/link-sub` | DB `repo/.crush`, ws **`repo`** | như local |
| sub → gốc → sub | 1 DB, ws `repo/sub`/`repo`/`repo/sub` | 1 DB, ws `repo` cả 3 | như local |
| có sẵn 2 `.crush` (bản gốc tạo), mở sub rồi gốc | cảnh báo: sub có, gốc **không** | cảnh báo: sub có, gốc **có** | như local |

- **Nâng cấp rồi CHỈ mở từ gốc** (`rootonly.sh`; bản gốc tạo `sub/.crush` rồi `repo/.crush`, binary đo chỉ mở gốc):
  - Before: `(không cảnh báo)` ở local và server.
  - Sau (`after3`): `warning: …/repo/sub/.crush is not used: … To open the sessions stored there: CRUSH_CLIENT_SERVER=0 agent-cli -c …/repo/sub -D …/repo/sub/.crush …` ở cả local lẫn server.
- **Ngữ cảnh** (`ctx.sh`, `p2c-ctx-r2final`): mở từ `sub`, local và server đều `sub/AGENTS.md trong prompt=False, AGENTS.md gốc trong prompt=True`. Vòng 1 local là `True/False`, nên local ≡ server đã đạt.
- **`-D` lệch qua server** (`dmis.sh`, `p2c-dmis-r2final`). Client đầu mở từ `sub` với `-D x`, đang chạy lượt dài.
  - Client sau ở gốc, không `-D` (`G-second`): `warning: this project's agent-cli server uses the data directory …/repo/sub/x, chosen with --data-dir by an earlier client …`.
  - Client với `-D repo/.crush` (`G-third`): `warning: --data-dir …/repo/.crush was ignored: … already uses …/repo/sub/x …`.
  - Chỉ có `repo/sub/x/crush.db`; first-wins vẫn như cũ, nay có báo.
- **Hồi quy** (`after4`):
  - `workspaces.sh` → `WORKSPACES: PASS`; `test-agent-cli-run-json.sh` → `PASS`; `faults.sh` → `FAULTS: PASS`.
  - Log: `~/.cache/v3ev/p2c-reg2-*.log`.

### 9.4 Kiểm răng, suite, race

**Mutant vòng 2:** `bash ~/.cache/v3ev/p2c-scripts/mutants2.sh`, log ở `mutants2.log`. 14 lượt, đều ĐỎ. Sha256 của `git diff` trước và sau: cùng `ac987da2…` (khôi phục bằng bản sao tệp, không dùng git).

| Mutant | Test đỏ |
|---|---|
| R1 local `config.Init(cwd)` | `TestLocalWorkspaceFromSubdirIsTheProject` expected `…/repo`, actual `…/repo/sub` |
| R2 `data_directory` ghép theo thư mục mở | `TestDataDirFromSubdirectory` |
| R3 `InitProject` bỏ ghép `-D` theo thư mục mở | `TestPathDedupe_SubdirSharesProjectWorkspace` expected `…/sub/x`, actual `…/x` |
| R4 `NormalizeWorkspace` không dùng `ProjectRoot` | `TestSubdirSharesEntryWithRepoRoot`, `TestRegisterKeepsKnownDataDirs` (2 mục sổ) |
| R5 git không có hạn | `TestProjectRootGitTimeout` "10.003s is not less than 3s" |
| R6 `ProjectRoot` bỏ `EvalSymlinks` (mục soát 5) | `TestProjectRoot` "outside git: the physical path" |
| R7 `LegacyDataDirs` bỏ `known` | `TestLegacyDataDirs` "from the root, known to the registry"; `TestClientServerWarnsAboutDataDirs` |
| R8 `Register` bỏ `Locations` | `TestRegisterKeepsKnownDataDirs` "copied as is" |
| R9a/R9b bỏ lời gọi cảnh báo (local / server; mục soát 5) | `TestLocalWorkspaceFromSubdirIsTheProject` / `TestClientServerWarnsAboutDataDirs` |
| R10a bỏ lời gọi cảnh báo `-D` | `TestClientServerWarnsAboutDataDirs` "… was ignored" |
| R10b server không đặt `dataDirFromFlag` | `TestPathDedupe_SubdirSharesProjectWorkspace` |
| R11 Home tin `data_dir` của sổ | `TestHomeSourceAdd` "the data dir the TUI opens, not the old sub/.crush" |
| R12 Home không báo | `TestLegacyDataDirNote` |

Mutant vòng 1 (M1–M8, mục 5) vẫn áp cho các cơ chế không đổi. M3 (`.crush` gần nhất) và M5 (khoá theo đường) nay tương ứng R2/R4 và test backend.

**Suite theo TÊN:**
- Lệnh: `go test -json -count=1 -p 2 ./...`, so với base `642fe075` cộng tệp T1 (dùng lại lượt base vòng 1, cây base không đổi) bằng `cmp.py`.
- Base: 4794 test, package FAIL `internal/backend` (6 fail = T1). After: 4801 test, 85 package, **0 package FAIL, 0 test đỏ**.
- Mất 5: đúng các test ở mục "Xoá". Mới 12: đều pass. Đổi trạng thái 6: T1 và 5 subtest, fail → pass. Không test nào khác đổi.
- Thô: `~/.cache/v3ev/p2c-scripts/suite-r2.json`.

**Race:**
- Lệnh: `flock … CGO_ENABLED=1 … go test -race -count=1 -p 2` trên `backend`, `cmd`, `projects`, `ui/home`, `config`, `proto`, `fsext`, `gitutil`.
- Kết quả: rc=0, 9 package ok, `DATA RACE` = 0 (`race-r2.log`).

**Format và vet:** `gofmt -l` rỗng; `go vet` 8 package ok.

### 9.5 Đã chứng minh / Mới nghi / Giới hạn đã biết

**Đã chứng minh:**
- T1, T1M, T3 NOT_EXPOSED.
- Local ≡ server: workspace ở gốc, cùng DB, cùng ngữ cảnh.
- Mở từ gốc vẫn cảnh báo `.crush` cũ mà sổ biết (sau nâng cấp); Home dùng DB ở gốc và có dòng báo.
- `-D` lệch được báo, kể cả với client không `-D`.
- Mỗi sửa có mutant đỏ.

**Mới nghi:**
- **Skill riêng của thư mục con không nạp** (`ProjectSkillsDir(root)`): suy từ mã, chưa chạy. Với `crush.json` thì mã cũng cho thấy `lookupConfigs(root)` không thấy `sub/crush.json` (người soát đã xác nhận ở mục soát 2a), nhưng tôi chưa chạy e2e cho `crush.json`.
- **Dòng báo trong Home** mới kiểm bằng unit test ở chiều rộng 120 cột. Terminal hẹp hơn sẽ cắt lệnh mở lại bằng `…`. Chưa xem trong TUI thật.
- **`ProjectRoot` nhớ kết quả suốt tiến trình**, kể cả khi git thất bại hay quá hạn. Home và server chạy lâu sẽ không thấy một thư mục vừa được `git init`, và nhớ luôn một lần git quá hạn cho tới khi khởi động lại. Điều này có chủ đích, để trong một tiến trình khoá workspace và data dir không bao giờ lệch nhau (mục soát 1d).
- **`session`, `stats`, `models`, `logs` vẫn nạp cấu hình ở thư mục mở.** Data dir đúng (quy tắc nằm trong `setDefaults`), nhưng các lệnh này đọc cả `crush.json` của thư mục con nếu có. Chưa đo tác động.

**Giới hạn đã biết** (mục soát 1c, 1e, 4a, 4c, 4d; không sửa theo điều phối):
- **1c:** launcher (server) và binary trần hoặc `CRUSH_CLIENT_SERVER=0` (local) mở cùng dự án: cùng DB, hai coordinator ở hai tiến trình. Local không lấy `crush.lock` (`db/connect.go:115-125`). Lỗi này có từ trước.
- **1e:** `$HOME` là repo git, hoặc `GIT_DIR`/`GIT_WORK_TREE` đang export: mọi thư mục quy về gốc đó (data dir, workspace, working dir). Máy này `~` không phải repo.
- **4a:** `.crush` cố ý đặt ở thư mục con bị bỏ qua và được báo là "cũ". CHANGELOG có hướng dẫn: cho thư mục con repo riêng, hoặc dùng `-D`.
- **4c:** `traverseUpBounded` trả đường vật lý cho mọi caller, kể cả `lookupConfigs` (đường tệp cấu hình trong `loadedPaths` là đường vật lý).
- **4d:** `fsext.LookupClosestBounded` (upstream) không còn caller ngoài test; giữ lại để merge upstream rẻ.
- **Giới hạn tìm `.crush` cũ:** chỉ qua sổ dự án và đường đi lên. `.crush` mà sổ không biết (vd sổ đã bị xoá) và không nằm trên đường từ thư mục mở lên gốc thì không được báo.
- **Cảnh báo ra stderr** bị TUI che cho tới khi thoát. Home có dòng báo riêng; TUI chưa có toast.
- **`internal/ui/home` `TestSessionOpensWhereItLives`** vẫn mô tả "session ở data dir của thư mục con". Cơ chế `Session.Dir` vẫn dùng cho session từ server sống; tôi không sửa test này.

**Phạm vi:** không gọi LLM thật; không đụng `~/.config/agent-cli`, `~/.local/lib/agent-cli`, crontab; không đọc `.env`; không đụng `internal/agent/agent.go`; chưa chạy `task lint`. Script đo (`table.sh`, `ctx.sh`, `dmis.sh`, `rootonly.sh`, `mutants*.sh`, `cmp.py`) nằm ở `~/.cache/v3ev/p2c-scripts/`, không vào repo.

## 10. Vòng 3 (sau soát vòng 2 KHONG-DAT)

### 10.1 CHẶN: tiến trình ở `sub` trong khi agent ở gốc

**Tái hiện ĐỎ trước khi sửa.** Thêm assert `os.Getwd() == gốc` vào hai test đã có (`internal/cmd/datadir_test.go`):
- `TestLocalWorkspaceFromSubdirIsTheProject`: `setupLocalWorkspace` thật, mở bằng `-c sub`.
- `TestClientServerWarnsAboutDataDirs`: ca mới mở từ `sub` KHÔNG có `-c` (`t.Chdir(sub)`), qua server giả.

Lệnh: `flock … go test -p 2 -count=1 -run 'TestClientServerWarnsAboutDataDirs|TestLocalWorkspaceFromSubdirIsTheProject' ./internal/cmd/` (log ở `p2c-scripts/r3-red.log`, `r3-green.log`):

```
# trước khi sửa
--- FAIL: TestClientServerWarnsAboutDataDirs … expected: "…/001/repo" actual: "…/001/repo/sub"  Messages: process cwd is the workspace's working dir
--- FAIL: TestLocalWorkspaceFromSubdirIsTheProject … expected: "…/001/repo" actual: "…/001/repo/sub"  Messages: process cwd is the workspace's working dir
# sau khi sửa
ok  	github.com/charmbracelet/crush/internal/cmd	0.141s
```

**Sửa** (`internal/cmd/root.go`): thêm `enterWorkspaceDir(dir)` theo mẫu chdir của `ResolveCwd`.
- Gọi ở `setupLocalWorkspace` với `store.WorkingDir()`, ngay sau `InitProject`.
- Gọi ở `connectToServer` với `ws.Path`, ngay sau khi tạo workspace. Hàm này phủ TUI, `run`, `send`, `jobs`, `logout`, `loop`.
- Không chdir được (workspace trên máy khác) thì chỉ `slog.Warn`.

**Soát chỗ đọc cwd tiến trình** (`grep -rn 'os.Getwd()\|filepath.Abs(\|ListDirectory(".")' internal/` trừ test, tools, config, fsext, projects, db, backend):
- `ui/completions/completions.go:408` `ListDirectory(".")`: nay ở gốc.
- `ui/model/ui.go:5162` đính kèm `filepath.Abs(path)`, và `ui.go:6625`, `:6742` dán đường dẫn: tương đối theo gốc, khớp tool (`SmartJoin(workingDir, …)`).
- `ui/dialog/filepicker.go:131`: dùng `Workspace.WorkingDir()` trước, `Getwd` chỉ là dự phòng.
- `cmd/home.go:73`: Home chạy trước khi có workspace.
- `cmd/root.go:126` `startDir`: chụp TRƯỚC khi dựng workspace, chỉ dùng cho `execHome`.
- `cmd/stats.go:214`: nay `ProjectRoot(cwd)`.
- `filetracker/service.go:63,83` `relpath`: đường tương đối theo cwd tiến trình. Ở local nay là gốc, khớp working dir. Ở server, gốc là cwd của tiến trình server (có từ trước, không đổi).

**Hệ quả của chdir đã xét:**
- Đường tương đối trên dòng lệnh: `-D` đã được ghép theo thư mục mở TRƯỚC khi chdir (`InitProject`; qua server thì `Path` gửi đi là tuyệt đối). `--cwd` ghép trước. `stats --crawl-dir` không dựng workspace. `loop` chạy verify với `Dir` tường minh (`p.Cwd`). Không cờ nào khác nhận đường dẫn (`grep` flags).
- Ở TUI, `@x.go` nay là `<gốc>/x.go`. Đã ghi ở CHANGELOG và docs.

### 10.2 Các mục NÊN SỬA

- **(a)** Câu cảnh báo nay: `To read the sessions stored there: CRUSH_CLIENT_SERVER=0 agent-cli -c <sub> -D <sub>/.crush. That opens them at the project root, while relative paths in them meant <sub>, so continuing them can edit the wrong file; …`. CHANGELOG ghi tương tự (chỉ nên đọc).
- **(b)** `session.go`, `stats.go` (2 chỗ, cộng tên dự án), `models.go`, `logs.go` chuyển sang `config.InitProject(config.ProjectRoot(cwd), cwd, -D, …)`.
- **(c)** `worktreeRoot`:
  - Không nhớ kết quả khi `ctx.Err() != nil` hoặc git không chạy được; vẫn nhớ khi git đã chạy và trả "không phải repo".
  - Dùng `LoadOrStore`.
  - `InitProject` nhận `root` do người gọi tính: backend truyền `key`, không tính lại.
  - `-D` tương đối nay được tuyệt đối hoá theo `filepath.Abs(dir)`, đúng cả khi `dir` rỗng (cờ `logs -c` không đặt).
- **(d)** `recreateArgs` (`internal/workspace/client_workspace.go`) chỉ gửi `DataDir` khi `ws.DataDirFromFlag`.
- **GHI CHÚ:**
  - Đổi tên `TestSessionOpensWhereItLives` thành `TestSessionOpensInItsWorkspaceDir`, mô tả cơ chế hiện hành (`Session.Dir` = đường của workspace trên server sống).
  - CHANGELOG có đoạn **Giới hạn còn lại**; câu "git quá 5 giây" sửa thành "lần mở đó coi như ngoài git (không nhớ, lần sau hỏi lại)".

### 10.3 Test mới/đổi (vòng 3)

- `cmd` `TestLocalWorkspaceFromSubdirIsTheProject`, `TestClientServerWarnsAboutDataDirs`: thêm assert cwd tiến trình (10.1); thêm ca server không `-c`.
- `config` `TestProjectRootGitTimeout`: `git` giả treo ở lần ĐẦU rồi chuyển sang git thật. Lần 1 ra `sub` trong dưới 3 giây; lần 2 ra `repo`, tức không nhớ lần quá hạn.
- `backend` `TestCreateWorkspace_GitTimeoutKeepsKeyAndConfigTogether` (MỚI, khoảng 6,7 giây vì chờ hạn 5 giây thật): git treo đúng lần tính khoá thì `ws.Path == ws.Cfg.WorkingDir()`.
- `workspace` `TestRecreateArgsSendsDataDirOnlyFromFlag` (MỚI).
- `ui/home` đổi tên test (ghi chú ở 10.2).

### 10.4 Kiểm răng (mục 1, 4, 5)

Lệnh: `bash ~/.cache/v3ev/p2c-scripts/mutants3.sh`, log ở `mutants3.log`. Khôi phục bằng bản sao tệp; `git diff | sha256sum` trước và sau đều là `83a1aaac…`.

| Mutant | Test đỏ |
|---|---|
| U1a bỏ `enterWorkspaceDir(store.WorkingDir())` | `TestLocalWorkspaceFromSubdirIsTheProject` actual `…/repo/sub` |
| U1b bỏ `enterWorkspaceDir(ws.Path)` | `TestClientServerWarnsAboutDataDirs` actual `…/repo/sub` |
| U4a nhớ cả lần git quá hạn | `TestProjectRootGitTimeout` "the timeout was not remembered: git is asked again" (actual `…/a/b`) |
| U4b backend tính lại gốc thay vì dùng `key` | `TestCreateWorkspace_GitTimeoutKeepsKeyAndConfigTogether`: Path `…/sub`, WorkingDir `…/005` |
| U5 `recreateArgs` luôn gửi `DataDir` | `TestRecreateArgsSendsDataDirOnlyFromFlag` "Should be empty, but was /r/.crush" |

### 10.5 Hồi quy và phơi nhiễm

Binary cuối là `agent-cli-after5`. `git diff -- '*.go' | sha256sum` lúc build: `4cca2e5c…`, bằng đúng lúc bàn giao.

- **Phơi nhiễm:**
  - `exposure.sh ~/.cache/v3ev/p2c-r3-after5 … T1M T3`: `T1M: NOT_EXPOSED … queued; … workspaces=['repo']`; `T3: NOT_EXPOSED … repo1=1 … 3/3` (local và server).
  - Binary 642fe075 vẫn EXPOSED cả hai (`p2c-r2-before`, cùng binary).
  - T1: trong suite, fail → pass.
- **Bảng chân trị** (`p2c-tbl-r3`): giống 9.3. Local và server đều ws `repo`, 1 DB; ca có 2 `.crush` cảnh báo cả khi mở từ sub lẫn từ gốc.
- **Chỉ mở từ gốc sau nâng cấp** (`p2c-ro-r3`): 2/2 có cảnh báo.
- **Script:** `workspaces.sh` WORKSPACES: PASS; `test-agent-cli-run-json.sh` PASS; `faults.sh` FAULTS: PASS; `test-agent-cli-loop.sh` PASS (chạy thêm vì `loop` đi qua chỗ chdir). Log: `~/.cache/v3ev/p2c-reg3-*.log`.
- **Toàn kho theo TÊN** so với base 642fe075 (`suite-r3.json`):
  - Base 4794 test → after 4803 test, 0 package FAIL, 0 test đỏ.
  - Mất 6: 5 test của vòng 2 cộng tên cũ `TestSessionOpensWhereItLives`, nay có tên mới.
  - Mới 15, đều pass. Đổi trạng thái 6: T1 và 5 subtest, fail → pass.
- **Race:** `-race` trên 10 package đã chạm (backend, cmd, clientserverrace, projects, ui/home, config, proto, fsext, gitutil, workspace): rc=0, 0 `DATA RACE` (`race-r3.log`).
- **Format và vet:** `gofmt -l` rỗng; `go vet` 9 package ok.

### 10.6 Đã chứng minh / Mới nghi / Giới hạn

**Đã chứng minh:**
- Cwd tiến trình bằng gốc ở cả hai chế độ (đỏ → xanh, có mutant).
- Lần git quá hạn không bị nhớ.
- Khoá workspace và working dir không lệch khi git treo.
- `recreateArgs` không còn tự gắn cờ `-D`.
- T1, T1M, T3 NOT_EXPOSED.

**Mới nghi:**
- Chưa chạy TUI thật để gõ `@tệp`. Bằng chứng mới ở mức cwd tiến trình cộng việc đọc mã của completion và đính kèm.
- `filetracker`: session cũ mở ở local từ `sub` lưu đường đã đọc theo `sub`, nay được ghép theo gốc. Tool sửa tệp có thể đòi đọc lại. Đọc mã, chưa đo.

**Giới hạn còn lại** (CHANGELOG cũng ghi):
- 1c: local và server cùng một DB thì có hai tiến trình.
- 1e: `$HOME` là repo git, hoặc `GIT_DIR`/`GIT_WORK_TREE` đang export.
- 4a, 4c, 4d: như mục 9.5.
- Workspace trên máy khác: tiến trình không chuyển được sang gốc, chỉ ghi log.
- Cảnh báo ra stderr bị TUI che cho tới khi thoát.
- Dòng báo trong Home bị cắt ở terminal hẹp.
- Chưa chạy `task lint`.

## 11. Vòng 4 (soát vòng 3 DAT; 2 mục NÊN SỬA trước commit)

### 11.1 `loop` mở từ thư mục con

Quyết định của điều phối: verify vẫn chạy ở thư mục mở.
- (a) Prompt nêu thư mục verify, tính tương đối theo gốc, khi nó khác working dir; khi trùng thì prompt giữ nguyên từng byte.
- (b) Dấu vân tay git tính trên cả repo.

**ĐỎ trước khi sửa** (`internal/cmd/loop_test.go`):
- `TestExecuteLoop_VerifyDirIsNamedInThePrompt`: `executeLoop` với `Cwd=repo/sub` và `Cwd=repo`.
- `TestExecuteLoop_FingerprintCoversTheWholeRepo`: agent tạo tệp ở GỐC mỗi lượt, verify luôn cùng một lỗi.

Lệnh: `flock … go test -p 2 -count=1 -run 'TestExecuteLoop_VerifyDirIsNamedInThePrompt|TestExecuteLoop_FingerprintCoversTheWholeRepo' ./internal/cmd/` (log ở `p2c-scripts/r4-red.log`, `r4-green.log`):

```
--- FAIL: TestExecuteLoop_FingerprintCoversTheWholeRepo … Error: Should not be: "STALLED"  Messages: the agent changed the repo outside sub/
--- FAIL: TestExecuteLoop_VerifyDirIsNamedInThePrompt … "do the thing\n\nDefinition of done: running `test -f done.txt` exits with code 0.\n…" does not contain "Definition of done: running `test -f done.txt` in `sub/` exits with code 0."
# sau khi sửa (cùng các test TestExecuteLoop*, TestGitFingerprint*, TestShellVerify*)
ok  	github.com/charmbracelet/crush/internal/cmd	0.506s
```

**Sửa:**
- `internal/loop/loop.go`: thêm `Spec.VerifyDir` (thư mục verify, tương đối theo working dir của agent; rỗng khi trùng).
  - `goalPrompt` thêm `` in `sub/` `` sau lệnh.
  - `repairPrompt` thêm `` (run in `sub/`) ``.
  - Khi rỗng, chuỗi định dạng ra đúng từng byte như cũ.
- `internal/cmd/loop.go` `executeLoop`:
  - `root := config.ProjectRoot(p.Cwd)` (cùng hàm với workspace); `VerifyDir = Rel(root, EvalSymlinks(p.Cwd)) + "/"` khi khác `.`.
  - `Fingerprint: gitFingerprint(root)`: `ls-files --others` chạy ở gốc nên thấy cả repo; `git diff HEAD` vốn đã phủ cả repo.
  - `Verify: shellVerify(p.Cwd, …)` giữ nguyên.

**Test thêm** (`internal/loop`) `TestPromptsNameTheVerifyDirOnlyWhenItDiffers`:
- `VerifyDir` rỗng thì `goalPrompt` và `repairPrompt` BẰNG đúng chuỗi literal theo định dạng cũ.
- `"sub/"` thì nêu thư mục ở cả hai prompt.

**Mutant** (`p2c-scripts/mutants4.sh`, log `mutants4.log`; khôi phục bằng bản sao, sha `git diff` trước và sau đều `1d226273…`):

| Mutant | Test đỏ |
|---|---|
| V1 cmd không đặt `VerifyDir` | `TestExecuteLoop_VerifyDirIsNamedInThePrompt` |
| V2 dấu vân tay ở thư mục mở (như cũ) | `TestExecuteLoop_FingerprintCoversTheWholeRepo` "Should not be: STALLED" |
| V3 prompt đổi cả khi `VerifyDir` rỗng | `TestPromptsNameTheVerifyDirOnlyWhenItDiffers` (Not equal) |
| V4 lượt sửa không nêu thư mục | `TestPromptsNameTheVerifyDirOnlyWhenItDiffers` và `TestExecuteLoop_VerifyDirIsNamedInThePrompt` |

**E2E** (`p2c-scripts/loopsub.sh`; LLM giả; `loop --verify "test -f done.txt"` mở từ `repo/sub`; đọc tin user gửi LLM):
- Bản gốc: `Definition of done: running `test -f done.txt` exits with code 0.` và `Verify command: `test -f done.txt`` (không nêu thư mục), ở local và server.
- `agent-cli-after6`: `… running `test -f done.txt` in `sub/` exits with code 0.` và `Verify command: `test -f done.txt` (run in `sub/`)`, ở local và server.
- Dừng `STALLED` ở cả hai binary: đúng, vì agent giả không đổi gì.

### 11.2 CHANGELOG

- **Lưu ý nâng cấp, mục filetracker:** session tạo khi mở từ thư mục con TRƯỚC bản này (kể cả session nằm trong `<gốc>/.crush`) lưu danh sách tệp đã đọc theo `sub`; nay ghép theo gốc.
  - Hệ quả: `write` (`internal/agent/tools/write.go:75-76`) có thể coi tệp CÙNG TÊN ở gốc là đã đọc rồi ghi đè; `--yolo`/`loop` không hỏi quyền.
  - Khuyên bắt đầu session mới cho việc ở thư mục con.
- **Mục `loop`:** verify chạy ở thư mục mở, agent ở gốc; prompt nêu thư mục; STALLED/NO_CHANGE xét cả repo; mở ở gốc thì prompt như cũ.

### 11.3 Hồi quy

Binary cuối là `agent-cli-after6`. `git diff -- '*.go' | sha256sum` lúc build: `40e6a243…`, bằng đúng lúc bàn giao.

- **Loop và qwen-only:** `AGENT_CLI_BIN=… scripts/test-agent-cli-loop.sh` → `PASS`. `scripts/eval/qwen-only.sh` (có ca loop `NO_CHANGE`) → `QWEN-ONLY: PASS`. Log: `~/.cache/v3ev/p2c-reg4-*.log`.
- **Phơi nhiễm:** `exposure.sh ~/.cache/v3ev/p2c-r4-after6 … T1M T3` → `T1M: NOT_EXPOSED`, `T3: NOT_EXPOSED` (local và server).
- **Toàn kho theo TÊN** so với 642fe075 (`suite-r4.json`):
  - Base 4794 test → after 4806 test, 0 package FAIL, 0 test đỏ.
  - Mất 6, giống vòng 3. Mới 18 (thêm 3 test loop), đều pass.
  - Đổi trạng thái 6: chỉ T1, fail → pass.
- **Race:** `-race` trên `cmd`, `cmd/clientserverrace`, `loop`: rc=0, 0 `DATA RACE` (`race-r4.log`).
- **Format và vet:** `gofmt -l` rỗng; `go vet ./internal/cmd ./internal/loop` ok.

### 11.4 Giới hạn

- Không có giới hạn mới ngoài mục 10.6.
- Verify vẫn chạy ở thư mục mở, theo quyết định của điều phối. Agent phải tự `cd` khi muốn chạy lại lệnh verify; prompt đã nói chỗ.
- **Có từ trước, phát hiện khi soát:** filetracker của server ghép đường theo cwd của TIẾN TRÌNH SERVER, tức thư mục của client đã dựng nó. Server dựng lại từ thư mục khác thì cùng dạng rủi ro ghi đè như 11.2. Không sửa trong P2c.
