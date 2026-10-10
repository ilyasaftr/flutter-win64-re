// scripts/frida_find.js — Frida helper for ReqableMacHack.
//
// Find the ARM64 twin of the Windows license-check target in the live
// Reqable process, dump it with context, and optionally patch it.
//
//   frida -f /Applications/Reqable.app/Contents/MacOS/Reqable -l frida_find.js
//   # inside REPL: find()  -> prints matches
//   # inside REPL: patch() -> applies add x0,x22,#0x20; ret (needs RWX;
//
// Static scan (no process needed):
//   frida -l frida_find.js -f /bin/false -- --scan /path/to/App
//
// Or run with --patch to patch + continue:
//   frida -f /Applications/Reqable.app/Contents/MacOS/Reqable \
//         -l frida_find.js -- --patch --resume

// Masked 17-word pattern (see ReqableMacHack/patch_core.h).
const WORDS = 17;
const MASKS = [
    0xFFFFFFFF, 0xFFFFFFFF, 0xFFC003FF, 0xFFFFFFFF, 0xFFFFFFFF,
    0xFF00001F, 0xFC000000, 0xFFC003E0, 0xFFC003E0, -1,
    0xFF00001F, 0xFFC00000, -1, 0xFF00001F, 0xFFFFFFE0, 0xFC000000, -2,
];
const VALS = [
    0xA9BF79FD, 0xAA0F03FD, 0xD10001EF, 0xF9401F50, 0xEB1001FF,
    0x54000009, 0x94000000, 0xF8407000, 0xF9400000 | (27 << 5), 0,
    0x54000000, 0xF8400000 | (0x2F << 12), 0, 0x54000000,
    0xF9000000 | (15 << 5), 0x94000000, 0,
];
const PATCH = [0xC0, 0x82, 0x00, 0x91, 0xC0, 0x03, 0x5F, 0xD6];

function wordOk(w, k) {
    const m = MASKS[k] >>> 0, v = VALS[k] >>> 0;
    w >>>= 0;
    if (m === 0xFFFFFFFF) return w === v;
    if (m === -1) return (w & 0xFF000000) >>> 0 === 0xEB000000; // cmp any
    if (m === -2) { // tbz/tbnz w0,#4, any target
        const top = (w & 0xFF000000) >>> 0;
        if (top !== 0x36000000 && top !== 0x37000000) return false;
        const lo = (w & 0x00FFFFFF) >>> 0;
        return lo === 0x00200060 || lo === 0x002000A0;
    }
    if (m === 0xFF00001F) {
        if (v === 0x54000009) return (w & 0xFF00001F) >>> 0 === 0x54000009;
        return (w & 0xFF00001F) >>> 0 === 0x54000000;
    }
    return ((w & m) >>> 0) === ((v & m) >>> 0);
}

function scanRange(base, size) {
    const hits = [];
    const n = Math.floor(size / 4);
    for (let i = 0; i + WORDS <= n; i++) {
        let ok = true;
        for (let k = 0; k < WORDS; k++) {
            if (!wordOk(base.add(i * 4 + k * 4).readU32(), k)) { ok = false; break; }
        }
        if (ok) hits.push(base.add(i * 4));
    }
    return hits;
}

function appTextRange() {
    const mod = Process.findModuleByName('App');
    if (mod !== null) {
        const sec = mod.findSectionByName('__text');
        if (sec) return { base: sec.address, size: sec.size, via: 'App.__text' };
        return { base: mod.base, size: mod.size, via: 'App(full)' };
    }
    for (const m of Process.enumerateModules()) {
        if (m.path.indexOf('App.framework') !== -1 && /(^|\/)App$/.test(m.path)) {
            const sec = m.findSectionByName('__text');
            if (sec) return { base: sec.address, size: sec.size, via: m.path };
            return { base: m.base, size: m.size, via: m.path };
        }
    }
    return null;
}

function dumpWithContext(addr, before, after) {
    before = before || 0; after = after || 24;
    try {
        console.log(Instruction.parse(addr).toString());
        for (let i = 1; i <= after; i++) {
            try { console.log('  ' + Instruction.parse(addr.add(i * 4)).toString()); }
            catch (e) { break; }
        }
    } catch (e) { console.log('disasm failed: ' + e); }
    try {
        console.log('bytes: ' + addr.readByteArray(WORDS * 4).map(
            b => ('0' + (b & 0xFF).toString(16)).slice(-2)).join(' '));
    } catch (e) {}
}

function find() {
    const r = appTextRange();
    if (!r) { console.log('App image not found (is Reqable running?)'); return []; }
    console.log('scanning ' + r.via + ' base=' + r.base + ' size=0x' + r.size.toString(16));
    const hits = scanRange(r.base, r.size);
    console.log('hits: ' + hits.length);
    hits.forEach((h, i) => {
        console.log('--- hit ' + i + ' @ ' + h + ' (offset 0x' + h.sub(r.base).toString(16) + ')');
        dumpWithContext(h, 0, 24);
    });
    return hits;
}

function patchOne(addr) {
    Memory.protect(addr, 8, 'rwx');
    addr.writeByteArray(PATCH);
    try { Instruction.invalidate && Instruction.invalidate(addr, 8); } catch (e) {}
    console.log('patched ' + addr);
}

function patch() {
    const hits = find();
    hits.forEach(patchOne);
    return hits;
}

// CLI-ish: honor --patch / --resume / --scan <file> when embedded via frida -f ... -- ...
(function main() {
    const args = typeof script !== 'undefined' && script.args ? script.args : [];
    if (args.indexOf('--scan') !== -1) {
        const file = args[args.indexOf('--scan') + 1];
        console.log('static file scan of ' + file + ' is done by host-tools/verify instead');
        return;
    }
    if (args.indexOf('--patch') !== -1) patch(); else find();
})();
