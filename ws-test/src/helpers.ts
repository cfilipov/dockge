import { TestClient } from "./client.js";

const BASE_URL = process.env.TEST_WS_URL ?? "ws://localhost:5053/ws";
const HTTP_BASE = process.env.TEST_HTTP_URL ?? "http://localhost:5053";

export async function resetMockState(): Promise<void> {
    const resp = await fetch(`${HTTP_BASE}/api/mock/reset`, { method: "POST" });
    if (!resp.ok) {
        throw new Error(`Failed to reset mock state: ${resp.status} ${resp.statusText}`);
    }
}

/** Reset Go backend DB state: wipes users, re-seeds admin/testpass123, clears rate limiter. */
export async function resetDB(): Promise<void> {
    const resp = await fetch(`${HTTP_BASE}/api/dev/reset-db`, { method: "POST" });
    if (!resp.ok) {
        throw new Error(`Failed to reset DB: ${resp.status} ${resp.statusText}`);
    }
}

export async function withClient<T>(fn: (client: TestClient) => Promise<T>): Promise<T> {
    const client = await TestClient.connect(BASE_URL);
    try {
        return await fn(client);
    } finally {
        client.close();
    }
}

export async function withAuthClient<T>(fn: (client: TestClient) => Promise<T>): Promise<T> {
    const client = await TestClient.connect(BASE_URL);
    try {
        await client.login();
        return await fn(client);
    } finally {
        client.close();
    }
}

export function connectClient(): Promise<TestClient> {
    return TestClient.connect(BASE_URL);
}
