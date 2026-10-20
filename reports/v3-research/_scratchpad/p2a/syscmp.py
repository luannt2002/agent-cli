# So system prompt hai lượt chạy exposure.sh: chuẩn hoá đường dẫn thư mục chạy + hash commit git của repo thử.
import sys, os, re, json, difflib
base, new, case = sys.argv[1], sys.argv[2], sys.argv[3]
def reqs(d):
    return [json.loads(l) for l in open(os.path.join(d, case, "log", "requests.jsonl"))]
def sysof(d, r):
    s = open(os.path.join(d, case, "log", "sys-%s.txt" % r["system_sha"])).read()
    s = s.replace(os.path.abspath(d), "<OUT>")
    return re.sub(r"(?m)^[0-9a-f]{7} init$", "<HASH> init", s)
A, B = reqs(base), reqs(new)
for role in sorted({r["role"] for r in A} | {r["role"] for r in B}):
    ra = [r for r in A if r["role"] == role]; rb = [r for r in B if r["role"] == role]
    if not ra or not rb:
        print("%-14s base=%d new=%d (missing one side)" % (role, len(ra), len(rb))); continue
    sa, sb = sysof(base, ra[0]), sysof(new, rb[0])
    rem = lambda r: any(u.startswith("<system_reminder>This is a reminder that your todo list") for u in r["user_texts"])
    print("%-14s sys_identical=%s bytes %d->%d | todo_reminder %s->%s | tools_identical=%s | mcp-instructions %s->%s" % (
        role, sa == sb, len(sa), len(sb), rem(ra[0]), rem(rb[0]), ra[0]["tools"] == rb[0]["tools"],
        "<mcp-instructions>" in sa, "<mcp-instructions>" in sb))
    if sa != sb:
        for l in list(difflib.unified_diff(sa.splitlines(), sb.splitlines(), lineterm="", n=0))[2:12]:
            print("     ", l[:150])
