// ReqableMacHack injector dylib (macOS ARM64).
// DYLD_INSERT_LIBRARIES=libreqablemachack.dylib /Applications/Reqable.app/Contents/MacOS/Reqable
//
// Finds the App framework __text at runtime via dyld, masked-scans for the
// Dart AOT license-check twin (see patch_core.h), and rewrites its entry
// with `add x0,x22,#0x20; ret` (return true).
//
// Log output goes to stderr so it is visible when launching from Terminal.

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <unistd.h>
#include <stdatomic.h>
#include <mach/mach.h>
#include <mach-o/dyld.h>
#include <mach-o/loader.h>
#include <mach-o/getsect.h>
#include <libkern/OSCacheControl.h>

#include "patch_core.h"

#define LOG(fmt, ...) fprintf(stderr, "[ReqableMacHack] " fmt "\n", ##__VA_ARGS__)

static _Atomic int g_patched = 0;

static int patch_at(void *addr) {
    vm_size_t page_size = (vm_size_t)sysconf(_SC_PAGESIZE);
    if (page_size == 0) page_size = 0x4000;
    vm_address_t page = (vm_address_t)addr & ~(page_size - 1);
    vm_size_t len = (vm_size_t)((uint8_t *)addr + RMH_PATCH_SIZE - (uint8_t *)page);
    // Round up len to page boundary
    len = (len + page_size - 1) & ~(page_size - 1);
    kern_return_t kr = vm_protect(mach_task_self(), page, len, FALSE,
                                  VM_PROT_READ | VM_PROT_WRITE | VM_PROT_COPY);
    if (kr != KERN_SUCCESS) {
        LOG("vm_protect RW failed: %d", kr);
        return 0;
    }
    memcpy(addr, kRmhPatch, RMH_PATCH_SIZE);
    sys_icache_invalidate(addr, RMH_PATCH_SIZE);
    kr = vm_protect(mach_task_self(), page, len, FALSE, VM_PROT_READ | VM_PROT_EXECUTE);
    if (kr != KERN_SUCCESS) {
        LOG("vm_protect RX restore failed: %d (continuing)", kr);
    }
    return 1;
}

static int is_target_image(const char *name) {
    if (!name) return 0;
    const char *fw = strstr(name, "App.framework");
    if (!fw) return 0;
    const char *tail = strrchr(name, '/');
    if (!tail) return 0;
    return strcmp(tail, "/App") == 0;
}

static void scan_image(const struct mach_header *mh, const char *name) {
    if (atomic_load(&g_patched)) return;
    unsigned long segSize = 0;
    char *seg = getsegmentdata((struct mach_header_64 *)mh, SEG_TEXT, &segSize);
    uint8_t *base = (uint8_t *)seg;
    if (!base) {
        unsigned long secSize = 0;
        char *sec = getsectiondata((struct mach_header_64 *)mh, SEG_TEXT, SECT_TEXT, &secSize);
        if (sec) {
            base = (uint8_t *)sec;
            segSize = secSize;
        }
    }
    if (!base || segSize < RMH_WORDS * 4) {
        LOG("%s: no __text (%p/%lu) — skip", name, base, segSize);
        return;
    }
    long hits[16];
    long n = rmh_scan_all((const uint32_t *)base, segSize / 4, hits, 16);
    LOG("%s: scan %lu bytes -> %ld hit(s)", name, segSize, n);
    for (long i = 0; i < n && i < 16; ++i) {
        void *addr = base + hits[i];
        int ok = patch_at(addr);
        LOG("%s: patch @ offset 0x%lx (addr %p) %s", name, (unsigned long)hits[i],
            addr, ok ? "SUCCESS" : "FAILED");
        if (ok) atomic_store(&g_patched, 1);
    }
    if (n == 0) {
        LOG("%s: pattern not found (version drift? see scripts/)", name);
    }
}

static void image_added_callback(const struct mach_header *mh, intptr_t slide) {
    (void)slide;
    // dyld may call this for every image; filter quickly
    // We can't safely call _dyld_get_image_name here for the new image index
    // in all dyld versions, so brute-scan by header match against known names.
    // Simpler: iterate all images and try to identify the new one.
    uint32_t count = _dyld_image_count();
    for (uint32_t i = 0; i < count; ++i) {
        const struct mach_header *h = _dyld_get_image_header(i);
        if (h != mh) continue;
        const char *name = _dyld_get_image_name(i);
        if (is_target_image(name)) {
            LOG("late load detected: %s", name);
            scan_image(mh, name);
        }
        break;
    }
}

__attribute__((constructor)) static void rmh_init(void) {
    LOG("init (pid %d)", getpid());
    // 1) Try images already loaded
    uint32_t count = _dyld_image_count();
    for (uint32_t i = 0; i < count; ++i) {
        const char *name = _dyld_get_image_name(i);
        if (is_target_image(name)) {
            LOG("target image already loaded: %s", name);
            scan_image(_dyld_get_image_header(i), name);
            if (atomic_load(&g_patched)) return;
        }
    }
    // 2) App loads lazily (Flutter engine dlopens it) — watch for late loads
    LOG("App.framework not yet loaded, registering dyld callback (hardened runtime? %s)",
        getenv("DYLD_INSERT_LIBRARIES") ? "DYLD active" : "DYLD ignored — re-sign the app");
    _dyld_register_func_for_add_image(image_added_callback);
}
