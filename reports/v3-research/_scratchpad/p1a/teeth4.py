import subprocess, sys, re, os
tree, gosh, out = sys.argv[1:4]
RUN = os.path.join(tree, "internal/cmd/run.go")
MS = os.path.join(tree, "internal/message/message.go")
orig = {p: open(p).read() for p in (RUN, MS)}
M = {
 "Y1_reply_model_keeps_placeholder": (RUN, ' && !message.IsPromptPlaceholder(m) {', ' {', "./internal/cmd/", "TestLastReplyModelSkipsPromptPlaceholders"),
 "Y2_batch_allows_empty_create": (MS, "	if len(create) == 0 {\n		return nil, errors.New(\"message: WriteBatch needs at least one message to create\")\n	}\n", "", "./internal/message/", "TestWriteBatch"),
}
res = []
for name, (path, old, new, pkg, run) in M.items():
    for p in orig: open(p, "w").write(orig[p])
    s = open(path).read(); assert s.count(old) == 1, name
    open(path, "w").write(s.replace(old, new))
    r = subprocess.run([gosh, "test", "-p", "2", "-count=1", "-run", run, pkg], cwd=tree, capture_output=True, text=True)
    fails = sorted(set(re.findall(r"--- FAIL: (\S+)", r.stdout)))
    res.append("%s rc=%d red=%s" % (name, r.returncode, ",".join(fails) or "NONE"))
for p in orig: open(p, "w").write(orig[p])
open(out, "w").write("\n".join(res) + "\n")
