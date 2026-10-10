# reqable-writer

Fast Go replacement for `scripts/grok_direct_write.py`. Writes Reqable's
JSON config files directly, bypassing the MCP premium gate
(`capture_rewrite_create` etc. return "premium feature" even when logged
in). Restart Reqable afterwards — the app caches config in Box LMDB at
startup.

## Build

```sh
go -C tools/reqable-writer build -o ../../build/reqable-writer .
# -> build/reqable-writer (arm64, ~4M, no deps, stdlib only)
```

`list` ~4ms vs ~160ms for the Python helper (40x faster startup).

## Usage

```sh
./build/reqable-writer list
./build/reqable-writer rewrite --name NAME --url 'https://example.invalid/*' --payload '{"ok": true}'
./build/reqable-writer rewrite-modify --name NAME --url 'https://...' --pattern '"a":true' --replacement '"a":false'
./build/reqable-writer breakpoint --name NAME --url 'https://example.invalid/*'
./build/reqable-writer collection --name NAME
./build/reqable-writer script --name NAME --url 'https://...' --code ./hook.py
./build/reqable-writer delete-rewrite --id ID
./build/reqable-writer delete-breakpoint --id ID

# reload:
pkill -f ReqableDYLD
DYLD_INSERT_LIBRARIES=$PWD/build/ReqableMacHack/libreqablemachack.dylib /tmp/ReqableDYLD.app/Contents/MacOS/Reqable
```

The writer auto-selects the active user dir (most rewrites wins) instead of
hardcoding the uid like the Python helper. Writes are atomic (tmp + rename).
