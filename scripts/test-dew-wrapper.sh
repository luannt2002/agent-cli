#!/usr/bin/env bash
# Kiểm launcher dew: một server chung cho mỗi thư mục gốc dự án.
# Cài vào thư mục tạm với binary giả; không đụng bản cài thật hay repo này.
set -euo pipefail
here="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
t="$(mktemp -d)"
trap 'rm -rf "$t"' EXIT

# Repo giả để install script không tạo .crush.json trong repo thật.
mkdir -p "$t/fake/scripts" "$t/inst/lib/dew" "$t/home"
cp "$here/scripts/install-dew.sh" "$here/scripts/dew-tmux.sh" "$t/fake/scripts/"
cp "$here"/dew.crushrc.example "$here"/dew.env.example "$t/fake/"
cat > "$t/inst/lib/dew/crush" <<'STUB'
#!/usr/bin/env bash
echo "CS=${CRUSH_CLIENT_SERVER-unset}"
echo "L=${DEW_LAUNCHER-unset}"
printf 'ARG=%s\n' "$@"
STUB
chmod 755 "$t/inst/lib/dew/crush"
install1() { HOME="$t/home" DEW_INSTALL_DIR="$t/inst" DEW_SKIP_BUILD=1 bash "$t/fake/scripts/install-dew.sh" 2>&1; }
msg_first="$(install1)"
wrapper="$t/inst/bin/dew"

for r in repoA repoB; do
  mkdir -p "$t/$r/sub1" "$t/$r/sub2"
  git -C "$t/$r" init -q
done
mkdir -p "$t/plain/x"
ln -s "$t/plain" "$t/plainlink"

fail=0
check() { # check <tên> <điều kiện: 0 = đạt>
  if [[ "$2" == 0 ]]; then echo "ok   $1"; else echo "FAIL $1"; fail=1; fi
}
# run <thư mục> <env...> -- <args...>: chạy wrapper, in output của stub.
run() {
  local dir="$1"; shift
  local envs=()
  while [[ "$1" != -- ]]; do envs+=("$1"); shift; done
  shift
  (cd "$dir" && env HOME="$t/home" "${envs[@]}" "$wrapper" "$@")
}
same() { [[ -n "$1" && "$1" == "$2" ]]; } # rỗng == rỗng không được tính là đạt
host_of() { sed -n 's/^ARG=\(unix:.*\)$/\1/p' <<<"$1" | head -1; }
count_host() { grep -cE '^ARG=(--host|-H)' <<<"$1" || true; }
sha8() { printf %s "$1" | sha1sum | cut -c1-8; }

root_a="$(git -C "$t/repoA" rev-parse --show-toplevel)"
# Thư mục socket mong đợi, tính như launcher: /run/user/UID -> XDG_RUNTIME_DIR -> $HOME/.cache/dew.
rt="/run/user/$(id -u)"
[[ -d "$rt" ]] || rt="${XDG_RUNTIME_DIR:-}"
[[ -d "$rt" ]] || rt="$t/home/.cache/dew"
want="unix://$rt/dew-$(sha8 "$root_a").sock"

out_a1="$(run "$t/repoA/sub1" -- run hello)"
out_a2="$(run "$t/repoA/sub2" -- run hello)"
out_b="$(run "$t/repoB/sub1" -- run hello)"
h_a1="$(host_of "$out_a1")"; h_a2="$(host_of "$out_a2")"; h_b="$(host_of "$out_b")"

check "host đúng dạng R9 (sha1 gốc git)" "$([[ "$h_a1" == "$want" ]]; echo $?)"
check "hai thư mục con cùng repo -> cùng host" "$([[ -n "$h_a1" && "$h_a1" == "$h_a2" ]]; echo $?)"
check "hai repo khác nhau -> host khác nhau" "$([[ -n "$h_b" && "$h_a1" != "$h_b" ]]; echo $?)"
check "CRUSH_CLIENT_SERVER mặc định 1" "$(grep -qx 'CS=1' <<<"$out_a1"; echo $?)"
check "đối số gốc giữ nguyên thứ tự sau --host" "$([[ "$(grep '^ARG=' <<<"$out_a1" | tail -2 | tr '\n' ' ')" == "ARG=run ARG=hello " ]]; echo $?)"

u="unix:///tmp/mine.sock"
out="$(run "$t/repoA" -- --host "$u" run x)"
check "--host của user giữ nguyên, không thêm" "$([[ "$(count_host "$out")" == 1 && "$(host_of "$out")" == "$u" ]]; echo $?)"
out="$(run "$t/repoA" -- -H "$u" run x)"
check "-H của user giữ nguyên, không thêm" "$([[ "$(count_host "$out")" == 1 && "$(host_of "$out")" == "$u" ]]; echo $?)"
out="$(run "$t/repoA" -- --host="$u" run x)"
check "--host=... của user giữ nguyên, không thêm" "$([[ "$(grep -c "^ARG=--host=" <<<"$out")" == 1 && "$(count_host "$out")" == 1 ]]; echo $?)"

out="$(run "$t" -- -c "$t/repoA/sub1" run x)"
check "-c <dir> được hiểu" "$(same "$(host_of "$out")" "$h_a1"; echo $?)"
out="$(run "$t" -- --cwd="$t/repoB/sub2" run x)"
check "--cwd=<dir> được hiểu" "$(same "$(host_of "$out")" "$h_b"; echo $?)"
out="$(run "$t" -- --cwd "$t/repoA" run x)"
check "--cwd <dir> được hiểu" "$(same "$(host_of "$out")" "$h_a1"; echo $?)"
out="$(cd "$t/repoB" && env HOME="$t/home" "$wrapper" -c sub1 run x)"
check "-c đường dẫn tương đối" "$(same "$(host_of "$out")" "$h_b"; echo $?)"

h_p="$(host_of "$(run "$t/plain/x" -- run x)")"
h_pl="$(host_of "$(run "$t/plainlink/x" -- run x)")"
check "ngoài git: realpath, symlink cùng host với đích" "$([[ -n "$h_p" && "$h_p" == "$h_pl" && "$h_p" == "unix://$rt/dew-$(sha8 "$t/plain/x").sock" ]]; echo $?)"

out="$(run "$t/repoA" CRUSH_CLIENT_SERVER=0 -- run x)"
check "CRUSH_CLIENT_SERVER=0: giữ 0, không thêm --host" "$(grep -qx 'CS=0' <<<"$out" && [[ "$(count_host "$out")" == 0 ]]; echo $?)"
out="$(run "$t/repoA" -- --version)"
check "--version vẫn chạy (chỉ thêm --host phía trước)" "$(grep -qx 'ARG=--version' <<<"$out"; echo $?)"

# `ps` tự quét mọi server khi không có --host: launcher không được chèn --host cho nó.
out="$(run "$t/repoA" -- ps)"
check "ps -> không thêm --host" "$([[ "$(count_host "$out")" == 0 ]] && grep -qx 'ARG=ps' <<<"$out"; echo $?)"
out="$(run "$t/repoA" -- ps --json)"
check "ps --json -> không thêm --host" "$([[ "$(count_host "$out")" == 0 ]] && grep -qx 'ARG=--json' <<<"$out"; echo $?)"
out="$(run "$t" -- -c "$t/repoA" ps --json)"
check "-c dir ps -> không thêm --host" "$([[ "$(count_host "$out")" == 0 ]]; echo $?)"
out="$(run "$t" -- --cwd "$t/repoA" -D "$t/dd" ps)"
check "cờ có giá trị trước ps (--cwd dir -D dir) -> không thêm --host" "$([[ "$(count_host "$out")" == 0 ]]; echo $?)"
out="$(run "$t/repoA" -- ps -H "$u")"
check "ps -H x giữ host của user" "$([[ "$(count_host "$out")" == 1 && "$(host_of "$out")" == "$u" ]]; echo $?)"
out="$(run "$t/repoA" -- run ps)"
check "run ps (ps là prompt) vẫn thêm --host" "$(same "$(host_of "$out")" "$h_a1"; echo $?)"
out="$(run "$t/repoA" -- -c "$t/repoA" run x)"
check "run vẫn thêm --host" "$(same "$(host_of "$out")" "$h_a1"; echo $?)"
out="$(run "$t/repoA" --)"
check "không đối số, không phải terminal (TUI thẳng) vẫn thêm --host" "$(same "$(host_of "$out")" "$h_a1"; echo $?)"
check "launcher tự khai đường dẫn của nó (DEW_LAUNCHER) cho Home" "$(grep -qx "L=$wrapper" <<<"$out"; echo $?)"

# Trên terminal, dew trần (hoặc chỉ -d) là Home: chưa có workspace nên không chọn server.
# `script` cấp pty để stdin/stdout là terminal như khi gõ tay.
run_tty() { # run_tty <thư mục> <args...>
  local dir="$1"; shift
  (cd "$dir" && HOME="$t/home" script -qec "$(printf '%q ' "$wrapper" "$@")" /dev/null </dev/null) | tr -d '\r'
}
if command -v script >/dev/null; then
  out="$(run_tty "$t/repoA")"
  check "terminal, không đối số (Home) -> không thêm --host" "$(grep -qx 'CS=1' <<<"$out" && [[ "$(count_host "$out")" == 0 ]]; echo $?)"
  out="$(run_tty "$t/repoA" -d)"
  check "terminal, chỉ -d (Home) -> không thêm --host" "$(grep -qx 'ARG=-d' <<<"$out" && [[ "$(count_host "$out")" == 0 ]]; echo $?)"
  out="$(run_tty "$t" -c "$t/repoB/sub1")"
  check "terminal, -c dir -> --host của dir" "$(same "$(host_of "$out")" "$h_b"; echo $?)"
  for a in "-s abc" "--session abc" "-C" "--continue" "-y" "--yolo"; do
    # shellcheck disable=SC2086 # tách cờ và giá trị có chủ đích
    out="$(run_tty "$t/repoA" $a)"
    check "terminal, $a -> vẫn thêm --host của thư mục hiện tại" "$(same "$(host_of "$out")" "$h_a1"; echo $?)"
  done
  out="$(run_tty "$t/repoA" run x)"
  check "terminal, run -> vẫn thêm --host" "$(same "$(host_of "$out")" "$h_a1"; echo $?)"
else
  echo "skip ca terminal: thiếu lệnh script"
fi

# Thứ tự thư mục socket: không bao giờ /tmp, ~/.cache/dew mode 0700.
out="$(run "$t/repoA" DEW_RUNTIME_DIR=/nonexistent XDG_RUNTIME_DIR= -- run x)"
check "không có /run/user, không XDG -> \$HOME/.cache/dew" "$([[ "$(host_of "$out")" == "unix://$t/home/.cache/dew/dew-$(sha8 "$root_a").sock" ]]; echo $?)"
check "thư mục dự phòng mode 700" "$([[ "$(stat -c %a "$t/home/.cache/dew")" == 700 ]]; echo $?)"
mkdir -p "$t/xdg"
out="$(run "$t/repoA" DEW_RUNTIME_DIR=/nonexistent XDG_RUNTIME_DIR="$t/xdg" -- run x)"
check "không có /run/user -> XDG_RUNTIME_DIR" "$([[ "$(host_of "$out")" == "unix://$t/xdg/dew-$(sha8 "$root_a").sock" ]]; echo $?)"
long="$t/$(printf 'd%.0s' {1..100})"; mkdir -p "$long"
# Socket quá dài: KHÔNG chạy tiếp không --host (client sẽ rơi về server mặc định dùng chung, mất cách ly theo dự án).
rc=0; out="$(run "$t/repoA" DEW_RUNTIME_DIR="$long" -- run x 2>"$t/err")" || rc=$?
check "socket quá 107 byte -> dừng mã 1, không chạy binary" "$([[ "$rc" == 1 && -z "$out" ]]; echo $?)"
check "socket quá dài -> lỗi nêu độ dài và DEW_RUNTIME_DIR" "$(grep -q 'quá dài' "$t/err" && grep -q DEW_RUNTIME_DIR "$t/err" && grep -q 'CRUSH_CLIENT_SERVER=0' "$t/err"; echo $?)"
rc=0; out="$(run "$t/repoA" DEW_RUNTIME_DIR="$long" CRUSH_CLIENT_SERVER=0 -- run x 2>/dev/null)" || rc=$?
check "socket quá dài nhưng CRUSH_CLIENT_SERVER=0 -> vẫn chạy (không cần socket)" "$([[ "$rc" == 0 ]] && grep -qx 'CS=0' <<<"$out"; echo $?)"
rc=0; out="$(run "$t/repoA" DEW_RUNTIME_DIR="$long" -- --host "$u" run x 2>/dev/null)" || rc=$?
check "socket quá dài nhưng user tự truyền --host -> vẫn chạy, giữ host của user" "$([[ "$rc" == 0 && "$(host_of "$out")" == "$u" ]]; echo $?)"
rc=0; out="$(run "$t/repoA" DEW_RUNTIME_DIR="$long" -- ps 2>/dev/null)" || rc=$?
check "socket quá dài nhưng lệnh ps (không cần host) -> vẫn chạy" "$([[ "$rc" == 0 ]] && grep -qx 'ARG=ps' <<<"$out"; echo $?)"

# Repo nằm ở đường dẫn rất dài: socket vẫn ngắn vì tên chỉ dùng 8 ký tự sha1.
deep="$t/$(printf 'e%.0s' {1..60})/$(printf 'f%.0s' {1..60})/$(printf 'g%.0s' {1..60})"
mkdir -p "$deep" && git -C "$deep" init -q
out="$(run "$deep" -- run x)"
h_deep="$(host_of "$out")"
check "repo đường dẫn >150 ký tự" "$([[ ${#deep} -gt 150 ]]; echo $?)"
check "repo đường dẫn dài -> socket vẫn có và <=107 byte" "$([[ -n "$h_deep" && $((${#h_deep} - 7)) -le 107 ]]; echo $?)"

# Thiếu sha1sum/shasum -> dừng với lỗi rõ (cùng lý do với socket quá dài), trừ khi tắt client/server.
mkdir -p "$t/nohash"
for c in bash env git mkdir chmod; do ln -s "$(command -v $c)" "$t/nohash/$c"; done
rc=0; out="$(run "$t/repoA" PATH="$t/nohash" -- run x 2>"$t/err")" || rc=$?
check "thiếu công cụ hash -> dừng mã 1, không chạy binary, lỗi nêu sha1sum" "$([[ "$rc" == 1 && -z "$out" ]] && grep -q sha1sum "$t/err" && grep -q 'CRUSH_CLIENT_SERVER=0' "$t/err"; echo $?)"
rc=0; out="$(run "$t/repoA" PATH="$t/nohash" CRUSH_CLIENT_SERVER=0 -- run x 2>/dev/null)" || rc=$?
check "thiếu công cụ hash nhưng CRUSH_CLIENT_SERVER=0 -> vẫn chạy" "$([[ "$rc" == 0 ]] && grep -qx 'CS=0' <<<"$out"; echo $?)"

# CRUSH_CLIENT_SERVER hiểu đúng như strconv.ParseBool của Go: bật (1 t T TRUE true True) thì thêm --host,
# tắt (0 f F FALSE false False) thì không, giá trị lạ Go coi là tắt nên launcher cũng không thêm --host và cảnh báo.
for v in 1 t T TRUE true True; do
  out="$(run "$t/repoA" CRUSH_CLIENT_SERVER=$v -- run x 2>/dev/null)"
  check "CRUSH_CLIENT_SERVER=$v (bật theo ParseBool) -> thêm --host, giữ nguyên giá trị" "$(same "$(host_of "$out")" "$h_a1" && grep -qx "CS=$v" <<<"$out"; echo $?)"
done
for v in 0 f F FALSE false False; do
  out="$(run "$t/repoA" CRUSH_CLIENT_SERVER=$v -- run x 2>"$t/err")"
  check "CRUSH_CLIENT_SERVER=$v (tắt theo ParseBool) -> không --host, không cảnh báo" "$([[ "$(count_host "$out")" == 0 ]] && grep -qx "CS=$v" <<<"$out" && [[ ! -s "$t/err" ]]; echo $?)"
done
for v in yes no on off 2 tRuE y; do
  out="$(run "$t/repoA" CRUSH_CLIENT_SERVER=$v -- run x 2>"$t/err")"
  check "CRUSH_CLIENT_SERVER=$v (Go không hiểu, coi là tắt) -> không --host, cảnh báo stderr" "$([[ "$(count_host "$out")" == 0 ]] && grep -q 'không hợp lệ' "$t/err" && grep -qx 'ARG=run' <<<"$out"; echo $?)"
done

# DEW_SKIP_BUILD chỉ bỏ build khi đúng bằng 1. go giả ghi dấu mỗi lần được gọi.
mkdir -p "$t/gobin" "$t/inst2/lib/dew"
# go giả giữ đúng hợp đồng của go build: sinh tệp -o (installer build ra tệp tạm rồi mv).
printf '#!/bin/sh\ntouch "%s/go-called"\necho "CGO_ENABLED=$CGO_ENABLED GOEXPERIMENT=$GOEXPERIMENT" > "%s/go-env"\nwhile [ $# -gt 0 ]; do [ "$1" = -o ] && cp "%s" "$2"; shift; done\n' "$t" "$t" "$t/inst/lib/dew/crush" > "$t/gobin/go"; chmod 755 "$t/gobin/go"
cp "$t/inst/lib/dew/crush" "$t/inst2/lib/dew/crush"
inst2() { HOME="$t/home" PATH="$t/gobin:$PATH" DEW_INSTALL_DIR="$t/inst2" DEW_SKIP_BUILD="$1" bash "$t/fake/scripts/install-dew.sh" 2>&1; }
rm -f "$t/go-called" "$t/go-env"; msg="$(CGO_ENABLED=1 GOEXPERIMENT=nogreenteagc inst2 0)"
check "DEW_SKIP_BUILD=0 vẫn build" "$([[ -e "$t/go-called" ]] && ! grep -q 'Bỏ qua go build' <<<"$msg"; echo $?)"
check "build như bản chính thức: CGO_ENABLED=0 GOEXPERIMENT=greenteagc (đè biến của user)" "$([[ "$(cat "$t/go-env")" == "CGO_ENABLED=0 GOEXPERIMENT=greenteagc" ]]; echo $?)"
rm -f "$t/go-called"; msg="$(inst2 1)"
check "DEW_SKIP_BUILD=1 bỏ build và báo" "$([[ ! -e "$t/go-called" ]] && grep -q 'Bỏ qua go build' <<<"$msg"; echo $?)"

# Cài đặt cấu hình chung: crushrc + env mẫu, không .crush.json, không ghi đè lần hai.
cfg="$t/home/.config/dew"
check "cài lần đầu tạo crushrc từ mẫu" "$(cmp -s "$here/dew.crushrc.example" "$cfg/crushrc"; echo $?)"
check "cài lần đầu tạo env từ mẫu, mode 600" "$(cmp -s "$here/dew.env.example" "$cfg/env" && [[ "$(stat -c %a "$cfg/env")" == 600 ]]; echo $?)"
check "cài đặt không tạo .crush.json" "$([[ ! -e "$t/fake/.crush.json" && ! -e "$cfg/crush.json" ]]; echo $?)"
check "không có crush.json cũ -> không in hướng dẫn chuyển đổi" "$(! grep -q 'crush.json cũ' <<<"$msg_first"; echo $?)"
echo '# của user' >> "$cfg/crushrc"; echo 'DEW_MODEL=mine' >> "$cfg/env"
msg_second="$(install1)"
check "cài lần hai không ghi đè crushrc" "$(tail -1 "$cfg/crushrc" | grep -qx '# của user'; echo $?)"
check "cài lần hai không ghi đè env" "$(tail -1 "$cfg/env" | grep -qx 'DEW_MODEL=mine'; echo $?)"
check "cài lần đầu (tệp = mẫu) -> không in gợi ý diff" "$(! grep -q 'diff -u' <<<"$msg_first"; echo $?)"
# repo giả là $t/fake nên mẫu trong gợi ý nằm ở đó.
check "cài lần hai, crushrc khác mẫu -> gợi ý diff -u crushrc mẫu" "$(grep -F 'diff -u' <<<"$msg_second" | grep -F "$cfg/crushrc" | grep -qF "$t/fake/dew.crushrc.example"; echo $?)"
check "cài lần hai, env khác mẫu -> gợi ý diff -u env mẫu" "$(grep -F 'diff -u' <<<"$msg_second" | grep -F "$cfg/env" | grep -qF "$t/fake/dew.env.example"; echo $?)"
check "gợi ý diff chỉ là gợi ý: hai tệp của user vẫn nguyên" "$(tail -1 "$cfg/crushrc" | grep -qx '# của user' && tail -1 "$cfg/env" | grep -qx 'DEW_MODEL=mine'; echo $?)"
echo '{}' > "$cfg/crush.json"
msg_old="$(install1)"
check "có crush.json cũ -> giữ nguyên và in hướng dẫn" "$(grep -q 'crush.json cũ' <<<"$msg_old" && [[ "$(cat "$cfg/crush.json")" == '{}' ]]; echo $?)"
rm -f "$cfg/crush.json"

# Thư mục cấu hình chứa env: tạo mới -> 700; quyền hẹp hơn do user đặt giữ nguyên; quá rộng -> siết.
check "installer tạo thư mục cấu hình mode 700" "$([[ "$(stat -c %a "$cfg")" == 700 ]]; echo $?)"
chmod 500 "$cfg"; install1 >/dev/null; mode_narrow="$(stat -c %a "$cfg")"; chmod 700 "$cfg"
check "thư mục cấu hình hẹp hơn 700 (500) được giữ nguyên" "$([[ "$mode_narrow" == 500 ]]; echo $?)"
chmod 755 "$cfg"; install1 >/dev/null
check "thư mục cấu hình mở rộng (755) được siết về 700" "$([[ "$(stat -c %a "$cfg")" == 700 ]]; echo $?)"

# crush.json đang khai provider thì không tạo crushrc/env (crushrc sẽ ghi đè nó); chỉ để bản .example.
install_in() { HOME="$1" DEW_INSTALL_DIR="$t/inst" DEW_SKIP_BUILD=1 bash "$t/fake/scripts/install-dew.sh" 2>&1; }
mkdir -p "$t/hp/.config/dew"; cj='{"providers":{"mine":{"type":"openai-compat"}},"models":{"large":{"provider":"mine","model":"x"}}}'
printf '%s' "$cj" > "$t/hp/.config/dew/crush.json"
msg_p="$(install_in "$t/hp")"; c="$t/hp/.config/dew"
check "crush.json có provider -> không tạo crushrc/env" "$([[ ! -e "$c/crushrc" && ! -e "$c/env" ]]; echo $?)"
check "crush.json có provider -> có crushrc.example, env.example" "$([[ -f "$c/crushrc.example" && -f "$c/env.example" ]]; echo $?)"
check "crush.json có provider -> in hướng dẫn, crush.json không đổi" "$(grep -q 'KHÔNG tạo crushrc' <<<"$msg_p" && [[ "$(cat "$c/crush.json")" == "$cj" ]]; echo $?)"
echo '# sửa' >> "$c/crushrc.example"; msg_re="$(install_in "$t/hp")"
check "chạy lại không ghi đè crushrc.example" "$(tail -1 "$c/crushrc.example" | grep -qx '# sửa'; echo $?)"
check "cài lại -> sao lưu bản cũ và in lệnh quay lại" "$(ls -d "$t/inst/lib/dew"/backup-*/crush >/dev/null 2>&1 && grep -q 'Quay lại' <<<"$msg_re"; echo $?)"
mkdir -p "$t/hs/.config/dew" "$t/store"; printf '%s' "$cj" > "$t/store/real.json"
ln -s "$t/store/real.json" "$t/hs/.config/dew/crush.json"
install_in "$t/hs" >/dev/null
check "crush.json là symlink có provider -> theo symlink, không tạo crushrc, đích không đổi" "$([[ ! -e "$t/hs/.config/dew/crushrc" && -L "$t/hs/.config/dew/crush.json" && "$(cat "$t/store/real.json")" == "$cj" ]]; echo $?)"
mkdir -p "$t/hn/.config/dew"; printf '%s' '{"options":{"tui":{"active_theme":"x"}}}' > "$t/hn/.config/dew/crush.json"
install_in "$t/hn" >/dev/null
check "crush.json không có provider -> tạo crushrc + env, giữ crush.json" "$([[ -f "$t/hn/.config/dew/crushrc" && -f "$t/hn/.config/dew/env" && -e "$t/hn/.config/dew/crush.json" ]]; echo $?)"
mkdir -p "$t/hb/.config/dew"; printf '%s' '{hỏng' > "$t/hb/.config/dew/crush.json"
install_in "$t/hb" >/dev/null
check "crush.json hỏng -> coi như có provider, không tạo crushrc" "$([[ ! -e "$t/hb/.config/dew/crushrc" ]]; echo $?)"
# Thiếu jq -> không xác định được provider -> an toàn: không tạo crushrc.
mkdir -p "$t/nojq" "$t/hj/.config/dew"
for c in bash env dirname mkdir install find chmod ln cat cp mv rm ls tail date sha1sum cmp grep sed wc cut tr readlink; do ln -sf "$(command -v $c)" "$t/nojq/$c"; done
printf '%s' "$cj" > "$t/hj/.config/dew/crush.json"
msg_j="$(PATH="$t/nojq" install_in "$t/hj")"
check "thiếu jq (PATH không có jq) -> không tạo crushrc, in hướng dẫn" "$([[ ! -e "$t/hj/.config/dew/crushrc" && -f "$t/hj/.config/dew/crushrc.example" ]] && grep -q 'KHÔNG tạo crushrc' <<<"$msg_j"; echo $?)"

# dew-tmux được chép (install -m 755), không còn symlink vào checkout.
tmux_inst="$t/inst/bin/dew-tmux"
check "dew-tmux là bản chép (không symlink), mode 755, giống tệp trong repo" "$([[ -f "$tmux_inst" && ! -L "$tmux_inst" && "$(stat -c %a "$tmux_inst")" == 755 ]] && cmp -s "$here/scripts/dew-tmux.sh" "$tmux_inst"; echo $?)"
mv "$t/fake/scripts/dew-tmux.sh" "$t/fake/scripts/dew-tmux.sh.away"
out="$(bash "$tmux_inst" --help 2>&1)" || true
check "dew-tmux đã cài vẫn chạy khi tệp trong repo biến mất" "$(grep -q 'Dùng:' <<<"$out"; echo $?)"
mv "$t/fake/scripts/dew-tmux.sh.away" "$t/fake/scripts/dew-tmux.sh"
# Bản cài cũ là symlink vào repo: cài lại phải thay chính symlink, không ghi xuyên vào tệp trong repo.
rm -f "$tmux_inst"; ln -s "$t/fake/scripts/dew-tmux.sh" "$tmux_inst"
before_sum="$(cksum < "$t/fake/scripts/dew-tmux.sh")"
install1 >/dev/null
check "cài lại thay symlink dew-tmux của bản cài cũ bằng bản chép" "$([[ -f "$tmux_inst" && ! -L "$tmux_inst" ]]; echo $?)"
check "thay symlink không ghi xuyên vào tệp trong repo" "$([[ "$(cksum < "$t/fake/scripts/dew-tmux.sh")" == "$before_sum" ]]; echo $?)"

# DEW_INSTALL_DIR tương đối: wrapper phải ghi đường dẫn binary tuyệt đối.
mkdir -p "$t/relcwd/rel-inst/lib/dew"
cp "$t/inst/lib/dew/crush" "$t/relcwd/rel-inst/lib/dew/crush"
(cd "$t/relcwd" && HOME="$t/home" DEW_INSTALL_DIR=rel-inst DEW_SKIP_BUILD=1 bash "$t/fake/scripts/install-dew.sh" >/dev/null 2>&1)
check "DEW_INSTALL_DIR tương đối -> wrapper ghi đường dẫn binary tuyệt đối" "$(grep -Fqx "exec \"$t/relcwd/rel-inst/lib/dew/crush\" \"\$@\"" "$t/relcwd/rel-inst/bin/dew"; echo $?)"
out="$(cd / && HOME="$t/home" "$t/relcwd/rel-inst/bin/dew" run x 2>&1)" || true
check "wrapper cài bằng đường dẫn tương đối chạy được từ thư mục khác" "$(grep -qx 'ARG=run' <<<"$out"; echo $?)"

# Thư mục cấu hình là symlink: siết thư mục ĐÍCH (của user) và nói rõ; symlink hỏng thì dừng rõ ràng.
mkdir -p "$t/hl/.config" "$t/hl-real"; chmod 755 "$t/hl-real"
ln -s "$t/hl-real" "$t/hl/.config/dew"
msg_l="$(install_in "$t/hl")"
check "cấu hình là symlink -> siết thư mục đích về 700, symlink còn nguyên, nói rõ" "$([[ "$(stat -c %a "$t/hl-real")" == 700 && -L "$t/hl/.config/dew" ]] && grep -q 'symlink' <<<"$msg_l" && grep -qF "$t/hl-real" <<<"$msg_l"; echo $?)"
check "cấu hình là symlink -> crushrc và env tạo trong thư mục đích" "$([[ -f "$t/hl-real/crushrc" && -f "$t/hl-real/env" ]]; echo $?)"
mkdir -p "$t/hd/.config"; ln -s "$t/hd-missing" "$t/hd/.config/dew"
rc=0; msg_d="$(install_in "$t/hd")" || rc=$?
check "cấu hình là symlink hỏng -> dừng mã khác 0, nói rõ, không tạo đích" "$([[ "$rc" != 0 && ! -e "$t/hd-missing" ]] && grep -q 'symlink hỏng' <<<"$msg_d"; echo $?)"

# --- Chuyển từ tên cũ agent-cli. Bố cục cũ dựng bằng CHÍNH trình cài cũ (commit trước khi đổi tên),
# rồi thêm env có key canary, crush.json, registry, worktree routine, log server.
old_rev=5680d80b
m="$t/mig"; mh="$m/home"
mkdir -p "$m/fake/scripts" "$m/inst/lib/agent-cli" "$mh"
for f in scripts/install-agent-cli.sh scripts/agent-cli-tmux.sh agent-cli.crushrc.example agent-cli.env.example; do
  git -C "$here" show "$old_rev:$f" > "$m/fake/$f"
done
cp "$t/inst/lib/dew/crush" "$m/inst/lib/agent-cli/crush"
HOME="$mh" AGENT_CLI_INSTALL_DIR="$m/inst" AGENT_CLI_SKIP_BUILD=1 bash "$m/fake/scripts/install-agent-cli.sh" >/dev/null 2>&1
old_cfg="$mh/.config/agent-cli"
check "bố cục cũ dựng được bằng trình cài cũ" "$([[ -f "$m/inst/bin/agent-cli" && -f "$m/inst/bin/agent-cli-tmux" && -f "$old_cfg/crushrc" && -f "$old_cfg/env" ]]; echo $?)"
cat > "$old_cfg/env" <<'ENV'
# AGENT_CLI_MODEL trong comment cũng đổi
AGENT_CLI_API_KEY='CANARY_SECRET_9f3'
export AGENT_CLI_ALLOWED_MODELS="zai:glm-5.3"
MY_AGENT_CLI_KEEP=1
ENV
printf '%s\n' '{"providers":{"p":{"api_key":"$AGENT_CLI_API_KEY"}}}' > "$old_cfg/crush.json"
echo 'export AGENT_CLI_MODEL=CANARY_RC_5e2' > "$mh/.bashrc"
cp -p "$old_cfg/env" "$m/env.orig"; cp -p "$old_cfg/crushrc" "$m/crushrc.orig"
mkdir -p "$mh/.local/share/agent-cli/routines/r1/wt" "$mh/.cache/agent-cli/server-x"
echo log > "$mh/.cache/agent-cli/server-x/crush.log"
git init -q "$m/repo" && git -C "$m/repo" -c user.email=t@t -c user.name=t commit -q --allow-empty -m init
git -C "$m/repo" worktree add -q -b routine/r1-x "$mh/.local/share/agent-cli/routines/r1/wt/x" HEAD
printf '{"projects":[{"path":"%s","data_dir":"%s/.crush"},{"path":"/w/agent-cli-fork","data_dir":"/w/agent-cli-fork/.crush"}]}\n' \
  "$mh/.local/share/agent-cli/routines/r1/wt/x" "$mh/.local/share/agent-cli/routines/r1/wt/x" > "$mh/.local/share/agent-cli/projects.json"
# install_mig: trình cài MỚI trên cùng HOME/thư mục cài; shell export một biến tên cũ mang giá trị canary.
install_mig() { HOME="$mh" DEW_INSTALL_DIR="$m/inst" DEW_SKIP_BUILD=1 AGENT_CLI_TOKEN_CANARY=CANARY_SHELL_7d1 bash "$t/fake/scripts/install-dew.sh" 2>&1; }
rc=0; msg_m1="$(install_mig)" || rc=$?
cfgn="$mh/.config/dew"
check "chuyển đổi: rc 0" "$([[ $rc == 0 ]]; echo $?)"
check "chuyển đổi: 4 thư mục cũ chuyển sang dew, không còn thư mục cũ" "$([[ -d "$cfgn" && -d "$mh/.local/share/dew" && -f "$mh/.cache/dew/server-x/crush.log" && -f "$m/inst/lib/dew/crush" && ! -e "$old_cfg" && ! -e "$mh/.local/share/agent-cli" && ! -e "$mh/.cache/agent-cli" && ! -e "$m/inst/lib/agent-cli" ]]; echo $?)"
check "chuyển đổi: thư mục cấu hình giữ mode 700, env giữ 600" "$([[ "$(stat -c %a "$cfgn")" == 700 && "$(stat -c %a "$cfgn/env")" == 600 ]]; echo $?)"
printf '%s\n' "# DEW_MODEL trong comment cũng đổi" "DEW_API_KEY='CANARY_SECRET_9f3'" 'export DEW_ALLOWED_MODELS="zai:glm-5.3"' "MY_AGENT_CLI_KEEP=1" > "$m/env.want"
check "chuyển đổi: env đổi đúng tên biến, giữ biến chỉ chứa chuỗi cũ ở giữa tên" "$(cmp -s "$m/env.want" "$cfgn/env"; echo $?)"
check "chuyển đổi: in đúng số chỗ đổi trong env (3)" "$(grep -qF "Đã đổi 3 chỗ tên cũ (AGENT_CLI_ -> DEW_, đường .../agent-cli -> .../dew) trong $cfgn/env" <<<"$msg_m1"; echo $?)"
check "chuyển đổi: crushrc cũ không còn AGENT_CLI_, tệp env trỏ đường mới" "$(! grep -q AGENT_CLI_ "$cfgn/crushrc" && grep -qF 'env_file="${DEW_ENV_FILE:-$HOME/.config/dew/env}"' "$cfgn/crushrc"; echo $?)"
check "chuyển đổi: crush.json đổi \$AGENT_CLI_API_KEY" "$(grep -qF '"$DEW_API_KEY"' "$cfgn/crush.json"; echo $?)"
bk="$(ls -d "$cfgn"/pre-dew-* 2>/dev/null | head -1)"
check "chuyển đổi: bản sao lưu env/crushrc mode 600, đúng nội dung cũ" "$([[ -n "$bk" && "$(stat -c %a "$bk/env")" == 600 ]] && cmp -s "$bk/env" "$m/env.orig" && cmp -s "$bk/crushrc" "$m/crushrc.orig"; echo $?)"
check "chuyển đổi: registry trỏ đường mới, đường lạ có chữ agent-cli giữ nguyên" "$(grep -qF "\"$mh/.local/share/dew/routines/r1/wt/x\"" "$mh/.local/share/dew/projects.json" && ! grep -qF '.local/share/agent-cli' "$mh/.local/share/dew/projects.json" && grep -qF '"/w/agent-cli-fork"' "$mh/.local/share/dew/projects.json"; echo $?)"
check "chuyển đổi: worktree routine nối lại với repo (không prunable)" "$(git -C "$m/repo" worktree list | grep -F "$mh/.local/share/dew/routines/r1/wt/x" | grep -vq prunable; echo $?)"
check "chuyển đổi: gỡ agent-cli và agent-cli-tmux của trình cài, có dew và dew-tmux" "$([[ ! -e "$m/inst/bin/agent-cli" && ! -e "$m/inst/bin/agent-cli-tmux" && -x "$m/inst/bin/dew" && -x "$m/inst/bin/dew-tmux" ]]; echo $?)"
check "chuyển đổi: cảnh báo nêu TÊN biến shell cũ" "$(grep -q 'AGENT_CLI_TOKEN_CANARY' <<<"$msg_m1"; echo $?)"
check "chuyển đổi: output KHÔNG chứa giá trị canary (tệp env, shell)" "$(! grep -q 'CANARY_SECRET_9f3\|CANARY_SHELL_7d1' <<<"$msg_m1"; echo $?)"
check "chuyển đổi: in lệnh quay lại" "$(grep -q 'Quay về agent-cli' <<<"$msg_m1"; echo $?)"
check "chuyển đổi: liệt kê tệp khởi động shell còn tên cũ, không in nội dung" "$(grep -qF "  $mh/.bashrc" <<<"$msg_m1" && ! grep -q CANARY_RC_5e2 <<<"$msg_m1"; echo $?)"
# Lần 2: không đổi gì ở cấu hình/dữ liệu/cache (lib có backup-* mới của mỗi lần cài là bình thường).
snap() { (cd "$mh" && find . -path ./.local/share/dew/routines -prune -o -printf '%p %m %s\n' | sort; find . -type f -path './.config/*' -exec md5sum {} + | sort); }
before="$(snap)"; rc=0; msg_m2="$(install_mig)" || rc=$?
check "chạy lại lần 2: rc 0, không chuyển/đổi gì, cây cấu hình y nguyên" "$([[ $rc == 0 && "$(snap)" == "$before" ]] && ! grep -q 'Đã chuyển\|Đã đổi\|Đã gỡ\|Quay về agent-cli' <<<"$msg_m2"; echo $?)"
# Lệnh quay lại in ở lần 1 phải đưa máy về bố cục cũ chạy được (binary cũ, wrapper cũ, env tên cũ).
check "lệnh quay lại in ra = $bk/undo.sh" "$(grep -qF "bash $bk/undo.sh" <<<"$msg_m1" && [[ "$(sed -n '/tức là:$/,$p' <<<"$msg_m1" | sed -n 's/^  //p' | head -n "$(($(wc -l <"$bk/undo.sh") - 2))")" == "$(tail -n +3 "$bk/undo.sh")" ]]; echo $?)"
rc=0; HOME="$mh" bash "$bk/undo.sh" >"$t/undo.log" 2>&1 || { rc=$?; sed "s/^/  undo: /" "$t/undo.log"; }
check "lệnh quay lại chạy được, trả lại bố cục cũ" "$([[ $rc == 0 && -f "$m/inst/bin/agent-cli" && ! -e "$m/inst/bin/dew" && -d "$old_cfg" && ! -e "$cfgn" && -f "$m/inst/lib/agent-cli/crush" ]] && cmp -s "$old_cfg/env" "$m/env.orig"; echo $?)"
out="$(cd "$m/repo" && HOME="$mh" "$m/inst/bin/agent-cli" run x 2>&1)" || true
check "sau quay lại: wrapper cũ chạy binary ở lib/agent-cli" "$(grep -qx 'ARG=run' <<<"$out"; echo $?)"

# Bố cục cũ dựng bằng trình cài cũ, dùng cho các ca dừng giữa chừng.
mk_old() { # mk_old <thư mục>
  mkdir -p "$1/home" "$1/inst/lib/agent-cli"
  cp "$t/inst/lib/dew/crush" "$1/inst/lib/agent-cli/crush"
  HOME="$1/home" AGENT_CLI_INSTALL_DIR="$1/inst" AGENT_CLI_SKIP_BUILD=1 bash "$m/fake/scripts/install-agent-cli.sh" >/dev/null 2>&1
}
tree_of() { (cd "$1" && find . -printf '%p %m\n' | sort); }
# go build lỗi trên máy còn bố cục cũ: build chạy TRƯỚC mọi chuyển đổi nên không đổi gì.
g="$t/gofail"; mk_old "$g"; before="$(tree_of "$g")"
mkdir -p "$t/gofailbin"; printf '#!/bin/sh\necho "build broke" >&2\nexit 1\n' > "$t/gofailbin/go"; chmod 755 "$t/gofailbin/go"
rc=0; msg_g="$(HOME="$g/home" PATH="$t/gofailbin:$PATH" DEW_INSTALL_DIR="$g/inst" bash "$t/fake/scripts/install-dew.sh" 2>&1)" || rc=$?
check "go build lỗi trên bố cục cũ: rc!=0, cây y nguyên (không chuyển, không để thư mục build tạm)" "$([[ $rc != 0 && "$(tree_of "$g")" == "$before" ]] && grep -q 'chưa đổi gì' <<<"$msg_g"; echo $?)"
# Dừng giữa chuyển đổi (env không đọc được): rc!=0, in lệnh quay lại, undo.sh trả bố cục cũ.
if [[ "$(id -u)" != 0 ]]; then
  z="$t/env000"; mk_old "$z"; chmod 000 "$z/home/.config/agent-cli/env"
  rc=0; msg_z="$(HOME="$z/home" DEW_INSTALL_DIR="$z/inst" DEW_SKIP_BUILD=1 bash "$t/fake/scripts/install-dew.sh" 2>&1)" || rc=$?
  undo_z="$(ls "$z"/home/.config/dew/pre-dew-*/undo.sh 2>/dev/null | head -1)"
  check "env mode 000: rc!=0, báo dừng giữa chừng kèm undo.sh" "$([[ $rc != 0 && -f "$undo_z" ]] && grep -q 'Cài DỪNG giữa chừng' <<<"$msg_z" && grep -qF "$undo_z" <<<"$msg_z"; echo $?)"
  rc=0; HOME="$z/home" bash "$undo_z" >"$t/undo-z.log" 2>&1 || { rc=$?; sed "s/^/  undo: /" "$t/undo-z.log"; }
  check "env mode 000: undo.sh trả bố cục cũ" "$([[ $rc == 0 && -d "$z/home/.config/agent-cli" && ! -e "$z/home/.config/dew" && -f "$z/inst/lib/agent-cli/crush" && ! -e "$z/inst/lib/dew" && -f "$z/inst/bin/agent-cli" ]]; echo $?)"
  chmod 600 "$z/home/.config/agent-cli/env"
else
  echo "skip ca env mode 000: chạy bằng root"
fi

# crush.json chung là symlink GÃY (vd đã dời/đổi tên thư mục chứa tệp đích): trình cài không đổi được tên cũ bên
# trong, provider khai ở đó mất lặng lẽ -> phải cảnh báo, nêu symlink và đích.
y="$t/brokenlink"; mk_old "$y"; ln -sfn "$y/nowhere/.crush.json" "$y/home/.config/agent-cli/crush.json"
msg_y="$(HOME="$y/home" DEW_INSTALL_DIR="$y/inst" DEW_SKIP_BUILD=1 bash "$t/fake/scripts/install-dew.sh" 2>&1)" || true
check "crush.json là symlink gãy -> cảnh báo nêu symlink và đích" "$(grep -q 'symlink gãy' <<<"$msg_y" && grep -qF "$y/nowhere/.crush.json" <<<"$msg_y"; echo $?)"

# Có thư mục dữ liệu cũ nhưng không có worktree routine (ca thường gặp): undo.sh vẫn thoát 0.
u="$t/noroutine"; mk_old "$u"; mkdir -p "$u/home/.local/share/agent-cli"; echo '{"projects":[]}' > "$u/home/.local/share/agent-cli/projects.json"
HOME="$u/home" DEW_INSTALL_DIR="$u/inst" DEW_SKIP_BUILD=1 bash "$t/fake/scripts/install-dew.sh" >/dev/null 2>&1
undo_u="$(ls "$u"/home/.config/dew/pre-dew-*/undo.sh 2>/dev/null | head -1)"
rc=0; HOME="$u/home" bash "$undo_u" >"$t/undo-u.log" 2>&1 || { rc=$?; sed "s/^/  undo: /" "$t/undo-u.log"; }
check "thư mục dữ liệu cũ, không routine: undo.sh rc 0, trả bố cục cũ" "$([[ -n "$undo_u" && $rc == 0 && -d "$u/home/.local/share/agent-cli" && ! -e "$u/home/.local/share/dew" && -d "$u/home/.config/agent-cli" && -f "$u/inst/bin/agent-cli" ]]; echo $?)"

# Cả hai thư mục cùng có: KHÔNG gộp, giữ nguyên cả hai, dùng dew. Wrapper agent-cli không do trình cài tạo: giữ.
b="$t/both"; mkdir -p "$b/home/.config/agent-cli" "$b/home/.config/dew" "$b/inst/lib/dew" "$b/inst/bin"
echo 'AGENT_CLI_MODEL=old' > "$b/home/.config/agent-cli/env"; echo 'DEW_MODEL=new' > "$b/home/.config/dew/env"
printf '#!/bin/sh\necho mine\n' > "$b/inst/bin/agent-cli"
mkdir -p "$b/home/.config/systemd/user"; echo 'ExecStart=/x/bin/agent-cli routine run n' > "$b/home/.config/systemd/user/agent-cli-routine-n.service"
cp "$t/inst/lib/dew/crush" "$b/inst/lib/dew/crush"
msg_b="$(HOME="$b/home" DEW_INSTALL_DIR="$b/inst" DEW_SKIP_BUILD=1 bash "$t/fake/scripts/install-dew.sh" 2>&1)" || true
check "cả hai thư mục: không gộp, cả hai nguyên vẹn, cảnh báo" "$([[ "$(cat "$b/home/.config/agent-cli/env")" == AGENT_CLI_MODEL=old && "$(cat "$b/home/.config/dew/env")" == DEW_MODEL=new ]] && grep -q 'KHÔNG gộp' <<<"$msg_b"; echo $?)"
check "unit systemd tên cũ: cảnh báo nêu tệp, không sửa" "$(grep -qF "$b/home/.config/systemd/user/agent-cli-routine-n.service" <<<"$msg_b" && grep -q 'ExecStart=/x/bin/agent-cli' "$b/home/.config/systemd/user/agent-cli-routine-n.service"; echo $?)"
check "wrapper agent-cli lạ: giữ nguyên, cảnh báo" "$([[ "$(cat "$b/inst/bin/agent-cli")" == "$(printf '#!/bin/sh\necho mine')" ]] && grep -q 'không do trình cài tạo' <<<"$msg_b"; echo $?)"

# Còn tiến trình chạy binary cũ: dừng trước khi đụng gì.
r="$t/running"; mkdir -p "$r/home/.config/agent-cli" "$r/inst/lib/agent-cli"
cp "$(command -v sleep)" "$r/inst/lib/agent-cli/crush"
"$r/inst/lib/agent-cli/crush" 30 & sleep_pid=$!
rc=0; msg_r="$(HOME="$r/home" DEW_INSTALL_DIR="$r/inst" DEW_SKIP_BUILD=1 bash "$t/fake/scripts/install-dew.sh" 2>&1)" || rc=$?
kill "$sleep_pid" 2>/dev/null || true
check "tiến trình cũ đang chạy: rc 1, nêu pid, không chuyển gì" "$([[ $rc == 1 && -d "$r/home/.config/agent-cli" && ! -e "$r/home/.config/dew" && -d "$r/inst/lib/agent-cli" ]] && grep -q "pid $sleep_pid" <<<"$msg_r"; echo $?)"
rc=0; HOME="$t/home" AGENT_CLI_INSTALL_DIR="$t/inst" DEW_SKIP_BUILD=1 bash "$t/fake/scripts/install-dew.sh" >/dev/null 2>&1 || rc=$?
check "AGENT_CLI_INSTALL_DIR không có DEW_INSTALL_DIR: dừng mã 1" "$([[ $rc == 1 ]]; echo $?)"

# Repo đổi tên thư mục agent-cli -> dew: mục registry trỏ đường không còn mà anh em tên dew có thì được viết lại
# (path, data_dir, locations); mục khác giữ nguyên; có sao lưu + undo; chạy lại không đổi gì.
rn="$t/reporename"; rnh="$rn/home"; rnj="$rnh/.local/share/dew/projects.json"
mkdir -p "$rnh/.local/share/dew" "$rn/inst/lib/dew" "$rn/ws/dew/.crush" "$rn/ws/other/dew" "$rn/ws/agent-cli-x" "$rn/ws/keep/agent-cli" "$rn/ws/keep/dew"
cp "$t/inst/lib/dew/crush" "$rn/inst/lib/dew/crush"
cat > "$rnj" <<JSON
{"projects":[
{"path":"$rn/ws/agent-cli","data_dir":"$rn/ws/agent-cli/.crush","host":"$(uname -n)","pinned":true,"last_accessed":"2026-10-01T00:00:00Z","locations":[{"cwd":"$rn/ws/agent-cli/sub","data_dir":"$rn/ws/agent-cli/sub/.crush","last_accessed":"2026-10-01T00:00:00Z"},{"cwd":"$rn/ws/agent-cli-tools/x","data_dir":"$rn/ws/agent-cli-tools/.crush"}]},
{"path":"$rn/gone/agent-cli","data_dir":"$rn/gone/agent-cli/.crush","host":"$(uname -n)"},
{"path":"$rn/ws/other/agent-cli","data_dir":"$rn/ws/other/agent-cli/.crush","host":"other-host","machine_id":"other-machine"},
{"path":"$rn/ws/agent-cli-x","data_dir":"$rn/ws/agent-cli-x/.crush","host":"$(uname -n)"},
{"path":"$rn/ws/keep/agent-cli","data_dir":"$rn/ws/keep/agent-cli/.crush","host":"$(uname -n)"},
{"path":"$rnh/.local/share/agent-cli/routines/r/wt/x","data_dir":"x"},
{"path":"$rn/ws/agent-cli","data_dir":"$rn/ws/agent-cli/.crush","host":"other-host","machine_id":"other-machine"}
]}
JSON
rn_orig="$(cat "$rnj")"
rn_install() { HOME="$rnh" DEW_INSTALL_DIR="$rn/inst" DEW_SKIP_BUILD=1 bash "$t/fake/scripts/install-dew.sh" 2>&1; }
msg_rn1="$(rn_install)"
check "đổi tên repo: path/data_dir/locations của mục có anh em dew được viết lại" "$([[ "$(jq -r '.projects[0] | [.path, .data_dir, .locations[0].cwd, .locations[0].data_dir] | join("|")' "$rnj")" == "$rn/ws/dew|$rn/ws/dew/.crush|$rn/ws/dew/sub|$rn/ws/dew/sub/.crush" ]]; echo $?)"
check "đổi tên repo: trường khác (pinned, host, thời gian) giữ nguyên" "$([[ "$(jq -c '.projects[0] | [.pinned, .host, .last_accessed, .locations[0].last_accessed]' "$rnj")" == "[true,\"$(uname -n)\",\"2026-10-01T00:00:00Z\",\"2026-10-01T00:00:00Z\"]" ]]; echo $?)"
check "đổi tên repo: mục thiếu anh em dew, của máy khác, tên gần giống, đường còn tồn tại: giữ nguyên" "$([[ "$(jq -c '.projects[1:5]' "$rnj")" == "$(jq -c '.projects[1:5]' <<<"$rn_orig")" ]]; echo $?)"
check "đổi tên repo: cùng đường agent-cli nhưng của máy khác (registry dùng chung): giữ nguyên" "$([[ "$(jq -c '.projects[6]' "$rnj")" == "$(jq -c '.projects[6]' <<<"$rn_orig")" ]]; echo $?)"
check "đổi tên repo: location tên gần giống (agent-cli-tools) trong mục được viết lại: giữ nguyên, không cắt nửa tên" "$([[ "$(jq -c '.projects[0].locations[1]' "$rnj")" == "$(jq -c '.projects[0].locations[1]' <<<"$rn_orig")" ]]; echo $?)"
check "đổi tên repo: cùng lượt, đường dữ liệu agent-cli vẫn do bước đổi tên cũ xử lý (hai bước cùng một tệp)" "$([[ "$(jq -r '.projects[5].path' "$rnj")" == "$rnh/.local/share/dew/routines/r/wt/x" ]]; echo $?)"
check "đổi tên repo: có thông báo, đúng 1 mục (mục của máy khác không bị tính)" "$(grep -qF "Đã đổi đường repo agent-cli -> dew" <<<"$msg_rn1" && grep -qF "(1 mục)" <<<"$msg_rn1"; echo $?)"
rnbk="$(ls -d "$rnh"/.config/dew/pre-dew-* 2>/dev/null | head -1)"
check "đổi tên repo: sao lưu mode 600; bản gốc (projects.json) đúng nội dung cũ, bản trung gian còn đường repo agent-cli" "$([[ -n "$rnbk" && "$(stat -c %a "$rnbk/projects.before-repo-rename.json")" == 600 && "$(cat "$rnbk/projects.json")" == "$rn_orig" && "$(jq -r '.projects[0].path' "$rnbk/projects.before-repo-rename.json")" == "$rn/ws/agent-cli" ]]; echo $?)"
rn_snap="$(cat "$rnj")"; msg_rn2="$(rn_install)"
check "đổi tên repo: chạy lại lần 2 không đổi registry, không thông báo" "$([[ "$(cat "$rnj")" == "$rn_snap" ]] && ! grep -q "Đã đổi đường repo" <<<"$msg_rn2"; echo $?)"
rc=0; HOME="$rnh" bash "$rnbk/undo.sh" >"$t/undo-rn.log" 2>&1 || rc=$?
check "đổi tên repo: undo.sh trả registry y nguyên bản cũ" "$([[ $rc == 0 && "$(cat "$rnj")" == "$rn_orig" ]]; echo $?)"
# Anh em dew đã có mục riêng: không tạo hai mục trùng đường (Home báo mục agent-cli là missing, ctrl+x gỡ).
rd="$t/repodup"; mkdir -p "$rd/home/.local/share/dew" "$rd/inst/lib/dew" "$rd/ws/dew"; cp "$t/inst/lib/dew/crush" "$rd/inst/lib/dew/crush"
printf '{"projects":[{"path":"%s/ws/agent-cli","data_dir":"x"},{"path":"%s/ws/dew","data_dir":"y"}]}\n' "$rd" "$rd" > "$rd/home/.local/share/dew/projects.json"
HOME="$rd/home" DEW_INSTALL_DIR="$rd/inst" DEW_SKIP_BUILD=1 bash "$t/fake/scripts/install-dew.sh" >/dev/null 2>&1
check "đổi tên repo: đích đã đăng ký -> không gộp, không trùng đường" "$([[ "$(jq -r '[.projects[].path] | sort | join(",")' "$rd/home/.local/share/dew/projects.json")" == "$rd/ws/agent-cli,$rd/ws/dew" ]]; echo $?)"

# Không có /etc/machine-id lẫn /var/lib/dbus/machine-id: không rác stderr, vẫn so theo host và viết lại.
rm_="$t/repomid"; mkdir -p "$rm_/home/.local/share/dew" "$rm_/inst/lib/dew" "$rm_/ws/dew"; cp "$t/inst/lib/dew/crush" "$rm_/inst/lib/dew/crush"
printf '{"projects":[{"path":"%s/ws/agent-cli","data_dir":"x","host":"%s"}]}\n' "$rm_" "$(uname -n)" > "$rm_/home/.local/share/dew/projects.json"
HOME="$rm_/home" DEW_INSTALL_DIR="$rm_/inst" DEW_SKIP_BUILD=1 DEW_MACHINE_ID_FILES="$rm_/no1 $rm_/no2" bash "$t/fake/scripts/install-dew.sh" >/dev/null 2>"$rm_/err" || true
check "đổi tên repo: không có tệp machine-id -> stderr không có 'No such file'" "$(! grep -q 'No such file' "$rm_/err"; echo $?)"
check "đổi tên repo: không có tệp machine-id -> so theo host, vẫn viết lại" "$([[ "$(jq -r '.projects[0].path' "$rm_/home/.local/share/dew/projects.json")" == "$rm_/ws/dew" ]]; echo $?)"
# machine-id lấy từ tệp thứ hai khi tệp đầu không có (như /var/lib/dbus/machine-id sau /etc/machine-id): mục khác host nhưng cùng machine-id vẫn là của máy này.
echo "MID-TEST" > "$rm_/id2"
printf '{"projects":[{"path":"%s/ws/agent-cli","data_dir":"x","host":"other-host","machine_id":"MID-TEST"}]}\n' "$rm_" > "$rm_/home/.local/share/dew/projects.json"
HOME="$rm_/home" DEW_INSTALL_DIR="$rm_/inst" DEW_SKIP_BUILD=1 DEW_MACHINE_ID_FILES="$rm_/no1 $rm_/id2" bash "$t/fake/scripts/install-dew.sh" >/dev/null 2>&1 || true
check "đổi tên repo: machine-id từ tệp dự phòng (tệp đầu không có) -> mục cùng machine-id được viết lại" "$([[ "$(jq -r '.projects[0].path' "$rm_/home/.local/share/dew/projects.json")" == "$rm_/ws/dew" ]]; echo $?)"
# Registry không có đường agent-cli: bước im lặng hẳn, kể cả khi thiếu jq; có đường agent-cli mà thiếu jq: cảnh báo.
rj="$t/repojq"; mkdir -p "$rj/home/.local/share/dew" "$rj/inst/lib/dew"; cp "$t/inst/lib/dew/crush" "$rj/inst/lib/dew/crush"
echo '{"projects":[{"path":"/w/x","data_dir":"/w/x/.crush"}]}' > "$rj/home/.local/share/dew/projects.json"
msg_rj="$(HOME="$rj/home" PATH="$t/nojq" DEW_INSTALL_DIR="$rj/inst" DEW_SKIP_BUILD=1 bash "$t/fake/scripts/install-dew.sh" 2>&1)" || true
check "registry không có đường agent-cli + thiếu jq: không cảnh báo thiếu jq" "$(! grep -q 'thiếu jq, không kiểm được registry' <<<"$msg_rj"; echo $?)"
echo '{"projects":[{"path":"/w/agent-cli","data_dir":"/w/agent-cli/.crush"}]}' > "$rj/home/.local/share/dew/projects.json"
msg_rj="$(HOME="$rj/home" PATH="$t/nojq" DEW_INSTALL_DIR="$rj/inst" DEW_SKIP_BUILD=1 bash "$t/fake/scripts/install-dew.sh" 2>&1)" || true
check "registry có đường agent-cli + thiếu jq: cảnh báo thiếu jq" "$(grep -q 'thiếu jq, không kiểm được registry' <<<"$msg_rj"; echo $?)"

if [[ $fail == 0 ]]; then echo PASS; else echo "FAILED"; exit 1; fi
