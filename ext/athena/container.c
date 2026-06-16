/* Athena container parsing/decryption. See docs/FORMAT.md. */
#include "container.h"

#include <stdlib.h>
#include <string.h>
#include <sodium.h>

const unsigned char athena_magic[ATHENA_MAGIC_LEN] = {
    0x41, 0x54, 0x48, 0x4E, 0x00, 0x01 /* "ATHN",0x00,0x01 */
};

static uint32_t rd_le32(const unsigned char *p)
{
    return (uint32_t)p[0] | ((uint32_t)p[1] << 8) |
           ((uint32_t)p[2] << 16) | ((uint32_t)p[3] << 24);
}

long athena_find_magic(const char *buf, size_t len)
{
    if (buf == NULL || len < ATHENA_MAGIC_LEN) {
        return -1;
    }
    for (size_t i = 0; i + ATHENA_MAGIC_LEN <= len; i++) {
        if ((unsigned char)buf[i] == athena_magic[0] &&
            memcmp(buf + i, athena_magic, ATHENA_MAGIC_LEN) == 0) {
            return (long)i;
        }
    }
    return -1;
}

int athena_container_decrypt(const unsigned char *c, size_t clen,
                             const unsigned char *key, uint32_t keyid,
                             unsigned char **out, size_t *outlen)
{
    const size_t prefix = ATHENA_HEADER_LEN + ATHENA_NONCE_LEN;

    if (clen < prefix + ATHENA_TAG_LEN) {
        return ATHENA_ERR_SHORT;
    }
    if (memcmp(c, athena_magic, ATHENA_MAGIC_LEN) != 0) {
        return ATHENA_ERR_MAGIC;
    }
    if (c[6] != 1) {
        return ATHENA_ERR_VERSION;
    }
    if (rd_le32(c + 10) != keyid) {
        return ATHENA_ERR_KEYID;
    }

    uint32_t origlen = rd_le32(c + 14);
    const unsigned char *ad   = c;                  /* header = AEAD AAD */
    const unsigned char *npub = c + ATHENA_HEADER_LEN;
    const unsigned char *ct   = c + prefix;
    size_t ctlen = clen - prefix;                   /* ciphertext incl. tag */

    unsigned char *m = (unsigned char *)malloc(ctlen ? ctlen : 1);
    if (m == NULL) {
        return ATHENA_ERR_MEM;
    }

    unsigned long long mlen = 0;
    if (crypto_aead_xchacha20poly1305_ietf_decrypt(
            m, &mlen, NULL, ct, ctlen, ad, ATHENA_HEADER_LEN, npub, key) != 0) {
        free(m);
        return ATHENA_ERR_AUTH;
    }
    if ((uint32_t)mlen != origlen) {
        sodium_memzero(m, (size_t)mlen);
        free(m);
        return ATHENA_ERR_ORIGLEN;
    }

    *out = m;
    *outlen = (size_t)mlen;
    return ATHENA_OK;
}

const char *athena_strerror(int rc)
{
    switch (rc) {
    case ATHENA_OK:          return "ok";
    case ATHENA_ERR_SHORT:   return "container truncated";
    case ATHENA_ERR_MAGIC:   return "bad magic";
    case ATHENA_ERR_VERSION: return "unsupported version";
    case ATHENA_ERR_KEYID:   return "key mismatch";
    case ATHENA_ERR_AUTH:    return "authentication failed (tampered or wrong key)";
    case ATHENA_ERR_ORIGLEN: return "plaintext length mismatch";
    case ATHENA_ERR_MEM:     return "out of memory";
    default:                 return "unknown error";
    }
}
