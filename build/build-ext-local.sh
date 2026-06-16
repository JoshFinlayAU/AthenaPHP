#!/usr/bin/env bash
# Build athena.so against the PHP installed on this machine (no Docker).
# Handy for development. The result targets your local PHP version, which may
# differ from the 8.3/8.4 deployment targets - use the Docker build for releases.
#
# Overrides:
#   PHPIZE=phpize8.3 PHP_CONFIG=php-config8.3 build/build-ext-local.sh
#   ATHENA_CONFIGURE_FLAGS="CFLAGS=-I/opt/homebrew/include LDFLAGS=-L/opt/homebrew/lib"
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EXT="$ROOT/ext/athena"
OUT="$ROOT/build/out/local"
PHPIZE="${PHPIZE:-phpize}"
PHP_CONFIG="${PHP_CONFIG:-php-config}"
JOBS="$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 2)"

command -v "$PHPIZE" >/dev/null || { echo "error: $PHPIZE not found (install php-dev, or set PHPIZE=...)" >&2; exit 1; }
command -v "$PHP_CONFIG" >/dev/null || { echo "error: $PHP_CONFIG not found (set PHP_CONFIG=...)" >&2; exit 1; }
[ -f "$EXT/athena_key.h" ] || { echo "error: $EXT/athena_key.h missing; run 'make key' first" >&2; exit 1; }

# Help configure find Homebrew's libsodium on macOS if pkg-config can't.
if [ -z "${ATHENA_CONFIGURE_FLAGS:-}" ] && ! pkg-config --exists libsodium 2>/dev/null; then
    if command -v brew >/dev/null 2>&1; then
        P="$(brew --prefix libsodium 2>/dev/null || true)"
        [ -n "$P" ] && ATHENA_CONFIGURE_FLAGS="CFLAGS=-I$P/include LDFLAGS=-L$P/lib"
    fi
fi

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
cp "$EXT"/*.c "$EXT"/*.h "$EXT/config.m4" "$WORK/"

pushd "$WORK" >/dev/null
    "$PHPIZE"
    # shellcheck disable=SC2086
    ./configure --enable-athena --with-php-config="$(command -v "$PHP_CONFIG")" ${ATHENA_CONFIGURE_FLAGS:-}
    make -j"$JOBS"
    mkdir -p "$OUT"
    cp modules/athena.so "$OUT/athena.so"
popd >/dev/null

echo "built $OUT/athena.so for PHP $("$PHP_CONFIG" --version)"
echo "try it:  php -d extension=$OUT/athena.so -m | grep athena"
