#!/usr/bin/env python3
"""So hai lượt `go test -json` theo TÊN (package + test). Dùng: suite-compare.py <base.json> <after.json>"""
import json
import sys


def load(path):
    res, pkgs = {}, {}
    for line in open(path, errors="replace"):
        try:
            e = json.loads(line)
        except ValueError:
            continue
        act, pkg, test = e.get("Action"), e.get("Package", ""), e.get("Test")
        if act not in ("pass", "fail", "skip"):
            continue
        if test:
            res[(pkg, test)] = act
        else:
            pkgs[pkg] = act
    return res, pkgs


b, bp = load(sys.argv[1])
a, ap = load(sys.argv[2])
print("base: %d tests (%d fail), %d pkg (%d fail)" % (len(b), sum(v == "fail" for v in b.values()), len(bp), sum(v == "fail" for v in bp.values())))
print("after: %d tests (%d fail), %d pkg (%d fail)" % (len(a), sum(v == "fail" for v in a.values()), len(ap), sum(v == "fail" for v in ap.values())))
lost = sorted(set(b) - set(a))
new = sorted(set(a) - set(b))
changed = sorted(k for k in set(a) & set(b) if a[k] != b[k])
print("lost (in base, not in after): %d" % len(lost))
for k in lost:
    print("  LOST %s %s (%s)" % (k[0], k[1], b[k]))
print("new: %d" % len(new))
for k in new:
    print("  NEW %s %s -> %s" % (k[0], k[1], a[k]))
print("status changed: %d" % len(changed))
for k in changed:
    print("  CHANGED %s %s: %s -> %s" % (k[0], k[1], b[k], a[k]))
print("fail in base (top-level):")
for k in sorted(k for k, v in b.items() if v == "fail" and "/" not in k[1]):
    print("  BASEFAIL %s %s" % k)
print("fail in after (top-level):")
for k in sorted(k for k, v in a.items() if v == "fail" and "/" not in k[1]):
    print("  AFTERFAIL %s %s" % k)
print("pkg fail base: %s" % sorted(p for p, v in bp.items() if v == "fail"))
print("pkg fail after: %s" % sorted(p for p, v in ap.items() if v == "fail"))
