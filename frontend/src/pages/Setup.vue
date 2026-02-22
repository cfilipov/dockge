<template>
    <div class="form-container" data-cy="setup-form">
        <div class="form">
            <form @submit.prevent="submit">
                <div>
                    <object width="64" height="64" data="/icon.svg" />
                    <div style="font-size: 28px; font-weight: bold; margin-top: 5px;">
                        Dockge
                    </div>
                </div>

                <p class="mt-3">
                    {{ $t("Create your admin account") }}
                </p>

                <div class="form-floating">
                    <select id="language" v-model="language" class="form-select">
                        <option v-for="(lang, i) in $i18n.availableLocales" :key="`Lang${i}`" :value="lang">
                            {{ $i18n.messages[lang].languageName }}
                        </option>
                    </select>
                    <label for="language" class="form-label">{{ $t("Language") }}</label>
                </div>

                <div class="form-floating mt-3">
                    <input id="floatingInput" v-model="username" type="text" class="form-control" :placeholder="$t('Username')" required data-cy="username-input">
                    <label for="floatingInput">{{ $t("Username") }}</label>
                </div>

                <div class="form-floating mt-3">
                    <input id="floatingPassword" v-model="password" type="password" class="form-control" :placeholder="$t('Password')" required data-cy="password-input">
                    <label for="floatingPassword">{{ $t("Password") }}</label>
                </div>

                <div class="form-floating mt-3">
                    <input id="repeat" v-model="repeatPassword" type="password" class="form-control" :placeholder="$t('Repeat Password')" required data-cy="password-repeat-input">
                    <label for="repeat">{{ $t("Repeat Password") }}</label>
                </div>

                <button class="w-100 btn btn-primary mt-3" type="submit" :disabled="processing" data-cy="submit-setup-form">
                    {{ $t("Create") }}
                </button>
            </form>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRouter } from "vue-router";
import { useSocket } from "../composables/useSocket";
import { useLang } from "../composables/useLang";
import { useAppToast } from "../composables/useAppToast";

const router = useRouter();
const { getSocket, login } = useSocket();
const { language } = useLang();
const { toastRes, toastError } = useAppToast();

const processing = ref(false);
const username = ref("");
const password = ref("");
const repeatPassword = ref("");

onMounted(() => {
    getSocket().emit("needSetup", (needSetup: boolean) => {
        if (!needSetup) {
            router.push("/");
        }
    });
});

function submit() {
    processing.value = true;

    if (password.value !== repeatPassword.value) {
        toastError("PasswordsDoNotMatch");
        processing.value = false;
        return;
    }

    getSocket().emit("setup", username.value, password.value, (res: any) => {
        processing.value = false;
        toastRes(res);

        if (res.ok) {
            processing.value = true;

            login(username.value, password.value, "", "", () => {
                processing.value = false;
                router.push("/");
            });
        }
    });
}
</script>

<style lang="scss" scoped>
.form-container {
    display: flex;
    align-items: center;
    padding-top: 40px;
    padding-bottom: 40px;
}

.form-floating {
    > .form-select {
        padding-left: 1.3rem;
        padding-top: 1.525rem;
        line-height: 1.35;

        ~ label {
            padding-left: 1.3rem;
        }
    }

    > label {
        padding-left: 1.3rem;
    }

    > .form-control {
        padding-left: 1.3rem;
    }
}

.form {

    width: 100%;
    max-width: 330px;
    padding: 15px;
    margin: auto;
    text-align: center;
}
</style>
