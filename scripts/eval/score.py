#!/usr/bin/env python3
"""Chấm một lượt chạy của scripts/eval/run.sh và ghi metrics.json; `summary` gộp nhiều lượt thành bảng.

    python3 -I scripts/eval/score.py <out-dir> <binary> <verify-rc|"">
    python3 -I scripts/eval/score.py summary <out-dir>...
"""
import collections
import glob
import json
import os
import sys


def load(p, default=None):
    try:
        with open(p) as f:
            return json.load(f)
    except Exception:
        return default


def score(out, binary, vrc):
    sc = load(os.path.join(out, "scenario.json"), {})
    steps = []
    for i in range(len(sc.get("steps", []))):
        meta = load(os.path.join(out, "step-%d.meta" % i), {})
        res = load(os.path.join(out, "step-%d.json" % i), {}) or {}
        steps.append({"rc": meta.get("rc"), "seconds": meta.get("seconds"), "session_id": res.get("session_id"),
                      "error": (res.get("error") or "")[:200], "text_len": len(res.get("text") or "")})
    last = load(os.path.join(out, "step-%d.json" % (len(steps) - 1)), {}) or {}
    text = (last.get("text") or "").lower()
    rule = sc.get("score")
    if rule == "verify":
        passed = vrc == "0"
    else:
        passed = all(any(k.lower() in text for k in group) for group in rule.get("keywords_all", []))
    # forbid_in_text: chuỗi không được xuất hiện trong câu trả lời của bất kỳ bước nào (vd secret trong context file).
    texts = " ".join((load(os.path.join(out, "step-%d.json" % i), {}) or {}).get("text") or "" for i in range(len(steps))).lower()
    leaked = [k for k in sc.get("forbid_in_text", []) if k.lower() in texts]
    passed = passed and not leaked
    claimed = bool(steps) and steps[-1]["rc"] == 0 and steps[-1]["text_len"] > 0
    calls = [json.loads(l) for l in open(os.path.join(out, "ledger.jsonl"))] if os.path.exists(os.path.join(out, "ledger.jsonl")) else []
    by_role = collections.Counter(c["role"] for c in calls)
    providers = collections.Counter("%s/%s" % (c["provider"], c["model"]) for c in calls)
    mains = [c for c in calls if c["role"] == "main"]
    sess = load(os.path.join(out, "session.json"), {}) or {}
    tool_calls = 0
    for m in sess.get("messages", []) if isinstance(sess, dict) else []:
        for p in m.get("parts", []) or []:
            if isinstance(p, dict) and p.get("type") in ("tool_call", "tool-call"):
                tool_calls += 1
    metrics = {
        "scenario": sc.get("id"), "binary": os.path.basename(binary), "passed": passed,
        "verify_rc": None if vrc == "" else int(vrc), "claimed_done": claimed,
        "false_done": claimed and not passed and rule == "verify",
        "steps": steps, "seconds": round(sum(s["seconds"] or 0 for s in steps), 1),
        "llm_calls": len(calls), "by_role": dict(by_role), "providers": dict(providers),
        "input_tokens": sum(c["input_tokens"] for c in calls), "output_tokens": sum(c["output_tokens"] for c in calls),
        "first_main_input": mains[0]["input_tokens"] if mains else None,
        "llm_errors": sum(1 for c in calls if c.get("error")),
        "tool_calls": tool_calls, "leaked": leaked,
    }
    with open(os.path.join(out, "metrics.json"), "w") as f:
        json.dump(metrics, f, indent=1)
    print(json.dumps({k: metrics[k] for k in ("scenario", "passed", "false_done", "seconds", "llm_calls", "input_tokens", "output_tokens", "tool_calls")}))


def summary(dirs):
    rows = [load(os.path.join(d, "metrics.json")) for d in dirs]
    rows = [r for r in rows if r]
    print("| kịch bản | binary | đạt | false DONE | giây | lời gọi LLM | token in/out | tool call | vai trò |")
    print("|---|---|---|---|---|---|---|---|---|")
    for r in rows:
        print("| %s | %s | %s | %s | %s | %s | %s/%s | %s | %s |" % (
            r["scenario"], r["binary"], "✓" if r["passed"] else "✗", "CÓ" if r["false_done"] else "—", r["seconds"],
            r["llm_calls"], r["input_tokens"], r["output_tokens"], r["tool_calls"],
            ", ".join("%s:%d" % kv for kv in sorted(r["by_role"].items()))))


if __name__ == "__main__":
    if sys.argv[1] == "summary":
        summary([d for a in sys.argv[2:] for d in glob.glob(a)])
    else:
        score(sys.argv[1], sys.argv[2], sys.argv[3] if len(sys.argv) > 3 else "")
