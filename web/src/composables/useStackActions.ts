import { ref, type Ref } from "vue";
import { useRouter } from "vue-router";
import { useSocket } from "./useSocket";
import { useAppToast } from "./useAppToast";
import type ProgressTerminal from "../components/ProgressTerminal.vue";

export function useStackActions(
    endpoint: Ref<string>,
    stack: Record<string, any>,
    progressTerminalRef: Ref<InstanceType<typeof ProgressTerminal> | undefined>,
) {
    const router = useRouter();
    const { emitAgent } = useSocket();
    const { toastRes } = useAppToast();

    const processing = ref(true);
    const errorDelete = ref(false);
    const showDeleteDialog = ref(false);
    const deleteStackFiles = ref(false);
    const showForceDeleteDialog = ref(false);
    const showUpdateDialog = ref(false);

    function startComposeAction() {
        processing.value = true;
        progressTerminalRef.value?.show();
    }

    function stopComposeAction() {
        processing.value = false;
    }

    function startStack() {
        startComposeAction();
        emitAgent(endpoint.value, "startStack", stack.name, (res: any) => {
            stopComposeAction();
            toastRes(res);
        });
    }

    function stopStack() {
        startComposeAction();
        emitAgent(endpoint.value, "stopStack", stack.name, (res: any) => {
            stopComposeAction();
            toastRes(res);
        });
    }

    function downStack() {
        startComposeAction();
        emitAgent(endpoint.value, "downStack", stack.name, (res: any) => {
            stopComposeAction();
            toastRes(res);
        });
    }

    function restartStack() {
        startComposeAction();
        emitAgent(endpoint.value, "restartStack", stack.name, (res: any) => {
            stopComposeAction();
            toastRes(res);
        });
    }

    function doUpdateStack(data: { pruneAfterUpdate: boolean; pruneAllAfterUpdate: boolean }) {
        startComposeAction();
        emitAgent(endpoint.value, "updateStack", stack.name, data.pruneAfterUpdate, data.pruneAllAfterUpdate, (res: any) => {
            stopComposeAction();
            toastRes(res);
        });
    }

    function deleteDialog() {
        emitAgent(endpoint.value, "deleteStack", stack.name, { deleteStackFiles: deleteStackFiles.value }, (res: any) => {
            toastRes(res);
            if (res.ok) {
                router.push("/stacks");
            } else {
                errorDelete.value = true;
            }
        });
    }

    function forceDeleteDialog() {
        emitAgent(endpoint.value, "forceDeleteStack", stack.name, (res: any) => {
            toastRes(res);
            if (res.ok) {
                router.push("/stacks");
            }
        });
    }

    function checkImageUpdates(onSuccess?: () => void) {
        processing.value = true;
        emitAgent(endpoint.value, "checkImageUpdates", stack.name, (res: any) => {
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
