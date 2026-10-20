#!/usr/bin/env python3
"""Kiểm răng P0-2b: mỗi mutant gỡ một phần bản sửa, build test binary của gói, chạy đúng test canh nó,
rồi hoàn nguyên và kiểm sha256 của tệp. Kết quả mong muốn: test ĐỎ dưới mutant.
Dùng: python3 -I mut.py <thư mục kết quả mới>"""
import hashlib, os, subprocess, sys

W = "/home/luannt/application/opensource/agent-cli-p02b"
S = os.path.dirname(os.path.abspath(__file__))
OUT = sys.argv[1]
os.makedirs(OUT)

M = [
    ("M1 homeEnv giữ launcher của .env", "internal/cmd/home.go",
     "\tenv := slices.DeleteFunc(execpolicy.ChildEnv(os.Environ()), func(kv string) bool {\n\t\treturn strings.HasPrefix(kv, launcherEnv+\"=\")\n\t})\n",
     "\tenv := slices.DeleteFunc(execpolicy.ChildEnv(os.Environ()), func(kv string) bool {\n\t\treturn false\n\t})\n",
     "./internal/cmd/", "internal/cmd", "^TestChildEnvsIgnoreProjectDotenv$"),
    ("M1b homeEnv bỏ luôn launcher gốc", "internal/cmd/home.go",
     "\tif l := execpolicy.OriginalEnv(launcherEnv); l != \"\" {",
     "\tif l := execpolicy.OriginalEnv(launcherEnv); false && l != \"\" {",
     "./internal/cmd/", "internal/cmd", "^TestChildEnvsIgnoreProjectDotenv$"),
    ("M2 !cmd local dùng os.Environ()", "internal/workspace/app_workspace.go",
     "\t\tEnv:       execpolicy.ChildEnv(os.Environ()),\n", "\t\tEnv:       append(execpolicy.ChildEnv(nil), os.Environ()...),\n",
     "./internal/workspace/", "internal/workspace", "^TestShellCommandIgnoresProjectDotenvTrustedKeys$"),
    ("M3 !cmd server bỏ ChildEnv", "internal/backend/agent.go",
     "\t\tEnv:       execpolicy.ChildEnv(append(os.Environ(), ws.Env...)),\n", "\t\tEnv:       append(execpolicy.ChildEnv(nil), append(os.Environ(), ws.Env...)...),\n",
     "./internal/backend/", "internal/backend", "^TestRunShellCommandIgnoresProjectDotenvTrustedKeys$"),
    ("M4 CLI con dùng env sau .env", "internal/agent/tools/provider/exec.go",
     "\tcmd.Env = childEnv(execpolicy.OriginalEnviron())\n", "\tcmd.Env = childEnv(append(execpolicy.OriginalEnviron()[:0:0], cmd.Environ()...))\n",
     "./internal/agent/tools/provider/", "internal/agent/tools/provider", "^TestRunCommandIgnoresProjectDotenv$"),
    ("M6 ensureServer bỏ nhánh errStaleServerBusy", "internal/cmd/root.go",
     "\t\t\tif errors.Is(err, errStaleServerBusy) {\n\t\t\t\treturn err\n\t\t\t}\n", "",
     "./internal/cmd/", "internal/cmd", "^TestEnsureServer_RefusesModelCommandOnBusyStaleServer$"),
    ("M7 OriginalEnviron duyệt env gốc thay vì env hiện tại", "internal/execpolicy/policy.go",
     "\tfor _, kv := range os.Environ() {\n\t\tk, _, _ := strings.Cut(kv, \"=\")\n\t\tif v, ok := original[k]; ok {",
     "\tfor k := range original {\n\t\tif v, ok := original[k]; ok {",
     "./internal/execpolicy/", "internal/execpolicy", "^TestOriginalEnvironDropsLaterChanges$"),
    ("M8 ChildEnv bỏ trạng thái không người trực hiện hành", "internal/execpolicy/policy.go",
     "\t\tif unattendedOn.Load() {\n\t\t\tswitch k {", "\t\tif false && unattendedOn.Load() {\n\t\t\tswitch k {",
     "./internal/cmd/", "internal/cmd", "^TestVerifyCarriesTheCurrentUnattendedState$"),
    ("M8u (như M8, test gói execpolicy)", "internal/execpolicy/policy.go",
     "\t\tif unattendedOn.Load() {\n\t\t\tswitch k {", "\t\tif false && unattendedOn.Load() {\n\t\t\tswitch k {",
     "./internal/execpolicy/", "internal/execpolicy", "^TestChildEnvCarriesTheCurrentUnattendedState$"),
    ("M9 ChildEnv tin giá trị công tắc trong env (kể cả .env)", "internal/execpolicy/policy.go",
     "\t\tif unattendedOn.Load() {\n\t\t\tswitch k {", "\t\tif ev, has := os.LookupEnv(k); has && (k == UnattendedEnv || k == UnattendedCommitEnv) {\n\t\t\tv, ok = ev, true\n\t\t} else if unattendedOn.Load() {\n\t\t\tswitch k {",
     "./internal/cmd/", "internal/cmd", "^TestVerifyCarriesTheCurrentUnattendedState$"),
    ("M10 commit luôn 1 khi không người trực", "internal/execpolicy/policy.go",
     "\t\t\t\tif unattendedCommit.Load() {", "\t\t\t\tif true || unattendedCommit.Load() {",
     "./internal/cmd/", "internal/cmd", "^TestVerifyCarriesTheCurrentUnattendedState$"),

    ("M11 homeEnv về ChildEnv(os.Environ())", "internal/cmd/home.go",
     "func homeEnv() []string { return execpolicy.OriginalEnviron() }", "func homeEnv() []string { return execpolicy.ChildEnv(os.Environ()) }",
     "./internal/cmd/", "internal/cmd", "^TestChildEnvsIgnoreProjectDotenv$"),
    ("M12 serverEnv về ChildEnv(StripEnv(os.Environ()))", "internal/cmd/root.go",
     "func serverEnv() []string { return herdr.StripEnv(execpolicy.OriginalEnviron()) }", "func serverEnv() []string { return execpolicy.ChildEnv(herdr.StripEnv(os.Environ())) }",
     "./internal/cmd/", "internal/cmd", "^TestChildEnvsIgnoreProjectDotenv$"),
]


def sha(p):
    return hashlib.sha256(open(p, "rb").read()).hexdigest()


log = open(os.path.join(OUT, "mutants.txt"), "w")
for name, rel, old, new, pkg, pdir, test in M:
    if len(sys.argv) > 2 and name.split()[0] not in sys.argv[2].split(","):
        continue
    path = os.path.join(W, rel)
    orig = open(path).read()
    before = sha(path)
    assert orig.count(old) == 1, (name, "pattern not unique")
    try:
        open(path, "w").write(orig.replace(old, new))
        tb = os.path.join(OUT, "t.test")
        b = subprocess.run([os.path.join(S, "gp.sh"), "go", "test", "-c", "-p", "2", "-o", tb, pkg],
                           capture_output=True, text=True)
        if b.returncode != 0:
            res, detail = "BUILD-FAIL", b.stderr.strip().splitlines()[-1:]
        else:
            h = os.path.join(OUT, "home-" + name.split()[0])
            os.makedirs(h)
            env = {k: v for k, v in os.environ.items() if not k.startswith("CRUSH_")}
            env.update(HOME=h, XDG_CACHE_HOME=h + "/.cache", XDG_CONFIG_HOME=h + "/.config",
                       XDG_DATA_HOME=h + "/.local/share", GOMAXPROCS="2")
            r = subprocess.run([tb, "-test.count=1", "-test.run", test], cwd=os.path.join(W, pdir),
                               capture_output=True, text=True, env=env, timeout=300)
            res = "RED" if r.returncode != 0 else "GREEN(mutant survived)"
            detail = [l.strip() for l in r.stdout.splitlines() if l.strip().startswith(("--- FAIL", "Error:", "Messages:"))][:4]
            os.remove(tb)
    finally:
        open(path, "w").write(orig)
    ok = sha(path) == before
    line = "%s | %s | %s | restored sha ok=%s | %s" % (name, test, res, ok, " / ".join(detail))
    print(line)
    log.write(line + "\n")
    log.flush()
    assert ok
