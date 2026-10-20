#!/usr/bin/env python3
# So hai lần `go test -json ./...` theo TÊN (gói/test), không theo số đếm.
# Dùng: python3 -I cmp.py <trước.json> <sau.json>
import json, sys
def load(p):
    res = {}
    for line in open(p):
        try:
            e = json.loads(line)
        except Exception:
            continue
        if e.get("Test") and e.get("Action") in ("pass", "fail", "skip"):
            res[e["Package"].split("crush/")[-1] + " " + e["Test"]] = e["Action"]
    return res
b, a = load(sys.argv[1]), load(sys.argv[2])
assert len(b) > 0 and len(a) > 0, "empty run is not green"
print("names: before=%d after=%d" % (len(b), len(a)))
for title, keys in [
    ("FAIL before", [k for k in b if b[k] == "fail"]),
    ("FAIL after", [k for k in a if a[k] == "fail"]),
    ("lost (before only)", [k for k in b if k not in a]),
    ("new (after only)", [k for k in a if k not in b]),
    ("status changed", [k for k in a if k in b and a[k] != b[k]]),
]:
    print("== %s: %d" % (title, len(keys)))
    for k in sorted(keys):
        print("   %s %s->%s" % (k, b.get(k, "-"), a.get(k, "-")))
