<template>
    <div class="input-group mb-3">
        <input
            ref="input"
            v-model="model"
            :type="visibility"
            class="form-control"
            :placeholder="placeholder"
            :maxlength="maxlength"
            :autocomplete="autocomplete"
            :required="required"
            :readonly="readonly"
        >

        <a v-if="visibility == 'password'" class="btn btn-outline-primary" @click="showInput()">
            <font-awesome-icon icon="eye" />
        </a>
        <a v-if="visibility == 'text'" class="btn btn-outline-primary" @click="hideInput()">
            <font-awesome-icon icon="eye-slash" />
        </a>
    </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";

const props = withDefaults(defineProps<{
    modelValue?: string;
    placeholder?: string;
    maxlength?: number;
    autocomplete?: string;
    required?: boolean;
    readonly?: string;
}>(), {
    modelValue: "",
    placeholder: "",
    maxlength: 255,
    autocomplete: "new-password",
    required: false,
    readonly: undefined,
});

const emit = defineEmits<{
    (e: "update:modelValue", value: string): void;
}>();

const visibility = ref("password");

const model = computed({
    get() {
        return props.modelValue;
    },
    set(value: string) {
        emit("update:modelValue", value);
    }
});

function showInput() {
    visibility.value = "text";
}

function hideInput() {
    visibility.value = "password";
}
</script>
