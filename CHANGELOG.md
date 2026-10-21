# Changelog

Theo [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) và [SemVer](https://semver.org/lang/vi/).
Mỗi bản có một tag git có chú thích `vX.Y.Z`. `dew --version` hiện đúng số đó khi build từ commit
được gắn tag (Go gắn version từ tag, không cần `-ldflags`; build từ commit sau tag sẽ hiện dạng
`vX.Y.Z-0.<ngày>-<hash>`, cây bẩn có đuôi `+dirty`).

Quy trình phát hành: xem [CLAUDE.md](CLAUDE.md), mục P6.

## [Unreleased]

## [0.3.1] - 2026-10-21

Đợt tiếp theo sau một ngày dùng v0.3.0 với routine chạy song song: thấy rõ hơn, đo được độ trễ, xuất patch để soát.

### Thêm

- **`dew routine stats [NAME] [--since D] [--json]`:** đọc journal các lượt và in theo từng routine: số lượt, số request
  LLM, thời gian tới byte đầu (TTFT) và tổng thời gian p50/p95, số lần stream im/chạy lại/hết hạn mức, lỗi khác, token
  vào/ra, và số routine khác chạy cùng lúc với mỗi request (p50/max).
  - Journal có sự kiện mới `llm_request` cho mỗi request LLM streaming: `ttft_ms`, `total_ms`, `prompt_tokens` (gồm
    phần cache), `completion_tokens`, `retry`, `status`, `http_status`.
  - Chỉ đo được ở đường in-process (routine luôn chạy đường này) và khi `request_timeout` > 0. Journal trước v0.3.1
    không có số đo.
- **`dew routine export NAME [--run STAMP] [-o FILE] [--include-ignored]`:** một patch so với `base_sha`, gồm commit,
  thay đổi chưa commit, tệp mới chưa theo dõi và tệp nhị phân. `git apply` được vào checkout sạch của base; in danh
  sách tệp và sha256. Không đụng index hay object store của repo. Từ chối khi worktree đã bị gỡ (chỉ chỗ `crush.db`).
- **`DEW_ROUTINE_MIN_MEM_MB`** (mặc định 0 = tắt): RAM trống (MemAvailable) dưới ngưỡng thì `routine run` thoát 75.
  Chỉ đo trên Linux.
- **`routine list` đầy đủ** (text và `--json`, chỉ thêm trường): `state` (RUNNING chỉ khi tiến trình còn sống; ABANDONED,
  UNKNOWN, UNREADABLE), `model`, `duration_seconds`, `verify` (PASS chỉ khi GREEN/NO_CHANGE; PASS_NOT_GREEN; FAIL;
  NOT_RUN), `changed_count`, `attempts`, `pid`. `dew ps` hiện model của routine.
- **Home:**
  - Routine đang chạy hiện "● running" bằng màu cảnh báo, không còn trùng sắc xanh với green.
  - Tiêu đề "Routines (5/39)" khi còn routine bị ẩn; xem tất cả bằng `dew routine list`.
  - Dòng routine có model. Dòng dự án hiện "· N routines running" khi repo đó có lượt đang chạy.

### Thay đổi

- **`DEW_ROUTINE_SLOTS` mặc định 2** (trước là 3). Đo ngày 21/10: chạy từ 4 lượt song song trở lên thì 0/14 GREEN.
- **`routine run` tự đóng lượt "ma" của routine khác:** lượt mới nhất còn ghi RUNNING mà pid đã chết thì chuyển thành
  ABANDONED (chỉ trên Linux; báo cáo không có pid để lại cho `dew routine gc`).
- **Báo cáo RUNNING có `model` ngay từ lần ghi đầu**, và cập nhật thành provider/model sau lượt agent đầu.
- **Chỉ còn một phép kiểm tiến trình sống** (`internal/routine.RunAlive`) cho `ps`, `routine list`, GC và Home.

### Sửa lỗi

- **Stream im sau byte đầu không còn làm treo phân loại lỗi.** `net/http` trả về chính lỗi hết hạn chờ làm cause, khiến
  lỗi tự trỏ vào chính nó và `errors.Is/As` chạy mãi. Nay cause là `context.DeadlineExceeded`. Có thể đây là lý do một
  phần các lượt "agent turn timed out after 45m0s" (mới nghi).
- **Installer: repo đổi tên agent-cli sang dew không còn biến mất khỏi Home.** Mục `projects.json` của máy này trỏ đường
  `.../agent-cli` (đã không còn) được viết lại sang `.../dew` (path, data_dir, locations), có sao lưu
  `projects.before-repo-rename.json` và lệnh quay lại. Mục của máy khác trong registry dùng chung được giữ nguyên.

## [0.3.0] - 2026-10-21

Bản đầu tiên mang tên **DEW** (Daily Engineering Workspace, trước là agent-cli), kèm đợt sửa sau một ngày dùng thật với
routine chạy song song trên gateway LLM nội bộ ([audit 21/10](reports/DEW_AUDIT_2026-10-21.md)). Điểm chính:

- Vòng `loop`/`routine`; routine báo cáo đúng sự thật (không còn `verify_exit_code=0` giả, không còn lượt "ma"), tự dọn
  worktree, có trần song song, hiện trong `dew ps` và Home.
- Lượt bị lỗi mạng hay provider tạm thời thì chạy lại, không mất cả run. Sửa lỗi `dew loop` qua server chết
  `workspace not found` khi verify chậm.
- `dew clean`: rác chia ba hạng; cache build Go có trần dung lượng; gác chỗ trống đĩa; chặn agent không người trực
  xoá dữ liệu.

Chi tiết từng mục ở dưới.

### Thêm

- **Lý do dừng mới `STREAM_STALL`:** stream của LLM im (hết `request_timeout`) sau khi đã chạy lại hết số lần cho phép.
- **Lý do dừng mới `QUOTA`:** provider báo hết hạn mức hoặc hết tiền. Gồm HTTP 402, hoặc 429/403/lỗi giữa stream mà lời
  provider có "usage limit", "exceeded your current quota", "insufficient_quota" hay "insufficient balance". Không áp cho
  giới hạn theo phút ("per minute", "rate limit"); chữ "quota"/"billing" đứng một mình không đủ. Không chạy lại;
  detail có lời provider và mã HTTP; thoát 1. Thư viện provider (fantasy) vẫn tự thử lại 429 vài lần trước khi dew thấy lỗi.
- **Số lần chạy lại liên tiếp cấu hình được:** trường `stream_retries` của routine và cờ `dew loop --stream-retries`
  (0–5, mặc định 2, 0 = không chạy lại; sai giá trị thì thoát 2).

- **`dew clean [--apply] [--json]`: dew biết rác nào xoá được.** Bảng ba hạng, kèm dung lượng đo thật và lệnh tương ứng.
  - **SAFE** (`--apply` xoá):
    - Cache build Go vượt `DEW_GOCACHE_MAX`: trim về 80% trần, mục dùng lâu nhất trước. Không xoá mục dùng trong
      2 giờ qua (Go chỉ cập nhật mtime khi mục cũ hơn 1 giờ, còn build giữ đường dẫn tệp đã tra để đọc sau) hay mục
      được dùng sau lúc quét (xem lại mtime ngay trước khi xoá); không đụng tệp lạ hay symlink, và chỉ trim khi
      `README` đúng là của Go. Trim không làm hỏng cache; một build tra đúng mục đó trong khoảng micro giây giữa lúc
      xem lại và lúc xoá có thể hỏng một lần.
    - Worktree routine không còn việc (quy tắc của `routine gc`, không `--include-ignored`).
  - **ASK** (chỉ in dung lượng và lệnh): cache npm, kho pnpm, cache pip, Docker image và build cache.
  - **NEVER** (chỉ báo lý do): module cache Go (build offline cần), Docker volume (dữ liệu), tệp thuộc user khác
    trong worktree routine.
  - Không `--apply` thì không xoá gì. Bảng in chỗ trống của đĩa chứa dữ liệu dew (`--json`: `free_bytes`). Đĩa dưới
    `DEW_MIN_FREE_GB` mà cache Go còn dưới trần thì có thêm dòng ASK `go clean -cache`.
  - Lý do (đo 21/10): cache build Go lên 68 GB, tăng 10–20 GB/ngày khi agent sửa và test liên tục. Go chỉ dọn mục cũ
    hơn 5 ngày, không có trần dung lượng. Thí nghiệm cho thấy đổi đường dẫn worktree chỉ thêm 17/805 MB, nên `-trimpath`
    không đáng làm.
- **Bước dọn đầu `routine run`.**
  - Cache build Go lớn hơn `DEW_GOCACHE_MAX` GB (mặc định 20, `0` = không trần) bị trim về 80% trần, tối đa mỗi giờ
    một lần, dưới khoá chung, in một dòng stderr.
  - Sau đó, nếu đĩa chứa dữ liệu dew còn ít hơn `DEW_MIN_FREE_GB` (mặc định 10, `0` = không kiểm) thì không chạy:
    thoát 75, lời báo nêu `dew clean`. Không đo được chỗ trống thì chỉ cảnh báo.
  - Hai biến đọc như `DEW_ROUTINE_SLOTS`: biến export trước, rồi tệp env. Giá trị là số GB nguyên (10^9 byte); sai
    thì thoát 2.
- **Chặn thêm khi không người trực:** `go clean -modcache`, kể cả khi bật qua `GOFLAGS` (gán đầu lệnh, `export`, `env`,
  `sh -c`, `go env -w`); `docker volume rm|prune`, `docker system prune`, `docker compose down -v/--volumes`,
  `docker rm -v`, `docker compose rm -v` (cả `docker-compose`). Đây là dữ liệu, hoặc thứ build offline cần, không dựng lại
  được. `go clean -cache`, `docker ps`, `docker compose down` vẫn qua.
- **`routine gc` giữ worktree có thứ bạn không xoá được**: tệp thuộc user khác (vd build trong Docker chạy bằng
  root) hay thư mục chỉ đọc của chính bạn (vd module cache Go), thay vì để `git worktree remove` hỏng giữa chừng:
  git đã xoá `.git` và bỏ đăng ký worktree trước khi báo lỗi.
  - Dòng `keep` (cả `--dry-run` và `dew clean`) nêu chủ tệp và cách xử lý: `sudo chown -R "$(id -u):$(id -g)" <worktree>`
    hoặc `chmod -R u+w <worktree>`, rồi chạy lại `dew routine gc`.
  - Build Docker nên chạy với `--user "$(id -u):$(id -g)"`.

- **`dew loop --verify "<lệnh>" "<việc>"`**: chạy agent, rồi lệnh verify, sửa tiếp cho tới khi verify
  thoát 0 hoặc gặp điều kiện dừng. "Xong" là mã thoát của `--verify`, không phải lời agent. Mỗi lượt là một
  session mới với prompt tự chứa đủ (đề bài + đuôi output verify vừa đo, kèm lệnh cấm sửa test/cổng để qua).
  Lý do dừng: `GREEN`, `NO_CHANGE` (verify xanh từ trước mọi thay đổi và cây làm việc không đổi: không có gì
  chứng minh việc của agent), `MAX_REPAIRS` (mặc định 2 lượt sửa, tổng 3 lượt, không có lần vá thứ ba), `STALLED` (cùng
  chữ ký lỗi và cây làm việc không đổi sau một lượt sửa), `TIMEOUT` (`--max-duration`, kiểm trước mỗi lượt),
  `RUN_ERROR` (lượt agent lỗi không tạm thời, lỗi tạm thời còn sau 2 lần chạy lại, hoặc quá `--run-timeout`), `CANCELLED` (Ctrl+C/SIGTERM), `UNMEASURED` (verify
  không chạy được: không tìm thấy lệnh, lỗi cú pháp, quá hạn; không đốt lượt agent nào). Verify đã xanh sẵn
  trước mọi thay đổi thì ghi cảnh báo "cổng có thể không đo đúng mục tiêu" nhưng vẫn chạy.
  Verify chạy với `set -o pipefail`: `go test ./... | tee log` không che được lỗi; exit 141 (SIGPIPE, thường do
  `| head`, `| grep -q`) vẫn là FAIL nhưng output có gợi ý. Verify xanh từ baseline VÀ cây không đổi thì dừng
  `NO_CHANGE` (thoát 1); chỉ một trong hai thì vẫn `GREEN`, dòng dừng và `detail` của `--json` ghi cảnh báo.
  Journal JSON Lines (`<data>/loops/<tên>-<thời điểm>.jsonl`, 0600) được ghi và `Sync` TRƯỚC mỗi bước.
  Mã thoát: 0 `GREEN`, 1 mọi lý do dừng khác (kể cả `NO_CHANGE`), 2 dùng sai. `--json` in một object ở cuối. Không bao giờ commit,
  push hay merge. Cùng mô hình quyền như `run` (tự duyệt tool): dành cho việc bạn khởi chạy và trông.
- **`dew routine new|list|show|run|schedule|gc`**: routine là định nghĩa đã lưu (đích, lệnh verify, repo, giới
  hạn, lịch tuỳ chọn) chạy bằng engine của `loop`. Tệp `~/.config/dew/routines/<tên>.json` (0600, nằm ở cấu
  hình của bạn, không nằm trong repo nên repo lạ không tự thêm được việc chạy theo lịch). Hai mode: `maintenance`
  (cơ học, kiểm được, vô tận: được lên lịch) và `goal` (có đích như làm tính năng: chỉ chạy tay; `--scheduled`
  bị từ chối). Định nghĩa bắt buộc có `verify` ("xong" là lệnh nào), từ chối trường lạ, `max_repairs` 0–5.
  `routine run` mỗi lần (từ chối `--cwd`): giữ khoá (lượt thứ hai thoát **75**, không có khoá mồ côi); làm việc trong **git worktree
  riêng** `routine/<tên>-<thời điểm>` ngoài repo (checkout của bạn không bị đụng; worktree và nhánh được giữ lại để
  bạn duyệt, kèm lệnh xem diff, cho tới khi được dọn: xem mục Thay đổi); chạy agent in-process với **chế độ không người trực** bật; ghi báo
  cáo 0600 ngoài worktree (sha định nghĩa, nhánh, base, lý do dừng, verify cuối, journal). Không commit trừ khi
  định nghĩa có `"commit": true`; không bao giờ push hay merge. `--expect-sha` từ chối chạy khi tệp định nghĩa đã
  đổi so với lúc cài lịch. Mã thoát: 0 GREEN (và `NO_CHANGE` của routine `maintenance`: không có việc), 1 khác (kể
  cả `NO_CHANGE` của routine `goal`), 2 dùng sai, 75 đang chạy hoặc hết chỗ (`DEW_ROUTINE_SLOTS`).
  `routine schedule NAME --print` chỉ IN unit systemd `--user` (`Persistent=true`, `TZ`/`PATH` tường minh,
  `TimeoutStartSec`) và/hoặc dòng crontab, kèm lệnh cài để bạn tự chạy; dew không bao giờ sửa timer hay crontab.
  Lịch in ra mang theo biến của launcher và `DEW_ENV_FILE` (bản đầu thiếu chúng nên không chạy được, xem Sửa
  lỗi). Allowlist provider KHÔNG được chép vào lịch: lượt theo lịch đọc nó từ tệp env, nên siết allowlist trong tệp
  env là có hiệu lực ngay; allowlist chỉ đặt ở shell thì `schedule --print` in cảnh báo ra stderr.
  Báo cáo từng lượt: `RUNNING` ghi ngay khi bắt đầu; lượt chết giữa chừng hiện `ABANDONED`; hỏng trước khi agent chạy
  (vd không tạo được worktree) hiện `ERROR` kèm nguyên nhân; `routine show` liệt kê mọi lượt (`runs`). Báo cáo có
  `changed_files`, `touched_tests`, `touched_gate` (GREEN mà agent đã đổi test hay tệp mà verify gọi thì stderr cảnh báo),
  `baseline_pass`, `unchanged_tree`.
- **Chế độ không người trực** (`DEW_UNATTENDED=1`; `routine run` tự bật và tự xuất biến này cho mọi tiến
  trình con rồi trả lại): lớp chặn CỨNG ở trình thực thi shell, trước cả quyền đã cấp và YOLO. Đây là danh sách
  đen cộng một danh sách TRẮNG cho git, KHÔNG phải sandbox. Cụ thể:
  - `git` chỉ cho: lệnh đọc (`status`, `diff`, `log`, `show`, `blame`, `grep`, `ls-files`, `rev-parse`...),
    `add/rm/mv/apply`, `reset` không `--hard`, `branch`/`tag` không xoá/đổi tên, `checkout -b`/`switch -c`,
    `restore --staged`, `stash list|show`, `remote -v|show|get-url`, `worktree list`, `config` chỉ đọc.
    Lệnh con lạ, rỗng hoặc chưa mở rộng bị chặn (nên alias `git -c alias.p=push p`, `git config alias.p push`,
    cờ toàn cục lạ như `--attr-source`, `git-push`, `send-pack`, `xargs git`, `bash -c 'git "$1"' _ push` đều bị
    chặn); `-c`/`--config-env` bị cấm. `git commit`, `am`, `cherry-pick`, `revert` chỉ mở khi định nghĩa có
    `"commit": true` (`DEW_UNATTENDED_COMMIT=1`); không mở thứ gì khác.
  - `rm` có đích tuyệt đối, `~`, `$VAR`, `..` hoặc không có đích (tham số từ `xargs`) bị chặn, đệ quy hay không;
    `rsync`, `sftp`, `gh`, `glab`, `socat`, `ncat` bị chặn (cùng `ssh`, `scp`, `curl`, `sudo`, `crontab`,
    `systemctl` đã bị cấm sẵn). Bóc lớp bọc: `env`, `timeout -s`, `taskset -c 0,2`, `flock FILE`, `strace`,
    `systemd-run --user`, `nice`, `chrt`, `xargs`... và script trong `sh -c`/`eval` (vòng lặp, `$(...)`).
  - Tool bash từ chối `working_dir` ngoài workspace; tool giao việc `claude_code`/`codex` không được đăng ký;
    `dew loop`/`routine` gọi từ bash bị chặn như `run`/`send`/`server`.
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
- **Sổ ghi lời gọi LLM (opt-in): `DEW_LLM_LEDGER=<tệp>`.** Mỗi lời gọi LLM, kể cả lần thử lại, thêm một dòng JSON:
  vai trò (`main`, `title`, `summary`, `subagent`, `agentic_fetch`), provider và model thật đã gọi, kích thước prompt,
  hash system prompt, tên tool, token của chính lời gọi đó, lý do kết thúc, loại lỗi (chỉ loại và mã HTTP). Không chép
  nội dung prompt, câu trả lời hay văn bản lỗi; tệp mode 600. Biến tin cậy: chỉ đọc từ env gốc, `.env` của repo không
  bật hay chuyển được sổ. Dùng để kiểm "phiên chỉ dùng provider X" và để đếm token thật (bộ đếm của session chỉ giữ bước cuối).
- **`run --json` có `provider` và `model`** của câu trả lời cuối (đọc từ tin đã lưu, không phải model đang chọn).
- **`scripts/eval/`: đo và cổng hồi quy bằng LLM giả.** `fakellm.py` (server tương thích OpenAI, ghi mỗi request:
  provider, model, vai trò, kích thước); `faults.sh` (tiêm lỗi provider, ca model đang chọn thiếu key); `qwen-only.sh`
  (đi qua mọi đường gọi LLM với model nhỏ cố ý ở provider khác); `workspaces.sh` (nhiều workspace, nhiều session, chế độ local);
  `ledger.sh` (sổ ghi LLM ghi lỗi: `run --json` có `ledger_error`, lượt vẫn xong, local lẫn qua server);
  `session-model.sh` (`run -s` không đổi model của dự án, local lẫn qua server);
  `fixtures.py`/`run.sh`/`score.py` (bộ kịch bản đo trước/sau, chạy bằng cấu hình provider thật của bạn nên tốn hạn mức). `faults.sh`, `qwen-only.sh`,
  `workspaces.sh`, `ledger.sh`, `session-model.sh` thoát 1 khi hỏng.

### Thay đổi

- **Lượt agent quá `run_timeout`/`--run-timeout` dừng `TIMEOUT`, không còn `RUN_ERROR`.** Detail ghi "(per-turn run
  timeout)" để phân biệt với `max_duration`.
- **Run bị cắt vẫn đo phần đã làm.** Run hết hạn lượt, hoặc gặp lỗi tạm thời đã hết lần chạy lại, vẫn chạy verify MỘT
  lần trên cây agent để lại và ghi `verify_status`, `verify_exit_code` thật cùng `changed_files`.
  - Run bị cắt không bao giờ GREEN, kể cả khi verify xanh. Detail ghi "verify PASS after the agent was cut off — review
    before trusting", và Home không hiện PASS cho `RUN_ERROR`/`TIMEOUT`/`STREAM_STALL`/`QUOTA`.
  - Lỗi 5xx hết lần chạy lại vẫn là `RUN_ERROR`.
- **`routine new` không có `--model` ghi chết model large mặc định hiện tại** (`provider/model`) vào định nghĩa và in ra
  model nào. Đổi mặc định sau đó không đổi routine. Không phân giải được thì vẫn cảnh báo như cũ và để trống.
  - Model ghi chết trùng model mặc định lúc chạy thì lượt chạy dùng nguyên cấu hình mặc định (max_tokens, temperature,
    reasoning_effort, model small).
  - Nếu sau đó đổi model mặc định, routine chạy model đã ghi chết như `--model`: chỉ model đó, với tuỳ chọn mặc định
    của provider. Report và stderr luôn nêu model nào chạy.
- **Routine tự dọn worktree cũ, có `dew routine gc`, giới hạn số lượt chạy cùng lúc và hiện trong `dew ps`.** Trước
  đây worktree của mọi lượt tích luỹ mãi (máy dùng thật: 28 worktree, 3,4 GB sau khoảng 1,5 ngày) và routine không
  hiện ở đâu cả.
  - Mỗi `routine run`, sau khi giữ khoá, gỡ worktree và nhánh của các lượt cũ đã xong của routine đó, giữ worktree
    của lượt mới nhất. Chỉ gỡ khi: không có thay đổi chưa commit hay tệp chưa theo dõi (kiểm bằng
    `--untracked-files=all`, bất kể `status.showUntrackedFiles`); không có commit ngoài base, kể cả trên HEAD tách
    rời; không có tệp bị gitignore ngoài `.crush/`, trừ thư mục dựng lại được ở bất kỳ cấp nào (`node_modules`, `dist`, `build`, `coverage`, `.venv`, `venv`, `__pycache__`, `.pytest_cache`, `.mypy_cache`, `.ruff_cache`, `.tox`, `target`, `.next`, `.nuxt`, `.turbo`, `.gradle`, `.parcel-cache`: xoá cùng worktree); không có tệp bị giấu khỏi `git status`
    (assume-unchanged, skip-worktree có tệp trên đĩa); `crush.db*` không được ghi sau `finished` + 1 phút (có người mở
    lại worktree); báo cáo trỏ đúng `<data>/routines/<tên>/wt/<stamp>` và nhánh `routine/<tên>-<stamp>`; lần dọn
    trước không lỗi (`gc_error`). Trước khi gỡ, `crush.db` (cùng `-wal`, `-shm`) và `.crush/logs` được chuyển ra cạnh
    `report.json` (`session_db`, `session_logs`); chuyển hỏng thì giữ worktree. Nhánh mà `branch -D` hỏng được thử lại ở lượt dọn sau,
    chỉ khi không có commit ngoài base.
  - `dew routine gc [NAME] [--dry-run] [--include-ignored]` dọn tay theo cùng quy tắc, giữ khoá từng routine (đang
    chạy thì `skip`). `--dry-run` in từng worktree sẽ gỡ hay giữ, kèm lý do và dung lượng. `--include-ignored` cho
    gỡ cả worktree có tệp bị gitignore.
  - `DEW_ROUTINE_SLOTS` (mặc định 3, `0` là không giới hạn): tổng số `routine run` chạy cùng lúc; hết chỗ thì thoát
    75. Đọc biến export trước, rồi tệp env của dew; `.env` của repo không đổi được nó. Giá trị sai thì thoát 2.
  - `dew ps` có hàng `ROUTINE` cho mỗi lượt đang chạy (pid còn sống và đúng là `routine run <tên>`), path là worktree,
    `--json` thêm `routine`, `pid`, `started`, `busy: true`; in cả khi không có server nào. Windows không liệt kê;
    macOS không nhận ra PID bị tái dùng.
- **`loop`/`routine` chạy lại lượt khi provider lỗi tạm thời.** Lỗi 5xx (kể cả 524), 429, hết hạn chờ stream
  (`LLM stream received no data…`) hay mất kết nối (reset/refused/EOF) không còn kết thúc cả run bằng `RUN_ERROR`:
  loop chạy lại đúng đề bài của lượt đó trong cùng session (kèm một dòng báo lượt trước bị provider cắt), tối đa 2 lần
  liên tiếp, chờ 30 s rồi 60 s. Ctrl+C trong lúc chờ là `CANCELLED` ngay; lần chạy lại sẽ bắt đầu sau `--max-duration`
  thì dừng `TIMEOUT`. Lượt chạy lại không tốn lượt sửa; `attempts` (JSON, báo cáo routine, sự kiện `stop` của journal)
  đếm mọi lượt agent, kể cả lượt chạy lại. Lỗi không tạm thời (401/403, tràn context, model không được phép, lỗi của
  tool, lỗi lạ) dừng như cũ. Journal ghi `error_class` và `http_status` mỗi lần. Qua server lỗi chỉ còn dạng chữ nên
  chỉ nhận các mẫu đã biết; mẫu lạ dừng như cũ (`internal/agent/transient.go`, `internal/loop/loop.go`).
- **`dew loop --json`: `verify_exit_code` chỉ có khi verify đã chạy sau một lượt agent.** Trước đây in `0` cả khi
  verify chưa chạy, đọc như đạt.
- **Sổ dự án không nhận worktree của routine.** Worktree dưới `<data>/routines/` không còn được đăng ký làm dự án, và
  các mục đã lọt vào từ trước bị ẩn khỏi Home và `dew projects` (`projects.json` không bị ghi lại). Trước đây mỗi lượt
  routine thêm một mục, nên 5 ô Recent của Home toàn là thư mục dấu thời gian (`internal/projects/projects.go`).
  Chưa làm: `dew stats --all` gom số liệu theo sổ dự án nên không tính các lượt routine mới.
- **Help nói DEW là gì.** `dew --help` và `dew run --help` mô tả Daily Engineering Workspace, ví dụ thật thay cho câu
  giới thiệu và ví dụ của upstream. Tên client khi đăng ký OAuth với MCP server là `DEW`. Giữ nguyên có chủ ý: socket
  mặc định khi không qua launcher (`crush-<uid>.sock`) và User-Agent `crush/1.0` của tool `fetch`/`download`/
  `sourcegraph` (cassette VCR của `TestCoderAgent` khớp request theo header này).

- **Đổi tên agent-cli → DEW (Daily Engineering Workspace).** Lệnh `dew`, `dew-tmux` (thay `agent-cli`,
  `agent-cli-tmux`); binary ở `~/.local/lib/dew/`; cấu hình, dữ liệu, cache ở `~/.config/dew/`, `~/.local/share/dew/`,
  `~/.cache/dew/`; biến `DEW_*` thay `AGENT_CLI_*` (cả `DEW_ALLOWED_MODELS`, `DEW_DELEGATION`, `DEW_CLAUDE_*`,
  `DEW_CODEX_*`); socket `dew-<hash>.sock`; unit routine in ra là `dew-routine-<tên>`. Script: `scripts/install-dew.sh`,
  `scripts/dew-tmux.sh`, `scripts/test-dew-*.sh`; mẫu `dew.crushrc.example`, `dew.env.example`; docs
  `docs/dew-local-setup.md`. Không đọc song song tên cũ: biến tin cậy tên cũ (`AGENT_CLI_ALLOWED_MODELS`,
  `AGENT_CLI_CLAUDE_PERMISSION_MODE`…) còn trong env gốc hay tệp env thì dew dừng với lỗi nêu cả hai tên, thay vì lặng
  lẽ nới ranh giới; `~/.config/agent-cli/env` còn sót (khi không đặt `DEW_ENV_FILE`) cũng vậy. Client kiểm chính sách
  này trước khi dựng server. `docs/hooks/examples/verify-gate.sh` chặn khi chỉ còn `AGENT_CLI_VERIFY_CMD`. Bash của
  agent không gọi được `dew-tmux` (như `dew run` lồng). Giữ: `crush.json`, `CRUSH_*`, `.crush/`, giá trị provider
  `agent-cli-pending` đã lưu trong DB.

  **Lưu ý nâng cấp:**
  - Thoát mọi client `agent-cli` (TUI, `run`, `loop`) và chờ server cũ tự tắt (khoảng một phút) trước khi cài: trình
    cài dừng, chưa đổi gì, khi còn tiến trình chạy binary trong `~/.local/lib/agent-cli/`. Server cũ đang chạy không
    được `dew ps` liệt kê (socket tên cũ) và không dùng chung với client `dew`.
  - Chạy `bash scripts/install-dew.sh` để chuyển: bốn thư mục `…/agent-cli` sang `…/dew` (cả hai cùng có thì không
    gộp, cảnh báo), đổi `AGENT_CLI_` thành `DEW_` và đường `…/agent-cli` thành `…/dew` trong `env`, `crushrc`,
    `crush.json` và `projects.json` (sao lưu trước ở `~/.config/dew/pre-dew-<thời điểm>/`), nối lại worktree routine,
    gỡ `agent-cli`/`agent-cli-tmux` do trình cài cũ tạo, in lệnh quay lại. Build chạy trước mọi chuyển đổi; lệnh quay lại
    được ghi dần vào `pre-dew-<thời điểm>/undo.sh` và in ra khi trình cài dừng giữa chừng.
  - Symlink trỏ vào thư mục clone cũ (vd `~/.config/agent-cli/crush.json` → `<clone>/.crush.json`) gãy nếu bạn đổi tên
    thư mục đó; trình cài cảnh báo symlink gãy nhưng không tự trỏ lại (xem `docs/dew-sync.md` mục 3).
  - Biến `AGENT_CLI_*` bạn tự export ở shell (`~/.bashrc`…) phải đổi tay; trình cài liệt kê TÊN biến và tên các tệp
    khởi động shell còn nhắc tên cũ. Unit
    systemd/crontab của routine đã cài theo tên cũ: in lại bằng `dew routine schedule <tên> --print` rồi cài tay.

- **Một workspace cho mỗi dự án, ở mọi chế độ.** Mở dew từ thư mục con của repo (hay qua symlink trỏ vào đó) nay
  làm việc ở GỐC repo, cả qua launcher (server) lẫn `CRUSH_CLIENT_SERVER=0` (local): working dir của agent, lệnh `!`, git
  status trong prompt và các tool đều ở gốc, và chính tiến trình (TUI, `run`, `send`…) cũng chuyển sang gốc sau khi mở
  workspace. Trước đây working dir là thư mục bạn mở. Gốc là git toplevel vật lý (ngoài git: đường vật lý của thư mục,
  cùng quy tắc với launcher), tính bằng một hàm chung cho data dir, workspace của server, chế độ local, Home và các lệnh
  `session`, `stats`, `models`, `logs`; git không trả lời trong 5 giây thì lần mở đó coi như ngoài git (không nhớ, lần
  sau hỏi lại). Hệ quả khi mở từ thư mục con:
  - Tệp ngữ cảnh (`AGENTS.md`, `CRUSH.md`, `CLAUDE.md`…), `crush.json`/`.crush.json`/`crushrc` và skill (`.agents/skills`…)
    của riêng thư mục con không còn được nạp, chỉ của gốc; vì vậy `permissions`, hook và MCP khai trong `sub/crush.json`
    cũng không còn áp dụng.
  - `sub/.crush/crush.json` (model, effort mà bản trước ghi khi mở từ `sub`) không còn áp dụng; lựa chọn của dự án nằm ở
    `<gốc>/.crush/crush.json`.
  - `!lệnh` chạy ở gốc repo. Đường tương đối gõ trong TUI (`@` nhắc tệp, đính kèm, dán đường dẫn, completion) tính
    theo gốc repo, khớp với nơi agent đọc và sửa tệp: `@package.json` là `<gốc>/package.json`, muốn tệp của `sub` thì
    gõ `@sub/package.json`.
  - "Trong dự án" của tự duyệt quyền (`edit`/`write` của session đã cho phép) và của việc chặn ghi ra ngoài (chế độ không
    người trực) nay là cả repo, không chỉ thư mục con đã mở.
  - Qua server, workspace của dự án giữ cờ và env của client mở nó trước (gồm `.env` của thư mục client đó) cho mọi
    client sau, như trước với cùng một đường, nay áp cho cả dự án.
  - `options.data_directory` tương đối tính theo gốc dự án (trước: theo thư mục mở). `-D/--data-dir` tương đối vẫn tính
    theo thư mục mở. Qua server, workspace giữ data dir của client mở trước: client sau có `-D` khác, hoặc không có `-D`
    trong khi client đầu đã chọn bằng `-D`, nhận cảnh báo trên stderr (trước: chỉ log Debug).
  - Một `.crush` cố ý đặt ở thư mục con cũng không còn được dùng; muốn thư mục con là dự án riêng thì cho nó repo git
    riêng (hoặc worktree), hay mở bằng `-D`.
  **Lưu ý nâng cấp:**
  - `.crush` cũ ở thư mục con (bản trước tạo khi mở từ đó) không được mở nữa, KHÔNG được tự gộp hay xoá. Khi còn
    `crush.db`, mỗi lần mở dự án (từ gốc hay thư mục con) dew in cảnh báo ra stderr kèm lệnh mở lại:
    `CRUSH_CLIENT_SERVER=0 dew -c <thư mục con> -D <thư mục con>/.crush`. Lệnh đó mở chúng ở GỐC repo, trong khi
    đường tương đối trong các session ấy tính theo thư mục con: chỉ nên đọc; làm tiếp thì agent có thể sửa nhầm tệp
    cùng tên ở gốc. Nó được tìm trong sổ dự án
    (`projects.json`, trường `locations` của bản trước, nay chỉ đọc và giữ nguyên) và trên đường từ thư mục mở lên gốc;
    không quét cả cây repo, nên `.crush` mà sổ không biết và không nằm trên đường đó thì không được báo. Gộp vào DB của
    dự án là việc tay.
  - Cảnh báo in ra trước khi TUI chiếm màn hình, nên với TUI chỉ thấy sau khi thoát; `run`/`send` thấy ngay.
  - Session tạo khi mở từ thư mục con TRƯỚC bản này (kể cả session nằm trong `<gốc>/.crush`) ghi danh sách tệp đã đọc
    theo đường tương đối với thư mục con; nay đường đó được ghép theo gốc. Tool `write` dựa vào danh sách này để biết tệp
    đã được đọc chưa, nên tệp CÙNG TÊN ở gốc có thể bị coi là đã đọc và bị ghi đè mà agent chưa đọc nó (`--yolo` và
    `loop` không hỏi quyền). Việc ở thư mục con thì bắt đầu session mới thay vì làm tiếp session cũ.
  - `loop` mở từ thư mục con: lệnh `--verify` vẫn chạy ở thư mục bạn mở, agent làm việc ở gốc; prompt của lượt nói rõ
    verify chạy ở đâu (vd ``running `go test ./...` in `sub/` ``), và việc phát hiện "cây không đổi" (`STALLED`, `NO_CHANGE`)
    xét cả repo. Mở ở gốc thì prompt như trước.
  - Home liệt kê session của `.crush` ở gốc (DB mà TUI sẽ mở), kể cả với dự án mà lần mở cuối bằng bản trước là từ thư mục
    con; session trong `.crush` cũ không hiện, danh sách session có một dòng báo kèm lệnh mở lại (terminal hẹp thì dòng
    này bị cắt).
  **Giới hạn còn lại:** mở cùng dự án bằng launcher (server) và bằng binary trần hay `CRUSH_CLIENT_SERVER=0` (local) vẫn là
  hai tiến trình trên cùng một `crush.db` (có từ trước; local không lấy khoá data dir). `$HOME` là repo git, hoặc
  `GIT_DIR`/`GIT_WORK_TREE` đang được export, thì mọi thư mục quy về gốc đó. Workspace trên server ở máy khác (đường không
  có trên máy này) thì tiến trình không chuyển được sang gốc, chỉ ghi log. Session của bản trước mở ở chế độ local từ
  thư mục con có danh sách tệp đã đọc tính theo thư mục con, nay được ghép theo gốc.

### Sửa lỗi

- **Report routine và `dew loop --json` không còn ghi `verify_exit_code: 0` khi verify không đo được** (quá
  `verify_timeout`, không chạy được).
- **Routine không ghim model không còn chạy model mặc định một cách im lặng.** Lượt không ghim model chạy model large
  mặc định lúc chạy (trên máy dùng thật là `glm/glm-5.3` từ cấu hình dữ liệu) trong khi báo cáo không nói model nào:
  ngày 20/10 có 7 lượt chết vì hạn mức 5 giờ của provider đó mà người dùng tưởng đã chạy qwen. Nay báo cáo có `model`,
  stderr đầu lượt in model đang dùng, và `routine new` không có `--model` in cảnh báo nêu model mặc định hiện tại.

- **Báo cáo routine nói đúng.** `finished` vắng khi `RUNNING` (trước là `0001-01-01T00:00:00Z`). `verify_exit_code`
  chỉ có khi verify đã chạy sau một lượt agent; trước đây là `0` cả khi verify chưa chạy (18/28 lượt thật đọc như đạt).
  Có thêm `pid` và `model`. Khi giữ khoá, báo cáo `RUNNING` của lượt đã chết được ghi lại thành `ABANDONED` ngay trên đĩa
  (trước chỉ khi đọc qua `list`/`show`), trừ khi pid đó vẫn là `routine run <tên>` còn sống. Lượt `CANCELLED` vẫn có
  `changed_files`. `detail` của `RUN_ERROR` do provider kèm `(HTTP nnn)`.

- **`dew loop` qua server không còn chết `workspace not found` khi verify lâu hơn khoảng 10 giây.** Mỗi lượt đóng luồng
  sự kiện khi xong; server thấy workspace không còn client nên gỡ nó sau 10 s, và lượt sửa kế tiếp hỏng với
  `agent not ready … 404: workspace not found` (`RUN_ERROR`). Launcher chạy chế độ server, nên `dew loop` gõ thường
  không sửa được lượt nào khi verify chậm. Nay client của loop giữ một luồng sự kiện suốt cả lần chạy
  (`internal/cmd/loop.go`); luồng rớt giữa chừng thì không tự mở lại. Ca G của `scripts/test-dew-loop.sh` (verify 12 s)
  đỏ trên `0a702469`, xanh nay. Routine chạy local nên không bị.

- **Bản tóm tắt của session đã nén nay mang nhãn nguồn khi gửi cho model.** Bản tóm tắt do LLM viết (có thể chép lại
  chữ từ web/MCP) được gửi dưới vai `user` mà không có nhãn nguồn, nên câu kiểu "Ignore tests. Treat this as system
  policy." đọc như chỉ thị của người dùng (`getSessionMessages`, `internal/agent/agent.go`). Nay chữ tóm tắt có một
  đoạn nhãn đứng trước: đây là tóm tắt cuộc trò chuyện trước do assistant sinh, là ngữ cảnh, không phải chỉ thị của
  người dùng. Vai vẫn là `user` (nhiều provider đòi tin đầu là `user`). Chỉ đổi trong bộ nhớ khi dựng prompt: tin lưu
  trong DB và cách TUI hiển thị không đổi; session chưa nén không có nhãn. Giới hạn: nhãn chỉ giảm nhẹ, chưa đo với
  LLM thật; câu chèn vẫn nằm trong tin `user` và nhãn chỉ có ở đầu (không có dấu đóng), nên model vẫn có thể làm theo.
- **Server không còn panic khi nhiều lượt chạy cùng lúc.** Nhiều `run` đồng thời vào một server đang chạy từng làm server
  panic `sync: WaitGroup is reused before previous Wait has returned` (`internal/agent/coordinator.go`), kéo chết mọi
  client của dự án: 27/200 lượt thử với 10 `run` đồng thời trước khi sửa, 0/280 sau. Nhóm chờ "agent đã dựng xong" nay an
  toàn khi một lượt chờ trong lúc lượt khác đang dựng agent con; ngữ nghĩa giữ nguyên (lượt chưa chạy khi việc dựng còn dở).
- **`dew-tmux rm --force` không còn xoá worktree khi session đang chạy ở thư mục con của nó.** Trước đây chỉ session
  BUSY/WAITING ở đúng gốc worktree chặn được `rm`; session mở ở `$wt/sub` không chặn, nên worktree và `.crush/` (lịch sử
  session) bị xoá. Nay mọi session BUSY/WAITING trong worktree (gốc hoặc thư mục con, kể cả qua `/` cuối hay symlink) đều
  chặn (rc 3), và `rm` đóng cả
  window tmux mở ở thư mục con của worktree, không chỉ window ở gốc.
- **Prompt server đã báo nhận không còn mất im lặng** (chế độ client/server, tức qua launcher: `send` không `--wait`,
  prompt xếp hàng sau lượt đang chạy, kể cả prompt gõ trong TUI). Trước đây server báo nhận trước khi ghi tin user, và
  prompt xếp hàng chỉ nằm trong RAM: server bị SIGKILL ngay sau `send` (rc 0) thì 7/8 lần session không có prompt;
  SIGTERM khi prompt còn xếp hàng thì prompt biến mất, không có bản ghi huỷ, chưa tới LLM (binary thật, LLM giả; nay có test đơn vị
  `TestSendMessage_AckPersistsUserMessageBeforeRun` và `TestSendMessage_QueuedPromptSurvivesShutdown` trong
  `internal/backend/durable_ack_test.go`). Nay server ghi tin user cùng một lời trả lời giữ chỗ TRƯỚC khi báo nhận; lượt chạy
  tiếp quản chúng (không ghi tin user lần hai, kể cả khi prompt được gộp vào lượt đang chạy). Prompt không chạy được
  vẫn hiện trong session với đúng trạng thái: huỷ (Esc, Ctrl+C trên `run`, xoá hàng đợi, server tắt), lỗi thật (vd
  provider thiếu key hay bị tắt), hoặc "Interrupted" khi server chết (ghi lúc mở lại session, cơ chế settle sẵn có).
  Không tự chạy lại. Prompt chưa từng tới model không vào lịch sử gửi model của các lượt sau (như trước khi có bản
  sửa, khi nó biến mất), và không lọt vào request hay bản tóm tắt của lượt khác. Prompt xếp hàng nay hiện trong
  session ngay khi gửi, kèm lời trả lời đang chờ. Chế độ in-process (`CRUSH_CLIENT_SERVER=0`) không đổi. Lời trả lời
  đang chờ được lưu với provider `agent-cli-pending`: thống kê theo model và thời gian phản hồi không tính nó, và
  config báo lỗi nếu có provider tự đặt trùng ID này.
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
- **Model đang chọn không dùng được thì dew không đổi sang provider khác.** Trước đây, khi provider dựng sẵn của
  model đang chọn thiếu API key (hoặc ID model không còn), dew lặng lẽ chạy bằng model của provider khác và ghi
  lựa chọn đó xuống `~/.local/share/dew/crush.json`. Đo bằng provider giả (`scripts/eval/faults.sh`, ca
  `selected-no-key`: chọn `openai/gpt-4o` không có key): bản trước thoát 0 sau 2 request sang provider `glm`; bản này
  thoát 1, 0 request. Nay: chỉ mất ID model thì dùng model mặc định của CÙNG provider (log cảnh báo, không ghi xuống
  đĩa); provider không dùng được thì giữ nguyên lựa chọn: `run`, `loop`, `routine` dừng ngay với lỗi nêu provider/model
  và cách sửa (`run -m` không vượt qua lỗi này), còn app vẫn khởi động không agent như lúc chưa cấu hình, nên
  `dew login <nền tảng>` chạy được và TUI mở thẳng màn chọn model.
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
  `docs/dew-local-setup.md`, mục "Giới hạn đã biết".
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
  `CRUSH_GLOBAL_CONFIG`, `CRUSH_GLOBAL_DATA`, `CRUSH_CACHE_DIR`, ba biến tắt metrics/cập nhật và `DEW_ENV_FILE`,
  đọc từ env gốc (`.env` của thư mục hiện tại không chen vào được); giá trị có ký tự lạ thì lệnh từ chối.
- **Routine bị kill giữa chừng không còn bị che.** Bản đầu không để lại báo cáo, `list`/`show` hiện lượt GREEN cũ; lỗi
  trước khi agent chạy hiện như lượt bỏ dở. Nay có `RUNNING`/`ABANDONED`/`ERROR` (mục Thêm).
- **Verify của loop/routine không còn GREEN nhờ `| cat`.** `pipefail` (mục Thêm).
- **Chế độ không người trực bịt thêm lỗ.** Bản đầu để lọt: ghi tệp ra ngoài bằng tool ghi tệp (vd `~/.config/dew/crushrc`,
  chạy như Bash ở lần mở sau), `git -C <repo khác>`, option viết tắt (`--har`), biến `GIT_DIR=… git rm`. Nay đều bị chặn
  (mục Thêm).
- **Tiêu đề markdown (H1) vô hình trong theme tuỳ chỉnh.** Theme tuỳ chỉnh (vd `github-dark-dim` trong
  `~/.config/dew/themes/`) không nhận lớp "yên tĩnh" của theme dựng sẵn, nên H1 rơi về kiểu `quickStyle`
  gốc: chữ `warning_subtle` trên nền `primary`. Với `#ffa657` trên `#79c0ff` đó là 1,00:1 (đo bằng công thức
  WCAG). Nay theme tuỳ chỉnh dùng chữ `fg_base` cho H1, heading và inline code, nền `bg_less_visible`, H6 dùng
  `fg_subtle`; chỉ dùng token của chính theme nên theme sáng vẫn đúng. Theme dựng sẵn (Charmtone, Gruvbox,
  mặc định) không đổi. Phải cài lại binary mới thấy (`scripts/install-dew.sh`).
  Chưa sửa: logo gradient (`secondary → primary`) và các style khác của `quickStyle` trong theme tuỳ chỉnh.
- **Client không còn lặng lẽ chạy theo chính sách tin cậy của client khác.** Server của dự án nạp chính sách
  (`DEW_ALLOWED_MODELS`, `DEW_DELEGATION` và các biến tin cậy khác) một lần, theo client đã dựng nó;
  client mở sau với chính sách khác từng chạy theo chính sách cũ mà không cảnh báo (allowlist `qwen` của nó không
  chặn lượt đi provider `pp`). Nay client gửi chính sách tin cậy của nó (env gốc và tệp env, không theo `.env` của
  repo) và server từ chối khi khác: lỗi `trust policy differs from the running dew server` nêu tên biến
  (không giá trị) và bảo thoát mọi client của dự án. `--yolo` khác workspace đang mở cũng bị từ chối thay cho cảnh
  báo cũ, cả hai chiều, và lỗi nói dự án đang mở có hay không `--yolo` ("pass --yolo too" / "drop --yolo"); `run`, `send`, `loop` nay
  nhận `--yolo` như TUI để khớp được. Home và `dew-tmux` mở dự án không kèm `--yolo`, nên bị từ chối khi
  dự án đang giữ bởi client mở bằng `--yolo` (thoát client đó hoặc mở lại không `--yolo`). `ctrl+y`/`shift+tab`
  bật YOLO lúc chạy thì không bị so: client sau vào được, và cũng thành YOLO cho cả workspace. Client cũ không gửi chính sách cũng bị từ chối, và lệnh chạy model không tái dùng server khác bản đang bận.
  Chỉ lệnh chạy model (TUI, `run`, `send`, `loop`) bị so; `jobs`, `login`, `logout` thì không. Biến thường (kể cả
  `$VAR` của `api_key`/`base_url` trong `crush.json`) không được so và vẫn theo client đã dựng server, như trước (trừ `loop --cwd`, xem mục `.env`).
  Tiến trình con mà Home dựng và lệnh `--verify` của loop không còn nhận biến tin cậy từ `.env` của repo (gõ
  `dew` trong repo có `.env` đặt `DEW_DELEGATION=on` từng bật giao việc cho workspace mở từ Home).
- **Kết quả tool MCP dạng chữ có trần như bash và giữ cờ lỗi.** MCP trả 5 MiB từng đi nguyên vào request kế tiếp và
  `crush.db` (+5,3 MB mỗi nơi); nay cắt và spill ra `<data>/shell-output/` như bash. `isError=true` nay lưu `is_error=true`.
  Chưa sửa: chữ đi kèm ảnh/media và tool `read_mcp_resource` chưa có trần; MCP trả ảnh kèm `isError=true` vẫn lưu
  `is_error=false`.
- **Mở hay tiếp session cũ không còn tự đổi model.** Trước: `run -s/-C` qua server và TUI mở session ghi model lượt
  cuối của session vào `<dự án>/.crush/crush.json`, nên session khác và session mới lặng lẽ đổi provider; `run -s/-C`
  ở chế độ local chạy model của session (đổi trong RAM). Nay mọi đường chạy model đang chọn của dự án và không ghi gì;
  session từng chạy model khác thì báo một dòng nêu cả hai model (stderr của `run`; thanh trạng thái TUI, kèm
  `alt+p` khi khác provider, `alt+m` khi cùng provider). Model của session không còn trong catalog nay cũng được báo
  (trước chỉ ghi log Debug). Model chỉ đổi khi bạn chọn: `-m`, `--small-model`, `alt+p`/`alt+m`,
  `dew-tmux new --model`; thinking và effort (`alt+e`, `--reasoning-effort`) đổi tham số của model đang chọn.
- **`.env` của repo không còn lái lệnh `!` và CLI con giao việc.** Lệnh `!` gõ trong TUI (local lẫn server) nhận
  biến tin cậy theo env gốc: `!dew run …` từng nhận `DEW_DELEGATION`, allowlist hay tệp env từ `.env`.
  Claude Code/Codex được giao việc nhận env gốc của dew (trước `.env` và `env` của `crush.json`, kể cả
  `crush.json` toàn cục), cả khi mở qua Home và ở chế độ server: `ANTHROPIC_BASE_URL`, `ANTHROPIC_API_KEY`,
  `OPENAI_BASE_URL`, `OPENAI_API_KEY`, `CLAUDE_CONFIG_DIR`, `CODEX_HOME`… đặt trong `.env` từng tới CLI con (đổi endpoint
  hay khoá của nó); giá trị export ở shell vẫn tới. Workspace mà Home mở và server của dự án nay được dựng với env gốc
  và tự nạp `.env` như trước (`loop --cwd <dir>` ở chế độ server: chỉ `.env` của `<dir>`). DEW gọi lồng
  (`!dew run …`) vẫn chụp biến thường của `.env` làm env gốc; CLI con của nó nhận chúng khi lời gọi lồng dựng
  server mới hoặc chạy in-process, còn dự án đã có server thì theo env gốc của server đó. Home mở lại bằng Alt+W khi chạy
  binary không qua launcher cũng không còn lấy `DEW_LAUNCHER` từ `.env` (từng để `.env` chọn chương trình mà
  Home đó chạy).
- **`-c`/`--cwd` tương đối không còn đặt workspace và DB sai chỗ.** `dew -c .. run …` gọi từ `repo/internal`
  gửi nguyên chuỗi `..` cho server. Server tính chuỗi đó theo thư mục CỦA NÓ, tức thư mục của client đã dựng nó. Kết
  quả: workspace thành thư mục cha của một dự án khác, `crush.db` rơi ra ngoài cả hai repo, kể cả khi đi qua launcher.
  Ở chế độ local, `-c foo` ghi DB vào `foo/foo/foo/.crush`: chuỗi `foo` bị ghép lại sau khi đã chdir vào `foo`, rồi
  data dir còn bị ghép gốc thêm một lần. `projects.json` lưu đường tương đối. `loop -c ..` đọc cờ hai lần nên đưa
  agent (tự duyệt mọi tool) lên thư mục cha của repo trong khi verify chạy ở chỗ khác; ở chế độ local có lúc ra
  `GREEN` cho một thư mục không phải repo. Nay `-c` được đổi thành đường tuyệt đối theo thư mục hiện tại của lệnh
  trước khi dùng, một lần cho cả lệnh. Data dir chỉ ghép gốc một lần và luôn tuyệt đối. Server nhận đường tương đối
  từ client khác thì workspace và data dir vẫn tính theo thư mục của server, chỉ không còn bị ghép gốc hai lần.
  `-D/--data-dir` tương đối vẫn tính theo thư mục workspace (sau `-c`), nay cho cùng kết quả ở local và qua server.
  `-c` tuyệt đối và lệnh không có `-c` không đổi.
  **Lưu ý nâng cấp:** session đã tạo bằng `-c` tương đối nằm ở chỗ sai cũ (vd `r/foo/foo/foo/.crush`; `r/foo/foo/x`
  khi có `-D x`; `<thư mục cha>/.crush` khi qua server) vẫn còn trên đĩa nhưng bản này không mở chúng nữa và không
  cảnh báo. Cần thì mở lại bằng `-D <đường cũ>`.
- **Một dự án một `crush.db` và một workspace, mở từ đâu cũng vậy.** Data dir từng là `.crush` gần nhất tính từ thư mục
  mở. Mở `repo/sub` trước khi `repo` có `.crush` thì tạo `repo/sub/.crush`; mở `repo` sau đó lại tạo `repo/.crush`: hai
  `crush.db`, session chia đôi (gốc thấy 1/3 session, `sub` thấy 2/3; cả local lẫn server). Đi qua symlink trỏ vào
  `repo/sub` thì việc tìm đi lên theo chữ: ra `.crush` của thư mục chứa symlink, nằm ngoài repo (vd `~/code/.crush`),
  hoặc một DB thứ hai trong `sub`; nhánh git đọc qua symlink ra rỗng. Trên server của dự án, `repo` và `repo/sub` là
  hai workspace dùng chung một `crush.db` mà trạng thái bận không thấy nhau: prompt gửi từ `repo/sub` cho session đang
  chạy ở `repo` chạy song song thay vì xếp hàng, tức một session chạy hai lượt cùng lúc. Nay data dir mặc định luôn là
  `.crush` ở gốc dự án (git toplevel vật lý; ngoài git là đường vật lý của thư mục), bất kể thư mục con, symlink hay thứ
  tự mở; việc tìm `crush.json` và nhánh git đi lên theo đường vật lý; mọi chế độ dùng một workspace cho cả dự án (mục
  Thay đổi), nên prompt thứ hai xếp hàng. Data dir cũ không tự gộp: xem Lưu ý nâng cấp ở mục Thay đổi.
- **Sổ ghi LLM ghi lỗi không còn im lặng.** Bật `DEW_LLM_LEDGER` mà ghi lỗi (đĩa đầy; đo bằng `/dev/full`) thì
  trước đây lượt chạy tiếp không một dấu hiệu: sổ thiếu dòng mà vẫn trông như bằng chứng. Nay mỗi prompt bị ảnh hưởng
  (gồm các lượt tự làm tiếp của nó) có một dòng log WARN `Failed to write the LLM ledger…` (một lần mỗi prompt, không
  phải mỗi lời gọi; lời gọi đặt tiêu đề không gắn session nên mỗi lần một dòng). `run --json` (chạy local lẫn qua server)
  và `send --wait --json` có trường `ledger_error`. Lượt vẫn chạy và thoát 0: sổ là công cụ phụ, không dừng việc chính.
  Không mở được tệp sổ (vd thư mục không ghi được) thì giữ như cũ: cảnh báo một lần mỗi tiến trình, không có
  `ledger_error`. Kiểm: `scripts/eval/ledger.sh <thư-mục-mới> <binary>` (binary thật, LLM giả; thoát 1 khi thiếu
  cảnh báo hay trường).
- **Chỉ dẫn của MCP server (`<mcp-instructions>`) chỉ gửi cho lượt có đường tới server đó.** Trước đây chỉ dẫn của
  mọi MCP đang kết nối được gắn vào system prompt của mọi lượt, kể cả agent con (`agent`), `agentic_fetch` và plan,
  vốn không có tool MCP nào: model nhận lời dặn gọi tool nó không có. Nay lượt nhận chỉ dẫn của một server khi có
  ít nhất một tool của server đó, hoặc có `list_mcp_resources`/`read_mcp_resource` (hai tool này đọc được mọi
  server). Agent chính có MCP vẫn nhận đủ như cũ. Các khối chỉ dẫn nay xếp theo tên server: trước đây thứ tự ngẫu
  nhiên khi có từ 2 server, nên system prompt đổi giữa các lượt và prompt cache của provider bị trượt. Test:
  `TestMCPInstructionsFollowRunTools`, `TestRunMCPInstructionsFollowRunTools`, `TestMCPInstructionsOrderIsStable`.
- **Có hook `PreToolUse`, hai bộ lọc tool MCP vẫn nhận ra tool MCP.** Hook bọc mọi tool của agent chính trong một
  lớp, còn hai bộ lọc nhận diện tool MCP theo kiểu nên bỏ sót hết: server đã tắt bằng "Toggle MCPs" vẫn đưa tool cho
  model và model gọi được (ca: một tiến trình dew khác trên cùng repo tắt server trong khi tiến trình này còn giữ
  kết nối); lượt từ một channel thấy cả tool gửi tin của channel khác. Đo bằng binary thật, LLM giả, MCP stub và
  hook: bản trước gửi tool của server đã tắt trong 3/3 request và server nhận `tools/call`; bản này 0/3, 0 lần gọi.
  Không có hook thì hành vi như cũ.

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
