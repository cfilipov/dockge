<template>
    <div ref="modalEl" class="modal fade" tabindex="-1">
        <div class="modal-dialog">
            <div class="modal-content">
                <div class="modal-header">
                    <h5 id="exampleModalLabel" class="modal-title">
                        {{ title || $t("Confirm") }}
                    </h5>
                    <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close" />
                </div>
                <div class="modal-body">
                    <slot />
                </div>
                <div class="modal-footer">
                    <button type="button" class="btn" :class="btnStyle" data-bs-dismiss="modal" @click="emit('yes')">
                        {{ yesText }}
                    </button>
                    <button type="button" class="btn btn-secondary" data-bs-dismiss="modal" @click="emit('no')">
                        {{ noText }}
                    </button>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { Modal } from "bootstrap";

withDefaults(defineProps<{
    btnStyle?: string;
    yesText?: string;
    noText?: string;
    title?: string | null;
}>(), {
    btnStyle: "btn-primary",
    yesText: "Yes",
    noText: "No",
    title: null,
});

const emit = defineEmits<{
    (e: "yes"): void;
    (e: "no"): void;
}>();

const modalEl = ref<HTMLElement>();
let modal: Modal | null = null;

onMounted(() => {
    modal = new Modal(modalEl.value!);
});

function show() {
    modal?.show();
}

defineExpose({ show });
</script>
