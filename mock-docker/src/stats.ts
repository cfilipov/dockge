import type { ContainerInspect } from "./types.js";
import type { Clock } from "./clock.js";
import { deterministicInt, serviceSeed, hashToSeed } from "./deterministic.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface ContainerStats {
    read: string;
    preread: string;
    pids_stats: { current: number; limit: number };
    blkio_stats: {
        io_service_bytes_recursive: Array<{ major: number; minor: number; op: string; value: number }>;
        io_serviced_recursive: Array<{ major: number; minor: number; op: string; value: number }>;
    };
    num_procs: number;
    storage_stats: Record<string, never>;
    cpu_stats: {
        cpu_usage: {
            total_usage: number;
            usage_in_kernelmode: number;
            usage_in_usermode: number;
            percpu_usage?: number[];
        };
        system_cpu_usage: number;
        online_cpus: number;
        throttling_data: { periods: number; throttled_periods: number; throttled_time: number };
    };
    precpu_stats: {
        cpu_usage: {
            total_usage: number;
            usage_in_kernelmode: number;
            usage_in_usermode: number;
            percpu_usage?: number[];
        };
        system_cpu_usage: number;
        online_cpus: number;
        throttling_data: { periods: number; throttled_periods: number; throttled_time: number };
    };
    memory_stats: {
        usage: number;
        stats: {
            active_anon: number;
            inactive_anon: number;
            active_file: number;
            inactive_file: number;
            cache: number;
            pgfault: number;
            pgmajfault: number;
        };
        max_usage: number;
        limit: number;
    };
    name: string;
    id: string;
    networks: {
        eth0: {
            rx_bytes: number;
            rx_packets: number;
            rx_errors: number;
            rx_dropped: number;
            tx_bytes: number;
            tx_packets: number;
            tx_errors: number;
            tx_dropped: number;
        };
    };
}

// ---------------------------------------------------------------------------
// Seed derivation
// ---------------------------------------------------------------------------

function containerSeed(container: ContainerInspect): string {
    const labels = container.Config.Labels ?? {};
    const project = labels["com.docker.compose.project"];
    const service = labels["com.docker.compose.service"];
    if (project && service) return serviceSeed(project, service);
    return hashToSeed(["portge-mock-v1", "container", container.Id]);
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

export function generateStats(container: ContainerInspect, counter: number, clock: Clock): ContainerStats {
    const seed = containerSeed(container);

    // Base values from deterministic seed
    const baseCpuPercent = deterministicInt(seed + "cpu", 1, 80);
    const baseMemMB = deterministicInt(seed + "mem", 20, 400);
    const baseNetRxRate = deterministicInt(seed + "netrx", 1000, 100000);  // bytes/tick
    const baseNetTxRate = deterministicInt(seed + "nettx", 500, 50000);
    const basePids = deterministicInt(seed + "pids", 2, 50);
    const baseBlkioRead = deterministicInt(seed + "blkrd", 1000, 500000);
    const baseBlkioWrite = deterministicInt(seed + "blkwr", 500, 200000);

    // Sine-wave variation
    const cpuVariation = Math.sin(counter * 0.1) * 5;
    const memVariation = Math.sin(counter * 0.07) * 20;

    const cpuPercent = Math.max(0, Math.min(100, baseCpuPercent + cpuVariation));
    const memUsageMB = Math.max(1, baseMemMB + memVariation);
    const memLimit = container.HostConfig.Memory || 1_073_741_824; // 1GB default
    const memUsageBytes = Math.floor(memUsageMB * 1024 * 1024);

    // CPU: total_usage grows with counter
    const onlineCpus = 4;
    const systemCpuUsage = counter * 1_000_000_000; // ns
    const totalUsage = Math.floor(cpuPercent / 100 * systemCpuUsage / onlineCpus);
    const prevCounter = Math.max(0, counter - 1);
    const prevTotalUsage = Math.floor((cpuPercent / 100) * (prevCounter * 1_000_000_000) / onlineCpus);
    const kernelUsage = Math.floor(totalUsage * 0.3);
    const userUsage = totalUsage - kernelUsage;

    const now = clock.now();
    const read = now.toISOString();
    const preread = new Date(now.getTime() - 1000).toISOString();

    return {
        read,
        preread,
        pids_stats: {
            current: basePids,
            limit: 4096,
        },
        blkio_stats: {
            io_service_bytes_recursive: [
                { major: 8, minor: 0, op: "read", value: baseBlkioRead * (counter + 1) },
                { major: 8, minor: 0, op: "write", value: baseBlkioWrite * (counter + 1) },
                { major: 8, minor: 0, op: "sync", value: baseBlkioWrite * (counter + 1) },
                { major: 8, minor: 0, op: "async", value: baseBlkioRead * (counter + 1) },
                { major: 8, minor: 0, op: "total", value: (baseBlkioRead + baseBlkioWrite) * (counter + 1) },
            ],
            io_serviced_recursive: [
                { major: 8, minor: 0, op: "read", value: Math.floor(baseBlkioRead / 100) * (counter + 1) },
                { major: 8, minor: 0, op: "write", value: Math.floor(baseBlkioWrite / 100) * (counter + 1) },
                { major: 8, minor: 0, op: "total", value: Math.floor((baseBlkioRead + baseBlkioWrite) / 100) * (counter + 1) },
            ],
        },
        num_procs: 0,
        storage_stats: {},
        cpu_stats: {
            cpu_usage: {
                total_usage: totalUsage,
                usage_in_kernelmode: kernelUsage,
                usage_in_usermode: userUsage,
            },
            system_cpu_usage: systemCpuUsage,
            online_cpus: onlineCpus,
            throttling_data: { periods: 0, throttled_periods: 0, throttled_time: 0 },
        },
        precpu_stats: {
            cpu_usage: {
                total_usage: prevTotalUsage,
                usage_in_kernelmode: Math.floor(prevTotalUsage * 0.3),
                usage_in_usermode: prevTotalUsage - Math.floor(prevTotalUsage * 0.3),
            },
            system_cpu_usage: prevCounter * 1_000_000_000,
            online_cpus: onlineCpus,
            throttling_data: { periods: 0, throttled_periods: 0, throttled_time: 0 },
        },
        memory_stats: {
            usage: memUsageBytes,
            stats: {
                active_anon: Math.floor(memUsageBytes * 0.6),
                inactive_anon: Math.floor(memUsageBytes * 0.1),
                active_file: Math.floor(memUsageBytes * 0.15),
                inactive_file: Math.floor(memUsageBytes * 0.1),
                cache: Math.floor(memUsageBytes * 0.25),
                pgfault: 1000 * (counter + 1),
                pgmajfault: counter,
            },
            max_usage: Math.floor(memUsageBytes * 1.2),
            limit: memLimit,
        },
        name: container.Name,
        id: container.Id,
        networks: {
            eth0: {
                rx_bytes: baseNetRxRate * (counter + 1),
                rx_packets: Math.floor(baseNetRxRate / 100) * (counter + 1),
                rx_errors: 0,
                rx_dropped: 0,
                tx_bytes: baseNetTxRate * (counter + 1),
                tx_packets: Math.floor(baseNetTxRate / 100) * (counter + 1),
                tx_errors: 0,
                tx_dropped: 0,
            },
        },
    };
}
