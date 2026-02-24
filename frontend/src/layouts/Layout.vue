<template>
    <div :class="classes">
        <div v-if="! socketIO.connected && ! socketIO.firstConnect" class="lost-connection">
            <div class="container-fluid">
                {{ socketIO.connectionErrorMsg }}
                <div v-if="socketIO.showReverseProxyGuide">
                    {{ $t("reverseProxyMsg1") }} <a href="https://github.com/louislam/uptime-kuma/wiki/Reverse-Proxy" target="_blank">{{ $t("reverseProxyMsg2") }}</a>
                </div>
            </div>
        </div>

        <!-- Desktop header -->
        <header v-if="! isMobile" class="d-flex flex-wrap justify-content-center py-3 mb-3 border-bottom">
            <router-link to="/stacks" class="d-flex align-items-center mb-3 mb-md-0 me-md-auto text-dark text-decoration-none">
                <object class="bi me-2 ms-4" width="40" height="40" data="/icon.svg" />
                <span class="fs-4 title">Dockge</span>
            </router-link>

            <a v-if="hasNewVersion" target="_blank" href="https://github.com/louislam/dockge/releases" class="btn btn-warning me-3">
                <font-awesome-icon icon="arrow-alt-circle-up" /> {{ $t("newUpdate") }}
            </a>

            <ul class="nav nav-pills">
                <li v-if="loggedIn" class="nav-item me-2">
                    <router-link to="/stacks" class="nav-link">
                        <font-awesome-icon icon="layer-group" /> {{ $t("stacks") }}
                    </router-link>
                </li>

                <li v-if="loggedIn" class="nav-item me-2">
                    <router-link :to="containersTabLink" class="nav-link">
                        <font-awesome-icon icon="cubes" /> {{ $t("containersNav") }}
                    </router-link>
                </li>

                <li v-if="loggedIn" class="nav-item me-2">
                    <router-link to="/networks" class="nav-link">
                        <font-awesome-icon icon="network-wired" /> {{ $t("networksNav") }}
                    </router-link>
                </li>

                <li v-if="loggedIn" class="nav-item me-2">
                    <router-link :to="logsTabLink" class="nav-link">
                        <font-awesome-icon icon="file-lines" /> {{ $t("logs") }}
                    </router-link>
                </li>

                <li v-if="loggedIn" class="nav-item me-2">
                    <router-link :to="shellTabLink" class="nav-link">
                        <font-awesome-icon icon="code" /> {{ $t("shell") }}
                    </router-link>
                </li>

                <li v-if="loggedIn" class="nav-item me-2">
                    <router-link to="/yaml" class="nav-link">
                        <font-awesome-icon icon="file-code" /> {{ $t("yaml") }}
                    </router-link>
                </li>

                <li v-if="loggedIn" class="nav-item me-2">
                    <router-link to="/images" class="nav-link">
                        <font-awesome-icon icon="box-archive" /> {{ $t("imagesNav") }}
                    </router-link>
                </li>

                <li v-if="loggedIn" class="nav-item me-2">
                    <router-link to="/volumes" class="nav-link">
                        <font-awesome-icon icon="hard-drive" /> {{ $t("volumesNav") }}
                    </router-link>
                </li>

                <li v-if="loggedIn" class="nav-item me-2">
                    <router-link to="/console" class="nav-link">
                        <font-awesome-icon icon="terminal" /> {{ $t("console") }}
                    </router-link>
                </li>

                <li v-if="loggedIn" class="nav-item">
                    <div class="dropdown dropdown-profile-pic">
                        <div class="nav-link" data-bs-toggle="dropdown">
                            <div class="profile-pic">{{ usernameFirstChar }}</div>
                            <font-awesome-icon icon="angle-down" />
                        </div>

                        <!-- Header's Dropdown Menu -->
                        <ul class="dropdown-menu">
                            <!-- Username -->
                            <li>
                                <i18n-t v-if="username != null" tag="span" keypath="signedInDisp" class="dropdown-item-text">
                                    <strong>{{ username }}</strong>
                                </i18n-t>
                                <span v-if="username == null" class="dropdown-item-text">{{ $t("signedInDispDisabled") }}</span>
                            </li>

                            <li><hr class="dropdown-divider"></li>

                            <!-- Functions -->

                            <li>
                                <button class="dropdown-item" @click="scanFolder">
                                    <font-awesome-icon icon="arrows-rotate" /> {{ $t("scanFolder") }}
                                </button>
                            </li>

                            <li>
                                <router-link to="/settings/general" class="dropdown-item" :class="{ active: $route.path.includes('settings') }">
                                    <font-awesome-icon icon="cog" /> {{ $t("Settings") }}
                                </router-link>
                            </li>

                            <li>
                                <button class="dropdown-item" @click="logout">
                                    <font-awesome-icon icon="sign-out-alt" />
                                    {{ $t("Logout") }}
                                </button>
                            </li>
                        </ul>
                    </div>
                </li>
            </ul>
        </header>

        <main>
            <div v-if="socketIO.connecting" class="container mt-5">
                <h4>{{ $t("connecting...") }}</h4>
            </div>

            <router-view v-if="loggedIn" />
            <Login v-if="! loggedIn && allowLoginDialog" />
        </main>
    </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import { compareVersions } from "compare-versions";
import { ALL_ENDPOINTS } from "../../../common/util-common";
import Login from "../components/Login.vue";
import { useSocket } from "../composables/useSocket";
import { useTheme } from "../composables/useTheme";
import { useAppToast } from "../composables/useAppToast";

const route = useRoute();

const {
    socketIO,
    loggedIn,
    allowLoginDialog,
    username,
    usernameFirstChar,
    info,
    emitAgent,
    logout,
} = useSocket();

const { theme, isMobile } = useTheme();
const { toastRes } = useAppToast();

// Which container-centric tab we're currently on (if any)
const currentTab = computed(() => {
    if (route.path.startsWith("/containers")) return "containers";
    if (route.path.startsWith("/logs")) return "logs";
    if (route.path.startsWith("/shell")) return "shell";
    return "";
});

// The currently selected container name (shared across containers/logs/shell tabs)
const selectedContainer = computed(() => (route.params.containerName as string) || "");

// Tab links: carry the selected container to the other tab, or go home if clicking the same tab
const containersTabLink = computed(() => {
    if (currentTab.value === "containers") return "/containers";
    if (selectedContainer.value) return { name: "containerDetail", params: { containerName: selectedContainer.value } };
    return "/containers";
});

const logsTabLink = computed(() => {
    if (currentTab.value === "logs") return "/logs";
    if (selectedContainer.value) return { name: "containerLogs", params: { containerName: selectedContainer.value } };
    return "/logs";
});

const shellTabLink = computed(() => {
    if (currentTab.value === "shell") return "/shell";
    if (selectedContainer.value) return { name: "containerShell", params: { containerName: selectedContainer.value, type: "bash" } };
    return "/shell";
});

const classes = computed(() => {
    const cls: Record<string, boolean> = {};
    cls[theme.value] = true;
    cls["mobile"] = isMobile.value;
    return cls;
});

const hasNewVersion = computed(() => {
    if (info.value.latestVersion && info.value.version) {
        return compareVersions(info.value.latestVersion, info.value.version) >= 1;
    } else {
        return false;
    }
});

function scanFolder() {
    emitAgent(ALL_ENDPOINTS, "requestStackList", (res: any) => {
        toastRes(res);
    });
}
</script>

<style lang="scss" scoped>
@import "../styles/vars.scss";

.nav-link {
    &.status-page {
        background-color: rgba(255, 255, 255, 0.1);
    }
}

.bottom-nav {
    z-index: 1000;
    position: fixed;
    bottom: 0;
    height: calc(60px + env(safe-area-inset-bottom));
    width: 100%;
    left: 0;
    background-color: #fff;
    box-shadow: 0 15px 47px 0 rgba(0, 0, 0, 0.05), 0 5px 14px 0 rgba(0, 0, 0, 0.05);
    text-align: center;
    white-space: nowrap;
    padding: 0 10px env(safe-area-inset-bottom);

    a {
        text-align: center;
        width: 25%;
        display: inline-block;
        height: 100%;
        padding: 8px 10px 0;
        font-size: 13px;
        color: #c1c1c1;
        overflow: hidden;
        text-decoration: none;

        &.router-link-exact-active, &.active {
            color: $primary;
            font-weight: bold;
        }

        div {
            font-size: 20px;
        }
    }
}

main {
    min-height: calc(100vh - 160px);
}

.title {
    font-weight: bold;
}

.nav {
    margin-right: 25px;
}

.lost-connection {
    padding: 5px;
    background-color: crimson;
    color: white;
    position: fixed;
    width: 100%;
    z-index: 99999;
}

// Profile Pic Button with Dropdown
.dropdown-profile-pic {
    user-select: none;

    .nav-link {
        cursor: pointer;
        display: flex;
        gap: 6px;
        align-items: center;
        background-color: rgba(200, 200, 200, 0.2);
        padding: 0.5rem 0.8rem;

        &:hover {
            background-color: rgba(255, 255, 255, 0.2);
        }
    }

    .dropdown-menu {
        transition: all 0.2s;
        padding-left: 0;
        padding-bottom: 0;
        margin-top: 8px !important;
        border-radius: 16px;
        overflow: hidden;

        .dropdown-divider {
            margin: 0;
            border-top: 1px solid rgba(0, 0, 0, 0.4);
            background-color: transparent;
        }

        .dropdown-item-text {
            font-size: 14px;
            padding-bottom: 0.7rem;
        }

        .dropdown-item {
            padding: 0.7rem 1rem;
        }

        .dark & {
            background-color: $dark-bg;
            color: $dark-font-color;
            border-color: $dark-border-color;

            .dropdown-item {
                color: $dark-font-color;

                &.active {
                    color: $dark-font-color2;
                    background-color: $highlight !important;
                }

                &:hover {
                    background-color: $dark-bg2;
                }
            }
        }
    }

    .profile-pic {
        display: flex;
        align-items: center;
        justify-content: center;
        color: white;
        background-color: $primary;
        width: 24px;
        height: 24px;
        margin-right: 5px;
        border-radius: 50rem;
        font-weight: bold;
        font-size: 10px;
    }
}

.dark {
    header {
        background-color: $dark-header-bg;
        border-bottom-color: $dark-header-bg !important;

        span {
            color: #f0f6fc;
        }
    }

    .bottom-nav {
        background-color: $dark-bg;
    }
}
</style>
