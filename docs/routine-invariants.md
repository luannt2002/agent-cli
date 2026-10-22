# Bất biến của loop và routine: một câu hỏi, một câu trả lời

Mỗi câu hỏi dưới đây chỉ có **một** hàm trả lời. Nơi nào cần câu trả lời thì gọi hàm đó, không tự viết lại.

Lý do (audit 22/10): 9/9 phần việc của v0.3.0–v0.3.1 bị soát độc lập chặn ở lần đầu. Phần lớn lỗi nằm ở những chỗ hai
nơi trả lời cùng một câu hỏi theo hai quy tắc khác nhau. Ví dụ: bộ dọn coi tệp bị gitignore là việc thật, còn dấu vân tay
cây thì bỏ qua chúng, nên báo NO_CHANGE oan.

Sửa hành vi ở bảng nào thì sửa bảng trước, rồi viết test theo bảng.

## Q1. Lượt còn sống không? → `routine.RunAlive(pid, name)`

| Báo cáo | pid | Kiểm được không | Trạng thái hiện ra |
|---|---|---|---|
| RUNNING | có, tiến trình sống và cmdline có `routine` `run` `<tên>` | Linux | running |
| RUNNING | có, chắc chắn đã chết | Linux | ABANDONED |
| RUNNING | không có (bản cũ) hoặc không kiểm được (Windows) | — | UNKNOWN: không mở, không ghi đè |
| khác RUNNING | — | — | đúng `reason` của báo cáo |

- Không bao giờ thử khoá của routine khác để đoán; thử khoá làm một lượt đang khởi động thoát 75.
- Chỉ ghi ABANDONED lên đĩa khi đọc lại báo cáo thấy vẫn RUNNING với cùng pid.

## Q2. Verify đạt không?

| Lý do dừng | verify_status | Hiện ra |
|---|---|---|
| GREEN, NO_CHANGE | PASS | PASS |
| mọi lý do khác (TIMEOUT, STREAM_STALL, QUOTA, RUN_ERROR, ERROR…) | PASS | PASS (run not green, cần soát) |
| bất kỳ | FAIL | FAIL |
| bất kỳ | vắng | not run |

- `verify_exit_code` chỉ có khi verify thật sự chạy: PASS, FAIL, hoặc UNMEASURED có mã thoát thật khác 0.
- Lượt bị cắt không bao giờ ra GREEN từ chính lần verify sau lúc cắt. GREEN chỉ đến sau một lượt agent chạy trọn.

## Q3. Agent đã làm ra gì trong cây? → một bộ phân loại duy nhất

Mọi mục của `git status --porcelain -z --untracked-files=all --ignored=matching`, cộng commit ngoài base, được xếp vào
đúng một hạng:

| Hạng | Ví dụ | Là "việc"? |
|---|---|---|
| tệp theo dõi bị đổi | ` M src/a.go` | có |
| tệp chưa theo dõi | `?? notes.md` | có |
| bị ignore, ngoài `.crush/`, không phải thư mục dựng lại được | `!! .env`, `!! build/x.env` (khi `build/` được theo dõi) | có |
| bị ignore, đúng là thư mục dựng lại được | `!! web/node_modules/` | không |
| dữ liệu của dew | `!! .crush/…` | không (chuyển ra ngoài trước khi gỡ) |
| bị giấu khỏi status (assume-unchanged, hoặc skip-worktree mà tệp có trên đĩa) | `h …`, `S …` | có |
| commit ngoài base, trên HEAD hoặc nhánh của lượt | — | có |

Ghi chú cho bảng trên:
- **Dữ liệu của dew** là mọi mục chưa theo dõi hoặc bị ignore dưới `.crush/`, kể cả `??` khi `.crush/.gitignore` bị
  xoá. Tệp theo dõi dưới `.crush/` bị đổi vẫn là việc, nhưng export không bao giờ xuất `.crush/`.
- **Thư mục bị ignore trọn** hiện một lần (`out/`) trong `changed_files` và trong dòng giữ (keep) của bộ dọn.
- **skip-worktree mà tệp không có trên đĩa** không phải việc; export không xuất nó thành "đã xoá".
- **`base_sha` của báo cáo không phải SHA** (rỗng, bị sửa) thì không đếm được commit: bộ dọn giữ worktree, không bao
  giờ coi là "không có commit".
- **Giới hạn của git:** tệp theo dõi `out` bị thay bằng thư mục `out/` bị ignore thì `--ignored=matching` không liệt kê
  tệp bên trong. Bộ dọn vẫn giữ (có ` D out`), nhưng `--include-ignored` không xuất các tệp đó.

Dùng chung bởi:
- **dấu vân tay cây** (NO_CHANGE, STALLED): băm HEAD, diff của tệp theo dõi, nội dung tệp chưa theo dõi, và **chỉ tên** của
  các mục bị ignore là việc. Lý do: lệnh verify thường ghi lại artefact bị ignore (`junit.xml`, `cover.out`,
  `.phpunit.result.cache`) với nội dung đổi sau mỗi lần chạy. Băm nội dung thì STALLED không bao giờ bắt được. Agent tạo
  tệp bị ignore mới thì tên mới xuất hiện, nên vẫn thấy cây đổi. Giới hạn: artefact mang tên mới mỗi lần chạy
  (`.nyc_output/<uuid>.json`, `allure-results/`) vẫn làm STALLED không bắt được; loop chỉ chạy thêm tới hết `max_repairs`.
  Repo lồng chưa theo dõi (`?? nested/`) chỉ tính tên;
- **`changed_files`** trong báo cáo;
- **bộ dọn**: chỉ gỡ khi tập "việc" rỗng;
- **export**: patch gồm các mục "việc"; mục bị ignore chỉ khi có `--include-ignored`, `.crush/` không bao giờ.

## Q4. Được xoá gì?

Chỉ xoá worktree của một lượt khi **đủ cả**:
- báo cáo đã kết thúc;
- đường worktree và nhánh đúng của lượt đó;
- Q3 không còn "việc";
- không phải lượt mới nhất;
- không có tệp thuộc user khác hay thư mục chỉ đọc;
- `crush.db*` không bị ghi sau `finished` + 1 phút.

Trước khi xoá, chuyển `.crush/` ra cạnh `report.json`. Cache Go chỉ xoá mục của chính Go, không đụng mục dùng trong 2 giờ
qua. Không bao giờ xoá module cache Go hay Docker volume.

## Q5. Từ chối chạy (thoát 75): loại nào?

| Loại | Chạy lại sau có ích? |
|---|---|
| `lock_held`: routine này đang chạy | **Không**: chạy lại là chạy trùng |
| `no_slot`: đã đủ `DEW_ROUTINE_SLOTS` lượt | Có |
| `low_memory`: RAM dưới `DEW_ROUTINE_MIN_MEM_MB` | Có |
| `low_disk`: đĩa dưới `DEW_MIN_FREE_GB` | Có, sau khi dọn |

`routine run --json` in một object trên stdout: báo cáo cuối (như `report.json`, kể cả `ERROR`), hoặc khi bị từ chối
`{"refused": "<loại>", "routine": "<tên>", "detail": "…"}`, hoặc khi lỗi trước lúc có báo cáo (repo không còn là
repo git, lỗi IO) `{"routine": "<tên>", "error": "…"}`. Mã thoát không đổi (75 khi từ chối). Dùng sai cú pháp (thoát 2)
thì không in gì.

## Q6. Một lượt agent không chạy trọn

| Nguyên nhân | Làm gì |
|---|---|
| lỗi tạm thời (5xx, 429 không phải hạn mức, stream im, mạng) | chạy lại cùng đề bài, tối đa `stream_retries` lần liên tiếp |
| hết lần chạy lại vì stream im | dừng `STREAM_STALL`, verify một lần |
| hết hạn mức (`QUOTA`), xác thực | dừng ngay, không chạy lại |
| quá `run_timeout` | verify một lần. PASS → dừng `TIMEOUT` (cần soát). FAIL mà còn lượt sửa → **chạy tiếp một lượt sửa**, báo cho agent biết lượt trước bị cắt. FAIL mà hết lượt sửa → `TIMEOUT` |

Lượt sửa sau khi bị cắt:
- chạy trong **session mới**, như mọi lượt sửa;
- đề gồm một dòng báo lượt trước bị cắt (kèm thời lượng `run_timeout`), rồi tới đề sửa thường;
- tốn đúng một lượt sửa;
- journal ghi sự kiện `continue`.

Lượt sửa đó gặp lỗi tạm thời thì lần chạy lại chỉ mang lời báo lỗi tạm thời: lời báo `run_timeout` đã nằm ở lần gửi đầu
và session đó đã đọc. Hết `max_duration` ngay sau khi quyết chạy tiếp thì detail của `TIMEOUT` có thêm `; the last turn
was cut by run_timeout`.

Ngay sau khi cắt thì không xét STALLED; lượt sửa chạy trọn sau đó mới so với lần đo lúc cắt. Verify sau khi cắt mà
UNMEASURED → `TIMEOUT`; bị huỷ giữa lúc verify → `CANCELLED`.
