import { ref, watch, type Ref } from "vue";
import { useSocket } from "./useSocket";
import { useAppToast } from "./useAppToast";
import { useContainerStore } from "../stores/containerStore";
import type ProgressTerminal from "../components/ProgressTerminal.vue";

export function useServiceActions(
    stackName: Ref<string>,
    serviceName: Ref<string>,
    progressTerminalRef: Ref<InstanceType<typeof ProgressTerminal> | undefined>,
) {
    const { emit } = useSocket();
    const { toastRes, toastSuccess } = useAppToast();
    const containerStore = useContainerStore();

    const processing = ref(false);
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

    // Watch for Docker events matching the current service action.
    // Only fires while processing is true to avoid spurious toasts.
    watch(() => containerStore.lastEvent, (evt) => {
        if (!evt || !processing.value || !pendingAction) return;
        if (evt.stackName !== stackName.value) return;
        // For service-level actions, also filter by serviceName
        if (serviceName.value && evt.serviceName && evt.serviceName !== serviceName.value) return;

        switch (pendingAction) {
            case "start":
                if (evt.action === "start") {
                    toastSuccess("Started");
                    stopComposeAction();
                }
                break;
            case "stop":
                if (evt.action === "die") {
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
            case "recreate":
                if (evt.action === "start") {
                    toastSuccess("Recreated");
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

    function startService() {
        pendingAction = "start";
        startComposeAction();
        emit("startService", stackName.value, serviceName.value, (res: any) => {
            if (!res.ok) {
                stopComposeAction();
                toastRes(res);
            }
        });
    }

    function stopService() {
        pendingAction = "stop";
        startComposeAction();
        emit("stopService", stackName.value, serviceName.value, (res: any) => {
            if (!res.ok) {
                stopComposeAction();
                toastRes(res);
            }
        });
    }

    function restartService() {
        pendingAction = "restart";
        startComposeAction();
        emit("restartService", stackName.value, serviceName.value, (res: any) => {
            if (!res.ok) {
                stopComposeAction();
                toastRes(res);
            }
        });
    }

    function recreateService() {
        pendingAction = "recreate";
        startComposeAction();
        emit("recreateService", stackName.value, serviceName.value, (res: any) => {
            if (!res.ok) {
                stopComposeAction();
                toastRes(res);
            }
        });
    }

    function doUpdate(data: { pruneAfterUpdate: boolean; pruneAllAfterUpdate: boolean }) {
        pendingAction = "update";
        startComposeAction();
        emit("updateService", stackName.value, serviceName.value, data.pruneAfterUpdate, data.pruneAllAfterUpdate, (res: any) => {
            if (!res.ok) {
                stopComposeAction();
                toastRes(res);
            }
        });
    }

    function checkImageUpdates() {
        processing.value = true;
        emit("checkImageUpdates", stackName.value, (res: any) => {
            processing.value = false;
            toastRes(res);
        });
    }

    return {
        processing,
        showUpdateDialog,
        startComposeAction,
        stopComposeAction,
        startService,
        stopService,
        restartService,
        recreateService,
        doUpdate,
        checkImageUpdates,
    };
}
