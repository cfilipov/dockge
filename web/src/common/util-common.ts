/*
 * Common utilities for backend and frontend
 */
import yaml, { Document, Pair, Scalar } from "yaml";

// Init dayjs
import dayjs from "dayjs";
import timezone from "dayjs/plugin/timezone";
import utc from "dayjs/plugin/utc";
import relativeTime from "dayjs/plugin/relativeTime";
// @ts-ignore
import { replaceVariablesSync } from "@inventage/envsubst";

dayjs.extend(utc);
dayjs.extend(timezone);
dayjs.extend(relativeTime);

export interface LooseObject {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    [key: string]: any
}

export interface BaseRes {
    ok: boolean;
    msg?: string;
}

function randomBytes(numBytes: number): Uint8Array {
    const bytes = new Uint8Array(numBytes);
    for (let i = 0; i < numBytes; i += 65536) {
        crypto.getRandomValues(bytes.subarray(i, i + Math.min(numBytes - i, 65536)));
    }
    return bytes;
}

// Stack Status
export const UNKNOWN = 0;
export const CREATED_FILE = 1;
export const CREATED_STACK = 2;
export const RUNNING = 3;
export const EXITED = 4;
export const RUNNING_AND_EXITED = 5;
export const UNHEALTHY = 6;

/**
 * Stack status info registry — maps status IDs to display properties.
 * Replaces the older statusName/statusNameShort/statusColor functions.
 */
export class StackStatusInfo {

    private static INFOS_BY_ID = new Map<number, StackStatusInfo>();
    private static DEFAULT = new StackStatusInfo("?", [], "secondary", "secondary");
    static ALL: StackStatusInfo[] = [];

    static {
        this.addInfo(new StackStatusInfo("unhealthy", [ UNHEALTHY ], "danger", "danger"));
        this.addInfo(new StackStatusInfo("active", [ RUNNING ], "primary", "primary"));
        this.addInfo(new StackStatusInfo("partially", [ RUNNING_AND_EXITED ], "info", "info"));
        this.addInfo(new StackStatusInfo("exited", [ EXITED ], "warning", "warning"));
        this.addInfo(new StackStatusInfo("down", [ CREATED_FILE, CREATED_STACK ], "dark", "secondary"));
    }

    private static addInfo(info: StackStatusInfo) {
        for (const id of info.statusIds) {
            this.INFOS_BY_ID.set(id, info);
        }
        this.ALL.push(info);
    }

    static get(statusId: number) {
        return this.INFOS_BY_ID.get(statusId) ?? this.DEFAULT;
    }

    constructor(readonly label: string, readonly statusIds: number[], readonly badgeColor: string, readonly textColor: string) {}
}

/**
 * Stack filter state for the dashboard stack list
 */
export class StackFilter {
    status = new StackFilterCategory<string>("status");
    attributes = new StackFilterCategory<string>("attribute");

    categories = [ this.status, this.attributes ];

    isFilterSelected() {
        for (const category of this.categories) {
            if (category.isFilterSelected()) {
                return true;
            }
        }
        return false;
    }

    clear() {
        for (const category of this.categories) {
            category.selected.clear();
        }
    }
}

export class StackFilterCategory<T> {
    options: Record<string, T> = {};
    selected: Set<T> = new Set();

    constructor(readonly label: string) { }

    hasOptions() {
        return Object.keys(this.options).length > 0;
    }

    isFilterSelected() {
        if (this.selected.size === 0) {
            return false;
        }

        for (const ov of Object.values(this.options)) {
            if (this.selected.has(ov)) {
                return true;
            }
        }

        return false;
    }

    toggleSelected(value: T) {
        if (this.selected.has(value)) {
            this.selected.delete(value);
        } else {
            this.selected.add(value);
        }
    }
}

/** @deprecated Use StackStatusInfo.get(status).label instead */
export function statusName(status : number) : string {
    switch (status) {
        case CREATED_FILE:
            return "draft";
        case CREATED_STACK:
            return "created_stack";
        case RUNNING:
            return "running";
        case RUNNING_AND_EXITED:
            return "partially_running";
        case UNHEALTHY:
            return "unhealthy";
        case EXITED:
            return "exited";
        default:
            return "unknown";
    }
}

/** @deprecated Use StackStatusInfo.get(status).label instead */
export function statusNameShort(status : number) : string {
    switch (status) {
        case CREATED_FILE:
            return "down";
        case CREATED_STACK:
            return "down";
        case RUNNING:
            return "active";
        case RUNNING_AND_EXITED:
            return "partially";
        case UNHEALTHY:
            return "unhealthy";
        case EXITED:
            return "exited";
        default:
            return "?";
    }
}

/** @deprecated Use StackStatusInfo.get(status).badgeColor instead */
export function statusColor(status : number) : string {
    switch (status) {
        case CREATED_FILE:
            return "dark";
        case CREATED_STACK:
            return "dark";
        case RUNNING:
            return "primary";
        case RUNNING_AND_EXITED:
            return "info";
        case UNHEALTHY:
            return "danger";
        case EXITED:
            return "warning";
        default:
            return "secondary";
    }
}

/**
 * Container status info — maps Docker container state/health to display properties.
 * Uses actual Docker state strings as labels (not stack status labels).
 */
export class ContainerStatusInfo {
    static readonly RUNNING = new ContainerStatusInfo("running", "primary");
    static readonly UNHEALTHY = new ContainerStatusInfo("unhealthy", "danger");
    static readonly EXITED = new ContainerStatusInfo("exited", "warning");
    static readonly PAUSED = new ContainerStatusInfo("paused", "info");
    static readonly CREATED = new ContainerStatusInfo("created", "dark");
    static readonly DEAD = new ContainerStatusInfo("dead", "dark");
    static readonly UNKNOWN = new ContainerStatusInfo("down", "dark");

    static ALL = [this.RUNNING, this.UNHEALTHY, this.EXITED, this.PAUSED, this.CREATED, this.DEAD];

    constructor(readonly label: string, readonly badgeColor: string) {}

    /** Map split container state/health fields to a ContainerStatusInfo. */
    static from(c: { state: string; health?: string }): ContainerStatusInfo {
        if (c.state === "running" && c.health === "unhealthy") return this.UNHEALTHY;
        if (c.state === "running") return this.RUNNING;
        if (c.state === "exited") return this.EXITED;
        if (c.state === "dead") return this.DEAD;
        if (c.state === "paused") return this.PAUSED;
        if (c.state === "created") return this.CREATED;
        return this.UNKNOWN;
    }

    /** Map a combined status string (from serviceStatusList) to a ContainerStatusInfo. */
    static fromStatus(status: string): ContainerStatusInfo {
        if (status === "healthy") return this.RUNNING;
        if (status === "unhealthy") return this.UNHEALTHY;
        return this.from({ state: status });
    }
}

export const isDev = import.meta.env.DEV;
export const TERMINAL_COLS = 105;
export const TERMINAL_ROWS = 10;
export const PROGRESS_TERMINAL_ROWS = 8;

export const COMBINED_TERMINAL_COLS = 58;
export const COMBINED_TERMINAL_ROWS = 20;

export const ERROR_TYPE_VALIDATION = 1;

export const acceptedComposeFileNames = [
    "compose.yaml",
    "docker-compose.yaml",
    "docker-compose.yml",
    "compose.yml",
];

export const acceptedComposeOverrideFileNames = [
    "compose.override.yaml",
    "compose.override.yml",
    "docker-compose.override.yaml",
    "docker-compose.override.yml",
];

/**
 * Generate a decimal integer number from a string
 * @param str Input
 * @param length Default is 10 which means 0 - 9
 */
export function intHash(str : string, length = 10) : number {
    // A simple hashing function (you can use more complex hash functions if needed)
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
        hash += str.charCodeAt(i);
    }
    // Normalize the hash to the range [0, 10]
    return (hash % length + length) % length; // Ensure the result is non-negative
}

/**
 * Delays for specified number of seconds
 * @param ms Number of milliseconds to sleep for
 */
export function sleep(ms: number) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

export function isRecord(obj: unknown): obj is Record<string, unknown> {
    return typeof obj === "object" && obj !== null;
}

export function getNested<T>(
    obj: unknown,
    keys: string[]
): T | undefined {
    let current = obj;
    for (const key of keys) {
        if (!isRecord(current) || !(key in current)) {
            return undefined;
        }
        current = current[key];
    }
    return current as T;
}

/**
 * Generate a random alphanumeric string of fixed length
 * @param length Length of string to generate
 * @returns string
 */
export function genSecret(length = 64) {
    let secret = "";
    const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
    const charsLength = chars.length;
    for ( let i = 0; i < length; i++ ) {
        secret += chars.charAt(getCryptoRandomInt(0, charsLength - 1));
    }
    return secret;
}

/**
 * Get a random integer suitable for use in cryptography between upper
 * and lower bounds.
 * @param min Minimum value of integer
 * @param max Maximum value of integer
 * @returns Cryptographically suitable random integer
 */
export function getCryptoRandomInt(min: number, max: number):number {
    // synchronous version of: https://github.com/joepie91/node-random-number-csprng

    const range = max - min;
    if (range >= Math.pow(2, 32)) {
        console.log("Warning! Range is too large.");
    }

    let tmpRange = range;
    let bitsNeeded = 0;
    let bytesNeeded = 0;
    let mask = 1;

    while (tmpRange > 0) {
        if (bitsNeeded % 8 === 0) {
            bytesNeeded += 1;
        }
        bitsNeeded += 1;
        mask = mask << 1 | 1;
        tmpRange = tmpRange >>> 1;
    }

    const bytes = randomBytes(bytesNeeded);
    let randomValue = 0;

    for (let i = 0; i < bytesNeeded; i++) {
        randomValue |= bytes[i] << 8 * i;
    }

    randomValue = randomValue & mask;

    if (randomValue <= range) {
        return min + randomValue;
    } else {
        return getCryptoRandomInt(min, max);
    }
}

export function getComposeTerminalName(stack : string) {
    return "compose-" + stack;
}

export function getCombinedTerminalName(stack : string) {
    return "combined-" + stack;
}

export function getContainerTerminalName(stackName : string, container : string, shell: string, index : number) {
    return "container-terminal-" + stackName + "-" + container + "-" + shell + "-" + index;
}

export function getContainerExecTerminalName(stackName : string, container : string, index : number) {
    return "container-exec-" + stackName + "-" + container + "-" + index;
}

export function getContainerLogName(stackName : string, container : string) {
    return "container-log-" + container;
}

export function copyYAMLComments(doc : Document, src : Document) {
    doc.comment = src.comment;
    doc.commentBefore = src.commentBefore;

    if (doc && doc.contents && src && src.contents) {
        // @ts-ignore
        copyYAMLCommentsItems(doc.contents.items, src.contents.items);
    }
}

/**
 * Copy yaml comments from srcItems to items
 * Attempts to preserve comments by matching content rather than just array indices
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function copyYAMLCommentsItems(items: any, srcItems: any) {
    if (!items || !srcItems) {
        return;
    }

    // Pre-compute source item identities to avoid repeated JSON.stringify in O(n²) loop
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const srcIdentities = srcItems.map((srcItem: any) => ({
        value: String(srcItem.value),
        key: String(srcItem.key),
    }));

    // First pass - try to match items by their content
    for (let i = 0; i < items.length; i++) {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const item: any = items[i];

        // Try to find matching source item by content
        const itemValue = String(item.value);
        const itemKey = String(item.key);
        const srcIndex = srcIdentities.findIndex((id: { value: string; key: string }) =>
            id.value === itemValue && id.key === itemKey
        );

        if (srcIndex !== -1) {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const srcItem: any = srcItems[srcIndex];
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const nextSrcItem: any = srcItems[srcIndex + 1];

            if (item.key && srcItem.key) {
                item.key.comment = srcItem.key.comment;
                item.key.commentBefore = srcItem.key.commentBefore;
            }

            if (srcItem.comment) {
                item.comment = srcItem.comment;
            }

            // Handle comments between array items
            if (nextSrcItem && nextSrcItem.commentBefore) {
                if (items[i + 1]) {
                    items[i + 1].commentBefore = nextSrcItem.commentBefore;
                }
            }

            // Handle trailing comments after array items
            if (srcItem.value && srcItem.value.comment) {
                if (item.value) {
                    item.value.comment = srcItem.value.comment;
                }
            }

            if (item.value && srcItem.value) {
                if (typeof item.value === "object" && typeof srcItem.value === "object") {
                    item.value.comment = srcItem.value.comment;
                    item.value.commentBefore = srcItem.value.commentBefore;

                    if (item.value.items && srcItem.value.items) {
                        copyYAMLCommentsItems(item.value.items, srcItem.value.items);
                    }
                }
            }
        }
    }
}

/**
 * Possible Inputs:
 * ports:
 *   - "3000"
 *   - "3000-3005"
 *   - "8000:8000"
 *   - "9090-9091:8080-8081"
 *   - "49100:22"
 *   - "8000-9000:80"
 *   - "127.0.0.1:8001:8001"
 *   - "127.0.0.1:5000-5010:5000-5010"
 *   - "0.0.0.0:8080->8080/tcp"
 *   - "6060:6060/udp"
 * @param input
 * @param hostname
 */
export function parseDockerPort(input : string | number | Record<string, unknown>, hostname : string) {
    // Long-syntax port object: { target, published, protocol, host_ip }
    if (typeof input === "object" && input !== null) {
        const obj = input as Record<string, unknown>;
        const published = obj.published ?? obj.target;
        const target = obj.target ?? published;
        const proto = obj.protocol ? `/${obj.protocol}` : "";
        const hostIp = obj.host_ip ? `${obj.host_ip}:` : "";
        input = `${hostIp}${published}:${target}${proto}`;
    } else if (typeof input === "number") {
        input = String(input);
    }

    let port;
    let display;

    const parts = input.split("/");
    let part1 = parts[0];
    let protocol = parts[1] || "tcp";

    // coming from docker ps, split host part
    const arrow = part1.indexOf("->");
    if (arrow >= 0) {
        part1 = part1.split("->")[0];
        const colon = part1.indexOf(":");
        if (colon >= 0) {
            part1 = part1.split(":")[1];
        }
    }

    // Split the last ":"
    const lastColon = part1.lastIndexOf(":");

    if (lastColon === -1) {
        // No colon, so it's just a port or port range
        // Check if it's a port range
        const dash = part1.indexOf("-");
        if (dash === -1) {
            // No dash, so it's just a port
            port = part1;
        } else {
            // Has dash, so it's a port range, use the first port
            port = part1.substring(0, dash);
        }

        display = part1;

    } else {
        // Has colon, so it's a port mapping
        let hostPart = part1.substring(0, lastColon);
        display = hostPart;

        // Check if it's a port range
        const dash = part1.indexOf("-");

        if (dash !== -1) {
            // Has dash, so it's a port range, use the first port
            hostPart = part1.substring(0, dash);
        }

        // Check if it has a ip (ip:port)
        const colon = hostPart.indexOf(":");

        if (colon !== -1) {
            // Has colon, so it's a ip:port
            hostname = hostPart.substring(0, colon);
            port = hostPart.substring(colon + 1);
        } else {
            // No colon, so it's just a port
            port = hostPart;
        }
    }

    let portInt = parseInt(port);

    if (portInt == 443) {
        protocol = "https";
    } else if (protocol === "tcp") {
        protocol = "http";
    }

    return {
        url: protocol + "://" + hostname + ":" + portInt,
        display: display,
    };
}

/**
 * Parse a .env file string into key-value pairs.
 * Handles comments, empty lines, quoted values, and export prefix.
 */
export function parseEnv(src: string): Record<string, string> {
    const result: Record<string, string> = {};
    for (const line of src.split("\n")) {
        const trimmed = line.trim();
        if (!trimmed || trimmed.startsWith("#")) continue;

        // Strip optional "export " prefix
        const assign = trimmed.startsWith("export ") ? trimmed.slice(7) : trimmed;
        const eq = assign.indexOf("=");
        if (eq === -1) continue;

        const key = assign.substring(0, eq).trim();
        let value = assign.substring(eq + 1).trim();

        // Remove surrounding quotes (single, double, or backtick)
        if (value.length >= 2) {
            const q = value[0];
            if ((q === '"' || q === "'" || q === "`") && value.endsWith(q)) {
                value = value.slice(1, -1);
            }
        }
        result[key] = value;
    }
    return result;
}

export function envsubst(string : string, variables : LooseObject) : string {
    return replaceVariablesSync(string, variables)[0];
}

/**
 * Traverse all values in the yaml and for each value, if there are template variables, replace it environment variables
 * Emulates the behavior of how docker-compose handles environment variables in yaml files
 * @param content Yaml string
 * @param env Environment variables
 * @returns string Yaml string with environment variables replaced
 */
export function envsubstYAML(content : string, env : Record<string, string>) : string {
    const doc = yaml.parseDocument(content);
    if (doc.contents) {
        // @ts-ignore
        for (const item of doc.contents.items) {
            traverseYAML(item, env);
        }
    }
    return doc.toString();
}

/**
 * Used for envsubstYAML(...)
 * @param pair
 * @param env
 */
function traverseYAML(pair : Pair, env : Record<string, string>) : void {
    // @ts-ignore
    if (pair.value && pair.value.items) {
        // @ts-ignore
        for (const item of pair.value.items) {
            if (item instanceof Pair) {
                traverseYAML(item, env);
            } else if (item instanceof Scalar) {
                let value = item.value as unknown;

                if (typeof(value) === "string") {
                    item.value = envsubst(value, env);
                }
            }
        }
    // @ts-ignore
    } else if (pair.value && typeof(pair.value.value) === "string") {
        // @ts-ignore
        pair.value.value = envsubst(pair.value.value, env);
    }
}
