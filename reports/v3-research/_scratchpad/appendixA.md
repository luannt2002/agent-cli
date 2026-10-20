| # | tạo | tiêu đề (rút gọn, theo bản đồ) | họ (chính; phụ) | lớp | v2 test | tên ca trong tệp test | r4 |
|---|---|---|---|---|---|---|---|
| 1282 | 2026-10-10 | Cursor MCP spawn fails, bare npx not on PATH | HOST; INSTALL | B | `GP1/absolute-node-path-cursor-windows` | có | wont-fix |
| 1280 | 2026-10-08 | turndown missing, silent bg install fails | FALSE; INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1274 | 2026-10-08 | OpenCode resume snapshot injected into other session | IDENT; PI | A | `REMOVED:D9` | — | covered-by-test |
| 1269 | 2026-10-06 | curl detector aborts whole compound command | ROUTE | A | `REMOVED:ADR-0008` | — | closed-by-design |
| 1268 | 2026-10-06 | Native ZCode adapter | FEAT | Q | `N/A:deferred-post-v2.0` | — | wont-fix |
| 1267 | 2026-10-05 | Read tip re-injected every Read >50 KB (`??` fallback) | ROUTE; OVH | A | `REMOVED:D11` | — | covered-by-test |
| 1263 | 2026-10-04 | Pi injects memory as role:"user" | PI; HOST | A(partlyB) | `GP2/pi-hint-not-user-role` | có | wont-fix |
| 1262 | 2026-10-04 | parent-liveness spawns `ps` every 30 s | PERF | B | `GP9/no-periodic-ps-spawn` | có | - |
| 1261 | 2026-10-04 | Pre/PostToolUse hooks cost 55-70 ms CPU per Bash | PERF | B | `GP3/hook-perf-posttool-async` | có | - |
| 1259 | 2026-10-04 | intent classifier replayed as live directive on Pi | PI; STALE | A | `REMOVED:D9` | — | covered-by-test |
| 1258 | 2026-10-04 | DeepSeek Harness support | FEAT | Q | `N/A:open-question` | — | wont-fix |
| 1255 | 2026-10-03 | opencode missing from clientInfo map, wrong store | HOST; IDENT,FALSE | A | `REMOVED:D3` | — | covered-by-test |
| 1254 | 2026-10-03 | ctx_stats skips OpenCode (v1 layout only) | STATS; HOST | A | `REMOVED:D3` | — | closed-by-design |
| 1251 | 2026-10-03 | ensure-deps npm retry storm, console windows | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1248 | 2026-10-02 | stats heartbeat reopens every session DB each minute | PERF | B | `GP8/stats-heartbeat-no-timer-db-open` | có | closed-by-design |
| 1242 | 2026-10-02 | ctx-debug.sh fails under POSIX sh | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1226 | 2026-09-30 | hooks.json has no default timeouts | INSTALL; DEL | B | `GP3/hook-timeout-present` | có | covered-by-test |
| 1225 | 2026-09-30 | vulnerable ip-address dep chain | SEC | B | `GP11/npm-audit-zero-high-critical` | có | covered-by-test |
| 1222 | 2026-09-29 | bare `mcp__` matcher matches nothing (CC 2.1.195) | ROUTE; FALSE,HOST | B | `GP11/hook-matcher-valid-regex` | có | closed-by-design |
| 1221 | 2026-09-29 | OMP hard-blocks curl in quotes/grep | ROUTE; HOST | A | `REMOVED:ADR-0008` | — | wont-fix |
| 1219 | 2026-09-28 | No docs for Codex Desktop install | DOC | B | `GP1/install-docs-per-host` | có | wont-fix |
| 1217 | 2026-09-28 | ctx_execute shell has PATH stripped to /usr/bin:/bin | HOST; EXEC | B | `GP4/windows-path-inherit` | có | - |
| 1215 | 2026-09-28 | project-scope plugin enabled globally by heal | INSTALL | A | `REMOVED:D2` | — | covered-by-test |
| 1214 | 2026-09-27 | server takes every project's events.md | IDENT; DEL | A | `REMOVED:D8` | — | closed-by-design |
| 1213 | 2026-09-27 | stats opens every session DB read-write | PERF | B | `GP8/sibling-db-readonly` | có | - |
| 1212 | 2026-09-27 | PowerShell tool not routed; Bash nudge once | ROUTE; HOST | B | `GP3/powershell-routed-like-bash` | có | covered-by-test |
| 1210 | 2026-09-27 | Codex Windows project dir falls back to plugin cache | IDENT; HOST | B | `GP7/codex-session-project-dir` | có | closed-by-design |
| 1208 | 2026-09-26 | ctx_execute python/node fails with shell-script shims | HOST; EXEC | B | `GP4/windows-pyenv-shim` | có | - |
| 1206 | 2026-09-26 | agy rejects hooks.json (grouped Stop) | HOST | B | `GP11/agy-hooks-schema-valid` | có | wont-fix |
| 1205 | 2026-09-26 | Pi bridge on Bun spins, disk I/O error | STORAGE; HOST | A | `REMOVED:D2` | — | covered-by-test |
| 1204 | 2026-09-25 | index-time preview + ctx_page tool | FEAT | Q | `N/A:open-question` | — | - |
| 1201 | 2026-09-24 | OpenClaw capture breaks computed import paths | HOST | B | `GP1/adapter-bundle-static-imports` | có | wont-fix |
| 1200 | 2026-09-24 | ctx_search rejects singular `query` | TOOLC | B | `GP5/search-param-schema-match` | có | covered-by-test |
| 1199 | 2026-09-24 | OpenCode 2.x plugin fails to load | HOST | B | `GP1/opencode-v2-plugin-api` | có | - |
| 1198 | 2026-09-24 | ctx_* events credited to latest session | IDENT | A | `REMOVED:D8` | — | - |
| 1196 | 2026-09-23 | ensure-deps re-codesigns .node every hook | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1193 | 2026-09-22 | Bun seed writes ABI-mismatched cache | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1192 | 2026-09-22 | integrate with pi-blackhole | FEAT | C | `N/A:deferred-post-v2.0` | — | wont-fix |
| 1191 | 2026-09-22 | healScript version filter unanchored | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1188 | 2026-09-21 | ensure-deps dlopen then rebuild, SIGSEGV | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1187 | 2026-09-21 | OpenCode V2 support | HOST | D | `N/A:duplicate-of-#1199` | — | - |
| 1186 | 2026-09-21 | Windows ctx_stats 4 s + 1.6 s execSync probes; coexistence detector | PERF; HOST | B | `GP10/windows-stats-no-execsync-probes` | có | wont-fix |
| 1185 | 2026-09-20 | multi-agent context isolation per worktree | IDENT | C | `GP7/worktree-label-session-first` | có | - |
| 1184 | 2026-09-19 | fetch via Firecrawl API | FEAT | Q | `REMOVED:D15` | — | covered-by-test |
| 1179 | 2026-09-18 | Pi context message without timestamp → 400 | HOST | B | `GP3/injected-message-required-fields` | KHÔNG | covered-by-test |
| 1175 | 2026-09-18 | ctx_execute ignores Esc / no progress | ZOMB | B | `GP4/cancel-kills-child-tree-1s` | có | covered-by-test |
| 1174 | 2026-09-17 | start.mjs background npm install dies silently | INSTALL; FALSE | A | `REMOVED:D2` | — | closed-by-design |
| 1173 | 2026-09-17 | infinite compaction loop burns quota | OVH; PI,ZOMB | A | `REMOVED:D9` | — | covered-by-test |
| 1168 | 2026-09-16 | Pi paths hardcoded to legacy ~/.pi dirs | HOST | B | `GP11/pi-discovery-dirs-doctor` | có | - |
| 1163 | 2026-09-15 | snippet slicing makes lone surrogates → 500s | ENC | B | `GP5/snippet-code-point-safe` | có | - |
| 1162 | 2026-09-15 | start-http.mjs lacks turndown install (systemd) | INSTALL | Q | `N/A:open-question` | — | closed-by-design |
| 1159 | 2026-09-14 | Windows runtime probes take 30 s before initialize | RACE; HOST | B | `GP4/cold-start-no-path-probes` | có | covered-by-test |
| 1156 | 2026-09-13 | eviction deletes critical events first | DEL; STALE | A | `REMOVED:D8` | — | closed-by-design |
| 1152 | 2026-09-13 | doctor says Codex hooks off when default on | FALSE; HOST | B | `GP11/codex-hooks-default-true` | KHÔNG | - |
| 1151 | 2026-09-13 | binary/HTTP bytes counted as saved tokens | STATS; FALSE | B | `GP10/binary-bytes-excluded-from-savings` | KHÔNG | closed-by-design |
| 1150 | 2026-09-13 | Pi renderCall drops args | HOST | B | `GP5/call-row-language-code-line` | có | covered-by-test |
| 1142 | 2026-09-09 | hook cwd rewrite defeats permission allowlist | ROUTE; SEC | B | `GP3/no-updated-input-rewrite` | có | covered-by-test |
| 1141 | 2026-09-09 | ctx-debug.sh dumps secrets; template requires pasting | SEC | B | `GP11/redact-secrets-in-doctor-output` | có | covered-by-test |
| 1140 | 2026-09-09 | cleanupOldSessions evicts active sessions | DEL | A | `REMOVED:D6` | — | closed-by-design |
| 1139 | 2026-09-09 | Windows npm install better-sqlite3 crashes | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1136 | 2026-09-08 | empty `<table>` crashes turndown | FETCH | B | `GP5/rowless-tables-survive` | có | covered-by-test |
| 1135 | 2026-09-08 | fetch error returns raw Node crash stack | FETCH | B | `GP4/fetch-error-one-line-cause` | có | - |
| 1134 | 2026-09-08 | Codex auto-review rejects echo rewrite | PI; ROUTE | A | `REMOVED:ADR-0008` | — | closed-by-design |
| 1131 | 2026-09-08 | CLAUDE.md continuity contradicts routing block | PI | A | `REMOVED:D11` | — | closed-by-design |
| 1130 | 2026-09-08 | CLAUDE.md says blocked, code does not block | PI; FALSE | A | `REMOVED:D11` | — | closed-by-design |
| 1125 | 2026-09-06 | curl pipe to head still redirected, block dropped | ROUTE | A | `REMOVED:ADR-0008` | — | closed-by-design |
| 1119 | 2026-09-03 | ctx_search negative limit | TOOLC | B | `GP5/negative-limit-rejected` | có | closed-by-design |
| 1117 | 2026-09-02 | NODE_OPTIONS prefix breaks `for`/`while` | EXEC | B | `GP4/env-via-spawn-env` | có | covered-by-test |
| 1112 | 2026-09-01 | Pi ephemeral tail injection breaks prompt cache | OVH; PI | B | `GP2/stable-prefix-across-requests` | có | covered-by-test |
| 1107 | 2026-08-31 | guidanceOnce keyed on ppid never throttles; 2149 tmp dirs | OVH; ROUTE | B | `GP3/hint-once-per-session` | có | closed-by-design |
| 1106 | 2026-08-31 | curl regex matches `which curl` | ROUTE | A | `REMOVED:ADR-0008` | — | closed-by-design |
| 1105 | 2026-08-31 | better-sqlite3 ABI rebuild loop | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1101 | 2026-08-29 | same as 1100 | ZOMB | D | `N/A:duplicate-of-#1100` | — | closed-by-design |
| 1100 | 2026-08-29 | Server orphaned after session close (100% CPU) | ZOMB | B | `GP9/stdin-eof-exit` | có | - |
| 1099 | 2026-08-28 | OMP deny points to ctx_execute that is not registered | CAP; RACE | A | `REMOVED:ADR-0008` | — | closed-by-design |
| 1098 | 2026-08-28 | Pi bridge failure after init is sticky | WORKER | B | `GP1/pi-bridge-retry-next-turn` | có | covered-by-test |
| 1096 | 2026-08-27 | healScript statSync ELOOP | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1094 | 2026-08-27 | Pi collapsed view shows only code fence | HOST | B | `GP4/pi-collapsed-view-plain-text-first-line` | có | covered-by-test |
| 1090 | 2026-08-27 | plugin.json command pinned to versioned node path | INSTALL | B | `GP1/plugin-json-unpinned-node` | có | - |
| 1088 | 2026-08-26 | Show input/progress live for ctx_execute | FEAT | C(later) | `N/A:deferred-post-v2.0` | — | wont-fix |
| 1085 | 2026-08-25 | OpenCode extra system message breaks Qwen single-system templates | PI; HOST | B | `GP2/opencode-single-system-message` | có | closed-by-design |
| 1083 | 2026-08-25 | Codex agent greps ~/.codex and ingests own history | FEAT | C(later) | `N/A:deferred-post-v2.0` | — | wont-fix |
| 1081 | 2026-08-24 | timeout unit (ms) missing from schema | TOOLC | B | `GP2/tool-schema-timeout-unit` | có | covered-by-test |
| 1079 | 2026-08-24 | cleanupStaleSources(14) drops unchanged sources by age | DEL | Q | `N/A:open-question` | — | closed-by-design |
| 1078 | 2026-08-24 | cache-heal gates on wrong plugin key | INSTALL; FALSE | A | `REMOVED:D2` | — | closed-by-design |
| 1077 | 2026-08-24 | ctx_* calls verbose in Codex; batch repeats sections per query | OVH | B | `GP5/batch-execute-dedup-section` | có | - |
| 1075 | 2026-08-24 | `//` and `~/` permission anchors never match; deny fails open | SEC | B | `GP3/v17-read-glob-deny-anchors` | có | - |
| 1069 | 2026-08-21 | Pi does not route read/grep, only bash HTTP | ROUTE; HOST | B | `GP3/pi-read-hint-parity` | có | covered-by-test |
| 1068 | 2026-08-21 | cm-fs preload marker leaks to grandchild stderr | EXEC | B | `GP4/no-preload-marker-leak-grandchild` | có | closed-by-design |
| 1067 | 2026-08-19 | pi collapsed row shows ```javascript fence | HOST | B | `GP6/code-echo-default` | có | - |
| 1065 | 2026-08-19 | Hosted Insights D1 SQL vars / owner team | FEAT | C(rejectforOSS) | `N/A:rejected` | — | closed-by-design |
| 1063 | 2026-08-18 | "Is the project abandoned?" | NA | D | `N/A:not-a-bug` | — | wont-fix |
| 1058 | 2026-08-17 | Uninstall leaves cache-heal hook, cache dir, live MCP | ZOMB | A+B | `GP1/clean-uninstall-no-residue` | có | covered-by-test |
| 1055 | 2026-08-15 | isMCPReady counts sibling sessions → WebFetch denied | RACE; CAP | A | `REMOVED:ADR-0008` | — | closed-by-design |
| 1053 | 2026-08-14 | isMCPReady swallows redirects in OpenCode embedded mode | RACE; FALSE,HOST | A | `REMOVED:ADR-0008` | — | closed-by-design |
| 1052 | 2026-08-14 | OpenCode `webfetch` alias missing | ROUTE; HOST | B | `GP3/tool-name-map-opencode-webfetch` | có | closed-by-design |
| 1050 | 2026-08-13 | Pi single-row collapsed rendering | FEAT | C(later) | `N/A:deferred-post-v2.0` | — | wont-fix |
| 1049 | 2026-08-12 | cache-heal re-injects duplicate absolute-path hook | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1048 | 2026-08-12 | Plan mode blocks ctx_batch_execute | CAP; HOST | Q | `N/A:open-question` | — | wont-fix |
| 1047 | 2026-08-12 | Configurable execution echo + summary arg | OVH | Q | `GP6/code-echo-default` | có | covered-by-test |
| 1046 | 2026-08-12 | LongHorizon-Harness integration | FEAT | C(reject) | `N/A:rejected` | — | wont-fix |
| 1045 | 2026-08-11 | neuforge-pay token metering | FEAT | C(reject) | `N/A:rejected` | — | wont-fix |
| 1039 | 2026-08-07 | ctx_fetch_and_index strips HTTP(S)_PROXY | FETCH; FALSE | B | `GP5/http-proxy-honored` | có | closed-by-design |
| 1037 | 2026-08-06 | WebFetch denied for subagents without ctx_* | CAP | A | `LIVE:T-SUB` | — | closed-by-design |
| 1036 | 2026-08-06 | OpenCode cumulative cost appended per step | STATS | B | `GP10/usage-events-idempotent-per-turn` | có | - |
| 1032 | 2026-08-04 | VS Code Remote-WSL root = VS Code install dir | IDENT; HOST | B | `GP7/workspace-folder-remote-wsl` | có | covered-by-test |
| 1031 | 2026-08-03 | Tool descriptions ~6.2K tokens | OVH; PI | Q/B | `GP2/tool-description-token-budget` | có | closed-by-design |
| 1028 | 2026-08-02 | PreCompact snapshot uses bare ctx_search | HOST; CAP | A | `REMOVED:D9` | — | closed-by-design |
| 1026 | 2026-08-01 | Bash nudge once-per-session vs periodic | ROUTE | B | `GP3/hint-once-per-session` | có | wont-fix |
| 1025 | 2026-08-01 | reduction_pct double-counts returned bytes | STATS | B | `GP10/reduction-pct-denominator` | có | - |
| 1024 | 2026-08-01 | Stale DB cleanup deletes live idle sessions (WAL mtime) | DEL | A | `REMOVED:D6` | — | closed-by-design |
| 1023 | 2026-08-01 | "ran N× longer" is byte ratio shown as time | STATS; FALSE | B | `GP10/stats-wording-bytes-not-time` | có | closed-by-design |
| 1022 | 2026-08-01 | Resume snapshot ignores maxBytes (196 KB) | OVH | A | `REMOVED:D9` | — | covered-by-test |
| 1021 | 2026-08-01 | README says Pi hooks High not Full | DOC | A | `REMOVED:D16` | — | closed-by-design |
| 1018 | 2026-07-30 | CC tool naming assumes plugin install | HOST; CAP | B | `GP3/tool-prefix-standalone-mcp` | có | closed-by-design |
| 1017 | 2026-07-30 | Copilot CLI / Codex root = plugin install dir | IDENT; SEC | B | `GP7/adapter-root-path-containment` | có | closed-by-design |
| 1016 | 2026-07-30 | Clean uninstall command | INSTALL | C(v2.0) | `N/A:deferred-v2.0-scope` | — | closed-by-design |
| 1014 | 2026-07-29 | No way to count docs per source; doctor PASS on empty index | FALSE | C(v2.0) | `N/A:deferred-v2.0-scope` | — | closed-by-design |
| 1006 | 2026-07-27 | WebFetch deny breaks claude.ai artifact URLs | ROUTE; CAP | A | `REMOVED:ADR-0008` | — | closed-by-design |
| 1005 | 2026-07-27 | ctx upgrade writes old-version paths (bun) | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 1004 | 2026-07-27 | buildRulesSection inlines AGENTS.md into resume | OVH | A | `REMOVED:D9` | — | closed-by-design |
| 1003 | 2026-07-27 | Denies defeated by parallel batches; no WebFetch opt-out | ROUTE; FALSE | A(part)/Q | `LIVE:T-PAR` | — | wont-fix |
| 1000 | 2026-07-26 | PreToolUse double-fires; no PostToolUse timeout | ROUTE; PERF | B | `GP1/hooks-json-one-matcher-per-tool` | có | closed-by-design |
| 999 | 2026-07-26 | start.mjs busy-loop, 23 instances | ZOMB | B | `GP9/idle-cpu-stdin-exit` | có | covered-by-test |
| 993 | 2026-07-23 | Windows hook commands fail under PowerShell | HOST | Q | `N/A:open-question` | — | closed-by-design |
| 992 | 2026-07-23 | Bun SQLITE_IOERR wedge; doctor green | STORAGE | A | `REMOVED:D2` | — | closed-by-design |
| 985 | 2026-07-22 | Multi-session contention on shared DB, busy-wait | STORAGE; ZOMB | A | `REMOVED:D3` | — | closed-by-design |
| 984 | 2026-07-22 | WebFetch no exemption for artifact URLs | ROUTE | D | `N/A:duplicate-of-#1006` | — | closed-by-design |
| 983 | 2026-07-21 | Pi reload leaks MCP helper | ZOMB | B | `GP9/pi-reload-no-leak` | có | closed-by-design |
| 982 | 2026-07-21 | Windows MCP orphans spin after session end | ZOMB; HOST | B | `GP9/windows-client-kill-exit` | có | closed-by-design |
| 979 | 2026-07-20 | Headless CC still denied (dead formatter) | ROUTE; HOST | A | `REMOVED:D7` | — | covered-by-test |
| 967 | 2026-07-17 | Injected <context_window_protection> trips auto-mode classifier | PI; CAP | A | `REMOVED:D11` | — | closed-by-design |
| 964 | 2026-07-16 | Env knob to disable SessionStart routing block | OVH; PI | A | `REMOVED:D11` | — | closed-by-design |
| 962 | 2026-07-16 | RFC: bound trigram/vocab growth for logs/JSONL | STORAGE | Q | `N/A:open-question` | — | closed-by-design |
| 961 | 2026-07-16 | RFC: indexed byte/chunk budgets for batch | OVH | B | `GP5/batch-index-byte-budget` | có | covered-by-test |
| 960 | 2026-07-15 | PreToolUse `mcp__` matcher hits all MCP tools | ROUTE | B | `GP3/unrelated-mcp-tool-no-hook` | có | - |
| 959 | 2026-07-15 | Pi: hung ctx_execute cannot be aborted | ZOMB | B | `GP4/hung-execute-cancel-kills-tree` | có | - |
| 956 | 2026-07-14 | prebuild-install deprecated warning | INSTALL | A | `REMOVED:D2` | — | closed-by-design |
| 954 | 2026-07-13 | execute_file rejects external worktrees, batch accepts | IDENT; SEC | B | `GP7/worktree-containment-consistent` | có | closed-by-design |
| 953 | 2026-07-13 | Codex Desktop Win keeps one MCP server per task, os error 32 | ZOMB; HOST | B | `GP9/windows-stdio-close-plugin-dir-deletable` | có | wont-fix |
| 951 | 2026-07-12 | Stale cm-fs-preload-<pid>.js via NODE_OPTIONS leak | EXEC; ZOMB | B | `GP4/node-options-no-leak-across-servers` | có | covered-by-test |
| 950 | 2026-07-11 | ctx_stats self-contradicts (part > whole) | STATS | B | `GP10/stats-totals-consistent` | có | - |
| 949 | 2026-07-11 | sessions/ has no retention | STORAGE | A | `REMOVED:D6` | — | covered-by-test |
| 947 | 2026-07-10 | batch timeout does not bound indexing/search | ZOMB | B | `GP4/batch-timeout-bounds-whole-handler` | có | - |
| 946 | 2026-07-10 | Agent-prompt injection blocked by auto-mode classifier | PI; CAP | A | `REMOVED:ADR-0008` | — | closed-by-design |
| 944 | 2026-07-10 | Codex external-file check reads wrong settings | SEC; HOST | Q | `N/A:open-question` | — | closed-by-design |
| 941 | 2026-07-09 | Codex guidance: JS default, not only | PI; HOST | A | `REMOVED:D11` | — | closed-by-design |
| 940 | 2026-07-09 | OS-specific Codex guidance overlay | OVH | A | `REMOVED:D11` | — | closed-by-design |
| 938 | 2026-07-09 | WebFetch redirect breaks claude.ai artifact URLs | ROUTE | A | `REMOVED:ADR-0008` | — | closed-by-design |
| 936 | 2026-07-09 | stdio ctx_execute hangs, no server timeout | ZOMB | B | `GP4/host-rpc-timeout-governs-no-default` | có | - |
| 925 | 2026-07-06 | Wrapper breaks commands starting with if/for/( | EXEC | B | `GP4/shell-syntax-with-spawn-env` | có | covered-by-test |
| 923 | 2026-07-06 | start.mjs chdir breaks terminal cwd | EXEC | B | `GP9/no-chdir-server-cwd-stable` | có | covered-by-test |
| 919 | 2026-07-05 | Support CommandCode CLI | FEAT | C | `N/A:deferred-post-v2.0` | — | closed-by-design |
| 915 | 2026-07-04 | runPool `capped` true when jobs < concurrency | FALSE | B | `GP8/runpool-capped-flag` | có | covered-by-test |
| 914 | 2026-07-04 | index --source bare label vs indexDirectory | TOOLC | Q | `N/A:open-question` | — | wont-fix |
| 911 | 2026-07-03 | session_continuity framing trips auto-mode classifier | PI | A | `REMOVED:D9` | — | covered-by-test |
| 908 | 2026-07-02 | Configurable redirect exclusions per domain | ROUTE | A | `REMOVED:ADR-0008` | — | closed-by-design |
| 905 | 2026-07-02 | Pi bridge Bun SQLITE_IOERR | STORAGE | A | `REMOVED:D2` | — | closed-by-design |
| 903 | 2026-07-02 | SessionStart truncation splits UTF-16 surrogates | ENC; PI | B | `GP4/code-point-safe-truncation-cut` | có | covered-by-test |
| 902 | 2026-07-01 | Resume snapshot stale, priority eviction | DEL; STALE | A | `REMOVED:D8` | — | closed-by-design |
| 901 | 2026-07-01 | upgrade updates one tree, doctor reads other (Win) | INSTALL; FALSE | Q | `N/A:open-question` | — | closed-by-design |
| 895 | 2026-06-29 | ctx_search ranks stale cross-session above fresh | RETR; STALE | A | `GP5/session-worktree-project-rank` | có | wont-fix |
| 894 | 2026-06-29 | Statusline stuck, build/ missing | FALSE; STATS | A | `REMOVED:D17` | — | closed-by-design |
| 891 | 2026-06-28 | Emacs Lisp runtime | FEAT | C | `N/A:deferred-post-v2.0` | — | wont-fix |
| 890 | 2026-06-27 | omp: `with {type:"json"}` import breaks validator | INSTALL | B | `GP1/bundle-no-import-attributes` | có | covered-by-test |
| 889 | 2026-06-27 | tarball missing scripts/*.mjs | INSTALL; FALSE | A/B | `GP1/npm-pack-includes-runtime-files` | có | covered-by-test |
| 880 | 2026-06-26 | cleanupStaleContentDBs race poisons store (Pi) | RACE; DEL | A | `REMOVED:D3` | — | - |
| 878 | 2026-06-25 | chunkMarkdown oversized paragraphs | OVH | B | `GP5/chunk-byte-cap` | có | covered-by-test |
| 874 | 2026-06-24 | OMP tool_result never replaces output | FALSE; HOST | Q | `N/A:open-question` | — | wont-fix |
| 873 | 2026-06-24 | omp install does not copy SYSTEM.md | HOST | A | `REMOVED:D11` | — | closed-by-design |
| 867 | 2026-06-22 | Self-heal only at open-time, mid-session corrupt | STORAGE; DEL | Q | `N/A:open-question` | — | closed-by-design |
| 865 | 2026-06-22 | ctx_stats ignores $CLAUDE_CONFIG_DIR | STATS | A | `REMOVED:D3` | — | closed-by-design |
| 45 | 2026-03-03 | Beta testers wanted | NA | D | `N/A:not-a-bug` | — | closed-by-design |
