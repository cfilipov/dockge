import { reactive, ref, computed, watch } from "vue";
import jwtDecode from "jwt-decode";
import { Terminal } from "@xterm/xterm";
import { AgentSocket } from "../common/agent-socket";
import { router } from "../router";
import { i18n } from "../i18n";

// --- Plain WebSocket wrapper (replaces socket.io-client) ---

class DockgeWebSocket {
    private ws: WebSocket | null = null;
    private nextId = 1;
    private callbacks = new Map<number, Function>();
    private listeners = new Map<string, Function[]>();
    private url = "";
    private reconnectDelay = 1000;
    private maxReconnectDelay = 30000;
    private shouldReconnect = true;

    connect(url: string) {
        this.url = url;
        this.shouldReconnect = true;
        this.doConnect();
    }

    private doConnect() {
        try {
            this.ws = new WebSocket(this.url);
        } catch {
            this.scheduleReconnect();
            return;
        }

        this.ws.onopen = () => {
            this.reconnectDelay = 1000;
            this.fire("connect");
        };

        this.ws.onmessage = (e) => {
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
        const callback = hasCallback ? (args.pop() as Function) : null;

        const msg: Record<string, unknown> = {
            event,
            args,
        };

        if (callback) {
            const id = this.nextId++;
            msg.id = id;
            this.callbacks.set(id, callback);
        }

        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(msg));
        }
    }

    on(event: string, handler: Function) {
        const list = this.listeners.get(event) || [];
        list.push(handler);
        this.listeners.set(event, list);
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

        // Server push: has "event" + optional "args"
        if ("event" in msg) {
            const event = msg.event as string;
            const args = (msg.args as unknown[]) || [];
            // Unwrap single-element arrays for consistency with Socket.IO behavior
            if (Array.isArray(args)) {
                this.fire(event, ...args);
            } else {
                this.fire(event, args);
            }
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
let terminalMap: Map<string, Terminal> = new Map();

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

const info = ref<any>({});
const remember = ref(localStorage.remember !== "0");
const loggedIn = ref(false);
const allowLoginDialog = ref(false);
const username = ref<string | null>(null);
const composeTemplate = ref("");
const envTemplate = ref("");

const stackList = ref<Record<string, any>>({});
const containerList = ref<Record<string, any>[]>([]);
const allAgentStackList = ref<Record<string, any>>({});
const agentStatusList = ref<Record<string, any>>({});
const agentList = ref<Record<string, any>>({});

// Computed
const agentCount = computed(() => Object.keys(agentList.value).length);

const completeStackList = computed(() => {
    let list: Record<string, any> = {};

    for (let stackName in stackList.value) {
        list[stackName + "_"] = stackList.value[stackName];
    }

    for (let endpoint in allAgentStackList.value) {
        let instance = allAgentStackList.value[endpoint];
        for (let stackName in instance.stackList) {
            list[stackName + "_" + endpoint] = instance.stackList[stackName];
        }
    }
    return list;
});

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
watch(() => socketIO.connected, () => {
    if (socketIO.connected) {
        agentStatusList.value[""] = "online";
    } else {
        agentStatusList.value[""] = "offline";
    }
});

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

function emitAgent(endpoint: string, eventName: string, ...args: unknown[]) {
    getSocket().emit("agent", endpoint, eventName, ...args);
}

function endpointDisplayFunction(endpoint: string) {
    for (const [k, v] of Object.entries(agentList.value)) {
        if (endpoint) {
            if (endpoint === (v as any)["endpoint"] && (v as any)["name"] !== "") {
                return (v as any)["name"];
            }
            if (endpoint === (v as any)["endpoint"] && (v as any)["name"] === "") {
                return endpoint;
            }
        }
    }
}

function getJWTPayload() {
    const jwtToken = storage().token;

    if (jwtToken && jwtToken !== "autoLogin") {
        return jwtDecode(jwtToken);
    }
    return undefined;
}

function getTurnstileSiteKey(callback: Function) {
    getSocket().emit("getTurnstileSiteKey", callback);
}

function login(
    usernameVal: string,
    password: string,
    token: string,
    captchaToken: string,
    callback: Function
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
    socket.emit("requestContainerList", () => {});
}

function bindTerminal(endpoint: string, terminalName: string, terminal: Terminal) {
    // Load terminal, get terminal screen
    emitAgent(endpoint, "terminalJoin", terminalName, (res: any) => {
        if (res.ok) {
            terminal.write(res.buffer);
            terminalMap.set(terminalName, terminal);
        } else {
            // Import toast lazily to avoid circular dependency issues at module init
            import("./useAppToast").then(({ useAppToast }) => {
                useAppToast().toastRes(res);
            });
        }
    });
}

function unbindTerminal(terminalName: string) {
    terminalMap.delete(terminalName);
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

    // Handling events from agents
    let agentSocket = new AgentSocket();
    socket.on("agent", (eventName: unknown, ...args: unknown[]) => {
        agentSocket.call(eventName as string, ...args);
    });

    socket.on("connect", () => {
        console.log("Connected to the socket server");

        clearTimeout(connectingMsgTimeout);
        socketIO.connecting = false;

        socketIO.connectCount++;
        socketIO.connected = true;
        socketIO.showReverseProxyGuide = false;
        const token = storage().token;

        if (token) {
            if (token !== "autoLogin") {
                console.log("Logging in by token");
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
        console.log("disconnect");
        socketIO.connectionErrorMsg = `${t("Lost connection to the socket server. Reconnecting...")}`;
        socketIO.connected = false;
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

    socket.on("autoLogin", () => {
        loggedIn.value = true;
        storage().token = "autoLogin";
        socketIO.token = "autoLogin";
        allowLoginDialog.value = false;
        afterLogin();
    });

    socket.on("setup", () => {
        console.log("setup");
        router.push("/setup");
    });

    agentSocket.on("terminalWrite", (terminalName: unknown, data: unknown) => {
        const terminal = terminalMap.get(terminalName as string);
        if (!terminal) {
            return;
        }
        terminal.write(data as string);
    });

    agentSocket.on("stackList", (res: any) => {
        if (res.ok) {
            if (!res.endpoint) {
                stackList.value = res.stackList;
            } else {
                if (!allAgentStackList.value[res.endpoint]) {
                    allAgentStackList.value[res.endpoint] = {
                        stackList: {},
                    };
                }
                allAgentStackList.value[res.endpoint].stackList = res.stackList;
            }
        }
    });

    agentSocket.on("containerList", (res: any) => {
        if (res.ok) {
            containerList.value = res.containerList;
        }
    });

    socket.on("stackStatusList", (res: any) => {
        if (res.ok) {
            for (let stackName in res.stackStatusList) {
                const stackObj = stackList.value[stackName];
                if (stackObj) {
                    stackObj.status = res.stackStatusList[stackName];
                }
            }
        }
    });

    socket.on("agentStatus", (res: any) => {
        agentStatusList.value[res.endpoint] = res.status;

        if (res.msg) {
            import("./useAppToast").then(({ useAppToast }) => {
                useAppToast().toastError(res.msg);
            });
        }
    });

    socket.on("agentList", (res: any) => {
        if (res.ok) {
            agentList.value = res.agentList;
        } else if (res) {
            // Go backend sends agentList directly (not wrapped in {ok, agentList})
            agentList.value = res;
        }
    });

    socket.on("refresh", () => {
        location.reload();
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
        stackList,
        containerList,
        allAgentStackList,
        agentStatusList,
        agentList,

        // Computed
        agentCount,
        completeStackList,
        usernameFirstChar,
        frontendVersion,
        isFrontendBackendVersionMatched,

        // Methods
        endpointDisplayFunction,
        storage,
        getSocket,
        emitAgent,
        getJWTPayload,
        getTurnstileSiteKey,
        login,
        loginByToken,
        logout,
        bindTerminal,
        unbindTerminal,
    };
}
