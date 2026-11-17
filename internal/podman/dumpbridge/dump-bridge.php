<?php
// /usr/local/etc/lerd/dump-bridge.php
//
// Always-mounted auto_prepend_file, kept to syntax PHP 5.3 can parse. PHP
// compiles a whole file before it runs a line of it, so a guard inside the real
// bridge cannot protect an old runtime from its return and parameter type hints.
// This stub is the guard: it returns on anything older than PHP 7.2 and only
// then hands over to dump-bridge-impl.php, which holds the real bridge and
// may use PHP 7.2 syntax. Keep this file free of type hints and newer operators.

if (PHP_VERSION_ID < 70200) {
    return;
}

@include dirname(__FILE__) . '/dump-bridge-impl.php';
