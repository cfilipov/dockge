<template>
    <form @submit.prevent="submit">
        <div ref="modalRef" class="modal fade" tabindex="-1" data-bs-backdrop="static">
            <div class="modal-dialog">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title">
                            {{ $t("Setup 2FA") }}
                            <span v-if="twoFAStatus == true" class="badge bg-primary">{{ $t("Active") }}</span>
                            <span v-if="twoFAStatus == false" class="badge bg-primary">{{ $t("Inactive") }}</span>
                        </h5>
                        <button :disabled="processing" type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close" />
                    </div>
                    <div class="modal-body">
                        <div class="mb-3">
                            <div v-if="uri && twoFAStatus == false" class="mx-auto text-center" style="width: 210px;">
                                <vue-qrcode :key="uri" :value="uri" type="image/png" :quality="1" :color="{ light: '#ffffffff' }" />
                                <button v-show="!showURI" type="button" class="btn btn-outline-primary btn-sm mt-2" @click="showURI = true">{{ $t("Show URI") }}</button>
                            </div>
                            <p v-if="showURI && twoFAStatus == false" class="text-break mt-2">{{ uri }}</p>

                            <div v-if="!(uri && twoFAStatus == false)" class="mb-3">
                                <label for="current-password" class="form-label">
                                    {{ $t("Current Password") }}
                                </label>
                                <input
                                    id="current-password"
                                    v-model="currentPassword"
                                    type="password"
                                    class="form-control"
                                    autocomplete="current-password"
                                    required
                                />
                            </div>

                            <button v-if="uri == null && twoFAStatus == false" class="btn btn-primary" type="button" @click="prepare2FA()">
                                {{ $t("Enable 2FA") }}
                            </button>

                            <button v-if="twoFAStatus == true" class="btn btn-danger" type="button" :disabled="processing" @click="showConfirmDisable()">
                                {{ $t("Disable 2FA") }}
                            </button>

                            <div v-if="uri && twoFAStatus == false" class="mt-3">
                                <label for="basic-url" class="form-label">{{ $t("twoFAVerifyLabel") }}</label>
                                <div class="input-group">
                                    <input v-model="token" type="text" maxlength="6" class="form-control" autocomplete="one-time-code" required>
                                    <button class="btn btn-outline-primary" type="button" @click="verifyToken()">{{ $t("Verify Token") }}</button>
                                </div>
                                <p v-show="tokenValid" class="mt-2" style="color: green;">{{ $t("tokenValidSettingsMsg") }}</p>
                            </div>
                        </div>
                    </div>

                    <div v-if="uri && twoFAStatus == false" class="modal-footer">
                        <button type="submit" class="btn btn-primary" :disabled="processing || tokenValid == false" @click="showConfirmEnable()">
                            <div v-if="processing" class="spinner-border spinner-border-sm me-1"></div>
                            {{ $t("Save") }}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </form>

    <Confirm ref="confirmEnableRef" btn-style="btn-danger" :yes-text="$t('Yes')" :no-text="$t('No')" @yes="save2FA">
        {{ $t("confirmEnableTwoFAMsg") }}
    </Confirm>

    <Confirm ref="confirmDisableRef" btn-style="btn-danger" :yes-text="$t('Yes')" :no-text="$t('No')" @yes="disable2FA">
        {{ $t("confirmDisableTwoFAMsg") }}
    </Confirm>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { Modal } from "bootstrap";
import Confirm from "./Confirm.vue";
import VueQrcode from "vue-qrcode";
import { useToast } from "vue-toastification";
import { useSocket } from "../composables/useSocket";
import { useAppToast } from "../composables/useAppToast";

const toast = useToast();
const { getSocket } = useSocket();
const { toastRes } = useAppToast();

const modalRef = ref<HTMLElement>();
const confirmEnableRef = ref<InstanceType<typeof Confirm>>();
const confirmDisableRef = ref<InstanceType<typeof Confirm>>();

let modal: Modal | null = null;

const currentPassword = ref("");
const processing = ref(false);
const uri = ref<string | null>(null);
const tokenValid = ref(false);
const twoFAStatus = ref<boolean | null>(null);
const token = ref<string | null>(null);
const showURI = ref(false);

onMounted(() => {
    modal = new Modal(modalRef.value!);
    getStatus();
});

function show() {
    modal?.show();
}

function showConfirmEnable() {
    confirmEnableRef.value?.show();
}

function showConfirmDisable() {
    confirmDisableRef.value?.show();
}

function prepare2FA() {
    processing.value = true;
    getSocket().emit("prepare2FA", currentPassword.value, (res: any) => {
        processing.value = false;
        if (res.ok) {
            uri.value = res.uri;
        } else {
            toast.error(res.msg);
        }
    });
}

function save2FA() {
    processing.value = true;
    getSocket().emit("save2FA", currentPassword.value, (res: any) => {
        processing.value = false;
        if (res.ok) {
            toastRes(res);
            getStatus();
            currentPassword.value = "";
            modal?.hide();
        } else {
            toast.error(res.msg);
        }
    });
}

function disable2FA() {
    processing.value = true;
    getSocket().emit("disable2FA", currentPassword.value, (res: any) => {
        processing.value = false;
        if (res.ok) {
            toastRes(res);
            getStatus();
            currentPassword.value = "";
            modal?.hide();
        } else {
            toast.error(res.msg);
        }
    });
}

function verifyToken() {
    getSocket().emit("verifyToken", token.value, currentPassword.value, (res: any) => {
        if (res.ok) {
            tokenValid.value = res.valid;
        } else {
            toast.error(res.msg);
        }
    });
}

function getStatus() {
    getSocket().emit("twoFAStatus", (res: any) => {
        if (res.ok) {
            twoFAStatus.value = res.status;
        } else {
            toast.error(res.msg);
        }
    });
}

defineExpose({ show });
</script>

<style lang="scss" scoped>
@import "../styles/vars.scss";

.dark {
    .modal-dialog .form-text, .modal-dialog p {
        color: $dark-font-color;
    }
}
</style>
