"""Native application quit/reopen test with isolated deterministic Agent CLIs.
No real model messages or user settings are used. Build e2e binary first.
"""
import base64
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / 'test-results'
OUT.mkdir(exist_ok=True)

def wait_for(fn, timeout=25):
    deadline=time.monotonic()+timeout
    while time.monotonic()<deadline:
        value=fn()
        if value:
            return value
        time.sleep(.1)
    raise AssertionError('Timed out waiting for workspace restart condition')

with tempfile.TemporaryDirectory(prefix='cyrene-restart-') as directory:
    tmp=Path(directory)
    config=tmp/'cyrene-terminal'
    config.mkdir()
    invokes=tmp/'invocations'
    invokes.mkdir()
    agents=tmp/'agents'
    agents.mkdir()
    project=tmp/'project'
    project.mkdir()
    ps=agents/'fake.ps1'
    ps.write_text(r'''
$kind=$args[0]
$rest=@($args | Select-Object -Skip 1)
$restored=$false
$id=''
for($i=0;$i -lt $rest.Count;$i++) {
    if($rest[$i] -eq 'resume' -or $rest[$i] -eq '--resume') { $id=$rest[$i+1]; $restored=$true }
    if($rest[$i] -eq '--session-id') { $id=$rest[$i+1] }
}
if(!$id) { $id=[guid]::NewGuid().ToString() }
$memory=Join-Path $env:CYRENE_RESTART_ROOT ($kind+'-'+$id+'.memory')
if($restored) {
    if(!(Test-Path -LiteralPath $memory)) { throw 'Original conversation missing' }
    $message=[IO.File]::ReadAllText($memory)
} else {
    $message='conversation-memory-'+$id
    [IO.File]::WriteAllText($memory,$message)
}
if($kind -eq 'codex' -and $env:CYRENE_AGENT_SESSION_FILE) {
    [IO.File]::WriteAllText($env:CYRENE_AGENT_SESSION_FILE,$id)
}
$entry=@{kind=$kind;session=$id;restored=$restored;arguments=$rest;directory=(Get-Location).Path;memory=$message;pid=$PID} | ConvertTo-Json -Compress
[IO.File]::WriteAllText((Join-Path $env:CYRENE_RESTART_ROOT ('invocations/'+$kind+'-'+$PID+'.json')),$entry)
Write-Output ('READY '+$kind+' '+$id+' '+$message)
while($true) {Start-Sleep -Seconds 1}
''',encoding='utf-8')
    profiles=[]
    for kind,name in [('codex','Codex'),('claude','Claude')]:
        cmd=agents/(kind+'.cmd')
        cmd.write_text('@echo off\r\npowershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0fake.ps1" '+kind+' %*\r\n',encoding='utf-8')
        profiles.append(dict(id=kind,name=name,command=[str(cmd)]))
    profiles.append(dict(id='shell',name='Terminal',command=['powershell.exe','-NoLogo','-NoProfile']))
    (config/'settings.json').write_text(json.dumps(dict(profiles=profiles)),encoding='utf-8')
    env=dict(os.environ,APPDATA=str(tmp),CODEX_HOME=str(tmp/'codex-home'),CYRENE_RESTART_ROOT=str(tmp),MYGO_AUTOMATION=str(tmp/'automation.sock'),CYRENE_ACCEPTANCE_NO_FOCUS='1',CYRENE_ACCEPTANCE_INSTANCE=str(tmp))
    log=(OUT/'workspace-restart.log').open('w',encoding='utf-8')
    processes=[]
    client=None
    serial=0
    def launch():
        global client
        proc=subprocess.Popen([str(ROOT/'.mygo/cyrene-isolated-smoke.exe')],cwd=ROOT,env=env,stdout=log,stderr=log,creationflags=subprocess.CREATE_NO_WINDOW)
        processes.append(proc)
        client=subprocess.Popen(['go','run','./scripts/automation-client',env['MYGO_AUTOMATION']],cwd=ROOT,stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=log,creationflags=subprocess.CREATE_NO_WINDOW)
        rpc('windows')
        return proc
    def rpc(method,**params):
        global serial
        serial+=1
        client.stdin.write((json.dumps(dict(id=serial,method=method,params=params))+'\n').encode());client.stdin.flush()
        raw=client.stdout.readline()
        if not raw: raise RuntimeError('Automation connection closed')
        response=json.loads(raw)
        if 'error' in response: raise RuntimeError(response['error'])
        return response['result']
    def nodes(): return rpc('tree')['nodes']
    def click(label):
        node=wait_for(lambda:next((n for n in nodes() if n.get('label')==label and 'press' in n.get('actions',[])),None))
        rpc('act',id=node['id'],action='press');time.sleep(.6)
    def key(value): rpc('input',events=[dict(type='key',key=value)])
    def field(label,value):
        node=next(n for n in nodes() if n.get('label')==label and 'setValue' in n.get('actions',[]))
        rpc('act',id=node['id'],action='setValue',text=value)
    def palette(query):
        key('Ctrl+Shift+P');time.sleep(.5)
        rpc('input',events=[dict(type='text',text=query)]);key('Enter');time.sleep(.5)
    def entries():
        result=[]
        for path in invokes.glob('*.json'):
            try: result.append(json.loads(path.read_text(encoding='utf-8-sig')))
            except (ValueError,OSError): pass
        return result
    def quit_app(proc):
        rpc('menu',path=['退出应用'])
        assert proc.wait(timeout=25)==0
        client.stdin.close();client.wait(timeout=5)
    try:
        first=launch()
        for name in ['Codex','Claude']:
            key('Ctrl+Shift+T');time.sleep(.6);click('启动 '+name)
        initial=wait_for(lambda:entries() if len(entries())==2 else None)
        palette('rename tab');field('标签名称','Agent workspace');click('保存')
        palette('rename session');field('会话别名','Claude original');click('保存')
        codex=next(n for n in nodes() if n.get('label')=='内容标签 5')['bounds']
        target=next(n for n in nodes() if n.get('label')=='Terminal' and n.get('key')==7)['bounds']
        x,y=codex['x']+20,codex['y']+16
        dx,dy=target['x']+target['width']-20,target['y']+target['height']/2
        rpc('input',events=[dict(type='pointerDown',x=x,y=y),dict(type='pointerMove',x=x+10,y=y+3),dict(type='pointerMove',x=dx,y=dy)])
        time.sleep(.2);rpc('input',events=[dict(type='pointerUp',x=dx,y=dy)])
        wait_for(lambda:len([n for n in nodes() if n.get('label')=='Terminal'])==2)
        quit_app(first)
        saved=json.loads((config/'workspace.json').read_text(encoding='utf-8'))
        assert len(saved['Sessions'])==3 and len(saved['Screens'])==2
        original={p['kind']:p for p in initial}
        for pane in saved['Panes']:
            if pane['Profile']['id'] in original:
                assert pane['AgentSession']==original[pane['Profile']['id']]['session']
        second=launch()
        resumed=wait_for(lambda:[p for p in entries() if p['restored']] if len([p for p in entries() if p['restored']])==2 else None)
        for entry in resumed:
            old=original[entry['kind']]
            assert entry['session']==old['session'] and entry['memory']==old['memory']
            assert entry['pid']!=old['pid']
        wait_for(lambda:len([n for n in nodes() if n.get('label')=='Terminal'])==2)
        assert rpc('windows')['windows'][0]['title']=='Agent workspace · 2'
        image=rpc('capture')
        (OUT/'workspace-restarted.png').write_bytes(base64.b64decode(image['png']))
        quit_app(second)
        again=json.loads((config/'workspace.json').read_text(encoding='utf-8'))
        assert saved==again
        result=dict(native_quit_and_restart=True,exact_codex_and_claude_resume=True,original_conversation_memory_retained=True,aliases_groups_layout_focus_preserved=True,fresh_pids_same_sessions=True,real_model_requests=False)
        (OUT/'workspace-restart.json').write_text(json.dumps(result,indent=2),encoding='utf-8')
        print(json.dumps(result,indent=2))
    except Exception:
        (OUT/'workspace-restart-debug.json').write_text(json.dumps(dict(entries=entries(),workspace=json.loads((config/'workspace.json').read_text()),tree=nodes()),indent=2,ensure_ascii=False),encoding='utf-8')
        image=rpc('capture')
        (OUT/'workspace-restart-failure.png').write_bytes(base64.b64decode(image['png']))
        raise
    finally:
        if client and client.poll() is None:
            client.stdin.close()
            try: client.wait(timeout=5)
            except subprocess.TimeoutExpired: client.kill()
        for proc in processes:
            if proc.poll() is None:
                subprocess.run(['taskkill','/PID',str(proc.pid),'/T','/F'],capture_output=True)
        log.close()
