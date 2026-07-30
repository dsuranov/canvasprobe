# Data handling

```text
User token -> CanvasProbe process -> official Figma REST API
Figma response -> stdout
                   \-> optional user-local TTL cache for GET only

No project server
No telemetry
No third-party transfer
```

The cache is stored below `${XDG_CACHE_HOME:-$HOME/.cache}/canvasprobe`. Its directory mode is forced to `0700`; files are written atomically with `0600`.

Cache keys contain SHA-256-derived material, not plaintext tokens. Different tokens and API origins cannot share entries. `POST` and `DELETE` never read or write the cache.

`canvasprobe cache purge` removes `.json` and temporary CanvasProbe cache entries. It does not touch other files or directories.
