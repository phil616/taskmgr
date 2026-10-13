"""Native Qt shell with backend-specific browser profiles."""
import hashlib
import math
from pathlib import Path
import sys

import shiboken6

from PySide6.QtCore import QEvent, QLockFile, QPoint, QSettings, QSize, QStandardPaths, Qt, QTimer, QUrl
from PySide6.QtGui import QAction, QDesktopServices, QIcon, QKeySequence
from PySide6.QtWidgets import (
    QApplication, QDialog, QDialogButtonBox, QFileDialog, QFormLayout,
    QLabel, QLineEdit, QMainWindow, QMenu, QMessageBox, QSizePolicy, QToolBar, QToolButton,
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
    def __init__(self, *, profile_root=None):
        super().__init__()
        self.profile_root = Path(profile_root) if profile_root else None
        self.settings = QSettings()
        self.server = None
        self.profile = None
        self.view = None
        self.sessions = []  # Keep profiles alive until their pages have been destroyed.
        self.current_screen = None
        try:
            saved_zoom = float(self.settings.value('view/zoom_percent', 100))
            self.zoom_percent = max(50, min(200, round(saved_zoom))) if math.isfinite(saved_zoom) else 100
        except (ValueError, TypeError, OverflowError):
            self.zoom_percent = 100
        self.setWindowTitle('Task Manager · 任务工作台')
        available = self.screen().availableGeometry()
        self.setMinimumSize(min(640, available.width() - 32), min(480, available.height() - 64))
        self.resize(min(1280, round(available.width() * .88)), min(820, round(available.height() * .85)))
        self.move(available.center() - self.rect().center())
        geometry = self.settings.value('window/geometry')
        if geometry:
            self.restoreGeometry(geometry)
        self.setStyleSheet('''
            QMainWindow, QDialog { background: #f4f7fb; color: #24344b; }
            QToolBar { background: #14253e; border: none; spacing: 4px; padding: 6px; }
            QToolBar QLabel { color: #dce6f4; }
            QToolButton { color: #e9eff8; padding: 6px 8px; border-radius: 6px; }
            QToolButton:hover { background: #2b4260; }
            QPushButton { padding: 8px 14px; border-radius: 6px; background: #e2e9f2; color: #193655; }
            QPushButton:hover { background: #cfdeed; }
            QLineEdit { padding: 10px; background: white; color: #152640; border: 1px solid #becddd; border-radius: 6px; }
            QStatusBar { background: #edf2f8; color: #526780; }
        ''')
        toolbar = QToolBar('工作空间')
        self.toolbar = toolbar
        toolbar.setMovable(False)
        self.addToolBar(toolbar)
        brand = QLabel('  任务工作台  ')
        brand.setStyleSheet('font-weight: 600;')
        toolbar.addWidget(brand)
        self.add_action(toolbar, '后退', 'Alt+Left', lambda: self.view and self.view.back())
        self.add_action(toolbar, '刷新', 'Ctrl+R', lambda: self.view and self.view.reload())
        self.add_action(toolbar, '首页', 'Alt+Home', lambda: self.view and self.view.setUrl(QUrl(self.server.origin + '/dashboard')))
        spacer = QWidget()
        spacer.setSizePolicy(QSizePolicy.Policy.Expanding, QSizePolicy.Policy.Preferred)
        toolbar.addWidget(spacer)
        self.backend_label = QLabel('尚未连接')
        self.backend_label.setMaximumWidth(140)
        self.backend_label.setMinimumWidth(60)
        self.backend_label.setSizePolicy(QSizePolicy.Policy.Preferred, QSizePolicy.Policy.Preferred)
        toolbar.addWidget(self.backend_label)
        self.add_action(toolbar, '连接设置', 'Ctrl+,', self.configure)
        self.zoom_button = QToolButton()
        self.zoom_button.setPopupMode(QToolButton.ToolButtonPopupMode.InstantPopup)
        self.zoom_button.setToolTip('页面缩放（Ctrl+滚轮）；100% 遵循系统显示缩放')
        menu = QMenu(self.zoom_button)
        self.zoom_in = self.add_action(menu, '放大', ['Ctrl+=', 'Ctrl++'], lambda: self.zoom(10))
        self.zoom_out = self.add_action(menu, '缩小', 'Ctrl+-', lambda: self.zoom(-10))
        self.add_action(menu, '重置为 100%', 'Ctrl+0', lambda: self.set_zoom(100))
        self.zoom_button.setMenu(menu)
        toolbar.addWidget(self.zoom_button)
        self.set_zoom(self.zoom_percent)
        QApplication.instance().installEventFilter(self)
        placeholder = QLabel('欢迎使用任务工作台\n\n点击右上角「连接设置」，配置后端 URL 后开始使用。')
        placeholder.setAlignment(Qt.AlignmentFlag.AlignCenter)
        placeholder.setStyleSheet('font-size: 20px; color: #526780;')
        self.setCentralWidget(placeholder)
        backend = self.settings.value('connection/backend_url', '')
        if backend:
            self.connect_backend(backend)
        else:
            QTimer.singleShot(0, self.configure)

    def add_action(self, toolbar, text, shortcut, callback):
        action = QAction(text, self)
        action.setShortcuts([QKeySequence(key) for key in shortcut] if isinstance(shortcut, list)
                            else [QKeySequence(shortcut)])
        action.triggered.connect(callback)
        self.addAction(action)
        toolbar.addAction(action)
        return action

    def zoom(self, delta):
        self.set_zoom(self.zoom_percent + delta)

    def set_zoom(self, percent):
        self.zoom_percent = max(50, min(200, round(percent)))
        if self.view:
            self.view.setZoomFactor(self.zoom_percent / 100)
        self.settings.setValue('view/zoom_percent', self.zoom_percent)
        self.zoom_button.setText(f'{self.zoom_percent}%')
        self.zoom_in.setEnabled(self.zoom_percent < 200)
        self.zoom_out.setEnabled(self.zoom_percent > 50)

    def eventFilter(self, obj, event):
        if (event.type() == QEvent.Type.Wheel and event.modifiers() & Qt.KeyboardModifier.ControlModifier
                and self.view and isinstance(obj, QWidget)
                and (obj is self.view or self.view.isAncestorOf(obj)) and event.angleDelta().y()):
            self.zoom(10 if event.angleDelta().y() > 0 else -10)
            event.accept()
            return True
        return super().eventFilter(obj, event)

    def showEvent(self, event):
        super().showEvent(event)
        if self.current_screen is None:
            self.windowHandle().screenChanged.connect(self.bind_screen)
            self.bind_screen(self.windowHandle().screen())

    def bind_screen(self, screen):
        if self.current_screen:
            self.current_screen.availableGeometryChanged.disconnect(self.fit_to_screen)
            self.current_screen.logicalDotsPerInchChanged.disconnect(self.fit_to_screen)
        self.current_screen = screen
        if screen:
            screen.availableGeometryChanged.connect(self.fit_to_screen)
            screen.logicalDotsPerInchChanged.connect(self.fit_to_screen)
            QTimer.singleShot(0, self.fit_to_screen)

    def fit_to_screen(self, *_args):
        screen = self.current_screen or self.screen()
        available = screen.availableGeometry()
        frame = self.frameGeometry()
        extra_width = max(0, frame.width() - self.width())
        extra_height = max(0, frame.height() - self.height())
        limit = QSize(max(1, available.width() - extra_width), max(1, available.height() - extra_height))
        self.setMinimumSize(min(640, limit.width()), min(480, limit.height()))
        if self.isMaximized() or self.isFullScreen():
            return
        self.resize(self.size().boundedTo(limit))
        frame = self.frameGeometry()
        x = max(available.left(), min(frame.left(), available.right() - frame.width() + 1))
        y = max(available.top(), min(frame.top(), available.bottom() - frame.height() + 1))
        self.move(self.pos() + QPoint(x - frame.left(), y - frame.top()))

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
            if self.profile_root:
                profile.setPersistentStoragePath(str(self.profile_root / profile_id / 'storage'))
                profile.setCachePath(str(self.profile_root / profile_id / 'cache'))
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
        self.view.setZoomFactor(self.zoom_percent / 100)
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
        QApplication.instance().removeEventFilter(self)
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
    QApplication.setHighDpiScaleFactorRoundingPolicy(Qt.HighDpiScaleFactorRoundingPolicy.PassThrough)
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
