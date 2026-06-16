#!/usr/bin/env bash
# Assemble a .deb shipping athena.so for each PHP version plus a mods-available
# ini. Runs inside the Debian 13 container after build-ext.sh.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="$ROOT/build/out"
VERSIONS="${PHP_VERSIONS:-8.3 8.4}"
PKG_VERSION="${PKG_VERSION:-1.0.0}"
ARCH="$(dpkg --print-architecture)"
STAGE="$(mktemp -d)"
PKGROOT="$STAGE/athena-php"

mkdir -p "$PKGROOT/DEBIAN"

for V in $VERSIONS; do
    SO="$OUT/php-$V/athena.so"
    [ -f "$SO" ] || { echo "missing $SO; run build-ext.sh first" >&2; exit 1; }

    EXTDIR="$(php-config$V --extension-dir)"     # e.g. /usr/lib/php/20230831
    mkdir -p "$PKGROOT$EXTDIR"
    cp "$SO" "$PKGROOT$EXTDIR/athena.so"

    INIDIR="/etc/php/$V/mods-available"
    mkdir -p "$PKGROOT$INIDIR"
    cat > "$PKGROOT$INIDIR/athena.ini" <<EOF
; Athena PHP decoder. priority 90 so it initializes after OPcache (priority 10),
; leaving OPcache outermost to cache decoded opcodes.
; priority=90
extension=athena.so
EOF
done

INSTALLED_KB="$(du -sk "$PKGROOT" | cut -f1)"

cat > "$PKGROOT/DEBIAN/control" <<EOF
Package: athena-php
Version: $PKG_VERSION
Section: php
Priority: optional
Architecture: $ARCH
Depends: libsodium23, php-common
Maintainer: Athena Networks <josh@athenanetworks.com.au>
Installed-Size: $INSTALLED_KB
Description: Athena PHP decoder extension
 Runtime decoder for PHP source files encoded with the Athena encoder.
 Hooks the Zend compiler to transparently decrypt Athena containers for
 PHP $(echo $VERSIONS | tr ' ' ',').
EOF

cat > "$PKGROOT/DEBIAN/postinst" <<'EOF'
#!/bin/sh
set -e
if command -v phpenmod >/dev/null 2>&1; then
    phpenmod athena || true
fi
exit 0
EOF

cat > "$PKGROOT/DEBIAN/prerm" <<'EOF'
#!/bin/sh
set -e
if command -v phpdismod >/dev/null 2>&1; then
    phpdismod athena || true
fi
exit 0
EOF

chmod 0755 "$PKGROOT/DEBIAN/postinst" "$PKGROOT/DEBIAN/prerm"

DEB="$ROOT/build/athena-php_${PKG_VERSION}_${ARCH}.deb"
dpkg-deb --build --root-owner-group "$PKGROOT" "$DEB"
echo "=== Built: $DEB ==="
dpkg-deb --info "$DEB"
dpkg-deb --contents "$DEB"
rm -rf "$STAGE"
