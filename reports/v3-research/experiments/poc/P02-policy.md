# P02: client sau chạy theo chính sách tin cậy của client đầu (ca C5)

Worktree `/home/luannt/application/opensource/agent-cli-p02`, nhánh `v3-poc-p02-policy`, gốc `v2-final` (`7cba42ff`). Chưa commit, chưa push.
Báo cáo có ba vòng: vòng 1 (§1–§9), **vòng 2** (§10) và **vòng 3** (§11, sau lượt soát an ninh độc lập chấm KHONG-DAT).
Số trong header là của vòng 3.

- **Patch mã:** `P02-policy.patch` (24 tệp: 10 tệp mã, 11 tệp test, 3 tệp docs; không gồm 6 tệp test phơi nhiễm `??`).
  `sha256sum -c P02-policy.patch.sha256` → `9b5036d6189b4bf5b13614f837231ef7b3c3cb6b06408d93cf5dc350a48e4547`.
- **Patch harness:** `P02-exposure-harness.patch`, không đổi từ vòng 2. Chỉ sửa `scripts/eval/exposure.sh`, diff so với
  bản dùng chung sha `d3f9e0fb…da6f`. `sha256sum -c P02-exposure-harness.patch.sha256` →
  `3881fee29606d627e69a92cdbbc9448cc8a4c74b7854d96dc2e9c1e68d225a8d`.
- **Đã kiểm trên cây sạch `7cba42ff` (git archive):**
  - patch mã `git apply --check` cho rc=0;
  - sau khi apply, 24/24 tệp giống hệt worktree (`cmp`);
  - `go test` 6 gói `backend execpolicy client server workspace cmd` đều `ok`.

**Cờ cho người soát (đọc trước):**
1. **C5:** phần ENV có tài liệu, chấp nhận; phần ALLOW đã sửa. Dòng verdict C5 vẫn in `EXPOSED` vì phần ENV (§10.1).
2. **Vòng 3 #1 (cao, có từ V2) đã tái hiện ĐỎ trước khi sửa, bằng e2e lẫn unit (§11.1):**
   - Repo có `.env` đặt `AGENT_CLI_DELEGATION=on` và `AGENT_CLI_ENV_FILE=repo/policy.env`.
   - Đi đường Home (Enter) hoặc TUI → Alt+W: tiến trình agent-cli con nhận 4 khoá tin cậy lấy từ `.env`, và lượt chạy
     của nó có tool `claude_code`. Đối chứng chạy thẳng trong repo thì không có.
   - Đã sửa ở `homeChildCmd`, `execHome`, `--verify` của loop. Thêm một chỗ cùng lớp: `homeLauncher` đọc
     `AGENT_CLI_LAUNCHER`, nên `.env` từng chọn được chương trình mà Home exec.
3. **#2:** lệnh chạy model gặp server khác build đang bận nay bị từ chối; `jobs` vẫn dùng được server đó. Đã chứng minh
   bằng e2e với server V2 thật (§11.2).
4. **#3:** test duyệt cả cây cobra cùng test đường dây của `connectToServer`. Mutant gán cứng `NoAgentRuns: true` làm unit
   đỏ (§11.3). Lần chạy ĐỎ đầu của test đường dây là đỏ sai (panic do server giả thiếu config); sửa test rồi chứng
   minh lại bằng mutant.
5. **Test cũ bị sửa:**
   - Vòng 1: 3 test (§4.2).
   - Vòng 3: `restart_stale_test.go`, hai test tái dùng chạy bằng lệnh không prompt (`noPromptCmd`) vì lệnh chạy model
     nay bị từ chối.
   - Vòng 3: thông điệp assert và comment ở `backend_test.go:475,482` (#11).
6. **Scratchpad dùng chung:** phiên này dùng chung scratchpad với agent khác. `g.sh` của tôi (cd vào p02) bị một agent
   khác ghi đè lúc 19:18 bằng bản trỏ `agent-cli-p1c`. Từ vòng 3, mọi tệp của tôi nằm trong `scratchpad/p02w/`. Đã kiểm:
   các tệp đổi trong p02 đều là của tôi (§11.6).

## 1. Vấn đề

- **Triệu chứng (C5, tái hiện trên binary trước sửa, `/home/luannt/.cache/v3ev/p02-1`):**
  `ENV=EXPOSED ALLOW=EXPOSED`. Client sau đặt `AGENT_CLI_ALLOWED_MODELS=qwen:m-qwen`, dự án chọn `pp/m-pp`, và lượt của nó
  vẫn đi `['pp']` với `err=''`. Đối chứng (client đó tự dựng server) thì bị chặn: `['none']`.
- **Gốc (đã kiểm lại trước khi sửa):**
  - Chính sách được nạp một lần cho mỗi tiến trình. `execpolicy.current` là `sync.OnceValues`, đọc `TrustedEnv` (ảnh chụp
    env gốc đầu `main`) rồi tệp env.
  - Server lấy env từ client đã dựng nó: `startDetachedServer` gán `c.Env = execpolicy.ChildEnv(...)`.
  - `Backend.CreateWorkspace` không biết chính sách của client gọi tới. Hai nhánh workspace trùng chỉ kiểm channel rồi
    `logFirstWinsMismatch` (log debug). YOLO lệch chỉ được cảnh báo phía client (`yoloMismatchWarning`).
- **Rộng hơn mô tả ban đầu:** chính sách gắn với cả tiến trình server, không riêng một workspace. Client có chính sách khác
  mở dự án KHÁC trên cùng server cũng chạy theo chính sách của server. Nên phải kiểm ở mọi lần tạo workspace, không chỉ ở
  nhánh workspace trùng. Có test riêng cho ca này (`...OnNewWorkspace`).
- **Cùng lỗ, ngoài allowlist:** `AGENT_CLI_UNATTENDED`, `AGENT_CLI_LLM_LEDGER`, `AGENT_CLI_CLAUDE_BIN`/`CODEX_BIN`/mức quyền
  đều được server đọc qua `TrustedEnv` của chính nó. Client sau đặt khác thì bị bỏ qua lặng lẽ. Bản sửa bao luôn các biến này.

## 2. Thiết kế

**Client gửi gì.**
- `proto.Workspace` có sẵn `Env` (= `os.Environ()` SAU `.env`), nhưng không dùng được: `.env` của repo chọn được giá trị.
- Thêm trường `TrustedEnv []string` (`json:"trusted_env"`), lấy từ hàm mới `execpolicy.Inputs()`. Trường này:
  - liệt kê đủ 11 khoá của `trustedKeys`, dạng `KEY=value`, theo thứ tự cố định, khoá không đặt thì rỗng;
  - với `AGENT_CLI_ALLOWED_MODELS`/`AGENT_CLI_DELEGATION`, lấy giá trị đã gộp tệp env đúng như `Current` đọc. Các khoá còn
    lại theo ảnh chụp.
- `Inputs` nạp CÙNG lần với `Current`: `loadAll` đọc tệp env một lần cho cả hai. Nhờ vậy, ở server nó đúng là chính sách
  server đang áp. Nếu tệp env bị sửa sau khi server nạp, server vẫn so với cái nó đã nạp.
- Gán trường này trong `client.CreateWorkspace`, cạnh `ClientID`. Đây là điểm duy nhất mọi client đi qua (gồm `connectToServer`
  và đường hồi phục `ClientWorkspace.recreateArgs`).

**Server so với gì.** `checkTrustedEnv` chạy ở đầu `CreateWorkspace`, sau `validateClientID`, trước khi khoá và trước
`cancelShutdownLocked`. Client bị từ chối nên không huỷ được lịch tắt server.
- So `execpolicy.Inputs()` của chính tiến trình server với `args.TrustedEnv`, bằng `envKeysDiffer` có sẵn (chỉ trả tên).
- Lệch thì trả `ErrPolicyMismatch` (sentinel, theo mẫu `ErrChannelOptInMismatch`), kèm tên biến lệch và cách sửa:
  `trust policy differs from the running agent-cli server: AGENT_CLI_ALLOWED_MODELS; a running server keeps the policy it started with: exit every agent-cli client of this project (TUI, run, loop) and reopen once its server has exited (about a minute later, see agent-cli ps)`.
- Server tự lỗi nạp chính sách (tệp env hỏng) cũng từ chối, cùng lỗi.

**YOLO.** Ở hai nhánh workspace trùng (nhánh tra đầu và nhánh tra lại sau lúc khởi tạo chậm), cạnh phép kiểm channel:
`Cfg.Overrides().SkipPermissionRequests != args.YOLO` thì từ chối với tên `--yolo`.
- Điều kiện này đúng bằng điều kiện của cảnh báo cũ (`yoloMismatchWarning` so với `proto.YOLO` = Overrides).
- YOLO thuộc từng workspace, không thuộc cả server. Nên workspace MỚI cứ dùng `--yolo` của client tạo nó, không cần so.

**Client cũ (không gửi `trusted_env`): fail-closed** với lỗi `trust policy not sent (older agent-cli client)`. Client và
server là cùng một binary (server dựng từ `os.Executable()`), nên lệch chỉ xảy ra khi khác phiên bản.

**Phương án đã bỏ:**

| Phương án | Vì sao bỏ |
|---|---|
| Server tự nạp lại / server riêng cho mỗi chính sách | Người dùng cấm |
| So `Env` (`os.Environ()`) có sẵn | Có cả giá trị `.env` của repo, nên repo chọn được kết quả so. Client cũ và mới cũng không phân biệt được |
| Chỉ so env thô, không gộp tệp env | Sửa allowlist trong `~/.config/agent-cli/env` khi server đang chạy (đúng ca docs bảo "mở lại") sẽ lọt |
| Client tự so (server trả chính sách của nó) | Workspace mới đã được dựng xong (config, db, app) rồi mới huỷ; client cũ không kiểm gì. Người dùng chọn mẫu từ chối phía server |
| YOLO so trạng thái lúc chạy (`Permissions.SkipRequests()`) | Đã viết thử rồi bỏ: `ctrl+y` ở TUI sẽ làm mọi `run`/`send` sau bị từ chối. Mà README:70 ghi `ctrl+y` "áp cho cả workspace, kể cả send" là thiết kế |
| Ánh xạ lỗi sang 409 trong `handleError` | `ErrChannelOptInMismatch` (mẫu được chỉ định) cũng không ánh xạ (500). Thông điệp vẫn tới client nguyên văn |

## 3. Tệp đổi (mã)

| Tệp | Đổi |
|---|---|
| `internal/proto/proto.go` | thêm `TrustedEnv []string \`json:"trusted_env,omitempty"\`` |
| `internal/execpolicy/policy.go` | `current` trả `loaded{policy, inputs}`; thêm `Inputs()`, `loadAll`, `inputsOf`. `load` giữ nguyên chữ ký (test cũ không đổi). Chế độ test: `Inputs` kín, mọi khoá rỗng, giống `Current` trả `Policy{}` |
| `internal/client/proto.go` | `CreateWorkspace` gán `ws.TrustedEnv = execpolicy.Inputs()` (lỗi thì trả lỗi) |
| `internal/backend/backend.go` | `ErrPolicyMismatch`; gọi `checkTrustedEnv` đầu `CreateWorkspace`; kiểm `--yolo` ở 2 nhánh workspace trùng; thêm `checkTrustedEnv`, `policyMismatch` |
| Docs | `docs/agent-cli-local-setup.md` mục "Sửa env khi agent-cli đang chạy": câu cũ "nạp lại mỗi khi một workspace mới mở" chỉ đúng với cấu hình. Nay tách cấu hình (đọc khi dựng workspace) khỏi chính sách tin cậy (nạp MỘT lần mỗi tiến trình server), nêu lỗi từ chối, và thêm bước chờ server tắt. Mục giao việc (~dòng 604) và `README.md` (dòng 267, cùng dòng YOLO 70) cũng sửa. `CHANGELOG.md` `[Unreleased]`/Sửa lỗi thêm một mục |

## 4. Test

### 4.1 Test mới

| Test | Chứng minh |
|---|---|
| `backend TestCreateWorkspace_TrustPolicyMismatch` (7 ca) | cùng chính sách → gắn vào; khác biến thường (`V3_ROUTE`) → gắn vào; khác allowlist, giao việc, biến tin cậy khác (`AGENT_CLI_UNATTENDED`), `--yolo`, hoặc không gửi chính sách → `ErrPolicyMismatch`, lỗi có tên biến và "exit every agent-cli client of this project", KHÔNG có giá trị (`qwen:m-qwen`), client bị từ chối không được đăng ký |
| `backend ...TrustPolicyMismatchOnNewWorkspace` | chính sách lệch ở dự án KHÁC trên cùng server bị từ chối; không dựng workspace nào |
| `backend ...RuntimeYOLOToggleNotRefused` | `ctrl+y` (SetPermissionsSkip) không làm client sau bị từ chối. Đây là đối chứng, nên trước sửa cũng xanh |
| `backend ...ParallelCreatesRefuseOtherYOLO` | 2 create song song, `--yolo` khác nhau: đúng một được giữ, cái kia `ErrPolicyMismatch` (phủ nhánh tra lại sau khởi tạo chậm) |
| `execpolicy TestInputsUseSnapshotAndEnvFile` | `.env` (đặt sau ảnh chụp) không đổi `Inputs`; tệp env được gộp; export thắng tệp; đủ 11 khoá; dòng khác của tệp env (khoá API) không lọt |
| `execpolicy TestInputsHermeticInTests` | trong test `Inputs` kín (mọi khoá rỗng), nên client và server cùng tiến trình test luôn khớp |

### 4.2 Test cũ bị sửa (do hành vi đổi có chủ ý, không phải để che lỗi)

- `backend_test.go TestPathDedupe_FirstWinsKeepsOriginalEnv`: `argsB.YOLO = false` → `true`. Test vẫn kiểm Env/Debug
  first-wins. YOLO lệch nay bị từ chối và được kiểm ở test mới.
- `backend_test.go TestFirstWinsMismatch_LogsOnFlagDifferences`: bỏ ca `yolo` (YOLO không còn first-wins). Sửa comment.
  Ca này là tên test duy nhất "mất" khi so theo tên (§5.6).
- `env_log_test.go TestFirstWinsMismatch_NeverLogsEnvValues`: bỏ `argsB.YOLO = true`. Test chỉ kiểm việc không ghi giá trị
  env, và vẫn bắt được log ở mức Debug.
- Thêm `TrustedEnv` (như client thật gửi) vào các chỗ dựng request trực tiếp: `protoWS` (helper),
  `unavailable_model_test.go`, `backend_skills_test.go`, `server/e2e_test.go postWorkspace`.

## 5. Bảng chân trị

Binary: trước = `/home/luannt/.cache/v3ev/p02-bin/agent-cli-before` (`7cba42ff`); sau = `.../agent-cli-final` (worktree hiện
tại). `.../agent-cli-after` là bản trung gian (YOLO so trạng thái lúc chạy), chỉ dùng ở §5.4 và phép đo đầu của §5.2.

| Phép đo | Trước sửa | Sau sửa |
|---|---|---|
| Unit mới (backend) | ĐỎ: 5 ca refusal + `OnNewWorkspace` + `ParallelCreates` (`Expected error ... but got nil`) | XANH: 4 test backend (11 tên kể cả 7 ca con) + 2 test execpolicy |
| C5 ALLOW: client sau qwen-only | `['pp']`, `err=''` → EXPOSED | `['none']`, `trust policy differs...` → ok |
| C5 ENV: client sau `V3_ROUTE=c2` | `['c1']` → EXPOSED | `['c1']` → EXPOSED (theo thiết kế, R1) |
| C5 dòng verdict | `EXPOSED — ENV=EXPOSED ALLOW=EXPOSED` | `EXPOSED — ENV=EXPOSED ALLOW=ok` |
| ĐC s1 cùng chính sách (qwen/qwen) | chạy, `['qwen']` | chạy, `['qwen']` |
| ĐC s2 khác biến thường (`FOO=bar`) | chạy, `['pp']` | chạy, `['pp']`, không bị từ chối |
| ĐC s3 `.env` dự án đặt allowlist qwen + DELEGATION=on | chạy, `['pp']` | chạy, `['pp']`, không bị từ chối (`.env` không vào phép so) |
| ĐC s4 export allowlist khác | chạy `['pp']` (FAIL) | rc=1, 0 request, lỗi nêu `AGENT_CLI_ALLOWED_MODELS` |
| ĐC s5 TUI `--yolo` (tmux riêng) + `run` | chạy `['pp']` dưới YOLO (FAIL) | rc=1, 0 request, lỗi nêu `--yolo` |
| `faults.sh` | PASS rc=0 | PASS rc=0 |
| `qwen-only.sh` | PASS rc=0 | PASS rc=0 |
| `workspaces.sh` | PASS rc=0 | PASS rc=0 |
| `scripts/test-agent-cli-run-json.sh` (local + clientserver) | chưa chạy | PASS rc=0 (40 dòng ok) |
| `exposure.sh` đủ ca (16 dòng verdict) | 14 EXPOSED + T21 MEASURED + T23 NOT_EXPOSED; C5 ALLOW=EXPOSED; T6 có finding "plan" | verdict từng ca giống trước, trừ C5 ALLOW=ok; T6 mất phép đo phụ "plan" (§5.4) |
| Suite `go test -count=1 -p 2 ./...` | FAIL đúng 5 `TestV3Exposure_*` (14 tên kể cả ca con) | cùng 14 tên đó; xem §5.6 |
| `-race ./internal/backend/... ./internal/cmd/...` | chưa chạy | 0 `DATA RACE`; backend FAIL chỉ do `TestV3Exposure_T1` (có sẵn); cmd, clientserverrace ok |

### 5.1 Thư mục chứng cứ (`/home/luannt/.cache/v3ev/`)

| Thư mục | Nội dung |
|---|---|
| `p02-1` | C5 trước sửa |
| `p02-2` | C5 trên `after` |
| `p02-5` | C5 trên `final` |
| `p02-4` | đối chứng, binary trước |
| `p02-3` | đối chứng, binary `after` |
| `p02-6` | đối chứng, binary `final` |
| `p02-b-{faults,qwen-only,workspaces}` (+ `.out`) | ba script eval, trước sửa |
| `p02-f-{faults,qwen-only,workspaces}` (+ `.out`) | ba script eval, sau sửa |
| `p02-f-runjson.out` | run-json |
| `p02-x-before`, `p02-x-after` | exposure đủ ca |

### 5.2 Lưu ý về độ phủ e2e

- `qwen-only.sh` và `workspaces.sh` chạy local (`CRUSH_CLIENT_SERVER=0`), nên không đi qua đường sửa. Chúng chỉ là kiểm hồi quy.
- `faults.sh` có một ca qua server (`selected-no-key-server`).
- Đường server được phủ bởi C5, đối chứng s1–s5, run-json (clientserver) và các ca server của `exposure.sh`.

### 5.3 Kiểm răng hai chiều (mutant, hoàn nguyên tự động, `sha256sum -c` OK sau mỗi lần)

| Mutant | Kết quả |
|---|---|
| M1: bỏ `checkTrustedEnv` | FAIL 4 ca con (allowlist, delegation, other_trusted_var, older_client) + test cha + `OnNewWorkspace` |
| M2: bỏ kiểm `--yolo` ở nhánh tra đầu | FAIL `yolo_differs` |
| M3: bỏ kiểm `--yolo` ở nhánh tra lại | `ParallelCreatesRefuseOtherYOLO` FAIL 30/30 lần (`-count=30`) |
| M4: `Inputs` đọc `os.Getenv` (env sau `.env`) | FAIL `TestInputsUseSnapshotAndEnvFile` |
| M5: client không gửi `TrustedEnv` | 7 FAIL ở `internal/workspace` (client HTTP thật ↔ server) |
| e2e | binary trước = "không có bản sửa": s4, s5 FAIL; C5 `ALLOW=EXPOSED` |

### 5.4 Phát hiện phụ từ exposure đủ ca

- Không ca nào khác bị từ chối oan. `grep "trust policy differs"` trên `p02-x-after` chỉ trúng C5 và T6.
- T6 (`exposure.sh:410-415`) tạo workspace bằng `curl` không có `trusted_env`, nên nay bị từ chối (fail-closed). Hậu quả:
  bước bật plan và `run` "T6-plan" không đo được nữa. Dòng T6 sau sửa thiếu finding `non-interactive plan names question`
  mà bản trước có.
- Tôi không sửa harness: đó là tệp test phơi nhiễm. Muốn đo lại T6-plan, harness cần gửi `trusted_env` khớp server (với env
  cô lập của ca: `HOME=$P/home`, các khoá còn lại rỗng), hoặc tạo workspace qua một client.

### 5.5 Baseline suite

- `suite-before.json` chạy trên cây `7cba42ff` cùng 6 tệp phơi nhiễm, song song với lúc tôi bắt đầu sửa.
- Đối chiếu giờ start của từng gói: chỉ trường `proto.TrustedEnv` (thêm field, không có logic) có thể đã lọt vào lần build
  baseline. `execpolicy` và `backend` build trước khi tôi sửa chúng: đều PASS/FAIL đúng như cây sạch, không có lỗi build.

### 5.6 So suite theo TÊN (`cmp.py`, assert số tên > 0)

| | Số tên |
|---|---|
| Trước sửa | 4684 |
| Sau sửa | 4696 |
| FAIL trước | 14 (5 `TestV3Exposure_*` + ca con) |
| FAIL sau | đúng 14 tên đó |
| Đổi trạng thái | 0 |
| Mất | 1: `TestFirstWinsMismatch_LogsOnFlagDifferences/yolo` (bỏ có chủ ý, §4.2) |
| Mới | 13, đều pass |

## 6. Lệnh tái lập

```bash
cd /home/luannt/application/opensource/agent-cli-p02
G='flock /tmp/claude-1000/agent-go.lock env GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off'   # và free -m avail ≥ 3000
$G go test -count=1 -p 2 -v ./internal/backend/ -run 'TestCreateWorkspace_TrustPolicy|RuntimeYOLO|ParallelCreatesRefuseOtherYOLO'
$G go test -count=1 -p 2 ./internal/execpolicy/ -run TestInputs
$G env CGO_ENABLED=0 go build -p 2 -o /tmp/p02-bin .
bash scripts/eval/exposure.sh /home/luannt/.cache/v3ev/p02-N /tmp/p02-bin C5      # thư mục mới, ≤ 70 byte
bash <đối chứng ở Phụ lục> /home/luannt/.cache/v3ev/p02-M /tmp/p02-bin           # CONTROLS: PASS
for s in faults qwen-only workspaces; do bash scripts/eval/$s.sh /home/luannt/.cache/v3ev/p02-$s-N /tmp/p02-bin; done
AGENT_CLI_BIN=/tmp/p02-bin bash scripts/test-agent-cli-run-json.sh
$G go test -count=1 -p 2 -json ./... > after.json      # so theo tên với bản chạy trên 7cba42ff
$G go test -race -count=1 -p 2 ./internal/backend/... ./internal/cmd/...
# Đỏ: e2e dùng binary build từ 7cba42ff (C5, đối chứng); unit dùng mutant M1–M5 ở §5.3 (mỗi mutant gỡ một phần bản sửa).
```

## 7. Ngoài phạm vi (không làm)

- Server tự nạp lại; server riêng cho mỗi chính sách.
- Ánh xạ HTTP 409.
- Tự thay server rảnh khi lệch chính sách (`ensureServer` đã có cơ chế thay server khi lệch phiên bản; có thể mở rộng sau).
- Sửa harness `exposure.sh`.

## 8. Rủi ro

- **R1 (C5 ENV vẫn EXPOSED):** biến thường (vd biến mà `crush.json` expand trong `base_url`/`api_key`) vẫn theo env của
  client đã dựng server. Không cảnh báo, chỉ có log debug `env_keys_differ`. Đóng lỗ này cần quyết định: từ chối khi khác
  ở biến mà cấu hình dự án thực sự tham chiếu, hay cảnh báo.
- **R2 (YOLO):** TUI mở bằng `--yolo` làm `run`/`send`/`loop`/`jobs`/`logout` vào cùng dự án luôn bị từ chối, vì các lệnh này
  không có cờ `--yolo`. `jobs`/`logout` không chạy model mà vẫn bị chặn, nên với chúng đây là từ chối oan. Lối ra: mở TUI
  không `--yolo` rồi `ctrl+y`. Nếu không chấp nhận, đổi cách so: chỉ so `--yolo` khi client có cờ đó, hoặc chỉ từ chối
  chiều "workspace YOLO mà client không xin".
- **R3 (API thô):** mọi `POST /v1/workspaces` không có `trusted_env` bị từ chối (T6, script của người dùng nếu có).
- **R4 (server đang tắt dần):** sau khi thoát hết client, server còn sống khoảng 60 giây (`CRUSH_SERVER_IDLE_TIMEOUT`). Mở lại
  ngay thì vẫn bị từ chối; thông điệp và docs đã nói "chờ server tắt / `agent-cli ps` / kill".
- **R5 (so chuỗi thô):** `qwen:a,pp:b` và `pp:b qwen:a`, hay `on` và `1`, cùng nghĩa nhưng bị coi là khác (fail-closed, từ
  chối thừa). `HOME`/`AGENT_CLI_ENV_FILE` khác nhau cũng bị từ chối dù allowlist hiệu lực bằng nhau. Chủ ý: `HOME` quyết định
  cấu hình toàn cục mà server dùng.
- **R6 (client mới → server cũ):** server cũ bỏ qua `trusted_env`, nên lỗ vẫn còn tới khi server cũ tắt. Client chỉ có cảnh
  báo lệch phiên bản sẵn có (khi tạo workspace mới).
- **R7 (đường hồi phục):** `ClientWorkspace.recoverWorkspace` gặp server mới có chính sách khác thì bị từ chối và thử lại
  theo backoff mãi (fail-closed, có log lỗi).
- **R8 (dead code):**
  - `cmd.yoloMismatchWarning` và nhánh `slog.Warn` cho YOLO trong `logFirstWinsMismatch` gần như không còn chạy với server
    mới (chỉ còn với server cũ). Tôi giữ nguyên, không xoá.
  - `docs/UPSTREAM-README.md:521` (first-wins của `--yolo`) là tài liệu upstream, tôi không sửa.

## 9. Chưa kiểm

- Đường hồi phục (R7) với server thật có chính sách khác: chưa chạy e2e.
- Launcher thật (`la`, `install-agent-cli.sh`) với hai client khác chính sách: chưa chạy riêng. Các ca launcher của
  `exposure.sh` (T2c, T4) vẫn ra verdict như trước và không bị từ chối oan.
- `routine run` (in-process, export `AGENT_CLI_UNATTENDED=1` cho tiến trình con): theo mã, client con gắn vào server của dự
  án sẽ bị từ chối. Chưa chạy.
- `-race` chỉ trên `backend` và `cmd` như brief, không chạy bộ race đầy đủ ở CLAUDE.md P5. Chưa chạy `task lint`.
- `gofmt` (GOROOT) và `gofumpt -l` sạch trên các tệp đã đổi.

## 10. Vòng 2 (P0-2: C5 ENV ghi tài liệu, YOLO không từ chối oan, harness T6)

### 10.1 C5 ENV: giữ hành vi V2, ghi tài liệu

Không đổi mã. Guide (mục "Sửa env khi agent-cli đang chạy"), `README.md:268-270` và mục CHANGELOG nay nói rõ ba điều:
- chỉ 11 biến tin cậy bị so (liệt kê đủ tên trong guide);
- biến thường, kể cả biến mà `crush.json` dùng qua `$VAR` cho `api_key`/`base_url`, KHÔNG bị so: client vẫn gắn vào
  và chạy theo giá trị của client đã dựng server;
- đây là cùng loại giới hạn V2 đã ghi (allowlist chỉ so tên provider/model, không kiểm `base_url`).

Trạng thái: C5 ENV = có tài liệu, chấp nhận. C5 ALLOW = đã sửa.

### 10.2 YOLO (và chính sách) không từ chối lệnh không gửi prompt

**Đọc mã: lệnh nào tạo hoặc gắn workspace trên server.** Chỉ có `connectToServer` (`internal/cmd/root.go`) và đường hồi
phục `ClientWorkspace.recoverWorkspace` gọi `client.CreateWorkspace`.

| Lệnh | Đường vào | Gửi prompt? | Bị so? |
|---|---|---|---|
| TUI (`agent-cli`) | `setupWorkspaceWithProgressBar` → `setupClientServerWorkspace` → `connectToServer` | có | so |
| `run` | `run.go:160` `connectToServer` | có | so |
| `send` | `send.go:76` | có (gửi prompt vào session) | so |
| `loop` | `loop.go:388` `newLoopAgent` (khi qua server) | có | so |
| `jobs`, `jobs output`, `jobs kill` | `jobs.go:98` | không: chỉ liệt kê, xem output, dừng job bash | KHÔNG |
| `logout` | `logout.go:59` | không: xoá credential của provider | KHÔNG |
| `login` | `login.go:53` → `setupClientServerWorkspace` | không: OAuth/ghi key. Có `InitCoderAgent` nếu đã cấu hình, nhưng server dựng model theo chính sách của chính server, không gửi prompt | KHÔNG |

Các lệnh còn lại không tạo workspace, nên không bị ảnh hưởng:
- `ps`, `home`: chỉ `Health`/`ListWorkspaces`/`ListSessions`.
- `logs`, `dirs`, `models`, `session`, `stats`, `projects`, `keys`, `schema`, `update-providers`: không gọi server tạo
  workspace (grep `connectToServer|CreateWorkspace` chỉ ra 6 chỗ gọi kể trên).
- `routine run`: chạy agent in-process, không qua server.

**Vì sao bỏ luôn phép so chính sách tin cậy (không chỉ YOLO) cho lệnh không prompt:** cả 11 biến chỉ tác động lúc chạy
model hoặc tool:
- allowlist/giao việc: khi dựng model và gọi tool giao việc;
- `UNATTENDED*`: lớp chặn lệnh bash của agent;
- `LLM_LEDGER`: lời gọi LLM;
- `CLAUDE/CODEX_*`: tool giao việc;
- `HOME`/`ENV_FILE`: chỉ để tìm tệp env của chính sách.

`jobs`/`login`/`logout` không chạy model hay tool nào, nên so thì chỉ đẻ ra từ chối oan (đã đo: §10.4, vòng 1 từ chối
`jobs`/`logout`).

**Mã:**
- `proto.Workspace.NoAgentRuns` (`json:"no_agent_runs,omitempty"`).
- `cmd`: annotation `noAgentRunsAnnotation` gắn trên `jobsCmd`, `loginCmd`, `logoutCmd`. `runsNoAgent(cmd)` dò cả lệnh cha,
  nên phủ `jobs output`/`jobs kill`. `connectToServer` gửi `NoAgentRuns` và không in cảnh báo YOLO cho các lệnh này.
- `backend`: `checkTrustedEnv(args)` trả nil khi `NoAgentRuns`. Hai phép so `--yolo` thêm `!args.NoAgentRuns &&`.
- Mặc định là `false`, nghĩa là bị so. Client cũ và API thô không gửi trường này nên vẫn fail-closed.

**Test mới:**
- `backend TestCreateWorkspace_TrustPolicyMismatch`, thêm 2 ca:
  - `no-agent client, yolo differs` → gắn vào;
  - `no-agent client, allowlist differs` → gắn vào.
  - Phía model, các ca `yolo_differs`/`allowlist_differs` vẫn bị từ chối như cũ.
- `cmd TestRunsNoAgentMarksOnlyCommandsWithoutPrompts`, trên cây lệnh THẬT:
  - `jobs`, `jobs output`, `jobs kill`, `login`, `logout` → true;
  - `agent-cli` (root), `run`, `send`, `loop` → false.

**Đỏ → xanh:**
- Trước khi cài (stub `runsNoAgent` trả false, backend chưa miễn): 2 ca backend mới FAIL (`Received unexpected error`), và
  test cmd FAIL ở `agent-cli jobs`.
- Sau khi cài: tất cả PASS.

### 10.3 Harness T6 (`scripts/eval/exposure.sh`, chỉ bản p02)

- Request `POST /v1/workspaces` bằng curl nay mang `trusted_env`: đủ 11 khoá như `execpolicy.Inputs`. Env của ca chỉ đặt
  `HOME=$P/home`, các khoá khác rỗng, không có tệp env.
- Danh sách khoá chép tay, comment trỏ tới `trustedKeys` trong `internal/execpolicy/policy.go`.
- Thêm dòng `ws-create (API, for plan): ok|FAILED <message>` vào summary của T6. Nếu danh sách khoá lệch sau này, việc mất
  phép đo plan sẽ hiện ra, không im lặng như lần trước.
- Không đổi logic verdict.

| Binary | Kết quả T6 (harness đã sửa) |
|---|---|
| r2 | `ws-create ... ok`, `set-plan HTTP 200`, có request role `plan`, finding `non-interactive plan names question (rare combination)` có mặt lại, T6 `EXPOSED` như trước |
| trước sửa | cùng kết quả (server cũ bỏ qua `trusted_env`), nên harness tương thích ngược |

Vòng 1 (harness cũ, binary final): `plan: no request` và không có finding plan.

### 10.4 Bảng chân trị vòng 2

Binary:
- `before` = `7cba42ff`;
- `final` = vòng 1;
- `r2` = `/home/luannt/.cache/v3ev/p02-bin/agent-cli-r2`, cây hiện tại;
- `m11` = r2 nhưng `connectToServer` gửi `NoAgentRuns: false`.

Đối chứng chạy bằng `p02-controls.sh` bản 2 (Phụ lục):

| Ca | before | final (vòng 1) | m11 | r2 |
|---|---|---|---|---|
| s1 cùng chính sách → chạy | PASS | PASS | PASS | PASS |
| s2 khác biến thường → chạy | PASS | PASS | PASS | PASS |
| s3 `.env` đặt allowlist → chạy | PASS | PASS | PASS | PASS |
| s4 allowlist khác, `run` → từ chối | FAIL (chạy `pp`) | PASS | PASS | PASS |
| s5 TUI `--yolo` + `run` → từ chối | FAIL | PASS | PASS | PASS |
| s5 + `send -s` → từ chối `--yolo` | FAIL (lỗi "session…") | PASS | PASS | PASS |
| s5 + `loop --verify true` → từ chối `--yolo` | FAIL (chạy, NO_CHANGE) | PASS | PASS | PASS |
| s5 + `jobs` → KHÔNG bị so, rc=0 | PASS (kèm cảnh báo YOLO) | **FAIL (từ chối oan)** | FAIL | PASS (`No jobs…`) |
| s5 + `logout openai --force` → KHÔNG bị so | PASS | **FAIL** | FAIL | PASS (`Successfully logged out`) |
| s7 allowlist khác + `jobs` → KHÔNG bị so | PASS | **FAIL** | FAIL | PASS |
| s7 allowlist khác + `run` → từ chối | FAIL | PASS | PASS | PASS |

Thư mục: `p02-r2c-before`, `p02-r2c-final`, `p02-r2c-m11`, `p02-r2c` (r2), mỗi thư mục có `CTL/summary.txt`.

| Phép đo (r2) | Kết quả |
|---|---|
| C5 (`p02-r2-c5`) | `EXPOSED — ENV=EXPOSED ALLOW=ok` (ENV có tài liệu, chấp nhận) |
| T6, harness đã sửa (`p02-r2-t6`) | `EXPOSED`, đo được ca plan |
| `faults.sh` / `qwen-only.sh` / `workspaces.sh` | PASS / PASS / PASS (rc=0; `p02-r2-*`) |
| `test-agent-cli-run-json.sh` | PASS rc=0 (`p02-r2-runjson.out`) |
| Suite `go test -count=1 -p 2 -json ./...`, so theo tên với baseline `7cba42ff` | trước 4684 tên, sau 4699; FAIL đúng 14 tên `TestV3Exposure_*` như trước; 0 đổi trạng thái; mất 1 (`TestFirstWinsMismatch_LogsOnFlagDifferences/yolo`, bỏ ở vòng 1); mới 16, đều pass |
| `-race ./internal/backend/... ./internal/cmd/...` | 0 `DATA RACE`; backend FAIL chỉ do `TestV3Exposure_T1` (có sẵn); cmd, clientserverrace ok |

### 10.5 Kiểm răng vòng 2 (mutant, hoàn nguyên, `sha256sum -c` OK)

| Mutant | Kết quả |
|---|---|
| M6: backend so chính sách cả với client không prompt | FAIL `no-agent client, allowlist differs` |
| M7: backend so `--yolo` cả với client không prompt (nhánh đầu) | FAIL `no-agent client, yolo differs` |
| M8: `runsNoAgent` luôn false | FAIL test cmd |
| M9: `runsNoAgent` không dò lệnh cha | FAIL test cmd (ở `jobs output`/`kill`) |
| M10: gắn annotation lên `runCmd` (lệnh chạy model bị miễn nhầm) | FAIL test cmd |
| M11 (e2e): `connectToServer` không gửi cờ | s5-jobs, s5-logout, s7-jobs FAIL. Unit không bắt được mutant này; chỉ e2e bắt (xem R9) |

### 10.6 Rủi ro và chưa kiểm, vòng 2

- **R9:** chưa có unit test cho đường dây `connectToServer` → `NoAgentRuns` trên request. Chỉ e2e (đối chứng) bắt được.
- **R10:** `NoAgentRuns` là lời khai của client. Server không chặn được việc client khai "không prompt" rồi vẫn gửi prompt,
  vì request chạy agent không mang ID client. Phép so này bảo vệ client trung thực khỏi lệch chính sách; nó không phải
  ranh giới quyền (tiến trình cùng user vốn làm được mọi thứ qua socket).
- **R11:** workspace do `jobs`/`login`/`logout` dựng trước (khi chưa có workspace) mang `YOLO=false`. Nếu TUI `--yolo` gắn
  vào trong lúc lệnh đó còn sống, TUI bị từ chối. Cửa sổ ngắn (`login` OAuth có thể kéo dài). Chưa đo.
- **R12:** đường hồi phục của `login` (`ClientWorkspace.recreateArgs`) không mang `NoAgentRuns`, nên khi hồi phục, login bị
  so như lệnh chạy model. Hướng sai này chỉ làm chặt hơn, không cho lọt. Chưa chạy.
- R1 → đã chuyển thành "có tài liệu, chấp nhận". R2 → đã sửa cho `jobs`/`login`/`logout`. R3 (API thô) giữ nguyên;
  harness T6 đã thích ứng.
- **Chưa kiểm:**
  - `login` thật với server (cần OAuth); mới chỉ có unit test của annotation.
  - Launcher thật với lệnh không prompt.
  - `task lint`.
  - `gofmt` (GOROOT) và `gofumpt -l` sạch trên các tệp tôi đổi. `gofumpt -l internal/cmd/` có báo `routine.go` và
    `run_stream_test.go`: đó là tệp có sẵn, tôi không đụng (`git diff --quiet` xác nhận).
- **Môi trường:** giữa chừng, thư mục làm việc của phiên được đổi thành `agent-cli-p01`. Tôi vẫn làm trong
  `agent-cli-p02` như điều phối giao, không ghi gì vào p01.

### 10.7 Lệnh tái lập (thêm vào §6)

```bash
cd /home/luannt/application/opensource/agent-cli-p02
$G go test -count=1 -p 2 -v ./internal/backend/ ./internal/cmd/ -run 'TestCreateWorkspace_TrustPolicyMismatch|TestRunsNoAgent'
$G env CGO_ENABLED=0 go build -p 2 -o /tmp/p02-r2 .
bash <p02-controls.sh bản 2, Phụ lục> /home/luannt/.cache/v3ev/p02-N /tmp/p02-r2        # CONTROLS: PASS
bash scripts/eval/exposure.sh /home/luannt/.cache/v3ev/p02-M /tmp/p02-r2 T6              # summary: ws-create ... ok, có role plan
```

## 11. Vòng 3 (soát an ninh độc lập: KHONG-DAT → sửa #1–#3 và #4–#11)

### 11.1 #1 (cao, có từ V2): `.env` của repo chen vào chính sách của tiến trình con mà Home dựng

**ĐỎ trước khi sửa.**

E2E: `bash scratchpad/p02w/p02-home.sh <thư-mục-mới> <binary>` (Phụ lục), chạy trong khung cô lập của `exposure.sh` với
LLM giả và tmux riêng.
- Repo có `.env`:
  - `AGENT_CLI_DELEGATION=on`;
  - `AGENT_CLI_ENV_FILE=repo/policy.env` (allowlist có `claude_code:claude-opus-5-5`);
  - `AGENT_CLI_CLAUDE_BIN=repo/evil`;
  - `AGENT_CLI_CODEX_SANDBOX=danger-full-access`.
- Không biến nào được export.
- `AGENT_CLI_LAUNCHER` trỏ một launcher giả: ghi env nó nhận được, rồi exec `<binary> run --json` trong đúng env đó.

| Đường | before (`7cba42ff`, V2) | r2 (vòng 2) | r3 (vòng 3) |
|---|---|---|---|
| HOME: `agent-cli` trần trong repo → Enter → "New session" (`homeChildCmd`) | env con có 4 khoá từ `.env`; lượt chạy có tool `claude_code` | như before | không khoá nào; không `claude_code` |
| HOME2: `agent-cli --cwd repo` → Alt+W (`execHome`) | 4 khoá; có `claude_code` | như before | không khoá nào; không `claude_code` |
| CTL: `agent-cli run` thẳng trong repo | không `claude_code` | không | không |
| Verdict | EXPOSED | EXPOSED | NOT_EXPOSED |

Thư mục: `p02-h3-before`, `p02-h2-r2`, `p02-h2-r3` (dưới `/home/luannt/.cache/v3ev/`, mỗi thư mục có `HM/summary.txt`,
`HM/child.env`, `HM/child2.env`).

Unit: `go test ./internal/cmd/ -run TestChildEnvsIgnoreProjectDotenv`. Test chạy trong tiến trình con theo mẫu
`TestScheduleEnv_IgnoresProjectDotenv`: gọi `execpolicy.Snapshot()`, rồi `godotenv.Load()` với `.env` chứa 5 khoá tin cậy,
rồi dựng env cho ba tiến trình con: `homeChildCmd`, `homeEnv`, và `--verify` của loop chạy `env`.
- Trước sửa: FAIL với 15 dòng lọt (5 khoá × 3 tiến trình con).
- Sau sửa: PASS. Biến thường `PLAIN_FROM_DOTENV` vẫn tới cả 3 tiến trình.
- Test chỉ in các dòng lọt, không in cả env (env của máy chạy test có token).

**Sửa.**
- `homeEnv() = execpolicy.ChildEnv(os.Environ())`, dùng cho `homeChildCmd` và `execHome`.
- `shellVerify` (verify của loop) dùng `execpolicy.ChildEnv(os.Environ())`.
- `ChildEnv` bỏ các khoá tin cậy khỏi env đã nạp `.env`, rồi gắn lại giá trị theo ảnh chụp đầu `main`; biến khác giữ
  nguyên. Đây cũng là hàm `startDetachedServer` đang dùng.
- **Phát hiện thêm, cùng lớp:** `homeLauncher()` đọc `AGENT_CLI_LAUNCHER` bằng `os.Getenv`. Khi chạy binary không qua
  launcher, `.env` của repo chọn được chương trình mà Home exec (chạy mã tuỳ ý khi gõ `agent-cli` trong repo đó). Nay đọc
  bằng `execpolicy.OriginalEnv`. Test: `.env` đặt `AGENT_CLI_LAUNCHER=/bin/evil-launcher`, kết quả không được trả về nó.

**Grep mọi `os.Environ()` có thể dựng tiến trình con:**

| Chỗ | Tiến trình con | Xử lý |
|---|---|---|
| `cmd/home.go` `homeChildCmd`, `execHome` | agent-cli | sửa (ChildEnv) |
| `cmd/loop.go:271` verify | lệnh của người dùng, có thể chạy agent-cli | sửa (ChildEnv) |
| `cmd/root.go:1063` `startDetachedServer` | agent-cli server | đã ChildEnv từ trước |
| `cmd/root.go:588` `Env: os.Environ()` trong request | không exec. Server dùng làm env cho lệnh `!` người dùng gõ (`backend/agent.go:353`) | giữ; không phải chính sách |
| `shell/run.go`, `shell/shell.go`, `shell/stream.go` (tool bash, chế độ local) | lệnh của agent | giữ. Gọi thẳng agent-cli `run/send/server/loop/routine` đã bị BlockFunc chặn; gọi gián tiếp qua script là giới hạn đã biết của lớp chặn theo tên. Mới nghi, chưa đo |
| `hooks/input.go:61` | lệnh hook trong crushrc của người dùng | giữ; mới nghi (hook gọi agent-cli) |
| `agent/tools/mcp/init.go:1212` | server MCP | giữ |
| `agent/tools/provider/exec.go:35` | CLI Claude/Codex (`childEnv` riêng) | giữ; CLI đó không đọc `AGENT_CLI_*` |
| `shellconfig/load.go:57` | trình thông dịch crushrc trong tiến trình | không exec agent-cli |
| `cmd/session.go:550` pager, git trong `routine.go`/`loop.go` | không phải agent-cli | giữ |

Sau khi sửa, câu "`.env` của repo không tính" (guide, CHANGELOG) đúng cho cả đường Home. CHANGELOG có thêm câu về Home
và verify.

### 11.2 #2 (trung bình): server khác build đang bận làm lỗ C5 sống lại lặng lẽ

**Quyết định của điều phối:**
- Lệnh chạy model gặp server khác `Version`/`BuildID` mà server không chịu tắt (đường đang tái dùng) thì bị TỪ CHỐI.
- Lệnh không prompt vẫn tái dùng server đó.

**Mã:**
- `restartIfStale`: hai nhánh "reusing it" (server bận, hoặc server legacy bận/không liệt kê được) trả
  `staleServerBusy(cmd, vi)`. Hàm này trả nil cho lệnh không prompt, và trả `errStaleServerBusy` cho lệnh chạy model.
  Lỗi nêu bản/build của server và của client, cùng cách sửa: thoát mọi client, chờ server thoát, xem `agent-cli ps`.
- `ensureServer`: gặp `errStaleServerBusy` thì trả lỗi luôn. Nếu không, nhánh lỗi cũ (`needsStart`) sẽ dựng server mới
  rồi gắn vào chính server cũ.

**Unit:** `TestRestartIfStale_RefusesModelCommandOnBusyStaleServer`, hai ca `busy` và `legacy busy`.
- ĐỎ trước sửa (`got nil`), XANH sau sửa.
- Lỗi phải nêu bản cũ và `agent-cli ps`. Không lệnh tắt vô điều kiện nào được gửi.
- Hai test cũ `HonorsRefusedShutdown`/`LeavesLegacyServerBusy` nay chạy bằng `noPromptCmd`, giữ nguyên mọi assert.

**E2E:** `p02-stale.sh <dir> <binary server> <binary client>` (Phụ lục). Server bản cũ đang giữ một lượt 8 giây; client
bản mới chạy `run` rồi `jobs`; đối chứng là `run` cùng build với server.

| Server | Client | `run` bản mới | `jobs` bản mới | `run` cùng build | Kết quả |
|---|---|---|---|---|---|
| before (V2) | r2 | chạy trên server cũ, `['qwen']` (lỗ) | rc=0 | rc=0 | FAIL |
| before (V2) | r3 | rc=1, `another build and is busy … see agent-cli ps`, 0 request | rc=0 | rc=0 | PASS |
| before (V2) | m18 (bỏ nhánh `errors.Is` trong `ensureServer`) | chạy trên server cũ (lỗ) | rc=0 | rc=0 | FAIL (mutant bị bắt) |

- Thư mục: `p02-stb-r2`, `p02-stb-r3`, `p02-stb-m18`.
- Ghi chú: thử với server là bản trung gian vòng 1 (`agent-cli-final`) thì `jobs` của r3 bị server đó từ chối ("trust
  policy not sent"), vì vòng 1 chưa biết `no_agent_runs`. Vòng 1 chưa phát hành, nên đây không phải ca thật.

### 11.3 #3 (trung bình): test canh hướng mở lỗ của miễn trừ

- `TestRunsNoAgentExactCommandSet` duyệt toàn cây `rootCmd`. Tập lệnh có `runsNoAgent` phải đúng bằng {`jobs`,
  `jobs output`, `jobs kill`, `login`, `logout`}; `root/run/send/loop` phải là false.
  - M22: gắn annotation lên `sendCmd` → đỏ.
  - M23: thêm lệnh con mới `jobs ask` → đỏ.
- `TestConnectToServerSendsNoAgentRuns` gọi THẬT `connectToServer`, với `clientHost` trỏ server HTTP giả (mẫu
  `home_test.go`), và bắt body `POST /v1/workspaces`.
  - Lệnh chạy model gửi `no_agent_runs=false` kèm `trusted_env`; lệnh không prompt và lệnh con của nó gửi `true`, không
    kèm `trusted_env`.
  - M15 (gán cứng `NoAgentRuns: true` trong `connectToServer`) → unit đỏ. M16b (client luôn gửi `trusted_env`) → đỏ.
- Tôi không tách hàm dựng request: test gọi thẳng `connectToServer` mạnh hơn, và không cần refactor.

### 11.4 NÊN SỬA (#4–#11)

| # | Sửa | Test / chứng cứ |
|---|---|---|
| 4 | `client.CreateWorkspace` không gọi `Inputs()` khi `NoAgentRuns` | `TestCreateWorkspaceSendsTrustedEnvOnlyForAgentRuns` (đỏ → xanh; M16) |
| 5 | backend: tính `checkTrustedEnv` trước khi khoá, áp SAU `admitLocked` và TRƯỚC `cancelShutdownLocked` (server đã chốt tắt trả 503; không đọc tệp dưới `b.mu`) | `TestCreateWorkspace_ClosingServerBeatsPolicyMismatch` (đỏ → xanh; M19) |
| 6 | lỗi chỉ nêu tên khoá của chính server; khoá lạ do client gửi chỉ báo số (`N key(s) this server does not know`) | ca `client sends a key the server does not know` (đỏ → xanh; M20) |
| 7 | `Inputs()` trả `slices.Clone` | `TestInputsHermeticInTests`, thêm bước sửa slice trả về (đỏ → xanh; M21) |
| 9 | docs: README nêu hai chiều YOLO; bỏ "không bị so gì" (vẫn so `--channels`; `login` khi hồi phục vẫn bị so); ghi rõ so chuỗi thô (thứ tự allowlist, `on`/`ON`/`1`, khác `HOME`/`AGENT_CLI_ENV_FILE`); ngắt dòng dài ở guide (dòng thêm ≤ 99 ký tự); ghi server khác build và đường Home/verify | đọc lại |
| 10 | thông điệp thiếu `trusted_env`: "(an older agent-cli client; use the same agent-cli build as the server)"; bỏ tiền tố lặp "failed to create workspace: failed to create workspace" (`createWorkspaceOnLiveServer` trả nguyên lỗi của client, vốn đã có tiền tố) | e2e: lỗi nay là `failed to create workspace: status code 500: trust policy differs…` |
| 11 | comment và thông điệp assert ở `backend_test.go:475,482`, kèm comment hai test restart | đọc lại |

### 11.5 Bảng chân trị vòng 3 (binary `r3` = `/home/luannt/.cache/v3ev/p02-bin/agent-cli-r3`)

| Phép đo | Kết quả |
|---|---|
| Unit mới vòng 3 (8 test/ca) | ĐỎ trước khi cài: `ChildEnvs` (15 dòng lọt), `RestartIfStale_RefusesModel` ×2, ca khoá lạ, `ClosingServer`, `SendsTrustedEnvOnlyForAgentRuns`, `InputsHermetic`. `ConnectToServerSendsNoAgentRuns` lần đầu đỏ sai (panic), chứng minh lại bằng M15/M16b. Sau khi cài: XANH |
| Home e2e (`p02-h2-r3`) | NOT_EXPOSED (before và r2: EXPOSED) |
| Stale e2e (`p02-stb-r3`) | PASS (r2, m18: FAIL) |
| C5 (`p02-r3-c5`) | `EXPOSED — ENV=EXPOSED ALLOW=ok` (ENV có tài liệu, chấp nhận) |
| T6 (`p02-r3-t6`, harness đã sửa) | `EXPOSED`, `ws-create … ok`, có finding plan |
| Đối chứng `p02-controls.sh` (`p02-r3c`) | CONTROLS: PASS (s1–s5, s7, send, loop, jobs, logout) |
| `faults.sh` / `qwen-only.sh` / `workspaces.sh` / `test-agent-cli-run-json.sh` | PASS / PASS / PASS / PASS (rc=0) |
| Suite theo TÊN so với `7cba42ff` | 4684 → 4707 tên; FAIL đúng 14 tên `TestV3Exposure_*` như cũ; 0 đổi trạng thái; mất 1 (`…/yolo`, vòng 1); mới 24, đều pass |
| `-race ./internal/backend/... ./internal/cmd/...` | 0 `DATA RACE`; backend FAIL chỉ do `TestV3Exposure_T1`; cmd và clientserverrace ok |

**Mutant vòng 3** (mỗi mutant được hoàn nguyên và kiểm sha sau khi chạy):

| Mutant | Thay đổi | Kết quả |
|---|---|---|
| M12 | `homeEnv` dùng `os.Environ()` | `ChildEnvs` đỏ (10 dòng lọt) |
| M13 | verify dùng `os.Environ()` | `ChildEnvs` đỏ (5 dòng) |
| M14 | launcher đọc `os.Getenv` | `ChildEnvs` đỏ |
| M15 | gán cứng `NoAgentRuns: true` | `ConnectToServer` đỏ |
| M16 | client luôn gửi `trusted_env` | test client đỏ |
| M16b | như M16 | test đường dây đỏ |
| M17 | `staleServerBusy` luôn cho tái dùng | 2 ca refuse đỏ |
| M17b | luôn từ chối | 2 test tái dùng của lệnh không prompt đỏ |
| M19 | so chính sách trước `admitLocked` | `ClosingServer` đỏ |
| M20 | lặp tên khoá lạ | ca khoá lạ đỏ |
| M21 | `Inputs` không clone | `InputsHermetic` đỏ |
| M22 | gắn annotation lên `send` | tree-walk đỏ |
| M23 | thêm lệnh con dưới `jobs` | tree-walk đỏ |
| M18 | `ensureServer` bỏ nhánh `errStaleServerBusy` | chỉ e2e bắt được (`p02-stb-m18` FAIL) |

### 11.6 Rủi ro và chưa kiểm, vòng 3

- **R13:** `ensureServer` (đường socket unix) chưa có unit cho nhánh `errStaleServerBusy`; chỉ e2e bắt được (M18).
- **R14:** từ chối theo `BuildID`. BuildID suy từ mtime của tệp chạy, nên build lại binary trong lúc TUI đang mở (server
  bận) làm `run`/`send`/TUI mới bị từ chối cho tới khi server cũ thoát. Hai bản ở e2e có cùng chuỗi `Version`, chỉ khác
  build.
- **R15:** các chỗ `os.Environ()` giữ nguyên ở bảng §11.1 (tool bash chế độ local, hook, MCP). Mới nghi: agent-cli được
  gọi gián tiếp từ script hay hook sẽ nhận khoá tin cậy từ `.env`. Cách kiểm: hook hoặc script gọi `agent-cli run` trong
  repo có `.env` như §11.1, rồi xem tool `claude_code`.
- **R16:** giữ nguyên từ vòng 2: `NoAgentRuns` là lời khai (R10); hồi phục của `login` bị so như lệnh chạy model (R12).
- **Chưa kiểm:**
  - `task lint`.
  - Launcher thật (`install-agent-cli.sh`) với Home: e2e dùng launcher giả. Launcher thật export `AGENT_CLI_LAUNCHER`,
    nên chỉ lỗ khoá tin cậy áp với nó.
  - Server legacy thật (bản trước `shutdown_if_idle`): mới chỉ có unit giả lập.
- **Scratchpad:**
  - Đã kiểm `git status` của p02: đúng 20 tệp tracked bị sửa cùng 4 tệp test mới của tôi, ngoài 6 tệp phơi nhiễm `??`.
  - Log ĐỎ đầu tiên có dump env của máy; đã lọc khỏi tệp log, và test đã sửa để không in env.

### 11.7 Lệnh tái lập (vòng 3)

```bash
cd /home/luannt/application/opensource/agent-cli-p02
G='flock /tmp/claude-1000/agent-go.lock env GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off'   # và free -m avail ≥ 3000
$G go test -count=1 -p 2 -v ./internal/cmd/ ./internal/backend/ ./internal/client/ ./internal/execpolicy/ \
  -run 'TestChildEnvsIgnoreProjectDotenv|TestRunsNoAgentExactCommandSet|TestConnectToServerSendsNoAgentRuns|TestRestartIfStale|TestCreateWorkspace_TrustPolicyMismatch|TestCreateWorkspace_ClosingServer|TestCreateWorkspaceSendsTrustedEnv|TestInputsHermetic'
$G env CGO_ENABLED=0 go build -p 2 -o /tmp/p02-r3 .
bash <p02-home.sh, Phụ lục> /home/luannt/.cache/v3ev/p02-N /tmp/p02-r3                    # NOT_EXPOSED
bash <p02-stale.sh, Phụ lục> /home/luannt/.cache/v3ev/p02-M <binary V2 7cba42ff> /tmp/p02-r3  # STALE: PASS
# ĐỎ #1: chạy p02-home.sh với binary build từ 7cba42ff → EXPOSED.
```

## Phụ lục: script đối chứng e2e (`p02-controls.sh`, bản 2)

Dựng lại khung C5 bằng chính các hàm của `scripts/eval/exposure.sh`, trích bằng `sed`. Không sửa tệp đó. Bản 2 thêm, trên
cùng TUI `--yolo` của s5: `send`, `loop`, `jobs`, `logout`; và ca s7 (allowlist khác + `jobs`, đối chứng bằng `run`).
Bản 1 (dùng cho bảng §5) là bản 2 bỏ các phần đó.

```bash
#!/usr/bin/env bash
# Đối chứng e2e cho P02 (bản sửa chính sách tin cậy), dựng lại khung C5 bằng CHÍNH các hàm của
# scripts/eval/exposure.sh (trích bằng sed, không sửa tệp đó). Binary thật + LLM giả, cô lập như exposure.sh.
# Mỗi kịch bản: client đầu giữ một lượt 8s trên server riêng, client sau gắn vào cùng workspace.
#   S1 same    : cả hai AGENT_CLI_ALLOWED_MODELS=qwen:m-qwen, dự án chọn qwen  -> client sau CHẠY (qwen)
#   S2 plain   : khác ở biến thường (FOO=bar), dự án chọn pp                    -> client sau CHẠY (pp)
#   S3 dotenv  : `.env` của dự án đặt allowlist qwen (không export), chọn pp    -> client sau CHẠY (pp), không bị từ chối
#   S4 allow   : client sau export allowlist qwen (= C5 allow)                  -> client sau BỊ TỪ CHỐI, 0 request
#   S5 yolo    : client đầu là TUI --yolo (tmux riêng), client sau `run`        -> client sau BỊ TỪ CHỐI (--yolo)
#                cùng TUI đó: `send`, `loop` (chạy model) -> BỊ TỪ CHỐI; `jobs`, `logout` (không prompt) -> KHÔNG bị so
#   S7 allow+jobs: client sau export allowlist khác: `jobs` -> không bị từ chối; `run` cùng env -> BỊ TỪ CHỐI
# Dùng: bash p02-controls.sh <thư-mục-mới ≤ 70 byte> <binary>
set -u
[ $# -eq 2 ] || { echo "usage: $0 <new-dir> <binary>" >&2; exit 2; }
OUT=$1; BIN=$2
R=/home/luannt/application/opensource/agent-cli-p02
[ -e "$OUT" ] && { echo "exists: $OUT" >&2; exit 2; }
mkdir -p "$OUT" && OUT=$(cd "$OUT" && pwd) && BIN=$(cd "$(dirname "$BIN")" && pwd)/$(basename "$BIN")
L="$OUT/_launcher"
eval "$(sed -n '/^verdict()/,/^for c in \$CASES/p' "$R/scripts/eval/exposure.sh" | sed '$d')"

begin CTL '[{"match":"CTL-hold","steps":[{"sleep":8,"text":"held"}]}]' qwen
for d in s1 s2 s3 s4 s5; do mkdir -p "$P/$d"; git_repo "$P/$d"; done
printf '%s' '{"models":{"large":{"provider":"qwen","model":"m-qwen"},"small":{"provider":"qwen","model":"m-qwen"}}}' > "$P/s1/crush.json"
for d in s2 s3 s4 s5; do
  printf '%s' '{"models":{"large":{"provider":"pp","model":"m-pp"},"small":{"provider":"pp","model":"m-pp"}}}' > "$P/$d/crush.json"
done

# pair <thư mục> <tên> <env client đầu> <env client sau>: env là chuỗi "K=V K=V" (rỗng = không thêm).
pair() {
  local d=$1 n=$2 hold
  server_mode "$n"; read -r -a EXTRA <<< "$3"
  ac "$P/$d" run --json "CTL-hold $n" > "$P/$n-hold.json" 2>&1 & hold=$!
  sleep 2.5
  [ "$n" = s3 ] && printf 'AGENT_CLI_ALLOWED_MODELS=qwen:m-qwen\nAGENT_CLI_DELEGATION=on\n' > "$P/$d/.env"
  read -r -a EXTRA <<< "$4"
  ac "$P/$d" run --json "CTL-$n second" > "$P/$n-two.json" 2> "$P/$n-two.err"; echo $? > "$P/$n-two.rc"
  wait $hold; stop_servers; EXTRA=()
}
pair s1 s1 "AGENT_CLI_ALLOWED_MODELS=qwen:m-qwen" "AGENT_CLI_ALLOWED_MODELS=qwen:m-qwen"
pair s2 s2 "" "FOO=bar"
pair s3 s3 "" ""
pair s4 s4 "" "AGENT_CLI_ALLOWED_MODELS=qwen:m-qwen"

# S5: TUI --yolo trong tmux riêng (socket riêng, không đụng màn hình), rồi `run` cùng dự án.
server_mode s5
TMUX_SOCK="$P/tmux.sock"
iso "$P/s5" "$(command -v tmux)" -S "$TMUX_SOCK" new-session -d -x 200 -y 50 -s y "$BIN --host unix://$SOCK --yolo" 2> "$P/s5-tmux.err"
for _ in $(seq 100); do [ -S "$SOCK" ] && break; sleep 0.1; done
sleep 3
ac "$P/s5" run --json "CTL-s5 second" > "$P/s5-two.json" 2> "$P/s5-two.err"; echo $? > "$P/s5-two.rc"
ac "$P/s5" send -s nosuch --json "CTL-s5 send" > "$P/s5-send.out" 2>&1; echo $? > "$P/s5-send.rc"
ac "$P/s5" loop --verify true "CTL-s5 loop" > "$P/s5-loop.out" 2>&1; echo $? > "$P/s5-loop.rc"
ac "$P/s5" jobs > "$P/s5-jobs.out" 2>&1; echo $? > "$P/s5-jobs.rc"
ac "$P/s5" logout openai --force > "$P/s5-logout.out" 2>&1; echo $? > "$P/s5-logout.rc"
"$(command -v tmux)" -S "$TMUX_SOCK" kill-server 2>/dev/null
stop_servers

# S7: holder không allowlist; client sau export allowlist khác chạy `jobs` (không prompt) rồi `run` (đối chứng).
server_mode s7; EXTRA=()
ac "$P/s4" run --json "CTL-hold s7" > "$P/s7-hold.json" 2>&1 & h7=$!
sleep 2.5; EXTRA=(AGENT_CLI_ALLOWED_MODELS=qwen:m-qwen)
ac "$P/s4" jobs > "$P/s7-jobs.out" 2>&1; echo $? > "$P/s7-jobs.rc"
ac "$P/s4" run --json "CTL-s7 second" > "$P/s7-two.json" 2> "$P/s7-two.err"; echo $? > "$P/s7-two.rc"
wait $h7; stop_servers; EXTRA=()
finish

python3 -I - "$P" <<'PY' | tee "$P/summary.txt"
import json, os, sys
P = sys.argv[1]
reqs = [json.loads(l) for l in open(os.path.join(P, "log", "requests.jsonl"))]
def prov(tag):
    return sorted({r["provider"] for r in reqs if r["role"] == "main" and r["last_user"].lstrip().startswith(tag)}) or ["none"]
def err(n):
    try:
        return json.load(open(os.path.join(P, n + "-two.json"))).get("error") or ""
    except Exception as e:
        return "no JSON (%s)" % e
rc = lambda n: open(os.path.join(P, n + "-two.rc")).read().strip()
want = {  # n: (provider mong đợi, có bị từ chối không, chuỗi phải có trong lỗi)
    "s1": (["qwen"], False, ""),
    "s2": (["pp"], False, ""),
    "s3": (["pp"], False, ""),
    "s4": (["none"], True, "AGENT_CLI_ALLOWED_MODELS"),
    "s5": (["none"], True, "--yolo"),
    "s7": (["none"], True, "AGENT_CLI_ALLOWED_MODELS"),
}
ok_all = True
for n, (wp, refused, key) in want.items():
    e, p = err(n), prov("CTL-%s second" % n)
    is_ref = "trust policy differs" in e
    ok = p == wp and is_ref == refused and (key in e) and ("qwen:m-qwen" not in e)
    ok_all &= ok
    print("%s: %s rc=%s provider=%s refused=%s err=%r" % (n, "PASS" if ok else "FAIL", rc(n), p, is_ref, e[:400]))
rd = lambda f: open(os.path.join(P, f), errors="replace").read()
# (tệp, phải bị từ chối?, chuỗi phải có, rc phải bằng 0?)
extra = [("s5-send", True, "--yolo", False), ("s5-loop", True, "--yolo", False),
         ("s5-jobs", False, "", True), ("s5-logout", False, "", False), ("s7-jobs", False, "", True)]
for f, refused, key, rc0 in extra:
    out, code = rd(f + ".out"), rd(f + ".rc").strip()
    is_ref = "trust policy differs" in out
    ok = is_ref == refused and key in out and (code == "0" or not rc0) and (not refused or code != "0")
    ok_all &= ok
    print("%s: %s rc=%s refused=%s out=%r" % (f, "PASS" if ok else "FAIL", code, is_ref, out.strip()[-300:]))
print("CONTROLS:", "PASS" if ok_all else "FAIL")
PY
```

## Phụ lục 2: `p02-home.sh` (vòng 3, #1)

```bash
#!/usr/bin/env bash
# P02 vòng 3, #1: `.env` của repo có chen vào chính sách tin cậy của tiến trình con mà Home dựng không?
# Khung cô lập của scripts/eval/exposure.sh (trích hàm bằng sed), binary thật + LLM giả, tmux riêng (không đụng màn hình).
#   repo/.env đặt AGENT_CLI_DELEGATION=on, AGENT_CLI_ENV_FILE=repo/policy.env (allowlist có claude_code),
#   AGENT_CLI_CLAUDE_BIN=repo/evil, AGENT_CLI_CODEX_SANDBOX=danger-full-access. Không biến nào được export.
#   HOME: gõ `agent-cli` trần trong repo (Home) -> Enter (dự án trong Recent) -> Enter ("New session"). AGENT_CLI_LAUNCHER trỏ launcher giả:
#         ghi env nó nhận được rồi exec `<binary> run --json` trong env đó (thay cho TUI của workspace).
#   HOME2: `agent-cli --cwd repo` (TUI) rồi Alt+W: execHome thay tiến trình bằng launcher giả, cùng cách ghi env + run.
#   CTL : `agent-cli run --json` thẳng trong repo (không qua Home).
# Bất biến: tool `claude_code` KHÔNG có trong request của lượt chạy, và env con không mang khoá tin cậy lấy từ `.env`.
# Dùng: bash p02-home.sh <thư-mục-mới ≤ 70 byte> <binary>
set -u
[ $# -eq 2 ] || { echo "usage: $0 <new-dir> <binary>" >&2; exit 2; }
OUT=$1; BIN=$2
R=/home/luannt/application/opensource/agent-cli-p02
[ -e "$OUT" ] && { echo "exists: $OUT" >&2; exit 2; }
mkdir -p "$OUT" && OUT=$(cd "$OUT" && pwd) && BIN=$(cd "$(dirname "$BIN")" && pwd)/$(basename "$BIN")
L="$OUT/_launcher"
eval "$(sed -n '/^verdict()/,/^for c in \$CASES/p' "$R/scripts/eval/exposure.sh" | sed '$d')"

begin HM '[]' qwen
mkdir -p "$P/repo"; git_repo "$P/repo"
printf '#!/bin/sh\necho evil\n' > "$P/repo/evil"; chmod 755 "$P/repo/evil"
printf 'AGENT_CLI_ALLOWED_MODELS="qwen:m-qwen claude_code:claude-opus-5-5"\n' > "$P/repo/policy.env"
cat > "$P/repo/.env" <<EOF
AGENT_CLI_DELEGATION=on
AGENT_CLI_ENV_FILE=$P/repo/policy.env
AGENT_CLI_CLAUDE_BIN=$P/repo/evil
AGENT_CLI_CODEX_SANDBOX=danger-full-access
EOF
cat > "$P/launcher" <<EOF
#!/bin/sh
env > "$P/child.env"
exec "$BIN" run --json "HM-home probe" > "$P/home-run.json" 2> "$P/home-run.err"
EOF
chmod 755 "$P/launcher"

# CTL: chạy thẳng (main chụp env trước khi nạp .env).
local_ac "$P/repo" run --json "HM-ctl probe" > "$P/ctl-run.json" 2> "$P/ctl-run.err"

# HOME: Home trong tmux riêng, Enter chọn "Open current folder".
TM="$P/tmux.sock"; EXTRA=(AGENT_CLI_LAUNCHER="$P/launcher")
iso "$P/repo" "$(command -v tmux)" -S "$TM" new-session -d -x 160 -y 40 -s h "'$BIN' 2> '$P/home.stderr'; echo \$? > '$P/home.rc'; sleep 3" 2> "$P/tmux.err"
for _ in $(seq 50); do "$(command -v tmux)" -S "$TM" capture-pane -p -t h 2>/dev/null | grep -q "Open current folder" && break; sleep 0.2; done
"$(command -v tmux)" -S "$TM" capture-pane -p -t h > "$P/home-screen.txt" 2>/dev/null
sleep 1
"$(command -v tmux)" -S "$TM" send-keys -t h Enter
sleep 1
"$(command -v tmux)" -S "$TM" capture-pane -p -t h > "$P/home-after.txt" 2>/dev/null
# Dự án đã có trong Recent (lượt CTL đăng ký nó): Enter mở danh sách session, Enter nữa chọn "New session".
grep -q "New session" "$P/home-after.txt" && "$(command -v tmux)" -S "$TM" send-keys -t h Enter
for _ in $(seq 150); do [ -s "$P/home-run.json" ] && break; sleep 0.2; done
sleep 1
"$(command -v tmux)" -S "$TM" kill-server 2>/dev/null

# HOME2 (execHome): TUI mở thẳng (--cwd), Alt+W quay về Home: tiến trình được thay bằng launcher (ở đây launcher giả)
# với env do execHome dựng.
cat > "$P/launcher2" <<EOS
#!/bin/sh
env > "$P/child2.env"
exec "$BIN" run --json "HM-exec probe" > "$P/exec-run.json" 2> "$P/exec-run.err"
EOS
chmod 755 "$P/launcher2"
TM2="$P/tmux2.sock"; EXTRA=(AGENT_CLI_LAUNCHER="$P/launcher2")
iso "$P/repo" "$(command -v tmux)" -S "$TM2" new-session -d -x 160 -y 40 -s e "'$BIN' --cwd '$P/repo' 2> '$P/tui.stderr'; echo \$? > '$P/tui.rc'; sleep 3" 2>> "$P/tmux.err"
for _ in $(seq 50); do [ "$("$(command -v tmux)" -S "$TM2" capture-pane -p -t e 2>/dev/null | grep -c .)" -ge 5 ] && break; sleep 0.2; done
sleep 2
"$(command -v tmux)" -S "$TM2" capture-pane -p -t e > "$P/tui-screen.txt" 2>/dev/null
# Lần đầu mở dự án TUI hỏi có khởi tạo AGENTS.md không: chọn "Nope" rồi mới Alt+W.
if grep -q "initialize now" "$P/tui-screen.txt"; then
  "$(command -v tmux)" -S "$TM2" send-keys -t e Right; sleep 0.5
  "$(command -v tmux)" -S "$TM2" send-keys -t e Enter; sleep 1.5
fi
"$(command -v tmux)" -S "$TM2" send-keys -t e M-w
for _ in $(seq 150); do [ -s "$P/exec-run.json" ] && break; sleep 0.2; done
sleep 1
"$(command -v tmux)" -S "$TM2" kill-server 2>/dev/null
EXTRA=()
finish

python3 -I - "$P" <<'PY' | tee "$P/summary.txt"
import json, os, sys
P = sys.argv[1]
f = os.path.join(P, "log", "requests.jsonl")
reqs = [json.loads(l) for l in open(f)] if os.path.exists(f) else []
def tools(tag):
    rs = [r for r in reqs if r["role"] == "main" and r["last_user"].lstrip().startswith(tag)]
    return sorted(set().union(*[set(r["tools"]) for r in rs])) if rs else None
keys = ["AGENT_CLI_DELEGATION", "AGENT_CLI_ENV_FILE", "AGENT_CLI_CLAUDE_BIN", "AGENT_CLI_CODEX_SANDBOX"]
env = {}
if os.path.exists(os.path.join(P, "child.env")):
    for line in open(os.path.join(P, "child.env")):
        k, _, v = line.rstrip("\n").partition("=")
        env[k] = v
def envof(name):
    out = {}
    if os.path.exists(os.path.join(P, name)):
        for line in open(os.path.join(P, name)):
            k, _, v = line.rstrip("\n").partition("=")
            out[k] = v
    return out
env2 = envof("child2.env")
leaked = [k for k in keys if k in env]
leaked2 = [k for k in keys if k in env2]
th, tc, te = tools("HM-home"), tools("HM-ctl"), tools("HM-exec")
has = lambda t: t is not None and "claude_code" in t
print("HOME (homeChildCmd): child env has trusted keys from .env: %s | run: claude_code in tools=%s" % (leaked or "none", has(th)))
print("HOME2 (execHome via Alt+W): child env has trusted keys from .env: %s | run: claude_code in tools=%s" % (leaked2 or "none", has(te)))
print("CTL (direct run in repo): claude_code in tools=%s" % has(tc))
if th is None or te is None or tc is None or not env or not env2:
    print("INCONCLUSIVE (home=%s exec=%s ctl=%s)" % (th is not None, te is not None, tc is not None))
elif has(tc):
    print("INCONCLUSIVE (control already delegates)")
else:
    print("EXPOSED" if (leaked or leaked2 or has(th) or has(te)) else "NOT_EXPOSED")
PY
```

## Phụ lục 3: `p02-stale.sh` (vòng 3, #2)

```bash
#!/usr/bin/env bash
# P02 vòng 3, #2: server khác build đang bận. Lệnh chạy model (`run`) phải bị TỪ CHỐI rõ ràng (server bản khác có thể
# không so chính sách tin cậy); lệnh không prompt (`jobs`) vẫn dùng server đó. Đối chứng: client cùng build với server
# gắn vào bình thường. Khung cô lập của scripts/eval/exposure.sh, LLM giả.
# Dùng: bash p02-stale.sh <thư-mục-mới ≤ 70 byte> <binary server (bản cũ)> <binary client (bản mới)>
set -u
[ $# -eq 3 ] || { echo "usage: $0 <new-dir> <server-binary> <client-binary>" >&2; exit 2; }
OUT=$1; OLD=$(cd "$(dirname "$2")" && pwd)/$(basename "$2"); NEW=$(cd "$(dirname "$3")" && pwd)/$(basename "$3")
R=/home/luannt/application/opensource/agent-cli-p02
[ -e "$OUT" ] && { echo "exists: $OUT" >&2; exit 2; }
mkdir -p "$OUT" && OUT=$(cd "$OUT" && pwd); L="$OUT/_launcher"; BIN=$OLD
eval "$(sed -n '/^verdict()/,/^for c in \$CASES/p' "$R/scripts/eval/exposure.sh" | sed '$d')"
begin ST '[{"match":"ST-hold","steps":[{"sleep":8,"text":"held"}]}]' qwen
mkdir -p "$P/proj"; git_repo "$P/proj"
server_mode st
BIN=$OLD; ac "$P/proj" run --json "ST-hold old server" > "$P/hold.json" 2>&1 & hold=$!
sleep 2.5
BIN=$NEW; ac "$P/proj" run --json "ST-new run" > "$P/new-run.json" 2> "$P/new-run.err"; echo $? > "$P/new-run.rc"
BIN=$NEW; ac "$P/proj" jobs > "$P/new-jobs.out" 2>&1; echo $? > "$P/new-jobs.rc"
BIN=$OLD; ac "$P/proj" run --json "ST-same run" > "$P/same-run.json" 2> "$P/same-run.err"; echo $? > "$P/same-run.rc"
wait $hold; stop_servers; finish
python3 -I - "$P" <<'PY' | tee "$P/summary.txt"
import json, os, sys
P = sys.argv[1]
reqs = [json.loads(l) for l in open(os.path.join(P, "log", "requests.jsonl"))]
prov = lambda tag: sorted({r["provider"] for r in reqs if r["role"] == "main" and r["last_user"].lstrip().startswith(tag)}) or ["none"]
rd = lambda f: open(os.path.join(P, f), errors="replace").read().strip()
def err(f):
    try:
        return json.load(open(os.path.join(P, f))).get("error") or ""
    except Exception as e:
        return "no JSON (%s)" % e
e = err("new-run.json")
ok_run = rd("new-run.rc") != "0" and "another build and is busy" in e and prov("ST-new") == ["none"]
ok_jobs = rd("new-jobs.rc") == "0" and "busy" not in rd("new-jobs.out")
ok_same = rd("same-run.rc") == "0" and prov("ST-same") == ["qwen"]
print("new-build run on busy old server: %s rc=%s provider=%s err=%r" % ("PASS" if ok_run else "FAIL", rd("new-run.rc"), prov("ST-new"), e[:300]))
print("new-build jobs on busy old server: %s rc=%s out=%r" % ("PASS" if ok_jobs else "FAIL", rd("new-jobs.rc"), rd("new-jobs.out")[-120:]))
print("same-build run (control): %s rc=%s provider=%s" % ("PASS" if ok_same else "FAIL", rd("same-run.rc"), prov("ST-same")))
print("STALE:", "PASS" if ok_run and ok_jobs and ok_same else "FAIL")
PY
```

## Sau soát (điều phối, 19/10)

- Soát an ninh độc lập vòng 3: **KET-LUAN: DAT** (3 mục phải sửa đóng đúng; #4–#11 đúng; sửa `restart_stale_test.go` không
  phải sửa test cho xanh). Điều phối chạy lại: Home e2e EXPOSED trên `v2-final` → NOT_EXPOSED; C5 `ALLOW=ok`; các gói
  bị đụng: 534 pass, chỉ `TestV3Exposure_*` (vốn đỏ) đỏ.
- Commit `26bed1e9` trên nhánh `v3-poc-p02-policy` (24 tệp mã/docs/test; không gồm 6 tệp test phơi nhiễm và `exposure.sh`).
  Patch cuối thay bản cũ cùng tên; `P02-exposure-harness.patch` giữ riêng.
- **P0-2b (nên sửa, chưa làm, cần soát riêng):** (1) `homeEnv` đặt lại `AGENT_CLI_LAUNCHER` theo env gốc (bước thứ hai qua
  Alt+W khi chạy binary trần) + assert; (2) `ChildEnv` cho env của lệnh `!cmd` (local lọt khoá tin cậy từ `.env`);
  (3) unit test `ensureServer` với server cũ đang bận; (4) `TestConnectToServerSendsNoAgentRuns` đặt `CRUSH_CACHE_DIR`
  (đang ghi log vào `~/.cache/crush` thật); (5) README:24 thêm "trước đó lệnh chạy model mới bị từ chối".
- **Việc riêng, có từ V2:** CLI con claude/codex nhận `ANTHROPIC_BASE_URL`/`ANTHROPIC_API_KEY`/`OPENAI_BASE_URL`/
  `CLAUDE_CONFIG_DIR`/`CODEX_HOME` từ `.env` của repo khi đã bật giao việc; lỗ hẹp của kiểm bản server (`--host tcp://`,
  `VersionInfo` lỗi, đua spawn, `recoverWorkspace`).
