import { defineStore } from "pinia";
import { ref } from "vue";

/** Matches the Go ContainerBroadcast type. */
export interface ContainerBroadcast {
    name: string;
    containerId: string;
    serviceName: string;
    stackName: string;
    state: string;
    health: string;
    image: string;
    imageId: string;
    networks: Record<string, { ipv4: string; ipv6: string; mac: string }>;
    mounts: { name: string; type: string }[];
    ports: { hostPort: number; containerPort: number; protocol: string }[];
}

export const useContainerStore = defineStore("containers", () => {
    const containers = ref<ContainerBroadcast[]>([]);
    const loading = ref(true);

    function setContainers(data: ContainerBroadcast[]) {
        containers.value = data;
        loading.value = false;
    }

    /** Containers belonging to a specific compose project (stack). */
    function byStack(stackName: string): ContainerBroadcast[] {
        return containers.value.filter((c) => c.stackName === stackName);
    }

    /** Containers connected to a specific network. */
    function byNetwork(networkName: string): ContainerBroadcast[] {
        return containers.value.filter((c) => networkName in (c.networks || {}));
    }

    /** Containers using a specific image (by image ID). */
    function byImage(imageId: string): ContainerBroadcast[] {
        return containers.value.filter((c) => c.imageId === imageId);
    }

    /** Containers using a specific volume (by mount name). */
    function byVolume(volumeName: string): ContainerBroadcast[] {
        return containers.value.filter((c) =>
            (c.mounts || []).some((m) => m.name === volumeName)
        );
    }

    return {
        containers,
        loading,
        setContainers,
        byStack,
        byNetwork,
        byImage,
        byVolume,
    };
});
