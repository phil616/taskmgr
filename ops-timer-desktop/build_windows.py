"""Build a Windows EXE with shared Qt DLLs and a portable ZIP."""
import argparse
from importlib.metadata import distributions, version
from pathlib import Path
import platform
import shutil
import subprocess
import sys

from build_icons import build_icons


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--skip-web', action='store_true', help='Reuse previously built web assets')
    parser.add_argument('--output-dir', type=Path, help='Build into a separate directory when an existing release is running')
    args = parser.parse_args()
    if sys.platform != 'win32':
        parser.error('Build the Windows release on Windows.')
    root = Path(__file__).resolve().parent
    output = args.output_dir.resolve() if args.output_dir else root / 'dist'
    existing_exe = output / 'TaskmgrDesktop/TaskmgrDesktop.exe'
    if existing_exe.exists():
        try:
            # Windows refuses writing a loaded EXE. Check before PyInstaller
            # removes any files from the existing distribution.
            with existing_exe.open('r+b'):
                pass
        except PermissionError:
            parser.error('Release EXE is running or not writable; use --output-dir dist/release.')
    if not args.skip_web:
        subprocess.run([sys.executable, str(root / 'build_web.py')], check=True)
    if not (root / 'taskmgr_desktop/web/index.html').is_file():
        parser.error('Missing web assets; run without --skip-web.')
    icon = build_icons()
    subprocess.run([
        sys.executable, '-m', 'PyInstaller', '--noconfirm', '--clean',
        '--onedir', '--windowed', '--noupx', '--name', 'TaskmgrDesktop',
        '--icon', str(icon),
        '--distpath', str(output), '--workpath', str(root / 'build/pyinstaller'),
        '--specpath', str(root / 'build'), '--paths', str(root),
        '--add-data', f'{root / "taskmgr_desktop/web"}:taskmgr_desktop/web',
        '--add-data', f'{root / "taskmgr_desktop/icon.svg"}:taskmgr_desktop',
        str(root / 'desktop_entry.py'),
    ], cwd=root, check=True)
    package_bundle(root, output)


def package_bundle(root, output=None):
    """Validate the runtime directory and create the distributable ZIP."""
    output = output or root / 'dist'
    bundle = output / 'TaskmgrDesktop'
    for filename in ('Qt6Core.dll', 'Qt6Widgets.dll', 'Qt6WebEngineCore.dll',
                     'qwindows.dll', 'QtWebEngineProcess.exe'):
        if not any(bundle.rglob(filename)):
            raise RuntimeError(f'Missing runtime dependency: {filename}')
    shutil.copy2(root / 'README.md', bundle / 'README.md')
    # Preserve the licenses shipped by Qt/PySide in the release.
    for metadata in distributions():
        package = metadata.metadata['Name']
        if not package.lower().replace('-', '_').startswith(('pyside6', 'shiboken6')):
            continue
        for entry in metadata.files or []:
            if 'licenses' in entry.parts:
                source = Path(metadata.locate_file(entry))
                relative = Path(*entry.parts[entry.parts.index('licenses') + 1:])
                destination = bundle / 'licenses' / package / relative
                destination.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(source, destination)
    archive = output / f'TaskmgrDesktop-{version("taskmgr-desktop")}-windows-{platform.machine().lower()}-portable'
    shutil.make_archive(str(archive), 'zip', root_dir=bundle.parent, base_dir=bundle.name)
    print(f'EXE: {bundle / "TaskmgrDesktop.exe"}')
    print(f'ZIP: {archive}.zip')


if __name__ == '__main__':
    main()
