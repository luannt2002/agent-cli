"""Tách tickets/v2-issue-map.md (nhánh v2-design) thành TSV + gắn họ lỗi sơ bộ theo từ khoá.

Dùng: python3 -I parse_map.py <map.md> <outcomes.md> <issue-refs.tsv> <out.tsv>
Họ gắn bằng từ khoá trên title+requirement+reason: CHỈ là gợi ý để người đọc soát tay, không phải phân loại cuối.
"""
import re
import sys

map_md, outcomes_md, refs_tsv, out = sys.argv[1:5]

FAMILIES = [
    ("CAPABILITY_MISMATCH", r"subagent|agent_id|agent_type|deferred|toolsearch|cannot call|without ctx|mcpToolsAvailable"),
    ("ROUTING_FAILURE", r"routing|redirect|curl|wget|webfetch|deny|denied|nudge|hint|matcher|blocks? curl|hard-?block|inline http|build tool|gradle"),
    ("PROMPT_INTERFERENCE", r"classifier|auto-mode|directive|role ?:? ?\"?user|injected|injection|caveman|tone|framing|priority_instructions|system prompt"),
    ("CONTEXT_OVERHEAD", r"token|overhead|bloat|context window|\bkb\b|bytes|snapshot.*(size|196|large)|maxbytes|too (long|large)|duplicate (instructions|injection)"),
    ("STARTUP_RACE", r"race|not ready|isMCPReady|sentinel|cold|startup|first call|before server|ready"),
    ("ZOMBIE_OPERATION", r"zombie|orphan|busy-?loop|spin|instances|leak(ed)? process|never exit|hang|stuck|idle|parent-?liveness|ppid|kill"),
    ("FALSE_SUCCESS", r"silent|silently|no-?op|pass(es)? on|false (pass|fail)|green|reports? success|swallow|ignored|exit 1|isError"),
    ("DELETION_DATA_LOSS", r"delet|wipe|lost|loss|purge|cleanup|clean-?up|prune|overwrit|truncat|dropped|erase"),
    ("HOST_ADAPTER_MISMATCH", r"windows|wsl|cursor|codex|gemini|copilot|vscode|vs code|opencode|kiro|pi\b|omp|openclaw|zed|antigravity|agy|qwen|kimi|jetbrains|adapter|platform"),
    ("SECURITY_BOUNDARY_CONFUSION", r"secur|sandbox|ssrf|secret|token leak|redact|env leak|permission|escape|vulnerab|cve|isolation|deny rule|path traversal"),
    ("PROJECT_IDENTITY_FAILURE", r"project ?dir|projectdir|wrong (store|project|session)|worktree|cwd|other session|cross-?project|attribut|credited|latest session"),
    ("INSTALL_UPGRADE", r"install|upgrade|heal|npm|bun|abi|better-sqlite|ensure-deps|marketplace|plugin cache|version"),
]

def norm(s):
    return s.strip()

rows = []
section = None
for line in open(map_md, encoding="utf-8"):
    m = re.match(r"^# (I[0-3]|P[0-3])\s*$", line)
    if m:
        section = m.group(1)
        continue
    if not re.match(r"^\| \d+ ", line):
        continue
    cells = [norm(c) for c in line.rstrip("\n").split("|")[1:-1]]
    if len(cells) < 7:
        cells += [""] * (7 - len(cells))
    num, title, who, klass, req, reason, v2test = cells[:7]
    rows.append(dict(section=section, num=int(num), title=title, who=who, klass=klass.replace(" ", ""),
                     req=req, reason=reason, v2test=v2test))

verdict = {}
for line in open(outcomes_md, encoding="utf-8"):
    m = re.match(r"^\| (\d+) \| ([a-z-]+) \|", line)
    if m:
        verdict[int(m.group(1))] = m.group(2)

refs = {}
for line in open(refs_tsv, encoding="utf-8"):
    if line.startswith("#") or line.startswith("ref\t"):
        continue
    p = line.rstrip("\n").split("\t")
    if len(p) != 8:
        continue
    n = int(p[0].lstrip("#"))
    refs.setdefault(n, []).append(p[1])

with open(out, "w", encoding="utf-8") as f:
    f.write("section\tnum\tkind\tclass\tv2test\toutcome_r4\tmain_commits\tfamilies\ttitle\twho\n")
    for r in rows:
        text = (r["title"] + " " + r["req"] + " " + r["reason"]).lower()
        fams = [name for name, rx in FAMILIES if re.search(rx, text, re.I)]
        kind = "issue" if r["section"].startswith("I") else "PR"
        f.write("\t".join([r["section"], str(r["num"]), kind, r["klass"], r["v2test"],
                           verdict.get(r["num"], "-"), ",".join(sorted(set(refs.get(r["num"], [])))) or "-",
                           ",".join(fams) or "-", r["title"], r["who"]]) + "\n")

print("rows", len(rows), "unique", len({r["num"] for r in rows}), "with_outcome",
      sum(1 for r in rows if r["num"] in verdict), "with_main_commit", sum(1 for r in rows if r["num"] in refs))
