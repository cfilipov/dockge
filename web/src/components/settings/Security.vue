<template>
    <div>
        <div v-if="settingsLoaded" class="my-4">
            <!-- Change Password -->
            <template v-if="!settings.disableAuth">
                <p>
                    {{ $t("Current User") }}: <strong>{{ username }}</strong>
                    <button v-if="! settings.disableAuth" id="logout-btn" class="btn btn-danger ms-4 me-2 mb-2" @click="logout">{{ $t("Logout") }}</button>
                </p>

                <h5 class="my-4 settings-subheading">{{ $t("Change Password") }}</h5>
                <form class="mb-3" @submit.prevent="savePassword">
                    <div class="mb-3">
                        <label for="current-password" class="form-label">
                            {{ $t("Current Password") }}
                        </label>
                        <input
                            id="current-password"
                            v-model="password.currentPassword"
                            type="password"
                            class="form-control"
                            autocomplete="current-password"
                            required
                        />
                    </div>

                    <div class="mb-3">
                        <label for="new-password" class="form-label">
                            {{ $t("New Password") }}
                        </label>
                        <input
                            id="new-password"
                            v-model="password.newPassword"
                            type="password"
                            class="form-control"
                            autocomplete="new-password"
                            required
                        />
                    </div>

                    <div class="mb-3">
                        <label for="repeat-new-password" class="form-label">
                            {{ $t("Repeat New Password") }}
                        </label>
                        <input
                            id="repeat-new-password"
                            v-model="password.repeatNewPassword"
                            type="password"
                            class="form-control"
                            :class="{ 'is-invalid': invalidPassword }"
                            autocomplete="new-password"
                            required
                        />
                        <div class="invalid-feedback">
                            {{ $t("passwordNotMatchMsg") }}
                        </div>
                    </div>

                    <div>
                        <button class="btn btn-primary" type="submit">
                            {{ $t("Update Password") }}
                        </button>
                    </div>
                </form>
            </template>

            <!-- TODO: Hidden for now -->
            <div v-if="! settings.disableAuth && false" class="mt-5 mb-3">
                <h5 class="my-4 settings-subheading">
                    {{ $t("Two Factor Authentication") }}
                </h5>
                <div class="mb-4">
                    <button
                        class="btn btn-primary me-2"
                        type="button"
                        @click="TwoFADialogRef?.show()"
                    >
                        {{ $t("2FA Settings") }}
                    </button>
                </div>
            </div>

            <div class="my-4">
                <!-- Advanced -->
                <h5 class="my-4 settings-subheading">{{ $t("Advanced") }}</h5>

                <div class="mb-4">
                    <button v-if="settings.disableAuth" id="enableAuth-btn" class="btn btn-outline-primary me-2 mb-2" @click="enableAuth">{{ $t("Enable Auth") }}</button>
                    <button v-if="! settings.disableAuth" id="disableAuth-btn" class="btn btn-primary me-2 mb-2" @click="confirmDisableAuth">{{ $t("Disable Auth") }}</button>
                </div>
            </div>
        </div>

        <TwoFADialog ref="TwoFADialogRef" />

        <Confirm ref="confirmDisableAuthRef" btn-style="btn-danger" :yes-text="$t('I understand, please disable')" :no-text="$t('Leave')" @yes="disableAuth">
            <!-- eslint-disable-next-line vue/no-v-html -->
            <p v-html="$t('disableauth.message1')"></p>
            <!-- eslint-disable-next-line vue/no-v-html -->
            <p v-html="$t('disableauth.message2')"></p>
            <p>{{ $t("Please use this option carefully!") }}</p>

            <div class="mb-3">
                <label for="current-password2" class="form-label">
                    {{ $t("Current Password") }}
                </label>
                <input
                    id="current-password2"
                    v-model="password.currentPassword"
                    type="password"
                    class="form-control"
                    required
                />
            </div>
        </Confirm>
    </div>
</template>

<script setup lang="ts">
import { ref, inject, watch, type Ref } from "vue";
import Confirm from "../../components/Confirm.vue";
import TwoFADialog from "../../components/TwoFADialog.vue";
import { useSocket } from "../../composables/useSocket";
import { useAppToast } from "../../composables/useAppToast";

const settings = inject<Ref<Record<string, any>>>("settings")!;
const saveSettings = inject<(callback?: () => void, currentPassword?: string) => void>("saveSettings")!;
const settingsLoaded = inject<Ref<boolean>>("settingsLoaded")!;

const { getSocket, username, socketIO, logout, storage } = useSocket();
const { toastRes } = useAppToast();

const invalidPassword = ref(false);
const password = ref({
    currentPassword: "",
    newPassword: "",
    repeatNewPassword: "",
});

const TwoFADialogRef = ref<InstanceType<typeof TwoFADialog>>();
const confirmDisableAuthRef = ref<InstanceType<typeof Confirm>>();

watch(() => password.value.repeatNewPassword, () => {
    invalidPassword.value = false;
});

function savePassword() {
    if (password.value.newPassword !== password.value.repeatNewPassword) {
        invalidPassword.value = true;
    } else {
        getSocket().emit("changePassword", password.value, (res: any) => {
            toastRes(res);
            if (res.ok) {
                password.value.currentPassword = "";
                password.value.newPassword = "";
                password.value.repeatNewPassword = "";
            }
        });
    }
}

function disableAuth() {
    settings.value.disableAuth = true;
    saveSettings(() => {
        password.value.currentPassword = "";
        username.value = "";
        socketIO.token = "autoLogin";
    }, password.value.currentPassword);
}

function enableAuth() {
    settings.value.disableAuth = false;
    saveSettings();
    storage().removeItem("token");
    location.reload();
}

function confirmDisableAuth() {
    confirmDisableAuthRef.value?.show();
}
</script>
