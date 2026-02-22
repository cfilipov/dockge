<template>
    <div>
        <form class="my-4" autocomplete="off" @submit.prevent="saveGeneral">
            <!-- Client side Timezone -->
            <div v-if="false" class="mb-4">
                <label for="timezone" class="form-label">
                    {{ $t("Display Timezone") }}
                </label>
                <select id="timezone" v-model="userTimezone" class="form-select">
                    <option value="auto">
                        {{ $t("Auto") }}: {{ guessTimezone }}
                    </option>
                    <option
                        v-for="(timezone, index) in timezoneList"
                        :key="index"
                        :value="timezone.value"
                    >
                        {{ timezone.name }}
                    </option>
                </select>
            </div>

            <!-- Server Timezone -->
            <div v-if="false" class="mb-4">
                <label for="timezone" class="form-label">
                    {{ $t("Server Timezone") }}
                </label>
                <select id="timezone" v-model="settings.serverTimezone" class="form-select">
                    <option value="UTC">UTC</option>
                    <option
                        v-for="(timezone, index) in timezoneList"
                        :key="index"
                        :value="timezone.value"
                    >
                        {{ timezone.name }}
                    </option>
                </select>
            </div>

            <!-- Primary Hostname -->
            <div class="mb-4">
                <label class="form-label" for="primaryBaseURL">
                    {{ $t("primaryHostname") }}
                </label>

                <div class="input-group mb-3">
                    <input
                        v-model="settings.primaryHostname"
                        class="form-control"
                        :placeholder="$t(`CurrentHostname`)"
                    />
                    <button class="btn btn-outline-primary" type="button" @click="autoGetPrimaryHostname">
                        {{ $t("autoGet") }}
                    </button>
                </div>

                <div class="form-text"></div>
            </div>

            <!-- Image Update Checks -->
            <div class="mb-4">
                <label class="form-label">
                    {{ $t("imageUpdateChecking") }}
                </label>
                <div class="form-check mb-2">
                    <input
                        id="imageUpdateCheckEnabled"
                        v-model="settings.imageUpdateCheckEnabled"
                        class="form-check-input"
                        type="checkbox"
                    />
                    <label class="form-check-label" for="imageUpdateCheckEnabled">
                        {{ $t("enableImageUpdateCheck") }}
                    </label>
                </div>
                <div v-if="settings.imageUpdateCheckEnabled" class="input-group" style="max-width: 300px;">
                    <input
                        v-model.number="settings.imageUpdateCheckInterval"
                        type="number"
                        class="form-control"
                        min="1"
                        max="168"
                    />
                    <span class="input-group-text">{{ $t("hours") }}</span>
                </div>
                <div v-if="settings.imageUpdateCheckEnabled" class="form-text">
                    {{ $t("imageUpdateCheckIntervalHelp") }}
                </div>
            </div>

            <!-- Save Button -->
            <div>
                <button class="btn btn-primary" type="submit">
                    {{ $t("Save") }}
                </button>
            </div>
        </form>
    </div>
</template>

<script setup lang="ts">
import { computed, inject, type Ref } from "vue";
import dayjs from "dayjs";
import { timezoneList as getTimezoneList } from "../../util-frontend";
import { useTheme } from "../../composables/useTheme";

const settings = inject<Ref<Record<string, any>>>("settings")!;
const saveSettings = inject<(callback?: () => void, currentPassword?: string) => void>("saveSettings")!;

const { userTimezone } = useTheme();

const timezoneList = getTimezoneList();
const guessTimezone = computed(() => dayjs.tz.guess());

function saveGeneral() {
    localStorage.timezone = userTimezone.value;
    saveSettings();
}

function autoGetPrimaryHostname() {
    settings.value.primaryHostname = location.hostname;
}
</script>
