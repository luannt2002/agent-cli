import json, sys
from collections import Counter


def load(path):
    res, pkgs = {}, {}
    for line in open(path):
        try:
            e = json.loads(line)
        except Exception:
            continue
        a = e.get("Action")
        if a not in ("pass", "fail", "skip"):
            continue
        if e.get("Test"):
            res[(e["Package"], e["Test"])] = a
        else:
            pkgs[e["Package"]] = a
    return res, pkgs


b, bp = load(sys.argv[1])
a, ap = load(sys.argv[2])
print("base: %d test, %d package, package FAIL=%s" % (len(b), len(bp), sorted(p for p, s in bp.items() if s == "fail")))
print("after: %d test, %d package, package FAIL=%s" % (len(a), len(ap), sorted(p for p, s in ap.items() if s == "fail")))
print("base status:", dict(Counter(b.values())), " after status:", dict(Counter(a.values())))
gone = sorted(set(b) - set(a))
new = sorted(set(a) - set(b))
changed = sorted(k for k in set(a) & set(b) if a[k] != b[k])
print("mất (%d):" % len(gone))
for k in gone:
    print("  -", k[0], k[1], b[k])
print("mới (%d):" % len(new))
for k in new:
    print("  +", k[0], k[1], a[k])
print("đổi trạng thái (%d):" % len(changed))
for k in changed:
    print("  ~", k[0], k[1], b[k], "->", a[k])
print("đỏ ở after:", sorted("%s %s" % k for k, s in a.items() if s == "fail"))
