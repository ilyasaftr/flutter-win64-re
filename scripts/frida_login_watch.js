// scripts/frida_login_watch.js — catch the app-side login gate via heap+pool scan.
//
// The 65 "…requires an account, please login…" literals live only in the App
// binary (Dart snapshot). At runtime each literal is a heap OneByteString,
// referenced by exactly one object-pool slot. The write handler loads the
// slot (ldr Xt,[x27,#imm]) when login is missing. This script:
//
//   1. Memory-scans the Reqable heap for a login literal (default:
//      "requires an account") -> heap String address.
//   2. Scans writable memory for 8-byte pointers TO that address -> the
//      object-pool slot(s).
//   3. Prints the slot address(es) for an lldb *read watchpoint*:
//        watchpoint set expression -s 8 -- <slot>
//      then trigger any MCP write while logged out: lldb stops inside the
//      handler that loads the message. `bt` + `disassemble -s <pc> -c 60`
//      reveals the login check; extend ReqableMacHack/patch_core.h from there.
//
// Usage:
//   frida -p $(pgrep -x Reqable) -l frida_login_watch.js
//   # optional arg: --needle <text>   (default "requires an account")
//   # optional arg: --full            (also scan all 65 ops' prefixes)

const NEEDLE_ARG = (() => {
    try {
        const a = (typeof script !== 'undefined' && script.args) ? script.args : [];
        const i = a.indexOf('--needle');
        if (i !== -1 && a[i + 1]) return a[i + 1];
    } catch (e) {}
    return 'requires an account';
})();

function appRanges() {
    const out = [];
    for (const m of Process.enumerateModules()) {
        if (m.path.indexOf('App.framework') !== -1 && /\/App$/.test(m.path)) {
            out.push({ name: 'App', base: m.base, size: m.size });
        }
    }
    return out;
}

function scanBytes(pattern) {
    const hits = [];
    for (const r of Process.enumerateRanges('---')) {
        try {
            const m = Memory.scanSync(r.base, r.size, pattern);
            for (const x of m) hits.push(x.address);
            if (hits.length > 40) break;
        } catch (e) { /* unreadable range */ }
    }
    return hits;
}

function isHeapPtr(addr, appRanges_) {
    // Heap = not inside any mapped image.
    for (const m of Process.enumerateModules()) {
        if (addr.compare(m.base) >= 0 && addr.compare(m.base.add(m.size)) < 0)
            return false;
    }
    return true;
}

async function main() {
    console.log('[*] needle: "' + NEEDLE_ARG + '"');
    const pat = Array.from(Memory.allocUtf8String(NEEDLE_ARG).readByteArray(NEEDLE_ARG.length))
        .map(b => ('0' + (b & 0xFF).toString(16)).slice(-2)).join(' ');
    console.log('[*] scanning memory for literal bytes...');
    const strHits = scanBytes(pat);
    console.log('[*] literal byte hits: ' + strHits.length);
    strHits.slice(0, 20).forEach(h => console.log('    bytes @ ' + h));

    // Dart OneByteString layout (64-bit): header(4) + length(4) + data...
    // The object address is a few bytes BEFORE the character data. Try the
    // common cases: data starts at +8 (uncompressed, no hash) — verify by
    // reading back the full C string and checking it extends past the needle.
    const strObjs = [];
    for (const h of strHits.slice(0, 40)) {
        for (const back of [4, 8, 12, 16]) {
            const obj = h.sub(back);
            try {
                const s = obj.add(8).readUtf8String(160);
                if (s && s.indexOf(NEEDLE_ARG) !== -1 && isHeapPtr(obj)) {
                    strObjs.push({ obj, off: back, str: s.slice(0, 90) });
                    break;
                }
            } catch (e) {}
        }
    }
    console.log('[*] heap String objects: ' + strObjs.length);
    strObjs.slice(0, 10).forEach(o =>
        console.log('    obj ' + o.obj + ' (data+' + o.off + '): ' + JSON.stringify(o.str)));

    if (!strObjs.length) {
        console.log('[-] no heap strings found. Is Reqable running the snapshot? ' +
            'Try triggering any write first, then re-run.');
        return;
    }

    // Pointer scan: find 8-byte LE slots whose value == a string object.
    // Restrict to rw ranges to keep it fast.
    console.log('[*] scanning for pool slots (pointers to the String objects)...');
    const seen = new Set();
    for (const o of strObjs.slice(0, 10)) {
        const le = [];
        let v = o.obj.toUInt32();
        // build 8-byte LE pattern of the address (64-bit aware)
        const u64 = o.obj; // NativePointer
        const bytes = [];
        // read the address bytes back from a scratch write
        const scratch = Memory.alloc(8);
        scratch.writePointer(u64);
        const raw = scratch.readByteArray(8);
        const hex = Array.from(new Uint8Array(raw)).map(
            b => ('0' + b.toString(16)).slice(-2)).join(' ');
        for (const r of Process.enumerateRanges('rw-')) {
            if (r.size > 64 * 1024 * 1024) continue; // skip giants
            try {
                const m = Memory.scanSync(r.base, r.size, hex);
                for (const x of m) {
                    const key = x.address.toString();
                    if (!seen.has(key)) {
                        seen.add(key);
                        console.log('    slot @ ' + x.address +
                            ' (range ' + r.base + '+0x' + r.size.toString(16) + ')' +
                            ' -> ' + o.obj + ' "' + o.str.slice(0, 50) + '"');
                    }
                    if (seen.size > 25) break;
                }
            } catch (e) {}
            if (seen.size > 25) break;
        }
        if (seen.size > 25) break;
    }
    console.log('[*] done. Next in lldb:');
    console.log('    watchpoint set expression -s 8 -- <slot>');
    console.log('    c   # then trigger an MCP write while logged out');
    console.log('    bt; disassemble -s $pc -c 60');
}

main();
