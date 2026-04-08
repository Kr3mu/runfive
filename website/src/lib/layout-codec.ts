/**
 * Dashboard layout codec — encodes/decodes widget layouts to compact URL-safe strings.
 *
 * Format (bit-packed, then base62-encoded):
 *   Header:  3 bits version (currently 0) + 3 bits widget count (1-8)
 *   Per widget: 5 bits type ID + 4 bits x + 4 bits y + 4 bits w + 4 bits h = 21 bits
 *
 * Resulting string lengths:
 *   1 widget  ->  5 chars
 *   2 widgets ->  8 chars
 *   3 widgets -> 12 chars
 *   4 widgets -> 15 chars
 *   6 widgets -> 22 chars
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
const MAX_VERSION_BITS = 3;
const MAX_COUNT_BITS = 3;
const HEADER_BITS = MAX_VERSION_BITS + MAX_COUNT_BITS; // 6
const BITS_PER_WIDGET = 21; // 5 + 4 + 4 + 4 + 4

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
    if (items.length === 0 || items.length > 8) {
        throw new Error(`Widget count must be 1-8, got ${items.length}`);
    }

    const writer = new BitWriter();
    writer.write(FORMAT_VERSION, MAX_VERSION_BITS);
    writer.write(items.length - 1, MAX_COUNT_BITS); // 0-7 -> 1-8

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

    return bytesToBase62(writer.toBytes());
}

export function decodeLayout(code: string): GridLayoutItem[] {
    // Calculate expected byte length from string
    // We need to try decoding to find the count, but we know the max possible bytes
    const totalBitsMax = HEADER_BITS + 8 * BITS_PER_WIDGET;
    const maxBytes = Math.ceil(totalBitsMax / 8);
    const bytes = base62ToBytes(code, maxBytes);

    const reader = new BitReader(bytes);
    const version = reader.read(MAX_VERSION_BITS);
    if (version !== FORMAT_VERSION) {
        throw new Error(`Unsupported layout format version: ${version}`);
    }

    const count = reader.read(MAX_COUNT_BITS) + 1;

    // Re-decode with exact byte length for precision
    const exactBits = HEADER_BITS + count * BITS_PER_WIDGET;
    const exactBytes = Math.ceil(exactBits / 8);
    const exactData = base62ToBytes(code, exactBytes);
    const exactReader = new BitReader(exactData);
    exactReader.read(MAX_VERSION_BITS); // skip version
    exactReader.read(MAX_COUNT_BITS);   // skip count

    const items: GridLayoutItem[] = [];
    for (let i = 0; i < count; i++) {
        const tid = exactReader.read(5);
        const x = exactReader.read(4);
        const y = exactReader.read(4);
        const w = exactReader.read(4) + 1;
        const h = exactReader.read(4) + 1;

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
