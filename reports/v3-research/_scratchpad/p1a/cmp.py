import json,sys
def load(p):
    res={}; pk={}
    for l in open(p):
        try: e=json.loads(l)
        except: continue
        if e.get("Test") and e.get("Action") in ("pass","fail","skip"):
            res[(e["Package"],e["Test"])]=e["Action"]
        elif not e.get("Test") and e.get("Action") in ("pass","fail","skip"):
            pk[e["Package"]]=e["Action"]
    return res,pk
a,pa=load(sys.argv[1]); b,pb=load(sys.argv[2])
from collections import Counter
print("base",dict(Counter(a.values())),"names",len(a))
print("fix ",dict(Counter(b.values())),"names",len(b))
print("missing_in_fix",sorted(set(a)-set(b)))
new=sorted(set(b)-set(a)); print("new",len(new),"new_not_pass",[k for k in new if b[k]!="pass"])
print("changed",[(k,a[k],b[k]) for k in sorted(set(a)&set(b)) if a[k]!=b[k]])
print("fail_top_base",sorted(k[1] for k,v in a.items() if v=="fail" and "/" not in k[1]))
print("fail_top_fix",sorted(k[1] for k,v in b.items() if v=="fail" and "/" not in k[1]))
