<template>
    <div>
        <div v-if="valid">
            <ul v-if="isArrayInited" class="list-group">
                <li v-for="(value, index) in array" :key="index" class="list-group-item">
                    <select v-model="array[index]" class="no-bg domain-input">
                        <option value="">{{ $t(`Select a network...`) }}</option>
                        <option v-for="option in options" :key="option" :value="option">{{ option }}</option>
                    </select>

                    <font-awesome-icon icon="times" class="action remove ms-2 me-3 text-danger" @click="remove(index)" />
                </li>
            </ul>

            <button class="btn btn-normal btn-sm mt-3" @click="addField">{{ $t("addListItem", [ displayName ]) }}</button>
        </div>
        <div v-else>
            Long syntax is not supported here. Please use the YAML editor.
        </div>
    </div>
</template>

<script setup lang="ts">
import { computed, inject, type ComputedRef } from "vue";

const props = defineProps<{
    name: string;
    placeholder?: string;
    displayName: string;
    options: string[];
}>();

const injectedService = inject<ComputedRef<Record<string, any>>>("service")!;

const service = computed(() => {
    return injectedService.value;
});

const array = computed(() => {
    if (!service.value[props.name]) {
        return [];
    }
    return service.value[props.name];
});

const isArrayInited = computed(() => {
    return service.value[props.name] !== undefined;
});

const valid = computed(() => {
    if (!Array.isArray(array.value)) {
        return false;
    }
    for (let item of array.value) {
        if (typeof item === "object") {
            return false;
        }
    }
    return true;
});

function addField() {
    if (!service.value[props.name]) {
        service.value[props.name] = [];
    }
    array.value.push("");
}

function remove(index: number) {
    array.value.splice(index, 1);
}
</script>

<style lang="scss" scoped>
@import "../styles/vars.scss";

.list-group {
    background-color: $dark-bg2;

    li {
        display: flex;
        align-items: center;
        padding: 10px 0 10px 10px;

        .domain-input {
            flex-grow: 1;
            background-color: $dark-bg2;
            border: none;
            color: $dark-font-color;
            outline: none;

            &::placeholder {
                color: #1d2634;
            }
        }
    }
}
</style>
