"""Extract a release to a fresh path and test without development tools on PATH."""
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import zipfile


archive = Path(sys.argv[1]).resolve()
report = archive.parent / 'packaged-smoke.json'
report.unlink(missing_ok=True)
with tempfile.TemporaryDirectory(prefix='taskmgr-zip-') as directory:
    extracted = Path(directory) / '解压 验证'
    with zipfile.ZipFile(archive) as release:
        release.extractall(extracted)
    exe = extracted / 'TaskmgrDesktop/TaskmgrDesktop.exe'
    env = {key: value for key, value in os.environ.items()
           if not key.upper().startswith(('PYTHON', 'QT', 'QML', 'VIRTUAL_ENV'))}
    env['PATH'] = str(Path(os.environ['SystemRoot']) / 'System32')
    completed = subprocess.run([str(exe), '--smoke-test', str(report)],
                               cwd=extracted, env=env, timeout=75)
    result = json.loads(report.read_text(encoding='utf-8'))
    print(json.dumps(result, ensure_ascii=False, indent=2))
    if completed.returncode or not result['passed']:
        raise SystemExit('Packaged EXE smoke test failed')
print('PASS: ZIP extracted to a path with spaces/Chinese characters; EXE runs with no Python/Qt on PATH')
