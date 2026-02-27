import { ref, type Ref } from "vue";
import { useSocket } from "./useSocket";
import { useAppToast } from "./useAppToast";
import type ProgressTerminal from "../components/ProgressTerminal.vue";

export function useServiceActions(
    endpoint: Ref<string>,
    stackName: Ref<string>,
    serviceName: Ref<string>,
    progressTerminalRef: Ref<InstanceType<typeof ProgressTerminal> | undefined>,
) {
    const { emitAgent } = useSocket();
    const { toastRes } = useAppToast();

    const processing = ref(false);
    const showUpdateDialog = ref(false);

    function startComposeAction() {
        processing.value = true;
        progressTerminalRef.value?.show();
    }

    function stopComposeAction() {
        processing.value = false;
    }

    function startService() {
        startComposeAction();
        emitAgent(endpoint.value, "startService", stackName.value, serviceName.value, (res: any) => {
            stopComposeAction();
            toastRes(res);
        });
    }

    function stopService() {
        startComposeAction();
        emitAgent(endpoint.value, "stopService", stackName.value, serviceName.value, (res: any) => {
            stopComposeAction();
            toastRes(res);
        });
    }

    function restartService() {
        startComposeAction();
        emitAgent(endpoint.value, "restartService", stackName.value, serviceName.value, (res: any) => {
            stopComposeAction();
            toastRes(res);
        });
    }

    function recreateService() {
        startComposeAction();
        emitAgent(endpoint.value, "restartService", stackName.value, serviceName.value, (res: any) => {
            stopComposeAction();
            toastRes(res);
        });
    }

    function doUpdate(data: { pruneAfterUpdate: boolean; pruneAllAfterUpdate: boolean }) {
        startComposeAction();
        emitAgent(endpoint.value, "updateService", stackName.value, serviceName.value, data.pruneAfterUpdate, data.pruneAllAfterUpdate, (res: any) => {
            stopComposeAction();
            toastRes(res);
        });
    }

    function checkImageUpdates() {
        processing.value = true;
        emitAgent(endpoint.value, "checkImageUpdates", stackName.value, (res: any) => {
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
