#!/bin/sh
set -eu
ROOT=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
SIYUAN_VERSION=${SIYUAN_VERSION:-3.8.4}
PACKAGE_VERSION=${PACKAGE_VERSION:-${SIYUAN_VERSION}-0003}
IMAGE="b3log/siyuan:v${SIYUAN_VERSION}"
NODE_IMAGE=${NODE_IMAGE:-node:20-alpine}
SISYPHUS_PACKAGE=${SISYPHUS_PACKAGE:-$ROOT/third_party/sisyphus/package.zip}
SISYPHUS_SHA256=57a07b31f086ac454e920682cd24d479a7f841abc7660328f9080a6a0ad8df2c
DIST="$ROOT/dist/siyuan/$PACKAGE_VERSION"
BUILD="$ROOT/build/siyuan"
rm -rf "$BUILD" "$DIST"
mkdir -p "$DIST"
[ "$(shasum -a 256 "$SISYPHUS_PACKAGE" | awk '{print $1}')" = "$SISYPHUS_SHA256" ] || { echo "unexpected Sisyphus package checksum" >&2; exit 1; }

for target in "amd64:x86:x86_64:libc.musl-x86_64.so.1:/lib/ld-musl-x86_64.so.1" "arm64:arm:arm64:libc.musl-aarch64.so.1:/lib/ld-musl-aarch64.so.1"; do
  arch=${target%%:*}; rest=${target#*:}
  platform=${rest%%:*}; rest=${rest#*:}
  fpk_arch=${rest%%:*}; rest=${rest#*:}
  libc_name=${rest%%:*}; loader_path=${rest#*:}
  out="$BUILD/$platform"
  mkdir -p "$out/app/siyuan" "$out/app/sisyphus" "$out/app/node/bin" "$out/app/lib" "$out/app/etc/ssl/certs" "$out/app/ui/images"
  cp -R "$ROOT/siyuan-fpk/." "$out/"

  docker pull --platform "linux/$arch" "$IMAGE" >/dev/null
  cid=$(docker create --platform "linux/$arch" --entrypoint /bin/true "$IMAGE")
  trap 'docker rm -f "$cid" >/dev/null 2>&1 || true' EXIT INT TERM
  docker cp "$cid:/opt/siyuan/." "$out/app/siyuan/"
  docker cp -L "$cid:$loader_path" "$out/app/lib/ld-musl.so.1"
  docker cp "$cid:/etc/ssl/certs/ca-certificates.crt" "$out/app/etc/ssl/certs/ca-certificates.crt"
  zone_dir=$(mktemp -d "${TMPDIR:-/tmp}/siyuan-zoneinfo.XXXXXX")
  docker cp "$cid:/usr/share/zoneinfo/." "$zone_dir/"
  (cd "$zone_dir" && zip -q -r "$out/app/zoneinfo.zip" .)
  rm -rf "$zone_dir"
  docker rm "$cid" >/dev/null
  trap - EXIT INT TERM
  cp "$out/app/lib/ld-musl.so.1" "$out/app/lib/$libc_name"

  docker pull --platform "linux/$arch" "$NODE_IMAGE" >/dev/null
  node_cid=$(docker create --platform "linux/$arch" --entrypoint /bin/true "$NODE_IMAGE")
  trap 'docker rm -f "$node_cid" >/dev/null 2>&1 || true' EXIT INT TERM
  docker cp "$node_cid:/usr/local/bin/node" "$out/app/node/bin/node"
  docker cp -L "$node_cid:/usr/lib/libstdc++.so.6" "$out/app/lib/libstdc++.so.6"
  docker cp -L "$node_cid:/usr/lib/libgcc_s.so.1" "$out/app/lib/libgcc_s.so.1"
  docker rm "$node_cid" >/dev/null
  trap - EXIT INT TERM
  cp "$ROOT/third_party/node/LICENSE" "$out/app/node/LICENSE"
  unzip -q "$SISYPHUS_PACKAGE" -d "$out/app/sisyphus"
  cp "$ROOT/third_party/sisyphus/LICENSE" "$out/app/sisyphus/LICENSE"

  icon="$out/app/siyuan/stage/icon-large.png"
  [ -f "$icon" ] || icon="$out/app/siyuan/stage/icon.png"
  cp "$icon" "$out/ICON.PNG"
  cp "$icon" "$out/ICON_256.PNG"
  for size in 64 128 256 512; do cp "$icon" "$out/app/ui/images/icon_${size}.png"; done
  sed -i.bak "s/^version=.*/version=$PACKAGE_VERSION/; s/^platform=.*/platform=$platform/; /^arch=/d" "$out/manifest"
  printf 'arch=%s\n' "$fpk_arch" >> "$out/manifest"
  rm -f "$out/manifest.bak"
  chmod +x "$out/cmd"/* "$out/app/siyuan/kernel" "$out/app/node/bin/node" "$out/app/lib/ld-musl.so.1"

  rm -f "$ROOT/daoliyu.siyuan.fpk"
  fnpack build -d "$out"
  [ -f "$ROOT/daoliyu.siyuan.fpk" ]
  mv "$ROOT/daoliyu.siyuan.fpk" "$DIST/daoliyu.siyuan-$PACKAGE_VERSION-$platform.fpk"
done
printf 'built %s\n' "$DIST"
