import { request as httpRequest, type IncomingMessage } from "node:http";
import { createConnection } from "node:net";

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
 * Open a bidirectional interactive stream over a Unix socket.
 * Uses a raw TCP connection instead of http.request because Bun compiled
 * binaries buffer http.request.write() and never flush until end() — which
 * deadlocks interactive streams where the server must see the config prefix
 * before it sends a response.
 */
export function requestInteractive(
    socketPath: string,
    method: string,
    path: string,
    configPrefix?: unknown,
): Promise<void> {
    return new Promise((resolve, reject) => {
        const conn = createConnection(socketPath, () => {
            // Build raw HTTP request with chunked transfer encoding
            const configStr = configPrefix !== undefined
                ? JSON.stringify(configPrefix)
                : "";
            const reqLines = [
                `${method} ${path} HTTP/1.1`,
                "Host: localhost",
                "Content-Type: application/vnd.docker.raw-stream",
                "Transfer-Encoding: chunked",
                "Connection: Upgrade",
                "Upgrade: tcp",
                "",
                "",
            ];
            conn.write(reqLines.join("\r\n"));

            // Send config prefix as the first chunk
            if (configStr) {
                conn.write(`${configStr.length.toString(16)}\r\n${configStr}\r\n`);
            }

            // Pipe stdin → chunked body
            process.stdin.on("data", (chunk: Buffer) => {
                conn.write(`${chunk.length.toString(16)}\r\n`);
                conn.write(chunk);
                conn.write("\r\n");
            });
            process.stdin.on("end", () => {
                conn.write("0\r\n\r\n");
                conn.end();
            });
            process.stdin.resume();
        });

        // Parse response: skip HTTP headers, decode chunked body to stdout
        let headersParsed = false;
        let headerBuf = "";
        let isChunked = false;
        let chunkBuf = Buffer.alloc(0);

        function processChunkedData(data: Buffer) {
            chunkBuf = Buffer.concat([chunkBuf, data]);
            while (true) {
                const crlfIdx = chunkBuf.indexOf("\r\n");
                if (crlfIdx === -1) break;
                const sizeStr = chunkBuf.subarray(0, crlfIdx).toString().trim();
                const chunkSize = parseInt(sizeStr, 16);
                if (isNaN(chunkSize)) break;
                if (chunkSize === 0) {
                    // Final chunk — done
                    conn.end();
                    return;
                }
                // Need size line + \r\n + chunk data + \r\n
                const needed = crlfIdx + 2 + chunkSize + 2;
                if (chunkBuf.length < needed) break;
                const payload = chunkBuf.subarray(crlfIdx + 2, crlfIdx + 2 + chunkSize);
                process.stdout.write(payload);
                chunkBuf = chunkBuf.subarray(needed);
            }
        }

        conn.on("data", (chunk: Buffer) => {
            if (!headersParsed) {
                headerBuf += chunk.toString("latin1");
                const headerEnd = headerBuf.indexOf("\r\n\r\n");
                if (headerEnd === -1) return; // wait for full headers
                headersParsed = true;
                isChunked = headerBuf.toLowerCase().includes("transfer-encoding: chunked");
                // Process any body data that came with the headers
                const bodyStart = headerEnd + 4;
                if (bodyStart < headerBuf.length) {
                    const remaining = Buffer.from(
                        headerBuf.slice(bodyStart),
                        "latin1",
                    );
                    if (isChunked) {
                        processChunkedData(remaining);
                    } else {
                        process.stdout.write(remaining);
                    }
                }
                return;
            }
            if (isChunked) {
                processChunkedData(chunk);
            } else {
                process.stdout.write(chunk);
            }
        });

        conn.on("end", () => {
            process.stdin.pause();
            process.stdin.removeAllListeners("data");
            resolve();
        });

        conn.on("error", (err) => {
            reject(err);
        });
    });
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
