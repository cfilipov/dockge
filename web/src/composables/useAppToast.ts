import { useToast } from "vue-toastification";
import type { ToastID } from "vue-toastification/dist/types/types";
import { i18n } from "../i18n";
import ToastBody from "../components/ToastBody.vue";

const toast = useToast();

function t(key: string, values?: any): string {
    return (i18n.global as any).t(key, values);
}

/**
 * When the user clicks the toast body, cancel auto-dismiss so the
 * message stays visible until the X button is used.
 */
function pinOnClick(id: ToastID) {
    return () => {
        toast.update(id, { options: { timeout: false } });
    };
}

function toastRes(res: any) {
    let msg = res.msg;
    if (res.msgi18n) {
        if (msg != null && typeof msg === "object") {
            msg = t(msg.key, msg.values);
        } else {
            msg = t(msg);
        }
    }

    const content = { component: ToastBody, props: { message: msg } };
    let id: ToastID;

    if (res.ok) {
        id = toast.success(content, { onClick: () => pinOnClick(id)() });
    } else {
        id = toast.error(content, { onClick: () => pinOnClick(id)() });
    }
}

function toastSuccess(msg: string) {
    const message = t(msg);
    const content = { component: ToastBody, props: { message } };
    let id: ToastID;
    id = toast.success(content, { onClick: () => pinOnClick(id)() });
}

function toastError(msg: string) {
    const message = t(msg);
    const content = { component: ToastBody, props: { message } };
    let id: ToastID;
    id = toast.error(content, { onClick: () => pinOnClick(id)() });
}

export function useAppToast() {
    return {
        toastRes,
        toastSuccess,
        toastError,
    };
}
