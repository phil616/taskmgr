"""Exercise the shipped EXE, Qt WebEngine and Vue using an isolated backend."""
import json
import ctypes
from pathlib import Path
import sys
import tempfile
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from PySide6.QtCore import QSettings
from PySide6.QtWidgets import QApplication
from .app import Window


def main():
    report = Path(sys.argv[sys.argv.index('--smoke-test') + 1]).resolve()
    result = {'passed': False}
    with tempfile.TemporaryDirectory(prefix='taskmgr-smoke-') as state:
        class Backend(BaseHTTPRequestHandler):
            def log_message(self, *_args):
                pass

            def do_GET(self):
                self.send_response(401 if self.path.endswith('/auth/profile') else 200)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(b'{"code":0,"data":[],"marker":"packaged-backend"}')

        backend = ThreadingHTTPServer(('127.0.0.1', 0), Backend)
        thread = threading.Thread(target=backend.serve_forever, daemon=True)
        thread.start()
        app = QApplication([])
        app.setOrganizationName('TaskmgrPackagedSmoke')
        app.setApplicationName('DesktopSmoke')
        QSettings.setDefaultFormat(QSettings.Format.IniFormat)
        QSettings.setPath(QSettings.Format.IniFormat, QSettings.Scope.UserScope, state)
        # Avoid opening the initial connection dialog or creating a real profile.
        original_configure = Window.configure
        Window.configure = lambda self: None
        window = Window(profile_root=Path(state) / 'profiles')
        Window.configure = original_configure
        window.show()
        try:
            dll_paths = {}
            if sys.platform == 'win32' and getattr(sys, 'frozen', False):
                kernel = ctypes.WinDLL('kernel32', use_last_error=True)
                kernel.GetModuleHandleW.argtypes = [ctypes.c_wchar_p]
                kernel.GetModuleHandleW.restype = ctypes.c_void_p
                kernel.GetModuleFileNameW.argtypes = [ctypes.c_void_p, ctypes.c_wchar_p, ctypes.c_uint]
                for name in ('Qt6Core.dll', 'Qt6Widgets.dll', 'Qt6WebEngineCore.dll'):
                    handle = kernel.GetModuleHandleW(name)
                    buffer = ctypes.create_unicode_buffer(32768)
                    if not handle or not kernel.GetModuleFileNameW(handle, buffer, len(buffer)):
                        raise RuntimeError(f'Qt DLL not loaded: {name}')
                    path = Path(buffer.value).resolve()
                    if not path.is_relative_to(Path(sys.executable).resolve().parent):
                        raise RuntimeError(f'Qt DLL loaded outside the release: {path}')
                    dll_paths[name] = str(path)
            window.connect_backend(f'http://127.0.0.1:{backend.server_port}')
            deadline = time.monotonic() + 45
            values = []
            script = '''(async () => {
                if (!document.body.innerText.includes('用户名')) return;
                const response = await fetch('/api/v1/health');
                const data = await response.json();
                window.packagedSmoke = document.documentElement.classList.contains('desktop-client')
                    && data.marker === 'packaged-backend';
            })()'''
            while time.monotonic() < deadline:
                app.processEvents()
                if values and values[-1] is True:
                    window.grab().save(str(report.with_suffix('.png')))
                    result = {'passed': True, 'checks': ['Qt DLL loading', 'WebEngine process',
                              'Vue login rendering', 'desktop styles', 'authenticated local proxy'],
                              'dll_paths': dll_paths}
                    break
                window.view.page().runJavaScript(script)
                window.view.page().runJavaScript('window.packagedSmoke === true', values.append)
                time.sleep(.05)
            else:
                raise RuntimeError('Packaged Vue/WebEngine smoke test timed out')
        except Exception as exc:
            result['error'] = str(exc)
        finally:
            window.close()
            app.processEvents()
            backend.shutdown()
            backend.server_close()
            thread.join()
            report.write_text(json.dumps(result, ensure_ascii=False, indent=2), encoding='utf-8')
    raise SystemExit(0 if result['passed'] else 1)
