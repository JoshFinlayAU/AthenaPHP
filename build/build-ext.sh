#!/usr/bin/env bash
# Build athena.so for each supported PHP version. Runs inside the Debian 13
# build container (see Dockerfile.debian13). Produces build/out/php-<ver>/athena.so.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EXT="$ROOT/ext/athena"
OUT="$ROOT/build/out"
VERSIONS="${PHP_VERSIONS:-8.3 8.4}"

if [ ! -f "$EXT/athena_key.h" ]; then
    echo "error: $EXT/athena_key.h missing." >&2
    echo "Generate it first on the host:  athena keygen -header $EXT/athena_key.h" >&2
    exit 1
fi

rm -rf "$OUT"
for V in $VERSIONS; do
    echo "=== Building athena.so for PHP $V ==="
    PHPIZE="phpize$V"
    PHPCONFIG="php-config$V"
    command -v "$PHPIZE" >/dev/null || { echo "missing $PHPIZE" >&2; exit 1; }

    # Build in an isolated copy so phpize artifacts don't collide between versions.
    WORK="$(mktemp -d)"
    cp "$EXT"/*.c "$EXT"/*.h "$EXT"/config.m4 "$WORK/"
    pushd "$WORK" >/dev/null
        "$PHPIZE"
        ./configure --enable-athena --with-php-config="$(command -v "$PHPCONFIG")"
        make -j"$(nproc)"
        DEST="$OUT/php-$V"
        mkdir -p "$DEST"
        cp modules/athena.so "$DEST/athena.so"
        file "$DEST/athena.so"
    popd >/dev/null
    rm -rf "$WORK"
done

echo "=== Done. Artifacts: ==="
find "$OUT" -name '*.so' -print
