#!/bin/sh
set -eu
ROOT=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
VERSION=${VERSION:-0.3.2}
OUT="$ROOT/dist/$VERSION/plugins"
mkdir -p "$OUT"
SISYPHUS_PACKAGE="$OUT/sisyphus-0.6.7.zip"
rm -f "$SISYPHUS_PACKAGE"
(cd "$ROOT/plugins/sisyphus" && zip -q "$SISYPHUS_PACKAGE" plugin.json icon.png LICENSE)
printf 'built %s\n' "$SISYPHUS_PACKAGE"
