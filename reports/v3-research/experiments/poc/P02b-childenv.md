# P02b: `.env` của repo không lái tiến trình con (các mục "nên sửa" sau soát P0-2)

Worktree `/home/luannt/application/opensource/agent-cli-p02b`, nhánh `v3-poc-p02b-childenv`, gốc `v3-poc-p02-policy`
(`26bed1e9`). Chưa commit, chưa push. Người làm: agent P0-2b. Soát an ninh độc lập lần 1 chấm KHONG-DAT; bản này là
vòng 2 (§8), chưa soát lại.

- **Patch:** `P02b-childenv.patch` (18 tệp: 6 tệp mã, 9 tệp test, 3 tệp docs; không gồm 6 tệp test phơi nhiễm `??` và
  `scripts/eval/exposure.sh`). `sha256sum -c P02b-childenv.patch.sha256` →
  `6dc3ae34b9f105b8499bdadc0bf87c071f3aef004a56c94b03ebf97db83f8699` (vòng 2; thay `9f75dfd0…` của mục 7 và
  `f2799537…` của mục 1–6).
- **Đã kiểm trên cây sạch `v3-poc-p02-policy` (git archive):** `git apply --whitespace=error-all` rc=0; 18/18 tệp giống
  hệt worktree (`cmp`); `go test` 5 gói `cmd backend workspace agent/tools/provider execpolicy` đều `ok`.
- **Scratch:** `scratchpad/p02b/` (script `gp.sh`, `runtb.sh`, `mut.py`, `p02b-e2e.sh`, `p02b-sv.sh`, `regress.sh`,
  `mkpatch.sh`, `p02b-routine-unattended.sh`, `p02b-home-cli.sh`, `round2.sh`). Chứng cứ e2e dưới
  `/home/luannt/.cache/v3ev/p02b-*`. Binary: `/home/luannt/.cache/v3ev/p02b-bin/` (`agent-cli-before` = `26bed1e9`,
  `agent-cli-after` = mục 1–6, `agent-cli-after7` = thêm mục 7, `agent-cli-r2` = bản cuối vòng 2, `agent-cli-m5` =
  mutant M5).

**Cờ cho người soát (đọc trước):**
0. **Vòng 2 (§8):** luồng hằng ngày `agent-cli` trần → Home → chọn dự án từng lọt biến của `.env` tới CLI con giao việc
   (e2e mới HCL/HCS: EXPOSED trên `agent-cli-after7`). Nay `homeEnv()` = `OriginalEnviron()`: con của Home không nhận
   biến nào của `.env`, kể cả biến thường (chúng tự nạp lại `.env` ở cùng cwd). Đoạn xử lý riêng launcher của mục 1 đã
   bỏ vì thừa. `serverEnv()` được tách ra và test bằng unit.
1. **Mục 3 đổi env lúc dựng server** (`startDetachedServer`): server nay nhận env GỐC của client (trước `.env`), thay
   cho `ChildEnv(env sau .env)`. Không đổi chỗ này thì chế độ server (đường của launcher) vẫn lộ: mutant M5 cho e2e
   `server: EXPOSED` dù CLI con đã dùng env gốc (§3). Server chạy ở cwd của client nên tự nạp lại đúng `.env` đó; đã
   đo tool bash ở chế độ server vẫn thấy biến thường của `.env` (đối chứng SV, trước/sau đều PASS). Hệ quả hành vi ở
   §9 (R1–R3).
2. **Hai mức khác nhau, cố ý:** lệnh `!` (mục 2) và verify của loop dùng `ChildEnv`: chỉ biến tin cậy theo env gốc,
   biến thường của `.env` vẫn tới (như bash tool). Tiến trình agent-cli con dài hạn (workspace và Home của Home, server)
   cùng CLI con giao việc dùng `OriginalEnviron`: env gốc hoàn toàn, kể cả `env` của `crush.json` không tới.
3. **Test cũ bị sửa** (không phải sửa cho xanh): `child_env_test.go` thêm assert launcher và một lần chạy con thứ hai;
   khi đỏ không còn in cả env (lọc dòng `KEY=value`, không dùng `Contains` trên cả env). Vòng 2 đổi assert cũ
   "`PLAIN_FROM_DOTENV=kept` tới cả 3 tiến trình con" (nó khẳng định đúng hành vi gây lỗ): nay HOMECHILD, HOMEEXEC,
   SERVER không có biến nào của `.env`; chỉ VERIFY còn biến thường. `restart_stale_test.go`: `newControlServer` có tuỳ chọn `unix`
   (test cũ không đổi). `no_agent_runs_test.go`: thêm `isolateServerCache`.
4. **Mục 7 (hồi quy do P0-2, điều phối yêu cầu sửa, §7):** từ `26bed1e9`, `--verify` của `routine run` không còn thấy
   `AGENT_CLI_UNATTENDED=1` mà `exportUnattendedEnv` đặt cho mọi tiến trình con. Đã sửa ở `ChildEnv`: khi tiến trình
   đang trong lượt không người trực (`SetUnattended`), hai công tắc theo trạng thái đó; ngoài lượt, theo ảnh chụp như
   cũ (`.env` vẫn không bật được). E2E trên V2 / `26bed1e9` / bản cuối: §7.

## 1. Mục 1: `homeEnv` đặt lại `AGENT_CLI_LAUNCHER` theo env gốc

> Vòng 2 (§8) thay cách sửa dưới đây bằng `homeEnv() = OriginalEnviron()`: launcher tự theo env gốc. M1/M1b không còn
> áp được (mã đã bỏ); M11 thay chúng.

**Vấn đề:** chạy binary trần (không launcher), mở TUI thẳng trong repo có `.env` đặt `AGENT_CLI_LAUNCHER`. Alt+W →
`execHome` exec chính binary với `homeEnv()`; env đó mang launcher của `.env` (không phải khoá tin cậy nên `ChildEnv`
giữ). Home mới chụp nó làm env GỐC, nên `homeLauncher()` (đọc `OriginalEnv`) trả chương trình của repo và Home chạy nó.

**ĐỎ trước khi sửa:**
- Unit: `bash scratchpad/p02b/runtb.sh <tb-red> <home mới>` (test binary build từ mã chưa sửa + test mới):
  `TestChildEnvsIgnoreProjectDotenv` FAIL, `Should be empty, but was [AGENT_CLI_LAUNCHER=/bin/evil-launcher
  AGENT_CLI_LAUNCHER=/bin/evil-launcher]` (một dòng trong env của HOMECHILD, một trong HOMEEXEC).
- E2E: `bash scratchpad/p02b/p02b-e2e.sh /home/luannt/.cache/v3ev/p02b-e-before <agent-cli-before>`, ca LN (tmux riêng,
  LLM giả): `LN: Home after Alt+W ran repo/evil (launcher from .env): EXPOSED`.

**Sửa** (`internal/cmd/home.go`): `homeEnv()` bỏ mọi dòng `AGENT_CLI_LAUNCHER=` khỏi `ChildEnv(os.Environ())`, rồi gắn
lại giá trị của env gốc nếu có. Không thêm launcher vào `trustedKeys`: danh sách đó cũng là chính sách mà client gửi
server để so (`Inputs`), thêm vào sẽ đổi phép so.

**XANH:** unit PASS (lần chạy con thứ nhất không có dòng launcher; lần thứ hai env gốc có launcher thật thì mỗi env có
đúng một dòng và `homeLauncher` trả nó). E2E LN với `agent-cli-after`: `Home after Alt+W started '<binary> --cwd repo',
not repo/evil: NOT_EXPOSED`. Home e2e của P02 (`p02w/p02-home.sh`, launcher giả ở env gốc): `NOT_EXPOSED`, launcher gốc
vẫn được dùng (có `child.env`, `child2.env`).

**Kiểm răng:** M1 (không bỏ launcher của `.env`) → đỏ; M1b (không gắn lại launcher gốc) → đỏ.

## 2. Mục 2: lệnh `!cmd` người dùng gõ

**Vấn đề:** chế độ local `AppWorkspace.AgentRunShellCommand` không đặt `Env` (shell dùng `os.Environ()`, sau `.env`);
chế độ server `Backend.RunShellCommand` dùng `append(os.Environ(), ws.Env...)`, mà `ws.Env` là env client gửi lên
(`root.go:603`, chụp sau `.env`). `!agent-cli run …` nhận khoá tin cậy từ `.env`.

**ĐỎ trước khi sửa** (cùng lệnh `runtb.sh`): hai test mới chạy ở tiến trình con (`Snapshot` → `godotenv.Load` → chạy
`env` qua đúng hàm thật), `.env` có 5 khoá tin cậy + `PLAIN_CANARY`, chỉ in TÊN biến mang giá trị canary.
- `workspace TestShellCommandIgnoresProjectDotenvTrustedKeys` (local, `config.Load` với HOME/XDG cô lập): FAIL,
  `actual: [AGENT_CLI_ALLOWED_MODELS AGENT_CLI_CLAUDE_BIN AGENT_CLI_CODEX_SANDBOX AGENT_CLI_DELEGATION AGENT_CLI_ENV_FILE
  PLAIN_CANARY]`.
- `backend TestRunShellCommandIgnoresProjectDotenvTrustedKeys` (server, `ws.Env = os.Environ()` như client gửi): FAIL,
  cùng 6 tên.

**Sửa:** local `Env: execpolicy.ChildEnv(os.Environ())`; server `execpolicy.ChildEnv(append(os.Environ(), ws.Env...))`
(ảnh chụp của server; server chỉ nhận client có chính sách tin cậy bằng của nó). Sửa ở backend thay vì ở `root.go:603`:
một chỗ chặn cho mọi client.

**XANH:** cả hai PASS, chỉ còn `PLAIN_CANARY` (biến thường của `.env` vẫn tới lệnh `!`, như trước).

**Kiểm răng:** M2 (local về env sau `.env`) → đỏ; M3 (server bỏ `ChildEnv`) → đỏ.

## 3. Mục 3: env của CLI con claude/codex khi giao việc

**Vấn đề:** `provider/exec.go` dùng `childEnv(os.Environ())`: env sau `.env` (và sau `env` của `crush.json`, áp bằng
`os.Setenv`). Ở chế độ server, env GỐC của chính server đã mang `.env` của client, vì `startDetachedServer` truyền
`ChildEnv(env sau .env)` (chỉ khoá tin cậy được đặt lại).

**Mã cũ có lý do dùng env sau `.env` không?** Không tìm thấy. Comment chỉ nói giữ "biến cấu hình của người dùng (khoá,
provider)", tức biến export ở shell; env gốc vẫn giữ chúng. Đó là tiện lợi thừa hưởng từ autoload `.env` của upstream.
Hệ quả của việc đổi: ai đặt khoá cho Claude Code/Codex trong `.env` của repo (hay `env` của `crush.json`) thì CLI con
không còn nhận (R1).

**ĐỎ trước khi sửa:**
- Unit `provider TestRunCommandIgnoresProjectDotenv` (tiến trình con, gọi `runCommand` thật với CLI giả là script in
  `NAME:<biến>:<nguồn>`; env gốc có `OPENAI_API_KEY=CANARY_ORIGINAL`; sau `.env` còn mô phỏng `env` của `crush.json`
  đè `OPENAI_API_KEY=CANARY_CONFIGENV`): FAIL, `actual: [CODEX_HOME:dotenv OPENAI_BASE_URL:dotenv
  OPENAI_API_KEY:configenv CLAUDE_CONFIG_DIR:dotenv ANTHROPIC_BASE_URL:dotenv ANTHROPIC_API_KEY:dotenv]`.
- E2E ca CLI (`p02b-e2e.sh`, binary thật, LLM giả gọi `claude_code`, `run --json` tự duyệt; env gốc bật giao việc với
  `AGENT_CLI_CLAUDE_BIN` = CLI giả ngoài repo): `agent-cli-before` local và server đều nhận 5 biến nguồn `dotenv`,
  `CLI: EXPOSED`.

**Sửa:**
- `execpolicy.OriginalEnviron()` (mới): env gốc dạng `KEY=value`, duyệt theo env hiện tại: biến `.env` thêm thì không
  có, biến bị đè về giá trị gốc, biến tiến trình đã gỡ (`AGENT_CLI_HOME_CHILD`) không quay lại. Chưa chụp thì
  `os.Environ()`.
- `provider/exec.go`: `cmd.Env = childEnv(execpolicy.OriginalEnviron())` (vẫn bỏ biến phiên Claude Code và `HERDR_*`).
- `cmd/root.go` `startDetachedServer`: `c.Env = herdr.StripEnv(execpolicy.OriginalEnviron())`.
- "Các biến agent-cli tự đặt cho CLI con": không có. Biến công tắc không người trực (`exportUnattendedEnv`) không cần,
  vì khi không người trực tool giao việc không được đăng ký (`provider.NewTools`).

**XANH:** unit PASS (`[OPENAI_API_KEY:original]`). E2E `agent-cli-after`: local và server chỉ còn
`OPENAI_API_KEY:original`, `CLI: NOT_EXPOSED`. Đối chứng SV (`p02b-sv.sh`, server mode, LLM giả gọi bash): biến thường
của `.env` vẫn tới tool bash, before và after đều `PASS`.

**Kiểm răng:** M4 (CLI con dùng env sau `.env`) → unit đỏ. M5 (`startDetachedServer` về
`ChildEnv(StripEnv(os.Environ()))`) → unit không bắt (không có unit cho env dựng server), e2e bắt: `local: OK,
server: 5 biến dotenv, CLI: EXPOSED` (`p02b-e-m5`). M7 (`OriginalEnviron` duyệt map gốc thay vì env hiện tại) →
`TestOriginalEnvironDropsLaterChanges` đỏ.

## 4. Mục 4: unit `ensureServer` gặp server cũ đang bận (socket unix)

**Test mới** `TestEnsureServer_RefusesModelCommandOnBusyStaleServer`, hai ca `busy` và `legacy busy`: server giả trên
socket unix trả `VersionInfo` khác build. Lệnh chạy model → `errStaleServerBusy`; lệnh không prompt → nil (tái dùng);
server chỉ nhận hai lệnh `shutdown_if_idle`, socket còn nguyên.

**Đỏ/xanh:** mã đã có bản sửa từ P0-2, nên test xanh ngay trên `26bed1e9`. Răng: M6 (bỏ nhánh `errors.Is(err,
errStaleServerBusy)` trong `ensureServer`, tức M18 của P0-2 vốn chỉ e2e bắt) → đỏ cả hai ca, `Expected error … but
got nil` (không có server mới nào được dựng: thăm dò sức khoẻ trả 200 nên nhánh spawn bỏ qua).

## 5. Mục 5: `TestConnectToServerSendsNoAgentRuns` không ghi vào `~/.cache` thật

**ĐỎ:** test binary build từ cây sạch `26bed1e9`, chạy với HOME/XDG cô lập (`scratchpad/p02b/h5-base`), `-run`
`TestConnectToServerSendsNoAgentRuns` cùng hai test `TestRestartIfStale_*` (chúng ghi `slog.Warn`): tạo
`HOME/.cache/crush/server-tcp___127.0.0.1_<port>/crush.log`. Lý do: `ensureServer` gọi `crushlog.Setup` (một lần mỗi
tiến trình) với `GlobalCacheDir()`; test sau ghi log vào đó. Chạy riêng một mình test này thì không có dòng log nào nên
không thấy tệp.

**Sửa:** helper `isolateServerCache(t)`: `t.Setenv("CRUSH_CACHE_DIR", t.TempDir())` và trả lại `slog.Default()` khi test
xong. Bản đầu chỉ đặt env thì test sau dựng lại thư mục tạm đã xoá (đo được 2 thư mục `/tmp/TestConnectToServer…` thừa;
đã xoá). Dùng cả cho test mục 4. Không đặt `CRUSH_GLOBAL_DATA`/`CONFIG`: đo không thấy test ghi vào đó.

**XANH:** cùng lệnh với test binary mới: không tệp nào dưới HOME cô lập; không còn thư mục `/tmp/TestConnectToServer*`
hay `/tmp/TestEnsureServer*` mới sau khi chạy cả bộ.

## 6. Mục 6: docs

- `README.md:24`: thêm "trước đó, lệnh chạy model mới (TUI, `run`, `send`, `loop`) của dự án có server bản cũ đang bận bị
  từ chối".
- `README.md` mục "Repo lạ": lệnh `!` và agent-cli con không nhận biến ranh giới từ `.env`; CLI con giao việc chỉ nhận env
  gốc.
- `docs/agent-cli-local-setup.md`: mục "Sửa env khi agent-cli đang chạy" thêm launcher của Home, lệnh `!` và vế "verify
  của `routine run` nhận công tắc không người trực của chính lượt chạy" (mục 7); mục giao
  việc thêm gạch "CLI con nhận env GỐC…" (liệt kê biến, cách export, chế độ server). Dòng thêm ≤ 98 ký tự.
- `CHANGELOG.md` `[Unreleased]` → `### Sửa lỗi`: một gạch mới ngay sau gạch P0-2.

## 7. Mục 7: verify của `routine run` mất công tắc không người trực (hồi quy do P0-2)

**Vấn đề:** `routine run` gọi `SetUnattended(true, commit)` và `exportUnattendedEnv(commit)` để "MỌI tiến trình con" thấy
`AGENT_CLI_UNATTENDED=1` và `AGENT_CLI_UNATTENDED_COMMIT=1|0`. P0-2 cho verify của loop chạy với `ChildEnv(os.Environ())`.
`ChildEnv` đặt lại mọi khoá tin cậy theo ảnh chụp đầu `main`, nên hai công tắc vừa xuất bị thay bằng giá trị gốc
(thường là không có).

**ĐỎ trước khi sửa:**
- Unit `cmd TestVerifyCarriesTheCurrentUnattendedState` (tiến trình con có `Snapshot` và `godotenv.Load`; `.env` đặt cả
  hai công tắc `=1`; chạy `executeRoutine` thật với agent giả, `commit` false/true, cùng `shellVerify` ngoài routine):
  FAIL, `actual: [LOOP on=unset commit=unset, ROUTINE commit=false on=unset commit=unset, ROUTINE commit=true on=unset
  commit=unset]` (`scratchpad/p02b/red7.log`).
- Unit `execpolicy TestChildEnvCarriesTheCurrentUnattendedState`: FAIL (`elements differ`).
- E2E `bash scratchpad/p02b/p02b-routine-unattended.sh <dir> <bin>` (binary thật, LLM giả, HOME cô lập; verify ghi giá
  trị của đúng hai biến này):

| Binary | routine, không `--commit` | routine `--commit` | `loop --verify`, `.env` đặt cả hai `=1` | Kết quả |
|---|---|---|---|---|
| V2 `7cba42ff` (`p02-bin/agent-cli-before`) | on=1 commit=0 | on=1 commit=1 | on=1 commit=1 (lỗ `.env` mà P0-2 đã đóng) | FAIL |
| `26bed1e9` (`agent-cli-before`) | on=unset commit=unset | on=unset commit=unset | on=unset commit=unset | FAIL (hồi quy) |
| mục 1–6 (`agent-cli-after`) | on=unset commit=unset | on=unset commit=unset | on=unset commit=unset | FAIL |
| bản cuối (`agent-cli-after7`) | on=1 commit=0 | on=1 commit=1 | on=unset commit=unset | PASS |

Thư mục: `p02b-ru-v2`, `p02b-ru-before`, `p02b-ru-after`, `p02b-ru-after7`.

**Sửa** (`internal/execpolicy/policy.go`, `ChildEnv`):
- Khi `unattendedOn` (đặt bởi `SetUnattended`, chỉ có mã gọi, `.env` không chạm được) bật thì env con có
  `AGENT_CLI_UNATTENDED=1`.
- Cùng lúc đó có `AGENT_CLI_UNATTENDED_COMMIT="1"` nếu `unattendedCommit` bật, không thì `"0"`. Giá trị này khớp đúng
  thứ `exportUnattendedEnv` xuất.
- Không bật thì như cũ: theo ảnh chụp.
- Không dùng thẳng `UnattendedAllowsCommit()`. Hàm đó trả true khi ảnh chụp có `AGENT_CLI_UNATTENDED_COMMIT=1`, kể cả
  với routine không `--commit`; còn `exportUnattendedEnv` cố ý xuất `"0"` để giá trị đó không lọt sang routine không mở
  commit.

**Ngữ nghĩa ở từng nơi gọi:**
- `ChildEnv`:
  - verify của loop: trong routine là trạng thái của lượt; ngoài routine là ảnh chụp.
  - `!cmd` local: TUI không bao giờ gọi `SetUnattended`, nên vẫn theo ảnh chụp. (Home dùng `ChildEnv` tới vòng 2; từ
    vòng 2 dùng `OriginalEnviron`, Home không bao giờ chạy không người trực.)
  - `!cmd` server: server không bao giờ gọi `SetUnattended`, nên vẫn theo ảnh chụp.
- `OriginalEnviron`, không đổi:
  - CLI con giao việc: không được đăng ký khi không người trực (`provider.NewTools`).
  - dựng server: routine chạy agent in-process, không dựng server. Nếu sau này có, server sẽ nhận ảnh chụp chứ không
    nhận trạng thái của lượt (R8).

**XANH:** hai unit PASS; e2e `UNATTENDED-VERIFY: PASS`; mọi test `TestExecuteRoutine_*` và `TestScheduleEnv_*` PASS
(`green7.log`).

**Kiểm răng** (`mut.py`, hoàn nguyên và kiểm sha):
- M8 (bỏ nhánh trạng thái hiện hành) → test cmd đỏ. M8u (cùng mutant) → test execpolicy đỏ.
- M9 (tin giá trị công tắc trong env, kể cả `.env`) → test cmd đỏ.
- M10 (commit luôn `"1"`) → test cmd đỏ.

## 8. Vòng 2 (soát an ninh độc lập: KHONG-DAT)

### 8.1 #1 (phải sửa): luồng `agent-cli` trần → Home → chọn dự án

**Vấn đề (người soát chỉ ra):**
- `homeEnv()` = `ChildEnv(os.Environ())` chỉ đặt lại 11 khoá tin cậy và launcher. Biến thường của `.env` ở thư mục mở
  Home (`ANTHROPIC_BASE_URL`, `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`…, không export) đi vào env của workspace con.
- Con chụp các biến đó làm env GỐC. Từ đó:
  - `serverEnv` truyền chúng cho server;
  - `provider/exec.go` đưa chúng cho CLI con.
- Chế độ local qua Home cũng vậy.
- E2E ca CLI cũ chạy `run --json` thẳng trong repo, nên không đi qua đường này.

**ĐỎ trước khi sửa (e2e mới, viết trước):** `bash scratchpad/p02b/p02b-home-cli.sh <dir> <bin>`.
- Dựng như sau:
  - tmux riêng, LLM giả;
  - gõ `agent-cli` trần trong repo có `.env` (6 biến canary), Enter chọn dự án;
  - env gốc có `AGENT_CLI_LAUNCHER` trỏ launcher giả (như launcher thật). Launcher giả thay TUI của workspace: nó exec
    `<binary> run --json` trong ĐÚNG env mà Home đưa;
  - LLM giả gọi `claude_code`; CLI giả chỉ ghi tên biến kèm nguồn.
- Hai ca: HCL (local), HCS (server, `--host unix`).
- Kết quả trên `agent-cli-after7` (`p02b-hc-after7`): cả hai EXPOSED.
  - Env mà Home đưa có 5 biến từ `.env`.
  - CLI con nhận `ANTHROPIC_API_KEY`, `ANTHROPIC_BASE_URL`, `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `OPENAI_BASE_URL` (nguồn
    dotenv).

**ĐỎ unit:** `TestChildEnvsIgnoreProjectDotenv` với assert mới: `HOMECHILD: no variable of a project .env may reach it`,
`Should be empty, but was [PLAIN_FROM_DOTENV=kept]` (`scratchpad/p02b/r2red.log`).

**Sửa:**
- `internal/cmd/home.go`: `homeEnv()` = `execpolicy.OriginalEnviron()`.
- Con chạy cùng cwd với Home nên tự nạp lại đúng `.env` đó:
  - `homeChildCmd` không đặt `Dir`;
  - `execHome` `chdir(startDir)` rồi exec;
  - launcher thật không đổi cwd (`install-agent-cli.sh` chỉ `cd` trong subshell để tính root).
- `AGENT_CLI_HOME_CHILD` đã bị gỡ ở đầu `RunE` nên không quay lại; `homeChildCmd` tự gắn lại cho workspace.
- Đoạn xử lý riêng `AGENT_CLI_LAUNCHER` của mục 1 đã bỏ: `OriginalEnviron` trả launcher gốc, hoặc bỏ launcher mà `.env`
  thêm.

**XANH:**
- Unit PASS. HOMECHILD, HOMEEXEC, SERVER không có biến nào của `.env`. VERIFY vẫn có `PLAIN_FROM_DOTENV=kept` và không có
  khoá tin cậy. Launcher gốc có đúng một lần trong env của Home.
- E2E `agent-cli-r2` (`p02b-hc-r2`): HCL và HCS đều NOT_EXPOSED. Env mà Home đưa có 0 biến từ `.env`; CLI con chỉ nhận
  `OPENAI_API_KEY:original`.

**Kiểm răng:** M11 (`homeEnv` về `ChildEnv(os.Environ())`) → unit đỏ, `HOMECHILD: … [AGENT_CLI_LAUNCHER=/bin/evil-launcher
PLAIN_FROM_DOTENV=kept]`.

### 8.2 #2 (nên): `serverEnv()` có unit

- Tách `serverEnv()` (`root.go`) khỏi `startDetachedServer`, không đổi hành vi.
- Tiến trình con của `TestChildEnvsIgnoreProjectDotenv` in thêm mục SERVER.
- M12 (`serverEnv` về `ChildEnv(StripEnv(os.Environ()))`, tương đương M5) → unit đỏ:
  `SERVER: … [PLAIN_FROM_DOTENV=kept AGENT_CLI_LAUNCHER=/bin/evil-launcher]`. Trước đây M5 chỉ e2e bắt.

### 8.3 #3 (nên): docs

- README "Repo lạ" thêm giới hạn: agent-cli gọi lồng từ tiến trình đã nạp `.env` (`!agent-cli run …`,
  `--verify "agent-cli run …"`) chụp biến thường của `.env` đó làm env gốc, nên CLI con của nó nhận chúng.
- Guide, mục giao việc:
  - CLI con không nhận `env` của `crush.json`, "kể cả `crush.json` toàn cục của bạn" (R1);
  - workspace mà Home mở và server cũng dựng bằng env gốc;
  - gạch "Giới hạn" về agent-cli lồng;
  - gạch R2: `agent-cli loop --cwd <dir>` ở chế độ server chỉ nạp `.env` của `<dir>`. Đã grep: `loop` là lệnh duy nhất
    gọi `ResolveCwd` trước `connectToServer`; `dirs`, `models`, `session`, `stats` không dựng server.
- CHANGELOG: gạch P0-2b nói thêm "kể cả `crush.json` toàn cục", "cả khi mở qua Home", `loop --cwd`, và giới hạn của
  agent-cli lồng.

### 8.4 Còn mở (#4–#7, ghi lại theo điều phối, không sửa)

- **#4:** Home lồng qua `!agent-cli` (PTY). Lệnh `!` vẫn nhận biến thường của `.env`, kể cả `AGENT_CLI_LAUNCHER`, nên
  Home lồng có thể chạy launcher của `.env`. Cùng R4.
- **#5:** hook toàn cục (`hooks/input.go`) chạy với `os.Environ()` sau `.env`. Hook gọi agent-cli thì agent-cli đó chụp
  biến của `.env`.
- **#6:** `$EDITOR`, `$PAGER`, `LD_PRELOAD`… từ `.env` (biến mới, godotenv không đè biến có sẵn) vẫn tới mọi tiến trình
  con dùng env sau `.env`: tool bash, `!cmd`, verify, hook, MCP, pager của `session`. Workspace và server của Home nay
  không nhận chúng qua env, nhưng tự nạp `.env` nên tiến trình con của chúng vẫn có.
- **#7:** trước đây `coordinator` đặt `ANTHROPIC_API_KEY=""` trong tiến trình (khoá dạng `Bearer`/MiniMax), nên CLI con
  nhận khoá rỗng. Nay CLI con nhận `ANTHROPIC_API_KEY` gốc của người dùng nếu đã export. Đây là đổi hành vi, chưa đo
  với Claude Code thật.

## 9. Rủi ro

- **R1 (hành vi đổi, đã ghi docs vòng 2):** CLI con không còn nhận biến từ `.env` của repo, cũng không nhận từ `env` của
  `crush.json`, kể cả `crush.json` toàn cục của người dùng (không phân biệt được sau khi đã `os.Setenv`). Ai dựa vào đó phải export ở shell.
  Tệp `~/.config/agent-cli/env` vốn chỉ được crushrc `source` trong trình thông dịch, chưa bao giờ tới CLI con.
- **R2 (đã ghi docs vòng 2):** server nạp `.env` ở cwd của nó. Lệnh `chdir` trước khi dựng server (`loop --cwd`, và các lệnh gọi `ResolveCwd`
  trước `connectToServer`) nay chỉ có `.env` của thư mục `--cwd`. Trước đây có cả `.env` của thư mục gọi lệnh (qua env
  dựng server) lẫn của `--cwd`. TUI, `run`, `send` gọi `ensureServer` trước `ResolveCwd` nên không đổi.
- **R3:** biến rỗng mà `PushPopCrushEnv` để lại (khi khôi phục `os.Setenv(k, "")`), nếu có trước lúc dựng server, không
  còn qua server. Mới nghi là vô hại, chưa đo.
- **R9 (vòng 2):** workspace và Home mà Home dựng không còn nhận biến thường của `.env` qua env; chúng tự nạp `.env` ở
  cùng cwd. Biến mà `env` của `crush.json` đặt trong tiến trình Home (Home có nạp cấu hình không: chưa kiểm) cũng không
  còn qua.
- **R4:** `!cmd` vẫn nhận biến thường của `.env`, kể cả `AGENT_CLI_LAUNCHER`. `!agent-cli` trần chạy trong PTY nên có thể
  hiện Home với launcher của `.env`; Home cần phím nên mới nghi không khai thác được. Cách kiểm: gõ `!agent-cli` trong TUI
  ở repo có `.env` đặt launcher.
- **R5:** `PATH` không bị `.env` đè (godotenv không ghi đè biến có sẵn), nhưng `env` của `crush.json` dự án đè được
  `PATH`. `NewTools` gọi `exec.LookPath("claude")` trên `PATH` của tiến trình. Cùng mức tin cấu hình dự án như upstream
  (crushrc của repo đã chạy như Bash); chưa xử lý.
- **R6:** chỗ `os.Environ()` khác giữ nguyên như bảng §11.1 của P02 (hook, MCP, LSP, tool bash).
- **R8:** tiến trình con dựng bằng `OriginalEnviron` (server, CLI con) không mang trạng thái không người trực của lượt
  đang chạy. Hiện không có đường nào dựng chúng trong lượt routine (§7); nếu thêm thì phải gắn lại như `ChildEnv`.
- **R7:** `TestClientServerSetupRegistersWorkspace` (có từ trước) vẫn để lại thư mục `/tmp/TestClientServerSetup…` mỗi lần
  chạy cả gói `cmd` (đếm được 4 thư mục mới trong phiên này). Cùng nguyên nhân với mục 5; dùng `isolateServerCache` là
  hết. Chưa sửa (ngoài phạm vi).

## 10. Bảng chân trị

| Phép đo | before (`26bed1e9`) | after |
|---|---|---|
| Unit mục 1 `TestChildEnvsIgnoreProjectDotenv` | FAIL (2 dòng launcher `.env`) | PASS |
| Unit mục 2 local / server | FAIL / FAIL (5 khoá tin cậy) | PASS / PASS |
| Unit mục 3 `TestRunCommandIgnoresProjectDotenv` | FAIL (6 biến dotenv/configenv) | PASS |
| Unit `TestOriginalEnvironDropsLaterChanges` | (hàm chưa có) | PASS |
| Unit mục 4 `TestEnsureServer_…` | PASS (mã đã đúng); M6 đỏ | PASS |
| Mục 5: tệp dưới HOME cô lập | `crush.log` ở `.cache/crush/server-tcp…` | không tệp nào |
| E2E CLI local / server (`p02b-e-*`) | EXPOSED / EXPOSED | NOT_EXPOSED / NOT_EXPOSED (m5: server EXPOSED) |
| E2E LN (Alt+W → Home → launcher) | EXPOSED (`repo/evil` chạy) | NOT_EXPOSED (Home chạy `<binary> --cwd repo`) |
| E2E vòng 2 HCL / HCS (`agent-cli` trần → Home → dự án → `claude_code`) | after7: EXPOSED / EXPOSED (5 biến dotenv) | r2: NOT_EXPOSED / NOT_EXPOSED |
| Unit vòng 2 `TestChildEnvsIgnoreProjectDotenv` (HOMECHILD/HOMEEXEC/SERVER không biến `.env`) | after7: FAIL (`PLAIN_FROM_DOTENV=kept`) | r2: PASS |
| Đối chứng SV: tool bash server thấy biến thường `.env` | PASS | PASS |
| Home e2e P02 (`p02b-r-home`) | — | NOT_EXPOSED |
| `exposure.sh C5` (`p02b-r-c5`) | — | `EXPOSED — ENV=EXPOSED ALLOW=ok` (ENV có tài liệu) |
| `faults.sh` / `qwen-only.sh` / `workspaces.sh` | — | PASS / PASS / PASS (rc=0) |
| `test-agent-cli-run-json.sh` (`AGENT_CLI_BIN`) | — | PASS (rc=0) |
| Mục 7 unit `TestVerifyCarriesTheCurrentUnattendedState` / `TestChildEnvCarriesTheCurrentUnattendedState` | FAIL / FAIL | PASS / PASS |
| Mục 7 e2e `p02b-routine-unattended.sh` | FAIL (routine verify thấy unset) | PASS |
| `test-agent-cli-routine.sh` / `test-agent-cli-loop.sh` (`AGENT_CLI_BIN`, bản cuối) | — | PASS / PASS (rc=0) |
| Suite theo TÊN (`go test -json ./...`, cây chép không git, bản cuối vòng 2) | 4707 tên, FAIL 14 | 4716 tên; FAIL đúng 14 `TestV3Exposure_*` như cũ; mất 0; mới 9, đều pass; đổi trạng thái 0 |
| Gói (pass/fail/skip) | 64/4/17 | 64/4/17 (lần chạy mục 1–6), 4 gói fail chỉ do `TestV3Exposure_*` |
| `-race` cmd, backend, workspace, provider, execpolicy (bản cuối vòng 2) | — | 0 `DATA RACE`; backend FAIL chỉ do `TestV3Exposure_T1` |
| Vòng 2, binary `agent-cli-r2`: `p02b-e2e` (CLI, LN), SV, `p02b-routine-unattended`, faults, qwen-only, workspaces, run-json, C5, Home P02, routine, loop | — | NOT_EXPOSED, NOT_EXPOSED, PASS, PASS, PASS, PASS, PASS, PASS, `ALLOW=ok`, NOT_EXPOSED, PASS, PASS (`scratchpad/p02b/round2.txt`) |
| Rò giá trị env: log test/suite/race có giá trị biến dạng KEY/TOKEN/SECRET của máy | — | 0 (quét tự động, chỉ in số) |

**Mutant** (`mut.py`: mỗi mutant được hoàn nguyên, kiểm sha từng tệp; `git diff` của worktree trước/sau bằng nhau):

| Mutant | Thay đổi | Kết quả |
|---|---|---|
| M1 | `homeEnv` không bỏ launcher của `.env` | `ChildEnvs` đỏ (vòng 1; mã đã bỏ ở vòng 2) |
| M1b | `homeEnv` không gắn lại launcher gốc | `ChildEnvs` đỏ (vòng 1; mã đã bỏ ở vòng 2) |
| M11 | vòng 2: `homeEnv` về `ChildEnv(os.Environ())` | `ChildEnvs` đỏ (HOMECHILD có biến `.env`) |
| M12 | vòng 2: `serverEnv` về `ChildEnv(StripEnv(os.Environ()))` | `ChildEnvs` đỏ (SERVER có biến `.env`) |
| M2 | `!cmd` local dùng env sau `.env` | test workspace đỏ |
| M3 | `!cmd` server bỏ `ChildEnv` | test backend đỏ |
| M4 | CLI con dùng env sau `.env` | test provider đỏ |
| M5 | server dựng bằng `ChildEnv(env sau .env)` | e2e bắt: `p02b-e-m5` server EXPOSED (vòng 2: unit bắt, M12) |
| M6 | `ensureServer` bỏ nhánh `errStaleServerBusy` | `TestEnsureServer_…` đỏ (2 ca) |
| M7 | `OriginalEnviron` duyệt map gốc | `TestOriginalEnvironDropsLaterChanges` đỏ |
| M8 / M8u | `ChildEnv` bỏ trạng thái không người trực hiện hành | test cmd mục 7 đỏ / test execpolicy mục 7 đỏ |
| M9 | `ChildEnv` tin giá trị công tắc trong env (kể cả `.env`) | test cmd mục 7 đỏ |
| M10 | commit luôn `"1"` khi không người trực | test cmd mục 7 đỏ |

(Lần chạy đầu M2/M3 lỗi build vì import thừa; đổi mutant giữ import rồi chạy lại: đỏ.)

## 11. Chưa kiểm

- `task lint`.
- Claude Code/Codex THẬT với env gốc (`scripts/test-agent-cli-delegation.sh`): chỉ CLI giả. CLI thật đọc thêm cấu hình ở
  `~/.claude`, `~/.codex`; không đổi ở đây.
- Lệnh `!` trong TUI thật: chỉ unit gọi đúng hàm `AgentRunShellCommand`/`RunShellCommand`; chưa gõ `!` trong tmux.
- Launcher thật (`install-agent-cli.sh`) với ca LN: launcher thật export `AGENT_CLI_LAUNCHER` ở env gốc, nên `.env` không
  đè được (godotenv không ghi đè); ca LN chỉ có ở binary trần.
- R2/R3 chưa có test riêng.
- Vòng 2: chưa chạy launcher THẬT (`install-agent-cli.sh`) qua Home với `claude_code`; e2e dùng launcher giả đặt ở env
  gốc. Chưa kiểm Home có áp `env` của `crush.json` trong tiến trình không (R9).
- Hồi quy cũ (`faults`, `qwen-only`, `workspaces`, `run-json`, C5, Home P02, `p02b-e2e`, SV) chạy lại với
  `agent-cli-after7`: kết quả như bản `after` (PASS/NOT_EXPOSED, C5 `ALLOW=ok`). Không tệp `.go` nào đổi sau khi build
  `agent-cli-after7` (`find -newer`); cây chạy suite trùng worktree (`diff -rq`).
- Mục 7 không thêm dòng CHANGELOG: hồi quy và bản sửa đều nằm trong `[Unreleased]`; lời hứa của V2 ("routine run tự
  xuất biến cho mọi tiến trình con") nay đúng lại. Guide có thêm một vế về verify của `routine run`.

## 12. Lệnh tái lập

```bash
S=/tmp/claude-1000/-home-luannt-application/1ba1a44b-c74c-4a9d-b148-f58af7166a87/scratchpad/p02b
cd /home/luannt/application/opensource/agent-cli-p02b
# Unit mới (đỏ trên mã 26bed1e9 + test mới; xanh sau sửa). runtb.sh chạy với HOME/XDG cô lập, in mục "files written".
TB=$S/tb-x; mkdir -p $TB
$S/gp.sh go test -c -p 2 -o $TB/cmd.test ./internal/cmd/
$S/gp.sh go test -c -p 2 -o $TB/workspace.test ./internal/workspace/
$S/gp.sh go test -c -p 2 -o $TB/backend.test ./internal/backend/
$S/gp.sh go test -c -p 2 -o $TB/provider.test ./internal/agent/tools/provider/
bash $S/runtb.sh $TB <home-mới>
$S/gp.sh go test -count=1 -p 2 -run 'TestOriginalEnviron|TestChildEnvUsesTheSnapshot' ./internal/execpolicy/
# Răng:
python3 -I $S/mut.py <thư-mục-mới> [M1,M2,...]                # mục 7: M8,M8u,M9,M10
# E2E (binary thật, LLM giả, tmux riêng):
$S/gp.sh env CGO_ENABLED=0 go build -p 2 -o <bin> .
bash $S/p02b-e2e.sh /home/luannt/.cache/v3ev/<mới> <bin>     # CLI: NOT_EXPOSED, LN: NOT_EXPOSED (before: EXPOSED cả hai)
bash $S/p02b-sv.sh  /home/luannt/.cache/v3ev/<mới> <bin>     # SV: PASS
bash $S/p02b-routine-unattended.sh /home/luannt/.cache/v3ev/<mới> <bin>   # UNATTENDED-VERIFY: PASS (26bed1e9: FAIL)
bash $S/p02b-home-cli.sh /home/luannt/.cache/v3ev/<mới> <bin>   # vòng 2: HCL/HCS NOT_EXPOSED (after7: EXPOSED)
bash $S/round2.sh                                               # vòng 2: build agent-cli-r2 + mọi e2e/hồi quy + suite + race
AGENT_CLI_BIN=<bin> bash scripts/test-agent-cli-routine.sh; AGENT_CLI_BIN=<bin> bash scripts/test-agent-cli-loop.sh
bash $S/regress.sh <tiền-tố> <bin>                           # faults, qwen-only, workspaces, run-json, C5, Home P02
# Suite theo tên: chép cây (không .git) rồi go test -json ./..., so bằng python3 -I scratchpad/p02w/cmp.py base.json after.json
# Patch:
bash $S/mkpatch.sh reports/.../P02b-childenv.patch <thư-mục-kiểm-mới>
```

## Sau soát (điều phối, 19/10)

- Soát an ninh: KHONG-DAT (vòng 1: Home → CLI con vẫn lộ) → **DAT** (vòng 2). Điều phối chạy lại: `p02b-home-cli.sh` vòng 1
  EXPOSED (5 biến từ `.env`), vòng 2 NOT_EXPOSED; routine/loop/unattended/Home/C5 như các mục trước.
- Commit `61892d77` trên nhánh `v3-poc-p02b-childenv` (nối `v3-poc-p02-policy`); patch = `git diff v3-poc-p02-policy HEAD`.
- Việc sau (không chặn): Home mở dự án B nhưng nạp `.env` của thư mục mở Home (có từ V2; muốn sửa phải đặt `cmd.Dir` +
  launcher tuyệt đối); `child_env_test.go` đoạn SERVER thiếu đối chứng dương (assert có `PATH`, đủ tiêu đề).
