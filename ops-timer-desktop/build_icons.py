"""Copy the frontend logo and render a multi-resolution Windows icon with Qt."""
from pathlib import Path
import shutil
import struct

from PySide6.QtCore import QByteArray, QBuffer, QIODevice, Qt
from PySide6.QtGui import QImage, QPainter
from PySide6.QtSvg import QSvgRenderer


def build_icons():
    root = Path(__file__).resolve().parent
    source = root.parent / 'ops-timer-frontend/public/favicon.svg'
    target = root / 'taskmgr_desktop'
    renderer = QSvgRenderer(str(source))
    if not renderer.isValid():
        raise RuntimeError(f'Invalid SVG logo: {source}')
    sizes = (16, 24, 32, 48, 64, 128, 256)
    images = []
    for size in sizes:
        image = QImage(size, size, QImage.Format.Format_ARGB32)
        image.fill(Qt.GlobalColor.transparent)
        painter = QPainter(image)
        renderer.render(painter)
        painter.end()
        data = QByteArray()
        buffer = QBuffer(data)
        buffer.open(QIODevice.OpenModeFlag.WriteOnly)
        if not image.save(buffer, 'PNG'):
            raise RuntimeError(f'Could not render {size}px logo')
        images.append(bytes(data))
    offset = 6 + 16 * len(sizes)
    entries = []
    for size, data in zip(sizes, images):
        dimension = size if size < 256 else 0
        entries.append(struct.pack('<BBBBHHII', dimension, dimension, 0, 0,
                                   1, 32, len(data), offset))
        offset += len(data)
    shutil.copy2(source, target / 'icon.svg')
    icon = target / 'icon.ico'
    icon.write_bytes(struct.pack('<HHH', 0, 1, len(sizes))
                     + b''.join(entries) + b''.join(images))
    return icon


if __name__ == '__main__':
    print(build_icons())
