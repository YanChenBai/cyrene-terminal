"""Exercise the real native Windows app, PTY, folder groups and tray lifecycle.

Build an inspector-enabled binary first: go build -tags e2e -o .mygo/cyrene-isolated-smoke.exe ./cmd/cyrene-terminal
Run: python scripts/native-smoke.py
Artifacts stay in test-results/; user settings and agent processes are untouched.
"""

import base64
import ctypes
import struct
import zlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "test-results"
OUT.mkdir(exist_ok=True)
USER32 = ctypes.windll.user32
USER32.IsWindowVisible.argtypes = [ctypes.c_void_p]
USER32.PostMessageW.argtypes = [ctypes.c_void_p, ctypes.c_uint, ctypes.c_size_t, ctypes.c_ssize_t]
KERNEL32 = ctypes.windll.kernel32
KERNEL32.OpenProcess.argtypes = [ctypes.c_ulong, ctypes.c_bool, ctypes.c_ulong]
KERNEL32.OpenProcess.restype = ctypes.c_void_p
KERNEL32.WaitForSingleObject.argtypes = [ctypes.c_void_p, ctypes.c_ulong]
KERNEL32.CloseHandle.argtypes = [ctypes.c_void_p]


def process_stopped(pid):
    handle = KERNEL32.OpenProcess(0x00100000, False, pid)
    if not handle:
        return True
    try:
        return KERNEL32.WaitForSingleObject(handle, 0) == 0
    finally:
        KERNEL32.CloseHandle(handle)


def wait_for(fn, timeout=25):
    deadline = time.monotonic() + timeout
    last = None
    while time.monotonic() < deadline:
        try:
            value = fn()
            if value:
                return value
        except (OSError, RuntimeError) as exc:
            last = exc
        time.sleep(0.1)
    raise AssertionError(f"Timed out waiting: {last}")


def png_color_count(data, color, tolerance=0):
    # Decode the non-interlaced RGB/RGBA PNGs emitted by MyGo, with stdlib only.
    offset, compressed = 8, bytearray()
    while offset < len(data):
        length = struct.unpack_from(">I", data, offset)[0]
        kind, payload = data[offset+4:offset+8], data[offset+8:offset+8+length]
        if kind == b"IHDR":
            width,height,depth,mode,_,_,interlace = struct.unpack(">IIBBBBB", payload)
            assert depth == 8 and mode in (2,6) and interlace == 0
        if kind == b"IDAT":
            compressed.extend(payload)
        offset += length+12
    raw, bpp = zlib.decompress(compressed), 4 if mode == 6 else 3
    stride, at, count = width*bpp, 0, 0
    previous = bytearray(stride)
    for _ in range(height):
        filt = raw[at]
        row = bytearray(raw[at+1:at+1+stride])
        at += stride+1
        for x in range(stride):
            left = row[x-bpp] if x>=bpp else 0
            up = previous[x]
            upper_left = previous[x-bpp] if x>=bpp else 0
            if filt == 1:
                row[x] = (row[x]+left)&255
            elif filt == 2:
                row[x] = (row[x]+up)&255
            elif filt == 3:
                row[x] = (row[x]+(left+up)//2)&255
            elif filt == 4:
                p = left+up-upper_left
                pa,pb,pc = abs(p-left),abs(p-up),abs(p-upper_left)
                predictor = left if pa<=pb and pa<=pc else up if pb<=pc else upper_left
                row[x] = (row[x]+predictor)&255
        target = bytes(color[:bpp])
        count += sum(all(abs(row[x+j]-target[j])<=tolerance for j in range(bpp)) for x in range(0,stride,bpp))
        previous = row
    return count


with tempfile.TemporaryDirectory(prefix="cyrene-smoke-") as tmp:
    tmp = Path(tmp)
    settings = tmp / "cyrene-terminal" / "settings.json"
    settings.parent.mkdir()
    settings.write_text(json.dumps({
        "directory": str(tmp),
        "profiles": [
            *[{"id": name.lower(), "name": name, "command": [name.lower()]} for name in ["Codex", "Claude", "Pi", "OMP", "OpenCode"]],
            {"id": "shell", "name": "Terminal", "command": [os.getenv("CYRENE_SMOKE_SHELL", "powershell.exe"), "-NoLogo", "-NoProfile"]},
        ],
    }), encoding="utf-8")
    env = dict(os.environ, APPDATA=str(tmp), MYGO_AUTOMATION=str(tmp / "automation.sock"),
               CYRENE_ACCEPTANCE_NO_FOCUS="1", CYRENE_ACCEPTANCE_INSTANCE=str(tmp))
    # Reproduce Explorer/PowerShell's mixed-case Windows environment key.
    env["Path"] = env.pop("PATH", os.defpath)
    if os.getenv("CYRENE_SMOKE_BUNDLED_ONLY") == "1":
        env["LOCALAPPDATA"] = str(tmp / "empty-local-app-data")
    executable = ROOT / ".mygo" / "cyrene-isolated-smoke.exe"
    log = (OUT / "native-smoke.log").open("w", encoding="utf-8")
    proc = subprocess.Popen([str(executable)], cwd=ROOT, env=env, stdout=log, stderr=log,
                            creationflags=subprocess.CREATE_NO_WINDOW)
    client = None
    serial = 0
    report = {}

    clipboard_path = None
    try:
        client = subprocess.Popen(["go", "run", "./scripts/automation-client", env["MYGO_AUTOMATION"]],
                                  cwd=ROOT, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                  stderr=log, creationflags=subprocess.CREATE_NO_WINDOW)

        def rpc(method, **params):
            global serial
            serial += 1
            client.stdin.write((json.dumps({"id": serial, "method": method, "params": params}) + "\n").encode())
            client.stdin.flush()
            response = json.loads(client.stdout.readline())
            if "error" in response:
                raise RuntimeError(response["error"])
            return response["result"]

        def tree():
            return rpc("tree")["nodes"]

        def click(label):
            node = wait_for(lambda: next((n for n in tree() if n.get("label") == label and "press" in n.get("actions", [])), None))
            rpc("act", id=node["id"], action="press")
            # Terminal cursors may keep drawing indefinitely. Allow the 420ms
            # opening transition to settle without requiring the whole app idle.
            time.sleep(0.8)

        def capture(name):
            result = rpc("capture")
            (OUT / name).write_bytes(base64.b64decode(result["png"]))
            return result

        def send_command(command, pane_key=None):
            terminal = wait_for(lambda: next((n for n in tree() if n.get("label") == "Terminal" and "focus" in n.get("actions", [])
                                               and (pane_key is None or n.get("key") == pane_key)
                                               and n.get("description", "").rstrip().endswith(">")), None))
            rpc("act", id=terminal["id"], action="focus")
            # Commit pointer focus before typing; a pending popover focus
            # restoration can otherwise consume a command in an offscreen window.
            bounds = terminal["bounds"]
            rpc("input", events=[{"type": "click", "x": bounds["x"]+24, "y": bounds["y"]+24}])
            time.sleep(0.2)
            rpc("input", events=[{"type": "text", "text": command}, {"type": "key", "key": "Enter"}])

        def wait_terminal(key):
            wait_for(lambda: any(n.get("label") == "Terminal" and n.get("key") == key and "focus" in n.get("actions", []) for n in tree()))

        windows = rpc("windows")["windows"]
        assert windows[0]["kind"] == "content", windows
        report["native_window"] = True
        wait_for(lambda: any(n.get("label") == "Terminal" and "focus" in n.get("actions", []) for n in tree()))
        home_marker = tmp / "home-cwd.txt"
        send_command(f"(Get-Location).Path | Set-Content '{home_marker}'")
        wait_for(home_marker.exists)
        assert Path(home_marker.read_text().strip()).samefile(Path.home())
        report["ungrouped_starts_in_user_home"] = True

        # Optional native clipboard check. Hold a materialized clipboard backup
        # in an STA process and restore it even when a terminal assertion fails.
        if os.getenv("CYRENE_SMOKE_CLIPBOARD") == "1":
            clipboard_script = r'''
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$backup = New-Object System.Windows.Forms.DataObject
$old = [System.Windows.Forms.Clipboard]::GetDataObject()
if ($null -ne $old) {
    foreach ($format in $old.GetFormats($false)) {
        $value = $old.GetData($format, $false)
        if ($value -is [System.Drawing.Image]) { $value = $value.Clone() }
        if ($null -ne $value) { $backup.SetData($format, $false, $value) }
    }
}
try {
    $image = New-Object System.Drawing.Bitmap 3, 2
    $image.SetPixel(0, 0, [System.Drawing.Color]::Red)
    [System.Windows.Forms.Clipboard]::SetImage($image)
    [Console]::WriteLine('READY')
    [Console]::ReadLine() | Out-Null
} finally {
    [System.Windows.Forms.Clipboard]::SetDataObject($backup, $true)
    $image.Dispose()
}
'''
            encoded = base64.b64encode(clipboard_script.encode("utf-16-le")).decode()
            holder = subprocess.Popen(["powershell.exe", "-NoLogo", "-NoProfile", "-STA", "-EncodedCommand", encoded],
                                      stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=log,
                                      creationflags=subprocess.CREATE_NO_WINDOW)
            try:
                assert holder.stdout.readline().strip() == b"READY"
                paste_marker = tmp / "clipboard-path.txt"
                send_command(f"$p = Read-Host 'CLIP_READY'; $p | Set-Content '{paste_marker}'")
                wait_for(lambda: any("CLIP_READY" in n.get("description", "") for n in tree() if n.get("label") == "Terminal"))
                rpc("input", events=[{"type": "key", "key": "Ctrl+V"}])
                wait_for(lambda: any("cyrene-clipboard-" in n.get("description", "") for n in tree() if n.get("label") == "Terminal"))
                rpc("input", events=[{"type": "key", "key": "Enter"}])
                wait_for(paste_marker.exists)
                clipboard_path = Path(paste_marker.read_text(encoding="utf-8-sig").strip().strip("'"))
                assert clipboard_path.is_file(), clipboard_path
                assert clipboard_path.read_bytes().startswith(b"\x89PNG\r\n\x1a\n")
                report["native_ctrl_v_screenshot_produces_readable_png"] = True
            finally:
                holder.communicate(input=b"\n", timeout=15)

        def set_field(label, text):
            node = wait_for(lambda: next((n for n in tree() if n.get("label") == label and n.get("role") in ("textField", "comboBox")), None))
            rpc("act", id=node["id"], action="setValue", text=text)

        def add_group(name, directory):
            directory.mkdir()
            click("新建文件夹分组")
            set_field("分组名称", name)
            set_field("绑定目录", str(directory))
            click("保存")
            wait_for(lambda: any(g["name"] == name for g in json.loads(settings.read_text()).get("groups", [])))

        project_a, project_b = tmp / "Project Alpha", tmp / "Project Beta"
        add_group("Alpha", project_a)
        add_group("Beta", project_b)
        click("在 Alpha 新建终端")
        wait_for(lambda: any(n.get("label") == "启动 Terminal" for n in tree()))
        assert not any(n.get("label", "").startswith("恢复 ") for n in tree()), "Plus launcher must only create sessions"
        report["plus_launcher_only_creates_sessions"] = True
        capture("launcher.png")
        anchor = next(n for n in tree() if n.get("label") == "在 Alpha 新建终端")
        item = next(n for n in tree() if n.get("label") == "启动 Codex")
        assert item["bounds"]["y"] >= anchor["bounds"]["y"] + anchor["bounds"]["height"]
        report["launcher_below_group_button"] = True
        click("启动 Terminal")
        wait_terminal(4)
        cwd_a = tmp / "alpha-cwd.txt"
        marker = tmp / "running.txt"
        command = f"$PID | Set-Content '{tmp / 'shell-pid.txt'}'; (Get-Location).Path | Set-Content '{cwd_a}'; [Console]::Write([char]27 + ']0;Project Alpha' + [char]7); Start-Sleep -Seconds 3; 'alive' | Set-Content '{marker}'"
        send_command(command)
        wait_for(cwd_a.exists)
        assert Path(cwd_a.read_text().strip()).samefile(project_a)
        wait_for(lambda: any(n.get("key")=="tab-5" and n.get("label")=="Project Alpha" for n in tree())
                 and any(n.get("label")=="内容标签 5" and n.get("description")=="Project Alpha" for n in tree()))
        assert rpc("windows")["windows"][0]["title"] == "Project Alpha"
        report["terminal_osc_title_in_sidebar_header_and_window"] = True
        wait_for(marker.exists)
        rpc("act",key="tab-2",action="press")
        wait_terminal(1)
        rpc("act",key="tab-5",action="press")
        wait_terminal(4)
        report["tab_switch_keeps_sessions"] = True
        click("内容标签 2")
        wait_terminal(1)
        click("内容标签 5")
        wait_terminal(4)
        report["center_tabs_reuse_terminal_sessions"] = True
        click("在 Beta 新建终端")
        click("启动 Terminal")
        wait_terminal(7)
        cwd_b = tmp / "beta-cwd.txt"
        send_command(f"(Get-Location).Path | Set-Content '{cwd_b}'; [Console]::Write([char]27 + ']0;Project Beta' + [char]7)")
        wait_for(cwd_b.exists)
        assert Path(cwd_b.read_text().strip()).samefile(project_b)
        report["different_groups_inherit_bound_directories"] = True
        assert not any(n.get("role") == "splitter" for n in tree())
        report["single_immersive_terminal"] = True
        rpc("input",events=[{"type":"pointerDown","x":244,"y":500}])
        time.sleep(.05)
        rpc("input",events=[{"type":"pointerMove","x":312,"y":500}])
        time.sleep(.05)
        rpc("input",events=[{"type":"pointerUp","x":312,"y":500}])
        wait_for(lambda: any(n.get("label")=="Terminal" and n["bounds"]["x"]>300 for n in tree()))
        report["sidebar_native_edge_drag_resizes"] = True
        wait_for(lambda: any(n.get("key")=="tab-8" and n.get("label")=="Project Beta" for n in tree())
                 and any(n.get("label")=="内容标签 8" and n.get("description")=="Project Beta" for n in tree()))
        time.sleep(0.5)
        capture("workspace.png")
        assert not any(n.get("label") == "后台会话" for n in tree())
        report["no_independent_background_feature"] = True
        click("关闭内容标签 8")
        time.sleep(0.5)
        capture("close-confirmation.png")
        click("取消")
        wait_terminal(7)
        report["close_tab_cancel_preserves_session"] = True

        chafa_debug = tmp / "yazi-debug.txt"
        send_command(f"yazi --debug | Set-Content -Encoding UTF8 '{chafa_debug}'")
        wait_for(chafa_debug.exists)
        debug = chafa_debug.read_text(encoding="utf-8-sig")
        report["yazi_chafa_available"] = any("chafa" in line and "1.18.3" in line for line in debug.splitlines())
        # Yazi's terminal probes can leave late replies in PSReadLine after
        # --debug exits. Clear that diagnostic line before the interactive run.
        time.sleep(1)
        rpc("input",events=[{"type":"key","key":"Ctrl+C"}])
        time.sleep(.2)
        if report["yazi_chafa_available"]:
            image_dir = tmp / "preview"
            image_dir.mkdir()
            def chunk(kind, data):
                return struct.pack(">I", len(data))+kind+data+struct.pack(">I", zlib.crc32(kind+data)&0xffffffff)
            raw = b"".join(b"\0"+bytes((240,80,80))*32+bytes((40,190,220))*32 for _ in range(64))
            preview_image = image_dir / "preview.png"
            preview_image.write_bytes(b"\x89PNG\r\n\x1a\n"+chunk(b"IHDR",struct.pack(">IIBBBBB",64,64,8,2,0,0,0))+chunk(b"IDAT",zlib.compress(raw))+chunk(b"IEND",b""))
            send_command(f"yazi '{preview_image}'")
            time.sleep(2)
            preview_shot = capture("yazi-preview.png")
            red_pixels = png_color_count(base64.b64decode(preview_shot["png"]), (240,80,80,255),2)
            cyan_pixels = png_color_count(base64.b64decode(preview_shot["png"]), (40,190,220,255),2)
            report["yazi_ansi_image_preview"] = red_pixels > 1000 and cyan_pixels > 1000
            assert report["yazi_ansi_image_preview"], (red_pixels,cyan_pixels)
            rpc("input",events=[{"type":"text","text":"q"}])
            time.sleep(.3)

        graphic = tmp / "kitty-sequence.txt"
        color = (10,210,150,255)
        pixels = bytes(color) * 64
        sequence = "\x1b_Ga=T,f=32,s=8,v=8,c=12,r=6,i=37,C=1,q=2;" + base64.b64encode(pixels).decode() + "\x1b\\"
        graphic.write_bytes(sequence.encode("ascii"))
        send_command(f"[Console]::Write([IO.File]::ReadAllText('{graphic}'))")
        def image_pixels():
            shot=rpc("capture")
            return png_color_count(base64.b64decode(shot["png"]),color)
        time.sleep(0.5)
        kitty_count = image_pixels()
        capture("kitty-native.png")
        report["kitty_graphics_through_real_windows_pty"] = kitty_count > 1000
        if kitty_count <= 1000:
            report["kitty_windows_limitation"] = "Renderer test passes; native PowerShell/ConPTY did not pass the Kitty APC image sequence through"
        graphic.write_bytes(b"\x1b_Ga=d,d=I,i=37,q=2\x1b\\")
        send_command(f"[Console]::Write([IO.File]::ReadAllText('{graphic}'))")
        wait_for(lambda:image_pixels()==0)
        report["kitty_graphics_delete"] = kitty_count > 1000
        click("切换侧栏")
        time.sleep(0.35)
        capture("sidebar-collapsed.png")
        assert not any(str(n.get("key", "")).startswith("tab-") for n in tree())
        toggle=next(n for n in tree() if n.get("label")=="切换侧栏")
        assert toggle["bounds"]["x"] < 20 and toggle["bounds"]["y"] < 10
        report["sidebar_fully_hidden_toggle_in_titlebar"] = True
        click("切换侧栏")
        click("外观设置")
        font = wait_for(lambda: next((n for n in tree() if n.get("label") == "终端字体" and n.get("role") in ("textField", "comboBox")), None))
        rpc("act", id=font["id"], action="setValue", text="Cascadia Mono")
        click("保存")
        wait_for(lambda: json.loads(settings.read_text(encoding="utf-8")).get("fontFamily") == "Cascadia Mono")
        report["settings_saved"] = True

        # Trigger the real window close event rather than an in-app hide button.
        handles = []
        callback_type = ctypes.WINFUNCTYPE(ctypes.c_bool, ctypes.c_void_p, ctypes.c_void_p)

        @callback_type
        def enum_window(hwnd, _):
            pid = ctypes.c_ulong()
            USER32.GetWindowThreadProcessId(ctypes.c_void_p(hwnd), ctypes.byref(pid))
            if pid.value == proc.pid and USER32.IsWindowVisible(hwnd):
                handles.append(hwnd)
            return True

        USER32.EnumWindows(enum_window, None)
        assert handles, "No visible native window"
        hidden_marker = tmp / "hidden.txt"
        send_command(f"Start-Sleep -Seconds 3; 'background' | Set-Content '{hidden_marker}'")
        USER32.PostMessageW(handles[0], 0x0010, 0, 0)  # WM_CLOSE
        wait_for(lambda: not USER32.IsWindowVisible(handles[0]))
        wait_for(hidden_marker.exists)
        assert proc.poll() is None
        report["window_close_keeps_shell_running"] = True
        second_env = dict(env)
        second_env.pop("MYGO_AUTOMATION", None)
        second = subprocess.Popen([str(executable)], cwd=ROOT, env=second_env, stdout=log, stderr=log,
                                  creationflags=subprocess.CREATE_NO_WINDOW)
        assert second.wait(timeout=15) == 0
        wait_for(lambda: USER32.IsWindowVisible(handles[0]))
        assert len([n for n in tree() if str(n.get("key", "")).startswith("tab-")]) == 3
        report["second_launch_restores_original_sessions"] = True
        capture("restored.png")
        # Capture full desktop shell presentation after restoring the named session.
        assert len(json.loads(settings.read_text())["groups"]) == 2
        report["group_bindings_persisted"] = True
        click("新建终端")
        click("启动 Terminal")
        wait_terminal(10)
        close_pid_file = tmp / "close-pid.txt"
        send_command(f"$PID | Set-Content '{close_pid_file}'")
        wait_for(close_pid_file.exists)
        close_pid = int(close_pid_file.read_text(encoding="utf-8-sig").strip())
        click("关闭终端 11")
        click("终止进程")
        wait_for(lambda: process_stopped(close_pid))
        assert len([n for n in tree() if str(n.get("key", "")).startswith("tab-")]) == 3
        report["close_tab_confirm_terminates_shell"] = True
        if os.getenv("CYRENE_SMOKE_TILING") == "1":
            def terminal_nodes():
                return [n for n in tree() if n.get("label")=="Terminal" and "focus" in n.get("actions", [])]

            def palette(query):
                rpc("input", events=[{"type":"key", "key":"Ctrl+Shift+P"}])
                wait_for(lambda:any(n.get("label")=="命令搜索" for n in tree()))
                time.sleep(.4)
                set_field("命令搜索",query)

            rpc("act", key="tab-5", action="press")
            wait_terminal(4)
            send_command("Clear-Host; Write-Host 'Build workspace' -ForegroundColor Cyan; Write-Host 'Terminal stays running while layouts change.'",4)
            wait_for(lambda: any("Build workspace" in n.get("description","") for n in terminal_nodes()))
            tabs=[n for n in tree() if n.get("label","").startswith("内容标签 ")]
            assert all(n["bounds"]["y"]+n["bounds"]["height"]<=32 for n in tabs),tabs
            assert not any(n.get("label")=="添加内容标签" for n in tree())
            rpc("input",events=[{"type":"pointerMove","x":600,"y":300}])
            time.sleep(.25)
            capture("tiling-01-titlebar.png")
            source=next(n for n in tree() if n.get("key")=="tab-8")["bounds"]
            target=next(n for n in terminal_nodes() if n.get("key")==4)["bounds"]
            sx,sy=source["x"]+30,source["y"]+20
            tx,ty=target["x"]+target["width"]-25,target["y"]+target["height"]*.5
            rpc("input",events=[{"type":"pointerDown","x":sx,"y":sy}])
            rpc("input",events=[{"type":"pointerMove","x":sx+12,"y":sy+3}])
            rpc("input",events=[{"type":"pointerMove","x":tx,"y":ty}])
            wait_for(lambda:any(n.get("label")=="向右分屏" for n in tree()))
            time.sleep(.2)
            capture("tiling-02-drop-preview.png")
            rpc("input",events=[{"type":"pointerUp","x":tx,"y":ty}])
            wait_for(lambda:len(terminal_nodes())==2)
            assert {n["key"] for n in terminal_nodes()}=={4,7}
            report["drag_across_groups_reuses_native_processes"]=True
            time.sleep(.25)
            divider=next(n for n in tree() if n.get("role")=="splitter")["bounds"]
            dx,dy=divider["x"]+divider["width"]/2,divider["y"]+divider["height"]/2
            before=next(n for n in terminal_nodes() if n["key"]==4)["bounds"]["width"]
            rpc("input",events=[{"type":"pointerDown","x":dx,"y":dy}])
            time.sleep(.08)
            rpc("input",events=[{"type":"pointerMove","x":dx-75,"y":dy}])
            time.sleep(.08)
            rpc("input",events=[{"type":"pointerUp","x":dx-75,"y":dy}])
            wait_for(lambda:next(n for n in terminal_nodes() if n["key"]==4)["bounds"]["width"]<before-30)
            report["native_split_divider_resizes"]=True
            pid_check=tmp/"alpha-pid-after-dock.txt"
            send_command(f"$PID | Set-Content '{pid_check}'",4)
            wait_for(pid_check.exists)
            assert pid_check.read_text(encoding="utf-8-sig").strip()==(tmp/"shell-pid.txt").read_text(encoding="utf-8-sig").strip()
            report["dock_preserves_original_shell_pid"]=True
            click("窗格 7")
            send_command("Clear-Host; Write-Host 'Review workspace' -ForegroundColor Green",7)
            palette("")
            capture("tiling-03-command-palette.png")
            set_field("命令搜索","split down")
            rpc("input",events=[{"type":"key","key":"Enter"}])
            wait_for(lambda:any(n.get("label")=="命令 新建 Terminal" for n in tree()))
            set_field("命令搜索","new Terminal")
            rpc("input",events=[{"type":"key","key":"Enter"}])
            wait_for(lambda:len(terminal_nodes())==3)
            extra=next(n["key"] for n in terminal_nodes() if n["key"] not in (4,7))
            cwd_check=tmp/"split-cwd.txt"
            split_pid=tmp/"split-pid.txt"
            send_command(f"$PID | Set-Content '{split_pid}'; (Get-Location).Path | Set-Content '{cwd_check}'; Clear-Host; Write-Host 'Project checks' -ForegroundColor Yellow",extra)
            wait_for(cwd_check.exists)
            assert Path(cwd_check.read_text().strip()).samefile(project_b)
            report["keyboard_split_inherits_target_directory"]=True
            time.sleep(.4)
            capture("tiling-04-three-panes.png")
            click(f"窗格 {extra}")
            palette("rename tab")
            rpc("input",events=[{"type":"key","key":"Enter"}])
            set_field("标签名称","Project workspace")
            click("保存")
            wait_for(lambda:any(n.get("label","").startswith("内容标签 ") and n.get("description")=="Project workspace · 3" for n in tree()))
            palette("rename session")
            rpc("input",events=[{"type":"key","key":"Enter"}])
            set_field("会话别名","Checks")
            click("保存")
            assert len([n for n in tree() if n.get("label")=="Checks" and str(n.get("key","")).startswith("tab-")])==1
            assert any(n.get("key")=="tab-5" and n.get("label")=="Project Alpha" for n in tree())
            assert any(n.get("key")=="tab-8" and n.get("label")=="Project Beta" for n in tree())
            assert rpc("windows")["windows"][0]["title"]=="Project workspace · 3"
            report["palette_renames_only_focused_tab_or_session"]=True
            capture("tabs-08-compact-rounded.png")
            palette("detach")
            rpc("input",events=[{"type":"key","key":"Enter"}])
            wait_for(lambda:len(terminal_nodes())==1 and terminal_nodes()[0]["key"]==extra)
            report["keyboard_detach_keeps_terminal"]=True
            capture("tiling-05-detached.png")
            # Put the single Checks terminal back into the composite screen,
            # then drag its pane title back to the tab strip and reorder it.
            rpc("act",key="tab-8",action="press")
            wait_for(lambda:len(terminal_nodes())==2)
            checks_tab=next(n for n in tree() if n.get("label","").startswith("内容标签 ") and n.get("description")=="Checks")["bounds"]
            beta=next(n for n in terminal_nodes() if n["key"]==7)["bounds"]
            cx,cy=checks_tab["x"]+20,checks_tab["y"]+16
            bx,by=beta["x"]+beta["width"]/2,beta["y"]+beta["height"]-20
            rpc("input",events=[{"type":"pointerDown","x":cx,"y":cy}])
            rpc("input",events=[{"type":"pointerMove","x":cx+10,"y":cy+4}])
            rpc("input",events=[{"type":"pointerMove","x":bx,"y":by}])
            time.sleep(.15)
            rpc("input",events=[{"type":"pointerUp","x":bx,"y":by}])
            wait_for(lambda:len(terminal_nodes())==3)
            handle=next(n for n in tree() if n.get("label")==f"窗格 {extra}")["bounds"]
            hx,hy=handle["x"]+25,handle["y"]+12
            rpc("input",events=[{"type":"pointerDown","x":hx,"y":hy}])
            rpc("input",events=[{"type":"pointerMove","x":hx+10,"y":hy+3}])
            rpc("input",events=[{"type":"pointerMove","x":950,"y":16}])
            time.sleep(.15)
            rpc("input",events=[{"type":"pointerUp","x":950,"y":16}])
            wait_for(lambda:len(terminal_nodes())==1 and terminal_nodes()[0]["key"]==extra)
            report["drag_pane_back_to_tab_strip"]=True
            tabs=sorted([n for n in tree() if n.get("label","").startswith("内容标签 ")],key=lambda n:n["bounds"]["x"])
            moved=next(n for n in tabs if n.get("description")=="Checks")["bounds"]
            first=tabs[0]["bounds"]
            rpc("input",events=[{"type":"pointerDown","x":moved["x"]+20,"y":16}])
            rpc("input",events=[{"type":"pointerMove","x":moved["x"]+10,"y":16}])
            rpc("input",events=[{"type":"pointerMove","x":first["x"]+2,"y":16}])
            time.sleep(.15)
            rpc("input",events=[{"type":"pointerUp","x":first["x"]+2,"y":16}])
            wait_for(lambda:sorted([n for n in tree() if n.get("label","").startswith("内容标签 ")],key=lambda n:n["bounds"]["x"])[0].get("description")=="Checks")
            report["native_tab_drag_reorders_screens"]=True
            after_pid=tmp/"split-pid-after-tab-drag.txt"
            send_command(f"$PID | Set-Content '{after_pid}'",extra)
            wait_for(after_pid.exists)
            assert after_pid.read_text(encoding="utf-8-sig").strip()==split_pid.read_text(encoding="utf-8-sig").strip()
            report["tab_drag_preserves_shell_pid"]=True
            capture("tabs-09-compact-reordered.png")
            rpc("input",events=[{"type":"key","key":"Ctrl+Shift+W"}])
            click("终止进程")
            wait_for(lambda:len([n for n in tree() if str(n.get("key", "")).startswith("tab-")])==3)
        if os.getenv("CYRENE_ACCEPTANCE_HOLD") == "1":
            (OUT / "acceptance-ready.json").write_text(json.dumps({"pid":proc.pid,"socket":env["MYGO_AUTOMATION"]}),encoding="utf-8")
            print("Native acceptance window is ready for visual inspection",flush=True)
            wait_for(lambda:(OUT / "acceptance-continue").exists(),timeout=240)
        rpc("menu",path=["退出应用"])
        assert proc.wait(timeout=15) == 0

        if clipboard_path is not None:
            assert not clipboard_path.exists(), "Clipboard image survived terminal shutdown"
            report["clipboard_image_cleaned_on_quit"] = True
        report["explicit_quit"] = True
        kernel32 = ctypes.windll.kernel32
        kernel32.OpenProcess.argtypes = [ctypes.c_ulong, ctypes.c_bool, ctypes.c_ulong]
        kernel32.OpenProcess.restype = ctypes.c_void_p
        kernel32.WaitForSingleObject.argtypes = [ctypes.c_void_p, ctypes.c_ulong]
        kernel32.CloseHandle.argtypes = [ctypes.c_void_p]
        shell_pid = int((tmp / "shell-pid.txt").read_text(encoding="utf-8-sig").strip())
        process_handle = kernel32.OpenProcess(0x00100000, False, shell_pid)
        if process_handle:
            try:
                assert kernel32.WaitForSingleObject(process_handle, 8000) == 0, "Shell survived explicit quit"
            finally:
                kernel32.CloseHandle(process_handle)
        report["quit_terminates_shell"] = True
        (OUT / "native-smoke.json").write_text(json.dumps(report, indent=2), encoding="utf-8")
        print(json.dumps(report, indent=2))
    except Exception:
        try:
            (OUT / "failure-tree.json").write_text(json.dumps(tree(), ensure_ascii=False, indent=2), encoding="utf-8")
            capture("failure.png")
        except Exception:
            pass
        raise
    finally:
        if client:
            client.stdin.close()
            client.wait(timeout=5)
        if proc.poll() is None:
            subprocess.run(["taskkill", "/PID", str(proc.pid), "/T", "/F"], capture_output=True)
        log.close()
