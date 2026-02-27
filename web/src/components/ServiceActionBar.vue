<template>
    <div class="btn-group me-2" role="group" aria-label="Service actions">
        <button v-if="!active" class="btn btn-primary" :disabled="processing" :title="$t('tooltipServiceStart', [serviceName])" @click="startService">
            <font-awesome-icon icon="play" class="me-1" />
            {{ $t("startStack") }}
        </button>

        <button v-if="active" class="btn btn-normal" :disabled="processing" :title="$t('tooltipServiceRestart', [serviceName])" @click="restartService">
            <font-awesome-icon icon="rotate" class="me-1" />
            {{ $t("restartStack") }}
        </button>

        <button class="btn" :class="recreateNecessary ? 'btn-info' : 'btn-normal'" :disabled="processing" :title="$t('tooltipServiceRecreate', [serviceName])" @click="recreateService">
            <font-awesome-icon icon="rocket" class="me-1" />
            <span class="d-none d-xl-inline">{{ $t("recreate") }}</span>
        </button>

        <button class="btn" :class="imageUpdatesAvailable ? 'btn-info' : 'btn-normal'" :disabled="processing" :title="$t('tooltipServiceUpdate', [serviceName])" @click="showDialog = true">
            <font-awesome-icon icon="cloud-arrow-down" class="me-1" />
            <span class="d-none d-xl-inline">{{ $t("updateStack") }}</span>
        </button>

        <UpdateDialog
            v-model="showDialog"
            :stack-name="stackName"
            :endpoint="endpoint"
            :service-name="serviceName"
            @update="doUpdate"
        />

        <button v-if="active" class="btn btn-normal" :disabled="processing" :title="$t('tooltipServiceStop', [serviceName])" @click="stopService">
            <font-awesome-icon icon="stop" class="me-1" />
            {{ $t("stopStack") }}
        </button>

        <BDropdown right text="" variant="normal" menu-class="overflow-dropdown">
            <BDropdownItem :title="$t('tooltipCheckUpdates')" @click="checkImageUpdates">
                <font-awesome-icon icon="search" class="me-1" />
                {{ $t("checkUpdates") }}
            </BDropdownItem>
        </BDropdown>
    </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import UpdateDialog from "./UpdateDialog.vue";

const props = defineProps<{
    active: boolean;
    processing: boolean;
    imageUpdatesAvailable: boolean;
    recreateNecessary: boolean;
    stackName: string;
    endpoint: string;
    serviceName: string;
}>();

const emit = defineEmits<{
    start: [];
    stop: [];
    restart: [];
    recreate: [];
    update: [data: { pruneAfterUpdate: boolean; pruneAllAfterUpdate: boolean }];
    checkUpdates: [];
}>();

const showDialog = ref(false);

function startService() { emit("start"); }
function stopService() { emit("stop"); }
function restartService() { emit("restart"); }
function recreateService() { emit("recreate"); }
function doUpdate(data: { pruneAfterUpdate: boolean; pruneAllAfterUpdate: boolean }) { emit("update", data); }
function checkImageUpdates() { emit("checkUpdates"); }
</script>
