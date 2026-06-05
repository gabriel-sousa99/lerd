<?php
// /usr/local/etc/lerd/spx-entry.php
//
// Dedicated, intentionally-empty entry script for the profiler.localhost
// vhost (SPX report UI). SPX is a Zend extension that intercepts the
// SPX_UI_URI request and serves its UI regardless of which script runs, so
// SCRIPT_FILENAME only needs to point at *a* real, harmless PHP file.
//
// It must NOT be dump-bridge.php: that file is the auto_prepend_file, so
// pointing SCRIPT_FILENAME at it makes PHP compile it twice per request (once
// as the prepend, once as the main script). Its top-level functions are
// early-bound at compile time, so the runtime include-guard can't stop the
// second compile from fatalling with "Cannot redeclare ...", returning HTTP
// 500 before SPX ever renders. This file declares nothing and emits nothing,
// so the prepend runs once and the main script is a clean no-op.
//
// Kept 7.2-parse-safe (no typed properties, match, arrow fns) for parity with
// the rest of the lerd-mounted PHP that runs down to the legacy tier.
