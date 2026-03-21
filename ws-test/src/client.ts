import WebSocket from "ws";

const ACK_TIMEOUT = 10_000;
const EVENT_TIMEOUT = 15_000;

interface PendingAck {
    resolve: (data: Record<string, unknown>) => void;
    reject: (err: Error) => void;
}

interface EventWaiter {
    event: string;
    resolve: (data: Record<string, unknown>) => void;
    reject: (err: Error) => void;
}

interface BinaryWaiter {
    resolve: (data: Buffer) => void;
    reject: (err: Error) => void;
}

interface QueuedEvent {
    event: string;
    data: Record<string, unknown>;
}

export class TestClient {
    private ws: WebSocket;
    private nextId = 1;
    private pendingAcks = new Map<number, PendingAck>();
    private eventWaiters: EventWaiter[] = [];
    private binaryWaiters: BinaryWaiter[] = [];
    private eventQueue: QueuedEvent[] = [];
    private binaryQueue: Buffer[] = [];
    private closeWaiters: Array<{ resolve: () => void; reject: (err: Error) => void }> = [];
    private closed = false;

    private constructor(ws: WebSocket) {
        this.ws = ws;
        this.ws.on("message", (raw: Buffer, isBinary: boolean) => {
            if (isBinary) {
                this.handleBinary(raw);
                return;
            }
            this.handleText(raw.toString());
        });
        this.ws.on("close", () => {
            this.closed = true;
            for (const w of this.closeWaiters) {
                w.resolve();
            }
            this.closeWaiters = [];
        });
        this.ws.on("error", () => {
            this.closed = true;
            for (const w of this.closeWaiters) {
                w.resolve();
            }
            this.closeWaiters = [];
        });
    }

    static async connect(url = "ws://localhost:5053/ws"): Promise<TestClient> {
        return new Promise((resolve, reject) => {
            const ws = new WebSocket(url);
            ws.once("open", () => resolve(new TestClient(ws)));
            ws.once("error", reject);
        });
    }

    private handleText(text: string): void {
        let msg: Record<string, unknown>;
        try {
            msg = JSON.parse(text);
        } catch {
            return;
        }

        // Ack response: has numeric "id" matching a pending request
        if (typeof msg.id === "number" && this.pendingAcks.has(msg.id)) {
            const pending = this.pendingAcks.get(msg.id)!;
            this.pendingAcks.delete(msg.id);
            pending.resolve(msg.data as Record<string, unknown>);
            return;
        }

        // Push event: has "event" field
        if (typeof msg.event === "string") {
            let data = (msg.data ?? {}) as Record<string, unknown>;

            // Unwrap ChannelBroadcast wrapper (if data has "items" key, return items)
            if (data.items && typeof data.items === "object" && !Array.isArray(data.items)) {
                data = data.items as Record<string, unknown>;
            }

            const evt: QueuedEvent = { event: msg.event as string, data };

            // Check if any waiter matches
            const idx = this.eventWaiters.findIndex((w) => w.event === evt.event);
            if (idx >= 0) {
                const waiter = this.eventWaiters.splice(idx, 1)[0];
                waiter.resolve(evt.data);
                return;
            }

            // Queue for later
            this.eventQueue.push(evt);
        }
    }

    private handleBinary(data: Buffer): void {
        if (this.binaryWaiters.length > 0) {
            const waiter = this.binaryWaiters.shift()!;
            waiter.resolve(data);
            return;
        }
        this.binaryQueue.push(data);
    }

    async sendAndReceive(event: string, ...args: unknown[]): Promise<Record<string, unknown>> {
        const id = this.nextId++;
        const msg = JSON.stringify({ id, event, args });
        this.ws.send(msg);

        return new Promise((resolve, reject) => {
            const timer = setTimeout(() => {
                this.pendingAcks.delete(id);
                reject(new Error(`Timeout waiting for ack of "${event}" (id=${id})`));
            }, ACK_TIMEOUT);

            this.pendingAcks.set(id, {
                resolve: (data) => {
                    clearTimeout(timer);
                    resolve(data);
                },
                reject: (err) => {
                    clearTimeout(timer);
                    reject(err);
                },
            });
        });
    }

    sendEvent(event: string, ...args: unknown[]): void {
        const msg = JSON.stringify({ event, args });
        this.ws.send(msg);
    }

    sendBinary(data: Buffer): void {
        this.ws.send(data);
    }

    async waitForEvent(eventName: string, timeout = EVENT_TIMEOUT): Promise<Record<string, unknown>> {
        // Check queue first
        const idx = this.eventQueue.findIndex((e) => e.event === eventName);
        if (idx >= 0) {
            const evt = this.eventQueue.splice(idx, 1)[0];
            return evt.data;
        }

        return new Promise((resolve, reject) => {
            const timer = setTimeout(() => {
                const wIdx = this.eventWaiters.findIndex((w) => w.resolve === resolveRef);
                if (wIdx >= 0) this.eventWaiters.splice(wIdx, 1);
                reject(new Error(`Timeout waiting for event "${eventName}"`));
            }, timeout);

            const resolveRef = (data: Record<string, unknown>) => {
                clearTimeout(timer);
                resolve(data);
            };

            this.eventWaiters.push({
                event: eventName,
                resolve: resolveRef,
                reject: (err) => {
                    clearTimeout(timer);
                    reject(err);
                },
            });
        });
    }

    async waitForBinary(timeout = EVENT_TIMEOUT): Promise<Buffer> {
        if (this.binaryQueue.length > 0) {
            return this.binaryQueue.shift()!;
        }

        return new Promise((resolve, reject) => {
            const timer = setTimeout(() => {
                const idx = this.binaryWaiters.findIndex((w) => w.resolve === resolveRef);
                if (idx >= 0) this.binaryWaiters.splice(idx, 1);
                reject(new Error("Timeout waiting for binary frame"));
            }, timeout);

            const resolveRef = (data: Buffer) => {
                clearTimeout(timer);
                resolve(data);
            };

            this.binaryWaiters.push({
                resolve: resolveRef,
                reject: (err) => {
                    clearTimeout(timer);
                    reject(err);
                },
            });
        });
    }

    async login(username = "admin", password = "testpass123"): Promise<string> {
        const resp = await this.sendAndReceive("login", username, password, "", "");
        if (!resp.ok) {
            throw new Error(`Login failed: ${JSON.stringify(resp)}`);
        }
        return resp.token as string;
    }

    async waitForClose(timeout = 5000): Promise<void> {
        if (this.closed) return;

        return new Promise((resolve, reject) => {
            const timer = setTimeout(() => {
                const idx = this.closeWaiters.findIndex((w) => w.resolve === resolveRef);
                if (idx >= 0) this.closeWaiters.splice(idx, 1);
                reject(new Error("Timeout waiting for connection close"));
            }, timeout);

            const resolveRef = () => {
                clearTimeout(timer);
                resolve();
            };

            this.closeWaiters.push({
                resolve: resolveRef,
                reject: (err) => {
                    clearTimeout(timer);
                    reject(err);
                },
            });
        });
    }

    async tryWaitForEvent(eventName: string, timeout: number): Promise<Record<string, unknown> | null> {
        try {
            return await this.waitForEvent(eventName, timeout);
        } catch {
            return null;
        }
    }

    async collectEvents(eventName: string, durationMs: number): Promise<Record<string, unknown>[]> {
        const results: Record<string, unknown>[] = [];
        const deadline = Date.now() + durationMs;
        while (Date.now() < deadline) {
            const remaining = deadline - Date.now();
            if (remaining <= 0) break;
            const evt = await this.tryWaitForEvent(eventName, remaining);
            if (evt === null) break;
            results.push(evt);
        }
        return results;
    }

    close(): void {
        if (!this.closed) {
            this.ws.close();
            this.closed = true;
        }
    }

    get isClosed(): boolean {
        return this.closed;
    }
}
