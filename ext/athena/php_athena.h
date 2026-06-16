#ifndef PHP_ATHENA_H
#define PHP_ATHENA_H

extern zend_module_entry athena_module_entry;
#define phpext_athena_ptr &athena_module_entry

#define PHP_ATHENA_VERSION "0.1.0"

#if defined(ZTS) && defined(COMPILE_DL_ATHENA)
ZEND_TSRMLS_CACHE_EXTERN()
#endif

#endif /* PHP_ATHENA_H */
