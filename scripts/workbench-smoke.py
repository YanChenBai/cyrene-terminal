"""Native no-sidebar workbench, inline children and embedded localhost browser.

Uses isolated settings, local HTTP fixtures and owned processes only.
Build .mygo/cyrene-isolated-smoke.exe with -tags e2e before running.
"""
import base64
import http.server
import json
import os
from pathlib import Path
import subprocess
import tempfile
import threading
import time

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / 'test-results'
OUT.mkdir(exist_ok=True)
hits = []


class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        hits.append(self.path)
        name = 'B' if self.path.startswith('/b') else 'A'
        body = f'''<!doctype html><html><head><title>Preview {name}</title>
<style>html{{background:#dcd3ee}}body{{background:#1d1b23;color:#e8e5ef;font:18px system-ui;margin:40px}}
a{{color:#bfb3e8}}button{{padding:10px 18px;background:#3c354d;color:white;border:0;border-radius:8px}}</style></head>
<body><small>LOCAL DEVELOPMENT</small><h1>Preview {name}</h1>
<p>Terminal and browser, in one workspace.</p><a href="/b">Open page B</a>
<p><button onclick="this.textContent='Clicked'">Try interaction</button></p>
<script>fetch('/ready?{name}');</script></body></html>'''.encode()
        if self.path.startswith('/ready'): body = b'OK'
        self.send_response(200)
        self.send_header('Content-Type', 'text/html; charset=utf-8')
        self.send_header('Content-Length', str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *args): pass


def wait_for(fn, timeout=25):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        value = fn()
        if value: return value
        time.sleep(.1)
    raise AssertionError('Workbench condition timed out')


server = http.server.ThreadingHTTPServer(('127.0.0.1', 0), Handler)
threading.Thread(target=server.serve_forever, daemon=True).start()
url = f'http://127.0.0.1:{server.server_port}'
with tempfile.TemporaryDirectory(prefix='cyrene-workbench-') as directory:
    tmp = Path(directory)
    config = tmp / 'cyrene-terminal'
    config.mkdir()
    (config / 'settings.json').write_text(json.dumps(dict(profiles=[dict(id='shell',name='Terminal',command=['powershell.exe','-NoLogo','-NoProfile'])])), encoding='utf-8')
    env = dict(os.environ, APPDATA=str(tmp), MYGO_AUTOMATION=str(tmp/'automation.sock'), CYRENE_ACCEPTANCE_NO_FOCUS='1', CYRENE_ACCEPTANCE_BROWSER_CAPTURE=str(tmp/'web-capture'), CYRENE_ACCEPTANCE_INSTANCE=str(tmp))
    log = (OUT/'workbench.log').open('w',encoding='utf-8')
    procs, serial, client = [], 0, None

    def launch():
        global client
        proc = subprocess.Popen([str(ROOT/'.mygo/cyrene-isolated-smoke.exe')], cwd=ROOT, env=env, stdout=log, stderr=log, creationflags=subprocess.CREATE_NO_WINDOW)
        procs.append(proc)
        client = subprocess.Popen(['go','run','./scripts/automation-client',env['MYGO_AUTOMATION']],cwd=ROOT,stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=log,creationflags=subprocess.CREATE_NO_WINDOW)
        rpc('windows')
        wait_for(lambda: any(n.get('label')=='Terminal' for n in nodes()))
        return proc

    def rpc(method, **params):
        global serial
        serial += 1
        client.stdin.write((json.dumps(dict(id=serial,method=method,params=params))+'\n').encode());client.stdin.flush()
        raw = client.stdout.readline()
        if not raw: raise RuntimeError('Automation disconnected')
        response = json.loads(raw)
        if 'error' in response: raise RuntimeError(response['error'])
        return response['result']

    def nodes(): return rpc('tree')['nodes']
    def find(label): return next((n for n in nodes() if n.get('label')==label),None)
    def click(label):
        node = wait_for(lambda: next((n for n in nodes() if n.get('label')==label and 'press' in n.get('actions',[])),None))
        rpc('act',id=node['id'],action='press');time.sleep(.65)
    def key(value): rpc('input',events=[dict(type='key',key=value)])
    def field(label,value):
        node=wait_for(lambda: next((n for n in nodes() if n.get('label')==label and 'setValue' in n.get('actions',[])),None))
        rpc('act',id=node['id'],action='setValue',text=value)
    def palette(query):
        key('Ctrl+Shift+P');time.sleep(.65);field('命令搜索',query);key('Enter');time.sleep(.5)
    def capture(name):
        (OUT/name).write_bytes(base64.b64decode(rpc('capture')['png']))
    def quit_app(proc):
        rpc('menu',path=['退出应用']);assert proc.wait(timeout=25)==0
        client.stdin.close();client.wait(timeout=5)
    def address(value):
        node=find('当前网页地址');rpc('act',id=node['id'],action='focus')
        field('当前网页地址',value);key('Enter')
    def drag(source, destination):
        x,y=source;dx,dy=destination
        rpc('input',events=[dict(type='pointerDown',x=x,y=y),dict(type='pointerMove',x=x+12,y=y+3),dict(type='pointerMove',x=dx,y=dy)])
        time.sleep(.25);rpc('input',events=[dict(type='pointerUp',x=dx,y=dy)]);time.sleep(.6)

    def native_corner_capture(proc):
        if os.environ.get('CYRENE_CAPTURE_NATIVE')!='1': return
        import ctypes
        from ctypes import wintypes
        from PIL import ImageGrab
        user=ctypes.windll.user32
        user.SetProcessDpiAwarenessContext(ctypes.c_void_p(-4))
        handles=[]
        callback=ctypes.WINFUNCTYPE(wintypes.BOOL,wintypes.HWND,wintypes.LPARAM)
        def enumerate_window(hwnd,_):
            pid=wintypes.DWORD();user.GetWindowThreadProcessId(hwnd,ctypes.byref(pid))
            if pid.value==proc.pid and user.IsWindowVisible(hwnd): handles.append(hwnd)
            return True
        user.EnumWindows(callback(enumerate_window),0)
        hwnd=handles[0]
        old=wintypes.RECT();user.GetWindowRect(hwnd,ctypes.byref(old))
        user.SetWindowPos.argtypes=[wintypes.HWND,wintypes.HWND,ctypes.c_int,ctypes.c_int,ctypes.c_int,ctypes.c_int,ctypes.c_uint]
        try:
            user.SetWindowPos(hwnd,wintypes.HWND(-1),48,48,1440,900,0x10|0x40)
            time.sleep(1.2)
            rect=wintypes.RECT();user.GetWindowRect(hwnd,ctypes.byref(rect))
            image=ImageGrab.grab(bbox=(rect.left,rect.top,rect.right,rect.bottom),all_screens=True)
            image.save(OUT/'workbench-native-corners.png')
            origin=wintypes.POINT();user.ClientToScreen(hwnd,ctypes.byref(origin))
            scale=user.GetDpiForWindow(hwnd)/96
            box=find('网页圆角视口')['bounds']
            x=round(origin.x-rect.left+box['x']*scale)
            y=round(origin.y-rect.top+box['y']*scale)
            corner=image.getpixel((x+1,y+1))[:3]
            backing=image.getpixel((x-2,y+1))[:3]
            inside=image.getpixel((x+round(20*scale),y+round(20*scale)))[:3]
            assert max(abs(a-b) for a,b in zip(corner,backing))<20, (corner,backing,inside)
            assert max(abs(a-b) for a,b in zip(inside,(220,211,238)))<8, inside
            (OUT/'native-corners.json').write_text(json.dumps(dict(corner=corner,pane=backing,page=inside)),encoding='utf8')
            before_overlay=len(hits)
            click('新建终端')
            wait_for(lambda:not any(n.get('webView') for n in nodes()))
            time.sleep(.7)
            ImageGrab.grab(bbox=(rect.left,rect.top,rect.right,rect.bottom),all_screens=True).save(OUT/'workbench-native-launcher.png')
            key('Escape');time.sleep(.6)
            wait_for(lambda:any(n.get('webView') for n in nodes()))
            click('外观设置')
            wait_for(lambda:not any(n.get('webView') for n in nodes()))
            time.sleep(.8)
            ImageGrab.grab(bbox=(rect.left,rect.top,rect.right,rect.bottom),all_screens=True).save(OUT/'workbench-native-modal.png')
            key('Escape');time.sleep(.6)
            wait_for(lambda:any(n.get('webView') for n in nodes()))
            assert len(hits)==before_overlay, 'overlay snapshots reloaded the live browser'
        finally:
            user.SetWindowPos(hwnd,wintypes.HWND(-2),old.left,old.top,old.right-old.left,old.bottom-old.top,0x10)
            time.sleep(.4)

    report = {}
    try:
        first=launch()
        assert not any(find(label) for label in ['切换侧栏','切换右侧栏','刷新文件树'])
        assert not find('新建终端'), 'standalone tab exposed local +'
        palette('rename tab');field('标签名称','Development');click('保存')
        palette('split right');palette_open=find('命令搜索');assert palette_open
        field('命令搜索','new web');key('Enter');time.sleep(.5)
        wait_for(lambda:find('当前网页地址'))
        assert not find('网页地址')
        address(url+'/a')
        wait_for(lambda:'/ready?A' in hits)
        wait_for(lambda:any(n.get('webView') for n in nodes()))
        wait_for(lambda:any(n.get('description')=='Preview A' for n in nodes()))
        children=[n for n in nodes() if n.get('label','').startswith('子标签 ')]
        assert len(children)==2
        web_child=next(n for n in children if n.get('label')=='子标签 4')
        shell_child=next(n for n in children if n.get('label')=='子标签 1')
        # Reorder existing children without reloading the browser.
        source=web_child['bounds'];target=shell_child['bounds'];before_hits=len(hits)
        drag((source['x']+8,source['y']+10),(target['x']+3,target['y']+10))
        assert find('子标签 4')['bounds']['x']<find('子标签 1')['bounds']['x']
        assert len(hits)==before_hits
        click(shell_child['label']);click(web_child['label'])
        address(url+'/b');wait_for(lambda:'/ready?B' in hits)
        wait_for(lambda:any(n.get('description')=='Preview B' for n in nodes()))
        click('网页后退');wait_for(lambda:find('当前网页地址')['value']==url+'/a')
        click('网页前进');wait_for(lambda:find('当前网页地址')['value']==url+'/b')
        count=hits.count('/ready?B');click('刷新网页');wait_for(lambda:hits.count('/ready?B')>count)
        count=len(hits)
        key('Ctrl+Shift+T');time.sleep(.5);click('启动 Terminal')
        assert not any(n.get('label','').startswith('子标签 ') for n in nodes())
        other_screens={n['label'] for n in nodes() if n.get('label','').startswith('内容标签 ')}
        click('内容标签 2')
        assert len([n for n in nodes() if n.get('label','').startswith('子标签 ')])==2
        assert len(hits)==count, 'switching workspace reloaded browser'
        capture('workbench-browser.png')
        wait_for(lambda:(tmp/'web-capture'/'pane-4.png').exists())
        (OUT/'workbench-browser-page.png').write_bytes((tmp/'web-capture'/'pane-4.png').read_bytes())
        # Drag a browser child out to the strip, then dock the same live browser.
        child=find('子标签 4')['bounds']
        drag((child['x']+12,child['y']+12),(900,16))
        assert len([n for n in nodes() if n.get('label','').startswith('内容标签 ')])==3
        assert len(hits)==count
        childscreen=next(n for n in nodes() if n.get('label','').startswith('内容标签 ') and n.get('label') not in other_screens)
        click('内容标签 2')
        term=next(n for n in nodes() if n.get('label')=='Terminal')['bounds']
        source=find(childscreen['label'])['bounds']
        drag((source['x']+15,16),(term['x']+term['width']-20,term['y']+term['height']/2))
        assert len([n for n in nodes() if n.get('label','').startswith('子标签 ')])==2
        assert len(hits)==count
        capture('workbench-docked.png')
        native_corner_capture(first)
        quit_app(first)
        saved=json.loads((config/'workspace.json').read_text(encoding='utf-8'))
        assert len(saved['Screens'])==2
        web=next(p for p in saved['Panes'] if p.get('BrowserURL'))
        assert web['BrowserURL']==url+'/b'
        second=launch()
        wait_for(lambda:hits.count('/ready?B')>count or any(n.get('description')=='Preview B' for n in nodes()))
        wait_for(lambda:any(n.get('webView') for n in nodes()))
        assert rpc('windows')['windows'][0]['title']=='Development · 2'
        capture('workbench-restarted.png')
        click('子标签 4')
        click('关闭窗格 4')
        assert not find('当前网页地址') and not find('终止进程')
        assert find('Terminal')
        before=len([n for n in nodes() if n.get('label','').startswith('内容标签 ')])
        click('新建终端')
        assert not find('启动分组')
        click('启动 Terminal')
        wait_for(lambda:len([n for n in nodes() if n.get('label')=='Terminal'])==2)
        assert len([n for n in nodes() if n.get('label','').startswith('内容标签 ')])==before
        click('新建终端');click('新建网页')
        wait_for(lambda:find('当前网页地址'))
        assert find('当前网页地址').get('value','')=='' and not find('网页地址')
        assert len([n for n in nodes() if n.get('label','').startswith('子标签 ')])==3
        capture('workbench-context-add.png')
        # Drag the other workspace into the active one, preserving its terminal.
        source=next(n for n in nodes() if n.get('label','').startswith('内容标签 ') and n.get('label')!='内容标签 2')['bounds']
        target=find('内容标签 2')['bounds']
        drag((source['x']+12,source['y']+12),(target['x']+target['width']/2,target['y']+12))
        assert len([n for n in nodes() if n.get('label','').startswith('内容标签 ')])==1
        assert len([n for n in nodes() if n.get('label','').startswith('关闭子标签 ')])==4
        click('新建工作区');click('启动 Terminal')
        wait_for(lambda:rpc('windows')['windows'][0]['title']=='Workspace2')
        assert find('新建终端'), 'new workspace has no local +'
        quit_app(second)
        log.flush()
        assert 'cannot compile the effectps shader' not in (OUT/'workbench.log').read_text(encoding='utf8'), 'GPU blur shader failed'
        report=dict(native_frosted_overlay_snapshots=os.environ.get('CYRENE_CAPTURE_NATIVE')=='1',no_sidebars=True,inline_children_focus_and_collapse=True,embedded_http_and_javascript=True,browser_back_forward_reload=True,browser_docking_without_reload=True,mixed_workspace_restart=True,browser_close_preserves_terminal=True,workspace_add_stays_in_origin=True,child_drag_order=True,external_workspace_join=True,child_close_accessible=True,default_workspace_names=True,standalone_without_local_add=True)
        (OUT/'workbench.json').write_text(json.dumps(report,indent=2),encoding='utf-8')
        print(json.dumps(report,indent=2))
    except Exception:
        if client and client.poll() is None:
            (OUT/'workbench-debug.json').write_text(json.dumps(dict(tree=nodes(),hits=hits),ensure_ascii=False,indent=2),encoding='utf-8')
            capture('workbench-failure.png')
        raise
    finally:
        if client and client.poll() is None:
            client.stdin.close()
            try: client.wait(timeout=5)
            except subprocess.TimeoutExpired: client.kill()
        for proc in procs:
            if proc.poll() is None: subprocess.run(['taskkill','/PID',str(proc.pid),'/T','/F'],capture_output=True)
        log.close()
server.shutdown()
