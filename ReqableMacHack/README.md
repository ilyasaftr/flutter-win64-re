# ReqableMacHack

macOS ARM64 equivalent of `ReqableHack` (Windows x64 `version.dll` proxy).

The Windows hack runtime-patches a single Dart AOT bool getter (found by an
8-byte `mov rax,[r15+0x70] / ret` overwrite) in every anonymous Dart AOT code
region. This project ports that work to the macOS ARM64 build of Reqable
(`Reqable.app/Contents/Frameworks/App.framework/Versions/A/App`).

Tested against: Reqable macOS arm64 `3.2.23` (`reqable-app-macos-arm64.dmg`,
Flutter `3.3.4` / Dart `3.3.4`), same app version as the Windows build the
x64 pattern was taken from.

## How it works

On macOS the Dart AOT code is **file-backed** (`App` framework `__text`,
`kDartIsolateSnapshotInstructions`), not anonymous, so there is no need for
the `MemMonitor` region watcher. Instead this dylib, injected with
`DYLD_INSERT_LIBRARIES`, locates the `App` image via `dyld`, masked-scans its
`__text` for the ARM64 twin of the Windows target function, and overwrites
the function entry with an unconditional `return true`:

```text
file off 0x34a878 (3.2.23):
  stp  x29,x30,[x15,#-0x10]!   FD 79 BF A9
  mov  x29,x15                 FD 03 0F AA
  sub  x15,x15,#0x10           EF 41 00 D1
  ldr  x16,[x26,#0x38]         50 1F 40 F9
  cmp  x15,x16                 FF 01 10 EB
  b.ls ...                     A9 03 00 54
  bl   <getter>                F8 36 FB 97
  ldur x1,[x0,#7]              01 70 40 F8
  ldr  x16,[x27,#0x40]         70 23 40 F9
  cmp  x1,x16                  3F 00 10 EB
  b.eq ...                     40 03 00 54
  ldur x0,[x1,#0x2f]           20 F0 42 F8
  cmp  x0,x22                  1F 00 16 EB
  b.eq ...                     A0 01 00 54
  str  x0,[x15]                E0 01 00 F9
  bl   <check>                 5E 02 00 94
  tbnz w0,#4,...               A0 00 20 37
  ...

patched to:
  add  x0,x22,#0x20            C0 82 00 91   ; x0 = true  (x22 = null, true = null+0x20)
  ret                           C0 03 5F D6
```

Cross-architecture evidence that `0x34a878` is the twin of the Windows target
(`app.so` file offset `0xe2795c`):

| fact | x64 (Windows) | ARM64 (macOS) |
|---|---|---|
| shape | `call; [rax+7]; cmp pool; je; [rcx+0x2f]; cmp sentinel; je; [rsp]=; call; test al,0x10` | `bl; [x0,#7]; cmp pool; b.eq; [x1,#0x2f]; cmp x22; b.eq; [x15]=; bl; tbnz w0,#4` |
| field offset | `+0x2f` | `+0x2f` |
| Smi bit test | `test al,0x10` | `tbnz w0,#4` |
| fallthrough return | `[r14+0x70]` (true) | `x22+0x20` (true) |
| taken return | `[r14+0x78]` | `x22+0x30` |
| pattern matches in-narrow-scan | exactly 1 | exactly 1 |
| direct callers | **142** | **142** |
| first callee | shared getter | shared getter `0x218470` (also called by the bigger sibling at `0x218354`) |

ARM64 Dart notes that fell out of the analysis and are baked into the
scanner: thread register is `x26` (`0x38` = stack limit, `0x68` = null which
is cached in `x22`), pool pointer is `x27`, `x22` holds the null object, and
the canonical bools are addressed as small offsets of `x22`
(`true = x22+0x20`, `false = x22+0x30`) rather than thread slots.

## Layout

```text
ReqableMacHack/
  CMakeLists.txt     host-side build (macOS only) for the injector dylib
  inject.c           constructor dylib: dyld image lookup + masked scan + patch
                     (handles late Flutter dlopen via _dyld_register_func_for_add_image)
  patch_core.h       scanner/patcher (also compiled into host-tools/verify)
host-tools/
  verify.c           Linux/macOS CLI: scans a copy of the App binary, prints
                     hits, optionally writes a patched copy (static file patch)
scripts/
  resign.sh          helper: copy + ad-hoc re-sign so DYLD injection is not blocked
  lldb_find.py       lldb helper: find + inspect + patch the live function
  frida_find.js      Frida helper: find + inspect (+ patch) the live function
```

## Build the dylib (on a Mac)

```sh
cmake -S ReqableMacHack -B build/ReqableMacHack -DCMAKE_OSX_ARCHITECTURES=arm64
cmake --build build/ReqableMacHack --config Release
# output: build/ReqableMacHack/libreqablemachack.dylib
```

## Try it (on a Mac, personal test environment only)

```sh
# 1. verify against your installed copy first (static, no code signing touched)
./build/host-verify /Applications/Reqable.app/Contents/Frameworks/App.framework/Versions/A/App

# expect: exactly one hit, file offset 0x34a878 (for 3.2.23)

# 2a. static file patch test (re-sign afterwards, see below)
./build/host-verify App --write App.patched
codesign -f -s - App.patched   # ad-hoc; real distribution needs proper signing

# 2b. runtime patch test (requires re-sign — hardened runtime blocks DYLD by default)
#     App.framework is dlopened lazily by the Flutter engine, so the dylib
#     registers a dyld add-image callback and patches on late load.
./scripts/resign.sh /Applications/Reqable.app /tmp/ReqableDYLD.app
DYLD_INSERT_LIBRARIES=$PWD/build/ReqableMacHack/libreqablemachack.dylib \
  /tmp/ReqableDYLD.app/Contents/MacOS/Reqable
# expect logs: "App.framework not yet loaded, registering dyld callback"
#              "late load detected: .../App.framework/Versions/A/App"
#              "scan ... -> 1 hit(s)" + "patch @ offset 0x346878 ... SUCCESS"
#     For quick one-liner (ad-hoc deep re-sign):
#       codesign --force --deep -s - /Applications/Reqable.app   # (keep a backup!)
```

> SIP / hardened runtime: `DYLD_INSERT_LIBRARIES` is ignored for
> hardened/signed apps (`flags=0x10000(runtime)`) unless you re-sign without
> hardening. Use `scripts/resign.sh` or `codesign --force --deep -s - <app>`
> to produce an ad-hoc signed copy where DYLD is honored. The static-patch
> route (2a) also needs `codesign -f -s -` after patching. Always keep a
> backup of the original.

## Verify with lldb / Frida (on a Mac)

```sh
# lldb: breaks at the twin, shows caller + both return constants
lldb -o "command script import scripts/lldb_find.py" \
     -o "reqfind /Applications/Reqable.app/Contents/Frameworks/App.framework/Versions/A/App" \
     -- /Applications/Reqable.app/Contents/MacOS/Reqable

# frida: same search from JS (+ optional patch with --patch)
frida -l scripts/frida_find.js -f /Applications/Reqable.app/Contents/MacOS/Reqable -- --patch
```

If a future Reqable update moves the function, the masked scan prints
`patch FAILED` (dylib) / `hits: 0`: re-run the finder scripts on the live
process — they report the new address and the exact bytes to refresh
`kPattern`/`kPatch` with.

## Porting notes (Windows -> macOS)

- `version.dll` proxy -> DYLD insert dylib (`__attribute__((constructor))` plus
  `_dyld_register_func_for_add_image` because `App.framework` is dlopened
  after `main` — straight constructor-time scan races the load).
- `MemMonitor` anonymous-region watcher -> dyld late-load callback (Flutter
  `App.framework` is file-backed `__TEXT`/`__text`, present via dyld after
  `FlutterDartVM` start, not anonymous).
- `VirtualProtect` + `FlushInstructionCache` -> `vm_protect` (RW, page-aligned) /
  `sys_icache_invalidate` + restore RX.
- x64 patch (`49 8B 46 70 C3`, 5 bytes) -> ARM64 patch
  (`C0 82 00 91 C0 03 5F D6`, 8 bytes).
- Hardened runtime: stock `Reqable.app` has `flags=0x10000(runtime)` so DYLD
  injection is silently ignored — needs `codesign --force --deep -s -` copy.

## Disclaimer

Personal逆向工程 / Flutter-Dart-AOT learning and security research only.
Reqable and all related trademarks/software belong to their respective
owners; this project is not affiliated with or endorsed by them. Use only on
software you are entitled to analyse, in your own test environment, and in
compliance with local law and the software licence agreement.
