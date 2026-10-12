import http.client
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
from pathlib import Path
import tempfile
import threading
import unittest

from taskmgr_desktop.server import DesktopServer, normalize_backend


class Backend(BaseHTTPRequestHandler):
    def log_message(self, *_args):
        pass

    def do_GET(self):
        size = int(self.headers.get('Content-Length', '0'))
        data = self.rfile.read(size).decode() if size else ''
        self.server.requests.append((self.path, dict(self.headers), data))
        if self.path.endswith('/redirect'):
            self.send_response(302)
            self.send_header('Location', 'https://example.com')
            self.end_headers()
            return
        self.send_response(401 if self.path.endswith('/denied') else 200)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Set-Cookie', 'session=test; Path=/api/v1; Secure; HttpOnly; SameSite=Strict; Max-Age=3600')
        self.send_header('Content-Disposition', 'attachment; filename="backup.json"')
        self.end_headers()
        self.wfile.write(json.dumps({'path': self.path, 'body': data}).encode())

    do_POST = do_PATCH = do_DELETE = do_GET


class ServerTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.assets = Path(self.temp.name)
        (self.assets / 'index.html').write_text('<h1>Desktop</h1>')
        (self.assets / 'app.js').write_text('console.log(1)')
        self.backend = ThreadingHTTPServer(('127.0.0.1', 0), Backend)
        self.backend.requests = []
        self.thread = threading.Thread(target=self.backend.serve_forever, daemon=True)
        self.thread.start()
        self.server = DesktopServer(f'http://127.0.0.1:{self.backend.server_port}/prefix', self.assets)

    def tearDown(self):
        self.server.close()
        self.backend.shutdown()
        self.backend.server_close()
        self.thread.join()
        self.temp.cleanup()

    def request(self, path, method='GET', body=None, headers=None, authenticated=True):
        conn = http.client.HTTPConnection('127.0.0.1', self.server.server_port)
        request_headers = {'X-Taskmgr-Desktop-Key': self.server.key} if authenticated else {}
        request_headers.update(headers or {})
        conn.request(method, path, body=body, headers=request_headers)
        response = conn.getresponse()
        result = response.status, dict(response.getheaders()), response.read()
        conn.close()
        return result

    def test_spa_and_assets(self):
        self.assertIn(b'Desktop', self.request('/todos')[2])
        self.assertEqual(self.request('/app.js')[0], 200)
        self.assertEqual(self.request('/missing.js')[0], 404)
        self.assertEqual(self.request('/%2e%2e/private')[0], 403)

    def test_private_origin(self):
        self.assertEqual(self.request('/api/v1/todos', authenticated=False)[0], 403)
        self.assertEqual(self.request('/api/v1/todos', headers={'Origin': 'https://evil.example'})[0], 403)
        self.assertEqual(self.request('/', headers={'Host': 'evil.example'})[0], 403)
        self.assertFalse(self.backend.requests)

    def test_proxy_preserves_body_query_cookie_and_errors(self):
        status, headers, data = self.request('/api/v1/todos?page=2', 'POST', '{"title":"test"}',
            {'Content-Type': 'application/json', 'Cookie': 'session=abc', 'Origin': self.server.origin})
        self.assertEqual(status, 200)
        self.assertEqual(json.loads(data)['path'], '/prefix/api/v1/todos?page=2')
        path, sent, body = self.backend.requests[-1]
        self.assertEqual(body, '{"title":"test"}')
        self.assertEqual(sent['Cookie'], 'session=abc')
        self.assertEqual(sent['Origin'], f'http://127.0.0.1:{self.backend.server_port}')
        self.assertNotIn('X-Taskmgr-Desktop-Key', sent)
        self.assertIn('HttpOnly', headers['Set-Cookie'])
        self.assertIn('Max-Age=3600', headers['Set-Cookie'])
        self.assertNotIn('Secure', headers['Set-Cookie'])
        self.assertIn('attachment', headers['Content-Disposition'])
        self.assertEqual(self.request('/api/v1/denied')[0], 401)

    def test_redirects_and_large_uploads_rejected(self):
        self.assertEqual(self.request('/api/v1/redirect')[0], 502)
        self.assertEqual(self.request('/api/v1/backup/import', 'POST', headers={'Content-Length': str(65 * 1024 * 1024)})[0], 413)

    def test_unavailable_backend_is_actionable(self):
        self.backend.shutdown()
        self.backend.server_close()
        status, _, body = self.request('/health')
        self.assertEqual(status, 502)
        self.assertIn('后端 URL', json.loads(body)['message'])


class URLTests(unittest.TestCase):
    def test_normalization(self):
        self.assertEqual(normalize_backend(' HTTPS://Example.COM/api/v1/ '), 'https://example.com')
        self.assertEqual(normalize_backend('http://localhost:8080/prefix/api/v1'), 'http://localhost:8080/prefix')
        self.assertEqual(normalize_backend('http://[::1]:8080'), 'http://[::1]:8080')

    def test_invalid_urls(self):
        for value in ('', 'localhost:8080', 'file:///tmp', 'https://user:pass@host',
                      'http://host?x=1', 'http://host#x', 'http://host:abc',
                      'http://host:0', 'http://host/a/../b', 'http://host/a\\b', 'http://ho st'):
            with self.subTest(value=value), self.assertRaises(ValueError):
                normalize_backend(value)


if __name__ == '__main__':
    unittest.main()
