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
import { useTheme } from "../composables/useTheme";
import { useTerminalMux, type TerminalSession } from "../composables/useTerminalMux";
import { createLogBuffer, type LogBuffer } from "../common/log-banners";

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
    terminalType: string;
    terminalParams?: Record<string, string>;
}>(), {
    rows: TERMINAL_ROWS,
    cols: TERMINAL_COLS,
    mode: "displayOnly",
    ariaLabel: undefined,
    terminalParams: undefined,
});

const emit = defineEmits<{
    (e: "has-data"): void;
}>();

const terminalEl = ref<HTMLElement>();
const terminal = shallowRef<Terminal | null>(null);
let terminalFitAddOn: FitAddon | null = null;
let first = true;
let stopDarkWatcher: (() => void) | null = null;
let termSession: TerminalSession | null = null;
let spinnerTimer: ReturnType<typeof setTimeout> | null = null;
let spinnerInterval: ReturnType<typeof setInterval> | null = null;
let logBuffer: LogBuffer | null = null;

function interactiveTerminalConfig() {
    terminal.value!.onKey(e => {
        if (e.key === "\u0016" || (e.ctrlKey && e.key === "v")) {
            handlePaste();
            return;
        }
        if (termSession) {
            termSession.sendInput(e.key);
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

/** Check if this terminal type should use banner interleaving. */
function isLogTerminal(): boolean {
    return props.terminalType === "container-log"
        || props.terminalType === "container-log-by-name"
        || props.terminalType === "combined";
}

function connectTerminal() {
    startSpinnerDebounce();

    const mux = useTerminalMux();
    termSession = mux.join({
        type: props.terminalType,
        stack: props.terminalParams?.stack,
        service: props.terminalParams?.service,
        container: props.terminalParams?.container,
        shell: props.terminalParams?.shell,
    });

    // Create log buffer for log terminal types
    if (isLogTerminal() && terminal.value) {
        logBuffer = createLogBuffer({
            terminal: terminal.value,
            terminalType: props.terminalType as "container-log" | "container-log-by-name" | "combined",
            containerName: props.terminalParams?.container || props.terminalParams?.service,
            stackName: props.terminalParams?.stack,
        });
    }

    let firstMessage = true;
    termSession.onData((data: Uint8Array) => {
        if (!terminal.value) return;
        if (firstMessage) {
            stopSpinner();
            terminal.value.reset();
            terminal.value.write("\x1b[?25h");
            firstMessage = false;
        }

        // Route log data through the log buffer, other terminals write directly
        if (logBuffer) {
            logBuffer.feed(data);
        } else {
            terminal.value.write(data);
        }

        if (first) {
            emit("has-data");
            first = false;
        }
    });

    termSession.onExited(() => {
        // Terminal process exited — could show a message or auto-reconnect
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
    if (termSession) {
        termSession.sendResize(rows, cols);
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
    if (props.mode === "interactive" && termSession) {
        termSession.sendInput(text);
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

    connectTerminal();

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
    if (logBuffer) {
        logBuffer.destroy();
        logBuffer = null;
    }
    window.removeEventListener("resize", onResizeEvent);
    if (terminalEl.value) {
        terminalEl.value.removeEventListener("contextmenu", handleContextMenu);
    }
    if (termSession) {
        console.debug("Terminal: leaving session", props.name);
        termSession.leave();
        termSession = null;
    }
    terminal.value?.dispose();
});

defineExpose({ terminal });
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
