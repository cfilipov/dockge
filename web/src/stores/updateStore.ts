import { defineStore } from "pinia";
import { ref } from "vue";

export const useUpdateStore = defineStore("updates", () => {
    /** Array of "stackName/serviceName" keys that have image updates available. */
    const updatedServices = ref<string[]>([]);
    const loading = ref(true);

    function setUpdates(data: string[]) {
        updatedServices.value = data;
        loading.value = false;
    }

    /** Check if a specific service has an update available. */
    function hasUpdate(key: string): boolean {
        return updatedServices.value.includes(key);
    }

    return {
        updatedServices,
        loading,
        setUpdates,
        hasUpdate,
    };
});
