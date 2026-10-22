# DEW — Daily Engineering Workspace

**One place for engineering.** DEW là nơi việc engineering nằm; provider là thứ thực thi việc đó.
Trước đây tên **agent-cli** (đổi 20/10/2026, mục [Tên cũ](#tên-cũ-agent-cli)); fork của Crush ([Nguồn gốc](#nguồn-gốc)).

## DEW là gì

Một chỗ để làm toàn bộ việc engineering hằng ngày: mở dự án, tiếp việc hôm qua, giao việc cho model, xem nó chạy, kiểm
kết quả, mai làm tiếp. Dựng cho một người làm FE/BE/devops chạy nhiều dự án và nhiều việc song song trong một terminal,
với model local hoặc provider qua API key, mà vẫn biết việc nào đang chạy, chạy bằng model nào, đã kiểm chứng tới đâu.

- **Provider chỉ là backend thực thi.** Session, lịch sử, job, quyền, bằng chứng nằm ở DEW. Đổi provider giữa chừng không
  mất mạch; DEW không tự chuyển provider ([Provider](#provider-dùng-nhiều-không-gọi-lén)).
- **CLI chỉ là một giao diện.** Hiện có TUI (`dew`) và lệnh cho script (`run`, `send`, `loop`, `routine`, `ps`…).
- **Vì sao đổi tên:** "agent-cli" là tên chung chung và trói sản phẩm vào một giao diện.
  Định danh chốt ở §41 [spec V3](reports/v3-research/spec/DEW_RECONSTRUCTION_WORKFLOW.md): tagline "One place for
  engineering", North Star "DEW is where engineering work lives. Providers are how the work gets done."
- **DEW không phải:** kho trí nhớ, RAG, knowledge store hay graph (nó nhớ bằng context file, lịch sử session, tự tóm
  tắt: [Memory](#memory-không-có-kho-riêng)); LLM chạy nền (observer, worker, hàng đợi tự gọi model để ghi nhớ sau lưng
  bạn); fallback chéo provider; sandbox. Các lớp trí nhớ học từ Context Mode, claude-mem, Hindsight bị hoãn hoặc bác vì
  chưa lỗi đo được nào đòi chúng ([V3_INTEGRATION_DECISION.md](reports/v3-research/V3_INTEGRATION_DECISION.md) §1, §7).

## Tên cũ: agent-cli

Đổi 20/10/2026: `agent-cli`/`agent-cli-tmux` → `dew`/`dew-tmux`; `~/.config|.local/share|.cache|.local/lib/agent-cli` →
`…/dew`; `AGENT_CLI_*` → `DEW_*`. Không có alias, dew không đọc song song tên cũ. Repo `luannt2002/dew` (trước là
`luannt2002/agent-cli`, đổi trên GitHub 20/10; URL cũ được chuyển hướng). Máy đang dùng agent-cli:

1. Thoát mọi client `agent-cli` (TUI, `run`, `loop`), chờ server cũ tự tắt (khoảng một phút); còn tiến trình chạy binary
   trong `~/.local/lib/agent-cli/` thì trình cài dừng, chưa đổi gì.
2. `git pull --ff-only && bash scripts/install-dew.sh`. Build chạy trước; rồi chuyển bốn thư mục `…/agent-cli` sang
   `…/dew` (cả hai cùng có thì không gộp, cảnh báo), đổi `AGENT_CLI_` → `DEW_` và `…/agent-cli` → `…/dew` trong `env`,
   `crushrc`, `crush.json`, `projects.json` (sao lưu ở `~/.config/dew/pre-dew-<thời điểm>/`), nối lại worktree routine,
   gỡ lệnh cũ do trình cài cũ tạo. Lệnh quay lại ghi dần vào `pre-dew-<thời điểm>/undo.sh`, in cả khi dừng giữa chừng.
3. Biến `AGENT_CLI_*` bạn tự export (`~/.bashrc`…) phải đổi tay; trình cài liệt kê tên biến và tệp khởi động shell còn
   nhắc tên cũ. Còn biến tin cậy tên cũ (`AGENT_CLI_ALLOWED_MODELS`…) hoặc `~/.config/agent-cli/env` (không đặt
   `DEW_ENV_FILE`, và không phải cùng tệp qua symlink) thì dew từ chối chạy, lỗi nêu cả hai tên.
4. Timer/crontab routine cài theo tên cũ: in lại bằng `dew routine schedule <tên> --print` rồi cài tay.

Đổi remote bản clone cũ: [docs/dew-sync.md](docs/dew-sync.md) mục 3. Phần cố ý giữ tên cũ:
[RENAME_PLAN.md](reports/documentation/RENAME_PLAN.md); chi tiết: [CHANGELOG.md](CHANGELOG.md) mục Unreleased.

## Nguồn gốc

Fork cá nhân của [Crush](https://github.com/charmbracelet/crush), coding agent TUI viết bằng Go của Charm. Giấy phép giữ
nguyên: FSL-1.1-MIT của Charmbracelet kèm thông báo MIT của Kujtim Hoxha ([LICENSE.md](LICENSE.md)); README gốc:
[docs/UPSTREAM-README.md](docs/UPSTREAM-README.md). Để gộp upstream rẻ, bên trong vẫn giữ chữ Crush: module Go
`github.com/charmbracelet/crush`, `crush.json`, `.crush/`, biến `CRUSH_*` ([Khác upstream](#khác-upstream)). Đã phát hành
tag `v0.1.0`, `v0.2.0`; `loop`/`routine` và các sửa lỗi V3 (gộp `main` 20/10) chưa có tag ([CHANGELOG](CHANGELOG.md)).

## Cài

```bash
git clone https://github.com/luannt2002/dew.git && cd dew
bash scripts/install-dew.sh          # build vào ~/.local, tạo crushrc + env mẫu nếu chưa có
export PATH="$HOME/.local/bin:$PATH" # thêm dòng này vào ~/.bashrc hoặc ~/.zshrc
dew --version                        # bản vừa cài
```

Cần `git`, Go ≥ 1.27 (theo `go.mod`); `sha1sum`/`shasum` cho launcher; `jq` cho installer và `dew-tmux` (bản mở dew
trong tmux); `tmux` cho `dew-tmux`; `python3` và `script` để chạy các script kiểm. Installer in sẵn lệnh quay về bản cũ
(giữ 3 bản sao lưu). Cập nhật: `git pull --ff-only && bash scripts/install-dew.sh`, rồi thoát mọi client của các dự án
đang mở (xem `dew ps`) để server nền dựng lại bằng bản mới; trước đó, lệnh chạy model (TUI, `run`, `send`, `loop`) của
dự án có server bản cũ đang bận bị từ chối.

**Lần chạy đầu:**

1. Điền provider, model, key vào `~/.config/dew/env` (mẫu và chú thích: `dew.env.example`; ví dụ ở
   [Provider](#provider-dùng-nhiều-không-gọi-lén)).
2. Kiểm cấu hình, không gọi model: `dew models`; provider của bạn phải hiện mà không kèm `(not configured)`.
3. Thử provider trong thư mục trống (`run` tự duyệt mọi quyền, [xem dưới](#khi-nào-không-có-hộp-thoại-quyền)):
   `cd "$(mktemp -d)" && dew run "Trả lời đúng một chữ: PONG"`. In `PONG` là chạy được; lỗi thì `run` in lỗi, thoát 1.
4. Mở TUI: `cd <dự án> && dew` → màn Home → `→ Open current folder` (hoặc `dew -c .` để vào thẳng dự án).

## Một ngày làm việc

1. **Mở:** `dew` → màn Home. **Chọn workspace (dự án):** đang chạy (Active), đã ghim, gần đây, hoặc
   `→ Open current folder`; `dew -c <dir>` vào thẳng.
2. **Chọn session:** session cũ để tiếp, hoặc `→ New session`. Trong TUI: `alt+s` danh sách session, `alt+n` session mới.
3. **Mô tả việc** rồi `enter`. Model gọi tool; sửa tệp, chạy lệnh không nằm trong danh sách an toàn, đọc ngoài dự án đều
   hỏi quyền (mặc định Deny), trừ các trường hợp ở [mục dưới](#khi-nào-không-có-hộp-thoại-quyền). Model chạy ở provider
   đang chọn; muốn khoá danh sách TÊN model (kể cả trước `crush.json` của repo lạ) thì đặt allowlist (không khoá
   `base_url`).
4. **Xem tiến độ:** khung chat, To-Do (`ctrl+t`), Jobs (`alt+j`). Từ shell khác: `dew ps`; trong thư mục dự án (hoặc
   kèm `-c <dir>`): `dew jobs`, `dew logs -f`.
5. **Kiểm chứng:** tự chạy lệnh kiểm, hoặc giao bằng `dew loop --verify "<lệnh>" "<việc>"` để "xong" là mã thoát của lệnh.
6. **Làm tiếp sau:** đóng TUI không huỷ việc đang chạy: server của dự án giữ nó, tối đa 6 giờ (30 phút nếu kẹt chờ bạn
   duyệt quyền hay trả lời). Mở lại `dew` rồi chọn session, hoặc `dew --continue` (gần nhất), `dew --session <id>`.

### Khi nào không có hộp thoại quyền

- `run`, `loop`, `routine` tự duyệt mọi yêu cầu của lượt chúng chạy, kể cả agent con và `agentic_fetch`; `run -s/-C`
  cũng vậy với session có sẵn. "Chưa sửa code" trong prompt chỉ là lời dặn model, không phải rào chắn. Chúng dùng mode
  hiện tại của dự án: TUI để dự án ở plan mode thì `run`/`loop` chỉ đọc và lập kế hoạch.
- YOLO (`--yolo`, `ctrl+y`, hoặc `shift+tab` tới "code + YOLO") duyệt mọi thứ, cho cả workspace (kể cả TUI khác và
  `send` vào dự án đó). `--yolo` (TUI, `run`, `send`, `loop`) phải khớp workspace đang mở, cả hai chiều; lỗi nói chiều
  ("pass --yolo too" / "drop --yolo"); `send --yolo` vào dự án chưa mở dựng workspace YOLO. Home và `dew-tmux` mở không
  kèm `--yolo` nên bị từ chối khi dự án đang mở bằng `--yolo` (thoát client đó hoặc mở lại không `--yolo`). `jobs`,
  `login`, `logout` không bị so; bật YOLO lúc chạy (`ctrl+y`/`shift+tab`) cũng không (client sau vào được, thành YOLO).
- Nút "Allow for Session" trong hộp thoại; `permissions allow <tool>` trong crushrc.
- Muốn được hỏi: TUI không bật YOLO, hoặc `dew send -s <id> "..."` (gửi vào session đang mở trong TUI; không tự duyệt,
  TUI đó trả lời hộp thoại; không TUI nào mở session đó thì `send --wait` dừng với lỗi và lượt chờ ở server).
- Hộp thoại: `ctrl+a` cho phép một lần · `ctrl+s` cả session · `enter` (mặc định), `d`, `esc` từ chối. Không phím chữ
  nào cấp quyền.

**Repo lạ:** mở một repo là chạy `crushrc`/`.crushrc` của nó như Bash. `.env` của thư mục bạn đứng khi chạy dew được nạp
vào môi trường (trừ allowlist và các biến ranh giới, đã chụp từ trước): lệnh `!` bạn gõ và tiến trình dew con không nhận
chúng từ `.env`; CLI con giao việc Claude Code/Codex chỉ nhận env gốc, không nhận biến nào của `.env`, kể cả
`ANTHROPIC_BASE_URL`, `OPENAI_API_KEY`, `CLAUDE_CONFIG_DIR`, `CODEX_HOME`. Giới hạn: dew gọi lồng từ tiến trình đã nạp
`.env` (`!dew run …`, `--verify "dew run …"`) chụp biến thường của `.env` làm env gốc, nên CLI con của nó nhận chúng khi
lời gọi lồng dựng server mới hoặc chạy in-process (dự án đã có server thì theo env gốc của server đó). `crush.json`,
`.crush.json`, `.crush/crush.json` của repo (tệp cuối thắng mọi cấu hình khác, kể cả env) nới được quyền tool, khai hook
và MCP server (tức là chạy lệnh; `$(...)` trong `api_key`/`base_url` chạy ngay lúc nạp), đổi model/provider hay
`base_url`; context file (`AGENTS.md`, `CLAUDE.md`, `CRUSH.md`) và skill trong `.agents/skills`, `.claude/skills`,
`.crush/skills`, `.cursor/skills` vào thẳng prompt. Không có bước hỏi tin cậy: đọc các tệp đó, cả thư mục `.crush/`,
trước khi mở repo không phải của bạn, và đừng `run`/`loop` trên repo chưa đọc.

## Khái niệm

- **Workspace:** một dự án: gốc git (vật lý), hoặc thư mục thật nếu không phải repo; mỗi dự án có server nền, session,
  job, log riêng. Mở từ thư mục con hay symlink thì làm việc ở GỐC repo (agent, `!lệnh`, tool, đường `@tệp`, cả tiến
  trình): một workspace và một `crush.db` (`<gốc>/.crush`) mỗi repo, ở mọi chế độ; cấu hình, context file, skill của
  thư mục con không được nạp. `.crush` cũ ở thư mục con không tự gộp: dew in cảnh báo kèm lệnh mở lại (chỉ nên đọc).
- **Session:** mạch liên tục của một việc: lịch sử hội thoại và To-Do. Tiếp bằng danh sách trong TUI, `-s <id>`, `-C`.
- **Run:** một lượt, từ prompt tới câu trả lời cuối. Prompt gửi thêm khi đang chạy thì xếp hàng. Huỷ: `esc` trong TUI;
  Ctrl+C trên `run` chỉ huỷ lượt của nó, prompt xếp hàng phía sau vẫn chạy.
- **Provider:** backend thực thi model: server local tương thích OpenAI, Z.AI, OpenAI, Anthropic…
- **Model:** một model của một provider, vd `qwen3.8`, `glm-5.3`; ghi `provider/model` khi chọn bằng `-m`,
  `provider:model` trong allowlist. TUI: `alt+p` provider, `alt+m` model của provider đó, `alt+e` effort (mức suy luận,
  nếu model có). Chọn trong TUI là của cả dự án, không riêng session.
- **Model chính, model nhỏ:** model chính (large) chạy lượt làm việc, tóm tắt và agent con; model nhỏ (small) đặt tiêu
  đề session và chạy `agentic_fetch`. **Provider chính** là provider của model chính.
- **Tool:** `bash`, `edit`, `write`, `view`, `grep`, job, MCP…; sửa tệp, chạy lệnh, đọc ngoài dự án đều qua lớp quyền.
  Kết quả chữ của bash và MCP có trần, phần thừa spill ra `<data>/shell-output/`. **YOLO:** duyệt mọi yêu cầu quyền cho
  cả workspace, tắt bằng `ctrl+y`. **Skill:** tệp `SKILL.md` model đọc khi thấy cần ([Skill](#skill)).
- **Agent con:** tool `agent`, agent chỉ đọc dùng model chính, chỉ chạy khi model gọi. **`agentic_fetch`:** tool đọc web
  bằng agent con chạy model nhỏ; crushrc mẫu tắt nó cùng các tool chạm mạng khác.
- **Loop:** `dew loop` chạy agent rồi lệnh verify, sửa tiếp tới khi verify thoát 0; bạn khởi chạy và trông.
  **Routine:** loop đã lưu thành định nghĩa (đích, lệnh verify, repo), chạy trong git worktree riêng, không người trực.
  Mode `maintenance`: việc máy móc lặp mãi, lên lịch được; mode `goal`: việc có đích, chỉ chạy tay.

## Provider: dùng nhiều, không gọi lén

**Khai provider** trong `~/.config/dew/env`: provider tự khai ID `main` (server của bạn qua `DEW_BASE_URL`), hoặc
provider dựng sẵn trong catalog (`zai`, `openai`, `anthropic`…: đặt `DEW_PROVIDER_ID`, chỉ cần model + key). Tệp env cấu
hình provider chính cùng `DEW_ALLOWED_MODELS`/`DEW_DELEGATION`; key provider khác đặt trong đó không tới tiến trình dew.
Provider dựng sẵn thứ hai dùng được khi key của nó được export ở shell trước khi mở dew (vd `ANTHROPIC_API_KEY` trong
`~/.bashrc`; tên biến: [docs/UPSTREAM-README.md](docs/UPSTREAM-README.md)), hoặc qua `dew login openai|grok|copilot`
(`login` không đối số là Charm Hyper; zai và anthropic chỉ dùng key). `alt+p` chỉ hiện provider khai bằng
`option visible-provider` trong crushrc (mẫu: provider chính, `openai`, `anthropic`); khác thì thêm dòng đó hoặc `-m`.

**Đổi provider:** `alt+p`, hoặc `dew run -s <id> -m <provider>/<model> "..."`. Cả hai đổi model của cả dự án (`run -m`:
ở chế độ server mặc định; chế độ local chỉ đổi trong lượt đó). Cùng session chạy tiếp với lịch sử cũ (đã thử với LLM
thật, một lần: bước 1 model local, bước 2 `-m` sang provider khác trên cùng session).

**Model của dự án** ghi ở `<dự án>/.crush/crush.json` khi chọn trong TUI hay `run -m` (chế độ server); từ đó tệp này
thắng `DEW_MODEL`. Mở lại hay tiếp session cũ (TUI, `run -s/-C`) không ghi gì: lượt mới chạy model đang chọn của dự án,
kể cả khi session từng chạy model khác; khi đó TUI báo ở thanh trạng thái, `run` in một dòng ra stderr nêu cả hai model.
Muốn dự án theo env trở lại: xoá `models` (và `recent_models`) trong tệp đó. `dew dirs` in thư mục cấu hình và dữ liệu.

**Không gọi lén:**

- Mọi lời gọi ẩn (đặt tiêu đề, `agentic_fetch`, tóm tắt, agent con) đi provider chính. Model nhỏ ở provider khác bị thay
  bằng model chính (kể cả `--small-model` chỉ định rõ); log ghi cảnh báo.
- Provider lỗi lúc chạy thì lượt báo lỗi; dew không chuyển sang provider khác.
- Model đang chọn không dùng được lúc khởi động (provider dựng sẵn thiếu key, provider không còn trong cấu hình): `run`,
  `loop`, `routine` dừng ngay, lỗi nêu provider/model (`-m` không vượt qua được); TUI mở thẳng màn chọn model;
  `dew login <nền tảng>` vẫn chạy. Sửa: đặt key, hoặc xoá `models` trong `<dự án>/.crush/crush.json`. Chỉ mất ID model
  mà provider vẫn dùng được thì dùng model mặc định của cùng provider, log ghi cảnh báo, không ghi xuống đĩa.
- `DEW_ALLOWED_MODELS` (tuỳ chọn, trong env): danh sách `provider:model`. Có ít nhất một mục không phải
  `claude_code:`/`codex:` thì model chính và model nhỏ phải là cặp trong danh sách, bất kể lựa chọn từ đâu (TUI, `-m`,
  `crush.json` của repo); ngoài danh sách thì dừng với `MODEL_NOT_ALLOWED`.
- Tool `claude_code`/`codex` (giao việc cho Claude Code/Codex) mặc định TẮT; chỉ có khi `DEW_DELEGATION=on`, allowlist
  có `claude_code:<id>`/`codex:<id>`, và máy có CLI đó. Mỗi lần giao vẫn hỏi quyền, trừ các trường hợp ở
  [mục trên](#khi-nào-không-có-hộp-thoại-quyền). Bash chặn gọi thẳng `claude`, `codex`, `gemini`, `qwen`… theo tên.

**Giới hạn của lời hứa trên:**

- Chỉ phủ lời gọi LLM do chính dew phát ra, không phủ việc model làm qua tool. Chặn `claude`/`codex`… chỉ theo tên: bản
  chép đổi tên, `python -c`, script tự viết hay `curl` tới API provider khác (bằng CLI đã đăng nhập hoặc key trên máy)
  không bị nhận ra; dưới `run`/`loop`/`routine`/YOLO chúng chạy không hỏi và sổ ghi LLM không ghi. Cần chắc thì chạy
  việc tự duyệt ở nơi không có CLI đã đăng nhập hay key của provider khác.
- Không đặt allowlist thì cấu hình (kể cả `crush.json` của repo lạ) chọn được model của provider khác mà không hỏi. TUI
  hiện model đang chọn; output thường của `run` thì không (xem bằng `run --json` hoặc sổ ghi LLM).
- Allowlist chỉ so TÊN, không kiểm `base_url` thật: ai sửa được cấu hình thì trỏ được cùng tên sang endpoint khác.
- Key của provider khác đã export ở shell, hay có trong `.env` của thư mục đang đứng, làm provider đó chọn được.
- Đã đăng nhập ChatGPT/Grok thì dew có thể gọi ra OpenAI/xAI (làm mới token, lấy danh sách model; không gửi prompt) kể
  cả khi chỉ dùng model local.

```bash
# ~/.config/dew/env: model local qua server của bạn, không cho fallback chéo provider
DEW_PROVIDER_TYPE=openai-compat
DEW_BASE_URL="http://127.0.0.1:<port>/v1"   # server local tương thích OpenAI của bạn, không cần key
DEW_API_KEY=
DEW_MODEL="qwen3.8"                         # ID model mà server của bạn báo
DEW_CONTEXT_WINDOW=32768                    # để mặc định 128000 thì tự tóm tắt có thể không kịp chạy
DEW_ALLOWED_MODELS="main:qwen3.8"           # provider tự khai có ID là main
# không đặt DEW_DELEGATION: giao việc sang provider khác vẫn tắt
```

Với cấu hình này, lời gọi LLM do dew phát ra tới provider ID khác `main` là **bug** của dew; kiểm bằng sổ ghi LLM
([Quan sát sâu](#quan-sát-sâu)). Sổ ghi ID provider và tên model, không ghi URL: nó không phát hiện được `base_url` của
`main` bị một cấu hình khác (vd của repo lạ) trỏ đi nơi khác.

## "Xong" nghĩa là gì

MODEL NÓI XONG ≠ DEW ĐÃ XONG.

- `dew run` thoát 0 nghĩa là lượt chạy hết mà không có lỗi runtime, không có nghĩa là việc đúng. Là lỗi (thoát 1):
  prompt kết thúc mà model chưa ra chữ hay tool call nào (kể cả khi bộ lọc nội dung của provider chặn), dừng vì lặp cùng
  một tool call, context window quá nhỏ ngay sau tự tóm tắt mà việc còn dở, quá 10 lần tự tóm tắt cho một prompt, Stop
  hook vẫn chặn sau 3 lần chạy lại, model đang chọn không dùng được, lỗi provider.
- `send` (không `--wait`) chỉ báo nhận SAU khi server đã ghi prompt vào session; lượt có thể chưa chạy, không chạy được
  thì prompt hiện là huỷ, lỗi, hoặc "Interrupted" nếu server chết. Không tự chạy lại.
- "Xong có bằng chứng" là mã thoát của `--verify` trong `loop`/`routine`. Verify đã xanh từ trước khi sửa VÀ cây làm
  việc không đổi thì không có gì chứng minh việc đã làm: dừng `NO_CHANGE` (`loop` và routine `goal` thoát 1; routine
  `maintenance` thoát 0: hôm nay không có việc). Chỉ một trong hai thì vẫn `GREEN`, kèm cảnh báo. Thư mục không phải repo
  git (hoặc chưa có commit) thì không biết cây có đổi không, nên không ra `NO_CHANGE`.
- `GREEN` chỉ chứng minh lệnh verify thoát 0. Agent tự duyệt mọi quyền nên sửa được chính test hay script mà verify
  chạy: luôn duyệt diff trước khi commit hay merge (routine tạo với `--commit` được commit trên nhánh của nó), nhất là
  tệp test, `Makefile`, script verify (báo cáo routine có cờ `touched_tests`/`touched_gate` khi agent đụng chúng).

| Lệnh | Thoát 0 | Thoát 1 | Khác |
|---|---|---|---|
| `run`, `send --wait` | lượt chạy hết, không lỗi runtime | lỗi | 2: sai cú pháp |
| `send` (không `--wait`) | server đã nhận và đã ghi prompt | lỗi | 2: sai cú pháp |
| `loop` | `GREEN` | mọi lý do dừng khác | 2: sai cú pháp |
| `routine run` | `GREEN`; `NO_CHANGE` (maintenance) | còn lại | 2: sai cú pháp, định nghĩa khác lịch; 75: từ chối chạy (`--json` in loại: `lock_held`, `no_slot`, `low_memory`, `low_disk`) |

## Loop và routine

```bash
dew loop --verify "go test ./..." "Sửa TestParse đang đỏ"
dew routine new vet --mode maintenance --repo ~/code/proj \
  --goal "Sửa mọi cảnh báo của go vet" --verify "go vet ./..."   # thêm --commit nếu cho phép commit
dew routine run vet                                        # chạy tay một lần
dew routine list                                           # lượt cuối: trạng thái, model, thời lượng, verify
dew routine stats --since 24h                              # độ trễ/lỗi/token từng request LLM
dew routine export tidy -o tidy.patch                      # patch của lượt cuối, kèm tệp mới
```

- `loop`: chạy agent, chạy verify, sửa tiếp tới khi verify thoát 0 hoặc gặp điều kiện dừng (mặc định 2 lượt sửa); mỗi
  lượt một session mới. Verify chạy với `pipefail` (`… | head` làm lệnh trước nhận SIGPIPE, thoát 141, tính là FAIL).
  Mở từ thư mục con: verify chạy ở thư mục bạn mở, agent làm ở gốc repo (prompt nói rõ). `loop` không commit hay push,
  nhưng agent được tự duyệt mọi lệnh và loop không có lớp chặn của routine: model vẫn chạy được `git commit`/`git push`
  nếu nó quyết vậy. Cần chặn thì chạy ở nơi không có quyền push, hoặc dùng routine.
- `routine run`: mỗi lần chạy trong git worktree riêng (nhánh `routine/*`), giữ khoá, bật chế độ không người trực: lệnh
  shell bị chặn theo mẫu (push, merge, `reset --hard`, `rm` ra ngoài worktree…; danh sách chặn, không chặn được mọi
  cách), tool ghi tệp không ghi ra ngoài thư mục làm việc. Không push, không merge; chỉ commit khi định nghĩa tạo với
  `--commit`. Báo cáo ghi ra ngoài worktree; bạn duyệt diff rồi dọn bằng lệnh được in.
- `routine list`/`show` cho thấy lượt đang chạy (`RUNNING`), chết giữa chừng (`ABANDONED`), lỗi sớm (`ERROR`).
- Lịch chỉ được IN ra (`routine schedule <tên> --print`, chỉ mode `maintenance`) sau khi bạn thêm khối `schedule` vào
  tệp định nghĩa (`~/.config/dew/routines/<tên>.json`, đường in ra lúc `routine new`; ví dụ trong guide); bạn tự cài
  timer/crontab. Lịch ghim sha256 của định nghĩa: sửa định nghĩa thì in và cài lại. Allowlist đặt trong tệp env: lịch
  không chép allowlist từ shell.
- Chế độ không người trực không phải sandbox: chỉ chạy routine trên repo và máy bạn tin. Chi tiết:
  [Loop và Routine](docs/dew-local-setup.md#loop-và-routine).

## Memory: không có kho riêng

dew không có kho memory riêng. Nó nhớ bằng ba thứ:

- Context file: `AGENTS.md`, `CRUSH.md`, `CLAUDE.md`… ở gốc dự án (mở từ thư mục con cũng vậy; tệp của thư mục con
  không được đọc), và `~/.config/dew/CRUSH.md` cho mọi dự án. Agent con và plan mode không nhận context file của dự án.
- Lịch sử session (tiếp bằng `-s`/`-C` hoặc danh sách session).
- Tự tóm tắt khi context gần đầy. Bản tóm tắt (LLM viết, có thể chép lại chữ từ web/MCP) gửi model kèm nhãn nguồn: ngữ
  cảnh do assistant sinh, không phải chỉ thị của người dùng. Nhãn chỉ giảm nhẹ, chưa đo với LLM thật.

Thứ tự nên tin khi lịch sử hay context file nói khác mã hiện tại: mã nguồn/runtime hiện tại > bằng chứng vừa kiểm >
lịch sử > suy luận của model (quy ước cho bạn và model; dew không ép bằng máy). Đã thử với LLM thật, một lần: `AGENTS.md`
chứa khẳng định cũ sai và một chỉ thị độc, model vẫn làm theo mã hiện tại; một lần không phải bảo đảm chống chỉ thị độc.

## Skill

- Skill là tệp `SKILL.md` dew tìm thấy: `~/.agents/skills`, `~/.claude/skills`, `.agents/skills` và `.claude/skills`
  của dự án…, cộng 3 skill dựng sẵn. Thư mục ẩn bên dưới các gốc đó (vd `.trash`) không được quét.
- Tên và mô tả skill vào system prompt; model tự quyết có đọc skill hay không. Tắt: `option disable-skill <tên>` trong
  crushrc (`options.disabled_skills` trong `crush.json`).
- Luật phải luôn áp dụng (đòi bằng chứng, không tự commit…) đặt trong context file (`~/.config/dew/CRUSH.md`,
  `AGENTS.md`/`CLAUDE.md` của dự án), không đặt trong skill: trong log 3 ngày dùng thật của tác giả (41 lượt), model chỉ
  nạp skill ở 2 lượt.
- Ponytail, Superpowers là bộ skill tuỳ chọn. Runtime dew không phụ thuộc chúng.

## Cấu hình

| Việc | Ở đâu |
|---|---|
| Provider, model, key, allowlist | biến `DEW_*` trong `~/.config/dew/env` (mode 600, mẫu: `dew.env.example`) |
| Model đã chọn cho một dự án | `<dự án>/.crush/crush.json` (thắng env) |
| Cấu hình chính, quyền công cụ | `~/.config/dew/crushrc` (mẫu: `dew.crushrc.example`) |
| Màu giao diện | `~/.config/dew/themes/*.json`, chọn bằng `options.tui.active_theme` |
| Dữ liệu phiên | `~/.local/share/dew`, và `.crush/` ở gốc từng dự án (tự có `.gitignore` bỏ qua mọi thứ bên trong) |
| Log | `dew logs` (đọc đúng tệp log của dự án); `dew dirs` in các thư mục đang dùng |

Key ghi thẳng trong env thì đặt trong nháy đơn (`'sk-…'`), hoặc `"$(pass show <entry> | head -n1)"`. Launcher `dew`
(wrapper ở `~/.local/bin`) đặt sẵn các biến `CRUSH_*`, tắt metrics, tắt cập nhật catalog provider và kiểm tra bản mới
của upstream. Sửa env khi server của dự án đang chạy thì chưa có tác dụng. Client chạy model mở sau với chính sách tin
cậy khác (allowlist, giao việc, `--yolo`…; so chuỗi thô, lấy từ env gốc và tệp env, không từ `.env` của repo) bị từ
chối (`trust policy differs from the running dew server`, nêu tên biến, không giá trị); server khác bản đang bận cũng
vậy. Biến thường (kể cả `$VAR` của `api_key`/`base_url` trong `crush.json`) không được so: nó theo client đã dựng
server. Thoát mọi client, chờ server tắt rồi mở lại ([chi tiết](docs/dew-local-setup.md#sửa-env-khi-dew-đang-chạy)).

## Phím tắt

Phím hiện dưới chân màn hình; `ctrl+g` xem thêm; `dew keys` in bảng phím của màn chat (không gồm Home và hộp thoại).
**Home:** gõ để tìm · `↑↓` chọn · `enter` mở · `ctrl+p` ghim/bỏ ghim · `ctrl+x` bỏ dự án đã mất khỏi danh sách · `esc`
xoá ô tìm, `esc` lần nữa thoát.

| Phím (màn chat) | Việc |
|---|---|
| `enter` · `ctrl+j` · `ctrl+o` | gửi · xuống dòng · mở editor ngoài |
| `@` · `/` hoặc `ctrl+p` · `ctrl+f` · `ctrl+v` | chèn file (đường theo gốc repo) · bảng lệnh · đính kèm ảnh · dán ảnh |
| `tab` · `shift+tab` · `ctrl+y` | đổi focus · xoay mode (code → plan → code + YOLO) · bật/tắt YOLO |
| `alt+n` · `f2` · `alt+s` · `alt+.`/`alt+,` | session mới · đổi tên session · danh sách session · session kế/trước |
| `alt+p` · `alt+m` · `alt+e` | provider · model của provider đó · effort |
| `alt+w` · `alt+j` · `ctrl+b` · `ctrl+t` · `ctrl+d` | về Home · bảng Jobs (bash nền) · sidebar · To-Do · chi tiết |
| `esc` · `ctrl+c` · `ctrl+z` | huỷ lượt đang chạy · thoát · treo |

`ctrl+c` ở chế độ server mặc định để lượt đang chạy chạy tiếp: muốn dừng thì `esc` trước. Tên đặt bằng `f2` không bị
tiêu đề do model sinh ghi đè. Terminal chiếm `f2` thì dùng `alt+s` rồi `ctrl+r`.

## Quan sát sâu

- `dew ps [--json]`: các lượt `routine run` đang chạy (hàng `ROUTINE`, path là worktree) và mọi session của các dự án có server đang chạy, trạng thái `BUSY` / `WAITING` (chờ bạn duyệt hay trả
  lời) / `IDLE`. Mặc định 9 session mới nhất mỗi dự án; `--all` bỏ trần đó. Không khởi động server nào.
- `dew jobs`, `jobs output <id>`, `jobs kill <id>`: lệnh bash agent đã chạy, kèm session sở hữu và mã thoát.
- `dew logs -f` (bám đuôi), `-t <N>` (N dòng cuối). `dew session list|show|last --json`: session dạng máy đọc.
- `dew run --json`: `session_id`, `text` (câu trả lời cuối; chế độ local `CRUSH_CLIENT_SERVER=0`: mọi chữ assistant của
  lượt nối lại; không có bản tự tóm tắt), `error`, `provider`/`model` (của tin trả lời cuối đã lưu, có cả khi request
  lỗi; bằng chứng provider nào đã được gọi là sổ ghi LLM), `prompt_tokens`/`completion_tokens` (bước cuối, không phải
  tổng), `cost` (cộng dồn của session), `ledger_error` (chỉ khi sổ ghi LLM bật mà lượt có lời gọi không ghi được).
- Sổ ghi LLM (opt-in): `DEW_LLM_LEDGER=<tệp>` ghi mỗi lời gọi LLM một dòng JSON (vai trò, provider, model, kích thước,
  token; không chép prompt). KHÔNG đọc từ `~/.config/dew/env`, lịch routine không mang theo: export ở shell trước khi
  server của dự án khởi động. Ghi sổ lỗi (vd đĩa đầy) thì lượt vẫn chạy, log có WARN mỗi prompt (đĩa đầy thật thì log
  cũng có thể không ghi được); chỉ `run --json` và `send --wait --json` có thêm `ledger_error`. Đừng dùng sổ làm bằng
  chứng cho lượt đó. [Giới hạn khác](docs/dew-local-setup.md#cô-lập-provider-sổ-ghi-llm-bằng-chứng-của-looproutine).

## Giới hạn còn lại

Danh sách: [docs/dew-local-setup.md, mục "Giới hạn đã biết"](docs/dew-local-setup.md#giới-hạn-đã-biết). Lớn nhất: mở
repo lạ là chạy cấu hình của nó ([trên](#khi-nào-không-có-hộp-thoại-quyền)); model chọn trong TUI hay `run -m` là của cả
dự án; allowlist chỉ so tên provider/model; context file chỉ nạp ở gốc dự án; `run` không có trần thời gian riêng (bọc
`timeout` nếu cần); chế độ không người trực là danh sách chặn, không phải sandbox; chữ đi kèm ảnh/media của MCP và
`read_mcp_resource` chưa có trần; mở cùng dự án bằng launcher và bằng `CRUSH_CLIENT_SERVER=0` là hai tiến trình trên
cùng một `crush.db`.

## Khác upstream

- Tên và giao diện dew; không gọi GitHub kiểm bản mới (chỉ bật khi chạy thẳng binary với
  `CRUSH_DISABLE_UPDATE_CHECK=false`). Crushrc mẫu tắt dòng `Assisted-by: Crush` / `Generated with Crush` trong commit
  và PR ([cách bật lại](docs/dew-local-setup.md#attribution-trong-commit-và-pr)).
- Nhiều session/dự án trong một terminal, run sống khi đóng TUI, `loop`/`routine`, ranh giới provider, hộp thoại quyền
  mặc định Deny.
- Chưa đổi: tên module Go, system prompt (còn tự xưng "Crush"), skill và tool nội bộ `crush-config`, `crush_info`.

## Tài liệu và phát triển

[Hướng dẫn đầy đủ](docs/dew-local-setup.md) (cấu hình, server, `dew-tmux`, loop/routine, giới hạn) ·
[đồng bộ giữa các máy](docs/dew-sync.md) · [CHANGELOG.md](CHANGELOG.md) · [DECISIONS.md](DECISIONS.md) (sổ quyết định) ·
[lịch sử từ fork Crush tới DEW](docs/project-history.md) · [bàn giao V3](reports/v3-research/HANDOFF_2026-10-20.md)
(đọc trước khi làm tiếp) · [kế hoạch đổi tên](reports/documentation/RENAME_PLAN.md) · [AGENTS.md](AGENTS.md) (quy ước
code, kiến trúc). Build: `go build .`; test: `GOMAXPROCS=2 go test ./... -p 2` (máy dùng chung: giới hạn CPU).
