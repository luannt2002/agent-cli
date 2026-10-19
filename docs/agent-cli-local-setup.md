# Cài agent-cli trên máy cá nhân hoặc server nhà

Fork từ Crush; giữ nguyên LICENSE.md và thông báo upstream.

## Cài đặt

Cần sẵn:

- `git` và `go` (phiên bản trong go.mod; Go có thể tải toolchain tương ứng): để lấy mã và build.
- `sha1sum` hoặc `shasum`: launcher dùng để chọn server riêng cho từng dự án (mục "Một server cho
  mỗi dự án"). Thiếu cả hai thì launcher dừng với lỗi rõ, trừ khi chạy `CRUSH_CLIENT_SERVER=0`.
- `jq`: installer dùng để đọc `crush.json` cũ (thiếu thì coi như `crush.json` đang khai provider,
  xem "Chuyển từ `crush.json` cũ"); `agent-cli-tmux new --model` và `agent-cli-tmux rm` cũng cần.
- `tmux`: chỉ cho `agent-cli-tmux`.
- Chạy các script kiểm trong `scripts/`: thêm `python3` (server LLM giả của
  `test-agent-cli-run-json.sh` và `test-agent-cli-delegation.sh`) và lệnh `script` của util-linux
  (ca terminal của `test-agent-cli-wrapper.sh`; thiếu thì ca đó bị bỏ qua).

```bash
git clone https://github.com/luannt2002/agent-cli.git
cd agent-cli
bash scripts/install-agent-cli.sh
export PATH="$HOME/.local/bin:$PATH"
```

Script build tối đa hai luồng với `CGO_ENABLED=0 GOEXPERIMENT=greenteagc` (như bản phát hành chính thức; phiên bản
hiển thị lấy từ VCS stamping của `go build`), rồi chép hai tệp mẫu nếu chưa có (không bao giờ ghi đè; riêng trường hợp đã có `crush.json` khai provider, xem "Chuyển từ `crush.json` cũ"):
`~/.config/agent-cli/crushrc` (cấu hình chính) và `~/.config/agent-cli/env` (mode 600,
chứa endpoint, model, key). Sau khi cài, mở `~/.config/agent-cli/env` và điền giá trị thật
theo mục dưới. Tệp env nằm ngoài repo; không commit key hoặc OAuth token. Sao lưu cấu hình
thật riêng bằng kênh an toàn.

Installer cài hai lệnh: `agent-cli` (wrapper, trỏ tới binary `lib/agent-cli/crush`) và `agent-cli-tmux`
(**bản chép** của `scripts/agent-cli-tmux.sh`, không phải symlink vào thư mục checkout; muốn lấy bản mới
thì chạy lại installer). Cuối lần chạy, nếu `crushrc` hoặc `env` đang dùng khác mẫu trong repo (mẫu có thể
đã thêm biến hay chú thích mới), installer in sẵn lệnh `diff -u <tệp của bạn> <mẫu>` để bạn tự xem và gộp
tay; nó không bao giờ ghi đè tệp của bạn (với env, `diff` hiện cả key: đừng chạy khi chia sẻ màn hình).
Nếu `~/.config/agent-cli` là symlink thì installer chỉ siết quyền (về 700) thư mục đích khi đích là của chính
bạn, và nói rõ đã làm; symlink hỏng thì dừng.

Biến môi trường của installer, launcher và crushrc:

| Biến | Dùng ở | Ý nghĩa | Mặc định |
|---|---|---|---|
| `AGENT_CLI_INSTALL_DIR` | installer | gốc cài: `bin/agent-cli`, `bin/agent-cli-tmux`, `lib/agent-cli/crush` nằm dưới đó. Đường dẫn tương đối được đổi sang tuyệt đối lúc cài (wrapper ghi cứng đường dẫn binary) | `~/.local` |
| `AGENT_CLI_SKIP_BUILD` | installer | đúng `1` thì bỏ `go build` và dùng binary sẵn có ở `<gốc cài>/lib/agent-cli/crush` (test launcher đặt binary giả vào đó); giá trị khác `1` vẫn build | không đặt |
| `AGENT_CLI_ENV_FILE` | crushrc | tệp env mà crushrc `source` (bỏ qua nếu không có) | `~/.config/agent-cli/env` |
| `AGENT_CLI_RUNTIME_DIR` | launcher | thư mục chứa socket server của từng dự án; phải đã tồn tại, thuộc bạn, mode 700 (launcher chỉ siết quyền thư mục dự phòng `~/.cache/agent-cli`), và đủ ngắn để socket không quá 107 byte | `/run/user/$UID`, rồi `$XDG_RUNTIME_DIR`, rồi `~/.cache/agent-cli` |

## Cấu hình model

Provider và model được cấu hình bằng biến dùng chung `AGENT_CLI_*` trong
`~/.config/agent-cli/env`. `crushrc` (bản mẫu: `agent-cli.crushrc.example`) `source` tệp này
(đổi đường dẫn bằng `AGENT_CLI_ENV_FILE`; bỏ qua nếu tệp không có) rồi khai provider chính cùng
model lớn/nhỏ: hoặc một provider tên `main` tự khai bằng `AGENT_CLI_BASE_URL` (cách 1 và 2 bên
dưới), hoặc một provider dựng sẵn trong catalog chọn bằng `AGENT_CLI_PROVIDER_ID` (cách 3). Mẫu env:
`agent-cli.env.example`, chỉ chứa placeholder.

| Biến | Ý nghĩa | Mặc định |
|---|---|---|
| `AGENT_CLI_PROVIDER_ID` | ID provider dựng sẵn trong catalog (vd `zai`, `openai`, `anthropic`); đặt thì dùng cách 3 và bỏ qua `AGENT_CLI_BASE_URL`, `AGENT_CLI_PROVIDER_TYPE`, `AGENT_CLI_PROVIDER_NAME`, `AGENT_CLI_CAN_REASON`, `AGENT_CLI_CONTEXT_WINDOW` | không có |
| `AGENT_CLI_PROVIDER_TYPE` | loại provider: `openai-compat`, `openai`, `anthropic`, `ollama`... | `openai-compat` |
| `AGENT_CLI_PROVIDER_NAME` | tên hiển thị của provider `main` | `main` |
| `AGENT_CLI_BASE_URL` | URL gốc của API (bắt buộc ở cách 1 và 2); dạng URL theo loại: `openai-compat`, `openai`, `ollama` tới `/v1`; `anthropic` là URL gốc KHÔNG có `/v1` | không có |
| `AGENT_CLI_API_KEY` | API key; để rỗng cho model local, và cho `openai`/`xai`/`copilot`/`hyper` đã đăng nhập (cách 3) | rỗng |
| `AGENT_CLI_MODEL` | model chính (bắt buộc) | không có |
| `AGENT_CLI_SMALL_MODEL` | model nhỏ (đặt tiêu đề...) | `AGENT_CLI_MODEL` |
| `AGENT_CLI_CAN_REASON` | model của provider `main` có suy luận không: `true`/`false` (nhận cả `1`/`0`, `yes`/`no`, không phân biệt hoa thường); giá trị khác làm khởi động dừng và nêu đúng tên biến này | `false` |
| `AGENT_CLI_CONTEXT_WINDOW` | context window khai cho model (áp cho cả model chính và nhỏ) | 128000 |
| `AGENT_CLI_MAX_TOKENS` | số token đầu ra tối đa (áp cho cả model chính và nhỏ); model có suy luận cần đủ lớn vì phần nghĩ cũng tính vào đây | min(32000, `AGENT_CLI_CONTEXT_WINDOW`/4) |

Tệp env là Bash nên cách đặt nháy quan trọng. Giá trị có ký tự đặc biệt (`<`, khoảng trắng) phải nằm
trong nháy. **Key dạng chữ thường nên đặt trong nháy ĐƠN** (`AGENT_CLI_API_KEY='sk-...'`): nháy kép để
shell mở rộng `$`, dấu backtick và `\`, nên key có các ký tự đó bị đổi âm thầm (vd `"ab$cd"` ra `ab`).
Nháy kép dành cho giá trị cần mở rộng, như `"$(pass show <entry> | head -n1)"` (xem cách 2).

Thiếu biến bắt buộc thì nạp cấu hình thất bại, và thông báo lỗi nêu đúng biến thiếu ở cả hai
chế độ: chế độ client/server (mặc định) nối phần cuối `stderr.log` của server vào câu
`failed to initialize agent-cli server: ...`, ví dụ `... exit status 1: AGENT_CLI_BASE_URL: đặt
AGENT_CLI_BASE_URL trong ...`.

Dạng `AGENT_CLI_BASE_URL` theo loại provider:

- `openai-compat`, `openai`, `ollama`: phần tới `/v1`, KHÔNG gồm `/chat/completions` (client
  tự thêm; thêm đuôi này vào sẽ ra URL sai).
- `anthropic`: URL gốc, KHÔNG có `/v1` (client tự thêm `/v1/messages`).

Crushrc mẫu tắt metrics và tự cập nhật catalog (dùng catalog nhúng sẵn), KHÔNG tắt provider dựng sẵn
(cần cho `agent-cli login` và `Alt+P`; muốn tắt thì bật dòng `option default-providers false` trong mẫu),
và ẩn các tool chạm mạng
`sourcegraph`, `fetch`, `agentic_fetch`, `download` (tên phải đúng tên tool; tên sai bị bỏ qua
không báo lỗi). Mẫu KHÔNG `permissions allow` tool đọc nào: trong thư mục dự án `view`, `ls`,
`glob`, `grep` chạy không hỏi; cả bốn tool hỏi quyền khi chạm ngoài dự án (ví dụ tệp env chứa
key), kể cả qua symlink trong dự án trỏ ra ngoài (đường dẫn được giải symlink trước khi so), và
allow sẽ bỏ luôn lời hỏi đó. Bản crushrc cài từ mẫu cũ còn dòng
`permissions allow view ls glob grep rg`: nên xoá dòng đó.

### Cách 1: model local (không cần key)

Chạy một server tương thích OpenAI trên máy (llama.cpp, vLLM, Ollama...), rồi:

```bash
AGENT_CLI_PROVIDER_TYPE=openai-compat
AGENT_CLI_BASE_URL="http://127.0.0.1:<port>/v1"
AGENT_CLI_API_KEY=
AGENT_CLI_MODEL="<model-id>"
```

`<model-id>` là tên model server đó báo ra. Trên server nhà, `127.0.0.1` chỉ là chính server
nhà: model phải chạy ở đó, hoặc trỏ tới địa chỉ thực truy cập được. Không mở cổng model ra
công khai chỉ để thử.

### Cách 2: provider qua API key

```bash
AGENT_CLI_PROVIDER_TYPE=openai-compat
AGENT_CLI_BASE_URL="https://<your-endpoint>/v1"
AGENT_CLI_API_KEY='<your-api-key>'
AGENT_CLI_MODEL="<model-id>"
```

Key không vào repo và không dán vào chat. Nên lấy key từ kho bí mật, vì tệp env được shell
nạp nên `$(lệnh)` chạy được:

```bash
AGENT_CLI_API_KEY="$(pass show <entry> | head -n1)"
```

`head -n1` vì `pass show` in cả các dòng ghi chú sau dòng mật khẩu; thiếu nó thì key gồm nhiều dòng.

Tránh để key dạng văn bản thường trong tệp env. Trong TUI, `view` hỏi quyền khi đọc tệp ngoài thư mục dự án
(kể cả `~/.config/agent-cli/env`), nhưng `run`, `loop`, `routine`, `--yolo` và "Allow for Session" cho qua lời hỏi
đó, nên model vẫn có thể đọc key. Mode 600 chỉ chặn người dùng khác, không chặn agent chạy dưới tài khoản của bạn.

Kiểm cấu hình đã nạp mà không gọi model: `agent-cli models` liệt kê `main/<model-id>`.

### Cách 3: provider dựng sẵn trong catalog

```bash
AGENT_CLI_PROVIDER_ID=zai
AGENT_CLI_MODEL="glm-5.3"
AGENT_CLI_SMALL_MODEL="glm-5.3-flash"
AGENT_CLI_API_KEY='<khoá z.ai>'
```

Catalog lo endpoint, danh sách model, mức effort (`Alt+E`) và khả năng suy luận, nên không cần
`AGENT_CLI_BASE_URL` hay `AGENT_CLI_CAN_REASON`. Mỗi provider có thông tin xác thực riêng:

- Phần lớn provider (`zai`, `anthropic`, `openrouter`...) bắt buộc `AGENT_CLI_API_KEY`; thiếu thì khởi
  động dừng và nêu đúng biến.
- `openai` (tài khoản ChatGPT), `xai` (Grok), `copilot`, `hyper` còn đăng nhập được bằng tài khoản
  (`agent-cli login openai`, `login grok`, `login copilot`, `login` không đối số cho hyper). Đã đăng nhập thì để
  TRỐNG `AGENT_CLI_API_KEY`: crushrc không phát key rỗng đè lên token đã lưu (nằm ở
  `~/.local/share/agent-cli/crush.json`, được nạp SAU crushrc). Ngược lại, đã đăng nhập thì token đăng nhập
  thắng `AGENT_CLI_API_KEY` (đã có test cho thứ tự nạp của `xai`); muốn dùng key thì `agent-cli logout <nền tảng>`
  trước. Chưa đăng nhập và chưa có key thì provider chỉ là chưa được cấu hình, agent-cli vẫn khởi động được
  (cần vậy để chạy được lệnh login). Cấu hình này đã kiểm bằng test nạp cấu hình với token giả; việc đăng nhập
  thật (trình duyệt) chưa được thử trong đợt sửa này.
- `AGENT_CLI_PROVIDER_ID` gõ sai (không có trong catalog, và không có base URL để thành provider tự khai)
  làm khởi động DỪNG, lỗi nêu tên provider và model đang chọn nó (`provider "foo" is selected as the large
  model but is not a built-in provider and has no base_url`). Trước đây provider bị bỏ qua lặng lẽ, agent-cli
  chọn model của provider khác và ghi lựa chọn đó xuống đĩa. Provider không ai chọn mà thiếu base URL vẫn chỉ bị
  cảnh báo trong log như cũ.

### Attribution trong commit và PR

Mặc định upstream bảo model thêm `Assisted-by: Crush:<model>` vào commit và dòng `Generated with Crush`
vào commit/PR do agent tạo (qua mô tả của tool bash). Crushrc mẫu tắt cả hai bằng `option attribution-trailer-style none` và
`option attribution-generated-with false`. Bản crushrc cài từ mẫu cũ chưa có hai dòng này: thêm vào, rồi
thoát hết client của dự án và mở lại. Muốn bật lại: `option attribution-trailer-style assisted-by` (hoặc `co-authored-by`) và
`option attribution-generated-with true`; dòng chú thích trong mẫu ghi sẵn cách này.

### Sửa env khi agent-cli đang chạy

Mỗi dự án có một server nền. Cấu hình (crushrc, crush.json, khoá trong tệp env mà crushrc `source`)
được đọc khi workspace của dự án được dựng. Chính sách tin cậy thì nạp MỘT lần cho mỗi tiến trình
server và không bao giờ nạp lại: `AGENT_CLI_ALLOWED_MODELS`, `AGENT_CLI_DELEGATION` (export ở shell
hay trong `~/.config/agent-cli/env`) cùng các biến tin cậy khác (`AGENT_CLI_UNATTENDED`,
`AGENT_CLI_LLM_LEDGER`, CLI và mức quyền của tool giao việc ở dưới), theo env của client đã dựng
server. Sửa `~/.config/agent-cli/env` (hay crushrc) KHÔNG tác dụng với workspace đang mở: còn một
client (TUI hay `run`) nào của dự án đó đang sống thì nó vẫn dùng giá trị cũ (đã đo: `touch
crushrc` cũng không kích hoạt nạp lại). Client chạy model (TUI, `run`, `send`, `loop`) mở sau mà
chính sách tin cậy (env gốc của nó và tệp env; `.env` của repo không tính) khác chính sách server
đang áp, hoặc `--yolo` khác workspace đang mở, thì bị TỪ CHỐI với lỗi `trust policy differs from
the running agent-cli server: <tên biến>` (chỉ tên, không giá trị). Chỉ 11 biến tin cậy bị so:
`AGENT_CLI_ALLOWED_MODELS`, `AGENT_CLI_DELEGATION`, `AGENT_CLI_UNATTENDED`,
`AGENT_CLI_UNATTENDED_COMMIT`, `AGENT_CLI_ENV_FILE`, `AGENT_CLI_LLM_LEDGER`, `HOME`,
`AGENT_CLI_CLAUDE_BIN`, `AGENT_CLI_CODEX_BIN`, `AGENT_CLI_CLAUDE_PERMISSION_MODE`,
`AGENT_CLI_CODEX_SANDBOX`. Biến môi trường thường, kể cả biến mà `crush.json` dùng qua `$VAR` cho
`api_key`/`base_url`, KHÔNG được so: client vẫn gắn vào và chạy theo giá trị của client đã dựng
server (giới hạn như allowlist chỉ so tên provider/model, không kiểm `base_url`). Phép so là so
chuỗi thô: khác thứ tự trong allowlist, `on`/`ON`/`1`, hay khác `HOME`/`AGENT_CLI_ENV_FILE` dù
allowlist hiệu lực bằng nhau đều bị từ chối. Lệnh không gửi prompt (`jobs`, `login`, `logout`)
không bị so chính sách tin cậy và `--yolo` (vẫn bị so `--channels`; `login` khi tự nối lại sau lúc
server mất workspace thì bị so như lệnh chạy model). Server khác bản/build mà đang bận (không thay
được) thì lệnh chạy model bị từ chối, lệnh không prompt vẫn dùng nó. Từ tiến trình con mà Home
dựng (workspace, Home khi quay lại, kể cả launcher mà Home đó chạy) tới lệnh `--verify` của loop
và lệnh `!` bạn gõ trong TUI (local lẫn server), biến tin cậy đều lấy theo env gốc, `.env` của
repo không chen vào được (biến thường của `.env` thì vẫn tới các lệnh đó; verify của `routine run`
nhận công tắc không người trực của chính lượt chạy). Để áp giá trị mới: thoát hết client của dự
án đó, chờ
server tắt (khoảng 60 giây sau khi client cuối thoát, xem `agent-cli ps`) rồi mở lại; hoặc dừng
thẳng server của dự án: `pgrep -af 'crush server --host'` liệt kê các server (tên socket
`agent-cli-<8 hex>` xem ở mục "Một server cho mỗi dự án" bên dưới), rồi `kill <pid>`; việc đang
chạy ở dự án đó sẽ mất.

Chỉ sửa tệp trên đĩa thì không nạp lại, nhưng MỌI lần ghi cấu hình của chính workspace đó (vd chọn model
trong TUI) làm nó nạp lại toàn bộ, gồm chạy lại crushrc và `source` lại env: đã đo ở tầng cấu hình (sửa
crushrc trên đĩa rồi ghi một trường cấu hình khác thì giá trị mới có hiệu lực). Đừng dựa vào đó để áp cấu hình; nó chỉ giải thích vì sao
đôi khi giá trị mới "tự" xuất hiện sau khi đổi model. Các tool `claude_code`/`codex` thì khác: biến của chúng
là env của tiến trình server (xem mục giao việc ở dưới).

### Chuyển từ `crush.json` cũ

`~/.config/agent-cli/crush.json` vẫn được đọc và gộp chung với `crushrc`, nhưng cùng thư mục thì
`crushrc` thắng. Vì vậy nếu `crush.json` (kể cả khi nó là symlink; installer theo symlink để
đọc, không sửa) đang khai provider, installer KHÔNG tạo `crushrc`/`env` (sẽ thay provider đang
chạy bằng placeholder): nó chỉ đặt `crushrc.example` và `env.example` cạnh `crush.json` (không
ghi đè bản có sẵn) và in hướng dẫn. Cũng vậy khi thiếu `jq` hoặc `crush.json` hỏng (không xác
định được thì coi như có provider). Để chuyển: xoá `providers` và `models` khỏi `crush.json`
(giữ theme...), đổi tên `crushrc.example` thành `crushrc`, `env.example` thành `env`
(`chmod 600`), điền giá trị rồi chạy lại. `crush.json` không khai provider thì installer tạo
`crushrc` + `env` như bình thường.

## Sử dụng

```bash
agent-cli                      # Home: chọn workspace rồi session (xem mục Home)
agent-cli -c /path/to/project  # vào thẳng workspace, bỏ qua Home (-c tương đối tính theo thư mục hiện tại)
agent-cli run "Đọc repo và tóm tắt kiến trúc, chưa sửa code."   # CẢNH BÁO: run TỰ DUYỆT mọi quyền (xem dưới)
agent-cli --continue
agent-cli login openai
agent-cli run -m main/<model-id> --reasoning-effort high -q "..."   # chọn model, effort; -q ẩn spinner, -v in log
git diff | agent-cli run "Review thay đổi này"                     # nội dung stdin đứng trước prompt
agent-cli session list --json    # cùng nhóm: show <id>, last, rename <id> <tên>, delete <id>
agent-cli logs -f                # bám log; -t N chỉ N dòng cuối
```

Trong TUI và `send`: sửa file, chạy shell (trừ lệnh chỉ đọc an toàn), đọc ngoài thư mục
dự án và giao việc cho Claude Code/Codex đều hỏi quyền; không dùng `--yolo` mặc định.
**`agent-cli run` thì tự duyệt MỌI lời xin quyền của session nó chạy** (session mới, hoặc
session có sẵn qua `-s`/`-C`; gồm bash, edit, claude_code, codex...): ràng buộc duy nhất là lời
trong prompt (dòng "chưa sửa code" ở ví dụ trên chỉ là lời dặn model, không phải rào chắn). Cần được hỏi
thì dùng TUI hoặc `send` (`send` KHÔNG tự duyệt). `run` không có tool `question` (không ai trả lời
được nên model làm tiếp thay vì dừng hỏi) và chờ MCP khởi động xong trước khi chạy. Nó dùng chế độ
agent hiện hành của dự án: TUI đang bật plan mode thì `run` cũng chỉ đọc và lập kế hoạch. Mọi lệnh
gọi sai cú pháp (cờ lạ, thiếu/thừa tham số, lệnh con sai tên) trả mã 2; chạy lỗi trả mã 1. Prompt kết thúc mà model
chưa ra chữ hay tool call nào, xét cả các lượt tự làm tiếp của prompt đó (`empty reply`; bộ lọc nội dung của provider
chặn thì báo riêng `content filter stopped the reply…`), hay lượt dừng vì lặp cùng một tool call, cũng là chạy lỗi.
Bị cắt vì hết ngân sách output thì vẫn tự làm tiếp trước; hết 3 lần mà chưa ra gì mới là lỗi. Mã 0 chỉ nghĩa là lượt
chạy hết không lỗi, không phải việc đã đúng: "xong có bằng chứng" là `loop --verify` (mục "Loop và Routine").
Cấu hình project có thể ghi đè cấu hình global.
Metrics và tự cập nhật catalog provider bị tắt (launcher đặt `CRUSH_DISABLE_METRICS`,
`CRUSH_DISABLE_PROVIDER_AUTO_UPDATE`; crushrc mẫu tắt thêm hai option tương ứng). Kiểm tra bản mới của
upstream (hỏi GitHub) **mặc định TẮT ngay trong binary**, kể cả khi chạy binary không qua launcher: fork không có
feed release riêng để so sánh. Chỉ bật khi đặt tường minh `CRUSH_DISABLE_UPDATE_CHECK=false` (hoặc `0`; giá trị
hiểu theo `strconv.ParseBool`); không đặt, rỗng, `1`/`true` hay giá trị không hiểu được đều là không gọi GitHub.
Launcher luôn đặt `CRUSH_DISABLE_UPDATE_CHECK=1` nên qua launcher kiểm tra này không bao giờ chạy; chỉ chạy thẳng
binary (`~/.local/lib/agent-cli/crush`) mới bật được.
Các cờ tắt này không chặn mọi request mạng: request model tới provider đang chọn, MCP và các tool mạng mà bạn
không `permissions deny` (crushrc mẫu chặn `sourcegraph`, `fetch`, `agentic_fetch`, `download`) vẫn đi.

Model chọn trong TUI (kể cả bật/tắt thinking, mức reasoning) giờ thuộc về dự án:
được ghi vào `<project>/.crush/crush.json`, nên đổi model ở worktree này không
đổi model ở worktree khác. Mở lại hay tiếp một session cũ (TUI, `run -s/-C`) không ghi
gì: tin nhắn sau chạy model đang chọn của dự án; session từng chạy model khác thì TUI báo ở
thanh trạng thái, `run` in ra stderr `session <id> last ran on <cũ>; continuing on the project
model <đang chọn> (pick another with -m)`. Model, effort đổi được bất cứ lúc nào, kể cả khi
có run ở session này hay session khác: mỗi lượt chạy giữ model và bộ tool lúc nó bắt đầu, prompt
xếp hàng giữ model lúc gửi, nên thiết lập mới chỉ áp cho tin nhắn gửi sau, không đổi provider
giữa chừng lượt đang chạy (agent con của lượt cũng vậy). Thứ tự gộp config, từ thấp đến cao: `~/.config/agent-cli/crush.json`
< `~/.local/share/agent-cli/crush.json` < `crush.json` / `.crush.json` trong repo
< `<project>/.crush/crush.json` (cao nhất). Hệ quả:

- Dự án chưa từng chọn model thì theo model mặc định toàn cục. Đã chọn một lần
  thì không còn theo mặc định nữa.
- Muốn dự án theo mặc định trở lại: xoá `models` (và `recent_models`) khỏi
  `<project>/.crush/crush.json`.
- Muốn đổi mặc định toàn cục: các bản TUI cũ ghi lựa chọn vào
  `~/.local/share/agent-cli/crush.json` (nằm TRÊN file trong `~/.config`), nên
  phải xoá `models` khỏi file đó rồi đặt `models` ở `~/.config/agent-cli/crush.json`;
  chỉ sửa file trong `~/.config` thì không có tác dụng nếu file data còn `models`.

## Home: chọn workspace và session

`agent-cli` không tham số (chỉ `-d/--debug` là được) trên terminal mở **Home**, kiểu "Open Recent":

- **Active**: workspace có server đang sống, kèm số session `running` / `waiting` (chờ duyệt hoặc trả lời).
  Server có socket mà không trả lời trong 3 giây hiện `server not responding (unknown)`, không bao giờ
  bị coi là rảnh. Home chỉ hỏi server đang chạy (quét socket như `agent-cli ps`), không khởi động server nào.
- **Pinned**, rồi **Recent**: 5 workspace mở gần nhất. 5 chỉ là giới hạn hiển thị: sổ
  (`~/.local/share/agent-cli/projects.json`) giữ hết; gõ để tìm theo tên/đường dẫn trong toàn bộ sổ,
  hoặc chọn `→ Show all / search`.
- `→ Open current folder`, `→ Open other folder…` (nhập đường dẫn thư mục đã có; chỉ đăng ký vào sổ,
  không `git init`/clone/tạo worktree). Thư mục con của repo quy về gốc repo.
- Phím: gõ để tìm · `↑↓` chọn · `enter` mở · `ctrl+p` ghim/bỏ ghim · `ctrl+x` bỏ khỏi danh sách một
  workspace có thư mục đã mất (chỉ sửa sổ, không xoá dữ liệu) · `esc` xoá ô tìm, `esc` lần nữa thoát.

Chọn workspace -> danh sách session (tối đa 50, mới nhất trước) và `→ New session`. Server đang sống
thì lấy session và trạng thái thật qua server; không có server thì đọc thẳng DB của workspace ở chế độ
chỉ đọc và trạng thái là `unknown`. Workspace từng mở từ thư mục con có DB riêng (`sub/.crush`) thì
session ở mọi DB đều hiện và mở đúng thư mục chứa nó. `→ New session` mở màn hình trống, chưa tạo
session tới khi bạn gửi prompt; chọn session đang chạy chỉ mở để xem/tiếp, không gửi lại gì. `esc` về
danh sách workspace.

Home chạy TUI của workspace như process con trên cùng terminal (`agent-cli -c <dir> [--session <id>]`,
qua chính launcher nên mỗi dự án vẫn có server riêng). Trong TUI, `Alt+W` thoát về Home và không gửi
lệnh huỷ nào; với client/server (mặc định của launcher) server giữ run đang chạy khi client cuối rời
(xem "Đã biết" bên dưới về các trần thời gian). **In-process
(`CRUSH_CLIENT_SERVER=0`): `Alt+W` đóng app nên huỷ run đang chạy**; khi đang có run, TUI hỏi
xác nhận ("Stop the running task and go to Home?") thay vì huỷ ngay. `Ctrl+P` → Home làm đúng như
`Alt+W` (dùng khi terminal không gửi phím Alt).
`agent-cli -c dir` mở thẳng rồi bấm `Alt+W` cũng về Home (process tự thay bằng Home).

Chỉ thư mục trên máy này (workspace của máy khác trong sổ bị ẩn). Làm việc trên server: SSH vào
server rồi chạy `agent-cli` ở đó.

## Một server cho mỗi dự án

Launcher bật sẵn `CRUSH_CLIENT_SERVER=1`: mọi `agent-cli` (TUI, `run`, script) chạy
như client và nối vào một server nền. Launcher tự thêm
`--host unix:///run/user/$UID/agent-cli-<8 hex đầu của sha1(thư mục gốc)>.sock`
khi bạn không tự truyền `--host`/`-H`. Thư mục gốc là `git rev-parse --show-toplevel`
của `-c/--cwd` (hoặc thư mục hiện tại); ngoài repo git thì lấy đường dẫn thật
(`pwd -P`). Kết quả: mọi process cùng một dự án dùng chung một server nên trạng thái
session (bận, chờ duyệt) đúng ở mọi nơi; dự án khác thì server khác, log và DB tách riêng.
Server tự tắt 60 giây sau khi workspace cuối được nhả (`CRUSH_SERVER_IDLE_TIMEOUT`, giây; `0` = tắt ngay); mở
workspace mới trong khoảng đó thì huỷ lệnh tắt. Socket nằm ở
`AGENT_CLI_RUNTIME_DIR` nếu bạn đặt, mặc định `/run/user/$UID`; thư mục đó chưa có thì tới `$XDG_RUNTIME_DIR`, rồi
cuối cùng tới `$HOME/.cache/agent-cli` (launcher tạo và siết mode 700). Cố ý KHÔNG bao giờ dùng `/tmp` vì ai
cũng ghi được và tên socket đoán được. `AGENT_CLI_RUNTIME_DIR` bạn tự chọn phải đã tồn tại, thuộc bạn và mode 700
(launcher không siết quyền hộ).

- Log server: `~/.cache/agent-cli/server-<host>/crush.log` (`<host>` là địa chỉ socket
  đã thay ký tự lạ bằng `_`). Mỗi dự án một thư mục, không trộn log. Khi tiến trình panic, tệp
  `crush-panic-<tên>-<thời điểm>.log` (quyền 600) nằm cùng thư mục với log, không còn rơi vào thư mục đang làm việc.
- Trả lời bị cắt vì hết ngân sách output (`max_tokens`) thì lượt tự làm tiếp (tin "Your previous reply was
  cut off..."), tối đa 3 lần liên tiếp không tiến triển cho mỗi prompt của bạn (lượt nào đã chạy xong
  ít nhất một tool call thì đếm lại từ đầu). Lượt làm tiếp chạy trước prompt bạn gõ thêm sau đó. Ngân sách mặc định của provider `main` là
  min(32000, `AGENT_CLI_CONTEXT_WINDOW`/4); đặt `AGENT_CLI_MAX_TOKENS` để đổi.
- Tự tóm tắt khi phần context còn trống ≤ 20% context window (window ≤ 200k token) hoặc ≤ 20.000 token (window
  lớn hơn); model không khai `context_window` thì không bao giờ tự tóm tắt. Bước tóm tắt có hạn chờ gấp 3
  `request-timeout` (mặc định 6 phút). Tóm
  tắt vẫn hỏng thì việc không dừng: lịch sử giữ nguyên, lượt làm tiếp chạy ngay (tin "Continue the task
  where you stopped..."), tự tóm tắt tạm tắt cho session tới prompt sau của bạn. Context đầy thật thì
  provider báo lỗi và lượt dừng với lỗi đó; khi đó tóm tắt tay (`Ctrl+P` → Summarize) hoặc mở session mới.
  Bản tóm tắt bị cắt vì hết ngân sách output cũng tính là hỏng (không thay lịch sử). Tóm tắt xong mà riêng
  phần đầu vào của bước đầu ở lượt làm tiếp đã vượt ngưỡng, và việc còn dở (bước đó còn gọi tool), thì lượt dừng
  với lỗi `context window too small…` thay vì tóm tắt lại mãi (phần cố định của prompt quá lớn so với window).
  Bước đó là câu trả lời cuối thì lượt xong bình thường; chỉ phần output (vd ghi một tệp lớn) đẩy tổng qua ngưỡng
  thì tóm tắt tiếp như thường. Một prompt tự tóm tắt tối đa 10 lần khi việc
  còn dở; quá thì lượt dừng với lỗi `auto-summarized 10 times…`, gửi prompt mới để làm tiếp.
- Server bị dừng (`kill <pid>`, SIGTERM, SIGHUP) thì dừng từng workspace trước: run đang
  chạy bị huỷ và được ghi lại, không bỏ dở giữa chừng. Server chết đột ngột (SIGKILL, mất
  điện) thì tool call đang chạy dở được ghi "Interrupted" khi session được mở lại (TUI,
  `session show`, `run -s`), không hiện "đang chạy" mãi; session con của sub-agent chỉ được
  ghi khi session cha không còn run. Kết quả ghi cho model dặn kiểm tra trạng thái thật
  trước khi chạy lại, vì lệnh có thể đã chạy một phần (bản vá upstream `fix/dangling-tool-calls`
  và `fix/shutdown-records-work`, port 15/10).
- Tắt: `CRUSH_CLIENT_SERVER=0 agent-cli` chạy in-process như trước và launcher không
  thêm `--host`. Giới hạn của chế độ này: danh sách session vẫn đủ (đọc từ DB) nhưng
  trạng thái bận/chờ duyệt chỉ đúng với session do chính process đó chạy. Launcher hiểu
  giá trị đúng như `strconv.ParseBool` của Go: bật là `1 t T TRUE true True`, tắt là
  `0 f F FALSE false False`. Giá trị khác (`yes`, `on`, `2`...) Go coi là tắt, nên launcher
  cũng chạy in-process và in cảnh báo thay vì thêm `--host`.
- Không chọn được server riêng thì launcher DỪNG (mã 1, lỗi rõ) thay vì chạy tiếp không `--host`: client sẽ rơi về
  server mặc định dùng chung và mất cách ly theo dự án (session, trạng thái bận, log lẫn nhau). Hai trường hợp: socket
  vượt 107 byte (đặt `AGENT_CLI_RUNTIME_DIR` ngắn hơn) và thiếu cả `sha1sum` lẫn `shasum`. Lối thoát: cài công
  cụ còn thiếu, hoặc `CRUSH_CLIENT_SERVER=0`. Bước chọn server (và hai lỗi trên) được bỏ qua khi bạn tự truyền
  `--host`/`-H`, với `ps` và với Home.
- Tự chọn server: `agent-cli --host unix:///đường/dẫn.sock ...` được giữ nguyên.
- Home (`agent-cli` không tham số trên terminal) không có `--host`: chưa chọn workspace nên chưa có
  server; Home gọi lại launcher với `-c <dir>` cho từng workspace.
- Đã biết: client mất stream mà không thoát sạch thì server giữ workspace (và run) thêm
  một khoảng detach grace, mặc định 10 giây (`backend.DefaultDetachGrace`, đổi bằng
  `CRUSH_SERVER_DETACH_GRACE`); thoát sạch thì bỏ qua grace.
- Hết grace mà workspace còn run thì server không dọn: kiểm lại mỗi 5 giây, dọn khi run
  xong (kiểm bằng `scripts/test-agent-cli-run-json.sh`: client bị SIGKILL giữa run, run vẫn
  xong). Hai trần thời gian:
  - `CRUSH_SERVER_ORPHAN_RUN_TIMEOUT` (giây, mặc định 1800; `0` = dọn ngay như cũ) chỉ tính
    khi run **kẹt**: đang chờ duyệt quyền hoặc trả lời câu hỏi mà không còn client nào. Run
    chạy lại được thì đồng hồ tính lại từ đầu.
  - `CRUSH_SERVER_ORPHAN_MAX_RUN` (giây, mặc định 21600 = 6 giờ; `0` = không trần) cho run
    mồ côi vẫn đang tiến triển: việc giao dài bằng `send` không bị huỷ ở phút 30, nhưng một
    run lặp vô tận không giữ server mãi.
- Lượt của `agent-cli run` gắn với chính tiến trình `run`: Ctrl+C, SIGTERM hay đóng terminal
  (SIGHUP) huỷ **đúng lượt đó** theo RunID, kể cả với `run -s/-C` (lượt khác và prompt xếp
  hàng của session giữ nguyên; Ctrl+C khi prompt chưa kịp tới server thì nó bị huỷ lúc tới).
  `run` bị kill hẳn (SIGKILL, OOM) thì server huỷ lượt của nó sau detach grace, thay vì để lượt
  chạy tiếp không ai giám sát. Server bản cũ chưa có huỷ theo RunID: `run -s/-C` chỉ thôi chờ
  và báo `stopped waiting; the run keeps going on session <id> (this server cannot cancel a
  single run)`.
- `send --wait` thì Ctrl+C/SIGHUP chỉ thôi chờ và báo `stopped waiting; the run keeps going on
  session <id>`: `send` giao việc rồi đi, lượt của nó không gắn với client.
- `run -s/-C` xếp hàng sau lượt khác (vd. lượt TUI) trong cùng session: câu hỏi và yêu cầu
  duyệt quyền của lượt đó để nguyên cho TUI (không tự duyệt, không huỷ session); `run` in một
  dòng `... from another turn is pending on session <id>; this run keeps waiting behind it`
  ra stderr rồi chờ tiếp. Nhận biết theo nội dung prompt trong hàng đợi: prompt trùng y hệt
  của người khác có thể bị đọc nhầm.
- Server đóng luồng sự kiện trước khi run xong (server chết hoặc khởi động lại): `run` và
  `send --wait` báo `event stream closed before the run completed`, mã 1, `--json` có `error`.
- Dừng server (SIGTERM, `kill <pid>`) trong lúc nó đang giữ run cho client đã rời thì run bị
  cắt giữa lượt và session không ghi được dấu kết thúc.
- Chỉ nhận `-c dir`, `-cdir`, `--cwd dir`, `--cwd=dir` đứng riêng; cờ gộp kiểu `-dc dir`
  không được hiểu (launcher quay về thư mục hiện tại).
- Kiểm launcher: `bash scripts/test-agent-cli-wrapper.sh` (cài vào thư mục tạm với binary
  giả, in `PASS`).

## Nhiều dự án trong một cửa sổ terminal

```bash
agent-cli-tmux ~/code/api ~/code/web ~/code/infra
```

Mở (hoặc nối vào) tmux session `agents`, mỗi thư mục một window đặt theo basename, chạy
`CRUSH_CLIENT_SERVER=1 agent-cli -c <đường dẫn tuyệt đối>`. Chạy lại với thư mục đã có thì
chỉ chọn window đó, không đẻ thêm (nhận diện theo đường dẫn, không theo tên). Thư mục không
tồn tại: in lỗi ra stderr, thoát mã 2 và không tạo gì. Đang ở trong tmux thì `switch-client`,
ngoài tmux thì `attach`.

- `prefix+w` liệt kê các window (dự án) để chọn; `prefix+n`/`prefix+p` chuyển nhanh.
- Đổi tên session: `AGENT_CLI_TMUX_SESSION=tên`. Không attach: `AGENT_CLI_TMUX_NO_ATTACH=1`.
- Installer chép `scripts/agent-cli-tmux.sh` thành `~/.local/bin/agent-cli-tmux` (bản chép mode 755, không
  symlink: đổi nhánh hay xoá checkout không làm hỏng nó; chạy lại installer để lấy bản mới). Bản cài cũ là
  symlink thì được thay bằng bản chép, không ghi xuyên vào tệp trong repo.
- Mỗi window là một client nối vào server của dự án đó (xem mục trên). Thoát agent-cli thì window đóng.
- Kiểm: `bash scripts/test-agent-cli-tmux.sh` (tmux server riêng qua `-S <socket trong thư mục tạm>` và
  agent-cli giả, in `PASS`; không đụng tmux thật của bạn).

### Mỗi việc một worktree: `new` / `rm` / `ls`

Chạy song song nhiều việc trên cùng một repo mà không đạp file nhau: mỗi việc có worktree
riêng (pwd riêng, DB/log `.crush/` riêng, model riêng) và một window tmux riêng.

```bash
cd ~/code/api
agent-cli-tmux new fix-login --model main/<model-id>   # worktree + branch agent/fix-login + window
agent-cli-tmux ls                                   # fix-login  agent/fix-login  clean|dirty|broken
agent-cli-tmux rm fix-login                         # dọn khi xong; từ chối nếu còn thay đổi, session đang chạy hoặc file gitignore
```

`rm` không kèm `--force` từ chối khi worktree có file bị gitignore, mà worktree từng chạy agent-cli hầu như
luôn có `.crush/` (DB session và log). Chỉ khi chắc chắn bỏ cả lịch sử đó mới thêm `--force`
(`agent-cli-tmux rm fix-login --force` xoá `.crush/` VĨNH VIỄN); cần giữ thì chép `.crush/` ra trước.

- `new <name> [--model <provider>/<model>] [--repo <dir>] [--base <ref>]`: `<repo>` là
  `git rev-parse --show-toplevel` của `--repo` hoặc thư mục hiện tại (không phải repo git:
  thoát mã 2). Worktree nằm ở `<cha của repo>/<tên repo>-agents/<name>`, trên branch
  `agent/<name>` tạo từ `--base` (mặc định `HEAD`). Worktree đã có thì chỉ mở window; branch
  đã có mà chưa có worktree thì gắn lại branch đó. `<name>` khớp `^[a-z0-9][a-z0-9-]{0,40}$`,
  sai: mã 2.
- `--model` ghi (gộp bằng `jq`, giữ phần cấu hình có sẵn) `models.large` vào
  `<worktree>/.crush/crush.json`; tách provider ở dấu `/` đầu tiên (`openrouter/anthropic/x`
  là provider `openrouter`, model `anthropic/x`). Thiếu `/`: mã 2, không tạo gì. `.crush/`
  được tạo kèm `.gitignore` chứa `*` nên không làm worktree dơ.
- `--model` trên worktree đã có vẫn ghi lại model; agent đang chạy trong window đó phải
  khởi động lại mới nhận.
- `rm <name> [--repo dir] [--force]`: worktree còn thay đổi chưa commit (kể cả file chưa theo dõi, bất kể
  `status.showUntrackedFiles`) hoặc không đọc được trạng thái git thì in lý do, mã 3, không
  xoá gì. Worktree còn session BUSY hoặc WAITING ở gốc hoặc thư mục con (theo `agent-cli ps --json --all`, so path đã chuẩn hoá `pwd -P`; ps lỗi, có stderr, JSON
  hỏng, hoặc hàng BUSY có path không phân giải được đều bị coi là không an toàn và từ chối) cũng bị từ
  chối, mã 3: dưới `CRUSH_CLIENT_SERVER=1` agent chạy ở server riêng nên đóng window KHÔNG
  dừng nó ngay (xem "Đã biết" ở mục trên về detach grace và trần mồ côi); thiếu `agent-cli` trên
  PATH thì bỏ qua kiểm này và cảnh báo. Sạch thì đóng window `<name>` và mọi window mở ở thư mục con của worktree, đợi tiến trình pane
  thoát, gỡ worktree (git kiểm lại, kể cả file agent vừa ghi) và in `branch agent/<name> kept`. Script KHÔNG bao giờ xoá branch; bỏ hẳn thì tự chạy
  `git branch -d agent/<name>` (hoặc `-D` nếu chưa merge).
- `rm` từ chối (mã 3) khi worktree có file bị gitignore, gồm `.crush/` (DB session và log của
  agent việc đó) và các file như `.env`, vì `git worktree remove` xoá chúng vĩnh viễn; stderr
  liệt kê từng mục. Chắc chắn bỏ thì thêm `--force` (chỉ nới kiểm này; kiểm thay đổi chưa
  commit và session BUSY/WAITING vẫn giữ). Vì vậy lần dọn đầu của một worktree đã chạy agent-cli thường bị
  từ chối cho tới khi bạn quyết định bỏ hay giữ `.crush/` (xem ví dụ ở trên).
- `--base` được phân giải tại nơi chạy lệnh (chạy từ worktree con thì lấy HEAD của worktree
  đó). `<name>` là symlink trong `<tên repo>-agents/` thì `new`/`rm` từ chối (mã 2).
- `new`/`rm`/`ls` luôn quy về worktree chính của repo, nên chạy từ trong một worktree con
  vẫn đặt/tìm ở cùng `<tên repo chính>-agents/`. `ls` báo `dirty` khi không đọc được trạng thái, `broken` khi dữ liệu git của worktree mất (vẫn liệt kê tiếp các worktree khác).
- Thư mục thật tên `new`/`rm`/`ls` thì viết `./new` để không bị hiểu là lệnh con.

## Giao việc vào session đang mở: `agent-cli send`

Gửi prompt vào một session của dự án đã có server (ví dụ TUI đang mở ở window tmux khác);
prompt hiện trong TUI đó như tin nhắn bình thường. Dành cho script hoặc agent điều phối
nhiều session/dự án.

```bash
agent-cli send -c ~/code/api --session 3f2a "Thêm test cho handler login"      # trả ngay, in session_id
agent-cli send -c ~/code/web --new --title review --wait "Review commit cuối"  # đợi run xong, in text cuối
agent-cli send --session 3f2a --wait --timeout 10m --json "Chạy lại test"     # một object JSON
```

- Bắt buộc đúng một trong `--session <id>` (ID đầy đủ hoặc tiền tố hash như `run -s`) hoặc
  `--new [--title t]`. Không thấy session: mã 1. Thiếu prompt, thiếu/thừa cờ, hoặc không có
  `CRUSH_CLIENT_SERVER=1` (launcher đã đặt sẵn): mã 2 kèm hướng dẫn.
- Luôn đi qua server của dự án (`-c dir` chọn dự án). Server chưa chạy thì `send` tự dựng như
  `run`; khi đó không có TUI nào để duyệt quyền, nên nhắm vào dự án đã mở.
  `send` chỉ khởi tạo agent khi server chưa có; server đã sẵn sàng thì giữ nguyên (không thay
  coordinator đang chạy của TUI).
- Không `--wait`: về ngay khi server nhận prompt; stdout là `session_id` (hoặc `{"session_id":"..."}`).
- `--wait`: đợi đúng run của lần gửi này (khớp session và run ID, run khác trên cùng session bị
  bỏ qua), rồi in text cuối. `--json` in cùng dạng với `run --json`: `session_id`, `text`,
  `error`, `prompt_tokens`, `completion_tokens`, `cost`. Token là usage của bước cuối, không phải tổng; `cost` là
  tổng cộng dồn của session. `send` không điền `provider`/`model`.
  `--timeout` (mặc định 0 = không giới hạn) hoặc Ctrl-C ngắt việc đợi; `send` không gửi lệnh huỷ run, và run
  sống tiếp khi client cuối thoát (đã kiểm: `scripts/test-agent-cli-run-json.sh`, client bị SIGKILL giữa run mà run
  vẫn xong; trần thời gian xem "Đã biết" ở mục Home: 30 phút khi kẹt chờ duyệt/trả lời, 6 giờ khi vẫn chạy).
  Lỗi run: mã 1, JSON vẫn có `error`.
- Run cần duyệt quyền hoặc trả lời câu hỏi mà không có TUI nào đang xem session đó: `--wait`
  dừng với lỗi (mã 1, JSON có `error`) nêu tool và session. Run vẫn treo ở server tới khi có ai
  mở session đó trong TUI (server gửi lại prompt đang chờ khi đó). Prompt của sub-agent (mang
  ID session con) được quy về session gốc nên cũng làm `--wait` dừng như trên.
- `send --session S --wait` xếp hàng sau lượt khác (vd. lượt TUI) trên S: câu hỏi/yêu cầu duyệt
  quyền của lượt đó không làm `send` thất bại; `send` in `... from another turn is pending on
  session <id>; this run keeps waiting behind it` ra stderr rồi chờ tiếp, giống `run -s`.
- `--title` chỉ đi với `--new`, `--timeout` chỉ đi với `--wait`: sai thì mã 2.
- Chỉ nhận prompt từ đối số, không đọc stdin.
- Chưa kiểm với server và model thật (test chỉ dùng client giả).

## Xem và dừng job bash: `agent-cli jobs`

Mỗi lệnh bash do agent chạy là một job, kèm session sở hữu, giờ bắt đầu, thời gian chạy và kết
quả. Job sống trong server của dự án, nên đóng TUI không làm mất danh sách; server dừng thì
danh sách mất theo (job chết cùng server, không có job "running" giả).

```bash
agent-cli jobs                # đang chạy trước, rồi job vừa xong; mới nhất trước
agent-cli jobs --json         # một mảng JSON: id, session_id, tool_call_id, command, status, exit_code, started_at, finished_at, background, output_bytes
agent-cli jobs output 01A     # output của job (stdout rồi stderr)
agent-cli jobs kill 01A       # dừng job đang chạy; job đã xong thì không làm gì
```

- `STATUS`: `RUNNING`, `SUCCEEDED`, `FAILED (mã thoát)`, `KILLED`. Cột `BG` là job chạy tiếp
  sau khi tool bash đã trả về (agent yêu cầu chạy nền, hoặc lệnh quá lâu nên tự chuyển nền).
- Chỉ liệt kê job trong thư mục của dự án này (một server có thể giữ nhiều workspace).
  Server nhớ 100 job đã xong gần nhất, giữ output của 20 job cuối và của mọi job nền còn trong
  bảng (job đã xong nằm trong bảng tối đa 8 giờ, tối đa 250 job).
- Output dài (dev server, `tail -f`) chỉ giữ 256 KiB đầu và 1 MiB cuối mỗi luồng, phần giữa
  thay bằng dòng `[... N bytes of output dropped ...]`, nên bộ nhớ server có trần.
- Trần 50 job đang chạy cùng lúc cho cả server; job đã xong không tính (bản vá upstream
  `0ac04e4f`: trước đây 50 job nền đã xong là mọi lệnh bash hỏng).
- Dừng bằng `jobs kill` vẫn để job trong bảng: agent gọi `job_output` sau đó thấy output và
  trạng thái đã dừng, không bị "not found".
- Cần server của dự án (launcher đã đặt `CRUSH_CLIENT_SERVER=1`); chế độ in-process mỗi lệnh là
  một tiến trình riêng nên không thấy job của TUI. Server cũ chưa có API job: lỗi nói rõ phải
  cài lại và khởi động lại server.

## Giao việc cho Claude Code / Codex: tool `claude_code`, `codex`

Model chính có thể giao một việc trọn vẹn cho Claude Code hoặc Codex đã đăng nhập sẵn
trên máy (dùng gói thuê bao của bạn, không cần API key). Đây là giao việc **sang provider khác**,
nên mặc định TẮT: cài CLI, đăng nhập hay có key đều không mở được.

### Ranh giới provider/model

Hai biến trong `~/.config/agent-cli/env` (hoặc tệp ở `AGENT_CLI_ENV_FILE`); agent-cli đọc thẳng đúng
hai dòng này (dạng `KEY=value`, có thể `export`; dạng shell khác như `declare -x`, `KEY+=` thì báo
lỗi lúc khởi động thay vì bỏ qua). Export ở shell cũng được và thắng giá trị trong tệp. Các biến này
(cùng `AGENT_CLI_ENV_FILE`, `HOME`, `AGENT_CLI_*_BIN`, `*_PERMISSION_MODE`, `*_SANDBOX`) được chụp ngay
đầu tiến trình, trước khi nạp `.env` của thư mục hiện tại hay `env` trong `crush.json` của dự án, và
server của dự án nhận đúng ảnh chụp đó: repo đang mở không đổi được chúng.

```bash
AGENT_CLI_ALLOWED_MODELS="zai:glm-5.3 zai:glm-5.3-flash main:qwen3.8 claude_code:claude-opus-5-5 claude_code:claude-sonnet-5-5 claude_code:claude-haiku-5-5 codex:gpt-6.1-sol"
AGENT_CLI_DELEGATION=on   # bỏ dòng này (hoặc off) = không giao việc
```

- Danh sách có provider chính (`zai`, `main`...) thì model chính và model nhỏ chỉ được là các cặp
  đó: Alt+M chỉ hiện chúng, cấu hình trỏ tới model khác thì dừng lúc khởi động với
  `MODEL_NOT_ALLOWED` kèm danh sách được phép. Model mới provider vừa thêm bị chặn cho tới khi bạn
  thêm vào (allowlist, không có denylist). Model nhỏ (đặt tiêu đề, `agentic_fetch`) cũng phải nằm trong
  danh sách; agent con và tóm tắt dùng model chính. Model nhỏ ở provider khác được thay bằng model chính trước
  khi kiểm (mục "Cô lập provider, sổ ghi LLM, bằng chứng của loop/routine"). Không đặt danh sách thì model chính
  không giới hạn như trước.
- Tool `claude_code` / `codex` chỉ có khi `AGENT_CLI_DELEGATION=on` VÀ backend có ít nhất một
  model `claude_code:<id>` / `codex:<id>` trong danh sách. Model phải là ID cụ thể: alias như
  `opus`, `latest`, `fable` bị từ chối vì CLI tự đổi chúng sang model mới nhất. Tool luôn truyền
  model (`claude --model=<id>`, `codex -c model="<id>"`), không để CLI tự chọn mặc định (vd Fable)
  và không có model dự phòng; model chính chỉ chọn được trong danh sách (mặc định là model đầu),
  model khác bị từ chối trước khi hỏi quyền, CLI không chạy.
- Tool bash chặn gọi thẳng CLI agent khác (`claude`, `claude-code`, `codex`, `gemini`, `qwen`,
  `opencode`, `aider`, `cursor-agent`, `amp`, `kiro`, kể cả đuôi `.exe`/`.js`): qua đường dẫn (script
  có shebang cũng vậy), `env`/`env -S`/`nohup`/`timeout`/`sudo`/`doas`/`xargs`, `find -exec`,
  `npx`/`bunx`/`npm exec`/`pnpm dlx`, `bash -c "..."`; cùng agent-cli lồng (`agent-cli run|send|server|loop|routine`,
  `jobs`/`ps`/`logs` thì được). Báo `command is not allowed`. `goose` (migration DB) không bị chặn.
  Đây là lớp thứ hai, theo tên: bản chép mang tên khác, `python -c` hay script tự viết rồi chạy không bị
  nhận ra, nhưng vẫn phải qua hộp thoại xin quyền của bash.
- Giới hạn còn lại: ranh giới so theo ID provider. Cấu hình của dự án (`crush.json`) vẫn được đổi
  `base_url` của một provider có sẵn (vd `zai`), như upstream vốn tin cấu hình dự án; mở repo lạ thì
  xem `crush.json`/`.crush.json` của nó trước.
- Hộp thoại giao việc ghi đường dẫn CLI sẽ chạy (vd `Claude Code (/home/u/.local/bin/claude)`).
- Tên model thật lấy từ CLI trên máy: Claude Code nhận ID như `claude-opus-5-5`; danh sách model
  của Codex ở `~/.codex/models_cache.json` (15/10 có `gpt-6.1-sol`, `gpt-6-luna`, `gpt-5.6-luna`,
  `gpt-6-astra`..., không có `gpt-6.1-luna`).
- Đổi biến xong phải để server của dự án khởi động lại (như mọi giá trị trong env).

### Cách hoạt động khi đã bật

- Tool chỉ có mặt khi tìm thấy CLI (`claude`, `codex` trong `PATH`, hoặc đường dẫn ở
  `AGENT_CLI_CLAUDE_BIN` / `AGENT_CLI_CODEX_BIN`) và không bị `permissions deny`. Không có CLI
  thì model không thấy tool và request không tốn token mô tả nó. Biến `*_BIN` đặt mà trỏ sai,
  hay `AGENT_CLI_CLAUDE_PERMISSION_MODE` / `AGENT_CLI_CODEX_SANDBOX` gõ sai giá trị (tool không bị
  deny), thì mọi prompt báo lỗi nêu đúng tên biến và các giá trị hợp lệ cho tới khi sửa và khởi
  động lại server.
- Mỗi lần giao đều hỏi quyền, hộp thoại hiện nguyên văn việc giao, model, thư mục và mức quyền (bằng lời,
  ví dụ "FULL ACCESS: runs any command…", kèm cờ thật). Ký tự điều khiển và ký tự đảo chiều trong
  prompt được thay bằng `�` để hộp thoại không hiện khác việc sẽ chạy.
  Ngoại lệ, đều do bạn chủ động bật: "Allow for Session" (mọi lần giao sau của CLI đó trong
  session), `agent-cli run` (tự duyệt mọi quyền của session nó chạy), `--yolo`,
  `permissions allow claude_code`, hook PreToolUse trả allow.
- `resume` chỉ nhận ID dạng chữ/số/`._:-` (đúng ID tool đã trả về); giá trị giống cờ bị từ chối
  trước khi hỏi quyền, để model không chèn được cờ vào lệnh gọi CLI con.
- Sau khi được duyệt, CLI con chạy không hỏi lại ai: mặc định Claude Code
  `--dangerously-skip-permissions`, Codex sandbox `danger-full-access`. Muốn chặt hơn:
  `AGENT_CLI_CLAUDE_PERMISSION_MODE=acceptEdits` (hoặc `plan`),
  `AGENT_CLI_CODEX_SANDBOX=workspace-write` (hoặc `read-only`).
- Kết quả trả về kèm session/thread ID; model truyền lại vào `resume` để nói tiếp đúng
  cuộc hội thoại đó.
- CLI con chạy trong nhóm tiến trình riêng: huỷ hoặc hết 15 phút thì dừng cả nhóm (kể cả dev
  server, watcher nó bật), không treo theo tiến trình cháu giữ stdout. Nó không nhận các biến
  gắn với một phiên Claude Code khác (`CLAUDECODE`, `CLAUDE_CODE_ENTRYPOINT`,
  `CLAUDE_CODE_SESSION_ID`, socket nhắn tin…), nên mở agent-cli từ trong Claude Code thì CLI con
  vẫn chạy như ở terminal thường; biến cấu hình của bạn (khoá, provider) giữ nguyên.
- CLI con nhận env GỐC của agent-cli: env lúc tiến trình khởi động, trước khi nạp `.env` của
  thư mục hiện tại và `env` trong `crush.json`. Biến lái endpoint, khoá hay thư mục cấu hình của
  CLI con (`ANTHROPIC_BASE_URL`, `ANTHROPIC_API_KEY`, `OPENAI_BASE_URL`, `OPENAI_API_KEY`,
  `CLAUDE_CONFIG_DIR`, `CODEX_HOME`...) đặt trong `.env` của repo, hay `env` của `crush.json` (kể
  cả `crush.json` toàn cục của bạn), không tới CLI con; muốn CLI con dùng chúng thì export ở shell
  (ví dụ `~/.bashrc`). Workspace mà Home mở và server của dự án cũng được dựng với env gốc rồi tự
  nạp `.env` cho riêng chúng, nên mở qua Home hay chế độ server cũng vậy.
- Giới hạn: agent-cli gọi lồng từ một tiến trình đã nạp `.env` (lệnh `!agent-cli run …` trong TUI,
  `agent-cli loop --verify "agent-cli run …"`) chụp env của tiến trình cha, gồm biến thường của
  `.env`, làm env gốc của nó: CLI con mà agent-cli lồng đó giao việc nhận các biến này.
- Server dựng sau khi lệnh đã vào `--cwd` (hiện là `agent-cli loop --cwd <dir>` ở chế độ server)
  nạp `.env` của `<dir>`, không nạp `.env` của thư mục bạn đứng khi gõ lệnh.

Các biến trên phải có trong môi trường của tiến trình agent-cli (export ở shell, ví dụ
`~/.bashrc`), KHÔNG đặt trong `~/.config/agent-cli/env`: tệp đó chỉ được crushrc `source`
trong trình thông dịch riêng. Chúng là biến tin cậy, env của TIẾN TRÌNH server, lấy từ client
đã dựng server đó: mở workspace mới không nạp lại, và client mở sau với giá trị khác bị từ chối
(lỗi nêu tên biến). Đổi xong phải để server cũ tắt (khoảng 60 giây sau khi client cuối thoát, lâu
hơn nếu còn run chạy nền; xem `agent-cli ps`) hoặc dừng nó như mục "Sửa env khi agent-cli đang
chạy", rồi mở lại từ shell đã có biến mới.
Codex đi kèm app ChatGPT desktop thường không nằm trong `PATH` (ví dụ
`/usr/lib/chatgpt/resources/codex`): đặt `AGENT_CLI_CODEX_BIN` hoặc tạo symlink vào `~/.local/bin`.
Không muốn dùng: `permissions deny claude_code codex` trong crushrc.

## Mode Shift+Tab và "accept edits"

Shift+Tab xoay vòng: code → plan → code + YOLO → code. Các bước này đổi plan/YOLO cho cả workspace nên bị chặn khi
có bất kỳ session nào đang chạy: Shift+Tab hiện cảnh báo và giữ nguyên mode.

**Accept edits hiện chỉ bật được qua API** (chế độ client/server), không có trong vòng Shift+Tab:
`POST /v1/workspaces/{id}/permissions/accept-edits?session_id=<id>`, body `{"accept_edits": true}` (`false` để tắt;
`GET` cùng đường dẫn để đọc). Khi nó đang bật, ô nhập TUI hiện `Accepting edits` và Shift+Tab tắt nó: lúc rảnh thì tắt rồi
sang plan, lúc có session đang chạy thì chỉ tắt (về code). Shift+Tab không bật lại.

- Chỉ tự duyệt `edit`, `multiedit`, `write` của **đúng session đó**; mọi tool khác (bash, fetch, MCP,
  `lsp_rename`...) vẫn hỏi.
- Không lan sang session khác, kể cả session con do sub-agent tạo (`<message>$$<tool call>`): sub-agent sửa file vẫn hỏi.
- Chỉ tự duyệt file nằm **trong thư mục dự án** (thư mục làm việc của workspace). Đường dẫn được giải symlink
  (`filepath.Abs` + `EvalSymlinks`); file chưa tồn tại thì xét theo thư mục cha gần nhất đang có. File ngoài dự án,
  symlink trỏ ra ngoài (kể cả symlink treo) hoặc đường dẫn không giải được đều vẫn hỏi.
- Đường dẫn tuyệt đối có thành phần `..` **luôn hỏi**, kể cả khi gộp `..` xong vẫn nằm trong dự án. Lý do: tool ghi
  đúng đường dẫn thô, kernel đi theo symlink trước rồi mới xử lý `..`, nên `<dự án>/link/../x` (với `link` trỏ ra
  ngoài) ghi ra cạnh đích của `link`, dù gộp chữ thì ra `<dự án>/x`. Đường dẫn tương đối thì tool tự gộp `..` với thư
  mục dự án trước (`filepath.Join`) và ghi đúng đường đã gộp, nên được xét theo đường đã gộp (`../x` vẫn hỏi).
- **Luôn hỏi**, kể cả trong dự án: file config của agent-cli (`crush.json`, `.crush.json`, `crushrc`, `.crushrc`, ở
  thư mục nào cũng vậy, kể cả `.crush/crush.json`), vì config khai báo hook và MCP server, tức là chạy được lệnh;
  mọi thứ dưới `.git` (hook, `core.hooksPath`, `core.fsmonitor` cũng chạy lệnh); và mọi file/thư mục tên `HEAD`
  (`x/HEAD` + `x/config` dựng thành một repo bare lồng, lệnh git chỉ-đọc được tự duyệt sẽ đọc config đó). So tên không
  phân biệt hoa thường (macOS không phân biệt), xét trên đường dẫn đã giải symlink.
- Trạng thái nằm trong bộ nhớ server, mất khi server tắt, bị xoá khi xoá session.

Lỗ đã biết (lý do chưa đưa vào Shift+Tab):

- Tên được bảo vệ chỉ xét trên đường dẫn đã giải symlink: nếu một tên được bảo vệ là symlink trỏ tới file thường
  trong dự án (`crushrc` → `x.txt`), ghi `x.txt` vẫn được tự duyệt dù nó đổi config.
- Hardlink: file thường là hardlink của `.git/config` hay của một hook mang tên khác, nên ghi qua nó được tự duyệt.
- Script hook nằm ngoài `.git` (ví dụ `.husky/`, hoặc thư mục `core.hooksPath` trỏ tới) không được bảo vệ: ghi chúng
  được tự duyệt và chúng chạy ở lần `git commit`/`git push` sau.

### Rủi ro còn lại của tự duyệt bash

Lệnh chỉ-đọc trong danh sách an toàn chạy không hỏi, kể cả khi không bật accept edits. Còn các rủi ro sau:

- Git chạy chương trình theo config của repo: `core.fsmonitor`, `remote.<tên>.uploadpack`, `alias.*`,
  `diff.external`... Cần một `.git/config` (hoặc repo bare lồng) đã bị cài sẵn: tool sửa file luôn hỏi khi ghi vào
  `.git` hay `HEAD`, nhưng file đã có sẵn trên đĩa thì không ai kiểm.
- `git ls-remote` và `git remote show <tên>` kết nối mạng tới remote.
- Từ v0.2.0, lệnh chỉ-đọc có đối số trỏ ra ngoài thư mục làm việc (đường dẫn tuyệt đối, `~`, `..`, `$VAR`,
  `--cờ=/đường/dẫn`, `-fPATH`, `git diff --no-index`) hoặc chạy với `working_dir` ngoài workspace thì HỎI quyền;
  `printenv`, `ps` (cờ `e` in env) và mọi `git config` không còn trong danh sách an toàn. Giới hạn còn lại: đường dẫn tương đối đi qua symlink
  trong dự án trỏ ra ngoài không bị nhận ra ở bước này.
- `date -s ...` và `hostname <tên>` đổi giờ/tên máy nếu agent-cli chạy bằng root.

## Loop và Routine

Hai lệnh dùng chung một engine: chạy agent, chạy lệnh verify, sửa tiếp cho tới khi verify thoát 0. "Xong" là mã
thoát của lệnh verify, không phải lời của agent.

**`agent-cli loop` (có người trông):**

```bash
agent-cli loop --verify "go test ./..." "Sửa TestParse đang đỏ"
agent-cli loop --verify "npm test" --max-duration 2h --json "Làm cho endpoint mới qua test"
```

Mỗi lượt là một session mới (prompt gồm đề bài + đuôi output verify vừa đo). Dừng khi `GREEN`, verify xanh từ trước
mọi thay đổi và cây không đổi (`NO_CHANGE`, thoát 1: không có gì chứng minh việc đã làm), hết 2 lượt sửa
(`MAX_REPAIRS`), cùng lỗi và cây không đổi (`STALLED`), quá `--max-duration` (`TIMEOUT`), lượt agent lỗi
(`RUN_ERROR`), Ctrl+C (`CANCELLED`) hoặc verify không chạy được (`UNMEASURED`). Journal JSONL nằm ở
`<data>/loops/` (đường dẫn in ra đầu lượt). Bản thân `loop` không commit hay push; quyền như `run` (tự duyệt tool),
nên model vẫn có thể chạy `git commit`/`git push`. Muốn chặn thì dùng môi trường không có quyền push hoặc dùng routine;
lớp chặn của routine cũng chỉ theo mẫu lệnh, không phải sandbox.
Cờ khác: `--max-repairs` (mặc định 2), `--run-timeout` (45m mỗi lượt agent), `--verify-timeout` (30m), `--name` (tên
journal), `-m`, `--reasoning-effort`. `--json` in `reason`, `detail`, `attempts`, `session_ids`, `verify_status`,
`verify_exit_code`, `journal`.

**`agent-cli routine` (chạy lặp lại, kể cả theo lịch):**

```bash
agent-cli routine new tidy --mode maintenance --repo ~/code/proj \
  --goal "Xoá các test không thể fail" --verify "go test ./..."
agent-cli routine run tidy          # chạy tay một lần
agent-cli routine list
agent-cli routine show tidy         # định nghĩa, sha256, nơi lưu khoá và báo cáo
```

Mỗi lượt chạy làm việc trong worktree riêng (nhánh `routine/tidy-<thời điểm>`, nằm ngoài repo), giữ khoá (lượt thứ
hai thoát 75), chạy với chế độ không người trực (không push/merge/reset --hard/xoá đệ quy ngoài workspace) và ghi
báo cáo ra ngoài worktree. Mã thoát: 0 khi `GREEN`, và khi `NO_CHANGE` ở routine `maintenance` (hôm nay không có việc);
`NO_CHANGE` ở routine `goal` và mọi lý do khác thoát 1. Bạn duyệt: `git -C <repo> diff <base>..routine/tidy-...`, rồi dọn bằng lệnh được in.

Lên lịch: thêm `"schedule": {"on_calendar": "Mon..Fri 09:00", "timezone": "Asia/Ho_Chi_Minh"}` (hoặc `"cron"`) vào
tệp định nghĩa của routine `maintenance`, rồi `agent-cli routine schedule tidy --print`. Lệnh chỉ IN unit systemd
`--user` và/hoặc dòng crontab; bạn tự cài. Lịch ghim sha256 của định nghĩa: sửa tệp xong phải in và cài lại. Muốn
timer chạy khi đã đăng xuất cần `loginctl enable-linger $USER` (chưa kiểm trên máy này). Việc có đích (`goal`) không
lên lịch được. Lịch in ra mang theo biến của launcher (mục kế).

Giới hạn cần biết: lớp chặn không người trực KHÔNG phải ranh giới an ninh. `git` chỉ cho một danh sách trắng
(đọc, add/rm/mv, nhánh mới...; push/merge/reset --hard/alias/`-c` bị chặn), `rm` phải ở trong workspace, tool giao
việc bị tắt, `working_dir` ngoài workspace bị từ chối, tool ghi tệp không ghi ra ngoài thư mục làm việc. Nhưng agent
viết mã rồi chạy `go test`/`make`/verify, và mã đó có thể làm bất cứ gì; chưa phủ `python`/`node`, `find -delete`, ghi
tệp qua redirect, `cd / && rm -rf x` (đủ ở mục kế). Worktree dùng chung refs, config và hooks với repo gốc. Chỉ cho
routine chạy trên repo và máy bạn tin.

## Cô lập provider, sổ ghi LLM, bằng chứng của loop/routine

Hành vi có trong bản sau v0.2.0 (xem `CHANGELOG.md`, mục `[Unreleased]`).

**Lời gọi ẩn ở lại provider của model chính.**

- Lượt chính, agent con, tóm tắt, đặt tiêu đề và `agentic_fetch` đều gọi provider của model chính. Model nhỏ ở
  provider khác (cấu hình, `run --small-model`) bị thay bằng model chính; log ghi một lần (mỗi cặp provider, mỗi
  tiến trình) `Small model is on another provider; using the main model for titles and agentic_fetch`. Cùng provider thì giữ model nhỏ.
- Đặt tiêu đề: session con (agent con, `agentic_fetch`) không gọi LLM đặt tiêu đề; bị cắt vì hết token thì không thử
  lại mà giữ tiêu đề suy ra từ prompt; chỉ thử thêm model chính khi nó khác model nhỏ.
- `agentic_fetch`: `disabled_tools` áp cả cho tool con của nó; tắt cả `web_fetch` lẫn `web_search` thì model không có
  `agentic_fetch`; tắt `web_fetch` thì chế độ URL bị từ chối (chế độ tìm kiếm vẫn được). Tắt `fetch` (tên tool trong
  cấu hình) cũng tính là tắt `web_fetch`. Session con của nó không còn
  tự duyệt: đọc ngoài thư mục tạm thì TUI hỏi. `run` (local cũng như qua server) duyệt mọi yêu cầu của session con
  thuộc chính lượt đó; không tra được session gốc thì không duyệt.
- Còn hở: allowlist chỉ so nhãn; `--small-model` ở provider khác cũng bị thay (mục "Giới hạn đã biết").

**Model đang chọn không dùng được: dừng, không đổi provider.**

- Lúc khởi động hoặc nạp lại cấu hình, provider của model đang chọn không dùng được (provider dựng sẵn thiếu API key,
  provider không còn trong cấu hình hay bị tắt): lựa chọn giữ nguyên, và bước dựng agent dừng với lỗi
  `… provider "<id>", model "<id>" is unavailable and agent-cli does not switch to another provider; …` ở `run`,
  `loop`, `routine` (lỗi chặn trước khi `-m` được áp, nên `-m` không vượt qua được).
- App vẫn khởi động, chỉ chưa có agent (như lúc chưa cấu hình provider nào): `agent-cli login <nền tảng>`, `session`,
  `jobs`… chạy được; TUI mở thẳng màn chọn model, chọn model dùng được thì agent mới được dựng. Lúc TUI đang chạy (nạp
  lại cấu hình) thì chọn model khác bằng `alt+m`.
- Cách gỡ khác: đặt key (export ở shell cho provider dựng sẵn không phải provider chính), hoặc xoá `models` trong
  `<dự án>/.crush/crush.json`.
- Chỉ mất ID model (provider vẫn dùng được): dùng model mặc định của CÙNG provider (theo catalog, không có thì model
  đầu), log ghi `Selected large model is unavailable; using a fallback for this process only (not saved)`, không ghi
  xuống đĩa.
- Kiểm: `scripts/eval/faults.sh <thư-mục-mới> <binary>`, ca `selected-no-key` (cổng thoát 1 khi có request nào lọt,
  khi `run` không dừng, hoặc khi `agent-cli login` không khởi động được).

**Sổ ghi LLM (`AGENT_CLI_LLM_LEDGER`, opt-in).**

- `AGENT_CLI_LLM_LEDGER=<tệp>`: mỗi lời gọi LLM, kể cả mỗi lần thử lại, nối một dòng JSON vào tệp (tạo mode 600):
  `time`, `duration_ms`, `role` (`main`, `title`, `summary`, `subagent`, `agentic_fetch`), `session_id`, `provider`,
  `model` (đúng model đã gọi), `stream`, `messages`, `system_bytes`, `system_sha` (12 ký tự hex), `tools`,
  `tool_names`, `input_tokens`, `output_tokens`, `cache_read_tokens`, `finish`, `error` (chỉ loại lỗi và mã HTTP).
- Không chép nội dung prompt, câu trả lời hay văn bản lỗi.
- Biến tin cậy: đọc từ env gốc chụp đầu tiến trình, nên `.env` của repo không bật hay chuyển được sổ; cũng không đọc từ
  `~/.config/agent-cli/env`. Export ở shell (hoặc đặt trước lệnh) trước khi server của dự án khởi động; server đang
  chạy thì dừng hẳn rồi mở lại.
- Đếm theo provider và vai trò: `jq -r '.provider + " " + .role' <tệp> | sort | uniq -c`.

**`run --json`.**

- Thêm `provider`, `model`: của tin trả lời cuối đã lưu (bỏ qua tin tóm tắt), không phải model "đang chọn". Chưa có
  câu trả lời nào thì không có hai trường này.
- `prompt_tokens`, `completion_tokens` là usage của bước cuối (session mới có thể gồm cả lời gọi đặt tiêu đề), không
  phải tổng. `cost` là tổng cộng dồn của session (với `-s`/`-C` gồm cả lượt trước). Token thật của cả lượt: sổ ghi LLM.
- `text`: qua server (mặc định của launcher) là câu trả lời cuối. Chế độ local (`CRUSH_CLIENT_SERVER=0`) là mọi chữ
  assistant của lượt nối lại (vd câu dẫn trước tool call rồi câu trả lời). Bản tự tóm tắt giữa lượt không được in ở
  cả hai chế độ.
- `provider`/`model` là nhãn ghi trên tin trả lời: có cả khi request lỗi trước khi tới provider (vd provider chết: 0
  request vẫn có `"provider"`). Bằng chứng đã gọi tới provider nào là sổ ghi LLM.

**Loop: verify và cảnh báo khi GREEN.**

- Lệnh verify chạy với `set -o pipefail`: `go test ./... | tee log` thất bại khi `go test` thất bại.
- Hệ quả: đuôi đọc rồi thoát sớm (`| head`, `| grep -q X`) làm lệnh trước nhận SIGPIPE, verify FAIL với mã 141, output
  có dòng gợi ý `exit 141 = SIGPIPE…`. Bỏ đuôi đó (vd ghi output ra tệp rồi `grep -q` trên tệp).
- Verify đã xanh ở baseline (trước mọi thay đổi) VÀ cây làm việc y như lúc baseline: dừng `NO_CHANGE` (`loop` thoát 1),
  `detail` ghi `verify already passed before any change and the working tree did not change: nothing proves the
  agent's work`. Chỉ một trong hai thì vẫn `GREEN`, dòng dừng và `detail` của `--json` ghi cảnh báo tương ứng. Cây
  làm việc tính theo HEAD, diff và tệp chưa theo dõi (không tính tệp bị gitignore); thư mục không phải repo git thì
  không biết cây có đổi không, nên không bao giờ ra `NO_CHANGE`.

**Routine: báo cáo từng lượt.**

- Báo cáo `RUNNING` được ghi ngay khi có thư mục lượt, trước khi tạo worktree, rồi thay bằng kết quả cuối (ghi qua tệp
  tạm rồi đổi tên).
- `routine list`/`show`: lượt mới nhất luôn thắng, không lùi về một lượt GREEN cũ. Lượt chết giữa chừng (kill, OOM, máy
  khởi động lại): `ABANDONED`, worktree và nhánh được giữ. Lượt mới nhất chưa có báo cáo mà khoá đang giữ: `RUNNING`.
  Lỗi trước khi agent chạy (vd không tạo được worktree): `ERROR` kèm nguyên nhân. `show` có thêm `runs` (mọi lượt, mới
  nhất trước).
- Báo cáo có thêm `changed_files` (tệp khác base, gồm tệp chưa theo dõi), `touched_tests` (có tệp test theo quy ước
  Go/JS/TS/Python), `touched_gate` (có script mà lệnh verify gọi, hoặc `Makefile` khi verify bắt đầu bằng `make`),
  `baseline_pass`, `unchanged_tree`. GREEN mà có `touched_tests`/`touched_gate` thì stderr in cảnh báo: duyệt các tệp
  đó trước khi tin kết quả.

**Lịch routine mang biến của launcher.**

- `routine schedule <tên> --print` ghi thêm biến (`Environment=` trong unit systemd, gán biến trên dòng crontab):
  `CRUSH_GLOBAL_CONFIG`, `CRUSH_GLOBAL_DATA`, `CRUSH_CACHE_DIR`, `CRUSH_DISABLE_METRICS`,
  `CRUSH_DISABLE_PROVIDER_AUTO_UPDATE`, `CRUSH_DISABLE_UPDATE_CHECK`, `AGENT_CLI_ENV_FILE`; biến nào rỗng thì bỏ. Trước đây lịch thiếu chúng nên mỗi lần chạy theo lịch tìm định nghĩa ở `~/.config/crush` và thoát 2.
- Giá trị lấy từ env gốc của lệnh (trước khi `.env` của thư mục hiện tại được nạp): chạy lệnh qua launcher `agent-cli`.
  Giá trị có ký tự ngoài `A-Za-z0-9/._:+,-` thì lệnh từ chối, mã 2.
- `AGENT_CLI_ALLOWED_MODELS` KHÔNG được chép vào lịch: lượt theo lịch đọc allowlist từ tệp env (`AGENT_CLI_ENV_FILE`
  hoặc `~/.config/agent-cli/env`), nên siết allowlist trong tệp env có hiệu lực ngay cho lịch đã cài (biến export ở
  lịch sẽ thắng tệp env và che việc siết đó). Allowlist chỉ đặt ở shell thì `schedule --print` in cảnh báo ra stderr:
  chuyển nó vào tệp env.

**Chế độ không người trực: phần thêm.**

- Tool ghi tệp (`write`, `edit`, `multiedit`, `download`, `lsp_replace_symbol`, `lsp_rename`) bị từ chối cứng khi đích
  nằm ngoài thư mục làm việc, trước cả YOLO và `allowed_tools`. Ví dụ: không ghi được `~/.config/agent-cli/crushrc`,
  thứ chạy như Bash ở lần mở sau. Bash (chuyển hướng `>`, `cp`, `tee`) KHÔNG qua kiểm tra này.
- `git -C <dir>`, `--git-dir`, `--work-tree` trỏ ra ngoài workspace (đường dẫn tuyệt đối, `~`, `$VAR`, `..`) bị chặn.
- Option dài viết tắt được so như git: `--har` là `--hard`, `--delet` là `--delete`.
- Luật theo chữ: lệnh nào có chữ nhắc tới biến trỏ git sang repo khác hoặc nạp cấu hình (`GIT_DIR`, `GIT_WORK_TREE`,
  `GIT_COMMON_DIR`, `GIT_INDEX_FILE`, `GIT_OBJECT_DIRECTORY`, `GIT_NAMESPACE`, `GIT_EXEC_PATH`, `GIT_CONFIG_*`) đều bị
  chặn, bất kể lệnh gì và giá trị gì, kể cả trong `sh -c`, `env -S`, `eval`. Biến kế thừa nguyên giá trị từ env của
  chính agent-cli (vd chạy trong git hook) được bỏ qua. Cố ý nghiêng về chặn: biến shell thường trùng tên, hay chỉ
  nhắc tên trong một chuỗi, cũng bị chặn (`env GIT_DIR=/o ls` bị chặn; `MY_GIT_DIR=x git status` vẫn qua).
- Lệnh bị chặn (lớp này hoặc lớp bảo vệ tiến trình agent-cli): output gửi model có thêm dòng `Blocked: <lý do>`
  (tối đa 5 lý do, mỗi lý do 300 ký tự, phần còn lại ghi "and N more blocked commands").
- Vẫn KHÔNG phủ: ghi tệp qua bash, thư mục symlink (`ln -s /khác r; git -C r …`), `cd` ra ngoài trước lệnh, tên biến
  ghép lúc chạy (`"GIT_""DIR"`), script đọc từ tệp hoặc stdin, `python`/`node`, `find -delete`. Đây là danh sách đen,
  không phải sandbox.

## Tổ hợp phím

Xem mọi action và phím hiện tại: `agent-cli keys` (bảng `ACTION  KEYS  HELP`) hoặc `agent-cli keys --json`.
Đổi phím bằng file `keybindings.json` nằm cạnh thư mục `themes/` (mặc định `~/.config/agent-cli/keybindings.json`,
theo `CRUSH_GLOBAL_CONFIG` nếu có). Dạng `{"<action>": ["phím", ...]}`; mảng rỗng là tắt phím đó. Ví dụ cho dev:

```json
{
  "chat.new_session": ["ctrl+n", "alt+n"],
  "chat.jump_session": ["ctrl+1", "ctrl+2", "ctrl+3"],
  "shift_tab": ["ctrl+e"],
  "suspend": []
}
```

- Tên action là đường dẫn snake_case của trường trong `KeyMap`: `Chat.NewSession` -> `chat.new_session`,
  `ShiftTab` (đổi mode) -> `shift_tab`. Action mới được thêm vào `KeyMap` tự có tên, không cần khai báo.
- Action sai tên, hoặc hai action đang bật trong cùng nhóm (`chat.*`, `editor.*`) dùng chung một phím
  (chỉ xét khi có action bị bạn đổi): vẫn áp dụng, nhưng hiện toast cảnh báo lúc khởi động và ghi log.
  JSON hỏng: dùng phím mặc định kèm một cảnh báo.
- `chat.up_down` và `chat.up_down_one_item` chỉ để hiện gợi ý ở thanh help (`agent-cli keys` ghi "display only"); gán phím cho chúng không có tác dụng. Đổi `chat.down`/`chat.up` thay thế.
- Bảng lệnh (command palette) vẫn hiện gợi ý phím mặc định sau khi đổi, vì chuỗi này viết cứng trong `dialog/commands.go`; phím thật vẫn là phím bạn đã đổi.
- File chỉ đọc lúc khởi động; sửa xong phải mở lại TUI.
- Phím chính (thanh footer ở chat): `Alt+W` workspace (`workspaces`, thoát về Home để chọn workspace/session khác, xem mục Home) · `Alt+S` sessions · `Alt+N` session mới (`chat.new_session`) · `Alt+.` / `Alt+,` sang session kế / trước, vòng trong đúng các session panel Sessions đang hiện (bấm nhanh hai lần là tiến hai bước; không còn session khác thì báo "No other session") (`chat.next_session`, `chat.prev_session`) · `Alt+P` provider (`providers`) · `Alt+M` model **của provider đang dùng** (`models`) · `Alt+E` mức effort của model (`effort`, chỉ khi model có các mức); `Ctrl+P` vẫn là Commands (lệnh "Switch Model" ở đó vẫn là danh sách đầy đủ; có thêm "Home" và "Switch Provider", gợi ý phím ghi đúng `alt+n`/`alt+s`/`alt+m`/`alt+p`/`alt+e`/`alt+w`). Phím cũ `Ctrl+S` / `Ctrl+N` / `Ctrl+L` vẫn dùng được.
- `Alt+J` mở **Jobs** (`jobs`; trong `Ctrl+P` là mục "Jobs", gõ `activity` cũng ra): các lệnh bash agent chạy trong dự án, đang chạy trước, kèm thời gian chạy, kết quả (`exit N`, `killed`, `bg` cho job chạy nền) và session sở hữu; tự làm tươi mỗi giây. Enter xem output (bám đuôi như `tail -f`, cuộn lên thì dừng bám, `End` bám lại), Esc quay lại. `Ctrl+K` dừng job: luôn hỏi lại, mặc định **No**, chỉ ←/→ đổi nút rồi Enter. Cùng dữ liệu với lệnh `agent-cli jobs`. Không nằm ở footer (footer giữ bốn phím core); có trong help đầy đủ (`Ctrl+G`).
- `Alt+Tab` không dùng được trong TUI: trình quản lý cửa sổ (Cinnamon, GNOME, KDE...) giữ nó để đổi cửa sổ, terminal không bao giờ nhận. Vì vậy chuyển session là `Alt+.` / `Alt+,`.
- Danh sách provider ở `Alt+P` lấy theo `option visible-provider` trong crushrc (đúng thứ tự), không khai thì là các provider đã cấu hình. Mỗi dòng ghi trạng thái: `current`, `ready`, `agent-cli login openai` (ChatGPT chưa đăng nhập), `needs API key`. Trong bảng model, `Alt+P` quay lại bảng provider.
- Hộp thoại xin quyền mặc định chọn **Deny**: Enter lỡ tay là từ chối. Cho phép một lần là `Ctrl+A`, cho phép cả session là `Ctrl+S` (hoặc chọn nút bằng ←/→ rồi Enter). **Không chữ cái nào cấp quyền**: `a`, `s`, `h`, `l`, Tab... gõ dở rơi vào hộp thoại không làm gì (chỉ `d`/Esc là từ chối), vì hộp thoại chỉ nuốt phím trong tối đa 1,5 giây đầu. Hộp thoại "khởi tạo dự án" mặc định **Nope**; không còn phím `y` tắt: muốn khởi tạo thì chọn Yep bằng ←/→ (hoặc Tab) rồi Enter, hoặc `Ctrl+P` → Initialize Project.
- `chat.jump_session` **không có phím mặc định** (chọn session theo tên, không theo số), nên panel Sessions không
  đánh số dòng. Muốn dùng thì gán trong `keybindings.json` như ví dụ trên; mô tả dưới đây áp khi đã gán.
- `chat.jump_session` (ví dụ gán `alt+1`..`alt+9`) nhảy thẳng tới session đánh số 1..9 trong panel Sessions (đúng thứ tự panel đang hiện), không đụng chữ đang gõ ở editor; số vượt quá danh sách hoặc trúng session đang xem thì không làm gì, session ghim ngoài top 9 không có số. Phím thứ i trong mảng ứng với session thứ i: đổi phím thì liệt kê theo thứ tự (ví dụ trên chỉ nhảy được tới session 1-3, nên panel chỉ đánh số 1-3, các dòng khác hiện `·`).
- Phím `alt+...` cần terminal gửi Alt/Meta đúng (kiểm bằng `agent-cli keys` rồi thử trong TUI).

### Chữ đang gõ không mất khi đổi session

Đổi session (`Alt+S`, `Alt+N`, ...) giữ riêng chữ và file đính kèm chưa gửi của từng session; quay lại là có lại. **Chữ** của từng session được lưu vào `<data dir>/drafts.json` (0600, chỉ bản nháp không rỗng) khoảng 2 giây sau lần gõ cuối, khi thoát TUI (kể cả `Alt+W` về Home) và khi đóng terminal/SSH rớt (SIGHUP), rồi nạp lại lần mở sau; bản nháp của session đã xoá được dọn. Đang duyệt lịch sử prompt (↑) mà đổi session thì bản nháp thật được giữ, không bị thay bằng prompt cũ. **File đính kèm không được lưu xuống đĩa** (mất khi thoát TUI); `kill -9`/mất điện thì mất phần gõ trong ~2 giây cuối.

## Theme (màu giao diện)

Theme mặc định là **`luannt-theme-everforest-dark`** (đổi tên cá nhân ngày 16/10 từ `everforest-dark-v2`; tên cũ vẫn là alias): nền tối dịu cho phiên làm việc dài, chữ thường `#E6E9E7`,
đường dẫn/metadata `#B2B8B6`, gợi ý và phím tắt `#99A09E`. Chỉ có MỘT màu nhấn (xanh ngọc `#93CFD1`), dành cho thứ
đang chọn/tương tác; xanh lá chỉ cho thành công và dòng thêm, đỏ chỉ cho lỗi và dòng xoá, vàng chỉ cho cảnh báo;
sidebar và logo trầm. Không cần cấu hình gì: TUI, Home, hộp thoại, banner khi thoát, màn `login`/`logout` và
help của CLI đều dùng bảng màu này (mọi provider, kể cả hyper).

Theme cũ `charmtone-panther` (và `gruvbox-dark`) không bị sửa và vẫn chọn được: `Ctrl+P` → **Themes** (↑/↓ xem thử,
Enter chọn, Esc quay lại), hoặc đặt trong `~/.config/agent-cli/crush.json`:

```json
{"options": {"tui": {"active_theme": "charmtone-panther"}}}
```

Bỏ trống hoặc xoá `active_theme` thì về `luannt-theme-everforest-dark`. Thứ tự ưu tiên khi nhiều nơi cùng đặt (đã thử):
`crush.json`/`crushrc` trong thư mục dự án thắng `<data dir>/crush.json` (hộp thoại Themes ghi vào đây, mặc định
`~/.local/share/agent-cli/crush.json`), và nó thắng `~/.config/agent-cli/crush.json`. Đã từng chọn theme trong TUI
thì giá trị đó còn trong `<data dir>/crush.json`: chọn lại `luannt-theme-everforest-dark` trong Themes để dùng mặc định mới.

Giới hạn đã biết: đường kẻ bảng markdown do glamour vẽ theo màu mặc định của terminal; màu cú pháp trong khối code
markdown theo theme đầu tiên của tiến trình cho tới khi mở lại (đổi theme giữa chừng thì khối code mới chưa đổi màu).
Đừng sửa `luannt-theme-everforest-dark` bằng trình soạn theme (`Ctrl+E`): bản đã sửa là theme người dùng nên mất các điều chỉnh
theo vai trò; `Ctrl+D` (revert) trả về bản dựng sẵn. Bảng màu nằm ở `internal/ui/styles/everforest_dark_v2.go`
(theme mặc định chọn ở `DefaultThemeName`, `internal/ui/styles/themes.go`); kiểm bằng `go test ./internal/ui/styles/`.

## Giới hạn đã biết

Mã PRB-xx là số trong sổ vấn đề của đợt soát 16/10 (`reports/system-evolution/05_CURRENT_PROBLEMS.md`, chưa vào git).
Giới hạn riêng của từng mục (detach grace, accept edits, theme…) nằm ngay trong mục đó.

- **Model đang chọn không dùng được thì `run`/`loop`/`routine` dừng, kể cả có `-m`.** Đây là cái giá của việc không đổi
  provider (mục "Model đang chọn không dùng được"): chọn model khác trong TUI, sửa key/login, hoặc sửa cấu hình.
- **Model nhỏ ở provider khác luôn bị thay, kể cả khi chỉ định rõ.** `run --small-model <provider khác>/<model>` cũng bị
  thay bằng model chính (log ghi một lần mỗi tiến trình); không có cách cho phép model nhỏ chéo provider.
- **Model là của cả dự án (PRB-05).** Chọn model trong TUI hay `run -m` qua server đều ghi model vào cấu hình của dự án:
  mọi session sau của dự án đổi theo, kể cả session cũ mở lại (chạy tiếp bằng model đang chọn, không bằng model nó
  từng dùng). TUI hiện model đang chọn, không phải model đã trả lời; model thật xem ở `run --json` hoặc sổ ghi LLM.
- **Allowlist chỉ so nhãn (PRB-30).** `provider:model` không ràng buộc `base_url`/`extra_body` (mục "Ranh giới provider/model"):
  ai sửa được cấu hình, kể cả `crush.json` của một repo lạ, thì trỏ được cùng tên sang endpoint khác.
- **Nhãn provider của `run --json` không phải bằng chứng.** `provider`/`model` lấy từ tin đã lưu, có cả khi request
  chưa tới provider. Báo cáo routine không ghi provider/model; cần bằng chứng thì bật sổ ghi LLM (lịch không tự mang
  `AGENT_CLI_LLM_LEDGER`: thêm vào unit/crontab nếu cần).
- **`routine list`/`show` có thể làm mất một lượt theo lịch.** Khi lượt mới nhất chưa có báo cáo hoặc báo cáo còn
  `RUNNING`, hai lệnh này thử khoá của routine rồi nhả ngay (cửa sổ rất ngắn, chưa đo); lượt theo lịch bắt đầu đúng lúc đó thấy khoá
  đang giữ, thoát 75 và không chạy lại.
- **Đăng nhập ChatGPT/Grok có kết nối ra ngoài kể cả khi dùng provider khác.** Đã `agent-cli login openai`/`xai` mà
  chưa có danh sách model của gói (lấy lúc login bị lỗi, hoặc đăng nhập từ bản cũ), thì mỗi lần dựng lại model
  agent-cli thử lấy lại danh sách đó (có thể làm mới token), kể cả trong phiên chỉ dùng Qwen local. Không phải lời gọi
  LLM và không gửi prompt; lấy được một lần thì thôi.
- **Skill viết cho công cụ khác vẫn được liệt kê.** Skill trong các gốc skill (vd skill đồng bộ từ ứng dụng Claude dùng
  computer-use hay trình duyệt) vào system prompt dù agent-cli không có tool mà nó cần. Tắt bằng `option disable-skill
  <tên>`.
- **Mở repo lạ.** `crushrc`/`.crushrc` của repo chạy như Bash, `crush.json` của repo có thể nới quyền tool; không có
  bước hỏi tin cậy. Đọc cấu hình của repo trước khi mở.
- **MCP chết sau App cuối (PRB-84, chế độ server).** App cuối của server đóng thì MCP client bị đóng và không mở lại được
  trong cùng tiến trình; client quay lại trong 60 giây chờ tắt nhận MCP chết. Dừng hẳn server của dự án (mục "Sửa env
  khi agent-cli đang chạy") để có tiến trình mới. Có từ trước; bản sửa job nền theo workspace làm hẹp lại.
- **Job của App mới có thể bị giết (PRB-85).** Dừng job theo gốc workspace: trong lúc App cũ của một dự án đang dừng, App
  mới cùng gốc (mở lại ngay dự án vừa đóng) có thể mất job nền của nó. Cửa sổ hẹp; sửa đúng cần chủ job theo App.
- **Context file (PRB-19, PRB-20).** Chỉ đọc ở thư mục làm việc, không ở gốc git; tên chữ thường (`agents.md`,
  `crush.md`) không được nạp trên Linux; plan mode và agent con không nhận context của dự án.
- **`run` không có trần thời gian cho cả lượt.** Mỗi request LLM chỉ có `request-timeout` (mặc định 2 phút không có
  dữ liệu); script cần trần thì tự bọc `timeout`. `loop`/`routine` có `--run-timeout`.
- **Server bị SIGKILL (PRB-16).** Tiến trình job còn lại không được theo dõi nữa, prompt đang xếp hàng mất không dấu
  vết, TUI không tự dựng lại server.
- **Routine chạy mã cấu hình của repo (PRB-68).** `crushrc`, `$(...)` và hook của repo chạy ngoài lớp chặn không người
  trực: chỉ chạy routine trên repo bạn tin. Worktree và nhánh của các lượt tích luỹ cho tới khi bạn dọn.

## Cập nhật và kiểm chứng

```bash
git pull --ff-only
bash scripts/install-agent-cli.sh   # cuối lần chạy in gợi ý diff -u nếu crushrc/env của bạn khác mẫu mới
agent-cli --version
agent-cli models                    # kiểm cấu hình nạp được, không gọi model
bash scripts/test-agent-cli-wrapper.sh   # tuỳ chọn: kiểm launcher + installer trong thư mục tạm, in PASS
```

(`git diff --check` chỉ dùng để soát khoảng trắng trong thay đổi CỦA BẠN trước khi commit; sau
`git pull --ff-only` cây sạch nên nó luôn rỗng và không kiểm được gì về bản vừa cập nhật.)

Installer sao lưu binary và wrapper đang cài vào `~/.local/lib/agent-cli/backup-<thời điểm>/`
(giữ 3 bản mới nhất), build ra tệp tạm rồi mới thay (build lỗi thì bản cũ còn nguyên), và in
lệnh `cp -p` để quay lại. `agent-cli-tmux` là bản chép nên cũng được thay bằng bản mới ở mỗi lần cài (không
nằm trong bản sao lưu).

Đã kiểm chứng: nạp `crushrc` + env với biến giả (provider `main`, model lớn/nhỏ, key rỗng
và key có giá trị, thiếu biến thì dừng kèm thông báo). Chưa kiểm chứng với endpoint mới (tool
calling và streaming cần thử thật trên model bạn chọn), server nhà, phiên dài/compact, API key
thật của provider, OAuth native (`agent-cli login openai`). Login subscription không đồng nghĩa API key.
