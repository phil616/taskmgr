# Qt 任务工作台

基于 PySide6 / Qt 6 WebEngine 的跨平台桌面端。内置本仓库 Vue 界面，直接连接 Go 后端，无需部署 Web 站点。仪表盘、计时单元、项目、待办坐标图、笔记、日程、预算、密钥、通知和设置沿用 Web 操作方式。

桌面布局采用深蓝导航栏、浅色卡片内容区、原生连接工具栏，支持窗口大小记忆、文件选择和下载保存。这里是 Qt WebEngine 混合客户端，业务页面复用 Vue，不是逐页重写的 QWidget 表单。

## 开发运行

需要 Python 3.10+、Node.js 20.19+（构建界面）和可用的 Go 后端。

```bash
cd ops-timer-desktop
python -m venv .venv
# Linux / macOS
source .venv/bin/activate
# Windows PowerShell 改用：.venv\Scripts\Activate.ps1
pip install -e .
python build_web.py
python -m taskmgr_desktop
```

`build_web.py` 执行 `npm ci` 和类型检查，生成独立的 `taskmgr_desktop/web/`，不修改 Web 的 `dist/`。构建时强制 API 使用 `/api/v1`，不会误用生产环境 `VITE_API_BASE_URL`。更新 Web 功能后重新运行此脚本。

Linux 需要桌面会话及 Qt WebEngine 的系统库（包括 libEGL、libGL、libnss3、libxkbcommon、libxcb-cursor）；缺少 Python venv 时安装发行版的 `python3-venv`。无需也不应关闭 Chromium 沙箱。

首次启动填写后端地址，例如 `http://localhost:8080` 或 `https://tasks.example.com`。也接受末尾 `/api/v1`，会自动规范化；支持反向代理路径前缀。仅支持 HTTP(S)，不接受 URL 中的账号密码、查询参数或片段。连接设置允许保存暂时离线的服务器，登录/业务请求失败会显示错误，联网后刷新重试。

## 配置与会话

- `QSettings` 自动保存后端 URL、窗口位置与尺寸。Linux 默认配置为 `~/.config/Taskmgr/TaskmgrDesktop.conf`，Windows 使用用户注册表，macOS 使用用户偏好设置。密码不写入配置。
- 每个规范化后端 URL 使用独立 Qt WebEngine profile，切换服务器不会把旧服务器的 Cookie 发给新服务器。切换前请保存尚未提交的编辑内容。
- 登录沿用后端浏览器会话；“记住我”持久 Cookie 由 Qt profile 保存，退出通过后端撤销会话。Qt 的 `AllowPersistentCookies` 也可能为崩溃恢复写入会话 Cookie，公共设备请主动退出。
- 本地 HTTP 服务仅绑定随机端口 `127.0.0.1`，每次启动生成访问密钥，由 Qt 请求拦截器注入；同时检查 Host 和 Origin。只转发 `/api/v1/` 与 `/health`，不跟随后端重定向。反向代理重写 Origin 为选定后端，不需要给后端追加桌面 CORS 来源。
- HTTPS 后端仍使用系统信任链验证证书。Cookie 进入私有 HTTP 回环源时移除 Secure、Domain 并重设 Path，保留 HttpOnly、SameSite、有效期。后端 HTTPS 传输和服务端 Cookie 配置不变。
- 外部 HTTP(S) 子资源被拦截，避免嵌入内容访问远端；因此笔记中的远程图片不会加载。普通外部链接交给系统浏览器打开。

快捷键：`Alt+Left` 返回、`Alt+Home` 首页、`Ctrl+R` 刷新、`Ctrl+,` 连接设置、`Ctrl+=` / `Ctrl+-` 缩放、`Ctrl+0` 重置。

## 分发

先构建界面，再构建 wheel；wheel 包含界面资源，安装后运行不需要 Node.js：

```bash
python build_web.py
pip install build
python -m build
# 在目标机器中（需 Python 及 Qt 所需系统库）
pip install dist/taskmgr_desktop-1.0.0-py3-none-any.whl
taskmgr-desktop
```

## 验证

```bash
python -m unittest discover -s tests -v
# Qt 集成测试需图形会话；Linux 无显示器时可使用 xvfb-run
python tests/qt_smoke.py
# 无显示器且不需要 GPU 的 CI 环境也可使用：
QT_QPA_PLATFORM=offscreen QTWEBENGINE_CHROMIUM_FLAGS=--disable-gpu python tests/qt_smoke.py
```
