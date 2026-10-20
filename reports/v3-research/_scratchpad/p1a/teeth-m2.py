# Kiểm răng: mỗi đột biến làm hỏng đúng một mảnh của bản sửa, chạy bộ test mới, ghi test nào đỏ.
# Dùng: python3 -I teeth.py <teeth-tree> <go.sh> <out.txt>
import subprocess, sys, shutil, re, os
tree, gosh, out = sys.argv[1:4]
AG = os.path.join(tree, "internal/agent/agent.go")
BE = os.path.join(tree, "internal/backend/agent.go")
orig = {p: open(p).read() for p in (AG, BE)}
def sub(path, old, new):
    def f(s):
        assert s.count(old) == 1, (path, old[:60])
        return s.replace(old, new)
    return (path, f)
M = {
 "M1_no_record_before_ack": [sub(BE, "if err := accept.Record(ws.ctx, call); err != nil {", "if err := error(nil); err != nil {")],
 "M2_drop_without_closing_trace": [sub(AG, "if err := a.persistCanceledTurn(ctx, d, true); err != nil {", "if err := error(nil); err != nil {")],
 "M3_never_move_pair": [sub(AG, "recordedAtTail = n >= 2 && msgs[n-2].ID == r.userMsgID && msgs[n-1].ID == r.pendingMsgID", "recordedAtTail = n >= 0")],
 "M4_fold_writes_new_user_msg": [sub(AG, "				if queued.recorded != nil {\n					var moved", "				if false {\n					var moved")],
 "M5_history_keeps_recorded_pair": [sub(AG, "msgs = slices.DeleteFunc(msgs, func(m message.Message) bool { return m.ID == r.userMsgID || m.ID == r.pendingMsgID })", "_ = r")],
 "M6_placeholder_never_removed": [sub(AG, "a.deleteMessages(callContext, call.SessionID, pendingDone...)", "_ = pendingDone")],
 "M7_cancel_ignores_recorded": [sub(AG, "	if call.recorded != nil {\n		pending, err := a.messages.Get(writeCtx, call.recorded.pendingMsgID)", "	if false {\n		pending, err := a.messages.Get(writeCtx, call.recorded.pendingMsgID)")],
}
res = []
for name, muts in [(k, v) for k, v in M.items() if k.startswith("M2")]:
    for p in orig: open(p, "w").write(orig[p])
    for path, f in muts:
        mutated = f(open(path).read())
        open(path, "w").write(mutated)
    r = subprocess.run([gosh, "test", "-p", "2", "-count=1", "-run", "TestSendMessage_|TestQueueMarkDropClosesRecordedPrompt",
                        "./internal/backend/", "./internal/agent/"], cwd=tree, capture_output=True, text=True)
    fails = sorted(set(re.findall(r"--- FAIL: (\S+)", r.stdout)))
    build = "build failed" if "[build failed]" in r.stdout or "cannot" in r.stderr else ""
    res.append("%s rc=%d %s red=%s" % (name, r.returncode, build, ",".join(fails) or "NONE"))
    print(res[-1], flush=True)
for p in orig: open(p, "w").write(orig[p])
open(out, "w").write("\n".join(res) + "\n")
