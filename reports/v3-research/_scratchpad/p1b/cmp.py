#!/usr/bin/env python3
# So hai lượt `go test -json` theo TÊN (package + test), không theo số đếm.
# Dùng: python3 -I cmp.py <trước.json> <sau.json>
import json, sys


def load(path):
    res, pkg_fail = {}, set()
    with open(path) as f:
        for line in f:
            line = line.strip()
            if not line.startswith("{"):
                continue
            try:
                e = json.loads(line)
            except ValueError:
                continue
            a = e.get("Action")
            if a not in ("pass", "fail", "skip"):
                continue
            if e.get("Test"):
                res[(e["Package"], e["Test"])] = a
            elif a == "fail":
                pkg_fail.add(e["Package"])
    return res, pkg_fail


b, bp = load(sys.argv[1])
a, ap = load(sys.argv[2])
assert b and a, "empty run is not green: before=%d after=%d names" % (len(b), len(a))
cnt = lambda r, s: sum(1 for v in r.values() if v == s)
print("before: %d names (pass %d, fail %d, skip %d), failed pkgs %d" % (len(b), cnt(b, "pass"), cnt(b, "fail"), cnt(b, "skip"), len(bp)))
print("after:  %d names (pass %d, fail %d, skip %d), failed pkgs %d" % (len(a), cnt(a, "pass"), cnt(a, "fail"), cnt(a, "skip"), len(ap)))
print("failed before:", sorted("%s %s" % k for k, v in b.items() if v == "fail"))
print("failed after: ", sorted("%s %s" % k for k, v in a.items() if v == "fail"))
print("NEW failures (fail after, not fail before):", sorted("%s %s" % k for k, v in a.items() if v == "fail" and b.get(k) != "fail"))
print("FIXED (fail before, pass after):", sorted("%s %s" % k for k, v in b.items() if v == "fail" and a.get(k) == "pass"))
print("MISSING after (name present before, absent after):", sorted("%s %s" % k for k in b if k not in a))
print("ADDED after (absent before):", sorted("%s %s" % k for k in a if k not in b))
print("failed pkgs before:", sorted(bp))
print("failed pkgs after: ", sorted(ap))
