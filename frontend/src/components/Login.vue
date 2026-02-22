<template>
    <div class="form-container">
        <div class="form">
            <form @submit.prevent="submit">
                <h1 class="h3 mb-3 fw-normal" />

                <div v-if="!tokenRequired" class="form-floating">
                    <input id="floatingInput" v-model="username" type="text" class="form-control" placeholder="Username" autocomplete="username" required>
                    <label for="floatingInput">{{ $t("Username") }}</label>
                </div>

                <div v-if="!tokenRequired" class="form-floating mt-3">
                    <input id="floatingPassword" v-model="password" type="password" class="form-control" placeholder="Password" autocomplete="current-password" required>
                    <label for="floatingPassword">{{ $t("Password") }}</label>
                </div>

                <div v-if="tokenRequired">
                    <div class="form-floating mt-3">
                        <input id="otp" v-model="token" type="text" maxlength="6" class="form-control" placeholder="123456" autocomplete="one-time-code" required>
                        <label for="otp">{{ $t("Token") }}</label>
                    </div>
                </div>

                <div class="form-check mb-3 mt-3 d-flex justify-content-center pe-4">
                    <div class="form-check">
                        <input id="remember" v-model="remember" type="checkbox" value="remember-me" class="form-check-input">

                        <label class="form-check-label" for="remember">
                            {{ $t("Remember me") }}
                        </label>
                    </div>
                </div>
                <div v-if="siteKey" id="turnstile-widget" ref="turnstileEl" class="mt-3 mb-3"></div>

                <button class="w-100 btn btn-primary" type="submit" :disabled="processing">
                    {{ $t("Login") }}
                </button>

                <div v-if="res && !res.ok" class="alert alert-danger mt-3" role="alert">
                    {{ $t(res.msg) }}
                </div>
            </form>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from "vue";
import { useSocket } from "../composables/useSocket";

const { login, remember, getTurnstileSiteKey: fetchTurnstileSiteKey } = useSocket();

const processing = ref(false);
const username = ref("");
const password = ref("");
const token = ref("");
const res = ref<any>(null);
const tokenRequired = ref(false);
const captchaToken = ref("");
const siteKey = ref("");
const turnstileEl = ref<HTMLElement>();

function submit() {
    processing.value = true;

    if (siteKey.value && !captchaToken.value) {
        console.error("CAPTCHA token is missing or invalid.");
        processing.value = false;
        res.value = { ok: false, msg: "Invalid CAPTCHA!" };
        resetTurnstile();
        return;
    }

    login(username.value, password.value, token.value, captchaToken.value, (r: any) => {
        processing.value = false;
        if (r.tokenRequired) {
            tokenRequired.value = true;
        } else if (!r.ok) {
            res.value = r;
            resetTurnstile();
        } else {
            res.value = r;
        }
    });
}

function resetTurnstile() {
    if ((window as any).turnstile && turnstileEl.value) {
        console.log("Resetting Turnstile widget...");
        (window as any).turnstile.reset(turnstileEl.value);
        captchaToken.value = "";
    }
}

function doGetTurnstileSiteKey() {
    fetchTurnstileSiteKey((r: any) => {
        if (r.ok) {
            siteKey.value = r.siteKey;
            if (siteKey.value) {
                console.log("Turnstile site key is provided. Loading Turnstile script...");
                const script = document.createElement("script");
                script.src = "https://challenges.cloudflare.com/turnstile/v0/api.js";
                script.async = true;
                script.defer = true;
                script.onload = () => {
                    console.log("Turnstile script loaded successfully.");
                    initializeTurnstile();
                };
                script.onerror = () => {
                    console.error("Failed to load Turnstile script.");
                };
                document.head.appendChild(script);
                console.log("Turnstile script loaded...");
            } else {
                console.warn("Turnstile site key is not provided. Widget will not be rendered.");
            }
        } else {
            console.error("Failed to fetch Turnstile site key from socket:", r.msg);
        }
    });
}

function initializeTurnstile() {
    if ((window as any).turnstile) {
        console.log("Initializing Turnstile widget...");
        (window as any).turnstile.render(turnstileEl.value, {
            sitekey: siteKey.value,
            callback: (t: string) => {
                captchaToken.value = t;
            },
            "error-callback": () => {
                console.error("Turnstile error occurred");
                captchaToken.value = "";
            },
        });
    }
}

onMounted(() => {
    doGetTurnstileSiteKey();
    document.title += " - Login";
});

onUnmounted(() => {
    document.title = document.title.replace(" - Login", "");
});
</script>

<style lang="scss" scoped>
.form-container {
    display: flex;
    align-items: center;
    padding-top: 40px;
    padding-bottom: 40px;
}

.form-floating {
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
