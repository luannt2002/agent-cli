import json, sys
def load(p):
    r = {}
    for l in open(p):
        try: d = json.loads(l)
        except Exception: continue
        if d.get("Test") and d.get("Action") in ("pass", "fail", "skip"):
            r[(d["Package"].rsplit("/", 1)[-1], d["Test"])] = d["Action"]
    return r
b, n = load(sys.argv[1]), load(sys.argv[2])
assert len(b) > 0 and len(n) > 0, "empty result set"
print(f"base={len(b)} new={len(n)}")
print("base fail:", sorted(k for k, v in b.items() if v == "fail"))
print("new fail:", sorted(k for k, v in n.items() if v == "fail"))
print("missing in new:", sorted(set(b) - set(n)))
print("added in new:"); [print("  ", k, n[k]) for k in sorted(set(n) - set(b))]
print("status changed:", sorted((k, b[k], n[k]) for k in set(b) & set(n) if b[k] != n[k]))
for pkg in sorted({k[0] for k in n}):
    print(pkg, "base", sum(1 for k in b if k[0] == pkg), "new", sum(1 for k in n if k[0] == pkg))
