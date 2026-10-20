# So hai lần go test -json theo TÊN (package, test). In: tổng, test mất, đổi pass->fail, test mới và kết quả.
import json, sys, collections
def load(p):
    res, pk = {}, {}
    for l in open(p):
        try: e = json.loads(l)
        except Exception: continue
        if e.get("Action") in ("pass", "fail", "skip"):
            if e.get("Test"): res[(e["Package"].split("crush/")[-1], e["Test"])] = e["Action"]
            else: pk[e["Package"].split("crush/")[-1]] = e["Action"]
    return res, pk
a, pa = load(sys.argv[1]); b, pb = load(sys.argv[2])
assert len(a) > 0 and len(b) > 0, "empty run"
print("base: tests=%d %s | packages %s" % (len(a), dict(collections.Counter(a.values())), dict(collections.Counter(pa.values()))))
print("new : tests=%d %s | packages %s" % (len(b), dict(collections.Counter(b.values())), dict(collections.Counter(pb.values()))))
missing = sorted(set(a) - set(b)); added = sorted(set(b) - set(a))
print("missing in new (%d):" % len(missing)); [print("  ", k) for k in missing]
chg = [(k, a[k], b[k]) for k in sorted(set(a) & set(b)) if a[k] != b[k]]
print("status changed (%d):" % len(chg)); [print("  ", k, x, "->", y) for k, x, y in chg]
print("added in new (%d):" % len(added)); [print("  ", k, b[k]) for k in added]
fa = {k for k, v in a.items() if v == "fail"}; fb = {k for k, v in b.items() if v == "fail"}
print("fail sets identical by name:", fa == fb, "| base fails=%d new fails=%d" % (len(fa), len(fb)))
