# FreeSWITCH Stack

## Research
- GitHub: signalwire/freeswitch
- No official Docker image; repo has example Dockerfiles for building from source
- Community image `safarov/freeswitch` is the most popular on Docker Hub
- FreeSWITCH is a telephony platform / softswitch

## Compose
- Uses `safarov/freeswitch:latest` community image
- Uses host networking (common for SIP/RTP to avoid NAT issues)
- Bind mounts for config, logs, and sound files
- Capabilities for real-time scheduling and network admin
