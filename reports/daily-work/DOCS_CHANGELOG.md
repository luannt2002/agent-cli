# Nhật ký thay đổi tài liệu (§85)

Phạm vi: các tệp `.md` trong ứng viên cuối (`scratchpad/evo/wt-b`, so với `4a0b25b6`): 11 tệp, đo bằng `git diff HEAD --stat -- '*.md'`. Cộng một dòng comment trong `agent-cli.crushrc.example`. Báo cáo của đợt (`reports/`) không tính ở đây: chúng chưa vào git, người dùng quyết có giữ không.

## Theo tệp

| Tệp | Đổi gì | Vì sao | Bằng chứng |
|---|---|---|---|
| `README.md` | Viết lại theo thứ tự "dùng hằng ngày trước" (§76): cài (kèm clone, `--version`), lần chạy đầu, một ngày làm việc, khái niệm (thêm model chính/nhỏ, **provider chính**), provider (dùng nhiều, không gọi lén, model không dùng được thì dừng, allowlist chỉ so tên), "Xong" nghĩa là gì (mã thoát, `NO_CHANGE`), loop/routine, memory M0, skill, cấu hình, phím, quan sát sâu, giới hạn. Bỏ ID nội bộ (PRB-xx, s13) và con trỏ mã nguồn | README ở `4a0b25b6` (86 dòng: cài, cấu hình, phím, khác upstream, phát triển) không nói ranh giới provider, "xong" nghĩa là gì, memory, skill, loop/routine. Hai lượt soát README với ngữ cảnh mới đều chấm NOT READY trước khi sửa: lượt 1 (thiếu cài/lần chạy đầu, "provider chính" chưa định nghĩa, ID nội bộ), lượt 2 (danh sách trường hợp tự duyệt quyền thiếu YOLO/Allow for Session và rủi ro repo lạ; mã thoát `NO_CHANGE` mâu thuẫn; routine không nêu `routine new/run` và mode; luật allowlist mơ hồ; không có cách thêm provider thứ hai; không nói model của dự án nằm ở `.crush/crush.json`). Lượt 3 còn 4 mục (dòng `send` trong bảng mã thoát sai, câu ranh giới provider viết như vô điều kiện, GREEN thiếu cảnh báo agent sửa được test, danh sách rủi ro repo lạ thiếu `.crush.json`/context file/skill). Sau mỗi lượt đều sửa và đối chiếu với mã/`--help`; lượt 4 ở [README_REVIEW](README_REVIEW.md) | [README_REVIEW](README_REVIEW.md) |
| `docs/agent-cli-local-setup.md` | Thêm mục "Cô lập provider, sổ ghi LLM, bằng chứng của loop/routine", "Model đang chọn không dùng được: dừng, không đổi provider", tự tóm tắt (trần 10, chốt tràn chỉ xét đầu vào), `run --json` (`text` chỉ câu trả lời; nhãn provider không phải bằng chứng), loop `NO_CHANGE`, routine `RUNNING`/`ABANDONED`/`ERROR`, lịch không chép allowlist, chế độ không người trực (phần thêm). Viết lại "Giới hạn đã biết" | hành vi mới của ứng viên | test và E2E nêu trong từng mục |
| `CHANGELOG.md` | Mục `[Unreleased]`: Thêm (`loop`, `routine`, chế độ không người trực, sổ ghi LLM, `run --json` provider/model, `scripts/eval/`), Sửa lỗi (lời gọi ẩn, tóm tắt, `run` báo thành công giả, `run` local in bản tóm tắt, model không dùng được thì dừng, …) | Keep a Changelog | số đo trong từng dòng lấy từ faults/qwen-only |
| `DECISIONS.md` | Thêm các quyết định 16/10: lời gọi ẩn ở lại provider chính, sổ ghi LLM, trần tóm tắt, lượt rỗng là lỗi, `agentic_fetch`, luật theo chữ cho biến git, M0, chính sách skill (đề xuất), fallback không ghi bền (đã lật một phần), **model không dùng được thì dừng**, **`NO_CHANGE`**, **allowlist của lịch chỉ một chủ** | ghi lý do + "lật lại khi" | — |
| `CLAUDE.md` | P1 giai đoạn (đã phát hành v0.1.0/v0.2.0; loop/routine ra sớm); P2 bỏ plan đã xong; trạng thái ledger `.superpowers/sdd` và `stash@{0}` đúng thực tế; P5 thêm E2E loop/routine và cổng `scripts/eval` | dòng cũ sai so với đĩa | đã kiểm `ls`, `git stash list` |
| `AGENTS.md` | `catwalk` là catalog provider (không phải golden test); config là `*config.ConfigStore`; đường dẫn hooks, ví dụ lệnh test trỏ tới tệp tồn tại | mô tả cũ của upstream sai so với mã | đọc mã |
| `internal/ui/AGENTS.md` | `common.Common` giữ `workspace.Workspace`, không phải `*app.App` | sai so với mã | đọc mã |
| `agent-cli.crushrc.example` | một dòng comment: "glob/grep không có lời hỏi nào" → "glob/grep cũng chỉ hỏi khi tìm NGOÀI dự án" | sai so với mã: `glob.go`, `grep.go` gọi `requestOutsideAccess` (người soát README vòng 1 bắt mâu thuẫn với guide) | `grep -n requestOutsideAccess internal/agent/tools/{glob,grep}.go` |
| `docs/config/README.md` | ghi chú fork: config ở `~/.config/agent-cli/`, lệnh là `agent-cli` | người dùng fork đọc nhầm đường dẫn upstream | — |
| `docs/hooks/README.md` | `crush` → `agent-cli`, đường dẫn config, timeout mặc định 120 | sai so với fork | — |
| `docs/reviews/2026-10-14-phase1-open-issues.md` | ghi chú "lỗi thời, giữ làm lịch sử"; C1 (PRB-04) đã đóng | tài liệu lịch sử, tránh người đọc làm theo | — |
| `docs/superpowers/plans/2026-10-14-agent-cli-personal-roadmap.md` | ghi chú: loop/routine ra sớm trước Phase 2–3 | roadmap thắng plan cũ, phải khớp thực tế | — |

## Vòng 7–8 (sau các lượt soát)

| Tệp | Đổi gì | Vì sao |
|---|---|---|
| `README.md` | Phạm vi của lời hứa provider: chỉ phủ lời gọi LLM do agent-cli phát ra; đường qua tool (CLI đổi tên, `python -c`, `curl` với key có sẵn) không bị chặn và sổ ghi LLM không thấy. Lần chạy đầu vào Home. Tệp env mang cả allowlist/delegation. Bảng tên biến key ở `docs/UPSTREAM-README.md`. Log qua `agent-cli logs`/`dirs`. Khối `schedule` trong tệp định nghĩa (`~/.config/agent-cli/routines/<tên>.json`). `--commit` chỉ cho phép commit. `jobs`/`logs` theo thư mục dự án. `.env` của thư mục đang đứng được nạp. Cập nhật phải thoát client. Plan mode lan sang `run`/`loop`. `run -m` chỉ ghi model ở chế độ server. Skill của repo ở 4 thư mục | soát README lượt 4, soát mã vòng 7–8 (mỗi câu đã đối chiếu mã/`--help`) |
| `docs/agent-cli-local-setup.md` | Model đang chọn không dùng được: app vẫn khởi động không agent, TUI vào màn chọn model, `login` chạy; giới hạn đổi thành "`run`/`loop`/`routine` dừng, kể cả có `-m`"; `text` local khác server | vòng 7 |
| `CHANGELOG.md`, `DECISIONS.md` | Mục F1 cập nhật theo vòng 7–8 (app khởi động không agent; provider bị tắt bị từ chối; lỗi tới client qua server); gạch "Vì" của F1 viết lại | vòng 7–8 |

## Tiếp quản 19/10 (bản `takeover-final.patch` + patch bổ sung)

| Tệp | Đổi gì | Vì sao |
|---|---|---|
| `README.md` | 15 câu sửa sau khi chốt vòng 8 (lượt soát README 5 và soát mã vòng 8), Codex khôi phục từ lịch sử phiên sau khi máy khởi động lại; đối chiếu từng câu với bản phía Claude: giống | `/tmp` mất khi khởi động lại |
| `docs/agent-cli-local-setup.md` | Đoạn `loop`: "Không commit, không push" → bản thân `loop` không commit/push nhưng model vẫn chạy được `git commit`/`git push`; chặn của routine chỉ theo mẫu lệnh | đồng bộ với README (reviewer Codex lượt 6) |
| `agent-cli.crushrc.example` | Comment giao việc `claude_code`/`codex`: mặc định TẮT, điều kiện bật, khi nào không hỏi quyền. Câu "Request chỉ đi tới provider…" → lời gọi LLM, kèm ngoại lệ OAuth | sai so với mã (soát README lượt 6b, N11); patch `reports/system-evolution/evidence/followup-2026-10-19/crushrc-followup.patch` |

## Thuật ngữ

| Thuật ngữ | Quy ước |
|---|---|
| model chính / model nhỏ | large / small trong cấu hình. Model nhỏ đặt tiêu đề và chạy `agentic_fetch` |
| provider chính | provider của model chính |
| lời gọi ẩn | mọi lời gọi LLM mà người dùng không gõ trực tiếp: tiêu đề, tóm tắt, agent con, `agentic_fetch` |
| `NO_CHANGE` | lý do dừng mới: verify xanh từ trước mọi thay đổi và cây làm việc không đổi |
| chế độ không người trực | `AGENT_CLI_UNATTENDED=1`, routine tự bật; danh sách chặn, không phải sandbox |

## Tuyên bố cũ đã gỡ

- "Fallback chéo provider lúc khởi động là giới hạn đã biết, giảm hại bằng allowlist": lỗi thời sau vòng 6, và red team chỉ ra allowlist không chặn được khi model mặc định của provider thay thế nằm trong danh sách.
- "GREEN mà verify xanh sẵn vẫn thoát 0" (loop/routine): thay bằng `NO_CHANGE`.
- "Lịch mang theo allowlist": thay bằng "allowlist đọc từ tệp env".
- "Chế độ local có thể chèn bản tóm tắt vào `text`" (giới hạn): đã sửa.
- Gợi ý `-m provider/model` khi model không dùng được: đo bằng binary là không gỡ được, đã bỏ khỏi thông báo lỗi và docs.
- "Model đang chọn không dùng được thì TUI không mở / `agent-cli login` vẫn chạy" (docs vòng 6): nửa đầu nay sai (TUI mở màn chọn model), nửa sau trước đây sai (login hỏng) và nay đúng (vòng 7, đo bằng binary).
- "`text` của `run --json` chỉ là câu trả lời cuối, cả local lẫn server": sai ở chế độ local (nối mọi chữ assistant); đã ghi đúng khác biệt.
- "Provider bị tắt thì dừng" (docs vòng 6): trước vòng 7 sai với provider dựng sẵn; nay đúng (`buildAgentModels` từ chối provider `Disable`).

## Tài liệu lịch sử

Không xoá tài liệu nào. Hai tệp lịch sử có ghi chú "lỗi thời, giữ làm lịch sử" ở đầu (`docs/reviews/2026-10-14-phase1-open-issues.md`, phần sửa đổi trong roadmap). Ledger `.superpowers/sdd` và `docs/progress/` không có trong git; CLAUDE.md ghi đúng vị trí thật của chúng.

## Giới hạn của chính tài liệu

- Mã PRB-xx trong `docs/agent-cli-local-setup.md` trỏ tới `reports/system-evolution/05_CURRENT_PROBLEMS.md`, tệp chưa vào git. Nếu không giữ `reports/` thì các mã này chỉ còn là nhãn.
- README tiếng Việt; README gốc của upstream giữ ở `docs/UPSTREAM-README.md`.
