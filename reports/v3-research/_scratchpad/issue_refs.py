# Sinh issue-refs.tsv từ git log của một repo (chỉ đọc).
# Dùng: python3 -I issue_refs.py <repo> <rev> <out.tsv>
import re, subprocess, sys
from collections import defaultdict

repo, rev, out = sys.argv[1], sys.argv[2], sys.argv[3]
SEP, END = "\x1f", "\x1e"
raw = subprocess.run(
    ["git", "-C", repo, "log", rev, "--format=%H" + SEP + "%ad" + SEP + "%s" + SEP + "%B" + END, "--date=short"],
    capture_output=True, text=True, check=True).stdout
# ref cùng repo: "#123" không đứng sau chữ/số/"/" (loại owner/repo#123) và không theo sau bởi chữ/số
REF = re.compile(r"(?<![\w/&])#(\d{1,5})(?![\w])")

def kind(subject):
    s = subject.strip()
    if re.match(r'(?i)^revert\b', s): return "revert"
    m = re.match(r'^([a-zA-Z]+)(\([^)]*\))?!?:', s)
    if m:
        t = m.group(1).lower()
        if t in ("fix", "feat", "revert"): return t
    return "khác"

rows = []
for rec in raw.split(END):
    rec = rec.strip("\n")
    if not rec: continue
    h, d, subj, body = rec.split(SEP, 3)
    refs = sorted({int(x) for x in REF.findall(subj + "\n" + body)})
    for r in refs:
        rows.append((r, h[:8], d, subj.replace("\t", " "), kind(subj)))

fix_commits = defaultdict(set); fix_subjects = defaultdict(set)
for r, h, d, s, k in rows:
    if k == "fix":
        fix_commits[r].add(h)
        fix_subjects[r].add(re.sub(r"\s+", " ", s.strip().lower()))

with open(out, "w") as f:
    f.write("# Sinh bởi: python3 -I issue_refs.py <repo> %s <out>  (script: scratchpad/issue_refs.py, nội dung tóm tắt dưới)\n" % rev)
    f.write("#   git -C <repo> log %s --format='%%H<US>%%ad<US>%%s<US>%%B<RS>' --date=short\n" % rev)
    f.write("#   ref = regex (?<![\\w/&])#(\\d{1,5})(?![\\w]) trên subject+body (loại owner/repo#N, &#N; HTML entity)\n")
    f.write("#   loại = tiền tố subject: fix|feat|revert (kể cả 'Revert \"...\"'), còn lại = khác\n")
    f.write("#   fix_commits = số commit loại fix nhắc ref; fix_subjects = số subject fix KHÁC NHAU (gộp commit trùng do merge/cherry-pick)\n")
    f.write("#   repeat = REPEAT khi fix_subjects >= 2 (ứng viên 'bug lặp', overlay mục 17) — CHỈ là ứng viên, chưa đọc diff\n")
    f.write("#   Giới hạn: số '#N' trong message có thể không phải issue (vd 'Issue #4' trỏ repo khác, '#1 of 5'); không lọc thủ công.\n")
    f.write("ref\tcommit\tngày\ttiêu đề commit\tloại\tfix_commits\tfix_subjects\trepeat\n")
    for r, h, d, s, k in sorted(rows, key=lambda x: (x[0], x[2], x[1])):
        fc, fs = len(fix_commits[r]), len(fix_subjects[r])
        f.write(f"#{r}\t{h}\t{d}\t{s}\t{k}\t{fc}\t{fs}\t{'REPEAT' if fs >= 2 else ''}\n")

refs = {r for r, *_ in rows}
rep = sorted(r for r in refs if len(fix_subjects[r]) >= 2)
print("rows", len(rows), "distinct_refs", len(refs), "repeat_candidates", len(rep))
print("commits_with_refs", len({h for _, h, *_ in rows}))
print("top repeat:", sorted(((len(fix_subjects[r]), r) for r in rep), reverse=True)[:30])
