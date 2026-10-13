"""Real Qt editor test against an isolated integration backend.

Set TASKMGR_TEST_BACKEND_URL and TASK_MANAGER_E2E_PASSWORD. Never uses user profiles.
"""
import json
import os
from pathlib import Path
import tempfile
import time

from PySide6.QtCore import QPoint, QSettings, Qt, QUrl
from PySide6.QtTest import QTest
from PySide6.QtWidgets import QApplication
from taskmgr_desktop.app import Window

base = os.environ['TASKMGR_TEST_BACKEND_URL']
password = os.environ['TASK_MANAGER_E2E_PASSWORD']
state = tempfile.TemporaryDirectory(ignore_cleanup_errors=True)
app = QApplication([])
app.setOrganizationName('TaskmgrNotesTest')
app.setApplicationName('NotesTest')
QSettings.setDefaultFormat(QSettings.Format.IniFormat)
QSettings.setPath(QSettings.Format.IniFormat, QSettings.Scope.UserScope, state.name)
QSettings().setValue('connection/backend_url', base)
window = Window(profile_root=Path(state.name) / 'profiles')
window.show()
clipboard = app.clipboard().text()
draft_id = None
content = '# Qt editor\n\n**Bold** 中文\n\n```javascript\nconst answer = 42;\n```\n\n$x^2$\n'


def wait(predicate, timeout=30):
    end = time.monotonic() + timeout
    while time.monotonic() < end:
        app.processEvents()
        if predicate():
            return
        QTest.qWait(20)
    raise AssertionError('Qt notes condition timed out')


def js(source):
    result = []
    window.view.page().runJavaScript(source, result.append)
    wait(lambda: bool(result))
    return result[0]


def api(path, method='GET', data=None):
    options = {'method': method, 'headers': {'Content-Type': 'application/json', 'X-Taskmgr-Browser': '1'}}
    if data is not None:
        options['body'] = json.dumps(data)
    js(f'window.testResult=null;fetch({json.dumps("/api/v1" + path)}, {json.dumps(options)})'
       '.then(async r=>({status:r.status,body:await r.text()})).then(r=>window.testResult=JSON.stringify(r))')
    wait(lambda: js('typeof window.testResult === "string"'))
    result = json.loads(js('window.testResult'))
    assert result['status'] < 300, result
    return json.loads(result['body']) if result['body'] else None


def click(text):
    rect = json.loads(js('JSON.stringify((()=>{const b=[...document.querySelectorAll("button")].find(b=>b.innerText.trim()==='
                         + json.dumps(text) + ');if(!b)return null;const r=b.getBoundingClientRect();return [r.x+r.width/2,r.y+r.height/2]})())'))
    assert rect, text
    QTest.mouseClick(window.view.focusProxy() or window.view, Qt.MouseButton.LeftButton,
                     pos=QPoint(round(rect[0]), round(rect[1])))
    QTest.qWait(250)


def paste(text):
    js('document.querySelector("#note-editor .cm-content").focus()')
    app.clipboard().setText(text)
    target = window.view.focusProxy() or window.view
    QTest.keyClick(target, Qt.Key.Key_A, Qt.KeyboardModifier.ControlModifier)
    QTest.keyClick(target, Qt.Key.Key_V, Qt.KeyboardModifier.ControlModifier)
    QTest.qWait(300)


def source():
    return js('[...document.querySelectorAll("#note-editor .cm-line")].map(e=>e.textContent).join("\\n")')


try:
    wait(lambda: js('document.body.innerText.includes("用户名")'))
    api('/auth/browser/login', 'POST', {'username': 'admin', 'password': password})
    window.view.setUrl(QUrl(window.server.origin + '/notes'))
    wait(lambda: js('document.body.innerText.includes("新建笔记")'))
    click('新建笔记')
    wait(lambda: js('!!document.querySelector("#note-editor .cm-content")'))
    paste(content)
    assert source() == content, repr(source())
    wait(lambda: js('!!document.querySelector("#note-editor .hljs-keyword") && !!document.querySelector("#note-editor .katex")'))
    assert js('(()=>{const svg=document.querySelector("#note-editor .md-editor-toolbar svg"),u=svg.querySelector("use");'
              'return !!document.querySelector(u.getAttribute("href")||u.getAttribute("xlink:href")).querySelector("path")'
              '&&getComputedStyle(svg).fill!=="none"})()')
    assert js('(()=>{const t=document.querySelector("#note-editor .hljs-keyword");return getComputedStyle(t).color!==getComputedStyle(t.closest("code")).color})()')
    click('全文复制')
    wait(lambda: app.clipboard().text().replace('\r\n', '\n') == content)
    # Save through the actual CodeMirror Ctrl+S binding.
    js('document.querySelector("#note-editor .cm-content").focus()')
    QTest.keyClick(window.view.focusProxy() or window.view, Qt.Key.Key_S, Qt.KeyboardModifier.ControlModifier)
    wait(lambda: js('document.body.innerText.includes("草稿已保存")'))
    drafts = api('/note-drafts')['data']
    draft = next(d for d in drafts if d['content'] == content)
    draft_id = draft['id']
    assert draft['title'] == ''
    click('关闭')
    # A new random loopback origin/profile still recovers server-backed drafts.
    window.connect_backend(base)
    wait(lambda: js('document.body.innerText.includes("告警单元")'))
    window.view.setUrl(QUrl(window.server.origin + '/notes'))
    wait(lambda: js('document.body.innerText.includes("草稿箱")'))
    js('[...document.querySelectorAll("button")].find(b=>b.innerText.startsWith("草稿箱")).click()')
    wait(lambda: js('document.body.innerText.includes("无标题草稿")'))
    click('恢复')
    wait(lambda: js('!!document.querySelector("#note-editor .cm-content")'))
    assert source() == content
    paste('- first')
    target = window.view.focusProxy() or window.view
    QTest.keyClick(target, Qt.Key.Key_End)
    QTest.keyClick(target, Qt.Key.Key_Return)
    QTest.keyClicks(target, 'second')
    assert source() == '- first\n- second', repr(source())
    QTest.keyClick(target, Qt.Key.Key_Tab)
    assert source() == '- first\n  - second', repr(source())
    QTest.keyClick(target, Qt.Key.Key_Backtab, Qt.KeyboardModifier.ShiftModifier)
    assert source() == '- first\n- second', repr(source())
    paste(content)
    QTest.qWait(500)
    window.grab().save(str(Path('build') / 'qt-notes-editor.png'))
    print('PASS: real Qt offline toolbar SVG/colors, Markdown/KaTeX highlight, full clipboard copy, Ctrl+S draft, reconnect recovery, list Enter and Tab/Shift+Tab')
finally:
    if draft_id:
        api('/note-drafts/' + draft_id, 'DELETE')
    app.clipboard().setText(clipboard)
    window.close()
    app.processEvents()
    # Chromium releases its profile files asynchronously on Windows.
    QTest.qWait(1500)
    state.cleanup()
