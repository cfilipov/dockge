<template>
    <div class="shadow-box" role="region" :aria-label="ariaLabel || 'Terminal'" :style="{ backgroundColor: isDark ? darkTheme.background : lightTheme.background }">
        <div v-pre ref="terminalEl" class="main-terminal"></div>
    </div>
</template>

<script setup lang="ts">
import { ref, shallowRef, watch, onMounted, onUnmounted } from "vue";
import { Terminal } from "@xterm/xterm";
import type { ITheme } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { TERMINAL_COLS, TERMINAL_ROWS } from "../common/util-common";
import { useSocket } from "../composables/useSocket";
import { useTheme } from "../composables/useTheme";
import { useTerminalSocket, type TerminalSocket } from "../composables/useTerminalSocket";

const { bindTerminal, unbindTerminal } = useSocket();
const { isDark } = useTheme();

// VS Code's default dark terminal palette
const darkTheme: ITheme = {
    background: "#000000",
    foreground: "#cccccc",
    cursor: "#aeafad",
    selectionBackground: "#264f78",
    black: "#000000",
    red: "#cd3131",
    green: "#0dbc79",
    yellow: "#e5e510",
    blue: "#2472c8",
    magenta: "#bc3fbc",
    cyan: "#11a8cd",
    white: "#e5e5e5",
    brightBlack: "#666666",
    brightRed: "#f14c4c",
    brightGreen: "#23d18b",
    brightYellow: "#f5f543",
    brightBlue: "#3b8eea",
    brightMagenta: "#d670d6",
    brightCyan: "#29b8db",
    brightWhite: "#e5e5e5",
};

// VS Code's default light terminal palette
const lightTheme: ITheme = {
    background: "#ffffff",
    foreground: "#333333",
    cursor: "#000000",
    selectionBackground: "#add6ff",
    black: "#000000",
    red: "#cd3131",
    green: "#00bc00",
    yellow: "#949800",
    blue: "#0451a5",
    magenta: "#bc05bc",
    cyan: "#0598bc",
    white: "#555555",
    brightBlack: "#666666",
    brightRed: "#cd3131",
    brightGreen: "#14ce14",
    brightYellow: "#b5ba00",
    brightBlue: "#0451a5",
    brightMagenta: "#bc05bc",
    brightCyan: "#0598bc",
    brightWhite: "#a5a5a5",
};

const props = withDefaults(defineProps<{
    name: string;
    rows?: number;
    cols?: number;
    mode?: string;
    ariaLabel?: string;
    channel?: "control" | "terminal";
    terminalType?: string;
    terminalParams?: Record<string, string>;
}>(), {
    rows: TERMINAL_ROWS,
    cols: TERMINAL_COLS,
    mode: "displayOnly",
    ariaLabel: undefined,
    channel: "control",
    terminalType: undefined,
    terminalParams: undefined,
});

const emit = defineEmits<{
    (e: "has-data"): void;
    (e: "has-buffer"): void;
}>();

const terminalEl = ref<HTMLElement>();
const terminal = shallowRef<Terminal | null>(null);
let terminalFitAddOn: FitAddon | null = null;
let first = true;
let stopDarkWatcher: (() => void) | null = null;
let termSocket: TerminalSocket | null = null;
let spinnerTimer: ReturnType<typeof setTimeout> | null = null;
let spinnerInterval: ReturnType<typeof setInterval> | null = null;

function bind(name?: string) {
    const termName = name || props.name;
    if (!termName) {
        console.debug("Terminal name not set");
        return;
    }
    unbindTerminal(termName);
    bindTerminal(termName, terminal.value!, (hasBuffer) => {
        if (hasBuffer) emit("has-buffer");
    });
    console.debug("Terminal bound: " + termName);
}

function interactiveTerminalConfig() {
    terminal.value!.onKey(e => {
        if (e.key === "\u0016" || (e.ctrlKey && e.key === "v")) {
            handlePaste();
            return;
        }
        if (termSocket) {
            termSocket.sendInput(e.key);
        }
    });
}

// Spinner for dedicated terminal WS connections (debounced 150ms)
const SPINNER_FRAMES = ["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"];

function startSpinnerDebounce() {
    let frameIdx = 0;
    spinnerTimer = setTimeout(() => {
        if (!terminal.value) return;
        terminal.value.write("\x1b[?25l"); // hide cursor
        terminal.value.write(`\r  ${SPINNER_FRAMES[0]}  Connecting...`);
        spinnerInterval = setInterval(() => {
            frameIdx = (frameIdx + 1) % SPINNER_FRAMES.length;
            terminal.value?.write(`\r  ${SPINNER_FRAMES[frameIdx]}  Connecting...`);
        }, 80);
    }, 150);
}

function stopSpinner() {
    if (spinnerTimer) {
        clearTimeout(spinnerTimer);
        spinnerTimer = null;
    }
    if (spinnerInterval) {
        clearInterval(spinnerInterval);
        spinnerInterval = null;
    }
}

function connectDedicatedTerminal() {
    if (!props.terminalType) return;

    startSpinnerDebounce();

    termSocket = useTerminalSocket({
        type: props.terminalType,
        params: props.terminalParams,
    });

    let firstMessage = true;
    termSocket.onData((data: Uint8Array) => {
        if (!terminal.value) return;
        if (firstMessage) {
            stopSpinner();
            // reset() fully clears the terminal including current line and scrollback,
            // unlike clear() which preserves the cursor line (leaving spinner residue).
            terminal.value.reset();
            // Restore cursor visibility — the spinner hides it with \x1b[?25l,
            // and reset() doesn't always re-enable it (xterm.js quirk).
            terminal.value.write("\x1b[?25h");
            firstMessage = false;
        }
        terminal.value.write(data);

        if (first) {
            emit("has-data");
            first = false;
        }
    });
}

function updateTerminalSize() {
    if (!terminalFitAddOn) {
        terminalFitAddOn = new FitAddon();
        terminal.value!.loadAddon(terminalFitAddOn);
        window.addEventListener("resize", onResizeEvent);
    }
    terminalFitAddOn.fit();
}

let lastSentRows = 0;
let lastSentCols = 0;

function onResizeEvent() {
    terminalFitAddOn?.fit();
    const rows = terminal.value!.rows;
    const cols = terminal.value!.cols;
    if (rows === lastSentRows && cols === lastSentCols) return;
    lastSentRows = rows;
    lastSentCols = cols;
    if (termSocket) {
        termSocket.sendResize(rows, cols);
    }
}

async function handlePaste() {
    try {
        const text = await navigator.clipboard.readText();
        if (text) {
            pasteText(text);
        }
    } catch (error) {
        console.error("Failed to read from clipboard:", error);
    }
}

function pasteText(text: string) {
    if (props.mode === "interactive" && termSocket) {
        termSocket.sendInput(text);
    }
}

function handleContextMenu(event: Event) {
    event.preventDefault();
    if (props.mode === "interactive") {
        handlePaste();
    }
}

function handleSelection() {
    const selectedText = terminal.value!.getSelection();
    if (selectedText && selectedText.length > 0) {
        copyToClipboard(selectedText);
    }
}

async function copyToClipboard(text: string) {
    try {
        await navigator.clipboard.writeText(text);
        console.debug("Text copied to clipboard:", text);
    } catch (error) {
        console.error("Failed to copy to clipboard:", error);
    }
}

onMounted(() => {
    let cursorBlink = true;
    if (props.mode === "displayOnly") {
        cursorBlink = false;
    }

    terminal.value = new Terminal({
        fontSize: 14,
        fontFamily: "'JetBrains Mono', monospace",
        cursorBlink,
        cols: props.cols,
        rows: props.rows,
        theme: isDark.value ? darkTheme : lightTheme,
    });

    if (props.mode === "interactive") {
        interactiveTerminalConfig();
    }

    terminal.value.open(terminalEl.value!);
    terminal.value.focus();

    terminalEl.value!.addEventListener("contextmenu", handleContextMenu);

    terminal.value.onSelectionChange(() => {
        handleSelection();
    });

    terminal.value.onCursorMove(() => {
        console.debug("onData triggered");
        if (first) {
            emit("has-data");
            first = false;
        }
    });

    stopDarkWatcher = watch(isDark, (dark) => {
        if (terminal.value) {
            terminal.value.options.theme = dark ? darkTheme : lightTheme;
        }
    });

    // Re-bind when the terminal name becomes available after mount.
    // Compose.vue sets stack.name in its own onMounted (after child mounts),
    // so the initial bind() below may get an empty name. This watcher
    // catches the first non-empty name and binds once.
    watch(() => props.name, (newName, oldName) => {
        if (!terminal.value || termSocket) return; // only for control WS path
        if (!oldName && newName) {
            bind(newName);
        }
    });

    if (props.channel === "terminal" && props.terminalType) {
        // Dedicated terminal WebSocket — no control WS lifecycle needed
        connectDedicatedTerminal();
    } else {
        // Control WS path (compose action ProgressTerminal)
        bind();
    }

    updateTerminalSize();

    // Re-measure font metrics after fonts finish loading.
    // Without this, xterm.js may cache fallback monospace metrics
    // before JetBrains Mono loads, causing misaligned rendering.
    document.fonts.ready.then(() => {
        if (!terminal.value) return;
        terminal.value.options.fontFamily = terminal.value.options.fontFamily;
        terminalFitAddOn?.fit();
    });
});

onUnmounted(() => {
    stopSpinner();
    stopDarkWatcher?.();
    window.removeEventListener("resize", onResizeEvent);
    if (terminalEl.value) {
        terminalEl.value.removeEventListener("contextmenu", handleContextMenu);
    }
    if (termSocket) {
        console.debug("Terminal: closing dedicated WS", props.name);
        termSocket.close();
        termSocket = null;
    } else {
        unbindTerminal(props.name);
    }
    terminal.value?.dispose();
});

defineExpose({ terminal, bind });
</script>

<style scoped lang="scss">
.main-terminal {
    height: 100%;
}
</style>

<style lang="scss">
.terminal {
    height: 100%;
}

// xterm.js hardcodes a black background on the viewport via
// .xterm:not(.allow-transparency) .xterm-viewport { background-color: #000 }
// Override it so the viewport inherits the theme background from .xterm.
.xterm:not(.allow-transparency) .xterm-viewport {
    background-color: inherit !important;
}
</style>
