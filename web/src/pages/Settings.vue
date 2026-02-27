<template>
    <div>
        <h1 v-show="show" class="mb-3">
            {{ $t("Settings") }}
        </h1>

        <div class="shadow-box shadow-box-settings">
            <div class="row">
                <div v-if="showSubMenu" class="settings-menu col-lg-3 col-md-5">
                    <router-link
                        v-for="(item, key) in subMenus"
                        :key="key"
                        :to="`/settings/${key}`"
                    >
                        <div class="menu-item">
                            {{ item.title }}
                        </div>
                    </router-link>

                    <!-- Logout Button -->
                    <a v-if="isMobile && loggedIn && socketIO.token !== 'autoLogin'" class="logout" @click.prevent="logout">
                        <div class="menu-item">
                            <font-awesome-icon icon="sign-out-alt" />
                            {{ $t("Logout") }}
                        </div>
                    </a>
                </div>
                <div class="settings-content col-lg-9 col-md-7">
                    <div v-if="currentPage" class="settings-content-header" role="heading" aria-level="2">
                        {{ subMenus[currentPage].title }}
                    </div>
                    <div class="mx-3">
                        <router-view v-slot="{ Component }">
                            <transition name="slide-fade" appear>
                                <component :is="Component" />
                            </transition>
                        </router-view>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, provide, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useSocket } from "../composables/useSocket";
import { useTheme } from "../composables/useTheme";
import { useAppToast } from "../composables/useAppToast";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const { getSocket, loggedIn, socketIO, logout } = useSocket();
const { isMobile } = useTheme();
const { toastRes, toastError } = useAppToast();

const show = ref(true);
const settings = ref<Record<string, any>>({});
const settingsLoaded = ref(false);

// Provide to child settings components (Security, General, GlobalEnv, About)
provide("settings", settings);
provide("saveSettings", saveSettings);
provide("settingsLoaded", settingsLoaded);

const currentPage = computed(() => {
    const pathSplit = route.path.split("/");
    const pathEnd = pathSplit[pathSplit.length - 1];
    if (!pathEnd || pathEnd === "settings") {
        return null;
    }
    return pathEnd;
});

const showSubMenu = computed(() => {
    if (isMobile.value) {
        return !currentPage.value;
    }
    return true;
});

const subMenus = computed<Record<string, { title: string }>>(() => ({
    general: { title: t("general") },
    appearance: { title: t("Appearance") },
    security: { title: t("Security") },
    globalEnv: { title: t("GlobalEnv") },
    about: { title: t("About") },
}));

watch(isMobile, () => {
    loadGeneralPage();
});

function loadGeneralPage() {
    if (!currentPage.value && !isMobile.value) {
        router.push("/settings/appearance");
    }
}

function loadSettings() {
    getSocket().emit("getSettings", (res: any) => {
        settings.value = res.data;
        if (settings.value.checkUpdate === undefined) {
            settings.value.checkUpdate = true;
        }
        if (settings.value.imageUpdateCheckEnabled === undefined) {
            settings.value.imageUpdateCheckEnabled = true;
        }
        if (settings.value.imageUpdateCheckInterval === undefined) {
            settings.value.imageUpdateCheckInterval = 6;
        }
        settingsLoaded.value = true;
    });
}

function saveSettings(callback?: () => void, currentPassword?: string) {
    const valid = validateSettings();
    if (valid.success) {
        getSocket().emit("setSettings", settings.value, currentPassword, (res: any) => {
            toastRes(res);
            loadSettings();
            if (callback) {
                callback();
            }
        });
    } else {
        toastError(valid.msg);
    }
}

function validateSettings() {
    if (settings.value.keepDataPeriodDays < 0) {
        return {
            success: false,
            msg: t("dataRetentionTimeError"),
        };
    }
    return {
        success: true,
        msg: "",
    };
}

onMounted(() => {
    loadSettings();
    loadGeneralPage();
});
</script>

<style lang="scss" scoped>
@import "../styles/vars.scss";

.shadow-box-settings {
    padding: 20px;
    min-height: calc(100vh - 155px);
}

footer {
    color: #aaa;
    font-size: 13px;
    margin-top: 20px;
    padding-bottom: 30px;
    text-align: center;
}

.settings-menu {
    a {
        text-decoration: none !important;
    }

    .menu-item {
        border-radius: 10px;
        margin: 0.5em;
        padding: 0.7em 1em;
        cursor: pointer;
        border-left-width: 0;
        transition: all ease-in-out 0.1s;
    }

    .menu-item:hover {
        background: $highlight-white;

        .dark & {
            background: $dark-header-bg;
        }
    }

    .active .menu-item {
        background: $highlight-white;
        border-left: 4px solid $primary;
        border-top-left-radius: 0;
        border-bottom-left-radius: 0;

        .dark & {
            background: $dark-header-bg;
        }
    }
}

.settings-content {
    .settings-content-header {
        width: calc(100% + 20px);
        border-bottom: 1px solid #dee2e6;
        border-radius: 0 10px 0 0;
        margin-top: -20px;
        margin-right: -20px;
        padding: 12.5px 1em;
        font-size: 26px;

        .dark & {
            background: $dark-header-bg;
            border-bottom: 0;
        }

        .mobile & {
            padding: 15px 0 0 0;

            .dark & {
                background-color: transparent;
            }
        }
    }
}

.logout {
    color: $danger !important;
}
</style>
