# Integration readiness modules

`check-integration-readiness.mjs` loads these files in the explicit order listed
by the CLI. They intentionally share one script scope so the split does not alter
the legacy check order, mutable timeout settings, or helper call semantics.

Keep declarations in their owning responsibility file. Cross-file declarations
are available at runtime because the parts are compiled as one script; do not run
an individual part directly.
