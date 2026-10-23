# Audit mã DEW v0.3.2 — nên làm gì tiếp

Ngày 22/10/2026, `main` = `5337a9f7` (v0.3.2). Năm lượt soát chỉ đọc chạy song song, mỗi lượt một mặt:
- A: kiến trúc và nợ;
- B: độ lệch upstream;
- C: an toàn khi chạy không người trực;
- D: test và cổng;
- E: giá trị dùng hằng ngày và vận hành.

Mọi số trong mục "Số đo" đều do người tổng hợp chạy lại; lệnh nằm ngay cạnh số. Số chỉ có trong báo cáo của từng lượt soát thì
ghi rõ "theo A/B/C/D/E".

## Kết luận

dew đang làm tốt một việc: **chạy việc nền có kiểm chứng**. Trong 46 giờ có 174 lượt routine, 113 lượt GREEN, và
không còn lượt nào báo sai sự thật. Nhưng có ba chỗ phải sửa trước khi chạy thêm:

1. **Lớp chặn khi không người trực hứa nhiều hơn khả năng thật.** Chỉ cần `cd ../<repo>` trong một lệnh shell là agent đổi
   được nhánh, xoá được việc chưa commit, và ghi được `.git/config` của checkout thật. Help của `dew routine` lại viết "your
   checkout is never touched".
2. **Vòng đời routine không có bước kết thúc.** 152 định nghĩa đều là `goal` chạy một lần. Kết quả là 148 nhánh rỗng,
   54 worktree chiếm 6,6 GB, và `dew clean` không thu hồi được byte nào, vì gc luôn giữ lượt mới nhất của mỗi routine.
3. **"Xanh" chỉ là cổng chạy tay trên một máy.** CI chưa chạy lần nào cho fork, lint chưa chạy lần nào, 3 test an ninh bị
   skip ở mọi lần chạy, và chốt allowlist model không test nào bắt được.

Còn engine của loop, gồm verify, báo cáo và các lý do dừng, thì đã đủ tốt. Không cần tính năng lớn mới cho engine trong 1–2 tuần
tới.

## Số đo (đã chạy lại)

| Chỉ số | Giá trị | Lệnh |
|---|---|---|
| Lượt routine theo lý do | 113 GREEN · 29 RUN_ERROR · 12 TIMEOUT · 12 CANCELLED · 4 RUNNING · 2 STALLED · 1 STREAM_STALL · 1 NO_CHANGE (174) | đọc `reason` của `~/.local/share/dew/routines/*/runs/*/report.json` |
| Định nghĩa routine | 152, đều `goal`, 0 có lịch | `dew routine list --json` |
| Nhánh `routine/*` còn lại | 148; 0 nhánh có commit vượt base | `git rev-list --count refs/heads/<branch> ^<base_sha>` theo từng báo cáo |
| Worktree routine trên đĩa | 54, dữ liệu dew tổng 6,6 GB; đĩa còn 84 GB (63%) | `ls -d …/wt/*/`, `du -sh ~/.local/share/dew`, `df -h /` |
| TTFT trung vị theo số lượt chạy cùng lúc | ≤1: 1,9 s (66 routine) · 2–3: 10,7 s (21 routine) | `dew routine stats --json --since 72h` |
| Upstream đi trước | 33 commit; merge thử: 29 tệp, 80 hunk conflict | clone upstream ở scratch, `git merge --no-commit` rồi `--abort` |
| Ghi chú `shortcut:` | 24 (con số 45 đo ban đầu đếm cả một worktree agent cũ trong `.claude/`) | `grep -rn --include=*.go '//.*shortcut:' . --exclude-dir=.claude` |
| CI của fork | 0 lần chạy | `api.github.com/repos/luannt2002/dew/actions/runs` |
| Test bị skip | 12; trong đó 3 test an ninh của fork vì máy không có binary `rg` (chỉ có hàm shell) | `after.json` của cổng v0.3.2, `type -a rg` |
| Thoát worktree | Đã tái lập: sau các lệnh chạy dưới guard, checkout "của người dùng" sang `agent-branch`, mất tệp đang sửa, mất `sub/`, có khoá config lạ | test `-overlay` của C (chỉ có ở scratch) trên repo nháp mới dựng |

## Phát hiện, xếp theo độ đắt khi cần tin kết quả

### P1. Agent không người trực sửa được checkout thật của bạn (C, đã tái lập)

- **Cách xảy ra:** `blockHandler` (`internal/shell/run.go:336`) chỉ xét argv, không xét thư mục hiện tại. Vì vậy `cd ../main && git
  checkout -b x`, `git rm -f`, `rm -rf main/sub` đều chạy.
- **Các lối lách khác:** script có shebang, `sh s.sh`, python/node, `git grep -O<prog>`, mạng (`curl`, `ssh`).
- **Không phải leo quyền,** vì agent chạy cùng UID. Cái nó phá là hai lời hứa của routine:
  1. "Duyệt diff là đủ": thay đổi trong `.git` dùng chung (config, hook, tag, nhánh) không hiện trong `changed_files` hay
     trong patch export.
  2. "Checkout không bị đụng".
- **Hook vẫn chạy ở những lệnh git dew tự gọi trên repo chính.** v0.3.2 mới tắt hook cho bộ phân loại, cho dấu vân tay và cho
  export. Còn hai chỗ chưa tắt:
  - closure `git` ở `internal/cmd/routine.go:993` (`worktree add`, `worktree remove --force`, `branch -D`) không có cờ an toàn nào;
  - `runGit` ở `routine.go:1456` chỉ tắt fsmonitor.

  Theo A: một hook do agent cài vào `.git/hooks` sẽ chạy ở lượt sau, khi dew gọi `worktree add`.
- **Việc cần làm:**
  - (S) Một helper git duy nhất với cờ cố định `--no-optional-locks -c core.fsmonitor=false -c core.hooksPath=/dev/null`,
    kèm một test fail khi có chỗ exec `"git"` nằm ngoài helper.
  - (S) Sửa câu chữ ở help `dew routine` (`routine.go:83-86`), comment unit lịch (`internal/routine/routine.go:214`) và docs
    dòng 765. Nói rõ: "guard-rail, không phải sandbox: agent đọc được tệp của bạn, ra được mạng, và đổi được `.git` dùng chung".
  - (M) Khi không người trực, từ chối lệnh nếu thư mục làm việc thật (sau `EvalSymlinks`) đã ra ngoài workspace. Việc này đóng
    lối `cd`, nhưng không đóng được tiến trình con.
  - (M) Chụp dấu vân tay của `$(git-common-dir)/{config,hooks,info}` và các ref ngoài `refs/heads/routine/*`, trước và sau mỗi
    lượt. Có đổi thì ghi `git_dir_changed` vào báo cáo và cảnh báo.
  - (L, để sau) Muốn có ranh giới thật thì chạy routine trong **clone riêng** thay cho worktree (hết `.git` dùng chung), cộng
    systemd `ProtectHome=read-only` hoặc bwrap.

### P1. Vòng đời routine thiếu bước kết thúc (E, đã đo lại)

- **gc không dọn được gì:** nó luôn giữ lượt mới nhất của mỗi routine (`routine.go:1280-1283`) và không báo số đã giữ. Với
  routine `goal` chạy một lần, mọi worktree đều là "mới nhất", nên gc không bao giờ gỡ.
- **Lượt hỏng vì cấu hình vẫn để rác:** 9 lượt hôm nay hỏng ngay từ đầu (model không tồn tại, thiếu key), mỗi lượt vẫn để lại
  một worktree và một nhánh, vì kiểm model diễn ra sau `worktree add` (`routine.go:1043`).
- **Lệnh "chỉ xem" giữ khoá routine (theo A, đã đọc mã).** `dew clean` không có `--apply` và `routine gc --dry-run` vẫn lấy khoá
  routine (`routine.go:1491`) trong suốt lúc đo dung lượng. Một lượt bắt đầu đúng lúc đó nhận `lock_held`, và theo
  `docs/routine-invariants.md` Q5, loại này là loại **không được chạy lại**: lượt đó mất mà không ai biết.
- **Việc cần làm:**
  - (S) Kiểm model, key và provider trước `worktree add`.
  - (S) Dry-run không lấy khoá (nó không ghi gì).
  - (S–M) Thêm hàng Q4 vào đặc tả trước, rồi mới viết mã: lượt mới nhất đã xong, sạch, không commit và không được mở lại thì
    gỡ được. `keepReason` giữ nguyên.
  - (S) Lệnh đóng routine sau khi đã nhận kết quả (gỡ worktree, nhánh và định nghĩa).
  - (S) In số worktree bị giữ vì là "mới nhất".

### P1. Cổng chỉ chạy tay, và có chốt không test nào bắt được (D, đã đo lại phần CI/skip)

- **Không có cổng tự động:** CI chưa chạy lần nào. `golangci-lint` và `task` không có trên máy, nên `task lint` chưa từng chạy.
  Theo D, `gofumpt -l` báo 32 tệp do fork chạm chưa đúng format.
- **Ba test an ninh của fork bị skip ở mọi lần chạy** (`TestGrepHostileArgsWithRealRg`, `TestGrepPatternIsNeverARipgrepFlag`,
  `TestGlobPrefixBypassAttempts`), vì `rg` chỉ là một hàm shell. Ngoài ra `rg.go:16` trả rỗng khi `testing.Testing()`.
- **Chốt allowlist model không test nào bắt được:** `execpolicy` luôn trả chính sách rỗng trong test binary
  (`internal/execpolicy/policy.go:234-240`). Xoá lời gọi kiểm model trong `coordinator.go:1129-1131` thì `go test ./...` vẫn
  xanh (D, suy luận từ mã).
- **Script cổng phát hành không nằm trong repo:** nó nằm ở scratchpad của phiên, và danh sách race bỏ sót `shell`, `execpolicy`,
  `housekeep`, `config`.
- **Việc cần làm:**
  - (S) Đưa script cổng vào `scripts/release-gates.sh`; cổng đỏ khi một trong 3 test an ninh bị skip, hoặc cài `ripgrep`.
  - (S) `gofumpt -w` các tệp của fork, rồi chạy lint một lần và ghi kết quả.
  - (S) Bật Actions cho fork nhưng **chỉ** giữ build và lint. `schema-update.yml` sẽ tự commit lên `main`;
    release/nightly/snapshot cần runner riêng.
  - (S–M) Chỗ tiêm chính sách cho coordinator, kèm test `MODEL_NOT_ALLOWED`. Thêm các ca `rm ..`, `x/..`, và `bash -c` lồng
    5 lớp: hiện `policy.go:600` cho qua (fail-open, đã nêu từ audit 16/10).

### P2. Trạng thái một lượt vẫn được suy ra ở 3 nơi (A)

- Schema `report.json` được khai hai lần: `routineReport` 29 trường, `runReport` 11 trường.
- Hai phép kiểm "còn sống": `listRuns` thử khoá, còn `RunAlive` kiểm pid và cmdline.
- Cùng một object JSON của `routine list` có thể tự mâu thuẫn: `last_reason=ABANDONED`, `state=UNKNOWN`. Đây đúng là lớp lỗi
  mà Q1 của đặc tả muốn xoá, nhưng v0.3.2 mới gom phần cây làm việc (Q3).
- Theo D: fixture test dựng báo cáo theo hình dạng mà writer không bao giờ ghi ra, nên hai reader có thể đọc khác nhau mà
  test không thấy.
- **Việc cần làm:**
  - (M) Gom schema, đọc/ghi, liệt kê và tính trạng thái vào `internal/routine`.
  - (S) Thêm một test hợp đồng: `writeReport` → `listRuns` và `LatestRuns` ra cùng kết quả.

### P2. Độ lệch upstream tăng dần (B, đã đo lại merge thử)

- **Tình hình:** 33 commit sau 8 ngày đã cho 29 tệp conflict.
  - 34 hunk là do fork port trước bản sửa settle/dangling-tool-call, trong khi upstream merge bản khác.
  - `agent.go` có conflict về logic.
- **Upstream có bản sửa fork chưa có:** agent tự phục hồi khi web request lỗi (`f5309364`), Ctrl+C lúc khởi động, cache
  discovery model, thu hẹp quyền token CI.
- **Các branch upstream đang làm** (super-yolo, hooks-galore, routing) chạm 18–45 tệp fork đã sửa. Branch routing đi ngược cam
  kết "ghim provider" của fork.
- **Việc cần làm:**
  - (S) Cherry-pick các bản sửa nhỏ.
  - (M) Merge upstream mỗi tuần, lấy bản settle của upstream.
  - (M) Chuyển hàm mới của fork khỏi `agent.go`, `root.go`, `backend.go` sang tệp `*_dew.go`; test fork thêm vào tệp test
    upstream thì chuyển sang `*_dew_test.go`.
  - (S) README giữ bản fork khi merge (`merge=ours`).
  - Để quá một tháng thì một lần sync sẽ thành việc cỡ L.

### P3. Còn lại

- **Test dễ flaky** (D): 27 lần `time.Sleep` ở 11 tệp test của fork (nặng nhất `queue_handoff_review_test.go`, 13 lần); 15
  assert chặn trên theo đồng hồ thật. `TestTrimTo_TimingOnFiftyThousandEntries` không assert gì.
- **LLM giả của E2E quá hiền** (D): tool call gửi một chunk, usage cố định 11/7 token, nên không bao giờ chạm compaction.
  Lần chạy cuối với provider thật là 16/10.
- **Docs lệch mã** (E):
  - README:48-49 vẫn ghi chưa có tag cho loop/routine.
  - `docs/dew-local-setup.md:767` bảo duyệt bằng `git diff <base>..routine/...`, nhưng không có `--commit` thì diff này rỗng;
    đường đúng là `dew routine export`.
  - Setup doc dài 98 KB, phần lớn là ca biên.
- **Ghép chặt vào upstream** (A): `loop.go:591` ép kiểu không kiểm `ok`; `settleTurn` dựa vào việc coordinator bị thay mỗi
  lượt; `transientFromText` đoán lỗi theo câu chữ của upstream.

## Nên làm gì: thứ tự đề xuất

| Đợt | Việc | Cỡ | Vì sao trước |
|---|---|---|---|
| 1, ngay | Helper git duy nhất có tắt hook; sửa help/docs cho đúng giới hạn; dry-run không lấy khoá; kiểm cấu hình trước `worktree add` | S×4 | Rẻ, đóng chỗ hứa sai và chỗ mất lượt im lặng |
| 1, ngay | Script cổng vào repo; cổng đỏ khi test an ninh bị skip; `gofumpt`; Actions chỉ build và lint | S×4 | Mỗi bản sau mới tin được màu xanh |
| 2, tuần này | Chặn lệnh khi cwd ra ngoài workspace; ghi `git_dir_changed` | M×2 | Đóng lớp lỗi mà C đã tái lập |
| 2, tuần này | Hàng Q4 mới cho lượt mới nhất, rồi gc và lệnh đóng routine | S–M | Chặn 3 GB/ngày và 148 nhánh rỗng |
| 3 | Gom trạng thái lượt vào `internal/routine` cộng test hợp đồng; chỗ tiêm execpolicy | M | Xoá nốt lớp "một câu hỏi, nhiều câu trả lời" |
| 3 | Cherry-pick 5 bản sửa upstream, rồi merge upstream định kỳ | S rồi M | Chi phí tăng theo thời gian |
| Không cần mã | Giữ 1–2 routine cùng lúc trên gateway hiện tại; chia việc cho vừa một lượt 45 phút; sau 2–3 ngày đo `routine stats` xem bước chạy tiếp sau khi cắt của v0.3.2 thu lại được bao nhiêu | — | TTFT tăng ~5,5 lần khi 2–3 lượt chạy cùng lúc |
| Quyết định của chủ dự án | dew là engine chạy nền, còn tương tác dùng Claude Code? Hay đi cổng WIN để đầu tư TUI? | — | 0 tin nhắn TUI ở dự án thật từ 21/10 (theo E) |

**Chưa nên làm:**
- **Hàng đợi (B1):** `no_slot` cộng bộ điều phối ngoài đã làm vai hàng đợi. Thêm hàng đợi sẽ đẩy nhiều lượt chạy cùng lúc hơn
  lên một gateway đang chậm.
- **Kiểm teeth (A3):** `touched_gate` = 0 trên 112 lượt GREEN (theo E). Chưa thấy GREEN giả nào.
- **Tách UI lớn:** `ui.go` chỉ có 59 dòng upstream đổi lần này.

## Chưa kiểm

- Chưa build và test bản merge upstream.
- Chưa chạy thật các ca "mới nghi" của A:
  - hook do agent cài chạy ở lượt sau qua `worktree add`;
  - stamp theo giây làm hai lượt ghi đè cùng thư mục;
  - `crush.json` do agent sửa được nạp lại giữa lượt.
- Chưa đột biến chốt execpolicy và 3 nhánh chưa có test của `leavesWorkspace`.
- Không đọc tệp env, nên chưa biết giá trị thật của `DEW_ROUTINE_SLOTS` và `DEW_GOCACHE_MAX`.
- Số liệu TUI (E) đếm từ `crush.db`, không tách được TUI với `run`/`send`.
