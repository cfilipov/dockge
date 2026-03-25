import { reactive, ref, computed, watch, nextTick } from "vue";
import jwtDecode from "jwt-decode";
import { router } from "../router";
import { i18n } from "../i18n";
import type { InfoData } from "../common/types";
import { useContainerStore } from "../stores/containerStore";
import { useStackStore } from "../stores/stackStore";
import { useNetworkStore } from "../stores/networkStore";
import { useImageStore } from "../stores/imageStore";
import { useVolumeStore } from "../stores/volumeStore";
import { useUpdateStore } from "../stores/updateStore";

// --- Plain WebSocket wrapper (replaces socket.io-client) ---

class DockgeWebSocket {
    private ws: WebSocket | null = null;
    private nextId = 1;
    private callbacks = new Map<number, (...args: unknown[]) => void>();
    private listeners = new Map<string, ((...args: unknown[]) => void)[]>();
    private url = "";
    private reconnectDelay = 1000;
    private maxReconnectDelay = 30000;
    private shouldReconnect = true;

    private binaryHandlers: ((sessionId: number, data: Uint8Array) => void)[] = [];

    connect(url: string) {
        this.url = url;
        this.shouldReconnect = true;
        this.doConnect();
    }

    private doConnect() {
        try {
            this.ws = new WebSocket(this.url);
            this.ws.binaryType = "arraybuffer";
        } catch {
            this.scheduleReconnect();
            return;
        }

        this.ws.onopen = () => {
            this.reconnectDelay = 1000;
            this.fire("connect");
        };

        this.ws.onmessage = (e) => {
            // Binary frame: terminal data
            if (e.data instanceof ArrayBuffer) {
                const arr = new Uint8Array(e.data);
                if (arr.length < 2) return;
                const sessionId = (arr[0] << 8) | arr[1];
                const payload = arr.subarray(2);
                for (const handler of this.binaryHandlers) {
                    handler(sessionId, payload);
                }
                return;
            }

            // Text frame: JSON message
            try {
                const msg = JSON.parse(e.data);
                this.handleMessage(msg);
            } catch (err) {
                console.error("WS message parse error:", err);
            }
        };

        this.ws.onclose = () => {
            this.fire("disconnect");
            this.scheduleReconnect();
        };

        this.ws.onerror = () => {
            this.fire("connect_error", new Error("WebSocket error"));
        };
    }

    emit(event: string, ...args: unknown[]) {
        // Last arg may be a callback (ack pattern)
        const last = args[args.length - 1];
        const hasCallback = typeof last === "function";
        const callback = hasCallback ? (args.pop() as (...a: unknown[]) => void) : null;

        const msg: Record<string, unknown> = {
            event,
            args,
        };

        if (callback) {
            const id = this.nextId++;
            msg.id = id;
            this.callbacks.set(id, callback);
            // Auto-clean orphaned callbacks after 30s
            setTimeout(() => {
                this.callbacks.delete(id);
            }, 30000);
        }

        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(msg));
        }
    }

    on(event: string, handler: (...args: unknown[]) => void) {
        const list = this.listeners.get(event) || [];
        list.push(handler);
        this.listeners.set(event, list);
    }

    off(event: string, handler: (...args: unknown[]) => void) {
        const list = this.listeners.get(event);
        if (list) {
            const idx = list.indexOf(handler);
            if (idx !== -1) list.splice(idx, 1);
        }
    }

    sendBinary(data: ArrayBuffer) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(data);
        }
    }

    onBinary(handler: (sessionId: number, data: Uint8Array) => void) {
        this.binaryHandlers.push(handler);
    }

    offBinary(handler: (sessionId: number, data: Uint8Array) => void) {
        const idx = this.binaryHandlers.indexOf(handler);
        if (idx !== -1) this.binaryHandlers.splice(idx, 1);
    }

    get connected(): boolean {
        return this.ws?.readyState === WebSocket.OPEN;
    }

    disconnect() {
        this.shouldReconnect = false;
        this.ws?.close();
    }

    private handleMessage(msg: Record<string, unknown>) {
        // Ack response: has "id" + "data"
        if ("id" in msg && this.callbacks.has(msg.id as number)) {
            const cb = this.callbacks.get(msg.id as number)!;
            this.callbacks.delete(msg.id as number);
            try {
                cb(msg.data);
            } catch (err) {
                console.error("WS ack callback error:", err);
            }
            return;
        }

        // Server push: has "event" + "data" (single payload value)
        if ("event" in msg) {
            const event = msg.event as string;
            this.fire(event, msg.data);
        }
    }

    private fire(event: string, ...args: unknown[]) {
        const handlers = this.listeners.get(event);
        if (handlers) {
            for (const handler of handlers) {
                try {
                    handler(...args);
                } catch (err) {
                    console.error(`WS event handler error (${event}):`, err);
                }
            }
        }
    }

    private scheduleReconnect() {
        if (!this.shouldReconnect) {
            return;
        }
        setTimeout(() => {
            this.reconnectDelay = Math.min(this.reconnectDelay * 2, this.maxReconnectDelay);
            this.doConnect();
        }, this.reconnectDelay);
    }
}

// --- Module-level state ---

let socket: DockgeWebSocket;

function t(key: string): string {
    return (i18n.global as any).t(key);
}

// Reactive state
const socketIO = reactive({
    token: null as string | null,
    firstConnect: true,
    connected: false,
    connectCount: 0,
    initedSocketIO: false,
    connectionErrorMsg: "",
    showReverseProxyGuide: true,
    connecting: false,
});

const info = ref<InfoData>({} as InfoData);
const remember = ref(localStorage.remember !== "0");
const loggedIn = ref(false);
const allowLoginDialog = ref(false);
const username = ref<string | null>(null);
const composeTemplate = ref("");
const envTemplate = ref("");

// Track initial data load — all 6 data channels + a complete updates payload.
// The "updates" event includes a "complete" flag: false while the background
// checker is still running, true once it has finished. If the checker already
// ran before this connection was established, the afterLogin "updates" message
// arrives with complete=true immediately.
const INIT_CHANNELS = ["stacks", "containers", "networks", "images", "volumes", "updates", "updatesComplete"] as const;
const receivedChannels = new Set<string>();
const dataReady = ref(false);

function markChannel(ch: string) {
    if (dataReady.value) return;
    receivedChannels.add(ch);
    if (receivedChannels.size >= INIT_CHANNELS.length) {
        dataReady.value = true;
        nextTick(() => {
            document.body.setAttribute("data-ready", "true");
        });
    }
}

// Reset on disconnect so reconnects re-track.
function resetDataReady() {
    receivedChannels.clear();
    dataReady.value = false;
    document.body.removeAttribute("data-ready");
}

const usernameFirstChar = computed(() => {
    if (typeof username.value === "string" && username.value.length >= 1) {
        return username.value.charAt(0).toUpperCase();
    } else {
        return "🐬";
    }
});

const frontendVersion = computed(() => {
    // eslint-disable-next-line no-undef
    return FRONTEND_VERSION;
});

const isFrontendBackendVersionMatched = computed(() => {
    if (!info.value.version) {
        return true;
    }
    return info.value.version === frontendVersion.value;
});

// Watchers
watch(remember, () => {
    localStorage.remember = remember.value ? "1" : "0";
});

// Reload the SPA if the server version is changed.
watch(() => info.value.version, (to, from) => {
    if (from && from !== to) {
        window.location.reload();
    }
});

// Methods
function storage(): Storage {
    return remember.value ? localStorage : sessionStorage;
}

function getSocket(): DockgeWebSocket {
    return socket;
}

function emit(eventName: string, ...args: unknown[]) {
    getSocket().emit(eventName, ...args);
}

function getJWTPayload() {
    const jwtToken = storage().token;

    if (jwtToken && jwtToken !== "autoLogin") {
        return jwtDecode(jwtToken);
    }
    return undefined;
}

function getTurnstileSiteKey(callback: (...args: unknown[]) => void) {
    getSocket().emit("getTurnstileSiteKey", callback);
}

function login(
    usernameVal: string,
    password: string,
    token: string,
    captchaToken: string,
    callback: (...args: unknown[]) => void,
) {
    getSocket().emit("login", {
        username: usernameVal,
        password,
        token,
        captchaToken,
    }, (res: any) => {
        if (res.tokenRequired) {
            callback(res);
        }

        if (res.ok) {
            storage().token = res.token;
            socketIO.token = res.token;
            loggedIn.value = true;
            username.value = (getJWTPayload() as any)?.username;

            afterLogin();

            // Trigger Chrome Save Password
            history.pushState({}, "");
        }

        callback(res);
    });
}

function loginByToken(token: string) {
    socket.emit("loginByToken", token, (res: any) => {
        allowLoginDialog.value = true;

        if (!res.ok) {
            logout();
        } else {
            loggedIn.value = true;
            username.value = (getJWTPayload() as any)?.username;
            afterLogin();
        }
    });
}

function logout() {
    socket.emit("logout", () => { });
    storage().removeItem("token");
    socketIO.token = null;
    loggedIn.value = false;
    username.value = null;
    clearData();
}

function clearData() {
    // Placeholder for future use
}

function afterLogin() {
    // Broadcasts (stacks, containers, networks, images, volumes, updates)
    // are sent automatically by the backend on authenticated connect.
}

// --- Initialization ---

export function initWebSocket() {
    // No need to re-init
    if (socketIO.initedSocketIO) {
        return;
    }

    socketIO.initedSocketIO = true;

    // Build WebSocket URL
    const wsProtocol = location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = wsProtocol + "//" + location.host + "/ws";

    let connectingMsgTimeout = setTimeout(() => {
        socketIO.connecting = true;
    }, 1500);

    socket = new DockgeWebSocket();

    socket.on("connect", () => {
        console.debug("Connected to the socket server");

        clearTimeout(connectingMsgTimeout);
        socketIO.connecting = false;

        socketIO.connectCount++;
        socketIO.connected = true;
        socketIO.showReverseProxyGuide = false;
        const token = storage().token;

        if (token) {
            if (token !== "autoLogin") {
                console.debug("Logging in by token");
                loginByToken(token);
            } else {
                // Timeout if it is not actually auto login
                setTimeout(() => {
                    if (!loggedIn.value) {
                        allowLoginDialog.value = true;
                        storage().removeItem("token");
                    }
                }, 5000);
            }
        } else {
            allowLoginDialog.value = true;
        }

        socketIO.firstConnect = false;
    });

    socket.on("disconnect", () => {
        console.debug("disconnect");
        socketIO.connectionErrorMsg = `${t("Lost connection to the socket server. Reconnecting...")}`;
        socketIO.connected = false;
        resetDataReady();
    });

    socket.on("connect_error", (err: any) => {
        console.error(`Failed to connect to the backend. WebSocket connect_error: ${err.message}`);
        socketIO.connectionErrorMsg = `${t("Cannot connect to the socket server.")} [${err}] ${t("reconnecting...")}`;
        socketIO.showReverseProxyGuide = true;
        socketIO.connected = false;
        socketIO.firstConnect = false;
        socketIO.connecting = false;
    });

    // Custom Events

    socket.on("info", (infoData: any) => {
        info.value = infoData;
    });

    socket.on("autoLogin", (...args: unknown[]) => {
        const user = typeof args[0] === "string" ? args[0] : undefined;
        loggedIn.value = true;
        storage().token = "autoLogin";
        socketIO.token = "autoLogin";
        if (user) {
            username.value = user;
        }
        allowLoginDialog.value = false;
        afterLogin();
    });

    socket.on("setup", () => {
        console.debug("setup");
        router.push("/setup");
    });

    socket.on("refresh", () => {
        location.reload();
    });

    // --- Broadcast channel listeners (normalized model) ---
    // Each channel pushes its data directly to the corresponding Pinia store.

    socket.on("stacks", (data: any) => {
        const broadcast = data?.items ?? data;
        useStackStore().mergeStacks(broadcast as Record<string, any>);
        markChannel("stacks");
    });

    socket.on("containers", (data: any) => {
        const broadcast = data?.items ?? data;
        useContainerStore().mergeContainers(broadcast as Record<string, any>);
        markChannel("containers");
    });

    socket.on("networks", (data: any) => {
        const broadcast = data?.items ?? data;
        useNetworkStore().mergeNetworks(broadcast as Record<string, any>);
        markChannel("networks");
    });

    socket.on("images", (data: any) => {
        const broadcast = data?.items ?? data;
        useImageStore().mergeImages(broadcast as Record<string, any>);
        markChannel("images");
    });

    socket.on("volumes", (data: any) => {
        const broadcast = data?.items ?? data;
        useVolumeStore().mergeVolumes(broadcast as Record<string, any>);
        markChannel("volumes");
    });

    socket.on("updates", (data: unknown) => {
        const payload = data as { keys?: string[]; complete?: boolean } | string[];
        if (Array.isArray(payload)) {
            // Legacy format: plain string[]
            useUpdateStore().setUpdates(payload);
        } else {
            useUpdateStore().setUpdates(payload.keys ?? []);
            if (payload.complete) {
                markChannel("updatesComplete");
            }
        }
        markChannel("updates");
    });

    socket.on("updateCheckComplete", (...args: unknown[]) => {
        const data = (typeof args[0] === "object" && args[0] !== null) ? args[0] as Record<string, unknown> : {};
        console.debug(`Image update check complete: ${data.servicesWithUpdates ?? 0} services with updates (${data.durationMs ?? 0}ms)`);
        markChannel("updateCheckComplete");
    });

    // --- Dedicated resource event channel ---
    // Docker events are sent individually on this channel, decoupled from list broadcasts.
    socket.on("resourceEvent", (data: any) => {
        const evt = data;
        if (!evt?.type) return;
        switch (evt.type) {
            case "container":
                useContainerStore().setLastEvent(evt);
                break;
            case "network":
                useNetworkStore().setLastEvent(evt);
                break;
            case "image":
                useImageStore().setLastEvent(evt);
                break;
            case "volume":
                useVolumeStore().setLastEvent(evt);
                break;
        }
    });

    socket.connect(wsUrl);
}

// --- Composable ---

export function useSocket() {
    return {
        // Reactive state
        socketIO,
        info,
        remember,
        loggedIn,
        allowLoginDialog,
        username,
        composeTemplate,
        envTemplate,

        // Computed
        usernameFirstChar,
        frontendVersion,
        isFrontendBackendVersionMatched,

        // Methods
        storage,
        getSocket,
        emit,
        getJWTPayload,
        getTurnstileSiteKey,
        login,
        loginByToken,
        logout,
    };
}
