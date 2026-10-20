"""Extract #NNN refs from every commit message reachable from a revision.

usage: python3 -I hs_refs.py <repo> <rev> <out.tsv>
Role of a ref:
  pr_merge  = trailing "(#N)" at end of subject (GitHub squash convention -> PR number; inference)
  subject   = other #N in subject
  closes    = "fixes|closes|resolves #N" in body
  body      = other #N in body
Cross-repo refs (owner/repo#N with repo != vectorize-io/hindsight) are skipped and counted.
"""
import re
import subprocess
import sys
from collections import defaultdict

repo, rev, out = sys.argv[1], sys.argv[2], sys.argv[3]
raw = subprocess.run(
    ["git", "-C", repo, "log", "--abbrev=9", "--date=short", "--format=%h%x1f%ad%x1f%s%x1f%b%x1e", rev],
    check=True, capture_output=True, text=True,
).stdout

REF = re.compile(r"(?:(?P<repo>[\w.-]+/[\w.-]+))?#(?P<n>\d+)\b")
CLOSES = re.compile(r"\b(?:fix(?:es|ed)?|close[sd]?|resolve[sd]?)\s*:?\s+(?:vectorize-io/hindsight)?#(\d+)\b", re.I)
TRAIL = re.compile(r"\(#(\d+)\)\s*$")
TYPE = re.compile(r"^(\w+)")

rows, skipped_cross, commits = [], 0, 0
for rec in raw.split("\x1e"):
    rec = rec.strip("\n")
    if not rec:
        continue
    commits += 1
    h, date, subj, body = rec.split("\x1f", 3)
    m = TYPE.match(subj)
    ctype = m.group(1).lower() if m else "other"
    roles = {}
    trail = TRAIL.search(subj)
    if trail:
        roles[trail.group(1)] = "pr_merge"
    for mm in REF.finditer(subj):
        if mm.group("repo") and mm.group("repo").lower() != "vectorize-io/hindsight":
            skipped_cross += 1
            continue
        roles.setdefault(mm.group("n"), "subject")
    closes = {x for x in CLOSES.findall(body)}
    for mm in REF.finditer(body):
        if mm.group("repo") and mm.group("repo").lower() != "vectorize-io/hindsight":
            skipped_cross += 1
            continue
        n = mm.group("n")
        if n not in roles:
            roles[n] = "closes" if n in closes else "body"
        elif roles[n] == "subject" and n in closes:
            roles[n] = "closes"
    title = subj.replace("\t", " ")
    for n, role in roles.items():
        rows.append([int(n), h, date, title, role, ctype])

# repeat candidates: non-pr_merge ref present in >=2 distinct commits whose type is fix
fix_commits = defaultdict(set)
for n, h, _, _, role, ctype in rows:
    if role != "pr_merge" and ctype == "fix":
        fix_commits[n].add(h)
lap = {n: len(s) for n, s in fix_commits.items() if len(s) >= 2}
# a PR number seen as pr_merge in >1 commit (revert / cherry-pick)
pr_dupe = defaultdict(set)
for n, h, _, _, role, _ in rows:
    if role == "pr_merge":
        pr_dupe[n].add(h)
pr_dupe = {n: len(s) for n, s in pr_dupe.items() if len(s) >= 2}

rows.sort(key=lambda r: (r[0], r[2], r[1]))
watch = {5251, 2413, 2347}
with open(out, "w", encoding="utf-8") as f:
    f.write(f"# Nguồn (chỉ đọc, chạy 2026-10-19): git -C {repo} log --abbrev=9 --date=short --format='%h%x1f%ad%x1f%s%x1f%b%x1e' {rev}\n")
    f.write("# Tách ref bằng Python 3 (re), trên subject + body của MỌI commit tới được từ rev:\n")
    f.write(r"#   REF    = (?:(?P<repo>[\w.-]+/[\w.-]+))?#(?P<n>\d+)\b   (repo khác vectorize-io/hindsight thì bỏ)" + "\n")
    f.write(r"#   CLOSES = \b(?:fix(?:es|ed)?|close[sd]?|resolve[sd]?)\s*:?\s+(?:vectorize-io/hindsight)?#(\d+)\b  (không phân biệt hoa thường, chỉ trong body)" + "\n")
    f.write(r"#   TRAIL  = \(#(\d+)\)\s*$  trên subject" + "\n")
    f.write("# Cột loai = <vai_trò>:<loại_commit>[;LAP=k]\n")
    f.write("#   vai_trò: pr_merge = (#N) cuối subject (quy ước squash của GitHub -> số PR; SUY LUẬN, không gọi API);\n")
    f.write("#            subject = #N khác trong subject; closes = Fixes/Closes/Resolves #N trong body; body = #N khác trong body.\n")
    f.write("#   loại_commit = từ đầu subject (fix/feat/docs/perf/...).\n")
    f.write("#   LAP=k: ref (không tính vai pr_merge) có mặt ở k>=2 commit loại fix KHÁC NHAU -> ỨNG VIÊN bug lặp.\n")
    f.write("#          Phải đọc từng commit trước khi kết luận: có thể là follow-up cùng PR, PR nhắc làm ngữ cảnh, hoặc nhiễu.\n")
    f.write("# Giới hạn: không phân biệt issue/PR ngoài quy ước trên; '#N' trong văn xuôi (vd '#1 on BEAM') là nhiễu;\n")
    f.write("#   họ bug lặp dưới SỐ KHÁC NHAU (vd #1751 -> #5542) không bị cột LAP bắt — xem 02_UPSTREAM_DELTA.md.\n")
    f.write(f"# Thống kê: {commits} commit, {len(rows)} dòng ref, {len({r[0] for r in rows})} số khác nhau, "
            f"{skipped_cross} ref chéo repo bị bỏ, {len(lap)} ref LAP>=2, {len(pr_dupe)} số pr_merge ở >=2 commit "
            f"({', '.join('#' + str(k) for k in sorted(pr_dupe))}).\n")
    for n in sorted(watch):
        hit = [r for r in rows if r[0] == n]
        f.write(f"# #{n}: " + ("; ".join(f"{r[1]} {r[2]} vai={r[4]}" for r in hit) if hit else "KHÔNG commit nào nhắc trong message") + "\n")
    f.write("# Ghi chú tay (pickaxe, không phải message): #5251 chỉ có trong NỘI DUNG mã của d3f0001ed (2026-10-12, PR #5508);\n")
    f.write("#   #2413 còn trong nội dung test của 815d99f5b (2026-06-26, PR #2418). Lệnh: git log -S'#5251' 5331da2f4\n")
    f.write("ref\tcommit\tngay\ttieu_de\tloai\n")
    for n, h, date, title, role, ctype in rows:
        tag = f"{role}:{ctype}" + (f";LAP={lap[n]}" if n in lap and role != "pr_merge" else "")
        f.write(f"#{n}\t{h}\t{date}\t{title}\t{tag}\n")

top = sorted(lap.items(), key=lambda kv: -kv[1])
print("commits", commits, "rows", len(rows), "distinct", len({r[0] for r in rows}), "cross", skipped_cross,
      "lap", len(lap), "pr_dupe", len(pr_dupe))
print("top LAP:", top[:40])
print("pr_dupe:", sorted(pr_dupe.items())[:30])
