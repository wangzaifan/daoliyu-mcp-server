#!/bin/sh
set -eu

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
  echo "usage: $0 PLUGIN_DIR [OUTPUT_ZIP]" >&2
  exit 2
fi

ROOT=$(CDPATH='' cd -- "$1" && pwd)
OUT=${2:-"$ROOT/../$(basename "$ROOT").zip"}
[ -f "$ROOT/plugin.json" ] || { echo "plugin.json is required" >&2; exit 1; }

python3 - "$ROOT/plugin.json" "$ROOT" <<'PY'
import json
import pathlib
import struct
import sys

manifest_path = pathlib.Path(sys.argv[1])
root = pathlib.Path(sys.argv[2])
manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
icon = manifest.get("icon")
if not icon:
    raise SystemExit("icon is required for third-party plugins")
if icon:
    icon_path = (root / icon).resolve()
    try:
        icon_path.relative_to(root.resolve())
    except ValueError:
        raise SystemExit("icon must stay inside the plugin directory")
    if icon_path.suffix.lower() != ".png" or not icon_path.is_file():
        raise SystemExit("icon must be a PNG file inside the plugin directory")
    if icon_path.stat().st_size > 512 * 1024:
        raise SystemExit("icon must be at most 512 KiB")
    data = icon_path.read_bytes()
    if len(data) < 24 or data[:8] != b"\x89PNG\r\n\x1a\n" or data[12:16] != b"IHDR":
        raise SystemExit("icon must be a valid PNG")
    width, height = struct.unpack(">II", data[16:24])
    if width != height or not 64 <= width <= 512:
        raise SystemExit("icon must be square and between 64x64 and 512x512")
if manifest.get("provider"):
    raise SystemExit("provider is reserved for built-in plugins")
PY

mkdir -p "$(dirname "$OUT")"
rm -f "$OUT"
python3 - "$ROOT" "$OUT" <<'PY'
import json
import pathlib
import sys
import zipfile

root = pathlib.Path(sys.argv[1])
out = pathlib.Path(sys.argv[2])
manifest = json.loads((root / "plugin.json").read_text(encoding="utf-8"))
files = [root / "plugin.json"]
entry = root / manifest.get("entry", "plugin")
if entry.is_file():
    files.append(entry)
icon = root / manifest["icon"] if manifest.get("icon") else None
if icon and icon.is_file():
    files.append(icon)
with zipfile.ZipFile(out, "w", compression=zipfile.ZIP_DEFLATED) as bundle:
    for path in files:
        bundle.write(path, path.relative_to(root).as_posix())
PY
unzip -t "$OUT" >/dev/null
printf 'built %s\n' "$OUT"
