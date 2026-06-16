# Athena PHP

Encodes a PHP project so the source can't be read or tampered with, and ships a
small Zend extension that decodes it at runtime. Same idea as ionCube or
SourceGuardian, just our own.

Two pieces:

- `athena` - a Go CLI that encrypts every `.php` file in a project.
- the `athena` extension - a PHP extension (C) that decrypts and runs them. No
  extension installed, nothing runs.

Targets PHP 8.3 and 8.4 on Debian 13, shipped as a `.deb`.

## Building

You need Go 1.24+ for the encoder. The extension and the `.deb` are built inside
a Debian 13 container, so there's no need to install PHP dev headers on your
machine (you do need Docker).

```sh
make build      # encoder -> bin/athena
make key        # generates athena.key + the embedded key header (run once)
make ext        # builds athena.so for 8.3 and 8.4
make deb        # builds the .deb
```

Docker on Apple Silicon produces arm64 by default. For amd64 servers:

```sh
docker build --platform linux/amd64 -t athena-build:deb13 -f build/Dockerfile.debian13 build
make ext deb
```

### Building the extension without Docker

If you've already got PHP dev headers and libsodium on the box (your laptop or a
Debian build host), you can skip the container and build straight against the
local PHP:

```sh
make ext-local                                   # uses `phpize` on PATH
PHPIZE=phpize8.3 PHP_CONFIG=php-config8.3 make ext-local   # pin a version
```

The `.so` lands in `build/out/local/`. This targets whatever PHP version you have
installed, so it's meant for development - for something you're going to ship,
build with Docker (or on a real Debian 13 host) so it matches 8.3/8.4.

### Debian 13 packages

To build on Debian 13 directly (no container), you need:

```sh
# PHP 8.3 isn't in Debian 13's repos, so add Ondřej Surý's:
sudo apt install -y ca-certificates curl lsb-release
curl -fsSL https://packages.sury.org/php/apt.gpg | sudo tee /etc/apt/trusted.gpg.d/sury-php.gpg >/dev/null
echo "deb https://packages.sury.org/php/ $(lsb_release -sc) main" | sudo tee /etc/apt/sources.list.d/sury-php.list
sudo apt update

# build deps
sudo apt install -y build-essential autoconf pkg-config file dpkg-dev \
    libsodium-dev php8.3-dev php8.4-dev
```

Plus Go 1.24+ for the encoder (from golang.org or `apt install golang` if it's
new enough).

On the target/runtime server you only need:

```sh
sudo apt install -y libsodium23 php-common php8.3-cli php8.4-cli   # +fpm if you use it
```

(`php-common` provides `phpenmod`, which the package uses to enable the extension.
The `.deb` already declares `libsodium23` and `php-common` as dependencies.)

## Encoding a project

```sh
# mirror the whole tree into a new dir, encode the PHP, copy everything else as-is
bin/athena encode -key athena.key -out /path/out /path/app

# or encode in place
bin/athena encode -key athena.key /path/app
```

`vendor/` is left alone (Composer's autoloader has to stay readable) and Blade
templates are skipped, because Laravel reads those as text and encrypting them
breaks rendering.

The rest of the commands:

```
athena keygen   generate a key, add -header to also write the C header
athena header   regenerate the C header from an existing key
athena info     check whether a file is encoded and print its header
```

## Installing on a server

```sh
sudo apt install ./athena-php_0.1.0_amd64.deb
```

The package installs `athena.so` into the extension directory for each PHP
version and enables it with `phpenmod`. It depends on `libsodium23`.

## How it works

Every file is encrypted with XChaCha20-Poly1305 and wrapped in a short PHP stub.
The stub only ever runs when the extension is missing, in which case it throws so
you get an obvious error instead of a half-broken page.

At runtime the extension hooks `zend_compile_file`. When it spots one of our
containers it decrypts the payload in memory and passes it to the normal
compiler; anything that isn't ours compiles as usual. OPcache sits on top, so a
file is only decrypted and compiled on a cache miss and is served from the opcode
cache after that. That keeps the runtime cost down to roughly nothing once warm.

The key is baked into the extension at build time, XOR-split so it doesn't fall
out of a `strings` dump. Anything encoded with a different key, or modified after
encoding, fails the auth check and won't load.

The on-disk format is written up in [docs/FORMAT.md](docs/FORMAT.md).

## What this does and doesn't protect

The encryption itself is fine - a copied `.php` file is useless without the
matching extension. The soft spot, same as any encoder of this kind, is that the
key lives in the extension on the same box. It's obfuscated, not bulletproof.
Keep `athena.key` and the built `.so` out of anywhere you wouldn't put a private
key. The decrypted source also exists in memory while a request runs, which can't
really be avoided.

If you're encoding a Laravel app, mind the framework caches. `config:cache`,
`route:cache` and `view:cache` all write your code back out as plain PHP under
`bootstrap/cache` and `storage`. Skip them, or accept that those generated copies
are readable. Your actual classes stay encoded regardless. Also run
`composer install` before encoding so `vendor/` is in place - it stays in the
clear on purpose.

## Layout

```
cmd/athena        the CLI
internal/         encoder, crypto, project walker, key handling, format
ext/athena        the PHP extension (C)
build/            Dockerfile + build/package/test scripts
test/fixtures     sample project used by the integration test
docs/FORMAT.md    container format
```

## Tests

```sh
make test    # Go unit tests (round trip, tamper, wrong key)
make itest   # encodes the fixtures and runs them under real 8.3 and 8.4 in Docker
```
