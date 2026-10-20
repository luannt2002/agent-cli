# Soát README với ngữ cảnh mới (§86)

Mỗi lượt là một agent MỚI, loại không có Edit/Write, chỉ được đọc README cùng các tệp nó trỏ tới và chạy `--help` của
binary trong môi trường cô lập. Không đọc mã nguồn: README chừa trống chỗ nào thì đó là phát hiện. Từ lượt 2, người soát
phải trả lời đúng 16 câu hỏi của §86 chỉ từ README; trả lời sai một câu là NOT READY.

## Lượt 1 (README sau lần viết lại đầu, cùng workflow red team cuối)

**NOT READY.** Câu 11 ("cái gì ngăn Qwen lặng lẽ dùng Claude") còn mơ hồ: phần fallback chéo provider (PRB-04) được gọi là
"giới hạn đã biết, một phần" ngay trong mục "không gọi lén". Kèm theo: thiếu bước clone, lần chạy đầu, `--version`;
"provider chính" chưa định nghĩa; ID nội bộ (PRB-04, PRB-05, s13); con trỏ vào mã nguồn (`internal/ui/model/keys.go`).

Xử lý: thêm cài/lần chạy đầu, định nghĩa model chính/nhỏ và provider chính, bỏ ID nội bộ; fallback chéo provider bị bỏ
hẳn ở vòng 6 (fail closed) nên câu 11 có câu trả lời rõ.

## Lượt 2

**NOT READY.** 13/16 câu trả lời được; câu 9, 11, 13 chưa trọn. 6 mục phải sửa:

| # | Mục | Đã sửa thế nào |
|---|---|---|
| 1 | Danh sách trường hợp không có hộp thoại quyền thiếu YOLO, `shift+tab` (xoay tới "code + YOLO"), "Allow for Session", `permissions allow`; thiếu rủi ro repo lạ | Mục mới "Khi nào không có hộp thoại quyền" + đoạn "Repo lạ"; thêm YOLO vào Khái niệm |
| 2 | Mã thoát `NO_CHANGE` tự mâu thuẫn (loop/routine thoát 1, rồi routine maintenance thoát 0); thiếu mã thoát khác | Sửa câu; thêm bảng mã thoát (`run`/`send`, `loop`, `routine run`, kể cả 2 và 75) |
| 3 | Mục routine dùng mode `maintenance`/`goal` mà không định nghĩa; không nêu `routine new`/`routine run`; `"commit": true` trỏ tới định dạng không có trong README | Ví dụ `routine new … --mode maintenance … [--commit]`, `routine run`; định nghĩa hai mode; trỏ tới guide |
| 4 | Luật allowlist dùng "provider chính" mơ hồ | Viết lại đúng luật trong mã (`execpolicy.Parse`: có ít nhất một mục không phải `claude_code:`/`codex:` thì model chính và nhỏ phải nằm trong danh sách, bất kể lựa chọn đến từ đâu) |
| 5 | Nhiều provider nhưng không có cách thêm provider thứ hai; giới hạn không nằm cạnh lời hứa | Hai loại provider (`main` tự khai, dựng sẵn qua `AGENT_CLI_PROVIDER_ID`), provider thứ hai qua key riêng hoặc `login`; mục "Giới hạn của lời hứa trên" ngay sau lời hứa |
| 6 | Không nói model của dự án nằm ở `<dự án>/.crush/crush.json` (thắng env); gợi ý `agent-cli login` không nền tảng là Charm Hyper | Đoạn "Model của dự án nằm ở đâu"; `login openai\|grok\|copilot` |

Kèm 15 mục nên sửa; đã làm phần rẻ: Go ≥ 1.27, PATH bền, `agent-cli models`, PONG trong thư mục trống, lệnh cập nhật,
nguồn của số "41 lượt / 2 lượt", câu memory không còn đọc như bảo đảm máy ép, `ps` giải nghĩa trạng thái, sổ ghi LLM
không đọc từ tệp env. Người soát cũng bắt một comment sai trong `agent-cli.crushrc.example` (glob/grep "không có lời hỏi
nào"); đã sửa theo mã.

## Lượt 3

**NOT READY.** 16/16 câu trả lời được. 4 mục phải sửa:

| # | Mục | Đã sửa thế nào |
|---|---|---|
| 1 | Dòng `send` trong bảng mã thoát sai: không `--wait` thì thoát 0 ngay khi server nhận prompt | Tách dòng `send --wait` và `send` (không `--wait`: server đã nhận, lượt chưa chạy) |
| 2 | Bước 5 của luồng hằng ngày hứa "model chỉ chạy trong ranh giới provider" vô điều kiện, trong khi mặc định chưa đặt allowlist | Viết lại: model chạy ở provider đang chọn, agent-cli không tự chuyển; muốn khoá cứng thì đặt allowlist |
| 3 | GREEN được gọi là bằng chứng mà không cảnh báo agent (tự duyệt quyền) sửa được chính test/script verify | Thêm cảnh báo, trỏ tới cờ `touched_tests`/`touched_gate` của routine |
| 4 | Danh sách rủi ro repo lạ thiếu `.crush.json`, hook/MCP, `base_url`, context file và skill của repo | Bổ sung đủ, kèm "đừng `run`/`loop` trên repo chưa đọc" |

Phần nên sửa đã làm: nơi đặt key của provider thứ hai (export ở shell; tệp env chỉ cấu hình provider chính),
`visible-provider`, phím của hộp thoại quyền, `ctrl+c` thoát mà lượt vẫn chạy ở chế độ server, cách dùng `send`, nghĩa
output của `agent-cli models`, chặn của routine là theo mẫu lệnh, câu "một lần chạy không phải bảo đảm".

## Lượt 4

**NOT READY.** 15/16 câu trả lời đúng. Câu 11 chỉ đúng một phần: người mới sẽ tin quá mức. 1 mục phải sửa:

| # | Mục | Đã sửa thế nào |
|---|---|---|
| 1 | Lời hứa "không gọi lén" không nói nó chỉ phủ lời gọi LLM do chính agent-cli phát ra. Dưới `run`/`loop`/`routine`/YOLO, model có thể gọi provider khác qua tool (CLI đổi tên, `python -c`, `curl` với key/CLI đã đăng nhập trên máy); sổ ghi LLM không thấy | Gạch đầu tiên của "Giới hạn của lời hứa trên" nói rõ phạm vi và đường qua tool; câu "lời gọi LLM nào do agent-cli phát ra… là bug"; dòng chặn theo tên ghi "không chặn bản đổi tên hay script" |

Phần nên sửa đã làm (đã đối chiếu với guide/`--help` trước khi viết): bước 4 của lần chạy đầu mở Home chứ không vào thẳng
dự án; tệp env cũng mang `AGENT_CLI_ALLOWED_MODELS`/`AGENT_CLI_DELEGATION`; tên biến key của từng provider (trỏ bảng
trong `docs/UPSTREAM-README.md`); log đọc bằng `agent-cli logs`/`dirs`; lịch cần khối `schedule` trong tệp định nghĩa;
`--commit` tự commit nên duyệt diff "trước khi commit hay merge"; `jobs`/`logs` theo thư mục dự án; `.env` của thư mục mở
được nạp; cập nhật phải thoát client để server dựng lại; plan mode của dự án lan sang `run`/`loop`.

## Lượt 5

**NOT READY.** 16/16 câu trả lời được. 3 mục phải sửa:

| # | Mục | Đã sửa thế nào |
|---|---|---|
| 1 | Danh sách rủi ro repo lạ thiếu `.crush/crush.json`: repo có thể commit nó, nó thắng mọi cấu hình khác, khai được hook/MCP, và `$(...)` trong `api_key`/`base_url` chạy lúc nạp | Thêm vào đoạn "Repo lạ", kèm "đọc cả thư mục `.crush/` nếu repo có" |
| 2 | Allowlist và sổ ghi LLM được trình bày mạnh hơn thực tế: allowlist khoá TÊN, không khoá `base_url`; sổ ghi không ghi URL | Bước 5 của luồng hằng ngày, câu "bug của agent-cli", mô tả `run --json` đều nói rõ giới hạn đó |
| 3 | "`loop` không commit, không push" đọc như bảo đảm an toàn git, trong khi loop tự duyệt mọi lệnh và không có lớp chặn của routine | Nói rõ model trong loop vẫn chạy được `git commit`/`git push`; cần chặn thì chạy ở nơi không có quyền push hoặc dùng routine |

Mục "nên sửa" số 1 (comment cuối dòng trong ví dụ env có thể làm hỏng khởi động) đã kiểm trong mã: bộ đọc tệp env lấy đúng
phần trong nháy, nên không hỏng. Phần nên sửa đã làm: key trong `.env` của thư mục đang đứng cũng làm provider chọn được;
`send` không `--wait` thì lượt "có thể chưa chạy xong"; `esc` ở Home xoá ô tìm trước; `keys` chỉ in phím màn chat; YOLO áp
cả cho TUI khác và `send`; ví dụ env có `AGENT_CLI_CONTEXT_WINDOW`; ví dụ routine đổi sang `go vet` (verify chứng minh được
đích).

Cùng lúc, người soát mã vòng 8 bắt 3 câu README khác (đường dẫn tệp định nghĩa routine, `--commit` chỉ cho phép commit,
`.env` của thư mục đang đứng); đã sửa.

## Lượt 6 — review thay thế khi tiếp quản ngày 19/10/2026

Lượt ngày 16/10 chưa có kết luận lưu lại khi Claude hết hạn mức. Phiên Codex đã
khôi phục các sửa README cuối từ transcript và giao reviewer ngữ cảnh mới,
chỉ đọc README, các tài liệu được dẫn và CLI help trong môi trường cô lập.

**README: READY · KET-LUAN: DAT.** Cả 16 câu hỏi trả lời được; không có MUST-FIX.
Một NICE-TO-HAVE: guide nói “Không commit, không push” cho loop trong khi model
vẫn có thể gọi Git; đã đồng bộ câu đó với README. Không thêm lời hứa bảo mật.
Chi tiết: [README_REVIEW_2026-10-19](README_REVIEW_2026-10-19.md).
Các kết quả runtime mới nằm ở [39](../system-evolution/39_TAKEOVER_2026-10-19.md),
không thay cho bằng chứng ngày 16/10.

## Lượt 6b — soát độc lập phía Claude trên README bản tiếp quản (19/10)

Người soát là agent mới, loại không có Edit/Write. Nó chỉ đọc README của worktree `codex/audit-r8-takeover`, các tệp README trỏ
tới, và `--help`/`keys` của binary bản tiếp quản (sha256 `2ad147de…`) với HOME/XDG/CRUSH_* cô lập. Nó không đọc mã nguồn
và không ghi gì vào worktree. Đây là người soát khác reviewer của Codex (lượt 6 ở trên).

**README: READY.** 16/16 câu trả lời đúng chỉ từ README và các tệp nó trỏ tới, không có MUST-FIX. Mọi lệnh, cờ và phím màn
chat trong README đều có trong `--help`/`keys`. Link và anchor đều tới được. Không còn ID nội bộ hay con trỏ mã nguồn.

NICE-TO-HAVE, chưa sửa vì README là bản tiếp quản đã chốt; mỗi mục là việc riêng nếu người dùng muốn:

| # | Chỗ | Đề nghị |
|---|---|---|
| N1 | "Hai loại, cả hai khai trong env" | Tệp env khai MỘT provider chính: đặt `AGENT_CLI_PROVIDER_ID` thì `AGENT_CLI_BASE_URL` bị bỏ qua |
| N2 | đoạn `loop` | Nói `loop` sửa thẳng cây làm việc hiện tại (commit/stash việc dở trước) |
| N3 | "tool ghi tệp không ghi được ra ngoài" | Thêm "ghi qua bash (`>`, `cp`, `tee`) thì không bị chặn" |
| N4 | "đọc ngoài dự án đều hỏi quyền" | Ngoại lệ symlink trong repo trỏ ra ngoài (guide đã ghi) |
| N5 | luật allowlist | "phải là một cặp" đọc như cặp chính+nhỏ; nói rõ danh sách chỉ có `claude_code:`/`codex:` thì model chính không bị giới hạn |
| N6 | "tệp cuối thắng mọi cấu hình khác, kể cả env" | Thêm "không vượt allowlist" |
| N7 | export key cho provider thứ hai | Server đang chạy giữ env cũ: thoát mọi client của dự án trước |
| N8 | tệp key | `run`/`loop`/`routine`/YOLO tự duyệt đọc ngoài dự án, model đọc được tệp key |
| N9 | thuật ngữ | "Stop hook", "Ponytail/Superpowers", "chế độ local", "Charm Hyper" chưa giải nghĩa tại chỗ |
| N10 | theme | Không nói `options.tui.active_theme` nằm ở tệp nào |
| N11 | `agent-cli.crushrc.example` | Comment giao việc `claude_code`/`codex` và câu "Request chỉ đi tới provider…" sai so với mã: **đã sửa** bằng patch bổ sung (xem dưới) |
| N12 | guide, mục "Giới hạn đã biết" | Mở bằng mã PRB-xx và đường dẫn `reports/` mà người dùng không mở được; vài con trỏ mã nguồn |
| N13 | `run --help` | Mô tả `--json` chưa liệt kê `provider`/`model` (sửa ở mã, không ở README; người soát chưa kiểm lúc chạy) |
| N14 | `ctrl+s` | Màn chat: danh sách session; hộp thoại quyền: cho phép cả session. Gõ theo thói quen dễ cấp quyền nhầm |

Người soát chưa kiểm: clone URL, installer, output thật của `models`/`run`/`loop`/`routine`, phím trong hộp thoại quyền và
hộp thoại session (không có trong `keys`), trường `provider` của `run --json` lúc chạy.

N11 là câu sai sự thật trong tệp mẫu người dùng chép về, nên được sửa ngay. Patch:
`reports/system-evolution/evidence/followup-2026-10-19/crushrc-followup.patch` (chỉ đổi comment, áp lên bản tiếp quản). Kết quả soát patch ghi ở
[WORKFLOW_PROGRESS](WORKFLOW_PROGRESS.md).
