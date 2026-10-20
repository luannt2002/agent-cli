import sys
def load(p):
    d = {}
    for l in open(p):
        a, pkg, t = l.rstrip("\n").split("\t")
        d[(pkg, t)] = a
    return d
b, f = load(sys.argv[1]), load(sys.argv[2])
print("names %s=%d %s=%d" % (sys.argv[1].split("/")[-1], len(b), sys.argv[2].split("/")[-1], len(f)))
for k in sorted(set(b) | set(f)):
    if b.get(k) != f.get(k):
        print("%-6s -> %-6s %s %s" % (b.get(k, "ABSENT"), f.get(k, "ABSENT"), k[0].split("crush/")[-1], k[1]))
