<template>
    <div class="shadow-box">
        <div v-pre ref="terminalEl" class="main-terminal"></div>
    </div>
</template>

<script setup lang="ts">
import { ref, shallowRef, onMounted, onUnmounted } from "vue";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { TERMINAL_COLS, TERMINAL_ROWS } from "../../../common/util-common";
import { useSocket } from "../composables/useSocket";
import { useAppToast } from "../composables/useAppToast";

const { emitAgent, bindTerminal, unbindTerminal } = useSocket();
const { toastRes, toastError } = useAppToast();

const props = withDefaults(defineProps<{
    name: string;
    endpoint: string;
    stackName?: string;
    serviceName?: string;
    containerName?: string;
    shell?: string;
    rows?: number;
    cols?: number;
    mode?: string;
}>(), {
    stackName: undefined,
    serviceName: undefined,
    containerName: undefined,
    shell: "bash",
    rows: TERMINAL_ROWS,
    cols: TERMINAL_COLS,
    mode: "displayOnly",
});

const emit = defineEmits<{
    (e: "has-data"): void;
}>();

const terminalEl = ref<HTMLElement>();
const terminal = shallowRef<Terminal | null>(null);
let terminalFitAddOn: FitAddon | null = null;
let first = true;
let terminalInputBuffer = "";
let cursorPosition = 0;

function bind(endpoint?: string, name?: string) {
    if (name) {
        unbindTerminal(name);
        bindTerminal(endpoint!, name, terminal.value!);
        console.debug("Terminal bound via parameter: " + name);
    } else if (props.name) {
        unbindTerminal(props.name);
        bindTerminal(props.endpoint, props.name, terminal.value!);
        console.debug("Terminal bound: " + props.name);
    } else {
        console.debug("Terminal name not set");
    }
}

function removeInput() {
    const textAfterCursorLength = terminalInputBuffer.length - cursorPosition;
    const spaces = " ".repeat(textAfterCursorLength);
    const backspaceCount = terminalInputBuffer.length;
    const backspaces = "\b \b".repeat(backspaceCount);
    cursorPosition = 0;
    terminal.value!.write(spaces + backspaces);
    terminalInputBuffer = "";
}

function clearCurrentLine() {
    const backspaces = "\b".repeat(cursorPosition);
    const spaces = " ".repeat(terminalInputBuffer.length);
    const moreBackspaces = "\b".repeat(terminalInputBuffer.length);
    terminal.value!.write(backspaces + spaces + moreBackspaces);
}

function mainTerminalConfig() {
    terminal.value!.onKey(e => {
        console.debug("Encode: " + JSON.stringify(e.key));

        if (e.key === "\r") {
            if (terminalInputBuffer.length === 0) {
                return;
            }
            const buffer = terminalInputBuffer;
            removeInput();
            emitAgent(props.endpoint, "terminalInput", props.name, buffer + e.key, (err: any) => {
                toastError(err.msg);
            });
        } else if (e.key === "\u007F") {
            if (cursorPosition > 0) {
                const beforeCursor = terminalInputBuffer.slice(0, cursorPosition - 1);
                const afterCursor = terminalInputBuffer.slice(cursorPosition);
                terminalInputBuffer = beforeCursor + afterCursor;
                cursorPosition--;
                terminal.value!.write("\b" + afterCursor + " \b".repeat(afterCursor.length + 1));
            }
        } else if (e.key === "\u001B\u005B\u0033\u007E") {
            if (cursorPosition < terminalInputBuffer.length) {
                const beforeCursor = terminalInputBuffer.slice(0, cursorPosition);
                const afterCursor = terminalInputBuffer.slice(cursorPosition + 1);
                terminalInputBuffer = beforeCursor + afterCursor;
                terminal.value!.write(afterCursor + " \b".repeat(afterCursor.length + 1));
            }
        } else if (e.key === "\u001B\u005B\u0041" || e.key === "\u001B\u005B\u0042") {
            // UP OR DOWN - do nothing
        } else if (e.key === "\u001B\u005B\u0043") {
            if (cursorPosition < terminalInputBuffer.length) {
                terminal.value!.write(terminalInputBuffer[cursorPosition]);
                cursorPosition++;
            }
        } else if (e.key === "\u001B\u005B\u0044") {
            if (cursorPosition > 0) {
                terminal.value!.write("\b");
                cursorPosition--;
            }
        } else if (e.key === "\u0003") {
            console.debug("Ctrl + C");
            emitAgent(props.endpoint, "terminalInput", props.name, e.key);
            removeInput();
        } else if (e.key === "\u0016" || (e.ctrlKey && e.key === "v")) {
            handlePaste();
        } else if (e.key === "\u0009" || e.key.startsWith("\u001B")) {
            // TAB or other special keys - do nothing
        } else {
            const textBeforeCursor = terminalInputBuffer.slice(0, cursorPosition);
            const textAfterCursor = terminalInputBuffer.slice(cursorPosition);
            terminalInputBuffer = textBeforeCursor + e.key + textAfterCursor;
            terminal.value!.write(e.key + textAfterCursor + "\b".repeat(textAfterCursor.length));
            cursorPosition++;
        }
    });
}

function interactiveTerminalConfig() {
    terminal.value!.onKey(e => {
        if (e.key === "\u0016" || (e.ctrlKey && e.key === "v")) {
            handlePaste();
            return;
        }
        emitAgent(props.endpoint, "terminalInput", props.name, e.key, (res: any) => {
            if (!res.ok) {
                toastRes(res);
            }
        });
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

function onResizeEvent() {
    terminalFitAddOn?.fit();
    const rows = terminal.value!.rows;
    const cols = terminal.value!.cols;
    emitAgent(props.endpoint, "terminalResize", props.name, rows, cols);
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
    if (props.mode === "mainTerminal") {
        const beforeCursor = terminalInputBuffer.slice(0, cursorPosition);
        const afterCursor = terminalInputBuffer.slice(cursorPosition);
        terminalInputBuffer = beforeCursor + text + afterCursor;
        clearCurrentLine();
        terminal.value!.write(terminalInputBuffer);
        cursorPosition += text.length;
        const backspaces = "\b".repeat(afterCursor.length);
        terminal.value!.write(backspaces);
    } else if (props.mode === "interactive") {
        emitAgent(props.endpoint, "terminalInput", props.name, text, (res: any) => {
            if (!res.ok) {
                toastRes(res);
            }
        });
    }
}

function handleContextMenu(event: Event) {
    event.preventDefault();
    if (props.mode === "mainTerminal" || props.mode === "interactive") {
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

onMounted(async () => {
    // Ensure JetBrains Mono is loaded before xterm measures character metrics.
    // Without this, xterm caches fallback monospace dimensions on first render,
    // causing inconsistent cell sizes depending on whether the font was
    // previously loaded by another component (e.g., CodeMirror on compose page).
    await document.fonts.load("14px 'JetBrains Mono'");

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
    });

    if (props.mode === "mainTerminal") {
        mainTerminalConfig();
    } else if (props.mode === "interactive") {
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

    bind();

    if (props.mode === "mainTerminal") {
        emitAgent(props.endpoint, "mainTerminal", props.name, (res: any) => {
            if (!res.ok) {
                toastRes(res);
            }
        });
    } else if (props.mode === "interactive" && props.containerName) {
        console.debug("Create container exec terminal:", props.name);
        emitAgent(props.endpoint, "containerExec", props.containerName, props.shell, (res: any) => {
            if (!res.ok) {
                toastRes(res);
            }
        });
    } else if (props.mode === "interactive") {
        console.debug("Create Interactive terminal:", props.name);
        emitAgent(props.endpoint, "interactiveTerminal", props.stackName!, props.serviceName!, props.shell, (res: any) => {
            if (!res.ok) {
                toastRes(res);
            }
        });
    }

    updateTerminalSize();
});

onUnmounted(() => {
    window.removeEventListener("resize", onResizeEvent);
    if (terminalEl.value) {
        terminalEl.value.removeEventListener("contextmenu", handleContextMenu);
    }
    unbindTerminal(props.name);
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
    background-color: black !important;
    height: 100%;
}
</style>
