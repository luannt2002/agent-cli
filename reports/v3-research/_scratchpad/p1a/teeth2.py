# Kiểm răng vòng 2: mỗi đột biến hỏng đúng một mảnh của bản sửa; chạy toàn bộ test mới; ghi test đỏ.
# Dùng: python3 -I teeth2.py <teeth-tree> <go.sh> <out.txt>
import subprocess, sys, re, os
tree, gosh, out = sys.argv[1:4]
AG = os.path.join(tree, "internal/agent/agent.go")
BE = os.path.join(tree, "internal/backend/agent.go")
SF = os.path.join(tree, "internal/agent/settle_fork.go")
orig = {p: open(p).read() for p in (AG, BE, SF)}
def sub(path, old, new, count=1):
    def f(s):
        assert s.count(old) == count, (path, old[:70], s.count(old))
        return s.replace(old, new)
    return (path, f)
LOCKFREE = "defer func() { a.publishCanceledQueueDrops(drops) }() // sau khi nhả khoá\n\tmu := a.sessionMu(sessionID)\n\tmu.Lock()\n\tdefer mu.Unlock()\n"
LOCKED = "mu := a.sessionMu(sessionID)\n\tmu.Lock()\n\tdefer mu.Unlock()\n\tdefer func() { a.publishCanceledQueueDrops(drops) }()\n"
M = {
 "M1_no_record_before_ack": [sub(BE, "if err := accept.Record(ws.ctx, call); err != nil {", "if err := error(nil); err != nil {")],
 "M2_drop_without_closing_trace": [sub(AG, "if err := a.persistCanceledTurn(ctx, d, true); err != nil {", "if err := error(nil); err != nil {")],
 "M3_never_move_pair": [sub(AG, "recordedAtTail = i >= 0 && len(withoutUnsentPrompts(msgs[i:])) == 0", "recordedAtTail = i >= 0")],
 "M4_fold_writes_new_user_msg": [sub(AG, "				if queued.recorded != nil {\n					var moved", "				if false {\n					var moved")],
 "M6_placeholder_never_removed": [sub(AG, "a.deleteMessages(callContext, call.SessionID, pendingDone...)", "_ = pendingDone")],
 "M7_cancel_ignores_recorded": [sub(AG, "	if call.recorded != nil {\n		pending, err := a.messages.Get(writeCtx, call.recorded.pendingMsgID)", "	if false {\n		pending, err := a.messages.Get(writeCtx, call.recorded.pendingMsgID)")],
 "N1_run_history_keeps_unsent": [sub(AG, "	msgs = withoutUnsentPrompts(msgs)\n\n	// Generate title", "	\n\n	// Generate title")],
 "N2_summary_keeps_unsent": [sub(AG, "	msgs = withoutUnsentPrompts(msgs)\n	if len(msgs) == 0 {", "	if len(msgs) == 0 {")],
 "N3_run_error_leaves_placeholder": [sub(AG, "			a.failRecordedPrompt(call.SessionID, call.recorded, retErr)", "			_ = call")],
 "N4_prerun_error_leaves_placeholder": [sub(BE, "	accept.Fail(err)\n", "")],
 "N5_settle_ignores_pending": [sub(SF, "return c.IsSessionBusy(sessionID) || c.HasPendingPrompts(sessionID)", "return c.IsSessionBusy(sessionID)")],
 "N6a_cancel_drops_under_lock": [sub(AG, "	var drops []SessionAgentCall\n	// Đăng ký trước Unlock nên chạy sau khi nhả khoá: không ghi DB dưới khoá.\n	defer func() { a.publishCanceledQueueDrops(drops) }()\n	mu := a.sessionMu(sessionID)\n	mu.Lock()\n	defer mu.Unlock()\n", "	var drops []SessionAgentCall\n	mu := a.sessionMu(sessionID)\n	mu.Lock()\n	defer mu.Unlock()\n	defer func() { a.publishCanceledQueueDrops(drops) }()\n")],
 "N6b_clearqueue_cancelrun_drops_under_lock": [sub(AG, LOCKFREE, LOCKED, 2)],
}
res = []
for name, muts in M.items():
    for p in orig: open(p, "w").write(orig[p])
    for path, f in muts:
        mutated = f(open(path).read())
        open(path, "w").write(mutated)
    r = subprocess.run([gosh, "test", "-p", "2", "-count=1", "-run",
                        "TestSendMessage_|TestQueueMarkDrop|TestRecordedPrompt|TestSummarizeLeaves|TestRunEarlyError|TestQueueDropWrites|TestSettleBusy",
                        "./internal/backend/", "./internal/agent/"], cwd=tree, capture_output=True, text=True)
    fails = sorted(set(re.findall(r"--- FAIL: (\S+)", r.stdout)))
    build = "BUILD-FAILED" if "[build failed]" in r.stdout or "[setup failed]" in r.stdout else ""
    res.append("%s rc=%d %s red=%s" % (name, r.returncode, build, ",".join(fails) or "NONE"))
    print(res[-1], flush=True)
for p in orig: open(p, "w").write(orig[p])
open(out, "w").write("\n".join(res) + "\n")
