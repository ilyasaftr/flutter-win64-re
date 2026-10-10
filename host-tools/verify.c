// host-tools/verify.c — portable (Linux/macOS) offline verifier / static patcher.
//
// Usage:
//   verify <AppBinary>                 scan, print hits (file offsets)
//   verify <AppBinary> --write <Out>   scan, write patched copy (entry overwritten)
//
// Exit code: 0 if exactly 1 hit (expected for 3.2.23), 2 otherwise.
// The scanner is the same masked core the injector dylib uses (patch_core.h).

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>

#include "patch_core.h"

static int read_file(const char *path, uint8_t **out, size_t *outLen) {
    FILE *f = fopen(path, "rb");
    if (!f) { perror(path); return 0; }
    fseek(f, 0, SEEK_END);
    long n = ftell(f);
    fseek(f, 0, SEEK_SET);
    if (n <= 0) { fclose(f); fprintf(stderr, "%s: empty?\n", path); return 0; }
    uint8_t *buf = (uint8_t *)malloc((size_t)n);
    if (!buf) { fclose(f); return 0; }
    if (fread(buf, 1, (size_t)n, f) != (size_t)n) {
        fclose(f); free(buf); return 0;
    }
    fclose(f);
    *out = buf;
    *outLen = (size_t)n;
    return 1;
}

int main(int argc, char **argv) {
    if (argc < 2) {
        fprintf(stderr, "usage: %s <AppBinary> [--write <Out>]\n", argv[0]);
        return 2;
    }
    uint8_t *buf = NULL;
    size_t len = 0;
    if (!read_file(argv[1], &buf, &len)) return 2;

    // Mach-O dylib: __text file offset/size from LC_SEGMENT_64.
    // Minimal parse: walk load commands for __TEXT..__text.
    // Handles both thin MH_MAGIC_64 and FAT (MH_MAGIC / FAT_CIGAM) binaries.
    size_t base = 0; // slice base for FAT
    if (len > 0x1000) {
        uint32_t magic = *(uint32_t *)buf;
        if (magic == 0xBEBAFECAu) { // FAT_CIGAM (big-endian)
            uint32_t nfat = (uint32_t)((buf[7]) | (buf[6] << 8) |
                                       (buf[5] << 16) | (buf[4] << 24));
            size_t ao = 8;
            for (uint32_t i = 0; i < nfat && ao + 20 <= len; ++i) {
                uint32_t cput = (uint32_t)((buf[ao]) << 24 | (buf[ao + 1] << 16) |
                                           (buf[ao + 2] << 8) | buf[ao + 3]);
                uint32_t off = (uint32_t)((buf[ao + 8]) << 24 | (buf[ao + 9] << 16) |
                                          (buf[ao + 10] << 8) | buf[ao + 11]);
                if (cput == 0x0100000Cu) { // CPU_TYPE_ARM64
                    base = off;
                    break;
                }
                ao += 20;
            }
            fprintf(stderr, "fat binary, arm64 slice @0x%zx\n", base);
            magic = base + 4 <= len ? *(uint32_t *)(buf + base) : 0;
        }
    }
    size_t textOff = 0, textSize = 0;
    if (len > 0x1000) {
        uint32_t magic = *(uint32_t *)(buf + base);
        if (magic == 0xFEEDFACFu) { // MH_MAGIC_64 LE
            uint32_t ncmds = *(uint32_t *)(buf + base + 16);
            size_t off = base + 32;
            for (uint32_t i = 0; i < ncmds && off + 8 <= len; ++i) {
                uint32_t cmd = *(uint32_t *)(buf + off);
                uint32_t sz = *(uint32_t *)(buf + off + 4);
                if (sz < 8 || off + sz > len) break;
                if (cmd == 0x19u) { // LC_SEGMENT_64
                    char seg[17];
                    memcpy(seg, buf + off + 8, 16);
                    seg[16] = 0;
                    if (!strcmp(seg, "__TEXT")) {
                        uint32_t nsects = *(uint32_t *)(buf + off + 64);
                        size_t so = off + 72;
                        for (uint32_t s = 0; s < nsects && so + 80 <= len; ++s) {
                            char sn[17], mn[17];
                            memcpy(sn, buf + so, 16); sn[16] = 0;
                            memcpy(mn, buf + so + 16, 16); mn[16] = 0;
                            uint64_t addr = *(uint64_t *)(buf + so + 32);
                            uint64_t size = *(uint64_t *)(buf + so + 40);
                            uint32_t fileoff = *(uint32_t *)(buf + so + 48);
                            (void)addr;
                            if (!strcmp(sn, "__text") && !strcmp(mn, "__TEXT")) {
                                textOff = base + fileoff;
                                textSize = (size_t)size;
                            }
                            so += 80;
                        }
                    }
                }
                off += sz;
            }
        }
    }
    if (!textSize) { // fallback: whole file
        textOff = 0;
        textSize = len;
        fprintf(stderr, "note: __text not located, scanning whole file\n");
    }
    fprintf(stderr, "__text: fileoff=0x%zx size=0x%zx\n", textOff, textSize);

    uint32_t *words = (uint32_t *)(buf + textOff);
    size_t nwords = textSize / 4;
    long hits[64];
    long n = rmh_scan_all(words, nwords, hits, 64);
    for (long i = 0; i < n && i < 64; ++i) {
        printf("hit %ld: fileoff=0x%lx\n", i, (unsigned long)(textOff + hits[i]));
        uint8_t *p = buf + textOff + hits[i];
        printf("  entry bytes: ");
        for (int b = 0; b < RMH_WORDS * 4; ++b) printf("%02X%c", p[b], b % 4 == 3 ? ' ' : ':');
        printf("\n");
    }
    printf("total: %ld\n", n);

    int rc = (n == 1) ? 0 : 2;
    if (argc >= 4 && !strcmp(argv[2], "--write")) {
        if (n < 1) {
            fprintf(stderr, "--write: no hit, nothing written\n");
            free(buf);
            return 2;
        }
        uint8_t *p = buf + textOff + hits[0];
        memcpy(p, kRmhPatch, RMH_PATCH_SIZE);
        FILE *f = fopen(argv[3], "wb");
        if (!f) { perror(argv[3]); free(buf); return 2; }
        fwrite(buf, 1, len, f);
        fclose(f);
        printf("wrote patched copy: %s (entry +0 -> %d bytes)\n", argv[3], RMH_PATCH_SIZE);
        printf("NOTE: re-sign before running, e.g. codesign -f -s - <file>\n");
        rc = 0;
    }
    free(buf);
    return rc;
}
