W=/home/luannt/application/opensource/agent-cli-p03; S=$(mktemp -d); L=$W/scripts/agent-cli-tmux.sh
mk() { mkdir -p "$S/$1/scripts"; cp "$W/scripts/test-agent-cli-tmux.sh" "$S/$1/scripts/"; }
mk r0; git -C "$W" show HEAD:scripts/agent-cli-tmux.sh > "$S/r0/scripts/agent-cli-tmux.sh"
mk m1; sed 's|\[\[ \$c != "\$abs" && \$c != "\$abs"/\* \]\]|[[ $c != "$abs" ]]|' "$L" > "$S/m1/scripts/agent-cli-tmux.sh"
mk m2; sed 's|ids=\$(find_window "\$abs" all)|ids=$(find_window "$abs")|' "$L" > "$S/m2/scripts/agent-cli-tmux.sh"
mk m3; sed 's|\$c != "\$abs"/\*|$c != "$abs"*|' "$L" > "$S/m3/scripts/agent-cli-tmux.sh"
mk m4; sed 's|index(\$2, d "/") == 1|index($2, d) == 1|' "$L" > "$S/m4/scripts/agent-cli-tmux.sh"
for m in r0 m1 m2 m3 m4; do diff -q "$L" "$S/$m/scripts/agent-cli-tmux.sh" >/dev/null && echo "$m: sed không đổi gì!"
  out=$(bash "$S/$m/scripts/test-agent-cli-tmux.sh" 2>&1); echo "$m rc=$? | $(tail -1 <<< "$out")"; done
