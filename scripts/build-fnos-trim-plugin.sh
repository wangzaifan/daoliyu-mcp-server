#!/bin/sh
set -eu

ROOT=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
SKILL_DIR=${FNOS_SKILL_DIR:-}
OUT=${1:-"$ROOT/dist/fnos-trim"}

if [ -z "$SKILL_DIR" ] || [ ! -f "$SKILL_DIR/manifest.json" ]; then
  echo "set FNOS_SKILL_DIR to the fnOS trim-cli V2 directory" >&2
  exit 2
fi
if [ "$(jq -r '.capabilities | length' "$SKILL_DIR/manifest.json")" -lt 8 ]; then
  echo "FNOS_SKILL_DIR does not look like trim-cli V2" >&2
  exit 1
fi
for binary in trim-cli-linux-x64 trim-cli-linux-arm64; do
  test -x "$SKILL_DIR/bin/$binary" || { echo "missing V2 binary: $binary" >&2; exit 1; }
done

mkdir -p "$OUT"
for target in "amd64:x86:trim-cli-linux-x64" "arm64:arm:trim-cli-linux-arm64"; do
  goarch=${target%%:*}
  rest=${target#*:}
  platform=${rest%%:*}
  binary=${rest##*:}
  stage=$(mktemp -d)
  trap 'rm -rf "$stage"' EXIT HUP INT TERM
  mkdir -p "$stage/bin"
  cp "$ROOT/plugins/fnos-trim/plugin.json" "$stage/plugin.json"
  cp "$ROOT/plugins/fnos-trim/icon.png" "$stage/icon.png"
  cp "$SKILL_DIR/bin/$binary" "$stage/bin/trim-cli"
  CGO_ENABLED=0 GOOS=linux GOARCH="$goarch" go build -trimpath -ldflags '-s -w' -o "$stage/plugin" "$ROOT/plugins/fnos-trim"
  "$ROOT/scripts/package-plugin.sh" "$stage" "$OUT/fnos-trim-0.1.0-$platform.zip"
  rm -rf "$stage"
  trap - EXIT HUP INT TERM
done
printf 'built %s\n' "$OUT"
