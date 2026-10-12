"""Build a separate desktop bundle, ignoring production API URL overrides."""
import os
from pathlib import Path
import shutil
import subprocess

root = Path(__file__).resolve().parent
frontend = root.parent / 'ops-timer-frontend'
npm = shutil.which('npm')
if npm is None:
    raise SystemExit('请先安装 Node.js / npm')
env = dict(os.environ, VITE_API_BASE_URL='/api/v1', VITE_DESKTOP='true')
subprocess.run([npm, 'ci'], cwd=frontend, check=True)
subprocess.run([npm, 'run', 'build', '--', '--mode', 'desktop', '--outDir',
                str(root / 'taskmgr_desktop' / 'web'), '--emptyOutDir'],
               cwd=frontend, env=env, check=True)
