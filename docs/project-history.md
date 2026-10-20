# Lịch sử dự án DEW (trước là agent-cli)

Tài liệu này kể quá trình tiến hoá kỹ thuật từ upstream Crush tới HEAD `5680d80b` (nhánh `main`, 20/10/2026), dựng
lại bằng `git log`/`git show`/`git diff` và các báo cáo trong repo. Không phải tài liệu kiến trúc hiện hành: mã nguồn,
test và hành vi binary là nguồn sự thật; tài liệu này chỉ giải thích vì sao hệ thống thành ra như vậy.

**Quy ước bằng chứng.**
- Commit ghi 8 ký tự đầu của hash; mọi hash trong tệp này đều thoả `git cat-file -e <hash>`.
- Báo cáo ghi dạng `tệp:mục` (vd `DECISIONS.md:2026-10-15 "Ranh giới provider/model"`). Viết tắt: SE =
  `reports/system-evolution/`, DW = `reports/daily-work/`, V3 = `reports/v3-research/`.
- Số đo chỉ chép từ báo cáo hoặc từ lệnh `git` chạy ngày 20/10, có ghi nguồn. Câu nào là suy luận thì ghi "Suy luận:".
- Ngày là author date (`git log --date=short`, múi +0700). Commit cherry-pick giữ author date gốc, nên thứ tự theo
  ngày có thể khác thứ tự trên `main` (vd `54ab374b` mang giờ sớm hơn `3899c5b1` nhưng nằm sau nó).

## 1. Nguồn gốc: FORKED FROM Crush

- **Dự án gốc:** Crush, TUI coding agent viết bằng Go của Charm (Charmbracelet, Inc.) (`docs/UPSTREAM-README.md`).
- **URL:** https://github.com/charmbracelet/crush (remote `upstream` trong `git remote -v`).
- **Giấy phép:** FSL-1.1-MIT ("Functional Source License, Version 1.1, MIT Future License"), "Copyright 2025-2026
  Charmbracelet, Inc.", cộng một giấy phép MIT riêng cho phần mã "Copyright (c) 2025-03-21 - 2025-05-30 Kujtim Hoxha"
  (`LICENSE.md:114-116`). Nghĩa vụ: giữ nguyên `LICENSE.md` và các thông báo bản quyền khi phân phối.
- **Điểm fork:** `a73e1bfa` (13/10/2026, "test(message): stop the debounce tests from racing the created event
  (#4067)"); `git merge-base --is-ancestor a73e1bfa main` đúng.
- **Upstream tới điểm fork:** 4.296 commit (`git rev-list --count a73e1bfa`), từ `4b0ea68d` ("initial", 21/03/2025);
  173 tên tác giả trong `git shortlog -sn a73e1bfa`, nhiều commit nhất là Kujtim Hoxha, Andrey Nering, Christian Rocha.
- **Merge từ upstream sau fork:** không có. `git log --merges a73e1bfa..main` chỉ ra 2 merge nội bộ của fork
  (`f86fd4ff`, `28259e14`).

**Upstream cung cấp gì tại điểm fork** (kiểm bằng `git ls-tree a73e1bfa internal/`, không chạy thử): TUI Bubble Tea
(`internal/ui`), agent/coordinator trên `charm.land/fantasy`, nhiều provider, session lưu SQLite + sqlc, chế độ
client/server (`CRUSH_CLIENT_SERVER` trong `internal/cmd/root.go`) với `internal/backend`, `server`, `client`, `proto`,
sổ dự án (`internal/projects`), hook (`internal/hooks`), skill, MCP, LSP, hệ quyền, trình thực thi shell. README
upstream liệt kê: nhiều model, đổi model giữa session, nhiều session mỗi dự án, LSP, MCP, chạy trên nhiều hệ điều hành.

**Fork đổi gì.** Thêm package không có ở upstream: `internal/loop`, `internal/routine`, `internal/execpolicy`,
`internal/ui/home` (màn Home). Đổi hành vi trong đường mã của upstream: tự duyệt bash, hộp thoại quyền, chọn và
fallback model, phân giải data dir/workspace, đồng bộ coordinator, vòng đời server. Kích thước (đo 20/10):
- `git diff --shortstat a73e1bfa main -- '*.go' ':!*_test.go'`: 185 tệp, +18.931/−1.528 (mã Go sản phẩm).
- Cùng lệnh tính cả test: 408 tệp, +51.875/−1.642. Thư mục `reports/`: 1.394 tệp, +207.098.

**Lấy lại từ upstream không qua merge.** `461b540f` port nguyên văn hai nhánh upstream `fix/dangling-tool-calls`,
`fix/shutdown-records-work` và ý của `fix/background-job-budget` (`DECISIONS.md:2026-10-15 "Port sớm ba bản vá
upstream"`). Lý do ghi ở đó: tự viết bản riêng thì lệch upstream, merge sau tốn hơn.

**Phần còn chung có chủ ý** (invariant `CLAUDE.md` P3: "để merge upstream rẻ"): module Go
`github.com/charmbracelet/crush` (`go.mod`), tệp cấu hình `crush.json`, biến `CRUSH_*`, DB `.crush/crush.db`, binary
cài tên `crush` dưới `~/.local/lib/agent-cli/`, system prompt mở đầu "You are Crush"
(`internal/agent/templates/coder.md.tpl:1`). Đổi system prompt làm đỏ cassette VCR của `TestCoderAgent`.

**Phần đã tách hẳn khỏi upstream về kiến trúc** (không merge ngược từ upstream, nên các phần này không còn đồng bộ):
- Ranh giới tin cậy riêng: package `internal/execpolicy` (allowlist model, giao việc tắt mặc định, đọc env gốc chứ không
  đọc `.env` của repo); client lệch chính sách bị từ chối (`3899c5b1`, `213411fb`).
- Danh tính dự án: một workspace và một DB cho mỗi repo ở mọi chế độ (`300c8dd1`); launcher dựng mỗi dự án một server và
  màn Home (`internal/ui/home`).
- Độ bền prompt: ghi bền trước khi báo nhận (`b5a487f1`).
- Lớp chạy không người trực: `internal/loop`, `internal/routine` (journal, cổng verify, khoá chống chạy chồng).

**Nghĩa vụ giấy phép.** Roadmap ghi "Giấy phép và thông báo upstream phải được giữ. Không coi fork là mã viết mới hoàn
toàn" và roadmap dành cho dùng cá nhân (`docs/superpowers/plans/2026-10-14-agent-cli-personal-roadmap.md` §1). Tài
liệu này không diễn giải điều khoản FSL; đọc nguyên văn `LICENSE.md`.

## 2. Mốc thời gian

| Ngày | Commit / tag | Sự kiện |
|---|---|---|
| 21/03/2025 | `4b0ea68d` | Commit đầu tiên của Crush |
| 13/10/2026 | `a73e1bfa` | Điểm fork |
| 14/10 | `c330628e` | Core MVP Phase 1: Home, nhiều session, server theo dự án (109 tệp, +15.095/−293) |
| 14/10 | `2ed75353` | Đổi tên hiển thị thành agent-cli, logo riêng, tắt kiểm tra cập nhật |
| 14/10 | `61d8acf9` | Tool giao việc cho Claude Code và Codex |
| 14/10 | `f86fd4ff`, `28259e14` | Gộp `feat/core-mvp-phase1`, `feat/personal-rename-providers` vào `main` |
| 14/10 | `9186c66b` | Sửa P0 sau audit gộp: giao việc phải xin quyền |
| 14/10 | `d27ce38e` | Mở `DECISIONS.md` |
| 15/10 | `461b540f` | Bảng Jobs, khôi phục tool call bỏ dở, port bản vá upstream (200 tệp, +17.625) |
| 15/10 | `24041325` | Allowlist provider/model, giao việc tắt mặc định |
| 15/10 | `7a7fda8f` | Agent không được giết tiến trình agent-cli |
| 16/10 | `87233bc8` = `v0.1.0` | Mốc cơ sở, tag gắn lùi (45 commit kể cả merge từ điểm fork) |
| 16/10 | `b7f02e4b` | Deep audit 16/10 |
| 16/10 | `8568a886` = `v0.2.0` | Bản sửa theo audit |
| 16/10 | `94a9fc3d`, `bd21776e` | Engine `loop`, lệnh `routine` |
| 16/10 | `4a0b25b6` = `evo/baseline-20261016` | Bịt lỗ chế độ không người trực; mốc xuất phát của V2 |
| 16/10–19/10 | (ngoài git) | V2 System Evolution + Daily Workspace trên patch; tiếp quản 19/10 |
| 19/10 | `89ac6fe5` … `7cba42ff` | V2 vào git: 8 commit (trước là nhánh `v2-final`) |
| 19/10–20/10 | `fdd759fa` … `300c8dd1` | V3: 19 commit sửa lỗi rút từ nghiên cứu (trước là nhánh `v3-candidate`) |
| 20/10 | `0c96404c`, `f5189b0d` | Báo cáo V2/V3 vào git; patch của từng nhánh phụ trước khi xoá nhánh |
| 20/10 | `0b5bf83e` | Đổi tên thành DEW ở mức repo |
| 20/10 | `5680d80b` | HEAD; chưa có tag sau `v0.2.0` |

Số commit (đo 20/10): `git rev-list --count --no-merges a73e1bfa..main` = 98; tính cả 2 merge = 100. Lịch sử sau
14/10 là đường thẳng: các nhánh V2/V3 được tua nhanh vào `main`, không có merge commit (V3 `HANDOFF_2026-10-20.md` §2).

## 3. Các giai đoạn

Ranh giới do người viết tài liệu đặt theo commit, tag và báo cáo. Mỗi giai đoạn theo khung của
`V3 spec/DEW_RECONSTRUCTION_WORKFLOW.md` §5. "Không có dữ kiện" nghĩa là không tìm thấy nguồn trong repo.

### Giai đoạn 0 — Upstream Crush (đến 13/10/2026)

- **BEFORE / PROBLEM / CHANGE:** không áp dụng (không thuộc lịch sử của fork).
- **AFTER:** nền ở mục 1.
- **Khoảng trống người dùng thấy ở nền này:** roadmap §3 ghi phần "chưa được coi là hoàn thành": Home liên
  workspace, vòng đời khi client cuối đóng, job bash có owner, draft riêng từng session, routing đồng thời của
  quyền/câu hỏi, cài sạch có nâng cấp không mất dữ liệu.
- **Các mục còn lại:** không áp dụng.

### Giai đoạn 1 — Fork và Core MVP Phase 1 (14/10; `a73e1bfa..d27ce38e`, 14 commit + 2 merge)

- **BEFORE:** Crush tại `a73e1bfa`, một TUI làm việc theo cwd.
- **PROBLEM:** một người làm FE/BE/devops cần nhiều session, nhiều dự án song song trong một terminal, với model
  local hoặc provider qua API key; yêu cầu "tái sử dụng lõi Crush; không viết lại framework agent hay TUI" (roadmap §1).
- **CHANGE:**
  - `c330628e`: Home (Active/Pinned/Recent), Alt+S/N/M, draft riêng từng session, hộp thoại quyền ở lại session của
    nó, server giữ run khi client cuối rời (trần mồ côi 30 phút), server riêng mỗi dự án, `ps --all`, `send`,
    `run --json`, `agent-cli-tmux`, Stop hook, cấu hình provider qua `AGENT_CLI_*`.
  - `2ed75353` đổi tên hiển thị; `61d8acf9` thêm tool `claude_code`/`codex`; `9aaf150d` thêm roadmap và plan.
  - Sau khi gộp: ba người soát độc lập chỉ đọc (`docs/reviews/2026-10-14-audit-merge-main.md`), dẫn tới
    `9186c66b` (P0), `7197739a` (stream treo khi server chết), `fe1d864e`, `0d858d17`, `ac1ac2f2`, `fd2cceee`
    (installer có sao lưu và lệnh quay lại), `056fd2bc` (test giao việc với CLI thật).
- **AFTER:** sau đợt sửa: 65 package xanh, race 31 package xanh, E2E `run --json` 39/39, giao việc thật 10/10
  (audit-merge-main "Kết quả kiểm sau khi sửa").
- **NEW CAPABILITY:** nhiều dự án và nhiều session trong một terminal; run sống khi đóng TUI; CLI cho script; giao
  việc cho Claude Code/Codex.
- **NEW COMPLEXITY:** server nền mỗi dự án mà client sau đi theo cấu hình client đầu (first-wins); sổ dự án có trường
  `locations`; CLI con giao việc chạy toàn quyền mặc định (`DECISIONS.md:2026-10-14 "Tool claude_code/codex"`).
- **LESSONS:** P0 "tool giao việc chạy CLI con toàn quyền mà không hỏi" được hai người soát độc lập cùng tìm ra.
  `CLAUDE.md` bắt dùng script không có trong git (audit P1): ledger `.superpowers/sdd/` của đợt này chưa từng vào git.
- **WHAT SURVIVED:** Home, server theo dự án, `run --json`/`send`/`ps`, cổng quyền cho giao việc, installer có rollback.
- **WHAT WAS REMOVED:** cổng `restoreModelFromSession` khi có run (thêm ở đợt sửa này) bị thay ngày 15/10 và bộ khôi
  phục model bị xoá ở V3 (`4741c543`).
- **WHAT WAS LATER REWORKED:** first-wins thành từ chối (`3899c5b1`); giao việc tắt mặc định (`24041325`); `locations`
  chỉ còn đọc (`300c8dd1`).

### Giai đoạn 2 — Dùng thật hằng ngày (15/10 → trưa 16/10; `d27ce38e..87233bc8`, 29 commit) → `v0.1.0`

- **BEFORE:** bản Phase 1 đã gộp; lúc audit 14/10 máy chưa có key GLM, LLM chính là server giả.
- **PROBLEM:** dùng thật làm lộ sự cố (`DECISIONS.md`, các mục 2026-10-15):
  - 08:12: mở agent-cli trong `~/application`, init tự chạy 10 lệnh bash và ghi `AGENTS.md` 10,6 KB vì phím lạc rơi
    vào nút mặc định Allow.
  - ~14:26 (bản `8eff139e`): agent coi client và server là "mồ côi" rồi `kill -9` cả hai.
  - ~14:38: tool call `write` có JSON bị cắt, request kế bị provider trả HTTP 400.
  - Session GLM/Qwen tự giao việc sang Claude Code chỉ vì CLI có trên máy.
  - Alt+M hiện model của mọi provider; tóm tắt timeout làm dừng run; câu trả lời bị cắt ở ngân sách 8192 token.
- **CHANGE:** `133eb476` (Alt+P chọn provider rồi model, Alt+E, hộp thoại chọn sẵn Deny); `6e67f1f0` (GLM qua provider
  `zai` của catalog); `461b540f` (Jobs Alt+J, tool call bỏ dở ghi "Interrupted", huỷ theo RunID, trần mồ côi 6 giờ);
  `24041325`, `3f461a07` (allowlist `AGENT_CLI_ALLOWED_MODELS`, env tin cậy chụp đầu `main`); `9d461aed`, `94afe474`
  (tóm tắt hỏng vẫn làm tiếp, tự làm tiếp khi bị cắt); `7a7fda8f` (BlockFunc chặn tự giết); `a4b0286a`, `79c568e6`,
  `8152a7de`, `e0e752fe` (không gửi JSON hỏng, không tự vá arguments của tool ghi). Trưa 16/10: `3a1cb9d6`,
  `ee5949f3` (F2 đổi tên session), `1d93f9dc` (ví dụ Stop hook "cổng nghiệm thu"), `e1801d65`, `87233bc8` (README).
- **AFTER:** `v0.1.0` gắn lùi vào `87233bc8`: "trạng thái trước audit" (`DECISIONS.md:2026-10-16 "Phiên bản"`).
- **NEW CAPABILITY:** ranh giới provider/model duy nhất ở `internal/execpolicy`; chặn ở lớp thực thi shell (trước cả
  YOLO); bảng Jobs; huỷ đúng lượt theo RunID.
- **NEW COMPLEXITY:** hai nguồn env (env gốc tin cậy và `.env` nạp sau); model và bộ tool ghim theo từng lượt
  (`DECISIONS.md:2026-10-15 "Mỗi lượt chạy ghim model"`). `2f34d8ce` là commit `wip` của một phiên Codex chạy song song
  (tiêu đề từ prompt, chặn qua interpreter), thân commit ghi "CHƯA nghiệm thu".
- **LESSONS:** "Cài/đăng nhập/có key không phải là cho phép" (mục allowlist). Một lần Enter lạc là đủ để lệnh chạy khi
  nút mặc định là Allow. Tự vá JSON bị cắt có thể ghi tệp cụt mà không báo, nên tool ghi không được vá.
- **WHAT SURVIVED:** `execpolicy`, `internal/shell/protect.go`, hộp thoại mặc định Deny, huỷ theo RunID, settle
  "Interrupted".
- **WHAT WAS REMOVED:** bảng Alt+M liệt kê mọi provider; phím chữ `a`/`S` cấp quyền (mục hộp thoại 15/10, (c)).
- **WHAT WAS LATER REWORKED:** guard của giai đoạn này chưa bắt `bash -c` (`DECISIONS.md` 15/10); giai đoạn 3 thêm bản
  tách
  lệnh bằng regex (`3d36cc6d`) rồi viết lại bằng parser (`659d0a3f`); khôi phục model khi mở
  session bị bỏ (`4741c543`). Tệp của `2f34d8ce` (`internal/session/title.go`, `internal/execpolicy/interpreter.go`)
  còn ở HEAD, không commit nào sau đó chạm tới; việc nghiệm thu: không có dữ kiện.

### Giai đoạn 3 — Audit sâu → `v0.2.0` (16/10; `87233bc8..8568a886`, 12 commit)

- **BEFORE:** `v0.1.0`.
- **PROBLEM:** deep audit chỉ đọc (`b7f02e4b`, `docs/reviews/2026-10-16-deep-audit-agent-cli.md` §1): "READY WITH
  RISKS", 6,5/10 độ phủ năng lực P0. CRITICAL X-01: bash tự duyệt đọc được tệp ngoài workspace
  (`git diff --no-index … ~/.config/agent-cli/env`). HIGH: không có cổng xác minh mặc định (bash lỗi vẫn hiện ✓,
  `run --json` rc=0). Agent Map không tồn tại trong mọi nhánh và binary đã cài.
- **CHANGE:** `ce9757fc`, `7cee200b`, `368d3c70` (safe-list hỏi quyền khi đối số ra ngoài workspace); `56ea31b3` (mã
  thoát bash vào metadata); `3d36cc6d` (guard tách script `-c` bằng regex), `659d0a3f` (viết lại bằng parser
  `mvdan.cc/sh/v3/syntax`);
  `9d95bfa8` (Stop hook vẫn chặn sau 3 lần thì lượt lỗi); `717c345d`; `bfa267b8` (CHANGELOG, quy trình phát hành).
- **AFTER:** `v0.2.0`; `go test ./...` 66 gói ok, 17 không có test, 0 FAIL (`CHANGELOG.md` [0.2.0] "Kiểm chứng").
- **NEW CAPABILITY:** SemVer + CHANGELOG + tag có chú thích; UI phân biệt lệnh bash lỗi.
- **NEW COMPLEXITY:** safe-list là danh sách đen theo cú pháp; guard phụ thuộc parser shell.
- **LESSONS:** câu "không bao giờ bỏ sót" ở `7cee200b` sai, lượt soát độc lập tìm thêm ba đường lọt
  (`DECISIONS.md:2026-10-16 "bash tự duyệt: bổ sung"`); bản guard regex bị phá ngay bằng `for p in N; do kill $p; done`.
- **WHAT SURVIVED:** cả đợt.
- **WHAT WAS REMOVED:** `printenv`, `ps`, `git config` khỏi danh sách an toàn.
- **WHAT WAS LATER REWORKED:** các mục "Chưa làm" của 0.2.0: A-01 (đóng một workspace giết job/MCP của workspace khác)
  sửa ở V2; A-02/A-03 (model bị thay lặng lẽ) sửa ở V2 và V3.

### Giai đoạn 4 — `loop`, `routine`, chế độ không người trực (16/10; `8568a886..4a0b25b6`, 8 commit)

- **BEFORE:** "xong" là lời agent, hoặc Stop hook tuỳ chọn.
- **PROBLEM:** "lượt xong" không phải bằng chứng (loop detection dừng với rc=0, Stop hook fail-open); server không phải
  daemon nên engine đặt trong server sẽ chết giữa chừng; audit R-01: `run` tự duyệt mọi quyền, lớp chặn không có
  `git push`, nên chạy theo lịch là toàn quyền mà không ai nhìn (`DECISIONS.md:2026-10-16` mục Loop và Routine).
- **CHANGE:** `fdfcbbbb` (sửa theme, đứng trước đợt); `94a9fc3d` engine `internal/loop`; `8b92fbc4` lệnh `loop`;
  `5641855c` chế độ không người trực; `5083dd01`, `bd21776e` lệnh `routine`; `d3ee745d` E2E; `4a0b25b6` git theo danh
  sách trắng sau khi lượt soát độc lập chạy `git` thật và chấm KHONG-DAT.
- **AFTER:** E2E `loop` 17/17, `routine` 24/24 (SE `00_BASELINE_CHECKPOINT.md`). Phase 4 của roadmap ra sớm, chưa phát
  hành (`CLAUDE.md` P1). Tag `evo/baseline-20261016` đặt ở `4a0b25b6`.
- **NEW CAPABILITY:** "xong" là mã thoát của lệnh `--verify` do driver chạy; mỗi lượt sửa là một session mới; routine
  chạy trong worktree riêng, có khoá flock, lịch chỉ in ra cho người dùng tự cài.
- **NEW COMPLEXITY:** chế độ không người trực là danh sách đen cộng danh sách trắng cho git, KHÔNG phải sandbox;
  worktree dùng chung refs, config và hooks với repo gốc (`CHANGELOG.md` [Unreleased]).
- **LESSONS:** bản đầu tự khẳng định "không thể bị bỏ qua" là sai (alias git, cờ toàn cục lạ, launcher chưa bóc).
  Trần token/cost bị bỏ vì cost đo ra 0, tức cổng giả.
- **WHAT SURVIVED:** engine `loop` là nền cho `routine` (SE `FINAL_SYSTEM_AUDIT.md` mục GIỮ).
- **WHAT WAS REMOVED:** không có dữ kiện.
- **WHAT WAS LATER REWORKED:** `NO_CHANGE`, trạng thái `RUNNING`/`ABANDONED`/`ERROR`, `pipefail`, lịch mang biến
  launcher (`1c3478a4`); chặn mọi lệnh nhắc biến `GIT_*` theo chữ (`0a58d2aa`); `loop -c ..` (`1cda7f1b`); `loop`
  mở từ thư mục con (`300c8dd1`).

### Giai đoạn 5 — V2 System Evolution (16/10, ngoài git)

- **BEFORE:** `main` @ `4a0b25b6`, cây sạch; 68 gói ok, 17 không có test, 0 FAIL; 121.920 dòng Go sản phẩm và 90.962
  dòng test trong `internal/` (SE `00_BASELINE_CHECKPOINT.md`).
- **PROBLEM:** chưa đo hệ thống hỏng ở đâu với provider thật: cô lập provider, chạy dài, skill, memory. Baseline LLM
  thật trên binary gốc: 10/10 kịch bản đạt (DW `WORKFLOW_PROGRESS.md` #2), nên bộ kịch bản không đo được "giỏi hơn" (SE
  `FINAL_SYSTEM_AUDIT.md`).
- **CHANGE:** audit theo nhóm + probe, gộp thành 88 ID vấn đề (SE `05_CURRENT_PROBLEMS.md`); harness LLM giả
  (`scripts/eval/`); đọc lỗi của Hindsight rút 7 bất biến (SE 26, 28); quyết định memory M0
  (`DECISIONS.md:2026-10-16 "Không có hệ memory riêng"`); đề xuất chính sách skill (chưa áp, chờ người dùng). Mọi
  sửa nằm trong worktree tạm, không commit (luật `CLAUDE.md` P1).
- **AFTER:** `wt-final.patch`, 72 tệp, trên `4a0b25b6` (DW `WORKFLOW_PROGRESS.md` #8).
- **NEW CAPABILITY:** cổng bằng binary thật + LLM giả (`faults`, `qwen-only`, `workspaces`); sổ ghi lời gọi LLM chỉ
  metadata.
- **NEW COMPLEXITY:** cách làm "patch + checksum ngoài git": bằng chứng trong `/tmp` mất khi máy khởi động lại 19/10
  (V3 `00_POST_V2_HANDOFF.md`: `metrics.json` từng kịch bản đã mất).
- **LESSONS:** dù có một điểm chặn provider, lời gọi đặt tiêu đề và `agentic_fetch` vẫn lọt sang provider khác ở 8/8 ca
  tiêm lỗi; 26/26 lời gọi tiêu đề baseline kết thúc vì hết token (SE `21_BASELINE_RESULTS.md`, DW
  `16_PRUNING_DECISIONS.md`); khối skill chiếm 60% system prompt mà 2/41 lượt dùng thật có nạp skill (DW
  `00_PREVIOUS_WORKFLOW_HANDOFF.md`).
- **WHAT SURVIVED:** điểm chặn duy nhất `buildAgentModels`; M0; engine `loop`.
- **WHAT WAS REMOVED:** tiêu đề cho session con, thử lại tiêu đề sau `length`, quét skill trong thư mục ẩn, ghi
  fallback xuống đĩa, `mcp.Close` của mọi App, ghi giá trị env vào log (DW `16_PRUNING_DECISIONS.md`).
- **WHAT WAS LATER REWORKED:** "fallback model chéo provider trong RAM" (quyết định cuối của SE) bị lật ở vòng 6
  (giai đoạn 6).

### Giai đoạn 6 — V2 Daily Workspace, tiếp quản, commit V2 (16/10 → 19/10; `4a0b25b6..7cba42ff`, 8 commit)

- **BEFORE:** `wt-final.patch`.
- **PROBLEM:** câu hỏi của đợt Daily: agent-cli có làm nơi làm việc chính hằng ngày được không. Red team cuối chấm
  KHONG-DAT: F1 HIGH (vẫn đổi provider lặng lẽ lúc khởi động), F4 (GREEN giả thoát 0) (DW `WORKFLOW_PROGRESS.md` #13).
- **CHANGE:**
  - Vòng 6–8 trên patch: fail closed khi model đang chọn không dùng được; app khởi động không agent để `login` vẫn chạy;
    `NO_CHANGE`; client giữ nội dung lỗi của server. Ứng viên `r8-final.patch`, 88 tệp, +5925/−306 (DW #33).
  - 19/10: phiên Codex tiếp quản trên nhánh `codex/audit-r8-takeover` (= `4a0b25b6`): dựng lại vòng 8 từ patch, khôi
    phục README, sửa harness `workspaces.sh` (SE `39_TAKEOVER_2026-10-19.md`).
  - Commit lên nhánh `v2-final`: `89ac6fe5` (cổng LLM giả), `3a0f8702`, `5336c141`, `de19a3f0`, `0a58d2aa`,
    `2f922b76` (giữ mọi lời gọi LLM ở provider đã chọn), `1c3478a4`, `7cba42ff` (README dùng hằng ngày). Cây của
    `7cba42ff` là `128d744a`, trùng cây đã kiểm (V3 `00_POST_V2_HANDOFF.md`).
- **AFTER:** SE "SYSTEM IMPROVED — WITH KNOWN LIMITATIONS"; Daily "DAILY WORKSPACE READY — WITH KNOWN LIMITATIONS".
  Số trước/sau (SE `FINAL_SYSTEM_AUDIT.md`): lời gọi lọt provider khác trong kịch bản chỉ Qwen 799 → 0; suite 4.501 →
  4.637 test; eval LLM thật 9/9 → 9/9, 87 → 69 lời gọi. Hiệu quả token/thời gian: chưa kết luận (mỗi kịch bản chạy
  1 lần).
- **NEW CAPABILITY:** provider fail closed; `run` báo lỗi cho lượt rỗng, bị bộ lọc nội dung chặn, lặp tool call; trạng
  thái lượt routine; sổ ghi LLM (`AGENT_CLI_LLM_LEDGER`).
- **NEW COMPLEXITY:** trạng thái mới "app chạy mà không có agent"; công việc chuyền giữa Claude và Codex qua patch
  (lượt soát README thứ 6 mất vì hết quota, SE 39).
- **LESSONS:** giả định "dừng lúc khởi động thì `login` không chạy" chỉ đúng một nửa (DW #14, #24); người điều phối ghi
  nhầm probe chế độ local thành "server" (DW #30); kết quả của agent khác phải kiểm lại bằng lệnh (DW #42).
- **WHAT SURVIVED:** toàn bộ `7cba42ff` làm nền V3.
- **WHAT WAS REMOVED:** fallback chéo provider (kể cả trong RAM); chép allowlist vào lịch routine; chặn biến git bằng
  phân tích phép gán, bị vòng qua bằng 7 dạng (DW `16_PRUNING_DECISIONS.md`).
- **WHAT WAS LATER REWORKED:** V3 tái hiện 13 giả định sai của V2 (V3 `FINAL_V3_AUDIT.md` Q2), xem giai đoạn 7.

### Giai đoạn 7 — V3: nghiên cứu ba dự án, sửa lỗi theo bất biến (19/10 → 20/10; `7cba42ff..300c8dd1`, 19 commit)

- **BEFORE:** `v2-final` `7cba42ff`, chưa push.
- **PROBLEM:** V3 spec hỏi agent-cli học được gì từ Context Mode, claude-mem, Hindsight về working context, continuity,
  historical knowledge. Giả thuyết ban đầu gồm các lớp trí nhớ mới.
- **CHANGE:**
  - Nghiên cứu mã, issue và fork của ba dự án (V3 `context-mode/`, `claude-mem/`, `hindsight/`); tổng hợp 22 họ lỗi
    và 5 bất biến (V3 `WORKFLOW_PROGRESS.md` #25).
  - Bộ test phơi nhiễm trên mã V2: lượt đầu 11/11 EXPOSED (#18); harness nay ở `scripts/eval/exposure.sh`.
  - Mỗi lỗi: tái hiện đỏ trên V2, sửa ở chủ sẵn có, soát độc lập DAT, rồi cherry-pick lên `v3-candidate`. Thứ tự:
    `fdd759fa` P0-1, `3899c5b1` P0-2, `54ab374b` P0-3, `33daa5bf` P1-b, `4741c543` P1-c, `213411fb` P0-2b,
    `1cda7f1b` P2-b, `b5a487f1` P1-a, `92182451` INTEG1, `cf955131` INTEG2, `01296899` P3-b, `acea254a` sửa test,
    `73ce25eb` P2-a, `dd5fe93f` P2-d, `642fe075` tích hợp P2-a×P2-d, `20212e24` INTEG3, `ef2b8050` docs, `474a51f5`
    P3-a, `300c8dd1` P2-c (V3 `HANDOFF_2026-10-20.md` §2).
- **AFTER:** trạng thái ĐỀ XUẤT "V3 PARTIALLY ADOPTED — VERIFIED", chốt sau khi chạy nốt các ca phơi nhiễm binary sau T3
  (V3 `FINAL_V3_AUDIT.md`). Trên `300c8dd1`: test so theo tên 4.637 → 4.782, 0 FAIL, 10 tên mất đều do POC xoá có chủ
  đích; race 0 trên 31 package. Eval LLM thật 12/12 → 12/12, nhưng chạy trên `73ce25eb`, trước 4 thay đổi sau (6
  commit).
- **NEW CAPABILITY:** prompt được ghi bền trước khi server báo nhận (`b5a487f1`); server so chính sách tin cậy giữa các
  client và từ chối khi khác (`3899c5b1`); một workspace và một DB cho mỗi repo ở mọi chế độ (`300c8dd1`); trần cho kết
  quả MCP (`33daa5bf`); nhãn nguồn cho bản tóm tắt (`474a51f5`); `ledger_error` (`01296899`).
- **NEW COMPLEXITY:** lời trả lời giữ chỗ với provider `agent-cli-pending`; ghi theo transaction; `trusted_env` trong
  proto; cảnh báo nâng cấp cho `.crush` cũ ở thư mục con (`CHANGELOG.md` [Unreleased] "Lưu ý nâng cấp").
- **LESSONS:** "gom quyết định về một chỗ chưa đủ; mọi chỗ tiêu thụ phải đọc nó", gặp 3 lần (P2-d, INTEG1/2, INTEG3);
  lỗi tích hợp chỉ lộ ở binary + kịch bản ghép, không unit test nào bắt được (V3 `V3_INTEGRATION_DECISION.md` §2).
  Ba dự án ngoài không đóng góp cơ chế nào, chỉ đóng góp bất biến (cùng tệp, §1).
- **WHAT SURVIVED:** cả 19 commit nằm trên `main`.
- **WHAT WAS REMOVED:** bộ khôi phục model khi mở session (3 test `TestRestoreModel*`, `4741c543`); cơ chế gộp DB qua
  `locations` của Home (6 test, `300c8dd1`). Các lớp trí nhớ (reflect, auto-recall, observation LLM, nhiều DB/worker)
  bị REJECT; graph, semantic, event sourcing, knowledge store bị DEFER, chưa từng có mã (cùng tệp, §7).
- **WHAT WAS LATER REWORKED:** không có dữ kiện (đây là đợt cuối có mã). Còn mở: một phần T6 bị chặn vì phải ghi lại
  cassette `TestCoderAgent`.

### Giai đoạn 8 — Gộp về `main`, bàn giao, đổi tên DEW (20/10; `300c8dd1..5680d80b`, 8 commit)

- **BEFORE:** việc nằm rải ở 19 worktree `agent-cli-*` và các nhánh `v2-final`, `v3-*`, `v3-poc-*` (V3 WP #100).
- **PROBLEM:** người dùng bảo dừng và đồng bộ mọi thứ lên git để máy khác làm tiếp (V3 `HANDOFF_2026-10-20.md` §1).
- **CHANGE:** `main` tua nhanh tới commit bàn giao; `0c96404c` đưa báo cáo V2/V3 vào git (1.369 tệp, +193.727);
  `85d315b9` đưa bộ phơi nhiễm vào nhánh chính; `f5189b0d` lưu mỗi nhánh phụ thành `git format-patch` kèm
  `SHA256SUMS` rồi xoá nhánh; `2ad5849a` ghi trạng thái; `d4a0cdf6`, `29cee812` thay bản clone nguồn nghiên cứu bằng
  script tải lại; `0b5bf83e` đổi tên DEW ở mức repo; `5680d80b` ghi trạng thái cuối.
- **AFTER:** HEAD `5680d80b`; chưa có tag sau `v0.2.0` (thông điệp `5680d80b`; `CLAUDE.md` P1).
- **NEW CAPABILITY:** không có thay đổi sản phẩm; thêm `docs/dew-sync.md` (lấy code, đồng bộ, đổi remote).
- **NEW COMPLEXITY:** ba tầng tên cùng tồn tại: DEW (tài liệu), `agent-cli` (lệnh, cấu hình, `AGENT_CLI_*`), `crush`
  (module, binary, `crush.json`, `CRUSH_*`).
- **LESSONS:** lệnh push bị bộ phân loại quyền chặn, người dùng tự push (V3 WP #100; `RENAME_PLAN.md` "Cần người
  dùng làm").
- **WHAT SURVIVED:** mọi commit của nhánh phụ còn dưới dạng patch trong `reports/v3-research/experiments/poc/`.
- **WHAT WAS REMOVED:** các nhánh và worktree phụ; bản clone của ba dự án nghiên cứu.
- **WHAT WAS LATER REWORKED:** phần DEFER của `RENAME_PLAN.md` (lệnh, cấu hình, biến môi trường) bị thay bằng quyết định
  đổi tên toàn bộ cùng ngày 20/10; mã đang đổi ở nhánh khác, chưa có trên `main` tại HEAD (mục 5).

## 4. Những sợi chỉ xuyên giai đoạn

Một số quyết định bị làm lại nhiều lần; đọc theo chủ đề dễ thấy áp lực kiến trúc hơn đọc theo ngày.

- **Model thuộc ai.** 14/10: không khôi phục model khi có run (audit-merge-main) → 15/10: ghim model theo lượt
  (`DECISIONS.md:2026-10-15`) → SE đặt đích "model của session, trong RAM" (SE `35_TARGET_ARCHITECTURE.md`, PRB-05
  mở) → 19/10 người dùng chốt: model thuộc workspace, mở session không ghi (`4741c543`, `DECISIONS.md:2026-10-19`).
- **Provider không dùng được.** Upstream: bỏ provider lặng lẽ, chọn model khác rồi ghi xuống đĩa → 15/10: dừng khi
  provider đang chọn thiếu endpoint → SE: không ghi bền nhưng fallback trong RAM → vòng 6–7: không bao giờ đổi
  provider, app khởi động không agent (`2f922b76`).
- **"Xong" là gì.** Stop hook (`c330628e`) → ví dụ verify-gate (`1d93f9dc`) → hook chặn mãi thì lượt lỗi (`9d95bfa8`)
  → `loop --verify` (`8b92fbc4`) → `NO_CHANGE` (`1c3478a4`) → lượt rỗng hay lặp tool call là lỗi (`2f922b76`).
- **Lớp chặn shell.** Siết tự duyệt bash (`c330628e`) → guard tự giết (`7a7fda8f`) → parser (`659d0a3f`) → không
  người trực, danh sách đen (`5641855c`) → git theo danh sách trắng (`4a0b25b6`) → chặn `GIT_*` theo chữ
  (`0a58d2aa`). Đến HEAD vẫn không phải sandbox.
- **Danh tính workspace/DB.** Server theo hash của dự án + `locations` (`c330628e`) → V3 đo được 2 DB cho một repo
  (T3) → một workspace + một DB mỗi repo (`300c8dd1`, quyết định Q5 của người dùng).
- **Giao việc cho CLI ngoài.** Thêm (`61d8acf9`) → xin quyền mỗi lần (`9186c66b`) → tắt mặc định + allowlist
  (`24041325`) → không đăng ký khi không người trực (`5641855c`) → CLI con nhận env gốc, không nhận `.env` của repo
  (`213411fb`).
- **Chính sách giữa các client.** First-wins + cảnh báo (Phase 1) → từ chối khi khác (`3899c5b1`) → thêm `--yolo` cho
  `run`/`send`/`loop` để khớp được (`20212e24`).
- **Memory.** M0: context file + lịch sử session + tự tóm tắt (`DECISIONS.md:2026-10-16`) → Memlake ngoài phạm vi
  (SE `40_MEMLAKE_SCOPE_2026-10-19.md`) → V3 DEFER/REJECT mọi lớp trí nhớ mới (V3 `V3_INTEGRATION_DECISION.md` §7).

**Ảnh hưởng bên ngoài (LEARNED FROM, không chép mã).** Hindsight (https://github.com/vectorize-io/hindsight) cho V2 một
"kho lỗi" thành 7 bất biến (SE `FINAL_SYSTEM_AUDIT.md`: "Không thành phần nào của Hindsight được nhận nguyên khối").
V3 học thêm từ Context Mode (https://github.com/mksglu/context-mode, Elastic License 2.0: chỉ học ý tưởng) và claude-mem
(https://github.com/thedotmack/claude-mem, Apache-2.0); "Không POC nào chép mã ngoài" (V3 `V3_INTEGRATION_DECISION.md`
§8; giấy phép ở V3 WP #8). Superpowers và Ponytail là bộ skill tuỳ chọn được audit ở SE 08, 09: tác dụng trong sản
phẩm không đo được (SE `FINAL_SYSTEM_AUDIT.md`).

## 5. DEW là gì và vì sao đổi tên

**DEW = Daily Engineering Workspace.** Định hướng ghi trong spec (V3 `spec/DEW_RECONSTRUCTION_WORKFLOW.md` §41): "DEW is
where engineering work lives. Providers are how the work gets done."; CLI là một giao diện, không nằm trong danh tính
sản phẩm. README đã nói ý này từ V2: "agent-cli là nơi bạn làm việc; provider là thứ thực thi" (diff của `0b5bf83e`).

**Vì sao đổi** (cùng spec, dòng 1–170):
- "Agent CLI" là tên chung chung, đã có nhiều dự án công khai dùng.
- Roadmap V2/V3 đã vượt khỏi một CLI: workspace, session, routine, provider governance, verification.
- Chính spec ghi DEW cũng ngắn và chung chung, chưa chắc là brand công khai cuối cùng. Spec khuyên đổi cách gọi nội bộ
  trước, đổi repo/binary sau như một đợt chuyển đổi riêng.

**Đợt 1, ở mức repo** (`0b5bf83e`, 20/10, theo `reports/documentation/RENAME_PLAN.md`): tiêu đề README, `CLAUDE.md`
P3, thư mục trên máy thành `opensource/dew`, thêm `docs/dew-sync.md` và `RENAME_PLAN.md`. Bản kế hoạch này còn để lệnh,
cấu hình và biến `AGENT_CLI_*` (30 biến, 543 lần xuất hiện, đo bằng `git grep` ngoài `reports/`) ở dạng DEFER.

**Đổi tên toàn bộ.** Cũng ngày 20/10, người dùng chốt đổi tên toàn bộ sang DEW, không giữ tên agent-cli: lệnh `dew`,
thư mục cấu hình/dữ liệu/cache và biến `DEW_*`, kèm chuyển đổi tự động cấu hình cũ khi cài. Quyết định này thay phần
DEFER của `RENAME_PLAN.md`. Việc đổi mã làm ở nhánh khác; tại HEAD `5680d80b` mã vẫn mang tên agent-cli, nên các
giai đoạn ở trên giữ tên agent-cli đúng như lúc đó.

**Ngoài phạm vi đổi tên theo `RENAME_PLAN.md`:** module path, `crush.json`, `CRUSH_*`, system prompt, `.crush/crush.db`
được xếp DO_NOT_RENAME (invariant P3, dữ liệu người dùng). Quyết định đổi tên toàn bộ có đổi lại các mục này hay không:
tài liệu này không có dữ kiện. Đổi tên repo trên GitHub là việc của người dùng (`RENAME_PLAN.md` "Cần người dùng làm").
Ngày 20/10 remote `origin` của bản clone này vẫn trỏ `luannt2002/agent-cli.git` (`git remote -v`).

## 6. Dữ kiện thiếu hoặc chưa chắc

- Brief giao việc ghi 97 commit; `git rev-list --count --no-merges a73e1bfa..main` đo được 98 (100 nếu tính 2 merge).
- Spec của đợt V2 (System Evolution, Daily) không nằm trong repo; báo cáo chỉ trích số mục (§). Spec V3 và DEW có ở
  `reports/v3-research/spec/`.
- Lịch sử bên trong Phase 1 (các lát `0.1.0-lat1`/`lat1.1`/`lat2`) chỉ ghi trong `~/.local/lib/agent-cli/VERSIONS.log`
  ngoài repo (`DECISIONS.md:2026-10-16 "Phiên bản"`); ledger `.superpowers/sdd/` và `docs/progress/PHASE1_STATUS.md`
  không có trong git (`CLAUDE.md` P2). Không dựng lại được từng lát.
- Giai đoạn 5–6 làm trên patch ngoài git: thứ tự sửa bên trong chỉ biết qua DW `WORKFLOW_PROGRESS.md`; một phần bằng
  chứng ở `/tmp` đã mất.
- Cách đếm V3 không thống nhất: `V3_INTEGRATION_DECISION.md` §5 ghi "13 POC + INTEG3"; `HANDOFF_2026-10-20.md` §1 ghi
  "15 sửa lỗi (14 POC + 1 sửa tích hợp)". Tài liệu này liệt kê theo commit.
- Trạng thái V3 mới là ĐỀ XUẤT: các ca phơi nhiễm binary sau T3 chưa chạy lại trên `300c8dd1`; eval LLM thật chưa chạy
  trên `300c8dd1`; `task lint` chưa chạy lần nào (V3 `HANDOFF_2026-10-20.md`).
- Ref `upstream/main` chưa được fetch về bản clone này: không đo được upstream đã đi bao xa sau `a73e1bfa`.
  Phần "upstream
  cung cấp gì" dựa trên cây tệp và README upstream, không chạy thử.
- Theo ref cục bộ, `origin/main` = `0b5bf83e` và `main` đi trước 1 commit (`5680d80b`); chưa kiểm với remote thật.
  `CLAUDE.md` P3 ghi repo là `luannt2002/dew` trong khi remote cục bộ vẫn là `agent-cli`.
- Nhánh còn trên máy (đều đã nằm trong `main`, `git merge-base --is-ancestor` đúng): `codex/master-b0-b1` (`e0e752fe`),
  `feat/agent-ux-2026-10-16` (`335709e4`), `feat/personal-rename-providers` (`9aaf150d`), `codex/audit-r8-takeover` (có
  worktree ở `~/.codex/…` của phiên Codex, giữ có chủ ý theo HANDOFF §8), `dew-rename` (nhánh đang làm việc đổi tên).
