<template>
    <div
        class="log-view shadow-box"
        :class="{ dark: isDark }"
        role="region"
        :aria-label="ariaLabel || 'Log view'"
    >
        <!-- Spinner shown until first data arrives -->
        <div v-if="!hasData" class="log-spinner">
            <span class="spinner-icon">{{ spinnerFrame }}</span> Connecting...
        </div>

        <VList
            ref="vlistRef"
            :data="logEntries"
            :shift="stickToBottom"
            class="log-vlist"
            @scroll="onScroll"
        >
            <template #default="{ item }">
                <div v-if="item.type === 'banner'" class="log-banner" :class="item.action">
                    <span class="log-banner-label">
                        {{ item.action === 'start' ? 'CONTAINER START' : 'CONTAINER STOP' }}
                        &mdash; {{ item.name }}
                    </span>
                </div>
                <pre v-else class="log-line" v-html="item.html" />
            </template>
        </VList>
    </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from "vue";
import { VList } from "virtua/vue";
import { useTheme } from "../composables/useTheme";
import { useSocket } from "../composables/useSocket";
import { createLogStore } from "../common/log-store";

const { isDark } = useTheme();

const props = withDefaults(defineProps<{
    name: string;
    ariaLabel?: string;
    terminalType: string;
    terminalParams?: Record<string, string>;
}>(), {
    ariaLabel: undefined,
    terminalParams: undefined,
});

const emit = defineEmits<{
    (e: "has-data"): void;
}>();

const vlistRef = ref<InstanceType<typeof VList> | null>(null);
const hasData = ref(false);
const stickToBottom = ref(true);

// Spinner animation
const SPINNER_FRAMES = ["\u280b", "\u2819", "\u2839", "\u2838", "\u283c", "\u2834", "\u2826", "\u2827", "\u2807", "\u280f"];
const spinnerFrame = ref(SPINNER_FRAMES[0]);
let spinnerInterval: ReturnType<typeof setInterval> | null = null;
let spinnerTimer: ReturnType<typeof setTimeout> | null = null;

function startSpinnerDebounce() {
    let frameIdx = 0;
    spinnerTimer = setTimeout(() => {
        spinnerInterval = setInterval(() => {
            frameIdx = (frameIdx + 1) % SPINNER_FRAMES.length;
            spinnerFrame.value = SPINNER_FRAMES[frameIdx];
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

// Log store — initialized eagerly so the template can bind to entries
const { getSocket, emit: socketEmit } = useSocket();

const store = createLogStore({
    terminalType: props.terminalType as "container-log" | "container-log-by-name" | "combined",
    containerName: props.terminalParams?.container || props.terminalParams?.service,
    stackName: props.terminalParams?.stack,
});

// Expose entries as a top-level computed so Vue tracks the shallowRef reactivity
const logEntries = computed(() => store.entries.value);

// Socket event listener — stored for cleanup
let offLogData: (() => void) | null = null;

function connectLogs() {
    startSpinnerDebounce();

    // Listen for logData events from the server
    const handler = (data: { ts: number; line: string }) => {
        if (!hasData.value) {
            stopSpinner();
            hasData.value = true;
            emit("has-data");
        }
        store.addLine(data.ts, data.line);
    };
    const socket = getSocket();
    socket.on("logData", handler);
    offLogData = () => socket.off("logData", handler);

    // Subscribe to log stream
    socketEmit("subscribeLogs", {
        type: props.terminalType,
        stack: props.terminalParams?.stack,
        service: props.terminalParams?.service,
        container: props.terminalParams?.container,
    });
}

// Stick-to-bottom: track scroll position via the VList's scroll event.
// If user scrolls up, disable stick-to-bottom. If they scroll back to
// the bottom, re-enable it.
function onScroll() {
    if (!vlistRef.value) {
        return;
    }
    const vl = vlistRef.value;
    const atBottom = vl.scrollOffset + vl.viewportSize >= vl.scrollSize - 20;
    stickToBottom.value = atBottom;
}

onMounted(() => {
    connectLogs();
});

onUnmounted(() => {
    stopSpinner();
    // Unregister socket listener to prevent leaks on remount
    if (offLogData) {
        offLogData();
        offLogData = null;
    }
    // Unsubscribe from server log stream
    socketEmit("unsubscribeLogs");
    store.destroy();
});
</script>

<style scoped lang="scss">
.log-view {
    height: 100%;
    font-family: 'JetBrains Mono', monospace;
    font-size: 14px;
    line-height: 1.3;
    overflow: hidden;
    position: relative;

    // Dark theme (default terminal look)
    background-color: #000000;
    color: #cccccc;

    &.dark {
        background-color: #000000;
        color: #cccccc;
    }

    &:not(.dark) {
        background-color: #ffffff;
        color: #333333;
    }
}

.log-vlist {
    height: 100%;
}

.log-spinner {
    padding: 8px 12px;
    color: #888;
}

.spinner-icon {
    display: inline-block;
    width: 1em;
}

// Log lines — monospace pre-formatted
.log-line {
    margin: 0;
    padding: 0 8px;
    white-space: pre-wrap;
    word-break: break-all;
    font-family: inherit;
    font-size: inherit;
    line-height: inherit;
    overflow: hidden; // prevent <pre> default overflow:auto from capturing scroll events
}

// VList item wrappers — remove any extra spacing
:deep(.log-vlist > div > div) {
    min-height: 0 !important;
}

// Banner — horizontal rule with centered label
.log-banner {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 6px 8px;
    margin: 2px 0;

    &::before,
    &::after {
        content: "";
        flex: 1;
        height: 1px;
    }

    &.start::before,
    &.start::after {
        background-color: #74c2ff;
    }

    &.die::before,
    &.die::after {
        background-color: #f8a306;
    }
}

.log-banner-label {
    font-weight: bold;
    font-size: 12px;
    white-space: nowrap;
    letter-spacing: 0.5px;
    text-transform: uppercase;
}

.log-banner.start .log-banner-label {
    color: #74c2ff;
}

.log-banner.die .log-banner-label {
    color: #f8a306;
}
</style>
