# Qt 任务工作台

DONOTCI

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

窗口使用 Qt 的逻辑像素和系统 DPI 缩放。首次启动按屏幕可用区域设置尺寸（最大
1280×820），恢复窗口或切换屏幕时会收回超出屏幕的窗口；支持缩小至 640×480，
更小的屏幕会继续限制最小尺寸。工具栏使用紧凑布局，登录表单保持适宜宽度，小窗口可纵向滚动。

右上角百分比菜单控制页面缩放，范围 50%–200%，每次调整 10 个百分点，也支持
`Ctrl+滚轮` 和 `Ctrl++`。100% 遵循系统显示缩放，不额外乘以 DPI；调整窗口大小
不会重置页面缩放。缩放比例独立保存，切换后端、刷新和重启后继续使用；`Ctrl+0` 恢复 100%。

## 分发

### Windows ZIP 免安装版（推荐）

在 Windows 构建，使用 Python 3.10+ 和 Node.js。PowerShell：

```powershell
cd ops-timer-desktop
python -m venv .venv
.venv\Scripts\python.exe -m pip install -e ".[packaging]"
.venv\Scripts\python.exe build_windows.py
```

产物为 `dist/TaskmgrDesktop-1.0.0-windows-amd64-portable.zip`。完整解压后双击
`TaskmgrDesktop/TaskmgrDesktop.exe` 启动，无需安装 Python、Node.js 或 Qt。
EXE 使用随目录分发的动态 Qt DLL，`_internal/` 包含 Python 运行时、Qt 插件、
WebEngine 子进程及界面资源。请保留整个目录，不要单独复制 EXE。
这里的 portable 指免安装运行；连接设置与会话仍保存在当前 Windows 用户目录/注册表中。
此发布方式提供 ZIP，不生成 MSI。

构建脚本默认重新构建前端（含类型检查）；仅在已构建界面时可以加 `--skip-web`。
若旧版 EXE 正在运行，使用 `--output-dir dist/release` 在独立目录构建，避免覆盖被占用的 DLL。
Qt WebEngine 体积较大，ZIP 包含完整浏览器运行时。

验证实际打包后的 EXE（使用临时配置和模拟后端，不需要真实服务器）：

```powershell
$report = Join-Path $PWD 'dist\packaged-smoke.json'
$process = Start-Process -FilePath '.\dist\TaskmgrDesktop\TaskmgrDesktop.exe' -ArgumentList '--smoke-test', $report -PassThru -Wait -WindowStyle Hidden
Get-Content $report
if ($process.ExitCode -ne 0) { throw '打包 EXE 验证失败' }
```

也可验证 ZIP 解压后的运行：此测试使用含中文和空格的新目录，并从子进程 PATH
移除开发工具，检查 Qt DLL 实际加载路径、登录页和本地代理。结果和截图保存在 `dist/packaged-smoke.json`、`dist/packaged-smoke.png`。

```powershell
.venv\Scripts\python.exe tests\verify_windows_zip.py dist\TaskmgrDesktop-1.0.0-windows-amd64-portable.zip
```

### Python wheel（开发者使用）

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

笔记编辑器的图标、代码高亮、格式化、数学公式与图表资源均在本地打包。
工具栏保存按钮和 Ctrl+S 保存草稿，草稿箱与 Web 共用后端数据；全文复制的是完整 Markdown 原文。
图标来源为 md-editor-v3 4.21.3 发布的 `https://at.alicdn.com/t/c/font_2605852_cmafimm6hot.js`，
已保存在前端 `public/vendor/md-editor-icons.js`，运行时无需访问该地址。

```bash
python -m unittest discover -s tests -v
# Qt 集成测试需图形会话；Linux 无显示器时可使用 xvfb-run
python tests/qt_smoke.py
# 本机模拟 100%、150%、200% 显示缩放，检查窗口/登录表单/页面缩放
python tests/qt_scale_smoke.py
# 无显示器且不需要 GPU 的 CI 环境也可使用：
QT_QPA_PLATFORM=offscreen QTWEBENGINE_CHROMIUM_FLAGS=--disable-gpu python tests/qt_smoke.py
```

针对已启动的隔离测试后端，验证实际 Qt 编辑器的图标颜色、代码/公式渲染、全文复制、
Ctrl+S 保存草稿、重新连接恢复，以及列表续写和 Tab / Shift+Tab（使用临时 Qt 配置）：

```powershell
$env:TASKMGR_TEST_BACKEND_URL = 'http://127.0.0.1:55440'
$env:TASK_MANAGER_E2E_PASSWORD = '隔离测试后端的管理员密码'
.venv\Scripts\python.exe tests\qt_notes_smoke.py
```
