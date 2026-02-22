import { useToast } from "vue-toastification";
import { i18n } from "../i18n";

const toast = useToast();

function t(key: string, values?: any): string {
    return (i18n.global as any).t(key, values);
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

    if (res.ok) {
        toast.success(msg);
    } else {
        toast.error(msg);
    }
}

function toastSuccess(msg: string) {
    toast.success(t(msg));
}

function toastError(msg: string) {
    toast.error(t(msg));
}

export function useAppToast() {
    return {
        toastRes,
        toastSuccess,
        toastError,
    };
}
