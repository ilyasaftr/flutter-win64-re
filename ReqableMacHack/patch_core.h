// Shared pattern core for ReqableMacHack (macOS ARM64 port of ReqableHack).
//
// Target: Dart AOT bool getter, Reqable 3.2.23 App __text file offset 0x34a878.
// Twin of the Windows x64 target (app.so file offset 0xe2795c):
//   x64: call; [rax+7]; cmp pool; je; [rcx+0x2f]; cmp sentinel; je;
//        [rsp]=; call; test al,0x10  -> return [r14+0x70] (true) / [r14+0x78]
//   arm64: bl; [x0,#7]; cmp pool; b.eq; [x1,#0x2f]; cmp x22; b.eq;
//        [x15]=; bl; tbnz w0,#4  -> return x22+0x20 (true) / x22+0x30
//
// Patch overwrites the 17-instruction entry with:
//   add x0,x22,#0x20   C0 82 00 91
//   ret                C0 03 5F D6
// (x22 = Dart null register; true == null+0x20 in this build.)
#pragma once
#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

#define RMH_WORDS 17
#define RMH_PATCH_SIZE 8

// Masked word pattern. Special kinds (mask field):
//   0xFFFFFFFF / value            exact word
//   RMH_CMP_ANY (mask)            any 64-bit SUBS cmp  ((w&0xFF000000)==0xEB000000)
//   RMH_BEQ     (mask)            b.eq with any target
//   RMH_BLS     (mask)            b.ls with any target
//   RMH_BL      (mask)            bl with any target
//   RMH_TB4     (mask)            tbz/tbnz w0,#4 with any target
#define RMH_CMP_ANY 0xEB000001u
#define RMH_BEQ     0x54000002u
#define RMH_BLS     0x54000003u
#define RMH_BL      0x94000004u
#define RMH_TB4     0x37000005u

static const uint32_t kRmhMask[RMH_WORDS] = {
    0xFFFFFFFFu, // 0  stp  x29,x30,[x15,#-0x10]!
    0xFFFFFFFFu, // 1  mov  x29,x15
    0xFFC003FFu, // 2  sub  x15,x15,#imm (frame size varies)
    0xFFFFFFFFu, // 3  ldr  x16,[x26,#0x38]  (stack limit; x26 = Thread)
    0xFFFFFFFFu, // 4  cmp  x15,x16
    RMH_BLS,     // 5  b.ls overflow
    RMH_BL,      // 6  bl   <getter>
    0xFFC003E0u, // 7  ldur Xt,[x0,#7]
    0xFFC003E0u, // 8  ldr  Xt,[x27,#imm]    (pool; x27 = PP)
    RMH_CMP_ANY, // 9  cmp  X?,X?
    RMH_BEQ,     // 10 b.eq null-path
    0xFFC00000u, // 11 ldur Xt,[Xy,#0x2f]    (license field; same offset as x64)
    RMH_CMP_ANY, // 12 cmp  X?,X?
    RMH_BEQ,     // 13 b.eq slow-path
    0xFFFFFFE0u, // 14 str  Xt,[x15]
    RMH_BL,      // 15 bl   <check>
    RMH_TB4,     // 16 tbz/tbnz w0,#4  (x64: test al,0x10)
};

static const uint32_t kRmhValue[RMH_WORDS] = {
    0xA9BF79FDu,
    0xAA0F03FDu,
    0xD10001EFu,
    0xF9401F50u,
    0xEB1001FFu,
    0x54000009u,
    0x94000000u,
    0xF8407000u,
    0xF9400000u | (27u << 5),
    0x00000000u,
    0x54000000u,
    0xF8400000u | (0x2Fu << 12),
    0x00000000u,
    0x54000000u,
    0xF9000000u | (15u << 5),
    0x94000000u,
    0x00000000u,
};

static const uint8_t kRmhPatch[RMH_PATCH_SIZE] = {
    0xC0, 0x82, 0x00, 0x91, // add x0,x22,#0x20  (x0 = true)
    0xC0, 0x03, 0x5F, 0xD6, // ret
};

static inline int rmh_word_ok(uint32_t w, uint32_t m, uint32_t v) {
    if (m == RMH_CMP_ANY) return (w & 0xFF000000u) == 0xEB000000u;
    if (m == RMH_BEQ) return (w & 0xFF00001Fu) == 0x54000000u;
    if (m == RMH_BLS) return (w & 0xFF00001Fu) == 0x54000009u;
    if (m == RMH_BL) return (w & 0xFC000000u) == 0x94000000u;
    if (m == RMH_TB4) {
        if ((w & 0xFF000000u) != 0x36000000u &&
            (w & 0xFF000000u) != 0x37000000u)
            return 0;
        // capstone-verified tbz/tbnz w0,#4 shapes: low 24 bits 0x200060/0x2000A0
        uint32_t lo = w & 0x00FFFFFFu;
        return lo == 0x00200060u || lo == 0x002000A0u;
    }
    return (w & m) == (v & m);
}

// Scan 4-aligned words. Returns byte offset of the entry or -1.
// `nwords` = number of uint32 words available at `words`.
static inline long rmh_scan(const uint32_t *words, size_t nwords) {
    if (nwords < RMH_WORDS) return -1;
    for (size_t i = 0; i + RMH_WORDS <= nwords; ++i) {
        int ok = 1;
        for (int k = 0; k < RMH_WORDS; ++k) {
            if (!rmh_word_ok(words[i + k], kRmhMask[k], kRmhValue[k])) {
                ok = 0;
                break;
            }
        }
        if (ok) return (long)(i * 4);
    }
    return -1;
}

// Scan all hits. Writes byte offsets into `out` (cap `out_cap`), returns total count.
static inline long rmh_scan_all(const uint32_t *words, size_t nwords,
                                long *out, long out_cap) {
    long total = 0;
    if (nwords < RMH_WORDS) return 0;
    for (size_t i = 0; i + RMH_WORDS <= nwords; ++i) {
        int ok = 1;
        for (int k = 0; k < RMH_WORDS; ++k) {
            if (!rmh_word_ok(words[i + k], kRmhMask[k], kRmhValue[k])) {
                ok = 0;
                break;
            }
        }
        if (ok) {
            if (total < out_cap) out[total] = (long)(i * 4);
            ++total;
        }
    }
    return total;
}

#ifdef __cplusplus
}
#endif
