import json, sys
# In tập (package, test) -> kết quả cuối; tóm tắt số pass/fail/skip và danh sách fail.
res = {}
pkgs = {}
for l in open(sys.argv[1]):
    try: e = json.loads(l)
    except Exception: continue
    a = e.get("Action")
    if a not in ("pass", "fail", "skip"): continue
    if e.get("Test"): res[(e["Package"], e["Test"])] = a
    else: pkgs[e["Package"]] = a
from collections import Counter
print("tests:", dict(Counter(res.values())), "packages:", dict(Counter(pkgs.values())))
for k, v in sorted(res.items()):
    if v == "fail" and "/" not in k[1]: print("FAIL", k[0], k[1])
for k, v in sorted(pkgs.items()):
    if v == "fail": print("PKGFAIL", k)
json.dump({"%s %s" % k: v for k, v in res.items()}, open(sys.argv[2], "w"))
