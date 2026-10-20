"""Ghép map.tsv với tiêu đề đầy đủ/ngày tạo/nhãn (tickets/data/*.json nhánh v2-design), kết luận r4 và trích tham chiếu test.

Dùng: python3 -I enrich.py <dir chứa map.tsv, is.json, is2.json, isall.json, pr2.json, prall.json, outcomes-r4.md, testrefs.tsv, gp_status.tsv> <out.tsv>
"""
import json
import os
import re
import sys

d, out = sys.argv[1:3]
J = lambda n: json.load(open(os.path.join(d, n), encoding="utf-8"))
full = {x["number"]: x for x in J("isall.json")}
full.update({x["number"]: x for x in J("prall.json")})
created = {x["number"]: x.get("createdAt", "")[:10] for x in J("is2.json") + J("pr2.json")}
labels = {x["number"]: ",".join(l["name"] if isinstance(l, dict) else str(l) for l in x.get("labels", [])) for x in J("is.json")}

arch = {}
for line in open(os.path.join(d, "outcomes-r4.md"), encoding="utf-8"):
    m = re.match(r"^\| (\d+) \| ([a-z-]+) \| (.*?) \| (.*?) \| (true|false)", line)
    if m:
        arch[int(m.group(1))] = (m.group(2), m.group(4))

testrefs = {}
for line in open(os.path.join(d, "testrefs.tsv"), encoding="utf-8"):
    n, locs = line.rstrip("\n").split("\t")
    testrefs[int(n)] = locs
gp = {}
for line in open(os.path.join(d, "gp_status.tsv"), encoding="utf-8"):
    n, case, st = line.rstrip("\n").split("\t")
    gp[int(n)] = st

with open(os.path.join(d, "map.tsv"), encoding="utf-8") as f, open(out, "w", encoding="utf-8") as o:
    head = f.readline().rstrip("\n").split("\t")
    o.write("num\tkind\tcreated\tstate_scan\tlabels\tclass\tv2test\tgp_name_in_tests\tcited_in_test_files\toutcome_r4\tfamilies_kw\ttitle_full\tintroduced_by\n")
    for line in f:
        r = dict(zip(head, line.rstrip("\n").split("\t")))
        n = int(r["num"])
        fx = full.get(n, {})
        a = arch.get(n, ("-", ""))
        intro = ";".join(sorted(set(re.findall(r"INTRODUCED_BY:\s*([0-9a-f]{7,8})", a[1]))))
        locs = testrefs.get(n, "-")
        files = sorted({x.split(":")[0] for x in locs.split(",")}) if locs != "-" else []
        o.write("\t".join([str(n), r["kind"], created.get(n, "-"), fx.get("state", "-"), labels.get(n, "-") or "-",
                           r["class"], r["v2test"], gp.get(n, "-"), ";".join(files) or "-", a[0], r["families"],
                           fx.get("title", r["title"]).replace("\t", " "), intro or "-"]) + "\n")
print("ok")
