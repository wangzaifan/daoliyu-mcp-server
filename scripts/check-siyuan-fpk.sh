#!/bin/sh
set -eu
ROOT=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
for json in "$ROOT/siyuan-fpk/wizard/install" "$ROOT/siyuan-fpk/wizard/uninstall" "$ROOT/siyuan-fpk/config/privilege" "$ROOT/siyuan-fpk/config/resource" "$ROOT/siyuan-fpk/app/ui/config"; do
  jq empty "$json"
done
shellcheck "$ROOT"/siyuan-fpk/cmd/* "$ROOT/scripts/build-siyuan-fpk.sh"
jq -e '.defaults["run-as"] == "root"' "$ROOT/siyuan-fpk/config/privilege" >/dev/null
grep -q 'setpriv --reuid=' "$ROOT/siyuan-fpk/cmd/main"
grep -q "SIYUAN_MCP_HOST: '127.0.0.1'" "$ROOT/siyuan-fpk/app/sisyphus-launcher.cjs"
if grep -q -- '--accessAuthCode=' "$ROOT/siyuan-fpk/cmd/main"; then exit 1; fi
for fpk in "$@"; do
  manifest=$(tar -xOzf "$fpk" manifest)
  printf '%s\n' "$manifest" | grep -q '^appname[[:space:]]*=[[:space:]]*daoliyu.siyuan$'
  config=$(tar -xOzf "$fpk" app.tgz | tar -xOzf - ui/config)
  printf '%s\n' "$config" | jq -e '.".url"."daoliyu.siyuan.main".type == "url"' >/dev/null
  if printf '%s\n' "$config" | grep -Eq 'gatewayPrefix|gatewaySocket|iframe'; then exit 1; fi
  files=$(tar -xOzf "$fpk" app.tgz | tar -tzf -)
  printf '%s\n' "$files" | grep -q '^siyuan/kernel$'
  printf '%s\n' "$files" | grep -q '^lib/ld-musl.so.1$'
  printf '%s\n' "$files" | grep -q '^siyuan/LICENSE$'
  printf '%s\n' "$files" | grep -q '^node/bin/node$'
  printf '%s\n' "$files" | grep -q '^node/LICENSE$'
  printf '%s\n' "$files" | grep -q '^sisyphus/mcp-server.cjs$'
  printf '%s\n' "$files" | grep -q '^sisyphus/LICENSE$'
  printf '%s\n' "$files" | grep -q '^sisyphus-launcher.cjs$'
done
printf 'SiYuan FPK check passed\n'
