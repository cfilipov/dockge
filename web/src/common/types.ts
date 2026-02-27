export type StatsData = {
    cpuPerc: string,
    memUsage: string,
    memPerc: string,
    netIO: string,
    blockIO: string
}

export type ServiceData = {
    name: string,
    containerName: string,
    image: string,
    state: string,
    status: string,
    health: string,
    recreateNecessary: boolean,
    imageUpdateAvailable: boolean,
    remoteImageDigest: string,
}

export type SimpleStackData = {
    name: string,
    status: number,
    started: boolean,
    recreateNecessary: boolean,
    imageUpdatesAvailable: boolean,
    tags: string[],
    isManagedByDockge: boolean,
    composeFileName: string,
    endpoint: string
}

export type StackData = SimpleStackData & {
    composeYAML: string,
    composeENV: string,
    primaryHostname: string,
    services: Record<string, ServiceData>
}

export type AgentData = {
    url: string,
    username: string,
    password: string,
    endpoint: string,
    name: string
}

export enum DockerArtefactAction {
    Prune = "prune",
    PruneAll = "pruneAll",
    Remove = "remove",
    Pull = "pull"
}

export type DockerArtefactInfo = {
    name: string,
    actions: DockerArtefactAction[]
}

export const DockerArtefactInfos: Record<string, DockerArtefactInfo> = {
    Container: {
        name: "container",
        actions: [ DockerArtefactAction.Prune, DockerArtefactAction.Remove ]
    },
    Image: {
        name: "image",
        actions: [ DockerArtefactAction.Prune, DockerArtefactAction.PruneAll, DockerArtefactAction.Pull, DockerArtefactAction.Remove ]
    },
    Network: {
        name: "network",
        actions: [ DockerArtefactAction.Prune, DockerArtefactAction.Remove ]
    },
    Volume: {
        name: "volume",
        actions: [ DockerArtefactAction.Prune, DockerArtefactAction.PruneAll, DockerArtefactAction.Remove ]
    }
};

export type DockerArtefactItem = {
    id: string,
    actionIds: Record<string, string>,
    values: Record<string, string | [string, string] | [string, number]>,
    dangling: boolean,
    danglingLabel: string,
    excludedActions: DockerArtefactAction[]
}

export type DockerArtefactData = {
    info: DockerArtefactInfo,
    data: DockerArtefactItem[]
}

// --- WebSocket response types ---

export type ApiResponse<T = undefined> = {
    ok: boolean,
    msg?: string,
} & (T extends undefined ? {} : T)

export type StackListResponse = ApiResponse<{
    stackList: Record<string, SimpleStackData>,
    endpoint?: string,
}>

export type ServiceStatusResponse = ApiResponse<{
    serviceStatusList: Record<string, string>,
}>

export type DockerStatsResponse = ApiResponse<{
    dockerStats: Record<string, StatsData>,
}>

export type ContainerListResponse = ApiResponse<{
    containerList: Record<string, unknown>[],
}>

export type TokenResponse = ApiResponse<{
    token: string,
    tokenRequired?: boolean,
}>

export type InfoData = {
    version: string,
    latestVersion?: string,
    primaryHostname?: string,
    serverTimezone?: string,
    serverTimezoneOffset?: string,
}

// --- Turnstile global ---

export interface TurnstileInstance {
    render(container: HTMLElement, options: {
        sitekey: string,
        callback: (token: string) => void,
        "error-callback"?: () => void,
    }): void;
    reset(container: HTMLElement): void;
}

declare global {
    interface Window {
        turnstile?: TurnstileInstance;
    }
}
