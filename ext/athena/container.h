/* Athena container parsing/decryption. Mirrors internal/format (Go). */
#ifndef ATHENA_CONTAINER_H
#define ATHENA_CONTAINER_H

#include <stddef.h>
#include <stdint.h>

#define ATHENA_MAGIC_LEN  6
#define ATHENA_HEADER_LEN 18
#define ATHENA_NONCE_LEN  24
#define ATHENA_TAG_LEN    16
#define ATHENA_KEYLEN     32

/* Result codes. */
#define ATHENA_OK           0
#define ATHENA_ERR_SHORT   (-1)
#define ATHENA_ERR_MAGIC   (-2)
#define ATHENA_ERR_VERSION (-3)
#define ATHENA_ERR_KEYID   (-4)
#define ATHENA_ERR_AUTH    (-5)
#define ATHENA_ERR_ORIGLEN (-6)
#define ATHENA_ERR_MEM     (-7)

extern const unsigned char athena_magic[ATHENA_MAGIC_LEN];

/* Return the byte offset of the container magic within buf, or -1. */
long athena_find_magic(const char *buf, size_t len);

/* Decrypt the container that begins at c (clen bytes, c[0] == magic[0]).
 * On success, allocates *out (malloc, caller frees) of *outlen bytes and
 * returns ATHENA_OK; otherwise returns a negative ATHENA_ERR_* code and
 * leaves *out untouched. key is ATHENA_KEYLEN bytes; keyid guards against
 * decoding files encrypted with a different key. */
int athena_container_decrypt(const unsigned char *c, size_t clen,
                             const unsigned char *key, uint32_t keyid,
                             unsigned char **out, size_t *outlen);

/* Human-readable description of an ATHENA_ERR_* code. */
const char *athena_strerror(int rc);

#endif /* ATHENA_CONTAINER_H */
