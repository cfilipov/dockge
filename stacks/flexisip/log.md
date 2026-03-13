# Flexisip Stack

## Research
- GitHub: BelledonneCommunications/flexisip
- No official Docker image; build-from-source Dockerfile exists in repo
- Community image `etnperlong/flexisip` found on Docker Hub
- SIP proxy server from Belledonne Communications (makers of Linphone)

## Compose
- Uses community Docker image `etnperlong/flexisip:latest`
- Exposes SIP ports 5060 (UDP/TCP) and 5061 (TLS)
- Bind mounts for config and logs directories
