# scripts/lldb_find.py — lldb helper for ReqableMacHack.
#
#   lldb -- /Applications/Reqable.app/Contents/MacOS/Reqable
#   (lldb) command script import /path/to/scripts/lldb_find.py
#   (lldb) reqfind          # scan App.__text, print hits + disasm
#   (lldb) reqpatch         # reqfind + overwrite entries with return-true
#
# Works on the live process (ASLR-safe: resolves App via dyld images).

import lldb

MASKS = [
    0xFFFFFFFF, 0xFFFFFFFF, 0xFFC003FF, 0xFFFFFFFF, 0xFFFFFFFF,
    0xFF00001F, 0xFC000000, 0xFFC003E0, 0xFFC003E0, None,
    0xFF00001F, 0xFFC00000, None, 0xFF00001F, 0xFFFFFFE0, 0xFC000000, "tb4",
]
VALS = [
    0xA9BF79FD, 0xAA0F03FD, 0xD10001EF, 0xF9401F50, 0xEB1001FF,
    0x54000009, 0x94000000, 0xF8407000, 0xF9400000 | (27 << 5), 0,
    0x54000000, 0xF8400000 | (0x2F << 12), 0, 0x54000000,
    0xF9000000 | (15 << 5), 0x94000000, 0,
]
WORDS = len(MASKS)
PATCH = bytes([0xC0, 0x82, 0x00, 0x91, 0xC0, 0x03, 0x5F, 0xD6])


def _u32(v):
    return v & 0xFFFFFFFF


def word_ok(w, k):
    w = _u32(w)
    m = MASKS[k]
    v = VALS[k]
    if m is None:  # cmp X?,X?
        return (w & 0xFF000000) == 0xEB000000
    if m == "tb4":
        if (w & 0xFF000000) not in (0x36000000, 0x37000000):
            return False
        return (w & 0x00FFFFFF) in (0x00200060, 0x002000A0)
    if m == 0xFF00001F:
        return (w & m) == (v & m)
    return (w & m) == (v & m)


def app_text_range(target):
    for m in target.modules:
        path = m.GetFileSpec().GetFilename() or ""
        full = str(m.GetFileSpec())
        if "App.framework" in full and full.rstrip("/").endswith("/App"):
            for s in m.sections:
                if s.GetName() == "__text":
                    addr = s.GetLoadAddress(target)
                    return addr, s.GetByteSize(), full
    return None


def scan(process, base, size):
    err = lldb.SBError()
    data = process.ReadMemory(base, size, err)
    if not err.Success():
        return None
    n = len(data) // 4
    hits = []
    import struct
    for i in range(n - WORDS):
        ok = True
        for k in range(WORDS):
            (w,) = struct.unpack_from("<I", data, i * 4 + k * 4)
            if not word_ok(w, k):
                ok = False
                break
        if ok:
            hits.append(base + i * 4)
    return hits


def reqfind(debugger, cmd, result, _dict):
    target = debugger.GetSelectedTarget()
    process = target.GetProcess()
    if not process or not process.IsValid():
        result.PutCString("no live process (run first)")
        return
    r = app_text_range(target)
    if not r:
        result.PutCString("App.framework/App image not found")
        return
    base, size, path = r
    result.PutCString("scanning %s base=0x%x size=0x%x" % (path, base, size))
    hits = scan(process, base, size)
    result.PutCString("hits: %d" % len(hits))
    for h in hits[:16]:
        result.PutCString("hit @ 0x%x (fileoff+slide)" % h)
        interp = debugger.GetCommandInterpreter()
        out = lldb.SBCommandReturnObject()
        interp.HandleCommand("disassemble -s 0x%x -c 24" % h, out)
        result.PutCString(out.GetOutput())


def reqpatch(debugger, cmd, result, _dict):
    target = debugger.GetSelectedTarget()
    process = target.GetProcess()
    r = app_text_range(target)
    if not r:
        result.PutCString("App image not found")
        return
    base, size, _ = r
    hits = scan(process, base, size)
    result.PutCString("hits: %d" % len(hits))
    for h in hits[:16]:
        err = lldb.SBError()
        process.WriteMemory(h, PATCH, err)
        result.PutCString("patch @ 0x%x %s" % (h, "OK" if err.Success() else str(err)))


def __lldb_init_module(debugger, _dict):
    debugger.HandleCommand(
        "command script add -f lldb_find.reqfind reqfind "
        "-h 'scan App.__text for the ReqableMacHack license-check twin'")
    debugger.HandleCommand(
        "command script add -f lldb_find.reqpatch reqpatch "
        "-h 'reqfind + overwrite entries with return-true'")
    print("ReqableMacHack helpers loaded: reqfind, reqpatch")
