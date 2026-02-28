import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { useContainerStore, type ContainerBroadcast } from "./containerStore";
import { useUpdateStore } from "./updateStore";
import {
    CREATED_FILE as STATUS_CREATED_FILE,
    CREATED_STACK as STATUS_CREATED_STACK,
    RUNNING as STATUS_RUNNING,
    EXITED as STATUS_EXITED,
    RUNNING_AND_EXITED as STATUS_RUNNING_AND_EXITED,
    UNHEALTHY as STATUS_UNHEALTHY,
} from "../common/util-common";

/** Matches the Go StackBroadcastEntry type. */
export interface StackBroadcastEntry {
    name: string;
    composeFileName: string;
    ignoreStatus?: Record<string, boolean>;
    images: Record<string, string>;
    isManagedByDockge: boolean;
}

export interface EnrichedStack {
    name: string;
    composeFileName: string;
    images: Record<string, string>;
    ignoreStatus?: Record<string, boolean>;
    isManagedByDockge: boolean;
    status: number;
    started: boolean;
    recreateNecessary: boolean;
    imageUpdatesAvailable: boolean;
    tags: string[];
    endpoint: string;
}

/** Derive stack status from container states. */
function deriveStatus(
    containers: ContainerBroadcast[],
    ignoreStatus?: Record<string, boolean>
): number {
    let running = 0;
    let exited = 0;
    let created = 0;
    let paused = 0;
    let unhealthy = 0;

    for (const c of containers) {
        // Skip ignored services
        if (ignoreStatus && ignoreStatus[c.serviceName]) {
            continue;
        }
        if (c.health === "unhealthy") {
            unhealthy++;
        } else {
            switch (c.state) {
                case "running":
                    running++;
                    break;
                case "exited":
                case "dead":
                    exited++;
                    break;
                case "created":
                    created++;
                    break;
                case "paused":
                    paused++;
                    break;
            }
        }
    }

    if (unhealthy > 0) return STATUS_UNHEALTHY;
    if (running > 0 && exited > 0) return STATUS_RUNNING_AND_EXITED;
    if (running > 0) return STATUS_RUNNING;
    if (exited > 0) return STATUS_EXITED;
    if (created > 0) return STATUS_CREATED_STACK;
    if (paused > 0) return STATUS_RUNNING; // paused counts as running for UI
    return STATUS_CREATED_FILE; // no containers = created (file only)
}

export const useStackStore = defineStore("stacks", () => {
    const rawStacks = ref<StackBroadcastEntry[]>([]);
    const loading = ref(true);

    function setStacks(data: StackBroadcastEntry[]) {
        rawStacks.value = data;
        loading.value = false;
    }

    /** Enriched managed stacks with status derived from container store. */
    const stacks = computed(() => {
        const containerStore = useContainerStore();
        const updateStore = useUpdateStore();

        return rawStacks.value.map((s): EnrichedStack => {
            const stackContainers = containerStore.byStack(s.name);
            const status = deriveStatus(stackContainers, s.ignoreStatus);
            const started = status === STATUS_RUNNING ||
                status === STATUS_RUNNING_AND_EXITED ||
                status === STATUS_UNHEALTHY;

            // Check recreate: compare running container images vs compose images
            let recreateNecessary = false;
            if (started) {
                for (const c of stackContainers) {
                    const composeImage = s.images[c.serviceName];
                    if (composeImage && c.image && c.image !== composeImage) {
                        recreateNecessary = true;
                        break;
                    }
                }
            }

            // Check updates: any service in this stack has an update
            let imageUpdatesAvailable = false;
            for (const c of stackContainers) {
                if (updateStore.hasUpdate(`${s.name}/${c.serviceName}`)) {
                    imageUpdatesAvailable = true;
                    break;
                }
            }

            return {
                name: s.name,
                composeFileName: s.composeFileName,
                images: s.images,
                ignoreStatus: s.ignoreStatus,
                isManagedByDockge: s.isManagedByDockge,
                status,
                started,
                recreateNecessary,
                imageUpdatesAvailable,
                tags: [],
                endpoint: "",
            };
        });
    });

    /** Unmanaged stacks: compose projects with containers but no compose file in stacks dir. */
    const unmanagedStacks = computed(() => {
        const containerStore = useContainerStore();
        const managedNames = new Set(rawStacks.value.map((s) => s.name));

        // Find all unique stack names from containers that aren't managed
        const seen = new Set<string>();
        const result: EnrichedStack[] = [];

        for (const c of containerStore.containers) {
            if (!c.stackName || c.stackName === "dockge" || managedNames.has(c.stackName) || seen.has(c.stackName)) {
                continue;
            }
            seen.add(c.stackName);

            const stackContainers = containerStore.byStack(c.stackName);
            const status = deriveStatus(stackContainers);
            const started = status === STATUS_RUNNING ||
                status === STATUS_RUNNING_AND_EXITED ||
                status === STATUS_UNHEALTHY;

            result.push({
                name: c.stackName,
                composeFileName: "",
                images: {},
                isManagedByDockge: false,
                status,
                started,
                recreateNecessary: false,
                imageUpdatesAvailable: false,
                tags: [],
                endpoint: "",
            });
        }
        return result;
    });

    /** All stacks: managed + unmanaged. */
    const allStacks = computed(() => [...stacks.value, ...unmanagedStacks.value]);

    return {
        rawStacks,
        loading,
        setStacks,
        stacks,
        unmanagedStacks,
        allStacks,
    };
});
