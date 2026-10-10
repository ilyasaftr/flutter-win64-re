// ReqableMacHack injector dylib (macOS ARM64).
// DYLD_INSERT_LIBRARIES=libreqablemachack.dylib /Applications/Reqable.app/Contents/MacOS/Reqable
//
// Finds the App framework __text at runtime via dyld, masked-scans for the
// Dart AOT license-check twin (see patch_core.h), and rewrites its entry
// with `add x0,x22,#0x20; ret` (return true).
//
// Log output goes to stderr + os_log so it is visible in Console.app.

#include <stdio.h>
#include <string.h>
#include <stdint.h>
#include <mach/mach.h>
#include <mach-o/dyld.h>
#include <mach-o/loader.h>
#include <mach-o/getsect.h>
#include <libkern/OSCacheControl.h>

#include "patch_core.h"

#define LOG(fmt, ...) fprintf(stderr, "[ReqableMacHack] " fmt "\n", ##__VA_ARGS__)

static int patch_at(void *addr) {
    vm_address_t page = (vm_address_t)addr & ~(vm_page_size - 1);
    vm_size_t len = (vm_size_t)((uint8_t *)addr + RMH_PATCH_SIZE - (uint8_t *)page);
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

static void scan_image(const struct mach_header *mh, const char *name) {
    unsigned long segSize = 0;
    // App is a dylib; __text lives in __TEXT.
    uint8_t *base = getsegmentdata((struct mach_header_64 *)mh, SEG_TEXT, &segSize);
    // Fallback: direct __TEXT,__text section lookup.
    if (!base) {
        unsigned long secSize = 0;
        char *sec = getsectdatafromheader_64((struct mach_header_64 *)mh,
                                             SEG_TEXT, SECT_TEXT, &secSize);
        if (sec) {
            base = (uint8_t *)sec;
            segSize = secSize;
        }
    }
    if (!base || segSize < RMH_WORDS * 4) {
        LOG("%s: no __text (%p/%lu)", name, base, segSize);
        return;
    }
    long hits[16];
    long n = rmh_scan_all((const uint32_t *)base, segSize / 4, hits, 16);
    LOG("%s: scan %lu bytes -> %ld hit(s)", name, segSize, n);
    for (long i = 0; i < n && i < 16; ++i) {
        void *addr = base + hits[i];
        int ok = patch_at(addr);
        LOG("%s: patch @ slide+%ld (0x%lx) %s", name, hits[i],
            (unsigned long)addr, ok ? "SUCCESS" : "FAILED");
    }
    if (n == 0) {
        LOG("%s: pattern not found (version drift? see scripts/)", name);
    }
}

__attribute__((constructor)) static void rmh_init(void) {
    LOG("init (pid %d)", getpid());
    uint32_t count = _dyld_image_count();
    for (uint32_t i = 0; i < count; ++i) {
        const char *name = _dyld_get_image_name(i);
        if (!name) continue;
        // Match the Flutter AOT image: .../App.framework/Versions/A/App
        const char *tail = strrchr(name, '/');
        const char *fw = strstr(name, "App.framework");
        if (fw && tail && strcmp(tail, "/App") == 0) {
            LOG("target image: %s", name);
            scan_image(_dyld_get_image_header(i), name);
            return;
        }
    }
    LOG("App.framework image not loaded yet at constructor time; "
        "check DYLD_INSERT_LIBRARIES order (report this log line).");
}
