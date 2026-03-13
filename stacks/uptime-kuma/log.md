# Uptime Kuma

## Sources
- Official repo: https://github.com/louislam/uptime-kuma
- Official compose.yaml in repo root: https://raw.githubusercontent.com/louislam/uptime-kuma/master/compose.yaml
- README Docker instructions

## Notes
- Uptime Kuma is a self-hosted monitoring tool (like "Uptime Robot" but self-hosted)
- Official image: `louislam/uptime-kuma:2`
- Compose taken directly from the official compose.yaml in the repository
- Data stored in bind mount at `./data:/app/data`
- NFS file systems are NOT supported for the data volume
- Web UI accessible on port 3001
