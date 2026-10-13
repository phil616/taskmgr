"""Absolute imports for the frozen Windows executable."""
import sys

if '--smoke-test' in sys.argv:
    from taskmgr_desktop.smoke import main
else:
    from taskmgr_desktop.app import main

main()
