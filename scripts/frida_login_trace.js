// scripts/frida_login_trace.js — find the app-side login gate for MCP writes.
//
// Background: mcp-server (open source: reqable/reqable-mcp-server) has NO
// login check of its own. It forwards ReqableHttpException.message, which is
// the app local-API error body, e.g.
//   "Creating breakpoint requires an account, please login to your Reqable account"
// The gate lives in Reqable.app (all 65 write ops have such a literal).
//
// This script catches the moment the app serves such an error over its local
// HTTP API and prints a native backtrace. Map the App PCs to file offsets
// (PC - App.base) to locate the login-check function and its callers, then
// extend ReqableMacHack/patch_core.h with a second pattern the same way the
// premium twin (App @0x34a878) was derived.
//
// Usage (on a Mac with the resigned/test copy, see ReqableMacHack/README.md):
//   1. launch Reqable (premium patch active), note its proxy port (default 9000)
//   2. attach:  frida -p $(pgrep -x Reqable) -l frida_login_trace.js
//   3. trigger: run any MCP write tool (e.g. collection_create) while logged OUT
//   4. read the backtrace: frames inside App.framework/App are the handlers.
//      fileoff = framePC - AppBase. Disassemble around them with host-verify
//      sources or `lldb: disassemble -s <addr> -c 40`.
//
// The script hooks at two levels and uses whichever fires:
//   A. libc send/write: filters buffers containing "requires an account".
//      (Local API responses go through the socket as JSON.)
//   B. Dart-level: hooks all App.__text `ret`-terminated leaves is too slow,
//      so instead it watches for the pool-string load pattern is skipped;
//      level A is the reliable one.

const LOGIN_NEEDLE = 'requires an account';

function appBase() {
    const m = Process.findModuleByName('App');
    if (m) return m.base;
    for (const mod of Process.enumerateModules()) {
        if (mod.path.indexOf('App.framework') !== -1 && /\/App$/.test(mod.path))
            return mod.base;
    }
    return null;
}

function backtraceHere() {
    const base = appBase();
    const bt = Thread.backtrace(this.context, Backtracer.ACCURATE)
        .map(DebugSymbol.fromAddress);
    bt.forEach((s, i) => {
        let extra = '';
        if (base && s.address.compare(base) >= 0) {
            extra = '  [App fileoff 0x' + s.address.sub(base).toString(16) + ']';
        }
        console.log('  #' + i + ' ' + s + extra);
    });
    console.log('---');
}

function hookFn(lib, names, onMatch) {
    let addr = null;
    try { addr = Module.getExportByName(lib, names[0]); } catch (e) {}
    if (!addr) {
        try {
            const m = Process.getModuleByName(lib);
            addr = m.findExportByName(names[0]);
        } catch (e) {}
    }
    if (!addr) { console.log('[-] ' + lib + '!' + names[0] + ' not found'); return; }
    Interceptor.attach(addr, {
        onEnter(args) {
            try {
                const fd = args[0].toInt32();
                const buf = args[1];
                const len = args[2].toInt32();
                if (len > 0 && len < 4 * 1024 * 1024) {
                    const s = buf.readUtf8String(Math.min(len, 4096));
                    if (s && s.indexOf(LOGIN_NEEDLE) !== -1) {
                        console.log('[+] ' + names[0] + ' fd=' + fd +
                            ' len=' + len + ' matched login gate:');
                        console.log('    ' + JSON.stringify(s.slice(0, 300)));
                        backtraceHere.call(this);
                    }
                }
            } catch (e) { /* ignore racing reads */ }
        }
    });
    console.log('[*] hooked ' + lib + '!' + names[0]);
}

// send()/write() cover both plain HTTP and (after TLS) the local API socket.
hookFn('libsystem_kernel.dylib', ['send']);
hookFn('libsystem_kernel.dylib', ['write']);
hookFn('libsystem_c.dylib', ['send']);
hookFn('libsystem_c.dylib', ['write']);

// Fallback: also watch Foundation -[NSJSONSerialization ...]? Not needed;
// the socket-level hook is sufficient because the local API speaks HTTP/WS.

// Optional: directly exercise the local API from here to trigger the gate:
//   rpc.exports.probe('collection_create', {...})
rpc.exports = {
    probe(route, payload) {
        return new Promise((resolve, reject) => {
            const { HttpClient } = ObjC; // placeholder - use fetch below instead
            resolve('use: curl -X POST 127.0.0.1:PORT' + route + ' from shell');
        });
    }
};

console.log('[*] frida_login_trace loaded. Waiting for "' + LOGIN_NEEDLE + '" ...');
