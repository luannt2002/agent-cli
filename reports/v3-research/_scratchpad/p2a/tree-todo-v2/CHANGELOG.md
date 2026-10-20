# Changelog

Theo [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) và [SemVer](https://semver.org/lang/vi/).
Mỗi bản có một tag git có chú thích `vX.Y.Z`. `agent-cli --version` hiện đúng số đó khi build từ commit
được gắn tag (Go gắn version từ tag, không cần `-ldflags`; build từ commit sau tag sẽ hiện dạng
`vX.Y.Z-0.<ngày>-<hash>`, cây bẩn có đuôi `+dirty`).

Quy trình phát hành: xem [CLAUDE.md](CLAUDE.md), mục P6.

## [Unreleased]

### Thêm

- **`agent-cli loop --verify "<lệnh>" "<việc>"`**: chạy agent, rồi lệnh verify, sửa tiếp cho tới khi verify
  thoát 0 hoặc gặp điều kiện dừng. "Xong" là mã thoát của `--verify`, không phải lời agent. Mỗi lượt là một
  session mới với prompt tự chứa đủ (đề bài + đuôi output verify vừa đo, kèm lệnh cấm sửa test/cổng để qua).
  Lý do dừng: `GREEN`, `NO_CHANGE` (verify xanh từ trước mọi thay đổi và cây làm việc không đổi: không có gì
  chứng minh việc của agent), `MAX_REPAIRS` (mặc định 2 lượt sửa, tổng 3 lượt, không có lần vá thứ ba), `STALLED` (cùng
  chữ ký lỗi và cây làm việc không đổi sau một lượt sửa), `TIMEOUT` (`--max-duration`, kiểm trước mỗi lượt),
  `RUN_ERROR` (lượt agent lỗi hoặc quá `--run-timeout`), `CANCELLED` (Ctrl+C/SIGTERM), `UNMEASURED` (verify
  không chạy được: không tìm thấy lệnh, lỗi cú pháp, quá hạn; không đốt lượt agent nào). Verify đã xanh sẵn
  trước mọi thay đổi thì ghi cảnh báo "cổng có thể không đo đúng mục tiêu" nhưng vẫn chạy.
  Verify chạy với `set -o pipefail`: `go test ./... | tee log` không che được lỗi; exit 141 (SIGPIPE, thường do
  `| head`, `| grep -q`) vẫn là FAIL nhưng output có gợi ý. Verify xanh từ baseline VÀ cây không đổi thì dừng
  `NO_CHANGE` (thoát 1); chỉ một trong hai thì vẫn `GREEN`, dòng dừng và `detail` của `--json` ghi cảnh báo.
  Journal JSON Lines (`<data>/loops/<tên>-<thời điểm>.jsonl`, 0600) được ghi và `Sync` TRƯỚC mỗi bước.
  Mã thoát: 0 `GREEN`, 1 mọi lý do dừng khác (kể cả `NO_CHANGE`), 2 dùng sai. `--json` in một object ở cuối. Không bao giờ commit,
  push hay merge. Cùng mô hình quyền như `run` (tự duyệt tool): dành cho việc bạn khởi chạy và trông.
- **`agent-cli routine new|list|show|run|schedule`**: routine là định nghĩa đã lưu (đích, lệnh verify, repo, giới
  hạn, lịch tuỳ chọn) chạy bằng engine của `loop`. Tệp `~/.config/agent-cli/routines/<tên>.json` (0600, nằm ở cấu
  hình của bạn, không nằm trong repo nên repo lạ không tự thêm được việc chạy theo lịch). Hai mode: `maintenance`
  (cơ học, kiểm được, vô tận: được lên lịch) và `goal` (có đích như làm tính năng: chỉ chạy tay; `--scheduled`
  bị từ chối). Định nghĩa bắt buộc có `verify` ("xong" là lệnh nào), từ chối trường lạ, `max_repairs` 0–5.
  `routine run` mỗi lần (từ chối `--cwd`): giữ khoá (lượt thứ hai thoát **75**, không có khoá mồ côi); làm việc trong **git worktree
  riêng** `routine/<tên>-<thời điểm>` ngoài repo (checkout của bạn không bị đụng; worktree và nhánh được giữ lại để
  bạn duyệt, kèm lệnh xem diff và lệnh dọn); chạy agent in-process với **chế độ không người trực** bật; ghi báo
  cáo 0600 ngoài worktree (sha định nghĩa, nhánh, base, lý do dừng, verify cuối, journal). Không commit trừ khi
  định nghĩa có `"commit": true`; không bao giờ push hay merge. `--expect-sha` từ chối chạy khi tệp định nghĩa đã
  đổi so với lúc cài lịch. Mã thoát: 0 GREEN (và `NO_CHANGE` của routine `maintenance`: không có việc), 1 khác (kể
  cả `NO_CHANGE` của routine `goal`), 2 dùng sai, 75 đang chạy.
  `routine schedule NAME --print` chỉ IN unit systemd `--user` (`Persistent=true`, `TZ`/`PATH` tường minh,
  `TimeoutStartSec`) và/hoặc dòng crontab, kèm lệnh cài để bạn tự chạy; agent-cli không bao giờ sửa timer hay crontab.
  Lịch in ra mang theo biến của launcher và `AGENT_CLI_ENV_FILE` (bản đầu thiếu chúng nên không chạy được, xem Sửa
  lỗi). Allowlist provider KHÔNG được chép vào lịch: lượt theo lịch đọc nó từ tệp env, nên siết allowlist trong tệp
  env là có hiệu lực ngay; allowlist chỉ đặt ở shell thì `schedule --print` in cảnh báo ra stderr.
  Báo cáo từng lượt: `RUNNING` ghi ngay khi bắt đầu; lượt chết giữa chừng hiện `ABANDONED`; hỏng trước khi agent chạy
  (vd không tạo được worktree) hiện `ERROR` kèm nguyên nhân; `routine show` liệt kê mọi lượt (`runs`). Báo cáo có
  `changed_files`, `touched_tests`, `touched_gate` (GREEN mà agent đã đổi test hay tệp mà verify gọi thì stderr cảnh báo),
  `baseline_pass`, `unchanged_tree`.
- **Chế độ không người trực** (`AGENT_CLI_UNATTENDED=1`; `routine run` tự bật và tự xuất biến này cho mọi tiến
  trình con rồi trả lại): lớp chặn CỨNG ở trình thực thi shell, trước cả quyền đã cấp và YOLO. Đây là danh sách
  đen cộng một danh sách TRẮNG cho git, KHÔNG phải sandbox. Cụ thể:
  - `git` chỉ cho: lệnh đọc (`status`, `diff`, `log`, `show`, `blame`, `grep`, `ls-files`, `rev-parse`...),
    `add/rm/mv/apply`, `reset` không `--hard`, `branch`/`tag` không xoá/đổi tên, `checkout -b`/`switch -c`,
    `restore --staged`, `stash list|show`, `remote -v|show|get-url`, `worktree list`, `config` chỉ đọc.
    Lệnh con lạ, rỗng hoặc chưa mở rộng bị chặn (nên alias `git -c alias.p=push p`, `git config alias.p push`,
    cờ toàn cục lạ như `--attr-source`, `git-push`, `send-pack`, `xargs git`, `bash -c 'git "$1"' _ push` đều bị
    chặn); `-c`/`--config-env` bị cấm. `git commit`, `am`, `cherry-pick`, `revert` chỉ mở khi định nghĩa có
    `"commit": true` (`AGENT_CLI_UNATTENDED_COMMIT=1`); không mở thứ gì khác.
  - `rm` có đích tuyệt đối, `~`, `$VAR`, `..` hoặc không có đích (tham số từ `xargs`) bị chặn, đệ quy hay không;
    `rsync`, `sftp`, `gh`, `glab`, `socat`, `ncat` bị chặn (cùng `ssh`, `scp`, `curl`, `sudo`, `crontab`,
    `systemctl` đã bị cấm sẵn). Bóc lớp bọc: `env`, `timeout -s`, `taskset -c 0,2`, `flock FILE`, `strace`,
    `systemd-run --user`, `nice`, `chrt`, `xargs`... và script trong `sh -c`/`eval` (vòng lặp, `$(...)`).
  - Tool bash từ chối `working_dir` ngoài workspace; tool giao việc `claude_code`/`codex` không được đăng ký;
    `agent-cli loop`/`routine` gọi từ bash bị chặn như `run`/`send`/`server`.
  - Tool ghi tệp (`write`, `edit`, `multiedit`, `download`, `lsp_replace_symbol`, `lsp_rename`) bị từ chối khi đích
    nằm ngoài thư mục làm việc, trước cả YOLO và `allowed_tools`. Bash thì không qua kiểm tra này.
  - `git -C`/`--git-dir`/`--work-tree` trỏ ra ngoài workspace bị chặn; option dài viết tắt được so như git
    (`--har` là `--hard`).
  - Mọi lệnh có chữ nhắc tới biến trỏ git sang repo khác hoặc nạp cấu hình (`GIT_DIR`, `GIT_WORK_TREE`,
    `GIT_INDEX_FILE`, `GIT_CONFIG_*`…) bị chặn, bất kể lệnh và giá trị, kể cả trong `sh -c`, `env -S`, `eval`. Cố ý
    nghiêng về chặn: biến shell thường trùng tên cũng bị chặn.
  - Biến là biến tin cậy: đọc từ env gốc đã chụp đầu main, nên `.env` của repo không bật/tắt được.
  KHÔNG phải ranh giới an ninh: agent viết mã rồi chạy `go test`/`make`/verify, và mã đó có thể gọi `git push`.
  Chưa phủ `python`/`node` tự xoá hay mở socket, `find -delete`, ghi tệp qua redirect/`tee`/`cp`/`mv`, `cd /; rm
  -rf x` (rm không xét `cd`), thư mục symlink trỏ ra ngoài, tên biến git ghép lúc chạy (`"GIT_""DIR"`), script nạp từ
  tệp hoặc stdin. Worktree dùng CHUNG refs, `.git/config` và hooks với repo gốc; hooks của repo chạy lúc
  `git worktree add`. Vì vậy chỉ chạy routine trên repo và máy bạn tin, và coi `--scheduled` là khai báo (rào thật
  là: định nghĩa `goal` không thể mang lịch và không in được lịch).
  Chưa có: UI (Alt+R, sidebar, dòng trạng thái), lịch tự cài, pause/resume, nhiều worker ghi song song.
- **Sổ ghi lời gọi LLM (opt-in): `AGENT_CLI_LLM_LEDGER=<tệp>`.** Mỗi lời gọi LLM, kể cả lần thử lại, thêm một dòng JSON:
  vai trò (`main`, `title`, `summary`, `subagent`, `agentic_fetch`), provider và model thật đã gọi, kích thước prompt,
  hash system prompt, tên tool, token của chính lời gọi đó, lý do kết thúc, loại lỗi (chỉ loại và mã HTTP). Không chép
  nội dung prompt, câu trả lời hay văn bản lỗi; tệp mode 600. Biến tin cậy: chỉ đọc từ env gốc, `.env` của repo không
  bật hay chuyển được sổ. Dùng để kiểm "phiên chỉ dùng provider X" và để đếm token thật (bộ đếm của session chỉ giữ bước cuối).
- **`run --json` có `provider` và `model`** của câu trả lời cuối (đọc từ tin đã lưu, không phải model đang chọn).
- **`scripts/eval/`: đo và cổng hồi quy bằng LLM giả.** `fakellm.py` (server tương thích OpenAI, ghi mỗi request:
  provider, model, vai trò, kích thước); `faults.sh` (tiêm lỗi provider, ca model đang chọn thiếu key); `qwen-only.sh`
  (đi qua mọi đường gọi LLM với model nhỏ cố ý ở provider khác); `workspaces.sh` (nhiều workspace, nhiều session, chế độ local);
  `fixtures.py`/`run.sh`/`score.py` (bộ kịch bản đo trước/sau, chạy bằng cấu hình provider thật của bạn nên tốn hạn mức). `faults.sh`, `qwen-only.sh`,
  `workspaces.sh` thoát 1 khi hỏng.

### Sửa lỗi

- **Lời gọi ẩn không còn sang provider khác.** Model nhỏ nằm ở provider khác model chính từng nhận lời gọi đặt tiêu đề
  (kèm nguyên văn prompt đầu của bạn) và `agentic_fetch`, không cảnh báo: phiên chọn Qwen local vẫn gửi prompt sang
  provider của model nhỏ. Bộ tiêm lỗi với provider giả: bản trước có lời gọi lọt ở cả 8/8 ca (kể cả khi provider
  chính chết); bản này 0/8. Nay model nhỏ ở provider khác được thay bằng model chính, log cảnh báo một lần. Tóm tắt
  và agent con vốn đã dùng model chính.
- **Bớt lời gọi đặt tiêu đề vô ích.** Session con (agent con, `agentic_fetch`) không còn gọi LLM đặt tiêu đề. Tiêu đề bị
  cắt vì hết token thì không thử lại bằng model khác, giữ tiêu đề suy ra từ prompt (đo baseline: 26/26 lời gọi tiêu đề
  kết thúc vì hết token, không lần nào ra tiêu đề). Chỉ thử thêm model chính khi nó khác model nhỏ.
- **Tự tóm tắt không còn lặp vô hạn.** Khi phần cố định của prompt gần đầy context window, tóm tắt không giải phóng được
  chỗ và từng lặp mãi (755 lời gọi chính + 755 lời gọi tóm tắt trong 90 giây với provider giả). Nay lượt dừng với lỗi
  `context window too small…` khi việc còn dở mà riêng phần đầu vào sau tóm tắt đã vượt ngưỡng (output lớn, vd ghi
  tệp lớn, thì vẫn tóm tắt tiếp). Thêm trần 10 lần tự tóm tắt cho một prompt khi việc còn dở (một kịch bản thật với
  Qwen từng chạy 1.338 lời gọi chính trong 120 giây); quá trần thì lượt dừng có lỗi, gửi prompt mới để làm tiếp.
- **Bản tóm tắt bị cắt** (hết ngân sách output) không còn thay cho toàn bộ lịch sử: tính là tóm tắt hỏng.
- **Sau tự tóm tắt, lượt làm tiếp chạy trước prompt đang xếp hàng**; lỗi của prompt xếp hàng không còn bị báo là
  "tóm tắt hỏng".
- **Mở lại session sau khi tiến trình chết:** lời trả lời chỉ chữ hoặc rỗng và bản tóm tắt viết dở được đóng lại (kết
  thúc lỗi), UI thôi hiện đang chạy, bản dở không vào lịch sử gửi model.
- **`run` không còn báo thành công giả.** Prompt kết thúc mà model chưa ra chữ hay tool call nào, xét cả các lượt tự
  làm tiếp của prompt đó, nay là lỗi `empty reply`; bộ lọc nội dung của provider chặn trước khi có gì thì báo lỗi riêng
  (TUI vẫn hiện Refused); bị cắt vì hết ngân sách output thì vẫn tự làm tiếp trước, hết lượt mới báo lỗi. Lượt bị dừng
  vì lặp cùng một tool call cũng là lỗi. `run` thoát 1 cho các trường hợp này. Đã gọi tool hay ra chữ rồi im lặng thì
  không phải lỗi.
- **`run` chạy local không còn in bản tự tóm tắt.** Lượt có tự tóm tắt từng in cả bản tóm tắt dính liền câu trả lời ra
  stdout và vào `text` của `--json` (vd `## Current State…SUMMARY…DONE`); nay bản tóm tắt không còn được in. Chế độ
  local vẫn in mọi chữ assistant của lượt (kể cả câu dẫn trước tool call); qua server `text` chỉ có câu trả lời cuối.
- **Lệnh bash bị lớp chặn từ chối thì model biết lý do.** Khi lớp bảo vệ tiến trình hoặc lớp không người trực chặn một
  lệnh, output gửi model có thêm dòng `Blocked: <lý do>` thay vì chỉ "not allowed for security reasons". Dòng này có
  trần (5 lý do, mỗi lý do 300 ký tự) vì nó nằm ngoài bước cắt output.
- **Model đang chọn không dùng được thì agent-cli không đổi sang provider khác.** Trước đây, khi provider dựng sẵn của
  model đang chọn thiếu API key (hoặc ID model không còn), agent-cli lặng lẽ chạy bằng model của provider khác và ghi
  lựa chọn đó xuống `~/.local/share/agent-cli/crush.json`. Đo bằng provider giả (`scripts/eval/faults.sh`, ca
  `selected-no-key`: chọn `openai/gpt-4o` không có key): bản trước thoát 0 sau 2 request sang provider `glm`; bản này
  thoát 1, 0 request. Nay: chỉ mất ID model thì dùng model mặc định của CÙNG provider (log cảnh báo, không ghi xuống
  đĩa); provider không dùng được thì giữ nguyên lựa chọn: `run`, `loop`, `routine` dừng ngay với lỗi nêu provider/model
  và cách sửa (`run -m` không vượt qua lỗi này), còn app vẫn khởi động không agent như lúc chưa cấu hình, nên
  `agent-cli login <nền tảng>` chạy được và TUI mở thẳng màn chọn model.
- **Panic log không còn rơi vào thư mục đang làm việc.** Tệp `crush-panic-*.log` nay nằm cùng thư mục với log, quyền 600
  (trước đây ở thư mục hiện tại, quyền mặc định, dễ bị commit nhầm). Bước dọn dẹp sau panic chạy cả khi không ghi được tệp.
- **Chi phí của agent con được cộng vào session cha cả khi agent con lỗi** (nó có thể đã tiêu tiền rồi mới lỗi).
- **Hộp thoại quyền của một session không còn chặn session khác.** Yêu cầu đang chờ huỷ được; session được tự duyệt
  (lượt `run`) không xếp hàng sau hộp thoại của session khác.
- **`run` không còn treo vì quyền của session con.** Chạy local, yêu cầu quyền của agent con hay `agentic_fetch` không ai
  trả lời nên treo tới hết giờ. Nay `run` (local như qua server) duyệt session con của chính lượt đó; không tra được
  session gốc thì không duyệt.
- **`agentic_fetch` tôn trọng `disabled_tools`.** Tool con bị lọc theo danh sách tắt; tắt cả `web_fetch` lẫn `web_search`
  thì không đăng ký `agentic_fetch`; tắt `web_fetch` thì chế độ URL bị từ chối. Tắt `fetch` (tên tool trong cấu hình)
  cũng tính là tắt `web_fetch` của `agentic_fetch`. Session con của nó không còn tự duyệt
  (nội dung web có thể bảo agent đọc tệp ngoài rồi gửi đi); trong TUI, đọc ngoài thư mục tạm thì hỏi.
- **Đóng một workspace không còn giết job nền và đóng MCP của workspace khác cùng server.** App đóng chỉ dừng job của
  gốc mình; App cuối mới dừng hết và đóng MCP. Còn hai giới hạn hẹp (MCP sau App cuối, job của App mới cùng gốc): xem
  `docs/agent-cli-local-setup.md`, mục "Giới hạn đã biết".
- **`job_output`/`job_kill` chỉ thấy job của session đang gọi.** Job của session khác trả "not found" (ID job đánh số lại
  sau mỗi lần khởi động nên từng có thể trùng).
- **Job nền lỗi không còn hiện ✓.** `job_output` trả `exit_code` khi job đã xong; UI đổi icon lỗi và ghi `exit=N`. Chưa
  xem trong TUI thật.
- **Skill trong thư mục ẩn (vd `~/.claude/skills/.trash`) không còn được liệt kê.** Trên máy tác giả, khối skill gửi kèm
  mỗi request nhỏ đi 4.691 B (31.860 → 27.169 B). Gốc skill nằm trong thư mục ẩn (`~/.agents/skills`) vẫn được quét,
  kể cả khi viết kèm `/` cuối.
- **Log không còn ghi giá trị env của client** (có thể chứa API key đã export) khi hai client mở cùng workspace với cờ
  khác nhau: chỉ ghi tên các biến khác nhau.
- **Lịch routine in ra chạy được.** Bản đầu (trên `main` sau v0.2.0) thiếu biến của launcher (`CRUSH_GLOBAL_CONFIG`…),
  nên mỗi lần chạy theo lịch tìm định nghĩa ở `~/.config/crush`, thoát 2 và không chạy routine. Nay lịch mang theo
  `CRUSH_GLOBAL_CONFIG`, `CRUSH_GLOBAL_DATA`, `CRUSH_CACHE_DIR`, ba biến tắt metrics/cập nhật và `AGENT_CLI_ENV_FILE`,
  đọc từ env gốc (`.env` của thư mục hiện tại không chen vào được); giá trị có ký tự lạ thì lệnh từ chối.
- **Routine bị kill giữa chừng không còn bị che.** Bản đầu không để lại báo cáo, `list`/`show` hiện lượt GREEN cũ; lỗi
  trước khi agent chạy hiện như lượt bỏ dở. Nay có `RUNNING`/`ABANDONED`/`ERROR` (mục Thêm).
- **Verify của loop/routine không còn GREEN nhờ `| cat`.** `pipefail` (mục Thêm).
- **Chế độ không người trực bịt thêm lỗ.** Bản đầu để lọt: ghi tệp ra ngoài bằng tool ghi tệp (vd `~/.config/agent-cli/crushrc`,
  chạy như Bash ở lần mở sau), `git -C <repo khác>`, option viết tắt (`--har`), biến `GIT_DIR=… git rm`. Nay đều bị chặn
  (mục Thêm).
- **Tiêu đề markdown (H1) vô hình trong theme tuỳ chỉnh.** Theme tuỳ chỉnh (vd `github-dark-dim` trong
  `~/.config/agent-cli/themes/`) không nhận lớp "yên tĩnh" của theme dựng sẵn, nên H1 rơi về kiểu `quickStyle`
  gốc: chữ `warning_subtle` trên nền `primary`. Với `#ffa657` trên `#79c0ff` đó là 1,00:1 (đo bằng công thức
  WCAG). Nay theme tuỳ chỉnh dùng chữ `fg_base` cho H1, heading và inline code, nền `bg_less_visible`, H6 dùng
  `fg_subtle`; chỉ dùng token của chính theme nên theme sáng vẫn đúng. Theme dựng sẵn (Charmtone, Gruvbox,
  mặc định) không đổi. Phải cài lại binary mới thấy (`scripts/install-agent-cli.sh`).
  Chưa sửa: logo gradient (`secondary → primary`) và các style khác của `quickStyle` trong theme tuỳ chỉnh.
- **Chỉ dẫn của MCP server (`<mcp-instructions>`) chỉ gửi cho lượt có đường tới server đó.** Trước đây chỉ dẫn của
  mọi MCP đang kết nối được gắn vào system prompt của mọi lượt, kể cả agent con (`agent`), `agentic_fetch` và plan,
  vốn không có tool MCP nào: model nhận lời dặn gọi tool nó không có. Nay lượt nhận chỉ dẫn của một server khi có
  ít nhất một tool của server đó, hoặc có `list_mcp_resources`/`read_mcp_resource` (hai tool này đọc được mọi
  server). Agent chính có MCP vẫn nhận đủ như cũ. Các khối chỉ dẫn nay xếp theo tên server: trước đây thứ tự ngẫu
  nhiên khi có từ 2 server, nên system prompt đổi giữa các lượt và prompt cache của provider bị trượt. Test:
  `TestMCPInstructionsFollowRunTools`, `TestRunMCPInstructionsFollowRunTools`, `TestMCPInstructionsOrderIsStable`.
- **Lời nhắc todo chỉ gửi cho lượt có tool `todos`.** Trước đây mọi agent không phải agent con nhận lời nhắc "dùng
  tool `todos`", kể cả `agentic_fetch` và plan vốn không có tool đó; request tóm tắt (không có tool nào) cũng nhận. Nay
  lời nhắc đi theo bộ tool của chính lượt đó. Agent chính (có `todos`) không đổi. Test:
  `TestTodoReminderOnlyWithTodosTool`.

## [0.2.0] - 2026-10-16

Đợt sửa theo [audit 16/10](docs/reviews/2026-10-16-deep-audit-agent-cli.md) (mã nguồn `87233bc8`).
Plan: [docs/superpowers/plans/2026-10-16-audit-fixes-v0.2.0.md](docs/superpowers/plans/2026-10-16-audit-fixes-v0.2.0.md).

### Bảo mật

- **bash tự duyệt không còn đọc được ngoài workspace** (audit X-01). Lệnh chỉ-đọc có đối số là đường dẫn
  tuyệt đối, `~`, `..`, `$VAR`, `--cờ=/đường/dẫn`, `git diff --no-index`, hoặc chạy với `working_dir` ngoài
  workspace nay HỎI quyền. Đối số được kiểm sau khi bỏ nháy, `\` và chặn brace expansion, vì shell ghép chuỗi
  trước khi chạy (`ls \/etc`, `ls {/etc,}`, `ls .''./etc` từng lọt ở bản thử đầu).
  `printenv`, `ps` (cờ `e` in biến môi trường) và mọi `git config` không còn trong danh sách an toàn;
  giá trị dính liền cờ ngắn (`date -f/etc/hostname`) cũng bị coi là đường dẫn.
- **Self-kill guard kín hơn** (B-05): script sau `sh/bash/zsh/dash/ksh/rbash -c` (kể cả `-c --`) và `eval` được
  PHÂN TÍCH CÚ PHÁP bằng `mvdan.cc/sh/v3/syntax` rồi soi từng lệnh đơn, kể cả trong `for/if/while`, hàm,
  `$(...)`, backtick; nháy và `\` chen giữa từ được giải (`ki""ll`). Bản đầu dùng regex tách chuỗi và bị lượt
  soát độc lập phá ngay (`for p in N; do kill $p; done`). Thêm: chặn `xargs kill` (PID từ stdin), `kill -1` /
  pid 1, `kill` không PID tường minh (trừ `kill -l`); bóc thêm `exec`, `builtin`, `time`, `taskset`, `busybox`
  và đối số thời gian `5s` của `timeout`; tên lệnh động (`$k X`, `"$@"`) cạnh chữ kill thì chặn. Tối đa 3 lớp
  `sh -c` lồng nhau, sâu hơn thì chặn. Có test nối `blockFuncs()` vào shell thật (C-05): xoá dòng nối ở `bash.go`
  thì test đỏ.
  Cố ý KHÔNG phủ: script nạp từ tệp hoặc stdin (`bash x.sh`, `echo 'kill N' | sh`), `env -S`, `find -exec`,
  `script -c`, ngôn ngữ khác (`python -c "os.kill"`). Chặn oan đã biết: `kill $VAR` bên trong `bash -c` (không
  biết giá trị lúc phân tích); dùng `/bin/kill $VAR` ở lớp ngoài (đã expand thành số) hoặc tool `job_kill`.

### Sửa lỗi

- **Lệnh bash thoát mã ≠ 0 không còn hiện ✓ xanh** (X-02). `BashResponseMetadata.ExitCode` mới; UI đổi icon
  thành lỗi và ghi `(exit=N)`. Không đặt `IsError`, nên nội dung gửi model và cassette VCR giữ nguyên.
- **Stop hook vẫn chặn sau 3 lần chạy lại thì lượt báo lỗi** (C-01): `RunComplete` cuối mang
  `Stop hook still blocking after 3 re-runs: …`; `run` coi `RunComplete.Error` là lỗi (`agent run failed: …`,
  `internal/cmd/run.go`) nên thoát khác 0. Test chỉ phủ phần `RunComplete`; chưa có test `run` end-to-end.
  Timeout, crash và halt vẫn kết thúc như thành công (hook không quyết định được).
- **Giới hạn duyệt tệp theo workspace, không theo cwd tiến trình** (A-04): server thừa hưởng cwd của client
  đầu tiên nên mở Home từ `$HOME` từng làm mọi repo git sau đó bị giới hạn `ls` depth 2 / 100 mục.

### Tài liệu

- `verify-gate.sh` chỉ đúng chỗ đặt `AGENT_CLI_VERIFY_CMD` (inline trong lệnh hook; tệp
  `~/.config/agent-cli/env` không tới được hook) và cảnh báo ra stderr khi biến rỗng (C-02).
- `docs/agent-cli-local-setup.md` và `docs/hooks/README.md` cập nhật theo hành vi mới.
- Thêm báo cáo audit và plan sửa; thêm `CHANGELOG.md`.

### Chưa làm (đã biết, có trong audit)

- Tắt một workspace giết job nền + MCP của workspace khác cùng server (A-01).
- `.crushrc` của dự án chạy như shell không hỏi (B-02, mã upstream); cần mô hình tin cậy thư mục.
- Model cấu hình mất khỏi catalog bị thay lặng lẽ và ghi vào config global (A-02); resume không báo khi không
  khôi phục được model (A-03).
- IO HTTP đồng bộ trong `Update` của TUI, không timeout (D-03).
- Footer cắt `ctrl+g more` ở terminal <162 cột (D-01); "Modified Files" chỉ đếm tool edit/write (F-02).
- Safe-list chưa nhận ra đường dẫn tương đối đi qua symlink trong dự án trỏ ra ngoài (ghi `shortcut:` ở `safe.go`).
- `git ls-remote <url>` vẫn tự duyệt: có thể đẩy dữ liệu ra mạng qua URL hoặc liệt kê repo ngoài (có từ trước).
- Danh sách an toàn là danh sách đen theo cú pháp; hướng đúng là danh sách trắng cờ cho từng lệnh.
- Stop hook chạm trần: lỗi đặt lên `RunComplete` của lượt chính và của prompt xếp hàng xả ra giữa chuỗi chạy
  lại, không đặt lên prompt xả ở lượt cuối; chưa có test cho trường hợp prompt xếp hàng.
- Sau khi cài lại binary (`mv -f`), server cũ có `/proc/<pid>/exe` kết thúc bằng ` (deleted)` nên tiến trình
  mới không nhận ra nó là "cùng binary" để bảo vệ (`pkill`/`killall` theo tên vẫn bị chặn).
- Mọi mục LOW của audit.

### Kiểm chứng

- `go test ./...`: 66 gói ok, 17 không có test, 0 FAIL; cùng tập gói và trạng thái với baseline trước khi sửa.
  `go test -race` sạch cho `internal/agent`, `internal/agent/tools`, `internal/config`, `internal/shell`,
  `internal/ui/chat`.
- Soát độc lập (agent riêng, không có quyền sửa, Opus): lượt 1 bắt được 3 lỗ còn lại của safe-list (đã sửa, mỗi
  đòn thành test); lượt 2 bắt được các lỗ của guard (đã sửa bằng cách phân tích cú pháp). Bản guard sau khi viết
  lại CHƯA qua lượt soát thứ ba; bảng tấn công của reviewer đã được chuyển thành test, và guard là lớp chặn thêm
  (bash vẫn hỏi quyền cho lệnh không an toàn, trừ `run`/YOLO).
- Chưa chạy: `run --json` end-to-end với Stop hook chạm trần; build có `GOEXPERIMENT=greenteagc` như installer.

### Lưu ý nâng cấp

- Một số lệnh chỉ-đọc từng tự duyệt nay hỏi quyền (đối số ngoài workspace, `printenv`, `ps`, `git config`).
  Đó là chủ ý; cấp quyền theo phiên nếu cần.

## [0.1.0] - 2026-10-16

Mốc cơ sở, gắn vào commit `87233bc8`: trạng thái fork cá nhân của [Crush](https://github.com/charmbracelet/crush)
sau Phase 1 (45 commit từ điểm fork `a73e1bfa`, 14–16/10/2026). Tóm tắt từ `git log`; chi tiết ở
`DECISIONS.md` và `docs/reviews/`.

### Thêm

- Đổi tên thành agent-cli, logo và nhãn riêng, tắt kiểm tra cập nhật mặc định, bản cài cục bộ có sao lưu và lệnh
  quay lại (`scripts/install-agent-cli.sh`).
- Nhiều session và nhiều dự án trong một terminal: Home, server dùng chung theo dự án, run sống khi đóng TUI,
  khôi phục tool call bỏ dở, `run --json` / `send --wait` / `ps` / `jobs` / `logs` / `session` cho script.
- Phím Alt: `Alt+W/S/N/M/P/E/J`, `Alt+.` / `Alt+,`, `F2` đổi tên session, bảng Jobs, chọn provider rồi model.
- Ranh giới provider/model (`AGENT_CLI_ALLOWED_MODELS`), giao việc Claude Code/Codex tắt mặc định và hỏi quyền
  mỗi lần.
- Bảo vệ tiến trình điều khiển (không tự giết agent-cli, không tắt máy) ở lớp thực thi shell.
- Stop hook "cổng nghiệm thu" tuỳ chọn (`docs/hooks/examples/verify-gate.sh`).
- Theme deep-charcoal, chữ và accent sáng hơn; tool output dùng hết bề ngang chat.

### Sửa

- Không gửi JSON tool call hỏng cho provider; tiếp tục khi auto-summarize lỗi; nối lại câu trả lời bị cắt ở giới
  hạn output; giữ lịch sử và yêu cầu thật khi tiếp tục lượt.
- Không để tiêu đề LLM ghi đè tên đặt tay.

[Unreleased]: https://github.com/luannt2002/agent-cli/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/luannt2002/agent-cli/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/luannt2002/agent-cli/releases/tag/v0.1.0
