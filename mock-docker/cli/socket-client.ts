import { request as httpRequest, type IncomingMessage } from "node:http";

export interface Response {
    statusCode: number;
    headers: Record<string, string | string[] | undefined>;
    body: string;
}

/**
 * Make an HTTP request over a Unix socket.
 */
export function request(
    socketPath: string,
    method: string,
    path: string,
    body?: unknown,
): Promise<Response> {
    return new Promise((resolve, reject) => {
        const bodyStr = body !== undefined ? JSON.stringify(body) : undefined;
        const headers: Record<string, string> = {};
        if (bodyStr) {
            headers["Content-Type"] = "application/json";
            headers["Content-Length"] = String(Buffer.byteLength(bodyStr));
        }

        const req = httpRequest(
            {
                socketPath,
                method,
                path,
                headers,
            },
            (res: IncomingMessage) => {
                const chunks: Buffer[] = [];
                res.on("data", (chunk: Buffer) => chunks.push(chunk));
                res.on("end", () => {
                    resolve({
                        statusCode: res.statusCode || 0,
                        headers: res.headers as Record<string, string | string[] | undefined>,
                        body: Buffer.concat(chunks).toString(),
                    });
                });
            },
        );

        req.on("error", (err) => {
            reject(err);
        });

        if (bodyStr) req.write(bodyStr);
        req.end();
    });
}

export interface RawResponse {
    statusCode: number;
    headers: Record<string, string | string[] | undefined>;
    body: Buffer;
}

/**
 * Make an HTTP request over a Unix socket, returning the body as a raw Buffer.
 * Use this instead of request() when the response may contain binary data
 * (e.g. Docker multiplexed streams).
 */
export function requestRaw(
    socketPath: string,
    method: string,
    path: string,
    body?: unknown,
): Promise<RawResponse> {
    return new Promise((resolve, reject) => {
        const bodyStr = body !== undefined ? JSON.stringify(body) : undefined;
        const headers: Record<string, string> = {};
        if (bodyStr) {
            headers["Content-Type"] = "application/json";
            headers["Content-Length"] = String(Buffer.byteLength(bodyStr));
        }

        const req = httpRequest(
            {
                socketPath,
                method,
                path,
                headers,
            },
            (res: IncomingMessage) => {
                const chunks: Buffer[] = [];
                res.on("data", (chunk: Buffer) => chunks.push(chunk));
                res.on("end", () => {
                    resolve({
                        statusCode: res.statusCode || 0,
                        headers: res.headers as Record<string, string | string[] | undefined>,
                        body: Buffer.concat(chunks),
                    });
                });
            },
        );

        req.on("error", (err) => {
            reject(err);
        });

        if (bodyStr) req.write(bodyStr);
        req.end();
    });
}

/**
 * Make an HTTP request and parse the JSON response.
 */
export async function requestJSON<T = unknown>(
    socketPath: string,
    method: string,
    path: string,
    body?: unknown,
): Promise<{ statusCode: number; data: T }> {
    const res = await request(socketPath, method, path, body);
    let data: T;
    try {
        data = JSON.parse(res.body) as T;
    } catch {
        data = res.body as unknown as T;
    }
    return { statusCode: res.statusCode, data };
}

/**
 * Stream an HTTP response line by line.
 * Used for logs --follow, events, etc.
 */
export function requestStream(
    socketPath: string,
    method: string,
    path: string,
    onData: (chunk: string) => void,
    body?: unknown,
): Promise<void> {
    return new Promise((resolve, reject) => {
        const bodyStr = body !== undefined ? JSON.stringify(body) : undefined;
        const headers: Record<string, string> = {};
        if (bodyStr) {
            headers["Content-Type"] = "application/json";
            headers["Content-Length"] = String(Buffer.byteLength(bodyStr));
        }

        const req = httpRequest(
            {
                socketPath,
                method,
                path,
                headers,
            },
            (res: IncomingMessage) => {
                res.setEncoding("utf8");
                let buffer = "";
                res.on("data", (chunk: string) => {
                    buffer += chunk;
                    const lines = buffer.split("\n");
                    buffer = lines.pop() || "";
                    for (const line of lines) {
                        if (line.trim()) onData(line);
                    }
                });
                res.on("end", () => {
                    if (buffer.trim()) onData(buffer);
                    resolve();
                });
            },
        );

        req.on("error", reject);
        if (bodyStr) req.write(bodyStr);
        req.end();
    });
}
