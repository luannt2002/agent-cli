#!/usr/bin/env python3
"""Trích mọi #N trong commit message (subject + body) -> TSV.
Dùng: python3 -I refs.py <gitlog.txt> <out.tsv>
gitlog.txt = git log --format='%x1e%H%x1f%ad%x1f%s%x1f%b' --date=short
"""
import re, sys, collections

src, out = sys.argv[1], sys.argv[2]
raw = open(src, encoding='utf-8', errors='replace').read()
REF = re.compile(r'(?<![\w/&])#(\d{1,5})(?!\w)')
CLOSE = re.compile(r'\b(close[sd]?|fix(e[sd])?|resolve[sd]?)\b[:\s]*((?:#\d+[,\s]*(?:and\s+)?)+)', re.I)
TYPE = re.compile(r'^(revert|merge|fix|feat|docs?|refactor|test|tests|chore|perf|build|ci|style|release|hotfix)\b', re.I)

rows = []
for rec in raw.split('\x1e'):
    if not rec.strip():
        continue
    parts = rec.split('\x1f')
    if len(parts) < 4:
        continue
    h, date, subj, body = parts[0].strip(), parts[1], parts[2], parts[3]
    msg = subj + '\n' + body
    m = TYPE.match(subj)
    ctype = m.group(1).lower() if m else 'other'
    if ctype in ('doc',): ctype = 'docs'
    if ctype in ('tests',): ctype = 'test'
    if subj.startswith('Merge'): ctype = 'merge'
    closes = set()
    for cm in CLOSE.finditer(msg):
        closes.update(re.findall(r'#(\d+)', cm.group(3)))
    pr_self = set(re.findall(r'\(#(\d+)\)\s*$', subj))
    mp = re.match(r'Merge pull request #(\d+)', subj)
    if mp: pr_self.add(mp.group(1))
    seen = set()
    for r in REF.findall(msg):
        if r in seen:
            continue
        seen.add(r)
        if r in closes: rel = 'closes'
        elif r in pr_self: rel = 'pr'
        else: rel = 'mention'
        insubj = re.search(r'(?<![\w/&])#'+r+r'(?!\w)', subj) is not None
        rows.append((int(r), h[:10], date, subj.replace('\t', ' '), ctype, rel, insubj))

# ứng viên bug lặp: ref xuất hiện ở >=2 commit fix với quan hệ khác 'pr' (không tính số PR của chính commit)
fixcount = collections.Counter()
loose = collections.Counter()
for ref, h, d, s, t, rel, insubj in rows:
    if t == 'fix' and rel != 'pr':
        loose[ref] += 1
        if insubj or rel == 'closes':
            fixcount[ref] += 1

rows.sort(key=lambda x: (x[0], x[2], x[1]))
with open(out, 'w', encoding='utf-8') as f:
    f.write('# claude-mem issue/PR refs trong commit message (subject + body)\n')
    f.write('# repo: /home/luannt/application/opensource/v3-src/claude-mem  HEAD=3fae1eb45ba1a7fbd4818527f48b439991a273da (3479 commit)\n')
    f.write("# lệnh 1: git log --format='%x1e%H%x1f%ad%x1f%s%x1f%b' --date=short > gitlog.txt\n")
    f.write('# lệnh 2: python3 -I refs.py gitlog.txt issue-refs.tsv   (regex (?<![\\w/&])#(\\d{1,5})(?!\\w)); refs.py là script tạm trong scratchpad phiên, logic = đúng các luật mô tả ở các dòng # này\n')
    f.write('# kiểu commit: tiền tố subject khớp ^(revert|merge|fix|feat|docs?|refactor|test|tests|chore|perf|build|ci|style|release|hotfix)\\b (không phân biệt hoa thường), subject bắt đầu "Merge" -> merge, còn lại -> other\n')
    f.write('# thống kê lúc tạo: 3249 dòng (ref, commit), 2228 ref khác nhau, 18 ref REPEAT_FIX (quy tắc chặt), 208 nếu tính cả ref chỉ ở body\n')
    f.write('# loại = <kiểu commit theo tiền tố subject>:<quan hệ>; quan hệ: closes (closes/fixes/resolves #N), pr (số PR của chính commit: "(#N)" cuối subject hoặc "Merge pull request #N"), mention (nhắc khác)\n')
    f.write('# REPEAT_FIX=n: ref nằm trong SUBJECT (hoặc closes/fixes) của n>=2 commit fix, không tính số PR của chính commit -> ứng viên bug lặp (overlay mục 17). (body) = ref chỉ ở body; body hay liệt kê hàng loạt ref liên quan nên KHÔNG tính vào REPEAT_FIX. Vẫn cần lọc tay.\n')
    f.write('# giới hạn: #N trong body có thể là số bước/plan chứ không phải issue (nhiễu, chủ yếu N nhỏ); subject chứa TAB đã thay bằng dấu cách\n')
    f.write('ref\tcommit\tngay\ttieu_de\tloai\n')
    for ref, h, d, s, t, rel, insubj in rows:
        kind = f'{t}:{rel}' + ('' if insubj else '(body)')
        if fixcount[ref] >= 2:
            kind += f';REPEAT_FIX={fixcount[ref]}'
        f.write(f'#{ref}\t{h}\t{d}\t{s}\t{kind}\n')

nref = len({r[0] for r in rows})
rep = sorted([(c, r) for r, c in fixcount.items() if c >= 2], reverse=True)
print('rows', len(rows), 'distinct_refs', nref, 'repeat_fix_refs_strict', len(rep), 'loose', sum(1 for c in loose.values() if c>=2))
print('top repeat:', rep[:40])
