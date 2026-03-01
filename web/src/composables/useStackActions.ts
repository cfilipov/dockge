import { ref, type Ref } from "vue";
import { useRouter } from "vue-router";
import { useSocket } from "./useSocket";
import { useAppToast } from "./useAppToast";
import type ProgressTerminal from "../components/ProgressTerminal.vue";

export function useStackActions(
    stack: Record<string, any>,
    progressTerminalRef: Ref<InstanceType<typeof ProgressTerminal> | undefined>,
) {
    const router = useRouter();
    const { emit } = useSocket();
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
        emit("startStack", stack.name, (res: any) => {
            stopComposeAction();
            toastRes(res);
        });
    }

    function stopStack() {
        startComposeAction();
        emit("stopStack", stack.name, (res: any) => {
            stopComposeAction();
            toastRes(res);
        });
    }

    function downStack() {
        startComposeAction();
        emit("downStack", stack.name, (res: any) => {
            stopComposeAction();
            toastRes(res);
        });
    }

    function restartStack() {
        startComposeAction();
        emit("restartStack", stack.name, (res: any) => {
            stopComposeAction();
            toastRes(res);
        });
    }

    function doUpdateStack(data: { pruneAfterUpdate: boolean; pruneAllAfterUpdate: boolean }) {
        startComposeAction();
        emit("updateStack", stack.name, data.pruneAfterUpdate, data.pruneAllAfterUpdate, (res: any) => {
            stopComposeAction();
            toastRes(res);
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
