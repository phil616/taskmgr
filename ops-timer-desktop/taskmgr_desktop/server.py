"""Private loopback origin for bundled assets and same-origin backend requests.

Every request needs a per-process capability added by the Qt interceptor. The
backend is immutable for the lifetime of a server, so in-flight requests cannot
accidentally be sent to a newly selected backend.
"""
import http.client
import json
import mimetypes
import secrets
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import unquote, urlsplit, urlunsplit

MAX_BODY = 64 * 1024 * 1024


def normalize_backend(value: str) -> str:
    value = value.strip().rstrip('/')
    try:
        url = urlsplit(value)
        port = url.port
    except ValueError as exc:
        raise ValueError('后端 URL 格式或端口无效') from exc
    if (url.scheme not in ('http', 'https') or not url.hostname
            or url.username is not None or url.password is not None
            or url.query or url.fragment or any(c.isspace() for c in value)
            or '\\' in value or (port is not None and port == 0)):
        raise ValueError('请输入完整的 http:// 或 https:// 地址，不含账号、查询参数或片段')
    path = url.path.rstrip('/')
    if path.endswith('/api/v1'):
        path = path[:-7]
    if any(unquote(part) in ('.', '..') for part in path.split('/')):
        raise ValueError('后端路径不能包含 . 或 ..')
    return urlunsplit((url.scheme, url.netloc.lower(), path, '', ''))


class DesktopServer(ThreadingHTTPServer):
    daemon_threads = True

    def __init__(self, backend: str, assets: Path):
        self.backend = normalize_backend(backend)
        self.assets = assets.resolve()
        self.key = secrets.token_urlsafe(32)
        super().__init__(('127.0.0.1', 0), Handler)
        self.origin = f'http://127.0.0.1:{self.server_port}'
        self.worker = threading.Thread(target=self.serve_forever, daemon=True)
        self.worker.start()

    def close(self):
        self.shutdown()
        self.server_close()
        self.worker.join(timeout=2)


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *_args):
        pass  # Never log URLs, cookies, credentials or imported backup content.

    def reply(self, status, body=b'', content_type='application/json', headers=()):
        self.send_response(status)
        self.send_header('Content-Type', content_type)
        self.send_header('Content-Length', str(len(body)))
        self.send_header('Cache-Control', 'no-store')
        self.send_header('X-Content-Type-Options', 'nosniff')
        for name, value in headers:
            self.send_header(name, value)
        self.end_headers()
        if self.command != 'HEAD':
            self.wfile.write(body)

    def handle_request(self):
        server = self.server
        if (self.headers.get('Host') != urlsplit(server.origin).netloc
                or not secrets.compare_digest(self.headers.get('X-Taskmgr-Desktop-Key', ''), server.key)
                or self.headers.get('Origin', server.origin) != server.origin):
            self.reply(403)
            return
        url = urlsplit(self.path)
        if url.scheme or url.netloc or not url.path.startswith('/'):
            self.reply(400)
            return
        if url.path.startswith('/api/v1/') or url.path == '/health':
            self.proxy()
            return
        if self.command not in ('GET', 'HEAD'):
            self.reply(405)
            return
        path = (server.assets / unquote(url.path).lstrip('/')).resolve()
        if not path.is_relative_to(server.assets):
            self.reply(403)
            return
        if not path.is_file():
            if Path(url.path).suffix:
                self.reply(404)
                return
            path = server.assets / 'index.html'
        if not path.is_file():
            self.reply(503, b'Build desktop web assets first.', 'text/plain')
            return
        self.reply(200, path.read_bytes(), mimetypes.guess_type(path)[0] or 'application/octet-stream')

    def proxy(self):
        try:
            size = int(self.headers.get('Content-Length', '0'))
        except ValueError:
            self.reply(400)
            return
        if self.headers.get('Transfer-Encoding') or size < 0 or size > MAX_BODY:
            self.reply(413)
            return
        backend = urlsplit(self.server.backend)
        connection_type = http.client.HTTPSConnection if backend.scheme == 'https' else http.client.HTTPConnection
        conn = connection_type(backend.hostname, backend.port, timeout=20)
        headers = {name: self.headers[name] for name in (
            'Content-Type', 'Cookie', 'Authorization', 'X-API-Token', 'Accept'
        ) if name in self.headers}
        headers.update({'Origin': f'{backend.scheme}://{backend.netloc}',
                        'X-Taskmgr-Browser': '1', 'Accept-Encoding': 'identity'})
        try:
            self.connection.settimeout(20)
            body = self.rfile.read(size) if size else None
            conn.request(self.command, backend.path + self.path, body=body, headers=headers)
            response = conn.getresponse()
            # Do not follow redirects: credentials must remain on the chosen backend.
            if 300 <= response.status < 400:
                self.reply(502, json.dumps({'message': '后端返回重定向，请填写最终后端 URL'}).encode())
                return
            forwarded = []
            for name, value in response.getheaders():
                if name.lower() in ('content-disposition', 'set-cookie'):
                    if name.lower() == 'set-cookie':
                        # The private local origin is HTTP. TLS verification remains
                        # enabled on HTTPS backend connections. Preserve HttpOnly and
                        # SameSite; cookies stay inside this backend's Qt profile.
                        parts = [part.strip() for part in value.split(';')]
                        parts = [p for p in parts if p.lower() != 'secure'
                                 and not p.lower().startswith(('domain=', 'path='))]
                        value = '; '.join(parts + ['Path=/'])
                    forwarded.append((name, value))
            self.reply(response.status, response.read(),
                       response.getheader('Content-Type', 'application/json'), forwarded)
        except (OSError, http.client.HTTPException):
            self.reply(502, json.dumps({'message': '无法连接后端，请检查后端 URL、网络及证书'}).encode())
        finally:
            conn.close()

    do_GET = do_POST = do_PUT = do_PATCH = do_DELETE = do_HEAD = do_OPTIONS = handle_request
