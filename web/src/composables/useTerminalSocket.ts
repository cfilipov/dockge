import { ref, type Ref } from "vue";

export interface TerminalSocketOptions {
    type: string;
    params?: Record<string, string>;
}

export interface TerminalSocket {
    connected: Ref<boolean>;
    onData: (handler: (data: Uint8Array) => void) => void;
    onDisconnect: (handler: () => void) => void;
    sendInput: (data: string) => void;
    sendResize: (rows: number, cols: number) => void;
    close: () => void;
}

/**
 * Opens a dedicated binary WebSocket to /ws/terminal/{type}?token=...&params...
 * Returns handlers for receiving data and sending input/resize.
 * Auto-reconnects on close/error with exponential backoff (1s → 30s).
 * Call close() to stop reconnecting and tear down.
 */
export function useTerminalSocket(options: TerminalSocketOptions): TerminalSocket {
    const connected = ref(false);
    let ws: WebSocket | null = null;
    let dataHandler: ((data: Uint8Array) => void) | null = null;
    let disconnectHandler: (() => void) | null = null;
    const encoder = new TextEncoder();
    let shouldReconnect = true;
    let reconnectDelay = 1000;
    const maxReconnectDelay = 30000;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;

    // Build WebSocket URL
    const wsProtocol = location.protocol === "https:" ? "wss:" : "ws:";
    const token = localStorage.token || sessionStorage.token || "";
    const params = new URLSearchParams(options.params || {});
    if (token && token !== "autoLogin") {
        params.set("token", token);
    }
    const paramStr = params.toString();
    const url = `${wsProtocol}//${location.host}/ws/terminal/${options.type}${paramStr ? "?" + paramStr : ""}`;

    function doConnect() {
        ws = new WebSocket(url);
        ws.binaryType = "arraybuffer";

        ws.onopen = () => {
            connected.value = true;
            reconnectDelay = 1000;
        };

        ws.onmessage = (e: MessageEvent) => {
            if (dataHandler && e.data instanceof ArrayBuffer) {
                dataHandler(new Uint8Array(e.data));
            }
        };

        ws.onclose = () => {
            connected.value = false;
            disconnectHandler?.();
            scheduleReconnect();
        };

        ws.onerror = () => {
            connected.value = false;
        };
    }

    function scheduleReconnect() {
        if (!shouldReconnect) return;
        reconnectTimer = setTimeout(() => {
            reconnectTimer = null;
            reconnectDelay = Math.min(reconnectDelay * 2, maxReconnectDelay);
            doConnect();
        }, reconnectDelay);
    }

    doConnect();

    function onData(handler: (data: Uint8Array) => void) {
        dataHandler = handler;
    }

    function onDisconnect(handler: () => void) {
        disconnectHandler = handler;
    }

    function sendInput(data: string) {
        if (!ws || ws.readyState !== WebSocket.OPEN) return;
        const encoded = encoder.encode(data);
        const msg = new Uint8Array(1 + encoded.length);
        msg[0] = 0x00; // input prefix
        msg.set(encoded, 1);
        ws.send(msg.buffer);
    }

    function sendResize(rows: number, cols: number) {
        if (!ws || ws.readyState !== WebSocket.OPEN) return;
        const msg = new Uint8Array(5);
        msg[0] = 0x01; // resize prefix
        const view = new DataView(msg.buffer);
        view.setUint16(1, rows, false); // big-endian
        view.setUint16(3, cols, false);
        ws.send(msg.buffer);
    }

    function close() {
        shouldReconnect = false;
        if (reconnectTimer) {
            clearTimeout(reconnectTimer);
            reconnectTimer = null;
        }
        if (ws) {
            ws.onclose = null; // prevent re-triggering
            ws.close();
            ws = null;
        }
        connected.value = false;
    }

    return { connected, onData, onDisconnect, sendInput, sendResize, close };
}
