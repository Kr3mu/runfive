/**
 * Dashboard layout codec — encodes/decodes widget layouts to compact URL-safe strings.
 *
 * Format:
 *   Char 0: format version (single base62 digit)
 *   Char 1: widget count (single base62 digit, 1-8)
 *   Char 2+: bit-packed widget data, 21 bits per widget
 *           (5 bits type ID + 4 bits x + 4 bits y + 4 bits w + 4 bits h),
 *           base62-encoded as a single big-endian unsigned integer over
 *           ceil(count*21/8) bytes
 *
 * The version+count prefix gives the decoder an unambiguous byte length,
 * which the bigint round-trip cannot recover on its own (leading zeros
 * are not preserved across base62 conversion).
 *
 * Resulting string lengths (incl. 2-char prefix):
 *   1 widget  ->   6 chars
 *   2 widgets ->  10 chars
 *   3 widgets ->  13 chars
 *   4 widgets ->  17 chars
 *   8 widgets ->  31 chars
 */

import type { GridLayoutItem } from "$lib/types/grid-layout";

// -- Widget type registry ----------------------------------------------
// Order matters — indices are the encoded IDs. Append new types at the end.
const WIDGET_TYPES = [
    "console",
    "players",
    "stats",
    "bans",
    "chat",
    "cpu",
    "ram",
    "disk",
    "network",
    "tickrate",
    "map",
    "logs",
    "plugins",
    "whitelist",
    "scheduler",
    "backups",
    "world",
    "events",
    "rules",
    "motd",
] as const;

const typeToId = new Map<string, number>(
    WIDGET_TYPES.map((t, i) => [t, i]),
);
const idToType = WIDGET_TYPES;

const FORMAT_VERSION = 0;
const BITS_PER_WIDGET = 21; // 5 + 4 + 4 + 4 + 4
const MAX_WIDGETS = 8;

// -- Base62 ------------------------------------------------------------
const BASE62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz";
const BASE = BigInt(BASE62.length); // 62n

function bytesToBase62(bytes: Uint8Array): string {
    // Interpret bytes as a big-endian unsigned integer
    let n = 0n;
    for (const b of bytes) {
        n = (n << 8n) | BigInt(b);
    }
    if (n === 0n) return BASE62[0];
    const chars: string[] = [];
    while (n > 0n) {
        chars.push(BASE62[Number(n % BASE)]);
        n = n / BASE;
    }
    return chars.reverse().join("");
}

function base62ToBytes(s: string, byteLen: number): Uint8Array {
    let n = 0n;
    for (const ch of s) {
        const idx = BASE62.indexOf(ch);
        if (idx === -1) throw new Error(`Invalid base62 character: '${ch}'`);
        n = n * BASE + BigInt(idx);
    }
    const bytes = new Uint8Array(byteLen);
    for (let i = byteLen - 1; i >= 0; i--) {
        bytes[i] = Number(n & 0xffn);
        n = n >> 8n;
    }
    return bytes;
}

// -- Bit writer / reader -----------------------------------------------
class BitWriter {
    private bits: number[] = [];

    write(value: number, numBits: number): void {
        for (let i = numBits - 1; i >= 0; i--) {
            this.bits.push((value >> i) & 1);
        }
    }

    toBytes(): Uint8Array {
        // Pad to byte boundary
        while (this.bits.length % 8 !== 0) {
            this.bits.push(0);
        }
        const bytes = new Uint8Array(this.bits.length / 8);
        for (let i = 0; i < bytes.length; i++) {
            let byte = 0;
            for (let b = 0; b < 8; b++) {
                byte = (byte << 1) | this.bits[i * 8 + b];
            }
            bytes[i] = byte;
        }
        return bytes;
    }
}

class BitReader {
    private pos = 0;

    constructor(private bytes: Uint8Array) {}

    read(numBits: number): number {
        let value = 0;
        for (let i = 0; i < numBits; i++) {
            const byteIdx = Math.floor(this.pos / 8);
            const bitIdx = 7 - (this.pos % 8);
            value = (value << 1) | ((this.bytes[byteIdx] >> bitIdx) & 1);
            this.pos++;
        }
        return value;
    }
}

// -- Public API --------------------------------------------------------

export function encodeLayout(items: GridLayoutItem[]): string {
    if (items.length === 0 || items.length > MAX_WIDGETS) {
        throw new Error(`Widget count must be 1-${MAX_WIDGETS}, got ${items.length}`);
    }

    const writer = new BitWriter();
    for (const item of items) {
        const tid = typeToId.get(item.id);
        if (tid === undefined) {
            throw new Error(`Unknown widget type: '${item.id}'`);
        }
        writer.write(tid, 5);           // 0-31 (20 types used)
        writer.write(item.x, 4);        // 0-11
        writer.write(item.y, 4);        // 0-11
        writer.write(item.w - 1, 4);    // 1-12 -> 0-11
        writer.write(item.h - 1, 4);    // 1-12 -> 0-11
    }

    // Two-char prefix gives the decoder an unambiguous byte length:
    // base62 of a bigint silently drops leading zeros, so the data length
    // cannot be recovered from the encoded string alone.
    return BASE62[FORMAT_VERSION] + BASE62[items.length] + bytesToBase62(writer.toBytes());
}

export function decodeLayout(code: string): GridLayoutItem[] {
    if (code.length < 2) {
        throw new Error("layout code too short");
    }
    const version = BASE62.indexOf(code[0]);
    if (version !== FORMAT_VERSION) {
        throw new Error(`unsupported layout format version: ${version}`);
    }
    const count = BASE62.indexOf(code[1]);
    if (count < 1 || count > MAX_WIDGETS) {
        throw new Error(`invalid widget count: ${count}`);
    }

    const dataBytes = Math.ceil((count * BITS_PER_WIDGET) / 8);
    const bytes = base62ToBytes(code.slice(2), dataBytes);
    const reader = new BitReader(bytes);

    const items: GridLayoutItem[] = [];
    for (let i = 0; i < count; i++) {
        const tid = reader.read(5);
        const x = reader.read(4);
        const y = reader.read(4);
        const w = reader.read(4) + 1;
        const h = reader.read(4) + 1;

        if (tid >= idToType.length) {
            throw new Error(`Unknown widget type ID: ${tid}`);
        }

        items.push({ id: idToType[tid], x, y, w, h });
    }

    return items;
}

/**
 * Generate a shareable URL fragment for the current layout.
 * Example: /dashboard?l=A3kBx9Qm
 */
export function layoutToParam(items: GridLayoutItem[]): string {
    return encodeLayout(items);
}

/**
 * Parse a layout code from a URL parameter.
 * Returns null if the code is invalid.
 */
export function paramToLayout(code: string): GridLayoutItem[] | null {
    try {
        return decodeLayout(code);
    } catch {
        return null;
    }
}
