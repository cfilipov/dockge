import { ref, watch, type Ref } from "vue";
import { useRouter } from "vue-router";
import { useSocket } from "./useSocket";
import { useAppToast } from "./useAppToast";
import { useContainerStore } from "../stores/containerStore";
import type ProgressTerminal from "../components/ProgressTerminal.vue";

export function useStackActions(
    stack: Record<string, any>,
    progressTerminalRef: Ref<InstanceType<typeof ProgressTerminal> | undefined>,
) {
    const router = useRouter();
    const { emit } = useSocket();
    const { toastRes, toastSuccess } = useAppToast();
    const containerStore = useContainerStore();

    const processing = ref(true);
    const errorDelete = ref(false);
    const showDeleteDialog = ref(false);
    const deleteStackFiles = ref(false);
    const showForceDeleteDialog = ref(false);
    const showUpdateDialog = ref(false);

    // Track which action is in flight so the event watcher knows what toast to show.
    let pendingAction: string | null = null;

    function startComposeAction() {
        processing.value = true;
        progressTerminalRef.value?.show();
    }

    function stopComposeAction() {
        processing.value = false;
        pendingAction = null;
    }

    // Watch for Docker events matching the current stack action.
    // Only fires while processing is true to avoid spurious toasts.
    watch(() => containerStore.lastEvent, (evt) => {
        if (!evt || !processing.value || !pendingAction) return;
        if (evt.stackName !== stack.name) return;

        switch (pendingAction) {
            case "start":
                if (evt.action === "start") {
                    toastSuccess("Started");
                    stopComposeAction();
                }
                break;
            case "stop":
                if (evt.action === "die" || evt.action === "stop") {
                    toastSuccess("Stopped");
                    stopComposeAction();
                }
                break;
            case "down":
                if (evt.action === "destroy") {
                    toastSuccess("Stopped");
                    stopComposeAction();
                }
                break;
            case "restart":
                if (evt.action === "start") {
                    toastSuccess("Restarted");
                    stopComposeAction();
                }
                break;
            case "update":
                if (evt.action === "start") {
                    toastSuccess("Updated");
                    stopComposeAction();
                }
                break;
        }
    });

    function startStack() {
        pendingAction = "start";
        startComposeAction();
        emit("startStack", stack.name, (res: any) => {
            if (!res.ok) {
                stopComposeAction();
                toastRes(res);
            }
        });
    }

    function stopStack() {
        pendingAction = "stop";
        startComposeAction();
        emit("stopStack", stack.name, (res: any) => {
            if (!res.ok) {
                stopComposeAction();
                toastRes(res);
            }
        });
    }

    function downStack() {
        pendingAction = "down";
        startComposeAction();
        emit("downStack", stack.name, (res: any) => {
            if (!res.ok) {
                stopComposeAction();
                toastRes(res);
            }
        });
    }

    function restartStack() {
        pendingAction = "restart";
        startComposeAction();
        emit("restartStack", stack.name, (res: any) => {
            if (!res.ok) {
                stopComposeAction();
                toastRes(res);
            }
        });
    }

    function doUpdateStack(data: { pruneAfterUpdate: boolean; pruneAllAfterUpdate: boolean }) {
        pendingAction = "update";
        startComposeAction();
        emit("updateStack", stack.name, data.pruneAfterUpdate, data.pruneAllAfterUpdate, (res: any) => {
            if (!res.ok) {
                stopComposeAction();
                toastRes(res);
            }
        });
    }

    function deleteDialog() {
        emit("deleteStack", stack.name, { deleteStackFiles: deleteStackFiles.value }, (res: any) => {
            toastRes(res);
            if (res.ok) {
                router.push("/stacks");
            } else {
                errorDelete.value = true;
            }
        });
    }

    function forceDeleteDialog() {
        emit("forceDeleteStack", stack.name, (res: any) => {
            toastRes(res);
            if (res.ok) {
                router.push("/stacks");
            }
        });
    }

    function checkImageUpdates(onSuccess?: () => void) {
        processing.value = true;
        emit("checkImageUpdates", stack.name, (res: any) => {
            processing.value = false;
            toastRes(res);
            if (res.ok && onSuccess) {
                onSuccess();
            }
        });
    }

    return {
        processing,
        errorDelete,
        showDeleteDialog,
        deleteStackFiles,
        showForceDeleteDialog,
        showUpdateDialog,
        startComposeAction,
        stopComposeAction,
        startStack,
        stopStack,
        downStack,
        restartStack,
        doUpdateStack,
        deleteDialog,
        forceDeleteDialog,
        checkImageUpdates,
    };
}
