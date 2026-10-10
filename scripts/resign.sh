#!/bin/zsh
set -euo pipefail
# resign.sh — ad-hoc re-sign a copy so DYLD_INSERT_LIBRARIES is not blocked
# Usage: ./scripts/resign.sh [/Applications/Reqable.app] [/tmp/ReqableDYLD.app]
SRC="${1:-/Applications/Reqable.app}"
DST="${2:-/tmp/ReqableDYLD.app}"
echo "Copy $SRC -> $DST"
rm -rf "$DST"
cp -R "$SRC" "$DST"
# --deep re-signs the Flutter App.framework nested bundle too
codesign --force --deep -s - "$DST"
codesign -v "$DST" && echo "verify: ok"
codesign -dv "$DST/Contents/MacOS/Reqable" 2>&1 | grep -E "Identifier|flags"
echo "Done: $DST"
echo "Run: DYLD_INSERT_LIBRARIES=\$PWD/build/ReqableMacHack/libreqablemachack.dylib $DST/Contents/MacOS/Reqable"
