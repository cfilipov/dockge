import { createHash } from "node:crypto";

const ROOT_SEED = "portge-mock-v1";

/**
 * SHA-256 hex digest of parts joined by null bytes.
 */
export function hashToSeed(parts: string[]): string {
    return createHash("sha256").update(parts.join("\0")).digest("hex");
}

/**
 * 64 hex-char deterministic ID from seed + purpose.
 */
export function deterministicId(seed: string, purpose: string): string {
    return createHash("sha256").update(seed + purpose).digest("hex");
}

/**
 * Deterministic MAC address in Docker's locally-administered range: 02:42:xx:xx:xx:xx
 */
export function deterministicMac(seed: string): string {
    const hash = createHash("sha256").update(seed).digest();
    return `02:42:${hash[0].toString(16).padStart(2, "0")}:${hash[1].toString(16).padStart(2, "0")}:${hash[2].toString(16).padStart(2, "0")}:${hash[3].toString(16).padStart(2, "0")}`;
}

/**
 * Deterministic IP within a CIDR subnet.
 * E.g. deterministicIp(seed, "172.18.0.0/16") → "172.18.x.x"
 */
export function deterministicIp(seed: string, subnet: string): string {
    const [base, prefixStr] = subnet.split("/");
    const prefix = parseInt(prefixStr, 10);
    const hostBits = 32 - prefix;
    const maxHost = (1 << hostBits) - 1; // e.g. 65535 for /16

    const hash = createHash("sha256").update(seed).digest();
    const hashVal = hash.readUInt32BE(0);
    // Map to host range [1, maxHost-1] to avoid network and broadcast addresses
    const hostPart = (hashVal % (maxHost - 1)) + 1;

    const baseParts = base.split(".").map(Number);
    const baseInt = ((baseParts[0] << 24) | (baseParts[1] << 16) | (baseParts[2] << 8) | baseParts[3]) >>> 0;
    const ipInt = (baseInt | hostPart) >>> 0;

    return `${(ipInt >>> 24) & 0xff}.${(ipInt >>> 16) & 0xff}.${(ipInt >>> 8) & 0xff}.${ipInt & 0xff}`;
}

/**
 * Deterministic ISO 8601 timestamp: base date + deterministic offset (0–86400s).
 */
export function deterministicTimestamp(seed: string, base: string): string {
    const hash = createHash("sha256").update(seed).digest();
    const offsetSeconds = hash.readUInt32BE(0) % 86400;
    const date = new Date(base);
    date.setSeconds(date.getSeconds() + offsetSeconds);
    return date.toISOString();
}

/**
 * Deterministic integer in [min, max] inclusive.
 */
export function deterministicInt(seed: string, min: number, max: number): number {
    const hash = createHash("sha256").update(seed).digest();
    const val = hash.readUInt32BE(0);
    return min + (val % (max - min + 1));
}

// --- Seed hierarchy ---

export function projectSeed(project: string): string {
    return hashToSeed([ROOT_SEED, "project", project]);
}

export function serviceSeed(project: string, service: string): string {
    return hashToSeed([ROOT_SEED, "service", project, service]);
}

export function networkSeed(networkName: string): string {
    return hashToSeed([ROOT_SEED, "network", networkName]);
}

export function imageSeed(imageRef: string): string {
    return hashToSeed([ROOT_SEED, "image", imageRef]);
}
