# P03: `agent-cli-tmux rm --force` xoá worktree khi còn session BUSY ở thư mục con

Worktree `/home/luannt/application/opensource/agent-cli-p03`, nhánh `v3-poc-p03-tmuxrm`, gốc `v2-final` (`7cba42ff`). Chưa commit, chưa push.
Patch: `P03-tmuxrm.patch` (2 tệp: `scripts/agent-cli-tmux.sh`, `scripts/test-agent-cli-tmux.sh`; 31 dòng thêm, 7 dòng xoá; không gồm 6 tệp test phơi nhiễm `??`).
`sha256sum -c P03-tmuxrm.patch.sha256` → `9571582154e622279c04553e2dc0c3749192f3f36b774770157e0ee8d2e6545b`. `git apply --check` trên `v2-final` cho rc=0.

## 1. Vấn đề

- **Triệu chứng (ca T4 của `scripts/eval/exposure.sh`):** có một session BUSY chạy ở `$wt/sub`. Khi đó `rm <tên> --force` vẫn xoá worktree (rc=0), kèm luôn `.crush/`, tức lịch sử session. Đối chứng: session BUSY ở đúng gốc `$wt` thì rm từ chối (rc=3).
- **Gốc (đã đọc lại trước khi sửa):** `scripts/agent-cli-tmux.sh:106`, `[[ $c != "$abs" ]]`. Lệnh so đường dẫn của session với gốc worktree bằng phép bằng nhau tuyệt đối. Đúng ra phải so "nằm trong worktree".
- **Cùng kiểu so sánh trong script (đã soát mọi `==`/`!=`/`index(` và mọi chỗ dùng `$abs`):**

| Chỗ | Dùng cho | Cùng lỗi? | Xử lý |
|---|---|---|---|
| dòng 106: so path session BUSY/WAITING | rm: chặn xoá khi còn session chạy | CÓ (lỗi chính) | sửa |
| `find_window`, gọi từ rm (dòng 112) | rm: đóng window và đợi pane thoát rồi mới kiểm lại (bảo vệ N1) | CÓ. Window mở bằng `agent-cli-tmux $wt/sub` không bị đóng, nên agent sống tiếp trong thư mục đã xoá, và bước "đóng → đợi → kiểm lại file gitignore" bị bỏ qua cho window đó | sửa: rm gọi `find_window "$abs" all` |
| `find_window`, gọi từ lệnh mở (dòng 170) | mỗi thư mục một window, khử trùng theo path | KHÔNG: `$wt` và `$wt/sub` phải là hai window riêng, nên so bằng nhau là đúng | giữ nguyên (không truyền `all`) |
| `ls` (dòng 63–71) | liệt kê worktree và trạng thái git | Không so với path session nào | không đổi |

## 2. Cách sửa (`scripts/agent-cli-tmux.sh`)

- Dòng 106 đổi thành `[[ $c != "$abs" && $c != "$abs"/* ]]`.
  - Phần trong ngoặc kép là chuỗi literal, nên ký tự glob trong path không bị diễn giải.
  - Có `/` ở biên, nên `/a/wt` không khớp `/a/wt2`.
- `find_window` nhận tham số thứ hai tuỳ chọn.
  - Có tham số: in MỌI window có `@agent_dir` bằng `$1` hoặc bắt đầu bằng `$1/`.
  - Không có tham số: giữ hành vi cũ (khớp đúng một, `exit`).
- Nhánh rm gom pid của tất cả window đó trước, rồi mới kill từng window. Vòng đợi pid giữ nguyên.
- Chuẩn hoá đường dẫn theo cách script đang làm. Hai vế đều đi qua `pwd -P`:
  - `abs`: `cd "$wt" && pwd -P`.
  - Path session: `cd "$p" && pwd -P`.
  - `@agent_dir`: được ghi bằng `pwd -P` lúc mở window.
- Nhờ vậy `/` cuối và symlink đã được phân giải trước khi so. Không thêm bước chuẩn hoá mới.

## 3. Test (`scripts/test-agent-cli-tmux.sh`, thêm 3 ca vào khối N5)

- **(d)** Có session BUSY ở `$wt/sub`, ở `$wt/sub/` (có `/` cuối), và ở `$work/busy-link/sub` (symlink trỏ vào worktree). Chạy `rm busy --force` ⇒ phải ra rc=3, worktree và window còn nguyên. stderr phải chứa `BUSY/WAITING`, để không đếm nhầm thành từ chối vì path không phân giải được.
- **(e)** Ca biên tên: có hai worktree `pfx` và `pfx2`. Session BUSY ở `pfx2` và `pfx2/sub`. Chạy `rm pfx --force` ⇒ phải ra rc=0, `pfx` mất, còn `pfx2` cùng window của nó thì còn. Ca này chặn hai lỗi: từ chối oan, và đóng nhầm window.
- **(f)** Mở thêm window bằng `agent-cli-tmux $wt/sub`. Chạy `rm subw` ⇒ phải ra rc=0, và cả window gốc lẫn window `sub` đều mất.

## 4. Bằng chứng (đều từ lệnh vừa chạy)

### ĐỎ, trước khi sửa

```
$ bash scripts/test-agent-cli-tmux.sh            # bản gốc + test mới
FAIL: N5 thư mục con '/tmp/tmp.lkruS5M2h2/proj-agents/busy/sub': rc=0:
rc=1
$ bash scripts/eval/exposure.sh /home/luannt/.cache/v3ev/p03-1 /home/luannt/.cache/v3ev/agent-cli T4
T4: EXPOSED — control: ... rm --force rc=3, worktree still exists=yes ... sub: busy rows=['repo-agents/t4sub/sub'] | rm --force rc=0, worktree still exists=no, rm stderr='agent-cli-tmux: --force: xoá kèm 1 mục bị gitignore ...'
```

Chu trình thứ hai: đã sửa dòng 106 nhưng chưa sửa window. Kết quả: `FAIL: window thư mục con: rc=0, window sub còn=1:`.

### XANH, sau khi sửa

```
$ bash -n scripts/agent-cli-tmux.sh && bash -n scripts/test-agent-cli-tmux.sh   → OK
$ bash scripts/test-agent-cli-tmux.sh   → PASS (rc=0), chạy lại 3 lần liên tiếp đều PASS; ca cũ giữ nguyên
$ bash scripts/eval/exposure.sh /home/luannt/.cache/v3ev/p03-2 /home/luannt/.cache/v3ev/agent-cli T4
T4: NOT_EXPOSED — control: ... rm --force rc=3, worktree still exists=yes ... sub: busy rows=['repo-agents/t4sub/sub'] | rm --force rc=3, worktree still exists=yes, rm stderr='... worktree còn session BUSY/WAITING (agent-cli ps), không xoá: .../t4sub'
```

`cmp` xác nhận launcher mà exposure.sh cài vào `p03-2/_launcher` đúng là script đã sửa. Binary dùng chung là `agent-cli version v0.2.1-0.20261019085449-7cba42ff3ca4+dirty`. Không build lại, vì bản sửa chỉ nằm ở shell.

### Kiểm răng hai chiều

Chạy test mới trên bản sao đột biến của launcher, đặt trong scratchpad, không đụng worktree. Lệnh ở mục 6:

| Bản launcher | Kết quả test | Ca bắt |
|---|---|---|
| bản sửa nguyên vẹn | PASS rc=0 | — |
| r0: hoàn toàn bộ (HEAD) | FAIL rc=1 | (d) `busy/sub: rc=0` |
| m1: chỉ hoàn dòng 106 | FAIL rc=1 | (d) |
| m2: rm gọi `find_window` không có `all` | FAIL rc=1 | (f) `window sub còn=1` |
| m3: dòng 106 so tiền tố thô `"$abs"*` (thiếu `/` biên) | FAIL rc=1 | (e) `rc=3` (từ chối oan pfx) |
| m4: awk so tiền tố thô `index($2, d)` | FAIL rc=1 | (e) `pfx2 window=0` (đóng nhầm) |

## 5. Bảng chân trị: `rm <wt> --force`, worktree sạch

| Tình huống | Trước | Sau |
|---|---|---|
| BUSY ở `$wt` (đối chứng) | từ chối rc=3 | từ chối rc=3 |
| BUSY ở `$wt/sub` | **xoá rc=0, mất `.crush/`** | từ chối rc=3 |
| BUSY ở `$wt/sub/` | **xoá rc=0** | từ chối rc=3 |
| BUSY ở symlink → `$wt/sub` | **xoá rc=0** | từ chối rc=3 |
| BUSY ở `wt2` (tên trùng tiền tố) | xoá rc=0 | xoá rc=0 (không chặn oan) |
| BUSY ở path không phân giải được / không tuyệt đối | từ chối rc=3 | từ chối rc=3 (giữ nguyên) |
| IDLE ở `$wt` | xoá rc=0 | xoá rc=0 |
| Window `@agent_dir=$wt/sub` khi rm | **còn sống, agent ở trong thư mục đã xoá** | bị đóng, đợi pane thoát rồi mới kiểm lại |
| Window của `wt2` khi rm `wt` | còn | còn |
| Mở `agent-cli-tmux $wt/sub` khi đã có window `$wt` | window riêng | window riêng (khử trùng vẫn so bằng nhau) |

## 6. Lệnh tái lập

```bash
cd /home/luannt/application/opensource/agent-cli-p03          # hoặc: git checkout v2-final && git apply <patch>
bash -n scripts/agent-cli-tmux.sh && bash scripts/test-agent-cli-tmux.sh      # PASS
bash scripts/eval/exposure.sh /home/luannt/.cache/v3ev/p03-<n mới> /home/luannt/.cache/v3ev/agent-cli T4   # NOT_EXPOSED
```

Kiểm răng: chạy test trên các bản đột biến, mỗi bản phải FAIL. Mọi bản sao nằm trong thư mục nháp, không đụng worktree. Dán khối dưới vào bash, với `W` = worktree và `S` = thư mục nháp mới:

```bash
W=/home/luannt/application/opensource/agent-cli-p03; S=$(mktemp -d); L=$W/scripts/agent-cli-tmux.sh
mk() { mkdir -p "$S/$1/scripts"; cp "$W/scripts/test-agent-cli-tmux.sh" "$S/$1/scripts/"; }
mk r0; git -C "$W" show HEAD:scripts/agent-cli-tmux.sh > "$S/r0/scripts/agent-cli-tmux.sh"
mk m1; sed 's|\[\[ \$c != "\$abs" && \$c != "\$abs"/\* \]\]|[[ $c != "$abs" ]]|' "$L" > "$S/m1/scripts/agent-cli-tmux.sh"
mk m2; sed 's|ids=\$(find_window "\$abs" all)|ids=$(find_window "$abs")|' "$L" > "$S/m2/scripts/agent-cli-tmux.sh"
mk m3; sed 's|\$c != "\$abs"/\*|$c != "$abs"*|' "$L" > "$S/m3/scripts/agent-cli-tmux.sh"
mk m4; sed 's|index(\$2, d "/") == 1|index($2, d) == 1|' "$L" > "$S/m4/scripts/agent-cli-tmux.sh"
for m in r0 m1 m2 m3 m4; do diff -q "$L" "$S/$m/scripts/agent-cli-tmux.sh" >/dev/null && echo "$m: sed không đổi gì!"
  out=$(bash "$S/$m/scripts/test-agent-cli-tmux.sh" 2>&1); echo "$m rc=$? | $(tail -1 <<< "$out")"; done
```

Cả hai test đều chạy tmux trên socket riêng, không đụng server tmux mặc định và không gọi LLM thật. T4 dùng `fakellm.py`.

## 7. Rủi ro và chỗ chưa phủ

- **Đổi hành vi:** `rm` giờ đóng cả window mở ở thư mục con của worktree. Trước đây window đó sống sót với cwd đã bị xoá. Đây là chủ ý, nhưng là thay đổi có người dùng nhìn thấy.
- **Chưa phủ (mới nghi, ngoài phạm vi):**
  - Session BUSY ở thư mục CHA của worktree (ví dụ gốc repo, hay `<repo>-agents`) vẫn không chặn rm. Bất biến được giao chỉ là "nằm trong worktree". Cách kiểm: thêm hàng BUSY có path `$top` vào `ps.json`.
  - Window tmux mà người dùng tự mở (không có `@agent_dir`) với cwd trong worktree không bị đóng. Hành vi này có từ trước, script chỉ quản window do nó tạo.
  - `awk -v d=...` diễn giải dấu `\` trong path. Path repo có `\` thì cả cách so bằng nhau cũ lẫn cách so "nằm trong" mới đều không tìm ra window. Lỗi này có từ trước, chưa sửa; tên việc đã bị giới hạn `^[a-z0-9-]`, nhưng thư mục cha thì không.
- **Không làm:** chưa thêm mục vào `CHANGELOG.md`. `docs/agent-cli-local-setup.md:413` vốn đã ghi "từ chối nếu còn … session đang chạy", nên bản sửa đưa hành vi về khớp tài liệu, không cần sửa docs. Không có shellcheck trên máy, nên chưa chạy lint.
- Người làm tự kiểm, chưa qua soát độc lập.

## Sau soát (điều phối, 19/10)

- Soát độc lập: **KET-LUAN: DAT** (phép so biên, glob trong nháy kép, `find_window … all`, test đo hiệu ứng, phạm vi
  không nở). Điều phối chạy lại: `test-agent-cli-tmux.sh` rc 0; T4 NOT_EXPOSED.
- Theo người soát, điều phối sửa chữ: CHANGELOG ("mọi session BUSY/WAITING … gốc hoặc thư mục con") và
  `docs/agent-cli-local-setup.md:434,438` (chặn ở gốc hoặc thư mục con; đóng cả window ở thư mục con). Báo cáo ở trên
  ghi "2 tệp / chưa CHANGELOG" là trạng thái trước bước này.
- Commit trên nhánh `v3-poc-p03-tmuxrm` (4 tệp; 6 tệp test phơi nhiễm không commit). Patch cuối (`git diff v2-final
  HEAD`) thay bản cũ cùng tên.
- Còn mở (ngoài phạm vi): session ở thư mục CHA của worktree không chặn `rm`; window người dùng tự mở không bị đóng;
  `awk -v` diễn giải `\` trong path (có từ trước; hỏng theo hướng không đóng window).
