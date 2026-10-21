# Plan: dew tự quản lý rác và cache build (housekeeping)

> Người thi hành: Opus, effort high, **được quyền phản biện plan**. Plan do điều phối viết từ số đo ngày 21/10; chỗ nào sai thì nói.
> Cấm: viết số chưa chạy ra · nở phạm vi ngoài plan · commit/push · xoá ngoài các đích nêu dưới.

## Vì sao (đã đo trên luannt-pc, 21/10/2026)

- Đĩa từng lên 91% (20 GB trống). `~/.cache/go-build` chiếm **68 GB**. Theo tuổi tệp: 10,8 GB tạo trong 24 giờ qua, 37 GB trong 1–3 ngày, 22,9 GB trong 3–7 ngày, 0 GB cũ hơn 7 ngày.
- Go chỉ tự dọn **theo tuổi**: mỗi ngày một lần, bỏ mục không dùng quá 5 ngày (`cmd/go/internal/cache/cache.go`, `trimLimit`). **Không có trần dung lượng.**
- Thí nghiệm (cùng mã ở hai đường dẫn, mỗi lần một cache trống, `go test -run '^$'` cho 4 package): lần 1 tạo 787 MB, đường dẫn khác chỉ thêm 17 MB, chạy lại lần 1 thêm 0 MB. Vậy nhân bản theo đường dẫn **không** phải nguyên nhân, nên **không** làm `-trimpath`. Nguyên nhân là sửa mã liên tục × nhiều cấu hình build (race, CGO, GOOS, GOEXPERIMENT) × giữ 5 ngày.
- Go coi mục cache bị mất là miss (`errMissing`), nên trim không làm hỏng cache. Nhưng một build giữ đường dẫn tệp cache đã tra và đọc nó sau (`cmd/go/internal/work/buildid.go`, `a.built = file`), còn mtime chỉ được cập nhật khi mục cũ hơn 1 giờ (`mtimeInterval`): xoá đúng mục đó thì build ấy có thể hỏng một lần. Vì vậy trim không xoá mục dùng trong 2 giờ qua và xem lại mtime từng mục ngay trước khi xoá (sửa sau lượt soát độc lập 21/10). Một mục `-d` có thể là **thư mục** (cache executable) và phải xoá bằng RemoveAll.
- Build chạy trong Docker bằng root để lại tệp thuộc root (2.395 tệp trong `target/` của một repo, 24.230 tệp trong thùng rác), user không xoá được. Bộ dọn phải **phát hiện và nói rõ**, không im lặng thất bại.
- Rác do dew tự sinh thì nhỏ và đã có trần (log xoay vòng 10 MB/30 ngày, backup binary giữ 3 bản). Worktree routine đã có GC. Không làm thêm gì cho phần này.

## Mục tiêu

Dew phân loại và kiểm soát rác của chính việc nó chạy, theo ba hạng:

| Hạng | Gồm | Dew làm gì |
|---|---|---|
| SAFE | cache build Go vượt trần; worktree routine không còn việc (GC sẵn có) | tự động trước `routine run`; `dew clean --apply` |
| ASK | cache npm, kho pnpm, cache pip, Docker image/build cache thu hồi được | chỉ báo dung lượng + lệnh chính xác |
| NEVER | module cache Go (`go env GOMODCACHE`; build offline cần), Docker volume, tệp thuộc user khác | chỉ báo, kèm lý do; chặn agent xoá khi không người trực |

## Phạm vi code

1. `internal/housekeep` (mới):
   - Tìm GOCACHE: env `GOCACHE`, rồi `go env GOCACHE` (có hạn giờ), không có `go` hoặc `off` thì bỏ qua.
   - Chỉ coi là cache Go khi dòng đầu `README` đúng chuỗi của Go.
   - Đo dung lượng, kèm hồ sơ tuổi (<1 ngày, 1–3, 3–7, >7).
   - `TrimTo(max, dryRun)`:
     - xếp các mục `^[0-9a-f]{64}-[ad]$` trong thư mục con `^[0-9a-f]{2}$` theo mtime cũ nhất trước, xoá tới khi ≤ max;
     - mục `-d` là thư mục thì RemoveAll;
     - không theo symlink, không đụng `README`/`trim.txt`/`testexpire.txt`;
     - trả về số byte và số mục đã xoá, cùng số mục không xoá được (quyền).
   - Kho mục ASK/NEVER:
     - đo đường dẫn chuẩn của npm (`~/.npm`), pnpm (`pnpm store path`), pip (`pip cache dir`), module cache Go;
     - Docker: `docker system df` có hạn giờ, chỉ đọc, bỏ qua nếu không có docker.
2. `dew clean [--apply] [--json]` (`internal/cmd/clean.go`):
   - In bảng: hạng · mục · đường dẫn · dung lượng · xoá thì được/mất gì · lệnh.
   - `--apply` chỉ làm hạng SAFE: trim cache Go về trần, cộng `routine gc` (quy tắc an toàn sẵn có, không `--include-ignored`).
   - Báo worktree routine có tệp không thuộc user, kèm gợi ý `sudo`, hoặc chạy Docker với `--user "$(id -u):$(id -g)"`.
3. Tự động ở đầu `routine run`:
   - Tối đa mỗi giờ một lần (tệp dấu thời gian trong data dir), dưới flock.
   - Cache Go > `DEW_GOCACHE_MAX` (mặc định 20 GB, `0` = tắt) thì trim về 80% trần, in một dòng stderr.
   - Sau đó, chỗ trống của filesystem chứa data dir < `DEW_MIN_FREE_GB` (mặc định 10, `0` = tắt) thì từ chối chạy: thoát 75, lời báo nêu `dew clean`.
   - Hai biến đọc như `DEW_ROUTINE_SLOTS` (`execpolicy.FileEnv`). Giá trị sai thì thoát 2.
4. Chặn khi không người trực (`internal/shell/unattended.go`, cùng kiểu các luật sẵn có):
   - `go clean -modcache`
   - `docker volume rm|prune`
   - `docker system prune`
   - `docker compose down -v/--volumes` (cả `docker-compose`)
   - Lời báo nêu lý do: dữ liệu hoặc thứ build offline cần, không dựng lại được.
5. GC worktree (`internal/cmd/routine.go`): khi gỡ thất bại vì quyền, `gc_error` nói rõ tệp thuộc ai và cách xử lý.

Ngoài phạm vi: thực thi dọn npm/pnpm/pip/Docker; `-trimpath`; lịch chạy nền; dọn log và backup của dew.

## Chỗ chưa chắc (kèm cách kiểm)

- Lấy mtime cho khoảng 200 nghìn tệp có nhanh không? → đo trên cache thật (sau khi đã `go clean`, cache nhỏ; dựng cache giả 50 nghìn tệp trong test để đo thời gian).
- `pnpm store path` / `pip cache dir` có thể chậm hoặc gọi mạng? → đặt hạn giờ 5 giây, lỗi thì ghi "không đo được".
- Trần 20 GB có hợp lý không? → 1 ngày làm nặng sinh 10–20 GB. Ghi trong docs là chỉnh được.

## Cách chứng minh là xong

- Unit test (cache giả trong `t.TempDir()`):
  - xoá đúng mục cũ nhất tới ≤ trần;
  - không đụng tệp lạ hay symlink;
  - từ chối thư mục không có README của Go;
  - mục `-d` là thư mục bị xoá;
  - tệp không xoá được (thư mục chỉ đọc) được đếm, không làm vỡ.
- Test lệnh: `dew clean` không xoá gì; `--apply` chỉ xoá SAFE.
- Test routine: cache vượt trần thì trim, giới hạn 1 lần/giờ; đĩa thiếu thì thoát 75 (giả lập hàm đo chỗ trống).
- Test unattended: từng lệnh cấm bị chặn; `go clean -cache`, `docker ps` vẫn qua.
- Mỗi test đỏ trước khi sửa, kèm một đột biến cho mỗi điều kiện then chốt (hoàn tác bằng `cp -p`).
- Bộ test các package bị chạm xanh; `go vet` (kèm `GOOS=windows`); E2E `scripts/test-dew-routine.sh` vẫn PASS.
