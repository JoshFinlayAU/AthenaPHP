PHP_ARG_ENABLE([athena],
  [whether to enable the Athena decoder],
  [AS_HELP_STRING([--enable-athena], [Enable the Athena decoder extension])],
  [no])

if test "$PHP_ATHENA" != "no"; then

  dnl Locate libsodium, preferring pkg-config.
  AC_MSG_CHECKING([for libsodium])
  if test -x "$(command -v pkg-config)" && pkg-config --exists libsodium; then
    PHP_EVAL_INCLINE([$(pkg-config --cflags libsodium)])
    PHP_EVAL_LIBLINE([$(pkg-config --libs libsodium)], [ATHENA_SHARED_LIBADD])
    AC_MSG_RESULT([found via pkg-config])
  else
    AC_MSG_RESULT([using default search paths])
    AC_CHECK_HEADER([sodium.h], [],
      [AC_MSG_ERROR([libsodium headers not found; install libsodium-dev])])
    PHP_ADD_LIBRARY([sodium], 1, ATHENA_SHARED_LIBADD)
  fi

  PHP_SUBST([ATHENA_SHARED_LIBADD])
  PHP_NEW_EXTENSION([athena], [athena.c container.c], [$ext_shared],,
    [-DZEND_ENABLE_STATIC_TSRMLS_CACHE=1])
fi
