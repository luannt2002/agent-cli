import subprocess,os,json,time,concurrent.futures,hashlib
from pathlib import Path
R=Path('/home/luannt/.cache/agent-cli-takeover-20261019')
W=Path('/home/luannt/.codex/worktrees/audit-r8-takeover/agent-cli')
BIN=R/'agent-cli'
env=dict(os.environ,GOMAXPROCS='2')
def stage(name,args,extra=None,limit=1200):
 e=dict(env,**(extra or {}));start=time.monotonic()
 (R/(name+'.state.json')).write_text(json.dumps({'state':'running','command':args,'start_utc':time.strftime('%Y-%m-%dT%H:%M:%SZ',time.gmtime())})+'\n')
 with (R/(name+'.log')).open('w') as log:
  try:rc=subprocess.run(args,cwd=W,env=e,stdout=log,stderr=subprocess.STDOUT,timeout=limit).returncode
  except subprocess.TimeoutExpired:rc=124
 result={'case':name,'exit':rc,'seconds':round(time.monotonic()-start,2),'command':args,'state':'finished'}
 (R/(name+'.state.json')).write_text(json.dumps(result)+'\n')
 print(json.dumps(result),flush=True)
 return rc
if stage('build',['go','build','-p','2','-o',str(BIN),'.'],{'CGO_ENABLED':'0','GOEXPERIMENT':'greenteagc'})!=0:raise SystemExit(1)
(R/'binary.sha256').write_text(hashlib.sha256(BIN.read_bytes()).hexdigest()+'  agent-cli\n')
def golang():
 stage('suite',['go','test','-p','2','./...','-count=1','-json'])
 stage('vet',['go','vet','-p','2','./...'])
 stage('race',['go','test','-race','-p','2','-count=1','./internal/backend/...','./internal/agent/...','./internal/cmd/...','./internal/ui/...','./internal/workspace/...','./internal/projects/...','./internal/permission/...','./internal/server/...'])
def integrations():
 extra={'AGENT_CLI_BIN':str(BIN)}
 for case in ['run-json','loop','routine']:
  stage(case,['bash','scripts/test-agent-cli-'+case+'.sh'],extra,600)
 for case in ['faults','qwen-only','workspaces']:
  stage(case,['bash','scripts/eval/'+case+'.sh',str(R/(case+'-data')),str(BIN)],extra,600)
with concurrent.futures.ThreadPoolExecutor(max_workers=2) as ex:
 jobs=[ex.submit(golang),ex.submit(integrations)]
 for f in jobs:f.result()
