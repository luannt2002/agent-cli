import json, sys
fails, passes, pkgfail, races = set(), set(), set(), set()
for line in open(sys.argv[1], errors="replace"):
    try:
        e = json.loads(line)
    except Exception:
        continue
    a, t, p = e.get("Action"), e.get("Test"), e.get("Package")
    if a == "fail":
        (fails.add(p + " " + t) if t else pkgfail.add(p))
    elif a == "pass" and t:
        passes.add(p + " " + t)
    if a == "output" and "WARNING: DATA RACE" in e.get("Output", ""):
        races.add(p + " " + (t or "-"))
print("passed_tests=%d failed_tests=%d failed_pkgs=%d race_warnings_in=%d" % (len(passes), len(fails), len(pkgfail), len(races)))
for x in sorted(fails): print("FAIL", x)
for x in sorted(pkgfail): print("PKGFAIL", x)
for x in sorted(races): print("RACE", x)
