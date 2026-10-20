# claude-mem — khảo cổ fork và nhánh thí nghiệm (overlay §8–10)

- Repo chỉ đọc: `/home/luannt/application/opensource/v3-src/claude-mem`, HEAD `3fae1eb45ba1a7fbd4818527f48b439991a273da`.
- **Ngân sách fork đã dùng: 6/6 trang**: (1) danh sách fork "active" trang 1, (2) `tt12b/claude-mem`, (3) `tt12b/claude-mem/commits/main`, (4) `shantanukhanwalkar/claude-mem`, (5) `tqtensor/claude-mem`, (6) commit `rodboev/claude-mem@2b38daa`. Không có 404 hay tường đăng nhập. Ba trang repo fork **không hiện dòng "ahead/behind"** trong nội dung WebFetch trả về, nên chưa đếm được số commit chỉ-có-ở-fork bằng trang compare (không còn ngân sách).
- Nguồn bổ sung không tốn trang fork: các SHA của fork hiện trong timeline 35 trang issue/PR đã đọc (03 §0), đối chiếu với clone. Ngoài ra là 763 nhánh remote trong clone (`git for-each-ref refs/remotes/origin`).
- Mức bằng chứng: E0 = trang/thân commit; E1 = git trong clone; E2 = test tồn tại. Phân loại theo overlay §8–10.

**Kết luận ngắn:**
- Soi 13 fork ở mức khác nhau (§1). **1 fork có thay đổi kiến trúc** (`tt12b`, biến thể triển khai tự host). **1 fork sửa scope** có phân kỳ thật (`rodboev`, chuỗi heuristic monorepo). 9 fork còn lại là nhánh nguồn PR (ý tưởng đã vào upstream) hoặc mirror/không đủ bằng chứng.
- Thông tin kiến trúc đáng giá hơn nằm ở **nhánh remote của chính upstream**: `branch-memory`, `canary`, dòng `community-edge`, `bugfix/security-hardening`, `enhancement/supersession` (§3).

---

## 1. Toàn cảnh fork

- Trang danh sách (bộ lọc Active, 2 năm, sắp theo cập nhật gần nhất) hiện "page 1 of 61", 25 fork, **tất cả 0 sao**, cập nhật trong ngày 2026-10-19 (E0). **Suy luận:** khoảng ≤1.525 fork active (61 × 25), chưa đếm.
- Các fork đầu danh sách là fork của người gửi PR cho upstream. Ví dụ `akushonkamen` là tác giả `2b4bb4377`, `137360788`; `rudycelekli` (Rudy Mizrahi Celekli) là tác giả `21d884c12`, `6d5d79faf`, `3724ced80`; `costa-marcello` là đồng tác giả `9624d0a70` (E1 `git show`). → Xếp theo sao hay theo cập nhật đều không lọc ra fork có ý nghĩa. Cách chọn ở đây: lấy fork mà timeline issue cho thấy có commit **khác SHA với upstream**.

| Fork | Nguồn phát hiện | Mức soi | Phân loại |
|---|---|---|---|
| tt12b | timeline #2117, #1911, #2663, #2201, #2782 | trang repo + trang commits | **DEPLOYMENT_VARIANT / ARCHITECTURAL_FORK** |
| rodboev | timeline #2882 (01) | trang commit 2b38daa + clone | **SCOPE_FIX** (một phần là nguồn PR) |
| tqtensor | timeline #2117, #1911, #2201, #1505 | trang repo | MIRROR |
| shantanukhanwalkar | timeline #3523, #2966 | trang repo | TRACKING_FORK (INSUFFICIENT_EVIDENCE) |
| surfingdoggo | timeline #3641 (`38685a9`) | clone (squash `276f845d2`) | nguồn PR, SCOPE_FIX → đã vào upstream |
| DanielCarmingham | timeline #3641 (`0eace99`) | clone (`f08f145fb`) | nguồn PR, SCOPE_FIX → đã vào upstream |
| escooterclinic | timeline #3802 (`5b9ca92`) | clone (`817825876`) | nguồn PR → đã vào upstream |
| yetanotherflo | timeline #3678, PR #4251 | clone | nguồn PR (HOST_PORT OpenCode) → đã vào upstream |
| ntdatt812 | timeline #3802 (`62cf4d7`) | tiêu đề trùng `1e284feb0` | nguồn PR → đã vào upstream |
| stantheman0128 | timeline #3274 (`f9e7150`) | tiêu đề trùng `a533acc33` | nguồn PR → đã vào upstream |
| zichen0116 | timeline #4087 (`28cd877`) | thân `a5cbafd7a` chứa cùng tiêu đề | nguồn PR → đã vào upstream |
| y1024 | timeline #2663 (`daf6d9d`) | trùng SHA rút gọn với upstream `daf6d9dc0` | MIRROR |
| juliuswiener | timeline #4087 (`4fee855`) | không soi | INSUFFICIENT_EVIDENCE |

Kiểm trong clone: 24 SHA rút gọn của fork lấy từ timeline (ef153dc, 2b38daa, 38685a9, 0eace99, 5b9ca92, …) **không tồn tại** trong clone (`git cat-file -t` → missing). Chỉ `1055ea4` (rodboev) có, và nó nằm ở nhánh upstream `origin/copilot/merge-all-features-to-community-edge` (E1).

---

## 2. Fork có ý nghĩa và ForkChange

### 2.1 tt12b/claude-mem — DEPLOYMENT_VARIANT / ARCHITECTURAL_FORK

- **Base:** chưa xác định chính xác (trang không hiện ahead/behind). `main` có 2.613 commit (E0). README tả Postgres + Valkey (BullMQ) + API/dashboard + worker. Đó là hình dạng của runtime `server-beta` upstream (`36b0929fa`, 2026-05-08: "Postgres storage + independent runtime + BullMQ queue"). **Suy luận:** fork dựng trên server-beta.
- **Commit chỉ-có-ở-fork thấy được:** 35 commit trên trang 1 (2026-09-10 → 09-22), tác giả "tt12b and claude", tiền tố `DEV-INTERNAL`, tiêu đề tiếng Hàn (E0, bản dịch của WebFetch). Có xoá liên kết, nút star và docs của upstream (`8b84157`, `0143927`) → phân kỳ có chủ ý.
- **Hệ con đổi (README + commit, E0):**
  - lưu trữ Postgres + pgvector;
  - hàng đợi Valkey/BullMQ;
  - **tóm tắt định kỳ mỗi N phút thay cho gọi provider theo từng sự kiện**, để tiết kiệm hạn mức free-tier;
  - danh sách model fallback khi hết quota;
  - observation bằng tiếng Hàn;
  - tìm kiếm bỏ trợ từ tiếng Hàn;
  - **đệm sự kiện hook tại chỗ khi server sập**, phục hồi và đưa lại job bị mất vào hàng đợi;
  - dashboard không có auth (README khuyên bind Tailscale/127.0.0.1).
- **Trạng thái:** còn sống tới 2026-09-22 (commit mới nhất thấy được).

```
ForkChange { project: claude-mem, fork: tt12b/claude-mem, fork_base: không xác định (suy luận: server-beta, sau 2026-05-08),
  commit_or_series: README + 17a5754 "advance watermark only by what was in the prompt", ba5799f "reclaim abandoned claims into outbox", 0c321c0, 39bf753 (2026-09-10..22) [E0],
  original_upstream_behavior: worker local gọi observer cho từng sự kiện tool (Claude batch=1, SettingsDefaultsManager.ts:451-453); bàn giao spool khi vào buffer RAM (shared.ts:335-336) [E1],
  changed_behavior: tóm tắt theo chu kỳ; watermark chỉ tiến theo phần đã thật sự nằm trong prompt; claim bỏ dở được thu hồi về outbox [E0],
  problem_addressed: chi phí provider (free-tier) + mất dữ liệu khi con trỏ tiến trước kết quả,
  architecture_change: chuyển từ "một sự kiện → một lời gọi" sang "cửa sổ thời gian → một lời gọi"; outbox + watermark,
  complexity_added: Postgres + Valkey + 4 container,
  tests_added: không thấy được (trang commits không liệt kê tệp),
  runtime_evidence: không có,
  new_failure_modes: dashboard không auth (README tự cảnh báo); độ trễ memory = chu kỳ N phút (suy luận),
  current_upstream_equivalent: (a) gom lô opt-in d55d20f7e (2026-10-15, CLAUDE_MEM_CLAUDE_OBSERVATION_BATCH_SIZE mặc định 1, gom theo cùng prompt/agent/cwd); thân commit đo backlog ~4.800 sự kiện và "A worker restart discards whatever is still queued in memory" [E1/E0]. (b) checkpoint theo bản ghi của transcript watcher 2fcd3905f/ee763b95c (03 §9) [E1]. Không có bằng chứng upstream lấy từ fork; khả năng là tiến hoá song song,
  still_relevant: có — upstream vẫn mặc định một lời gọi mỗi sự kiện và buffer RAM,
  agent_cli_equivalent_problem: (câu hỏi) worker nền của agent-cli gọi LLM theo từng sự kiện hay theo lô? con trỏ tiến khi nào?,
  decision: ADAPT_IDEA (ứng viên; chưa audit agent-cli) }
```

```
ForkChange { project: claude-mem, fork: tt12b/claude-mem, fork_base: như trên,
  commit_or_series: README "local buffering of hook events when server is down, recovery and re-enqueue of lost jobs"; 315b393 "ask moves to next model on 503", 6b9dd18 [E0],
  original_upstream_behavior: thời điểm fork (server-beta), ghi lỗi tạm thời rơi xuống spool worker local, đọc không bao giờ đọc local (01 L7) [E1 HEAD],
  changed_behavior: đệm tại chỗ + đưa lại job đã mất; fallback theo danh sách model,
  problem_addressed: mất observation khi server không với tới; hết quota,
  architecture_change: client-side outbox cho runtime server,
  complexity_added: trung bình,
  tests_added: không thấy, runtime_evidence: không có,
  new_failure_modes: không rõ,
  current_upstream_equivalent: HookSpool bền (fe1709c30, 2026-10-03), spool tới worker từ xa (29685038b #4626, 2026-10-15), quota fallback (062946640 opt-in → 3dc125f2d mặc định 'claude') [E1],
  still_relevant: một phần (upstream đã có spool; fallback upstream đổi chính sách provider mặc định, 03 §15),
  agent_cli_equivalent_problem: (câu hỏi) agent-cli có outbox phía client khi backend không với tới không?,
  decision: DEFER }
```

### 2.2 rodboev — SCOPE_FIX (chuỗi heuristic monorepo)

- **Bối cảnh:** Rod Boev là tác giả upstream (78 commit, `02` §0). Chuỗi #2882 có ở hai nơi: (a) nhánh upstream community-edge, 1055ea404 → 0f54f6896, 9 commit ngày 06-24..06-30 (E1, `git log HEAD..origin/copilot/merge-all-features-to-community-edge -- src/utils/project-name.ts`); (b) fork, với `ef153dc`, `2b38daa` không có trong clone.
- **Commit fork `2b38daa`** (E0, trang commit): "require declared workspaces before splitting plain subprojects (thedotmack#2882)". Bỏ `hasNestedPackageJson`; chỉ tách thư mục con khi `package.json` gốc khai `workspaces`. Diff `src/utils/project-name.ts` +6/−52, test +14/−2.
- **Phân kỳ:** bản tích hợp community-edge (#3047, plan 07-07 dòng 185-189) ghi "derive repo-relative project keys when the repository declares workspaces **or contains nested package roots**" (E1). Fork thì **bỏ** nhánh "nested package roots". Nghĩa là heuristic monorepo đảo chiều ngay trong một chuỗi của cùng tác giả.

```
ForkChange { project: claude-mem, fork: rodboev/claude-mem (+ dòng community-edge upstream), fork_base: main ~2026-06-24 (mb của chuỗi 1055ea404),
  commit_or_series: 1055ea404, f464f945a, a90e45b6e, 1dbd7fd57, f83a73ab3, 530ff817a, f6fc0e6f9, 0f54f6896 (upstream edge) + 2b38daa (fork) [E1/E0],
  original_upstream_behavior: basename(git toplevel) cho mọi thư mục con (daf6d9dc0) → thư mục con monorepo mất memory (#2882) [E1],
  changed_behavior: khoá repo-relative cho package con khi repo khai workspaces (bản fork); edge: workspaces HOẶC có package lồng,
  problem_addressed: #2882 (monorepo),
  architecture_change: không — thêm heuristic vào resolver chung,
  complexity_added: phát hiện package.json, symlink realpath, phân biệt worktree tổ tiên không liên quan,
  tests_added: tests/utils/project-name.test.ts (cả hai nơi),
  runtime_evidence: không,
  new_failure_modes: heuristic phụ thuộc hệ sinh thái Node (repo không phải Node phải có nhánh riêng: f6fc0e6f9 "keep non-node repos on repo-root memory keys"); đổi khoá mặc định của thư mục con = mất memory kiểu #2882 theo chiều ngược,
  current_upstream_equivalent: main KHÔNG có; lối thoát là CLAUDE_MEM_PROJECT_ENVIRONMENTS opt-in (a27e248e3) + marker ngoài git (e192903e8). Thử khoá repo-relative trên main 107ef946e bị revert sau 14 giờ (02 §3) [E1],
  still_relevant: có — #4782 P1 vẫn mở, "needs a decision on project identity" [E0],
  agent_cli_equivalent_problem: (câu hỏi) agent-cli định danh workspace con trong monorepo thế nào, và người dùng khai báo được không?,
  decision: IGNORE heuristic; giữ bài học "khai báo tường minh thắng suy đoán" (ADAPT_IDEA cho phần khai báo) }
```

### 2.3 Fork nguồn PR — ý tưởng đã vào upstream (E1 thân squash)

```
ForkChange { project: claude-mem, fork: surfingdoggo/claude-mem, fork_base: main ~2026-08/09,
  commit_or_series: 38685a9 "fix(worktree): never adopt a key another checkout still owns" → nằm trong squash 276f845d2 (#2827, 2026-09-30) [E1 thân],
  original_upstream_behavior: sweep adoption coi khoá có dạng <parent>/<child> mà checkout đã mất là worktree mồ côi,
  changed_behavior: chỉ adopt khi mọi checkout đã ghi nhận còn tồn tại đều resolve vào repo này, hoặc khi không có ghi nhận thì khoá phải giữ hàng do chính máy này ghi,
  problem_addressed: slug git-remote org/repo có cùng hình dạng với composite <parent>/<worktree> → bị nuốt nhầm,
  architecture_change: bằng chứng ghi nhận checkout (sdk_sessions.cwd) làm điều kiện migration,
  complexity_added: thấp-trung bình, tests_added: resolver test org==repo (thân commit),
  new_failure_modes: không thấy,
  current_upstream_equivalent: có, cộng keySource (5beeceedd #4291),
  still_relevant: đã hấp thụ,
  agent_cli_equivalent_problem: (câu hỏi) migration của agent-cli có dựa vào hình dạng chuỗi để đoán nguồn gốc dữ liệu không?,
  decision: OBSOLETE (đã ở upstream); bài học → 03 I1 }
```

```
ForkChange { project: claude-mem, fork: escooterclinic/claude-mem, fork_base: main ~2026-09,
  commit_or_series: 5b9ca92 "fix(context): replace the ROW SOURCE, not the hook's output" → 817825876 (#4112, 2026-10-01) [E1 thân],
  original_upstream_behavior: SessionStart ở runtime server dựng output riêng, lệch renderer/ngân sách local,
  changed_behavior: chỉ thay nguồn hàng (server rows map vào row shape local); renderer, fitContextToBudget, token counter giữ nguyên; server runtime không mở SQLite local,
  problem_addressed: hai renderer cho hai runtime → hai hành vi; #2991 lớp "runtime đọc nhầm store",
  architecture_change: tách "nguồn dữ liệu" khỏi "trình bày" (một renderer, nhiều nguồn),
  tests_added: 9 test row source/row mapping (thân commit),
  new_failure_modes: không thấy,
  current_upstream_equivalent: có,
  still_relevant: đã hấp thụ,
  agent_cli_equivalent_problem: (câu hỏi) agent-cli có nhiều đường dựng context theo backend không?,
  decision: OBSOLETE; bài học: một renderer + ngân sách, nhiều nguồn }
```

Các fork nguồn PR khác (yetanotherflo: OpenCode #3767/#3803 và setting nguồn SessionStart #4251; DanielCarmingham: ghi nhận checkout v53 + reconcile `f08f145fb`; ntdatt812: #3802; stantheman0128: #3274; zichen0116: #4087) không thêm ý tưởng ngoài bản đã merge. Xem 03 §6, §7, §11, §12.

### 2.4 Mirror / không đủ bằng chứng
- **tqtensor:** 1.840 commit, README giống upstream, không nêu mục đích (E0). Trang #1505 ghi "Fork copies by tqtensor (May 6, 2026): 5a42006 and 6d67a85, with the same messages" → MIRROR (bản chép upstream khoảng 2026-05-06).
- **shantanukhanwalkar:** 2.506 commit, không hiện commit gần nhất (E0). Commit `5611dc5` "HAR-1103: Trim contract 7 hook tests to upstream's exit-0 path" (2026-10-06, trang #3523/#2966). **Suy luận:** fork nội bộ theo ticket, bám upstream. Không soi được nội dung → INSUFFICIENT_EVIDENCE.
- **y1024:** `daf6d9d` trùng SHA rút gọn của commit upstream `daf6d9dc0` → MIRROR.

---

## 3. Nhánh remote của upstream (763)

### 3.1 Số liệu (lệnh vừa chạy)
- `git for-each-ref refs/remotes/origin` (bỏ `origin/HEAD`): 763 nhánh. Với từng nhánh, `git merge-base --is-ancestor <tip> HEAD`: **121 MERGED, 642 NOT_MERGED**.
- NOT_MERGED ≠ bị từ chối. Upstream squash nhiều: so khớp tiêu đề commit đầu nhánh với `git log --format=%s HEAD` thì **136/642** nhánh NOT_MERGED có tiêu đề xuất hiện ở main. Đây là con số gần đúng; còn bỏ sót các squash đổi tiêu đề.
- Theo tiền tố: (gốc) 137, `fix/` 127, `copilot/` 80, `cursor/` 65, `posthog-self-driving/` 64, `feat/` 44, `bugfix/` 42, `feature/` 36, `claude/` 31, `thedotmack/` 26, `posthog-code/` 14, …
- Tiền tố của công cụ agent (`copilot`, `cursor`, `posthog-self-driving`, `claude`, `posthog-code`, `codex`) cộng lại 80+65+64+31+14+4 = **258**. Ví dụ đã giao bằng squash: `cursor/session-identity-stable-*` → `#4027` ở main; `cursor/ship-3716-worktree-ancestry-9c7d` → `4b317b139` (#4005) (E1 so tiêu đề).

### 3.2 Nhánh đáng kể

| Nhánh | Ngày tip | HEAD..nhánh | Nội dung | Vì sao không vào main | Main có tương đương về sau? |
|---|---|---|---|---|---|
| `origin/branch-memory` | 2026-03-16 | 23 commit, 48 tệp, +3349/−85 | Gắn `branch` + `commit_sha` cho observation/summary; SessionStart và MCP search chỉ hiện observation có commit là tổ tiên HEAD (`git merge-base --is-ancestor`); fallback hiện tất cả khi không phải git/detached/shallow/lỗi; Chroma metadata; 12+ tệp test | PR #1246 đóng 2026-04-15: "Closing during backlog consolidation", refile #2038 (E0) | **Không**: không có danh tính branch ở HEAD (01 §4.2) |
| `origin/canary`, `origin/bugfix/search-bugs-1` | 2026-04-18 | 32 | Gồm `64fc4f406`: "#1911: Default project filter to current project in normalizeParams()" | Không rõ; issue #1911 vẫn đóng "completed" nhờ `ba1ef6c42` (chỉ lọc khi CÓ project) | **Không**: thiếu project = toàn cục, test khoá (03 §2.3, §5) |
| `origin/copilot/fix-default-project-name` | 2025-12-14 | 5 | `getMostRecentProject()` làm project mặc định cho `getRecentContext` thay cho `basename(process.cwd())` của worker (thư mục plugin) (`279d316f1`) | UNKNOWN | Không: `/api/context/recent` vẫn dùng cwd của worker (01 L5) |
| `origin/copilot/add-project-level-memory-option` | 2025-12-14 | 6 | `.claude-mem.json` cấu hình theo project | UNKNOWN | Có, theo cách khác: override `projectName` (#3005, trên edge), marker ngoài git (`e192903e8` #3194, main 09-30) |
| `origin/fix/sdk-boundary-isolation` | 2026-05-28 | 4 | Cô lập SDK + trần `MAX_OBSERVATION_BYTES` + mở rộng skip-list; rồi "revert: drop byte cap and skip-list extension band-aids" | Phần cô lập vào main qua `46d204ee9`; phần trần bị coi là "band-aid" (maintainer trên #2201: "silent payload truncation isn't the right fix") | Cô lập: có (`hardened-options.ts:229-230`); trần: không |
| `origin/bugfix/security-hardening` (cả `evals-locomo`, `mac-toasts`, `claude/silly-spence`) | 2026-02-22 | 61 | `a4dbae24f`: lọc tag nguy hiểm ở tầng render + "mark observations as unverified" (audit #1050); `7ce22b5d2` bearer auth; `5a37db1e3` quyền tệp | UNKNOWN cho framing. Bearer auth vào main 04-20 (#1932) rồi bị gỡ trong cùng đợt: "added friction … with no benefit" (thân `03748acd6`) | Framing SessionStart: **không** (03 §14); chỉ Grok/search có khung |
| `origin/enhancement/supersession` | 2025-12-20 | 1 | Báo cáo hệ "supersession nửa vời": cột `status` (`active`/`superseded`/`deprecated`/`meta_observation`), `superseded_by`; API và skill đã bị gỡ; kèm bảng xếp hạng 18 feature request (hạng 1: "Git Branch Metadata") | Báo cáo để ra quyết định, không thấy quyết định | **Không**: `grep superseded_by src` = 0 (ngoài sync/server) |
| `origin/evals-locomo` | 2026-09-12 | 88 (285 tệp) | Prototype đánh giá LoCoMo: script ingest/eval; `qa-prototype-scored.json` có 20 câu, F1 trung bình **0,041** (tính lúc này bằng `python3 -I` trên tệp của nhánh) | UNKNOWN; nhánh lẫn nhiều việc khác | Main chỉ có `evals/read-gate/` (eval cho File Read Gate, `bd847dff8` #4549) và playbook kế hoạch `.maestro/playbooks/Wizard-2026-02-22/2026-02-22-LoCoMo-Eval/*.md`. `git ls-tree -r HEAD \| grep -i locomo` không ra script hay kết quả LoCoMo nào ngoài playbook → không có eval lợi ích memory trên main |
| `origin/fix/514-orphaned-observer-sessions` | 2026-01-01 | 3 | "always pass deterministic session ID to prevent orphaned files" | UNKNOWN | Cùng lớp với #2909 (06/2026); main sửa bằng `--no-session-persistence` (`641ff245f`, 06-15) → lỗi biết từ 01/2026, sửa sau 5 tháng |
| Dòng community-edge: `origin/plan/community-bleeding-edge` (143), `origin/copilot/merge-all-features-to-community-edge` (639), tag `archive/community-edge` (tip 07-09, 176) | 07-05..07-09 | xem cột trái | Theo plan 3 nhánh (`plans/2026-07-05-three-release-branches.md`): community-edge = "integrated community PRs", chạy từ source, không lên npm. Chứa: chuỗi monorepo Rod Boev (#3047), fold submodule #2856 + revert cùng ngày (`be4dc3b12`), namespace `:dream` (#2858, `5cf48a591`, `92bc32c48`: `isDreamProject = project.endsWith(':dream')`), git-remote #2827, environments #2737 | Dòng đã lưu trữ thành tag `archive/community-edge` | Một phần: #2827, #2737 vào main 09-30 dạng opt-in; dream/monorepo/submodule-fold không vào |
| tag `archive/core-dev` | 2026-10-11 | 7 | "root-cause holistic fixes" (plan); 7 commit chưa ở main: tiếp tục transcript Codex sau archival, chống mã hoá đôi payload tool, cảnh báo đếm buffer khi teardown | Dòng lưu trữ | UNKNOWN |
| `origin/pre-queue-engine-rewrite` | 2026-05-02 | 79 | Ảnh chụp trước đợt viết lại hàng đợi | Ảnh chụp | n/a |

### 3.3 Điều nhánh cho thấy (E1 + suy luận ghi rõ)

1. **Bản sửa đúng ý có thể nằm trên nhánh mà issue vẫn đóng "completed"** (#1911 → `canary`). Trạng thái issue không đủ làm bằng chứng (03 §2.1).

2. **Dòng community-edge là nơi upstream đặt thay đổi danh tính rủi ro.** Khoá chuẩn trên main `107ef946e` bị revert 15:24 ngày 07-05, ngay trước khi merge `plan/stable-curated` và release v13.10.2 (15:42–15:47). Hai ngày sau, khoá repo-relative cho monorepo vào edge (`bb1c84b4f`). **Suy luận:** đổi danh tính bị đẩy khỏi stable.

3. **Plan edge 07-07 tự ghi** (dòng 207-208): "#2737, #2827, #2665: competing project-name/environment models; require one chosen project identity design" (E1). Main sau đó nhận cả hai dạng opt-in (09-30). Đây là bằng chứng mạnh nhất cho câu "vì sao lỗi danh tính cứ quay lại" (03 §3).

4. **Namespace `:dream`** (UNKNOWN của 01 §8) đến từ PR #2858 của Rod Boev, tích hợp trên community-edge, không có ở main (E1). Nó là khoá chuỗi có hậu tố (`<project>:dream`) cho memory dẫn xuất. Đây lại là một biến thể "nhiều nghĩa trong một chuỗi khoá".

5. **Ý tưởng kiến trúc không sống sót, xếp theo giá trị cho V3:**
   - branch-scope theo tổ tiên git;
   - supersession/lifecycle;
   - khung untrusted ở tầng render;
   - eval lợi ích memory.

   Cả bốn đều được làm một phần rồi bỏ khi gom backlog hoặc không có quyết định (E0/E1). Main hiện đi hướng **hợp nhất lúc đọc + opt-in + vá theo kênh**.

---

## Hợp đồng đầu ra (spec §63)

- **SCOPE:** fork có ý nghĩa của claude-mem (overlay §8–10) và 763 nhánh remote trong clone.
- **REVISION AUDITED:** `3fae1eb45ba1a7fbd4818527f48b439991a273da`.
- **FILES/COMMITS/ISSUES INSPECTED:**
  - 6 trang fork (đầu tệp).
  - SHA fork trong timeline của 35 trang issue/PR (03 §0).
  - `git for-each-ref` toàn bộ nhánh, kèm merge status và số commit HEAD..nhánh cho cả 763.
  - `git log/show` các nhánh `branch-memory`, `canary`, `copilot/fix-default-project-name`, `copilot/add-project-level-memory-option`, `fix/dynamic-project-name-detection`, `fix/sdk-boundary-isolation`, `bugfix/security-hardening` (`a4dbae24f`), `enhancement/supersession`, `evals-locomo`, `fix/514-orphaned-observer-sessions`, `cursor/session-identity-stable-*`, `cursor/ship-3716-worktree-ancestry-9c7d`, các nhánh `fix/*` ngày 09-30, dòng community-edge, `archive/core-dev`.
  - Tệp `plans/2026-07-05-three-release-branches.md` (main) và `plans/2026-07-07-community-edge-batches-4-9-integration.md` (đọc qua `git show bb1c84b4f:`).
  - Commit `107ef946e`, `77869056e`, `bb1c84b4f`, `be4dc3b12`, `92bc32c48`, `279d316f1`, `276f845d2`, `817825876`, `d55d20f7e`.
- **VERIFIED FACTS:**
  - số nhánh MERGED/NOT_MERGED và số theo tiền tố (lệnh vừa chạy);
  - nội dung và trạng thái ancestry của các nhánh ở §3.2 (E1);
  - nội dung trang fork (E0);
  - 24/25 SHA fork trong timeline không có trong clone.
- **INFERENCES:**
  - tt12b dựng trên server-beta;
  - đổi danh tính bị đẩy sang edge vì rủi ro;
  - tqtensor là bản chép upstream;
  - shantanukhanwalkar là fork nội bộ theo ticket;
  - ước lượng ≤1.525 fork active;
  - không có bằng chứng upstream học từ tt12b (tiến hoá song song).
- **UNKNOWNs:**
  - số commit chỉ-có-ở-fork của tt12b/tqtensor/shantanukhanwalkar (không còn ngân sách cho trang compare; trang repo không hiện ahead/behind);
  - lý do không merge của `canary`, `bugfix/security-hardening` (framing), `enhancement/supersession`, `evals-locomo`;
  - nội dung PR #2038;
  - kết quả các playbook LoCoMo trong `.maestro` (chỉ có kế hoạch ở main).
- **FAILURE MODES:**
  - PROJECT_IDENTITY_FAILURE (heuristic monorepo đảo chiều, khoá `:dream`, slug ≡ composite);
  - BRANCH_CONTAMINATION (thiết kế branch-scope bị bỏ);
  - STALE_MEMORY/INVALIDATION_FAILURE (supersession bỏ dở);
  - MEMORY_INSTRUCTION_INJECTION (framing không merge);
  - WRONG_SCOPE_RETRIEVAL (bản mặc định scope nằm trên nhánh);
  - OBSERVABILITY_FAILURE (dashboard fork không auth).
- **AGENT CLI RELEVANCE:** chỉ là câu hỏi trong từng ForkChange. KHÔNG kết luận phơi nhiễm. Ứng viên ý tưởng (chưa duyệt):
  - lô/chu kỳ cho worker LLM nền với con trỏ chỉ tiến theo phần đã nằm trong prompt (tt12b);
  - một renderer, nhiều nguồn hàng (escooterclinic);
  - migration dựa trên bằng chứng ghi nhận, không dựa trên hình dạng chuỗi (surfingdoggo);
  - tri thức gắn revision, hiển thị theo tổ tiên git (`branch-memory`).
- **PROPOSED TESTS:**
  1. Tri thức ghi trên nhánh A không được tiêm như hiện hành trên nhánh B không chứa commit A.
  2. Migration không đổi khoá của dữ liệu có nguồn khác hình dạng.
  3. Worker nền theo lô: giết giữa chu kỳ → không mất, không xử lý đôi.
  4. Kênh render context dùng chung cho mọi backend: cùng đầu vào → cùng đầu ra và cùng ngân sách.
- **NO-CHANGE FINDINGS:**
  - soi 13 fork: 1 thay đổi kiến trúc (tt12b, biến thể triển khai), 1 sửa scope có phân kỳ (rodboev), 7 nguồn PR đã hấp thụ, 2 mirror, 2 không đủ bằng chứng;
  - không fork nào trong số đã soi giải được bài danh tính bằng một resolver chuẩn.
