# FINAL DAILY WORK AUDIT (§91)

> Báo cáo dưới giữ nguyên kết quả lịch sử ngày 16/10. Bản tiếp quản, README
> đã soát lại và receipt ngày 19/10: [báo cáo 39](../system-evolution/39_TAKEOVER_2026-10-19.md).

> **Revision cuối (19/10):** `main` @ `4a0b25b6` + `reports/system-evolution/evidence/takeover-2026-10-19/takeover-final.patch` (sha256 `f4201cfd…`,
> cây `be56dccb`) + `reports/system-evolution/evidence/followup-2026-10-19/crushrc-followup.patch` (sha256 `e607f9de…`, chỉ comment). Mã sản phẩm
> giống hệt vòng 8, đã chứng minh bằng so cây trên index tạm (SE [37](../system-evolution/37_FINAL_PROOF.md), mục cuối). Chưa
> commit. Các mục dưới nói "revision này" là nói bản trên; số đo của vòng 8 vẫn áp cho nó vì mã không đổi.

Revision: ứng viên cuối `scratchpad/evo/r8-final.patch` (sha256 `e4081c0b…`, 88 tệp; bản lưu bền
`reports/system-evolution/evidence/r8-final.patch`) trên `main` @ `4a0b25b6`, chưa commit. Bằng chứng chạy trên đúng revision này: SE [37](../system-evolution/37_FINAL_PROOF.md). Báo cáo System Evolution:
[FINAL_SYSTEM_AUDIT](../system-evolution/FINAL_SYSTEM_AUDIT.md).

## Trả lời §91

**agent-cli bây giờ là gì?** Một TUI + CLI lập trình chạy trong terminal (fork của Crush): nhiều dự án, mỗi dự án một
server nền, nhiều session mỗi dự án; provider là backend thực thi; `run` cho script, `loop`/`routine` cho việc có lệnh
verify; sổ ghi LLM để chứng minh lời gọi nào đi đâu.

**Vì sao tồn tại?** Để một người làm FE/BE/devops làm việc hằng ngày ở MỘT chỗ, với model local hoặc provider qua key, mà
biết việc nào đang chạy, chạy bằng model nào, đã kiểm chứng tới đâu (README).

**Có thật sự làm nơi làm việc chính hằng ngày được không?** Được, với giới hạn ghi rõ: luồng mở dự án → chọn session → giao
việc → xem → kiểm → mai tiếp đều có trong agent-cli và đã chạy (13 kịch bản LLM thật; E2E bằng binary). Còn lệch: model là
của workspace chứ không của session (PRB-05); chế độ không người trực là danh sách chặn, không phải sandbox; cổng WIN của
roadmap ("người dùng đi trọn daily workflow trên launcher thật với provider thật") cần người dùng tự đi.

**Workspace sở hữu gì?** Gốc dự án, server nền, DB `.crush/` (session, tin nhắn), job bash nền (gốc), log, model đang chọn
của dự án (`.crush/crush.json`).

**Session sở hữu gì?** Lịch sử hội thoại, To-Do, chuỗi lượt; job bash có chủ là session (chỉ session chủ đọc/dừng).

**Run sở hữu gì?** Một prompt tới câu trả lời cuối: model và bộ tool ghim lúc bắt đầu, `RunID` để huỷ đúng lượt, chuỗi
lượt tự làm tiếp (trần tóm tắt 10, trần làm tiếp sau `length` 3), lỗi cuối cùng.

**Cái gì thuộc agent-cli?** Workspace, session, run, quyền, tool, skill (quét + liệt kê), context file, verify của
loop/routine, bằng chứng (journal, báo cáo, sổ ghi), lựa chọn provider/model của người dùng.

**Cái gì thuộc provider?** Chỉ việc chạy model cho từng lời gọi. Provider không giữ trạng thái dự án; đổi provider không mất
mạch.

**Ranh giới provider có được ép thật không?** Có, ở một điểm chặn duy nhất (`buildAgentModels`): lời gọi ẩn ở lại provider
chính; provider không dùng được hay bị tắt thì dừng có lỗi, không đổi; allowlist (nếu đặt) áp cho cả model chính lẫn nhỏ.
Bằng chứng: faults 11 ca glm = 0; hai ca `selected-no-key` (local và qua server) fail closed, lỗi nêu provider ([10](10_PROVIDER_ISOLATION_RESULTS.md)). Giới hạn:
không đặt allowlist thì cấu hình (kể cả của repo lạ) chọn được provider khác; allowlist chỉ so tên; key đã export làm
provider đó chọn được.

**Chỉ-Qwen có chứng minh được 0 lời gọi trả tiền ngoài ý muốn không?** Cho các đường gọi LLM mà kịch bản đi qua: có, đếm
độc lập ở server giả và sổ ghi LLM (12/12 request ở qwen, 0 ở glm, 5 vai trò). Không phủ: kết nối không phải LLM khi đã
login ChatGPT/Grok (làm mới token, danh sách model); đường giao việc `claude_code`/`codex` (mặc định tắt).

**Đổi provider có chủ ý mà không mất mạch được không?** Được: s13 (bước 2 `-m` sang provider khác, cùng session) đạt với
LLM thật. Đổi là của cả dự án.

**Skill được chọn thế nào?** Model đọc danh sách tên + mô tả trong system prompt và tự `view` SKILL.md khi thấy hợp. Không có
cơ chế "luôn bật" ([05](05_SKILL_TRACE.md), [11](11_SKILL_EFFECTIVENESS.md)).

**Quy ước nào thành chính sách thay vì skill?** Đề xuất (chờ duyệt): luật "luôn áp dụng" (đòi bằng chứng, không tự commit,
xác minh trước khi báo xong, khẩu vị ponytail cốt lõi) đưa vào `~/.config/agent-cli/CRUSH.md` (context toàn cục, nạp mọi
phiên) và tắt 21 skill. Trong mã: "xong" = mã thoát verify, `NO_CHANGE`, lượt rỗng là lỗi: đó là chính sách máy ép.

**Ponytail và Superpowers nằm ở đâu?** Bộ skill tuỳ chọn; runtime không phụ thuộc. Phần chạy được ở agent-cli:
`systematic-debugging`, TDD, `ponytail-review`. Phần không chạy được: các skill dựa trên agent con ghi mã.

**Khi nào nhiều agent có ích?** Ở bộ đo này: chưa thấy. Cùng một việc, nhiều agent tốn hơn nhiều lời gọi và thời gian mà kết
quả như một agent ([12](12_MULTI_AGENT_RESULTS.md)).

**Đã đạt "số agent tối thiểu đủ dùng" chưa?** Có: agent con chỉ chạy khi model gọi tool `agent`; session con không còn gọi LLM
đặt tiêu đề.

**Worktree thuộc ai?** Người dùng (repo của mình); routine tạo worktree riêng `routine/<tên>-<thời điểm>` mỗi lượt, giữ lại
để người duyệt, kèm lệnh dọn.

**Routine có dùng lại runtime không?** Có: `routine` → engine `loop` → agent in-process qua cùng coordinator; không hàng đợi
hay worker riêng ([13](13_ROUTINE_RECOVERY_RESULTS.md)).

**Routine có khôi phục được không?** Có: kill giữa lượt → `ABANDONED`, khoá nhả, lượt sau chạy được (E2E routine mục 8); lỗi
sớm → `ERROR`; đang chạy → `RUNNING`.

**DONE là gì?** Mã thoát của lệnh verify trong `loop`/`routine` (GREEN); verify xanh sẵn mà không đổi gì là `NO_CHANGE`. `run`
thoát 0 chỉ nghĩa là không có lỗi runtime. GREEN không chống được việc agent sửa chính test: phải duyệt diff (README).

**Kiểm chứng làm thế nào?** Lệnh verify do driver chạy sau mỗi lượt, với `pipefail`; báo cáo routine gắn cờ khi agent đụng
test/cổng; ngoài sản phẩm: E2E bằng binary và cổng LLM giả trong `scripts/`.

**Bằng chứng có còn tươi không?** Có: mọi số trong SE [37](../system-evolution/37_FINAL_PROOF.md) chạy trên revision cuối;
bằng chứng cũ hơn được ghi rõ là của revision trước.

**Memory nhớ gì?** Context file (người viết, git duyệt), lịch sử session, bản tự tóm tắt ([14](14_MEMORY_DAILY_VALUE.md)).

**Memory không bao giờ được đè cái gì?** Mã, test, hành vi runtime hiện tại. agent-cli không ép điều này bằng máy; đó là thứ
tự tin ghi trong README.

**Ý tưởng nào của Hindsight còn lại sau đánh giá?** 7 bất biến vòng đời/bằng chứng (SE [28](../system-evolution/28_HINDSIGHT_DECISIONS.md)).

**Ý tưởng nào bị bác/hoãn?** Bác: memory bank, retain, reflect, consolidation, mental model. Hoãn: recall nhiều tầng, hiệu
lực thời gian, mang bài học giữa các lượt routine.

**Cái gì được làm đơn giản đi?** Chặn biến git bằng luật theo chữ thay cho phân tích phép gán; allowlist chỉ một chủ (tệp
env); không còn fallback chéo provider (bớt một nhánh quyết định).

**Cái gì bị bỏ?** [16](16_PRUNING_DECISIONS.md).

**Kiến trúc đã trôi ở đâu?** Model là của workspace thay vì session (C-2); ba bộ phân tích lệnh shell (G-05); fallback chéo
provider (đã sửa); verify "xong" có thể xanh mà không đo gì (đã sửa bằng `NO_CHANGE`).

**Cái gì vốn đã đúng?** Một điểm chặn provider; tách workspace/session; khôi phục tool call bỏ dở; `RunComplete` một lần;
agent con theo yêu cầu; M0 đủ cho tiếp việc.

**Ma sát hằng ngày nào còn lại?** [02](02_FRICTION_MAP.md): F-03 (context file chỉ ở cwd), F-04/F-05 (chính sách skill chờ
duyệt), F-10 (dọn worktree routine), F-12 (`run` không có trần thời gian), PRB-05 (model của workspace).

**Giới hạn nào còn?** `docs/agent-cli-local-setup.md`, mục "Giới hạn đã biết"; tóm tắt trong README.

**Còn gì chưa biết?** FINAL_SYSTEM_AUDIT câu 25.

**README/docs đã đồng bộ chưa?** [DOCS_CONSISTENCY_AUDIT](DOCS_CONSISTENCY_AUDIT.md), [README_REVIEW](README_REVIEW.md).

**agent-cli cuối có tốt hơn đo được không?** [15](15_BEFORE_AFTER_METRICS.md).

## Bằng chứng còn tươi (§95)

| Điều kiện hoàn tất | Trạng thái | Bằng chứng |
|---|---|---|
| System Evolution xong | Có | SE [FINAL_SYSTEM_AUDIT](../system-evolution/FINAL_SYSTEM_AUDIT.md) |
| Kiến trúc hiện tại đã kiểm | Có | [03](03_RUNTIME_TRACE.md), [08](08_CAPABILITY_OWNERSHIP.md), [09](09_TARGET_DAILY_ARCHITECTURE.md) |
| Kịch bản hằng ngày đã kiểm | Có (A–H, J–M; I là NOT_APPLICABLE) | [01](01_BASELINE.md), [15](15_BEFORE_AFTER_METRICS.md) |
| Workspace/session | Có | `workspaces.sh` PASS trên binary cuối |
| Cô lập provider | Có | `faults.sh` 11 ca PASS trên binary cuối |
| Hồi quy chỉ-Qwen | Có | `qwen-only.sh` PASS trên binary cuối |
| Khôi phục | Có | E2E routine (SIGKILL → `ABANDONED`) PASS trên binary cuối |
| Kiểm chứng lại sau khởi động lại (19/10) | Có: build, suite (4.637 test, 0 fail, 0 mất), 3 E2E, faults, qwen-only, workspaces, race đều PASS trên binary bản tiếp quản; vet chỉ còn cảnh báo có sẵn | SE [39](../system-evolution/39_TAKEOVER_2026-10-19.md), WORKFLOW_PROGRESS 42 |
| Kiểm chứng/bằng chứng hiện hành | Có | SE [37](../system-evolution/37_FINAL_PROOF.md): mọi số E3 và suite chạy trên `r8-final.patch`; eval E5 đầy đủ trên bản vòng 7, smoke E5 qua server trên bản vòng 8 |
| So trước/sau | Có | [15](15_BEFORE_AFTER_METRICS.md) |
| Hồi quy đã xử lý | Có | 8 lượt soát mã độc lập, lượt cuối DAT (SE [38](../system-evolution/38_RED_TEAM.md)) |
| Cắt bỏ | Có | [16](16_PRUNING_DECISIONS.md) |
| README / docs đồng bộ, kiểm nhất quán | Có | [DOCS_CHANGELOG](DOCS_CHANGELOG.md), [DOCS_CONSISTENCY_AUDIT](DOCS_CONSISTENCY_AUDIT.md) |
| Soát README ngữ cảnh mới | Có: READY ở hai người soát độc lập trên README bản tiếp quản (lượt 6 Codex, lượt 6b Claude), 16/16, 0 MUST-FIX | [README_REVIEW](README_REVIEW.md) |
| Báo cáo cuối đủ | Có | §90: 23 tệp trong `reports/daily-work/` |

Bản lưu bền của ứng viên: `reports/system-evolution/evidence/r8-final.patch` (+ `.sha256`), log then chốt trong
`evidence/final-logs/`. Patch áp sạch lên `4a0b25b6` (kiểm bằng `git apply --cached --check` trên index tạm dựng từ HEAD); để áp
vào cây làm việc chính phải bỏ thư mục `scripts/eval/` cũ chưa theo dõi trước (patch mang bản mới).

## Giới hạn còn lại khi chốt

- Cổng WIN của roadmap (người dùng đi trọn luồng hằng ngày trên launcher thật với provider thật) chưa có người dùng đi.
- Eval LLM thật chạy trên binary vòng 7; đường mã nó đi qua giống hệt bản cuối, và smoke LLM thật qua server đã chạy trên
  vòng 8. Bản tiếp quản không chạy lại LLM thật (mã sản phẩm không đổi).
- `go vet` còn copylocks ở `internal/csync/maps.go:156` (mã upstream, có từ trước). `task lint`/`golangci-lint` chưa có trên
  máy nên chưa chạy; đã chạy riêng gofmt của toolchain 1.27 và kiểm viết hoa log.
- README còn 14 NICE-TO-HAVE (README_REVIEW lượt 6b). Guide, mục "Giới hạn đã biết", còn mã PRB-xx trỏ vào `reports/`,
  thư mục này chưa vào git.
- Model là của workspace chứ không của session (PRB-05). Chế độ không người trực là danh sách chặn, không phải sandbox.
  Lời hứa provider chỉ phủ lời gọi LLM do agent-cli phát ra.
- Chưa commit. Áp bản cuối: `git apply` lần lượt hai patch trên `4a0b25b6`. Phải bỏ `scripts/eval/` cũ, chưa theo dõi, trước
  khi áp.

## Trạng thái cuối (§96)

System Evolution: **SYSTEM IMPROVED — WITH KNOWN LIMITATIONS** ([FINAL_SYSTEM_AUDIT](../system-evolution/FINAL_SYSTEM_AUDIT.md)).

Daily Workspace: **DAILY WORKSPACE READY — WITH KNOWN LIMITATIONS**

## Phụ lục sau chốt (19/10): lỗi V2 do test phơi nhiễm V3 tìm ra

Hai lỗi của chính mã `v2-final`, tìm ra khi đo phơi nhiễm cho V3 (`reports/v3-research/experiments/00_EXPOSURE_TESTS.md`),
điều phối chạy lại độc lập bằng binary tự build (`~/.cache/v3ev/m1.out`):

- **Server panic khi nhiều `run` đồng thời (T20).** Server đang chạy, 10 lệnh `run` cùng lúc: 4/10 lượt thử panic
  `sync: WaitGroup is reused before previous Wait has returned` (`internal/agent/coordinator.go:313`), kéo chết mọi client
  đang gắn (agent đo lần trước: 7/10; là race nên tỉ lệ dao động). 3 hoặc 5 client: 0/10 (theo lần đo của agent).
- **Allowlist / env của client thứ hai bị bỏ qua (C5).** Workspace đã do client đầu dựng thì client sau chạy theo env của
  client đầu; allowlist `qwen` của client sau không chặn được lượt đi provider khác. Khi chính client đó dựng server thì
  allowlist chặn đúng. Đây là lỗ của bất biến cô lập provider: allowlist chỉ chắc chắn có hiệu lực với env của tiến trình
  đã dựng server (`internal/backend/backend.go`, lần tạo workspace đầu thắng).

Trạng thái §96 giữ nguyên ("WITH KNOWN LIMITATIONS") nhưng hai mục trên là lỗi cần sửa, không phải giới hạn chấp nhận được;
đã đưa vào danh sách POC đầu tiên của V3 (EXTEND_V2). Chưa sửa.
- **(ĐÃ TÁI HIỆN, E3) `.env` của repo lạ chọn được chính sách tin cậy qua Home.** Soát an ninh POC P0-2 (19/10) đọc mã:
  `agent-cli` trần → Home nạp `.env` của cwd rồi dựng tiến trình con bằng `os.Environ()` (`internal/cmd/home.go:127,154`),
  nên `AGENT_CLI_ENV_FILE`, `AGENT_CLI_DELEGATION`, `AGENT_CLI_CLAUDE_BIN`… trong `.env` của repo thành env "tin cậy" của tiến
  trình con. Trái với docs V2 ("`.env` của repo không bật/tắt được"). Chưa đo trên binary; đang dựng test đỏ trong P0-2.
  Trong lúc chờ: đừng mở bằng Home một repo lạ có `.env`.
- **(ĐÃ TÁI HIỆN, E3) `loop -c ..` tương đối làm việc sai thư mục, local còn báo GREEN giả.** Gọi từ `repo/internal`,
  binary `v2-final`: chế độ local → agent (tự duyệt mọi tool) làm việc ở thư mục cách repo hai cấp, lệnh thoát `GREEN` rc 0;
  chế độ server → agent ở thư mục cha của repo, `MAX_REPAIRS`. Gốc: `-c` tương đối được tính lại theo cwd đã đổi
  (`ResolveCwd` gọi hai lần trong `loop`) và data dir ghép gốc hai lần. Đã sửa ở POC P2-b vòng 2 (NOT_EXPOSED), chưa merge.
  Đo: `scratchpad/p2b/loop-cwd.sh`, `~/.cache/v3ev/lcwd2-base.out`. Trong lúc chưa merge: dùng `-c` đường tuyệt đối.
