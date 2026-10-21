# Plugin SDK experimental packages

This package contains experimental packages

## History

- Experimental package `sdata` has been moved into its own package after few experiments here. If you are using `sdata` from plugin SDK, consider it from dataplane package which is available in [`github.com/grafana/dataplane/sdata`](https://github.com/grafana/dataplane/tree/main/sdata).
- Experimental package `macros` was removed. It had no users outside its own tests. Use [`github.com/grafana/macropro`](https://github.com/grafana/macropro) to parse and expand macros in a plugin backend.
