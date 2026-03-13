# ShellHub

## Sources
- Official repo: https://github.com/shellhub-io/shellhub
- Official docker-compose.yml: https://raw.githubusercontent.com/shellhub-io/shellhub/master/docker-compose.yml
- Deployment docs: https://docs.shellhub.io/self-hosted/deploying

## Notes
- ShellHub is a centralized SSH gateway for remotely accessing Linux devices
- Official images: shellhubio/gateway, shellhubio/ssh, shellhubio/api, shellhubio/ui, shellhubio/cli
- Compose simplified from the official docker-compose.yml (removed cloud/enterprise-only env vars)
- Requires key generation before first start: `make keygen` or manually create ssh_private_key, api_private_key, api_public_key files
- After starting, run `./bin/setup` (from the cloned repo) to create first user
- Web UI accessible on port 80, SSH on port 22
- Redis used for caching (no persistence)
- Official deployment recommends cloning the repo and using `make start` for full setup
