#!/bin/sh
set -eu
config="$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)/fpk/app/ui/config"
grep -Eq '"type"[[:space:]]*:[[:space:]]*"url"' "$config"
! grep -Eq '"type"[[:space:]]*:[[:space:]]*"iframe"|gatewayPrefix|gatewaySocket' "$config"
