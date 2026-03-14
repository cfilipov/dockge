import { ref, type Ref } from "vue";
import { useSocket } from "./useSocket";

export interface TerminalJoinOptions {
    type: string;
    stack?: string;
    service?: string;
    container?: string;
    shell?: string;
}

export interface TerminalSession {
    sessionId: Ref<number | null>;
    connected: Ref<boolean>;
    onData: (handler: (data: Uint8Array) => void) => void;
    onExited: (handler: () => void) => void;
    sendInput: (data: string) => void;
    sendResize: (rows: number, cols: number) => void;
    leave: () => void;
}

interface PendingSession {
    sessionId: Ref<number | null>;
    connected: Ref<boolean>;
    dataHandler: ((data: Uint8Array) => void) | null;
    exitedHandler: (() => void) | null;
    opts: TerminalJoinOptions;
}

const encoder = new TextEncoder();
let muxInstance: TerminalMux | null = null;

class TerminalMux {
    private sessions = new Map<number, PendingSession>();
    private binaryHandler: ((sessionId: number, data: Uint8Array) => void) | null = null;

    constructor() {
        const { getSocket } = useSocket();
        const socket = getSocket();

        // Register binary frame handler
        this.binaryHandler = (sessionId: number, data: Uint8Array) => {
            const session = this.sessions.get(sessionId);
            if (session?.dataHandler) {
                session.dataHandler(data);
            }
        };
        socket.onBinary(this.binaryHandler);

        // Listen for terminalExited events
        socket.on("terminalExited", (data: any) => {
            const sessionId = data?.sessionId;
            if (sessionId != null) {
                const session = this.sessions.get(sessionId);
                if (session?.exitedHandler) {
                    session.exitedHandler();
                }
            }
        });

        // On reconnect, re-join all active sessions
        socket.on("connect", () => {
            // Small delay to let auth complete before re-joining
            setTimeout(() => {
                for (const [, session] of this.sessions) {
                    session.sessionId.value = null;
                    session.connected.value = false;
                    this.doJoin(session);
                }
            }, 500);
        });
    }

    join(opts: TerminalJoinOptions): TerminalSession {
        const sessionId = ref<number | null>(null);
        const connected = ref(false);

        const pending: PendingSession = {
            sessionId,
            connected,
            dataHandler: null,
            exitedHandler: null,
            opts,
        };

        this.doJoin(pending);

        return {
            sessionId,
            connected,
            onData: (handler) => { pending.dataHandler = handler; },
            onExited: (handler) => { pending.exitedHandler = handler; },
            sendInput: (data: string) => {
                if (sessionId.value == null) return;
                const encoded = encoder.encode(data);
                const msg = new Uint8Array(3 + encoded.length);
                msg[0] = (sessionId.value >> 8) & 0xff;
                msg[1] = sessionId.value & 0xff;
                msg[2] = 0x00; // input opcode
                msg.set(encoded, 3);
                const { getSocket } = useSocket();
                getSocket().sendBinary(msg.buffer);
            },
            sendResize: (rows: number, cols: number) => {
                if (sessionId.value == null) return;
                const msg = new Uint8Array(7);
                msg[0] = (sessionId.value >> 8) & 0xff;
                msg[1] = sessionId.value & 0xff;
                msg[2] = 0x01; // resize opcode
                const view = new DataView(msg.buffer);
                view.setUint16(3, rows, false);
                view.setUint16(5, cols, false);
                const { getSocket } = useSocket();
                getSocket().sendBinary(msg.buffer);
            },
            leave: () => {
                if (sessionId.value != null) {
                    const sid = sessionId.value;
                    this.sessions.delete(sid);
                    const { emit } = useSocket();
                    emit("terminalLeave", { sessionId: sid });
                    sessionId.value = null;
                    connected.value = false;
                }
            },
        };
    }

    private doJoin(session: PendingSession) {
        const { emit } = useSocket();
        emit("terminalJoin", session.opts, (res: any) => {
            if (res?.ok && res.sessionId != null) {
                session.sessionId.value = res.sessionId;
                session.connected.value = true;
                this.sessions.set(res.sessionId, session);
            }
        });
    }
}

export function useTerminalMux(): TerminalMux {
    if (!muxInstance) {
        muxInstance = new TerminalMux();
    }
    return muxInstance;
}
