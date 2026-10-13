"""Run with a display (or xvfb-run). Uses an isolated mock backend/config."""
import json
import os
from pathlib import Path
import sys
import subprocess
import tempfile
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

# Isolate both QSettings and Chromium cookies from the user's real workspace.
state = tempfile.TemporaryDirectory()
os.environ['XDG_CONFIG_HOME'] = state.name
os.environ['XDG_DATA_HOME'] = state.name
os.environ['XDG_CACHE_HOME'] = state.name

from PySide6.QtCore import QEventLoop, QPoint, QPointF, QSettings, QTimer, Qt
from PySide6.QtGui import QWheelEvent
from PySide6.QtTest import QTest
from PySide6.QtWidgets import QApplication
from taskmgr_desktop.app import ConnectionDialog, Window


class Backend(BaseHTTPRequestHandler):
    requests = []

    def log_message(self, *_args):
        pass

    def do_GET(self):
        self.requests.append((self.path, self.headers.get('Cookie', '')))
        profile = self.path.endswith('/auth/profile')
        self.send_response(401 if profile and not self.headers.get('Cookie') else 200)
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        data = []
        if profile:
            data = {'id': 1, 'username': 'demo', 'display_name': '测试工作空间'}
        elif '/summary' in self.path:
            data = {'total': 0, 'active': 0, 'completed': 0}
        elif '/unread-count' in self.path:
            data = {'count': 0}
        self.wfile.write(json.dumps({'code': 0, 'data': data, 'meta': {'total': 0}, 'cookie': self.headers.get('Cookie', '')}).encode())

    def do_POST(self):
        self.rfile.read(int(self.headers.get('Content-Length', '0')))
        self.send_response(200)
        self.send_header('Set-Cookie', 'desktop_test=secret; Secure; HttpOnly; SameSite=Strict; Path=/api/v1; Max-Age=3600')
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        self.wfile.write(b'{"code":0,"data":{"user":{"id":1,"username":"demo","display_name":"Demo"}}}')


backend = ThreadingHTTPServer(('127.0.0.1', 0), Backend)
thread = threading.Thread(target=backend.serve_forever, daemon=True)
thread.start()
app = QApplication([])
app.setOrganizationName('TaskmgrSmoke')
app.setApplicationName('DesktopSmoke')
QSettings.setDefaultFormat(QSettings.Format.IniFormat)
QSettings.setPath(QSettings.Format.IniFormat, QSettings.Scope.UserScope, state.name)
base = f'http://127.0.0.1:{backend.server_port}'
settings = QSettings()
settings.setValue('connection/backend_url', base)
settings.sync()
assert QSettings().value('connection/backend_url') == base
persisted = subprocess.check_output([sys.executable, '-c',
    'from PySide6.QtCore import QCoreApplication, QSettings; '
    f'QSettings.setDefaultFormat(QSettings.Format.IniFormat); QSettings.setPath(QSettings.Format.IniFormat, QSettings.Scope.UserScope, {state.name!r}); '
    'app = QCoreApplication([]); app.setOrganizationName("TaskmgrSmoke"); '
    'app.setApplicationName("DesktopSmoke"); print(QSettings().value("connection/backend_url"))'], text=True)
assert persisted.strip() == base
window = Window(profile_root=Path(state.name) / 'profiles')
window.show()
expected_dpr = os.environ.get('TASKMGR_EXPECT_DPR')
if expected_dpr:
    assert abs(window.devicePixelRatioF() - float(expected_dpr)) < .05, (window.devicePixelRatioF(), expected_dpr)


def wait(predicate, timeout=20):
    end = time.monotonic() + timeout
    while time.monotonic() < end:
        app.processEvents()
        if predicate():
            return
        time.sleep(.01)
    raise AssertionError('Qt condition timed out')


def js(source):
    values = []
    window.view.page().runJavaScript(source, values.append)
    wait(lambda: bool(values))
    return values[0]


def settle_paint():
    loop = QEventLoop()
    QTimer.singleShot(400, loop.quit)
    loop.exec()


def loaded():
    wait(lambda: js('document.body && document.body.innerText.includes("用户名")'))
    assert js('document.documentElement.classList.contains("desktop-client")')


def fetch(path, options='{}'):
    js(f'window.smokeResult = null; fetch({json.dumps(path)}, {options}).then(r => r.json()).then(v => window.smokeResult = JSON.stringify(v)).catch(e => window.smokeResult = JSON.stringify({{error: String(e)}}))')
    wait(lambda: js('typeof window.smokeResult === "string"'))
    return json.loads(js('window.smokeResult'))


try:
    loaded()
    available = window.screen().availableGeometry()
    assert available.contains(window.frameGeometry()), (available, window.frameGeometry())
    for width, height in ((640, 480), (960, 640), (1280, 820)):
        window.resize(min(width, available.width() - 32), min(height, available.height() - 64))
        window.fit_to_screen()
        settle_paint()
        assert js('document.documentElement.scrollWidth <= window.innerWidth + 1')
        assert js('document.querySelector(".desktop-login .v-card").getBoundingClientRect().width <= 460')
        assert abs(js('window.innerWidth') - window.view.width()) <= 2
        if js('window.innerHeight') >= 660:
            assert js('document.documentElement.scrollHeight <= window.innerHeight + 1')
    # Oversized/offscreen saved windows must fit the current logical work area.
    window.resize(6000, 4000)
    window.move(-5000, -5000)
    window.fit_to_screen()
    settle_paint()
    assert available.contains(window.frameGeometry()), (available, window.frameGeometry())
    assert js('document.querySelector(".desktop-login .v-card").getBoundingClientRect().width <= 460')
    window.set_zoom(100)
    window.view.setFocus()
    QTest.keyClick(window.view, Qt.Key.Key_Plus, Qt.KeyboardModifier.ControlModifier)
    assert window.zoom_percent == 110
    wheel = QWheelEvent(QPointF(10, 10), QPointF(window.view.mapToGlobal(QPoint(10, 10))),
                        QPoint(), QPoint(0, 120), Qt.MouseButton.NoButton,
                        Qt.KeyboardModifier.ControlModifier, Qt.ScrollPhase.NoScrollPhase, False)
    QApplication.sendEvent(window.view.focusProxy() or window.view, wheel)
    assert window.zoom_percent == 120
    for _ in range(30):
        window.zoom(10)
    assert window.view.zoomFactor() == 2 and not window.zoom_in.isEnabled()
    for _ in range(30):
        window.zoom(-10)
    assert window.view.zoomFactor() == .5 and not window.zoom_out.isEnabled()
    window.set_zoom(100)
    settle_paint()
    assert js('document.querySelector(".desktop-login .v-card").getBoundingClientRect().width <= 460')
    screenshot = os.environ.get('TASKMGR_SMOKE_SCREENSHOT')
    if screenshot:
        settle_paint()
        window.grab().save(screenshot)
    dialog = ConnectionDialog('localhost:8080', window)
    dialog.validate()
    assert dialog.error.text()
    dialog.url.setText(base + '/api/v1/')
    dialog.validate()
    assert dialog.backend == base
    fetch('/api/v1/auth/browser/login', '{method:"POST",headers:{"Content-Type":"application/json"},body:"{}"}')
    assert fetch('/api/v1/check')['cookie'] == 'desktop_test=secret'
    assert 'secret' not in js('document.cookie')
    window.set_zoom(130)
    window.connect_backend(base + '/other')
    assert window.view.zoomFactor() == 1.3
    loaded()
    assert fetch('/api/v1/check')['cookie'] == ''
    window.connect_backend(base)
    assert window.view.zoomFactor() == 1.3
    wait(lambda: js('document.body && document.body.innerText.includes("告警单元")'))
    assert fetch('/api/v1/check')['cookie'] == 'desktop_test=secret'
    if screenshot:
        settle_paint()
        window.grab().save(str(Path(screenshot).with_stem(Path(screenshot).stem + '-dashboard')))
    window.close()
    app.processEvents()
    window = Window(profile_root=Path(state.name) / 'profiles')
    window.show()
    settle_paint()
    assert window.zoom_percent == 130 and window.view.zoomFactor() == 1.3
    assert available.contains(window.frameGeometry())
    window.view.setFocus()
    QTest.keyClick(window.view, Qt.Key.Key_0, Qt.KeyboardModifier.ControlModifier)
    assert window.zoom_percent == 100 and window.view.zoomFactor() == 1
    assert window.settings.value('view/zoom_percent') == 100
    window.configure = lambda: None
    print(f'Display DPR: {window.devicePixelRatioF():g}; logical work area: {available.width()}x{available.height()}')
    print('PASS: responsive window/login sizes, offscreen recovery, zoom shortcuts/wheel/bounds/persistence, Vue rendering, URL settings, HttpOnly cookies and backend isolation')
finally:
    window.close()
    app.processEvents()
    backend.shutdown()
    backend.server_close()
    thread.join()
    state.cleanup()
