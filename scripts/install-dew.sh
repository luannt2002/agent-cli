#!/usr/bin/env bash
# Build và cài CLI cá nhân; không ghi đè cấu hình thật.
set -euo pipefail
repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
install_dir="${DEW_INSTALL_DIR:-$HOME/.local}"
# Đổi sang đường dẫn tuyệt đối ngay từ đầu: wrapper ghi cứng đường dẫn binary vào chính nó,
# nên nếu để tương đối thì wrapper chỉ chạy được khi đứng đúng thư mục cài.
mkdir -p -- "$install_dir"
install_dir="$(cd -- "$install_dir" && pwd)"
config_dir="$HOME/.config/dew"
lib_dir="$install_dir/lib/dew"
bin_path="$lib_dir/crush"
wrapper_path="$install_dir/bin/dew"

# --- Chuyển từ tên cũ agent-cli (trước 20/10/2026) sang dew. Chạy lại bao nhiêu lần cũng được: chỉ làm
# phần còn thiếu. Không in nội dung tệp cấu hình (có key), chỉ in tên và số chỗ đã đổi.
if [[ -n "${AGENT_CLI_INSTALL_DIR:-}" && -z "${DEW_INSTALL_DIR:-}" ]]; then
  echo "AGENT_CLI_INSTALL_DIR đã đổi tên thành DEW_INSTALL_DIR: đặt DEW_INSTALL_DIR rồi chạy lại." >&2
  exit 1
fi
old_lib="$install_dir/lib/agent-cli"
stamp="$(date +%Y%m%d-%H%M%S)"
mig_backup=""     # $config_dir/pre-dew-<thời điểm> (700), tạo khi có thứ cần sao lưu
undo_files=()     # lệnh quay lại cho tệp (chạy trước, khi thư mục còn ở chỗ mới)
undo_dirs=()      # lệnh quay lại cho thư mục
mig_backup_dir() {
  if [[ -z "$mig_backup" ]]; then
    mig_backup="$config_dir/pre-dew-$stamp"
    [[ -d "$config_dir" ]] || install -d -m 700 "$config_dir" # có sẵn thì không đổi quyền (xem dưới)
    install -d -m 700 "$mig_backup"
  fi
}
# Đổi tên biến và đường cũ trong một tệp cấu hình, tại chỗ (ghi xuyên symlink, giữ quyền), sau khi chép
# bản sao lưu 600. old_re khớp cả tiền tố biến lẫn đường ~/.config|.local/share|.cache|.local/lib/agent-cli.
old_re='(^|[^A-Za-z0-9_])AGENT_CLI_|\.(config|cache)/agent-cli([^A-Za-z0-9_.-]|$)|\.local/(share|lib)/agent-cli([^A-Za-z0-9_.-]|$)'
rewrite_old_names() { # rewrite_old_names <tệp>
  local f="$1" n tmp
  [[ -f "$f" ]] || return 0
  n=$({ grep -oE -- "$old_re" "$f" || [[ $? == 1 ]]; } | wc -l) # grep lỗi (không phải "không khớp") thì dừng
  ((n > 0)) || return 0
  mig_backup_dir
  cp -- "$f" "$mig_backup/${f##*/}"
  chmod 600 "$mig_backup/${f##*/}"
  tmp="$mig_backup/${f##*/}.new"
  (umask 077 && sed -E -e 's/(^|[^A-Za-z0-9_])AGENT_CLI_/\1DEW_/g' \
    -e 's#\.(config|cache)/agent-cli([^A-Za-z0-9_.-]|$)#.\1/dew\2#g' \
    -e 's#\.local/(share|lib)/agent-cli([^A-Za-z0-9_.-]|$)#.local/\1/dew\2#g' -- "$f" >"$tmp")
  cat -- "$tmp" >"$f"
  rm -f -- "$tmp"
  echo "Đã đổi $n chỗ tên cũ (AGENT_CLI_ -> DEW_, đường .../agent-cli -> .../dew) trong $f; bản cũ: $mig_backup/${f##*/}"
  undo_files+=("$(printf 'cp -- %q %q' "$mig_backup/${f##*/}" "$f")")
}
move_old_dir() { # move_old_dir <cũ> <mới>: không gộp khi cả hai cùng có
  local from="$1" to="$2"
  [[ -e "$from" || -L "$from" ]] || return 0
  if [[ -L "$from" && ! -e "$from" ]]; then
    echo "Cảnh báo: $from là symlink hỏng, không chuyển; sửa hoặc gỡ nó." >&2
  elif [[ -e "$to" || -L "$to" ]]; then
    echo "Cảnh báo: có cả $from và $to: KHÔNG gộp, dew dùng $to. Xem rồi tự xoá $from khi không cần nữa." >&2
  else
    mv -- "$from" "$to" # thư mục giữ nguyên quyền; symlink thì chuyển chính symlink, đích không đổi
    echo "Đã chuyển $from -> $to"
    undo_dirs+=("$(printf 'mv -- %q %q' "$to" "$from")")
  fi
}
old_dirs=("$HOME/.config/agent-cli" "$HOME/.local/share/agent-cli" "$HOME/.cache/agent-cli" "$old_lib")
new_dirs=("$config_dir" "$HOME/.local/share/dew" "$HOME/.cache/dew" "$lib_dir")
have_old=
for d in "${old_dirs[@]}"; do [[ -e "$d" || -L "$d" ]] && have_old=1; done
if [[ -n "$have_old" && -d "$old_lib" ]]; then
  # Server/client cũ còn chạy sẽ ghi lại vào thư mục cũ sau khi chuyển: dừng trước khi đụng gì.
  old_lib_real="$(cd -P -- "$old_lib" && pwd)"
  running=$(find /proc/[0-9]*/exe -maxdepth 0 -lname "$old_lib_real/*" 2>/dev/null | cut -d/ -f3 | tr '\n' ' ') || true
  if [[ -n "$running" ]]; then
    echo "Còn tiến trình agent-cli đang chạy (pid ${running% }): thoát mọi client (TUI, run, loop), chờ server tự tắt (khoảng một phút), rồi chạy lại. Chưa đổi gì." >&2
    exit 1
  fi
fi
moved_share=
for i in "${!old_dirs[@]}"; do
  before=${#undo_dirs[@]}
  move_old_dir "${old_dirs[i]}" "${new_dirs[i]}"
  [[ $i == 1 && ${#undo_dirs[@]} -gt $before ]] && moved_share=1
done
for f in env crushrc crush.json; do rewrite_old_names "$config_dir/$f"; done
rewrite_old_names "$HOME/.local/share/dew/projects.json" # registry: đường worktree của routine
if [[ -n "$moved_share" ]] && command -v git >/dev/null; then
  # Worktree của routine nằm trong thư mục dữ liệu: báo cho repo gốc đường mới của chúng.
  for wt in "$HOME/.local/share/dew"/routines/*/wt/*; do
    [[ -f "$wt/.git" ]] && git -C "$wt" worktree repair >/dev/null 2>&1 && echo "Đã sửa đường worktree routine: $wt"
  done
fi
for name in agent-cli agent-cli-tmux; do # chỉ gỡ bản do trình cài tạo, nhận theo nội dung
  p="$install_dir/bin/$name" ours=
  [[ -e "$p" || -L "$p" ]] || continue
  if [[ -L "$p" ]]; then
    [[ "$name" == agent-cli-tmux && "$(readlink -- "$p")" == */agent-cli-tmux.sh ]] && ours=1
  elif [[ "$name" == agent-cli ]]; then
    grep -qF 'export CRUSH_GLOBAL_CONFIG="$HOME/.config/agent-cli"' -- "$p" && ours=1
  else
    grep -qF 'AGENT_CLI_TMUX_SESSION' -- "$p" && ours=1
  fi
  if [[ -z "$ours" ]]; then
    echo "Cảnh báo: $p không do trình cài tạo, giữ nguyên." >&2
    continue
  fi
  mig_backup_dir
  if [[ -L "$p" ]]; then
    undo_files+=("$(printf 'ln -s -- %q %q' "$(readlink -- "$p")" "$p")")
  else
    cp -p -- "$p" "$mig_backup/$name"
    undo_files+=("$(printf 'cp -p -- %q %q' "$mig_backup/$name" "$p")")
  fi
  rm -f -- "$p"
  echo "Đã gỡ $p (bản cài tên cũ; thay bằng ${name/agent-cli/dew})"
  undo_files+=("$(printf 'rm -f -- %q' "$install_dir/bin/${name/agent-cli/dew}")")
done
# Chỉ in TÊN biến, không in giá trị.
mapfile -t old_vars < <(compgen -e | grep '^AGENT_CLI_' || true)
if ((${#old_vars[@]} > 0)); then
  echo "Cảnh báo: shell còn export biến tên cũ (dew không đọc; biến của chính sách tin cậy làm dew dừng với lỗi nêu tên): ${old_vars[*]}. Đổi AGENT_CLI_ thành DEW_ trong ~/.bashrc (hoặc nơi export), mở shell mới." >&2
fi
mapfile -t old_units < <(grep -l -s -- agent-cli "$HOME"/.config/systemd/user/*.service "$HOME"/.config/systemd/user/*.timer || true)
if ((${#old_units[@]} > 0)); then
  echo "Cảnh báo: unit systemd còn nhắc agent-cli (installer không sửa; in lại bằng dew routine schedule <tên> --print):" >&2
  printf '  %s\n' "${old_units[@]}" >&2
fi
n_cron=$(crontab -l 2>/dev/null | grep -c -- agent-cli) || n_cron=0
if ((n_cron > 0)); then
  echo "Cảnh báo: crontab còn $n_cron dòng nhắc agent-cli (installer không sửa; crontab -e)." >&2
fi
# --- Hết phần chuyển đổi; lệnh quay lại của nó in cùng lệnh quay lại của bản cài, ở dưới.

mkdir -p "$install_dir/bin" "$lib_dir"
# Thư mục cấu hình chứa env (có key): tạo mới thì 700; đã có thì chỉ siết nếu còn mở cho nhóm/người khác.
# Nếu nó là symlink thì xét và siết thư mục ĐÍCH đã phân giải, có chủ đích: chỉ siết khi đích là của
# chính user (đích có thể là thư mục dùng chung), và nói rõ đã đổi gì. `find` không theo symlink nên
# xét symlink thì luôn thấy mode 777 và chmod sẽ đổi quyền đích một cách mù quáng.
if [[ -L "$config_dir" && ! -d "$config_dir" ]]; then
  echo "$config_dir là symlink hỏng (đích không tồn tại): tạo đích hoặc gỡ symlink rồi cài lại." >&2
  exit 1
elif [[ ! -d "$config_dir" ]]; then
  install -d -m 700 "$config_dir"
else
  real_config_dir="$(cd -P -- "$config_dir" && pwd)"
  if [[ -n "$(find "$real_config_dir" -maxdepth 0 -perm /077)" ]]; then
    if [[ ! -O "$real_config_dir" ]]; then
      echo "Cảnh báo: $real_config_dir (đích của $config_dir) không thuộc user hiện tại và còn mở cho nhóm/người khác; installer không đổi quyền. Tệp env trong đó chứa key: tự kiểm tra." >&2
    else
      chmod go-rwx "$real_config_dir"
      if [[ -L "$config_dir" ]]; then
        echo "$config_dir là symlink: đã siết quyền thư mục đích $real_config_dir về go-rwx (chứa env có key)."
      fi
    fi
  fi
fi
# Cấu hình chung: crushrc đọc biến DEW_* từ tệp env. Không ghi đè tệp có sẵn.
# crushrc thắng crush.json cùng thư mục: nếu crush.json (theo cả symlink, không sửa) đang khai
# provider thì KHÔNG tạo crushrc, kẻo thay provider đang chạy bằng placeholder. jq thiếu hoặc
# JSON hỏng thì coi như có provider (an toàn hơn là ghi đè).
legacy_providers=
if [[ -e "$config_dir/crush.json" ]]; then
  n=$(jq '(.providers // {}) | length' "$config_dir/crush.json" 2>/dev/null) || n=1
  [[ "$n" != 0 ]] && legacy_providers=1
fi
if [[ ! -e "$config_dir/crushrc" && -n "$legacy_providers" ]]; then
  [[ -e "$config_dir/crushrc.example" ]] || install -m 644 "$repo_dir/dew.crushrc.example" "$config_dir/crushrc.example"
  [[ -e "$config_dir/env.example" ]] || install -m 644 "$repo_dir/dew.env.example" "$config_dir/env.example"
  echo "Có $config_dir/crush.json đang khai provider: KHÔNG tạo crushrc (nó sẽ ghi đè provider đó). Đã để crushrc.example và env.example cạnh nó; để chuyển sang biến DEW_*, xoá 'providers' và 'models' khỏi crush.json (giữ theme), rồi đổi tên: crushrc.example -> crushrc, env.example -> env (chmod 600) và điền giá trị."
else
  if [[ ! -e "$config_dir/crushrc" ]]; then
    install -m 644 "$repo_dir/dew.crushrc.example" "$config_dir/crushrc"
    echo "Đã tạo $config_dir/crushrc."
  fi
  if [[ ! -e "$config_dir/env" ]]; then
    install -m 600 "$repo_dir/dew.env.example" "$config_dir/env"
    echo "Đã tạo $config_dir/env (mode 600); điền DEW_BASE_URL, DEW_MODEL, DEW_API_KEY trước khi gọi AI."
  fi
  if [[ -e "$config_dir/crush.json" || -L "$config_dir/crush.json" ]]; then
    echo "Có $config_dir/crush.json cũ: giữ nguyên (không khai provider). Cấu hình model nay nằm ở crushrc + env."
  fi
fi
# DEW_SKIP_BUILD=1: dùng binary có sẵn (test launcher đặt stub vào đó).
if [[ "${DEW_SKIP_BUILD:-}" == 1 ]]; then
  echo "Bỏ qua go build (DEW_SKIP_BUILD=1); dùng binary có sẵn."
else
  # Build ra tệp tạm: build lỗi thì dừng ở đây, bản đang chạy và các bản sao lưu còn nguyên.
  # Biến build như bản chính thức (.goreleaser.yml, Taskfile.yaml): CGO_ENABLED=0 và
  # GOEXPERIMENT=greenteagc. Đã kiểm với go1.27.0: tên experiment này được nhận (tên lạ bị báo
  # `unknown GOEXPERIMENT`). Phiên bản hiển thị lấy từ VCS stamping của go build, không cần -ldflags.
  (cd "$repo_dir" && CGO_ENABLED=0 GOEXPERIMENT=greenteagc GOMAXPROCS=2 go build -p 2 -o "$bin_path.new" .)
  built=1
fi
# Wrapper: mọi tiến trình dew trong cùng thư mục gốc dự án dùng chung một server.
wrapper_tpl=$(cat <<'EOF'
#!/usr/bin/env bash
export CRUSH_GLOBAL_CONFIG="$HOME/.config/dew"
export CRUSH_GLOBAL_DATA="$HOME/.local/share/dew"
export CRUSH_CACHE_DIR="$HOME/.cache/dew"
export CRUSH_DISABLE_METRICS=1
export CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1
export CRUSH_DISABLE_UPDATE_CHECK=1
export CRUSH_CLIENT_SERVER="${CRUSH_CLIENT_SERVER:-1}"
# Home mở workspace qua chính launcher này để mỗi workspace có --host riêng của nó.
export DEW_LAUNCHER="$0"
# Giá trị hiểu đúng như strconv.ParseBool của Go (useClientServer trong internal/cmd/root.go): bật là
# 1 t T TRUE true True, tắt là 0 f F FALSE false False. Giá trị khác Go coi là tắt, nên launcher cũng
# coi là tắt (cảnh báo) thay vì thêm --host cho một tiến trình sẽ chạy in-process.
case "$CRUSH_CLIENT_SERVER" in 0 | f | F | FALSE | false | False) ;;
1 | t | T | TRUE | true | True)
  # shortcut: chỉ nhận -c/--cwd đứng riêng (không gộp cờ kiểu -dc), nâng cấp khi cần.
  have_host= cwd=$PWD sub= skip= bare=1
  args=("$@")
  for ((i = 0; i < ${#args[@]}; i++)); do
    if [[ -n "$skip" ]]; then skip=; continue; fi
    # Chỉ -d/--debug giữ được Home; cờ khác (-c, -s, -C, -y, -D, -H...) là vào thẳng workspace.
    [[ "${args[i]}" == -d || "${args[i]}" == --debug ]] || bare=
    case "${args[i]}" in
      --) break ;;
      --host | -H) have_host=1 skip=1 ;;
      --host=* | -H?*) have_host=1 ;;
      -c | --cwd) cwd=${args[i + 1]-} skip=1 ;;
      --cwd=*) cwd=${args[i]#--cwd=} ;;
      -c?*) cwd=${args[i]#-c} ;;
      -D | --data-dir | --channels | -s | --session) skip=1 ;;
      -*) ;;
      *) sub=${sub:-${args[i]}} ;;
    esac
  done
  # `ps` tự quét mọi server khi không có --host; thêm --host sẽ chỉ còn dự án hiện tại.
  # dew trần trên terminal là Home (khớp shouldRunHome trong internal/cmd/home.go):
  # chưa chọn workspace nên không chọn server; Home tự gọi lại launcher với -c <dir>.
  home=
  [[ -n "$bare" && -t 0 && -t 1 ]] && home=1
  if [[ -z "$have_host" && -z "$home" && "$sub" != ps ]] && root=$(cd -- "$cwd" 2>/dev/null && { git rev-parse --show-toplevel 2>/dev/null || pwd -P; }); then
    # Không dùng /tmp: ai cũng ghi được và tên đoán được -> user khác chiếm socket trước.
    rt=${DEW_RUNTIME_DIR:-/run/user/$UID}
    [[ -d "$rt" ]] || rt=${XDG_RUNTIME_DIR:-}
    if [[ ! -d "$rt" ]]; then
      rt=$HOME/.cache/dew
      mkdir -p "$rt" && chmod 700 "$rt"
    fi
    h=$(printf %s "$root" | { sha1sum || shasum -a 1; } 2>/dev/null)
    sock="$rt/dew-${h:0:8}.sock"
    # Không chọn được server riêng thì DỪNG, đừng chạy tiếp không --host: client sẽ rơi về server
    # mặc định dùng chung và mất cách ly theo dự án (session, trạng thái bận, log lẫn giữa các dự án).
    if [[ ${#h} -lt 8 ]]; then
      echo "dew: thiếu sha1sum/shasum nên không chọn được server riêng cho dự án; cài một trong hai, hoặc chạy CRUSH_CLIENT_SERVER=0 dew (in-process)." >&2
      exit 1
    elif [[ ${#sock} -gt 107 ]]; then # giới hạn sun_path 108 byte gồm NUL
      echo "dew: đường dẫn socket quá dài (${#sock} byte, tối đa 107): $sock" >&2
      echo "dew: đặt DEW_RUNTIME_DIR trỏ tới thư mục riêng ngắn hơn (mode 700), hoặc chạy CRUSH_CLIENT_SERVER=0 dew (in-process)." >&2
      exit 1
    fi
    set -- --host "unix://$sock" "$@"
  fi ;;
*) echo "dew: CRUSH_CLIENT_SERVER=\"$CRUSH_CLIENT_SERVER\" không hợp lệ (dùng 1/0 hoặc true/false); chạy in-process như CRUSH_CLIENT_SERVER=0." >&2 ;;
esac
exec "__CRUSH_BIN__" "$@"
EOF
)
# Sao lưu bản đang cài ngay trước khi thay (luật dự án: mỗi lần cài có bản sao
# lưu và lệnh quay lại); giữ 3 bản mới nhất.
backup_dir=
if [[ -e "$bin_path" || -e "$wrapper_path" ]]; then
  backup_dir="$lib_dir/backup-$(date +%Y%m%d-%H%M%S)"
  mkdir -p "$backup_dir"
  [[ -e "$bin_path" ]] && cp -p -- "$bin_path" "$backup_dir/crush"
  [[ -e "$wrapper_path" ]] && cp -p -- "$wrapper_path" "$backup_dir/dew"
  mapfile -t old_backups < <(ls -1dt -- "$lib_dir"/backup-* 2>/dev/null | tail -n +4)
  for d in "${old_backups[@]}"; do rm -rf -- "$d"; done
fi
# mv thay được cả binary đang chạy (cp thẳng lên nó báo "Text file busy").
[[ -n "${built:-}" ]] && mv -f -- "$bin_path.new" "$bin_path"
# Ghi tệp tạm rồi mv: nguyên tử, và thay chính symlink chứ không ghi xuyên qua nó.
printf '%s\n' "${wrapper_tpl//__CRUSH_BIN__/"$bin_path"}" > "$wrapper_path.tmp"
chmod 755 "$wrapper_path.tmp"
mv -f -- "$wrapper_path.tmp" "$wrapper_path"
# dew-tmux: chép (không symlink) để bản cài không phụ thuộc thư mục checkout (đổi nhánh, di
# chuyển hay xoá repo không làm hỏng nó). Ghi tệp tạm rồi mv: thay được cả symlink của bản cài cũ
# mà không ghi xuyên qua nó vào tệp trong repo.
install -m 755 "$repo_dir/scripts/dew-tmux.sh" "$install_dir/bin/dew-tmux.tmp"
mv -f -- "$install_dir/bin/dew-tmux.tmp" "$install_dir/bin/dew-tmux"
# In lệnh quay lại TRƯỚC khi chạy thử: bản mới hỏng ngay ở --version mới là lúc cần nó.
if [[ -n "$backup_dir" ]]; then
  echo "Bản cũ đã sao lưu ở $backup_dir. Quay lại bằng:"
  [[ -e "$backup_dir/crush" ]] && printf '  cp -p %q %q && mv -f %q %q\n' "$backup_dir/crush" "$bin_path.new" "$bin_path.new" "$bin_path"
  [[ -e "$backup_dir/dew" ]] && printf '  cp -p %q %q && mv -f %q %q\n' "$backup_dir/dew" "$wrapper_path.tmp" "$wrapper_path.tmp" "$wrapper_path"
fi
if ((${#undo_files[@]} + ${#undo_dirs[@]} > 0)); then
  echo "Lần cài này đã chuyển từ agent-cli. Quay về agent-cli: chạy các lệnh quay lại ở trên (binary cũ), rồi theo thứ tự:"
  printf '  %s\n' "${undo_files[@]}"
  for ((i = ${#undo_dirs[@]} - 1; i >= 0; i--)); do printf '  %s\n' "${undo_dirs[i]}"; done
  if [[ -n "$moved_share" ]]; then
    echo '  for wt in ~/.local/share/agent-cli/routines/*/wt/*; do [ -f "$wt/.git" ] && git -C "$wt" worktree repair; done'
  fi
fi
"$wrapper_path" --version
echo "Thêm $install_dir/bin vào PATH nếu terminal chưa nhận dew."
# Mẫu trong repo có thể đã thêm biến hoặc chú thích mới so với tệp bạn đang dùng. Không bao giờ
# ghi đè tệp của user: chỉ gợi ý lệnh để tự xem khác biệt rồi gộp tay.
diff_hints=()
for pair in "crushrc:dew.crushrc.example" "env:dew.env.example"; do
  live="$config_dir/${pair%%:*}" example="$repo_dir/${pair#*:}"
  if [[ -f "$live" ]] && ! cmp -s -- "$live" "$example"; then
    diff_hints+=("diff -u $(printf %q "$live") $(printf %q "$example")")
  fi
done
if ((${#diff_hints[@]} > 0)); then
  echo "Cấu hình đang dùng khác mẫu trong repo (installer không ghi đè tệp của bạn). Xem khác biệt:"
  printf '  %s\n' "${diff_hints[@]}"
  echo "(env chứa giá trị của bạn nên luôn khác mẫu; diff sẽ in cả key, đừng chạy khi chia sẻ màn hình.)"
fi
