#!/bin/sh
set -eu
ROOT=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
VERSION=${VERSION:-0.3.3}
DIST="$ROOT/dist/$VERSION"
rm -rf "$ROOT/build" "$DIST"
mkdir -p "$DIST"
for target in "amd64:x86:x86_64" "arm64:arm:arm64"; do
  arch=${target%%:*}; rest=${target#*:}; platform=${rest%%:*}; fpk_arch=${rest##*:}; out="$ROOT/build/$platform"
  mkdir -p "$out/app/server"
  CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "$out/app/server/daoliyu-mcp" "$ROOT/cmd/daoliyu-mcp"
  cp -R "$ROOT/fpk/." "$out/"
  sed -i.bak "s/^version=.*/version=$VERSION/; s/^platform=.*/platform=$platform/; /^arch=/d" "$out/manifest"
  printf 'arch=%s\n' "$fpk_arch" >> "$out/manifest"
  rm -f "$out/manifest.bak"
  chmod +x "$out/cmd"/*
  rm -f "$ROOT/daoliyu.mcp.fpk"
  fnpack build -d "$out"
  fpk="$ROOT/daoliyu.mcp.fpk"
  test -f "$fpk"
  mv "$fpk" "$DIST/daoliyu.mcp-$VERSION-$platform.fpk"
done
VERSION="$VERSION" "$ROOT/scripts/build-plugins.sh"
printf 'built %s\n' "$DIST"
