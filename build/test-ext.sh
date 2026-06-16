#!/usr/bin/env bash
# Integration test: run the encoded fixtures under each PHP version with OPcache
# enabled and the athena extension loaded; assert output and tamper rejection.
# Runs inside the Debian 13 container. Encoded fixtures are produced on the host
# (make itest) into build/test-encoded with the same key compiled into the .so.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="$ROOT/build/out"
ENC="$ROOT/build/test-encoded"
EXPECTED="$ROOT/test/fixtures/expected.txt"
VERSIONS="${PHP_VERSIONS:-8.3 8.4}"

[ -f "$ENC/main.php" ] || { echo "missing encoded fixtures in $ENC (run: make itest)" >&2; exit 1; }

fail=0
for V in $VERSIONS; do
    SO="$OUT/php-$V/athena.so"
    PHP="php$V"
    # OPcache is already enabled via the image's default ini; just turn it on for CLI.
    OPTS="-d opcache.enable_cli=1 -d extension=$SO"

    echo "=== PHP $V ==="

    # 1. Correct decode (strip the version-specific php: line before comparing).
    got="$($PHP $OPTS "$ENC/main.php" 2>&1 | grep -v '^php:')"
    if [ "$got" = "$(cat "$EXPECTED")" ]; then
        echo "  decode: PASS"
    else
        echo "  decode: FAIL"; echo "--- got ---"; echo "$got"; echo "--- want ---"; cat "$EXPECTED"
        fail=1
    fi

    # 2. Tamper rejection: flip a byte in a copy and expect a fatal error.
    tdir="$(mktemp -d)"
    cp "$ENC/main.php" "$tdir/main.php"
    cp "$ENC/calc.php" "$tdir/calc.php"
    # Flip the last byte (inside the ciphertext/tag) so authentication must fail.
    tf="$tdir/calc.php"; sz="$(stat -c%s "$tf")"
    last="$(tail -c1 "$tf" | od -An -tu1 | tr -d ' ')"
    new=$(( last ^ 255 ))
    printf "$(printf '\\%03o' "$new")" | dd of="$tf" bs=1 seek=$((sz-1)) conv=notrunc status=none
    if $PHP $OPTS "$tdir/main.php" >/dev/null 2>&1; then
        echo "  tamper: FAIL (tampered file ran)"; fail=1
    else
        echo "  tamper: PASS (rejected)"
    fi
    rm -rf "$tdir"

    # 3. Without the extension, the stub must abort (not silently run).
    if php$V "$ENC/main.php" >/dev/null 2>&1; then
        echo "  no-ext: FAIL (ran without extension)"; fail=1
    else
        echo "  no-ext: PASS (stub aborted)"
    fi
done

[ "$fail" -eq 0 ] && echo "ALL TESTS PASSED" || echo "TESTS FAILED"
exit $fail
