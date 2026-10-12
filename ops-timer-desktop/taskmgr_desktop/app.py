"""Native Qt shell with backend-specific browser profiles."""
import hashlib
from pathlib import Path
import sys

import shiboken6

from PySide6.QtCore import QLockFile, QSettings, QStandardPaths, Qt, QUrl
from PySide6.QtGui import QAction, QDesktopServices, QIcon, QKeySequence
from PySide6.QtWidgets import (
    QApplication, QDialog, QDialogButtonBox, QFileDialog, QFormLayout,
    QLabel, QLineEdit, QMainWindow, QMessageBox, QToolBar,
    QVBoxLayout, QWidget,
)
from PySide6.QtWebEngineCore import QWebEnginePage, QWebEngineProfile, QWebEngineUrlRequestInterceptor
from PySide6.QtWebEngineWidgets import QWebEngineView

from .server import DesktopServer, normalize_backend


class ConnectionDialog(QDialog):
    def __init__(self, backend, parent=None):
        super().__init__(parent)
        self.setWindowTitle('连接工作空间')
        self.setMinimumWidth(520)
        layout = QVBoxLayout(self)
        title = QLabel('连接你的任务管理后端')
        title.setStyleSheet('font-size: 22px; font-weight: 600; color: #152640; margin: 12px 0;')
        layout.addWidget(title)
        help_text = QLabel('地址会自动保存，下次启动无需重复输入。\n切换后端将重新加载工作空间，请先保存正在编辑的内容。')
        help_text.setWordWrap(True)
        layout.addWidget(help_text)
        form = QFormLayout()
        self.url = QLineEdit(backend)
        self.url.setPlaceholderText('https://tasks.example.com 或 http://localhost:8080')
        self.url.setClearButtonEnabled(True)
        form.addRow('后端 URL', self.url)
        layout.addLayout(form)
        self.error = QLabel()
        self.error.setStyleSheet('color: #c0392b;')
        self.error.setWordWrap(True)
        layout.addWidget(self.error)
        buttons = QDialogButtonBox(QDialogButtonBox.StandardButton.Save | QDialogButtonBox.StandardButton.Cancel)
        buttons.button(QDialogButtonBox.StandardButton.Save).setText('保存并连接')
        buttons.button(QDialogButtonBox.StandardButton.Cancel).setText('取消')
        buttons.accepted.connect(self.validate)
        buttons.rejected.connect(self.reject)
        layout.addWidget(buttons)

    def validate(self):
        try:
            self.backend = normalize_backend(self.url.text())
        except ValueError as exc:
            self.error.setText(str(exc))
            return
        self.accept()


class Interceptor(QWebEngineUrlRequestInterceptor):
    def __init__(self, server, parent):
        super().__init__(parent)
        self.origin = server.origin
        self.key = server.key.encode()

    def interceptRequest(self, info):
        url = info.requestUrl()
        origin = f'{url.scheme()}://{url.host()}:{url.port()}'
        if origin == self.origin:
            info.setHttpHeader(b'X-Taskmgr-Desktop-Key', self.key)
        elif url.scheme() in ('http', 'https'):
            # Keep embedded remote content from receiving workspace data.
            info.block(True)


class Page(QWebEnginePage):
    def __init__(self, profile, origin, parent):
        super().__init__(profile, parent)
        self.origin = origin
        self.newWindowRequested.connect(self.open_external_window)

    def open_external_window(self, request):
        url = request.requestedUrl()
        if request.isUserInitiated() and url.scheme() in ('http', 'https', 'mailto'):
            QDesktopServices.openUrl(url)

    def acceptNavigationRequest(self, url, navigation_type, is_main_frame):
        if url.toString().startswith(self.origin + '/') or url.toString() == 'about:blank':
            return True
        if (is_main_frame and navigation_type == QWebEnginePage.NavigationType.NavigationTypeLinkClicked
                and url.scheme() in ('http', 'https', 'mailto')):
            QDesktopServices.openUrl(url)
        return False


class Window(QMainWindow):
    def __init__(self):
        super().__init__()
        self.settings = QSettings()
        self.server = None
        self.profile = None
        self.view = None
        self.sessions = []  # Keep profiles alive until their pages have been destroyed.
        self.setWindowTitle('Task Manager · 任务工作台')
        self.resize(1440, 960)
        self.setMinimumSize(960, 680)
        geometry = self.settings.value('window/geometry')
        if geometry:
            self.restoreGeometry(geometry)
        self.setStyleSheet('''
            QMainWindow, QDialog { background: #f4f7fb; color: #24344b; }
            QToolBar { background: #14253e; border: none; spacing: 10px; padding: 10px; }
            QToolBar QLabel { color: #dce6f4; }
            QToolButton { color: #e9eff8; padding: 8px 12px; border-radius: 6px; }
            QToolButton:hover { background: #2b4260; }
            QPushButton { padding: 8px 14px; border-radius: 6px; background: #e2e9f2; color: #193655; }
            QPushButton:hover { background: #cfdeed; }
            QLineEdit { padding: 10px; background: white; color: #152640; border: 1px solid #becddd; border-radius: 6px; }
            QStatusBar { background: #edf2f8; color: #526780; }
        ''')
        toolbar = QToolBar('工作空间')
        toolbar.setMovable(False)
        self.addToolBar(toolbar)
        brand = QLabel('  ◈  TASK MANAGER  /  任务工作台  ')
        brand.setStyleSheet('font-size: 15px; font-weight: 600;')
        toolbar.addWidget(brand)
        self.add_action(toolbar, '后退', 'Alt+Left', lambda: self.view and self.view.back())
        self.add_action(toolbar, '刷新', 'Ctrl+R', lambda: self.view and self.view.reload())
        self.add_action(toolbar, '首页', 'Alt+Home', lambda: self.view and self.view.setUrl(QUrl(self.server.origin + '/dashboard')))
        spacer = QWidget()
        from PySide6.QtWidgets import QSizePolicy
        spacer.setSizePolicy(QSizePolicy.Policy.Expanding, QSizePolicy.Policy.Preferred)
        toolbar.addWidget(spacer)
        self.backend_label = QLabel('尚未连接')
        toolbar.addWidget(self.backend_label)
        self.add_action(toolbar, '连接设置', 'Ctrl+,', self.configure)
        self.add_action(toolbar, '放大', 'Ctrl+=', lambda: self.zoom(0.1))
        self.add_action(toolbar, '缩小', 'Ctrl+-', lambda: self.zoom(-0.1))
        self.add_action(toolbar, '重置缩放', 'Ctrl+0', lambda: self.view and self.view.setZoomFactor(1))
        placeholder = QLabel('欢迎使用任务工作台\n\n点击右上角「连接设置」，配置后端 URL 后开始使用。')
        placeholder.setAlignment(Qt.AlignmentFlag.AlignCenter)
        placeholder.setStyleSheet('font-size: 20px; color: #526780;')
        self.setCentralWidget(placeholder)
        backend = self.settings.value('connection/backend_url', '')
        if backend:
            self.connect_backend(backend)
        else:
            from PySide6.QtCore import QTimer
            QTimer.singleShot(0, self.configure)

    def add_action(self, toolbar, text, shortcut, callback):
        action = QAction(text, self)
        action.setShortcut(QKeySequence(shortcut))
        action.triggered.connect(callback)
        toolbar.addAction(action)

    def zoom(self, delta):
        if self.view:
            self.view.setZoomFactor(max(0.5, min(2, self.view.zoomFactor() + delta)))

    def configure(self):
        dialog = ConnectionDialog(self.settings.value('connection/backend_url', 'http://localhost:8080'), self)
        if dialog.exec() == QDialog.DialogCode.Accepted:
            self.settings.setValue('connection/backend_url', dialog.backend)
            self.settings.sync()
            if self.settings.status() != QSettings.Status.NoError:
                QMessageBox.warning(self, '保存失败', '无法写入配置文件，请检查当前用户配置目录权限。')
            self.connect_backend(dialog.backend)

    def connect_backend(self, backend):
        assets = Path(__file__).parent / 'web'
        if not (assets / 'index.html').is_file():
            QMessageBox.warning(self, '缺少界面资源', '请先在 ops-timer-desktop 目录执行：\npython build_web.py')
            return
        try:
            server = DesktopServer(backend, assets)
        except (ValueError, OSError) as exc:
            QMessageBox.warning(self, '连接配置无效', str(exc))
            return
        if self.view:
            self.view.stop()
            shiboken6.delete(self.takeCentralWidget())
        if self.server:
            self.server.close()
        self.server = server
        # A distinct profile per backend prevents cookies leaking across servers.
        profile_id = hashlib.sha256(server.backend.encode()).hexdigest()
        profile = next((p for p in self.sessions if p.storageName() == profile_id), None)
        if profile is None:
            profile = QWebEngineProfile(profile_id, self)
            profile.setPersistentCookiesPolicy(QWebEngineProfile.PersistentCookiesPolicy.AllowPersistentCookies)
            self.sessions.append(profile)
            profile.downloadRequested.connect(self.download)
        interceptor = Interceptor(server, profile)
        profile.setUrlRequestInterceptor(interceptor)
        profile._interceptor = interceptor
        self.profile = profile
        self.view = QWebEngineView(self)
        page = Page(profile, server.origin, self.view)
        self.view.setPage(page)
        self.view.loadFinished.connect(self.loaded)
        self.setCentralWidget(self.view)
        self.backend_label.setText(QUrl(server.backend).host())
        self.backend_label.setToolTip(server.backend)
        self.statusBar().showMessage(f'后端：{server.backend}  ·  正在加载工作空间…')
        self.view.setUrl(QUrl(server.origin + '/'))

    def loaded(self, success):
        self.statusBar().showMessage(
            f'后端：{self.server.backend}  ·  ' + ('界面已就绪' if success else '界面加载失败，请刷新或检查连接设置'))

    def download(self, request):
        directory = QStandardPaths.writableLocation(QStandardPaths.StandardLocation.DownloadLocation)
        path, _ = QFileDialog.getSaveFileName(self, '保存文件', str(Path(directory) / request.downloadFileName()))
        if path:
            request.setDownloadDirectory(str(Path(path).parent))
            request.setDownloadFileName(Path(path).name)
            request.accept()
        else:
            request.cancel()

    def closeEvent(self, event):
        self.settings.setValue('window/geometry', self.saveGeometry())
        self.settings.sync()
        if self.view:
            self.view.stop()
            # Destroy pages before profiles (WebEngine requires this ordering).
            shiboken6.delete(self.takeCentralWidget())
        if self.server:
            self.server.close()
        for profile in self.sessions:
            shiboken6.delete(profile)
        event.accept()


def main():
    app = QApplication(sys.argv)
    app.setOrganizationName('Taskmgr')
    app.setApplicationName('TaskmgrDesktop')
    app.setApplicationDisplayName('任务工作台')
    app.setWindowIcon(QIcon(str(Path(__file__).parent / 'icon.svg')))
    app.setStyle('Fusion')
    data_dir = Path(QStandardPaths.writableLocation(QStandardPaths.StandardLocation.AppLocalDataLocation))
    data_dir.mkdir(parents=True, exist_ok=True)
    lock = QLockFile(str(data_dir / 'desktop.lock'))
    if not lock.tryLock(100):
        QMessageBox.information(None, '任务工作台', '任务工作台已经运行，或配置目录不可写。请检查已有窗口与目录权限。')
        return
    window = Window()
    window.show()
    sys.exit(app.exec())
