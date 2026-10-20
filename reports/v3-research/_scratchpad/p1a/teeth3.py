# Kiểm răng vòng 3: đột biến cho từng sửa mới (+ vài sửa cũ), chạy toàn bộ test P1-a, ghi test đỏ.
# Dùng: python3 -I teeth3.py <teeth-tree> <go.sh> <out.txt> [--check]
import subprocess, sys, re, os
tree, gosh, out = sys.argv[1:4]
check_only = "--check" in sys.argv
AG = os.path.join(tree, "internal/agent/agent.go")
BE = os.path.join(tree, "internal/backend/agent.go")
SF = os.path.join(tree, "internal/agent/settle_fork.go")
MS = os.path.join(tree, "internal/message/message.go")
ST = os.path.join(tree, "internal/db/stats.sql.go")
CF = os.path.join(tree, "internal/config/load.go")
orig = {p: open(p).read() for p in (AG, BE, SF, MS, ST, CF)}
def sub(path, old, new, count=1):
    def f(s):
        assert s.count(old) == count, (path, old[:70], s.count(old))
        return s.replace(old, new)
    return (path, f)
M = {
 "M1_no_record_before_ack": [sub(BE, "if err := accept.Record(ws.ctx, call); err != nil {", "if err := error(nil); err != nil {")],
 "M2_drop_without_closing_trace": [sub(AG, "if err := a.persistCanceledTurn(ctx, d, true); err != nil {", "if err := error(nil); err != nil {")],
 "N1_run_history_keeps_unsent": [sub(AG, "	msgs = slices.DeleteFunc(msgs, unsent)\n", "	_ = unsent\n")],
 "N5_settle_ignores_pending": [sub(SF, "return c.IsSessionBusy(sessionID) || c.HasPendingPrompts(sessionID)", "return c.IsSessionBusy(sessionID)")],
 "X1_filter_by_position_not_link": [sub(AG, """	for _, m := range msgs {
		if message.IsPromptPlaceholder(m) {
			linked[m.Model] = true
		}
	}""", """	for i := 1; i < len(msgs); i++ {
		if message.IsPromptPlaceholder(msgs[i]) && msgs[i-1].Role == message.User {
			linked[msgs[i-1].ID] = true
		}
	}""")],
 "X2_batch_without_transaction": [sub(MS, """	tx, err := queries.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck
	qtx := queries.WithTx(tx)
""", "	qtx := queries\n"), sub(MS, """	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}
""", "")],
 "X3_failed_move_drops_old_pair": [sub(AG, """		var moved *recordedPrompt
		if _, moved, err = a.moveRecordedPrompt(ctx, call); err == nil {
			call.recorded = moved
		}""", "		_, call.recorded, err = a.moveRecordedPrompt(ctx, call)")],
 "X4_stats_count_placeholders": [sub(ST, "  AND COALESCE(provider, '') != 'agent-cli-pending' -- message.PendingReplyProvider: prompt chưa chạy\n", "", 2)],
 "X5_reserved_provider_allowed": [sub(CF, "		if id == message.PendingReplyProvider {", "		if id == message.PendingReplyProvider && false {")],
 "X6_reply_then_delete_separately": [sub(AG, "if created, err = a.messages.WriteBatch(callContext, call.SessionID, []message.CreateMessageParams{reply}, replaced); err == nil {", "if created, err = a.messages.WriteBatch(callContext, call.SessionID, []message.CreateMessageParams{reply}, nil); err == nil {\n\t\t\t\t\t_, _ = a.messages.WriteBatch(callContext, call.SessionID, nil, replaced)")],
 "X7_move_then_delete_separately": [sub(AG, "	}}, deleteIDs)\n", "	}}, nil)\n\tif err == nil {\n\t\t_, _ = a.messages.WriteBatch(ctx, call.SessionID, nil, deleteIDs)\n\t}\n")],
}
res = []
for name, muts in M.items():
    for p in orig: open(p, "w").write(orig[p])
    for path, f in muts:
        mutated = f(open(path).read())
        open(path, "w").write(mutated)
    if check_only:
        res.append(name + " applies"); continue
    r = subprocess.run([gosh, "test", "-p", "2", "-count=1", "-run",
                        "TestSendMessage_|TestQueueMarkDrop|TestRecordedPrompt|TestSummarizeLeaves|TestRunEarlyError|TestQueueDropWrites|TestSettleBusy|TestUnsentPrompt|TestStatsSkip|TestWriteBatch|TestConfig_configureProvidersRejectsReservedID",
                        "./internal/backend/", "./internal/agent/", "./internal/message/", "./internal/config/"], cwd=tree, capture_output=True, text=True)
    fails = sorted(set(re.findall(r"--- FAIL: (\S+)", r.stdout)))
    build = "BUILD-FAILED" if "[build failed]" in r.stdout or "[setup failed]" in r.stdout else ""
    res.append("%s rc=%d %s red=%s" % (name, r.returncode, build, ",".join(fails) or "NONE"))
    print(res[-1], flush=True)
for p in orig: open(p, "w").write(orig[p])
open(out, "w").write("\n".join(res) + "\n")
