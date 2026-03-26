# Mock Docker Daemon — Technical Specification

## 1. Overview

### 1.1 Purpose

A mock Docker daemon and mock Docker CLI implemented in TypeScript that replaces the real Docker engine for Portge's e2e testing. The mock:

- Listens on a Unix socket, implementing the Docker Engine API
- Maintains in-memory state representing containers, networks, volumes, and images
- Generates state deterministically from real Docker Compose files checked into the repo as test fixtures
- Fires Docker events on state mutations, matching real Docker's event sequences
- Provides a compiled Node.js binary (`docker`) that the Go backend shells out to for lifecycle operations (`docker compose up`, `docker compose stop`, etc.)
- Supports a reset API to reinitialize all state from disk between test runs
- Handles copying stacks from the source directory into the runtime directory on startup and on reset

The Go backend must not require any code changes or special-case logic to talk to the mock (other than the `/_mock/reset` endpoint). It calls the same SDK methods against the socket and shells out to the same `docker` binary name. The mock is transparent.

### 1.2 Architecture

```
Svelte 5 Frontend
    ↕ WebSocket
Go Backend
    ├── Docker Go SDK → Unix Socket → Mock Daemon (TypeScript HTTP server)
    ├── Shell out → Mock CLI (compiled TypeScript binary)
    └── fsnotify → Stacks directory on filesystem
```

All Docker communication flows through the Go backend. The frontend never talks to Docker directly. The mock only needs to fool the Go backend.

### 1.3 Key Design Principles

- **Inspect objects are the source of truth.** `ContainerInspect`, `NetworkInspect`, `VolumeInspect`, and `ImageInspect` objects are stored in memory as maps. List endpoint responses are derived from these via projection functions.
- **Single-threaded, no race conditions.** Node.js's event loop guarantees mutation ordering. No concurrent state updates, no locks, no diffing.
- **Fully deterministic. Do NOT use `Math.random()` or any non-deterministic source anywhere in the codebase.** All generated data (IDs, MAC addresses, IP addresses, timestamps, stats, logs, process lists) is derived from stable seeds (stack name + service name). If a value is not explicitly specified in this spec, it must still be derived deterministically from the nearest relevant seed (project + service, network name, image reference, etc.). E2e tests rely on consistent state across runs. Even seemingly unimportant fields like overlay2 paths, sandbox IDs, and endpoint IDs must be deterministic.
- **State consistency across resources.** Mutations update all affected resources atomically. Stopping a container updates the container's state AND removes it from its networks' `Containers` maps. Events are emitted inline after the state update.
- **Compose-driven.** The initial state is generated from real Docker Compose files. Every field in the compose file that has a corresponding field in the inspect response is mapped. The full Docker Compose v2 specification must be supported. Reference: https://docs.docker.com/reference/compose-file/
- **The daemon has no concept of compose projects.** Just like the real Docker daemon. The daemon manages containers, networks, volumes, and images. Compose project awareness lives entirely in the mock CLI and the compose labels on containers. The daemon does what it's told by API callers — it does not track which project "owns" a resource.
- **Events are owned by mutation functions.** Each mutation function in the daemon is the single place where both state updates and event emission happen. API handlers call mutation functions and never emit events directly. The mock CLI never emits events — it makes API calls to the daemon, which triggers mutations, which emit events.

### 1.4 Existing Go Implementation

There is an existing mock implementation (daemon + CLI) written in Go in this project. **Do NOT copy its patterns or architecture.** It has race conditions and overly complex state management. You may briefly scan it to check for any Docker API behaviors it mocks that this spec doesn't cover, but do not use it as a reference for code structure or style.

### 1.5 When This Spec Doesn't Cover Something

When encountering a situation not covered by this spec:

- Prefer the simplest correct behavior.
- Return sensible defaults rather than errors.
- If a Docker API field isn't specified here, use the zero value for its type (empty string, `0`, `false`, `null`, empty array/object) — but generate it deterministically if it could vary per resource.
- If a mutation's side effects aren't fully specified, consult the cross-resource reference table (§19) and update both sides of any reference.
- When in doubt about Docker's real behavior, check the Docker Engine API documentation: https://docs.docker.com/reference/api/engine/version/v1.53/
- Never introduce non-determinism. If you need a value and don't know how to derive it, hash something stable.

---

## 2. State Architecture

### 2.1 Core State Object

```typescript
interface MockState {
  containers: Map<string, ContainerInspect>;  // keyed by container ID
  networks:   Map<string, NetworkInspect>;    // keyed by network ID
  volumes:    Map<string, VolumeInspect>;     // keyed by volume name
  images:     Map<string, ImageInspect>;      // keyed by image ID
  execSessions: Map<string, ExecInspect>;     // keyed by exec ID
}
```

### 2.2 List Responses Are Derived

List endpoints do not store separate data. They project from the inspect maps:

- `GET /containers/json` → iterate `containers` map, apply `projectToContainerListEntry()` for each, apply filters
- `GET /networks` → iterate `networks` map, drop `ConfigFrom`, `ConfigOnly`, `Labels`, `Peers`, `Services`, `Status` fields, apply filters
- `GET /volumes` → wrap `volumes` map values in `{ Volumes: [...], Warnings: [] }`, apply filters
- `GET /images/json` → iterate `images` map, apply `projectToImageListEntry()` for each, apply filters

Filtering is critical — the Go backend relies heavily on label filters, particularly `com.docker.compose.project={project}` to find containers belonging to a stack. See §18 for the full list of supported filters.

### 2.3 Projection Functions

**Container list projection (`projectToContainerListEntry`):**

| List field | Source in ContainerInspect | Transformation |
|---|---|---|
| `Id` | `Id` | Direct |
| `Names` | `Name` | Wrap in array |
| `Image` | `Config.Image` | Direct (human-readable name) |
| `ImageID` | `Image` | Direct (sha256) |
| `ImageManifestDescriptor` | `ImageManifestDescriptor` | Direct |
| `Command` | `Path` + `Args` | Concatenate: `Path + " " + Args.join(" ")` |
| `Created` | `Created` | Convert ISO 8601 to unix epoch string |
| `Ports` | `NetworkSettings.Ports` | Flatten map to array of `{PrivatePort, PublicPort, Type}` |
| `SizeRw` | `SizeRw` | Direct |
| `SizeRootFs` | `SizeRootFs` | Direct |
| `Labels` | `Config.Labels` | Direct |
| `State` | `State.Status` | Direct |
| `Status` | `State.*` | Computed human-readable string (see §2.4) |
| `HostConfig.NetworkMode` | `HostConfig.NetworkMode` | Direct |
| `HostConfig.Annotations` | `HostConfig.Annotations` | Direct |
| `NetworkSettings.Networks` | `NetworkSettings.Networks` | Direct |
| `Mounts` | `Mounts` | Direct |
| `Health` | `State.Health` | Subset: only `Status` + `FailingStreak` |

**Image list projection (`projectToImageListEntry`):**

| List field | Source in ImageInspect | Transformation |
|---|---|---|
| `Id` | `Id` | Direct |
| `RepoTags` | `RepoTags` | Direct |
| `RepoDigests` | `RepoDigests` | Direct |
| `Created` | `Created` | Convert ISO 8601 to unix epoch string |
| `Size` | Sum of manifest layer sizes | Computed |
| `SharedSize` | Cross-image layer analysis | Computed or hardcoded |
| `Labels` | `Config.Labels` | Direct |
| `Containers` | Count containers referencing this image | Computed from containers map |
| `Manifests` | `Manifests` | Direct |
| `Descriptor` | `Descriptor` | Direct |
| `ParentId` | N/A | Empty string |

### 2.4 Container Status String

The `Status` field in the container list response is a human-readable string computed from inspect state:

- Running + healthy: `"Up X hours (healthy)"`
- Running + unhealthy: `"Up X hours (unhealthy)"`
- Running + no healthcheck: `"Up X hours"`
- Exited: `"Exited (CODE) X hours ago"`
- Paused: `"Up X hours (Paused)"`
- Created (not started): `"Created"`

For the mock, relative time can be simplified. Since timestamps are deterministic and tests don't assert on time strings, a basic formatter is sufficient (e.g., always say "Up 2 hours" or "Exited (0) 5 minutes ago").

---

## 3. Initialization

### 3.1 Daemon Arguments

The mock daemon takes these required arguments:

- `--stacks-dir /path/to/.run/{instance}/stacks` — the runtime stacks directory where compose files are read and written. This is the directory the Go backend also watches via fsnotify.
- `--stacks-source /path/to/test-data/stacks` — the checked-in source directory of test fixture stacks. Used for initial copy and reset.
- `--socket /path/to/.run/{instance}/docker.sock` — Unix socket path to listen on.

Multiple instances of the mock can run simultaneously (dev server, e2e tests, second dev server), each with their own `.run/` directory, socket, and stacks.

### 3.2 Init Sequence

Init is a single code path used both at startup and as part of reset.

1. **Copy source stacks into runtime stacks dir** using a recursive copy from `--stacks-source` to `--stacks-dir`. (On reset, the runtime dir contents are cleared first — see §3.7.)

2. **Read global mock config** at `{stacks-dir}/.mock.yaml` — create any global networks, volumes, and other shared resources not owned by any stack.

3. **Read pre-captured image data** from `images.json` (Layer 1 — real image data fetched from registries). Location TBD (likely bundled with the mock or in the stacks source dir).

4. **Single pass over all compose files:**
   a. Scan stacks dir for all subdirectories. Not every subdirectory will have a compose file — some are projects that were crawled but don't have Docker support. Skip subdirectories that have no compose file (`compose.yaml`, `compose.yml`, `docker-compose.yml`, `docker-compose.yaml`). These directories still exist on disk and the Go backend sees them via fsnotify — the mock just doesn't generate any Docker state for them.
   b. For each stack with a compose file, read its `.mock.yaml` sidecar if present.
   c. If the sidecar specifies `deployed: false`, skip this stack entirely — do not create any Docker resources. The compose file exists on disk (the Go backend sees it via fsnotify) but the daemon has no record of it.
   d. Parse the compose YAML (must support the full Docker Compose v2 spec, including both short and long syntax for `ports`, `volumes`, `networks`, `healthcheck`, etc.). Reference: https://docs.docker.com/reference/compose-file/
   e. For each top-level `networks:` entry: resolve the network name (see §4.3.1). If it doesn't exist in the networks map, create a `NetworkInspect`. If it already exists (created as a stub by a previous stack's `external: true` reference or by the global `.mock.yaml`), merge/update the details.
   f. For each top-level `volumes:` entry: same approach — create or update.
   g. For each `services:` entry: generate a `ContainerInspect` (see §4), add it to the containers map. Apply state overrides from `.mock.yaml`.
   h. For each referenced image not in the images map: check pre-captured data (Layer 1), else generate a synthetic `ImageInspect` (Layer 3).

5. **Post-process:** update `NetworkInspect.Containers` maps to reflect which containers are attached to which networks.

6. **Delete runtime directories for stacks marked `untracked: true`** in their `.mock.yaml`. This simulates compose stacks that exist in Docker but are not managed by Portge (no corresponding directory in the stacks dir). The Docker state was already generated in step 4, so the containers/networks/volumes exist in the API. But with the directory deleted, the Go backend won't see them via fsnotify — exactly matching how unmanaged external stacks look in production.

7. **Start HTTP server** on the Unix socket (only on initial startup, not on reset).

### 3.3 Global Mock Config

Located at `{stacks-dir}/.mock.yaml`. Defines resources that exist outside of any compose stack (e.g., networks or volumes created manually or by tools outside compose).

```yaml
networks:
  proxy-net:
    driver: bridge
    subnet: 172.18.0.0/16
  internal-services:
    driver: bridge
    internal: true

volumes:
  shared-media:
    driver: local
  nfs-backups:
    driver: local
```

These resources are created first during initialization, before any stacks are processed. If a stack later references one of these via `external: true`, it resolves correctly because the resource already exists in the map.

### 3.4 Per-Stack Mock Sidecar

Located at `{stacks-dir}/{stack}/.mock.yaml`. Defines stack deployment status and per-service state overrides. Absence of this file means all defaults: deployed, tracked, all services running and healthy, no update available, no recreation necessary.

```yaml
# Stack-level settings:
deployed: false    # If false, no Docker resources are created for this stack.
                   # The compose file still exists on disk. Default: true (deployed).

untracked: true    # If true, the stack's runtime directory is deleted after state
                   # generation. Docker resources exist in the API but Portge has no
                   # corresponding directory. Simulates external/unmanaged stacks.
                   # Default: false.

# Service-level overrides (only relevant when deployed: true):
services:
  redis:
    state: exited           # running | exited | paused | created (default: running)
    exit_code: 0            # only relevant if state: exited (default: 0)
    health: healthy         # healthy | unhealthy | starting | none (default: healthy if healthcheck defined, else none)
    update_available: true  # triggers digest mismatch in distribution inspect (default: false)
    needs_recreation: true  # triggers image ref mismatch between compose and container (default: false)
  worker:
    state: paused
```

A stack that has been `docker compose down`'d has no Docker resources at all — no containers, no networks, no volumes in the daemon state. This is the same end result as `deployed: false`. The difference is `deployed: false` is the initial state from the sidecar, while `docker compose down` is a runtime mutation that removes resources via API calls. Both result in the same daemon state: nothing for that project.

### 3.4 Image Data Layers

**Layer 1 — Pre-captured real data.** A JSON file (`images.json`) containing real `ImageInspect` responses fetched from container registries using the OCI Distribution API. These provide real `Config` objects (Env, Cmd, Entrypoint, ExposedPorts, Volumes, Labels, Healthcheck, WorkingDir, User, Shell), real `RootFS.Layers` digests, `Architecture`, `Os`, and `History`. Generated by a one-time capture script that:

1. Parses all compose files, collects unique image references.
2. For each image, resolves the registry (Docker Hub, ghcr.io, etc.).
3. Fetches an auth token.
4. Fetches the manifest (handles fat manifests — picks linux/amd64).
5. Fetches the config blob (small JSON, no layer downloads).
6. Combines into an `ImageInspect` shape and writes to `images.json`.

Fields NOT available from the registry that must be synthesized: `GraphDriver` (generate deterministic overlay2 paths), `Metadata.LastTagTime` (fixed timestamp), `Size` (approximate from manifest layer sizes).

**Layer 2 — Compose-derived data.** The generator merges compose service config on top of the image config. Compose overrides for `environment`, `command`, `entrypoint`, `user`, `working_dir`, `healthcheck`, `labels`, `exposed ports`, `volumes`, `stop_signal`, `stop_grace_period` take precedence over image defaults.

**Layer 3 — Synthetic fallback.** For images not in the pre-captured set, generate a minimal but valid `ImageInspect` with a deterministic ID (hash of image reference), a single fake layer, and sensible defaults (empty env, `/bin/sh` entrypoint, no healthcheck). The image won't look realistic but ensures the mock never crashes on an unknown image.

### 3.6 Build Directive Handling

When a compose service has `build:` (with or without `image:`):

- If `image:` is also specified, use that as the image name.
- If only `build:` exists, the image name becomes `{project}-{service}`.
- A Dockerfile in the stack dir may exist — the mock ignores its contents but the file is part of the test fixture.
- The mock generates an `ImageInspect` for the built image with `RepoTags` reflecting the local name (no registry prefix) and no `RepoDigests` (since it was never pushed).
- The container's `Config.Image` references this local image name.

### 3.7 Reset

`POST /_mock/reset` — the only non-standard Docker API endpoint.

Reset is simply: clear state, then run init.

1. Close all active event stream connections, log stream connections, and stats stream connections.
2. Clear all in-memory maps (containers, networks, volumes, images, exec sessions).
3. **Clear the contents of the runtime stacks directory, but NOT the directory itself.** Delete all files and subdirectories inside `{stacks-dir}/*` but keep `{stacks-dir}/` as an empty directory. This is critical because the Go backend uses fsnotify to watch this directory — deleting the directory itself would break the watcher.
4. Run the init sequence (§3.2). This copies fresh stacks from the source directory, reads configs, generates state, and deletes untracked dirs.

The test runner's sequence for each test:
1. Call `POST /_mock/reset`
2. Wait for reset to complete (response returns after init is done)
3. Run test
4. Repeat

---

## 4. Compose-to-Inspect Mapping

### 4.1 Container Generation

For each service in a compose file, the generator produces a `ContainerInspect` object. The project name is derived from the stack directory name. Below is the complete mapping from compose service fields to inspect fields.

**Important:** The compose file supports both short and long syntax for several fields (`ports`, `volumes`, `networks`, `healthcheck`, `devices`, `ulimits`, etc.). The parser must handle both forms. Refer to the Docker Compose v2 specification for the full syntax: https://docs.docker.com/reference/compose-file/

For the full Docker Engine API response schemas, reference: https://docs.docker.com/reference/api/engine/

#### 4.1.1 Identity and Metadata

| Compose field | Inspect field | Notes |
|---|---|---|
| (service key) | `Config.Labels["com.docker.compose.service"]` | Service name |
| `container_name` | `Name` | If specified. Otherwise `/{project}-{service}-1` (note leading slash) |
| (stack dir) | `Config.Labels["com.docker.compose.project"]` | Project name |
| `image` | `Config.Image` | Image reference as written in compose |
| `image` (resolved) | `Image` | Resolved to sha256 ID from images map |
| — | `Id` | Deterministic: `sha256(hash(project + service))` |
| — | `Created` | Deterministic ISO 8601 timestamp |
| — | `Path` | From image config's `Entrypoint[0]` or `/bin/sh` |
| — | `Args` | From image config's `Cmd` or compose `command` |

#### 4.1.2 State

| Compose field | Inspect field | Notes |
|---|---|---|
| — | `State.Status` | Default `"running"`, overridden by `.mock.yaml` |
| — | `State.Running` | `true` if status is `running` |
| — | `State.Paused` | `true` if status is `paused` |
| — | `State.Restarting` | `false` |
| — | `State.OOMKilled` | `false` |
| — | `State.Dead` | `false` |
| — | `State.Pid` | Deterministic integer from seed |
| — | `State.ExitCode` | `0`, overridden by `.mock.yaml` |
| — | `State.Error` | `""` |
| — | `State.StartedAt` | Deterministic ISO 8601 timestamp |
| — | `State.FinishedAt` | Deterministic, or `"0001-01-01T00:00:00Z"` if running |
| `healthcheck` | `State.Health.Status` | `"healthy"` if healthcheck defined, overridden by `.mock.yaml` |
| `healthcheck` | `State.Health.FailingStreak` | `0` |
| `healthcheck` | `State.Health.Log` | One deterministic log entry |

#### 4.1.3 Config (from compose + image merge)

Compose service fields override the corresponding image config defaults.

| Compose field | Inspect field | Notes |
|---|---|---|
| `command` | `Config.Cmd` | Overrides image's Cmd. Handles both string and list syntax |
| `entrypoint` | `Config.Entrypoint` | Overrides image's Entrypoint |
| `environment` / `env_file` | `Config.Env` | Merged with image's Env. Format: `["KEY=value", ...]`. For `env_file`, read the file from the stack dir and parse into env vars |
| `user` | `Config.User` | Overrides image's User |
| `working_dir` | `Config.WorkingDir` | Overrides image's WorkingDir |
| `hostname` | `Config.Hostname` | Default: first 12 chars of container ID |
| `domainname` | `Config.Domainname` | Default: `""` |
| `tty` | `Config.Tty` | Default: `false` |
| `stdin_open` | `Config.OpenStdin` | Default: `false` |
| `expose` | `Config.ExposedPorts` | Merged with image's ExposedPorts and `ports` entries |
| `ports` | `Config.ExposedPorts` | The container-side ports also appear here |
| `stop_signal` | `Config.StopSignal` | Default from image or `"SIGTERM"` |
| `stop_grace_period` | `Config.StopTimeout` | Convert duration string to integer seconds |
| `healthcheck` | `Config.Healthcheck` | `Test`, `Interval`, `Timeout`, `Retries`, `StartPeriod`, `StartInterval` |
| `labels` | `Config.Labels` | Merged with compose-generated labels (see §4.1.8) |
| — | `Config.Image` | Same as top-level compose `image` field |
| — | `Config.AttachStdin` | `false` |
| — | `Config.AttachStdout` | `true` |
| — | `Config.AttachStderr` | `true` |
| — | `Config.StdinOnce` | `false` |
| — | `Config.ArgsEscaped` | `false` |
| `volumes` (anonymous) | `Config.Volumes` | Map of anonymous volume mount destinations |
| — | `Config.Shell` | From image config, default `["/bin/sh", "-c"]` |
| — | `Config.OnBuild` | From image config, default `[]` |
| — | `Config.NetworkDisabled` | `false` |

#### 4.1.4 HostConfig

| Compose field | Inspect field | Notes |
|---|---|---|
| `ports` | `HostConfig.PortBindings` | Map of `"port/proto"` → `[{HostIp, HostPort}]`. Must handle short (`"8080:80"`) and long syntax (`target`, `published`, `protocol`, `host_ip`) |
| `volumes` (bind mounts) | `HostConfig.Binds` | Array of `"source:dest:mode"` strings for bind mounts |
| `volumes` (named/anonymous) | `HostConfig.Mounts` | Mount objects for non-bind volumes. Must handle short (`"vol:/data"`) and long syntax (`type`, `source`, `target`, `read_only`, etc.) |
| `volumes_from` | `HostConfig.VolumesFrom` | Resolve service names to container names |
| `network_mode` | `HostConfig.NetworkMode` | See §4.2 for network mode handling |
| `restart` | `HostConfig.RestartPolicy` | `{Name: "no"/"always"/"on-failure"/"unless-stopped", MaximumRetryCount: N}` |
| `privileged` | `HostConfig.Privileged` | Default: `false` |
| `cap_add` | `HostConfig.CapAdd` | Array of capability strings |
| `cap_drop` | `HostConfig.CapDrop` | Array of capability strings |
| `devices` | `HostConfig.Devices` | Array of `{PathOnHost, PathInContainer, CgroupPermissions}` |
| `dns` | `HostConfig.Dns` | Array of DNS server IPs |
| `dns_search` | `HostConfig.DnsSearch` | Array of search domains |
| `dns_opt` | `HostConfig.DnsOptions` | Array of DNS options |
| `extra_hosts` | `HostConfig.ExtraHosts` | Array of `"hostname:ip"` strings |
| `links` | `HostConfig.Links` | Array of `"container_name:alias"` strings. Resolve service names to container names |
| `logging` | `HostConfig.LogConfig` | `{Type: "driver", Config: {...}}` |
| `pid` | `HostConfig.PidMode` | `"host"` or container reference |
| `ipc` | `HostConfig.IpcMode` | `"host"`, `"private"`, `"shareable"`, or container reference |
| `shm_size` | `HostConfig.ShmSize` | Integer bytes |
| `sysctls` | `HostConfig.Sysctls` | Map of sysctl key→value |
| `tmpfs` | `HostConfig.Tmpfs` | Map of mountpoint→options |
| `ulimits` | `HostConfig.Ulimits` | Array of `{Name, Soft, Hard}` |
| `read_only` | `HostConfig.ReadonlyRootfs` | Default: `false` |
| `security_opt` | `HostConfig.SecurityOpt` | Array of strings |
| `storage_opt` | `HostConfig.StorageOpt` | Map of key→value |
| `mem_limit` | `HostConfig.Memory` | Integer bytes |
| `memswap_limit` | `HostConfig.MemorySwap` | Integer bytes |
| `mem_reservation` | `HostConfig.MemoryReservation` | Integer bytes |
| `cpus` | `HostConfig.NanoCpus` | Multiply float by 1e9 |
| `cpu_shares` | `HostConfig.CpuShares` | Integer |
| `cpuset` | `HostConfig.CpusetCpus` | String like `"0-3"` |
| `pids_limit` | `HostConfig.PidsLimit` | Integer |
| `blkio_config` | `HostConfig.BlkioWeight`, etc. | Multiple related fields |
| `oom_kill_disable` | `HostConfig.OomKillDisable` | Boolean |
| `oom_score_adj` | `HostConfig.OomScoreAdj` | Integer |
| `group_add` | `HostConfig.GroupAdd` | Array of group strings |
| `init` | `HostConfig.Init` | Boolean |
| `runtime` | `HostConfig.Runtime` | String |
| `isolation` | `HostConfig.Isolation` | `"default"`, `"hyperv"`, `"process"` |
| `userns_mode` | `HostConfig.UsernsMode` | String |
| `cgroup_parent` | `HostConfig.CgroupParent` | String |
| — | `HostConfig.AutoRemove` | `false` |
| — | `HostConfig.ContainerIDFile` | `""` |
| — | `HostConfig.PublishAllPorts` | `false` |
| — | `HostConfig.VolumeDriver` | `""` |
| — | `HostConfig.ConsoleSize` | `[0, 0]` |
| — | `HostConfig.MaskedPaths` | Default Docker masked paths list |
| — | `HostConfig.ReadonlyPaths` | Default Docker readonly paths list |

#### 4.1.5 NetworkSettings

| Compose field | Inspect field | Notes |
|---|---|---|
| `networks` | `NetworkSettings.Networks` | Map of network name → endpoint config (see §4.2) |
| `ports` | `NetworkSettings.Ports` | Same structure as `HostConfig.PortBindings` but includes entries for exposed ports with `null` bindings |
| `links` | `NetworkSettings.Networks[x].Links` | Array of linked container names |
| — | `NetworkSettings.SandboxID` | Deterministic hash |
| — | `NetworkSettings.SandboxKey` | `"/var/run/docker/netns/{deterministic}"` |

Each network entry in `NetworkSettings.Networks` contains:

| Field | Source | Notes |
|---|---|---|
| `NetworkID` | From the network's inspect `Id` | Resolved from networks map |
| `EndpointID` | Deterministic hash | From seed: project + service + network name |
| `Gateway` | From network's IPAM config | Or deterministic default |
| `IPAddress` | Deterministic | Derived from seed within subnet range |
| `IPPrefixLen` | From network's IPAM config | Or default `16` |
| `MacAddress` | Deterministic | Derived from seed, format `"02:42:xx:xx:xx:xx"` |
| `Aliases` | From compose `networks.{net}.aliases` | Plus service name and container hostname |
| `DNSNames` | Auto-generated | Service name, container name, aliases |
| `Links` | From compose `links` field | Resolved container names |
| `IPAMConfig` | Only if static IP assigned | From compose `networks.{net}.ipv4_address` / `ipv6_address` |
| `GlobalIPv6Address` | Deterministic or empty | |
| `IPv6Gateway` | From IPAM or empty | |

#### 4.1.6 Other Top-Level Fields

| Inspect field | Source | Notes |
|---|---|---|
| `ResolvConfPath` | Generated | `/var/lib/docker/containers/{id}/resolv.conf` |
| `HostnamePath` | Generated | `/var/lib/docker/containers/{id}/hostname` |
| `HostsPath` | Generated | `/var/lib/docker/containers/{id}/hosts` |
| `LogPath` | Generated | `/var/lib/docker/containers/{id}/{id}-json.log` |
| `RestartCount` | `0` | |
| `Driver` | `"overlay2"` | |
| `Platform` | `"linux"` | |
| `MountLabel` | `""` | |
| `ProcessLabel` | `""` | |
| `AppArmorProfile` | `""` | |
| `ExecIDs` | `[]` initially | Updated when exec sessions are created |
| `GraphDriver.Name` | `"overlay2"` | |
| `GraphDriver.Data` | Generated | Deterministic overlay2 paths using hash of container ID |
| `SizeRw` | Deterministic integer | Based on seed |
| `SizeRootFs` | From image size | |

#### 4.1.7 Mounts Array

The top-level `Mounts` array is built from the compose `volumes` field. Each entry:

```javascript
{
  Type: "volume" | "bind" | "tmpfs",
  Name: "volume_name",                    // only for named volumes
  Source: "/host/path" or "/var/lib/docker/volumes/name/_data",
  Destination: "/container/path",
  Driver: "local",                        // only for volumes
  Mode: "rw" | "ro" | "z" | "Z",
  RW: true | false,
  Propagation: "rprivate" | "" | etc.
}
```

Bind mounts: `Type: "bind"`, `Source` is the host path from compose, no `Name` or `Driver`.
Named volumes: `Type: "volume"`, `Source` is `/var/lib/docker/volumes/{name}/_data`, `Name` is the volume name.
Anonymous volumes: `Type: "volume"`, `Source` is `/var/lib/docker/volumes/{deterministic_hash}/_data`, no `Name`.
Tmpfs: `Type: "tmpfs"`, `Source` is `""`, no `Name` or `Driver`.

#### 4.1.8 Compose-Generated Labels

Every container gets these labels in `Config.Labels`, in addition to any user-defined labels from the compose file:

```javascript
{
  "com.docker.compose.project": "{project_name}",
  "com.docker.compose.service": "{service_name}",
  "com.docker.compose.container-number": "1",
  "com.docker.compose.oneoff": "False",
  "com.docker.compose.project.working_dir": "{absolute_path_to_stack_dir}",
  "com.docker.compose.project.config_files": "{absolute_path_to_compose_file}",
  "com.docker.compose.version": "2.30.0",
  "com.docker.compose.config-hash": "{deterministic_hash_of_service_config}",
  "com.docker.compose.image": "sha256:{image_id}"
}
```

The `com.docker.compose.project` label is critical — Portge uses it to associate containers with stacks.

#### 4.1.9 Compose Fields That Do Not Map to Inspect

These compose fields are purely orchestration-time or build-time and have no representation in the container inspect response:

- `depends_on` — orchestration ordering only
- `profiles` — compose CLI filtering only
- `extends` — resolved at parse time
- `deploy` (swarm mode) — not relevant
- `configs` / `secrets` (swarm mode) — not relevant
- `annotations` — relatively new, rare
- `attach` — controls log attachment
- `develop` / `watch` — compose dev tooling only
- `pull_policy` — controls pull behavior, not stored on container

### 4.2 Network Mode Handling

The compose `network_mode` field changes how a container's networking is set up:

**Default (no `network_mode` specified, no explicit `networks`):**
- `HostConfig.NetworkMode = "{project}_default"`
- Container is attached to the `{project}_default` network
- `NetworkSettings.Networks` has one entry for the default network

**Explicit networks (no `network_mode`, has `networks:`):**
- `HostConfig.NetworkMode = "{first_network_name}"`
- Container is attached to all declared networks
- `NetworkSettings.Networks` has an entry per declared network

**`network_mode: host`:**
- `HostConfig.NetworkMode = "host"`
- `NetworkSettings.Networks = { "host": { /* minimal, no IP addresses */ } }`
- Container has no `PortBindings` (ports are exposed directly on host)

**`network_mode: none`:**
- `HostConfig.NetworkMode = "none"`
- `NetworkSettings.Networks = {}`

**`network_mode: bridge`:**
- `HostConfig.NetworkMode = "bridge"`
- Container is attached to the default Docker bridge network

**`network_mode: "service:other_service"`:**
- Resolve `other_service` to its container ID within the same project
- `HostConfig.NetworkMode = "container:{resolved_container_id}"`
- `NetworkSettings.Networks = {}` (shares the target container's network namespace)

### 4.3 Network Generation

For each top-level `networks:` entry in a compose file (plus the implicit `{project}_default`):

#### 4.3.1 Network Name Resolution

```typescript
function resolveNetworkName(project: string, key: string, networkConfig: any): string {
  if (networkConfig?.external) {
    // external networks use the name as-is, or the explicit name
    return networkConfig.name || key;
  }
  // non-external networks get the project prefix unless name is explicit
  return networkConfig?.name || `${project}_${key}`;
}
```

#### 4.3.2 Single-Pass Creation

When the generator encounters a network reference:

1. Resolve the network name.
2. Check if it exists in the networks map.
3. If not, create it:
   - For non-external: create with full details (driver, IPAM, labels, options).
   - For external: create a stub with just the name and defaults.
4. If it already exists:
   - For non-external (this stack owns it): merge/update details into the existing entry.
   - For external: leave it as-is.

#### 4.3.3 NetworkInspect Fields

| Compose field | Inspect field | Notes |
|---|---|---|
| (resolved name) | `Name` | |
| — | `Id` | Deterministic hash of network name |
| — | `Created` | Deterministic ISO 8601 timestamp |
| `driver` | `Driver` | Default: `"bridge"` |
| `driver_opts` | `Options` | Map of key→value |
| `internal` | `Internal` | Default: `false` |
| `attachable` | `Attachable` | Default: `false` |
| `labels` | `Labels` | Including compose labels |
| `ipam` | `IPAM` | `{Driver, Config: [{Subnet, IPRange, Gateway, AuxiliaryAddresses}], Options}` |
| `enable_ipv4` | `EnableIPv4` | Default: `true` |
| `enable_ipv6` | `EnableIPv6` | Default: `false` |
| — | `Scope` | `"local"` |
| — | `Ingress` | `false` |
| — | `ConfigFrom` | `{Network: ""}` |
| — | `ConfigOnly` | `false` |
| — | `Containers` | Map of container ID → endpoint info. Built during post-processing |

The `Containers` map in `NetworkInspect` is populated after all containers are generated. For each container attached to the network:

```javascript
{
  "{container_id}": {
    Name: "{container_name}",
    EndpointID: "{deterministic}",
    MacAddress: "{deterministic}",
    IPv4Address: "{deterministic_ip}/{prefix}",
    IPv6Address: ""
  }
}
```

#### 4.3.4 Implicit Default Network

If any service in a compose file has no explicit `networks:` declaration and no `network_mode:`, the project gets an implicit `{project}_default` network. This network:

- Has `Driver: "bridge"`
- Has a deterministic IPAM subnet
- Has compose labels: `com.docker.compose.project: "{project}"`, `com.docker.compose.network: "default"`

### 4.4 Volume Generation

For each top-level `volumes:` entry in a compose file:

#### 4.4.1 Volume Name Resolution

Same pattern as networks:

```javascript
function resolveVolumeName(project, key, volumeConfig) {
  if (volumeConfig.external) {
    return volumeConfig.name || key;
  }
  return volumeConfig.name || `${project}_${key}`;
}
```

#### 4.4.2 VolumeInspect Fields

| Compose field | Inspect field | Notes |
|---|---|---|
| (resolved name) | `Name` | |
| `driver` | `Driver` | Default: `"local"` |
| `driver_opts` | `Options` | Map of key→value |
| `labels` | `Labels` | Including compose labels |
| — | `Mountpoint` | `/var/lib/docker/volumes/{name}/_data` |
| — | `CreatedAt` | Deterministic ISO 8601 timestamp |
| — | `Scope` | `"local"` |
| — | `Status` | `{}` |
| — | `UsageData` | `{Size: -1, RefCount: -1}` |

---

## 5. Deterministic Data Generation

### 5.1 Seed Hierarchy

All generated data derives from stable seeds. The seed hierarchy prevents one stack's data from shifting when another stack is added or removed.

```
Root seed: fixed constant (e.g., "portge-mock-v1")
  ├── Per-project seed: hash(root + project_name)
  │     ├── Per-service seed: hash(project_seed + service_name)
  │     │     ├── Container ID: hash(service_seed + "container-id")
  │     │     ├── MAC address: hash(service_seed + "mac")
  │     │     ├── IP address: hash(service_seed + "ip")
  │     │     ├── PID: hash(service_seed + "pid")
  │     │     ├── Timestamps: hash(service_seed + "created")
  │     │     ├── Stats baseline: hash(service_seed + "stats")
  │     │     └── Log content: hash(service_seed + "logs")
  │     └── Per-network seed: hash(project_seed + network_name)
  │           ├── Network ID: hash(network_seed + "network-id")
  │           └── Subnet: hash(network_seed + "subnet")
  └── Per-image seed: hash(root + image_reference)
        ├── Image ID: hash(image_seed + "image-id")
        └── Layer digests: hash(image_seed + "layer-N")
```

### 5.2 Deterministic Hash Function

Use a fast, deterministic hash (e.g., SHA-256 via Node's `crypto` module). Format outputs to match Docker's conventions:

- Container/image IDs: 64-character hex string, prefixed with `sha256:` where appropriate
- MAC addresses: `02:42:xx:xx:xx:xx` (Docker's OUI prefix is `02:42`)
- IP addresses: take hash bytes, map to host portion of the subnet
- PIDs: `hash mod 65536`, minimum 100
- Timestamps: base date (e.g., `2025-01-15T00:00:00Z`) + deterministic offset in seconds from hash

### 5.3 What Gets Deterministic Generation

| Data | Seed | Format |
|---|---|---|
| Container ID | project + service | 64 hex chars |
| Container PID | project + service | Integer 100-65535 |
| Container Created timestamp | project + service | ISO 8601 |
| Container StartedAt | project + service | ISO 8601, after Created |
| MAC address | project + service + network | `02:42:xx:xx:xx:xx` |
| IP address | project + service + network | Host portion from hash, within subnet |
| EndpointID | project + service + network | 64 hex chars |
| SandboxID | project + service | 64 hex chars |
| Network ID | network name | 64 hex chars |
| Image ID (synthetic) | image reference | `sha256:` + 64 hex chars |
| Layer digests (synthetic) | image reference + layer N | `sha256:` + 64 hex chars |
| SizeRw | project + service | Integer |
| Overlay2 paths | container ID | `/var/lib/docker/overlay2/{hash}/...` |
| Stats (CPU, memory, disk) | project + service + counter | See §9 |
| Log content | project + service + line N | See §8 |
| Process list | project + service | See §10 |

### 5.4 IP Address Conflicts

IP address collisions are explicitly acceptable. The mock does not track allocated IPs or ensure uniqueness. The data just needs to look plausible. If two containers happen to get the same IP from their seed hash, that's fine — the UI renders whatever the API returns and doesn't validate IP uniqueness.

---

## 6. Mutations and State Consistency

### 6.1 Mutation Principles

Every mutation function:

1. Updates the primary resource's inspect object.
2. Updates all cross-referenced resources (e.g., network `Containers` maps).
3. Emits Docker events in the correct sequence (see §7).
4. Is synchronous — no background state changes.
5. Lives in the mutations module. API handlers call mutation functions and never modify state directly or emit events directly.

### 6.2 Container Mutations

#### `containerStart(id)`
1. Set `State.Status = "running"`, `State.Running = true`, `State.Paused = false`.
2. Set `State.StartedAt` to current clock time.
3. Set `State.Pid` to deterministic PID.
4. Set `State.FinishedAt = "0001-01-01T00:00:00Z"`.
5. For each network in `NetworkSettings.Networks`: add this container to that network's `Containers` map.
6. If healthcheck defined: set `State.Health.Status = "starting"`, then transition to `"healthy"` (or schedule transition in non-e2e mode).
7. Emit events: `start`.

#### `containerStop(id)`
1. Set `State.Status = "exited"`, `State.Running = false`.
2. Set `State.FinishedAt` to current clock time.
3. Set `State.Pid = 0`.
4. Set `State.ExitCode = 0` (or 137 for kill).
5. For each network in `NetworkSettings.Networks`: remove this container from that network's `Containers` map.
6. Clear `State.Health` if present.
7. Emit events: `kill` (with signal), `die` (with exitCode), `stop`.

#### `containerRestart(id)`
1. Call `containerStop(id)`.
2. Call `containerStart(id)`.
3. Emit events: `kill`, `die`, `stop`, `start`, `restart`.

#### `containerPause(id)`
1. Set `State.Status = "paused"`, `State.Paused = true`.
2. Emit event: `pause`.

#### `containerUnpause(id)`
1. Set `State.Status = "running"`, `State.Paused = false`.
2. Emit event: `unpause`.

#### `containerRemove(id)`
1. Must be stopped first (or force flag, which stops it first).
2. Remove from all networks' `Containers` maps.
3. Delete from `containers` map.
4. Emit event: `destroy`.

#### `containerCreate(config)`
1. Build a `ContainerInspect` from the provided config.
2. Add to `containers` map.
3. Emit event: `create`.
4. Return the container ID.

#### `containerRename(id, newName)`
1. Update `Name` field.
2. Emit event: `rename`.

#### `containerKill(id, signal)`
1. If signal is SIGKILL or SIGTERM: stop the container.
2. Set `State.ExitCode` based on signal (137 for SIGKILL, 143 for SIGTERM).
3. Emit events: `kill` (with signal attribute).

### 6.3 Network Mutations

#### `networkCreate(config)`
1. Build a `NetworkInspect` from config.
2. Add to `networks` map.
3. Emit event: `{Type: "network", Action: "create"}`.

#### `networkRemove(id)`
1. Check no containers are attached (or it's empty).
2. Delete from `networks` map.
3. Emit event: `{Type: "network", Action: "destroy"}`.

#### `networkConnect(networkId, containerId, config)`
1. Add endpoint config to `container.NetworkSettings.Networks[networkName]`.
2. Add container entry to `network.Containers[containerId]`.
3. Emit event: `{Type: "network", Action: "connect"}`.

#### `networkDisconnect(networkId, containerId)`
1. Remove from `container.NetworkSettings.Networks[networkName]`.
2. Remove from `network.Containers[containerId]`.
3. Emit event: `{Type: "network", Action: "disconnect"}`.

### 6.4 Volume Mutations

#### `volumeCreate(config)`
1. Build a `VolumeInspect` from config.
2. Add to `volumes` map.
3. Emit event: `{Type: "volume", Action: "create"}`.

#### `volumeRemove(name)`
1. Check no containers reference this volume.
2. Delete from `volumes` map.
3. Emit event: `{Type: "volume", Action: "destroy"}`.

### 6.5 Image Mutations

#### `imageRemove(nameOrId)`
1. Check no containers reference this image (or force flag).
2. Delete from `images` map.
3. Emit events: `{Type: "image", Action: "untag"}` for each tag, then `{Type: "image", Action: "delete"}`.

#### `imagePrune(all)`
1. Find images not referenced by any container.
2. If `all` is false: only remove dangling images (no tags).
3. If `all` is true: remove all unreferenced images.
4. Delete from `images` map.
5. Emit delete events for each.
6. Return `{ImagesDeleted: [...], SpaceReclaimed: N}`.

### 6.6 Compose Lifecycle (via Mock CLI → Daemon API)

The mock CLI reads the compose file and issues daemon API calls. The daemon has no concept of compose projects — it's the CLI's job to know which containers, networks, and volumes belong to a project. The CLI reads the compose file to determine what to create/remove, and uses label filters to find existing containers for a project.

#### `docker compose up -p {project}`
1. CLI parses the compose file.
2. CLI creates networks (if not exist): `POST /networks/create` for each.
3. CLI creates volumes (if not exist): `POST /volumes/create` for each.
4. CLI creates containers: `POST /containers/create` for each service.
5. CLI starts containers: `POST /containers/{id}/start` for each.
6. CLI outputs animated TTY progress for each step.

#### `docker compose up --force-recreate -p {project}`
1. CLI queries daemon for existing containers with project label.
2. CLI stops existing containers.
3. CLI removes existing containers (new container IDs will be generated).
4. CLI creates new containers from current compose config.
5. CLI starts new containers.

#### `docker compose up --build -p {project}`
1. For services with `build:`: CLI triggers image regeneration (mock just creates a new image ID).
2. Then proceeds as normal `up`.

#### `docker compose up --remove-orphans -p {project}`
1. After creating/starting compose services, CLI finds containers with label `com.docker.compose.project={project}` whose `com.docker.compose.service` is NOT in the current compose file.
2. CLI stops and removes those orphan containers.

#### `docker compose down -p {project}`
1. CLI reads compose file to know which networks and volumes the project declared.
2. CLI queries daemon for containers with label `com.docker.compose.project={project}`.
3. CLI stops all those containers.
4. CLI removes those containers.
5. CLI removes project networks (by resolving names from the compose file and issuing remove calls). If a network still has containers from other stacks, the daemon returns an error and the CLI skips it (matching real behavior).
6. CLI outputs animated TTY progress.

#### `docker compose down --volumes -p {project}`
1. Same as `down`, plus CLI removes project-declared volumes (by resolving names from compose file).

#### `docker compose down --remove-orphans -p {project}`
1. Same as `down`, plus remove containers with the project label but whose service name isn't in the current compose file.

#### `docker compose stop -p {project}`
1. Stop all running containers with the project label.
2. Do NOT remove containers, networks, or volumes.

#### `docker compose start -p {project}`
1. Start all stopped containers with the project label.

#### `docker compose restart -p {project}`
1. Restart all containers with the project label.

#### `docker compose pull -p {project}`
1. For each service: simulate pulling the image.
2. If `update_available` is set in the mock sidecar for this service, update the image's digest in the images map (simulates a newer image being available locally after pull).
3. CLI outputs pull progress with TTY animations.

#### `docker compose ps -p {project}`
1. List containers with the project label.
2. Print formatted table output.

#### `docker compose config -f {file}`
1. Parse and normalize the compose file.
2. Print the resolved YAML to stdout.

#### `docker compose logs -p {project} [-f] [service]`
1. Stream log output for matching containers.
2. With `-f`: keep the connection open and stream new log lines as they're generated.

#### `docker compose exec {service} {command}`
1. Resolve service to container ID within the project.
2. Create exec session: `POST /containers/{id}/exec`.
3. Start exec session: `POST /exec/{id}/start`.

### 6.7 Docker Run (via Mock CLI → Daemon API)

`docker run` creates a one-off container not associated with a compose project.

1. Parse flags: `-d`, `--name`, `-p`, `-v`, `-e`, `--network`, `--restart`, `--hostname`, `-it`, `--rm`, `--privileged`, `--cap-add`, `--cap-drop`, etc.
2. Build a container config from the flags.
3. `POST /containers/create` with the config.
4. `POST /containers/{id}/start`.
5. If `-d`: print container ID and exit.
6. If not `-d`: attach to container output (stream logs until container stops).

---

## 7. Events

### 7.1 Event Format

Docker events follow this structure:

```javascript
{
  Type: "container" | "network" | "volume" | "image",
  Action: "create" | "start" | "stop" | "die" | "kill" | "pause" | "unpause" | "destroy" | "rename" | "connect" | "disconnect" | ...,
  Actor: {
    ID: "{resource_id}",
    Attributes: {
      // varies by resource type and action
      name: "{container_name}",
      image: "{image_name}",
      // ... other attributes
    }
  },
  time: 1234567890,       // unix timestamp
  timeNano: 1234567890000000000  // nanosecond timestamp
}
```

### 7.2 Event Sequences

Real Docker emits multiple events per operation in a specific order. The mock must match these sequences because Portge listens for specific event types and actions to update the UI.

**Container start:**
```
container start
```

**Container stop (graceful):**
```
container kill  (signal=15, i.e. SIGTERM)
container die   (exitCode=0)
container stop
```

**Container stop (forced/kill):**
```
container kill  (signal=9, i.e. SIGKILL)
container die   (exitCode=137)
container stop
```

**Container restart:**
```
container kill  (signal=15)
container die   (exitCode=0)
container stop
container start
container restart
```

**Container create:**
```
container create
```

**Container destroy (remove):**
```
container destroy
```

**Container pause/unpause:**
```
container pause
container unpause
```

**Container rename:**
```
container rename (oldName=X)
```

**Network lifecycle:**
```
network create
network connect   (container=ID)
network disconnect (container=ID)
network destroy
```

**Volume lifecycle:**
```
volume create
volume destroy
```

**Image lifecycle:**
```
image untag  (for each tag)
image delete
```

**Full `docker compose up` sequence:**
```
network create  (for each network)
volume create   (for each volume)
container create  (for each service)
network connect   (for each container+network pair)
container start   (for each service)
```

**Full `docker compose down` sequence:**
```
container kill    (for each container)
container die     (for each container)
container stop    (for each container)
network disconnect  (for each container+network pair)
container destroy   (for each container)
network destroy     (for each project network, if empty)
```

### 7.3 Event Emission Ownership

**Mutation functions own event emission.** The flow is always:

```
API handler → calls mutation function → mutation updates state → mutation emits events
```

- API route handlers (in `api/*.ts`) parse the request, validate input, call the appropriate mutation function, and return the response. They NEVER emit events directly.
- Mutation functions (in `mutations.ts`) are the ONLY place where state changes and event emissions happen. Each mutation function is responsible for emitting the correct sequence of events after updating state.
- The mock CLI NEVER emits events. It makes HTTP API calls to the daemon, which triggers mutations in the daemon, which emit events.

The events endpoint (`GET /events`) holds open an HTTP connection and writes events as newline-delimited JSON:

```typescript
class EventEmitter {
  private listeners = new Set<http.ServerResponse>();

  emit(event: DockerEvent): void {
    const payload = JSON.stringify(event) + "\n";
    for (const listener of this.listeners) {
      listener.write(payload);
    }
  }

  subscribe(res: http.ServerResponse): void {
    this.listeners.add(res);
    res.on("close", () => this.listeners.delete(res));
  }
}
```

The `GET /events` endpoint supports query parameters for filtering:
- `filters={"type":["container"],"event":["start","stop"]}`
- `since` and `until` timestamps

### 7.4 Container Events — Actor Attributes

Container events include these attributes in `Actor.Attributes`:

```javascript
{
  name: "{container_name_without_leading_slash}",
  image: "{image_name}",
  "com.docker.compose.project": "{project}",
  "com.docker.compose.service": "{service}",
  // plus all other container labels
  // action-specific:
  signal: "15",      // for kill events
  exitCode: "0",     // for die events
  oldName: "/old",   // for rename events
}
```

---

## 8. Logs

### 8.1 Log Generation

Logs are generated, not stored. Each container has a deterministic log stream derived from its seed.

**Log template (generic, works for all containers):**

```javascript
const logLevels = ["INFO", "DEBUG", "WARN", "INFO", "INFO", "INFO"]; // weighted toward INFO
const components = ["server", "handler", "worker", "scheduler", "cache", "db"];
const messages = [
  "Request processed successfully",
  "Connection established",
  "Task completed",
  "Health check passed",
  "Cache hit",
  "Processing batch",
  "Cleaning up resources",
  "Configuration loaded",
  "Listening on port {port}",
  "Worker started",
];
```

The seed selects which level, component, and message template to use for each log line. The line number serves as an additional seed input so each line is different but reproducible.

### 8.2 Log Phases

**Startup phase (emitted on container start, 5-8 lines):**
```
{timestamp} INFO  [main] Initializing service
{timestamp} INFO  [config] Configuration loaded from /etc/config
{timestamp} INFO  [server] Starting server on 0.0.0.0:{port}
{timestamp} INFO  [server] Server ready, accepting connections
{timestamp} INFO  [health] Health check endpoint available at /health
```

Port number and other specifics come from the container's config (exposed ports, environment variables).

**Periodic phase (emitted at intervals while running):**
```
{timestamp} {level} [{component}] {message}
```

One line every N seconds (configurable; default 5s for interactive, disabled for e2e).

**Shutdown phase (emitted on container stop, 2-3 lines):**
```
{timestamp} INFO  [server] Received shutdown signal
{timestamp} INFO  [server] Graceful shutdown complete
```

### 8.3 Log API

`GET /containers/{id}/logs` supports:

- `stdout=true|false` — include stdout
- `stderr=true|false` — include stderr
- `follow=true|false` — stream new logs as they're generated
- `tail=N` — return only the last N lines
- `since=TIMESTAMP` — return logs after this time
- `until=TIMESTAMP` — return logs before this time
- `timestamps=true|false` — prefix each line with RFC3339Nano timestamp

When `follow=true`, the response stays open and new periodic log lines are written as they're generated. The mock uses a timer (or the injectable clock for e2e) to generate new lines.

### 8.4 Log Stream Format

Docker log streams use a multiplexed format with an 8-byte header per frame:

```
[stream_type(1 byte)][0x00][0x00][0x00][size(4 bytes big-endian)][payload]
```

Where `stream_type` is `1` for stdout, `2` for stderr. The mock should output in this format when the request is not TTY mode. If `tty: true` in the container config, raw output without framing is used.

### 8.5 E2E Mode

Periodic logs are disabled in e2e mode (configurable flag on the mock daemon). Only startup and shutdown logs are generated. This makes log output deterministic and finite — tests can assert on exact log content.

---

## 9. Stats Streaming

### 9.1 Stats API

`GET /containers/{id}/stats` streams JSON objects, one per second (or per `stream=false` for a single snapshot).

### 9.2 Deterministic Stats Generation

Each container's stats baseline is derived from its seed:

```javascript
function generateStats(seed, counter) {
  const rng = seededRandom(seed + counter);

  // Base values derived from seed (stable per container)
  const baseCpuPercent = hashToRange(seed, "cpu", 1, 80);    // 1-80%
  const baseMemUsage = hashToRange(seed, "mem", 50, 500);     // 50-500 MB
  const baseNetRx = hashToRange(seed, "netrx", 100, 10000);   // bytes/sec
  const baseNetTx = hashToRange(seed, "nettx", 100, 10000);

  // Vary slightly with counter to simulate live changes
  // Use sine wave for smooth variation
  const cpuVariation = Math.sin(counter * 0.1) * 5;
  const memVariation = Math.sin(counter * 0.07) * 20;

  return {
    cpu_stats: {
      cpu_usage: {
        total_usage: /* cumulative based on baseCpuPercent + cpuVariation */,
        percpu_usage: [/* distributed across cores */],
        usage_in_kernelmode: /* portion of total */,
        usage_in_usermode: /* portion of total */,
      },
      system_cpu_usage: /* host CPU cumulative */,
      online_cpus: 4,
      throttling_data: { periods: 0, throttled_periods: 0, throttled_time: 0 }
    },
    precpu_stats: { /* previous interval's cpu_stats */ },
    memory_stats: {
      usage: (baseMemUsage + memVariation) * 1024 * 1024,
      max_usage: (baseMemUsage + 50) * 1024 * 1024,
      limit: 1024 * 1024 * 1024,  // 1GB or from container's memory limit
      stats: { /* cgroup memory stats */ }
    },
    networks: {
      eth0: {
        rx_bytes: baseNetRx * counter,
        tx_bytes: baseNetTx * counter,
        rx_packets: /* derived */,
        tx_packets: /* derived */,
        rx_errors: 0,
        tx_errors: 0,
        rx_dropped: 0,
        tx_dropped: 0,
      }
    },
    blkio_stats: {
      io_service_bytes_recursive: [
        { major: 8, minor: 0, op: "read", value: hashToRange(seed, "diskr", 0, 1000000) * counter },
        { major: 8, minor: 0, op: "write", value: hashToRange(seed, "diskw", 0, 500000) * counter }
      ]
    },
    pids_stats: { current: hashToRange(seed, "pids", 1, 50) },
    read: /* current timestamp */,
    preread: /* previous timestamp */,
  };
}
```

### 9.3 E2E Mode

In e2e mode, stats are returned as a single snapshot (`stream=false` behavior always). The counter is fixed at 1, producing consistent values. No streaming, no variation.

---

## 10. Process List (Top)

### 10.1 Top API

`GET /containers/{id}/top?ps_args=aux` returns a process list.

### 10.2 Deterministic Process List

The process list is generated from the container's seed and image config:

```javascript
function generateTop(container) {
  const seed = container._seed;
  const processes = [];

  // PID 1 is always the main process (from entrypoint/cmd)
  processes.push({
    PID: "1",
    USER: container.Config.User || "root",
    COMMAND: container.Path + " " + (container.Args || []).join(" "),
    "%CPU": hashToRange(seed, "cpu-1", 0, 10).toFixed(1),
    "%MEM": hashToRange(seed, "mem-1", 0, 5).toFixed(1),
    VSZ: hashToRange(seed, "vsz-1", 10000, 500000).toString(),
    RSS: hashToRange(seed, "rss-1", 5000, 100000).toString(),
    TTY: "?",
    STAT: "Ss",
    START: "00:00",
    TIME: "0:01",
  });

  // Generate 2-8 additional worker/helper processes
  const numWorkers = hashToRange(seed, "nprocs", 2, 8);
  const workerNames = ["worker", "handler", "scheduler", "gc", "logger", "monitor"];

  for (let i = 0; i < numWorkers; i++) {
    processes.push({
      PID: (i + 2).toString(),
      USER: container.Config.User || "root",
      COMMAND: workerNames[i % workerNames.length],
      "%CPU": hashToRange(seed, `cpu-${i+2}`, 0, 5).toFixed(1),
      "%MEM": hashToRange(seed, `mem-${i+2}`, 0, 2).toFixed(1),
      VSZ: hashToRange(seed, `vsz-${i+2}`, 5000, 200000).toString(),
      RSS: hashToRange(seed, `rss-${i+2}`, 1000, 50000).toString(),
      TTY: "?",
      STAT: "Sl",
      START: "00:00",
      TIME: "0:00",
    });
  }

  return {
    Titles: ["PID", "USER", "%CPU", "%MEM", "VSZ", "RSS", "TTY", "STAT", "START", "TIME", "COMMAND"],
    Processes: processes.map(p => [p.PID, p.USER, p["%CPU"], p["%MEM"], p.VSZ, p.RSS, p.TTY, p.STAT, p.START, p.TIME, p.COMMAND])
  };
}
```

---

## 11. Mock Shell

### 11.1 Purpose

The UI has a "shell" feature that opens a terminal to a container via the Docker exec API. The mock provides a fake shell that responds to a handful of commands with plausible output.

### 11.2 Exec API Flow

1. `POST /containers/{id}/exec` — create an exec session. Body includes `Cmd`, `AttachStdin`, `AttachStdout`, `AttachStderr`, `Tty`. Returns `{Id: "{exec_id}"}`.
2. `POST /exec/{id}/start` — start the exec session. This upgrades to a WebSocket or streaming connection.
3. `GET /exec/{id}/json` — inspect exec session (running status, exit code, etc.).

The exec session ID is added to the container's `ExecIDs` array. On session end, it's removed.

### 11.3 Fake Shell Commands

The mock shell presents a prompt (`root@{hostname}:/# ` or `{user}@{hostname}:/{workdir}$ `) and processes input line by line.

Supported commands:

| Command | Response |
|---|---|
| `ls [path]` | Deterministic file listing based on the container's image archetype. Default directory contents vary slightly per container seed. |
| `ls -la [path]` | Same but with permissions, ownership, sizes, dates. |
| `ps aux` | Same output as the top endpoint. |
| `cat /etc/hostname` | Container's `Config.Hostname`. |
| `cat /etc/hosts` | Generated hosts file with container IP, localhost, and extra_hosts entries. |
| `cat /etc/resolv.conf` | Generated resolv.conf with DNS servers from `HostConfig.Dns`. |
| `env` | Container's `Config.Env`, one per line. |
| `printenv [VAR]` | Specific env var value. |
| `whoami` | User from `Config.User`, or `root`. |
| `id` | `uid=0(root) gid=0(root) groups=0(root)` or derived from `Config.User`. |
| `hostname` | Container's `Config.Hostname`. |
| `echo [...]` | Echo back arguments. |
| `pwd` | Container's `Config.WorkingDir` or `/`. |
| `uname -a` | `Linux {hostname} 5.15.0-mock #1 SMP x86_64 GNU/Linux` |
| `date` | Current mock clock time. |
| `uptime` | Derived from container's `StartedAt`. |
| `free -m` | Generated memory stats consistent with the container's memory limit. |
| `df -h` | Generated disk usage stats. |
| `exit` | Close the exec session. |
| `cd [path]` | Update the tracked working directory (affects prompt). |
| Anything else | `bash: {command}: command not found` |

### 11.4 Stateful Shell Session

Each exec session tracks:
- Current working directory (starts at `Config.WorkingDir`)
- Environment variables (from `Config.Env`, plus any `export` commands within the session)

This state is per-session, not shared between sessions or persisted.

---

## 12. Update Detection

### 12.1 Distribution Inspect

`GET /distribution/{name}/json` returns image distribution information, primarily the digest.

The mock uses this to simulate update availability:

```javascript
handleDistributionInspect(imageName) {
  const image = this.findImageByName(imageName);
  const container = this.findContainerByImage(imageName);
  const mockConfig = this.getMockConfig(container);

  let digest = image.Descriptor.digest;

  if (mockConfig?.update_available) {
    // Return a different digest to simulate a newer version on the registry
    digest = deterministicHash(digest + "-updated");
    // Format as proper digest
    digest = "sha256:" + digest;
  }

  return {
    Descriptor: {
      mediaType: image.Descriptor.mediaType,
      digest: digest,
      size: image.Descriptor.size,
    }
  };
}
```

Portge's update check flow:
1. Call `DistributionInspect(imageName)` — gets the registry digest.
2. Call `ImageInspect(imageName)` — gets the local digest.
3. Compare digests. If different → update available.

### 12.2 Needs Recreation

When `.mock.yaml` specifies `needs_recreation: true` for a service, the generator stores a different image reference in `Config.Image` than what's in the compose file.

If compose says `image: nginx:1.27`, the container's `Config.Image` is set to `nginx:1.25` (deterministically derived older-looking tag).

Portge's recreation check flow:
1. Read the compose file to get the current image reference for the service.
2. Call `ContainerInspect(id)` to get `Config.Image`.
3. Compare. If different → recreation necessary → show rocket icon.

---

## 13. Mock CLI

### 13.1 Overview

A single TypeScript binary compiled via `bun build --compile` (or Node SEA, or Deno compile). Named `docker`, placed in PATH before the real Docker binary. Handles both `docker compose ...` and `docker ...` subcommands.

The mock CLI is a thin client. It parses arguments, makes HTTP requests to the mock daemon over the Unix socket, and prints output. All state mutation happens in the daemon. The CLI never modifies state directly and never emits events.

### 13.2 Architecture

```
process.argv → parse subcommand
  ├── "compose" → parseComposeArgs() → compose handler
  └── anything else → parseDockerArgs() → docker handler

Handlers make HTTP requests to the mock daemon over the Unix socket.
The socket path is read from DOCKER_HOST env var (format: "unix:///path/to/socket").
```

### 13.3 TTY Output

The mock CLI always assumes TTY mode since xterm.js provides a TTY when Portge shells out. Non-TTY is a defensive fallback detected via `process.stdout.isTTY`.

**TTY mode (always the case when Portge shells out, since xterm.js provides a TTY):**

Uses ANSI escape sequences for animated progress output.

`docker compose up` progress:
```
[+] Running 3/3
 ✔ Network mystack_default  Created    0.1s
 ✔ Container mystack-db-1   Started    0.3s
 ✔ Container mystack-web-1  Started    0.5s
```

Lines update in place using `\r` and ANSI cursor movement. Simulated delays between steps (short but nonzero — 100-300ms per step).

`docker compose pull` progress:
```
[+] Pulling 2/2
 ✔ db Pulled                           1.2s
 ✔ web Pulled                          0.8s
```

`docker compose down` progress:
```
[+] Running 3/3
 ✔ Container mystack-web-1  Stopped    0.3s
 ✔ Container mystack-db-1   Stopped    0.5s
 ✔ Network mystack_default  Removed    0.1s
```

**Non-TTY mode (fallback):**

Plain sequential output without ANSI codes:
```
Network mystack_default  Created
Container mystack-db-1   Started
Container mystack-web-1  Started
```

### 13.4 Compose Commands

All compose commands support `-p {project}` / `--project-name {project}` and `-f {file}` / `--file {file}`.

| Command | Daemon API calls | Output |
|---|---|---|
| `docker compose up [-d]` | Create networks, volumes, containers; start containers | TTY progress |
| `docker compose up --force-recreate` | Stop, remove, recreate, start | TTY progress |
| `docker compose up --build` | Regenerate built images, then up | TTY progress |
| `docker compose up --remove-orphans` | Normal up, then remove orphan containers | TTY progress |
| `docker compose down` | Stop, remove containers and networks | TTY progress |
| `docker compose down --volumes` | Down + remove volumes | TTY progress |
| `docker compose down --remove-orphans` | Down + remove orphans | TTY progress |
| `docker compose stop` | Stop containers | TTY progress |
| `docker compose start` | Start containers | TTY progress |
| `docker compose restart` | Restart containers | TTY progress |
| `docker compose pull` | Simulate image pull, update digests if update_available | TTY progress |
| `docker compose ps` | List containers for project | Formatted table |
| `docker compose config` | Parse + normalize compose file | YAML to stdout |
| `docker compose logs [-f] [service]` | Stream container logs | Log lines to stdout |
| `docker compose exec {service} {cmd}` | Exec API | Interactive or command output |

### 13.5 Docker Commands

| Command | Daemon API calls | Output |
|---|---|---|
| `docker run [flags] IMAGE [CMD]` | Create + start container | Container ID (detached) or attach |
| `docker start CONTAINER` | Start container | Container name |
| `docker stop CONTAINER` | Stop container | Container name |
| `docker restart CONTAINER` | Restart container | Container name |
| `docker rm [-f] CONTAINER` | Remove container | Container name |
| `docker kill [--signal SIG] CONTAINER` | Kill container | Container name |
| `docker pause CONTAINER` | Pause container | Container name |
| `docker unpause CONTAINER` | Unpause container | Container name |
| `docker inspect CONTAINER\|NETWORK\|VOLUME\|IMAGE` | Inspect resource | JSON to stdout |
| `docker logs [-f] CONTAINER` | Stream logs | Log lines to stdout |
| `docker top CONTAINER` | Process list | Formatted table |
| `docker stats [CONTAINER...]` | Stats stream | Formatted table (updates in place if TTY) |
| `docker exec [-it] CONTAINER CMD` | Exec API | Command output or interactive shell |
| `docker network create [flags] NAME` | Create network | Network ID |
| `docker network rm NAME` | Remove network | Network name |
| `docker network connect [flags] NETWORK CONTAINER` | Connect | (empty) |
| `docker network disconnect NETWORK CONTAINER` | Disconnect | (empty) |
| `docker volume create [flags] NAME` | Create volume | Volume name |
| `docker volume rm NAME` | Remove volume | Volume name |
| `docker image prune [-a]` | Prune images | Summary |
| `docker ps [flags]` | List containers | Formatted table |
| `docker images` | List images | Formatted table |
| `docker network ls` | List networks | Formatted table |
| `docker volume ls` | List volumes | Formatted table |

### 13.6 Exit Codes

The mock CLI must return correct exit codes since the Go backend may check them:

- `0` — success
- `1` — general error (container not found, invalid args, etc.)
- `125` — Docker daemon error
- `126` — command cannot be invoked (exec)
- `127` — command not found (exec)

---

## 14. Daemon API Endpoints

### 14.1 System

| Endpoint | Notes |
|---|---|
| `GET /_ping` | Returns `OK` with `API-Version` header |
| `HEAD /_ping` | Same headers, no body |
| `GET /version` | Returns mock Docker version info |
| `GET /info` | Returns mock system info (OS, kernel, containers count, images count, etc.) |
| `GET /events` | Streaming endpoint, returns events as newline-delimited JSON. Supports `filters`, `since`, `until` query params |

### 14.2 Containers

| Endpoint | Notes |
|---|---|
| `GET /containers/json` | List containers. Supports `all`, `limit`, `size`, `filters` query params. **Filters are critical** — the Go backend uses label filters extensively |
| `POST /containers/create` | Create container. Query param `name`. Body is container config |
| `GET /containers/{id}/json` | Inspect container. Supports `size` query param |
| `DELETE /containers/{id}` | Remove container. Query params `v` (remove volumes), `force`, `link` |
| `POST /containers/{id}/start` | Start container |
| `POST /containers/{id}/stop` | Stop container. Query param `t` (timeout seconds) |
| `POST /containers/{id}/restart` | Restart container. Query param `t` |
| `POST /containers/{id}/kill` | Kill container. Query param `signal` |
| `POST /containers/{id}/pause` | Pause container |
| `POST /containers/{id}/unpause` | Unpause container |
| `POST /containers/{id}/rename` | Rename container. Query param `name` |
| `POST /containers/{id}/update` | Update container resource limits. Body is update config |
| `GET /containers/{id}/top` | Process list. Query param `ps_args` |
| `GET /containers/{id}/logs` | Log stream. Query params `stdout`, `stderr`, `follow`, `tail`, `since`, `until`, `timestamps` |
| `GET /containers/{id}/stats` | Stats stream. Query param `stream` (default true), `one-shot` |
| `POST /containers/{id}/exec` | Create exec. Body includes `Cmd`, `AttachStdin/Stdout/Stderr`, `Tty` |

### 14.3 Exec

| Endpoint | Notes |
|---|---|
| `POST /exec/{id}/start` | Start exec. Upgrades to streaming connection. Body includes `Detach`, `Tty` |
| `GET /exec/{id}/json` | Inspect exec session |

### 14.4 Images

| Endpoint | Notes |
|---|---|
| `GET /images/json` | List images. Supports `all`, `filters`, `shared-size`, `manifests` query params |
| `GET /images/{name}/json` | Inspect image |
| `DELETE /images/{name}` | Remove image. Query params `force`, `noprune` |
| `POST /images/prune` | Prune unused images. Query param `filters` |
| `GET /distribution/{name}/json` | Distribution inspect (for update checking) |

### 14.5 Networks

| Endpoint | Notes |
|---|---|
| `GET /networks` | List networks. Supports `filters` query param |
| `GET /networks/{id}` | Inspect network |
| `POST /networks/create` | Create network. Body is network config |
| `DELETE /networks/{id}` | Remove network |
| `POST /networks/{id}/connect` | Connect container. Body includes `Container`, `EndpointConfig` |
| `POST /networks/{id}/disconnect` | Disconnect container. Body includes `Container`, `Force` |

### 14.6 Volumes

| Endpoint | Notes |
|---|---|
| `GET /volumes` | List volumes. Supports `filters` query param |
| `GET /volumes/{name}` | Inspect volume |
| `POST /volumes/create` | Create volume. Body is volume config |
| `DELETE /volumes/{name}` | Remove volume. Query param `force` |

### 14.7 Mock-Only

| Endpoint | Notes |
|---|---|
| `POST /_mock/reset` | Clear all state, reload from disk (see §3.6) |

### 14.8 ID Resolution

All endpoints that take `{id}` must support both full IDs and short prefixes (minimum 3 characters). Also support resolution by container name. The resolution order:

1. Exact full ID match
2. Short ID prefix match (must be unambiguous)
3. Name match (with or without leading `/`)

---

## 15. Project Structure

```
mock-docker/
  src/
    state.ts              // MockState class — the core maps
    generator.ts          // Compose files → inspect objects
    deterministic.ts      // Seeded hash functions for IDs, MACs, IPs, timestamps, etc.
    projections.ts        // Inspect → list transformations
    mutations.ts          // All state mutation functions — THE ONLY place state changes and events are emitted
    events.ts             // Event emitter + event types + event sequence definitions
    logs.ts               // Log template engine (startup, periodic, shutdown phases)
    stats.ts              // Deterministic stats generator (CPU, memory, network, disk)
    top.ts                // Process list generator
    shell.ts              // Fake shell command router
    compose-parser.ts     // Compose file parsing + normalization (full v2 spec)
    network-modes.ts      // Network mode resolution logic
    name-resolution.ts    // Container/network/volume name and ID resolution + short prefix matching
    clock.ts              // Injectable clock (real time vs fixed time for e2e)
    filters.ts            // Filter parsing and matching logic (shared across all list endpoints)
    api/
      containers.ts       // /containers/* route handlers
      networks.ts         // /networks/* route handlers
      volumes.ts          // /volumes/* route handlers
      images.ts           // /images/* route handlers
      system.ts           // /_ping, /version, /info, /events
      exec.ts             // /exec/* route handlers
      distribution.ts     // /distribution/* route handlers
      mock.ts             // /_mock/reset
    server.ts             // HTTP server on Unix socket
  cli/
    index.ts              // Mock CLI entry point (docker + docker compose)
    compose-handler.ts    // docker compose subcommand handlers
    docker-handler.ts     // docker subcommand handlers
    tty-output.ts         // ANSI animated progress output
    socket-client.ts      // HTTP client for Unix socket communication
  test/
    ...
  images.json             // Pre-captured real image data (Layer 1)
  package.json
  tsconfig.json
```

---

## 16. Configuration and Modes

### 16.1 Daemon Configuration

The mock daemon accepts configuration via CLI flags:

| Flag | Default | Notes |
|---|---|---|
| `--socket` | `/var/run/docker.sock` | Unix socket path to listen on |
| `--stacks-dir` | `./stacks` | Runtime stacks directory (read/write) |
| `--stacks-source` | `./test-data/stacks` | Source directory of test fixture stacks (read-only) |
| `--e2e` | `false` | Enables e2e mode (see §16.2) |
| `--clock-base` | `2025-01-15T00:00:00Z` | Base time for deterministic timestamp generation (clock is always fixed) |
| `--log-interval` | `5000` | Milliseconds between periodic log lines (ignored in e2e mode) |
| `--stats-interval` | `1000` | Milliseconds between stats updates (ignored in e2e mode) |

### 16.2 E2E Mode Behavior

When `--e2e` is set:

- Periodic log generation is disabled. Only startup and shutdown logs are emitted.
- Stats endpoint returns a single snapshot, never streams.
- Health check transitions are immediate (starting → healthy on container start, no delay).
- No background state changes (no containers randomly going unhealthy).

### 16.3 Interactive Mode Behavior

When `--e2e` is NOT set:

- Periodic logs tick at `--log-interval`.
- Stats endpoint streams updates at `--stats-interval`.
- Health checks transition from `starting` → `healthy` after a plausible delay.

---

## 17. Transport — Unix Socket HTTP Server

### 17.1 Server Setup

The mock daemon listens on a Unix socket using Node's `http` module:

```typescript
const server = http.createServer(requestHandler);
server.listen(socketPath);
```

The Docker API uses HTTP/1.1 over the Unix socket. The mock must handle:

- Standard REST endpoints (JSON request/response)
- Streaming responses (events, logs, stats) — chunked transfer encoding, connection kept open
- Connection upgrades for exec (WebSocket or raw stream)
- Query parameter parsing (especially `filters` which is JSON-encoded)

### 17.2 API Version Prefix

Docker API requests are prefixed with the API version: `/v1.46/containers/json`. The mock should accept requests both with and without the version prefix. Strip the version prefix during routing.

### 17.3 Error Responses

Docker API errors follow this format:

```json
{
  "message": "No such container: abc123"
}
```

With appropriate HTTP status codes:
- `404` — resource not found
- `409` — conflict (e.g., trying to remove a running container)
- `304` — not modified (e.g., starting an already-started container)
- `500` — internal server error

---

## 18. Filters

### 18.1 Filter Format

Many list endpoints support a `filters` query parameter, which is a JSON-encoded object. The Go backend relies heavily on these, particularly label filters. The mock MUST implement filter support correctly.

```
GET /containers/json?filters={"label":["com.docker.compose.project=mystack"],"status":["running"]}
```

Each filter key maps to an array of values. A resource matches a filter if it matches ANY value for that key (OR within a key). A resource must match ALL keys to be included (AND across keys).

### 18.2 Supported Filters

**Container list filters:**
- `id` — container ID prefix
- `name` — container name
- `label` — `key` or `key=value`
- `status` — `created`, `running`, `paused`, `exited`, `dead`
- `ancestor` — image name or ID
- `network` — network name or ID
- `volume` — volume name

**Image list filters:**
- `dangling` — `true` or `false`
- `label` — `key` or `key=value`
- `reference` — image reference pattern

**Network list filters:**
- `driver` — network driver
- `id` — network ID prefix
- `label` — `key` or `key=value`
- `name` — network name
- `scope` — `local` or `swarm`
- `type` — `custom` or `builtin`

**Volume list filters:**
- `dangling` — `true` or `false`
- `driver` — volume driver
- `label` — `key` or `key=value`
- `name` — volume name

**Event filters:**
- `type` — `container`, `network`, `volume`, `image`
- `event` — action name (e.g., `start`, `stop`, `create`)
- `container` — container name or ID
- `image` — image name or ID
- `label` — `key` or `key=value`
- `network` — network name or ID
- `volume` — volume name

---

## 19. Summary of Cross-Resource References

This table catalogs every place where one resource type references another, and which mutations must update both sides:

| Resource A | Resource B | A→B reference | B→A reference | Mutations that touch both |
|---|---|---|---|---|
| Container | Network | `NetworkSettings.Networks[name]` | `network.Containers[id]` | start, stop, create, remove, networkConnect, networkDisconnect |
| Container | Image | `Image` (sha256 ID), `Config.Image` (name) | (image list `Containers` count is computed, not stored) | create, remove |
| Container | Volume | `Mounts[]`, `HostConfig.Binds[]` | (volumes don't track consumers) | create, remove |
| Container | Exec | `ExecIDs[]` | `exec.ContainerID` | execCreate, exec session end |
| Container | Container | `HostConfig.NetworkMode = "container:{id}"` | (implicit) | target container remove |
| Container | Container | `HostConfig.Links[]` | (implicit) | linked container remove/rename |
| Container | Container | `HostConfig.VolumesFrom[]` | (implicit) | source container remove |
| Network | Network | (none) | (none) | — |
| Volume | Volume | (none) | (none) | — |
| Image | Image | `ParentId` (rare) | (none) | — |
