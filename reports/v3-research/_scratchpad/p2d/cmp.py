# So kết quả go test -json theo TÊN (Package/Test), không đếm số.
import json, sys
def load(p):
    res = {}
    for line in open(p):
        try:
            e = json.loads(line)
        except Exception:
            continue
        a = e.get("Action")
        if a in ("pass", "fail", "skip"):
            res[(e.get("Package"), e.get("Test") or "<pkg>")] = a
    return res
b, f = load(sys.argv[1]), load(sys.argv[2])
cnt = lambda r, a: sum(1 for v in r.values() if v == a)
for n, r in (("base", b), ("fix", f)):
    print("%s: names=%d pass=%d fail=%d skip=%d" % (n, len(r), cnt(r, "pass"), cnt(r, "fail"), cnt(r, "skip")))
assert len(b) > 0 and len(f) > 0, "rỗng không phải xanh"
new_fail = sorted(k for k, v in f.items() if v == "fail" and b.get(k) != "fail")
base_fail = sorted(k for k, v in f.items() if v == "fail" and b.get(k) == "fail")
fixed = sorted(k for k, v in b.items() if v == "fail" and f.get(k) == "pass")
missing = sorted(k for k in b if k not in f)
added = sorted(k for k in f if k not in b)
for title, ks in (("NEW FAIL (đỏ mới)", new_fail), ("fail cả hai (baseline)", base_fail), ("base fail -> fix pass", fixed),
                  ("có ở base, mất ở fix", missing), ("mới ở fix", added)):
    print("%s: %d" % (title, len(ks)))
    for k in ks:
        print("   %s %s  base=%s fix=%s" % (k[0], k[1], b.get(k), f.get(k)))
