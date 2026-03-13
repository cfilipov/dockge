// Docker Engine API v1.47 response types.
// Every field from the API schema is included; fields we don't populate yet are optional.

export interface PortBinding {
    HostIp: string;
    HostPort: string;
}

export interface RestartPolicy {
    Name: string;
    MaximumRetryCount: number;
}

export interface LogConfig {
    Type: string;
    Config: Record<string, string>;
}

export interface DeviceMapping {
    PathOnHost: string;
    PathInContainer: string;
    CgroupPermissions: string;
}

export interface DeviceRequest {
    Driver?: string;
    Count?: number;
    DeviceIDs?: string[];
    Capabilities?: string[][];
    Options?: Record<string, string>;
}

export interface Ulimit {
    Name: string;
    Soft: number;
    Hard: number;
}

export interface ThrottleDevice {
    Path: string;
    Rate: number;
}

export interface WeightDevice {
    Path: string;
    Weight: number;
}

export interface HostConfig {
    Binds?: string[];
    ContainerIDFile?: string;
    LogConfig?: LogConfig;
    NetworkMode?: string;
    PortBindings?: Record<string, PortBinding[]>;
    RestartPolicy?: RestartPolicy;
    AutoRemove?: boolean;
    VolumeDriver?: string;
    VolumesFrom?: string[];
    ConsoleSize?: [number, number];
    Annotations?: Record<string, string>;
    CapAdd?: string[];
    CapDrop?: string[];
    CgroupnsMode?: string;
    Dns?: string[];
    DnsOptions?: string[];
    DnsSearch?: string[];
    ExtraHosts?: string[];
    GroupAdd?: string[];
    IpcMode?: string;
    Cgroup?: string;
    Links?: string[];
    OomScoreAdj?: number;
    PidMode?: string;
    Privileged?: boolean;
    PublishAllPorts?: boolean;
    ReadonlyRootfs?: boolean;
    SecurityOpt?: string[];
    StorageOpt?: Record<string, string>;
    Tmpfs?: Record<string, string>;
    UTSMode?: string;
    UsernsMode?: string;
    ShmSize?: number;
    Sysctls?: Record<string, string>;
    Runtime?: string;
    Isolation?: string;
    CpuShares?: number;
    Memory?: number;
    NanoCpus?: number;
    CgroupParent?: string;
    BlkioWeight?: number;
    BlkioWeightDevice?: WeightDevice[];
    BlkioDeviceReadBps?: ThrottleDevice[];
    BlkioDeviceWriteBps?: ThrottleDevice[];
    BlkioDeviceReadIOps?: ThrottleDevice[];
    BlkioDeviceWriteIOps?: ThrottleDevice[];
    CpuPeriod?: number;
    CpuQuota?: number;
    CpuRealtimePeriod?: number;
    CpuRealtimeRuntime?: number;
    CpusetCpus?: string;
    CpusetMems?: string;
    Devices?: DeviceMapping[];
    DeviceCgroupRules?: string[];
    DeviceRequests?: DeviceRequest[];
    KernelMemoryTCP?: number;
    MemoryReservation?: number;
    MemorySwap?: number;
    MemorySwappiness?: number;
    OomKillDisable?: boolean;
    PidsLimit?: number;
    Ulimits?: Ulimit[];
    CpuCount?: number;
    CpuPercent?: number;
    IOMaximumIOps?: number;
    IOMaximumBandwidth?: number;
    MaskedPaths?: string[];
    ReadonlyPaths?: string[];
    Init?: boolean;
}

export interface HealthcheckConfig {
    Test?: string[];
    Interval?: number;
    Timeout?: number;
    Retries?: number;
    StartPeriod?: number;
    StartInterval?: number;
}

export interface ContainerConfig {
    Hostname?: string;
    Domainname?: string;
    User?: string;
    AttachStdin?: boolean;
    AttachStdout?: boolean;
    AttachStderr?: boolean;
    ExposedPorts?: Record<string, Record<string, never>>;
    Tty?: boolean;
    OpenStdin?: boolean;
    StdinOnce?: boolean;
    Env?: string[];
    Cmd?: string[];
    Healthcheck?: HealthcheckConfig;
    ArgsEscaped?: boolean;
    Image?: string;
    Volumes?: Record<string, Record<string, never>>;
    WorkingDir?: string;
    Entrypoint?: string[];
    NetworkDisabled?: boolean;
    MacAddress?: string;
    OnBuild?: string[];
    Labels?: Record<string, string>;
    StopSignal?: string;
    StopTimeout?: number;
    Shell?: string[];
}

export interface HealthLogEntry {
    Start: string;
    End: string;
    ExitCode: number;
    Output: string;
}

export interface HealthState {
    Status: string;
    FailingStreak: number;
    Log: HealthLogEntry[];
}

export interface ContainerState {
    Status: string;
    Running: boolean;
    Paused: boolean;
    Restarting: boolean;
    OOMKilled: boolean;
    Dead: boolean;
    Pid: number;
    ExitCode: number;
    Error: string;
    StartedAt: string;
    FinishedAt: string;
    Health?: HealthState;
}

export interface MountPoint {
    Type: string;
    Name?: string;
    Source: string;
    Destination: string;
    Driver?: string;
    Mode: string;
    RW: boolean;
    Propagation?: string;
}

export interface GraphDriverData {
    Name: string;
    Data: Record<string, string>;
}

export interface IPAMConfig {
    IPv4Address?: string;
    IPv6Address?: string;
    LinkLocalIPs?: string[];
}

export interface EndpointSettings {
    IPAMConfig?: IPAMConfig;
    Links?: string[];
    Aliases?: string[];
    MacAddress?: string;
    DriverOpts?: Record<string, string>;
    NetworkID: string;
    EndpointID: string;
    Gateway: string;
    IPAddress: string;
    IPPrefixLen: number;
    IPv6Gateway?: string;
    GlobalIPv6Address?: string;
    GlobalIPv6PrefixLen?: number;
    DNSNames?: string[];
}

export interface Address {
    Addr: string;
    PrefixLen: number;
}

export interface NetworkSettings {
    Bridge?: string;
    SandboxID?: string;
    HairpinMode?: boolean;
    LinkLocalIPv6Address?: string;
    LinkLocalIPv6PrefixLen?: number;
    Ports?: Record<string, PortBinding[] | null>;
    SandboxKey?: string;
    SecondaryIPAddresses?: Address[];
    SecondaryIPv6Addresses?: Address[];
    EndpointID?: string;
    Gateway?: string;
    GlobalIPv6Address?: string;
    GlobalIPv6PrefixLen?: number;
    IPAddress?: string;
    IPPrefixLen?: number;
    IPv6Gateway?: string;
    MacAddress?: string;
    Networks?: Record<string, EndpointSettings>;
}

// --- OCI / Image Manifest types ---

export interface OCIPlatform {
    architecture: string;
    os: string;
    "os.version"?: string;
    "os.features"?: string[];
    variant?: string;
}

export interface OCIDescriptor {
    mediaType: string;
    digest: string;
    size: number;
    urls?: string[];
    annotations?: Record<string, string>;
    data?: string;       // base64-encoded
    platform?: OCIPlatform;
    artifactType?: string;
}

export type ImageManifestKind = "image" | "attestation" | "unknown";

export interface ImageManifestImageData {
    Platform: OCIPlatform;
    Containers: string[];
    Size: {
        Unpacked: number;
    };
}

export interface ImageManifestAttestationData {
    For: string;  // digest of the image manifest this attests
}

export interface ImageManifestSummary {
    ID: string;
    Descriptor: OCIDescriptor;
    Available: boolean;
    Size: {
        Total: number;
        Content: number;
    };
    Kind: ImageManifestKind;
    ImageData?: ImageManifestImageData;
    AttestationData?: ImageManifestAttestationData;
}

export interface ContainerInspect {
    Id: string;
    Created: string;
    Path: string;
    Args: string[];
    State: ContainerState;
    Image: string;
    ResolvConfPath?: string;
    HostnamePath?: string;
    HostsPath?: string;
    LogPath?: string;
    Name: string;
    RestartCount?: number;
    Driver?: string;
    Platform?: string;
    MountLabel?: string;
    ProcessLabel?: string;
    AppArmorProfile?: string;
    ExecIDs?: string[] | null;
    HostConfig: HostConfig;
    GraphDriver?: GraphDriverData;
    SizeRw?: number;
    SizeRootFs?: number;
    Mounts: MountPoint[];
    Config: ContainerConfig;
    NetworkSettings: NetworkSettings;
    ImageManifestDescriptor?: OCIDescriptor;
}

// --- Network types ---

export interface IPAMPoolConfig {
    Subnet?: string;
    IPRange?: string;
    Gateway?: string;
    AuxiliaryAddresses?: Record<string, string>;
}

export interface IPAM {
    Driver?: string;
    Config?: IPAMPoolConfig[];
    Options?: Record<string, string>;
}

export interface NetworkContainer {
    Name: string;
    EndpointID: string;
    MacAddress: string;
    IPv4Address: string;
    IPv6Address?: string;
}

export interface NetworkPeer {
    Name: string;
    IP: string;
}

export interface NetworkInspect {
    Name: string;
    Id: string;
    Created: string;
    Scope: string;
    Driver: string;
    EnableIPv4: boolean;
    EnableIPv6: boolean;
    IPAM: IPAM;
    Internal: boolean;
    Attachable: boolean;
    Ingress: boolean;
    ConfigFrom?: { Network: string };
    ConfigOnly?: boolean;
    Containers?: Record<string, NetworkContainer>;
    Options?: Record<string, string>;
    Labels?: Record<string, string>;
    Peers?: NetworkPeer[];
}

// --- Volume types ---

export interface VolumeUsageData {
    Size: number;
    RefCount: number;
}

export interface VolumeInspect {
    Name: string;
    Driver: string;
    Mountpoint: string;
    CreatedAt: string;
    Status?: Record<string, string>;
    Labels?: Record<string, string>;
    Scope: string;
    Options?: Record<string, string>;
    UsageData?: VolumeUsageData;
}

// --- Image types ---

export interface RootFS {
    Type: string;
    Layers?: string[];
}

export interface ImageMetadata {
    LastTagTime?: string;
}

/** @deprecated Use OCIDescriptor directly */
export type ImageDescriptor = OCIDescriptor;

export type ImageConfig = ContainerConfig;

export interface ImageInspect {
    Id: string;
    RepoTags: string[];
    RepoDigests: string[];
    Parent?: string;
    Comment?: string;
    Created: string;
    DockerVersion?: string;
    Author?: string;
    Config?: ContainerConfig;
    Architecture: string;
    Variant?: string;
    Os: string;
    OsVersion?: string;
    Size: number;
    GraphDriver?: GraphDriverData;
    RootFS: RootFS;
    Metadata?: ImageMetadata;
    Manifests?: ImageManifestSummary[];
    Descriptor?: OCIDescriptor;
}

// --- Exec types ---

export interface ExecProcessConfig {
    tty?: boolean;
    entrypoint?: string;
    arguments?: string[];
    privileged?: boolean;
    user?: string;
}

export interface ExecInspect {
    ID: string;
    Running: boolean;
    ExitCode: number;
    ProcessConfig?: ExecProcessConfig;
    OpenStdin: boolean;
    OpenStdout: boolean;
    OpenStderr: boolean;
    ContainerID: string;
    Pid: number;
    CanRemove?: boolean;
}
