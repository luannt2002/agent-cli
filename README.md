# agent-cli

**agent-cli là nơi bạn làm việc; provider là thứ thực thi.**

Trợ lý lập trình AI chạy trong terminal, bản fork cá nhân của [Crush](https://github.com/charmbracelet/crush) (Charm).
Fork tồn tại để một người làm FE/BE/devops chạy nhiều dự án và nhiều việc song song trong một terminal, với model local
hoặc provider qua API key, mà vẫn biết việc nào đang chạy, chạy bằng model nào, và đã kiểm chứng tới đâu.

Giấy phép FSL-1.1-MIT của upstream giữ nguyên ([LICENSE.md](LICENSE.md)). README gốc: [docs/UPSTREAM-README.md](docs/UPSTREAM-README.md).
Hướng dẫn đầy đủ (cấu hình, server, loop/routine, giới hạn): [docs/agent-cli-local-setup.md](docs/agent-cli-local-setup.md).

## Cài

```bash
git clone https://github.com/luannt2002/agent-cli.git && cd agent-cli
bash scripts/install-agent-cli.sh          # build vào ~/.local, tạo crushrc + env mẫu nếu chưa có
export PATH="$HOME/.local/bin:$PATH"       # thêm dòng này vào ~/.bashrc hoặc ~/.zshrc
agent-cli --version                        # bản vừa cài
```

Cần `git`, Go ≥ 1.27 (theo `go.mod`); `sha1sum`/`shasum` cho launcher; `jq` cho installer và `agent-cli-tmux` (bản
mở agent-cli trong tmux); `tmux` cho `agent-cli-tmux`; `python3` và `script` để chạy các script kiểm. Installer in
sẵn lệnh quay về bản cũ (giữ 3 bản sao lưu). Cập nhật: `git pull --ff-only && bash scripts/install-agent-cli.sh`, rồi
thoát mọi client của các dự án đang mở (xem `agent-cli ps`) để server nền dựng lại bằng bản mới.

**Lần chạy đầu:**

1. Điền provider, model, key vào `~/.config/agent-cli/env` (mẫu và chú thích: `agent-cli.env.example`; ví dụ ở mục
   [Provider](#provider-dùng-nhiều-không-gọi-lén)).
2. Kiểm cấu hình, không gọi model: `agent-cli models` liệt kê model của các provider đã biết; provider của bạn phải
   hiện mà không kèm `(not configured)`.
3. Thử provider trong một thư mục trống (`run` tự duyệt mọi quyền, xem [Khi nào không có hộp thoại
   quyền](#khi-nào-không-có-hộp-thoại-quyền)): `cd "$(mktemp -d)" && agent-cli run "Trả lời đúng một chữ: PONG"`. In
   ra `PONG` là provider chạy được; lỗi thì `run` in lỗi và thoát 1.
4. Mở TUI: `cd <dự án> && agent-cli` → màn Home → `→ Open current folder` (hoặc `agent-cli -c .` để vào thẳng dự án).

## Một ngày làm việc

"Daily engineering workspace" nghĩa là: mở dự án, tiếp việc hôm qua, giao việc, xem nó chạy, kiểm kết quả, mai làm tiếp.
Tất cả trong agent-cli; provider chỉ là nơi model chạy. Các từ workspace, session, run: mục [Khái niệm](#khái-niệm).

1. **Mở:** `agent-cli` trong terminal → màn Home.
2. **Chọn workspace (dự án):** dự án đang chạy (Active), đã ghim, gần đây, hoặc `→ Open current folder`. `agent-cli -c <dir>` vào thẳng.
3. **Chọn session:** session cũ để tiếp, hoặc `→ New session`. Trong TUI: `alt+s` danh sách session, `alt+n` session mới.
4. **Mô tả việc:** gõ yêu cầu, `enter`.
5. **agent-cli chạy theo chính sách:** model gọi tool; sửa tệp, chạy lệnh không nằm trong danh sách an toàn, đọc ngoài
   dự án đều hỏi quyền (mặc định Deny), trừ các trường hợp ở [mục dưới](#khi-nào-không-có-hộp-thoại-quyền). Model chạy
   ở provider đang chọn và agent-cli không tự chuyển provider; muốn khoá danh sách TÊN model (kể cả trước `crush.json`
   của repo lạ) thì đặt allowlist; nó không khoá `base_url` (mục [Provider](#provider-dùng-nhiều-không-gọi-lén)).
6. **Xem tiến độ:** khung chat, To-Do (`ctrl+t`), Jobs (`alt+j`). Từ shell khác: `agent-cli ps`; trong thư mục dự án
   (hoặc kèm `-c <dir>`): `agent-cli jobs`, `agent-cli logs -f`.
7. **Kiểm chứng:** tự chạy lệnh kiểm, hoặc giao bằng `agent-cli loop --verify "<lệnh>" "<việc>"` để "xong" là mã thoát của lệnh đó.
8. **Làm tiếp sau:** đóng TUI không huỷ việc đang chạy: server của dự án giữ nó, tối đa 6 giờ (30 phút nếu nó kẹt chờ
   bạn duyệt quyền hay trả lời). Mở lại `agent-cli` rồi chọn session, hoặc `agent-cli --continue` (session gần nhất),
   `agent-cli --session <id>`.

```bash
cd /path/to/project
agent-cli                                                        # Home → workspace → session
agent-cli --continue                                             # tiếp session gần nhất
agent-cli run "Đọc repo và tóm tắt kiến trúc, chưa sửa code."    # một lượt, không TUI (tự duyệt quyền!)
agent-cli loop --verify "go test ./..." "Sửa TestParse đang đỏ"  # lặp tới khi verify thoát 0, tối đa 2 lượt sửa
```

### Khi nào không có hộp thoại quyền

- `run`, `loop`, `routine` tự duyệt mọi yêu cầu của lượt chúng chạy, kể cả agent con và `agentic_fetch` của lượt;
  `run -s/-C` cũng vậy với session có sẵn. "Chưa sửa code" trong prompt chỉ là lời dặn model, không phải rào chắn. Chúng
  dùng mode hiện tại của dự án: nếu TUI để dự án ở plan mode thì `run`/`loop` chỉ đọc và lập kế hoạch.
- YOLO (`--yolo`, `ctrl+y`, hoặc `shift+tab` tới bước "code + YOLO"): duyệt mọi thứ, áp cho cả workspace (kể cả TUI khác
  và `send` vào dự án đó).
- Nút "Allow for Session" trong hộp thoại; `permissions allow <tool>` trong crushrc.
- Muốn được hỏi: dùng TUI không bật YOLO, hoặc `agent-cli send -s <id> "..."` (gửi prompt vào session đang mở trong
  TUI; không tự duyệt, TUI đó trả lời hộp thoại; không có TUI nào mở session đó thì `send --wait` dừng với lỗi và lượt
  chờ ở server).
- Hộp thoại quyền: `ctrl+a` cho phép một lần · `ctrl+s` cho phép cả session · `enter` (mặc định), `d`, `esc` là từ chối.
  Không phím chữ nào cấp quyền.

**Repo lạ:** mở một repo là chạy `crushrc`/`.crushrc` của nó như Bash; `.env` của thư mục bạn đứng khi chạy agent-cli
được nạp vào môi trường của nó (trừ allowlist và các biến ranh giới, đã chụp từ trước); `crush.json`, `.crush.json` và
`.crush/crush.json` của repo (tệp cuối thắng mọi cấu hình khác, kể cả env) nới được quyền tool, khai hook và MCP server
(tức là chạy lệnh; `$(...)` trong `api_key`/`base_url` chạy ngay lúc nạp), đổi model/provider hay `base_url`; context
file (`AGENTS.md`, `CLAUDE.md`, `CRUSH.md`) và skill trong `.agents/skills`, `.claude/skills`, `.crush/skills`,
`.cursor/skills` của repo vào thẳng prompt. Không có bước hỏi tin cậy: đọc các tệp đó, cả thư mục `.crush/` nếu repo có,
trước khi mở repo không phải của bạn, và đừng `run`/`loop` trên repo chưa đọc.

## Khái niệm

| Từ | Nghĩa trong agent-cli |
|---|---|
| Workspace | Một dự án: gốc git, hoặc thư mục thật nếu không phải repo. Mỗi dự án có server nền riêng, session, job và log riêng. |
| Session | Mạch liên tục của một việc: lịch sử hội thoại và To-Do. Tiếp bằng danh sách trong TUI, `-s <id>` hoặc `-C` (gần nhất). |
| Run | Một lượt: từ prompt tới câu trả lời cuối. Prompt gửi thêm khi đang chạy thì xếp hàng. Huỷ: `esc` trong TUI; Ctrl+C trên `run` chỉ huỷ đúng lượt của nó, prompt xếp hàng phía sau vẫn chạy. |
| Provider | Backend thực thi model: server local tương thích OpenAI, Z.AI, OpenAI, Anthropic… (mục [Provider](#provider-dùng-nhiều-không-gọi-lén)). |
| Model | Một model của một provider, vd `qwen3.8`, `glm-5.3`. Ghi `provider/model` khi chọn bằng `-m`; ghi `provider:model` trong allowlist. Trong TUI: `alt+p` provider, `alt+m` model của provider đó, `alt+e` effort (mức suy luận, nếu model có). Chọn trong TUI là của cả dự án, không riêng một session. |
| Model chính, model nhỏ | Model chính (large) chạy lượt làm việc, tóm tắt và agent con. Model nhỏ (small) đặt tiêu đề session và chạy `agentic_fetch`. **Provider chính** là provider của model chính. |
| Tool | Thứ tác động lên môi trường: `bash`, `edit`, `write`, `view`, `grep`, job… Sửa tệp, chạy lệnh, đọc ngoài dự án đều qua lớp quyền. |
| YOLO | Chế độ duyệt mọi yêu cầu quyền, cho cả workspace. Tắt bằng `ctrl+y`. |
| Skill | Tệp `SKILL.md` tìm thấy trên máy; model đọc khi thấy cần (mục [Skill](#skill)). |
| Agent con | Tool `agent`: một agent chỉ đọc, dùng model chính. Chỉ chạy khi model gọi tool này. |
| `agentic_fetch` | Tool đọc web bằng một agent con chạy model nhỏ. crushrc mẫu tắt nó cùng các tool chạm mạng khác. |
| Loop | `agent-cli loop`: chạy agent rồi lệnh verify, sửa tiếp tới khi verify thoát 0. Bạn khởi chạy và trông. |
| Routine | Một loop đã lưu thành định nghĩa (đích, lệnh verify, repo), chạy trong git worktree riêng, không người trực, lên lịch được. Mode `maintenance`: việc máy móc lặp mãi, được lên lịch; mode `goal`: việc có đích (như một tính năng), chỉ chạy tay. |

## Provider: dùng nhiều, không gọi lén

**Khai provider.** Hai loại, cả hai khai trong `~/.config/agent-cli/env`:

- Provider tự khai, ID `main`: server của bạn qua `AGENT_CLI_BASE_URL` (model local, hoặc endpoint riêng).
- Provider dựng sẵn trong catalog (`zai`, `openai`, `anthropic`…): đặt `AGENT_CLI_PROVIDER_ID`, chỉ cần model + key.

Provider dựng sẵn thứ hai dùng được khi key riêng của nó được export ở shell trước khi mở agent-cli (vd
`ANTHROPIC_API_KEY` trong `~/.bashrc`; tên biến của từng provider: bảng trong [docs/UPSTREAM-README.md](docs/UPSTREAM-README.md)).
Tệp env cấu hình provider chính cùng `AGENT_CLI_ALLOWED_MODELS`/`AGENT_CLI_DELEGATION`; key của provider khác đặt trong
đó không tới tiến trình agent-cli. Hoặc dùng `agent-cli login openai|grok|copilot` (`login` không đối số là Charm
Hyper; zai và anthropic chỉ dùng key). `alt+p` chỉ hiện provider khai bằng `option visible-provider` trong crushrc
(mẫu: provider chính, `openai`, `anthropic`); provider khác thì thêm dòng đó hoặc chọn bằng `-m`.

**Đổi provider.** `alt+p`, hoặc `agent-cli run -s <id> -m <provider>/<model> "..."`. Cả hai đổi model của cả dự án,
không chỉ một lượt. Đổi provider không làm mất mạch: cùng session chạy tiếp với lịch sử cũ (đã thử với LLM thật, một
lần: bước 1 dùng model local, bước 2 `-m` sang provider khác trên cùng session).

**Model của dự án nằm ở đâu.** Chọn model trong TUI, `run -m` (ở chế độ server mặc định; chế độ local chỉ đổi trong lượt
đó), hay mở lại một session cũ đều ghi model vào `<dự án>/.crush/crush.json`; từ đó tệp này thắng `AGENT_CLI_MODEL` trong
env. Muốn dự án theo env trở lại: xoá `models` (và `recent_models`) trong tệp đó. `agent-cli dirs` in các thư mục cấu hình
và dữ liệu đang dùng.

**Không gọi lén:**

- Mọi lời gọi ẩn (đặt tiêu đề, `agentic_fetch`, tóm tắt, agent con) đi provider chính. Model nhỏ nằm ở provider khác thì
  bị thay bằng model chính (kể cả `--small-model` chỉ định rõ); log ghi một cảnh báo.
- Provider lỗi lúc chạy thì lượt báo lỗi; agent-cli không chuyển sang provider khác.
- Model đang chọn không dùng được lúc khởi động (provider dựng sẵn thiếu key, provider không còn trong cấu hình):
  agent-cli cũng không đổi provider. `run`, `loop`, `routine` dừng ngay với lỗi nêu provider/model (`-m` không vượt qua
  được lỗi này); TUI mở thẳng màn chọn model, chọn xong mới dựng agent; `agent-cli login <nền tảng>` vẫn chạy. Cách
  khác: đặt key, hoặc xoá `models` trong `<dự án>/.crush/crush.json` như trên. Chỉ mất ID model mà provider vẫn dùng
  được thì dùng model mặc định của cùng provider, log ghi cảnh báo, không ghi xuống đĩa.
- `AGENT_CLI_ALLOWED_MODELS` (tuỳ chọn, đặt trong env): danh sách `provider:model`. Danh sách có ít nhất một mục không
  phải `claude_code:`/`codex:` thì model chính và model nhỏ phải là một cặp trong danh sách, bất kể lựa chọn đến từ đâu
  (TUI, `-m`, `crush.json` của repo); ngoài danh sách thì dừng với `MODEL_NOT_ALLOWED`.
- Tool `claude_code`/`codex` (giao việc cho Claude Code/Codex) mặc định TẮT. Chỉ có khi `AGENT_CLI_DELEGATION=on`,
  allowlist có model `claude_code:<id>`/`codex:<id>`, và máy có CLI đó. Mỗi lần giao vẫn hỏi quyền, trừ các trường hợp
  ở [mục trên](#khi-nào-không-có-hộp-thoại-quyền). Bash chặn gọi thẳng `claude`, `codex`, `gemini`, `qwen`… theo tên
  (không chặn bản đổi tên hay script; xem giới hạn dưới).

**Giới hạn của lời hứa trên:**

- Lời hứa chỉ phủ lời gọi LLM do chính agent-cli phát ra, không phủ việc model làm qua tool. Chặn `claude`/`codex`… chỉ
  theo tên: bản chép đổi tên, `python -c`, script tự viết hay `curl` tới API của provider khác (bằng CLI đã đăng nhập
  hoặc key có trên máy) không bị nhận ra. Dưới `run`/`loop`/`routine`/YOLO các lệnh đó chạy không hỏi, và sổ ghi LLM
  không ghi chúng. Cần chắc chắn thì chạy việc tự duyệt ở nơi không có CLI đã đăng nhập hay key của provider khác.
- Không đặt allowlist thì cấu hình (kể cả `crush.json` của repo lạ) chọn được model của provider khác mà không hỏi.
  TUI hiện model đang chọn; output thường của `run` thì không (xem bằng `run --json` hoặc sổ ghi LLM).
- Allowlist chỉ so TÊN: nó không kiểm `base_url` thật, nên ai sửa được cấu hình thì trỏ được cùng tên sang endpoint khác.
- Key của provider khác đã export ở shell, hay có trong `.env` của thư mục đang đứng, làm provider đó chọn được.
- Đã đăng nhập ChatGPT/Grok thì agent-cli có thể gọi ra OpenAI/xAI (làm mới token, lấy danh sách model; không phải lời
  gọi LLM, không gửi prompt) kể cả khi chỉ dùng model local.

Ví dụ: provider = model local qua server của bạn, fallback chéo provider = không cho phép.

```bash
# ~/.config/agent-cli/env
AGENT_CLI_PROVIDER_TYPE=openai-compat
AGENT_CLI_BASE_URL="http://127.0.0.1:<port>/v1"   # server local tương thích OpenAI của bạn, không cần key
AGENT_CLI_API_KEY=
AGENT_CLI_MODEL="qwen3.8"                         # ID model mà server của bạn báo
AGENT_CLI_CONTEXT_WINDOW=32768                    # context của server; để mặc định 128000 thì tự tóm tắt có thể không kịp chạy
AGENT_CLI_ALLOWED_MODELS="main:qwen3.8"           # provider tự khai có ID là main
# không đặt AGENT_CLI_DELEGATION: giao việc sang provider khác vẫn tắt
```

Với cấu hình này, lời gọi LLM do agent-cli phát ra tới một provider ID khác `main` là **bug** của agent-cli, không phải
tính năng; kiểm bằng sổ ghi LLM (mục [Quan sát sâu](#quan-sát-sâu)). Sổ ghi ghi ID provider và tên model, không ghi URL:
nó không phát hiện được `base_url` của `main` bị một cấu hình khác (vd của repo lạ) trỏ đi nơi khác.

## "Xong" nghĩa là gì

MODEL NÓI XONG ≠ AGENT-CLI ĐÃ XONG.

- `agent-cli run` thoát 0 nghĩa là lượt chạy hết mà không có lỗi runtime. Không có nghĩa là việc đúng.
- Những trường hợp sau là lỗi (`run` thoát 1): prompt kết thúc mà model chưa ra chữ hay tool call nào (kể cả khi bộ lọc
  nội dung của provider chặn), dừng vì lặp cùng một tool call, context window quá nhỏ ngay sau khi tự tóm tắt mà việc
  còn dở, quá 10 lần tự tóm tắt cho một prompt, Stop hook vẫn chặn sau 3 lần chạy lại, model đang chọn không dùng được,
  lỗi provider.
- "Xong có bằng chứng" là mã thoát của lệnh `--verify` trong `loop`/`routine`. Verify đã xanh từ trước khi sửa VÀ cây
  làm việc không đổi thì không có gì chứng minh việc đã làm: dừng `NO_CHANGE` (`loop` và routine `goal` thoát 1;
  routine `maintenance` thoát 0: hôm nay không có việc). Chỉ một trong hai thì vẫn `GREEN`, kèm cảnh báo. Thư mục
  không phải repo git (hoặc chưa có commit) thì không biết cây có đổi không, nên không ra `NO_CHANGE`.
- `GREEN` chỉ chứng minh lệnh verify thoát 0. Agent tự duyệt mọi quyền nên sửa được chính test hay script mà verify
  chạy: luôn duyệt diff trước khi commit hay merge (routine tạo với `--commit` được phép commit trên nhánh của nó), nhất là tệp test, `Makefile`, script verify (báo cáo routine có cờ
  `touched_tests`/`touched_gate` khi agent đụng chúng).

| Lệnh | Thoát 0 | Thoát 1 | Khác |
|---|---|---|---|
| `run`, `send --wait` | lượt chạy hết, không lỗi runtime (không có nghĩa là đúng) | lỗi | 2: sai cú pháp |
| `send` (không `--wait`) | server đã nhận prompt (lượt có thể chưa chạy xong) | lỗi | 2: sai cú pháp |
| `loop` | `GREEN` | mọi lý do dừng khác | 2: sai cú pháp |
| `routine run` | `GREEN`; `NO_CHANGE` của `maintenance` | lý do khác; `NO_CHANGE` của `goal` | 2: sai cú pháp, hoặc định nghĩa đã đổi so với lịch; 75: lượt khác đang giữ khoá |

## Loop và routine

```bash
agent-cli loop --verify "go test ./..." "Sửa TestParse đang đỏ"
agent-cli routine new vet --mode maintenance --repo ~/code/proj \
  --goal "Sửa mọi cảnh báo của go vet" --verify "go vet ./..."   # thêm --commit nếu cho phép commit
agent-cli routine run vet                                        # chạy tay một lần
agent-cli routine list                                           # lượt cuối của từng routine
```

- `loop`: chạy agent, chạy lệnh verify, sửa tiếp tới khi verify thoát 0 hoặc gặp điều kiện dừng (mặc định 2 lượt sửa).
  Mỗi lượt một session mới. Verify chạy với `pipefail` (`… | head` làm lệnh trước nhận SIGPIPE, thoát 141, tính là FAIL).
  Bản thân `loop` không commit hay push, nhưng agent trong loop được tự duyệt mọi lệnh và loop không có lớp chặn của
  routine: model vẫn chạy được `git commit`/`git push` nếu nó quyết định vậy. Cần chặn thì chạy ở nơi không có quyền
  push, hoặc dùng routine.
- `routine run`: mỗi lần chạy nằm trong git worktree riêng (nhánh `routine/*`), giữ khoá, với chế độ không người trực
  bật: lệnh shell bị chặn theo mẫu (push, merge, `reset --hard`, `rm` ra ngoài worktree…; danh sách chặn, không chặn
  được mọi cách), tool ghi tệp không ghi được ra ngoài thư mục làm việc. Không push, không merge; chỉ commit khi định
  nghĩa tạo với `--commit`. Báo cáo ghi ra ngoài worktree; bạn duyệt diff
  rồi dọn bằng lệnh được in.
- `routine list`/`show` cho thấy lượt đang chạy (`RUNNING`), lượt chết giữa chừng (`ABANDONED`), lượt lỗi sớm (`ERROR`).
- Lịch chỉ được IN ra (`routine schedule <tên> --print`, chỉ cho mode `maintenance`), sau khi bạn thêm khối `schedule`
  vào tệp định nghĩa (`~/.config/agent-cli/routines/<tên>.json`, đường dẫn in ra lúc `routine new`; ví dụ trong guide); bạn
  tự cài timer/crontab. Lịch ghim sha256
  của định nghĩa: sửa định nghĩa thì in và cài lại. Allowlist đặt trong tệp env: lịch không chép allowlist
  từ shell.
- Chế độ không người trực không phải sandbox: chỉ chạy routine trên repo và máy bạn tin. Chi tiết và ví dụ:
  [Loop và Routine](docs/agent-cli-local-setup.md#loop-và-routine).

## Memory: không có kho riêng

agent-cli không có kho memory riêng. Nó nhớ bằng ba thứ:

- Context file: `AGENTS.md`, `CRUSH.md`, `CLAUDE.md`… ở thư mục bạn mở agent-cli (không phải gốc git), và
  `~/.config/agent-cli/CRUSH.md` cho mọi dự án. Agent con và plan mode không nhận context file của dự án.
- Lịch sử session (tiếp bằng `-s`/`-C` hoặc danh sách session).
- Tự tóm tắt khi context gần đầy.

Thứ tự nên tin: mã nguồn/runtime hiện tại > bằng chứng vừa kiểm > lịch sử > suy luận của model. Đây là thứ tự bạn và
model nên áp khi lịch sử hay context file nói khác mã hiện tại; agent-cli không ép bằng máy. Đã thử với LLM thật, một
lần: `AGENTS.md` chứa khẳng định cũ sai và một chỉ thị độc, model vẫn làm theo mã hiện tại. Một lần chạy không phải bảo
đảm chống chỉ thị độc.

## Skill

- Skill là tệp `SKILL.md` mà agent-cli tìm thấy: `~/.agents/skills`, `~/.claude/skills`, `.agents/skills` và
  `.claude/skills` của dự án…, cộng 3 skill dựng sẵn. Thư mục ẩn bên dưới các gốc đó (vd `.trash`) không được quét.
- Tên và mô tả skill vào system prompt; model tự quyết có đọc skill hay không. Tắt một skill: `option disable-skill <tên>`
  trong crushrc (`options.disabled_skills` trong `crush.json`).
- Luật phải luôn áp dụng (đòi bằng chứng, không tự commit…) đặt trong context file (`~/.config/agent-cli/CRUSH.md`,
  `AGENTS.md`/`CLAUDE.md` của dự án), không đặt trong skill: trong log 3 ngày dùng thật của tác giả (41 lượt), model
  chỉ nạp skill ở 2 lượt.
- Ponytail, Superpowers là bộ skill tuỳ chọn. Runtime agent-cli không phụ thuộc chúng.

## Cấu hình

| Việc | Ở đâu |
|---|---|
| Provider, model, key, allowlist | biến `AGENT_CLI_*` trong `~/.config/agent-cli/env` (mode 600, mẫu: `agent-cli.env.example`); key ghi thẳng thì đặt trong nháy đơn (`'sk-…'`), hoặc `"$(pass show <entry> \| head -n1)"` |
| Model đã chọn cho một dự án | `<dự án>/.crush/crush.json` (thắng env; mục [Provider](#provider-dùng-nhiều-không-gọi-lén)) |
| Cấu hình chính, quyền công cụ | `~/.config/agent-cli/crushrc` (mẫu: `agent-cli.crushrc.example`) |
| Màu giao diện | `~/.config/agent-cli/themes/*.json`, chọn bằng `options.tui.active_theme` |
| Dữ liệu phiên | `~/.local/share/agent-cli`, và `.crush/` trong từng dự án (tự có `.gitignore` bỏ qua mọi thứ bên trong) |
| Log | `agent-cli logs` (đọc đúng tệp log của dự án); `agent-cli dirs` in các thư mục đang dùng |

Tên tệp và biến bên trong (`crush.json`, `CRUSH_*`, `.crush/`) giữ chữ Crush để dễ gộp upstream. Launcher `agent-cli`
(wrapper ở `~/.local/bin`) đặt sẵn các biến đó, tắt metrics, tắt cập nhật catalog provider và tắt kiểm tra bản mới của
upstream. Sửa env khi server của dự án đang chạy thì chưa có tác dụng: thoát mọi client rồi mở lại (xem
[Sửa env khi agent-cli đang chạy](docs/agent-cli-local-setup.md#sửa-env-khi-agent-cli-đang-chạy)).

## Phím tắt

Phím hiện dưới chân màn hình; `ctrl+g` xem thêm; `agent-cli keys` in bảng phím của màn chat (không gồm Home và hộp thoại).

**Home:** gõ để tìm · `↑↓` chọn · `enter` mở · `ctrl+p` ghim/bỏ ghim · `ctrl+x` bỏ dự án đã mất khỏi danh sách · `esc` xoá ô
tìm, `esc` lần nữa thoát.

| Phím (màn chat) | Việc |
|---|---|
| `enter` · `ctrl+j` · `ctrl+o` | gửi · xuống dòng · mở editor ngoài |
| `@` · `/` hoặc `ctrl+p` | chèn file vào ngữ cảnh · bảng lệnh |
| `ctrl+f` · `ctrl+v` | đính kèm ảnh · dán ảnh |
| `tab` · `shift+tab` · `ctrl+y` | đổi focus · xoay mode (code → plan → code + YOLO) · bật/tắt YOLO |
| `alt+n` · `f2` · `alt+s` · `alt+.`/`alt+,` | session mới · đổi tên session · danh sách session · session kế/trước |
| `alt+p` · `alt+m` · `alt+e` | provider · model của provider đó · effort |
| `alt+w` · `alt+j` | về Home · bảng Jobs (bash nền) |
| `ctrl+b` · `ctrl+t` · `ctrl+d` | sidebar · To-Do · chi tiết |
| `esc` · `ctrl+c` · `ctrl+z` | huỷ lượt đang chạy · thoát (ở chế độ server mặc định, lượt đang chạy vẫn chạy tiếp: muốn dừng thì `esc` trước) · treo |

Tên đặt bằng `f2` không bị tiêu đề do model sinh ghi đè. Terminal chiếm `f2` thì dùng `alt+s` rồi `ctrl+r`.

## Quan sát sâu

- `agent-cli ps [--json]`: mọi session của các dự án có server đang chạy, trạng thái `BUSY` (đang chạy) / `WAITING`
  (chờ bạn duyệt hay trả lời) / `IDLE`. Mặc định 9 session mới nhất mỗi dự án; `--all` bỏ trần đó. Không khởi động server nào.
- `agent-cli jobs`, `jobs output <id>`, `jobs kill <id>`: lệnh bash agent đã chạy, kèm session sở hữu và mã thoát.
- `agent-cli logs -f` (bám đuôi), `-t <N>` (N dòng cuối).
- `agent-cli session list|show|last --json`: session dạng máy đọc.
- `agent-cli run --json`: `session_id`, `text` (qua server: câu trả lời cuối; chế độ local `CRUSH_CLIENT_SERVER=0`: mọi
  chữ assistant của lượt nối lại, kể cả câu dẫn trước tool call; bản tự tóm tắt không có ở cả hai), `error`, `provider`/`model` (ghi trên tin trả
  lời cuối đã lưu: model agent-cli đã định gọi, có cả khi request lỗi; bằng chứng provider ID nào đã được gọi là sổ ghi
  LLM),
  `prompt_tokens` và `completion_tokens` (của bước cuối, không phải tổng), `cost` (cộng dồn của session).
- Sổ ghi LLM (opt-in): `AGENT_CLI_LLM_LEDGER=<tệp>` ghi mỗi lời gọi LLM một dòng JSON: vai trò, provider, model, kích
  thước, token. Không chép nội dung prompt. Biến này KHÔNG đọc từ `~/.config/agent-cli/env` và lịch routine không mang
  theo: export ở shell trước khi server của dự án khởi động.

## Giới hạn còn lại

Danh sách ở [docs/agent-cli-local-setup.md, mục "Giới hạn đã biết"](docs/agent-cli-local-setup.md#giới-hạn-đã-biết).
Lớn nhất: mở repo lạ là chạy cấu hình của nó (mục [trên](#khi-nào-không-có-hộp-thoại-quyền)); model chọn trong TUI hay
`run -m` là của cả dự án, không riêng một session; allowlist chỉ so tên provider/model; context file chỉ nạp ở thư mục
mở agent-cli; `run` không có trần thời gian riêng (bọc `timeout` nếu cần); chế độ không người trực là danh sách chặn,
không phải sandbox.

## Khác upstream

- Tên và giao diện agent-cli; không gọi GitHub kiểm bản mới (chỉ bật khi chạy thẳng binary với `CRUSH_DISABLE_UPDATE_CHECK=false`).
- Crushrc mẫu tắt dòng `Assisted-by: Crush` / `Generated with Crush` trong commit và PR
  ([cách bật lại](docs/agent-cli-local-setup.md#attribution-trong-commit-và-pr)).
- Nhiều session/dự án trong một terminal, run sống khi đóng TUI, `loop`/`routine`, ranh giới provider, hộp thoại quyền mặc định Deny.
- Chưa đổi: tên module Go, system prompt (còn tự xưng "Crush"), skill và tool nội bộ `crush-config`, `crush_info`.

## Phát triển

```bash
go build .
GOMAXPROCS=2 go test ./... -p 2     # máy dùng chung: giới hạn CPU
```

Quy ước code và kiến trúc: [AGENTS.md](AGENTS.md).
