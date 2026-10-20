# Dùng: python3 -I names.py <go-test.json> -> in "status<TAB>pkg<TAB>test" cho mọi test (kể cả subtest) và gói.
import json, sys
res = {}
for line in open(sys.argv[1], errors="replace"):
    line = line.strip()
    if not line.startswith("{"):
        continue
    try:
        e = json.loads(line)
    except Exception:
        continue
    a = e.get("Action")
    if a in ("pass", "fail", "skip"):
        res[(e.get("Package", ""), e.get("Test", ""))] = a
for (p, t), a in sorted(res.items()):
    print("%s\t%s\t%s" % (a, p, t or "<package>"))
