"""Run the GUI regressions at explicit display scale factors on this machine."""
import os
from pathlib import Path
import subprocess
import sys

from PySide6.QtWidgets import QApplication

root = Path(__file__).resolve().parents[1]
(root / 'dist').mkdir(exist_ok=True)
app = QApplication([])
native_scale = app.primaryScreen().devicePixelRatio()
for scale in (1, 1.5, 2):
    env = dict(os.environ)
    env.pop('QT_SCREEN_SCALE_FACTORS', None)
    env['QT_SCALE_FACTOR'] = str(scale / native_scale)
    env['TASKMGR_EXPECT_DPR'] = str(scale)
    env['TASKMGR_SMOKE_SCREENSHOT'] = str(root / 'dist' / f'sizing-scale-{round(scale * 100)}.png')
    print(f'Testing display scale {scale * 100:g}%', flush=True)
    subprocess.run([sys.executable, str(root / 'tests/qt_smoke.py')], cwd=root,
                   env=env, check=True, timeout=120)
