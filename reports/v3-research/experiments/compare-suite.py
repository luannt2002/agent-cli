#!/usr/bin/env python3
"""So hai lần chạy run-suite.sh theo TÊN kịch bản: chỉ tính kịch bản có ở cả hai.
   python3 compare-suite.py <root-A> <root-B>"""
import json, os, sys

def load(root):
    out = {}
    for d in sorted(os.listdir(root)):
        p = os.path.join(root, d, "metrics.json")
        if os.path.isfile(p):
            m = json.load(open(p))
            out[m["scenario"]] = m
    return out

a, b = load(sys.argv[1]), load(sys.argv[2])
common = sorted(set(a) & set(b))
print("chung:", len(common), "| chỉ A:", sorted(set(a) - set(b)), "| chỉ B:", sorted(set(b) - set(a)))
keys = ["passed", "false_done", "seconds", "llm_calls", "input_tokens", "output_tokens", "tool_calls", "llm_errors"]
print("| kịch bản | " + " | ".join("%s A→B" % k for k in keys) + " |")
print("|---" * (len(keys) + 1) + "|")
tot = {k: [0, 0] for k in keys}
for s in common:
    row = []
    for k in keys:
        va, vb = a[s].get(k), b[s].get(k)
        row.append("%s→%s" % (va, vb))
        tot[k][0] += int(va or 0); tot[k][1] += int(vb or 0)
    print("| %s | %s |" % (s, " | ".join(row)))
print("| **tổng** | " + " | ".join("%s→%s" % (tot[k][0], tot[k][1]) for k in keys) + " |")
leak = [(s, a[s].get("leaked"), b[s].get("leaked")) for s in common if a[s].get("leaked") or b[s].get("leaked")]
print("lọt provider:", leak or "0")
